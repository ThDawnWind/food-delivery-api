package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/go-chi/chi/v5/middleware"
)

func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				defer func() {
					recovered := recover()
					if recovered == nil {
						return
					}

					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}

					logger.ErrorContext(
						r.Context(),
						"http panic",
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
							"panic",
							recovered,
						),
						slog.String(
							"stack",
							string(
								debug.Stack(),
							),
						),
					)

					w.WriteHeader(
						http.StatusInternalServerError,
					)
				}()

				next.ServeHTTP(
					w,
					r,
				)
			},
		)
	}
}
