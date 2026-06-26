package orchestration

// =====================================================
// Metric Orchestration Service
// Schedules and coordinates dual-path execution
// =====================================================

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// OrchestrationConfig defines scheduling parameters
type OrchestrationConfig struct {
	// Real-time lane
	AtomicRefreshInterval time.Duration // e.g., 1 hour
	AtomicRefreshTimeout  time.Duration // e.g., 30 minutes

	// Batch lane
	MonthlyPoPSchedule string // cron expression or fixed time
	MonthlyPoPTimeout  time.Duration

	// Anomaly detection
	AnomalyDetectionSchedule string
	AnomalyDetectionTimeout  time.Duration

	// SLA enforcement
	SLACheckInterval time.Duration // e.g., 6 hours

	// Defaults for detection
	DefaultZScoreThreshold float64
	DefaultWindowDays      int
	DefaultMinDataPoints   int
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *OrchestrationConfig {
	return &OrchestrationConfig{
		AtomicRefreshInterval:    1 * time.Hour,
		AtomicRefreshTimeout:     30 * time.Minute,
		MonthlyPoPSchedule:       "0 2 1 * *", // 2 AM on 1st of month
		MonthlyPoPTimeout:        1 * time.Hour,
		AnomalyDetectionSchedule: "0 3 * * *", // 3 AM daily
		AnomalyDetectionTimeout:  1 * time.Hour,
		SLACheckInterval:         6 * time.Hour,
		DefaultZScoreThreshold:   2.5,
		DefaultWindowDays:        90,
		DefaultMinDataPoints:     7,
	}
}

// MetricOrchestrator manages execution scheduling and coordination
type MetricOrchestrator struct {
	registryService *services.MetricRegistryService
	config          *OrchestrationConfig
	done            chan bool
	ticker          *time.Ticker
}

// NewMetricOrchestrator creates a new orchestrator
func NewMetricOrchestrator(
	registryService *services.MetricRegistryService,
	config *OrchestrationConfig,
) *MetricOrchestrator {
	if config == nil {
		config = DefaultConfig()
	}

	return &MetricOrchestrator{
		registryService: registryService,
		config:          config,
		done:            make(chan bool, 1),
	}
}

// Start begins orchestration and scheduling
func (o *MetricOrchestrator) Start(ctx context.Context) {
	log.Println("[Orchestrator] Starting metric orchestration engine...")

	// Start real-time atomic refresh ticker
	go o.scheduleAtomicRefresh(ctx)

	// Start monthly PoP batch ticker
	go o.scheduleMonthlyPoP(ctx)

	// Start anomaly detection ticker
	go o.scheduleAnomalyDetection(ctx)

	// Start SLA enforcement ticker
	go o.enforceSLAs(ctx)

	log.Println("[Orchestrator] All schedulers started")
}

// Stop stops the orchestrator
func (o *MetricOrchestrator) Stop() {
	log.Println("[Orchestrator] Stopping...")
	o.done <- true
	if o.ticker != nil {
		o.ticker.Stop()
	}
}

// scheduleAtomicRefresh runs real-time atomic metric refresh
func (o *MetricOrchestrator) scheduleAtomicRefresh(ctx context.Context) {
	ticker := time.NewTicker(o.config.AtomicRefreshInterval)
	defer ticker.Stop()

	log.Printf("[Orchestrator] Real-time atomic refresh scheduled every %v\n", o.config.AtomicRefreshInterval)

	for {
		select {
		case <-o.done:
			log.Println("[Orchestrator] Stopping atomic refresh scheduler")
			return
		case <-ticker.C:
			log.Println("[Orchestrator] Executing atomic refresh lane...")

			ctxWithTimeout, cancel := context.WithTimeout(ctx, o.config.AtomicRefreshTimeout)
			logs, err := o.registryService.RefreshAtomicMetrics(ctxWithTimeout, nil)
			cancel()

			if err != nil {
				log.Printf("[Orchestrator] ERROR in atomic refresh: %v\n", err)
			} else {
				successCount := 0
				for _, log := range logs {
					if log.Status == "completed" {
						successCount++
					}
				}
				log.Printf("[Orchestrator] Atomic refresh completed: %d/%d successful\n", successCount, len(logs))
			}
		}
	}
}

// scheduleMonthlyPoP runs the batch PoP computation (monthly)
func (o *MetricOrchestrator) scheduleMonthlyPoP(ctx context.Context) {
	// For simplicity, run at 2 AM on the 1st of each month
	// In production, use a proper cron scheduler (e.g., robfig/cron)

	for {
		select {
		case <-o.done:
			log.Println("[Orchestrator] Stopping PoP scheduler")
			return
		default:
			now := time.Now()

			// Check if it's the 1st of the month and past 2 AM
			if now.Day() == 1 && now.Hour() >= 2 {
				log.Println("[Orchestrator] Executing monthly PoP computation batch...")

				ctxWithTimeout, cancel := context.WithTimeout(ctx, o.config.MonthlyPoPTimeout)

				// Compute PoP for all metrics
				execLog, err := o.registryService.ComputeMonthlyPoP(ctxWithTimeout, nil, nil, nil)
				cancel()

				if err != nil {
					log.Printf("[Orchestrator] ERROR in PoP computation: %v\n", err)
				} else {
					log.Printf("[Orchestrator] PoP computation completed: %s\n", execLog.Status)

					// Compute comparison periods
					ctxWithTimeout2, cancel2 := context.WithTimeout(ctx, o.config.MonthlyPoPTimeout)
					_, err2 := o.registryService.ComputeComparisonPeriods(ctxWithTimeout2, nil)
					cancel2()

					if err2 != nil {
						log.Printf("[Orchestrator] ERROR in comparison periods: %v\n", err2)
					} else {
						log.Println("[Orchestrator] Comparison periods computed")
					}
				}

				// Wait until next day to avoid re-running
				time.Sleep(24 * time.Hour)
			} else {
				// Sleep until next check (every 5 minutes)
				time.Sleep(5 * time.Minute)
			}
		}
	}
}

// scheduleAnomalyDetection runs z-score anomaly detection
func (o *MetricOrchestrator) scheduleAnomalyDetection(ctx context.Context) {
	// Run at 3 AM daily
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	log.Println("[Orchestrator] Anomaly detection scheduled daily at 3 AM")

	// Calculate initial delay to next 3 AM
	now := time.Now()
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if nextRun.Before(now) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	time.Sleep(time.Until(nextRun))

	for {
		select {
		case <-o.done:
			log.Println("[Orchestrator] Stopping anomaly detection scheduler")
			return
		case <-ticker.C:
			log.Println("[Orchestrator] Executing anomaly detection batch...")

			ctxWithTimeout, cancel := context.WithTimeout(ctx, o.config.AnomalyDetectionTimeout)

			// Detect anomalies for all metrics
			anomalies, err := o.registryService.DetectZScoreAnomalies(
				ctxWithTimeout,
				nil,
				o.config.DefaultZScoreThreshold,
				o.config.DefaultWindowDays,
				o.config.DefaultMinDataPoints,
			)
			cancel()

			if err != nil {
				log.Printf("[Orchestrator] ERROR in anomaly detection: %v\n", err)
			} else {
				log.Printf("[Orchestrator] Anomaly detection completed: %d anomalies detected\n", len(anomalies))
			}
		}
	}
}

// enforceSLAs checks and enforces SLA compliance for golden path metrics
func (o *MetricOrchestrator) enforceSLAs(ctx context.Context) {
	ticker := time.NewTicker(o.config.SLACheckInterval)
	defer ticker.Stop()

	log.Printf("[Orchestrator] SLA enforcement scheduled every %v\n", o.config.SLACheckInterval)

	for {
		select {
		case <-o.done:
			log.Println("[Orchestrator] Stopping SLA enforcement")
			return
		case <-ticker.C:
			log.Println("[Orchestrator] Checking golden path metrics SLA compliance...")

			readiness, err := o.registryService.GetGoldenPathReadiness(ctx)
			if err != nil {
				log.Printf("[Orchestrator] ERROR checking SLA compliance: %v\n", err)
				continue
			}

			breaches := 0
			for _, r := range readiness {
				readyMap := r
				if status, ok := readyMap["readiness_status"].(string); ok && status != "ready" {
					breaches++
					if name, ok := readyMap["name"].(string); ok {
						log.Printf("[Orchestrator] SLA BREACH: metric=%s status=%s\n", name, status)
					}
				}
			}

			if breaches > 0 {
				log.Printf("[Orchestrator] Found %d golden path metrics with SLA breaches\n", breaches)
				// TODO: Trigger alerts, notifications, escalations
			} else {
				log.Println("[Orchestrator] All golden path metrics are SLA compliant")
			}
		}
	}
}

// ExecuteMetricJob manually triggers a job for a specific metric
func (o *MetricOrchestrator) ExecuteMetricJob(ctx context.Context, metricID uuid.UUID, jobType string) error {
	switch jobType {
	case "atomic_refresh":
		_, err := o.registryService.RefreshAtomicMetrics(ctx, &metricID)
		return err

	case "pop_computation":
		_, err := o.registryService.ComputeMonthlyPoP(ctx, &metricID, nil, nil)
		return err

	case "comparison_periods":
		_, err := o.registryService.ComputeComparisonPeriods(ctx, &metricID)
		return err

	case "anomaly_detection":
		_, err := o.registryService.DetectZScoreAnomalies(
			ctx,
			&metricID,
			o.config.DefaultZScoreThreshold,
			o.config.DefaultWindowDays,
			o.config.DefaultMinDataPoints,
		)
		return err

	default:
		return fmt.Errorf("unknown job type: %s", jobType)
	}
}

// GetStatus returns orchestrator status and scheduling info
func (o *MetricOrchestrator) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"status":                     "running",
		"atomic_refresh_interval":    o.config.AtomicRefreshInterval.String(),
		"pop_computation_schedule":   o.config.MonthlyPoPSchedule,
		"anomaly_detection_schedule": o.config.AnomalyDetectionSchedule,
		"sla_check_interval":         o.config.SLACheckInterval.String(),
		"started_at":                 time.Now(),
	}
}
