package drift

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CandidateMatch struct {
	ColumnNodeID    uuid.UUID `db:"node_id"`
	ColumnName      string    `db:"node_name"`
	DataType        string    `db:"data_type"`
	ConfidenceScore float64   `db:"confidence_score"`
	Strategy        string    `db:"strategy"`
	Rationale       string    `db:"rationale"`
}

type DriftRemediationMatcher struct {
	db *sqlx.DB
}

func NewDriftRemediationMatcher(db *sqlx.DB) *DriftRemediationMatcher {
	return &DriftRemediationMatcher{db: db}
}

// FindCandidateMatches inspects active columns to find optimal replacement matches
func (m *DriftRemediationMatcher) FindCandidateMatches(
	ctx context.Context,
	tenantID, tableNodeID uuid.UUID,
	missingColumnName string,
) ([]CandidateMatch, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	var liveColumns []struct {
		NodeID   uuid.UUID `db:"node_id"`
		NodeName string    `db:"node_name"`
		DataType string    `db:"data_type"`
	}

	if m.db != nil {
		query := `
			SELECT node_id, node_name, COALESCE(properties->>'data_type', 'VARCHAR') AS data_type
			FROM public.catalog_node
			WHERE parent_node_id = $1 AND tenant_id = $2 AND node_type = 'COLUMN' AND is_active = TRUE;
		`
		err := m.db.SelectContext(ctx, &liveColumns, query, tableNodeID, tenantID)
		if err != nil {
			return nil, err
		}
	}

	matches := make([]CandidateMatch, 0)
	cleanMissing := strings.ToLower(strings.TrimSpace(missingColumnName))

	for _, col := range liveColumns {
		cleanCandidate := strings.ToLower(strings.TrimSpace(col.NodeName))
		if cleanCandidate == cleanMissing {
			continue
		}

		if isFinancialSynonym(cleanMissing, cleanCandidate) {
			matches = append(matches, CandidateMatch{
				ColumnNodeID:    col.NodeID,
				ColumnName:      col.NodeName,
				DataType:        col.DataType,
				ConfidenceScore: 0.96,
				Strategy:        "FINANCIAL_SYNONYM_DICTIONARY",
				Rationale:       fmt.Sprintf("Matched institutional financial synonym dictionary between '%s' and '%s'", missingColumnName, col.NodeName),
			})
			continue
		}

		if strings.Contains(cleanCandidate, cleanMissing) || strings.Contains(cleanMissing, cleanCandidate) {
			matches = append(matches, CandidateMatch{
				ColumnNodeID:    col.NodeID,
				ColumnName:      col.NodeName,
				DataType:        col.DataType,
				ConfidenceScore: 0.88,
				Strategy:        "SUBSTRING_SIMILARITY",
				Rationale:       fmt.Sprintf("Direct lexical substring containment between '%s' and '%s'", missingColumnName, col.NodeName),
			})
		}
	}

	return matches, nil
}

func isFinancialSynonym(a, b string) bool {
	synonyms := map[string][]string{
		"px_last":      {"last_price", "close_price", "market_price", "px_close", "price"},
		"isin":         {"id_isin", "isin_code", "security_isin"},
		"cusip":        {"id_cusip", "cusip_code"},
		"gross_amount": {"div_rate", "dividend_amount", "cash_rate", "amount"},
		"nav":          {"net_asset_value", "total_nav", "share_price"},
		"country":      {"country_code", "country_iso", "domicile_country"},
	}

	for k, list := range synonyms {
		if (a == k || contains(list, a)) && (b == k || contains(list, b)) {
			return true
		}
	}
	return false
}

func contains(arr []string, target string) bool {
	for _, s := range arr {
		if s == target {
			return true
		}
	}
	return false
}
