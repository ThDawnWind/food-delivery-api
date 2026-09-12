package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

type RepositoryInterface interface {
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context) ([]Product, error)
	Create(ctx context.Context, product *Product) (*Product, error)
	Update(ctx context.Context, product *Product) (*Product, error)
	Deactivate(ctx context.Context, id int64) error
}

type Service struct {
	repository RepositoryInterface
}

func NewService(repository RepositoryInterface) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Product, error) {
	if id <= 0 {
		return nil, ErrProductValidation
	}

	product, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}

		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	return product, nil
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	products, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	return products, nil
}

func (s *Service) Create(ctx context.Context, product *Product) (*Product, error) {
	if product == nil {
		return nil, fmt.Errorf(
			"%w: product is nil",
			ErrProductValidation,
		)
	}

	product.Name = strings.TrimSpace(product.Name)

	if product.Name == "" {
		return nil, fmt.Errorf(
			"%w: name is required",
			ErrProductValidation,
		)
	}

	if product.Price <= 0 {
		return nil, fmt.Errorf(
			"%w: price must be greater than zero",
			ErrProductValidation,
		)
	}

	if product.Weight <= 0 {
		return nil, fmt.Errorf(
			"%w: weight must be greater than zero",
			ErrProductValidation,
		)
	}

	if product.CategoryID <= 0 {
		return nil, fmt.Errorf(
			"%w: category id must be greater than zero",
			ErrProductValidation,
		)
	}

	createdProduct, err := s.repository.Create(ctx, product)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create product: %w",
			err,
		)
	}

	return createdProduct, nil
}

func (s *Service) Update(ctx context.Context, product *Product) (*Product, error) {
	if product == nil {
		return nil, fmt.Errorf(
			"%w: product is nil",
			ErrProductValidation,
		)
	}

	if product.ID <= 0 {
		return nil, fmt.Errorf(
			"%w: product id must be greater than zero",
			ErrProductValidation,
		)
	}

	product.Name = strings.TrimSpace(product.Name)

	if product.Name == "" {
		return nil, fmt.Errorf(
			"%w: name is required",
			ErrProductValidation,
		)
	}

	if product.Price <= 0 {
		return nil, fmt.Errorf(
			"%w: price must be greater than zero",
			ErrProductValidation,
		)
	}

	if product.Weight <= 0 {
		return nil, fmt.Errorf(
			"%w: weight must be greater than zero",
			ErrProductValidation,
		)
	}

	if product.CategoryID <= 0 {
		return nil, fmt.Errorf(
			"%w: category id must be greater than zero",
			ErrProductValidation,
		)
	}

	updatedProduct, err := s.repository.Update(ctx, product)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProductNotFound
		}

		return nil, fmt.Errorf(
			"failed to update product: %w",
			err,
		)
	}

	return updatedProduct, nil
}

func (s *Service) Deactivate(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrProductValidation
	}

	err := s.repository.Deactivate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrProductNotFound
		}

		return fmt.Errorf(
			"failed to deactivate product: %w",
			err,
		)
	}

	return nil
}
