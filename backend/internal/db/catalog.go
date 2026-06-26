package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
)

// GetDatasourceIDForNode retrieves the tenant_datasource_id for a given node ID.
func GetDatasourceIDForNode(ctx context.Context, db *sqlx.DB, nodeID string) (string, error) {
	var datasourceID string
	query := `SELECT tenant_datasource_id FROM public.catalog_node WHERE id = $1`
	err := db.GetContext(ctx, &datasourceID, query, nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("node with ID %s not found", nodeID)
		}
		return "", fmt.Errorf("failed to get datasource ID for node %s: %w", nodeID, err)
	}
	return datasourceID, nil
}

// GetSchemaHash retrieves the hash of the current schema for a datasource using GraphQL.
func GetSchemaHash(db *sqlx.DB, datasourceID uuid.UUID) (string, error) {
	var hash sql.NullString
	query := `
        SELECT md5(string_agg(qualified_path, '' ORDER BY qualified_path))
        FROM public.catalog_node
        WHERE tenant_datasource_id = $1`
	err := db.Get(&hash, query, datasourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get schema hash: %w", err)
	}
	if !hash.Valid {
		return "", nil // Return empty string if no nodes exist
	}
	return hash.String, nil
}

// GoldCopyNodeInfo holds ID and properties for a node in the gold copy
type GoldCopyNodeInfo struct {
	ID         uuid.UUID       `db:"id"`
	Properties json.RawMessage `db:"properties"`
}

// GetCatalogNodeMapForGoldCopy retrieves all nodes for a gold copy datasource using GraphQL.
func GetCatalogNodeMapForGoldCopy(db *sqlx.DB, goldCopyDatasourceID uuid.UUID) (map[string]GoldCopyNodeInfo, error) {
	nodes := []struct {
		ID            uuid.UUID       `db:"id"`
		NodeTypeID    uuid.UUID       `db:"node_type_id"`
		QualifiedPath string          `db:"qualified_path"`
		Properties    json.RawMessage `db:"properties"`
	}{}
	query := `SELECT id, node_type_id, qualified_path, properties FROM public.catalog_node WHERE tenant_datasource_id = $1 AND core_id IS NULL`
	err := db.Select(&nodes, query, goldCopyDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get catalog nodes for gold copy datasource %s: %w", goldCopyDatasourceID, err)
	}

	nodeMap := make(map[string]GoldCopyNodeInfo, len(nodes))
	for _, node := range nodes {
		key := fmt.Sprintf("%s:%s", node.NodeTypeID.String(), node.QualifiedPath)
		nodeMap[key] = GoldCopyNodeInfo{
			ID:         node.ID,
			Properties: node.Properties,
		}
	}
	return nodeMap, nil
}

// CleanupTempTables ensures temp tables are created and clean before starting
func CleanupTempTables(tx *sqlx.Tx, datasourceID uuid.UUID) error {
	// Create temp tables after dropping them to ensure fresh schema (session-local).
	_, err := tx.Exec(`
		DROP TABLE IF EXISTS temp_catalog_node;
		DROP TABLE IF EXISTS temp_catalog_edge;

		CREATE TEMP TABLE temp_catalog_node (
			id uuid,
			core_id uuid,
			tenant_id uuid,
			tenant_datasource_id uuid,
			node_type_id uuid,
			node_name text,
			qualified_path text,
			parent_id uuid,
			properties jsonb,
			description text,
			is_alpha boolean,
			created_at timestamptz DEFAULT NOW(),
			updated_at timestamptz DEFAULT NOW()
		) ON COMMIT DROP;

		CREATE TEMP TABLE temp_catalog_edge (
			id uuid,
			core_id uuid,
			tenant_id uuid,
			tenant_datasource_id uuid,
			source_node_id uuid,
			target_node_id uuid,
			edge_type_id uuid,
			edge_type text,
			properties jsonb,
			created_at timestamptz DEFAULT NOW(),
			updated_at timestamptz DEFAULT NOW()
		) ON COMMIT DROP;
	`)
	if err != nil {
		return fmt.Errorf("failed to create temp tables: %w", err)
	}

	// Just in case they already existed from a previous operation in the same transaction (unlikely with ON COMMIT DROP but safe),
	// or if we switch to ON COMMIT PRESERVE ROWS later.
	if _, err := tx.Exec(`TRUNCATE TABLE temp_catalog_edge`); err != nil {
		return fmt.Errorf("failed to cleanup temp edges: %w", err)
	}

	if _, err := tx.Exec(`TRUNCATE TABLE temp_catalog_node`); err != nil {
		return fmt.Errorf("failed to cleanup temp nodes: %w", err)
	}

	// Create indexes to speed up merges
	if _, err := tx.Exec(`
		CREATE INDEX IF NOT EXISTS idx_temp_node_lookup ON temp_catalog_node (tenant_datasource_id, node_type_id, qualified_path);
		CREATE INDEX IF NOT EXISTS idx_temp_node_parent ON temp_catalog_node (parent_id);
		CREATE INDEX IF NOT EXISTS idx_temp_edge_lookup ON temp_catalog_edge (tenant_datasource_id, edge_type_id);
	`); err != nil {
		return fmt.Errorf("failed to create temp table indexes: %w", err)
	}

	log.Printf("Initialized temp tables and indexes for datasource %s", datasourceID)
	return nil
}

// InsertTempCatalogNodes inserts a batch of nodes into the temporary table.
// Batches insertions to avoid PostgreSQL's 65535 parameter limit.
func InsertTempCatalogNodes(ctx context.Context, tx *sqlx.Tx, nodes []*models.CatalogNode, progress chan<- models.ScanProgress) error {
	if len(nodes) == 0 {
		return nil
	}

	// Each node has 11 parameters, so max batch size = 65535 / 11 ≈ 5957
	// Use 1000 to be safe and avoid long locks
	const batchSize = 1000

	query := `
		INSERT INTO temp_catalog_node (id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, description, is_alpha)
		VALUES (:id, :core_id, :tenant_id, :tenant_datasource_id, :node_type_id, :node_name, :qualified_path, :parent_id, :properties, :description, :is_alpha)`

	totalBatches := (len(nodes) + batchSize - 1) / batchSize

	for i := 0; i < len(nodes); i += batchSize {
		end := i + batchSize
		if end > len(nodes) {
			end = len(nodes)
		}

		batchIndex := i / batchSize
		batch := nodes[i:end] // Define batch before using it in progress message

		if progress != nil {
			percent := 10.0 + (float64(batchIndex)/float64(totalBatches))*40.0 // Map to 10-50% range
			progress <- models.ScanProgress{
				Phase:   "storing",
				Percent: percent,
				Message: fmt.Sprintf("Storing batch %d/%d (%d nodes)...", batchIndex+1, totalBatches, len(batch)),
			}
		}
		log.Printf("Inserting node batch %d-%d of %d", i, end, len(nodes))
		if _, err := tx.NamedExecContext(ctx, query, batch); err != nil {
			return fmt.Errorf("failed to insert node batch %d-%d: %w", i, end, err)
		}
	}

	log.Printf("Successfully inserted %d nodes in batches", len(nodes))
	return nil
}

// Enhanced InsertTempCatalogEdges with detailed debugging
func InsertTempCatalogEdges(ctx context.Context, tx *sqlx.Tx, edges []models.CatalogEdge) error {
	if len(edges) == 0 {
		log.Printf("No edges to insert")
		return nil
	}

	log.Printf("Attempting to insert %d edges into temp table", len(edges))

	// Debug: Print all edges to see what we're trying to insert
	uniqueKeys := make(map[string][]int) // key -> list of edge indices with that key

	for i, edge := range edges {
		// Create the same key that the unique constraint uses
		key := fmt.Sprintf("datasource=%s|source=%s|type=%s|target=%s",
			edge.TenantDatasourceId.String(),
			edge.SourceNodeID.String(),
			edge.EdgeTypeID.String(),
			edge.TargetNodeID.String())

		uniqueKeys[key] = append(uniqueKeys[key], i)
	}

	// Find duplicates (logging omitted for brevity)

	// If no duplicates, proceed with insert
	query := `
		INSERT INTO temp_catalog_edge (id, core_id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, edge_type_id, edge_type, properties)
		VALUES (:id, :core_id, :tenant_id, :tenant_datasource_id, :source_node_id, :target_node_id, :edge_type_id, :edge_type, :properties)`

	result, err := tx.NamedExecContext(ctx, query, edges)
	if err != nil {
		log.Printf("Error inserting edges: %v", err)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("Successfully inserted %d edge rows", rowsAffected)

	return nil
}

func MergeCatalogData(tx *sqlx.Tx, datasourceID uuid.UUID) (int64, int64, int64, error) {
	// Set timeouts for the entire merge transaction
	if _, err := tx.Exec(`SET LOCAL lock_timeout = '30s'`); err != nil {
		log.Printf("Warning: Failed to set lock_timeout: %v", err)
	}
	if _, err := tx.Exec(`SET LOCAL statement_timeout = '60s'`); err != nil {
		log.Printf("Warning: Failed to set statement_timeout: %v", err)
	}

	log.Printf("Starting MergeCatalogData V3 (Optimized) for datasource: %s", datasourceID)

	var totalInserted, totalUpdated, totalDeleted int64

	// Step 1: MERGE nodes from temp table to permanent table (PostgreSQL 15+)
	log.Printf("Attempting to merge nodes using MERGE statement...")

	res, err := tx.Exec(`
		MERGE INTO public.catalog_node AS target
		USING (
			SELECT id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, description, is_alpha
			FROM temp_catalog_node
			WHERE tenant_datasource_id = $1
		) AS source
		ON target.tenant_datasource_id = source.tenant_datasource_id 
		   AND target.node_type_id = source.node_type_id 
		   AND target.qualified_path = source.qualified_path
		WHEN MATCHED AND target.properties IS DISTINCT FROM (target.properties || source.properties) THEN
			UPDATE SET
				core_id = COALESCE(source.core_id, target.core_id),
				tenant_id = source.tenant_id,
				node_name = source.node_name,
				description = COALESCE(source.description, target.description),
				parent_id = source.parent_id,
				properties = target.properties || source.properties,
				is_alpha = source.is_alpha,
				updated_at = NOW()
		WHEN NOT MATCHED THEN
			INSERT (id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, description, is_alpha, created_at, updated_at)
			VALUES (source.id, source.core_id, source.tenant_id, source.tenant_datasource_id, source.node_type_id, source.node_name, source.qualified_path, source.parent_id, source.properties, source.description, source.is_alpha, NOW(), NOW())
	`, datasourceID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			log.Printf("PostgreSQL Error Details: Code=%s, Msg=%s, Detail=%s", pgErr.Code, pgErr.Message, pgErr.Detail)
		}
		return 0, 0, 0, fmt.Errorf("merge catalog nodes: %w", err)
	}

	nodeRowsAffected, _ := res.RowsAffected()
	totalInserted = nodeRowsAffected // Use = here since it's the first step
	log.Printf("Node merge completed: %d rows affected.", nodeRowsAffected)

	// INCREMENTAL SCAN: No deletions
	// Steps 2-3 previously deleted stale edges/nodes. This has been removed
	// to preserve existing catalog data across scans. Only new items are added
	// and changed items are updated. Semantic terms and manual mappings persist.
	log.Printf("Incremental scan mode: Skipping deletions (preserving existing data)")

	// Step 4: MERGE edges from temp table to permanent table (PostgreSQL 15+)
	log.Printf("Attempting to merge edges using MERGE statement...")

	// Optimization: Instead of joining the entire catalog_node table, we join only for the source/target of the temp edges.
	res, err = tx.Exec(`
		WITH node_id_mapping AS (
			-- Only map nodes that are actually referenced by current edges to keep the join small
			SELECT DISTINCT tn.id as temp_id, n.id AS final_id
			FROM temp_catalog_node tn
			JOIN public.catalog_node n ON tn.tenant_datasource_id = n.tenant_datasource_id 
				AND tn.qualified_path = n.qualified_path 
				AND tn.node_type_id = n.node_type_id
			WHERE tn.tenant_datasource_id = $1
			  AND (
				tn.id IN (SELECT source_node_id FROM temp_catalog_edge) OR
				tn.id IN (SELECT target_node_id FROM temp_catalog_edge)
			  )
		)
		MERGE INTO public.catalog_edge AS target
		USING (
			SELECT
				te.id, te.core_id, te.tenant_id, te.tenant_datasource_id,
				source_map.final_id AS final_source_id,
				target_map.final_id AS final_target_id,
				te.edge_type_id, te.edge_type, te.properties
			FROM temp_catalog_edge te
			LEFT JOIN node_id_mapping source_map ON te.source_node_id = source_map.temp_id
			LEFT JOIN node_id_mapping target_map ON te.target_node_id = target_map.temp_id
			WHERE te.tenant_datasource_id = $1
		) AS source
		ON target.tenant_datasource_id = source.tenant_datasource_id 
		   AND target.source_node_id = source.final_source_id 
		   AND target.target_node_id = source.final_target_id 
		   AND target.edge_type = source.edge_type
		WHEN MATCHED THEN
			UPDATE SET
				properties = source.properties,
				core_id = source.core_id,
				updated_at = NOW()
		WHEN NOT MATCHED THEN
			INSERT (id, core_id, tenant_id, tenant_datasource_id, source_node_id, target_node_id, edge_type_id, edge_type, properties, created_at, updated_at)
			VALUES (source.id, source.core_id, source.tenant_id, source.tenant_datasource_id, source.final_source_id, source.final_target_id, source.edge_type_id, source.edge_type, source.properties, NOW(), NOW())
	`, datasourceID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			log.Printf("Foreign key violation during edge merge: %s", pgErr.Detail)
		}
		return 0, 0, 0, fmt.Errorf("merge catalog edges: %w", err)
	}
	edgeRowsAffected, _ := res.RowsAffected()
	totalInserted += edgeRowsAffected
	log.Printf("Edge merge completed: %d rows affected", edgeRowsAffected)

	// Step 4.5: Update table constraint summaries (Fixup, ignoring stats for now)
	if _, err := tx.Exec(`
		WITH constraint_summary AS (
			SELECT tbl.id as table_id,
				(SELECT COUNT(*) FROM public.catalog_node cols WHERE cols.parent_id = tbl.id AND (cols.properties->>'is_primary_key')::boolean) as pk_column_count,
				(SELECT COUNT(DISTINCT e.properties->>'primary_constraint_name') FROM public.catalog_edge e JOIN public.catalog_node c ON e.source_node_id = c.id WHERE c.parent_id = tbl.id AND e.properties->>'primary_constraint_name' IS NOT NULL) as fk_constraint_count
			FROM public.catalog_node tbl
			WHERE tbl.tenant_datasource_id = $1 AND tbl.node_type_id = '49a50271-ae58-4d3e-ae1c-2f5b89d89192'
		)
		UPDATE public.catalog_node n
		SET properties = n.properties || jsonb_build_object('primary_key_count', cs.pk_column_count, 'foreign_key_count', cs.fk_constraint_count)
		FROM constraint_summary cs
		WHERE n.id = cs.table_id
		AND ((n.properties->>'primary_key_count')::int IS DISTINCT FROM cs.pk_column_count OR (n.properties->>'foreign_key_count')::int IS DISTINCT FROM cs.fk_constraint_count)
	`, datasourceID); err != nil {
		return 0, 0, 0, fmt.Errorf("update table constraint summaries: %w", err)
	}

	// Step 5: Clean up temp tables (Dropping them is not needed due to ON COMMIT DROP, but we can truncate manually if we want to reuse session)
	// Actually, since we use ON COMMIT DROP, they will disappear after commit.
	// But explicit drop doesn't hurt if we want to release memory early.
	// For now, let's just leave them to ON COMMIT DROP logic.
	// Removed DELETE statements.

	log.Printf("MergeCatalogData completed: +%d / ~%d / -%d", totalInserted, totalUpdated, totalDeleted)
	return totalInserted, totalUpdated, totalDeleted, nil
}

// LinkAlphaNodes updates non-gold copy nodes to link to their gold copy counterparts.
func LinkAlphaNodes(ctx context.Context, db *sqlx.DB) (int64, error) {
	log.Print("Starting to link non-gold copy nodes to alpha (gold copy) nodes.")

	query := `
		UPDATE public.catalog_node
		SET 
			core_id = gold_nodes.id,
			is_alpha = true
		FROM (
			SELECT n.id, n.qualified_path, n.tenant_id
			FROM public.catalog_node n
			JOIN public.tenants t ON t.id = n.tenant_id
			WHERE t.gold_copy = true
		) gold_nodes
		JOIN public.tenants t1 ON t1.id = catalog_node.tenant_id
		WHERE catalog_node.qualified_path = gold_nodes.qualified_path
		  AND t1.gold_copy = false
		  AND catalog_node.core_id IS DISTINCT FROM gold_nodes.id;
	`

	result, err := db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to link alpha nodes: %w", err)
	}

	return result.RowsAffected()
}

// LinkAlphaNodesForTenant updates non-gold copy nodes for a single tenant to link to their
// gold copy counterparts matched on qualified_path (and node_type_id when provided).
// If nodeTypeID is nil, it applies to all node types; otherwise only the specified type.
func LinkAlphaNodesForTenant(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, nodeTypeID *uuid.UUID) (int64, error) {
	log.Printf("Linking alpha nodes for tenant %s (nodeType=%v)", tenantID, nodeTypeID)

	// Build parameterized SQL with optional node_type filter using a CTE to dedupe gold nodes
	// across potential multiple gold tenants.
	baseCTE := `WITH gold_dedup AS (
		SELECT DISTINCT ON (n.qualified_path, n.node_type_id)
			   n.id, n.qualified_path, n.node_type_id, n.tenant_id
		FROM public.catalog_node n
		JOIN public.tenants t ON t.id = n.tenant_id
		WHERE t.gold_copy = true
	`
	args := []interface{}{}
	if nodeTypeID != nil {
		baseCTE += `  AND n.node_type_id = $1\n`
		args = append(args, *nodeTypeID)
	}
	baseCTE += `        ORDER BY n.qualified_path, n.node_type_id, n.id
	)
	UPDATE public.catalog_node cn
	SET core_id = g.id,
		is_alpha = true
	FROM gold_dedup g
	JOIN public.tenants t1 ON t1.id = cn.tenant_id
	WHERE cn.tenant_id = $` // placeholder index to be filled

	// Determine parameter positions
	tenantParamIdx := 1
	if nodeTypeID != nil {
		tenantParamIdx = 2
	}

	// Complete WHERE clause with joining conditions and safeguards
	query := baseCTE + fmt.Sprintf(`%d
	  AND t1.gold_copy = false
	  AND cn.node_type_id = g.node_type_id
	  AND cn.qualified_path = g.qualified_path
	  AND cn.tenant_id <> g.tenant_id
	  AND cn.core_id IS DISTINCT FROM g.id;`, tenantParamIdx)

	args = append(args, tenantID)

	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to link alpha nodes for tenant: %w", err)
	}
	return result.RowsAffected()
}

// AlphaLinkPreview represents a pending alpha link for a node (dry-run only)
type AlphaLinkPreview struct {
	ID            uuid.UUID `db:"id" json:"id"`
	NodeTypeID    uuid.UUID `db:"node_type_id" json:"node_type_id"`
	QualifiedPath string    `db:"qualified_path" json:"qualified_path"`
	CandidateCore uuid.UUID `db:"candidate_core_id" json:"candidate_core_id"`
	GoldTenantID  uuid.UUID `db:"gold_tenant_id" json:"gold_tenant_id"`
}

// PreviewAlphaLinksForTenant lists which nodes would be linked for a tenant (dry-run).
// If nodeTypeID is provided, the preview is filtered to that type. Limit/offset are optional pagination controls.
func PreviewAlphaLinksForTenant(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, nodeTypeID *uuid.UUID, limit, offset int) ([]AlphaLinkPreview, error) {
	// Build the same gold_dedup CTE used by the UPDATE, then select matches instead of updating
	baseCTE := `WITH gold_dedup AS (
		SELECT DISTINCT ON (n.qualified_path, n.node_type_id)
			   n.id, n.qualified_path, n.node_type_id, n.tenant_id
		FROM public.catalog_node n
		JOIN public.tenants t ON t.id = n.tenant_id
		WHERE t.gold_copy = true
	`
	args := []interface{}{}
	if nodeTypeID != nil {
		baseCTE += `  AND n.node_type_id = $1\n`
		args = append(args, *nodeTypeID)
	}
	baseCTE += `        ORDER BY n.qualified_path, n.node_type_id, n.id
	)
	SELECT cn.id, cn.node_type_id, cn.qualified_path, g.id as candidate_core_id, g.tenant_id as gold_tenant_id
	FROM public.catalog_node cn
	JOIN gold_dedup g ON cn.node_type_id = g.node_type_id AND cn.qualified_path = g.qualified_path
	JOIN public.tenants t1 ON t1.id = cn.tenant_id
	WHERE cn.tenant_id = $`

	tenantParamIdx := 1
	if nodeTypeID != nil {
		tenantParamIdx = 2
	}
	query := baseCTE + fmt.Sprintf(`%d
	  AND t1.gold_copy = false
	  AND cn.tenant_id <> g.tenant_id
	  AND cn.core_id IS DISTINCT FROM g.id
	  ORDER BY cn.qualified_path, cn.node_type_id, cn.id
	  LIMIT $%d OFFSET $%d;`, tenantParamIdx, tenantParamIdx+1, tenantParamIdx+2)

	args = append(args, tenantID)
	// sensible defaults if zero/negative
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows := []AlphaLinkPreview{}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to preview alpha links: %w", err)
	}
	return rows, nil
}

// CountPreviewAlphaLinksForTenant returns only the count of rows that would be linked.
func CountPreviewAlphaLinksForTenant(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, nodeTypeID *uuid.UUID) (int64, error) {
	baseCTE := `WITH gold_dedup AS (
		SELECT DISTINCT ON (n.qualified_path, n.node_type_id)
			   n.id, n.qualified_path, n.node_type_id, n.tenant_id
		FROM public.catalog_node n
		JOIN public.tenants t ON t.id = n.tenant_id
		WHERE t.gold_copy = true
	`
	args := []interface{}{}
	if nodeTypeID != nil {
		baseCTE += `  AND n.node_type_id = $1\n`
		args = append(args, *nodeTypeID)
	}
	baseCTE += `        ORDER BY n.qualified_path, n.node_type_id, n.id
	)
	SELECT COUNT(*)
	FROM public.catalog_node cn
	JOIN gold_dedup g ON cn.node_type_id = g.node_type_id AND cn.qualified_path = g.qualified_path
	JOIN public.tenants t1 ON t1.id = cn.tenant_id
	WHERE cn.tenant_id = $`

	tenantParamIdx := 1
	if nodeTypeID != nil {
		tenantParamIdx = 2
	}
	query := baseCTE + fmt.Sprintf(`%d
	  AND t1.gold_copy = false
	  AND cn.tenant_id <> g.tenant_id
	  AND cn.core_id IS DISTINCT FROM g.id;`, tenantParamIdx)

	args = append(args, tenantID)
	var count int64
	if err := db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count alpha link preview: %w", err)
	}
	return count, nil
}

// UpsertBusinessTermsFromGold copies or updates business term nodes from gold tenants
// into the specified non-gold tenant (by tenantID) for a target tenant_datasource_id.
// Matching is done by qualified_path; node_type_id is fixed to Business Term node type.
// businessTermTypeID is the UUID of the Business Term node type.
func UpsertBusinessTermsFromGold(ctx context.Context, db *sqlx.DB, tenantID uuid.UUID, tenantDatasourceID uuid.UUID, businessTermTypeID uuid.UUID) (int64, error) {
	// Ensure target tenant is non-gold; operation is a no-op for gold tenants
	var isGold bool
	if err := db.GetContext(ctx, &isGold, `SELECT gold_copy FROM public.tenants WHERE id = $1`, tenantID); err != nil {
		return 0, fmt.Errorf("check tenant gold flag: %w", err)
	}
	if isGold {
		return 0, nil
	}

	// Use distinct-by natural key view of gold business terms, then upsert into target tenant/datasource.
	// We set core_id to the gold node id and mark is_alpha=true for traceability.
	query := `
		WITH gold_terms AS (
			SELECT DISTINCT ON (n.qualified_path)
				   n.id            AS gold_id,
				   n.qualified_path,
				   n.node_name,
				   n.description,
				   n.properties
			FROM public.catalog_node n
			JOIN public.tenants t ON t.id = n.tenant_id
			WHERE t.gold_copy = true
			  AND n.node_type_id = $1
			ORDER BY n.qualified_path, n.id
		),
		upserted AS (
			INSERT INTO public.catalog_node (
				id, core_id, tenant_id, tenant_datasource_id, node_type_id, node_name, qualified_path, parent_id, properties, description, is_alpha, created_at, updated_at
			)
			SELECT
				gen_random_uuid(),
				g.gold_id,
				$2::uuid,
				$3::uuid,
				$1::uuid,
				g.node_name,
				g.qualified_path,
				NULL,
				g.properties,
				g.description,
				true,
				NOW(), NOW()
			FROM gold_terms g
			ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO UPDATE SET
				core_id = EXCLUDED.core_id,
				node_name = EXCLUDED.node_name,
				description = EXCLUDED.description,
				properties = EXCLUDED.properties,
				is_alpha = true,
				updated_at = NOW()
			RETURNING 1
		)
		SELECT COUNT(*) FROM upserted;`

	var count int64
	if err := db.GetContext(ctx, &count, query, businessTermTypeID, tenantID, tenantDatasourceID); err != nil {
		return 0, fmt.Errorf("upsert business terms from gold: %w", err)
	}
	return count, nil
}
