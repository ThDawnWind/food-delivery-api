package address

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type RepositoryInterface interface {
	Create(ctx context.Context, input *CreateAddress) (*Address, error)
	GetByID(ctx context.Context, addressID int64, userID int64) (*Address, error)
	ListByUser(ctx context.Context, userID int64) ([]Address, error)
	Update(ctx context.Context, addressID int64, userID int64, input *UpdateAddress) (*Address, error)
	Delete(ctx context.Context, addressID int64, userID int64) error
}

type Service struct {
	repository RepositoryInterface
}

func NewService(repository RepositoryInterface) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, input *CreateAddress) (*Address, error) {
	if input == nil {
		return nil, fmt.Errorf(
			"%w: address data is required",
			ErrAddressValidation,
		)
	}

	if input.UserID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrAddressValidation,
		)
	}

	input.City = strings.TrimSpace(input.City)
	input.Street = strings.TrimSpace(input.Street)
	input.HouseNumber = strings.TrimSpace(
		input.HouseNumber,
	)

	if input.City == "" {
		return nil, fmt.Errorf(
			"%w: city is required",
			ErrAddressValidation,
		)
	}

	if input.Street == "" {
		return nil, fmt.Errorf(
			"%w: street is required",
			ErrAddressValidation,
		)
	}

	if input.HouseNumber == "" {
		return nil, fmt.Errorf(
			"%w: house number is required",
			ErrAddressValidation,
		)
	}

	address, err := s.repository.Create(
		ctx,
		input,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create address: %w",
			err,
		)
	}

	return address, nil
}

func (s *Service) GetByID(ctx context.Context, addressID int64, userID int64) (*Address, error) {
	if addressID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid address id",
			ErrAddressValidation,
		)
	}

	if userID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrAddressValidation,
		)
	}

	address, err := s.repository.GetByID(
		ctx,
		addressID,
		userID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAddressNotFound
		}

		return nil, fmt.Errorf(
			"failed to get address: %w",
			err,
		)
	}

	return address, nil
}

func (s *Service) ListByUser(ctx context.Context, userID int64) ([]Address, error) {
	if userID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrAddressValidation,
		)
	}

	addresses, err := s.repository.ListByUser(
		ctx,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list addresses: %w",
			err,
		)
	}

	return addresses, nil
}

func (s *Service) Delete(ctx context.Context, addressID int64, userID int64) error {
	if addressID <= 0 {
		return fmt.Errorf(
			"%w: invalid address id",
			ErrAddressValidation,
		)
	}

	if userID <= 0 {
		return fmt.Errorf(
			"%w: invalid user id",
			ErrAddressValidation,
		)
	}

	err := s.repository.Delete(
		ctx,
		addressID,
		userID,
	)
	if err != nil {
		if errors.Is(err, ErrAddressNotFound) {
			return ErrAddressNotFound
		}

		return fmt.Errorf(
			"failed to delete address: %w",
			err,
		)
	}

	return nil
}

func (s *Service) Update(ctx context.Context, addressID int64, userID int64, input *UpdateAddress) (*Address, error) {
	if addressID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid address id",
			ErrAddressValidation,
		)
	}

	if userID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrAddressValidation,
		)
	}

	if input == nil {
		return nil, fmt.Errorf(
			"%w: address data is required",
			ErrAddressValidation,
		)
	}

	if input.City != nil {
		city := strings.TrimSpace(*input.City)

		if city == "" {
			return nil, fmt.Errorf(
				"%w: city cannot be empty",
				ErrAddressValidation,
			)
		}

		input.City = &city
	}

	if input.Street != nil {
		street := strings.TrimSpace(*input.Street)

		if street == "" {
			return nil, fmt.Errorf(
				"%w: street cannot be empty",
				ErrAddressValidation,
			)
		}

		input.Street = &street
	}

	if input.HouseNumber != nil {
		houseNumber := strings.TrimSpace(
			*input.HouseNumber,
		)

		if houseNumber == "" {
			return nil, fmt.Errorf(
				"%w: house number cannot be empty",
				ErrAddressValidation,
			)
		}

		input.HouseNumber = &houseNumber
	}

	address, err := s.repository.Update(
		ctx,
		addressID,
		userID,
		input,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAddressNotFound
		}

		return nil, fmt.Errorf(
			"failed to update address: %w",
			err,
		)
	}

	return address, nil
}
