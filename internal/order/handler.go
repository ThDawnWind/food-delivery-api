package order

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ThDawnWind/food-delivery-api/internal/auth"
	"github.com/go-chi/chi/v5"
)

type ServiceInterface interface {
	Create(ctx context.Context, input *CreateOrder) (*Order, error)
	GetByID(ctx context.Context, id int64) (*Order, error)
	ListByUser(ctx context.Context, userID int64, limit int, offset int) ([]Order, error)
	UpdateStatus(ctx context.Context, id int64, status Status) error
}

type Handler struct {
	service ServiceInterface
}

func NewHandler(service ServiceInterface) *Handler {
	return &Handler{
		service: service,
	}
}

type createOrderRequest struct {
	DeliveryAddress string                   `json:"delivery_address"`
	Items           []createOrderItemRequest `json:"items"`
}

type createOrderItemRequest struct {
	ProductID int64 `json:"product_id"`
	Quantity  int   `json:"quantity"`
}

type updateOrderStatusRequest struct {
	Status Status `json:"status"`
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createOrderRequest

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

	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	input := &CreateOrder{
		UserID:          userID,
		DeliveryAddress: req.DeliveryAddress,
		Items: make(
			[]CreateItem,
			0,
			len(req.Items),
		),
	}

	for _, item := range req.Items {
		input.Items = append(
			input.Items,
			CreateItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			},
		)
	}

	order, err := h.service.Create(
		r.Context(),
		input,
	)
	if err != nil {
		if errors.Is(err, ErrOrderValidation) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "internal server error",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusCreated,
		order,
	)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")

	id, err := strconv.ParseInt(idParam, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid order id",
			},
		)
		return
	}

	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	order, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "order not found",
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "internal server error",
			},
		)
		return
	}

	if order.UserID != userID {
		writeJSON(
			w,
			http.StatusNotFound,
			map[string]string{
				"error": "order not found",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		order,
	)
}

func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	limit := 0

	if value := r.URL.Query().Get("limit"); value != "" {
		parrsedlimit, err := strconv.Atoi(value)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "invalid limit",
				},
			)
			return
		}

		limit = parrsedlimit
	}

	offset := 0

	if value := r.URL.Query().Get("offset"); value != "" {
		parsedOffset, err := strconv.Atoi(value)
		if err != nil {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": "invalid offset",
				},
			)
			return
		}

		offset = parsedOffset
	}

	orders, err := h.service.ListByUser(
		r.Context(),
		userID,
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, ErrOrderValidation) {
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)
			return
		}

		writeJSON(
			w,
			http.StatusInternalServerError,
			map[string]string{
				"error": "internal server error",
			},
		)
		return
	}

	writeJSON(
		w,
		http.StatusOK,
		orders,
	)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || id <= 0 {
		writeJSON(
			w,
			http.StatusBadRequest,
			map[string]string{
				"error": "invalid order id",
			},
		)
		return
	}

	role, ok := auth.RoleFromContext(
		r.Context(),
	)
	if !ok {
		writeJSON(
			w,
			http.StatusUnauthorized,
			map[string]string{
				"error": "unauthorized",
			},
		)
		return
	}

	if role != "admin" {
		writeJSON(
			w,
			http.StatusForbidden,
			map[string]string{
				"error": "forbidden",
			},
		)
		return
	}

	var req updateOrderStatusRequest

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

	err = h.service.UpdateStatus(
		r.Context(),
		id,
		req.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderValidation):
			writeJSON(
				w,
				http.StatusBadRequest,
				map[string]string{
					"error": err.Error(),
				},
			)
		case errors.Is(err, ErrOrderNotFound):
			writeJSON(
				w,
				http.StatusNotFound,
				map[string]string{
					"error": "order not found",
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

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.ListByUser)
	r.Get("/{id}", h.GetByID)
	r.Patch("/{id}/status", h.UpdateStatus)

	return r
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(data)
}
