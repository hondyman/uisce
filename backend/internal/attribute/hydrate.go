package attribute

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// FieldProjection maps a logical BO/API field name onto a JSONB key.
type FieldProjection struct {
	LogicalName string // BO field_name or attribute name exposed to API
	FieldCd     string // key inside custom_attributes
	DataType    string
}

// LoadProjectionsForTable returns projections for custom attributes on a table.
// Logical names include field_cd and, when present, BO field names bound to the semantic term.
func LoadProjectionsForTable(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, tableRef string) ([]FieldProjection, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		`SELECT set_config('app.current_tenant', $1, true)`, tenantID.String()); err != nil {
		return nil, err
	}

	rows, err := tx.QueryxContext(ctx, `
		SELECT a.field_cd, a.data_type, a.semantic_term_id,
		       COALESCE(bf.field_name, a.field_cd) AS logical_name
		FROM public.attribute_def_effective a
		LEFT JOIN public.business_object_fields bf
		  ON bf.term_node_id = a.semantic_term_id
		WHERE a.is_active
		  AND a.table_ref = $1
	`, tableRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := map[string]bool{}
	var out []FieldProjection
	for rows.Next() {
		var fieldCd, dataType, logical string
		var termID *uuid.UUID
		if err := rows.Scan(&fieldCd, &dataType, &termID, &logical); err != nil {
			return nil, err
		}
		add := func(name string) {
			key := strings.ToLower(name) + "|" + fieldCd
			if seen[key] || name == "" {
				return
			}
			seen[key] = true
			out = append(out, FieldProjection{
				LogicalName: name,
				FieldCd:     fieldCd,
				DataType:    dataType,
			})
		}
		add(fieldCd)
		add(logical)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = tx.Commit()
	return out, nil
}

// DehydrateRecord moves custom-attribute logical keys from a flat record into
// custom_attributes JSONB. Returns the mutated record ready for INSERT/UPDATE.
func DehydrateRecord(record map[string]any, projections []FieldProjection) map[string]any {
	if record == nil || len(projections) == 0 {
		return record
	}

	byLogical := map[string]FieldProjection{}
	for _, p := range projections {
		byLogical[strings.ToLower(p.LogicalName)] = p
		byLogical[strings.ToLower(p.FieldCd)] = p
	}

	custom := map[string]any{}
	if raw, ok := record["custom_attributes"]; ok && raw != nil {
		switch v := raw.(type) {
		case map[string]any:
			for k, val := range v {
				custom[k] = val
			}
		case string:
			_ = json.Unmarshal([]byte(v), &custom)
		case []byte:
			_ = json.Unmarshal(v, &custom)
		}
	}

	for k, v := range record {
		lk := strings.ToLower(k)
		if lk == "custom_attributes" || lk == "customattributes" {
			continue
		}
		p, ok := byLogical[lk]
		if !ok {
			continue
		}
		custom[p.FieldCd] = v
		delete(record, k)
	}

	if len(custom) > 0 {
		record["custom_attributes"] = custom
	}
	return record
}

// HydrateRecord expands custom_attributes JSONB keys onto top-level logical fields.
// Physical custom_attributes blob is preserved.
func HydrateRecord(record map[string]any, projections []FieldProjection) map[string]any {
	if record == nil || len(projections) == 0 {
		return record
	}

	raw, ok := record["custom_attributes"]
	if !ok || raw == nil {
		return record
	}

	custom := map[string]any{}
	switch v := raw.(type) {
	case map[string]any:
		custom = v
	case string:
		if err := json.Unmarshal([]byte(v), &custom); err != nil {
			return record
		}
		record["custom_attributes"] = custom
	case []byte:
		if err := json.Unmarshal(v, &custom); err != nil {
			return record
		}
		record["custom_attributes"] = custom
	default:
		return record
	}

	for _, p := range projections {
		if val, ok := custom[p.FieldCd]; ok {
			if _, exists := record[p.LogicalName]; !exists {
				record[p.LogicalName] = val
			}
			if p.LogicalName != p.FieldCd {
				if _, exists := record[p.FieldCd]; !exists {
					record[p.FieldCd] = val
				}
			}
		}
	}
	return record
}

// HydrateRecords applies HydrateRecord to each row.
func HydrateRecords(records []map[string]any, projections []FieldProjection) {
	for i := range records {
		records[i] = HydrateRecord(records[i], projections)
	}
}

// SQLSelectJSONPathExpr builds a postgres select expression for a custom field.
func SQLSelectJSONPathExpr(tableAlias, fieldCd, asName string) string {
	alias := tableAlias
	if alias == "" {
		alias = "t"
	}
	return fmt.Sprintf("%s.custom_attributes->>%s AS %s",
		alias, quoteLiteral(fieldCd), mustQuoteIdent(asName))
}

func mustQuoteIdent(name string) string {
	ident, err := quoteIdent(name)
	if err != nil {
		return `"_invalid"`
	}
	return ident
}
