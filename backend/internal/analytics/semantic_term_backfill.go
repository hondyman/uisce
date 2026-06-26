package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// BackfillSemanticTermSQLProperties adds SQL properties to existing semantic terms
// that don't have them yet, based on their column mappings
func (s *SemanticMappingService) BackfillSemanticTermSQLProperties(ctx context.Context, tenantID, datasourceID string) (int, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting SQL property backfill for tenant %s, datasource %s", tenantID, datasourceID)

	// Find all semantic terms that either don't have a sql property or have NULL/empty properties
	query := `
		SELECT 
			st.id,
			st.node_name as semantic_term_name,
			st.properties,
			c.node_name as column_name
		FROM catalog_node st
		-- Join to find columns that map to this semantic term via MAPS_TO edge
		INNER JOIN catalog_edge e ON e.target_node_id = st.id AND e.edge_type_id = '99c86836-98ef-45a3-82df-4c62b5730ac6'
		INNER JOIN catalog_node c ON c.id = e.source_node_id
		WHERE st.tenant_id = $1 
			AND st.tenant_datasource_id = $2
			AND st.node_type_id = $3
			-- Only semantic terms without sql property or with empty properties
			AND (
				st.properties IS NULL 
				OR st.properties = '{}'
				OR st.properties::jsonb ->> 'sql' IS NULL
			)
		ORDER BY st.node_name
	`

	type semanticTermRow struct {
		ID               string  `db:"id"`
		SemanticTermName string  `db:"semantic_term_name"`
		Properties       *string `db:"properties"`
		ColumnName       string  `db:"column_name"`
	}

	var rows []semanticTermRow
	err := s.db.SelectContext(ctx, &rows, query, tenantID, datasourceID, SemanticTermNodeTypeID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch semantic terms for backfill: %w", err)
	}

	if len(rows) == 0 {
		logger.Infof("No semantic terms found that need SQL property backfill")
		return 0, nil
	}

	logger.Infof("Found %d semantic terms that need SQL property backfill", len(rows))

	// Track which semantic terms we've already processed (in case multiple columns map to same term)
	processed := make(map[string]bool)
	updatedCount := 0

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, row := range rows {
		// Skip if we already processed this semantic term
		if processed[row.ID] {
			continue
		}

		// Parse existing properties or create new map
		properties := make(map[string]interface{})
		if row.Properties != nil && *row.Properties != "" && *row.Properties != "{}" {
			if err := json.Unmarshal([]byte(*row.Properties), &properties); err != nil {
				logger.Warnf("Failed to parse properties for semantic term %s: %v", row.SemanticTermName, err)
				// Continue with empty properties
			}
		}

		// Add SQL property with {CUBE}.column_name format
		properties["sql"] = fmt.Sprintf("{CUBE}.%s", row.ColumnName)

		// If data_type is missing, default to Dimension
		if _, hasDataType := properties["data_type"]; !hasDataType {
			properties["data_type"] = "Dimension"
		}

		// Marshal back to JSON
		propertiesJSON, err := json.Marshal(properties)
		if err != nil {
			logger.Errorf("Failed to marshal properties for semantic term %s: %v", row.SemanticTermName, err)
			continue
		}

		// Update the semantic term
		updateQuery := `
			UPDATE catalog_node 
			SET properties = $1, updated_at = NOW()
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, updateQuery, string(propertiesJSON), row.ID)
		if err != nil {
			logger.Errorf("Failed to update semantic term %s: %v", row.SemanticTermName, err)
			continue
		}

		logger.Infof("Updated semantic term '%s' with SQL property: {CUBE}.%s", row.SemanticTermName, row.ColumnName)
		processed[row.ID] = true
		updatedCount++
	}

	if err := tx.Commit(); err != nil {
		return updatedCount, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Successfully backfilled SQL properties for %d semantic terms", updatedCount)
	return updatedCount, nil
}

// BackfillAllTenantsSemanticTermSQLProperties runs the backfill for all tenants and datasources
func (s *SemanticMappingService) BackfillAllTenantsSemanticTermSQLProperties(ctx context.Context) (map[string]int, error) {
	logger := logging.GetLogger().Sugar()
	logger.Info("Starting SQL property backfill for all tenants")

	// Get all tenant/datasource combinations
	query := `
		SELECT DISTINCT tenant_id, tenant_datasource_id
		FROM catalog_node
		WHERE node_type_id = $1
		ORDER BY tenant_id, tenant_datasource_id
	`

	type tenantDatasource struct {
		TenantID     string `db:"tenant_id"`
		DatasourceID string `db:"tenant_datasource_id"`
	}

	var combinations []tenantDatasource
	err := s.db.SelectContext(ctx, &combinations, query, SemanticTermNodeTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant/datasource combinations: %w", err)
	}

	results := make(map[string]int)
	totalUpdated := 0

	for _, combo := range combinations {
		count, err := s.BackfillSemanticTermSQLProperties(ctx, combo.TenantID, combo.DatasourceID)
		if err != nil {
			logger.Errorf("Failed to backfill for tenant %s, datasource %s: %v", combo.TenantID, combo.DatasourceID, err)
			continue
		}
		key := fmt.Sprintf("%s/%s", combo.TenantID, combo.DatasourceID)
		results[key] = count
		totalUpdated += count
	}

	logger.Infof("Completed SQL property backfill: %d semantic terms updated across %d tenant/datasource combinations",
		totalUpdated, len(combinations))

	return results, nil
}

// BackfillPhysicalMappings adds physical_mapping properties to existing semantic terms
// that don't have them yet, based on their linked columns
func (s *SemanticMappingService) BackfillPhysicalMappings(ctx context.Context, tenantID, datasourceID string) (int, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Starting physical_mapping backfill for tenant %s, datasource %s", tenantID, datasourceID)

	// Find semantic terms missing physical_mapping
	// We look for terms linked to columns via any edge (MAPS_TO or has_context)
	query := `
		SELECT 
			st.id,
			st.node_name as semantic_term_name,
			st.properties,
			c.node_name as column_name,
			t.node_name as table_name
		FROM catalog_node st
		-- Join to find columns that map to this semantic term
		-- Using a more generic join since edge types might vary (MAPS_TO vs has_context)
		INNER JOIN catalog_edge e ON e.target_node_id = st.id
		INNER JOIN catalog_node c ON c.id = e.source_node_id
		LEFT JOIN catalog_node t ON t.id = c.parent_id
		WHERE st.tenant_id = $1 
			AND st.tenant_datasource_id = $2
			AND st.node_type_id = $3
			AND c.node_type_id = $4
			-- Only semantic terms missing physical_mapping
			AND (
				st.properties IS NULL 
				OR st.properties = '{}'
				OR st.properties::jsonb ->> 'physical_mapping' IS NULL
			)
		ORDER BY st.node_name
	`

	type semanticTermBackfillRow struct {
		ID               string  `db:"id"`
		SemanticTermName string  `db:"semantic_term_name"`
		Properties       *string `db:"properties"`
		ColumnName       string  `db:"column_name"`
		TableName        *string `db:"table_name"`
	}

	var rows []semanticTermBackfillRow
	// Note: We assume DatabaseColumnNodeTypeID is available in package
	err := s.db.SelectContext(ctx, &rows, query, tenantID, datasourceID, SemanticTermNodeTypeID, DatabaseColumnNodeTypeID)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch semantic terms for physical_mapping backfill: %w", err)
	}

	if len(rows) == 0 {
		logger.Infof("No semantic terms found that need physical_mapping backfill")
		return 0, nil
	}

	logger.Infof("Found %d semantic terms that need physical_mapping backfill", len(rows))

	processed := make(map[string]bool)
	updatedCount := 0

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	for _, row := range rows {
		if processed[row.ID] {
			continue
		}

		// Skip if table name is missing (orphan column?)
		if row.TableName == nil || *row.TableName == "" {
			logger.Warnf("Skipping backfill for term %s: linked column %s has no parent table", row.SemanticTermName, row.ColumnName)
			continue
		}

		// Parse existing properties
		properties := make(map[string]interface{})
		if row.Properties != nil && *row.Properties != "" && *row.Properties != "{}" {
			if err := json.Unmarshal([]byte(*row.Properties), &properties); err != nil {
				logger.Warnf("Failed to parse properties for semantic term %s: %v", row.SemanticTermName, err)
			}
		}

		// Add physical_mapping
		properties["physical_mapping"] = map[string]string{
			"table":  *row.TableName,
			"column": row.ColumnName,
		}

		// Marshal back to JSON
		propertiesJSON, err := json.Marshal(properties)
		if err != nil {
			logger.Errorf("Failed to marshal properties for semantic term %s: %v", row.SemanticTermName, err)
			continue
		}

		// Update the semantic term
		updateQuery := `
			UPDATE catalog_node 
			SET properties = $1, updated_at = NOW()
			WHERE id = $2
		`
		_, err = tx.ExecContext(ctx, updateQuery, string(propertiesJSON), row.ID)
		if err != nil {
			logger.Errorf("Failed to update semantic term %s: %v", row.SemanticTermName, err)
			continue
		}

		logger.Infof("Updated semantic term '%s' with physical_mapping: %s.%s", row.SemanticTermName, *row.TableName, row.ColumnName)
		processed[row.ID] = true
		updatedCount++
	}

	if err := tx.Commit(); err != nil {
		return updatedCount, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Successfully backfilled physical_mapping for %d semantic terms", updatedCount)
	return updatedCount, nil
}

// BackfillAllTenantsPhysicalMappings runs the physical_mapping backfill for all tenants
func (s *SemanticMappingService) BackfillAllTenantsPhysicalMappings(ctx context.Context) (map[string]int, error) {
	logger := logging.GetLogger().Sugar()
	logger.Info("Starting physical_mapping backfill for all tenants")

	// Get all tenant/datasource combinations
	query := `
		SELECT DISTINCT tenant_id, tenant_datasource_id
		FROM catalog_node
		WHERE node_type_id = $1
		ORDER BY tenant_id, tenant_datasource_id
	`

	type tenantDatasource struct {
		TenantID     string  `db:"tenant_id"`
		DatasourceID *string `db:"tenant_datasource_id"`
	}

	var combinations []tenantDatasource
	err := s.db.SelectContext(ctx, &combinations, query, SemanticTermNodeTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tenant/datasource combinations: %w", err)
	}

	results := make(map[string]int)
	totalUpdated := 0

	for _, combo := range combinations {
		if combo.DatasourceID == nil {
			continue
		}
		count, err := s.BackfillPhysicalMappings(ctx, combo.TenantID, *combo.DatasourceID)
		if err != nil {
			logger.Errorf("Failed to backfill physical_mapping for tenant %s, datasource %s: %v", combo.TenantID, *combo.DatasourceID, err)
			continue
		}
		key := fmt.Sprintf("%s/%s", combo.TenantID, *combo.DatasourceID)
		results[key] = count
		totalUpdated += count
	}

	logger.Infof("Completed physical_mapping backfill: %d semantic terms updated across %d tenant/datasource combinations",
		totalUpdated, len(combinations))

	return results, nil
}
