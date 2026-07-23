package rules

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ScenarioRepository defines database operations for scenarios
type ScenarioRepository interface {
	CreateScenario(ctx context.Context, scenario *RuleScenario) error
	CreateScenarioVersion(ctx context.Context, version *RuleScenarioVersion) error
	GetScenario(ctx context.Context, id string) (*RuleScenario, error)
	GetScenarioVersion(ctx context.Context, id string) (*RuleScenarioVersion, error)
	GetLatestScenarioVersion(ctx context.Context, scenarioID string) (*RuleScenarioVersion, error)

	// Test Run operations
	CreateTestRun(ctx context.Context, run *RuleTestRun) error
	UpdateTestRun(ctx context.Context, run *RuleTestRun) error
	GetTestRun(ctx context.Context, id string) (*RuleTestRun, error)
}

type ScenarioService struct {
	repo ScenarioRepository
}

func NewScenarioService(repo ScenarioRepository) *ScenarioService {
	return &ScenarioService{repo: repo}
}

// CreateRuleScenario initializes a new scenario, optionally from an existing rule
func (s *ScenarioService) CreateRuleScenario(
	ctx context.Context,
	tenantID string,
	baseRuleID *string,
	name string,
	description string,
	createdBy string,
) (*RuleScenario, error) {
	scenarioID := uuid.New().String()
	scenario := &RuleScenario{
		ID:          scenarioID,
		TenantID:    tenantID,
		BaseRuleID:  baseRuleID,
		Name:        name,
		Description: description,
		Status:      "draft",
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.repo.CreateScenario(ctx, scenario); err != nil {
		return nil, err
	}

	return scenario, nil
}

// SaveScenarioVersion saves a new version of the rule snapshot
func (s *ScenarioService) SaveScenarioVersion(
	ctx context.Context,
	scenarioID string,
	ruleDraft json.RawMessage,
	createdBy string,
) (*RuleScenarioVersion, error) {
	// Get latest version to increment
	latest, err := s.repo.GetLatestScenarioVersion(ctx, scenarioID)
	nextVer := 1
	if err == nil && latest != nil {
		nextVer = latest.Version + 1
	}

	version := &RuleScenarioVersion{
		ID:           uuid.New().String(),
		ScenarioID:   scenarioID,
		Version:      nextVer,
		RuleSnapshot: ruleDraft,
		CreatedBy:    createdBy,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateScenarioVersion(ctx, version); err != nil {
		return nil, err
	}

	return version, nil
}
