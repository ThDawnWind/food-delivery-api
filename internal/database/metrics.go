package database

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type PoolMetrics struct {
	pool *pgxpool.Pool

	acquiredConns *prometheus.Desc
	idleConns     *prometheus.Desc
	totalConns    *prometheus.Desc
	maxConns      *prometheus.Desc

	emptyAcquireCount *prometheus.Desc
}

func NewPoolMetrics(
	pool *pgxpool.Pool,
) *PoolMetrics {
	return &PoolMetrics{
		pool: pool,

		acquiredConns: prometheus.NewDesc(
			"db_pool_acquired_connections",
			"Number of currently acquired database connections.",
			nil,
			nil,
		),

		idleConns: prometheus.NewDesc(
			"db_pool_idle_connections",
			"Number of currently idle database connections.",
			nil,
			nil,
		),

		totalConns: prometheus.NewDesc(
			"db_pool_total_connections",
			"Total number of database connections in the pool.",
			nil,
			nil,
		),

		maxConns: prometheus.NewDesc(
			"db_pool_max_connections",
			"Maximum number of database connections allowed in the pool.",
			nil,
			nil,
		),

		emptyAcquireCount: prometheus.NewDesc(
			"db_pool_empty_acquire_total",
			"Number of successful connection acquires that had to wait because the pool was empty.",
			nil,
			nil,
		),
	}
}

func (m *PoolMetrics) Describe(
	ch chan<- *prometheus.Desc,
) {
	ch <- m.acquiredConns
	ch <- m.idleConns
	ch <- m.totalConns
	ch <- m.maxConns
	ch <- m.emptyAcquireCount
}

func (m *PoolMetrics) Collect(
	ch chan<- prometheus.Metric,
) {
	stats := m.pool.Stat()

	ch <- prometheus.MustNewConstMetric(
		m.acquiredConns,
		prometheus.GaugeValue,
		float64(
			stats.AcquiredConns(),
		),
	)

	ch <- prometheus.MustNewConstMetric(
		m.idleConns,
		prometheus.GaugeValue,
		float64(
			stats.IdleConns(),
		),
	)

	ch <- prometheus.MustNewConstMetric(
		m.totalConns,
		prometheus.GaugeValue,
		float64(
			stats.TotalConns(),
		),
	)

	ch <- prometheus.MustNewConstMetric(
		m.maxConns,
		prometheus.GaugeValue,
		float64(
			stats.MaxConns(),
		),
	)

	ch <- prometheus.MustNewConstMetric(
		m.emptyAcquireCount,
		prometheus.CounterValue,
		float64(
			stats.EmptyAcquireCount(),
		),
	)
}
