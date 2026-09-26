package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5/middleware"
)

func TestLogInternalError(t *testing.T) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register?token=secret",
		nil,
	)

	request.Header.Set(
		middleware.RequestIDHeader,
		"test-request-id",
	)

	handler := middleware.RequestID(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				LogInternalError(
					logger,
					r,
					"failed to register user",
					errors.New(
						"database unavailable",
					),
				)
			},
		),
	)

	response := httptest.NewRecorder()

	handler.ServeHTTP(
		response,
		request,
	)

	var entry map[string]any

	if err := json.Unmarshal(
		buffer.Bytes(),
		&entry,
	); err != nil {
		t.Fatalf(
			"failed to decode log entry: %v",
			err,
		)
	}

	if entry["level"] != "ERROR" {
		t.Errorf(
			"expected ERROR level, got %v",
			entry["level"],
		)
	}

	if entry["msg"] != "failed to register user" {
		t.Errorf(
			"unexpected message: %v",
			entry["msg"],
		)
	}

	if entry["request_id"] != "test-request-id" {
		t.Errorf(
			"unexpected request ID: %v",
			entry["request_id"],
		)
	}

	if entry["method"] != http.MethodPost {
		t.Errorf(
			"unexpected method: %v",
			entry["method"],
		)
	}

	if entry["path"] != "/api/v1/auth/register" {
		t.Errorf(
			"unexpected path: %v",
			entry["path"],
		)
	}

	if entry["error"] != "database unavailable" {
		t.Errorf(
			"unexpected error: %v",
			entry["error"],
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

func TestWriteInternalError(t *testing.T) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/test",
		nil,
	)

	request.Header.Set(
		middleware.RequestIDHeader,
		"test-request-id",
	)

	response := httptest.NewRecorder()

	handler := middleware.RequestID(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				WriteInternalError(
					logger,
					w,
					r,
					"test internal error",
					errors.New(
						"database unavailable",
					),
				)
			},
		),
	)

	handler.ServeHTTP(
		response,
		request,
	)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.Code,
		)
	}

	var body map[string]string

	if err := json.NewDecoder(
		response.Body,
	).Decode(&body); err != nil {
		t.Fatalf(
			"failed to decode response: %v",
			err,
		)
	}

	if body["error"] != "internal server error" {
		t.Errorf(
			"unexpected client error: %q",
			body["error"],
		)
	}

	if bytes.Contains(
		response.Body.Bytes(),
		[]byte("database unavailable"),
	) {
		t.Fatal(
			"internal error must not be exposed to client",
		)
	}

	if !bytes.Contains(
		buffer.Bytes(),
		[]byte("database unavailable"),
	) {
		t.Fatal(
			"expected internal error in log",
		)
	}
}
