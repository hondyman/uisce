package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"
	"go.temporal.io/sdk/workflow"
)

// AuditEvent represents a Q&A interaction to be audited
type AuditEvent struct {
	ID         string
	TenantID   string
	UserID     string
	Question   string
	Provider   string
	Version    string
	Confidence string
	Answer     string
	Sources    []string
	Caveats    []string
	Timestamp  time.Time
}

// AuditWorkflowWithQuality processes an audit event with automatic data quality overlay
func AuditWorkflowWithQuality(ctx workflow.Context, event AuditEvent) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	
	// Initialize activities struct (will be registered separately)
	var a *AuditActivities
	
	// Step 1: Compute data quality metrics for all sources
	var dataQuality *services.DataQuality
	err := workflow.ExecuteActivity(ctx, a.ComputeDataQualityActivity, event.Sources, event.TenantID).Get(ctx, &dataQuality)
	if err != nil {
		return err
	}
	
	// Step 2: Check if SLA is violated - optionally block audit if critical
	if dataQuality.FreshnessStatus == "RED" {
		// Log warning but continue
		workflow.GetLogger(ctx).Warn("Data freshness SLA violated",
			"tenant", event.TenantID,
			"freshness", dataQuality.Freshness,
		)
	}
	
	// Step 3: Fetch last hash for tenant
	var lastHash string
	err = workflow.ExecuteActivity(ctx, a.FetchLastHashActivity, event.TenantID).Get(ctx, &lastHash)
	if err != nil {
		return err
	}
	
	// Step 4: Compute new hash with data quality included
	var newHash string
	err = workflow.ExecuteActivity(ctx, a.ComputeHashActivity, event, lastHash, dataQuality).Get(ctx, &newHash)
	if err != nil {
		return err
	}
	
	// Step 5: Persist audit log with data quality overlay
	err = workflow.ExecuteActivity(ctx, a.PersistAuditLogActivity, event, newHash, lastHash, dataQuality).Get(ctx, nil)
	if err != nil {
		return err
	}
	
	// Step 6: Update last hash pointer
	err = workflow.ExecuteActivity(ctx, a.UpdateLastHashActivity, event.TenantID, newHash).Get(ctx, nil)
	if err != nil {
		return err
	}
	
	return nil
}

// ChainValidationWorkflow validates audit chain integrity with data quality checks
func ChainValidationWorkflow(ctx workflow.Context, tenantID string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	
	var a *AuditActivities
	
	// Step 1: Fetch all audit logs for tenant
	var logs []AuditLogEntry
	err := workflow.ExecuteActivity(ctx, a.FetchAuditLogsActivity, tenantID).Get(ctx, &logs)
	if err != nil {
		return err
	}
	
	// Step 2: Validate hash chain
	var chainBroken bool
	err = workflow.ExecuteActivity(ctx, a.ValidateChainActivity, logs).Get(ctx, &chainBroken)
	if err != nil {
		return err
	}
	
	// Step 3: Validate data quality SLAs across all entries
	var slaViolations int
	err = workflow.ExecuteActivity(ctx, a.ValidateSLAComplianceActivity, logs).Get(ctx, &slaViolations)
	if err != nil {
		return err
	}
	
	// Step 4: Alert if issues detected
	if chainBroken || slaViolations > 0 {
		err = workflow.ExecuteActivity(ctx, a.EmitAlertActivity, tenantID, chainBroken, slaViolations).Get(ctx, nil)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// AuditLogEntry represents a complete audit log with data quality
type AuditLogEntry struct {
	ID          string
	TenantID    string
	Question    string
	Answer      string
	Sources     []string
	Caveats     []string
	Hash        string
	PrevHash    string
	DataQuality *services.DataQuality
	Timestamp   time.Time
}
