package analytics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	RuleEvaluationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rule_evaluations_total",
			Help: "Total count of validation rule evaluations partitioned by surface, domain, and severity.",
		},
		[]string{"surface", "domain", "severity"},
	)

	RuleEvalDurationSeconds = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rule_eval_duration_seconds",
			Help:    "Latency of validation rule evaluations in seconds.",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"surface"},
	)

	RulePushdownRowsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rule_pushdown_rows_total",
			Help: "Total rows evaluated via SQL pushdown partitioned by table and outcome (pass, violation, error).",
		},
		[]string{"table", "outcome"},
	)

	RuleImportOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rule_import_operations_total",
			Help: "Total rule import/export GitOps operations by status (success, error, dry_run).",
		},
		[]string{"status"},
	)

	RuleCacheStaleServesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rule_cache_stale_serves_total",
			Help: "Total evaluations served from stale in-memory snapshot cache during background refresh errors.",
		},
	)

	RuleCacheRefreshErrorsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "rule_cache_refresh_errors_total",
			Help: "Total errors encountered while refreshing in-memory rule snapshots from catalog.",
		},
	)
)
