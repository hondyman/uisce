package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
	"github.com/jmoiron/sqlx"
)

// SecurityStagingRow is one pending staging.security_data row.
type SecurityStagingRow struct {
	ID                 uuid.UUID       `db:"id"`
	TenantID           uuid.UUID       `db:"tenant_id"`
	SourceSystem       string          `db:"source_system"`
	SourceRowID        *string         `db:"source_row_id"`
	SecurityID         *string         `db:"security_id"`
	PrimaryIdentifier  *string         `db:"primary_identifier"`
	ISIN               *string         `db:"isin"`
	CUSIP              *string         `db:"cusip"`
	SEDOL              *string         `db:"sedol"`
	FIGI               *string         `db:"figi"`
	Ticker             *string         `db:"ticker"`
	SecurityName       *string         `db:"security_name"`
	AssetClass         *string         `db:"asset_class"`
	Currency           *string         `db:"currency"`
	Status             *string         `db:"status"`
	CustomAttributes   json.RawMessage `db:"custom_attributes"`
}

// LoadSecurityBatch loads pending staging.security_data into mdm.security_master.
func (l *TwoPoolLoader) LoadSecurityBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) (loaded int, warnings int, err error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	if l.AlphaPool == nil || l.DataPool == nil {
		return 0, 0, fmt.Errorf("two-pool loader requires AlphaPool and DataPool")
	}

	defs, err := l.loadEntityDefs(ctx, "SECURITY")
	if err != nil {
		return 0, 0, fmt.Errorf("load attribute_def from alpha: %w", err)
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
		return 0, 0, fmt.Errorf("set tenant guc: %w", err)
	}

	var rows []SecurityStagingRow
	if err := tx.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, source_system, source_row_id,
		       security_id, primary_identifier, isin, cusip, sedol, figi, ticker,
		       security_name, asset_class, currency, status,
		       COALESCE(custom_attributes, '{}'::jsonb) AS custom_attributes
		FROM staging.security_data
		WHERE tenant_id = $1 AND loaded_at_master IS NULL
		ORDER BY loaded_at
		LIMIT $2`, tenantID, batchSize); err != nil {
		return 0, 0, fmt.Errorf("select staging rows: %w", err)
	}
	if len(rows) == 0 {
		return 0, 0, tx.Commit()
	}

	runID := uuid.New()
	var allWarnings []Warning

	fieldRules := map[string]mdm.FieldRule{}
	if l.Surv != nil {
		fr, _, err := l.Surv.ResolveFieldRules(ctx, tenantID, "SECURITY")
		if err != nil {
			return 0, 0, fmt.Errorf("resolve survivorship rules: %w", err)
		}
		fieldRules = fr
	}
	if l.RequireSemanticTerms && l.Surv != nil && len(fieldRules) == 0 {
		allWarnings = append(allWarnings, Warning{
			FieldCd: "*", Reason: "NO_SEMANTIC_SURVIVORSHIP_RULES", Severity: "WARNING",
		})
	}

	groups := map[string][]SecurityStagingRow{}
	var order []string
	for _, row := range rows {
		key := securityNaturalKey(row)
		if key == "" {
			allWarnings = append(allWarnings, Warning{
				StagingRowID: row.ID, FieldCd: "security_id", Reason: "SECURITY_ID_REQUIRED", Severity: "ERROR",
			})
			continue
		}
		if _, ok := groups[key]; !ok {
			order = append(order, key)
		}
		groups[key] = append(groups[key], row)
	}

	// Collect issuer_id refs (typed custom_attributes or future staging column) for Party lookup.
	issuerIDs := collectSecurityIssuerIDs(rows)
	partyExists, _ := lookupExistingIDs(ctx, tx, "mdm.party", issuerIDs)

	engine := mdm.NewSurvivorshipEngine()
	now := time.Now().UTC()

	for _, key := range order {
		group := groups[key]
		merged, sourcesUsed, mergeWarnings, err := mergeSecurityGroup(ctx, engine, fieldRules, group, defByField, l.RequireSemanticTerms, now)
		if err != nil {
			return loaded, warnings, fmt.Errorf("survivorship merge %s: %w", key, err)
		}
		allWarnings = append(allWarnings, mergeWarnings...)

		custom := map[string]any{}
		_ = json.Unmarshal(merged.CustomAttributes, &custom)
		allWarnings = append(allWarnings, validateSecurityIssuerRefs(merged, custom, partyExists)...)

		if err := upsertSecurityMaster(ctx, tx, merged, custom, sourcesUsed); err != nil {
			return loaded, warnings, fmt.Errorf("upsert security %s: %w", key, err)
		}
		for _, row := range group {
			if _, err := tx.ExecContext(ctx, `
				UPDATE staging.security_data
				SET loaded_at_master = now(), load_run_id = $2
				WHERE id = $1`, row.ID, runID); err != nil {
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
			INSERT INTO staging.security_warnings
				(tenant_id, staging_row_id, load_run_id, field_cd, value, referenced_bo, reason, severity)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			tenantID, w.StagingRowID, runID, w.FieldCd, w.Value, nullIfEmpty(w.ReferencedBO), w.Reason, defaultSeverity(w.Severity),
		); err != nil {
			return loaded, warnings, fmt.Errorf("write warning: %w", err)
		}
		warnings++
	}

	if err := tx.Commit(); err != nil {
		return loaded, warnings, err
	}
	return loaded, warnings, nil
}

func (l *TwoPoolLoader) loadEntityDefs(ctx context.Context, entityType string) ([]AttributeDef, error) {
	var defs []AttributeDef
	err := l.AlphaPool.SelectContext(ctx, &defs, `
		SELECT field_cd, name, data_type, json_path, is_pii,
		       COALESCE(applies_to_types, '[]'::jsonb) AS applies_to_types,
		       semantic_term_id
		FROM public.attribute_def
		WHERE entity_type = $1 AND is_active
		ORDER BY display_order, field_cd`, entityType)
	return defs, err
}

func securityNaturalKey(row SecurityStagingRow) string {
	if row.SecurityID != nil && *row.SecurityID != "" {
		return *row.SecurityID
	}
	if row.PrimaryIdentifier != nil && *row.PrimaryIdentifier != "" {
		return *row.PrimaryIdentifier
	}
	if row.ISIN != nil && *row.ISIN != "" {
		return *row.ISIN
	}
	return ""
}

func mergeSecurityGroup(
	ctx context.Context,
	engine *mdm.SurvivorshipEngine,
	fieldRules map[string]mdm.FieldRule,
	group []SecurityStagingRow,
	defs map[string]AttributeDef,
	requireTerms bool,
	now time.Time,
) (SecurityStagingRow, map[string]string, []Warning, error) {
	if len(group) == 1 && len(fieldRules) == 0 {
		custom := map[string]any{}
		_ = json.Unmarshal(group[0].CustomAttributes, &custom)
		sources := map[string]string{"*": group[0].SourceSystem}
		var warns []Warning
		if requireTerms {
			warns = append(warns, semanticBindingWarningsSecurity(group[0], custom, defs)...)
		}
		return group[0], sources, warns, nil
	}

	sources := make([]mdm.SourcePayload, 0, len(group))
	for _, row := range group {
		sources = append(sources, mdm.SourcePayload{
			SourceID:  row.SourceSystem,
			Timestamp: time.Time{},
			Data:      securityRowToMap(row),
		})
	}
	golden, err := engine.MergeToGoldenRecord(ctx, sources, fieldRules, now)
	if err != nil {
		return SecurityStagingRow{}, nil, nil, err
	}
	merged := mapToSecurityRow(group[0], golden)
	sourceByField := map[string]string{}
	for field := range fieldRules {
		if v, ok := golden[field]; ok && v != nil {
			sourceByField[field] = winningSource(field, sources, fieldRules[field], now)
		}
	}
	if len(sourceByField) == 0 {
		for _, row := range group {
			sourceByField[row.SourceSystem] = row.SourceSystem
		}
	}
	custom := map[string]any{}
	_ = json.Unmarshal(merged.CustomAttributes, &custom)
	var warns []Warning
	if requireTerms {
		warns = append(warns, semanticBindingWarningsSecurity(merged, custom, defs)...)
	}
	return merged, sourceByField, warns, nil
}

func semanticBindingWarningsSecurity(row SecurityStagingRow, custom map[string]any, defs map[string]AttributeDef) []Warning {
	var out []Warning
	for field, def := range defs {
		if def.SemanticTermID == nil {
			if _, ok := custom[field]; ok {
				out = append(out, Warning{
					StagingRowID: row.ID, FieldCd: field,
					Reason: "SEMANTIC_TERM_UNBOUND", Severity: "WARNING",
				})
			}
		}
	}
	return out
}

func securityRowToMap(row SecurityStagingRow) map[string]any {
	m := map[string]any{}
	put := func(k string, v *string) {
		if v != nil && *v != "" {
			m[k] = *v
		}
	}
	put("security_id", row.SecurityID)
	put("primary_identifier", row.PrimaryIdentifier)
	put("isin", row.ISIN)
	put("cusip", row.CUSIP)
	put("sedol", row.SEDOL)
	put("figi", row.FIGI)
	put("ticker", row.Ticker)
	put("security_name", row.SecurityName)
	put("asset_class", row.AssetClass)
	put("currency", row.Currency)
	put("status", row.Status)
	custom := map[string]any{}
	_ = json.Unmarshal(row.CustomAttributes, &custom)
	for k, v := range custom {
		m[k] = v
	}
	return m
}

func mapToSecurityRow(base SecurityStagingRow, golden map[string]any) SecurityStagingRow {
	out := base
	strPtr := func(k string) *string {
		v, ok := golden[k]
		if !ok || v == nil {
			return nil
		}
		s := fmt.Sprint(v)
		return &s
	}
	if p := strPtr("security_id"); p != nil {
		out.SecurityID = p
	}
	if p := strPtr("primary_identifier"); p != nil {
		out.PrimaryIdentifier = p
	}
	if p := strPtr("isin"); p != nil {
		out.ISIN = p
	}
	if p := strPtr("cusip"); p != nil {
		out.CUSIP = p
	}
	if p := strPtr("sedol"); p != nil {
		out.SEDOL = p
	}
	if p := strPtr("figi"); p != nil {
		out.FIGI = p
	}
	if p := strPtr("ticker"); p != nil {
		out.Ticker = p
	}
	if p := strPtr("security_name"); p != nil {
		out.SecurityName = p
	}
	if p := strPtr("asset_class"); p != nil {
		out.AssetClass = p
	}
	if p := strPtr("currency"); p != nil {
		out.Currency = p
	}
	if p := strPtr("status"); p != nil {
		out.Status = p
	}
	typed := map[string]bool{
		"security_id": true, "primary_identifier": true, "isin": true, "cusip": true,
		"sedol": true, "figi": true, "ticker": true, "security_name": true,
		"asset_class": true, "currency": true, "status": true,
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
	return out
}

func upsertSecurityMaster(ctx context.Context, tx *sqlx.Tx, row SecurityStagingRow, custom map[string]any, sourcesUsed map[string]string) error {
	secID := deref(row.SecurityID)
	if secID == "" {
		secID = securityNaturalKey(row)
	}
	if secID == "" {
		return fmt.Errorf("security_id required")
	}
	name := deref(row.SecurityName)
	if name == "" {
		name = secID
	}
	primary := deref(row.PrimaryIdentifier)
	if primary == "" {
		primary = firstNonEmpty(deref(row.ISIN), deref(row.CUSIP), deref(row.Ticker), secID)
	}
	asset := deref(row.AssetClass)
	if asset == "" {
		asset = "Unknown"
	}
	ccy := deref(row.Currency)
	if ccy == "" {
		ccy = "USD"
	}
	status := deref(row.Status)
	if status == "" {
		status = "Active"
	}

	merged := map[string]any{}
	var priorCustom []byte
	err := tx.GetContext(ctx, &priorCustom, `
		SELECT custom_attributes FROM mdm.security_master
		WHERE tenant_id = $1 AND security_id = $2 AND valid_to IS NULL`,
		row.TenantID, secID)
	if err == nil && len(priorCustom) > 0 {
		_ = json.Unmarshal(priorCustom, &merged)
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("read prior custom_attributes: %w", err)
	}
	for k, v := range custom {
		merged[k] = v
	}
	customJSON, _ := json.Marshal(merged)
	srcJSON, _ := json.Marshal(sourcesUsed)
	if len(sourcesUsed) == 0 {
		srcJSON = []byte(fmt.Sprintf(`{"%s": true}`, strings.ReplaceAll(row.SourceSystem, `"`, "")))
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE mdm.security_master
		SET valid_to = now(), updated_at = now()
		WHERE tenant_id = $1 AND security_id = $2 AND valid_to IS NULL`,
		row.TenantID, secID); err != nil {
		return err
	}

	issuerID := issuerUUIDFromCustom(custom)

	_, err = tx.ExecContext(ctx, `
		INSERT INTO mdm.security_master (
			tenant_id, security_id, primary_identifier, isin, cusip, sedol, figi, ticker,
			security_name, asset_class, currency, status, issuer_id,
			custom_attributes, source_systems, valid_from, created_by
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,
			$9,$10,$11,$12,$13,
			$14::jsonb,$15::jsonb,now(),$16
		)`,
		row.TenantID, secID, primary, row.ISIN, row.CUSIP, row.SEDOL, row.FIGI, row.Ticker,
		name, asset, ccy, status, issuerID,
		string(customJSON), string(srcJSON), uuid.Nil,
	)
	return err
}

func collectSecurityIssuerIDs(rows []SecurityStagingRow) []string {
	seen := map[string]bool{}
	var out []string
	add := func(s string) {
		if s == "" || seen[s] || !looksLikeUUID(s) {
			return
		}
		seen[s] = true
		out = append(out, s)
	}
	for _, row := range rows {
		custom := map[string]any{}
		_ = json.Unmarshal(row.CustomAttributes, &custom)
		if v, ok := custom["issuer_id"].(string); ok {
			add(v)
		}
	}
	return out
}

func validateSecurityIssuerRefs(row SecurityStagingRow, custom map[string]any, partyExists map[string]bool) []Warning {
	v, _ := custom["issuer_id"].(string)
	if v == "" || !looksLikeUUID(v) {
		return nil
	}
	if partyExists[v] {
		return nil
	}
	return []Warning{{
		StagingRowID: row.ID,
		FieldCd:      "issuer_id",
		Value:        v,
		ReferencedBO: "PARTY",
		Reason:       "REFERENCED_ENTITY_MISSING",
		Severity:     "WARNING",
	}}
}

func issuerUUIDFromCustom(custom map[string]any) *uuid.UUID {
	v, _ := custom["issuer_id"].(string)
	if v == "" || !looksLikeUUID(v) {
		return nil
	}
	id, err := uuid.Parse(v)
	if err != nil {
		return nil
	}
	return &id
}
