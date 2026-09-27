package goldcopy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// AccountEngine materialises Account gold copies with approved overrides.
type AccountEngine struct {
	DB *sqlx.DB // data plane (crims)
}

func NewAccountEngine(db *sqlx.DB) *AccountEngine {
	return &AccountEngine{DB: db}
}

// BuildAndPersist loads the current account_master row, applies approved
// overrides, writes account_gold_copy + lineage, and returns the record.
func (e *AccountEngine) BuildAndPersist(ctx context.Context, tenantID uuid.UUID, accountCd string, publishedBy string) (*AccountMasterRecord, error) {
	tx, err := e.DB.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true), set_config('uisce.current_tenant', $1, true)`,
		tenantID.String()); err != nil {
		return nil, err
	}

	var row struct {
		ID               uuid.UUID       `db:"id"`
		AccountCd        string          `db:"account_cd"`
		AccountName      string          `db:"account_name"`
		AccountTypeCd    string          `db:"account_type_cd"`
		StatusCd         string          `db:"status_cd"`
		BaseCurrency     *string         `db:"base_currency"`
		Domicile         *string         `db:"domicile"`
		CustomAttributes json.RawMessage `db:"custom_attributes"`
		SourceSystems    json.RawMessage `db:"source_systems"`
		ConfidenceScore  int             `db:"confidence_score"`
		ValidFrom        time.Time       `db:"valid_from"`
	}
	if err := tx.GetContext(ctx, &row, `
		SELECT id, account_cd, account_name, account_type_cd, status_cd,
		       base_currency, domicile, COALESCE(custom_attributes, '{}'::jsonb) AS custom_attributes,
		       COALESCE(source_systems, '{}'::jsonb) AS source_systems,
		       confidence_score, valid_from
		FROM mdm.account_master
		WHERE tenant_id = $1 AND account_cd = $2 AND valid_to IS NULL
		LIMIT 1`, tenantID, accountCd); err != nil {
		return nil, fmt.Errorf("load account_master: %w", err)
	}

	custom := map[string]any{}
	_ = json.Unmarshal(row.CustomAttributes, &custom)
	sources := map[string]string{}
	_ = json.Unmarshal(row.SourceSystems, &sources)

	var overrides []AccountFieldOverride
	if err := tx.SelectContext(ctx, &overrides, `
		SELECT id, tenant_id, account_cd, semantic_term_id, field_cd, override_value,
		       reason, approval_status, expires_at
		FROM mdm.account_field_override
		WHERE tenant_id = $1 AND account_cd = $2
		  AND is_active AND approval_status = 'approved'
		  AND (expires_at IS NULL OR expires_at > now())`, tenantID, accountCd); err != nil {
		return nil, fmt.Errorf("load overrides: %w", err)
	}

	lineage := make([]AccountGoldLineage, 0, len(overrides))
	for _, o := range overrides {
		custom[o.FieldCd] = o.OverrideValue
		sources[o.FieldCd] = "STEWARD_OVERRIDE"
		oid := o.ID
		lineage = append(lineage, AccountGoldLineage{
			FieldCd:      o.FieldCd,
			ChosenValue:  o.OverrideValue,
			ChosenSource: "STEWARD_OVERRIDE",
			Strategy:     "manual_override",
			OverrideID:   &oid,
		})
	}

	var nextVer int
	if err := tx.GetContext(ctx, &nextVer, `
		SELECT COALESCE(MAX(gold_version), 0) + 1
		FROM mdm.account_gold_copy
		WHERE tenant_id = $1 AND account_cd = $2`, tenantID, accountCd); err != nil {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE mdm.account_gold_copy
		SET valid_to = now(), updated_at = now()
		WHERE tenant_id = $1 AND account_cd = $2 AND valid_to IS NULL`,
		tenantID, accountCd); err != nil {
		return nil, err
	}

	payload := map[string]any{
		"account_cd":        row.AccountCd,
		"account_name":      row.AccountName,
		"account_type_cd":   row.AccountTypeCd,
		"status_cd":         row.StatusCd,
		"base_currency":     derefStr(row.BaseCurrency),
		"domicile":          derefStr(row.Domicile),
		"custom_attributes": custom,
	}
	payloadJSON, _ := json.Marshal(payload)
	srcJSON, _ := json.Marshal(sources)
	dataHash := hashData(payload)

	goldID := uuid.New()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO mdm.account_gold_copy (
			id, tenant_id, account_master_id, account_cd, gold_version, payload,
			source_systems, confidence_score, published_at, change_type, change_reason,
			data_hash, valid_from
		) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,now(),'updated',$9,$10,now())`,
		goldID, tenantID, row.ID, accountCd, nextVer, string(payloadJSON),
		string(srcJSON), row.ConfidenceScore, "gold materialisation", dataHash,
	); err != nil {
		return nil, fmt.Errorf("insert gold copy: %w", err)
	}

	for _, l := range lineage {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO mdm.account_gold_lineage
				(tenant_id, account_cd, gold_version, field_cd, chosen_value, chosen_source, strategy, override_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			tenantID, accountCd, nextVer, l.FieldCd, l.ChosenValue, l.ChosenSource, l.Strategy, l.OverrideID,
		); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	rec := &AccountMasterRecord{
		ID:               goldID,
		TenantID:         tenantID,
		AccountCd:        row.AccountCd,
		AccountName:      row.AccountName,
		AccountTypeCd:    row.AccountTypeCd,
		StatusCd:         row.StatusCd,
		BaseCurrency:     derefStr(row.BaseCurrency),
		Domicile:         derefStr(row.Domicile),
		CustomAttributes: custom,
		SourceSystems:    sources,
		ConfidenceScore:  row.ConfidenceScore,
		GoldVersion:      nextVer,
		ValidFrom:        time.Now().UTC(),
	}
	_ = publishedBy
	return rec, nil
}

// AccountGoldLineage is one field decision written to mdm.account_gold_lineage.
type AccountGoldLineage struct {
	FieldCd      string
	ChosenValue  string
	ChosenSource string
	Strategy     string
	OverrideID   *uuid.UUID
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
