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

func (f fakeDatabasePinger) Ping(ctx context.Context) error {
	return f.err
}

func TestHealthHandler(t *testing.T) {
	request := httptest.NewRequest(
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
	handler := readinessHandler(
		fakeDatabasePinger{},
	)

	request := httptest.NewRequest(
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
	handler := readinessHandler(
		fakeDatabasePinger{
			err: errors.New("database unavailable"),
		},
	)

	request := httptest.NewRequest(
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
