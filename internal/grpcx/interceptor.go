package grpcx

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func LoggingInterceptor(
	logger *slog.Logger,
) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption) error {
		start := time.Now()

		err := invoker(
			ctx,
			method,
			req,
			reply,
			cc,
			opts...,
		)

		code := status.Code(err)

		logger.Info(
			"gRPC client request completed",
			slog.String("method", method),
			slog.String("code", code.String()),
			slog.Duration("duration", time.Since(start)),
			slog.Any("error", err),
		)

		return err
	}
}
