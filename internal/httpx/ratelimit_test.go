package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(
		rate.Every(time.Hour),
		2,
		10*time.Minute,
	)

	handler := limiter.Middleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		),
	)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(
			http.MethodPost,
			"/login",
			nil,
		)

		req.RemoteAddr = "192.0.2.1:12345"

		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf(
				"expected status %d, got %d",
				http.StatusOK,
				rec.Code,
			)
		}
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	req.RemoteAddr = "192.0.2.1:12345"

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusTooManyRequests,
			rec.Code,
		)
	}
}

func TestIPRateLimiter_SeparateIPs(t *testing.T) {
	limiter := NewIPRateLimiter(
		rate.Every(time.Hour),
		1,
		10*time.Minute,
	)

	handler := limiter.Middleware(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
		),
	)

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)
	firstRequest.RemoteAddr = "192.0.2.1:12345"

	firstRecorder := httptest.NewRecorder()
	handler.ServeHTTP(firstRecorder, firstRequest)

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)
	secondRequest.RemoteAddr = "192.0.2.2:12345"

	secondRecorder := httptest.NewRecorder()
	handler.ServeHTTP(secondRecorder, secondRequest)

	if firstRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected first IP status %d, got %d",
			http.StatusOK,
			firstRecorder.Code,
		)
	}

	if secondRecorder.Code != http.StatusOK {
		t.Fatalf(
			"expected second IP status %d, got %d",
			http.StatusOK,
			secondRecorder.Code,
		)
	}
}
