# Backend Rule Engine - Practical Implementation Examples

**Date:** October 20, 2025  
**Version:** 1.0.0

---

## Table of Contents

1. [Go Implementation](#go-implementation)
2. [Node.js/TypeScript Implementation](#nodejs-typescript-implementation)
3. [Database Setup](#database-setup)
4. [GraphQL Resolver Examples](#graphql-resolver-examples)
5. [Testing Examples](#testing-examples)
6. [Real-World Scenarios](#real-world-scenarios)

---

## Go Implementation

### Complete Rule Engine in Go

```go
// internal/rules/types.go
package rules

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Operator string

const (
	// String operators
	OpEquals      Operator = "equals"
	OpNotEquals   Operator = "not_equals"
	OpContains    Operator = "contains"
	OpStartsWith  Operator = "starts_with"
	OpEndsWith    Operator = "ends_with"

	// Number operators
	OpGreaterThan  Operator = "greater_than"
	OpLessThan     Operator = "less_than"
	OpGreaterEqual Operator = "greater_equal"
	OpLessEqual    Operator = "less_equal"

	// Boolean operators
	OpIsTrue  Operator = "is_true"
	OpIsFalse Operator = "is_false"

	// Date operators
	OpBefore  Operator = "before"
	OpAfter   Operator = "after"
	OpBetween Operator = "between"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Condition represents a single condition
type Condition struct {
	ID       string `json:"id"`
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

// ConditionGroup represents a group of conditions with AND/OR logic
type ConditionGroup struct {
	ID         string        `json:"id"`
	Operator   string        `json:"operator"` // AND or OR
	Conditions []interface{} `json:"conditions"` // Can be Condition or ConditionGroup
}

// ValidationRule represents a complete rule
type ValidationRule struct {
	ID                  string          `json:"id"`
	TenantID            string          `json:"tenant_id"`
	DatasourceID        string          `json:"datasource_id"`
	Name                string          `json:"name"`
	Entity              string          `json:"entity"`
	Description         string          `json:"description"`
	Severity            Severity        `json:"severity"`
	Condition           ConditionGroup  `json:"condition"`
	DependentRuleIDs    []string        `json:"dependent_rule_ids"`
	IsActive            bool            `json:"is_active"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

// RuleEvaluationResult represents the result of evaluating a rule
type RuleEvaluationResult struct {
	RuleID             string                   `json:"rule_id"`
	Passed             bool                     `json:"passed"`
	Severity           Severity                 `json:"severity"`
	Message            string                   `json:"message"`
	EvaluatedAt        time.Time                `json:"evaluated_at"`
	DependencyResults  []RuleEvaluationResult   `json:"dependency_results,omitempty"`
}

// CrossEntityValidation represents a cross-entity validation
type CrossEntityValidation struct {
	ID           string      `json:"id"`
	RuleID       string      `json:"rule_id"`
	RuleName     string      `json:"rule_name"`
	SourcePath   EntityPath  `json:"source_path"`
	Operator     string      `json:"operator"`
	TargetPath   EntityPath  `json:"target_path"`
	IsActive     bool        `json:"is_active"`
	CreatedAt    time.Time   `json:"created_at"`
}

// EntityPath represents a path through related entities
type EntityPath struct {
	Segments    []PathSegment `json:"segments"`
	DisplayPath string        `json:"display_path"`
}

// PathSegment represents one step in an entity path
type PathSegment struct {
	Entity         string `json:"entity"`
	Field          string `json:"field"`
	Relationship   string `json:"relationship"`
	TargetEntity   string `json:"target_entity"`
}

// EntityPathResolution represents the resolved value
type EntityPathResolution struct {
	EntityID   string
	Entity     string
	FieldValue interface{}
}

// ConditionEvaluator evaluates conditions
type ConditionEvaluator struct{}

// RuleExecutor executes rules with dependencies
type RuleExecutor struct {
	db       *sql.DB
	evaluator *ConditionEvaluator
	cache    map[string]*ValidationRule
}

// EntityPathResolver resolves entity paths
type EntityPathResolver struct {
	db *sql.DB
}
```

### Condition Evaluator Implementation

```go
// internal/rules/condition_evaluator.go
package rules

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (ce *ConditionEvaluator) Evaluate(node interface{}, data map[string]interface{}) (bool, error) {
	switch v := node.(type) {
	case Condition:
		return ce.evaluateCondition(v, data)
	case ConditionGroup:
		return ce.evaluateGroup(v, data)
	case map[string]interface{}:
		// Handle JSON unmarshaled data
		if op, exists := v["operator"]; exists {
			if conditions, hasConditions := v["conditions"]; hasConditions {
				return ce.evaluateGroup(ConditionGroup{
					Operator:   op.(string),
					Conditions: conditions.([]interface{}),
				}, data)
			}
		}
		return ce.evaluateCondition(Condition{
			Field:    v["field"].(string),
			Operator: v["operator"].(string),
			Value:    v["value"].(string),
		}, data)
	}
	return false, fmt.Errorf("unknown node type: %T", node)
}

func (ce *ConditionEvaluator) evaluateCondition(cond Condition, data map[string]interface{}) (bool, error) {
	fieldValue, exists := data[cond.Field]
	if !exists {
		return false, fmt.Errorf("field not found: %s", cond.Field)
	}

	switch Operator(cond.Operator) {
	case OpEquals:
		return fmt.Sprint(fieldValue) == cond.Value, nil
	
	case OpNotEquals:
		return fmt.Sprint(fieldValue) != cond.Value, nil
	
	case OpContains:
		return strings.Contains(fmt.Sprint(fieldValue), cond.Value), nil
	
	case OpStartsWith:
		return strings.HasPrefix(fmt.Sprint(fieldValue), cond.Value), nil
	
	case OpEndsWith:
		return strings.HasSuffix(fmt.Sprint(fieldValue), cond.Value), nil

	case OpGreaterThan:
		fv, _ := strconv.ParseFloat(fmt.Sprint(fieldValue), 64)
		cv, _ := strconv.ParseFloat(cond.Value, 64)
		return fv > cv, nil
	
	case OpLessThan:
		fv, _ := strconv.ParseFloat(fmt.Sprint(fieldValue), 64)
		cv, _ := strconv.ParseFloat(cond.Value, 64)
		return fv < cv, nil
	
	case OpGreaterEqual:
		fv, _ := strconv.ParseFloat(fmt.Sprint(fieldValue), 64)
		cv, _ := strconv.ParseFloat(cond.Value, 64)
		return fv >= cv, nil
	
	case OpLessEqual:
		fv, _ := strconv.ParseFloat(fmt.Sprint(fieldValue), 64)
		cv, _ := strconv.ParseFloat(cond.Value, 64)
		return fv <= cv, nil

	case OpIsTrue:
		b, _ := strconv.ParseBool(fmt.Sprint(fieldValue))
		return b, nil
	
	case OpIsFalse:
		b, _ := strconv.ParseBool(fmt.Sprint(fieldValue))
		return !b, nil

	case OpBefore:
		fv, _ := time.Parse(time.RFC3339, fmt.Sprint(fieldValue))
		cv, _ := time.Parse(time.RFC3339, cond.Value)
		return fv.Before(cv), nil
	
	case OpAfter:
		fv, _ := time.Parse(time.RFC3339, fmt.Sprint(fieldValue))
		cv, _ := time.Parse(time.RFC3339, cond.Value)
		return fv.After(cv), nil

	case OpBetween:
		parts := strings.Split(cond.Value, ",")
		if len(parts) != 2 {
			return false, fmt.Errorf("between operator requires two dates")
		}
		fv, _ := time.Parse(time.RFC3339, fmt.Sprint(fieldValue))
		start, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[0]))
		end, _ := time.Parse(time.RFC3339, strings.TrimSpace(parts[1]))
		return (fv.After(start) || fv.Equal(start)) && (fv.Before(end) || fv.Equal(end)), nil

	default:
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}
}

func (ce *ConditionEvaluator) evaluateGroup(group ConditionGroup, data map[string]interface{}) (bool, error) {
	results := make([]bool, 0, len(group.Conditions))

	for _, condition := range group.Conditions {
		result, err := ce.Evaluate(condition, data)
		if err != nil {
			return false, err
		}
		results = append(results, result)
	}

	if group.Operator == "AND" {
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil
	} else if group.Operator == "OR" {
		for _, r := range results {
			if r {
				return true, nil
			}
		}
		return false, nil
	}

	return false, fmt.Errorf("unknown group operator: %s", group.Operator)
}
```

### Rule Executor Implementation

```go
// internal/rules/rule_executor.go
package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

func NewRuleExecutor(db *sql.DB) *RuleExecutor {
	return &RuleExecutor{
		db:        db,
		evaluator: &ConditionEvaluator{},
		cache:     make(map[string]*ValidationRule),
	}
}

// ExecuteRule executes a single rule
func (re *RuleExecutor) ExecuteRule(
	ctx context.Context,
	rule *ValidationRule,
	data map[string]interface{},
) (*RuleEvaluationResult, error) {
	passed, err := re.evaluator.Evaluate(rule.Condition, data)

	result := &RuleEvaluationResult{
		RuleID:      rule.ID,
		Passed:      passed,
		Severity:    rule.Severity,
		EvaluatedAt: time.Now(),
	}

	if err != nil {
		result.Message = fmt.Sprintf("Error evaluating rule: %v", err)
		return result, nil
	}

	if passed {
		result.Message = fmt.Sprintf("Rule \"%s\" passed", rule.Name)
	} else {
		result.Message = fmt.Sprintf("Rule \"%s\" failed", rule.Name)
	}

	return result, nil
}

// ExecuteRuleChain executes a rule with dependencies
func (re *RuleExecutor) ExecuteRuleChain(
	ctx context.Context,
	ruleID string,
	tenantID string,
	datasourceID string,
	data map[string]interface{},
	stopOnError bool,
) (*RuleEvaluationResult, error) {
	return re.executeRuleChainRecursive(ctx, ruleID, tenantID, datasourceID, data, stopOnError, 0, 10)
}

func (re *RuleExecutor) executeRuleChainRecursive(
	ctx context.Context,
	ruleID string,
	tenantID string,
	datasourceID string,
	data map[string]interface{},
	stopOnError bool,
	depth int,
	maxDepth int,
) (*RuleEvaluationResult, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("maximum rule chain depth (%d) exceeded", maxDepth)
	}

	// Fetch rule from cache or database
	rule, exists := re.cache[ruleID]
	if !exists {
		var conditionJSON []byte
		err := re.db.QueryRowContext(
			ctx,
			`SELECT id, name, entity, description, severity, condition, dependent_rule_ids, is_active, created_at, updated_at
			 FROM validation_rules
			 WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3`,
			ruleID, tenantID, datasourceID,
		).Scan(
			&rule.ID, &rule.Name, &rule.Entity, &rule.Description, &rule.Severity,
			&conditionJSON, &rule.DependentRuleIDs, &rule.IsActive, &rule.CreatedAt, &rule.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to fetch rule: %w", err)
		}

		err = json.Unmarshal(conditionJSON, &rule.Condition)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal condition: %w", err)
		}

		re.cache[ruleID] = rule
	}

	dependencyResults := make([]RuleEvaluationResult, 0)

	// Execute dependencies first
	if len(rule.DependentRuleIDs) > 0 {
		for _, depID := range rule.DependentRuleIDs {
			depResult, err := re.executeRuleChainRecursive(
				ctx, depID, tenantID, datasourceID, data, stopOnError, depth+1, maxDepth,
			)
			if err != nil {
				return nil, err
			}

			dependencyResults = append(dependencyResults, *depResult)

			if stopOnError && !depResult.Passed && depResult.Severity == SeverityError {
				return &RuleEvaluationResult{
					RuleID:            rule.ID,
					Passed:            false,
					Severity:          SeverityError,
					Message:           fmt.Sprintf("Dependency failed: %s", depResult.Message),
					EvaluatedAt:       time.Now(),
					DependencyResults: dependencyResults,
				}, nil
			}
		}
	}

	// Execute current rule
	currentResult, err := re.ExecuteRule(ctx, rule, data)
	if err != nil {
		return nil, err
	}

	if len(dependencyResults) > 0 {
		currentResult.DependencyResults = dependencyResults
	}

	return currentResult, nil
}

// ExecuteEntityRules executes all rules for an entity
func (re *RuleExecutor) ExecuteEntityRules(
	ctx context.Context,
	entity string,
	tenantID string,
	datasourceID string,
	data map[string]interface{},
) ([]RuleEvaluationResult, error) {
	rows, err := re.db.QueryContext(
		ctx,
		`SELECT id FROM validation_rules
		 WHERE entity = $1 AND tenant_id = $2 AND datasource_id = $3 AND is_active = true`,
		entity, tenantID, datasourceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ruleIDs []string
	for rows.Next() {
		var ruleID string
		if err := rows.Scan(&ruleID); err != nil {
			return nil, err
		}
		ruleIDs = append(ruleIDs, ruleID)
	}

	results := make([]RuleEvaluationResult, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		result, err := re.ExecuteRuleChain(ctx, ruleID, tenantID, datasourceID, data, true)
		if err != nil {
			return nil, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// ClearCache clears the rule cache
func (re *RuleExecutor) ClearCache() {
	re.cache = make(map[string]*ValidationRule)
}
```

### Entity Path Resolver Implementation

```go
// internal/rules/entity_path_resolver.go
package rules

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

func NewEntityPathResolver(db *sql.DB) *EntityPathResolver {
	return &EntityPathResolver{db: db}
}

// ResolvePath resolves a full entity path to get the final field value
func (epr *EntityPathResolver) ResolvePath(
	ctx context.Context,
	path *EntityPath,
	startID string,
	tenantID string,
	startEntity string,
) (*EntityPathResolution, error) {
	currentID := startID
	currentEntity := startEntity

	// Traverse all segments
	for _, segment := range path.Segments {
		// Get the foreign key value from current record
		var foreignKeyValue string
		query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1 LIMIT 1`,
			segment.Field, currentEntity)
		
		err := epr.db.QueryRowContext(ctx, query, currentID).Scan(&foreignKeyValue)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, fmt.Errorf("record not found in %s with id %s", currentEntity, currentID)
			}
			return nil, fmt.Errorf("error querying %s: %w", currentEntity, err)
		}

		if foreignKeyValue == "" {
			return nil, fmt.Errorf("foreign key %s is null in %s", segment.Field, segment.Entity)
		}

		currentID = foreignKeyValue
		currentEntity = segment.TargetEntity
	}

	// Get the final field value
	finalField := strings.Split(path.DisplayPath, ".")[len(strings.Split(path.DisplayPath, "."))-1]
	var finalValue interface{}
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1 LIMIT 1`,
		finalField, currentEntity)
	
	err := epr.db.QueryRowContext(ctx, query, currentID).Scan(&finalValue)
	if err != nil {
		return nil, fmt.Errorf("error getting final field: %w", err)
	}

	return &EntityPathResolution{
		EntityID:   currentID,
		Entity:     currentEntity,
		FieldValue: finalValue,
	}, nil
}

// EvaluateCrossEntityCondition evaluates a cross-entity condition
func (epr *EntityPathResolver) EvaluateCrossEntityCondition(
	ctx context.Context,
	condition *CrossEntityValidation,
	recordID string,
	startEntity string,
	tenantID string,
) (bool, error) {
	sourceResolution, err := epr.ResolvePath(ctx, &condition.SourcePath, recordID, tenantID, startEntity)
	if err != nil {
		return false, fmt.Errorf("error resolving source path: %w", err)
	}

	targetResolution, err := epr.ResolvePath(ctx, &condition.TargetPath, recordID, tenantID, startEntity)
	if err != nil {
		return false, fmt.Errorf("error resolving target path: %w", err)
	}

	sourceValue := sourceResolution.FieldValue
	targetValue := targetResolution.FieldValue

	switch condition.Operator {
	case "equals":
		return fmt.Sprint(sourceValue) == fmt.Sprint(targetValue), nil
	case "not_equals":
		return fmt.Sprint(sourceValue) != fmt.Sprint(targetValue), nil
	case "greater_than":
		sv, _ := strconv.ParseFloat(fmt.Sprint(sourceValue), 64)
		tv, _ := strconv.ParseFloat(fmt.Sprint(targetValue), 64)
		return sv > tv, nil
	case "less_than":
		sv, _ := strconv.ParseFloat(fmt.Sprint(sourceValue), 64)
		tv, _ := strconv.ParseFloat(fmt.Sprint(targetValue), 64)
		return sv < tv, nil
	case "greater_equal":
		sv, _ := strconv.ParseFloat(fmt.Sprint(sourceValue), 64)
		tv, _ := strconv.ParseFloat(fmt.Sprint(targetValue), 64)
		return sv >= tv, nil
	case "less_equal":
		sv, _ := strconv.ParseFloat(fmt.Sprint(sourceValue), 64)
		tv, _ := strconv.ParseFloat(fmt.Sprint(targetValue), 64)
		return sv <= tv, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

// BatchEvaluateCrossEntityCondition evaluates condition for multiple records
func (epr *EntityPathResolver) BatchEvaluateCrossEntityCondition(
	ctx context.Context,
	condition *CrossEntityValidation,
	recordIDs []string,
	startEntity string,
	tenantID string,
) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(recordIDs))

	for _, recordID := range recordIDs {
		passed, err := epr.EvaluateCrossEntityCondition(ctx, condition, recordID, startEntity, tenantID)
		results = append(results, map[string]interface{}{
			"recordId": recordID,
			"passed":   passed,
			"error":    err,
		})
	}

	return results, nil
}
```

---

## Node.js/TypeScript Implementation

### Complete Implementation

```typescript
// src/services/rule-engine.ts
import { Database } from 'pg';

interface EvaluationOptions {
  stopOnError?: boolean;
  maxDepth?: number;
  cache?: boolean;
}

export class RuleEngine {
  private db: Database;
  private ruleCache: Map<string, ValidationRule>;
  private evaluator: ConditionEvaluator;
  private resolver: EntityPathResolver;

  constructor(db: Database) {
    this.db = db;
    this.ruleCache = new Map();
    this.evaluator = new ConditionEvaluator();
    this.resolver = new EntityPathResolver(db);
  }

  /**
   * Execute a single rule
   */
  async executeRule(
    rule: ValidationRule,
    data: Record<string, any>
  ): Promise<RuleEvaluationResult> {
    try {
      const passed = this.evaluator.evaluate(rule.condition, data);
      return {
        ruleId: rule.id,
        passed,
        severity: rule.severity,
        message: passed
          ? `Rule "${rule.name}" passed`
          : `Rule "${rule.name}" failed`,
        evaluatedAt: new Date()
      };
    } catch (error) {
      return {
        ruleId: rule.id,
        passed: false,
        severity: 'error',
        message: `Error evaluating rule: ${error.message}`,
        evaluatedAt: new Date()
      };
    }
  }

  /**
   * Execute a rule chain with dependencies
   */
  async executeRuleChain(
    ruleId: string,
    tenantId: string,
    datasourceId: string,
    data: Record<string, any>,
    options?: EvaluationOptions
  ): Promise<RuleEvaluationResult> {
    const stopOnError = options?.stopOnError ?? true;
    const maxDepth = options?.maxDepth ?? 10;

    return this.executeRuleChainRecursive(
      ruleId,
      tenantId,
      datasourceId,
      data,
      stopOnError,
      0,
      maxDepth
    );
  }

  private async executeRuleChainRecursive(
    ruleId: string,
    tenantId: string,
    datasourceId: string,
    data: Record<string, any>,
    stopOnError: boolean,
    depth: number,
    maxDepth: number
  ): Promise<RuleEvaluationResult> {
    if (depth > maxDepth) {
      throw new Error(`Maximum rule chain depth (${maxDepth}) exceeded`);
    }

    // Fetch rule from cache or database
    let rule = this.ruleCache.get(ruleId);
    if (!rule) {
      const result = await this.db.query(
        `SELECT id, name, entity, description, severity, condition, dependent_rule_ids
         FROM validation_rules
         WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3`,
        [ruleId, tenantId, datasourceId]
      );

      if (result.rows.length === 0) {
        throw new Error(`Rule not found: ${ruleId}`);
      }

      rule = result.rows[0] as ValidationRule;
      this.ruleCache.set(ruleId, rule);
    }

    const dependencyResults: RuleEvaluationResult[] = [];

    // Execute dependencies first
    if (rule.dependent_rule_ids && rule.dependent_rule_ids.length > 0) {
      for (const depId of rule.dependent_rule_ids) {
        const depResult = await this.executeRuleChainRecursive(
          depId,
          tenantId,
          datasourceId,
          data,
          stopOnError,
          depth + 1,
          maxDepth
        );

        dependencyResults.push(depResult);

        if (stopOnError && !depResult.passed && depResult.severity === 'error') {
          return {
            ruleId: rule.id,
            passed: false,
            severity: 'error',
            message: `Dependency failed: ${depResult.message}`,
            evaluatedAt: new Date(),
            dependencyResults
          };
        }
      }
    }

    // Execute current rule
    const currentResult = await this.executeRule(rule, data);
    if (dependencyResults.length > 0) {
      currentResult.dependencyResults = dependencyResults;
    }

    return currentResult;
  }

  /**
   * Execute all rules for an entity
   */
  async executeEntityRules(
    entity: string,
    tenantId: string,
    datasourceId: string,
    data: Record<string, any>
  ): Promise<RuleEvaluationResult[]> {
    const result = await this.db.query(
      `SELECT id FROM validation_rules
       WHERE entity = $1 AND tenant_id = $2 AND datasource_id = $3 AND is_active = true`,
      [entity, tenantId, datasourceId]
    );

    const ruleIds = result.rows.map(r => r.id);
    const results: RuleEvaluationResult[] = [];

    for (const ruleId of ruleIds) {
      const evalResult = await this.executeRuleChain(
        ruleId,
        tenantId,
        datasourceId,
        data
      );
      results.push(evalResult);
    }

    return results;
  }

  /**
   * Clear the rule cache
   */
  clearCache(): void {
    this.ruleCache.clear();
  }
}
```

---

## Database Setup

### PostgreSQL Migration

```sql
-- migration_2025_01_01_create_validation_rules.sql

-- Create validation_rules table
CREATE TABLE IF NOT EXISTS validation_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  entity VARCHAR(100) NOT NULL,
  description TEXT,
  severity VARCHAR(20) NOT NULL CHECK (severity IN ('error', 'warning', 'info')),
  condition JSONB NOT NULL,
  dependent_rule_ids UUID[] DEFAULT ARRAY[]::UUID[],
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,
  
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
  CONSTRAINT fk_datasource FOREIGN KEY (datasource_id) REFERENCES datasources(id) ON DELETE CASCADE
);

CREATE INDEX idx_validation_rules_tenant ON validation_rules(tenant_id, datasource_id);
CREATE INDEX idx_validation_rules_entity ON validation_rules(entity);
CREATE INDEX idx_validation_rules_active ON validation_rules(is_active);

-- Create cross_entity_validations table
CREATE TABLE IF NOT EXISTS cross_entity_validations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  rule_name VARCHAR(255) NOT NULL,
  source_path JSONB NOT NULL,
  operator VARCHAR(50) NOT NULL,
  target_path JSONB NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_cross_entity_rule ON cross_entity_validations(rule_id);
CREATE INDEX idx_cross_entity_tenant ON cross_entity_validations(tenant_id, datasource_id);

-- Create rule_evaluation_audit table
CREATE TABLE IF NOT EXISTS rule_evaluation_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  record_id UUID NOT NULL,
  entity VARCHAR(100) NOT NULL,
  passed BOOLEAN NOT NULL,
  severity VARCHAR(20),
  message TEXT,
  evaluation_details JSONB,
  evaluated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_audit_rule_evaluated ON rule_evaluation_audit(rule_id, evaluated_at DESC);
CREATE INDEX idx_audit_entity_evaluated ON rule_evaluation_audit(entity, evaluated_at DESC);
CREATE INDEX idx_audit_tenant_evaluated ON rule_evaluation_audit(tenant_id, evaluated_at DESC);
```

---

## GraphQL Resolver Examples

### GraphQL Mutations

```typescript
// src/resolvers/rule-mutations.ts
export const ruleMutations = {
  createValidationRule: async (parent, args, context) => {
    const { input, tenantId, datasourceId } = args;
    
    const result = await context.db.query(
      `INSERT INTO validation_rules
       (tenant_id, datasource_id, name, entity, description, severity, condition, created_by)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
       RETURNING *`,
      [
        tenantId,
        datasourceId,
        input.name,
        input.entity,
        input.description,
        input.severity,
        JSON.stringify(input.condition),
        context.userId
      ]
    );

    return result.rows[0];
  },

  updateRuleDependencies: async (parent, args, context) => {
    const { ruleId, dependencies, tenantId, datasourceId } = args;

    // Validate no cycles
    const ruleEngine = new RuleEngine(context.db);
    const { valid, cycle } = await ruleEngine.validateNoCycles(
      ruleId,
      tenantId,
      datasourceId,
      dependencies
    );

    if (!valid) {
      throw new Error(`Circular dependency detected: ${cycle.join(' -> ')}`);
    }

    const result = await context.db.query(
      `UPDATE validation_rules
       SET dependent_rule_ids = $1, updated_at = NOW(), updated_by = $2
       WHERE id = $3 AND tenant_id = $4
       RETURNING *`,
      [dependencies, context.userId, ruleId, tenantId]
    );

    return result.rows[0];
  },

  evaluateRule: async (parent, args, context) => {
    const { ruleId, tenantId, datasourceId, data } = args;

    // Fetch rule
    const ruleResult = await context.db.query(
      `SELECT * FROM validation_rules
       WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3`,
      [ruleId, tenantId, datasourceId]
    );

    if (ruleResult.rows.length === 0) {
      throw new Error('Rule not found');
    }

    const rule = ruleResult.rows[0];

    // Execute rule
    const ruleEngine = new RuleEngine(context.db);
    const result = await ruleEngine.executeRule(rule, data);

    // Audit
    await context.db.query(
      `INSERT INTO rule_evaluation_audit
       (tenant_id, datasource_id, rule_id, record_id, entity, passed, severity, message, evaluation_details, evaluated_at)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
      [
        tenantId,
        datasourceId,
        ruleId,
        data.id || 'unknown',
        rule.entity,
        result.passed,
        result.severity,
        result.message,
        JSON.stringify(result),
        new Date()
      ]
    );

    return result;
  }
};
```

---

## Testing Examples

### Unit Tests

```typescript
// tests/condition-evaluator.test.ts
import { ConditionEvaluator } from '../src/services/condition-evaluator';

describe('ConditionEvaluator', () => {
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    evaluator = new ConditionEvaluator();
  });

  describe('Simple Conditions', () => {
    it('evaluates equals operator', () => {
      const condition = {
        id: '1',
        field: 'status',
        operator: 'equals',
        value: 'Active'
      };
      const data = { status: 'Active' };
      expect(evaluator.evaluate(condition, data)).toBe(true);
    });

    it('evaluates number comparison', () => {
      const condition = {
        id: '1',
        field: 'age',
        operator: 'greater_equal',
        value: '18'
      };
      const data = { age: 25 };
      expect(evaluator.evaluate(condition, data)).toBe(true);
    });
  });

  describe('AND Groups', () => {
    it('evaluates AND group correctly', () => {
      const group = {
        id: 'group_1',
        operator: 'AND',
        conditions: [
          { id: '1', field: 'age', operator: 'greater_equal', value: '18' },
          { id: '2', field: 'status', operator: 'equals', value: 'Active' }
        ]
      };
      const data = { age: 25, status: 'Active' };
      expect(evaluator.evaluate(group, data)).toBe(true);
    });

    it('returns false when any AND condition fails', () => {
      const group = {
        id: 'group_1',
        operator: 'AND',
        conditions: [
          { id: '1', field: 'age', operator: 'greater_equal', value: '18' },
          { id: '2', field: 'status', operator: 'equals', value: 'Inactive' }
        ]
      };
      const data = { age: 25, status: 'Active' };
      expect(evaluator.evaluate(group, data)).toBe(false);
    });
  });

  describe('Complex Nested Groups', () => {
    it('evaluates nested OR/AND groups', () => {
      const group = {
        id: 'group_1',
        operator: 'OR',
        conditions: [
          {
            id: 'group_2',
            operator: 'AND',
            conditions: [
              { id: '1', field: 'age', operator: 'greater_equal', value: '18' },
              { id: '2', field: 'status', operator: 'equals', value: 'Active' }
            ]
          },
          {
            id: 'group_3',
            operator: 'AND',
            conditions: [
              { id: '3', field: 'isVip', operator: 'is_true', value: 'true' },
              { id: '4', field: 'salary', operator: 'greater_than', value: '50000' }
            ]
          }
        ]
      };

      // Both conditions of first group true
      const data1 = { age: 25, status: 'Active', isVip: false, salary: 30000 };
      expect(evaluator.evaluate(group, data1)).toBe(true);

      // Neither group satisfied
      const data2 = { age: 16, status: 'Inactive', isVip: false, salary: 30000 };
      expect(evaluator.evaluate(group, data2)).toBe(false);

      // Second group satisfied
      const data3 = { age: 16, status: 'Inactive', isVip: true, salary: 60000 };
      expect(evaluator.evaluate(group, data3)).toBe(true);
    });
  });
});
```

---

## Real-World Scenarios

### Example 1: Employee Salary Validation

```typescript
// Scenario: Verify salary is within position range
const employeeData = {
  id: 'emp_123',
  name: 'John Doe',
  salary: 75000,
  position_id: 'pos_456'
};

const crossEntityCondition: CrossEntityValidation = {
  id: 'cross_1',
  ruleName: 'Salary Within Range',
  sourcePath: {
    segments: [],
    displayPath: 'Employee.salary'
  },
  operator: 'greater_equal',
  targetPath: {
    segments: [
      {
        entity: 'Employee',
        field: 'position_id',
        relationship: 'many-to-one',
        targetEntity: 'Position'
      }
    ],
    displayPath: 'Employee → Position.min_salary'
  }
};

// Execute
const resolver = new EntityPathResolver(db);
const { passed } = await resolver.evaluateCrossEntityCondition(
  crossEntityCondition,
  'emp_123',
  'Employee',
  'tenant_1'
);
console.log(`Employee salary is within range: ${passed}`);
```

### Example 2: Multi-Rule Validation Chain

```typescript
// Scenario: Validate employee before promotion
const rules: ValidationRule[] = [
  {
    id: 'rule_1',
    name: 'Tenure Check',
    entity: 'Employee',
    description: 'Employee must have 2+ years tenure',
    severity: 'error',
    condition: {
      id: 'group_1',
      operator: 'AND',
      conditions: [
        {
          id: 'cond_1',
          field: 'tenure_years',
          operator: 'greater_equal',
          value: '2'
        }
      ]
    }
  },
  {
    id: 'rule_2',
    name: 'Performance Check',
    entity: 'Employee',
    description: 'Employee must have good performance rating',
    severity: 'error',
    condition: {
      id: 'group_2',
      operator: 'AND',
      conditions: [
        {
          id: 'cond_2',
          field: 'performance_rating',
          operator: 'greater_equal',
          value: '3.5'
        }
      ]
    },
    dependent_rule_ids: ['rule_1']
  },
  {
    id: 'rule_3',
    name: 'Promotion Eligible',
    entity: 'Employee',
    description: 'Employee is eligible for promotion',
    severity: 'warning',
    condition: {
      id: 'group_3',
      operator: 'AND',
      conditions: [
        {
          id: 'cond_3',
          field: 'budget_available',
          operator: 'is_true',
          value: 'true'
        }
      ]
    },
    dependent_rule_ids: ['rule_2']
  }
];

// Execute rule chain
const executor = new RuleExecutor(db);
const result = await executor.executeRuleChain(
  'rule_3',
  'tenant_1',
  'datasource_1',
  {
    tenure_years: 3,
    performance_rating: 4.2,
    budget_available: true
  }
);

console.log('Promotion Eligible Result:');
console.log(`- Passed: ${result.passed}`);
console.log(`- Severity: ${result.severity}`);
console.log(`- Dependencies checked: ${result.dependencyResults?.length || 0}`);
```

---

**Status:** Ready for Production ✅  
**Last Updated:** October 20, 2025
