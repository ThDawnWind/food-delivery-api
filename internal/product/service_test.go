package product

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRepository struct {
	product       *Product
	products      []Product
	err           error
	deactivatedID int64
}

func (f *fakeRepository) GetByID(ctx context.Context, id int64) (*Product, error) {
	return f.product, f.err
}

func (f *fakeRepository) List(ctx context.Context) ([]Product, error) {
	return f.products, f.err
}

func (f *fakeRepository) Create(ctx context.Context, product *Product) (*Product, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.product = product
	return product, nil
}
func (f *fakeRepository) Update(ctx context.Context, product *Product) (*Product, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.product = product
	return product, nil
}

func (f *fakeRepository) Deactivate(ctx context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}

	f.deactivatedID = id

	return nil
}

func TestService_GetByID(t *testing.T) {
	expected := &Product{
		ID:         1,
		Name:       "Pepperoni",
		Price:      59900,
		Weight:     450,
		CategoryID: 1,
		IsActive:   true,
	}

	repository := &fakeRepository{
		product: expected,
	}

	service := NewService(repository)

	product, err := service.GetByID(
		context.Background(),
		1,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if product == nil {
		t.Fatal("expected product, got nil")
	}

	if product.ID != expected.ID {
		t.Errorf(
			"expected ID %d, got %d",
			expected.ID,
			product.ID,
		)
	}

	if product.Name != expected.Name {
		t.Errorf(
			"expected name %q, got %q",
			expected.Name,
			product.Name,
		)
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	repository := &fakeRepository{
		err: pgx.ErrNoRows,
	}

	service := NewService(repository)

	product, err := service.GetByID(
		context.Background(),
		999,
	)

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}

	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf(
			"expected ErrProductNotFound, got %v",
			err,
		)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	repository := &fakeRepository{}

	service := NewService(repository)

	product, err := service.GetByID(
		context.Background(),
		0,
	)

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}

	if !errors.Is(err, ErrProductValidation) {
		t.Fatalf(
			"expected ErrProductValidation, got %v",
			err,
		)
	}
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		err: repositoryErr,
	}

	service := NewService(repository)

	product, err := service.GetByID(
		context.Background(),
		1,
	)

	if product != nil {
		t.Fatalf("expected nil product, got %+v", product)
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}

	if errors.Is(err, ErrProductNotFound) {
		t.Fatal("repository error must not become ErrProductNotFound")
	}
}

func TestService_List(t *testing.T) {
	expected := []Product{
		{
			ID:         1,
			Name:       "Pepperoni",
			Price:      59900,
			Weight:     450,
			CategoryID: 1,
			IsActive:   true,
		},
		{
			ID:         2,
			Name:       "Burger",
			Price:      39900,
			Weight:     300,
			CategoryID: 1,
			IsActive:   true,
		},
	}

	repository := &fakeRepository{
		products: expected,
	}

	service := NewService(repository)

	products, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(products) != 2 {
		t.Fatalf(
			"expected %d products, got %d",
			2,
			len(products),
		)
	}

	if products[0].ID != expected[0].ID {
		t.Errorf(
			"expected first product ID %d, got %d",
			expected[0].ID,
			products[0].ID,
		)
	}

	if products[1].Name != expected[1].Name {
		t.Errorf(
			"expected second product name %q, got %q",
			expected[1].Name,
			products[1].Name,
		)
	}
}

func TestService_List_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		err: repositoryErr,
	}

	service := NewService(repository)

	products, err := service.List(context.Background())

	if products != nil {
		t.Fatalf(
			"expected nil products, got %+v",
			products,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Create(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	product := &Product{
		Name:       "  Pepperoni  ",
		Price:      59900,
		Weight:     450,
		CategoryID: 1,
		IsActive:   true,
	}

	createdProduct, err := service.Create(
		context.Background(),
		product,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if createdProduct == nil {
		t.Fatal("expected product, got nil")
	}

	if createdProduct.Name != "Pepperoni" {
		t.Errorf(
			"expected trimmed name %q, got %q",
			"Pepperoni",
			createdProduct.Name,
		)
	}

	if repository.product != product {
		t.Error("expected product to be passed to repository")
	}
}

func TestService_Create_Validation(t *testing.T) {
	tests := []struct {
		name    string
		product *Product
	}{
		{
			name:    "nil product",
			product: nil,
		},
		{
			name: "empty name",
			product: &Product{
				Name:       "   ",
				Price:      59900,
				Weight:     450,
				CategoryID: 1,
			},
		},
		{
			name: "invalid price",
			product: &Product{
				Name:       "Pepperoni",
				Price:      0,
				Weight:     450,
				CategoryID: 1,
			},
		},
		{
			name: "invalid weight",
			product: &Product{
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     0,
				CategoryID: 1,
			},
		},
		{
			name: "invalid category id",
			product: &Product{
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     450,
				CategoryID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			product, err := service.Create(
				context.Background(),
				tt.product,
			)

			if product != nil {
				t.Fatalf(
					"expected nil product, got %+v",
					product,
				)
			}

			if !errors.Is(err, ErrProductValidation) {
				t.Fatalf(
					"expected ErrProductValidation, got %v",
					err,
				)
			}

			if repository.product != nil {
				t.Fatal(
					"repository must not be called on validation error",
				)
			}
		})
	}
}

func TestService_Create_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		err: repositoryErr,
	}

	service := NewService(repository)

	product := &Product{
		Name:       "Pepperoni",
		Price:      59900,
		Weight:     450,
		CategoryID: 1,
		IsActive:   true,
	}

	createdProduct, err := service.Create(
		context.Background(),
		product,
	)

	if createdProduct != nil {
		t.Fatalf(
			"expected nil product, got %+v",
			createdProduct,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_Update(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	product := &Product{
		ID:         1,
		Name:       "  Updated Pepperoni  ",
		Price:      69900,
		Weight:     500,
		CategoryID: 1,
		IsActive:   true,
	}

	updatedProduct, err := service.Update(
		context.Background(),
		product,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updatedProduct == nil {
		t.Fatal("expected product, got nil")
	}

	if updatedProduct.Name != "Updated Pepperoni" {
		t.Errorf(
			"expected trimmed name %q, got %q",
			"Updated Pepperoni",
			updatedProduct.Name,
		)
	}

	if repository.product != product {
		t.Error("expected product to be passed to repository")
	}
}

func TestService_Update_NotFound(t *testing.T) {
	repository := &fakeRepository{
		err: pgx.ErrNoRows,
	}

	service := NewService(repository)

	product := &Product{
		ID:         999,
		Name:       "Pepperoni",
		Price:      59900,
		Weight:     450,
		CategoryID: 1,
	}

	updatedProduct, err := service.Update(
		context.Background(),
		product,
	)

	if updatedProduct != nil {
		t.Fatalf(
			"expected nil product, got %+v",
			updatedProduct,
		)
	}

	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf(
			"expected ErrProductNotFound, got %v",
			err,
		)
	}
}

func TestService_Update_Validation(t *testing.T) {
	tests := []struct {
		name    string
		product *Product
	}{
		{
			name:    "nil product",
			product: nil,
		},
		{
			name: "invalid id",
			product: &Product{
				ID:         0,
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     450,
				CategoryID: 1,
			},
		},
		{
			name: "empty name",
			product: &Product{
				ID:         1,
				Name:       "   ",
				Price:      59900,
				Weight:     450,
				CategoryID: 1,
			},
		},
		{
			name: "invalid price",
			product: &Product{
				ID:         1,
				Name:       "Pepperoni",
				Price:      0,
				Weight:     450,
				CategoryID: 1,
			},
		},
		{
			name: "invalid weight",
			product: &Product{
				ID:         1,
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     0,
				CategoryID: 1,
			},
		},
		{
			name: "invalid category id",
			product: &Product{
				ID:         1,
				Name:       "Pepperoni",
				Price:      59900,
				Weight:     450,
				CategoryID: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeRepository{}
			service := NewService(repository)

			product, err := service.Update(
				context.Background(),
				tt.product,
			)

			if product != nil {
				t.Fatalf(
					"expected nil product, got %+v",
					product,
				)
			}

			if !errors.Is(err, ErrProductValidation) {
				t.Fatalf(
					"expected ErrProductValidation, got %v",
					err,
				)
			}

			if repository.product != nil {
				t.Fatal(
					"repository must not be called on validation error",
				)
			}
		})
	}
}

func TestService_Update_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		err: repositoryErr,
	}

	service := NewService(repository)

	product := &Product{
		ID:         1,
		Name:       "Pepperoni",
		Price:      59900,
		Weight:     450,
		CategoryID: 1,
	}

	updatedProduct, err := service.Update(
		context.Background(),
		product,
	)

	if updatedProduct != nil {
		t.Fatalf(
			"expected nil product, got %+v",
			updatedProduct,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}

	if errors.Is(err, ErrProductNotFound) {
		t.Fatal(
			"repository error must not become ErrProductNotFound",
		)
	}
}

func TestService_Deactivate(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	err := service.Deactivate(
		context.Background(),
		10,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repository.deactivatedID != 10 {
		t.Errorf(
			"expected deactivated ID %d, got %d",
			10,
			repository.deactivatedID,
		)
	}
}

func TestService_Deactivate_InvalidID(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)

	err := service.Deactivate(
		context.Background(),
		0,
	)

	if !errors.Is(err, ErrProductValidation) {
		t.Fatalf(
			"expected ErrProductValidation, got %v",
			err,
		)
	}

	if repository.deactivatedID != 0 {
		t.Fatal("repository must not be called on validation error")
	}
}

func TestService_Deactivate_NotFound(t *testing.T) {
	repository := &fakeRepository{
		err: pgx.ErrNoRows,
	}

	service := NewService(repository)

	err := service.Deactivate(
		context.Background(),
		999,
	)

	if !errors.Is(err, ErrProductNotFound) {
		t.Fatalf(
			"expected ErrProductNotFound, got %v",
			err,
		)
	}
}

func TestService_Deactivate_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeRepository{
		err: repositoryErr,
	}

	service := NewService(repository)

	err := service.Deactivate(
		context.Background(),
		1,
	)

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}

	if errors.Is(err, ErrProductNotFound) {
		t.Fatal(
			"repository error must not become ErrProductNotFound",
		)
	}
}
