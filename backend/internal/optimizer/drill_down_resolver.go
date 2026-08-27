package optimizer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DrillDownRequest struct {
	TenantID        uuid.UUID              `json:"tenantId"`
	AggregatedField string                 `json:"aggregatedField"` // e.g. "portfolio_xirr"
	FilterContext   map[string]interface{} `json:"filterContext"`   // e.g. {"account_id": "ACC-901"}
	PageSize        int                    `json:"pageSize"`
	Offset          int                    `json:"offset"`
}

type DrillDownResponse struct {
	TargetBOKey   string                   `json:"targetBoKey"`
	TargetPageKey *string                  `json:"targetPageKey,omitempty"`
	Columns       []string                 `json:"columns"`
	Rows          []map[string]interface{} `json:"rows"`
	TotalCount    int64                    `json:"totalCount"`
}

type DrillDownResolver struct {
	db *sqlx.DB
}

func NewDrillDownResolver(db *sqlx.DB) *DrillDownResolver {
	return &DrillDownResolver{db: db}
}

func (r *DrillDownResolver) ResolveDrillDown(ctx context.Context, req DrillDownRequest) (*DrillDownResponse, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id is required (Rule 7 Violation)")
	}

	// 1. Resolve Target BO and Table Mapping from Catalog (Rule 1: Config-Before-Code)
	var pathConfig struct {
		TargetBOKey   string  `db:"target_bo_key"`
		TargetPageKey *string `db:"target_page_key"`
		TargetTable   string  `db:"target_table"`
		ColumnsRaw    []byte  `db:"default_columns"`
	}

	var hasCustomConfig bool
	if r.db != nil {
		metaQuery := `
			SELECT target_bo_key, target_page_key, target_table, default_columns
			FROM semantic_drill.calculation_drill_paths
			WHERE tenant_id = $1 AND aggregated_metric_key = $2 AND is_active = TRUE
			LIMIT 1;
		`
		err := r.db.GetContext(ctx, &pathConfig, metaQuery, req.TenantID, req.AggregatedField)
		if err == nil {
			hasCustomConfig = true
		}
	}

	// Fallback to static resolution defaults if not explicitly configured in database
	if !hasCustomConfig {
		if req.AggregatedField == "portfolio_xirr" || req.AggregatedField == "irr" {
			pathConfig.TargetBOKey = "TaxLotCashFlows"
			pathConfig.TargetTable = "mdm.tax_lot_cash_flows"
			pathConfig.ColumnsRaw = []byte(`["lot_id", "cash_flow_date", "inflow_amount", "outflow_amount", "irr_weight"]`)
		} else {
			pathConfig.TargetBOKey = "PositionMaster"
			pathConfig.TargetTable = "mdm.positions"
			pathConfig.ColumnsRaw = []byte(`["position_id", "security_name", "isin", "shares_held", "market_value"]`)
		}
	}

	var columns []string
	_ = json.Unmarshal(pathConfig.ColumnsRaw, &columns)

	// 2. Build Dynamic Parameterized Predicates (Rule 7: Tenant Fencing)
	whereClauses := []string{"tenant_id = $1"}
	args := []interface{}{req.TenantID}
	argIdx := 2

	for k, v := range req.FilterContext {
		if v != nil && v != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", k, argIdx))
			args = append(args, v)
			argIdx++
		}
	}

	limit := req.PageSize
	if limit <= 0 {
		limit = 50
	}
	offset := req.Offset

	querySQL := fmt.Sprintf(`
		SELECT %s 
		FROM %s 
		WHERE %s 
		ORDER BY 1 DESC 
		LIMIT $%d OFFSET $%d;
	`, strings.Join(columns, ", "), pathConfig.TargetTable, strings.Join(whereClauses, " AND "), argIdx, argIdx+1)

	args = append(args, limit, offset)

	// 3. Execute Pushdown Query
	rows := make([]map[string]interface{}, 0)
	if r.db != nil {
		stmt, qErr := r.db.QueryxContext(ctx, querySQL, args...)
		if qErr == nil {
			defer stmt.Close()
			for stmt.Next() {
				row := make(map[string]interface{})
				if scanErr := stmt.MapScan(row); scanErr == nil {
					rows = append(rows, row)
				}
			}
		}
	}

	return &DrillDownResponse{
		TargetBOKey:   pathConfig.TargetBOKey,
		TargetPageKey: pathConfig.TargetPageKey,
		Columns:       columns,
		Rows:          rows,
		TotalCount:    int64(len(rows)),
	}, nil
}
