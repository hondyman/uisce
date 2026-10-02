package workflows

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// Activities holds the legacy workflow activities
type Activities struct {
	db *sql.DB
}

// NewActivities creates a new Activities instance
func NewActivities(db *sql.DB) *Activities {
	return &Activities{db: db}
}

func (a *Activities) LoadBPStepsActivity(ctx context.Context, processID, tenantID string) ([]BPStep, error) {
	if a.db == nil {
		return nil, nil
	}
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, fmt.Errorf("begin load steps tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if tenantID != "" {
		_, err = tx.ExecContext(ctx, fmt.Sprintf("SET LOCAL app.current_tenant = '%s'", tenantID))
		if err != nil {
			return nil, fmt.Errorf("set rls context in load steps: %w", err)
		}
	}

	var stepsJSON []byte
	err = tx.QueryRowContext(ctx, `
		SELECT steps_json FROM public.bp_process_definition
		WHERE process_id = $1
		ORDER BY version DESC LIMIT 1
	`, processID).Scan(&stepsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query bp steps: %w", err)
	}

	var steps []BPStep
	if len(stepsJSON) > 0 {
		if err := json.Unmarshal(stepsJSON, &steps); err != nil {
			return nil, fmt.Errorf("unmarshal bp steps: %w", err)
		}
	}
	return steps, nil
}

func (a *Activities) DataEntryActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) ValidationActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) ApprovalActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) EmailNotificationActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) SlackNotificationActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) GenericStepActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) EscalateStepActivity(ctx context.Context, step BPStep, signal map[string]interface{}) error {
	return nil
}

func (a *Activities) AutoEscalateActivity(ctx context.Context, step BPStep, eventData map[string]interface{}) error {
	return nil
}

// Portfolio Rebalancing Activities
func (a *Activities) GetPortfolioData(ctx context.Context, portfolioID string) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) CalculateDrift(ctx context.Context, portfolioData map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func (a *Activities) SendAlert(ctx context.Context, alertData map[string]interface{}) error {
	return nil
}

func (a *Activities) ExecuteTrade(ctx context.Context, tradeData map[string]interface{}) error {
	return nil
}

// RecordWorkflowRunStartActivity records the start of a workflow execution in bp_workflow_run
func (a *Activities) RecordWorkflowRunStartActivity(ctx context.Context, runID, workflowID, tenantID, processID, processName, triggerType, triggerName, entity, entityID string) error {
	if a.db == nil || runID == "" {
		return nil
	}
	query := `
		INSERT INTO public.bp_workflow_run (
			workflow_id, run_id, tenant_id, process_id, process_name,
			trigger_type, trigger_name, entity, entity_id, status,
			started_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9, 'RUNNING',
			NOW(), NOW(), NOW()
		)
		ON CONFLICT (run_id) DO NOTHING
	`
	_, err := a.db.ExecContext(ctx, query, workflowID, runID, tenantID, processID, processName, triggerType, triggerName, entity, entityID)
	return err
}

// RecordWorkflowRunTerminalActivity records the terminal status of a workflow execution
func (a *Activities) RecordWorkflowRunTerminalActivity(ctx context.Context, runID string, status string, errMsg string) error {
	if a.db == nil || runID == "" {
		return nil
	}
	query := `
		UPDATE public.bp_workflow_run
		SET 
			status = $1,
			error_message = $2,
			completed_at = NOW(),
			duration_ms = GREATEST(0, EXTRACT(EPOCH FROM (NOW() - started_at)) * 1000)::bigint,
			updated_at = NOW()
		WHERE run_id = $3
	`
	var errVal *string
	if errMsg != "" {
		errVal = &errMsg
	}
	_, err := a.db.ExecContext(ctx, query, status, errVal, runID)
	return err
}

