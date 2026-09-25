package httpx

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestRequestLogger(t *testing.T) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	handler := middleware.RequestID(
		RequestLogger(
			logger,
			nil,
		)(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(
						http.StatusCreated,
					)
				},
			),
		),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/orders?token=secret",
		nil,
	)

	request.Header.Set(
		middleware.RequestIDHeader,
		"test-request-id",
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(
		response,
		request,
	)

	if response.Code != http.StatusCreated {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusCreated,
			response.Code,
		)
	}

	var logEntry map[string]any

	if err := json.Unmarshal(
		buffer.Bytes(),
		&logEntry,
	); err != nil {
		t.Fatalf(
			"failed to decode log entry: %v",
			err,
		)
	}

	if logEntry["msg"] != "http request" {
		t.Errorf(
			"expected message %q, got %v",
			"http request",
			logEntry["msg"],
		)
	}

	if logEntry["request_id"] != "test-request-id" {
		t.Errorf(
			"expected request ID %q, got %v",
			"test-request-id",
			logEntry["request_id"],
		)
	}

	if logEntry["method"] != http.MethodPost {
		t.Errorf(
			"expected method %q, got %v",
			http.MethodPost,
			logEntry["method"],
		)
	}

	if logEntry["path"] != "/api/v1/orders" {
		t.Errorf(
			"expected path %q, got %v",
			"/api/v1/orders",
			logEntry["path"],
		)
	}

	if logEntry["status"] != float64(
		http.StatusCreated,
	) {
		t.Errorf(
			"expected status %d, got %v",
			http.StatusCreated,
			logEntry["status"],
		)
	}

	if _, ok := logEntry["duration_ms"]; !ok {
		t.Error(
			"expected duration_ms field",
		)
	}

	if _, ok := logEntry["client_ip"]; !ok {
		t.Error(
			"expected client_ip field",
		)
	}

	if bytes.Contains(
		buffer.Bytes(),
		[]byte("token=secret"),
	) {
		t.Fatal(
			"query string must not be logged",
		)
	}
}

func TestRequestLogger_DefaultStatusOK(
	t *testing.T,
) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	handler := RequestLogger(
		logger,
		nil,
	)(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write(
					[]byte("OK"),
				)
			},
		),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(
		response,
		request,
	)

	var logEntry map[string]any

	if err := json.Unmarshal(
		buffer.Bytes(),
		&logEntry,
	); err != nil {
		t.Fatalf(
			"failed to decode log entry: %v",
			err,
		)
	}

	if logEntry["status"] != float64(
		http.StatusOK,
	) {
		t.Errorf(
			"expected status %d, got %v",
			http.StatusOK,
			logEntry["status"],
		)
	}
}
