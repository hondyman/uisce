package catalog

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type SubtypeBOBuilder struct {
	loader SubtypeRegistryLoader
}

func NewSubtypeBOBuilder(loader SubtypeRegistryLoader) *SubtypeBOBuilder {
	return &SubtypeBOBuilder{loader: loader}
}

func (b *SubtypeBOBuilder) BuildForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) error {
	rows, err := b.loader.LoadAllForTenant(ctx, db, tenantID)
	if err != nil {
		return err
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, row := range rows {
		boNodeID := uuid.New()
		qualifiedPath := fmt.Sprintf("oms.%s/%s", row.RootObject, row.SubtypeCode)

		_, err := tx.ExecContext(ctx, `
			INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path, properties)
			VALUES ($1, $2, 'BUSINESS_OBJECT', $3, $4, $5, $6)
			ON CONFLICT (tenant_id, qualified_path) DO UPDATE
			SET node_name = EXCLUDED.node_name, properties = EXCLUDED.properties, updated_at = NOW()
		`, boNodeID, tenantID, row.SubtypeCode, row.DisplayName, qualifiedPath, fmt.Sprintf(`{"root_object": "%s", "subtype_code": "%s"}`, row.RootObject, row.SubtypeCode))
		if err != nil {
			return fmt.Errorf("failed upserting subtype BO node: %w", err)
		}

		for _, fieldName := range row.FieldAllowlist {
			attrPath := fmt.Sprintf("%s/%s", qualifiedPath, fieldName)
			attrNodeID := uuid.New()

			_, err := tx.ExecContext(ctx, `
				INSERT INTO catalog_node (node_id, tenant_id, node_type, node_key, node_name, qualified_path)
				VALUES ($1, $2, 'ATTRIBUTE', $3, $4, $5)
				ON CONFLICT (tenant_id, qualified_path) DO UPDATE
				SET updated_at = NOW()
			`, attrNodeID, tenantID, fieldName, fieldName, attrPath)
			if err != nil {
				return fmt.Errorf("failed upserting attribute node %s: %w", attrPath, err)
			}

			_, err = tx.ExecContext(ctx, `
				INSERT INTO catalog_edge (tenant_id, source_node_id, target_node_id, edge_type)
				SELECT $1, $2, $3, 'ATTRIBUTE_OF'
				WHERE NOT EXISTS (
					SELECT 1 FROM catalog_edge WHERE tenant_id = $1 AND source_node_id = $2 AND target_node_id = $3 AND edge_type = 'ATTRIBUTE_OF'
				)
			`, tenantID, attrNodeID, boNodeID)
			if err != nil {
				return fmt.Errorf("failed linking attribute edge: %w", err)
			}
		}
	}

	return tx.Commit()
}
