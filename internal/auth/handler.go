package auth

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/ThDawnWind/food-delivery-api/internal/user"
	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	Register(ctx context.Context,
		input *user.RegisterUser,
	) (*user.User, error)

	Login(
		ctx context.Context,
		input *user.LoginUser,
	) (*LoginResult, error)
}

type Handler struct {
	service ServiceInterface
	logger  *slog.Logger
}

func NewHandler(service ServiceInterface, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/register", h.Register)
	r.Post("/login", h.Login)

	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest

	if err := httpx.DecodeJSON(
		w,
		r,
		&req,
	); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	userData, err := h.service.Register(
		r.Context(),
		&user.RegisterUser{
			Username: req.Username,
			Email:    req.Email,
			Password: req.Password,
		},
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			user.ErrUserValidation,
		):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		case errors.Is(
			err,
			user.ErrUserConflict,
		):
			httpx.WriteError(
				w,
				http.StatusConflict,
				"user already exists",
			)

		default:
			httpx.WriteInternalError(
				h.logger,
				w,
				r,
				"failed to register user",
				err,
			)
		}

		return
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		userData,
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := httpx.DecodeJSON(
		w,
		r,
		&req,
	); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	res, err := h.service.Login(
		r.Context(),
		&user.LoginUser{
			Email:    req.Email,
			Password: req.Password,
		},
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			user.ErrUserValidation,
		):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		case errors.Is(
			err,
			user.ErrInvalidCredentials,
		):
			httpx.WriteError(
				w,
				http.StatusUnauthorized,
				"invalid credentials",
			)

		default:
			httpx.WriteInternalError(
				h.logger,
				w,
				r,
				"failed to login user",
				err,
			)
		}

		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		res,
	)
}
