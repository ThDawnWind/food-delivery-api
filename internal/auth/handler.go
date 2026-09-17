package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ThDawnWind/food-delivery-api/internal/user"

	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	Register(ctx context.Context, input *user.RegisterUser) (*user.User, error)
	Login(ctx context.Context, input *user.LoginUser) (*LoginResult, error)
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
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
		case errors.Is(err, user.ErrUserValidation):
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)

		case errors.Is(err, user.ErrUserConflict):
			writeJSON(
				w,
				http.StatusConflict,
				map[string]string{
					"error": "user already exists",
				},
			)

		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal server error",
				},
			)
		}

		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		userData,
	)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid request body",
			},
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
		case errors.Is(err, user.ErrUserValidation):
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)
		case errors.Is(err, user.ErrInvalidCredentials):
			writeJSON(
				w,
				http.StatusUnauthorized,
				map[string]string{
					"error": "invalid credentials",
				},
			)
		default:
			writeJSON(
				w,
				http.StatusInternalServerError,
				map[string]string{
					"error": "internal server error",
				},
			)
		}
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		res,
	)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
