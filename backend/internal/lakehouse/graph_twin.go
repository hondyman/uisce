package lakehouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GraphTwinCloner clones semantic mappings and manages shadow topology in catalog_edge.
type GraphTwinCloner interface {
	CloneSemanticMappings(ctx context.Context, tenantID, oltpTableNodeID, icebergTableNodeID uuid.UUID) error
}

type sqlGraphTwinCloner struct {
	db *sql.DB
}

func NewGraphTwinCloner(db *sql.DB) GraphTwinCloner {
	return &sqlGraphTwinCloner{db: db}
}

// CloneSemanticMappings performs the topological mirroring of MAPS_TO edges and draws SHADOWS edge.
func (c *sqlGraphTwinCloner) CloneSemanticMappings(ctx context.Context, tenantID, oltpTableNodeID, icebergTableNodeID uuid.UUID) error {
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Cardinal Rule 7: Enforce strict tenant isolation on both table nodes
	var oltpCount, icebergCount int
	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM catalog_node WHERE id = $1 AND tenant_id = $2 AND is_active = true`,
		oltpTableNodeID, tenantID,
	).Scan(&oltpCount)
	if err != nil || oltpCount == 0 {
		return fmt.Errorf("oltp table node %s unauthorized or not found for tenant %s", oltpTableNodeID, tenantID)
	}

	err = tx.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM catalog_node WHERE id = $1 AND tenant_id = $2 AND is_active = true`,
		icebergTableNodeID, tenantID,
	).Scan(&icebergCount)
	if err != nil || icebergCount == 0 {
		return fmt.Errorf("iceberg table node %s unauthorized or not found for tenant %s", icebergTableNodeID, tenantID)
	}

	// 2. Discover MAPS_TO and SHADOWS edge type IDs dynamically (Rule 1 & Rule 2: Config-Before-Code / Graph-First)
	var mapsToEdgeTypeID, shadowsEdgeTypeID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM catalog_edge_types WHERE edge_type_name = 'MAPS_TO' LIMIT 1`,
	).Scan(&mapsToEdgeTypeID)
	if err != nil {
		return fmt.Errorf("failed to resolve MAPS_TO edge type: %w", err)
	}

	err = tx.QueryRowContext(ctx,
		`SELECT id FROM catalog_edge_types WHERE edge_type_name = 'SHADOWS' LIMIT 1`,
	).Scan(&shadowsEdgeTypeID)
	if err != nil {
		return fmt.Errorf("failed to resolve SHADOWS edge type: %w", err)
	}

	// 3. Create or update SHADOWS edge: [Iceberg Table Node] ──► [OLTP Table Node]
	shadowProps, _ := json.Marshal(map[string]interface{}{
		"created_by": "lakehouse_graph_twin",
		"tier":       "COLD_LAKEHOUSE",
		"synced_at":  time.Now().UTC().Format(time.RFC3339),
	})

	_, err = tx.ExecContext(ctx, `
		INSERT INTO catalog_edge (
			id, source_node_id, target_node_id, edge_type_id, relationship_type, 
			tenant_id, properties, is_active, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, 'SHADOWS', 
			$4, $5, true, NOW(), NOW()
		)
		ON CONFLICT (source_node_id, target_node_id, edge_type_id, tenant_id) 
		DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW()
	`, icebergTableNodeID, oltpTableNodeID, shadowsEdgeTypeID, tenantID, shadowProps)
	if err != nil {
		return fmt.Errorf("failed to upsert SHADOWS relationship edge: %w", err)
	}

	// 4. Query source OLTP columns and their mapped semantic terms
	query := `
		SELECT 
			c_oltp.node_name AS column_name,
			e.target_node_id AS semantic_term_id,
			e.properties     AS edge_properties
		FROM catalog_node c_oltp
		JOIN catalog_edge e 
		  ON e.source_node_id = c_oltp.id 
		 AND e.edge_type_id = $1 
		 AND e.tenant_id = $2 
		 AND e.is_active = true
		WHERE c_oltp.parent_id = $3 
		  AND c_oltp.tenant_id = $2 
		  AND c_oltp.is_active = true
	`
	rows, err := tx.QueryContext(ctx, query, mapsToEdgeTypeID, tenantID, oltpTableNodeID)
	if err != nil {
		return fmt.Errorf("failed to query OLTP column semantic mappings: %w", err)
	}
	defer rows.Close()

	type semanticMapping struct {
		columnName     string
		semanticTermID uuid.UUID
		edgeProperties []byte
	}
	var mappings []semanticMapping

	for rows.Next() {
		var m semanticMapping
		if err := rows.Scan(&m.columnName, &m.semanticTermID, &m.edgeProperties); err != nil {
			return fmt.Errorf("failed to scan semantic mapping row: %w", err)
		}
		mappings = append(mappings, m)
	}

	// 5. Query matching child columns under the Iceberg shadow table
	icebergColRows, err := tx.QueryContext(ctx,
		`SELECT id, node_name FROM catalog_node WHERE parent_id = $1 AND tenant_id = $2 AND is_active = true`,
		icebergTableNodeID, tenantID,
	)
	if err != nil {
		return fmt.Errorf("failed to query Iceberg child columns: %w", err)
	}
	defer icebergColRows.Close()

	icebergCols := make(map[string]uuid.UUID)
	for icebergColRows.Next() {
		var colID uuid.UUID
		var colName string
		if err := icebergColRows.Scan(&colID, &colName); err != nil {
			return fmt.Errorf("failed to scan Iceberg child column: %w", err)
		}
		icebergCols[colName] = colID
	}

	// 6. Idempotently insert cloned MAPS_TO edges from Iceberg columns to Semantic Terms
	for _, m := range mappings {
		icebergColID, exists := icebergCols[m.columnName]
		if !exists {
			continue // Skip columns not mirrored in the Iceberg projection
		}

		var propsMap map[string]interface{}
		if len(m.edgeProperties) > 0 {
			_ = json.Unmarshal(m.edgeProperties, &propsMap)
		} else {
			propsMap = make(map[string]interface{})
		}
		propsMap["shadow_cloned"] = true
		propsMap["source_tier"] = "ICEBERG_COLD"
		clonedProps, _ := json.Marshal(propsMap)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (
				id, source_node_id, target_node_id, edge_type_id, relationship_type, 
				tenant_id, properties, is_active, created_at, updated_at
			) VALUES (
				gen_random_uuid(), $1, $2, $3, 'MAPS_TO', 
				$4, $5, true, NOW(), NOW()
			)
			ON CONFLICT (source_node_id, target_node_id, edge_type_id, tenant_id) 
			DO UPDATE SET properties = EXCLUDED.properties, updated_at = NOW()
		`, icebergColID, m.semanticTermID, mapsToEdgeTypeID, tenantID, clonedProps)
		if err != nil {
			return fmt.Errorf("failed to clone MAPS_TO edge for column %s: %w", m.columnName, err)
		}
	}

	return tx.Commit()
}
