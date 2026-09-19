package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/jackc/pgx/v5"
)

type ProductReader interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) (*Order, error)
	GetByID(ctx context.Context, id int64) (*Order, error)
	ListByUser(ctx context.Context, userID int64, limit int, offset int) ([]Order, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
	ListAll(ctx context.Context, limit int, offset int) ([]*Order, error)
}

type Service struct {
	products   ProductReader
	repository OrderRepository
}

func NewService(products ProductReader, repository OrderRepository) *Service {
	return &Service{
		products:   products,
		repository: repository,
	}
}

func (s *Service) Create(ctx context.Context, input *CreateOrder) (*Order, error) {
	if input == nil {
		return nil, fmt.Errorf(
			"%w: order is nil",
			ErrOrderValidation,
		)
	}

	if input.UserID <= 0 {
		return nil, fmt.Errorf(
			"%w: user id must be greater than zero",
			ErrOrderValidation,
		)
	}

	input.DeliveryAddress = strings.TrimSpace(
		input.DeliveryAddress,
	)

	if input.DeliveryAddress == "" {
		return nil, fmt.Errorf(
			"%w: delivery address is required",
			ErrOrderValidation,
		)
	}

	if len(input.Items) == 0 {
		return nil, fmt.Errorf(
			"%w: order must contain at least one item",
			ErrOrderValidation,
		)
	}

	seenProducts := make(map[int64]struct{})

	for _, item := range input.Items {
		if item.ProductID <= 0 {
			return nil, fmt.Errorf(
				"%w: product id must be greater than zero",
				ErrOrderValidation,
			)
		}

		if item.Quantity <= 0 {
			return nil, fmt.Errorf(
				"%w: quantity must be greater than zero",
				ErrOrderValidation,
			)
		}

		if _, exists := seenProducts[item.ProductID]; exists {
			return nil, fmt.Errorf(
				"%w: duplicate product id %d",
				ErrOrderValidation,
				item.ProductID,
			)
		}

		seenProducts[item.ProductID] = struct{}{}
	}

	order := &Order{
		UserID:          input.UserID,
		Status:          StatusNew,
		DeliveryAddress: input.DeliveryAddress,
		Items:           make([]OrderItem, 0, len(input.Items)),
	}

	for _, item := range input.Items {
		productData, err := s.products.GetByID(
			ctx,
			item.ProductID,
		)
		if err != nil {
			if errors.Is(err, product.ErrProductNotFound) {
				return nil, fmt.Errorf(
					"%w: product %d is unavailable",
					ErrOrderValidation,
					item.ProductID,
				)
			}

			return nil, fmt.Errorf(
				"failed to get product %d: %w",
				item.ProductID,
				err,
			)
		}

		if !productData.IsActive {
			return nil, fmt.Errorf(
				"%w: product %d is inactive",
				ErrOrderValidation,
				item.ProductID,
			)
		}

		orderItem := OrderItem{
			ProductID:     productData.ID,
			NameSnapshot:  productData.Name,
			PriceSnapshot: productData.Price,
			Quantity:      item.Quantity,
		}

		order.Items = append(order.Items, orderItem)

		order.TotalPrice += productData.Price * int64(item.Quantity)
	}

	createOrder, err := s.repository.Create(
		ctx,
		order,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"failed to create order: %w",
			err,
		)
	}

	return createOrder, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*Order, error) {
	if id <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid order id",
			ErrOrderValidation,
		)
	}

	order, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOrderNotFound
		}

		return nil, fmt.Errorf(
			"failed to get order: %w",
			err,
		)
	}

	return order, nil
}

func (s *Service) ListByUser(
	ctx context.Context,
	userID int64,
	limit int,
	offset int,
) ([]Order, error) {
	if userID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrOrderValidation,
		)
	}

	if offset < 0 {
		return nil, fmt.Errorf(
			"%w: invalid offset",
			ErrOrderValidation,
		)
	}

	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	orders, err := s.repository.ListByUser(
		ctx,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list user orders: %w",
			err,
		)
	}

	return orders, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status Status) error {
	if id <= 0 {
		return fmt.Errorf(
			"%w: invalid order id",
			ErrOrderValidation,
		)
	}

	order, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}

		return fmt.Errorf(
			"failed to get order: %w",
			err,
		)
	}

	if !CanTransitionStatus(order.Status, status) {
		return fmt.Errorf(
			"%w: cannot transition order from %q to %q",
			ErrOrderValidation,
			order.Status,
			status,
		)
	}

	if err := s.repository.UpdateStatus(
		ctx,
		id,
		status,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}

		return fmt.Errorf(
			"failed to update order status: %w",
			err,
		)
	}

	return nil
}

func (s *Service) ListAll(ctx context.Context, limit int, offset int) ([]*Order, error) {
	if limit <= 0 {
		return nil, fmt.Errorf(
			"%w: limit must be greater than zero",
			ErrOrderValidation,
		)
	}

	if limit > 100 {
		return nil, fmt.Errorf(
			"%w: limit must not exceed 100",
			ErrOrderValidation,
		)
	}

	if offset < 0 {
		return nil, fmt.Errorf(
			"%w: offset must not be negative",
			ErrOrderValidation,
		)
	}

	orders, err := s.repository.ListAll(
		ctx,
		limit,
		offset,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list orders: %w",
			err,
		)
	}

	return orders, nil
}
