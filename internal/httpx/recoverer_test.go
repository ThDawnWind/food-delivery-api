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

func TestRecoverer(t *testing.T) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	handler := middleware.RequestID(
		Recoverer(
			logger,
		)(
			http.HandlerFunc(
				func(
					w http.ResponseWriter,
					r *http.Request,
				) {
					panic(
						"test panic",
					)
				},
			),
		),
	)

	request := httptest.NewRequest(
		http.MethodGet,
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

	if response.Code != http.StatusInternalServerError {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusInternalServerError,
			response.Code,
		)
	}

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

	if entry["msg"] != "http panic" {
		t.Errorf(
			"expected message %q, got %v",
			"http panic",
			entry["msg"],
		)
	}

	if entry["request_id"] != "test-request-id" {
		t.Errorf(
			"expected request ID %q, got %v",
			"test-request-id",
			entry["request_id"],
		)
	}

	if entry["method"] != http.MethodGet {
		t.Errorf(
			"expected method %q, got %v",
			http.MethodGet,
			entry["method"],
		)
	}

	if entry["path"] != "/api/v1/orders" {
		t.Errorf(
			"expected path %q, got %v",
			"/api/v1/orders",
			entry["path"],
		)
	}

	if entry["panic"] != "test panic" {
		t.Errorf(
			"expected panic %q, got %v",
			"test panic",
			entry["panic"],
		)
	}

	stack, ok := entry["stack"].(string)
	if !ok || stack == "" {
		t.Fatal(
			"expected non-empty stack trace",
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

func TestRecoverer_NoPanic(t *testing.T) {
	var buffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&buffer,
			nil,
		),
	)

	handler := Recoverer(
		logger,
	)(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				w.WriteHeader(
					http.StatusCreated,
				)
			},
		),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
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

	if buffer.Len() != 0 {
		t.Fatalf(
			"expected no panic log, got %s",
			buffer.String(),
		)
	}
}

func TestRecoverer_AbortHandler(
	t *testing.T,
) {
	logger := slog.New(
		slog.NewJSONHandler(
			&bytes.Buffer{},
			nil,
		),
	)

	handler := Recoverer(
		logger,
	)(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				panic(
					http.ErrAbortHandler,
				)
			},
		),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)

	response := httptest.NewRecorder()

	defer func() {
		recovered := recover()

		if recovered != http.ErrAbortHandler {
			t.Fatalf(
				"expected http.ErrAbortHandler, got %v",
				recovered,
			)
		}
	}()

	handler.ServeHTTP(
		response,
		request,
	)
}
