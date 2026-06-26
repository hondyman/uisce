# Backend Rule Engine Implementation Guide

**Date:** October 20, 2025  
**Version:** 1.0.0  
**Status:** Ready for Implementation

---

## Table of Contents

1. [Overview](#overview)
2. [Core Components](#core-components)
3. [Condition Evaluation Engine](#condition-evaluation-engine)
4. [Rule Dependency Chain Executor](#rule-dependency-chain-executor)
5. [Cross-Entity Path Resolution](#cross-entity-path-resolution)
6. [Database Schema](#database-schema)
7. [GraphQL Integration](#graphql-integration)
8. [Implementation Guide](#implementation-guide)
9. [Testing Strategies](#testing-strategies)
10. [Performance Optimization](#performance-optimization)

---

## Overview

The backend rule engine provides three core capabilities:

1. **Condition Evaluation** - Recursive evaluation of complex nested conditions
2. **Rule Dependency Chain** - Sequential rule execution with prerequisite checks
3. **Cross-Entity Validation** - Path resolution and field comparison across related entities

### Architecture

```
Frontend (AdvancedRuleConfiguration)
  ↓
GraphQL API
  ↓
Rule Service Layer
  ├─ ConditionEvaluator (recursive evaluation)
  ├─ RuleExecutor (dependency management)
  └─ EntityPathResolver (relationship traversal)
  ↓
Database
  ├─ validation_rules
  ├─ rule_dependencies
  ├─ cross_entity_validations
  └─ (entity tables: employees, departments, etc.)
```

---

## Core Components

### 1. Type Definitions

```typescript
// Shared types for frontend and backend
interface Condition {
  field: string;
  operator: string;
  value: string; // Stored as string, converted at eval time
}

interface ConditionNode extends Condition {
  id: string;
}

interface ConditionGroup {
  id: string;
  operator: 'AND' | 'OR';
  conditions: (ConditionNode | ConditionGroup)[];
}

interface ValidationRule {
  id: string;
  name: string;
  entity: string;
  description: string;
  severity: 'error' | 'warning' | 'info';
  condition: ConditionGroup;
  dependent_rule_ids?: string[];
  createdAt: Date;
  updatedAt: Date;
  tenantId: string;
  datasourceId: string;
}

interface RuleDependency {
  id: string;
  ruleId: string;
  dependentRuleId: string;
  tenantId: string;
  datasourceId: string;
}

interface EntityPath {
  segments: Array<{
    entity: string;
    field: string;
    relationship: string;
  }>;
  displayPath: string;
}

interface CrossEntityCondition {
  id: string;
  ruleName: string;
  sourcePath: EntityPath;
  operator: string;
  targetPath: EntityPath;
  ruleId: string;
  tenantId: string;
  datasourceId: string;
}

interface RuleEvaluationResult {
  ruleId: string;
  passed: boolean;
  severity: 'error' | 'warning' | 'info';
  message: string;
  evaluatedAt: Date;
  dependencyResults?: RuleEvaluationResult[];
}

interface EntityPathResolution {
  entityId: string;
  entity: string;
  fieldValue: any;
}
```

### 2. Type Guards

```typescript
function isCondition(node: ConditionNode | ConditionGroup): node is ConditionNode {
  return 'field' in node && 'operator' in node && 'value' in node;
}

function isGroup(node: ConditionNode | ConditionGroup): node is ConditionGroup {
  return 'conditions' in node && Array.isArray(node.conditions);
}

function isValidOperator(operator: string): boolean {
  const validOperators = [
    'equals', 'not_equals', 'contains', 'starts_with', 'ends_with',
    'greater_than', 'less_than', 'greater_equal', 'less_equal',
    'is_true', 'is_false',
    'before', 'after', 'between'
  ];
  return validOperators.includes(operator);
}
```

---

## Condition Evaluation Engine

### Core Evaluator

```typescript
class ConditionEvaluator {
  /**
   * Recursively evaluates a condition node or group
   * @param node The condition/group to evaluate
   * @param data Record containing field values to test against
   * @returns boolean result of evaluation
   */
  evaluate(node: ConditionNode | ConditionGroup, data: Record<string, any>): boolean {
    if (isCondition(node)) {
      return this.evaluateCondition(node, data);
    } else if (isGroup(node)) {
      return this.evaluateGroup(node, data);
    }
    return false;
  }

  /**
   * Evaluate a single condition
   */
  private evaluateCondition(condition: Condition, data: Record<string, any>): boolean {
    const fieldValue = data[condition.field];
    const compareValue = condition.value;

    switch (condition.operator) {
      // String operators
      case 'equals':
        return String(fieldValue) === String(compareValue);
      
      case 'not_equals':
        return String(fieldValue) !== String(compareValue);
      
      case 'contains':
        return String(fieldValue).includes(String(compareValue));
      
      case 'starts_with':
        return String(fieldValue).startsWith(String(compareValue));
      
      case 'ends_with':
        return String(fieldValue).endsWith(String(compareValue));

      // Number operators
      case 'greater_than':
        return Number(fieldValue) > Number(compareValue);
      
      case 'less_than':
        return Number(fieldValue) < Number(compareValue);
      
      case 'greater_equal':
        return Number(fieldValue) >= Number(compareValue);
      
      case 'less_equal':
        return Number(fieldValue) <= Number(compareValue);

      // Boolean operators
      case 'is_true':
        return Boolean(fieldValue) === true;
      
      case 'is_false':
        return Boolean(fieldValue) === false;

      // Date operators
      case 'before':
        return new Date(fieldValue) < new Date(compareValue);
      
      case 'after':
        return new Date(fieldValue) > new Date(compareValue);
      
      case 'between':
        const [start, end] = compareValue.split(',');
        const date = new Date(fieldValue);
        return date >= new Date(start) && date <= new Date(end);

      default:
        throw new Error(`Unknown operator: ${condition.operator}`);
    }
  }

  /**
   * Evaluate a group of conditions with AND/OR logic
   */
  private evaluateGroup(group: ConditionGroup, data: Record<string, any>): boolean {
    const results = group.conditions.map(condition => 
      this.evaluate(condition, data)
    );

    if (group.operator === 'AND') {
      // All conditions must be true
      return results.every(result => result === true);
    } else if (group.operator === 'OR') {
      // At least one condition must be true
      return results.some(result => result === true);
    }

    return false;
  }

  /**
   * Get evaluation details (for debugging)
   */
  evaluateWithDetails(
    node: ConditionNode | ConditionGroup,
    data: Record<string, any>
  ): {
    result: boolean;
    details: Array<{ condition: string; result: boolean }>;
  } {
    const details: Array<{ condition: string; result: boolean }> = [];

    const evaluateRecursive = (
      n: ConditionNode | ConditionGroup,
      d: Record<string, any>
    ): boolean => {
      if (isCondition(n)) {
        const result = this.evaluateCondition(n, d);
        details.push({
          condition: `${n.field} ${n.operator} ${n.value}`,
          result
        });
        return result;
      } else if (isGroup(n)) {
        const groupResults = n.conditions.map(c => evaluateRecursive(c, d));
        const result = n.operator === 'AND'
          ? groupResults.every(r => r)
          : groupResults.some(r => r);
        
        details.push({
          condition: `(${n.operator} group)`,
          result
        });
        return result;
      }
      return false;
    };

    const result = evaluateRecursive(node, data);
    return { result, details };
  }
}
```

### Usage Examples

```typescript
// Example 1: Simple condition
const evaluator = new ConditionEvaluator();

const simpleCondition: ConditionNode = {
  id: 'cond_1',
  field: 'age',
  operator: 'greater_equal',
  value: '18'
};

const data = { age: 25, status: 'Active' };
const result = evaluator.evaluate(simpleCondition, data);
console.log(result); // true

// Example 2: Nested group with AND/OR
const complexCondition: ConditionGroup = {
  id: 'group_1',
  operator: 'OR',
  conditions: [
    {
      id: 'group_2',
      operator: 'AND',
      conditions: [
        { id: 'cond_1', field: 'age', operator: 'greater_equal', value: '18' },
        { id: 'cond_2', field: 'status', operator: 'equals', value: 'Active' }
      ]
    },
    {
      id: 'group_3',
      operator: 'AND',
      conditions: [
        { id: 'cond_3', field: 'isVip', operator: 'is_true', value: 'true' },
        { id: 'cond_4', field: 'salary', operator: 'greater_than', value: '50000' }
      ]
    }
  ]
};

// (Age >= 18 AND Status = 'Active') OR (IsVip = true AND Salary > 50000)
const result = evaluator.evaluate(complexCondition, data);
console.log(result); // true or false based on data

// Example 3: With details for debugging
const { result, details } = evaluator.evaluateWithDetails(complexCondition, data);
console.log('Result:', result);
console.log('Details:', details);
```

---

## Rule Dependency Chain Executor

### Core Executor

```typescript
class RuleExecutor {
  private evaluator: ConditionEvaluator;
  private db: Database; // Your database connection
  private cache: Map<string, ValidationRule> = new Map();

  constructor(db: Database, cache?: Map<string, ValidationRule>) {
    this.evaluator = new ConditionEvaluator();
    this.db = db;
    this.cache = cache || new Map();
  }

  /**
   * Execute a single rule without dependencies
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
   * Dependencies are evaluated first in order
   * If a dependency fails with 'error' severity, chain stops
   */
  async executeRuleChain(
    ruleId: string,
    tenantId: string,
    datasourceId: string,
    data: Record<string, any>,
    options?: {
      stopOnError?: boolean;
      maxDepth?: number;
    }
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

  /**
   * Recursive rule chain execution
   */
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
    let rule = this.cache.get(ruleId);
    if (!rule) {
      rule = await this.db.query(
        `SELECT * FROM validation_rules 
         WHERE id = ? AND tenant_id = ? AND datasource_id = ?`,
        [ruleId, tenantId, datasourceId]
      );
      this.cache.set(ruleId, rule);
    }

    if (!rule) {
      throw new Error(`Rule not found: ${ruleId}`);
    }

    const dependencyResults: RuleEvaluationResult[] = [];

    // Execute dependencies first
    if (rule.dependent_rule_ids && rule.dependent_rule_ids.length > 0) {
      for (const dependentRuleId of rule.dependent_rule_ids) {
        const depResult = await this.executeRuleChainRecursive(
          dependentRuleId,
          tenantId,
          datasourceId,
          data,
          stopOnError,
          depth + 1,
          maxDepth
        );

        dependencyResults.push(depResult);

        // Stop if dependency failed with error severity
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

    return {
      ...currentResult,
      dependencyResults: dependencyResults.length > 0 ? dependencyResults : undefined
    };
  }

  /**
   * Execute multiple rules in parallel
   */
  async executeRules(
    ruleIds: string[],
    tenantId: string,
    datasourceId: string,
    data: Record<string, any>
  ): Promise<RuleEvaluationResult[]> {
    const promises = ruleIds.map(ruleId =>
      this.executeRuleChain(ruleId, tenantId, datasourceId, data)
    );

    return Promise.all(promises);
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
    const rules = await this.db.query(
      `SELECT * FROM validation_rules 
       WHERE entity = ? AND tenant_id = ? AND datasource_id = ?`,
      [entity, tenantId, datasourceId]
    );

    return this.executeRules(
      rules.map(r => r.id),
      tenantId,
      datasourceId,
      data
    );
  }

  /**
   * Validate rule dependencies for circular references
   */
  async validateNoCycles(
    ruleId: string,
    tenantId: string,
    datasourceId: string,
    newDependencies: string[]
  ): Promise<{ valid: boolean; cycle?: string[] }> {
    const visited = new Set<string>();
    const recursionStack = new Set<string>();

    const hasCycle = async (id: string, path: string[] = []): Promise<boolean> => {
      if (recursionStack.has(id)) {
        return true; // Cycle detected
      }

      if (visited.has(id)) {
        return false; // Already checked, no cycle from this path
      }

      visited.add(id);
      recursionStack.add(id);

      const rule = await this.db.query(
        `SELECT dependent_rule_ids FROM validation_rules 
         WHERE id = ? AND tenant_id = ? AND datasource_id = ?`,
        [id, tenantId, datasourceId]
      );

      const deps = rule?.dependent_rule_ids || [];

      for (const depId of deps) {
        if (await hasCycle(depId, [...path, id])) {
          return true;
        }
      }

      recursionStack.delete(id);
      return false;
    };

    // Test if adding new dependencies creates a cycle
    for (const depId of newDependencies) {
      if (await hasCycle(ruleId)) {
        return {
          valid: false,
          cycle: Array.from(recursionStack)
        };
      }
    }

    return { valid: true };
  }
}
```

### Usage Examples

```typescript
const db = new Database(connectionString);
const executor = new RuleExecutor(db);

// Example 1: Execute single rule
const rule: ValidationRule = {
  id: 'rule_1',
  name: 'Age Verification',
  entity: 'Employee',
  description: 'Employee must be at least 18',
  severity: 'error',
  condition: {
    id: 'group_1',
    operator: 'AND',
    conditions: [
      { id: 'cond_1', field: 'age', operator: 'greater_equal', value: '18' }
    ]
  },
  tenantId: 'tenant_1',
  datasourceId: 'datasource_1'
};

const employeeData = { age: 25, name: 'John' };
const result = await executor.executeRule(rule, employeeData);
console.log(result);
// { ruleId: 'rule_1', passed: true, severity: 'error', ... }

// Example 2: Execute rule chain with dependencies
const result = await executor.executeRuleChain(
  'rule_3', // Rule with dependencies
  'tenant_1',
  'datasource_1',
  employeeData
);
console.log(result);
// { ruleId: 'rule_3', passed: boolean, dependencyResults: [...] }

// Example 3: Execute multiple rules
const results = await executor.executeRules(
  ['rule_1', 'rule_2', 'rule_3'],
  'tenant_1',
  'datasource_1',
  employeeData
);

// Example 4: Validate no cycles before adding dependencies
const { valid, cycle } = await executor.validateNoCycles(
  'rule_1',
  'tenant_1',
  'datasource_1',
  ['rule_2', 'rule_3']
);
if (!valid) {
  console.error('Circular dependency detected:', cycle);
}
```

---

## Cross-Entity Path Resolution

### Core Resolver

```typescript
class EntityPathResolver {
  private db: Database;
  private entitySchema: Map<string, EntitySchema> = new Map();
  private cache: Map<string, any> = new Map();

  constructor(db: Database) {
    this.db = db;
  }

  /**
   * Register entity schema for relationship resolution
   */
  registerEntity(entity: string, schema: EntitySchema): void {
    this.entitySchema.set(entity, schema);
  }

  /**
   * Resolve a full entity path to get the final field value
   * Example: Employee → Position → min_salary
   */
  async resolvePath(
    path: EntityPath,
    startId: string,
    tenantId: string,
    startEntity: string
  ): Promise<EntityPathResolution> {
    let currentId = startId;
    let currentEntity = startEntity;

    // Traverse all segments except the last
    for (const segment of path.segments) {
      const schema = this.entitySchema.get(segment.entity);
      if (!schema) {
        throw new Error(`Schema not found for entity: ${segment.entity}`);
      }

      // Get the foreign key from current record
      const record = await this.db.query(
        `SELECT ${segment.field} FROM ${currentEntity} WHERE id = ? LIMIT 1`,
        [currentId]
      );

      if (!record || record.length === 0) {
        throw new Error(
          `Record not found in ${currentEntity} with id: ${currentId}`
        );
      }

      currentId = record[0][segment.field];
      currentEntity = segment.targetEntity;

      if (!currentId) {
        throw new Error(
          `Foreign key ${segment.field} is null in ${segment.entity}`
        );
      }
    }

    // Get the final field value
    const finalField = path.displayPath.split('.').pop();
    if (!finalField) {
      throw new Error('Invalid path: no field specified');
    }

    const finalRecord = await this.db.query(
      `SELECT ${finalField} FROM ${currentEntity} WHERE id = ? LIMIT 1`,
      [currentId]
    );

    if (!finalRecord || finalRecord.length === 0) {
      throw new Error(
        `Final record not found in ${currentEntity} with id: ${currentId}`
      );
    }

    return {
      entityId: currentId,
      entity: currentEntity,
      fieldValue: finalRecord[0][finalField]
    };
  }

  /**
   * Resolve both sides of a cross-entity validation
   */
  async resolveCrossEntityCondition(
    condition: CrossEntityCondition,
    recordId: string,
    startEntity: string,
    tenantId: string
  ): Promise<{
    sourceValue: any;
    targetValue: any;
    sourcePath: string;
    targetPath: string;
  }> {
    const sourceResolution = await this.resolvePath(
      condition.sourcePath,
      recordId,
      tenantId,
      startEntity
    );

    const targetResolution = await this.resolvePath(
      condition.targetPath,
      recordId,
      tenantId,
      startEntity
    );

    return {
      sourceValue: sourceResolution.fieldValue,
      targetValue: targetResolution.fieldValue,
      sourcePath: condition.sourcePath.displayPath,
      targetPath: condition.targetPath.displayPath
    };
  }

  /**
   * Evaluate a cross-entity validation condition
   */
  async evaluateCrossEntityCondition(
    condition: CrossEntityCondition,
    recordId: string,
    startEntity: string,
    tenantId: string
  ): Promise<{
    passed: boolean;
    sourceValue: any;
    targetValue: any;
    operator: string;
  }> {
    const { sourceValue, targetValue } = await this.resolveCrossEntityCondition(
      condition,
      recordId,
      startEntity,
      tenantId
    );

    let passed = false;

    switch (condition.operator) {
      case 'equals':
        passed = sourceValue === targetValue;
        break;
      case 'not_equals':
        passed = sourceValue !== targetValue;
        break;
      case 'greater_than':
        passed = Number(sourceValue) > Number(targetValue);
        break;
      case 'less_than':
        passed = Number(sourceValue) < Number(targetValue);
        break;
      case 'greater_equal':
        passed = Number(sourceValue) >= Number(targetValue);
        break;
      case 'less_equal':
        passed = Number(sourceValue) <= Number(targetValue);
        break;
      default:
        throw new Error(`Unknown operator: ${condition.operator}`);
    }

    return {
      passed,
      sourceValue,
      targetValue,
      operator: condition.operator
    };
  }

  /**
   * Batch evaluate cross-entity conditions for multiple records
   */
  async evaluateCrossEntityConditionBatch(
    condition: CrossEntityCondition,
    recordIds: string[],
    startEntity: string,
    tenantId: string
  ): Promise<Array<{ recordId: string; passed: boolean; error?: string }>> {
    return Promise.all(
      recordIds.map(async (recordId) => {
        try {
          const { passed } = await this.evaluateCrossEntityCondition(
            condition,
            recordId,
            startEntity,
            tenantId
          );
          return { recordId, passed };
        } catch (error) {
          return { recordId, passed: false, error: error.message };
        }
      })
    );
  }

  /**
   * Clear cache
   */
  clearCache(): void {
    this.cache.clear();
  }
}

// Entity schema definition
interface EntitySchema {
  name: string;
  table: string;
  fields: Array<{
    name: string;
    type: string;
    relationship?: {
      targetEntity: string;
      foreignKey: string;
    };
  }>;
}
```

### Usage Examples

```typescript
const db = new Database(connectionString);
const resolver = new EntityPathResolver(db);

// Register entity schemas
resolver.registerEntity('Employee', {
  name: 'Employee',
  table: 'employees',
  fields: [
    { name: 'id', type: 'uuid' },
    { name: 'name', type: 'string' },
    { name: 'position_id', type: 'uuid', relationship: { targetEntity: 'Position', foreignKey: 'position_id' } },
    { name: 'salary', type: 'number' }
  ]
});

resolver.registerEntity('Position', {
  name: 'Position',
  table: 'positions',
  fields: [
    { name: 'id', type: 'uuid' },
    { name: 'title', type: 'string' },
    { name: 'min_salary', type: 'number' }
  ]
});

// Example 1: Resolve entity path
const path: EntityPath = {
  segments: [
    { entity: 'Employee', field: 'position_id', relationship: 'many-to-one', targetEntity: 'Position' }
  ],
  displayPath: 'Employee → Position.min_salary'
};

const resolution = await resolver.resolvePath(
  path,
  'employee_123',
  'tenant_1',
  'Employee'
);
console.log(resolution);
// { entityId: 'position_456', entity: 'Position', fieldValue: 50000 }

// Example 2: Evaluate cross-entity condition
const condition: CrossEntityCondition = {
  id: 'cross_1',
  ruleName: 'Salary Within Range',
  sourcePath: {
    segments: [],
    displayPath: 'Employee.salary'
  },
  operator: 'greater_equal',
  targetPath: path,
  ruleId: 'rule_1',
  tenantId: 'tenant_1',
  datasourceId: 'datasource_1'
};

const evaluation = await resolver.evaluateCrossEntityCondition(
  condition,
  'employee_123',
  'Employee',
  'tenant_1'
);
console.log(evaluation);
// { passed: true, sourceValue: 75000, targetValue: 50000, operator: 'greater_equal' }

// Example 3: Batch evaluate for multiple employees
const results = await resolver.evaluateCrossEntityConditionBatch(
  condition,
  ['employee_1', 'employee_2', 'employee_3'],
  'Employee',
  'tenant_1'
);
console.log(results);
// [
//   { recordId: 'employee_1', passed: true },
//   { recordId: 'employee_2', passed: false },
//   { recordId: 'employee_3', passed: true }
// ]
```

---

## Database Schema

### Validation Rules Table

```sql
CREATE TABLE validation_rules (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  name VARCHAR(255) NOT NULL,
  entity VARCHAR(100) NOT NULL,
  description TEXT,
  severity VARCHAR(20) NOT NULL CHECK (severity IN ('error', 'warning', 'info')),
  condition JSONB NOT NULL, -- Stores the ConditionGroup structure
  dependent_rule_ids UUID[] DEFAULT ARRAY[]::UUID[],
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  created_by UUID,
  updated_by UUID,
  
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT fk_datasource FOREIGN KEY (datasource_id) REFERENCES datasources(id),
  INDEX idx_tenant_datasource ON validation_rules(tenant_id, datasource_id),
  INDEX idx_entity ON validation_rules(entity),
  INDEX idx_active ON validation_rules(is_active)
);
```

### Rule Dependencies Table

```sql
CREATE TABLE rule_dependencies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  dependent_rule_id UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_dependent_rule FOREIGN KEY (dependent_rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  CONSTRAINT unique_dependency UNIQUE(rule_id, dependent_rule_id),
  INDEX idx_rule_dependencies ON rule_dependencies(rule_id, dependent_rule_id)
);
```

### Cross-Entity Validations Table

```sql
CREATE TABLE cross_entity_validations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  rule_name VARCHAR(255) NOT NULL,
  source_path JSONB NOT NULL, -- EntityPath structure
  operator VARCHAR(50) NOT NULL,
  target_path JSONB NOT NULL, -- EntityPath structure
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES validation_rules(id) ON DELETE CASCADE,
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  INDEX idx_rule_cross_entity ON cross_entity_validations(rule_id),
  INDEX idx_tenant_datasource ON cross_entity_validations(tenant_id, datasource_id)
);
```

### Rule Evaluation Audit Table

```sql
CREATE TABLE rule_evaluation_audit (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  datasource_id UUID NOT NULL,
  rule_id UUID NOT NULL,
  record_id UUID NOT NULL,
  entity VARCHAR(100) NOT NULL,
  passed BOOLEAN NOT NULL,
  severity VARCHAR(20),
  message TEXT,
  evaluation_details JSONB, -- Details for debugging
  evaluated_at TIMESTAMP DEFAULT NOW(),
  
  CONSTRAINT fk_rule FOREIGN KEY (rule_id) REFERENCES validation_rules(id),
  CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  INDEX idx_rule_evaluation ON rule_evaluation_audit(rule_id, evaluated_at),
  INDEX idx_entity_evaluation ON rule_evaluation_audit(entity, evaluated_at),
  INDEX idx_tenant_evaluation ON rule_evaluation_audit(tenant_id, evaluated_at)
);
```

---

## GraphQL Integration

### Queries

```graphql
type ValidationRule {
  id: ID!
  name: String!
  entity: String!
  description: String!
  severity: RuleSeverity!
  condition: ConditionGroup!
  dependentRuleIds: [ID!]
  isActive: Boolean!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type RuleEvaluationResult {
  ruleId: ID!
  passed: Boolean!
  severity: RuleSeverity!
  message: String!
  evaluatedAt: DateTime!
  dependencyResults: [RuleEvaluationResult!]
}

type CrossEntityValidation {
  id: ID!
  ruleName: String!
  sourcePath: EntityPath!
  operator: String!
  targetPath: EntityPath!
}

type Query {
  # Fetch validation rules
  validationRules(
    tenantId: ID!
    datasourceId: ID!
    entity: String
    isActive: Boolean
  ): [ValidationRule!]!

  validationRule(id: ID!, tenantId: ID!): ValidationRule

  # Evaluate a rule
  evaluateRule(
    ruleId: ID!
    tenantId: ID!
    datasourceId: ID!
    data: JSON!
  ): RuleEvaluationResult!

  # Evaluate rule chain
  evaluateRuleChain(
    ruleId: ID!
    tenantId: ID!
    datasourceId: ID!
    data: JSON!
    stopOnError: Boolean = true
  ): RuleEvaluationResult!

  # Evaluate all rules for entity
  evaluateEntityRules(
    entity: String!
    tenantId: ID!
    datasourceId: ID!
    data: JSON!
  ): [RuleEvaluationResult!]!

  # Cross-entity validations
  crossEntityValidations(
    ruleId: ID!
    tenantId: ID!
  ): [CrossEntityValidation!]!

  # Test cross-entity condition
  testCrossEntityCondition(
    conditionId: ID!
    recordId: ID!
    entity: String!
    tenantId: ID!
  ): CrossEntityEvaluationResult!
}
```

### Mutations

```graphql
type Mutation {
  # Create validation rule
  createValidationRule(
    input: CreateValidationRuleInput!
    tenantId: ID!
    datasourceId: ID!
  ): ValidationRule!

  # Update validation rule
  updateValidationRule(
    id: ID!
    input: UpdateValidationRuleInput!
    tenantId: ID!
  ): ValidationRule!

  # Delete validation rule
  deleteValidationRule(
    id: ID!
    tenantId: ID!
  ): Boolean!

  # Update rule dependencies
  updateRuleDependencies(
    ruleId: ID!
    dependencies: [ID!]!
    tenantId: ID!
    datasourceId: ID!
  ): ValidationRule!

  # Create cross-entity validation
  createCrossEntityValidation(
    input: CreateCrossEntityValidationInput!
    tenantId: ID!
    datasourceId: ID!
  ): CrossEntityValidation!

  # Bulk evaluate rules
  bulkEvaluateRules(
    ruleIds: [ID!]!
    tenantId: ID!
    datasourceId: ID!
    records: [JSON!]!
  ): [RuleEvaluationResult!]!
}

input CreateValidationRuleInput {
  name: String!
  entity: String!
  description: String!
  severity: RuleSeverity!
  condition: ConditionGroupInput!
}

input ConditionGroupInput {
  operator: String!
  conditions: [ConditionNodeInput!]!
}

input ConditionNodeInput {
  field: String!
  operator: String!
  value: String!
}

input CreateCrossEntityValidationInput {
  ruleName: String!
  ruleId: ID!
  sourcePath: EntityPathInput!
  operator: String!
  targetPath: EntityPathInput!
}

input EntityPathInput {
  segments: [PathSegmentInput!]!
  displayPath: String!
}

input PathSegmentInput {
  entity: String!
  field: String!
  relationship: String!
  targetEntity: String!
}
```

---

## Implementation Guide

### Step 1: Set Up Type System

```typescript
// src/types/rules.ts
export * from './rule-types';
export * from './condition-types';
export * from './cross-entity-types';
```

### Step 2: Implement Evaluators

```typescript
// src/services/condition-evaluator.ts
export class ConditionEvaluator {
  // Implementation from above
}

// src/services/rule-executor.ts
export class RuleExecutor {
  // Implementation from above
}

// src/services/entity-path-resolver.ts
export class EntityPathResolver {
  // Implementation from above
}
```

### Step 3: Create GraphQL Resolvers

```typescript
// src/resolvers/rule-resolvers.ts
export const ruleResolvers = {
  Query: {
    validationRules: async (parent, args, context) => {
      // Query implementation
    },
    evaluateRule: async (parent, args, context) => {
      // Evaluation implementation
    }
  },
  Mutation: {
    createValidationRule: async (parent, args, context) => {
      // Create implementation
    }
  }
};
```

### Step 4: Database Migrations

```bash
# Create migration files
npx migrate create add-validation-rules
npx migrate create add-rule-dependencies
npx migrate create add-cross-entity-validations
npx migrate create add-rule-evaluation-audit
```

### Step 5: Integration with Existing Services

```typescript
// src/services/validation-service.ts
export class ValidationService {
  private executor: RuleExecutor;
  private resolver: EntityPathResolver;

  constructor(db: Database) {
    this.executor = new RuleExecutor(db);
    this.resolver = new EntityPathResolver(db);
  }

  async validateEntity(
    entity: string,
    entityId: string,
    data: Record<string, any>,
    tenantId: string,
    datasourceId: string
  ): Promise<ValidationResult> {
    // Validate against all rules
    const results = await this.executor.executeEntityRules(
      entity,
      tenantId,
      datasourceId,
      data
    );

    return {
      valid: results.every(r => r.passed || r.severity !== 'error'),
      results,
      errors: results.filter(r => !r.passed && r.severity === 'error'),
      warnings: results.filter(r => !r.passed && r.severity === 'warning')
    };
  }
}
```

---

## Testing Strategies

### Unit Tests

```typescript
// tests/condition-evaluator.test.ts
describe('ConditionEvaluator', () => {
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    evaluator = new ConditionEvaluator();
  });

  it('evaluates simple condition', () => {
    const condition: ConditionNode = {
      id: '1',
      field: 'age',
      operator: 'greater_equal',
      value: '18'
    };
    const data = { age: 25 };
    expect(evaluator.evaluate(condition, data)).toBe(true);
  });

  it('evaluates AND group', () => {
    const group: ConditionGroup = {
      id: '1',
      operator: 'AND',
      conditions: [
        { id: '1', field: 'age', operator: 'greater_equal', value: '18' },
        { id: '2', field: 'status', operator: 'equals', value: 'Active' }
      ]
    };
    const data = { age: 25, status: 'Active' };
    expect(evaluator.evaluate(group, data)).toBe(true);
  });

  // More tests...
});
```

### Integration Tests

```typescript
// tests/rule-executor.test.ts
describe('RuleExecutor', () => {
  let executor: RuleExecutor;
  let mockDb: Database;

  beforeEach(() => {
    mockDb = new MockDatabase();
    executor = new RuleExecutor(mockDb);
  });

  it('executes rule with dependencies', async () => {
    // Setup mocks
    // Execute rule
    // Assert results
  });

  // More tests...
});
```

---

## Performance Optimization

### Caching Strategy

```typescript
class CachedRuleExecutor extends RuleExecutor {
  private ruleCache: Map<string, ValidationRule> = new Map();
  private resultCache: Map<string, RuleEvaluationResult> = new Map();
  private ttl = 5 * 60 * 1000; // 5 minutes

  async executeRule(
    rule: ValidationRule,
    data: Record<string, any>
  ): Promise<RuleEvaluationResult> {
    const cacheKey = `${rule.id}:${JSON.stringify(data)}`;
    const cached = this.resultCache.get(cacheKey);

    if (cached) {
      return cached;
    }

    const result = await super.executeRule(rule, data);
    this.resultCache.set(cacheKey, result);

    // Clear cache after TTL
    setTimeout(() => this.resultCache.delete(cacheKey), this.ttl);

    return result;
  }
}
```

### Batch Processing

```typescript
async function batchEvaluateRules(
  ruleIds: string[],
  records: Array<Record<string, any>>,
  executor: RuleExecutor
): Promise<Map<string, RuleEvaluationResult[]>> {
  const results = new Map<string, RuleEvaluationResult[]>();

  // Process in parallel chunks
  const chunkSize = 10;
  for (let i = 0; i < ruleIds.length; i += chunkSize) {
    const chunk = ruleIds.slice(i, i + chunkSize);
    const chunkResults = await Promise.all(
      chunk.map(ruleId =>
        Promise.all(
          records.map(record =>
            executor.executeRule(getRuleSync(ruleId), record)
          )
        )
      )
    );

    chunk.forEach((ruleId, index) => {
      results.set(ruleId, chunkResults[index]);
    });
  }

  return results;
}
```

---

**Status:** Implementation Ready ✅  
**Last Updated:** October 20, 2025  
**Next Step:** Database schema setup and GraphQL resolver implementation
