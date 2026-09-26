package address

import (
	"errors"
	"time"
)

type Address struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Label           *string   `json:"label,omitempty"`
	City            string    `json:"city"`
	Street          string    `json:"street"`
	HouseNumber     string    `json:"house_number"`
	ApartmentNumber *string   `json:"apartment_number,omitempty"`
	Entrance        *string   `json:"entrance,omitempty"`
	Floor           *string   `json:"floor,omitempty"`
	Comment         *string   `json:"comment,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type CreateAddress struct {
	UserID          int64
	Label           *string
	City            string
	Street          string
	HouseNumber     string
	ApartmentNumber *string
	Entrance        *string
	Floor           *string
	Comment         *string
}

type UpdateAddress struct {
	Label           *string
	City            *string
	Street          *string
	HouseNumber     *string
	ApartmentNumber *string
	Entrance        *string
	Floor           *string
	Comment         *string
}

var (
	ErrAddressNotFound   = errors.New("address not found")
	ErrAddressValidation = errors.New("address validation error")
)
