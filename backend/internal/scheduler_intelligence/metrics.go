package scheduler_intelligence

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// SLOBreaches tracks the total number of SLO breaches
	SLOBreaches = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "scheduler_slo_breaches_total",
		Help: "The total number of SLO breaches",
	}, []string{"tenant_id", "job_id", "category"})

	// JobLatency tracks the duration of job runs
	JobLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "scheduler_job_duration_seconds",
		Help:    "Latency of job runs in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"tenant_id", "job_id", "category", "status"})

	// ErrorBudgetConsumption tracks how much of the SLO error budget has been used
	ErrorBudgetConsumption = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scheduler_error_budget_percentage",
		Help: "Current error budget consumption percentage",
	}, []string{"tenant_id", "job_id"})

	// ScheduledJobsTotal tracks the total number of managed jobs
	ScheduledJobsTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "scheduler_jobs_total",
		Help: "Total number of scheduled jobs",
	}, []string{"tenant_id", "scope", "is_active"})
)

// RecordSLOBreach increments the SLO breach counter
func RecordSLOBreach(tenantID, jobID, category string) {
	SLOBreaches.WithLabelValues(tenantID, jobID, category).Inc()
}

// RecordJobLatency records the duration of a job run
func RecordJobLatency(tenantID, jobID, category, status string, durationSeconds float64) {
	JobLatency.WithLabelValues(tenantID, jobID, category, status).Observe(durationSeconds)
}

// UpdateErrorBudget updates the error budget gauge
func UpdateErrorBudget(tenantID, jobID string, percentage float64) {
	ErrorBudgetConsumption.WithLabelValues(tenantID, jobID).Set(percentage)
}
