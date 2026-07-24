package cbo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DBPreAggRepository implements PreAggRepository using SQL
type DBPreAggRepository struct {
	db *sqlx.DB
}

// NewDBPreAggRepository creates a new DBPreAggRepository
func NewDBPreAggRepository(db *sqlx.DB) *DBPreAggRepository {
	return &DBPreAggRepository{db: db}
}

// ListForBO lists pre-aggregations for a BO
func (r *DBPreAggRepository) ListForBO(ctx context.Context, env string, tenantID *uuid.UUID, boName string, region string) ([]PreAggDescriptor, error) {
	// Query catalog_node table for pre_aggregation nodes
	// Note: We use the existing catalog_node schema
	// We assume 'pre_aggregation' node type exists

	// Query logic similar to analytics.PreAggregationService.ListByBO
	// But independent to avoid cycles

	query := `
		SELECT n.node_name, n.properties, n.config
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE nt.catalog_type_name = 'pre_aggregation'
		  AND (n.tenant_id = $1 OR n.tenant_id IS NULL)
		  AND n.properties->>'bo_name' = $2
		  AND n.properties->>'region' = $3
	`

	// Handle nil tenantID (global pre-aggs?)
	var tid uuid.UUID
	if tenantID != nil {
		tid = *tenantID
	}

	rows, err := r.db.QueryContext(ctx, query, tid, boName, region)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var descriptors []PreAggDescriptor

	for rows.Next() {
		var nodeName string
		var propsRaw, configRaw []byte
		if err := rows.Scan(&nodeName, &propsRaw, &configRaw); err != nil {
			continue
		}

		var props map[string]interface{}
		var config struct {
			TargetName      string `json:"target_name"` // Assuming simplified config
			Materialization struct {
				TargetName string `json:"target_name"`
			} `json:"materialization"`
			GroupBy  []string `json:"group_by"`
			Measures []string `json:"measures"`
		}

		if err := json.Unmarshal(propsRaw, &props); err != nil {
			continue
		}
		if err := json.Unmarshal(configRaw, &config); err != nil {
			continue
		}

		targetTable := config.Materialization.TargetName
		if targetTable == "" {
			targetTable = fmt.Sprintf("preagg_%s", nodeName) // Fallback
		}

		// Map to PreAggDescriptor
		// Note: Storage bytes and refresh info would come from properties or stats table
		storageBytes, _ := props["size_bytes"].(float64)

		regionProp := ""
		if rp, ok := props["region"].(string); ok {
			regionProp = rp
		}

		descriptors = append(descriptors, PreAggDescriptor{
			Name:                nodeName,
			TargetTable:         targetTable,
			Dimensions:          config.GroupBy,
			Measures:            config.Measures,
			RefreshFrequencySec: 3600, // Default or parse from props
			StorageBytes:        int64(storageBytes),
			AvgSpeedup:          10.0, // Default estimate
			Region:              regionProp,
		})
	}

	return descriptors, nil
}

// DBEntitlementRepository implements EntitlementRepository using SQL
type DBEntitlementRepository struct {
	db *sqlx.DB
}

// NewDBEntitlementRepository creates a new DBEntitlementRepository
func NewDBEntitlementRepository(db *sqlx.DB) *DBEntitlementRepository {
	return &DBEntitlementRepository{db: db}
}

// GetPoliciesForBO lists entitlement policies for a BO
func (r *DBEntitlementRepository) GetPoliciesForBO(ctx context.Context, tenantID *uuid.UUID, boName string) ([]EntitlementPolicy, error) {
	// For now, return a default policy
	// In future, table semantic.entitlements could store this

	// Placeholder: Look for a special entitlement config or default to JOIN
	// We return empty list which defaults to JOIN in planner
	return []EntitlementPolicy{}, nil
}
