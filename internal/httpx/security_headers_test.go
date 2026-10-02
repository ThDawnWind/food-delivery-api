package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		},
	)

	handler := SecurityHeaders(next)

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/test",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler.ServeHTTP(
		recorder,
		request,
	)

	tests := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "no-referrer",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "frame-ancestors 'none'",
	}

	for header, expected := range tests {
		actual := recorder.Header().Get(header)

		if actual != expected {
			t.Errorf(
				"expected %s=%q, got %q",
				header,
				expected,
				actual,
			)
		}
	}
}
