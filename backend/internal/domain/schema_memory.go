package domain

import "fmt"

// InMemorySchema provides asset schema information from memory
type InMemorySchema struct {
	Map map[string]AssetSchema
}

// GetAssetSchema returns the schema for the given asset ID
func (m *InMemorySchema) GetAssetSchema(assetID string) (AssetSchema, error) {
	if s, ok := m.Map[assetID]; ok {
		return s, nil
	}
	return AssetSchema{ColumnsByScope: map[string][]string{}}, nil
}

// GetTableSchema returns the schema for a specific table (not implemented in memory version)
func (m *InMemorySchema) GetTableSchema(assetID, tableName string) (TableSchema, error) {
	// For now, return empty schema - this would need to be implemented based on actual table structure
	return TableSchema{}, fmt.Errorf("getTableSchema not implemented for InMemorySchema")
}

// Example schema configuration
func NewExampleSchema() *InMemorySchema {
	return &InMemorySchema{
		Map: map[string]AssetSchema{
			"asset-orders": {
				ColumnsByScope: map[string][]string{
					"metrics":    {"avg_order_value", "total_orders", "net_margin"},
					"dimensions": {"region", "customer_id", "order_date"},
				},
				DefaultFilters: []string{"tenant_id = $1"},
			},
			"asset-customers": {
				ColumnsByScope: map[string][]string{
					"metrics":    {"lifetime_value", "order_count"},
					"dimensions": {"region", "segment", "signup_date"},
				},
				DefaultFilters: []string{"tenant_id = $1"},
			},
		},
	}
}
