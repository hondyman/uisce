package orm

import (
	"context"
	"database/sql"
)

// GetModelTargets returns the current target allocation for a model.
func GetModelTargets(ctx context.Context, tx *sql.Tx, tenantID, modelID string) ([]ModelTarget, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT target_type, target_reference_cd, security_id,
		       target_weight_pct, min_weight_pct, max_weight_pct
		FROM orm.model_portfolio_target
		WHERE tenant_id = $1 AND model_portfolio_id = $2 AND is_current = true
	`, tenantID, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ModelTarget
	for rows.Next() {
		var t ModelTarget
		if err := rows.Scan(&t.TargetType, &t.ReferenceCd, &t.SecurityID,
			&t.WeightPct, &t.MinPct, &t.MaxPct); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// AssignAccountToModel makes an SMA/UMA account follow a model.
func AssignAccountToModel(ctx context.Context, tx *sql.Tx, tenantID,
	accountID, modelID string, allocPct float64, driftTol sql.NullFloat64) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE orm.account_model_assignment
		SET is_current = false, effective_to = CURRENT_DATE
		WHERE tenant_id = $1 AND account_id = $2 AND is_current = true
	`, tenantID, accountID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.account_model_assignment (
			account_id, model_portfolio_id, allocation_pct,
			drift_tolerance_pct, effective_from, is_current, tenant_id
		) VALUES ($1,$2,$3,$4,CURRENT_DATE,true,$5)
	`, accountID, modelID, allocPct, driftTol, tenantID)
	return err
}

type ModelTarget struct {
	TargetType  string
	ReferenceCd sql.NullString
	SecurityID  sql.NullString
	WeightPct   float64
	MinPct      sql.NullFloat64
	MaxPct      sql.NullFloat64
}
