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
)

type fakeCalculator struct {
	orderAmount   int64
	deliveryPrice int64
	err           error
}

func (f *fakeCalculator) Calculate(
	_ context.Context,
	orderAmount int64,
) (int64, error) {
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
