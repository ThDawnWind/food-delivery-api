package httpx

import (
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func RequestLogger(logger *slog.Logger, trustedProxyCIDRs []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				start := time.Now()

				ww := middleware.NewWrapResponseWriter(
					w,
					r.ProtoMajor,
				)

				next.ServeHTTP(ww, r)

				status := ww.Status()
				if status == 0 {
					status = http.StatusOK
				}

				logger.Info(
					"http request",
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
					slog.Int(
						"status",
						status,
					),
					slog.Int64(
						"duration_ms",
						time.Since(start).Milliseconds(),
					),
					slog.String(
						"client_ip",
						clientIP(
							r,
							trustedProxyCIDRs,
						),
					),
				)
			},
		)
	}
}
