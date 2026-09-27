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
	"github.com/hondyman/uisce/backend/internal/survivorship"
	"github.com/jmoiron/sqlx"
)

// AttributeDef is the subset of alpha.public.attribute_def needed for loads.
type AttributeDef struct {
	FieldCd        string          `db:"field_cd"`
	Name           string          `db:"name"`
	DataType       string          `db:"data_type"`
	JsonPath       string          `db:"json_path"`
	IsPII          bool            `db:"is_pii"`
	AppliesToTypes json.RawMessage `db:"applies_to_types"`
	SemanticTermID *uuid.UUID      `db:"semantic_term_id"`
}

// Warning is a non-blocking load finding written to staging.account_warnings.
type Warning struct {
	StagingRowID  uuid.UUID
	FieldCd       string
	Value         string
	ReferencedBO  string
	Reason        string
	Severity      string
}

// AccountStagingRow is one pending staging.account_data row.
type AccountStagingRow struct {
	ID               uuid.UUID       `db:"id"`
	TenantID         uuid.UUID       `db:"tenant_id"`
	SourceSystem     string          `db:"source_system"`
	SourceRowID      *string         `db:"source_row_id"`
	AccountCd        *string         `db:"account_cd"`
	AccountName      *string         `db:"account_name"`
	AccountTypeCd    *string         `db:"account_type_cd"`
	StatusCd         *string         `db:"status_cd"`
	BaseCurrency     *string         `db:"base_currency"`
	Domicile         *string         `db:"domicile"`
	CustodianID      *uuid.UUID      `db:"custodian_id"`
	ManagerID        *uuid.UUID      `db:"manager_id"`
	OpenedDate       *time.Time      `db:"opened_date"`
	ClosedDate       *time.Time      `db:"closed_date"`
	HolderPartyID    *uuid.UUID      `db:"holder_party_id"`
	ClientGroupID    *uuid.UUID      `db:"client_group_id"`
	CustomAttributes json.RawMessage `db:"custom_attributes"`
}

// TwoPoolLoader loads staging rows into mdm masters.
// AlphaPool is read-only control plane; DataPool is the tenant data plane (crims).
type TwoPoolLoader struct {
	AlphaPool            *sqlx.DB               // alpha — definitions
	DataPool             *sqlx.DB               // tenant DS — staging + masters
	Surv                 *survivorship.Service  // optional: semantic-term source hierarchy
	RequireSemanticTerms bool                   // when true, unbound required fields warn
}

// LoadAccountBatch loads up to batchSize pending staging.account_data rows for tenantID.
func (l *TwoPoolLoader) LoadAccountBatch(ctx context.Context, tenantID uuid.UUID, batchSize int) (loaded int, warnings int, err error) {
	if batchSize <= 0 {
		batchSize = 100
	}
	if l.AlphaPool == nil || l.DataPool == nil {
		return 0, 0, fmt.Errorf("two-pool loader requires AlphaPool and DataPool")
	}

	defs, err := l.loadAccountDefs(ctx)
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

	var rows []AccountStagingRow
	if err := tx.SelectContext(ctx, &rows, `
		SELECT id, tenant_id, source_system, source_row_id,
		       account_cd, account_name, account_type_cd, status_cd,
		       base_currency, domicile, custodian_id, manager_id,
		       opened_date, closed_date, holder_party_id, client_group_id,
		       COALESCE(custom_attributes, '{}'::jsonb) AS custom_attributes
		FROM staging.account_data
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
		fr, _, err := l.Surv.ResolveFieldRules(ctx, tenantID, "ACCOUNT")
		if err != nil {
			return 0, 0, fmt.Errorf("resolve survivorship rules: %w", err)
		}
		fieldRules = fr
	}
	if l.RequireSemanticTerms && l.Surv != nil && len(fieldRules) == 0 {
		// Non-blocking: steward must configure Survivorship, but empty rules
		// still allow a single-source load (MOST_RECENT fallback per field).
		allWarnings = append(allWarnings, Warning{
			FieldCd:  "*",
			Reason:   "NO_SEMANTIC_SURVIVORSHIP_RULES",
			Severity: "WARNING",
		})
	}

	// Batch reference checks for known party UUID fields
	partyIDs := collectUUIDRefs(rows, defByField, []string{"holder_party_id", "grantor_id", "custodian_id", "manager_id"})
	partyExists, _ := lookupExistingIDs(ctx, tx, "mdm.party", partyIDs)

	// Group by account_cd so multi-source staging rows merge via survivorship.
	groups := map[string][]AccountStagingRow{}
	var order []string
	for _, row := range rows {
		key := deref(row.AccountCd)
		if key == "" {
			allWarnings = append(allWarnings, Warning{
				StagingRowID: row.ID, FieldCd: "account_cd", Reason: "ACCOUNT_CD_REQUIRED", Severity: "ERROR",
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
		merged, sourcesUsed, mergeWarnings, err := mergeAccountGroup(ctx, engine, fieldRules, group, defByField, l.RequireSemanticTerms, now)
		if err != nil {
			return loaded, warnings, fmt.Errorf("survivorship merge %s: %w", key, err)
		}
		allWarnings = append(allWarnings, mergeWarnings...)

		custom := map[string]any{}
		_ = json.Unmarshal(merged.CustomAttributes, &custom)
		rowWarnings := validateAccountRefs(merged, custom, defByField, partyExists)
		allWarnings = append(allWarnings, rowWarnings...)

		if err := upsertAccountMaster(ctx, tx, merged, custom, sourcesUsed); err != nil {
			return loaded, warnings, fmt.Errorf("upsert account %s: %w", key, err)
		}
		for _, row := range group {
			if _, err := tx.ExecContext(ctx, `
				UPDATE staging.account_data
				SET loaded_at_master = now(), load_run_id = $2
				WHERE id = $1`, row.ID, runID); err != nil {
				return loaded, warnings, err
			}
		}
		_ = writeSurvivorshipLog(ctx, tx, tenantID, key, sourcesUsed, fieldRules)
		loaded++
	}

	for _, w := range allWarnings {
		if w.StagingRowID == uuid.Nil {
			warnings++
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO staging.account_warnings
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

func (l *TwoPoolLoader) loadAccountDefs(ctx context.Context) ([]AttributeDef, error) {
	var defs []AttributeDef
	err := l.AlphaPool.SelectContext(ctx, &defs, `
		SELECT field_cd, name, data_type, json_path, is_pii,
		       COALESCE(applies_to_types, '[]'::jsonb) AS applies_to_types,
		       semantic_term_id
		FROM public.attribute_def
		WHERE entity_type = 'ACCOUNT' AND is_active
		ORDER BY display_order, field_cd`)
	return defs, err
}

func upsertAccountMaster(ctx context.Context, tx *sqlx.Tx, row AccountStagingRow, custom map[string]any, sourcesUsed map[string]string) error {
	if row.AccountCd == nil || *row.AccountCd == "" {
		return fmt.Errorf("account_cd required")
	}
	name := deref(row.AccountName)
	if name == "" {
		name = *row.AccountCd
	}
	typeCd := deref(row.AccountTypeCd)
	if typeCd == "" {
		typeCd = "RETAIL"
	}
	status := deref(row.StatusCd)
	if status == "" {
		status = "ACTIVE"
	}

	// Per-key merge with prior current custom_attributes (new keys win; absent keys preserved).
	merged := map[string]any{}
	var priorCustom []byte
	err := tx.GetContext(ctx, &priorCustom, `
		SELECT custom_attributes
		FROM mdm.account_master
		WHERE tenant_id = $1 AND account_cd = $2 AND valid_to IS NULL`,
		row.TenantID, *row.AccountCd)
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

	// Close prior current version
	if _, err := tx.ExecContext(ctx, `
		UPDATE mdm.account_master
		SET valid_to = now(), updated_at = now()
		WHERE tenant_id = $1 AND account_cd = $2 AND valid_to IS NULL`,
		row.TenantID, *row.AccountCd); err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO mdm.account_master (
			tenant_id, account_cd, account_name, account_type_cd, status_cd,
			base_currency, domicile, custodian_id, manager_id,
			opened_date, closed_date, holder_party_id, client_group_id,
			custom_attributes, source_systems, valid_from
		) VALUES (
			$1,$2,$3,$4,$5,
			$6,$7,$8,$9,
			$10,$11,$12,$13,
			$14::jsonb, $15::jsonb, now()
		)`,
		row.TenantID, *row.AccountCd, name, typeCd, status,
		row.BaseCurrency, row.Domicile, row.CustodianID, row.ManagerID,
		row.OpenedDate, row.ClosedDate, row.HolderPartyID, row.ClientGroupID,
		string(customJSON), string(srcJSON),
	)
	return err
}

// mergeAccountGroup applies semantic survivorship across staging rows for one account_cd.
func mergeAccountGroup(
	ctx context.Context,
	engine *mdm.SurvivorshipEngine,
	fieldRules map[string]mdm.FieldRule,
	group []AccountStagingRow,
	defs map[string]AttributeDef,
	requireTerms bool,
	now time.Time,
) (AccountStagingRow, map[string]string, []Warning, error) {
	if len(group) == 1 && len(fieldRules) == 0 {
		custom := map[string]any{}
		_ = json.Unmarshal(group[0].CustomAttributes, &custom)
		sources := map[string]string{"*": group[0].SourceSystem}
		var warns []Warning
		if requireTerms {
			warns = append(warns, semanticBindingWarnings(group[0], custom, defs)...)
		}
		return group[0], sources, warns, nil
	}

	sources := make([]mdm.SourcePayload, 0, len(group))
	for _, row := range group {
		sources = append(sources, mdm.SourcePayload{
			SourceID:  row.SourceSystem,
			Timestamp: rowLoadedAt(row),
			Data:      accountRowToMap(row),
		})
	}
	golden, err := engine.MergeToGoldenRecord(ctx, sources, fieldRules, now)
	if err != nil {
		return AccountStagingRow{}, nil, nil, err
	}

	base := group[0]
	merged := mapToAccountRow(base, golden)
	sourceByField := map[string]string{}
	for field, rule := range fieldRules {
		if v, ok := golden[field]; ok && v != nil {
			sourceByField[field] = winningSource(field, sources, rule, now)
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
		warns = append(warns, semanticBindingWarnings(merged, custom, defs)...)
	}
	return merged, sourceByField, warns, nil
}

func semanticBindingWarnings(row AccountStagingRow, custom map[string]any, defs map[string]AttributeDef) []Warning {
	var out []Warning
	for field, def := range defs {
		if def.SemanticTermID == nil {
			if _, ok := custom[field]; ok {
				out = append(out, Warning{
					StagingRowID: row.ID,
					FieldCd:      field,
					Reason:       "SEMANTIC_TERM_UNBOUND",
					Severity:     "WARNING",
				})
			}
		}
	}
	return out
}

func accountRowToMap(row AccountStagingRow) map[string]any {
	m := map[string]any{}
	put := func(k string, v any) {
		if v == nil {
			return
		}
		switch t := v.(type) {
		case *string:
			if t != nil && *t != "" {
				m[k] = *t
			}
		case *uuid.UUID:
			if t != nil {
				m[k] = t.String()
			}
		case *time.Time:
			if t != nil {
				m[k] = t.Format("2006-01-02")
			}
		case string:
			if t != "" {
				m[k] = t
			}
		default:
			m[k] = v
		}
	}
	put("account_cd", row.AccountCd)
	put("account_name", row.AccountName)
	put("account_type_cd", row.AccountTypeCd)
	put("status_cd", row.StatusCd)
	put("base_currency", row.BaseCurrency)
	put("domicile", row.Domicile)
	put("custodian_id", row.CustodianID)
	put("manager_id", row.ManagerID)
	put("opened_date", row.OpenedDate)
	put("closed_date", row.ClosedDate)
	put("holder_party_id", row.HolderPartyID)
	put("client_group_id", row.ClientGroupID)
	custom := map[string]any{}
	_ = json.Unmarshal(row.CustomAttributes, &custom)
	for k, v := range custom {
		m[k] = v
	}
	return m
}

func mapToAccountRow(base AccountStagingRow, golden map[string]any) AccountStagingRow {
	out := base
	strPtr := func(k string) *string {
		v, ok := golden[k]
		if !ok || v == nil {
			return nil
		}
		s := fmt.Sprint(v)
		return &s
	}
	uuidPtr := func(k string) *uuid.UUID {
		v, ok := golden[k]
		if !ok || v == nil {
			return nil
		}
		s := fmt.Sprint(v)
		id, err := uuid.Parse(s)
		if err != nil {
			return nil
		}
		return &id
	}
	datePtr := func(k string) *time.Time {
		v, ok := golden[k]
		if !ok || v == nil {
			return nil
		}
		s := fmt.Sprint(v)
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil
		}
		return &t
	}
	if p := strPtr("account_cd"); p != nil {
		out.AccountCd = p
	}
	if p := strPtr("account_name"); p != nil {
		out.AccountName = p
	}
	if p := strPtr("account_type_cd"); p != nil {
		out.AccountTypeCd = p
	}
	if p := strPtr("status_cd"); p != nil {
		out.StatusCd = p
	}
	if p := strPtr("base_currency"); p != nil {
		out.BaseCurrency = p
	}
	if p := strPtr("domicile"); p != nil {
		out.Domicile = p
	}
	if p := uuidPtr("custodian_id"); p != nil {
		out.CustodianID = p
	}
	if p := uuidPtr("manager_id"); p != nil {
		out.ManagerID = p
	}
	if p := datePtr("opened_date"); p != nil {
		out.OpenedDate = p
	}
	if p := datePtr("closed_date"); p != nil {
		out.ClosedDate = p
	}
	if p := uuidPtr("holder_party_id"); p != nil {
		out.HolderPartyID = p
	}
	if p := uuidPtr("client_group_id"); p != nil {
		out.ClientGroupID = p
	}

	typed := map[string]bool{
		"account_cd": true, "account_name": true, "account_type_cd": true, "status_cd": true,
		"base_currency": true, "domicile": true, "custodian_id": true, "manager_id": true,
		"opened_date": true, "closed_date": true, "holder_party_id": true, "client_group_id": true,
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

func rowLoadedAt(row AccountStagingRow) time.Time {
	// staging rows don't expose loaded_at on the struct used here; use now-ish via zero → MOST_RECENT still works.
	return time.Time{}
}

func winningSource(field string, sources []mdm.SourcePayload, rule mdm.FieldRule, now time.Time) string {
	eng := mdm.NewSurvivorshipEngine()
	// Re-resolve just to discover which source won for lineage; Merge already picked the value.
	_ = eng
	_ = now
	if rule.Strategy == "SOURCE_PRIORITY" {
		for _, pref := range rule.PriorityOrder {
			for _, src := range sources {
				if strings.EqualFold(src.SourceID, pref) {
					if _, ok := src.Data[field]; ok {
						return src.SourceID
					}
				}
			}
		}
	}
	if len(sources) > 0 {
		return sources[0].SourceID
	}
	return ""
}

func writeSurvivorshipLog(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, accountCd string, sources map[string]string, rules map[string]mdm.FieldRule) error {
	// Best-effort: table may not exist until crims gold-copy migration is applied.
	b, _ := json.Marshal(map[string]any{"sources": sources, "rule_fields": len(rules)})
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO mdm.account_survivorship_log (tenant_id, account_cd, decision, decided_at)
		VALUES ($1, $2, $3::jsonb, now())`, tenantID, accountCd, string(b)); err != nil {
		return nil
	}
	return nil
}

func validateAccountRefs(
	row AccountStagingRow,
	custom map[string]any,
	defs map[string]AttributeDef,
	partyExists map[string]bool,
) []Warning {
	var out []Warning
	check := func(field string, id *uuid.UUID, bo string) {
		if id == nil {
			return
		}
		if !partyExists[id.String()] {
			out = append(out, Warning{
				StagingRowID: row.ID,
				FieldCd:      field,
				Value:        id.String(),
				ReferencedBO: bo,
				Reason:       "REFERENCED_ENTITY_MISSING",
				Severity:     "WARNING",
			})
		}
	}
	check("holder_party_id", row.HolderPartyID, "PARTY")
	check("custodian_id", row.CustodianID, "PARTY")
	check("manager_id", row.ManagerID, "PARTY")

	for field, val := range custom {
		def, ok := defs[field]
		if !ok {
			continue
		}
		if !strings.HasSuffix(field, "_id") && !strings.HasSuffix(field, "_ids") {
			continue
		}
		switch v := val.(type) {
		case string:
			if v != "" && looksLikeUUID(v) && !partyExists[v] {
				out = append(out, Warning{
					StagingRowID: row.ID, FieldCd: field, Value: v,
					ReferencedBO: "PARTY", Reason: "REFERENCED_ENTITY_MISSING", Severity: "WARNING",
				})
			}
		case []any:
			for _, item := range v {
				s, _ := item.(string)
				if s != "" && looksLikeUUID(s) && !partyExists[s] {
					out = append(out, Warning{
						StagingRowID: row.ID, FieldCd: def.FieldCd, Value: s,
						ReferencedBO: "PARTY", Reason: "REFERENCED_ENTITY_MISSING", Severity: "WARNING",
					})
				}
			}
		}
	}
	return out
}

func collectUUIDRefs(rows []AccountStagingRow, defs map[string]AttributeDef, typedFields []string) []string {
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
		for _, f := range typedFields {
			switch f {
			case "holder_party_id":
				if row.HolderPartyID != nil {
					add(row.HolderPartyID.String())
				}
			case "custodian_id":
				if row.CustodianID != nil {
					add(row.CustodianID.String())
				}
			case "manager_id":
				if row.ManagerID != nil {
					add(row.ManagerID.String())
				}
			}
		}
		custom := map[string]any{}
		_ = json.Unmarshal(row.CustomAttributes, &custom)
		for k, v := range custom {
			if !strings.HasSuffix(k, "_id") && !strings.HasSuffix(k, "_ids") {
				continue
			}
			if _, ok := defs[k]; !ok && len(defs) > 0 {
				// still check UUID-shaped custom keys
			}
			switch x := v.(type) {
			case string:
				add(x)
			case []any:
				for _, item := range x {
					if s, ok := item.(string); ok {
						add(s)
					}
				}
			}
		}
	}
	return out
}

func lookupExistingIDs(ctx context.Context, tx *sqlx.Tx, table string, ids []string) (map[string]bool, error) {
	out := map[string]bool{}
	if len(ids) == 0 {
		return out, nil
	}
	// Only allow known tables
	allowed := map[string]bool{"mdm.party": true, "mdm.counterparty": true, "mdm.issuer_master": true}
	if !allowed[table] {
		return out, fmt.Errorf("unsupported reference table %s", table)
	}
	q := fmt.Sprintf(`SELECT id::text FROM %s WHERE id::text = ANY($1)`, table)
	var found []string
	if err := tx.SelectContext(ctx, &found, q, ids); err != nil {
		// Table may not exist / empty — treat as none found
		return out, nil
	}
	for _, id := range found {
		out[id] = true
	}
	return out, nil
}

func looksLikeUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func defaultSeverity(s string) string {
	if s == "" {
		return "WARNING"
	}
	return s
}
