package tiering

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// TieringRepository handles persistence for storage tiering plans
type TieringRepository struct {
	db *sqlx.DB
}

// NewTieringRepository creates a new tiering repository
func NewTieringRepository(db *sqlx.DB) *TieringRepository {
	return &TieringRepository{db: db}
}

// SavePlan persists a tiering plan
func (r *TieringRepository) SavePlan(ctx context.Context, plan *TieringPlan) error {
	rulesJSON, err := json.Marshal(plan.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	query := `
		INSERT INTO storage_tiering_plans (id, tenant_id, rules, summary, status, created_at, updated_at)
		VALUES (:id, :tenant_id, :rules, :summary, :status, :created_at, :updated_at)
		ON CONFLICT (id) DO UPDATE SET
			rules = EXCLUDED.rules,
			summary = EXCLUDED.summary,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`

	now := time.Now()
	if plan.ID == uuid.Nil {
		plan.ID = uuid.New()
	}
	if plan.CreatedAt.IsZero() {
		plan.CreatedAt = now
	}
	plan.UpdatedAt = now

	_, err = r.db.NamedExecContext(ctx, query, map[string]interface{}{
		"id":         plan.ID,
		"tenant_id":  plan.TenantID,
		"rules":      rulesJSON,
		"summary":    plan.Summary,
		"status":     plan.Status,
		"created_at": plan.CreatedAt,
		"updated_at": plan.UpdatedAt,
	})

	return err
}

// GetPlan retrieves a plan by ID
func (r *TieringRepository) GetPlan(ctx context.Context, id uuid.UUID) (*TieringPlan, error) {
	var dest struct {
		ID        uuid.UUID       `db:"id"`
		TenantID  string          `db:"tenant_id"`
		Rules     json.RawMessage `db:"rules"`
		Summary   string          `db:"summary"`
		Status    string          `db:"status"`
		CreatedAt time.Time       `db:"created_at"`
		UpdatedAt time.Time       `db:"updated_at"`
	}

	query := `SELECT id, tenant_id, rules, summary, status, created_at, updated_at FROM storage_tiering_plans WHERE id = $1`
	err := r.db.GetContext(ctx, &dest, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("plan not found: %s", id)
		}
		return nil, err
	}

	var rules []TieringRule
	if err := json.Unmarshal(dest.Rules, &rules); err != nil {
		return nil, err
	}

	return &TieringPlan{
		ID:        dest.ID,
		TenantID:  dest.TenantID,
		Rules:     rules,
		Summary:   dest.Summary,
		Status:    dest.Status,
		CreatedAt: dest.CreatedAt,
		UpdatedAt: dest.UpdatedAt,
	}, nil
}

// ListPlans retrieves plans for a tenant
func (r *TieringRepository) ListPlans(ctx context.Context, tenantID string) ([]TieringPlan, error) {
	var rows []struct {
		ID        uuid.UUID       `db:"id"`
		TenantID  string          `db:"tenant_id"`
		Rules     json.RawMessage `db:"rules"`
		Summary   string          `db:"summary"`
		Status    string          `db:"status"`
		CreatedAt time.Time       `db:"created_at"`
		UpdatedAt time.Time       `db:"updated_at"`
	}

	query := `SELECT id, tenant_id, rules, summary, status, created_at, updated_at FROM storage_tiering_plans WHERE tenant_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &rows, query, tenantID)
	if err != nil {
		return nil, err
	}

	plans := make([]TieringPlan, len(rows))
	for i, row := range rows {
		var rules []TieringRule
		json.Unmarshal(row.Rules, &rules)
		plans[i] = TieringPlan{
			ID:        row.ID,
			TenantID:  row.TenantID,
			Rules:     rules,
			Summary:   row.Summary,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		}
	}

	return plans, nil
}
