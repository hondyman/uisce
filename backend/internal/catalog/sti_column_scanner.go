package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type STIColumnScanner struct{}

func NewSTIColumnScanner() *STIColumnScanner {
	return &STIColumnScanner{}
}

func (s *STIColumnScanner) ScanAndEmit(ctx context.Context, tenantConn *sql.Conn, metaDB *sql.DB, tenantID uuid.UUID) error {
	query := `
		SELECT table_schema, table_name, column_name, data_type, is_nullable, ordinal_position
		FROM information_schema.columns
		WHERE table_schema IN ('oms', 'altinv', 'cash_flow', 'master')
		ORDER BY table_schema, table_name, ordinal_position;
	`
	rows, err := tenantConn.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query tenant information_schema: %w", err)
	}
	defer rows.Close()

	tx, err := metaDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin meta transaction: %w", err)
	}
	defer tx.Rollback()

	type tableKey struct {
		schema string
		table  string
	}
	tableNodeIDs := make(map[tableKey]uuid.UUID)

	for rows.Next() {
		var schema, table, colName, dataType, isNullable string
		var ordinal int
		if err := rows.Scan(&schema, &table, &colName, &dataType, &isNullable, &ordinal); err != nil {
			return fmt.Errorf("failed scanning column row: %w", err)
		}

		key := tableKey{schema: schema, table: table}
		tableNodeID, found := tableNodeIDs[key]
		if !found {
			tableNodeID = uuid.New()
			tableNodeIDs[key] = tableNodeID
			tablePath := fmt.Sprintf("%s.%s", schema, table)
			_, err = tx.ExecContext(ctx, `
				INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path)
				VALUES ($1, $2, 'TABLE', $3, $4, $5)
				ON CONFLICT (tenant_id, qualified_path) DO UPDATE
				SET updated_at = NOW()
			`, tableNodeID, tenantID, table, table, tablePath)
			if err != nil {
				return fmt.Errorf("failed upserting table node %s: %w", tablePath, err)
			}
		}

		qualifiedColPath := fmt.Sprintf("%s.%s/%s", schema, table, colName)
		colNodeID := uuid.New()

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path, properties)
			VALUES ($1, $2, 'ATTRIBUTE', $3, $4, $5, $6)
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE
			SET properties = EXCLUDED.properties, updated_at = NOW()
		`, colNodeID, tenantID, colName, colName, qualifiedColPath, fmt.Sprintf(`{"schema": "%s", "table": "%s", "data_type": "%s", "nullable": "%s"}`, schema, table, dataType, isNullable))
		if err != nil {
			return fmt.Errorf("failed upserting column node: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
			SELECT $1, $2, $3, 'COLUMN_OF'
			WHERE NOT EXISTS (
				SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'COLUMN_OF'
			)
		`, tenantID, colNodeID, tableNodeID)
		if err != nil {
			return fmt.Errorf("failed linking column edge: %w", err)
		}
	}

	return tx.Commit()
}
