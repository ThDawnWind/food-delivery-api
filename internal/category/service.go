package category

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCategoryNotFound   = errors.New("category not found")
	ErrCategoryValidation = errors.New("category validation error ")
)

type RepositoryInterface interface {
	GetByID(ctx context.Context, id int64) (*Category, error)
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, name, slug string) (*Category, error)
	Update(ctx context.Context, category *Category) error
	// Delete(ctx context.Context, id int64) error
}

type Service struct {
	repository RepositoryInterface
}

func NewService(repository RepositoryInterface) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Category, error) {
	category, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCategoryNotFound
		}
		return nil, fmt.Errorf("get category: %w", err)
	}

	return category, nil
}

func (s *Service) List(ctx context.Context) ([]Category, error) {
	categories, err := s.repository.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}

	return categories, nil
}

func (s *Service) Create(ctx context.Context, name, slug string) (*Category, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: name is required", ErrCategoryValidation)
	}

	if strings.TrimSpace(slug) == "" {
		return nil, fmt.Errorf("%w: slug is required", ErrCategoryValidation)
	}

	category, err := s.repository.Create(ctx, name, slug)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return category, nil
}

func (s *Service) Update(ctx context.Context, category *Category) error {
	if category == nil {
		return fmt.Errorf("%w: category is required", ErrCategoryValidation)
	}

	category.Name = strings.TrimSpace(category.Name)
	category.Slug = strings.TrimSpace(category.Slug)

	if category.Name == "" {
		return fmt.Errorf("%w: name is required", ErrCategoryValidation)
	}

	if category.Slug == "" {
		return fmt.Errorf("%w: slug is required", ErrCategoryValidation)
	}

	err := s.repository.Update(ctx, category)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCategoryNotFound
		}

		return fmt.Errorf("update category: %w", err)
	}

	return nil
}
