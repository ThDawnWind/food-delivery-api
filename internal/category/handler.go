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

type createCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type updateCategoryRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createCategoryRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)

		return
	}

	category, err := h.service.Create(
		r.Context(),
		req.Name,
		req.Slug,
	)
	if err != nil {
		if errors.Is(err, ErrCategoryValidation) {
			http.Error(
				w,
				"invalid category data",
				http.StatusBadRequest,
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
	w.WriteHeader(http.StatusCreated)

	err = json.NewEncoder(w).Encode(category)
	if err != nil {
		return
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	var req updateCategoryRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	category := &Category{
		ID:   id,
		Name: req.Name,
		Slug: req.Slug,
	}

	err = h.service.Update(r.Context(), category)
	if err != nil {
		if errors.Is(err, ErrCategoryValidation) {
			http.Error(
				w,
				"invalid category data",
				http.StatusBadRequest,
			)
			return
		}

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
			"Internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
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

	err = h.service.Delete(r.Context(), id)
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

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Get("/", h.List)
	r.Post("/", h.Create)

	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}
