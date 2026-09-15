package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThDawnWind/food-delivery-api/internal/product"
)

type ProductReader interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) (*Order, error)
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
