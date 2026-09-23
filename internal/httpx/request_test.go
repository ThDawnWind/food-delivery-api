package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRequest struct {
	Name string `json:"name"`
}

func TestDecodeJSON(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(`{"name":"test"}`),
	)

	recorder := httptest.NewRecorder()

	var body testRequest

	err := DecodeJSON(
		recorder,
		request,
		&body,
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if body.Name != "test" {
		t.Fatalf(
			"expected name %q, got %q",
			"test",
			body.Name,
		)
	}
}

func TestDecodeJSONUnknownField(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(
			`{"name":"test","unknown":"value"}`,
		),
	)

	recorder := httptest.NewRecorder()

	var body testRequest

	err := DecodeJSON(
		recorder,
		request,
		&body,
	)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestDecodeJSONMultipleValues(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(
			`{"name":"first"} {"name":"second"}`,
		),
	)

	recorder := httptest.NewRecorder()

	var body testRequest

	err := DecodeJSON(
		recorder,
		request,
		&body,
	)
	if err == nil {
		t.Fatal("expected error for multiple JSON values, got nil")
	}
}

func TestDecodeJSONEmptyBody(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		nil,
	)

	recorder := httptest.NewRecorder()

	var body testRequest

	err := DecodeJSON(
		recorder,
		request,
		&body,
	)
	if err == nil {
		t.Fatal("expected error for empty body, got nil")
	}
}

func TestDecodeJSONBodyTooLarge(t *testing.T) {
	payload := `{"name":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}`

	request := httptest.NewRequest(
		http.MethodPost,
		"/",
		strings.NewReader(payload),
	)

	recorder := httptest.NewRecorder()

	var body testRequest

	err := DecodeJSON(
		recorder,
		request,
		&body,
	)
	if err == nil {
		t.Fatal("expected error for request body exceeding limit, got nil")
	}
}
