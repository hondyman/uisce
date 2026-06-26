// scheduler/daily_etl.go
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/jmoiron/sqlx"
	"github.com/robfig/cron/v3"
	"golang.org/x/sync/errgroup"
)

// Tenant represents an active tenant workspace
type Tenant struct {
	ID   uuid.UUID
	Name string
}

// TenantRepository abstracts fetching workspaces
type TenantRepository interface {
	ListActiveTenants(ctx context.Context) ([]Tenant, error)
}

// DailyETLScheduler orchestrates daily ETL pipelines per tenant
// Aligns with Usice Architecture §6.2: Multi-tenant Enforcement
type DailyETLScheduler struct {
	cron           *cron.Cron
	complianceSvc  *services.ComplianceService
	riskSvc        *services.RiskService
	audit          audit.Logger
	tenantRepo     TenantRepository
	db             *sqlx.DB
	maxConcurrency int
}

// NewDailyETLScheduler creates a new scheduler instance
func NewDailyETLScheduler(
	complianceSvc *services.ComplianceService,
	riskSvc *services.RiskService,
	audit audit.Logger,
	tenantRepo TenantRepository,
	db *sqlx.DB,
	maxConcurrency int,
) *DailyETLScheduler {
	return &DailyETLScheduler{
		cron:           cron.New(cron.WithSeconds()),
		complianceSvc:  complianceSvc,
		riskSvc:        riskSvc,
		audit:          audit,
		tenantRepo:     tenantRepo,
		db:             db,
		maxConcurrency: maxConcurrency,
	}
}

// Start begins the scheduler with daily cron jobs
func (s *DailyETLScheduler) Start(ctx context.Context) error {
	// Daily at 2:00 AM UTC per tenant
	_, err := s.cron.AddFunc("0 0 2 * * *", func() {
		s.runDailyETL(context.Background())
	})
	if err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	s.cron.Start()
	s.audit.Log(ctx, audit.SchedulerStarted{
		Service: "DailyETLScheduler",
		Cron:    "0 0 2 * * *",
	})
	return nil
}

// Stop gracefully shuts down the scheduler
func (s *DailyETLScheduler) Stop(ctx context.Context) error {
	s.cron.Stop()
	s.audit.Log(ctx, audit.SchedulerStopped{
		Service: "DailyETLScheduler",
	})
	return nil
}

// runDailyETL executes ETL pipelines for all active tenants
func (s *DailyETLScheduler) runDailyETL(ctx context.Context) {
	start := time.Now()

	// 1. Load active tenants (Usice Architecture §6.2)
	tenants, err := s.tenantRepo.ListActiveTenants(ctx)
	if err != nil {
		s.audit.Log(ctx, audit.SchedulerError{
			Error: fmt.Errorf("list active tenants: %w", err).Error(),
		})
		return
	}

	// 2. Process tenants with concurrency limit
	sem := make(chan struct{}, s.maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for _, t := range tenants {
		sem <- struct{}{} // Acquire semaphore
		wg.Add(1)

		go func(tenant Tenant) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Create tenant-scoped context
			tenantCtx := context.WithValue(ctx, "tenant_id", tenant.ID.String())
			valuationDate := time.Now().AddDate(0, 0, -1).Format("2006-01-02") // T-1

			// 3. Track Compliance Run (Phase 18)
			compRunID := s.startEtlRun(tenantCtx, tenant.ID, valuationDate, "COMPLIANCE")
			compErr := s.runComplianceETL(tenantCtx, tenant.ID, valuationDate)
			s.endEtlRun(tenantCtx, compRunID, compErr)

			if compErr != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("tenant %s compliance: %w", tenant.ID, compErr))
				mu.Unlock()
			}

			// 4. Track Risk Run (Phase 18)
			riskRunID := s.startEtlRun(tenantCtx, tenant.ID, valuationDate, "RISK")
			riskErr := s.runRiskETL(tenantCtx, tenant.ID, valuationDate)
			s.endEtlRun(tenantCtx, riskRunID, riskErr)

			if riskErr != nil {
				mu.Lock()
				errors = append(errors, fmt.Errorf("tenant %s risk: %w", tenant.ID, riskErr))
				mu.Unlock()
			}
		}(t)
	}

	wg.Wait()

	// 3. Audit completion
	duration := time.Since(start)
	s.audit.Log(ctx, audit.DailyETLCompleted{
		TenantsProcessed: len(tenants),
		Errors:           len(errors),
		DurationMs:       duration.Milliseconds(),
	})

	if len(errors) > 0 {
		s.audit.Log(ctx, audit.SchedulerError{
			Error: fmt.Sprintf("%d errors during ETL: %v", len(errors), errors),
		})
	}
}

func (s *DailyETLScheduler) runComplianceETL(ctx context.Context, tenantID uuid.UUID, valuationDate string) error {
	// Get portfolios for tenant
	portfolios, err := s.complianceSvc.ListPortfolios(ctx, tenantID)
	if err != nil {
		return err
	}

	var g errgroup.Group
	g.SetLimit(10) // Per-portfolio concurrency

	for _, portfolio := range portfolios {
		portfolio := portfolio // Capture loop variable
		g.Go(func() error {
			return s.complianceSvc.EvaluatePortfolio(ctx, portfolio.ID, valuationDate)
		})
	}

	return g.Wait()
}

func (s *DailyETLScheduler) runRiskETL(ctx context.Context, tenantID uuid.UUID, valuationDate string) error {
	portfolios, err := s.riskSvc.ListPortfolios(ctx, tenantID)
	if err != nil {
		return err
	}

	var g errgroup.Group
	g.SetLimit(10)

	for _, portfolio := range portfolios {
		portfolio := portfolio
		g.Go(func() error {
			return s.riskSvc.ComputePortfolioRisk(ctx, portfolio.ID, valuationDate)
		})
	}

	return g.Wait()
}

// startEtlRun initializes the tracking row for Phase 18 Operational Intelligence
func (s *DailyETLScheduler) startEtlRun(ctx context.Context, tenantID uuid.UUID, valuationDate, engineName string) uuid.UUID {
	runID := uuid.New()
	query := `
		INSERT INTO edm.etl_run (run_id, tenant_id, valuation_date, engine_name, status)
		VALUES ($1, $2, $3, $4, 'RUNNING')
	`
	_, err := s.db.ExecContext(ctx, query, runID, tenantID, valuationDate, engineName)
	if err != nil {
		s.audit.Log(ctx, audit.SchedulerError{Error: fmt.Sprintf("failed to insert start_run: %v", err)})
	}
	return runID
}

// endEtlRun finalizes the run record based on execution error status
func (s *DailyETLScheduler) endEtlRun(ctx context.Context, runID uuid.UUID, runErr error) {
	status := "SUCCESS"
	errMsg := ""
	if runErr != nil {
		status = "FAILED"
		errMsg = runErr.Error()
	}

	query := `
		UPDATE edm.etl_run
		SET status = $1, error_message = $2, end_time = NOW()
		WHERE run_id = $3
	`
	_, err := s.db.ExecContext(ctx, query, status, errMsg, runID)
	if err != nil {
		s.audit.Log(ctx, audit.SchedulerError{Error: fmt.Sprintf("failed to update end_run: %v", err)})
	}
}
