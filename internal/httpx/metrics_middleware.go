package httpx

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func MetricsMiddleware(
	metrics *HTTPMetrics,
) func(http.Handler) http.Handler {
	return func(
		next http.Handler,
	) http.Handler {
		return http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				startedAt := time.Now()

				responseWriter := middleware.NewWrapResponseWriter(
					w,
					r.ProtoMajor,
				)

				next.ServeHTTP(
					responseWriter,
					r,
				)

				status := responseWriter.Status()
				if status == 0 {
					status = http.StatusOK
				}

				routePattern := chi.RouteContext(
					r.Context(),
				).RoutePattern()

				if routePattern == "" {
					routePattern = "unknown"
				}

				statusString := strconv.Itoa(
					status,
				)

				metrics.requestsTotal.
					WithLabelValues(
						r.Method,
						routePattern,
						statusString,
					).
					Inc()

				metrics.requestDuration.
					WithLabelValues(
						r.Method,
						routePattern,
						statusString,
					).
					Observe(
						time.Since(
							startedAt,
						).Seconds(),
					)
			},
		)
	}
}
