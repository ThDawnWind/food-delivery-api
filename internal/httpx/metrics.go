package httpx

import (
	"github.com/prometheus/client_golang/prometheus"
)

type HTTPMetrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

func NewHTTPMetrics(
	registerer prometheus.Registerer,
) *HTTPMetrics {
	metrics := &HTTPMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests.",
			},
			[]string{
				"method",
				"path",
				"status",
			},
		),

		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "HTTP request duration in seconds.",
				Buckets: []float64{
					0.005,
					0.01,
					0.025,
					0.05,
					0.1,
					0.25,
					0.5,
					1,
					2.5,
					5,
				},
			},
			[]string{
				"method",
				"path",
				"status",
			},
		),
	}

	registerer.MustRegister(
		metrics.requestsTotal,
		metrics.requestDuration,
	)

	return metrics
}
