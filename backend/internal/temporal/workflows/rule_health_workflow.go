package workflows

import (
	"fmt"
	"time"

	"github.com/hondyman/uisce/backend/internal/temporal/activities"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	RuleHealthScheduleID  = "rule-health-sentinel"
	RuleHealthInterval    = 15 * time.Minute
	SignalAcknowledgeAlert = "AcknowledgeAlert"
)

type HealthAlertInput struct {
	TenantID string `json:"tenant_id"`
	RuleID   string `json:"rule_id"`
	RuleKey  string `json:"rule_key"`
	Status   string `json:"status"`
	Details  string `json:"details"`
}

// RuleHealthCheckWorkflow runs periodically on a Temporal Schedule to inspect
// trailing 24h violation and error rates.
func RuleHealthCheckWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting RuleHealthCheckWorkflow")

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    5 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var acts *activities.RuleHealthActivities
	var res activities.RuleHealthCheckResult

	err := workflow.ExecuteActivity(ctx, acts.RunRuleHealthCheckActivity, activities.RuleHealthCheckInput{}).Get(ctx, &res)
	if err != nil {
		logger.Error("RuleHealthCheckActivity failed", "error", err)
		return fmt.Errorf("run rule health check: %w", err)
	}

	logger.Info("RuleHealthCheck completed", "evaluated", res.EvaluatedRules, "flagged", len(res.FlaggedRules))
	return nil
}
