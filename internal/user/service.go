package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type RepositoryInterface interface {
	Create(ctx context.Context, input *CreateUser) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

type Service struct {
	repository RepositoryInterface
}

func NewService(repository RepositoryInterface) *Service {
	return &Service{
		repository: repository,
	}
}

const (
	maxUsernameCharacters = 50
	maxEmailCharacters    = 100

	minPasswordCharacters = 8
	maxPasswordBytes      = 72
)

func (s *Service) Register(ctx context.Context, input *RegisterUser) (*User, error) {
	if input == nil {
		return nil, fmt.Errorf(
			"%w: user is required",
			ErrUserValidation,
		)
	}

	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	password := input.Password

	if username == "" {
		return nil, fmt.Errorf(
			"%w: username is required",
			ErrUserValidation,
		)
	}

	if utf8.RuneCountInString(username) > maxUsernameCharacters {
		return nil, fmt.Errorf(
			"%w: username must not exceed %d characters",
			ErrUserValidation,
			maxUsernameCharacters,
		)
	}

	if email == "" {
		return nil, fmt.Errorf(
			"%w: email is required",
			ErrUserValidation,
		)
	}

	if utf8.RuneCountInString(email) > maxEmailCharacters {
		return nil, fmt.Errorf(
			"%w: email must not exceed %d characters",
			ErrUserValidation,
			maxEmailCharacters,
		)
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil ||
		parsedEmail.Name != "" ||
		parsedEmail.Address != email {
		return nil, fmt.Errorf(
			"%w: invalid email",
			ErrUserValidation,
		)
	}

	if password == "" {
		return nil, fmt.Errorf(
			"%w: password is required",
			ErrUserValidation,
		)
	}

	if utf8.RuneCountInString(password) < minPasswordCharacters {
		return nil, fmt.Errorf(
			"%w: password must be at least %d characters",
			ErrUserValidation,
			minPasswordCharacters,
		)
	}

	if len(password) > maxPasswordBytes {
		return nil, fmt.Errorf(
			"%w: password must not exceed %d bytes",
			ErrUserValidation,
			maxPasswordBytes,
		)
	}
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to hash password: %w",
			err,
		)
	}

	user, err := s.repository.Create(
		ctx,
		&CreateUser{
			Username:     username,
			Email:        email,
			PasswordHash: string(hash),
			Role:         RoleUser,
		},
	)
	if err != nil {
		if errors.Is(err, ErrUserConflict) {
			return nil, ErrUserConflict
		}

		return nil, fmt.Errorf(
			"failed to create user: %w",
			err,
		)
	}

	return user, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrUserValidation,
		)
	}

	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, fmt.Errorf(
			"failed to get user: %w",
			err,
		)
	}

	return user, nil
}

func (s *Service) Login(ctx context.Context, input *LoginUser) (*User, error) {
	if input == nil {
		return nil, fmt.Errorf(
			"%w: login data is required",
			ErrUserValidation,
		)
	}

	email := strings.TrimSpace(input.Email)

	if email == "" {
	return nil, fmt.Errorf(
		"%w: email is required",
		ErrUserValidation,
	)
}

if input.Password == "" {
	return nil, fmt.Errorf(
		"%w: password is required",
		ErrUserValidation,
	)
}

	if utf8.RuneCountInString(email) > maxEmailCharacters {
		return nil, ErrInvalidCredentials
	}

	parsedEmail, err := mail.ParseAddress(email)
	if err != nil ||
		parsedEmail.Name != "" ||
		parsedEmail.Address != email {
		return nil, ErrInvalidCredentials
	}

	if len(input.Password) > maxPasswordBytes {
		return nil, ErrInvalidCredentials
	}

	user, err := s.repository.GetByEmail(
		ctx,
		email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf(
			"failed to get user by email: %w",
			err,
		)
	}

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(input.Password),
	)

	if err != nil {
		if errors.Is(
			err,
			bcrypt.ErrMismatchedHashAndPassword,
		) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf(
			"failed to compare password hash: %w",
			err,
		)
	}

	return user, nil
}
