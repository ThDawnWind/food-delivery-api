package category

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	GetByID(ctx context.Context, id int64) (*Category, error)
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, name, slug string) (*Category, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id int64) error
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	categories, err := h.service.List(r.Context())
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(categories)
	if err != nil {
		return
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid category id",
			http.StatusBadRequest,
		)

		return
	}

	category, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrCategoryNotFound) {
			http.Error(
				w,
				"category not found",
				http.StatusNotFound,
			)
			return
		}

		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(category)
	if err != nil {
		return
	}
}
