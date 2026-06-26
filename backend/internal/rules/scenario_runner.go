package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/starlib"
)

// SampleEntityRepository defines how to fetch sample data for testing
type SampleEntityRepository interface {
	SampleEntities(ctx context.Context, tenantID string, entityName string, sampleSize int, filter map[string]interface{}) ([]map[string]interface{}, error)
}

type ScenarioRunner struct {
	repo       ScenarioRepository
	sampleRepo SampleEntityRepository
	engine     *RuleEngine
}

func NewScenarioRunner(
	repo ScenarioRepository,
	sampleRepo SampleEntityRepository,
	engine *RuleEngine,
) *ScenarioRunner {
	return &ScenarioRunner{
		repo:       repo,
		sampleRepo: sampleRepo,
		engine:     engine,
	}
}

func (r *ScenarioRunner) RunScenario(
	ctx context.Context,
	tenantID string,
	scenarioVersionID string,
	entity string,
	sampleSize int,
	filter map[string]interface{},
) (*RuleTestRun, error) {
	// 1. Load scenario version
	sv, err := r.repo.GetScenarioVersion(ctx, scenarioVersionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load scenario version: %w", err)
	}

	var tr TenantValidationRule
	if err := json.Unmarshal(sv.RuleSnapshot, &tr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule snapshot: %w", err)
	}

	// 2. Create RuleTestRun record (status=RUNNING)
	runID := uuid.New().String()
	run := &RuleTestRun{
		ID:                runID,
		TenantID:          tenantID,
		ScenarioVersionID: &scenarioVersionID,
		Status:            "running",
		SampleSize:        sampleSize,
		StartedAt:         time.Now(),
	}
	if err := r.repo.CreateTestRun(ctx, run); err != nil {
		return nil, err
	}

	// 3. Fetch sample BO records
	records, err := r.sampleRepo.SampleEntities(ctx, tenantID, entity, sampleSize, filter)
	if err != nil {
		// Mark run as failed
		run.Status = "failed"
		_ = r.repo.UpdateTestRun(ctx, run)
		return nil, fmt.Errorf("failed to fetch sample entities: %w", err)
	}

	failures := 0
	for _, rec := range records {
		// Prepare Context
		// Assuming rec returns a flat map, we need to convert to BO context structure
		// using starlib.SplitDataIntoPageAndObjects or similar logic if needed.
		// For now, let's assume rec is already in a compatible format or purely flattened.
		page, objects := starlib.SplitDataIntoPageAndObjects(rec)
		boCtx := map[string]map[string]interface{}{
			"page": page,
		}
		for k, v := range objects {
			boCtx[k] = v
		}

		passed, err := r.engine.EvaluateTenantRule(ctx, &tr, boCtx)
		if err != nil {
			// Log error, treat as failure or skip?
			// failures++
			continue
		}
		if !passed {
			failures++
		}
	}

	// 4. Update test run
	now := time.Now()
	run.Status = "completed"
	run.CompletedAt = &now
	run.FailureCount = failures

	if err := r.repo.UpdateTestRun(ctx, run); err != nil {
		return nil, err
	}

	return run, nil
}
