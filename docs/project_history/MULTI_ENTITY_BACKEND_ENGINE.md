# Backend Engine: Multi-Entity Validation Implementation

## Overview

This guide provides the backend implementation for multi-entity validation support. The changes enable validation rules to apply across multiple entities simultaneously.

## Architecture

```
Frontend (React)
    ↓ [Multi-entity rule saved]
    ↓
API Handler (/api/validation-rules POST/PATCH)
    ↓
ValidationRulesService
    ↓
ValidationEngine
    ↓
Database Query (uses ANY() operator)
    ↓
Validation Results
```

## Key Changes

### 1. Data Model Updates

**File:** `/backend/models/validation_rule.go`

```go
package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// ValidationRule represents a single validation rule
type ValidationRule struct {
	ID               string                 `json:"id" db:"id"`
	TenantID         string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID     string                 `json:"datasource_id" db:"datasource_id"`
	RuleName         string                 `json:"rule_name" db:"rule_name"`
	RuleType         string                 `json:"rule_type" db:"rule_type"` // field_format, cardinality, uniqueness, referential_integrity, business_logic
	Description      string                 `json:"description" db:"description"`
	TargetEntity     string                 `json:"target_entity" db:"target_entity"` // Legacy: single entity
	TargetEntities   []string               `json:"target_entities" db:"target_entities"` // New: multi-entity support
	ConditionJSON    map[string]interface{} `json:"condition_json" db:"condition_json"`
	Severity         string                 `json:"severity" db:"severity"` // error, warning, info
	IsActive         bool                   `json:"is_active" db:"is_active"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy        string                 `json:"created_by" db:"created_by"`
	UpdatedBy        string                 `json:"updated_by" db:"updated_by"`
	AuditTrail       string                 `json:"audit_trail" db:"audit_trail"`
}

// Scan implements sql.Scanner interface for reading from database
func (vr *ValidationRule) Scan(value interface{}) error {
	bytes, _ := value.([]byte)
	return json.Unmarshal(bytes, &vr)
}

// Value implements driver.Valuer interface for writing to database
func (vr ValidationRule) Value() (driver.Value, error) {
	return json.Marshal(vr)
}
```

### 2. Validation Engine Implementation

**File:** `/backend/internal/engine/validation_engine.go`

```go
package engine

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/lib/pq"
	"yourmodule/models"
)

// ValidationEngine handles validation rule execution
type ValidationEngine struct {
	db *sql.DB
}

// NewValidationEngine creates a new validation engine
func NewValidationEngine(db *sql.DB) *ValidationEngine {
	return &ValidationEngine{db: db}
}

// GetRulesForEntity retrieves all validation rules applicable to a specific entity
func (ve *ValidationEngine) GetRulesForEntity(ctx context.Context, tenantID, datasourceID, entityName string) ([]models.ValidationRule, error) {
	query := `
		SELECT 
			id, tenant_id, datasource_id, rule_name, rule_type, 
			description, target_entity, target_entities, 
			condition_json, severity, is_active, 
			created_at, updated_at, created_by, updated_by, audit_trail
		FROM catalog_validation_rules
		WHERE tenant_id = $1
		  AND datasource_id = $2
		  AND ('global' = ANY(target_entities) OR $3 = ANY(target_entities))
		  AND is_active = true
		ORDER BY 
			CASE severity
				WHEN 'error' THEN 1
				WHEN 'warning' THEN 2
				WHEN 'info' THEN 3
				ELSE 4
			END ASC,
			created_at DESC
	`

	rows, err := ve.db.QueryContext(ctx, query, tenantID, datasourceID, entityName)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules for entity: %w", err)
	}
	defer rows.Close()

	var rules []models.ValidationRule

	for rows.Next() {
		var rule models.ValidationRule
		var conditionJSON []byte
		var targetEntities pq.StringArray

		err := rows.Scan(
			&rule.ID,
			&rule.TenantID,
			&rule.DatasourceID,
			&rule.RuleName,
			&rule.RuleType,
			&rule.Description,
			&rule.TargetEntity,
			&targetEntities,  // PostgreSQL array
			&conditionJSON,   // JSON
			&rule.Severity,
			&rule.IsActive,
			&rule.CreatedAt,
			&rule.UpdatedAt,
			&rule.CreatedBy,
			&rule.UpdatedBy,
			&rule.AuditTrail,
		)
		if err != nil {
			log.Printf("Error scanning rule: %v", err)
			continue
		}

		// Parse condition JSON
		if err := json.Unmarshal(conditionJSON, &rule.ConditionJSON); err != nil {
			log.Printf("Error parsing condition JSON: %v", err)
		}

		// Assign target entities
		rule.TargetEntities = targetEntities

		rules = append(rules, rule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

// ValidateEntity validates all applicable rules for an entity instance
func (ve *ValidationEngine) ValidateEntity(ctx context.Context, tenantID, datasourceID, entityName string, data map[string]interface{}) ([]models.ValidationError, error) {
	rules, err := ve.GetRulesForEntity(ctx, tenantID, datasourceID, entityName)
	if err != nil {
		return nil, err
	}

	var validationErrors []models.ValidationError

	for _, rule := range rules {
		// Apply rule based on type
		switch rule.RuleType {
		case "field_format":
			errors := ve.validateFieldFormat(rule, data)
			validationErrors = append(validationErrors, errors...)

		case "cardinality":
			errors := ve.validateCardinality(rule, data)
			validationErrors = append(validationErrors, errors...)

		case "uniqueness":
			errors := ve.validateUniqueness(ctx, rule, data)
			validationErrors = append(validationErrors, errors...)

		case "referential_integrity":
			errors := ve.validateForeignKey(ctx, rule, data)
			validationErrors = append(validationErrors, errors...)

		case "business_logic":
			errors := ve.validateBusinessLogic(rule, data)
			validationErrors = append(validationErrors, errors...)
		}
	}

	return validationErrors, nil
}

// validateFieldFormat validates field against regex pattern
func (ve *ValidationEngine) validateFieldFormat(rule models.ValidationRule, data map[string]interface{}) []models.ValidationError {
	var errors []models.ValidationError

	fieldName, ok := rule.ConditionJSON["field"].(string)
	if !ok {
		return errors
	}

	pattern, ok := rule.ConditionJSON["pattern"].(string)
	if !ok {
		return errors
	}

	value, exists := data[fieldName]
	if !exists {
		errors = append(errors, models.ValidationError{
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Field:     fieldName,
			Message:   fmt.Sprintf("Field '%s' is missing", fieldName),
			Severity:  rule.Severity,
			Timestamp: time.Now(),
		})
		return errors
	}

	strValue := fmt.Sprintf("%v", value)
	if strValue == "" {
		errors = append(errors, models.ValidationError{
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Field:     fieldName,
			Message:   fmt.Sprintf("Field '%s' cannot be empty", fieldName),
			Severity:  rule.Severity,
			Timestamp: time.Now(),
		})
		return errors
	}

	regex, err := regexp.Compile(pattern)
	if err != nil {
		log.Printf("Invalid regex pattern: %v", err)
		return errors
	}

	if !regex.MatchString(strValue) {
		errors = append(errors, models.ValidationError{
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Field:     fieldName,
			Message:   fmt.Sprintf("Field '%s' with value '%s' does not match pattern '%s'", fieldName, strValue, pattern),
			Severity:  rule.Severity,
			Timestamp: time.Now(),
		})
	}

	return errors
}

// validateCardinality validates field comparison
func (ve *ValidationEngine) validateCardinality(rule models.ValidationRule, data map[string]interface{}) []models.ValidationError {
	var errors []models.ValidationError

	fieldName, ok := rule.ConditionJSON["field"].(string)
	if !ok {
		return errors
	}

	operator, ok := rule.ConditionJSON["operator"].(string)
	if !ok {
		return errors
	}

	expectedValue := rule.ConditionJSON["value"]

	value, exists := data[fieldName]
	if !exists {
		errors = append(errors, models.ValidationError{
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Field:     fieldName,
			Message:   fmt.Sprintf("Field '%s' is missing", fieldName),
			Severity:  rule.Severity,
			Timestamp: time.Now(),
		})
		return errors
	}

	// Convert to numbers for comparison
	actualNum := parseNumber(value)
	expectedNum := parseNumber(expectedValue)

	result := compareNumbers(actualNum, expectedNum, operator)
	if !result {
		errors = append(errors, models.ValidationError{
			RuleID:    rule.ID,
			RuleName:  rule.RuleName,
			Field:     fieldName,
			Message:   fmt.Sprintf("Field '%s' value %v does not satisfy condition %s %v", fieldName, actualNum, operator, expectedNum),
			Severity:  rule.Severity,
			Timestamp: time.Now(),
		})
	}

	return errors
}

// validateUniqueness validates field uniqueness (would require checking database)
func (ve *ValidationEngine) validateUniqueness(ctx context.Context, rule models.ValidationRule, data map[string]interface{}) []models.ValidationError {
	var errors []models.ValidationError
	// Implementation depends on entity and field tracking
	// Would need to query database to check for duplicates
	return errors
}

// validateForeignKey validates referential integrity
func (ve *ValidationEngine) validateForeignKey(ctx context.Context, rule models.ValidationRule, data map[string]interface{}) []models.ValidationError {
	var errors []models.ValidationError

	sourceField, ok := rule.ConditionJSON["source_field"].(string)
	if !ok {
		return errors
	}

	sourceValue, exists := data[sourceField]
	if !exists || sourceValue == nil {
		// NULL values are typically allowed in foreign keys
		return errors
	}

	// In production, this would query the target table
	// For now, just validate the rule structure
	targetEntity, ok := rule.ConditionJSON["target_entity"].(string)
	if !ok {
		return errors
	}

	targetField, ok := rule.ConditionJSON["target_field"].(string)
	if !ok {
		return errors
	}

	// TODO: Query target table to verify reference exists
	log.Printf("FK Check: %s.%s = %v references %s.%s", rule.TargetEntity, sourceField, sourceValue, targetEntity, targetField)

	return errors
}

// validateBusinessLogic validates complex business rules
func (ve *ValidationEngine) validateBusinessLogic(rule models.ValidationRule, data map[string]interface{}) []models.ValidationError {
	var errors []models.ValidationError
	// Implementation depends on specific business logic
	return errors
}

// Helper functions

func parseNumber(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func compareNumbers(a, b float64, operator string) bool {
	switch operator {
	case ">":
		return a > b
	case "<":
		return a < b
	case ">=":
		return a >= b
	case "<=":
		return a <= b
	case "==":
		return a == b
	case "!=":
		return a != b
	default:
		return false
	}
}
```

### 3. Validation Service

**File:** `/backend/internal/services/validation_service.go`

```go
package services

import (
	"context"
	"fmt"
	"log"

	"yourmodule/internal/engine"
	"yourmodule/models"
)

// ValidationService provides validation business logic
type ValidationService struct {
	engine *engine.ValidationEngine
}

// NewValidationService creates a new validation service
func NewValidationService(engine *engine.ValidationEngine) *ValidationService {
	return &ValidationService{engine: engine}
}

// ValidateData validates data against all applicable rules
func (vs *ValidationService) ValidateData(ctx context.Context, tenantID, datasourceID, entityName string, data map[string]interface{}) ([]models.ValidationError, error) {
	log.Printf("Validating %s data for tenant %s", entityName, tenantID)

	errors, err := vs.engine.ValidateEntity(ctx, tenantID, datasourceID, entityName, data)
	if err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return errors, nil
}

// GetRulesForEntity retrieves applicable rules for an entity
func (vs *ValidationService) GetRulesForEntity(ctx context.Context, tenantID, datasourceID, entityName string) ([]models.ValidationRule, error) {
	return vs.engine.GetRulesForEntity(ctx, tenantID, datasourceID, entityName)
}
```

### 4. API Handler Updates

**File:** `/backend/internal/api/validation_handlers.go`

```go
package api

import (
	"encoding/json"
	"log"
	"net/http"

	"yourmodule/internal/services"
	"yourmodule/models"
)

// ValidationHandler handles validation-related HTTP requests
type ValidationHandler struct {
	service *services.ValidationService
}

// NewValidationHandler creates a new validation handler
func NewValidationHandler(service *services.ValidationService) *ValidationHandler {
	return &ValidationHandler{service: service}
}

// ValidateData validates entity data against rules
// POST /api/validate/:entityName
func (h *ValidationHandler) ValidateData(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	entityName := r.PathValue("entityName")

	if tenantID == "" || datasourceID == "" || entityName == "" {
		http.Error(w, "Missing required tenant/entity parameters", http.StatusBadRequest)
		return
	}

	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	errors, err := h.service.ValidateData(r.Context(), tenantID, datasourceID, entityName, data)
	if err != nil {
		log.Printf("Validation error: %v", err)
		http.Error(w, "Validation failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity":  entityName,
		"valid":   len(errors) == 0,
		"errors":  errors,
		"count":   len(errors),
	})
}

// GetRulesForEntity returns rules applicable to an entity
// GET /api/validation-rules?entity=Customer
func (h *ValidationHandler) GetRulesForEntity(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Header.Get("X-Tenant-ID")
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	entityName := r.URL.Query().Get("entity")

	if tenantID == "" || datasourceID == "" || entityName == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	rules, err := h.service.GetRulesForEntity(r.Context(), tenantID, datasourceID, entityName)
	if err != nil {
		log.Printf("Error fetching rules: %v", err)
		http.Error(w, "Failed to fetch rules", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"entity": entityName,
		"count":  len(rules),
		"rules":  rules,
	})
}
```

### 5. Route Registration

**File:** `/backend/internal/api/routes.go`

Add these routes:

```go
// Validation endpoints
validationService := services.NewValidationService(validationEngine)
validationHandler := api.NewValidationHandler(validationService)

router.HandleFunc("POST /api/validate/:entityName", validationHandler.ValidateData)
router.HandleFunc("GET /api/validation-rules", validationHandler.GetRulesForEntity)
```

## Multi-Entity Query Logic

### The Key Query

```sql
SELECT * FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND ('global' = ANY(target_entities) OR $3 = ANY(target_entities))
  AND is_active = true;
```

### How It Works

| Scenario | Query Logic | Result |
|----------|-----------|--------|
| Rule has `target_entities: ['global']` | `'global' = ANY(ARRAY['global'])` → TRUE | Rule applies |
| Rule has `target_entities: ['Customer', 'Employee']` and query entity is `'Customer'` | `'Customer' = ANY(ARRAY['Customer', 'Employee'])` → TRUE | Rule applies |
| Rule has `target_entities: ['Customer']` and query entity is `'Supplier'` | `'Supplier' = ANY(ARRAY['Customer'])` → FALSE | Rule doesn't apply |
| Rule has `target_entities: []` (empty, uses target_entity field) | Falls back to single-entity check | Backward compatible |

## Database Migration SQL

Run these commands to set up the database:

```sql
-- Add target_entities column
ALTER TABLE catalog_validation_rules
ADD COLUMN IF NOT EXISTS target_entities TEXT[] DEFAULT ARRAY['global'];

-- Create index for performance
CREATE INDEX idx_validation_rules_target_entities 
ON catalog_validation_rules USING GIN (target_entities);

-- Backfill existing rules to use their target_entity
UPDATE catalog_validation_rules
SET target_entities = ARRAY[target_entity]
WHERE target_entities IS NULL OR target_entities = ARRAY[]::text[];

-- Verify
SELECT id, rule_name, target_entity, target_entities FROM catalog_validation_rules LIMIT 10;
```

## Testing the Backend

### Test 1: Query Single-Entity Rule

```bash
curl -X GET "http://localhost:29080/api/validation-rules?entity=Customer" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0"
```

**Expected:** Rules with `target_entities: ['Customer']` are returned

### Test 2: Query Multi-Entity Rule

Rules with `target_entities: ['Customer', 'Employee']` returned for both:
- `?entity=Customer`
- `?entity=Employee`

But NOT for:
- `?entity=Supplier`

### Test 3: Query Global Rule

Rules with `target_entities: ['global']` returned for ANY entity query

### Test 4: Validate Data

```bash
curl -X POST "http://localhost:29080/api/validate/Customer" \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "cust-123",
    "phone_number": "invalid",
    "email": "test@example.com"
  }'
```

**Expected:** Validation errors returned for phone_number field

## Performance Optimization

### Index Strategy

```sql
-- GIN index for array containment
CREATE INDEX idx_validation_rules_entities_gin 
ON catalog_validation_rules USING GIN (target_entities);

-- Composite index for common queries
CREATE INDEX idx_validation_rules_lookup 
ON catalog_validation_rules (tenant_id, datasource_id, is_active)
INCLUDE (target_entities);

-- Partial index for active rules only
CREATE INDEX idx_validation_rules_active 
ON catalog_validation_rules (target_entities)
WHERE is_active = true;
```

### Query Analysis

```sql
EXPLAIN ANALYZE
SELECT * FROM catalog_validation_rules
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
  AND datasource_id = '982aef38-418f-46dc-acd0-35fe8f3b97b0'
  AND ('global' = ANY(target_entities) OR 'Customer' = ANY(target_entities))
  AND is_active = true;
```

Expected index usage:
- Sequential scan only if < 1000 rules
- Index scan if > 1000 rules
- Execution time < 10ms

## Error Handling

### Type Assertion Safety

```go
// Safe type assertion with fallback
fieldName, ok := rule.ConditionJSON["field"].(string)
if !ok {
    log.Printf("Field not found or not string in rule %s", rule.ID)
    return errors  // Skip validation
}
```

### Database Connection Safety

```go
func (ve *ValidationEngine) GetRulesForEntity(...) ([]models.ValidationRule, error) {
    if ve.db == nil {
        return nil, fmt.Errorf("database connection not initialized")
    }
    // ... query logic
}
```

## Logging

```go
log.Printf("Validation Engine: Retrieved %d rules for entity %s", len(rules), entityName)
log.Printf("Validation Engine: Applied %d rules, found %d errors", ruleCount, errorCount)
log.Printf("Validation Engine: Query took %dms", duration)
```

## Next Steps

1. ✅ **UI Implementation** - Multi-select and FK picker working
2. ✅ **Database Migration** - Run ALTER TABLE command
3. ⏳ **Backend Engine** - Implement GetRulesForEntity with multi-entity query (this guide)
4. ⏳ **Validation Service** - Wire up validation logic
5. ⏳ **API Handlers** - Add validation endpoints
6. ⏳ **Integration Tests** - Test end-to-end
7. ⏳ **Performance Tests** - Measure with real data
8. ⏳ **Deployment** - Roll out to production
