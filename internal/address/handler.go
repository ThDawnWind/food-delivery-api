package address

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/ThDawnWind/food-delivery-api/internal/auth"
	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/go-chi/chi/v5"
)

type AddressService interface {
	Create(ctx context.Context, input *CreateAddress) (*Address, error)
	GetByID(ctx context.Context, addressID int64, userID int64) (*Address, error)
	ListByUser(ctx context.Context, userID int64) ([]Address, error)
	Update(ctx context.Context, addressID int64, userID int64, input *UpdateAddress) (*Address, error)
	Delete(ctx context.Context, addressID int64, userID int64) error
}

type Handler struct {
	service AddressService
}

func NewHandler(service AddressService) *Handler {
	return &Handler{
		service: service,
	}
}

type createAddressRequest struct {
	Label           *string `json:"label"`
	City            string  `json:"city"`
	Street          string  `json:"street"`
	HouseNumber     string  `json:"house_number"`
	ApartmentNumber *string `json:"apartment_number"`
	Entrance        *string `json:"entrance"`
	Floor           *string `json:"floor"`
	Comment         *string `json:"comment"`
}

type updateAddressRequest struct {
	Label           *string `json:"label"`
	City            *string `json:"city"`
	Street          *string `json:"street"`
	HouseNumber     *string `json:"house_number"`
	ApartmentNumber *string `json:"apartment_number"`
	Entrance        *string `json:"entrance"`
	Floor           *string `json:"floor"`
	Comment         *string `json:"comment"`
}

func (h *Handler) Routes() http.Handler {
	router := chi.NewRouter()

	router.Post("/", h.Create)
	router.Get("/", h.ListByUser)
	router.Get("/{id}", h.GetByID)
	router.Patch("/{id}", h.Update)
	router.Delete("/{id}", h.Delete)

	return router
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	var request createAddressRequest

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	address, err := h.service.Create(
		r.Context(),
		&CreateAddress{
			UserID:          userID,
			Label:           request.Label,
			City:            request.City,
			Street:          request.Street,
			HouseNumber:     request.HouseNumber,
			ApartmentNumber: request.ApartmentNumber,
			Entrance:        request.Entrance,
			Floor:           request.Floor,
			Comment:         request.Comment,
		},
	)
	if err != nil {
		if errors.Is(
			err,
			ErrAddressValidation,
		) {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid address data",
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
		address,
	)
}

func (h *Handler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	addresses, err := h.service.ListByUser(
		r.Context(),
		userID,
	)
	if err != nil {
		if errors.Is(
			err,
			ErrAddressValidation,
		) {
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid address data",
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
		addresses,
	)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	addressID, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || addressID <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid address id",
		)
		return
	}

	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	address, err := h.service.GetByID(
		r.Context(),
		addressID,
		userID,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			ErrAddressValidation,
		):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid address data",
			)

		case errors.Is(
			err,
			ErrAddressNotFound,
		):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"address not found",
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
		address,
	)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	addressID, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || addressID <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid address id",
		)
		return
	}

	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	var request updateAddressRequest

	if err := json.NewDecoder(r.Body).Decode(
		&request,
	); err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)
		return
	}

	address, err := h.service.Update(
		r.Context(),
		addressID,
		userID,
		&UpdateAddress{
			Label:           request.Label,
			City:            request.City,
			Street:          request.Street,
			HouseNumber:     request.HouseNumber,
			ApartmentNumber: request.ApartmentNumber,
			Entrance:        request.Entrance,
			Floor:           request.Floor,
			Comment:         request.Comment,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrAddressValidation):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid address data",
			)

		case errors.Is(err, ErrAddressNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"address not found",
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
		address,
	)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	addressID, err := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)
	if err != nil || addressID <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid address id",
		)
		return
	}

	userID, ok := auth.UserIDFromContext(
		r.Context(),
	)
	if !ok || userID <= 0 {
		httpx.WriteError(
			w,
			http.StatusUnauthorized,
			"unauthorized",
		)
		return
	}

	err = h.service.Delete(
		r.Context(),
		addressID,
		userID,
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrAddressValidation):
			httpx.WriteError(
				w,
				http.StatusBadRequest,
				"invalid address data",
			)

		case errors.Is(err, ErrAddressNotFound):
			httpx.WriteError(
				w,
				http.StatusNotFound,
				"address not found",
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
