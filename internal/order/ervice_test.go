package order

import (
	"context"
	"errors"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/product"
)

type fakeProductReader struct {
	products map[int64]*product.Product
	err      error
}

func (f *fakeProductReader) GetByID(ctx context.Context, id int64) (*product.Product, error) {
	if f.err != nil {
		return nil, f.err
	}

	productData, ok := f.products[id]
	if !ok {
		return nil, product.ErrProductNotFound
	}

	return productData, nil
}

func TestService_Create(t *testing.T) {
	products := &fakeProductReader{
		products: map[int64]*product.Product{
			1: {
				ID:       1,
				Name:     "Pepperoni",
				Price:    59900,
				IsActive: true,
			},
			2: {
				ID:       2,
				Name:     "Burger",
				Price:    39900,
				IsActive: true,
			},
		},
	}

	service := NewService(products)

	input := &CreateOrder{
		UserID:          10,
		DeliveryAddress: "  Test street 1  ",
		Items: []CreateItem{
			{
				ProductID: 1,
				Quantity:  2,
			},
			{
				ProductID: 2,
				Quantity:  1,
			},
		},
	}

	order, err := service.Create(
		context.Background(),
		input,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order == nil {
		t.Fatal("expected order, got nil")
	}

	if order.UserID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			order.UserID,
		)
	}

	if order.Status != StatusNew {
		t.Errorf(
			"expected status %q, got %q",
			StatusNew,
			order.Status,
		)
	}

	if order.DeliveryAddress != "Test street 1" {
		t.Errorf(
			"expected trimmed address %q, got %q",
			"Test street 1",
			order.DeliveryAddress,
		)
	}

	expectedTotal := int64(59900*2 + 39900)

	if order.TotalPrice != expectedTotal {
		t.Errorf(
			"expected total price %d, got %d",
			expectedTotal,
			order.TotalPrice,
		)
	}

	if len(order.Items) != 2 {
		t.Fatalf(
			"expected %d items, got %d",
			2,
			len(order.Items),
		)
	}

	if order.Items[0].ProductID != 1 {
		t.Errorf(
			"expected first product ID %d, got %d",
			1,
			order.Items[0].ProductID,
		)
	}

	if order.Items[0].NameSnapshot != "Pepperoni" {
		t.Errorf(
			"expected name snapshot %q, got %q",
			"Pepperoni",
			order.Items[0].NameSnapshot,
		)
	}

	if order.Items[0].PriceSnapshot != 59900 {
		t.Errorf(
			"expected price snapshot %d, got %d",
			59900,
			order.Items[0].PriceSnapshot,
		)
	}

	if order.Items[0].Quantity != 2 {
		t.Errorf(
			"expected quantity %d, got %d",
			2,
			order.Items[0].Quantity,
		)
	}

	if order.Items[1].NameSnapshot != "Burger" {
		t.Errorf(
			"expected name snapshot %q, got %q",
			"Burger",
			order.Items[1].NameSnapshot,
		)
	}

	if order.Items[1].PriceSnapshot != 39900 {
		t.Errorf(
			"expected price snapshot %d, got %d",
			39900,
			order.Items[1].PriceSnapshot,
		)
	}
}

func TestService_Create_Validation(t *testing.T) {
	tests := []struct {
		name  string
		input *CreateOrder
	}{
		{
			name:  "nil order",
			input: nil,
		},
		{
			name: "invalid user id",
			input: &CreateOrder{
				UserID:          0,
				DeliveryAddress: "Test street 1",
				Items: []CreateItem{
					{
						ProductID: 1,
						Quantity:  1,
					},
				},
			},
		},
		{
			name: "empty delivery address",
			input: &CreateOrder{
				UserID:          1,
				DeliveryAddress: "   ",
				Items: []CreateItem{
					{
						ProductID: 1,
						Quantity:  1,
					},
				},
			},
		},
		{
			name: "empty items",
			input: &CreateOrder{
				UserID:          1,
				DeliveryAddress: "Test street 1",
				Items:           []CreateItem{},
			},
		},
		{
			name: "invalid product id",
			input: &CreateOrder{
				UserID:          1,
				DeliveryAddress: "Test street 1",
				Items: []CreateItem{
					{
						ProductID: 0,
						Quantity:  1,
					},
				},
			},
		},
		{
			name: "invalid quantity",
			input: &CreateOrder{
				UserID:          1,
				DeliveryAddress: "Test street 1",
				Items: []CreateItem{
					{
						ProductID: 1,
						Quantity:  0,
					},
				},
			},
		},
		{
			name: "duplicate product",
			input: &CreateOrder{
				UserID:          1,
				DeliveryAddress: "Test street 1",
				Items: []CreateItem{
					{
						ProductID: 1,
						Quantity:  1,
					},
					{
						ProductID: 1,
						Quantity:  2,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			products := &fakeProductReader{
				products: map[int64]*product.Product{
					1: {
						ID:       1,
						Name:     "Pepperoni",
						Price:    59900,
						IsActive: true,
					},
				},
			}

			service := NewService(products)

			order, err := service.Create(
				context.Background(),
				tt.input,
			)

			if order != nil {
				t.Fatalf(
					"expected nil order, got %+v",
					order,
				)
			}

			if !errors.Is(err, ErrOrderValidation) {
				t.Fatalf(
					"expected ErrOrderValidation, got %v",
					err,
				)
			}
		})
	}
}

func TestService_Create_InactiveProduct(t *testing.T) {
	products := &fakeProductReader{
		products: map[int64]*product.Product{
			1: {
				ID:       1,
				Name:     "Pepperoni",
				Price:    59900,
				IsActive: false,
			},
		},
	}

	service := NewService(products)

	input := &CreateOrder{
		UserID:          1,
		DeliveryAddress: "Test street 1",
		Items: []CreateItem{
			{
				ProductID: 1,
				Quantity:  1,
			},
		},
	}

	order, err := service.Create(
		context.Background(),
		input,
	)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, ErrOrderValidation) {
		t.Fatalf(
			"expected ErrOrderValidation, got %v",
			err,
		)
	}
}

func TestService_Create_ProductReaderError(t *testing.T) {
	productErr := errors.New("product service unavailable")

	products := &fakeProductReader{
		err: productErr,
	}

	service := NewService(products)

	input := &CreateOrder{
		UserID:          1,
		DeliveryAddress: "Test street 1",
		Items: []CreateItem{
			{
				ProductID: 1,
				Quantity:  1,
			},
		},
	}

	order, err := service.Create(
		context.Background(),
		input,
	)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, ErrOrderValidation) {
		t.Fatalf(
			"expected ErrOrderValidation, got %v",
			err,
		)
	}
}

func TestService_Create_ProductNotFound(t *testing.T) {
	products := &fakeProductReader{
		products: map[int64]*product.Product{},
	}

	service := NewService(products)

	input := &CreateOrder{
		UserID:          1,
		DeliveryAddress: "Test street 1",
		Items: []CreateItem{
			{
				ProductID: 999,
				Quantity:  1,
			},
		},
	}

	order, err := service.Create(
		context.Background(),
		input,
	)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, ErrOrderValidation) {
		t.Fatalf(
			"expected ErrOrderValidation, got %v",
			err,
		)
	}
}
