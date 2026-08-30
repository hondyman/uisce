package semanticmatch

import (
	"context"
	"database/sql"
	"encoding/json"
)

// Mirrors your existing catalog_edge dedupe pattern. Use 'IS_CLASSIFIED_AS'
// instead of 'MAPS_TO_SEMANTIC_TERM' if you want to reuse the existing edge type.
const upsertEdgeSQL = `
INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type, properties)
SELECT $1, $2, $3, 'MAPS_TO_SEMANTIC_TERM', $4::jsonb
WHERE NOT EXISTS (
	SELECT 1 FROM catalog_edge e
	WHERE e.tenant_id = $1
	  AND e.source_node_id = $2
	  AND e.target_node_id = $3
	  AND e.edge_type = 'MAPS_TO_SEMANTIC_TERM'
)`

type EdgeProps struct {
	Confidence float64 `json:"confidence"`
	Method     string  `json:"method"`
	Status     string  `json:"status"`
	Reasoning  string  `json:"reasoning,omitempty"`
	Model      string  `json:"model,omitempty"`
	DecidedAt  string  `json:"decided_at,omitempty"`
}

// WriteEdges persists outcomes as catalog edges. resolve maps a column and a
// term to their catalog node IDs (implement against your catalog_node schema,
// creating ATTRIBUTE / SEMANTIC_TERM nodes on demand). Only auto-linked and
// review outcomes produce edges; review-status edges let your UI filter them.
func WriteEdges(ctx context.Context, db *sql.DB, tenantID, model string, outcomes []MatchOutcome,
	resolve func(ColumnMeta, *SemanticTerm) (srcNodeID, dstNodeID string, ok bool)) (int, error) {

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, upsertEdgeSQL)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	written := 0
	for _, o := range outcomes {
		if o.Term == nil || o.Status == StatusNoMatch {
			continue
		}
		src, dst, ok := resolve(o.Column, o.Term)
		if !ok {
			continue
		}
		props, _ := json.Marshal(EdgeProps{
			Confidence: o.Confidence, Method: o.Method, Status: o.Status,
			Reasoning: o.Reasoning, Model: model,
			DecidedAt: o.DecidedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
		if _, err := stmt.ExecContext(ctx, tenantID, src, dst, string(props)); err != nil {
			return written, err
		}
		written++
	}
	return written, tx.Commit()
}
