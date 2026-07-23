package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Writer defines the interface for writing to the catalog graph
type Writer interface {
	CreateNode(ctx context.Context, node CatalogNode) error
	CreateNodes(ctx context.Context, nodes []CatalogNode) error
	CreateEdge(ctx context.Context, edge CatalogEdge) error
	CreateEdges(ctx context.Context, edges []CatalogEdge) error
	UpdateNode(ctx context.Context, node CatalogNode) error
	GetNode(ctx context.Context, nodeID string) (*CatalogNode, error)
	GetEdges(ctx context.Context, fromNode string) ([]CatalogEdge, error)
}

type catalogWriter struct {
	db *sql.DB
}

// NewWriter creates a new CatalogWriter
func NewWriter(db *sql.DB) Writer {
	return &catalogWriter{db: db}
}

// CreateNode creates a single node, with upsert semantics
func (w *catalogWriter) CreateNode(ctx context.Context, n CatalogNode) error {
	props, err := json.Marshal(n.Properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}

	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	n.UpdatedAt = time.Now().UTC()

	// Validate node type exists
	if err := w.validateNodeType(ctx, n.NodeType); err != nil {
		return fmt.Errorf("invalid node_type: %w", err)
	}

	// Validate tenant_id is not empty
	if n.TenantID == "" {
		return fmt.Errorf("tenant_id cannot be empty")
	}

	_, err = w.db.ExecContext(ctx, `
		INSERT INTO catalog_node (id, node_type, qualified_path, properties, tenant_id, tenant_datasource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET properties = EXCLUDED.properties,
		    tenant_id = EXCLUDED.tenant_id,
		    tenant_datasource_id = EXCLUDED.tenant_datasource_id,
		    updated_at = EXCLUDED.updated_at
	`, n.ID, n.NodeType, n.QualifiedPath, string(props), n.TenantID, n.DatasourceID, n.CreatedAt, n.UpdatedAt)

	return err
}

// CreateNodes creates multiple nodes in a transaction (batch mode)
func (w *catalogWriter) CreateNodes(ctx context.Context, nodes []CatalogNode) error {
	if len(nodes) == 0 {
		return nil
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO catalog_node (id, node_type, qualified_path, properties, tenant_id, tenant_datasource_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE
		SET properties = EXCLUDED.properties,
		    tenant_id = EXCLUDED.tenant_id,
		    tenant_datasource_id = EXCLUDED.tenant_datasource_id,
		    updated_at = EXCLUDED.updated_at
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, n := range nodes {
		// Validate node type
		if err := w.validateNodeType(ctx, n.NodeType); err != nil {
			return fmt.Errorf("invalid node_type %s: %w", n.NodeType, err)
		}

		if n.TenantID == "" {
			return fmt.Errorf("tenant_id cannot be empty for node %s", n.ID)
		}

		props, err := json.Marshal(n.Properties)
		if err != nil {
			return fmt.Errorf("marshal properties: %w", err)
		}

		if n.CreatedAt.IsZero() {
			n.CreatedAt = time.Now().UTC()
		}
		n.UpdatedAt = time.Now().UTC()

		if _, err := stmt.ExecContext(ctx, n.ID, n.NodeType, n.QualifiedPath, string(props), n.TenantID, n.DatasourceID, n.CreatedAt, n.UpdatedAt); err != nil {
			return fmt.Errorf("exec stmt: %w", err)
		}
	}

	return tx.Commit()
}

// CreateEdge creates a single edge (idempotent)
func (w *catalogWriter) CreateEdge(ctx context.Context, e CatalogEdge) error {
	props, err := json.Marshal(e.Properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}

	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}

	// Validate edge type exists
	if err := w.validateEdgeType(ctx, e.EdgeType); err != nil {
		return fmt.Errorf("invalid edge_type: %w", err)
	}

	_, err = w.db.ExecContext(ctx, `
		INSERT INTO catalog_edge (id, edge_type, from_node, to_node, properties, tenant_id, tenant_datasource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`, e.ID, e.EdgeType, e.FromNode, e.ToNode, string(props), e.TenantID, e.DatasourceID, e.CreatedAt)

	return err
}

// CreateEdges creates multiple edges in a transaction (batch mode)
func (w *catalogWriter) CreateEdges(ctx context.Context, edges []CatalogEdge) error {
	if len(edges) == 0 {
		return nil
	}

	tx, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO catalog_edge (id, edge_type, from_node, to_node, properties, tenant_id, tenant_datasource_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, e := range edges {
		// Validate edge type
		if err := w.validateEdgeType(ctx, e.EdgeType); err != nil {
			return fmt.Errorf("invalid edge_type %s: %w", e.EdgeType, err)
		}

		props, err := json.Marshal(e.Properties)
		if err != nil {
			return fmt.Errorf("marshal properties: %w", err)
		}

		if e.CreatedAt.IsZero() {
			e.CreatedAt = time.Now().UTC()
		}

		if _, err := stmt.ExecContext(ctx, e.ID, e.EdgeType, e.FromNode, e.ToNode, string(props), e.TenantID, e.DatasourceID, e.CreatedAt); err != nil {
			return fmt.Errorf("exec stmt: %w", err)
		}
	}

	return tx.Commit()
}

// UpdateNode updates an existing node's properties
func (w *catalogWriter) UpdateNode(ctx context.Context, n CatalogNode) error {
	props, err := json.Marshal(n.Properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}

	n.UpdatedAt = time.Now().UTC()

	_, err = w.db.ExecContext(ctx, `
		UPDATE catalog_node
		SET properties = $1, updated_at = $2
		WHERE id = $3
	`, string(props), n.UpdatedAt, n.ID)

	return err
}

// GetNode retrieves a node by ID
func (w *catalogWriter) GetNode(ctx context.Context, nodeID string) (*CatalogNode, error) {
	var node CatalogNode
	var propsJSON string

	err := w.db.QueryRowContext(ctx, `
		SELECT id, node_type, qualified_path, properties, tenant_id, created_at, updated_at
		FROM catalog_node
		WHERE id = $1
	`, nodeID).Scan(&node.ID, &node.NodeType, &node.QualifiedPath, &propsJSON, &node.TenantID, &node.CreatedAt, &node.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal([]byte(propsJSON), &node.Properties); err != nil {
		return nil, fmt.Errorf("unmarshal properties: %w", err)
	}

	return &node, nil
}

// GetEdges retrieves all edges from a given node
func (w *catalogWriter) GetEdges(ctx context.Context, fromNode string) ([]CatalogEdge, error) {
	rows, err := w.db.QueryContext(ctx, `
		SELECT id, edge_type, from_node, to_node, properties, created_at
		FROM catalog_edge
		WHERE from_node = $1
	`, fromNode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var edges []CatalogEdge
	for rows.Next() {
		var edge CatalogEdge
		var propsJSON string

		if err := rows.Scan(&edge.ID, &edge.EdgeType, &edge.FromNode, &edge.ToNode, &propsJSON, &edge.CreatedAt); err != nil {
			return nil, err
		}

		if err := json.Unmarshal([]byte(propsJSON), &edge.Properties); err != nil {
			return nil, fmt.Errorf("unmarshal properties: %w", err)
		}

		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// validateNodeType checks if a node type exists in catalog_node_type
func (w *catalogWriter) validateNodeType(ctx context.Context, nodeType string) error {
	var exists bool
	err := w.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM catalog_node_type WHERE id = $1)`,
		nodeType,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("unknown node_type: %s", nodeType)
	}
	return nil
}

// validateEdgeType checks if an edge type exists in catalog_edge_type
func (w *catalogWriter) validateEdgeType(ctx context.Context, edgeType string) error {
	var exists bool
	err := w.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM catalog_edge_type WHERE id = $1)`,
		edgeType,
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("unknown edge_type: %s", edgeType)
	}
	return nil
}
