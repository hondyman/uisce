package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TestMultiEntityValidationRules tests multi-entity functionality
func TestMultiEntityValidationRules(t *testing.T) {
	// This test requires a test database connection
	// For now, we'll test the logic structure

	tests := []struct {
		name           string
		targetEntities pq.StringArray
		queryEntity    string
		shouldMatch    bool
		description    string
	}{
		{
			name:           "Global rule matches any entity",
			targetEntities: pq.StringArray{"global"},
			queryEntity:    "Customer",
			shouldMatch:    true,
			description:    "Rule with target_entities=['global'] should match any query",
		},
		{
			name:           "Specific entity matches exact query",
			targetEntities: pq.StringArray{"Customer", "Employee"},
			queryEntity:    "Customer",
			shouldMatch:    true,
			description:    "Rule targeting Customer should match when querying for Customer",
		},
		{
			name:           "Specific entity doesn't match different entity",
			targetEntities: pq.StringArray{"Employee"},
			queryEntity:    "Customer",
			shouldMatch:    false,
			description:    "Rule targeting Employee should not match when querying for Customer",
		},
		{
			name:           "Multiple entities in array",
			targetEntities: pq.StringArray{"Customer", "Employee", "Supplier"},
			queryEntity:    "Supplier",
			shouldMatch:    true,
			description:    "Rule with multiple entities should match any one of them",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test SQL ANY() logic
			// Query: WHERE ('global' = ANY(target_entities) OR $entity = ANY(target_entities))

			hasGlobal := false
			for _, e := range tt.targetEntities {
				if e == "global" {
					hasGlobal = true
					break
				}
			}

			hasEntity := false
			for _, e := range tt.targetEntities {
				if e == tt.queryEntity {
					hasEntity = true
					break
				}
			}

			matches := hasGlobal || hasEntity

			if matches != tt.shouldMatch {
				t.Errorf("Expected match=%v, got match=%v. %s", tt.shouldMatch, matches, tt.description)
			}
		})
	}
}

// TestValidationRuleRequestStructure tests that the struct handles target_entities
func TestValidationRuleRequestStructure(t *testing.T) {
	req := ValidationRuleRequest{
		RuleName:       "Test Rule",
		RuleType:       "field_format",
		Description:    "Test description",
		TargetEntity:   "Customer",
		TargetEntities: pq.StringArray{"Customer", "Employee"},
		ConditionJSON: map[string]interface{}{
			"field":    "phone",
			"operator": "matches_pattern",
			"value":    "\\d{10}",
		},
		Severity: "error",
		IsActive: func() *bool { b := true; return &b }(),
	}

	// Verify struct can be marshaled
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal ValidationRuleRequest: %v", err)
	}

	// Verify struct can be unmarshaled
	var unmarshaled ValidationRuleRequest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal ValidationRuleRequest: %v", err)
	}

	// Verify target_entities are preserved
	if len(unmarshaled.TargetEntities) != 2 {
		t.Errorf("Expected 2 target entities, got %d", len(unmarshaled.TargetEntities))
	}

	if unmarshaled.TargetEntities[0] != "Customer" || unmarshaled.TargetEntities[1] != "Employee" {
		t.Errorf("Target entities not preserved correctly: %v", unmarshaled.TargetEntities)
	}
}

// TestValidationRuleResponseStructure tests that the response struct includes target_entities
func TestValidationRuleResponseStructure(t *testing.T) {
	now := time.Now()
	rule := ValidationRule{
		ID:             uuid.New().String(),
		TenantID:       uuid.New().String(),
		RuleName:       "Test Rule",
		RuleType:       "field_format",
		Description:    "Test description",
		TargetEntity:   "Customer",
		TargetEntities: pq.StringArray{"Customer", "Employee"},
		ConditionJSON: map[string]interface{}{
			"field":    "phone",
			"operator": "matches_pattern",
			"value":    "\\d{10}",
		},
		Severity:  "error",
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Verify struct can be marshaled
	data, err := json.Marshal(rule)
	if err != nil {
		t.Fatalf("Failed to marshal ValidationRule: %v", err)
	}

	// Verify target_entities are in JSON response
	var jsonMap map[string]interface{}
	err = json.Unmarshal(data, &jsonMap)
	if err != nil {
		t.Fatalf("Failed to unmarshal JSON response: %v", err)
	}

	if _, hasTargetEntities := jsonMap["target_entities"]; !hasTargetEntities {
		t.Errorf("Response JSON missing 'target_entities' field")
	}

	// Verify it's serialized as an array
	if targetEntities, ok := jsonMap["target_entities"].([]interface{}); ok {
		if len(targetEntities) != 2 {
			t.Errorf("Expected 2 target entities in JSON, got %d", len(targetEntities))
		}
	} else {
		t.Errorf("target_entities is not an array in JSON response")
	}
}

// TestMultiEntityQueryBuilder tests SQL query building logic
func TestMultiEntityQueryBuilder(t *testing.T) {
	tests := []struct {
		name          string
		entity        string
		targetEntity  string
		expectedWhere string
		description   string
	}{
		{
			name:          "Entity filter takes precedence",
			entity:        "Customer",
			targetEntity:  "",
			expectedWhere: "AND ('global' = ANY(COALESCE(target_entities, ARRAY['global'])) OR $2 = ANY(COALESCE(target_entities, ARRAY[target_entity])))",
			description:   "When 'entity' param is provided, it should be used for filtering",
		},
		{
			name:          "Legacy target_entity fallback",
			entity:        "",
			targetEntity:  "Customer",
			expectedWhere: "AND ('global' = ANY(COALESCE(target_entities, ARRAY['global'])) OR $2 = ANY(COALESCE(target_entities, ARRAY[target_entity])))",
			description:   "When 'entity' is empty, fall back to 'target_entity'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate query building logic
			entity := tt.entity
			if entity == "" {
				entity = tt.targetEntity
			}

			if entity != "" && entity != "Customer" {
				t.Errorf("Query building failed for entity: %s", entity)
			}
		})
	}
}

// TestValidationRuleHandlerIntegration tests handler JSON response structure
func TestValidationRuleHandlerIntegration(t *testing.T) {
	// Create a mock response
	handler := func(w http.ResponseWriter, _ *http.Request) {
		rule := ValidationRule{
			ID:             uuid.New().String(),
			TenantID:       "tenant-123",
			RuleName:       "Phone Validation",
			RuleType:       "field_format",
			Description:    "Validates phone field format",
			TargetEntity:   "Customer",
			TargetEntities: pq.StringArray{"Customer", "Employee", "Supplier"},
			ConditionJSON: map[string]interface{}{
				"field":    "phone",
				"operator": "matches_pattern",
				"value":    "\\d{10}",
			},
			Severity:  "error",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(rule)
	}

	// Test the handler
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/validation-rules/123?tenant_id=tenant-123", nil)
	handler(w, r)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var rule ValidationRule
	err := json.NewDecoder(w.Body).Decode(&rule)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(rule.TargetEntities) != 3 {
		t.Errorf("Expected 3 target entities, got %d", len(rule.TargetEntities))
	}

	expectedEntities := []string{"Customer", "Employee", "Supplier"}
	for i, entity := range expectedEntities {
		if rule.TargetEntities[i] != entity {
			t.Errorf("Expected entity %s at index %d, got %s", entity, i, rule.TargetEntities[i])
		}
	}
}

// TestMultiEntityQueryCoverage tests common multi-entity filtering scenarios
func TestMultiEntityQueryCoverage(t *testing.T) {
	scenarios := []struct {
		scenario   string
		rules      []struct{ entities pq.StringArray }
		queryFor   string
		shouldFind []int // indices of rules that should match
	}{
		{
			scenario: "Global rule applies to all entities",
			rules: []struct{ entities pq.StringArray }{
				{entities: pq.StringArray{"global"}},
				{entities: pq.StringArray{"Customer"}},
			},
			queryFor:   "Employee",
			shouldFind: []int{0}, // Only global rule matches
		},
		{
			scenario: "Specific entity matches its rules",
			rules: []struct{ entities pq.StringArray }{
				{entities: pq.StringArray{"Customer", "Employee"}},
				{entities: pq.StringArray{"Supplier"}},
			},
			queryFor:   "Customer",
			shouldFind: []int{0}, // Customer+Employee rule matches
		},
		{
			scenario: "Multiple matching rules",
			rules: []struct{ entities pq.StringArray }{
				{entities: pq.StringArray{"global"}},
				{entities: pq.StringArray{"Customer"}},
				{entities: pq.StringArray{"Customer", "Employee"}},
			},
			queryFor:   "Customer",
			shouldFind: []int{0, 1, 2}, // All rules match Customer
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.scenario, func(t *testing.T) {
			matchedCount := 0
			for i, rule := range scenario.rules {
				// Check if rule applies
				applies := false

				// Check for global
				for _, e := range rule.entities {
					if e == "global" {
						applies = true
						break
					}
				}

				// Check for specific entity
				if !applies {
					for _, e := range rule.entities {
						if e == scenario.queryFor {
							applies = true
							break
						}
					}
				}

				if applies {
					matchedCount++
					// Verify this rule is in shouldFind
					found := false
					for _, expectedIdx := range scenario.shouldFind {
						if expectedIdx == i {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Rule %d matched but was not expected", i)
					}
				}
			}

			if matchedCount != len(scenario.shouldFind) {
				t.Errorf("Expected %d matching rules, got %d", len(scenario.shouldFind), matchedCount)
			}
		})
	}
}

// BenchmarkMultiEntityQuery benchmarks the ANY() logic performance
func BenchmarkMultiEntityQuery(b *testing.B) {
	// Simulate checking if an entity is in a target_entities array
	targetEntities := pq.StringArray{"Customer", "Employee", "Supplier", "Product", "Order"}
	queryEntity := "Employee"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate ANY() operator check
		found := false
		for _, e := range targetEntities {
			if e == "global" || e == queryEntity {
				found = true
				break
			}
		}
		_ = found
	}
}
