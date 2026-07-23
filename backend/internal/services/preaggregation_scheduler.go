package services

import (
	"context"
	"fmt"
	"log"

	"github.com/robfig/cron/v3"
)

// PreaggregationScheduler manages automated refresh of preaggregated metrics
type PreaggregationScheduler struct {
	preaggService *PreaggregationService
	cron          *cron.Cron
	logger        *log.Logger
	jobs          map[string]cron.EntryID
}

// NewPreaggregationScheduler creates a new scheduler for preaggregation jobs
func NewPreaggregationScheduler(preaggService *PreaggregationService) *PreaggregationScheduler {
	return &PreaggregationScheduler{
		preaggService: preaggService,
		cron:          cron.New(),
		logger:        log.New(log.Writer(), "[SCHEDULER]", log.LstdFlags),
		jobs:          make(map[string]cron.EntryID),
	}
}

// Start begins the scheduler
func (s *PreaggregationScheduler) Start() {
	s.logger.Println("Starting preaggregation scheduler")

	// Schedule daily jobs (6 AM UTC)
	s.scheduleJob("net_irr_daily", "0 6 * * *", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		if err := s.preaggService.PrecomputeNetIRR(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute Net IRR: %v", err)
		}
	})

	s.scheduleJob("xirr_daily", "0 6 * * *", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		if err := s.preaggService.PrecomputeXIRR(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute XIRR: %v", err)
		}
	})

	s.scheduleJob("gross_irr_daily", "0 6 * * *", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		if err := s.preaggService.PrecomputeGrossIRR(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute Gross IRR: %v", err)
		}
	})

	s.scheduleJob("fee_ratio_daily", "0 6 * * *", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		if err := s.preaggService.PrecomputeFeeRatio(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute Fee Ratio: %v", err)
		}
	})

	s.scheduleJob("deployment_pace_daily", "0 6 * * *", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		if err := s.preaggService.PrecomputeDeploymentPace(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute Deployment Pace: %v", err)
		}
	})

	// Schedule weekly jobs (Monday 6 AM UTC)
	s.scheduleJob("gross_moic_weekly", "0 6 * * 1", func() {
		ctx := context.Background()
		grain := []string{"fund_id", "quarter"}
		if err := s.preaggService.PrecomputeGrossMOIC(ctx, grain); err != nil {
			s.logger.Printf("Failed to precompute Gross MOIC: %v", err)
		}
	})

	s.cron.Start()
	s.logger.Println("Preaggregation scheduler started successfully")
}

// Stop stops the scheduler
func (s *PreaggregationScheduler) Stop() {
	s.logger.Println("Stopping preaggregation scheduler")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Println("Preaggregation scheduler stopped")
}

// scheduleJob adds a job to the scheduler
func (s *PreaggregationScheduler) scheduleJob(name, schedule string, job func()) {
	id, err := s.cron.AddFunc(schedule, job)
	if err != nil {
		s.logger.Printf("Failed to schedule job %s: %v", name, err)
		return
	}

	s.jobs[name] = id
	s.logger.Printf("Scheduled job: %s (%s)", name, schedule)
}

// RunJobManually executes a job immediately for testing
func (s *PreaggregationScheduler) RunJobManually(jobName string) error {
	switch jobName {
	case "net_irr_daily":
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		return s.preaggService.PrecomputeNetIRR(ctx, grain)

	case "xirr_daily":
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		return s.preaggService.PrecomputeXIRR(ctx, grain)

	case "gross_irr_daily":
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		return s.preaggService.PrecomputeGrossIRR(ctx, grain)

	case "gross_moic_weekly":
		ctx := context.Background()
		grain := []string{"fund_id", "quarter"}
		return s.preaggService.PrecomputeGrossMOIC(ctx, grain)

	case "fee_ratio_daily":
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		return s.preaggService.PrecomputeFeeRatio(ctx, grain)

	case "deployment_pace_daily":
		ctx := context.Background()
		grain := []string{"fund_id", "month"}
		return s.preaggService.PrecomputeDeploymentPace(ctx, grain)

	default:
		return fmt.Errorf("unknown job: %s", jobName)
	}
}

// GetJobStatus returns the status of scheduled jobs
func (s *PreaggregationScheduler) GetJobStatus() map[string]interface{} {
	status := make(map[string]interface{})

	for name, id := range s.jobs {
		entry := s.cron.Entry(id)
		status[name] = map[string]interface{}{
			"next_run": entry.Next,
			"last_run": entry.Prev,
			"schedule": "configured",
		}
	}

	return status
}

// PreaggregationJobRunner provides a simple interface for running preaggregation jobs
type PreaggregationJobRunner struct {
	scheduler *PreaggregationScheduler
}

// NewPreaggregationJobRunner creates a new job runner
func NewPreaggregationJobRunner(scheduler *PreaggregationScheduler) *PreaggregationJobRunner {
	return &PreaggregationJobRunner{
		scheduler: scheduler,
	}
}

// RunAllDailyJobs executes all daily preaggregation jobs
func (r *PreaggregationJobRunner) RunAllDailyJobs() error {
	jobs := []string{
		"net_irr_daily",
		"xirr_daily",
		"gross_irr_daily",
		"fee_ratio_daily",
		"deployment_pace_daily",
	}

	for _, job := range jobs {
		r.scheduler.logger.Printf("Running daily job: %s", job)
		if err := r.scheduler.RunJobManually(job); err != nil {
			r.scheduler.logger.Printf("Failed to run job %s: %v", job, err)
			// Continue with other jobs even if one fails
		}
	}

	return nil
}

// RunAllWeeklyJobs executes all weekly preaggregation jobs
func (r *PreaggregationJobRunner) RunAllWeeklyJobs() error {
	jobs := []string{
		"gross_moic_weekly",
	}

	for _, job := range jobs {
		r.scheduler.logger.Printf("Running weekly job: %s", job)
		if err := r.scheduler.RunJobManually(job); err != nil {
			r.scheduler.logger.Printf("Failed to run job %s: %v", job, err)
		}
	}

	return nil
}
