package bp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type DesignerService struct {
	db *sql.DB
}

func NewDesignerService(db *sql.DB) *DesignerService {
	return &DesignerService{db: db}
}

type SaveDesignerInput struct {
	TenantID  string
	BpDefID   *string
	BpKey     string
	BpVersion int
	Status    string
	Steps     []BPStepPayload
}

type BPStepPayload struct {
	Seq                   int                    `json:"seq"`
	StepKey               string                 `json:"stepKey"`
	Type                  string                 `json:"type"`
	ActivityName          string                 `json:"activityName"`
	SignalName            string                 `json:"signalName"`
	ConditionExprType     string                 `json:"conditionExprType"`
	ConditionExpr         string                 `json:"conditionExpr"`
	PreValidationRuleIDs  []string               `json:"preValidationRuleIds"`
	PostValidationRuleIDs []string               `json:"postValidationRuleIds"`
	ApprovalChain         map[string]interface{} `json:"approvalChain"`
	RoutingRules          map[string]interface{} `json:"routingRules"`
	Escalations           []EscalationStep       `json:"escalations"`
	DelayExprType         string                 `json:"delayExprType"`
	DelayExpr             string                 `json:"delayExpr"`
	SLAExprType           string                 `json:"slaExprType"`
	SLAExpr               string                 `json:"slaExpr"`
}

func (s *DesignerService) SaveDesigner(ctx context.Context, in SaveDesignerInput) (string, error) {
	var bpDefID string
	var err error

	// 1. Insert or update business_process_definition
	if in.BpDefID != nil && *in.BpDefID != "" {
		bpDefID = *in.BpDefID
		// Update existing
		_, err = s.db.ExecContext(ctx, `
            update business_process_definition
            set status = $1, created_at = now() 
            where id = $2 and tenant_id = $3
        `, in.Status, bpDefID, in.TenantID)
	} else {
		// Insert new
		row := s.db.QueryRowContext(ctx, `
            insert into business_process_definition
            (tenant_id, key, version, name, entity, status, created_by)
            values ($1, $2, $3, $4, $5, $6, $7)
            RETURNING id
        `, in.TenantID, in.BpKey, in.BpVersion, in.BpKey, "Manual", in.Status, "system")
		err = row.Scan(&bpDefID)
	}

	if err != nil {
		return "", fmt.Errorf("failed to save bp definition: %w", err)
	}

	// 2. Delete old steps (full replace for designer state)
	if _, err := s.db.ExecContext(ctx, `delete from business_process_step where bp_def_id = $1`, bpDefID); err != nil {
		return "", fmt.Errorf("failed to cleanup steps: %w", err)
	}

	// 3. Insert new steps
	for _, step := range in.Steps {
		approvalChainJSON, _ := json.Marshal(step.ApprovalChain)
		routingRulesJSON, _ := json.Marshal(step.RoutingRules)
		escalationsJSON, _ := json.Marshal(step.Escalations)

		// Handle string array for PG
		// Simplified: passing specific args
		_, err = s.db.ExecContext(ctx, `
            insert into business_process_step
            (bp_def_id, seq, step_key, type, activity_name, signal_name,
             condition_expr_type, condition_expr,
             approval_chain, routing_rules, escalations,
             delay_expr_type, delay_expr, sla_expr_type, sla_expr)
            values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
        `,
			bpDefID, step.Seq, step.StepKey, step.Type, step.ActivityName, step.SignalName,
			step.ConditionExprType, step.ConditionExpr,
			approvalChainJSON, routingRulesJSON, escalationsJSON,
			step.DelayExprType, step.DelayExpr, step.SLAExprType, step.SLAExpr,
		)
		if err != nil {
			return "", fmt.Errorf("failed to save step %s: %w", step.StepKey, err)
		}
	}

	return bpDefID, nil
}

func (s *DesignerService) GetDesigner(ctx context.Context, bpDefID string) (map[string]interface{}, error) {
	var def struct {
		ID        string
		Key       string
		Version   int
		Status    string
		CreatedAt time.Time
	}
	err := s.db.QueryRowContext(ctx, `
        select id, key, version, status, created_at
        from business_process_definition
        where id = $1
    `, bpDefID).Scan(&def.ID, &def.Key, &def.Version, &def.Status, &def.CreatedAt)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
        select id, seq, step_key, type, activity_name, signal_name,
               condition_expr_type, condition_expr,
               approval_chain, routing_rules, escalations,
               delay_expr_type, delay_expr, sla_expr_type, sla_expr
        from business_process_step
        where bp_def_id = $1
        order by seq
    `, bpDefID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	steps := []map[string]interface{}{}
	for rows.Next() {
		var step struct {
			ID                string
			Seq               int
			StepKey           string
			Type              string
			ActivityName      sql.NullString
			SignalName        sql.NullString
			ConditionExprType sql.NullString
			ConditionExpr     sql.NullString
			ApprovalChain     []byte
			RoutingRules      []byte
			Escalations       []byte
			DelayExprType     sql.NullString
			DelayExpr         sql.NullString
			SLAExprType       sql.NullString
			SLAExpr           sql.NullString
		}
		if err := rows.Scan(
			&step.ID, &step.Seq, &step.StepKey, &step.Type, &step.ActivityName, &step.SignalName,
			&step.ConditionExprType, &step.ConditionExpr,
			&step.ApprovalChain, &step.RoutingRules, &step.Escalations,
			&step.DelayExprType, &step.DelayExpr, &step.SLAExprType, &step.SLAExpr,
		); err != nil {
			return nil, err
		}

		// Unmarshal JSONBs
		var ac map[string]interface{}
		_ = json.Unmarshal(step.ApprovalChain, &ac)
		var rr map[string]interface{}
		_ = json.Unmarshal(step.RoutingRules, &rr)
		var esc []EscalationStep
		_ = json.Unmarshal(step.Escalations, &esc)

		steps = append(steps, map[string]interface{}{
			"id":                step.ID,
			"seq":               step.Seq,
			"stepKey":           step.StepKey,
			"type":              step.Type,
			"activityName":      step.ActivityName.String,
			"signalName":        step.SignalName.String,
			"conditionExprType": step.ConditionExprType.String,
			"conditionExpr":     step.ConditionExpr.String,
			"approvalChain":     ac,
			"routingRules":      rr,
			"escalations":       esc,
			"delayExprType":     step.DelayExprType.String,
			"delayExpr":         step.DelayExpr.String,
			"slaExprType":       step.SLAExprType.String,
			"slaExpr":           step.SLAExpr.String,
		})
	}

	return map[string]interface{}{
		"bpDefId":     def.ID,
		"bpKey":       def.Key,
		"bpVersion":   def.Version,
		"status":      def.Status,
		"steps":       steps,
		"lastSavedAt": def.CreatedAt,
	}, nil
}

// GetSteps retrieves strongly-typed steps for workflow compilation
func (s *DesignerService) GetSteps(ctx context.Context, bpDefID string) ([]BPStep, error) {
	rows, err := s.db.QueryContext(ctx, `
        select id, seq, step_key, type, activity_name, signal_name,
               condition_expr_type, condition_expr,
               approval_chain, routing_rules, escalations,
               delay_expr_type, delay_expr, sla_expr_type, sla_expr
        from business_process_step
        where bp_def_id = $1
        order by seq
    `, bpDefID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []BPStep
	for rows.Next() {
		var step BPStep
		var activityName, signalName, condExprType, condExpr sql.NullString
		var delayExprType, delayExpr, slaExprType, slaExpr sql.NullString
		var approvalChain, routingRules, escalations []byte

		if err := rows.Scan(
			&step.ID, &step.Seq, &step.StepKey, &step.Type, &activityName, &signalName,
			&condExprType, &condExpr,
			&approvalChain, &routingRules, &escalations,
			&delayExprType, &delayExpr, &slaExprType, &slaExpr,
		); err != nil {
			return nil, err
		}

		step.ActivityName = activityName.String
		step.SignalName = signalName.String
		step.ConditionExprType = condExprType.String
		step.ConditionExpr = condExpr.String
		step.DelayExprType = delayExprType.String
		step.DelayExpr = delayExpr.String
		step.SLAExprType = slaExprType.String
		step.SLAExpr = slaExpr.String

		if len(approvalChain) > 0 {
			_ = json.Unmarshal(approvalChain, &step.ApprovalChain)
		}
		if len(routingRules) > 0 {
			_ = json.Unmarshal(routingRules, &step.RoutingRules)
		}
		if len(escalations) > 0 {
			_ = json.Unmarshal(escalations, &step.Escalations)
		}

		steps = append(steps, step)
	}
	return steps, nil
}
