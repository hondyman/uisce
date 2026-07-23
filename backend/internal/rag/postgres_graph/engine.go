package postgres_graph

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresGraphEngine handles graph operations using SQL
type PostgresGraphEngine struct {
	db *sqlx.DB
}

func NewPostgresGraphEngine(db *sqlx.DB) *PostgresGraphEngine {
	return &PostgresGraphEngine{db: db}
}

type UBOResult struct {
	OwnerName          string      `db:"owner_name"`
	EffectiveOwnership float64     `db:"effective_ownership"`
	Depth              int         `db:"depth"`
	Path               []uuid.UUID `db:"path"`
}

// GetUBO calculates Ultimate Beneficial Ownership using Recursive CTE
func (e *PostgresGraphEngine) GetUBO(ctx context.Context, targetEntityID uuid.UUID) ([]UBOResult, error) {
	query := `
	WITH RECURSIVE ownership_chain AS (
		-- Anchor Member: Identify direct owners
		SELECT 
			r.source_entity_id,
			r.target_entity_id,
			r.percentage_ownership AS direct_ownership,
			r.percentage_ownership AS effective_ownership,
			1 AS depth,
			ARRAY[r.target_entity_id] AS path,
			FALSE AS is_cycle
		FROM ownership_relationships r
		WHERE r.target_entity_id = $1

		UNION ALL

		-- Recursive Member: Traverse upwards
		SELECT 
			r.source_entity_id,
			r.target_entity_id,
			r.percentage_ownership,
			(oc.effective_ownership * r.percentage_ownership) AS effective_ownership,
			oc.depth + 1,
			path || r.target_entity_id,
			r.target_entity_id = ANY(path)
		FROM ownership_relationships r
		JOIN ownership_chain oc ON r.target_entity_id = oc.source_entity_id
		WHERE oc.depth < 20 AND NOT is_cycle
	)
	SELECT 
		e.name AS owner_name,
		oc.effective_ownership,
		oc.depth,
		oc.path
	FROM ownership_chain oc
	JOIN financial_entities e ON oc.source_entity_id = e.entity_id
	WHERE oc.effective_ownership > 0.25 -- Significant control threshold
	ORDER BY oc.effective_ownership DESC;
	`

	var results []UBOResult
	// We need to handle the array scanning manually or use sqlx with pq.Array if supported directly for UUIDs,
	// but standard sqlx struct scan might struggle with UUID arrays without a custom scanner.
	// For simplicity in this step, we'll assume standard scanning works or we'd add a wrapper.
	// Note: pq.Array is often needed for array types.

	rows, err := e.db.QueryContext(ctx, query, targetEntityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r UBOResult
		var path []string // Scan as string array first
		if err := rows.Scan(&r.OwnerName, &r.EffectiveOwnership, &r.Depth, pq.Array(&path)); err != nil {
			return nil, err
		}
		// Convert path strings to UUIDs
		for _, p := range path {
			uid, _ := uuid.Parse(p)
			r.Path = append(r.Path, uid)
		}
		results = append(results, r)
	}

	return results, nil
}

// HybridSearch performs RRF combining vector and text search
func (e *PostgresGraphEngine) HybridSearch(ctx context.Context, query string, embedding []float32) ([]uuid.UUID, error) {
	// Placeholder for RRF implementation
	// In a real implementation, this would execute the complex SQL query defined in the roadmap
	return []uuid.UUID{}, nil
}
