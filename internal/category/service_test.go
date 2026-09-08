package category

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	category   *Category
	categories []Category
	err        error
}

func (f *fakeRepository) GetByID(ctx context.Context, id int64) (*Category, error) {
	return f.category, f.err
}

func (f *fakeRepository) List(ctx context.Context) ([]Category, error) {
	return f.categories, f.err
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

func TestService_List(t *testing.T) {
	expected := []Category{
		{ID: 1, Name: "Pizza", Slug: "pizza"},
		{ID: 2, Name: "Sushi", Slug: "sushi"},
	}

	repo := &fakeRepository{
		categories: expected,
	}

	service := NewService(repo)

	categories, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if categories == nil {
		t.Errorf("expected categories, got nil")
	}

	if len(categories) != len(expected) {
		t.Fatalf(
			"expected %d categories, got %d",
			len(expected),
			len(categories),
		)
	}

	for i := range expected {
		if categories[i].ID != expected[i].ID {
			t.Errorf(
				"category %d: expected ID %d, got %d",
				i,
				expected[i].ID,
				categories[i].ID,
			)
		}

		if categories[i].Name != expected[i].Name {
			t.Errorf(
				"category %d: expected name %q, got %q",
				i,
				expected[i].Name,
				categories[i].Name,
			)
		}

		if categories[i].Slug != expected[i].Slug {
			t.Errorf(
				"category %d: expected slug %q, got %q",
				i,
				expected[i].Slug,
				categories[i].Slug,
			)
		}
	}
}

func TestService_List_Error(t *testing.T) {
	expectedErr := errors.New("repository failure")

	repo := &fakeRepository{
		err: expectedErr,
	}

	service := NewService(repo)

	categories, err := service.List(context.Background())

	if categories != nil {
		t.Fatalf("expected nil categories, got %+v", categories)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
}
