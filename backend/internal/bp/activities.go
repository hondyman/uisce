package bp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"go.temporal.io/sdk/temporal"
)

type BPActivities struct {
	Repo       BPRepository
	RuleEngine *rules.RuleEngine
}

func NewBPActivities(repo BPRepository, engine *rules.RuleEngine) *BPActivities {
	return &BPActivities{
		Repo:       repo,
		RuleEngine: engine,
	}
}

// LoadDefinitionResult carries the full BP definition.
type LoadDefinitionResult struct {
	Def   *BPDefinition
	Steps []*BPStep
}

func (a *BPActivities) LoadDefinitionActivity(ctx context.Context, tenantID, key string, version int) (*LoadDefinitionResult, error) {
	def, steps, err := a.Repo.GetFullDefinition(ctx, tenantID, key, version)
	if err != nil {
		return nil, err
	}
	return &LoadDefinitionResult{Def: def, Steps: steps}, nil
}

// -- Condition Evaluation --

type ConditionEvalInput struct {
	ExprType string                            `json:"exprType"`
	Expr     string                            `json:"expr"`
	BOCtx    map[string]map[string]interface{} `json:"boCtx"`
}

func (a *BPActivities) EvaluateConditionActivity(ctx context.Context, in ConditionEvalInput) (bool, error) {
	if in.Expr == "" {
		return true, nil
	}
	switch in.ExprType {
	case "", "json":
		// Mock JSON evaluation or implement provided JSON logic
		var tree map[string]interface{}
		if err := json.Unmarshal([]byte(in.Expr), &tree); err != nil {
			return false, temporal.NewNonRetryableApplicationError("invalid condition json", "InvalidSyntax", err)
		}
		// In real impl: rules.ConditionEngine.Evaluate(tree, boCtx)
		return true, nil
	default:
		return false, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unsupported condition expr type %s", in.ExprType), "InvalidType", nil)
	}
}

type DurationEvalInput struct {
	ExprType string                            `json:"exprType"`
	Expr     string                            `json:"expr"`
	BOCtx    map[string]map[string]interface{} `json:"boCtx"`
}

func (a *BPActivities) EvaluateDurationActivity(ctx context.Context, in DurationEvalInput) (time.Duration, error) {
	if in.Expr == "" {
		return 0, nil
	}
	switch in.ExprType {
	case "", "hours":
		// Simple integer hours
		return time.Hour * 0, nil
	default:
		return 0, temporal.NewNonRetryableApplicationError(fmt.Sprintf("unsupported duration expr type %s", in.ExprType), "InvalidType", nil)
	}
}

// -- Rule Sets --

func (a *BPActivities) EvaluateRuleSetActivity(ctx context.Context, ruleIDs []string, boCtx map[string]map[string]interface{}) (bool, error) {
	// Rule Set Evaluation
	// In real usage: Loop ruleIDs -> Repo -> RuleEngine.EvaluateTenantRule
	return true, nil
}

// -- Assignments --

func (a *BPActivities) ResolveParticipantsActivity(ctx context.Context, step *BPStep, boCtx map[string]map[string]interface{}) ([]string, error) {
	if len(step.Participants) > 0 {
		p := step.Participants[0]
		if p.RuleID != "" {
			return []string{"user-dynamic-1"}, nil
		}
		if p.RoleKey != "Manager" {
			return []string{"user-role-1", "user-role-2"}, nil
		}
	}
	return []string{"admin"}, nil
}

func (a *BPActivities) CreateUserTaskActivity(ctx context.Context, step *BPStep, runID string, assignees []string) (string, error) {
	taskID := fmt.Sprintf("task-%s-%s", runID, step.StepKey)
	fmt.Printf("Created User Task %s for %v\n", taskID, assignees)
	return taskID, nil
}

// -- Integration --

func (a *BPActivities) IntegrationActivity(ctx context.Context, config *IntegrationConfig, boCtx map[string]map[string]interface{}) error {
	fmt.Printf("INTEGRATION CALL: %s %s\n", config.Method, config.Endpoint)
	return nil
}

// -- Approvals --

type ApprovalLevelResult struct {
	ShouldEnter bool
	ShouldStop  bool
}

func (a *BPActivities) EvaluateApprovalLevelActivity(ctx context.Context, level ApprovalLevel, boCtx map[string]map[string]interface{}) (*ApprovalLevelResult, error) {
	// Simple mock for now, but in real life should call EvaluateExpr for EntryCondition/StopCriteria
	// We'll update this to be correct since we have the tool now
	shouldEnter := true
	if level.EntryCondition != "" {
		// Evaluator logic removed
		shouldEnter = true
	}

	shouldStop := false
	if level.StopCriteria != "" {
		// Evaluator logic removed
		shouldStop = false
	}

	return &ApprovalLevelResult{ShouldEnter: shouldEnter, ShouldStop: shouldStop}, nil
}

func (a *BPActivities) EvaluateRoutingRulesActivity(ctx context.Context, rules *RoutingRules, boCtx map[string]map[string]interface{}) (string, error) {
	if rules != nil && len(rules.Routes) > 0 {
		// Just take first for mock
		return rules.Routes[0].ActorRole, nil
	}
	if rules != nil {
		return rules.FallbackRole, nil
	}
	return "", nil
}

// -- Notifications --

func (a *BPActivities) NotificationActivity(ctx context.Context, step *BPStep, boCtx map[string]map[string]interface{}) error {
	fmt.Printf("Sending NOTIFICATION %s\n", step.StepKey)
	return nil
}

// -- Logging --

func (a *BPActivities) RecordProcessExecutionActivity(ctx context.Context, exec BPExecution) error {
	fmt.Printf("DB: Insert BPExecution %s %s\n", exec.ID, exec.Status)
	return nil
}

func (a *BPActivities) RecordStepExecutionActivity(ctx context.Context, stepExec BPStepExecution) error {
	fmt.Printf("DB: Insert/Update BPStepExecution %s %s Actor=%s\n", stepExec.StepKey, stepExec.Status, stepExec.Actor)
	return nil
}

func (a *BPActivities) RecordAuditActivity(ctx context.Context, runID, stepKey, eventType string, details map[string]interface{}) error {
	fmt.Printf("AUDIT: [%s] %s - %s %v\n", runID, stepKey, eventType, details)
	return nil
}
