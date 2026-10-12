package delivery

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThDawnWind/food-delivery-api/internal/httpx"
	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Calculator interface {
	Calculate(
		ctx context.Context,
		orderAmount int64,
	) (int64, error)
}

type Handler struct {
	calculator Calculator
	timeout    time.Duration
	logger     *slog.Logger
}

func NewHandler(
	calculator Calculator,
	timeout time.Duration,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		calculator: calculator,
		timeout:    timeout,
		logger:     logger,
	}
}

type calculateRequest struct {
	OrderAmount int64 `json:"order_amount"`
}

type calculateResponse struct {
	DeliveryPrice int64 `json:"delivery_price"`
}

func (h *Handler) Calculate(
	w http.ResponseWriter,
	r *http.Request,
) {
	var request calculateRequest

	err := httpx.DecodeJSON(
		w,
		r,
		&request,
	)
	if err != nil {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"invalid request body",
		)

		return
	}

	if request.OrderAmount <= 0 {
		httpx.WriteError(
			w,
			http.StatusBadRequest,
			"order amount must be greater than zero",
		)

		return
	}

	ctx, cancel := context.WithTimeout(
		r.Context(),
		h.timeout,
	)
	defer cancel()

	requestID := middleware.GetReqID(r.Context())

	if requestID != "" {
		ctx = metadata.AppendToOutgoingContext(
			ctx,
			"request-id",
			requestID,
		)
	}

	deliveryPrice, err := h.calculator.Calculate(
		ctx,
		request.OrderAmount,
	)
	if err != nil {
		switch {
		case errors.Is(err, context.DeadlineExceeded),
			status.Code(err) == codes.DeadlineExceeded:
			httpx.WriteError(
				w,
				http.StatusGatewayTimeout,
				"delivery service timeout",
			)

		case status.Code(err) == codes.Unavailable:
			httpx.WriteError(
				w,
				http.StatusServiceUnavailable,
				"delivery service unavailable",
			)

		default:
			httpx.WriteInternalError(
				h.logger,
				w,
				r,
				"failed to calculate delivery",
				err,
			)
		}

		return
	}

	httpx.WriteJSON(
		w,
		http.StatusOK,
		calculateResponse{
			DeliveryPrice: deliveryPrice,
		},
	)
}
