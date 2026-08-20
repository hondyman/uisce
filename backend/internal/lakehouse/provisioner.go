package lakehouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProvisionRequest defines parameters to create and shadow an Iceberg table tier.
type ProvisionRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	DatasourceID    uuid.UUID `json:"datasource_id"`
	OLTPTableNodeID uuid.UUID `json:"oltp_table_node_id"`
	PartitionColumn string    `json:"partition_column"`
}

// ProvisionResult returns the newly registered shadow node and its properties.
type ProvisionResult struct {
	IcebergTableNodeID uuid.UUID `json:"iceberg_table_node_id"`
	PhysicalPath       string    `json:"physical_path"`
	ColumnsProvisioned int       `json:"columns_provisioned"`
	ProvisionedAt      time.Time `json:"provisioned_at"`
}

// LakehouseProvisioner manages catalog registration and semantic twinning for Iceberg tables.
type LakehouseProvisioner interface {
	ReconcileShadowTable(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error)
}

type provisionerService struct {
	db        *sql.DB
	twinClone GraphTwinCloner
}

func NewLakehouseProvisioner(db *sql.DB, twinClone GraphTwinCloner) LakehouseProvisioner {
	return &provisionerService{
		db:        db,
		twinClone: twinClone,
	}
}

func (p *provisionerService) ReconcileShadowTable(ctx context.Context, req ProvisionRequest) (*ProvisionResult, error) {
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// 1. Fetch OLTP Table details with Rule 7 tenant guard
	var oltpName, oltpPath, oltpDesc string
	var oltpConfigRaw, oltpPropsRaw []byte
	err = tx.QueryRowContext(ctx, `
		SELECT node_name, qualified_path, description, config, properties 
		FROM catalog_node 
		WHERE id = $1 AND tenant_id = $2 AND is_active = true
	`, req.OLTPTableNodeID, req.TenantID).Scan(
		&oltpName, &oltpPath, &oltpDesc, &oltpConfigRaw, &oltpPropsRaw,
	)
	if err != nil {
		return nil, fmt.Errorf("target OLTP node not found or unauthorized: %w", err)
	}

	// 2. Resolve Table and Column node type IDs (Rule 1 & Rule 2: Config-Before-Code)
	var tableNodeTypeID, colNodeTypeID uuid.UUID
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM catalog_node_types WHERE catalog_type_name = 'table' LIMIT 1`,
	).Scan(&tableNodeTypeID)
	if err != nil {
		return nil, fmt.Errorf("catalog_node_type 'table' missing: %w", err)
	}

	err = tx.QueryRowContext(ctx,
		`SELECT id FROM catalog_node_types WHERE catalog_type_name = 'column' LIMIT 1`,
	).Scan(&colNodeTypeID)
	if err != nil {
		return nil, fmt.Errorf("catalog_node_type 'column' missing: %w", err)
	}

	// 3. Construct deterministically formatted physical path (e.g. iceberg.t_<short_tenant>.ds_<short_ds>_<table>)
	shortTenant := strings.ReplaceAll(req.TenantID.String()[:8], "-", "")
	shortDS := strings.ReplaceAll(req.DatasourceID.String()[:8], "-", "")
	tableNameClean := strings.ToLower(strings.ReplaceAll(oltpName, " ", "_"))
	physicalPath := fmt.Sprintf("iceberg.t_%s.ds_%s_%s", shortTenant, shortDS, tableNameClean)
	icebergQualifiedPath := fmt.Sprintf("/iceberg/%s/%s", shortTenant, tableNameClean)

	// 4. Provision or Update the Iceberg catalog_node (Rule 6: Metadata in Graph, State in Lakehouse)
	nodeProps := map[string]interface{}{
		"storage_tier":      "COLD_LAKEHOUSE",
		"engine":            "ICEBERG",
		"physical_path":     physicalPath,
		"partition_column":  req.PartitionColumn,
		"shadow_of_node_id": req.OLTPTableNodeID.String(),
	}
	propsJSON, _ := json.Marshal(nodeProps)

	configMap := map[string]interface{}{
		"format":            "PARQUET",
		"write_compression": "ZSTD",
		"datasource_id":     req.DatasourceID.String(),
	}
	configJSON, _ := json.Marshal(configMap)

	var icebergTableNodeID uuid.UUID
	err = tx.QueryRowContext(ctx, `
		INSERT INTO catalog_node (
			id, tenant_id, node_type_id, parent_id, node_name, 
			qualified_path, description, config, properties, is_active, created_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, $2, $3, $4, 
			$5, $6, $7, $8, true, NOW(), NOW()
		)
		ON CONFLICT (tenant_id, qualified_path)
		DO UPDATE SET 
			config = EXCLUDED.config, 
			properties = EXCLUDED.properties, 
			updated_at = NOW()
		RETURNING id
	`, req.TenantID, tableNodeTypeID, req.DatasourceID, oltpName+" (Iceberg Lakehouse)",
		icebergQualifiedPath, "Iceberg shadow tier for "+oltpDesc, configJSON, propsJSON,
	).Scan(&icebergTableNodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to register Iceberg catalog node: %w", err)
	}

	// 5. Reconcile Child Column Nodes under the Iceberg Table
	colRows, err := tx.QueryContext(ctx, `
		SELECT node_name, description, config, properties 
		FROM catalog_node 
		WHERE parent_id = $1 AND tenant_id = $2 AND is_active = true
	`, req.OLTPTableNodeID, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query source OLTP child columns: %w", err)
	}
	defer colRows.Close()

	colCount := 0
	for colRows.Next() {
		var cName, cDesc string
		var cConfig, cProps []byte
		if err := colRows.Scan(&cName, &cDesc, &cConfig, &cProps); err != nil {
			return nil, fmt.Errorf("failed to scan column data: %w", err)
		}

		colPath := fmt.Sprintf("%s/%s", icebergQualifiedPath, cName)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO catalog_node (
				id, tenant_id, node_type_id, parent_id, node_name, 
				qualified_path, description, config, properties, is_active, created_at, updated_at
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, 
				$5, $6, $7, $8, true, NOW(), NOW()
			)
			ON CONFLICT (tenant_id, qualified_path)
			DO UPDATE SET 
				config = EXCLUDED.config, 
				properties = EXCLUDED.properties, 
				updated_at = NOW()
		`, req.TenantID, colNodeTypeID, icebergTableNodeID, cName, colPath, cDesc, cConfig, cProps)
		if err != nil {
			return nil, fmt.Errorf("failed to register Iceberg child column %s: %w", cName, err)
		}
		colCount++
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit table provisioning transaction: %w", err)
	}

	// 6. Invoke Graph Twin Cloner to draw SHADOWS and replicate MAPS_TO semantic edges
	if err := p.twinClone.CloneSemanticMappings(ctx, req.TenantID, req.OLTPTableNodeID, icebergTableNodeID); err != nil {
		return nil, fmt.Errorf("failed to complete semantic graph cloning: %w", err)
	}

	return &ProvisionResult{
		IcebergTableNodeID: icebergTableNodeID,
		PhysicalPath:       physicalPath,
		ColumnsProvisioned: colCount,
		ProvisionedAt:      time.Now().UTC(),
	}, nil
}
