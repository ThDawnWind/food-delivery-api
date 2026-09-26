package httpx

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

func LogInternalError(
	logger *slog.Logger,
	r *http.Request,
	message string,
	err error,
) {
	logger.ErrorContext(
		r.Context(),
		message,
		slog.String(
			"request_id",
			middleware.GetReqID(
				r.Context(),
			),
		),
		slog.String(
			"method",
			r.Method,
		),
		slog.String(
			"path",
			r.URL.Path,
		),
		slog.Any(
			"error",
			err,
		),
	)
}

func WriteInternalError(
	logger *slog.Logger,
	w http.ResponseWriter,
	r *http.Request,
	message string,
	err error,
) {
	LogInternalError(
		logger,
		r,
		message,
		err,
	)

	WriteError(
		w,
		http.StatusInternalServerError,
		"internal server error",
	)
}
