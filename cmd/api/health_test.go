package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDatabasePinger struct {
	err error
}

func (f fakeDatabasePinger) Ping(_ context.Context) error {
	return f.err
}

func TestHealthHandler(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/health",
		nil,
	)

	recorder := httptest.NewRecorder()

	healthHandler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if recorder.Body.String() != "OK" {
		t.Fatalf(
			"expected body %q, got %q",
			"OK",
			recorder.Body.String(),
		)
	}
}

func TestReadinessHandlerReady(t *testing.T) {
	t.Parallel()

	handler := readinessHandler(
		fakeDatabasePinger{},
	)

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/ready",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			recorder.Code,
		)
	}

	if recorder.Body.String() != "OK" {
		t.Fatalf(
			"expected body %q, got %q",
			"OK",
			recorder.Body.String(),
		)
	}
}

func TestReadinessHandlerDatabaseUnavailable(t *testing.T) {
	t.Parallel()

	handler := readinessHandler(
		fakeDatabasePinger{
			err: errors.New("database unavailable"),
		},
	)

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/ready",
		nil,
	)

	recorder := httptest.NewRecorder()

	handler(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusServiceUnavailable,
			recorder.Code,
		)
	}
}
