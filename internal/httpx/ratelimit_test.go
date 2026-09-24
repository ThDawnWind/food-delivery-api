package httpx

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

func TestIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(
		rate.Every(time.Hour),
		2,
		10*time.Minute,
		nil,
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
		nil,
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

func TestClientIPIgnoresForwardedForFromUntrustedPeer(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	request.RemoteAddr = "203.0.113.10:12345"

	request.Header.Set(
		"X-Forwarded-For",
		"198.51.100.20",
	)

	ip := clientIP(
		request,
		[]netip.Prefix{
			netip.MustParsePrefix(
				"172.18.0.0/16",
			),
		},
	)

	if ip != "203.0.113.10" {
		t.Fatalf(
			"expected remote IP, got %q",
			ip,
		)
	}
}

func TestClientIPUsesForwardedForFromTrustedProxy(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	request.RemoteAddr = "172.18.0.5:12345"

	request.Header.Set(
		"X-Forwarded-For",
		"203.0.113.25",
	)

	ip := clientIP(
		request,
		[]netip.Prefix{
			netip.MustParsePrefix(
				"172.18.0.0/16",
			),
		},
	)

	if ip != "203.0.113.25" {
		t.Fatalf(
			"expected client IP, got %q",
			ip,
		)
	}
}

func TestClientIPSkipsTrustedProxyChain(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	request.RemoteAddr = "172.18.0.5:12345"

	request.Header.Set(
		"X-Forwarded-For",
		"203.0.113.25, 172.18.0.4",
	)

	ip := clientIP(
		request,
		[]netip.Prefix{
			netip.MustParsePrefix(
				"172.18.0.0/16",
			),
		},
	)

	if ip != "203.0.113.25" {
		t.Fatalf(
			"expected client IP, got %q",
			ip,
		)
	}
}

func TestClientIPInvalidForwardedForFallsBackToRemote(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/login",
		nil,
	)

	request.RemoteAddr = "172.18.0.5:12345"

	request.Header.Set(
		"X-Forwarded-For",
		"definitely-not-an-ip",
	)

	ip := clientIP(
		request,
		[]netip.Prefix{
			netip.MustParsePrefix(
				"172.18.0.0/16",
			),
		},
	)

	if ip != "172.18.0.5" {
		t.Fatalf(
			"expected proxy IP fallback, got %q",
			ip,
		)
	}
}
