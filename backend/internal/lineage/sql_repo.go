package lineage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// DBLineageRepository implements LineageRepository using recursive SQL queries
type DBLineageRepository struct {
	db *sqlx.DB
}

// NewDBLineageRepository creates a new DBLineageRepository
func NewDBLineageRepository(db *sqlx.DB) *DBLineageRepository {
	return &DBLineageRepository{db: db}
}

// UpsertNode inserts or updates a lineage node in the public catalog
func (r *DBLineageRepository) UpsertNode(ctx context.Context, node LineageNode) error {
	// Map 'bo' type to its UUID if possible, but keep it flexible
	// We use catalog_node_type to resolve the string type to its UUID
	query := `
		INSERT INTO public.catalog_node (
			id, tenant_id, datasource_id, node_name, node_type_id, 
			properties, updated_at
		)
		VALUES (
			:id, COALESCE(:tenant_id, 'default'), 'lineage', :name, 
			(SELECT id FROM catalog_node_type WHERE catalog_type_name = :type LIMIT 1),
			:metadata, now()
		)
		ON CONFLICT (id) DO UPDATE SET
			tenant_id = EXCLUDED.tenant_id,
			node_name = EXCLUDED.node_name,
			node_type_id = EXCLUDED.node_type_id,
			properties = EXCLUDED.properties,
			updated_at = now()
	`
	_, err := r.db.NamedExecContext(ctx, query, node)
	return err
}

// UpsertEdge inserts or updates a lineage edge in the public catalog
func (r *DBLineageRepository) UpsertEdge(ctx context.Context, edge LineageEdge) error {
	// Map edge_type_name to its UUID if possible, but store the name directly in edge_type_name
	query := `
		INSERT INTO public.catalog_edge (
			tenant_id, tenant_datasource_id, source_node_id, target_node_id, 
			edge_type_name, edge_type_id, properties, created_at
		)
		VALUES (
			COALESCE(:tenant_id, '00000000-0000-0000-0000-000000000000'), '00000000-0000-0000-0000-000000000000', :from_id, :to_id,
			:type, (SELECT id FROM catalog_edge_type WHERE edge_type_name = :type LIMIT 1),
			:metadata, now()
		)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_name, target_node_id) 
		DO UPDATE SET
			properties = EXCLUDED.properties
	`
	_, err := r.db.NamedExecContext(ctx, query, edge)
	return err
}

// DeleteNode removes a node from the public catalog (cascading deletes for relationships handled at DB level)
func (r *DBLineageRepository) DeleteNode(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM public.catalog_node WHERE id = $1", id)
	return err
}

// DeleteEdge removes a specific relationship from the public catalog
func (r *DBLineageRepository) DeleteEdge(ctx context.Context, fromID, toID, edgeType string) error {
	query := `
		DELETE FROM public.catalog_edge 
		WHERE source_node_id = $1 AND target_node_id = $2 
		AND edge_type_name = $3
	`
	_, err := r.db.ExecContext(ctx, query, fromID, toID, edgeType)
	return err
}

// SyncDatasource is a no-op for SQL repo (data already in DB)
func (r *DBLineageRepository) SyncDatasource(ctx context.Context, datasourceID string) error {
	return nil
}

// FindDownstreamGraph finds downstream dependencies using recursive CTE on catalog tables
func (r *DBLineageRepository) FindDownstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	// Fetch nodes using catalog_node and catalog_edge
	query := `
		WITH RECURSIVE deps AS (
			-- Base case: the root node itself (only if it exists)
			SELECT cn.id, 0 AS depth
			FROM catalog_node cn
			WHERE cn.id = $1::uuid
			
			UNION
			
			-- Recursive step
			SELECT 
				ce.target_node_id AS id, 
				d.depth + 1
			FROM catalog_edge ce
			JOIN deps d ON ce.source_node_id = d.id
			WHERE d.depth < $2
		)
		SELECT DISTINCT cn.id, cnt.catalog_type_name as node_type, 'prod' as env, cn.tenant_id, cn.node_name, 
		       COALESCE(cn.properties, '{}'::jsonb)::text as metadata, cn.qualified_path
		FROM deps
		JOIN catalog_node cn ON cn.id = deps.id
		LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
	`

	rows, err := r.db.QueryContext(ctx, query, rootID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to query downstream graph nodes: %w", err)
	}
	defer rows.Close()

	var nodes []LineageNode
	nodeIDs := make(map[string]bool)
	for rows.Next() {
		var n LineageNode
		var qPath sql.NullString
		var metaStr string
		if err := rows.Scan(&n.ID, &n.Type, &n.Env, &n.TenantID, &n.Name, &metaStr, &qPath); err != nil {
			return nil, fmt.Errorf("scan downstream node failed: %w", err)
		}
		n.Metadata = []byte(metaStr)
		metaMap := make(map[string]interface{})
		_ = json.Unmarshal(n.Metadata, &metaMap)

		if qPath.Valid && qPath.String != "" {
			metaMap["qualified_path"] = qPath.String
		}
		metaMap["direction"] = "downstream"
		n.Metadata, _ = json.Marshal(metaMap)

		nodes = append(nodes, n)
		nodeIDs[n.ID] = true
	}

	// Fetch downstream edges where BOTH source and target are in our node set
	edgeQuery := `
		SELECT ce.source_node_id, ce.target_node_id, 
		       COALESCE(ce.edge_type_name, ce.edge_type_id::text, 'related_to') as type, 
		       'prod' as env, ce.tenant_id, 
		       COALESCE(ce.properties, '{}'::jsonb)::text as metadata
		FROM catalog_edge ce
		WHERE ce.source_node_id IN (
			WITH RECURSIVE deps AS (
				SELECT cn.id, 0 AS depth
				FROM catalog_node cn
				WHERE cn.id = $1::uuid
				UNION
				SELECT ce2.target_node_id AS id, d.depth + 1
				FROM catalog_edge ce2
				JOIN deps d ON ce2.source_node_id = d.id
				WHERE d.depth < $2
			)
			SELECT id FROM deps
		)
		AND ce.target_node_id IN (
			WITH RECURSIVE deps AS (
				SELECT cn.id, 0 AS depth
				FROM catalog_node cn
				WHERE cn.id = $1::uuid
				UNION
				SELECT ce2.target_node_id AS id, d.depth + 1
				FROM catalog_edge ce2
				JOIN deps d ON ce2.source_node_id = d.id
				WHERE d.depth < $2
			)
			SELECT id FROM deps
		)
	`
	var edges []LineageEdge

	edgeRows, err := r.db.QueryContext(ctx, edgeQuery, rootID, depth)
	if err != nil {
		return &Graph{Nodes: nodes, Edges: []LineageEdge{}}, nil
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var e LineageEdge
		var metaStr string
		if err := edgeRows.Scan(&e.FromID, &e.ToID, &e.Type, &e.Env, &e.TenantID, &metaStr); err != nil {
			return nil, fmt.Errorf("scan downstream edge failed: %w", err)
		}
		e.Metadata = []byte(metaStr)
		metaMap := make(map[string]interface{})
		_ = json.Unmarshal(e.Metadata, &metaMap)
		metaMap["direction"] = "downstream"
		e.Metadata, _ = json.Marshal(metaMap)
		edges = append(edges, e)
	}

	return &Graph{Nodes: nodes, Edges: edges}, nil
}

// FindUpstreamGraph finds upstream dependencies using recursive CTE on catalog tables
func (r *DBLineageRepository) FindUpstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	// Fetch upstream nodes
	query := `
		WITH RECURSIVE deps AS (
			-- Base case: the root node itself (only if it exists)
			SELECT cn.id, 0 AS depth
			FROM catalog_node cn
			WHERE cn.id = $1::uuid
			
			UNION
			
			-- Recursive step
			SELECT 
				ce.source_node_id AS id, 
				d.depth + 1
			FROM catalog_edge ce
			JOIN deps d ON ce.target_node_id = d.id
			WHERE d.depth < $2
		)
		SELECT DISTINCT cn.id, cnt.catalog_type_name as node_type, 'prod' as env, cn.tenant_id, cn.node_name, 
		       COALESCE(cn.properties, '{}'::jsonb)::text as metadata, cn.qualified_path
		FROM deps
		JOIN catalog_node cn ON cn.id = deps.id
		LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
	`

	rows, err := r.db.QueryContext(ctx, query, rootID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to query upstream graph nodes: %w", err)
	}
	defer rows.Close()

	var nodes []LineageNode
	for rows.Next() {
		var n LineageNode
		var qPath sql.NullString
		var metaStr string
		if err := rows.Scan(&n.ID, &n.Type, &n.Env, &n.TenantID, &n.Name, &metaStr, &qPath); err != nil {
			return nil, fmt.Errorf("scan upstream node failed: %w", err)
		}
		n.Metadata = []byte(metaStr)
		metaMap := make(map[string]interface{})
		_ = json.Unmarshal(n.Metadata, &metaMap)

		if qPath.Valid && qPath.String != "" {
			metaMap["qualified_path"] = qPath.String
		}
		metaMap["direction"] = "upstream"
		n.Metadata, _ = json.Marshal(metaMap)
		nodes = append(nodes, n)
	}

	// Fetch upstream edges where BOTH source and target are in our node set
	edgeQuery := `
		SELECT ce.source_node_id, ce.target_node_id, 
		       COALESCE(ce.edge_type_name, ce.edge_type_id::text, 'related_to') as type, 
		       'prod' as env, ce.tenant_id, 
		       COALESCE(ce.properties, '{}'::jsonb)::text as metadata
		FROM catalog_edge ce
		WHERE ce.source_node_id IN (
			WITH RECURSIVE deps AS (
				SELECT cn.id, 0 AS depth
				FROM catalog_node cn
				WHERE cn.id = $1::uuid
				UNION
				SELECT ce2.source_node_id AS id, d.depth + 1
				FROM catalog_edge ce2
				JOIN deps d ON ce2.target_node_id = d.id
				WHERE d.depth < $2
			)
			SELECT id FROM deps
		)
		AND ce.target_node_id IN (
			WITH RECURSIVE deps AS (
				SELECT cn.id, 0 AS depth
				FROM catalog_node cn
				WHERE cn.id = $1::uuid
				UNION
				SELECT ce2.source_node_id AS id, d.depth + 1
				FROM catalog_edge ce2
				JOIN deps d ON ce2.target_node_id = d.id
				WHERE d.depth < $2
			)
			SELECT id FROM deps
		)
	`
	var edges []LineageEdge

	edgeRows, err := r.db.QueryContext(ctx, edgeQuery, rootID, depth)
	if err != nil {
		return &Graph{Nodes: nodes, Edges: []LineageEdge{}}, nil
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var e LineageEdge
		var metaStr string
		if err := edgeRows.Scan(&e.FromID, &e.ToID, &e.Type, &e.Env, &e.TenantID, &metaStr); err != nil {
			return nil, fmt.Errorf("scan upstream edge failed: %w", err)
		}
		e.Metadata = []byte(metaStr)
		metaMap := make(map[string]interface{})
		_ = json.Unmarshal(e.Metadata, &metaMap)
		metaMap["direction"] = "upstream"
		e.Metadata, _ = json.Marshal(metaMap)
		edges = append(edges, e)
	}

	return &Graph{Nodes: nodes, Edges: edges}, nil
}

// FindBiDirectionalGraph finds both upstream and downstream dependencies
// Adds direction metadata to distinguish upstream/downstream nodes
func (r *DBLineageRepository) FindBiDirectionalGraph(ctx context.Context, rootID string, depth int) (*Graph, error) {
	upstream, err := r.FindUpstreamGraph(ctx, rootID, depth)
	if err != nil {
		return nil, err
	}
	downstream, err := r.FindDownstreamGraph(ctx, rootID, depth)
	if err != nil {
		return upstream, nil // Return what we have
	}

	// Add direction metadata to upstream nodes
	for i := range upstream.Nodes {
		var metaMap map[string]interface{}
		if len(upstream.Nodes[i].Metadata) > 0 {
			_ = json.Unmarshal(upstream.Nodes[i].Metadata, &metaMap)
		}
		if metaMap == nil {
			metaMap = make(map[string]interface{})
		}
		metaMap["direction"] = "upstream"
		metaMap["is_lineage"] = true
		upstream.Nodes[i].Metadata, _ = json.Marshal(metaMap)
	}

	// Add direction metadata to downstream nodes
	for i := range downstream.Nodes {
		var metaMap map[string]interface{}
		if len(downstream.Nodes[i].Metadata) > 0 {
			_ = json.Unmarshal(downstream.Nodes[i].Metadata, &metaMap)
		}
		if metaMap == nil {
			metaMap = make(map[string]interface{})
		}
		metaMap["direction"] = "downstream"
		metaMap["is_impact"] = true
		downstream.Nodes[i].Metadata, _ = json.Marshal(metaMap)
	}

	// Add direction metadata to edges
	for i := range upstream.Edges {
		var metaMap map[string]interface{}
		if len(upstream.Edges[i].Metadata) > 0 {
			_ = json.Unmarshal(upstream.Edges[i].Metadata, &metaMap)
		}
		if metaMap == nil {
			metaMap = make(map[string]interface{})
		}
		metaMap["direction"] = "upstream"
		upstream.Edges[i].Metadata, _ = json.Marshal(metaMap)
	}

	for i := range downstream.Edges {
		var metaMap map[string]interface{}
		if len(downstream.Edges[i].Metadata) > 0 {
			_ = json.Unmarshal(downstream.Edges[i].Metadata, &metaMap)
		}
		if metaMap == nil {
			metaMap = make(map[string]interface{})
		}
		metaMap["direction"] = "downstream"
		downstream.Edges[i].Metadata, _ = json.Marshal(metaMap)
	}

	// Merge graphs and deduplicate nodes
	nodeMap := make(map[string]LineageNode)
	for _, n := range upstream.Nodes {
		nodeMap[n.ID] = n
	}
	for _, n := range downstream.Nodes {
		if existing, found := nodeMap[n.ID]; found {
			// Node appears in both - mark as bidirectional
			var metaMap map[string]interface{}
			if len(existing.Metadata) > 0 {
				_ = json.Unmarshal(existing.Metadata, &metaMap)
			}
			if metaMap == nil {
				metaMap = make(map[string]interface{})
			}
			metaMap["direction"] = "both"
			metaMap["is_lineage"] = true
			metaMap["is_impact"] = true
			existing.Metadata, _ = json.Marshal(metaMap)
			nodeMap[n.ID] = existing
		} else {
			nodeMap[n.ID] = n
		}
	}

	// Convert map back to slice
	nodes := make([]LineageNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	// Deduplicate edges
	edgeMap := make(map[string]LineageEdge)
	for _, e := range upstream.Edges {
		key := e.FromID + "->" + e.ToID
		edgeMap[key] = e
	}
	for _, e := range downstream.Edges {
		key := e.FromID + "->" + e.ToID
		if existing, found := edgeMap[key]; found {
			// Edge appears in both directions - mark as bidirectional
			var metaMap map[string]interface{}
			if len(existing.Metadata) > 0 {
				_ = json.Unmarshal(existing.Metadata, &metaMap)
			}
			if metaMap == nil {
				metaMap = make(map[string]interface{})
			}
			metaMap["direction"] = "both"
			existing.Metadata, _ = json.Marshal(metaMap)
			edgeMap[key] = existing
		} else {
			edgeMap[key] = e
		}
	}

	edges := make([]LineageEdge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, e)
	}

	return &Graph{Nodes: nodes, Edges: edges}, nil
}

// FindGraphByDatasource finds nodes and edges associated with a specific datasource
func (r *DBLineageRepository) FindGraphByDatasource(ctx context.Context, datasourceID string) (*Graph, error) {
	// Stub implementation - SQL repo is legacy/backup
	return &Graph{Nodes: []LineageNode{}, Edges: []LineageEdge{}}, nil
}

// Helper types for sqlx scanning if needed (sqlx handles structs well if tags match)
