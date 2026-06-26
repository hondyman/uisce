package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HasuraInsertTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "hasura_insert_total", Help: "Hasura inserts"},
		[]string{"status"},
	)
	HasuraInsertLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "hasura_insert_ms", Help: "Hasura insert latency (ms)",
		Buckets: prometheus.ExponentialBuckets(5, 2, 8),
	})
)

// ObserveInsert records latency and increments the appropriate counter.
func ObserveInsert(start time.Time, err error) {
	ms := float64(time.Since(start).Milliseconds())
	HasuraInsertLatency.Observe(ms)
	status := "success"
	if err != nil {
		status = "error"
	}
	HasuraInsertTotal.WithLabelValues(status).Inc()
}
