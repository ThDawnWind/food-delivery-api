package category

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	GetByID(ctx context.Context, id int64) (*Category, error)
	List(ctx context.Context) ([]Category, error)
	Create(ctx context.Context, name string, slug string) (*Category, error)
	Update(ctx context.Context, category *Category) error
	Delete(ctx context.Context, id int64) error
}

type Handler struct {
	service ServiceInterface
	logger  *slog.Logger
}

func NewHandler(
	service ServiceInterface,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

type createCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type updateCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

func (h *Handler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	categories, err := h.service.List(
		r.Context(),
	)
	if err != nil {
		httpx.WriteInternalError(
			h.logger,
			w,
			r,
			"failed to list categories",
			err,
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		categories,
	)
}

func (h *Handler) GetByID(
	w http.ResponseWriter,
	r *http.Request,
) {
	idParam := chi.URLParam(
		r,
		"id",
	)

	id, err := strconv.ParseInt(
		idParam,
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid category id",
		)
		return
	}

	category, err := h.service.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrCategoryNotFound,
		) {
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"category not found",
			)
			return
		}

		httpx.WriteInternalError(
			h.logger,
			w,
			r,
			"failed to get category",
			err,
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		category,
	)
}

func (h *Handler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req createCategoryRequest

	err := httpx.DecodeJSON(
		w,
		r,
		&req,
	)
	if err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	category, err := h.service.Create(
		r.Context(),
		req.Name,
		req.Slug,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrCategoryValidation,
		) {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid category data",
			)
			return
		}

		httpx.WriteInternalError(
			h.logger,
			w,
			r,
			"failed to create category",
			err,
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		category,
	)
}

func (h *Handler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	idParam := chi.URLParam(
		r,
		"id",
	)

	id, err := strconv.ParseInt(
		idParam,
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid category id",
		)
		return
	}

	var req updateCategoryRequest

	err = httpx.DecodeJSON(
		w,
		r,
		&req,
	)
	if err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	category := &Category{
		ID:   id,
		Name: req.Name,
		Slug: req.Slug,
	}

	err = h.service.Update(
		r.Context(),
		category,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrCategoryValidation,
		):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid category data",
			)

		case errors.Is(
			err,
			ErrCategoryNotFound,
		):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"category not found",
			)

		default:
			httpx.WriteInternalError(
				h.logger,
				w,
				r,
				"failed to update category",
				err,
			)
		}

		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	idParam := chi.URLParam(
		r,
		"id",
	)

	id, err := strconv.ParseInt(
		idParam,
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid category id",
		)
		return
	}

	err = h.service.Delete(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrCategoryNotFound,
		) {
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"category not found",
			)
			return
		}

		httpx.WriteInternalError(
			h.logger,
			w,
			r,
			"failed to delete category",
			err,
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}
