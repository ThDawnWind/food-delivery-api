package user

import (
	"errors"
	"time"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateUser struct {
	Username     string
	Email        string
	PasswordHash string
	Role         Role
}

type RegisterUser struct {
	Username string
	Email    string
	Password string
}

type LoginUser struct {
	Email    string
	Password string
}

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserValidation     = errors.New("user validation error")
	ErrUserConflict       = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
