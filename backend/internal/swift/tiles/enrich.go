package tiles

import (
	"context"
	"database/sql"
	"fmt"
)

type LookupSpec struct {
	SourceField string `json:"source_field"` // from semantic map
	TargetTable string `json:"target_table"` // "oms.security"
	TargetField string `json:"target_field"` // output field name
	LookupKey   string `json:"lookup_key"`   // column to match on (default: same as source_field)
}

type EnrichConfig struct {
	Lookups []LookupSpec `json:"lookups"`
}

func NewEnrichTransform(cfg EnrichConfig, db *sql.DB) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string
		tctx := TenantFromContext(ctx)

		for _, rec := range records {
			semantic, ok := rec["semantic"].(map[string]any)
			if !ok {
				errs = append(errs, "enrich: missing semantic map")
				continue
			}

			for _, spec := range cfg.Lookups {
				val, ok := semantic[spec.SourceField]
				if !ok || val == nil {
					continue
				}

				lookupKey := spec.LookupKey
				if lookupKey == "" {
					lookupKey = spec.SourceField
				}

				query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s = $1 AND (tenant_id = $2 OR tenant_id = public.uisce_gold_copy_tenant_id()) ORDER BY (tenant_id = $2) DESC NULLS LAST LIMIT 1`, spec.TargetField, spec.TargetTable, lookupKey)

				var result string
				err := db.QueryRowContext(ctx, query, val, tctx.TenantID).Scan(&result)
				if err != nil {
					if err != sql.ErrNoRows {
						errs = append(errs, fmt.Sprintf("enrich: lookup failed for %s: %v", spec.SourceField, err))
					}
					continue
				}
				semantic[spec.TargetField] = result
			}

			rec["semantic"] = semantic
			out = append(out, rec)
		}

		return out, errs, nil
	}
}
