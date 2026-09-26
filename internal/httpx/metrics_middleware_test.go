package httpx

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsMiddleware_RoutePatternAndStatus(t *testing.T) {
	registry := prometheus.NewRegistry()

	metrics := NewHTTPMetrics(
		registry,
	)

	router := chi.NewRouter()

	router.Use(
		MetricsMiddleware(
			metrics,
		),
	)

	router.Get(
		"/api/v1/products/{id}",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			id := chi.URLParam(
				r,
				"id",
			)

			if id == "999999" {
				http.NotFound(
					w,
					r,
				)
				return
			}

			w.WriteHeader(
				http.StatusOK,
			)
		},
	)

	successRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/products/1",
		nil,
	)

	successResponse := httptest.NewRecorder()

	router.ServeHTTP(
		successResponse,
		successRequest,
	)

	notFoundRequest := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/products/999999",
		nil,
	)

	notFoundResponse := httptest.NewRecorder()

	router.ServeHTTP(
		notFoundResponse,
		notFoundRequest,
	)

	successCount := testutil.ToFloat64(
		metrics.requestsTotal.WithLabelValues(
			http.MethodGet,
			"/api/v1/products/{id}",
			"200",
		),
	)

	if successCount != 1 {
		t.Fatalf(
			"expected 1 successful request, got %v",
			successCount,
		)
	}

	notFoundCount := testutil.ToFloat64(
		metrics.requestsTotal.WithLabelValues(
			http.MethodGet,
			"/api/v1/products/{id}",
			"404",
		),
	)

	if notFoundCount != 1 {
		t.Fatalf(
			"expected 1 not found request, got %v",
			notFoundCount,
		)
	}
}

func TestMetricsMiddleware_RecordsRecoveredPanic(
	t *testing.T,
) {
	registry := prometheus.NewRegistry()

	metrics := NewHTTPMetrics(
		registry,
	)

	var logBuffer bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(
			&logBuffer,
			nil,
		),
	)

	router := chi.NewRouter()

	router.Use(
		MetricsMiddleware(
			metrics,
		),
	)

	router.Use(
		Recoverer(
			logger,
		),
	)

	router.Get(
		"/panic",
		func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			panic(
				"test panic",
			)
		},
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/panic",
		nil,
	)

	response := httptest.NewRecorder()

	router.ServeHTTP(
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

	count := testutil.ToFloat64(
		metrics.requestsTotal.WithLabelValues(
			http.MethodGet,
			"/panic",
			"500",
		),
	)

	if count != 1 {
		t.Fatalf(
			"expected 1 panic request, got %v",
			count,
		)
	}

	if !bytes.Contains(
		logBuffer.Bytes(),
		[]byte(`"msg":"http panic"`),
	) {
		t.Fatal(
			"expected panic to be logged",
		)
	}
}
