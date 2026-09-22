package product

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	GetByID(ctx context.Context, id int64) (*Product, error)
	List(ctx context.Context, filter ListFilter) ([]Product, error)
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
	query := r.URL.Query()

	filter := ListFilter{
		Search: query.Get("search"),
	}

	if rawCategoryID := query.Get("category_id"); rawCategoryID != "" {
		categoryID, err := strconv.ParseInt(
			rawCategoryID,
			10,
			64,
		)
		if err != nil {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid category_id",
			)
			return
		}

		filter.CategoryID = &categoryID
	}

	if rawLimit := query.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid limit",
			)
			return
		}

		filter.Limit = limit
	}

	if rawOffset := query.Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid offset",
			)
			return
		}

		filter.Offset = offset
	}

	products, err := h.service.List(
		r.Context(),
		filter,
	)
	if err != nil {
		if errors.Is(err, ErrProductValidation) {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		httpx.WriteError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		products,
	)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	productData, err := h.service.GetByID(
		r.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"product not found",
			)

		case errors.Is(err, ErrProductValidation):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid product id",
			)

		default:
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		productData,
	)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	isActive := true

	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	images := make(
		[]ProductImage,
		0,
		len(req.Images),
	)

	for _, image := range req.Images {
		images = append(
			images,
			ProductImage{
				URL:       image.URL,
				SortOrder: image.SortOrder,
				IsPrimary: image.IsPrimary,
			},
		)
	}

	productData := &Product{
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
		productData,
	)
	if err != nil {
		if errors.Is(err, ErrProductValidation) {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)
			return
		}

		httpx.WriteError(
			w,
			http.StatusInternalServerError,
			"internal server error",
		)
		return
	}

	httpx.WriteJSON(
		w,
		http.StatusCreated,
		createdProduct,
	)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	var req updateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	if req.IsActive == nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"is_active is required",
		)
		return
	}

	images := make(
		[]ProductImage,
		0,
		len(req.Images),
	)

	for _, image := range req.Images {
		images = append(
			images,
			ProductImage{
				URL:       image.URL,
				SortOrder: image.SortOrder,
				IsPrimary: image.IsPrimary,
			},
		)
	}

	productData := &Product{
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
		productData,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductValidation):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				err.Error(),
			)

		case errors.Is(err, ErrProductNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"product not found",
			)

		default:
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		updatedProduct,
	)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid product id",
		)
		return
	}

	err = h.service.Deactivate(
		r.Context(),
		id,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductValidation):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid product id",
			)

		case errors.Is(err, ErrProductNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"product not found",
			)

		default:
			httpx.WriteError(
				w,
				http.StatusInternalServerError,
				"internal server error",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
