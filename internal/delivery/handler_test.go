package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"google.golang.org/grpc/metadata"
)

type fakeCalculator struct {
	ctx           context.Context
	orderAmount   int64
	deliveryPrice int64
	err           error
}

func (f *fakeCalculator) Calculate(
	ctx context.Context,
	orderAmount int64,
) (int64, error) {
	f.ctx = ctx
	f.orderAmount = orderAmount

	if f.err != nil {
		return 0, f.err
	}

	return f.deliveryPrice, nil
}

func TestHandler_Calculate(t *testing.T) {
	t.Parallel()

	calculator := &fakeCalculator{
		deliveryPrice: 19900,
	}

	handler := NewHandler(
		calculator,
		3*time.Second,
		slog.Default(),
	)

	body := []byte(`{"order_amount":120000}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/delivery/calculate",
		bytes.NewReader(body),
	)

	rec := httptest.NewRecorder()

	handler.Calculate(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	if calculator.orderAmount != 120000 {
		t.Fatalf(
			"order amount = %d, want %d",
			calculator.orderAmount,
			int64(120000),
		)
	}

	var response calculateResponse

	err := json.NewDecoder(rec.Body).Decode(&response)
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.DeliveryPrice != 19900 {
		t.Fatalf(
			"delivery price = %d, want %d",
			response.DeliveryPrice,
			int64(19900),
		)
	}
}

func TestHandler_Calculate_PropagatesRequestID(t *testing.T) {
	t.Parallel()

	calculator := &fakeCalculator{
		deliveryPrice: 19900,
	}

	handler := NewHandler(
		calculator,
		3*time.Second,
		slog.Default(),
	)

	body := []byte(`{"order_amount":120000}`)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/delivery/calculate",
		bytes.NewReader(body),
	)

	req.Header.Set(
		middleware.RequestIDHeader,
		"test-request-id",
	)

	rec := httptest.NewRecorder()

	httpHandler := middleware.RequestID(
		http.HandlerFunc(handler.Calculate),
	)

	httpHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d, want %d",
			rec.Code,
			http.StatusOK,
		)
	}

	md, ok := metadata.FromOutgoingContext(calculator.ctx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}

	values := md.Get("request-id")

	if len(values) != 1 {
		t.Fatalf(
			"request-id values = %v, want one value",
			values,
		)
	}

	if values[0] != "test-request-id" {
		t.Fatalf(
			"request-id = %q, want %q",
			values[0],
			"test-request-id",
		)
	}
}
