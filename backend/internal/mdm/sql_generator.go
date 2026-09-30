package mdm

import (
	"errors"
	"fmt"
	"strings"
)

// RenderedQuery represents the serialized SQL query and its bound parameter arguments
type RenderedQuery struct {
	SQL  string        `json:"sql"`
	Args []interface{} `json:"args"`
}

// SQLGenerator converts a SurvivorshipBatchPlan IR into a high-performance, deterministic CTE query
type SQLGenerator struct{}

// NewSQLGenerator creates a new SQL generator instance
func NewSQLGenerator() *SQLGenerator {
	return &SQLGenerator{}
}

// RenderSQL generates the complete PostgreSQL CTE query for set-based batch survivorship
func (g *SQLGenerator) RenderSQL(plan *SurvivorshipBatchPlan) (*RenderedQuery, error) {
	if plan == nil {
		return nil, errors.New("survivorship batch plan is required")
	}
	if len(plan.Fields) == 0 {
		return nil, errors.New("plan must contain at least one field")
	}

	args := []interface{}{
		plan.TenantID,
		plan.BatchCutoff.UTC(),
	}

	// 1. Collect select columns for staging CTE
	stagingCols := []string{
		fmt.Sprintf("s.%s AS source_row_id", plan.PKField),
		fmt.Sprintf("s.%s AS entity_key", plan.EntityKeyField),
		fmt.Sprintf("s.%s AS source_cd", plan.SourceIDField),
		fmt.Sprintf("s.%s AS as_of", plan.AsOfField),
	}

	for _, f := range plan.Fields {
		stagingCols = append(stagingCols, fmt.Sprintf("s.%s", f.SourceColumn))
	}

	// 2. Build window ranking expressions for ranked_contributions CTE
	var rankingCols []string
	for _, f := range plan.Fields {
		rankAlias := fmt.Sprintf("rank_%s", f.Attribute)
		var orderClauses []string

		// Prioritize non-null values first
		orderClauses = append(orderClauses, fmt.Sprintf(
			"CASE WHEN %s IS NOT NULL THEN 0 ELSE 1 END ASC",
			f.SourceColumn,
		))

		// Add staleness ordering if configured (0 = fresh, 1 = stale)
		if f.MaxStaleSeconds != nil && *f.MaxStaleSeconds > 0 {
			orderClauses = append(orderClauses, fmt.Sprintf(
				"CASE WHEN as_of >= ($2::timestamptz - INTERVAL '%d seconds') THEN 0 ELSE 1 END ASC",
				*f.MaxStaleSeconds,
			))
			rankingCols = append(rankingCols, fmt.Sprintf(
				"CASE WHEN as_of >= ($2::timestamptz - INTERVAL '%d seconds') THEN true ELSE false END AS is_fresh_%s",
				*f.MaxStaleSeconds,
				f.Attribute,
			))
		}

		switch f.Strategy {
		case StrategySourcePriority:
			orderClauses = append(orderClauses, fmt.Sprintf(
				"array_position(ARRAY[%s]::text[], source_cd) NULLS LAST",
				formatPriorityArray(f.PriorityOrder),
			))
			orderClauses = append(orderClauses, "as_of DESC", "source_row_id ASC")

		case StrategyMostRecent:
			orderClauses = append(orderClauses, "as_of DESC", "source_row_id ASC")

		case StrategyConservativeMin:
			orderClauses = append(orderClauses, fmt.Sprintf("%s ASC NULLS LAST", f.SourceColumn))
			orderClauses = append(orderClauses, "as_of DESC", "source_row_id ASC")

		case StrategyConservativeMax:
			orderClauses = append(orderClauses, fmt.Sprintf("%s DESC NULLS LAST", f.SourceColumn))
			orderClauses = append(orderClauses, "as_of DESC", "source_row_id ASC")

		case StrategyMostFrequent:
			freqExpr := fmt.Sprintf("COUNT(*) OVER (PARTITION BY entity_key, %s)", f.SourceColumn)
			rankingCols = append(rankingCols, fmt.Sprintf("%s AS freq_%s", freqExpr, f.Attribute))
			orderClauses = append(orderClauses,
				fmt.Sprintf("COUNT(*) OVER (PARTITION BY entity_key, %s) DESC", f.SourceColumn),
				fmt.Sprintf("%s::text ASC", f.SourceColumn),
				"as_of DESC",
				"source_row_id ASC",
			)

		default:
			orderClauses = append(orderClauses, "as_of DESC", "source_row_id ASC")
		}

		rankExpr := fmt.Sprintf("ROW_NUMBER() OVER (PARTITION BY entity_key ORDER BY %s) AS %s",
			strings.Join(orderClauses, ", "),
			rankAlias,
		)
		rankingCols = append(rankingCols, rankExpr)
	}

	// 3. Build aggregation winner expressions
	var winnerCols []string
	winnerCols = append(winnerCols, "entity_key")

	for _, f := range plan.Fields {
		rankAlias := fmt.Sprintf("rank_%s", f.Attribute)
		if f.MaxStaleSeconds != nil && *f.MaxStaleSeconds > 0 {
			isFreshAlias := fmt.Sprintf("is_fresh_%s", f.Attribute)
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 AND %s THEN %s END) AS %s",
				rankAlias, isFreshAlias, f.SourceColumn, f.TargetColumn,
			))
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 THEN CASE WHEN %s THEN source_cd ELSE 'ALL_SOURCES_STALE' END END) AS winner_source_%s",
				rankAlias, isFreshAlias, f.Attribute,
			))
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 AND %s THEN as_of END) AS winner_as_of_%s",
				rankAlias, isFreshAlias, f.Attribute,
			))
		} else {
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 THEN %s END) AS %s",
				rankAlias, f.SourceColumn, f.TargetColumn,
			))
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 THEN source_cd END) AS winner_source_%s",
				rankAlias, f.Attribute,
			))
			winnerCols = append(winnerCols, fmt.Sprintf(
				"MAX(CASE WHEN %s = 1 THEN as_of END) AS winner_as_of_%s",
				rankAlias, f.Attribute,
			))
		}
	}

	sqlBuilder := strings.Builder{}
	sqlBuilder.WriteString("WITH staging_rows AS (\n")
	sqlBuilder.WriteString(fmt.Sprintf("    SELECT %s\n", strings.Join(stagingCols, ", ")))
	sqlBuilder.WriteString(fmt.Sprintf("    FROM %s s\n", plan.StagingTable))
	sqlBuilder.WriteString("    WHERE s.tenant_id = $1 AND s.as_of <= $2::timestamptz\n")
	sqlBuilder.WriteString("),\n")
	sqlBuilder.WriteString("ranked_contributions AS (\n")
	sqlBuilder.WriteString("    SELECT *,\n")
	sqlBuilder.WriteString(fmt.Sprintf("        %s\n", strings.Join(rankingCols, ",\n        ")))
	sqlBuilder.WriteString("    FROM staging_rows\n")
	sqlBuilder.WriteString("),\n")
	sqlBuilder.WriteString("golden_winners AS (\n")
	sqlBuilder.WriteString(fmt.Sprintf("    SELECT\n        %s\n", strings.Join(winnerCols, ",\n        ")))
	sqlBuilder.WriteString("    FROM ranked_contributions\n")
	sqlBuilder.WriteString("    GROUP BY entity_key\n")
	sqlBuilder.WriteString(")\n")
	sqlBuilder.WriteString("SELECT * FROM golden_winners ORDER BY entity_key ASC;")

	return &RenderedQuery{
		SQL:  sqlBuilder.String(),
		Args: args,
	}, nil
}

// RenderUpsertSQL generates the complete PostgreSQL CTE query with target table UPSERT
func (g *SQLGenerator) RenderUpsertSQL(plan *SurvivorshipBatchPlan, targetKey string) (*RenderedQuery, error) {
	rendered, err := g.RenderSQL(plan)
	if err != nil {
		return nil, err
	}
	if targetKey == "" {
		targetKey = plan.EntityKeyField
	}

	var (
		insertTargetCols []string
		selectWinnerCols []string
		updateSetCols    []string
	)

	insertTargetCols = append(insertTargetCols, "tenant_id", targetKey)
	selectWinnerCols = append(selectWinnerCols, "$1", "gw.entity_key")

	for _, f := range plan.Fields {
		if strings.EqualFold(f.TargetColumn, targetKey) || strings.EqualFold(f.TargetColumn, "tenant_id") {
			continue
		}
		insertTargetCols = append(insertTargetCols, f.TargetColumn)
		selectWinnerCols = append(selectWinnerCols, fmt.Sprintf("gw.%s", f.TargetColumn))
		updateSetCols = append(updateSetCols, fmt.Sprintf("%s = EXCLUDED.%s", f.TargetColumn, f.TargetColumn))
	}

	ctePrefix := strings.TrimSuffix(rendered.SQL, "SELECT * FROM golden_winners ORDER BY entity_key ASC;")

	upsertSQL := fmt.Sprintf(`%sINSERT INTO %s (%s)
SELECT %s
FROM golden_winners gw
ON CONFLICT (tenant_id, %s) DO UPDATE
SET %s
RETURNING %s;`,
		ctePrefix,
		plan.TargetTable,
		strings.Join(insertTargetCols, ", "),
		strings.Join(selectWinnerCols, ", "),
		targetKey,
		strings.Join(updateSetCols, ", "),
		targetKey,
	)

	return &RenderedQuery{
		SQL:  upsertSQL,
		Args: rendered.Args,
	}, nil
}

