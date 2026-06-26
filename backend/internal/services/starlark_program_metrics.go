package services

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	starlarkProgramCacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_starlark_program_cache_hits_total",
			Help: "Number of Starlark program cache hits.",
		},
		[]string{"filename"},
	)

	starlarkProgramCacheMissesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_starlark_program_cache_misses_total",
			Help: "Number of Starlark program cache misses (lookup misses; may include misses resolved by another goroutine).",
		},
		[]string{"filename"},
	)

	starlarkProgramCompilesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "semlayer_starlark_program_compiles_total",
			Help: "Number of Starlark program compilation attempts.",
		},
		[]string{"filename", "outcome"},
	)

	starlarkProgramCompileDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "semlayer_starlark_program_compile_duration_seconds",
			Help:    "Wall-clock duration of Starlark program compilation.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"filename"},
	)

	starlarkProgramCacheEntries = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "semlayer_starlark_program_cache_entries",
			Help: "Current number of entries in the Starlark program cache.",
		},
	)

	starlarkProgramMetricsOnce sync.Once
)

func ensureStarlarkProgramMetricsRegistered() {
	starlarkProgramMetricsOnce.Do(func() {
		registerOrReuse := func(c prometheus.Collector) {
			if err := prometheus.Register(c); err != nil {
				if _, ok := err.(prometheus.AlreadyRegisteredError); ok {
					return
				}
			}
		}
		registerOrReuse(starlarkProgramCacheHitsTotal)
		registerOrReuse(starlarkProgramCacheMissesTotal)
		registerOrReuse(starlarkProgramCompilesTotal)
		registerOrReuse(starlarkProgramCompileDurationSeconds)
		registerOrReuse(starlarkProgramCacheEntries)
	})
}

func observeStarlarkProgramCacheHit(filename string) {
	ensureStarlarkProgramMetricsRegistered()
	starlarkProgramCacheHitsTotal.WithLabelValues(filename).Inc()
}

func observeStarlarkProgramCacheMiss(filename string) {
	ensureStarlarkProgramMetricsRegistered()
	starlarkProgramCacheMissesTotal.WithLabelValues(filename).Inc()
}

func observeStarlarkProgramCompile(filename string, d time.Duration, outcome string) {
	ensureStarlarkProgramMetricsRegistered()
	starlarkProgramCompilesTotal.WithLabelValues(filename, outcome).Inc()
	starlarkProgramCompileDurationSeconds.WithLabelValues(filename).Observe(d.Seconds())
}
