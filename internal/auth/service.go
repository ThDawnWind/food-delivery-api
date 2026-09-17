package auth

import (
	"context"
	"fmt"

	"github.com/ThDawnWind/food-delivery-api/internal/user"
)

type UserAuthenticator interface {
	Register(ctx context.Context, input *user.RegisterUser) (*user.User, error)
	Login(ctx context.Context, input *user.LoginUser) (*user.User, error)
}

type TokenGenerator interface {
	Generate(userID int64, role string) (string, error)
}

type Service struct {
	users  UserAuthenticator
	tokens TokenGenerator
}

func NewService(users UserAuthenticator, tokens TokenGenerator) *Service {
	return &Service{
		users:  users,
		tokens: tokens,
	}
}

type LoginResult struct {
	AccessToken string     `json:"access_token"`
	User        *user.User `json:"user"`
}

func (s *Service) Login(ctx context.Context, input *user.LoginUser) (*LoginResult, error) {
	userData, err := s.users.Login(
		ctx,
		input,
	)
	if err != nil {
		return nil, err
	}

	token, err := s.tokens.Generate(
		userData.ID,
		string(userData.Role),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate access token: %w",
			err,
		)
	}

	return &LoginResult{
		AccessToken: token,
		User:        userData,
	}, nil
}

func (s *Service) Register(ctx context.Context, input *user.RegisterUser) (*user.User, error) {
	userData, err := s.Register(
		ctx,
		input,
	)

	if err != nil {
		return nil, err
	}

	return userData, nil
}
