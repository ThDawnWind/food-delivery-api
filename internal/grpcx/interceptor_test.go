package grpcx

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestLoggingInterceptor(t *testing.T) {
	t.Parallel()

	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	interceptor := LoggingInterceptor(logger)

	invokerCalled := false

	invoker := func(
		_ context.Context,
		method string,
		req any,
		reply any,
		_ *grpc.ClientConn,
		_ ...grpc.CallOption,
	) error {
		invokerCalled = true

		if method != "/test.Service/TestMethod" {
			t.Fatalf(
				"method = %q, want %q",
				method,
				"/test.Service/TestMethod",
			)
		}

		if req != "test-request" {
			t.Fatalf(
				"request = %v, want %v",
				req,
				"test-request",
			)
		}

		return nil
	}

	err := interceptor(
		context.Background(),
		"/test.Service/TestMethod",
		"test-request",
		new(string),
		nil,
		invoker,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !invokerCalled {
		t.Fatal("expected invoker to be called")
	}
}

func TestLoggingInterceptor_ReturnsInvokerError(t *testing.T) {
	t.Parallel()

	logger := slog.New(
		slog.NewTextHandler(
			io.Discard,
			nil,
		),
	)

	interceptor := LoggingInterceptor(logger)

	invoker := func(
		_ context.Context,
		_ string,
		_ any,
		_ any,
		_ *grpc.ClientConn,
		_ ...grpc.CallOption,
	) error {
		return status.Error(
			codes.Unavailable,
			"delivery service unavailable",
		)
	}

	err := interceptor(
		context.Background(),
		"/test.Service/TestMethod",
		"test-request",
		new(string),
		nil,
		invoker,
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if status.Code(err) != codes.Unavailable {
		t.Fatalf(
			"status code = %s, want %s",
			status.Code(err),
			codes.Unavailable,
		)
	}
}
