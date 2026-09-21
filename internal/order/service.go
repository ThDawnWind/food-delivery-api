package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThDawnWind/food-delivery-api/internal/address"
	"github.com/ThDawnWind/food-delivery-api/internal/product"
	"github.com/jackc/pgx/v5"
)

type ProductReader interface {
	GetByID(ctx context.Context, id int64) (*product.Product, error)
}

type AddressReader interface {
	GetByID(ctx context.Context, addressID int64, userID int64) (*address.Address, error)
}

type OrderRepository interface {
	Create(ctx context.Context, order *Order) (*Order, error)
	GetByID(ctx context.Context, id int64) (*Order, error)
	ListByUser(ctx context.Context, userID int64, limit int, offset int) ([]Order, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
	ListAll(ctx context.Context, filter ListOrdersFilter) ([]*Order, error)
	CountAll(ctx context.Context, filter ListOrdersFilter) (int64, error)
}

type Service struct {
	products   ProductReader
	repository OrderRepository
	addresses  AddressReader
}

func NewService(products ProductReader, repository OrderRepository, addresses AddressReader) *Service {
	return &Service{
		products:   products,
		repository: repository,
		addresses:  addresses,
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

	if input.AddressID <= 0 {
		return nil, fmt.Errorf(
			"%w: address id must be greater than zero",
			ErrOrderValidation,
		)
	}

	addressData, err := s.addresses.GetByID(
		ctx,
		input.AddressID,
		input.UserID,
	)
	if err != nil {
		if errors.Is(err, address.ErrAddressNotFound) {
			return nil, fmt.Errorf(
				"%w: address not found",
				ErrOrderValidation,
			)
		}

		return nil, fmt.Errorf(
			"failed to get address: %w",
			err,
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
		DeliveryAddress: buildAddressSnapshot(addressData),
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

func (s *Service) ListAll(ctx context.Context, filter ListOrdersFilter) (*ListOrdersResult, error) {
	if filter.Limit <= 0 {
		return nil, fmt.Errorf(
			"%w: limit must be greater than zero",
			ErrOrderValidation,
		)
	}

	if filter.Limit > 100 {
		return nil, fmt.Errorf(
			"%w: limit must not exceed 100",
			ErrOrderValidation,
		)
	}

	if filter.Offset < 0 {
		return nil, fmt.Errorf(
			"%w: offset must not be negative",
			ErrOrderValidation,
		)
	}

	if filter.UserID != nil && *filter.UserID <= 0 {
		return nil, fmt.Errorf(
			"%w: invalid user id",
			ErrOrderValidation,
		)
	}

	if filter.CreatedFrom != nil &&
		filter.CreatedTo != nil &&
		filter.CreatedFrom.After(*filter.CreatedTo) {
		return nil, fmt.Errorf(
			"%w: created_from must not be after created_to",
			ErrOrderValidation,
		)
	}

	if filter.Status != "" {
		switch Status(filter.Status) {
		case StatusNew,
			StatusConfirmed,
			StatusCooking,
			StatusReady,
			StatusDelivering,
			StatusCompleted,
			StatusCancelled:

		default:
			return nil, fmt.Errorf(
				"%w: invalid order status",
				ErrOrderValidation,
			)
		}
	}

	orders, err := s.repository.ListAll(
		ctx,
		filter,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to list orders: %w",
			err,
		)
	}

	total, err := s.repository.CountAll(
		ctx,
		filter,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to count orders: %w",
			err,
		)
	}

	return &ListOrdersResult{
		Items:  orders,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (s *Service) Cancel(ctx context.Context, orderID int64, userID int64) error {
	if orderID <= 0 {
		return fmt.Errorf(
			"%w: invalid order id",
			ErrOrderValidation,
		)
	}

	if userID <= 0 {
		return fmt.Errorf(
			"%w: invalid user id",
			ErrOrderValidation,
		)
	}

	order, err := s.repository.GetByID(
		ctx,
		orderID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}

		return fmt.Errorf(
			"failed to get order: %w",
			err,
		)
	}

	if order.UserID != userID {
		return ErrOrderNotFound
	}

	switch order.Status {
	case StatusNew, StatusConfirmed:

	default:
		return fmt.Errorf(
			"%w: order cannot be cancelled in status %q",
			ErrOrderValidation,
			order.Status,
		)
	}

	if err := s.repository.UpdateStatus(
		ctx,
		orderID,
		StatusCancelled,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}

		return fmt.Errorf(
			"failed to cancel order: %w",
			err,
		)
	}

	return nil
}

func buildAddressSnapshot(a *address.Address) string {
	parts := []string{
		a.City,
		a.Street,
		a.HouseNumber,
	}

	if a.ApartmentNumber != nil &&
		strings.TrimSpace(*a.ApartmentNumber) != "" {
		parts = append(
			parts,
			"apt. "+strings.TrimSpace(*a.ApartmentNumber),
		)
	}

	if a.Entrance != nil &&
		strings.TrimSpace(*a.Entrance) != "" {
		parts = append(
			parts,
			"entrance "+strings.TrimSpace(*a.Entrance),
		)
	}

	if a.Floor != nil &&
		strings.TrimSpace(*a.Floor) != "" {
		parts = append(
			parts,
			"floor "+strings.TrimSpace(*a.Floor),
		)
	}

	if a.Comment != nil &&
		strings.TrimSpace(*a.Comment) != "" {
		parts = append(
			parts,
			"comment: "+strings.TrimSpace(*a.Comment),
		)
	}

	return strings.Join(parts, ", ")
}
