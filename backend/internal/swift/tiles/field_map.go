package tiles

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type FieldMapConfig struct {
	SemanticRoot string `json:"semantic_root"` // "settlement"
	DropUnmapped bool   `json:"drop_unmapped"`
}

// TagMappingLoader abstracts the DB lookup (for testing with fakes).
type TagMappingLoader interface {
	LoadMappings(ctx context.Context, tenantID, swiftVersion, msgType string) ([]FieldMapping, error)
}

type FieldMapping struct {
	FieldTag      string
	SemanticField string
	Required      bool
	DefaultValue  sql.NullString
	TransformFn   sql.NullString
}

func NewFieldMapTransform(cfg FieldMapConfig, loader TagMappingLoader) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string

		tctx := TenantFromContext(ctx)

		for _, rec := range records {
			msgType, _ := rec["msg_type"].(string)
			swiftVersion, ok := rec["swift_version"].(string)
			if !ok {
				swiftVersion = "MT"
			}

			mappings, err := loader.LoadMappings(ctx, tctx.TenantID, swiftVersion, msgType)
			if err != nil {
				errs = append(errs, fmt.Sprintf("field_map: failed to load mappings: %v", err))
				continue
			}

			fields, ok := rec["fields"].(map[string]string)
			if !ok {
				errs = append(errs, "field_map: missing fields map from decode step")
				continue
			}

			semantic := make(map[string]any)
			for _, m := range mappings {
				val, exists := fields[m.FieldTag]
				if !exists {
					if m.DefaultValue.Valid {
						val = m.DefaultValue.String
						exists = true
					}
				}

				if !exists {
					if m.Required {
						errs = append(errs, fmt.Sprintf("field_map: required field %s not found (tag %s)", m.SemanticField, m.FieldTag))
					}
					continue
				}

				if m.TransformFn.Valid && m.TransformFn.String != "" {
					val = applyTransformFn(val, m.TransformFn.String)
				}
				semantic[m.SemanticField] = val
			}

			newRec := make(Record, len(rec)+1)
			for k, v := range rec {
				newRec[k] = v
			}
			newRec["semantic"] = semantic
			out = append(out, newRec)
		}

		return out, errs, nil
	}
}

func applyTransformFn(val, fn string) string {
	switch fn {
	case "upper":
		return strings.ToUpper(val)
	case "lower":
		return strings.ToLower(val)
	case "parse_swift_date":
		re := regexp.MustCompile(`\d{8}`)
		match := re.FindString(val)
		if match != "" {
			t, err := time.Parse("20060102", match)
			if err == nil {
				return t.Format(time.RFC3339)
			}
		}
		return val
	case "parse_amount_ccy":
		val = strings.ReplaceAll(val, ",", ".")
		reNum := regexp.MustCompile(`\d+(?:\.\d+)?`)
		num := reNum.FindString(val)
		if num != "" {
			return num
		}
		return val
	case "parse_isin":
		val = strings.Replace(val, "ISIN ", "", 1)
		return strings.TrimSpace(val)
	}
	return val
}

type DBTagMappingLoader struct {
	DB *sql.DB
}

func (l *DBTagMappingLoader) LoadMappings(ctx context.Context, tenantID, swiftVersion, msgType string) ([]FieldMapping, error) {
	query := `
SELECT field_tag, semantic_field, required, default_value, transform_fn
FROM swift_field_map
WHERE (tenant_id = $1 OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
  AND swift_version = $2
  AND msg_type = $3
ORDER BY (tenant_id = $1) DESC NULLS LAST, semantic_field
`
	rows, err := l.DB.QueryContext(ctx, query, tenantID, swiftVersion, msgType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// We deduplicate by semantic_field so that tenant overrides gold_copy
	// since we ORDER BY (tenant_id = $1) DESC NULLS LAST
	// That means tenant rows come first, then gold copy. We just take the first one we see.
	var mappings []FieldMapping
	seen := make(map[string]bool)
	for rows.Next() {
		var m FieldMapping
		if err := rows.Scan(&m.FieldTag, &m.SemanticField, &m.Required, &m.DefaultValue, &m.TransformFn); err != nil {
			return nil, err
		}
		if !seen[m.SemanticField] {
			seen[m.SemanticField] = true
			mappings = append(mappings, m)
		}
	}
	return mappings, rows.Err()
}
