package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type SubtypeSemanticLinker struct{}

func NewSubtypeSemanticLinker() *SubtypeSemanticLinker {
	return &SubtypeSemanticLinker{}
}

func (l *SubtypeSemanticLinker) LinkTerms(ctx context.Context, db *sql.DB, tenantID uuid.UUID) error {
	query := `
		INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type, properties)
		SELECT
			st.tenant_id,
			st.node_id AS source_node_id,
			attr.node_id AS target_node_id,
			'IS_CLASSIFIED_AS' AS edge_type,
			'{"confidence": 1.0, "source": "exact_name_match"}'::jsonb
		FROM catalog_node st
		JOIN catalog_node attr
		  ON attr.tenant_id = st.tenant_id
		  AND attr.node_type = 'ATTRIBUTE'
		  AND (attr.node_key = st.node_key OR attr.node_name = st.node_name)
		WHERE st.tenant_id = $1
		  AND st.node_type = 'SEMANTIC_TERM'
		  AND NOT EXISTS (
			SELECT 1 FROM catalog_edge ce
			WHERE ce.tenant_id = st.tenant_id
			  AND ce.source_node_id = st.node_id
			  AND ce.target_node_id = attr.node_id
			  AND ce.edge_type = 'IS_CLASSIFIED_AS'
		  );
	`
	_, err := db.ExecContext(ctx, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed linking semantic terms to attributes: %w", err)
	}
	return nil
}
