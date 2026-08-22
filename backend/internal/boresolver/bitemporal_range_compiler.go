package boresolver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExecutionTierStrategy defines the resolved execution plan for bitemporal queries.
type ExecutionTierStrategy string

const (
	StrategyPureColdIceberg ExecutionTierStrategy = "PURE_COLD_ICEBERG"
	StrategyPureHotOLAP     ExecutionTierStrategy = "PURE_HOT_OLAP"
	StrategySplitAndStitch  ExecutionTierStrategy = "SPLIT_AND_STITCH"
)

// BitemporalRangeRequest defines parameters for a bitemporal range query compilation.
type BitemporalRangeRequest struct {
	TenantID           uuid.UUID `json:"tenant_id"`
	BusinessObjectID   uuid.UUID `json:"business_object_id"`
	EffectiveStartDate time.Time `json:"effective_start_date"`
	EffectiveEndDate   time.Time `json:"effective_end_date"`
	KnowledgeDate      time.Time `json:"knowledge_date"`
	WatermarkDate      time.Time `json:"watermark_date"`
	HotTableName       string    `json:"hot_table_name"`
	ColdTableName      string    `json:"cold_table_name"`
	TemporalColumn     string    `json:"temporal_column"`
	BusinessKeyColumns []string  `json:"business_key_columns"` // Entity identity for deduplication
	SelectedColumns    []string  `json:"selected_columns"`
	Dimensions         []string  `json:"dimensions"`
	Measures           []string  `json:"measures"`
}

// DateRange encapsulates a temporal range boundary.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// BitemporalCompilationResult contains the compilation strategy and generated SQL.
type BitemporalCompilationResult struct {
	Strategy         ExecutionTierStrategy `json:"strategy"`
	CompiledSQL      string                `json:"compiled_sql"`
	ColdRangeApplied *DateRange            `json:"cold_range_applied,omitempty"`
	HotRangeApplied  *DateRange            `json:"hot_range_applied,omitempty"`
}

// BitemporalRangeCompiler compiles tier-aware bitemporal queries spanning the hot/cold seam.
type BitemporalRangeCompiler struct{}

// NewBitemporalRangeCompiler creates a new compiler instance.
func NewBitemporalRangeCompiler() *BitemporalRangeCompiler {
	return &BitemporalRangeCompiler{}
}

// CompileRangeQuery generates tier-aware bitemporal SQL with PK coalescence across the hot/cold seam.
func (c *BitemporalRangeCompiler) CompileRangeQuery(
	ctx context.Context,
	req BitemporalRangeRequest,
) (*BitemporalCompilationResult, error) {
	if req.TenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id must be a valid UUID")
	}
	if req.EffectiveStartDate.After(req.EffectiveEndDate) {
		return nil, fmt.Errorf("invalid range: effective_start_date cannot be after effective_end_date")
	}
	if req.KnowledgeDate.IsZero() {
		req.KnowledgeDate = time.Now().UTC()
	}

	dateCol := req.TemporalColumn
	if dateCol == "" {
		dateCol = "effective_date"
	}

	pkCols := "id"
	if len(req.BusinessKeyColumns) > 0 {
		pkCols = strings.Join(req.BusinessKeyColumns, ", ")
	}

	cols := "*"
	if len(req.SelectedColumns) > 0 {
		cols = strings.Join(req.SelectedColumns, ", ")
	}

	kDateStr := req.KnowledgeDate.Format(time.RFC3339)
	startStr := req.EffectiveStartDate.Format("2006-01-02")
	endStr := req.EffectiveEndDate.Format("2006-01-02")
	wtStr := req.WatermarkDate.Format("2006-01-02")

	// Determine Tier Strategy based on mathematical interval decomposition:
	// R_cold = [Te_start, min(Te_end, Wt))
	// R_hot  = [max(Te_start, Wt), Te_end]
	switch {
	case req.EffectiveEndDate.Before(req.WatermarkDate):
		// Pure Cold (Iceberg)
		sql := fmt.Sprintf(`SELECT %s 
FROM %s
WHERE tenant_id = '%s'
  AND is_deleted = FALSE
  AND %s >= '%s' AND %s <= '%s'
  AND system_valid_from <= '%s'
  AND (system_valid_to > '%s' OR system_valid_to IS NULL)`,
			cols, req.ColdTableName, req.TenantID,
			dateCol, startStr, dateCol, endStr,
			kDateStr, kDateStr)

		return &BitemporalCompilationResult{
			Strategy:    StrategyPureColdIceberg,
			CompiledSQL: strings.TrimSpace(sql),
			ColdRangeApplied: &DateRange{
				Start: req.EffectiveStartDate,
				End:   req.EffectiveEndDate,
			},
		}, nil

	case !req.EffectiveStartDate.Before(req.WatermarkDate):
		// Pure Hot (StarRocks / OLAP)
		sql := fmt.Sprintf(`SELECT %s 
FROM %s
WHERE tenant_id = '%s'
  AND is_deleted = FALSE
  AND %s >= '%s' AND %s <= '%s'
  AND system_valid_from <= '%s'
  AND (system_valid_to > '%s' OR system_valid_to IS NULL)`,
			cols, req.HotTableName, req.TenantID,
			dateCol, startStr, dateCol, endStr,
			kDateStr, kDateStr)

		return &BitemporalCompilationResult{
			Strategy:    StrategyPureHotOLAP,
			CompiledSQL: strings.TrimSpace(sql),
			HotRangeApplied: &DateRange{
				Start: req.EffectiveStartDate,
				End:   req.EffectiveEndDate,
			},
		}, nil

	default:
		// Straddling Seam: Deduplicated Split & Stitch with Late-Mutation Recovery
		// Precedence: Hot (1) > Cold (2), with latest system_valid_from winning
		sql := fmt.Sprintf(`WITH bitemporal_cold_iceberg AS (
    SELECT %s, 2 AS source_precedence
    FROM %s
    WHERE tenant_id = '%s'
      AND is_deleted = FALSE
      AND %s >= '%s' AND %s < '%s'
      AND system_valid_from <= '%s'
      AND (system_valid_to > '%s' OR system_valid_to IS NULL)
),
bitemporal_hot_starrocks AS (
    SELECT %s, 1 AS source_precedence
    FROM %s
    WHERE tenant_id = '%s'
      AND is_deleted = FALSE
      AND (
          (%s >= '%s' AND %s <= '%s')
          OR (%s < '%s' AND system_valid_from >= '%s')
      )
      AND system_valid_from <= '%s'
      AND (system_valid_to > '%s' OR system_valid_to IS NULL)
),
bitemporal_raw_seam AS (
    SELECT * FROM bitemporal_cold_iceberg
    UNION ALL
    SELECT * FROM bitemporal_hot_starrocks
),
bitemporal_unified_seam AS (
    SELECT %s
    FROM (
        SELECT *,
               ROW_NUMBER() OVER (
                   PARTITION BY %s, %s 
                   ORDER BY system_valid_from DESC, source_precedence ASC
               ) AS __row_rank
        FROM bitemporal_raw_seam
    ) __deduped
    WHERE __row_rank = 1
)
SELECT %s FROM bitemporal_unified_seam`,
			cols, req.ColdTableName, req.TenantID,
			dateCol, startStr, dateCol, wtStr,
			kDateStr, kDateStr,
			cols, req.HotTableName, req.TenantID,
			dateCol, wtStr, dateCol, endStr,
			dateCol, wtStr, wtStr,
			kDateStr, kDateStr,
			cols, pkCols, dateCol, cols)

		return &BitemporalCompilationResult{
			Strategy:    StrategySplitAndStitch,
			CompiledSQL: strings.TrimSpace(sql),
			ColdRangeApplied: &DateRange{
				Start: req.EffectiveStartDate,
				End:   req.WatermarkDate,
			},
			HotRangeApplied: &DateRange{
				Start: req.WatermarkDate,
				End:   req.EffectiveEndDate,
			},
		}, nil
	}
}
