package bp

import (
	"context"
	"database/sql"
)

type BPRepository interface {
	GetDefinition(ctx context.Context, tenantID, key string, version int) (*BPDefinition, error)
	GetSteps(ctx context.Context, bpDefID string) ([]*BPStep, error)
	GetStepParticipants(ctx context.Context, stepID string) ([]BPStepParticipant, error)
	GetFullDefinition(ctx context.Context, tenantID, key string, version int) (*BPDefinition, []*BPStep, error)
}

type SQLBPRepository struct {
	db *sql.DB
}

func NewSQLBPRepository(db *sql.DB) *SQLBPRepository {
	return &SQLBPRepository{db: db}
}

func (r *SQLBPRepository) GetDefinition(ctx context.Context, tenantID, key string, version int) (*BPDefinition, error) {
	query := `
		SELECT id, tenant_id, key, version, name, entity, status, description, created_at, created_by
		FROM business_process_definition
		WHERE tenant_id = $1 AND key = $2 AND version = $3
	`
	var def BPDefinition
	err := r.db.QueryRowContext(ctx, query, tenantID, key, version).Scan(
		&def.ID, &def.TenantID, &def.Key, &def.Version, &def.Name, &def.Entity,
		&def.Status, &def.Description, &def.CreatedAt, &def.CreatedBy,
	)
	if err != nil {
		return nil, err
	}
	return &def, nil
}

func (r *SQLBPRepository) GetSteps(ctx context.Context, bpDefID string) ([]*BPStep, error) {
	query := `
		SELECT id, bp_def_id, seq, step_key, type, activity_name, signal_name, description,
		       pre_validation_rule_ids, post_validation_rule_ids, condition_expr, created_at
		FROM business_process_step
		WHERE bp_def_id = $1
		ORDER BY seq ASC
	`
	rows, err := r.db.QueryContext(ctx, query, bpDefID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*BPStep
	for rows.Next() {
		var s BPStep
		if err := rows.Scan(
			&s.ID, &s.BPDefID, &s.Seq, &s.StepKey, &s.Type, &s.ActivityName, &s.SignalName,
			&s.Description, &s.PreValidationRuleIDs, &s.PostValidationRuleIDs, &s.ConditionExpr, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		steps = append(steps, &s)
	}
	return steps, nil
}

func (r *SQLBPRepository) GetStepParticipants(ctx context.Context, stepID string) ([]BPStepParticipant, error) {
	query := `SELECT id, step_id, role_key, created_at FROM business_process_step_participant WHERE step_id = $1`
	rows, err := r.db.QueryContext(ctx, query, stepID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []BPStepParticipant
	for rows.Next() {
		var p BPStepParticipant
		if err := rows.Scan(&p.ID, &p.StepID, &p.RoleKey, &p.CreatedAt); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}

func (r *SQLBPRepository) GetFullDefinition(ctx context.Context, tenantID, key string, version int) (*BPDefinition, []*BPStep, error) {
	def, err := r.GetDefinition(ctx, tenantID, key, version)
	if err != nil {
		return nil, nil, err
	}
	steps, err := r.GetSteps(ctx, def.ID)
	if err != nil {
		return nil, nil, err
	}

	for _, s := range steps {
		parts, err := r.GetStepParticipants(ctx, s.ID)
		if err != nil {
			return nil, nil, err
		}
		s.Participants = parts
	}

	return def, steps, nil
}
