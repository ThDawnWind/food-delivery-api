package product

import (
	"errors"
	"time"
)

type Product struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Price       int64          `json:"price"`
	Weight      int            `json:"weight"`
	CategoryID  int64          `json:"category_id"`
	IsActive    bool           `json:"is_active"`
	Images      []ProductImage `json:"images"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type ProductImage struct {
	ID        int64     `json:"id"`
	ProductID int64     `json:"product_id"`
	URL       string    `json:"url"`
	SortOrder int       `json:"sort_order"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	ErrProductNotFound   = errors.New("product not found")
	ErrProductValidation = errors.New("product validation error")
)
