package reporting

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ExecutionContextType represents the institutional operating scope.
type ExecutionContextType string

const (
	ContextRegulatoryABOR  ExecutionContextType = "REGULATORY_ABOR"
	ContextManagementIBOR  ExecutionContextType = "MANAGEMENT_IBOR"
	ContextClientStatement ExecutionContextType = "CLIENT_STATEMENT"
	ContextTaxOptimized    ExecutionContextType = "TAX_OPTIMIZED"
)

// CellLineagePassport captures deterministic, cell-level explainability provenance.
type CellLineagePassport struct {
	CellID              string               `json:"cellId"`
	TermKey             string               `json:"termKey"`
	TermDisplayName     string               `json:"termDisplayName"`
	ClassificationL3    string               `json:"classificationL3"`
	ResolvedValue       interface{}          `json:"resolvedValue"`
	FormattedValue      string               `json:"formattedValue"`
	ContextType         ExecutionContextType `json:"contextType"`
	ResolverKey         string               `json:"resolverKey"`
	CompiledSQL         string               `json:"compiledSql"`
	SourcePartitions    []string             `json:"sourcePartitions"`
	HistoricalWatermark time.Time            `json:"historicalWatermark"`
	ReconciliationDrift float64              `json:"reconciliationDriftBps"`
	IsReconciled        bool                 `json:"isReconciled"`
	StateSHA256         string               `json:"stateSha256"`
	EvaluatedAt         time.Time            `json:"evaluatedAt"`
}

// SemanticTermResolver resolves a cell's lineage passport across execution contexts.
type SemanticTermResolver interface {
	ResolveCell(
		ctx context.Context,
		tenantID uuid.UUID,
		termKey string,
		contextType ExecutionContextType,
		asOfDate time.Time,
		parameters map[string]interface{},
	) (*CellLineagePassport, error)
}
