package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context) ([]Product, error)
	Create(ctx context.Context, product *Product) (*Product, error)
	Update(ctx context.Context, product *Product) (*Product, error)
	Deactivate(ctx context.Context, id int64) error
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

type createProductRequest struct {
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	Price       int64                 `json:"price"`
	Weight      int                   `json:"weight"`
	CategoryID  int64                 `json:"category_id"`
	IsActive    *bool                 `json:"is_active"`
	Images      []productImageRequest `json:"images"`
}

type productImageRequest struct {
	URL       string `json:"url"`
	SortOrder int    `json:"sort_order"`
	IsPrimary bool   `json:"is_primary"`
}

type updateProductRequest struct {
	Name        string                `json:"name"`
	Description *string               `json:"description"`
	Price       int64                 `json:"price"`
	Weight      int                   `json:"weight"`
	CategoryID  int64                 `json:"category_id"`
	IsActive    *bool                 `json:"is_active"`
	Images      []productImageRequest `json:"images"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.service.List(r.Context())
	if err != nil {
		http.Error(
			w,
			"internal server error",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(products); err != nil {
		return
	}
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	product, err := h.service.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			http.Error(
				w,
				"product not found",
				http.StatusNotFound,
			)
			return
		}

		if errors.Is(err, ErrProductValidation) {
			http.Error(
				w,
				"invalid product id",
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

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(product); err != nil {
		return
	}
}

func (h *Handler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req createProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	isActive := true

	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	images := make([]ProductImage, 0, len(req.Images))

	for _, image := range req.Images {
		images = append(images, ProductImage{
			URL:       image.URL,
			SortOrder: image.SortOrder,
			IsPrimary: image.IsPrimary,
		})
	}

	product := &Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Weight:      req.Weight,
		CategoryID:  req.CategoryID,
		IsActive:    isActive,
		Images:      images,
	}

	createdProduct, err := h.service.Create(
		r.Context(),
		product,
	)
	if err != nil {
		if errors.Is(err, ErrProductValidation) {
			http.Error(
				w,
				err.Error(),
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

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdProduct); err != nil {
		return
	}
}

func (h *Handler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	var req updateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	if req.IsActive == nil {
		http.Error(
			w,
			"is_active is required",
			http.StatusBadRequest,
		)
		return
	}

	images := make([]ProductImage, 0, len(req.Images))

	for _, image := range req.Images {
		images = append(images, ProductImage{
			URL:       image.URL,
			SortOrder: image.SortOrder,
			IsPrimary: image.IsPrimary,
		})
	}

	product := &Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Weight:      req.Weight,
		CategoryID:  req.CategoryID,
		IsActive:    *req.IsActive,
		Images:      images,
	}

	updatedProduct, err := h.service.Update(
		r.Context(),
		product,
	)
	if err != nil {
		if errors.Is(err, ErrProductValidation) {
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)
			return
		}

		if errors.Is(err, ErrProductNotFound) {
			http.Error(
				w,
				"product not found",
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

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(updatedProduct); err != nil {
		return
	}
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		http.Error(
			w,
			"invalid product id",
			http.StatusBadRequest,
		)
		return
	}

	err = h.service.Deactivate(
		r.Context(),
		id,
	)
	if err != nil {
		if errors.Is(err, ErrProductValidation) {
			http.Error(
				w,
				"invalid product id",
				http.StatusBadRequest,
			)
			return
		}

		if errors.Is(err, ErrProductNotFound) {
			http.Error(
				w,
				"product not found",
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
