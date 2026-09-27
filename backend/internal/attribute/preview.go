package attribute

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ValueDB resolves a *sqlx.DB for reading entity values (often crims / MDM).
// When nil, Service falls back to the alpha DB (same pool as definitions).
type ValueDB interface {
	DBForTable(ctx context.Context, tenantID uuid.UUID, tableRef string) (*sqlx.DB, error)
}

func (s *Service) Preview(ctx context.Context, req PreviewRequest, valueDB ValueDB) (*PreviewResponse, error) {
	if req.EntityType == "" && req.TableRef == "" {
		return nil, fmt.Errorf("%w: entity_type or table_ref required", ErrInvalidInput)
	}
	if req.Limit <= 0 || req.Limit > 500 {
		req.Limit = 50
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	var defs []AttributeDef
	var tableRef string
	err := s.withTenant(ctx, req.TenantID, func(tx *sqlx.Tx) error {
		entityType := strings.ToUpper(req.EntityType)
		q := `
			SELECT id, tenant_id, core_id, is_shadow, entity_type, table_ref, field_cd,
			       name, COALESCE(description,'') AS description, data_type, json_path,
			       is_required, is_searchable, is_pii,
			       validation_rules, picklist_values, default_value,
			       display_order, COALESCE(section,'') AS section, is_active,
			       origin, semantic_term_id, created_at, updated_at
			FROM public.attribute_def_effective
			WHERE is_active`
		args := []any{}
		if entityType != "" {
			args = append(args, entityType)
			q += fmt.Sprintf(` AND entity_type = $%d`, len(args))
		}
		if req.TableRef != "" {
			args = append(args, req.TableRef)
			q += fmt.Sprintf(` AND table_ref = $%d`, len(args))
		}
		if req.AccountType != "" {
			args = append(args, `["`+strings.ReplaceAll(req.AccountType, `"`, ``)+`"]`)
			q += fmt.Sprintf(` AND (
				jsonb_array_length(COALESCE(applies_to_types, '[]'::jsonb)) = 0
				OR COALESCE(applies_to_types, '[]'::jsonb) @> $%d::jsonb
			)`, len(args))
		}
		q += ` ORDER BY display_order, field_cd`

		rows, err := tx.QueryxContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			d, err := scanDef(rows)
			if err != nil {
				return err
			}
			defs = append(defs, d)
			if tableRef == "" {
				tableRef = d.TableRef
			}
		}
		return rows.Err()
	})
	if err != nil {
		return nil, err
	}
	if tableRef == "" {
		tableRef = req.TableRef
	}
	if tableRef == "" && strings.EqualFold(req.EntityType, "ACCOUNT") {
		tableRef = "mdm.account_master"
	}
	if tableRef == "" {
		return nil, fmt.Errorf("%w: could not resolve table_ref", ErrInvalidInput)
	}

	db := s.db
	if valueDB != nil {
		if v, err := valueDB.DBForTable(ctx, req.TenantID, tableRef); err == nil && v != nil {
			db = v
		}
	}

	coreCols, err := loadCoreColumns(ctx, db, tableRef)
	if err != nil {
		// Fall back to a minimal identity set so preview still works.
		coreCols = []CoreColumn{
			{Name: "id", DataType: "uuid", Label: "Id"},
			{Name: "tenant_id", DataType: "uuid", Label: "Tenant Id"},
		}
	}

	query, args, err := buildPreviewQuery(req, tableRef, defs, coreCols)
	if err != nil {
		return nil, err
	}

	conn, err := db.Connx(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`,
		req.TenantID.String(),
	); err != nil {
		return nil, fmt.Errorf("set tenant guc: %w", err)
	}

	rows, err := conn.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("preview query: %w", err)
	}
	defer rows.Close()

	result, err := scanRowsAsMaps(rows)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = []map[string]any{}
	}

	hasTenantID := false
	hasAccountType := false
	for _, c := range coreCols {
		if strings.EqualFold(c.Name, "tenant_id") {
			hasTenantID = true
		}
		if strings.EqualFold(c.Name, "account_type_cd") {
			hasAccountType = true
		}
	}
	total, _ := countPreviewRows(ctx, conn, tableRef, req.TenantID, hasTenantID, req.AccountType, hasAccountType)
	columns := buildColumnMeta(defs, coreCols)
	if columns == nil {
		columns = []ColumnMeta{}
	}

	return &PreviewResponse{
		Columns: columns,
		Rows:    result,
		Total:   total,
		Showing: len(result),
	}, nil
}

func loadCoreColumns(ctx context.Context, db *sqlx.DB, tableRef string) ([]CoreColumn, error) {
	schema, table, err := sanitizeTableRef(tableRef)
	if err != nil {
		return nil, err
	}
	rows, err := db.QueryxContext(ctx, `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		  AND column_name NOT IN ('custom_attributes')
		ORDER BY ordinal_position`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []CoreColumn
	for rows.Next() {
		var c CoreColumn
		if err := rows.Scan(&c.Name, &c.DataType); err != nil {
			return nil, err
		}
		c.Label = humanLabel(c.Name)
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func coreColumnSet(coreCols []CoreColumn) map[string]bool {
	out := make(map[string]bool, len(coreCols))
	for _, c := range coreCols {
		out[strings.ToLower(c.Name)] = true
	}
	return out
}

func buildPreviewQuery(
	req PreviewRequest,
	tableRef string,
	defs []AttributeDef,
	coreCols []CoreColumn,
) (string, []any, error) {
	tblSQL, err := safeTableSQL(tableRef)
	if err != nil {
		return "", nil, err
	}
	alias := "t"
	cols := coreColumnSet(coreCols)
	var selectExprs []string
	for _, c := range coreCols {
		ident, err := quoteIdent(c.Name)
		if err != nil {
			continue
		}
		selectExprs = append(selectExprs, fmt.Sprintf("%s.%s", alias, ident))
	}
	for _, d := range defs {
		ident, err := quoteIdent(d.FieldCd)
		if err != nil {
			continue
		}
		selectExprs = append(selectExprs, fmt.Sprintf("%s AS %s", buildCustomColumnExpr(alias, d), ident))
	}
	if len(selectExprs) == 0 {
		return "", nil, fmt.Errorf("no columns to select")
	}

	var args []any
	whereParts := []string{"TRUE"}
	if cols["tenant_id"] {
		args = append(args, req.TenantID)
		whereParts = []string{fmt.Sprintf("%s.tenant_id = $%d", alias, len(args))}
	}
	if req.AccountType != "" && cols["account_type_cd"] {
		args = append(args, req.AccountType)
		whereParts = append(whereParts, fmt.Sprintf("%s.account_type_cd = $%d", alias, len(args)))
	}
	where := strings.Join(whereParts, " AND ")

	orderBy := "1"
	for _, candidate := range []string{"updated_at", "created_at", "id"} {
		if cols[candidate] {
			ident, _ := quoteIdent(candidate)
			orderBy = fmt.Sprintf("%s.%s DESC", alias, ident)
			break
		}
	}
	if req.OrderBy != "" {
		parts := strings.Fields(req.OrderBy)
		col := parts[0]
		dir := "ASC"
		if len(parts) > 1 && strings.EqualFold(parts[1], "DESC") {
			dir = "DESC"
		}
		if cols[strings.ToLower(col)] {
			if ident, err := quoteIdent(col); err == nil {
				orderBy = fmt.Sprintf("%s.%s %s", alias, ident, dir)
			}
		}
	}

	args = append(args, req.Limit, req.Offset)
	limitIdx := len(args) - 1
	offsetIdx := len(args)

	query := fmt.Sprintf(`
		SELECT %s
		FROM %s %s
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`,
		strings.Join(selectExprs, ", "),
		tblSQL, alias,
		where,
		orderBy,
		limitIdx, offsetIdx,
	)
	return query, args, nil
}

func countPreviewRows(ctx context.Context, conn *sqlx.Conn, tableRef string, tenantID uuid.UUID, hasTenantID bool, accountType string, hasAccountType bool) (int64, error) {
	tblSQL, err := safeTableSQL(tableRef)
	if err != nil {
		return 0, err
	}
	where := []string{"TRUE"}
	args := []any{}
	if hasTenantID {
		args = append(args, tenantID)
		where = []string{fmt.Sprintf("t.tenant_id = $%d", len(args))}
	}
	if accountType != "" && hasAccountType {
		args = append(args, accountType)
		where = append(where, fmt.Sprintf("t.account_type_cd = $%d", len(args)))
	}
	var n int64
	err = conn.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT count(*) FROM %s t WHERE %s`, tblSQL, strings.Join(where, " AND ")),
		args...,
	).Scan(&n)
	return n, err
}

func scanRowsAsMaps(rows *sqlx.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := make(map[string]any, len(cols))
		for i, c := range cols {
			m[c] = normalizeValue(raw[i])
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func normalizeValue(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339)
	case [16]byte:
		return uuid.UUID(x).String()
	default:
		return v
	}
}

func buildColumnMeta(defs []AttributeDef, coreCols []CoreColumn) []ColumnMeta {
	out := make([]ColumnMeta, 0, len(coreCols)+len(defs))
	for _, c := range coreCols {
		out = append(out, ColumnMeta{
			Key:    c.Name,
			Label:  c.Label,
			Type:   c.DataType,
			Source: "CORE",
		})
	}
	for _, d := range defs {
		src := d.Origin
		if src == "" {
			src = "CUSTOM"
		}
		out = append(out, ColumnMeta{
			Key:     d.FieldCd,
			Label:   d.Name,
			Type:    d.DataType,
			Source:  src,
			Section: d.Section,
			FieldCd: d.FieldCd,
		})
	}
	return out
}

// AlphaValueDB uses the same alpha pool for value reads (dev / co-located MDM).
type AlphaValueDB struct {
	DB *sqlx.DB
}

func (a AlphaValueDB) DBForTable(ctx context.Context, tenantID uuid.UUID, tableRef string) (*sqlx.DB, error) {
	if a.DB == nil {
		return nil, sql.ErrConnDone
	}
	return a.DB, nil
}
