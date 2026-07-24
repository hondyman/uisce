package scheduler_intelligence

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DriftWatcher monitors jobs for semantic drift
type DriftWatcher struct {
	repo           *Repository
	semanticClient SemanticClient
	logger         *zap.Logger
	stopChan       chan struct{}
}

// NewDriftWatcher creates a new drift watcher
func NewDriftWatcher(repo *Repository, semanticClient SemanticClient, logger *zap.Logger) *DriftWatcher {
	return &DriftWatcher{
		repo:           repo,
		semanticClient: semanticClient,
		logger:         logger,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the monitoring loop
func (w *DriftWatcher) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	w.logger.Sugar().Infof("starting semantic drift watcher with interval %v", interval)

	go func() {
		for {
			select {
			case <-ticker.C:
				w.checkDrift(ctx)
			case <-w.stopChan:
				ticker.Stop()
				return
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop stops the monitoring loop
func (w *DriftWatcher) Stop() {
	close(w.stopChan)
}

func (w *DriftWatcher) checkDrift(ctx context.Context) {
	w.logger.Info("performing semantic drift check")

	// 1. List all active jobs with semantic bindings
	// For now, we fetch all active jobs. Ideally, we filter in DB.
	jobs, _, err := w.repo.ListJobs(ctx, JobListFilters{IsActive: boolPtr(true), Limit: 1000})
	if err != nil {
		w.logger.Sugar().Errorf("failed to list jobs for drift check: %v", err)
		return
	}

	for _, job := range jobs {
		w.processJobDrift(ctx, job)
	}
}

func (w *DriftWatcher) processJobDrift(ctx context.Context, job Job) {
	// Collect all semantic IDs
	var ids []string
	ids = append(ids, job.SemanticBindings.BOIDs...)
	ids = append(ids, job.SemanticBindings.APIIDs...)
	ids = append(ids, job.SemanticBindings.PageIDs...)
	ids = append(ids, job.SemanticBindings.WorkflowIDs...)
	ids = append(ids, job.SemanticBindings.PreAggIDs...)

	if len(ids) == 0 {
		return
	}

	// 2. Query drift status
	status, err := w.semanticClient.GetDriftStatus(ctx, ids)
	if err != nil {
		w.logger.Sugar().Warnf("failed to get drift status for job %s: %v", job.ID, err)
		return
	}

	// 3. Handle detected drift
	if len(status) > 0 {
		w.logger.Sugar().Warnf("detected semantic drift for job %s: %d issues", job.ID, len(status))
		w.triggerDriftSuggestion(ctx, job, status)
	}
}

func (w *DriftWatcher) triggerDriftSuggestion(ctx context.Context, job Job, issues []DriftStatus) {
	// Create an AI Suggestion for the drift
	title := fmt.Sprintf("Semantic Drift Detected: %s", job.Name)
	desc := "The following semantic objects used by this job have changed or drifted:\n"
	for _, issue := range issues {
		desc += fmt.Sprintf("- %s: [%s] %s\n", issue.SemanticID, issue.Severity, issue.Message)
	}

	suggestion := &AISuggestion{
		ID:             uuid.New(),
		TenantID:       *job.TenantID,
		SuggestionType: "semantic_drift",
		TargetType:     stringPtr("JOB"),
		TargetID:       &job.ID,
		Title:          title,
		Description:    desc,
		RiskLevel:      "HIGH",
		Status:         "PENDING",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := w.repo.CreateAISuggestion(ctx, suggestion); err != nil {
		w.logger.Sugar().Errorf("failed to save drift suggestion: %v", err)
	}
}
