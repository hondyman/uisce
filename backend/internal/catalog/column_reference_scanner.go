package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ColumnReferenceScanner struct {
	sourceDB *sql.DB
	schemas  []string
	dbPrefix string
}

func NewColumnReferenceScanner(sourceDB *sql.DB, schemas []string, dbPrefix string) *ColumnReferenceScanner {
	return &ColumnReferenceScanner{
		sourceDB: sourceDB,
		schemas:  schemas,
		dbPrefix: dbPrefix,
	}
}

type ScanResult struct {
	TablesScanned    int `json:"tables_scanned"`
	ColumnsScanned  int `json:"columns_scanned"`
	EdgesCreated    int `json:"edges_created"`
	FKsFound        int `json:"fks_found"`
	SkippedNonWhitelisted int `json:"skipped_non_whitelisted"`
}

func (s *ColumnReferenceScanner) ScanAndEmit(ctx context.Context, metaDB *sql.DB, tenantID uuid.UUID, datasourceID uuid.UUID) (*ScanResult, error) {
	result := &ScanResult{}

	tx, err := metaDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin meta transaction: %w", err)
	}
	defer tx.Rollback()

	if err := s.upsertTableAndColumnNodes(ctx, tx, tenantID, datasourceID, result); err != nil {
		return nil, fmt.Errorf("phase A (nodes): %w", err)
	}

	if err := s.upsertColumnReferenceEdges(ctx, tx, tenantID, result); err != nil {
		return nil, fmt.Errorf("phase B (edges): %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return result, nil
}

func (s *ColumnReferenceScanner) upsertTableAndColumnNodes(ctx context.Context, tx *sql.Tx, tenantID, datasourceID uuid.UUID, result *ScanResult) error {
	schemaList := fmt.Sprintf("('%s')", strings.Join(s.schemas, "','"))

	query := fmt.Sprintf(`
		SELECT 
			'%s.' || table_schema || '.' || table_name AS qualified_path,
			table_schema,
			table_name,
			column_name,
			data_type,
			is_nullable,
			ordinal_position
		FROM information_schema.columns
		WHERE table_schema IN %s
		ORDER BY table_schema, table_name, ordinal_position;
	`, s.dbPrefix, schemaList)

	rows, err := s.sourceDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query columns: %w", err)
	}
	defer rows.Close()

	type tableKey struct {
		schema string
		table  string
	}
	tableNodeIDs := make(map[tableKey]uuid.UUID)
	tablesSeen := make(map[tableKey]bool)

	for rows.Next() {
		var qPath, schema, table, colName, dataType, isNullable string
		var ordinal int
		if err := rows.Scan(&qPath, &schema, &table, &colName, &dataType, &isNullable, &ordinal); err != nil {
			return fmt.Errorf("scan column row: %w", err)
		}

		key := tableKey{schema: schema, table: table}
		if !tablesSeen[key] {
			tablesSeen[key] = true
			result.TablesScanned++
		}
		result.ColumnsScanned++

		tableNodeID, found := tableNodeIDs[key]
		if !found {
			tableNodeID = uuid.New()
			tableNodeIDs[key] = tableNodeID
			_, err = tx.ExecContext(ctx, `
				INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path)
				VALUES ($1, $2, 'TABLE', $3, $4, $5)
				ON CONFLICT (tenant_id, qualified_path) DO UPDATE
				SET updated_at = NOW()
			`, tableNodeID, tenantID, table, table, qPath)
			if err != nil {
				return fmt.Errorf("upsert table node %s: %w", qPath, err)
			}
		}

		qualifiedColPath := fmt.Sprintf("%s/%s", qPath, colName)
		colNodeID := uuid.New()

		props := map[string]string{
			"schema":    schema,
			"table":     table,
			"data_type": dataType,
			"nullable":  isNullable,
		}
		propsJSON, _ := json.Marshal(props)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path, properties)
			VALUES ($1, $2, 'ATTRIBUTE', $3, $4, $5, $6)
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE
			SET properties = EXCLUDED.properties, updated_at = NOW()
		`, colNodeID, tenantID, colName, colName, qualifiedColPath, string(propsJSON))
		if err != nil {
			return fmt.Errorf("upsert column node %s: %w", qualifiedColPath, err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
			SELECT $1, $2, $3, 'COLUMN_OF'
			WHERE NOT EXISTS (
				SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'COLUMN_OF'
			)
		`, tenantID, colNodeID, tableNodeID)
		if err != nil {
			return fmt.Errorf("link column edge for %s: %w", qualifiedColPath, err)
		}
	}

	return rows.Err()
}

func (s *ColumnReferenceScanner) upsertColumnReferenceEdges(ctx context.Context, tx *sql.Tx, tenantID uuid.UUID, result *ScanResult) error {
	schemaList := fmt.Sprintf("('%s')", strings.Join(s.schemas, "','"))

	query := fmt.Sprintf(`
		SELECT
			con.conname AS constraint_name,
			sa.attname AS source_column,
			ta.attname AS target_column,
			'%s.' || sn.nspname || '.' || sc.relname || '/' || sa.attname AS source_qpath,
			'%s.' || tn.nspname || '.' || tc.relname || '/' || ta.attname AS target_qpath,
			CASE con.confdeltype WHEN 'a' THEN 'NO ACTION' WHEN 'r' THEN 'RESTRICT' WHEN 'c' THEN 'CASCADE'
								WHEN 'n' THEN 'SET NULL' WHEN 'd' THEN 'SET DEFAULT' END AS on_delete
		FROM pg_catalog.pg_constraint con
		JOIN pg_catalog.pg_namespace sn ON sn.oid = con.connamespace
		JOIN pg_catalog.pg_class sc ON sc.oid = con.conrelid
		JOIN pg_catalog.pg_namespace tn ON tn.oid = tc.relnamespace
		JOIN pg_catalog.pg_class tc ON tc.oid = con.confrelid
		CROSS JOIN LATERAL unnest(con.conkey, con.confkey) WITH ORDINALITY AS k(src_attnum, tgt_attnum, ord)
		JOIN pg_catalog.pg_attribute sa ON sa.attrelid = con.conrelid AND sa.attnum = k.src_attnum
		JOIN pg_catalog.pg_attribute ta ON ta.attrelid = con.confrelid AND ta.attnum = k.tgt_attnum
		WHERE con.contype = 'f'
		  AND sn.nspname IN %s
		ORDER BY sn.nspname, sc.relname, sa.attnum;
	`, s.dbPrefix, s.dbPrefix, schemaList)

	rows, err := s.sourceDB.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("query FKs: %w", err)
	}
	defer rows.Close()

	edgeTypeID := uuid.MustParse("c0c1c2c3-d4e5-f6a7-b8c9-d0e1f2a3b4c5")

	for rows.Next() {
		var constraintName, sourceCol, targetCol, sourceQPath, targetQPath, onDelete string
		if err := rows.Scan(&constraintName, &sourceCol, &targetCol, &sourceQPath, &targetQPath, &onDelete); err != nil {
			return fmt.Errorf("scan FK row: %w", err)
		}

		result.FKsFound++

		var sourceNodeID, targetNodeID uuid.UUID
		err := tx.QueryRowContext(ctx, `
			SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND qualified_path = $2
		`, tenantID, sourceQPath).Scan(&sourceNodeID)
		if err == sql.ErrNoRows {
			result.SkippedNonWhitelisted++
			continue
		} else if err != nil {
			return fmt.Errorf("lookup source node %s: %w", sourceQPath, err)
		}

		err = tx.QueryRowContext(ctx, `
			SELECT node_id FROM catalog_node WHERE tenant_id = $1 AND qualified_path = $2
		`, tenantID, targetQPath).Scan(&targetNodeID)
		if err == sql.ErrNoRows {
			result.SkippedNonWhitelisted++
			continue
		} else if err != nil {
			return fmt.Errorf("lookup target node %s: %w", targetQPath, err)
		}

		edgeID := uuid.New()
		props := map[string]interface{}{
			"reference_kind":  "FK_UUID",
			"constraint_name":  constraintName,
			"is_enforced":     true,
			"cardinality":     "MANY_TO_ONE",
			"on_delete":       onDelete,
		}
		propsJSON, _ := json.Marshal(props)

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (id, tenant_id, source_node_id, target_node_id, edge_type_id, properties, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), NOW())
			WHERE NOT EXISTS (
				SELECT 1 FROM catalog_edge
				WHERE tenant_id = $2 AND source_node_id = $3 AND target_node_id = $4 AND edge_type_id = $5
			)
		`, edgeID, tenantID, sourceNodeID, targetNodeID, edgeTypeID, string(propsJSON))
		if err != nil {
			return fmt.Errorf("insert edge %s -> %s: %w", sourceQPath, targetQPath, err)
		}

		result.EdgesCreated++
	}

	return rows.Err()
}
