package order

import (
	"context"
	"errors"
	"testing"

	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/jackc/pgx/v5"
)

type fakeProductReader struct {
	products map[int64]*product.Product
	err      error
}

type fakeOrderRepository struct {
	order  *Order
	orders []Order
	err    error

	userID int64
	limit  int
	offset int

	updatedOrderID int64
	updatedStatus  Status

	listAllLimit  int
	listAllOffset int
	listAllOrders []*Order
	listAllErr    error
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

func (f *fakeOrderRepository) Create(ctx context.Context, order *Order) (*Order, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.order = order

	return order, nil
}

func (f *fakeOrderRepository) GetByID(ctx context.Context, id int64) (*Order, error) {
	if f.err != nil {
		return nil, f.err
	}

	if f.order == nil || f.order.ID != id {
		return nil, pgx.ErrNoRows
	}

	return f.order, nil
}

func (f *fakeOrderRepository) ListByUser(ctx context.Context, userID int64, limit int, offset int) ([]Order, error) {
	if f.err != nil {
		return nil, f.err
	}

	f.userID = userID
	f.limit = limit
	f.offset = offset

	return f.orders, nil
}

func (f *fakeOrderRepository) UpdateStatus(ctx context.Context, id int64, status Status) error {
	if f.err != nil {
		return f.err
	}

	f.updatedOrderID = id
	f.updatedStatus = status

	return nil
}

func (f *fakeOrderRepository) ListAll(ctx context.Context, limit int, offset int) ([]*Order, error) {
	f.listAllLimit = limit
	f.listAllOffset = offset

	if f.listAllErr != nil {
		return nil, f.listAllErr
	}

	return f.listAllOrders, nil
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

	repository := &fakeOrderRepository{}

	service := NewService(
		products,
		repository,
	)

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

	if repository.order == nil {
		t.Fatal("expected order to be passed to repository")
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

	if repository.order.TotalPrice != expectedTotal {
		t.Errorf(
			"expected repository total price %d, got %d",
			expectedTotal,
			repository.order.TotalPrice,
		)
	}

	if repository.order.Status != StatusNew {
		t.Errorf(
			"expected repository status %q, got %q",
			StatusNew,
			repository.order.Status,
		)
	}

	if len(repository.order.Items) != 2 {
		t.Fatalf(
			"expected repository to receive %d items, got %d",
			2,
			len(repository.order.Items),
		)
	}

	if repository.order.Items[0].NameSnapshot != "Pepperoni" {
		t.Errorf(
			"expected name snapshot %q, got %q",
			"Pepperoni",
			repository.order.Items[0].NameSnapshot,
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

			repository := &fakeOrderRepository{}

			service := NewService(
				products,
				repository,
			)

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

			if repository.order != nil {
				t.Fatal(
					"repository must not be called on validation error",
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

	repository := &fakeOrderRepository{}

	service := NewService(
		products,
		repository,
	)

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

	repository := &fakeOrderRepository{}

	service := NewService(
		products,
		repository,
	)

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

	if !errors.Is(err, productErr) {
		t.Fatalf(
			"expected product reader error, got %v",
			err,
		)
	}

	if errors.Is(err, ErrOrderValidation) {
		t.Fatal(
			"product reader error must not become ErrOrderValidation",
		)
	}
}

func TestService_Create_ProductNotFound(t *testing.T) {
	products := &fakeProductReader{
		products: map[int64]*product.Product{},
	}

	repository := &fakeOrderRepository{}

	service := NewService(
		products,
		repository,
	)

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

func TestService_Create_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

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

	repository := &fakeOrderRepository{
		err: repositoryErr,
	}

	service := NewService(
		products,
		repository,
	)

	input := &CreateOrder{
		UserID:          1,
		DeliveryAddress: "Test street 1",
		Items: []CreateItem{
			{
				ProductID: 1,
				Quantity:  2,
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

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_GetByID(t *testing.T) {
	expectedOrder := &Order{
		ID:              10,
		UserID:          5,
		Status:          StatusNew,
		TotalPrice:      159700,
		DeliveryAddress: "Test street 1",
		Items: []OrderItem{
			{
				ID:            1,
				OrderID:       10,
				ProductID:     1,
				NameSnapshot:  "Pepperoni",
				PriceSnapshot: 59900,
				Quantity:      2,
			},
		},
	}

	repository := &fakeOrderRepository{
		order: expectedOrder,
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	order, err := service.GetByID(
		context.Background(),
		10,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if order == nil {
		t.Fatal("expected order, got nil")
	}

	if order.ID != 10 {
		t.Errorf(
			"expected order ID %d, got %d",
			10,
			order.ID,
		)
	}

	if order.TotalPrice != 159700 {
		t.Errorf(
			"expected total price %d, got %d",
			159700,
			order.TotalPrice,
		)
	}

	if len(order.Items) != 1 {
		t.Fatalf(
			"expected %d items, got %d",
			1,
			len(order.Items),
		)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	service := NewService(
		&fakeProductReader{},
		&fakeOrderRepository{},
	)

	order, err := service.GetByID(
		context.Background(),
		0,
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

func TestService_GetByID_NotFound(t *testing.T) {
	service := NewService(
		&fakeProductReader{},
		&fakeOrderRepository{},
	)

	order, err := service.GetByID(
		context.Background(),
		999,
	)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf(
			"expected ErrOrderNotFound, got %v",
			err,
		)
	}
}

func TestService_GetByID_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	service := NewService(
		&fakeProductReader{},
		&fakeOrderRepository{
			err: repositoryErr,
		},
	)

	order, err := service.GetByID(
		context.Background(),
		1,
	)

	if order != nil {
		t.Fatalf(
			"expected nil order, got %+v",
			order,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}

	if errors.Is(err, ErrOrderNotFound) {
		t.Fatal(
			"repository error must not become ErrOrderNotFound",
		)
	}
}

func TestService_ListByUser(t *testing.T) {
	repository := &fakeOrderRepository{
		orders: []Order{
			{
				ID:         2,
				UserID:     10,
				Status:     StatusNew,
				TotalPrice: 59900,
			},
			{
				ID:         1,
				UserID:     10,
				Status:     StatusCompleted,
				TotalPrice: 39900,
			},
		},
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	orders, err := service.ListByUser(
		context.Background(),
		10,
		50,
		5,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(orders) != 2 {
		t.Fatalf(
			"expected %d orders, got %d",
			2,
			len(orders),
		)
	}

	if repository.userID != 10 {
		t.Errorf(
			"expected user ID %d, got %d",
			10,
			repository.userID,
		)
	}

	if repository.limit != 50 {
		t.Errorf(
			"expected limit %d, got %d",
			50,
			repository.limit,
		)
	}

	if repository.offset != 5 {
		t.Errorf(
			"expected offset %d, got %d",
			5,
			repository.offset,
		)
	}
}

func TestService_ListByUser_DefaultLimit(t *testing.T) {
	repository := &fakeOrderRepository{
		orders: []Order{},
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	_, err := service.ListByUser(
		context.Background(),
		1,
		0,
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repository.limit != 20 {
		t.Errorf(
			"expected default limit %d, got %d",
			20,
			repository.limit,
		)
	}
}

func TestService_ListByUser_MaxLimit(t *testing.T) {
	repository := &fakeOrderRepository{
		orders: []Order{},
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	_, err := service.ListByUser(
		context.Background(),
		1,
		500,
		0,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repository.limit != 100 {
		t.Errorf(
			"expected max limit %d, got %d",
			100,
			repository.limit,
		)
	}
}

func TestService_ListByUser_Validation(t *testing.T) {
	tests := []struct {
		name   string
		userID int64
		limit  int
		offset int
	}{
		{
			name:   "invalid user id",
			userID: 0,
			limit:  20,
			offset: 0,
		},
		{
			name:   "negative offset",
			userID: 1,
			limit:  20,
			offset: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeOrderRepository{}

			service := NewService(
				&fakeProductReader{},
				repository,
			)

			orders, err := service.ListByUser(
				context.Background(),
				tt.userID,
				tt.limit,
				tt.offset,
			)

			if orders != nil {
				t.Fatalf(
					"expected nil orders, got %+v",
					orders,
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

func TestService_ListByUser_RepositoryError(t *testing.T) {
	repositoryErr := errors.New("database unavailable")

	repository := &fakeOrderRepository{
		err: repositoryErr,
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	orders, err := service.ListByUser(
		context.Background(),
		1,
		20,
		0,
	)

	if orders != nil {
		t.Fatalf(
			"expected nil orders, got %+v",
			orders,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}

func TestService_UpdateStatus(t *testing.T) {
	repository := &fakeOrderRepository{
		order: &Order{
			ID:     10,
			UserID: 5,
			Status: StatusNew,
		},
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	err := service.UpdateStatus(
		context.Background(),
		10,
		StatusConfirmed,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repository.updatedOrderID != 10 {
		t.Errorf(
			"expected updated order ID %d, got %d",
			10,
			repository.updatedOrderID,
		)
	}

	if repository.updatedStatus != StatusConfirmed {
		t.Errorf(
			"expected status %q, got %q",
			StatusConfirmed,
			repository.updatedStatus,
		)
	}
}

func TestService_UpdateStatus_InvalidTransition(t *testing.T) {
	repository := &fakeOrderRepository{
		order: &Order{
			ID:     10,
			Status: StatusNew,
		},
	}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	err := service.UpdateStatus(
		context.Background(),
		10,
		StatusCompleted,
	)

	if !errors.Is(err, ErrOrderValidation) {
		t.Fatalf(
			"expected ErrOrderValidation, got %v",
			err,
		)
	}

	if repository.updatedOrderID != 0 {
		t.Fatal(
			"repository UpdateStatus must not be called for invalid transition",
		)
	}
}

func TestService_UpdateStatus_InvalidID(t *testing.T) {
	repository := &fakeOrderRepository{}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	err := service.UpdateStatus(
		context.Background(),
		0,
		StatusConfirmed,
	)

	if !errors.Is(err, ErrOrderValidation) {
		t.Fatalf(
			"expected ErrOrderValidation, got %v",
			err,
		)
	}

	if repository.updatedOrderID != 0 {
		t.Fatal(
			"repository must not be called for invalid order ID",
		)
	}
}

func TestService_UpdateStatus_NotFound(t *testing.T) {
	repository := &fakeOrderRepository{}

	service := NewService(
		&fakeProductReader{},
		repository,
	)

	err := service.UpdateStatus(
		context.Background(),
		999,
		StatusConfirmed,
	)

	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf(
			"expected ErrOrderNotFound, got %v",
			err,
		)
	}
}

func TestService_ListAll(t *testing.T) {
	repository := &fakeOrderRepository{
		listAllOrders: []*Order{
			{
				ID:     1,
				UserID: 10,
				Status: "new",
			},
			{
				ID:     2,
				UserID: 20,
				Status: "confirmed",
			},
		},
	}

	service := NewService(
		nil,
		repository,
	)

	orders, err := service.ListAll(
		context.Background(),
		20,
		0,
	)
	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if len(orders) != 2 {
		t.Fatalf(
			"expected %d orders, got %d",
			2,
			len(orders),
		)
	}

	if repository.listAllLimit != 20 {
		t.Errorf(
			"expected limit %d, got %d",
			20,
			repository.listAllLimit,
		)
	}

	if repository.listAllOffset != 0 {
		t.Errorf(
			"expected offset %d, got %d",
			0,
			repository.listAllOffset,
		)
	}
}

func TestService_ListAll_Validation(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{
			name:   "zero limit",
			limit:  0,
			offset: 0,
		},
		{
			name:   "negative limit",
			limit:  -1,
			offset: 0,
		},
		{
			name:   "limit too large",
			limit:  101,
			offset: 0,
		},
		{
			name:   "negative offset",
			limit:  20,
			offset: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repository := &fakeOrderRepository{}

			service := NewService(
				nil,
				repository,
			)

			orders, err := service.ListAll(
				context.Background(),
				tt.limit,
				tt.offset,
			)

			if orders != nil {
				t.Fatalf(
					"expected nil orders, got %+v",
					orders,
				)
			}

			if !errors.Is(
				err,
				ErrOrderValidation,
			) {
				t.Fatalf(
					"expected ErrOrderValidation, got %v",
					err,
				)
			}
		})
	}
}

func TestService_ListAll_RepositoryError(t *testing.T) {
	repositoryErr := errors.New(
		"database unavailable",
	)

	repository := &fakeOrderRepository{
		listAllErr: repositoryErr,
	}

	service := NewService(
		nil,
		repository,
	)

	orders, err := service.ListAll(
		context.Background(),
		20,
		0,
	)

	if orders != nil {
		t.Fatalf(
			"expected nil orders, got %+v",
			orders,
		)
	}

	if !errors.Is(err, repositoryErr) {
		t.Fatalf(
			"expected repository error, got %v",
			err,
		)
	}
}
