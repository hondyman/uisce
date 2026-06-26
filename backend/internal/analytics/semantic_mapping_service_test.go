package analytics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeColumnName(t *testing.T) {
	svc := &SemanticMappingService{}

	tests := []struct {
		input    string
		expected string
	}{
		{"USER_ID", "User Identifier"},
		{"ORDER_DT", "Order Date"},
		{"TOTAL_AMT", "Total Amount"},
		{"IS_ACTIVE", "Is Active"},
		{"customer_name", "Customer Name"},
		{"dim_product", "Product"},
		{"fct_sales", "Sales"},
	}

	for _, test := range tests {
		result := svc.normalizeColumnName(context.Background(), test.input)
		assert.Equal(t, test.expected, result)
	}
}

func TestDetermineTermType(t *testing.T) {
	svc := &SemanticMappingService{}

	tests := []struct {
		column   DatabaseColumn
		expected string
	}{
		{DatabaseColumn{Column: "USER_ID", DataType: "UUID"}, "Dimension"},
		{DatabaseColumn{Column: "ORDER_DT", DataType: "TIMESTAMP"}, "Time"},
		{DatabaseColumn{Column: "TOTAL_AMT", DataType: "DECIMAL"}, "Measure"},
		{DatabaseColumn{Column: "IS_ACTIVE", DataType: "BOOLEAN"}, "Dimension"},
		{DatabaseColumn{Column: "PRODUCT_NAME", DataType: "VARCHAR"}, "Dimension"},
		{DatabaseColumn{Column: "SALES_COUNT", DataType: "INTEGER"}, "Measure"},
	}

	for _, test := range tests {
		result, _ := svc.determineTermType(&test.column, nil)
		assert.Equal(t, test.expected, result)
	}
}

func TestGenerateBusinessTermName(t *testing.T) {
	svc := &SemanticMappingService{}

	tests := []struct {
		normalizedName string
		tableName      string
		expected       string
	}{
		{"User Identifier", "USERS", "User Identifier"},
		{"Order Date", "ORDERS", "Order Date"},
		{"Product Name", "PRODUCTS", "Product Name"},
		{"Sales Amount", "SALES", "Sales Amount"},
	}

	for _, test := range tests {
		result := svc.generateBusinessTermName(test.normalizedName, test.tableName)
		assert.Equal(t, test.expected, result)
	}
}

func TestDetermineDataDomain(t *testing.T) {
	svc := &SemanticMappingService{}

	tests := []struct {
		schema   string
		table    string
		expected []string
	}{
		{"SALES", "ORDERS", []string{"Sales", "Orders", "General"}},
		{"HR", "EMPLOYEES", []string{"Hr", "Employees", "General"}},
		{"PUBLIC", "DIM_PRODUCTS", []string{"Enterprise", "Products", "General"}},
	}

	for _, test := range tests {
		result := svc.determineDataDomain(test.schema, test.table)
		assert.Equal(t, test.expected, result)
	}
}

func TestInferSemanticTermProperties(t *testing.T) {
	svc := &SemanticMappingService{}

	tests := []struct {
		name               string
		column             *DatabaseColumn
		columnName         string
		termType           string
		expectedForeignKey bool
		expectedNullable   bool
		expectedTemporal   bool
		expectedStatusFlag bool
		shouldHaveSQL      bool
	}{
		{
			name: "Foreign key column",
			column: &DatabaseColumn{
				Column:      "USER_ID",
				Schema:      "public",
				Table:       "orders",
				Cardinality: 150000,
			},
			columnName:         "USER_ID",
			termType:           "Dimension",
			expectedForeignKey: true,
			expectedNullable:   false,
			expectedTemporal:   false,
			expectedStatusFlag: false,
			shouldHaveSQL:      true,
		},
		{
			name: "Regular column",
			column: &DatabaseColumn{
				Column: "CUSTOMER_NAME",
				Schema: "public",
				Table:  "customers",
			},
			columnName:         "CUSTOMER_NAME",
			termType:           "Dimension",
			expectedForeignKey: false,
			expectedNullable:   true,
			expectedTemporal:   false,
			expectedStatusFlag: false,
			shouldHaveSQL:      true,
		},
		{
			name: "Temporal column (CREATED_AT)",
			column: &DatabaseColumn{
				Column: "CREATED_AT",
				Schema: "public",
				Table:  "events",
			},
			columnName:         "CREATED_AT",
			termType:           "Time",
			expectedForeignKey: false,
			expectedNullable:   false,
			expectedTemporal:   true,
			expectedStatusFlag: false,
			shouldHaveSQL:      true,
		},
		{
			name: "Status flag column (IS_ACTIVE)",
			column: &DatabaseColumn{
				Column: "IS_ACTIVE",
				Schema: "public",
				Table:  "users",
			},
			columnName:         "IS_ACTIVE",
			termType:           "Dimension",
			expectedForeignKey: false,
			expectedNullable:   true,
			expectedTemporal:   false,
			expectedStatusFlag: true,
			shouldHaveSQL:      true,
		},
		{
			name: "Primary key column (ID)",
			column: &DatabaseColumn{
				Column: "ID",
				Schema: "public",
				Table:  "users",
			},
			columnName:         "ID",
			termType:           "Dimension",
			expectedForeignKey: true, // "ID" ends with ID, so it's marked as potential foreign key
			expectedNullable:   false,
			expectedTemporal:   false,
			expectedStatusFlag: false,
			shouldHaveSQL:      true,
		},
		{
			name: "Primary key column (PK_ID)",
			column: &DatabaseColumn{
				Column: "PK_ID",
				Schema: "public",
				Table:  "users",
			},
			columnName:         "PK_ID",
			termType:           "Dimension",
			expectedForeignKey: true, // Has PK_ prefix or _ID suffix
			expectedNullable:   false,
			expectedTemporal:   false,
			expectedStatusFlag: false,
			shouldHaveSQL:      true,
		},
		{
			name:               "Null column",
			column:             nil,
			columnName:         "",
			termType:           "Dimension",
			expectedForeignKey: false,
			expectedNullable:   false, // When column is nil, we don't set nullable
			expectedTemporal:   false,
			expectedStatusFlag: false,
			shouldHaveSQL:      false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			props := svc.inferSemanticTermProperties(test.column, test.termType, test.columnName)

			assert.Equal(t, test.termType, props["data_type"])

			// For foreign_key and nullable, check if they exist in props
			if test.column != nil {
				assert.Equal(t, test.expectedForeignKey, props["foreign_key"])
				assert.Equal(t, test.expectedNullable, props["nullable"])
			} else {
				// For null column, we only set data_type
				assert.Equal(t, test.termType, props["data_type"])
			}

			if test.expectedTemporal {
				assert.True(t, props["temporal"].(bool), "Expected temporal=true")
			}

			if test.expectedStatusFlag {
				assert.True(t, props["status_flag"].(bool), "Expected status_flag=true")
			}

			if test.shouldHaveSQL && test.columnName != "" {
				assert.NotNil(t, props["sql"], "Expected SQL property to be present")
			}

			if test.column != nil && test.column.Schema != "" {
				assert.Equal(t, test.column.Schema, props["schema"])
				assert.Equal(t, test.column.Table, props["table"])
				assert.Equal(t, test.column.Column, props["source_column"])

				// Verify physical_mapping is populated correctly
				pm, ok := props["physical_mapping"].(map[string]string)
				assert.True(t, ok, "physical_mapping should be a map[string]string")
				if ok {
					assert.Equal(t, test.column.Table, pm["table"])
					assert.Equal(t, test.column.Column, pm["column"])
				}
			}
		})
	}
}

func TestInferSemanticTermPropertiesCardinality(t *testing.T) {
	svc := &SemanticMappingService{}

	column := &DatabaseColumn{
		Column:           "PRODUCT_ID",
		Schema:           "public",
		Table:            "products",
		Cardinality:      5000,
		FrequentValues:   []string{"1", "2", "3"},
		InferredPatterns: []string{"numeric_id"},
	}

	props := svc.inferSemanticTermProperties(column, "Dimension", "PRODUCT_ID")

	// Cardinality might be int or int64 depending on the implementation
	cardinalityValue := props["cardinality"]
	assert.NotNil(t, cardinalityValue, "Cardinality should be present")

	// Convert to int for comparison
	assert.Equal(t, 5000, cardinalityValue)
	assert.NotNil(t, props["frequent_values"])
	assert.NotNil(t, props["inferred_patterns"])
	assert.Equal(t, "public", props["schema"])
	assert.Equal(t, "products", props["table"])
}
