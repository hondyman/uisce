package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/jmoiron/sqlx"
)

// PartyStagingRow is one pending staging.party_data row.
type PartyStagingRow struct {
	ID               uuid.UUID       `db:"id"`
	TenantID         uuid.UUID       `db:"tenant_id"`
	SourceSystem     string          `db:"source_system"`
	SourceRowID      *string         `db:"source_row_id"`
	PartyCd          *string         `db:"party_cd"`
	LegalName        *string         `db:"legal_name"`
	PartyType        *string         `db:"party_type"`
	Segment          *string         `db:"segment"`
	TaxID            *string         `db:"tax_id"`
	Domicile         *string         `db:"domicile"`
	Status           *string         `db:"status"`
	LEI              *string         `db:"lei"`
	CustomAttributes json.RawMessage `db:"custom_attributes"`
}

// LoadPartyBatch loads staging.party_data into mdm.party (upsert by party_cd).
func (l *TwoPoolLoader) LoadPartyBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) (loaded int, warnings int, err error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	if l.AlphaPool == nil || l.DataPool == nil {
		return 0, 0, fmt.Errorf("two-pool loader requires AlphaPool and DataPool")
	}
	defs, err := l.loadEntityDefs(ctx, "PARTY")
	if err != nil {
		return 0, 0, err
	}
	defByField := map[string]AttributeDef{}
	for _, d := range defs {
		defByField[d.FieldCd] = d
	}

	tx, err := l.DataPool.BeginTxx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true), set_config('uisce.current_tenant', $1, true)`,
		tenantID.String()); err != nil {
		return 0, 0, err
	}

	var rows []PartyStagingRow
	if err := tx.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, source_system, source_row_id,
		       party_cd, legal_name, party_type, segment, tax_id, domicile, status, lei,
		       COALESCE(custom_attributes, '{}'::jsonb) AS custom_attributes
		FROM staging.party_data
		WHERE tenant_id = $1 AND loaded_at_master IS NULL
		ORDER BY loaded_at LIMIT $2`, tenantID, batchSize); err != nil {
		return 0, 0, err
	}
	if len(rows) == 0 {
		return 0, 0, tx.Commit()
	}

	runID := uuid.New()
	var allWarnings []Warning
	fieldRules := map[string]mdm.FieldRule{}
	if l.Surv != nil {
		fr, _, err := l.Surv.ResolveFieldRules(ctx, tenantID, "PARTY")
		if err != nil {
			return 0, 0, err
		}
		fieldRules = fr
	}

	groups := map[string][]PartyStagingRow{}
	var order []string
	for _, row := range rows {
		key := deref(row.PartyCd)
		if key == "" {
			allWarnings = append(allWarnings, Warning{
				StagingRowID: row.ID, FieldCd: "party_cd", Reason: "PARTY_CD_REQUIRED", Severity: "ERROR",
			})
			continue
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}

	engine := mdm.NewSurvivorshipEngine()
	now := time.Now().UTC()
	for _, key := range order {
		group := groups[key]
		merged, sourcesUsed, mergeWarnings, err := mergePartyGroup(ctx, engine, fieldRules, group, defByField, l.RequireSemanticTerms, now)
		if err != nil {
			return loaded, warnings, err
		}
		allWarnings = append(allWarnings, mergeWarnings...)
		custom := map[string]any{}
		_ = json.Unmarshal(merged.CustomAttributes, &custom)
		if err := upsertParty(ctx, tx, merged, custom, sourcesUsed); err != nil {
			return loaded, warnings, fmt.Errorf("upsert party %s: %w", key, err)
		}
		for _, row := range group {
			if _, err := tx.ExecContext(ctx, `
				UPDATE staging.party_data SET loaded_at_master=now(), load_run_id=$2 WHERE id=$1`,
				row.ID, runID); err != nil {
				return loaded, warnings, err
			}
		}
		loaded++
	}

	for _, w := range allWarnings {
		if w.StagingRowID == uuid.Nil {
			warnings++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO staging.party_warnings
				(tenant_id, staging_row_id, load_run_id, field_cd, value, referenced_bo, reason, severity)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			tenantID, w.StagingRowID, runID, w.FieldCd, w.Value, nullIfEmpty(w.ReferencedBO), w.Reason, defaultSeverity(w.Severity),
		); err != nil {
			return loaded, warnings, err
		}
		warnings++
	}
	if err := tx.Commit(); err != nil {
		return loaded, warnings, err
	}
	return loaded, warnings, nil
}

func mergePartyGroup(
	ctx context.Context,
	engine *mdm.SurvivorshipEngine,
	fieldRules map[string]mdm.FieldRule,
	group []PartyStagingRow,
	defs map[string]AttributeDef,
	requireTerms bool,
	now time.Time,
) (PartyStagingRow, map[string]string, []Warning, error) {
	if len(group) == 1 && len(fieldRules) == 0 {
		custom := map[string]any{}
		_ = json.Unmarshal(group[0].CustomAttributes, &custom)
		var warns []Warning
		if requireTerms {
			for field, def := range defs {
				if def.SemanticTermID == nil {
					if _, ok := custom[field]; ok {
						warns = append(warns, Warning{StagingRowID: group[0].ID, FieldCd: field, Reason: "SEMANTIC_TERM_UNBOUND", Severity: "WARNING"})
					}
				}
			}
		}
		return group[0], map[string]string{"*": group[0].SourceSystem}, warns, nil
	}
	sources := make([]mdm.SourcePayload, 0, len(group))
	for _, row := range group {
		m := map[string]any{}
		put := func(k string, v *string) {
			if v != nil && *v != "" {
				m[k] = *v
			}
		}
		put("party_cd", row.PartyCd)
		put("legal_name", row.LegalName)
		put("party_type", row.PartyType)
		put("segment", row.Segment)
		put("tax_id", row.TaxID)
		put("domicile", row.Domicile)
		put("status", row.Status)
		put("lei", row.LEI)
		custom := map[string]any{}
		_ = json.Unmarshal(row.CustomAttributes, &custom)
		for k, v := range custom {
			m[k] = v
		}
		sources = append(sources, mdm.SourcePayload{SourceID: row.SourceSystem, Data: m})
	}
	golden, err := engine.MergeToGoldenRecord(ctx, sources, fieldRules, now)
	if err != nil {
		return PartyStagingRow{}, nil, nil, err
	}
	out := group[0]
	strPtr := func(k string) *string {
		v, ok := golden[k]
		if !ok || v == nil {
			return nil
		}
		s := fmt.Sprint(v)
		return &s
	}
	if p := strPtr("party_cd"); p != nil {
		out.PartyCd = p
	}
	if p := strPtr("legal_name"); p != nil {
		out.LegalName = p
	}
	if p := strPtr("party_type"); p != nil {
		out.PartyType = p
	}
	if p := strPtr("segment"); p != nil {
		out.Segment = p
	}
	if p := strPtr("tax_id"); p != nil {
		out.TaxID = p
	}
	if p := strPtr("domicile"); p != nil {
		out.Domicile = p
	}
	if p := strPtr("status"); p != nil {
		out.Status = p
	}
	if p := strPtr("lei"); p != nil {
		out.LEI = p
	}
	typed := map[string]bool{
		"party_cd": true, "legal_name": true, "party_type": true, "segment": true,
		"tax_id": true, "domicile": true, "status": true, "lei": true,
	}
	custom := map[string]any{}
	for k, v := range golden {
		if typed[k] {
			continue
		}
		custom[k] = v
	}
	b, _ := json.Marshal(custom)
	out.CustomAttributes = b
	sourcesUsed := map[string]string{}
	for field, rule := range fieldRules {
		if _, ok := golden[field]; ok {
			sourcesUsed[field] = winningSource(field, sources, rule, now)
		}
	}
	if len(sourcesUsed) == 0 {
		sourcesUsed["*"] = group[0].SourceSystem
	}
	return out, sourcesUsed, nil, nil
}

func upsertParty(ctx context.Context, tx *sqlx.Tx, row PartyStagingRow, custom map[string]any, sourcesUsed map[string]string) error {
	cd := deref(row.PartyCd)
	if cd == "" {
		return fmt.Errorf("party_cd required")
	}
	name := deref(row.LegalName)
	if name == "" {
		name = cd
	}
	ptype := deref(row.PartyType)
	if ptype == "" {
		ptype = "ORGANIZATION"
	}

	merged := map[string]any{}
	var priorCustom []byte
	var priorID uuid.UUID
	err := tx.QueryRowxContext(ctx, `
		SELECT id, custom_attributes FROM mdm.party
		WHERE tenant_id=$1 AND party_cd=$2`, row.TenantID, cd).Scan(&priorID, &priorCustom)
	if err == nil && len(priorCustom) > 0 {
		_ = json.Unmarshal(priorCustom, &merged)
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	for k, v := range custom {
		merged[k] = v
	}
	// Keep provenance lightly
	merged["_source_systems"] = sourcesUsed
	customJSON, _ := json.Marshal(merged)

	if err == nil {
		_, err = tx.ExecContext(ctx, `
			UPDATE mdm.party SET
				legal_name=$3, party_type=$4, segment=$5, tax_id=$6, domicile=$7,
				status=$8, lei=$9, custom_attributes=$10::jsonb, updated_at=now()
			WHERE id=$1 AND tenant_id=$2`,
			priorID, row.TenantID, name, ptype, row.Segment, row.TaxID, row.Domicile,
			row.Status, row.LEI, string(customJSON),
		)
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO mdm.party (
			tenant_id, party_cd, legal_name, party_type, segment, tax_id, domicile,
			status, lei, custom_attributes
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10::jsonb)`,
		row.TenantID, cd, name, ptype, row.Segment, row.TaxID, row.Domicile,
		row.Status, row.LEI, string(customJSON),
	)
	return err
}

// LookupPartyID returns the mdm.party id for a party_cd (for reference smoke tests).
func LookupPartyID(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, partyCd string) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.GetContext(ctx, &id, `
		SELECT id FROM mdm.party WHERE tenant_id=$1 AND party_cd=$2`, tenantID, partyCd)
	return id, err
}
