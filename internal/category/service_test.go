package category

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	category *Category
	err      error
}

func (f *fakeRepository) GetByID(ctx context.Context, id int64) (*Category, error) {
	return f.category, f.err
}

func TestService_GetByID(t *testing.T) {
	expected := &Category{
		ID:   1,
		Name: "Pizza",
		Slug: "pizza",
	}

	repo := &fakeRepository{
		category: expected,
	}

	service := NewService(repo)

	category, err := service.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if category == nil {
		t.Fatal("expected category, got nil")
	}

	if category.ID != expected.ID {
		t.Errorf("expected ID %d, got %d", expected.ID, category.ID)
	}

	if category.Name != expected.Name {
		t.Errorf("expected name %q, got %q", expected.Name, category.Name)
	}

	if category.Slug != expected.Slug {
		t.Errorf("expected slug %q, got %q", expected.Slug, category.Slug)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repo := &fakeRepository{
		err: pgx.ErrNoRows,
	}

	service := NewService(repo)

	category, err := service.GetByID(context.Background(), 999)

	if category != nil {
		t.Fatalf("expected nil category, got %+v", category)
	}

	if !errors.Is(err, ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound, got %v", err)
	}
}
