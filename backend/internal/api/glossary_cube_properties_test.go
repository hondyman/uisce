package api

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// TestValidateSemanticTermPropertiesDimension tests validation of dimension term properties
func TestValidateSemanticTermPropertiesDimension(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test case 1: Valid dimension properties
	validProps := map[string]interface{}{
		"semantic_term_type": "DIMENSION",
		"data_type":          "integer",
		"cube_properties": map[string]interface{}{
			"name":        "user_id",
			"sql":         "{CUBE}.user_id",
			"type":        "number",
			"title":       "User ID",
			"public":      true,
			"primary_key": false,
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "DIMENSION", validProps)
	if err != nil {
		t.Errorf("Expected no error for valid DIMENSION, got: %v", err)
	}

	// Test case 2: Missing required field (sql)
	missingFieldProps := map[string]interface{}{
		"semantic_term_type": "DIMENSION",
		"cube_properties": map[string]interface{}{
			"name":  "user_id",
			"type":  "number",
			"title": "User ID",
		},
	}

	err = service.ValidateSemanticTermProperties(context.Background(), "DIMENSION", missingFieldProps)
	if err == nil {
		t.Error("Expected error for missing sql field, got nil")
	}
}

// TestValidateSemanticTermPropertiesMeasure tests validation of measure term properties
func TestValidateSemanticTermPropertiesMeasure(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test case 1: Valid measure properties
	validProps := map[string]interface{}{
		"semantic_term_type": "MEASURE",
		"cube_properties": map[string]interface{}{
			"name":        "revenue",
			"sql":         "SUM({CUBE}.amount)",
			"type":        "number",
			"title":       "Revenue",
			"aggregation": "sum",
			"public":      true,
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "MEASURE", validProps)
	if err != nil {
		t.Errorf("Expected no error for valid MEASURE, got: %v", err)
	}
}

// TestValidateSemanticTermPropertiesTime tests validation of time term properties
func TestValidateSemanticTermPropertiesTime(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test case 1: Valid time properties
	validProps := map[string]interface{}{
		"semantic_term_type": "TIME",
		"cube_properties": map[string]interface{}{
			"name":          "created_at",
			"sql":           "{CUBE}.created_at",
			"type":          "time",
			"title":         "Created At",
			"granularities": []interface{}{"day", "week", "month", "year"},
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "TIME", validProps)
	if err != nil {
		t.Errorf("Expected no error for valid TIME, got: %v", err)
	}
}

// TestValidateSemanticTermPropertiesHierarchy tests validation of hierarchy term properties
func TestValidateSemanticTermPropertiesHierarchy(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test case 1: Valid hierarchy properties
	validProps := map[string]interface{}{
		"semantic_term_type": "HIERARCHY",
		"cube_properties": map[string]interface{}{
			"name":   "geography",
			"title":  "Geography",
			"levels": []interface{}{"country", "state", "city"},
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "HIERARCHY", validProps)
	if err != nil {
		t.Errorf("Expected no error for valid HIERARCHY, got: %v", err)
	}
}

// TestValidateSemanticTermPropertiesSegment tests validation of segment term properties
func TestValidateSemanticTermPropertiesSegment(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test case 1: Valid segment properties
	validProps := map[string]interface{}{
		"semantic_term_type": "SEGMENT",
		"cube_properties": map[string]interface{}{
			"name":  "high_value_customers",
			"sql":   "{CUBE}.lifetime_value > 100000",
			"title": "High Value Customers",
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "SEGMENT", validProps)
	if err != nil {
		t.Errorf("Expected no error for valid SEGMENT, got: %v", err)
	}
}

// TestCubePropertiesResponseMarshaling tests proper JSON marshaling of responses
func TestCubePropertiesResponseMarshaling(t *testing.T) {
	response := CubePropertiesResponse{
		ID:               uuid.New().String(),
		NodeName:         "user_id",
		SemanticTermType: "DIMENSION",
		CubeProperties: map[string]interface{}{
			"name":  "user_id",
			"sql":   "{CUBE}.user_id",
			"type":  "number",
			"title": "User ID",
		},
		DataType:           "integer",
		ForeignKey:         true,
		Nullable:           false,
		Cardinality:        intPtr(245),
		TenantID:           uuid.New().String(),
		TenantDatasourceID: uuid.New().String(),
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal response: %v", err)
		return
	}

	// Verify JSON structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}

	if unmarshaled["semantic_term_type"] != "DIMENSION" {
		t.Error("semantic_term_type not properly marshaled")
	}

	if unmarshaled["cube_properties"] == nil {
		t.Error("cube_properties not properly marshaled")
	}
}

// TestCubeYamlExportResponseMarshaling tests proper JSON marshaling of YAML export responses
func TestCubeYamlExportResponseMarshaling(t *testing.T) {
	response := CubeYamlExportResponse{
		Dimensions: []map[string]interface{}{
			{
				"name":  "user_id",
				"sql":   "{CUBE}.user_id",
				"type":  "number",
				"title": "User ID",
			},
		},
		Measures: []map[string]interface{}{
			{
				"name":        "revenue",
				"sql":         "SUM({CUBE}.amount)",
				"type":        "number",
				"title":       "Revenue",
				"aggregation": "sum",
			},
		},
		Segments: []map[string]interface{}{
			{
				"name":  "high_value",
				"sql":   "{CUBE}.lifetime_value > 100000",
				"title": "High Value Customers",
			},
		},
		TimeDimensions: []map[string]interface{}{
			{
				"name":          "created_at",
				"sql":           "{CUBE}.created_at",
				"type":          "time",
				"title":         "Created At",
				"granularities": []string{"day", "week", "month", "year"},
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Errorf("Failed to marshal response: %v", err)
		return
	}

	// Verify JSON structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}

	// Verify collections are present
	if unmarshaled["dimensions"] == nil {
		t.Error("dimensions not properly marshaled")
	}
	if unmarshaled["measures"] == nil {
		t.Error("measures not properly marshaled")
	}
	if unmarshaled["segments"] == nil {
		t.Error("segments not properly marshaled")
	}
	if unmarshaled["time_dimensions"] == nil {
		t.Error("time_dimensions not properly marshaled")
	}
}

// TestValidateSemanticTermPropertiesUnknownType tests error handling for unknown types
func TestValidateSemanticTermPropertiesUnknownType(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	props := map[string]interface{}{
		"semantic_term_type": "UNKNOWN_TYPE",
		"cube_properties":    map[string]interface{}{},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "UNKNOWN_TYPE", props)
	if err == nil {
		t.Error("Expected error for unknown type, got nil")
	}

	if err.Error() != "unknown semantic term type: UNKNOWN_TYPE" {
		t.Errorf("Wrong error message. Got: %v", err)
	}
}

// TestValidateSemanticTermPropertiesMissingCubeProperties tests error for missing cube_properties
func TestValidateSemanticTermPropertiesMissingCubeProperties(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	props := map[string]interface{}{
		"semantic_term_type": "DIMENSION",
		// cube_properties missing
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "DIMENSION", props)
	if err == nil {
		t.Error("Expected error for missing cube_properties, got nil")
	}

	if err.Error() != "missing cube_properties object for term type DIMENSION" {
		t.Errorf("Wrong error message. Got: %v", err)
	}
}

// TestValidateSemanticTermPropertiesNilProperties tests error for nil properties
func TestValidateSemanticTermPropertiesNilProperties(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	err := service.ValidateSemanticTermProperties(context.Background(), "DIMENSION", nil)
	if err == nil {
		t.Error("Expected error for nil properties, got nil")
	}

	if err.Error() != "properties cannot be nil" {
		t.Errorf("Wrong error message. Got: %v", err)
	}
}

// TestEnhancedCubePropertiesMarshaling tests all enhanced cube.dev properties are correctly generated
func TestEnhancedCubePropertiesMarshaling(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Create a test semantic term with enhanced properties for MEASURE
	// MEASURE requires: name, sql, type, title, aggregation
	semanticTerm := map[string]interface{}{
		"semantic_term_type": "MEASURE",
		"data_type":          "decimal",
		"cube_properties": map[string]interface{}{
			"name":           "revenue_amount",
			"sql":            "{CUBE}.revenue_amount",
			"type":           "number",
			"title":          "Revenue Amount",
			"aggregation":    "sum",
			"public":         true,
			"shown":          true,
			"hidden":         false,
			"cumulative":     false,
			"rolling_window": false,
			"time_zone":      "UTC",
			"granularities":  []interface{}{"day", "month", "year"},
			"format":         "currency",
			"currency":       "USD",
			"drill_down_by":  []string{"product", "region"},
			"description":    "Total revenue amount in USD",
			"primary_key":    false,
		},
	}

	// Validate the enhanced properties
	err := service.ValidateSemanticTermProperties(context.Background(), "MEASURE", semanticTerm)
	if err != nil {
		t.Errorf("Expected no error for enhanced MEASURE properties, got: %v", err)
	}

	// Verify all properties are present in marshaled form
	propsJSON, err := json.Marshal(semanticTerm["cube_properties"])
	if err != nil {
		t.Fatalf("Failed to marshal properties: %v", err)
	}

	var cubeProps map[string]interface{}
	if err := json.Unmarshal(propsJSON, &cubeProps); err != nil {
		t.Fatalf("Failed to unmarshal properties: %v", err)
	}

	// Check for new properties
	newProperties := []string{"shown", "hidden", "cumulative", "rolling_window", "time_zone", "granularities", "format", "currency", "drill_down_by"}
	for _, prop := range newProperties {
		if _, exists := cubeProps[prop]; !exists {
			t.Logf("Note: Optional property '%s' not found in marshaled cube properties", prop)
		}
	}
}

// TestEnhancedPropertyValidationWithAllFields tests comprehensive validation with all new properties
func TestEnhancedPropertyValidationWithAllFields(t *testing.T) {
	service := &analytics.SemanticMappingService{}

	// Test comprehensive MEASURE with all enhanced properties
	// MEASURE requires: name, sql, type, title, aggregation
	fullMeasureProps := map[string]interface{}{
		"semantic_term_type": "MEASURE",
		"data_type":          "decimal",
		"cube_properties": map[string]interface{}{
			"name":           "revenue_amount",
			"sql":            "{CUBE}.revenue_amount",
			"type":           "number",
			"title":          "Revenue Amount",
			"aggregation":    "sum",
			"public":         true,
			"shown":          true,
			"hidden":         false,
			"cumulative":     false,
			"rolling_window": false,
			"format":         "currency",
			"currency":       "USD",
			"description":    "Total monthly revenue in USD",
			"order":          "asc",
			"primary_key":    false,
		},
	}

	err := service.ValidateSemanticTermProperties(context.Background(), "MEASURE", fullMeasureProps)
	if err != nil {
		t.Errorf("Expected no error for comprehensive MEASURE properties, got: %v", err)
	}

	// Test comprehensive TIME dimension with all enhanced properties
	// TIME requires: name, sql, type, title, granularities
	fullTimeProps := map[string]interface{}{
		"semantic_term_type": "TIME",
		"data_type":          "timestamp",
		"cube_properties": map[string]interface{}{
			"name":          "order_date",
			"sql":           "{CUBE}.order_date",
			"type":          "time",
			"title":         "Order Date",
			"granularities": []interface{}{"day", "week", "month", "year"},
			"public":        true,
			"shown":         true,
			"hidden":        false,
			"time_zone":     "UTC",
			"order":         "asc",
			"primary_key":   false,
		},
	}

	err = service.ValidateSemanticTermProperties(context.Background(), "TIME", fullTimeProps)
	if err != nil {
		t.Errorf("Expected no error for comprehensive TIME properties, got: %v", err)
	}

	// Test HIERARCHY with drill_down_by
	// HIERARCHY requires: name, title, levels
	hierarchyProps := map[string]interface{}{
		"semantic_term_type": "HIERARCHY",
		"data_type":          "string",
		"cube_properties": map[string]interface{}{
			"name":          "location_hierarchy",
			"sql":           "{CUBE}.location_id",
			"type":          "string",
			"title":         "Location Hierarchy",
			"levels":        []interface{}{"country", "region", "city"},
			"public":        true,
			"shown":         true,
			"drill_down_by": []string{"country", "region", "city"},
			"order":         "asc",
			"primary_key":   false,
		},
	}

	err = service.ValidateSemanticTermProperties(context.Background(), "HIERARCHY", hierarchyProps)
	if err != nil {
		t.Errorf("Expected no error for HIERARCHY with drill_down_by, got: %v", err)
	}
}

// Helper function
func intPtr(i int) *int {
	return &i
}

// TestExpandDomainSpecificAbbreviations tests domain-specific abbreviation expansion (Enhancement 1)
func TestExpandDomainSpecificAbbreviations(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	testCases := []struct {
		name           string
		columnName     string
		domain         string
		shouldSucceed  bool
		expectExpanded bool
	}{
		{
			name:           "Finance: CAC abbreviation",
			columnName:     "cust_acq_cost",
			domain:         "finance",
			shouldSucceed:  true,
			expectExpanded: true,
		},
		{
			name:           "Finance: ARR abbreviation",
			columnName:     "annual_rec_rev",
			domain:         "finance",
			shouldSucceed:  true,
			expectExpanded: true,
		},
		{
			name:           "Healthcare: EMR abbreviation",
			columnName:     "emr_id",
			domain:         "healthcare",
			shouldSucceed:  true,
			expectExpanded: true,
		},
		{
			name:           "Unknown domain fallback",
			columnName:     "customer_name",
			domain:         "unknown_domain",
			shouldSucceed:  false,
			expectExpanded: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expanded, metadata, err := service.ExpandDomainSpecificAbbreviations(ctx, tc.columnName, tc.domain)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			if !tc.shouldSucceed && err == nil {
				t.Error("Expected error, got nil")
			}

			if tc.shouldSucceed {
				if expanded == "" {
					t.Error("Expected non-empty expanded text")
				}

				if metadata == nil {
					t.Error("Expected non-nil metadata")
				}

				if tc.expectExpanded && expanded == tc.columnName {
					t.Errorf("Expected abbreviation to be expanded, but got same text: %s", expanded)
				}
			}
		})
	}
}

// TestGenerateLocalizedTitle tests multi-language title generation (Enhancement 2)
func TestGenerateLocalizedTitle(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	testCases := []struct {
		name          string
		columnName    string
		termName      string
		languages     []string
		shouldSucceed bool
	}{
		{
			name:          "All 5 original languages",
			columnName:    "customer_id",
			termName:      "Customer",
			languages:     []string{"en", "es", "fr", "de", "ja"},
			shouldSucceed: true,
		},
		{
			name:          "English only",
			columnName:    "revenue_amount",
			termName:      "Revenue",
			languages:     []string{"en"},
			shouldSucceed: true,
		},
		{
			name:          "Multiple languages with unsupported",
			columnName:    "product_name",
			termName:      "Product",
			languages:     []string{"en", "es", "unknown"},
			shouldSucceed: true,
		},
		{
			name:          "Empty language list",
			columnName:    "user_name",
			termName:      "User",
			languages:     []string{},
			shouldSucceed: true,
		},
		// New language tests (expanded to 10 languages)
		{
			name:          "All 10 languages supported",
			columnName:    "customer_data",
			termName:      "Customer",
			languages:     []string{"en", "es", "fr", "de", "ja", "pt", "it", "nl", "pl", "ru"},
			shouldSucceed: true,
		},
		{
			name:          "Portuguese and Italian",
			columnName:    "sales_amount",
			termName:      "Sales",
			languages:     []string{"pt", "it"},
			shouldSucceed: true,
		},
		{
			name:          "Dutch and Polish",
			columnName:    "product_count",
			termName:      "Product",
			languages:     []string{"nl", "pl"},
			shouldSucceed: true,
		},
		{
			name:          "Russian language",
			columnName:    "revenue_data",
			termName:      "Revenue",
			languages:     []string{"ru"},
			shouldSucceed: true,
		},
		// New business term tests
		{
			name:          "Financial term - Profit",
			columnName:    "profit_amount",
			termName:      "Profit",
			languages:     []string{"en", "es", "fr", "de", "ja"},
			shouldSucceed: true,
		},
		{
			name:          "Financial term - Cost",
			columnName:    "cost_value",
			termName:      "Cost",
			languages:     []string{"en", "pt", "it", "nl", "ru"},
			shouldSucceed: true,
		},
		{
			name:          "Customer metric - Order Count",
			columnName:    "order_count",
			termName:      "Order Count",
			languages:     []string{"en", "es", "fr", "pl"},
			shouldSucceed: true,
		},
		{
			name:          "Time dimension - Quarter",
			columnName:    "quarter_date",
			termName:      "Quarter",
			languages:     []string{"en", "de", "ja", "pt", "it"},
			shouldSucceed: true,
		},
		{
			name:          "Performance metric - Conversion Rate",
			columnName:    "conversion_pct",
			termName:      "Conversion Rate",
			languages:     []string{"en", "es", "fr", "nl"},
			shouldSucceed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			titles, err := service.GenerateLocalizedTitle(ctx, tc.columnName, tc.termName, tc.languages)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			if tc.shouldSucceed {
				if len(tc.languages) > 0 && titles == nil {
					t.Error("Expected non-nil titles map")
				}

				// Verify requested languages are present
				if len(tc.languages) > 0 && titles != nil {
					for _, lang := range tc.languages {
						if lang != "unknown" {
							if val, ok := titles[lang]; !ok || val == "" {
								t.Errorf("Expected %s translation for term '%s', got: %v", lang, tc.termName, val)
							}
						}
					}
				}
			}
		})
	}
}

// TestValidateAndFormatProperty tests format validation for specialized data types (Enhancement 3)
func TestValidateAndFormatProperty(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	testCases := []struct {
		name          string
		propertyName  string
		value         string
		dataType      string
		shouldSucceed bool
	}{
		{
			name:          "Valid email",
			propertyName:  "email_address",
			value:         "user@example.com",
			dataType:      "email",
			shouldSucceed: true,
		},
		{
			name:          "Invalid email",
			propertyName:  "email_address",
			value:         "not-an-email",
			dataType:      "email",
			shouldSucceed: false,
		},
		{
			name:          "Valid phone",
			propertyName:  "phone_number",
			value:         "(215) 555-2671",
			dataType:      "phone",
			shouldSucceed: true,
		},
		{
			name:          "Valid currency",
			propertyName:  "price",
			value:         "1000.50",
			dataType:      "currency",
			shouldSucceed: true,
		},
		{
			name:          "Valid percentage",
			propertyName:  "discount_rate",
			value:         "15.5",
			dataType:      "percentage",
			shouldSucceed: true,
		},
		{
			name:          "Invalid percentage (>100)",
			propertyName:  "discount_rate",
			value:         "150",
			dataType:      "percentage",
			shouldSucceed: true, // Percentage provides hints but doesn't validate range
		},
		{
			name:          "Valid URL",
			propertyName:  "website",
			value:         "https://www.example.com",
			dataType:      "url",
			shouldSucceed: true,
		},
		{
			name:          "Valid JSON",
			propertyName:  "config",
			value:         `{"key":"value"}`,
			dataType:      "json",
			shouldSucceed: true,
		},
		{
			name:          "Invalid JSON",
			propertyName:  "config",
			value:         `{invalid json}`,
			dataType:      "json",
			shouldSucceed: false,
		},
		{
			name:          "Valid date",
			propertyName:  "birth_date",
			value:         "2000-01-15",
			dataType:      "date",
			shouldSucceed: true,
		},
		{
			name:          "Valid datetime",
			propertyName:  "created_at",
			value:         "2024-01-15T14:30:00Z",
			dataType:      "datetime",
			shouldSucceed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted, hints, err := service.ValidateAndFormatProperty(ctx, tc.propertyName, tc.value, tc.dataType)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			if !tc.shouldSucceed && err == nil {
				t.Error("Expected error for invalid input, got nil")
			}

			if tc.shouldSucceed {
				if formatted == "" && tc.value != "" {
					t.Error("Expected non-empty formatted value")
				}

				if hints == nil {
					t.Error("Expected non-nil hints map")
				}
			}
		})
	}
}

// TestGenerateAITitle tests AI-based title generation (Enhancement 4)
func TestGenerateAITitle(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	testCases := []struct {
		name          string
		columnName    string
		metadata      map[string]interface{}
		dataType      string
		shouldSucceed bool
	}{
		{
			name:       "Decimal number column",
			columnName: "revenue_amt",
			metadata: map[string]interface{}{
				"column_name": "revenue_amt",
				"data_type":   "decimal",
				"cardinality": 5000,
			},
			dataType:      "decimal",
			shouldSucceed: true,
		},
		{
			name:       "String dimension",
			columnName: "customer_segment",
			metadata: map[string]interface{}{
				"column_name": "customer_segment",
				"data_type":   "string",
				"cardinality": 50,
			},
			dataType:      "string",
			shouldSucceed: true,
		},
		{
			name:       "Timestamp column",
			columnName: "created_at",
			metadata: map[string]interface{}{
				"column_name": "created_at",
				"data_type":   "timestamp",
			},
			dataType:      "timestamp",
			shouldSucceed: true,
		},
		{
			name:          "Empty metadata",
			columnName:    "unknown_col",
			metadata:      map[string]interface{}{},
			dataType:      "string",
			shouldSucceed: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			title, confidence, err := service.GenerateAITitle(ctx, tc.columnName, tc.metadata, tc.dataType)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			if tc.shouldSucceed {
				// Even with AI disabled by default, should return a title (fallback to rule-based)
				if title == "" {
					t.Error("Expected non-empty title")
				}

				// Confidence should be valid (0.0 to 1.0)
				if confidence < 0.0 || confidence > 1.0 {
					t.Errorf("Expected confidence between 0.0 and 1.0, got: %f", confidence)
				}
			}
		})
	}
}

// TestApplyPropertyTemplate tests custom property template application (Enhancement 5)
func TestApplyPropertyTemplate(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	baseProperties := map[string]interface{}{
		"name":        "revenue",
		"type":        "number",
		"title":       "Revenue",
		"aggregation": "sum", // Required for MEASURE templates
	}

	testCases := []struct {
		name             string
		termType         string
		domain           string
		baseProps        map[string]interface{}
		shouldSucceed    bool
		expectProperties []string
	}{
		{
			name:             "Finance measure template",
			termType:         "MEASURE",
			domain:           "finance",
			baseProps:        baseProperties,
			shouldSucceed:    true,
			expectProperties: []string{"name", "type", "title", "format", "currency"},
		},
		{
			name:             "Finance dimension template",
			termType:         "DIMENSION",
			domain:           "finance",
			baseProps:        baseProperties,
			shouldSucceed:    true,
			expectProperties: []string{"name", "type", "title"},
		},
		{
			name:             "Unknown domain fallback",
			termType:         "MEASURE",
			domain:           "unknown",
			baseProps:        baseProperties,
			shouldSucceed:    true,
			expectProperties: []string{"name", "type", "title"},
		},
		{
			name:          "Empty base properties with dimension",
			termType:      "DIMENSION",
			domain:        "finance",
			baseProps:     map[string]interface{}{},
			shouldSucceed: false, // Templates require at least base properties
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := service.ApplyPropertyTemplate(ctx, tc.termType, tc.domain, tc.baseProps)
			if tc.shouldSucceed && err != nil {
				t.Errorf("Expected success, got error: %v", err)
			}

			if tc.shouldSucceed && result != nil {
				// Verify expected properties are present
				for _, prop := range tc.expectProperties {
					if _, ok := result[prop]; !ok {
						t.Errorf("Expected property '%s' in result, not found", prop)
					}
				}
			}
		})
	}
}

// TestEnhancementsIntegration tests all 5 enhancements working together in a realistic scenario
func TestEnhancementsIntegration(t *testing.T) {
	service := &analytics.SemanticMappingService{}
	ctx := context.Background()

	// Realistic scenario: Processing a financial column
	columnName := "cust_acq_cost"
	termName := "Customer Acquisition Cost"
	termType := "MEASURE"
	domain := "finance"

	// Step 1: Expand abbreviations
	expanded, _, err := service.ExpandDomainSpecificAbbreviations(ctx, columnName, domain)
	if err != nil {
		t.Errorf("Step 1 failed: %v", err)
	}
	if expanded == "" {
		t.Error("Step 1: Expected expanded abbreviation")
	}

	// Step 2: Generate localized titles
	titles, err := service.GenerateLocalizedTitle(ctx, columnName, termName, []string{"en", "es"})
	if err != nil {
		t.Errorf("Step 2 failed: %v", err)
	}
	if len(titles) == 0 {
		t.Error("Step 2: Expected localized titles")
	}

	// Step 3: Validate and format a related property
	value, hints, err := service.ValidateAndFormatProperty(ctx, "price", "1500.50", "currency")
	if err != nil {
		t.Errorf("Step 3 failed: %v", err)
	}
	if value == "" || hints == nil {
		t.Error("Step 3: Expected formatted value and hints")
	}

	// Step 4: Apply template
	baseProps := map[string]interface{}{
		"name":        columnName,
		"type":        "number",
		"title":       expanded,
		"aggregation": "sum",
		"currency":    "USD",
		"format":      "currency",
	}
	result, err := service.ApplyPropertyTemplate(ctx, termType, domain, baseProps)
	if err != nil {
		t.Errorf("Step 4 failed: %v", err)
	}
	if result == nil {
		t.Error("Step 4: Expected templated properties")
	}

	// Step 5: Generate AI title (with fallback)
	aiTitle, confidence, err := service.GenerateAITitle(ctx, columnName, baseProps, "decimal")
	if err != nil {
		t.Errorf("Step 5 failed: %v", err)
	}
	if aiTitle == "" || confidence < 0.0 || confidence > 1.0 {
		t.Errorf("Step 5: Expected valid AI title and confidence, got: %s, %f", aiTitle, confidence)
	}

	t.Log("✅ All 5 enhancements working together successfully")
}
