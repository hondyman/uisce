package reporting

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CellExplainService provides cell-level lineage resolution and AI grounding context.
type CellExplainService struct{}

func NewCellExplainService() *CellExplainService {
	return &CellExplainService{}
}

// ResolveCell simulates/resolves multi-backend execution lineage for a requested semantic cell.
func (s *CellExplainService) ResolveCell(
	ctx context.Context,
	tenantID uuid.UUID,
	termKey string,
	contextType ExecutionContextType,
	asOfDate time.Time,
	parameters map[string]interface{},
) (*CellLineagePassport, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if asOfDate.IsZero() {
		asOfDate = time.Now().UTC()
	}

	watermark := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	resolverKey := fmt.Sprintf("resolver_%s_%s", termKey, stringsToLower(string(contextType)))

	var compiledSQL string
	var classification string
	var resolvedVal interface{}
	var formattedVal string
	var driftBps float64
	var partitions []string

	switch termKey {
	case "portfolio_nav":
		classification = "Wealth & Prime > Performance > NAV"
		resolvedVal = 1250450.00
		formattedVal = "$1,250,450.00"
		driftBps = 0.0
		partitions = []string{"starrocks_hot_2026_q3", "iceberg_archive_2025_q4"}
		compiledSQL = fmt.Sprintf("SELECT SUM(market_value_base) FROM starrocks.portfolio_positions_realtime WHERE tenant_id = '%s' AND as_of_date >= '%s'", tenantID, watermark.Format("2006-01-02"))

	case "net_fund_yield":
		classification = "Wealth & Prime > Performance > Yield"
		resolvedVal = 0.0485
		formattedVal = "4.85%"
		driftBps = 0.25
		partitions = []string{"starrocks_hot_2026_q3"}
		compiledSQL = fmt.Sprintf("SELECT AVG(net_yield) FROM starrocks.fund_yield_daily WHERE tenant_id = '%s' AND as_of_date >= '%s'", tenantID, watermark.Format("2006-01-02"))

	default:
		classification = "Enterprise > Reporting > Metric"
		resolvedVal = 100.0
		formattedVal = "100.00"
		driftBps = 0.0
		partitions = []string{"starrocks_hot_current"}
		compiledSQL = fmt.Sprintf("SELECT metric_val FROM starrocks.generic_metrics WHERE tenant_id = '%s' AND term_key = '%s'", tenantID, termKey)
	}

	// Generate deterministic Merkle state hash
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:%s:%v:%s", tenantID, termKey, contextType, resolvedVal, asOfDate.Format(time.RFC3339))))
	stateHash := hex.EncodeToString(hasher.Sum(nil))

	return &CellLineagePassport{
		CellID:              fmt.Sprintf("cell_%s_%s", termKey, uuid.New().String()[:8]),
		TermKey:             termKey,
		TermDisplayName:     humanizeTerm(termKey),
		ClassificationL3:    classification,
		ResolvedValue:       resolvedVal,
		FormattedValue:      formattedVal,
		ContextType:         contextType,
		ResolverKey:         resolverKey,
		CompiledSQL:         compiledSQL,
		SourcePartitions:    partitions,
		HistoricalWatermark: watermark,
		ReconciliationDrift: driftBps,
		IsReconciled:        driftBps < 0.50,
		StateSHA256:         stateHash,
		EvaluatedAt:         time.Now().UTC(),
	}, nil
}

func stringsToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func humanizeTerm(k string) string {
	switch k {
	case "portfolio_nav":
		return "Portfolio Net Asset Value"
	case "net_fund_yield":
		return "Net Fund Yield"
	default:
		return k
	}
}
