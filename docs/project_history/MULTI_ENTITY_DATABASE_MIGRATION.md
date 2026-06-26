# Database Migration: Multi-Entity Validation Support

## Overview

This guide walks through adding multi-entity support to the validation rules system.

## Migration Steps

### Step 1: Add `target_entities` Column

**Run this SQL command** in your PostgreSQL database:

```sql
ALTER TABLE catalog_validation_rules
ADD COLUMN IF NOT EXISTS target_entities TEXT[] DEFAULT ARRAY['global'];
```

**Verify it worked:**

```sql
\d catalog_validation_rules

-- Look for the new column in the output:
-- target_entities | text[] | DEFAULT ARRAY['global'::text]
```

### Step 2: Backfill Existing Rules (Optional)

To make existing rules "global" (apply to all entities):

```sql
UPDATE catalog_validation_rules
SET target_entities = ARRAY['global']
WHERE target_entities IS NULL OR target_entities = '{}';
```

Or, to specify each existing rule's target entity:

```sql
UPDATE catalog_validation_rules
SET target_entities = ARRAY[target_entity]
WHERE target_entities IS NULL OR target_entities = '{}';
```

**Verify:**

```sql
SELECT id, rule_name, target_entity, target_entities FROM catalog_validation_rules LIMIT 10;
```

### Step 3: Create Index for Performance (Recommended)

For fast lookups when the rules table is large:

```sql
CREATE INDEX idx_validation_rules_target_entities 
ON catalog_validation_rules USING GIN (target_entities);
```

**Verify:**

```sql
\di catalog_validation_rules*

-- Should show: idx_validation_rules_target_entities
```

### Step 4: Update Backend Query Logic

The backend validation engine needs to query multi-entity rules. See the **Backend Engine** section below.

## Rollback Plan

If you need to undo the changes:

```sql
-- Drop the index
DROP INDEX IF EXISTS idx_validation_rules_target_entities;

-- Remove the new column
ALTER TABLE catalog_validation_rules
DROP COLUMN IF EXISTS target_entities;
```

## Verification Checklist

After running the migration:

- [ ] Column `target_entities` exists in `catalog_validation_rules` table
- [ ] Default value is `ARRAY['global']`
- [ ] Index `idx_validation_rules_target_entities` exists (if created)
- [ ] Backfilled data looks correct (run verification query above)
- [ ] Backend query logic updated (see section below)
- [ ] Frontend form includes multi-select autocomplete
- [ ] Test creating/editing multi-entity rules

## Backend Engine Updates

### Updated Query with Multi-Entity Support

**Current (Single Entity Only):**
```sql
SELECT * FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND target_entity = $3
  AND is_active = true;
```

**New (Multi-Entity Support):**
```sql
SELECT * FROM catalog_validation_rules
WHERE tenant_id = $1
  AND datasource_id = $2
  AND ('global' = ANY(target_entities) OR $3 = ANY(target_entities))
  AND is_active = true
ORDER BY severity DESC, created_at DESC;
```

### Implementation in Go

**File:** `/backend/internal/engine/validation_engine.go`

```go
// GetRulesForEntity retrieves validation rules that apply to a specific entity
func (e *ValidationEngine) GetRulesForEntity(ctx context.Context, tenantID, datasourceID, entityName string) ([]ValidationRule, error) {
	query := `
		SELECT id, rule_name, rule_type, target_entity, target_entities, condition_json, severity, is_active
		FROM catalog_validation_rules
		WHERE tenant_id = $1
		  AND datasource_id = $2
		  AND ('global' = ANY(target_entities) OR $3 = ANY(target_entities))
		  AND is_active = true
		ORDER BY severity DESC, created_at DESC
	`

	rows, err := e.db.QueryContext(ctx, query, tenantID, datasourceID, entityName)
	if err != nil {
		return nil, fmt.Errorf("failed to query rules: %w", err)
	}
	defer rows.Close()

	var rules []ValidationRule
	for rows.Next() {
		var rule ValidationRule
		var targetEntities pq.StringArray

		err := rows.Scan(
			&rule.ID,
			&rule.RuleName,
			&rule.RuleType,
			&rule.TargetEntity,
			&targetEntities,  // Multi-entity array
			&rule.ConditionJSON,
			&rule.Severity,
			&rule.IsActive,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}

		rule.TargetEntities = targetEntities  // Assign array
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}
```

### Updated ValidationRule Struct

**File:** `/backend/models/validation_rule.go`

```go
type ValidationRule struct {
	ID              string            `json:"id" db:"id"`
	TenantID        string            `json:"tenant_id" db:"tenant_id"`
	DatasourceID    string            `json:"datasource_id" db:"datasource_id"`
	RuleName        string            `json:"rule_name" db:"rule_name"`
	RuleType        string            `json:"rule_type" db:"rule_type"`
	Description     string            `json:"description" db:"description"`
	TargetEntity    string            `json:"target_entity" db:"target_entity"`            // Legacy
	TargetEntities  []string          `json:"target_entities" db:"target_entities"`        // New: Multi-entity
	ConditionJSON   map[string]interface{} `json:"condition_json" db:"condition_json"`
	Severity        string            `json:"severity" db:"severity"`
	IsActive        bool              `json:"is_active" db:"is_active"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}
```

### Validation Engine Method

```go
// ValidateEntity validates all applicable rules for an entity instance
func (e *ValidationEngine) ValidateEntity(ctx context.Context, tenantID, datasourceID, entityName string, data map[string]interface{}) ([]ValidationError, error) {
	rules, err := e.GetRulesForEntity(ctx, tenantID, datasourceID, entityName)
	if err != nil {
		return nil, err
	}

	var validationErrors []ValidationError

	for _, rule := range rules {
		// Apply rule logic based on rule type
		switch rule.RuleType {
		case "field_format":
			errors := e.validateFieldFormat(rule, data)
			validationErrors = append(validationErrors, errors...)
		case "referential_integrity":
			errors := e.validateForeignKey(ctx, rule, data)
			validationErrors = append(validationErrors, errors...)
		// ... other rule types
		}
	}

	return validationErrors, nil
}
```

## SQL Query Examples

### Find All Rules for an Entity

```sql
SELECT id, rule_name, rule_type, target_entities, severity
FROM catalog_validation_rules
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
  AND datasource_id = '982aef38-418f-46dc-acd0-35fe8f3b97b0'
  AND ('global' = ANY(target_entities) OR 'Customer' = ANY(target_entities))
  AND is_active = true;
```

### Count Rules by Entity Coverage

```sql
SELECT 
  CASE 
    WHEN 'global' = ANY(target_entities) THEN 'Global'
    ELSE target_entities::text
  END as entity_coverage,
  COUNT(*) as rule_count
FROM catalog_validation_rules
WHERE tenant_id = '910638ba-a459-4a3f-bb2d-78391b0595f6'
  AND is_active = true
GROUP BY entity_coverage;
```

### Find Overlapping Rules

```sql
SELECT r1.id as rule1_id, r1.rule_name as rule1_name,
       r2.id as rule2_id, r2.rule_name as rule2_name,
       r1.target_entities INTERSECT r2.target_entities as common_entities
FROM catalog_validation_rules r1
JOIN catalog_validation_rules r2 ON r1.id < r2.id
WHERE r1.tenant_id = r2.tenant_id
  AND r1.datasource_id = r2.datasource_id
  AND r1.target_entities && r2.target_entities  -- Array overlap operator
  AND r1.is_active = true
  AND r2.is_active = true;
```

## Testing the Migration

### Unit Test Example

**File:** `/backend/tests/validation_rules_test.go`

```go
func TestMultiEntityValidation(t *testing.T) {
	tests := []struct {
		name           string
		targetEntities []string
		queryEntity    string
		shouldMatch    bool
	}{
		{
			name:           "Global rule matches any entity",
			targetEntities: []string{"global"},
			queryEntity:    "Customer",
			shouldMatch:    true,
		},
		{
			name:           "Specific entity rule matches",
			targetEntities: []string{"Customer", "Employee"},
			queryEntity:    "Customer",
			shouldMatch:    true,
		},
		{
			name:           "Specific entity rule doesn't match other entity",
			targetEntities: []string{"Customer"},
			queryEntity:    "Supplier",
			shouldMatch:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules := GetRulesForEntity(tenantID, datasourceID, tt.queryEntity)
			
			matched := false
			for _, rule := range rules {
				if containsEntity(rule.TargetEntities, tt.queryEntity) || contains(rule.TargetEntities, "global") {
					matched = true
					break
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("Expected %v, got %v", tt.shouldMatch, matched)
			}
		})
	}
}
```

## Performance Monitoring

After deployment, monitor:

1. **Query Performance:**
   ```sql
   EXPLAIN ANALYZE SELECT * FROM catalog_validation_rules
   WHERE ('global' = ANY(target_entities) OR 'Customer' = ANY(target_entities));
   ```

2. **Index Usage:**
   ```sql
   SELECT indexname, idx_scan, idx_tup_read, idx_tup_fetch
   FROM pg_stat_user_indexes
   WHERE relname = 'catalog_validation_rules';
   ```

3. **Rule Count:**
   ```sql
   SELECT COUNT(*) FROM catalog_validation_rules WHERE is_active = true;
   ```

## Timeline

| Step | Task | Estimated Time |
|------|------|-----------------|
| 1 | Add column to table | 2 min |
| 2 | Backfill existing rules | 5 min |
| 3 | Create index | 2 min |
| 4 | Update backend query logic | 15 min |
| 5 | Test with frontend | 20 min |
| 6 | Verify data integrity | 10 min |
| 7 | Deploy and monitor | 30 min |

**Total: ~1.5 hours**

## Support

If you encounter issues:

1. Check the PostgreSQL logs for errors
2. Verify the column was created: `\d catalog_validation_rules`
3. Ensure backend engine is updated to use multi-entity query
4. Review test results to confirm behavior matches expectations
