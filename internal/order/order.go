package order

import (
	"errors"
	"time"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusConfirmed  Status = "confirmed"
	StatusCooking    Status = "cooking"
	StatusReady      Status = "ready"
	StatusDelivering Status = "delivering"
	StatusCompleted  Status = "completed"
	StatusCancelled  Status = "cancelled"
)

type Order struct {
	ID              int64       `json:"id"`
	UserID          int64       `json:"user_id"`
	Status          Status      `json:"status"`
	TotalPrice      int64       `json:"total_price"`
	DeliveryAddress string      `json:"delivery_address"`
	Items           []OrderItem `json:"items"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	ProductID     int64     `json:"product_id"`
	NameSnapshot  string    `json:"name_snapshot"`
	PriceSnapshot int64     `json:"price_snapshot"`
	Quantity      int       `json:"quantity"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderItems struct {
	ID            int64     `json:"id"`
	OrderID       int64     `json:"order_id"`
	ProductID     int64     `json:"product_id"`
	NameSnapshot  string    `json:"name_snapshot"`
	PriceSnapshot int64     `json:"price_snapshot"`
	Quantity      int       `json:"quantity"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateOrder struct {
	UserID          int64        `json:"user_id"`
	DeliveryAddress string       `json:"delivery_address"`
	Items           []CreateItem `json:"items"`
}

type CreateItem struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type ListOrdersFilter struct {
	Status string
	UserID *int64

	CreatedFrom *time.Time
	CreatedTo   *time.Time

	Limit  int
	Offset int
}

type ListOrdersResult struct {
	Items  []*Order `json:"items"`
	Total  int64    `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

var (
	ErrOrderValidation = errors.New("order validation error")
	ErrOrderNotFound   = errors.New("order not found")
)

func CanTransitionStatus(from, to Status) bool {
	switch from {
	case StatusNew:
		return to == StatusConfirmed ||
			to == StatusCancelled

	case StatusConfirmed:
		return to == StatusCooking ||
			to == StatusCancelled

	case StatusCooking:
		return to == StatusReady ||
			to == StatusCancelled

	case StatusReady:
		return to == StatusDelivering ||
			to == StatusCancelled

	case StatusDelivering:
		return to == StatusCompleted

	case StatusCompleted:
		return false

	case StatusCancelled:
		return false

	default:
		return false
	}
}
