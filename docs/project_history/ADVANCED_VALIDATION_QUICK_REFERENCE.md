# Advanced Validation: Integration Quick Reference

**Last Updated:** October 20, 2025

---

## 🎯 Quick Navigation

### Feature 1: Advanced Condition Builder
- **What it does:** Create complex validation logic with AND/OR groups
- **Where it is:** `/frontend/src/components/ExpressionBuilder/AdvancedConditionBuilder.tsx`
- **Evaluation:** Export `evaluateCondition()` function tests conditions against data
- **Example:** `(Age ≥ 18 AND Status = 'Active') OR (VIP = true AND Salary > 50000)`

### Feature 2: Cross-Entity Validation
- **What it does:** Validate fields across related entities with dependencies
- **Where it is:** `/frontend/src/components/validation/CrossEntityValidationBuilder.tsx`
- **Relationships:** 4 entities (Employee, Department, Position, Location) with 11 relationships
- **Example:** `Employee.salary >= Employee.Position.min_salary`

### Backend Evaluation
- **Where it is:** `/backend/internal/services/validation_rule_engine.go`
- **Interface:** `ValidationRuleEngine` - 9 methods
- **Core:** `EvaluateCondition()`, `EvaluateRule()`, `evaluateConditionTree()` (recursive)

---

## 🔗 How They Connect

```
┌─────────────────────────────────────────────────────────────┐
│ FRONTEND: Advanced Condition Builder (509 lines)            │
│ - AdvancedConditionBuilder component                         │
│ - evaluateCondition() → test conditions locally              │
│ - Type: ConditionNode (Condition | ConditionGroup)          │
│ - Operators: 15 across 4 types (string, number, date, bool) │
└─────────────────────────────────────────────────────────────┘
                            ↓
                 JSON serialization
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ API: POST /api/validation-rules (tenant-scoped)             │
│ Headers: X-Tenant-ID, X-Tenant-Datasource-ID               │
│ Body: { condition: ConditionNode, ... }                     │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ BACKEND: Validation Rule Engine (679 lines)                 │
│ - EvaluateRule() → orchestrator                             │
│ - EvaluateCondition() → single condition logic              │
│ - evaluateConditionTree() → recursive AND/OR/NOT            │
│ - Supports 12+ operators                                    │
│ - Returns RuleEvaluationResult with pass/fail + action      │
└─────────────────────────────────────────────────────────────┘
                            ↓
┌─────────────────────────────────────────────────────────────┐
│ DATABASE: validation_rules (PostgreSQL)                     │
│ - condition_json: RawMessage (stores ConditionNode JSON)    │
│ - field_path: TEXT[] (for hierarchy support)                │
│ - aggregation_type: VARCHAR (SUM, COUNT, AVG, MIN, MAX)    │
│ - Indexes: (tenant_id, datasource_id, field_path)          │
└─────────────────────────────────────────────────────────────┘
```

---

## 🚀 Cross-Entity Feature Detail

```
┌─────────────────────────────────────────────────────────────┐
│ CROSS-ENTITY VALIDATION LAYER                               │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│ 1. EntityPathPicker (Modal UI)                              │
│    ├─ Navigate entity tree: Employee → Department          │
│    ├─ Select final field: min_salary                        │
│    └─ Generates: EntityPath with segments[]                 │
│                                                              │
│ 2. RuleDependencyChain (Numbered flow)                       │
│    ├─ Rule 1 (ID: abc) → Rule 2 (ID: def) → Rule 3 (ID: ghi)│
│    ├─ Prevents circular dependencies                        │
│    └─ Stores dependent_rule_ids[]                           │
│                                                              │
│ 3. Comparison Operators                                     │
│    ├─ = (equals)                                            │
│    ├─ ≠ (not equals)                                        │
│    ├─ < (less than)                                         │
│    ├─ > (greater than)                                      │
│    ├─ ≤ (less or equal)                                     │
│    └─ ≥ (greater or equal)                                  │
│                                                              │
│ 4. Entity Relationship Map (Mock Data)                       │
│    Employee:                                                │
│      ├─ position_id → Position                              │
│      ├─ department_id → Department                          │
│      ├─ manager_id → Employee                               │
│      └─ location_id → Location                              │
│                                                              │
│    Position:                                                │
│      ├─ min_salary (field for validation)                   │
│      ├─ max_salary (field for validation)                   │
│      └─ job_level (field for validation)                    │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 📝 Example: Complete Validation Rule

```typescript
// Frontend creates this with AdvancedConditionBuilder
const advancedRule = {
  id: 'rule-1',
  name: 'High Earner VIP Check',
  entity: 'Employee',
  description: 'Validate VIP employees meet salary expectations',
  severity: 'error',
  condition: {
    type: 'group',
    operator: 'AND',
    conditions: [
      {
        id: 'check-1',
        field: 'vip_status',
        operator: 'equals',
        value: 'true',
        fieldType: 'boolean'
      },
      {
        id: 'check-2',
        field: 'salary',
        operator: 'greater_than',
        value: '100000',
        fieldType: 'number'
      }
    ]
  }
};

// With cross-entity dependency
const crossEntityRule = {
  id: 'rule-2',
  name: 'Salary Within Range',
  entity: 'Employee',
  description: 'Verify employee salary is within position range',
  severity: 'error',
  dependent_rule_ids: ['rule-1'], // Must execute rule-1 first
  condition: {
    type: 'cross_entity',
    sourcePath: {
      segments: [{ entity: 'Employee', field: 'salary', relationship: 'self' }],
      displayPath: 'Employee.salary'
    },
    operator: '<=',
    targetPath: {
      segments: [
        { entity: 'Employee', field: 'position_id', relationship: 'many-to-one' },
        { entity: 'Position', field: 'max_salary', relationship: 'self' }
      ],
      displayPath: 'Employee.Position.max_salary'
    }
  }
};

// Test locally with evaluateCondition
const testData = {
  vip_status: true,
  salary: 120000
};
const result = evaluateCondition(advancedRule.condition, testData); // true

// Send to backend for persistence and execution
const response = await fetch('/api/validation-rules', {
  method: 'POST',
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId,
    'Content-Type': 'application/json'
  },
  body: JSON.stringify({ rule: advancedRule, crossEntityRule })
});

// Backend evaluates with RuleEvaluationResult
const result = {
  ruleId: 'rule-1',
  passed: true,
  errorMessage: '',
  actionToTake: 'route:approval_queue',
  evaluationTime: '15ms',
  details: { conditions_checked: 2, conditions_passed: 2 }
};
```

---

## 🧪 Testing Support

### Frontend - Test evaluateCondition()
```typescript
import { evaluateCondition } from './AdvancedConditionBuilder';

const condition = {
  type: 'group',
  operator: 'AND',
  conditions: [
    { field: 'age', operator: 'greater_equal', value: '18' },
    { field: 'status', operator: 'equals', value: 'Active' }
  ]
};

const testData = { age: 25, status: 'Active' };
const result = evaluateCondition(condition, testData); // true
```

### Backend - Test with Test Cases
```go
// Use ValidationRuleEngine interface
engine := NewValidationRuleEngine(db)

testData := map[string]interface{}{
  "age": 25,
  "status": "Active",
  "salary": 60000,
}

result, err := engine.EvaluateCondition(condition, testData)
if err != nil {
  log.Fatal(err)
}

if !result {
  log.Println("Condition failed")
}
```

---

## 📊 Operators Reference

### String Operators (7)
| Operator | Symbol | Example |
|----------|--------|---------|
| equals | = | Status = 'Active' |
| not_equals | ≠ | Status ≠ 'Inactive' |
| contains | ∋ | Email contains '@domain.com' |
| starts_with | ⊢ | Code starts with 'EMP' |
| ends_with | ⊣ | Code ends with '001' |
| is_empty | ∅ | Notes is empty |
| is_not_empty | ∄ | Notes is not empty |

### Number Operators (6)
| Operator | Symbol | Example |
|----------|--------|---------|
| equals | = | Salary = 50000 |
| not_equals | ≠ | Salary ≠ 50000 |
| greater_than | > | Age > 18 |
| less_than | < | Age < 65 |
| greater_equal | ≥ | Age ≥ 18 |
| less_equal | ≤ | Age ≤ 65 |

### Date Operators (4)
| Operator | Symbol | Example |
|----------|--------|---------|
| equals | = | HireDate = 2025-01-01 |
| before | < | StartDate < 2025-01-01 |
| after | > | EndDate > 2025-01-01 |
| between | ⟷ | Date between 2025-01 and 2025-12 |

### Boolean Operators (2)
| Operator | Symbol | Example |
|----------|--------|---------|
| is_true | ✓ | IsActive = true |
| is_false | ✗ | IsArchived = false |

### Logical Operators (3)
| Operator | Meaning | Example |
|----------|---------|---------|
| AND | All conditions true | Cond1 AND Cond2 AND Cond3 |
| OR | Any condition true | Cond1 OR Cond2 OR Cond3 |
| NOT | Negate condition | NOT (Cond1) |

---

## 🔐 Tenant Scoping

All features enforce tenant isolation via:

```typescript
// Frontend - automatic via setupTenantFetch.ts
const response = await fetch('/api/validation-rules?tenant_id=...&datasource_id=...', {
  headers: {
    'X-Tenant-ID': tenantId,
    'X-Tenant-Datasource-ID': datasourceId
  }
});

// Backend - enforced in every query
func (vre *ValidationRuleEngineImpl) GetTenantRules(
  ctx context.Context,
  tenantID string,
) ([]ValidationRuleDefinition, error) {
  var rules []ValidationRuleDefinition
  err := vre.db.SelectContext(ctx, &rules,
    `SELECT * FROM validation_rules WHERE tenant_id = ? ORDER BY priority`,
    tenantID,
  )
  return rules, err
}

// Database - indexed for performance
CREATE INDEX idx_validation_rules_hierarchy 
ON validation_rules(tenant_id, datasource_id, field_path);
```

---

## 📦 Export Summary

### TypeScript Exports
```typescript
// AdvancedConditionBuilder.tsx
export interface Condition { ... }
export interface ConditionGroup { ... }
export type ConditionNode = Condition | ConditionGroup
export const AdvancedConditionBuilder: React.FC<AdvancedConditionBuilderProps>
export const evaluateCondition: (node: ConditionNode, data: Record<string, any>) => boolean

// CrossEntityValidationBuilder.tsx
export interface ValidationRule { ... }
export interface EntityPath { ... }
export interface CrossEntityCondition { ... }
export const CrossEntityValidationBuilder: React.FC<CrossEntityValidationBuilderProps>
export const RuleDependencyChain: React.FC<RuleDependencyChainProps>
export const EntityPathPicker: React.FC<EntityPathPickerProps>
```

### Go Exports
```go
// validation_rule_engine.go
type ValidationRuleEngine interface { ... }
type ValidationRuleEngineImpl struct { ... }
type RuleCondition struct { ... }
type ComplexCondition struct { ... }
type ValidationRuleDefinition struct { ... }
type RuleEvaluationResult struct { ... }

func NewValidationRuleEngine(db *sqlx.DB) ValidationRuleEngine
func (vre *ValidationRuleEngineImpl) EvaluateCondition(...) (bool, error)
func (vre *ValidationRuleEngineImpl) EvaluateComplexCondition(...) (bool, error)
func (vre *ValidationRuleEngineImpl) EvaluateRule(...) (*RuleEvaluationResult, error)
func (vre *ValidationRuleEngineImpl) EvaluateBPStep(...) ([]*RuleEvaluationResult, error)
func (vre *ValidationRuleEngineImpl) StoreRule(ctx, rule) error
func (vre *ValidationRuleEngineImpl) GetRulesForBPStep(...) ([]ValidationRuleDefinition, error)
func (vre *ValidationRuleEngineImpl) GetTenantRules(...) ([]ValidationRuleDefinition, error)
func (vre *ValidationRuleEngineImpl) DeleteRule(ctx, ruleID) error
func (vre *ValidationRuleEngineImpl) GetRuleByID(ctx, ruleID) (*ValidationRuleDefinition, error)
```

---

## ✅ Deployment Steps

1. **Database Migration** (if not already applied)
   ```bash
   psql -U postgres -d alpha < backend/db/migrations/2025_10_20_add_hierarchy_support.sql
   ```

2. **Backend Build**
   ```bash
   cd backend && go build ./cmd/server
   ```

3. **Frontend Build**
   ```bash
   cd frontend && npm run build
   ```

4. **Integration** (if in separate feature branches)
   - Import `AdvancedConditionBuilder` into rule editor
   - Import `CrossEntityValidationBuilder` for cross-entity rules
   - Wire to backend API endpoints

5. **Testing**
   - Test locally with mock data using `evaluateCondition()`
   - Test API endpoints with cURL (examples in separate doc)
   - Verify tenant isolation with different tenant IDs

---

## 🎓 Key Concepts

| Concept | Explanation |
|---------|-------------|
| **ConditionNode** | Either a single Condition or a ConditionGroup (recursive) |
| **Condition** | Single field-operator-value validation (e.g., Age > 18) |
| **ConditionGroup** | Multiple conditions combined with AND/OR logic |
| **AND Logic** | All conditions must be true (results.every(r => r)) |
| **OR Logic** | At least one condition must be true (results.some(r => r)) |
| **Type-Aware** | Operators change based on field type (string, number, date, boolean) |
| **Cross-Entity** | Validate fields across related entities via foreign keys |
| **EntityPath** | Sequence of entity-field-relationship segments |
| **Dependency Chain** | Rules that must execute in specific order, preventing circular deps |
| **Recursive Evaluation** | evaluateConditionTree() handles arbitrary nesting depth |

---

## 🚨 Common Mistakes to Avoid

❌ **Don't:** Pass ConditionNode directly without importing from component
✅ **Do:** Import types and functions from `AdvancedConditionBuilder` or `CrossEntityValidationBuilder`

❌ **Don't:** Mix AND/OR at root level
✅ **Do:** Use ConditionGroup at root to explicitly set operator

❌ **Don't:** Create circular dependencies
✅ **Do:** EntityPathPicker and RuleDependencyChain prevent this automatically

❌ **Don't:** Skip tenant headers in API calls
✅ **Do:** Always include `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers

❌ **Don't:** Evaluate conditions server-side without validation
✅ **Do:** Backend re-validates all conditions before taking action

---

## 📞 Support Resources

- **Frontend Components:** `/frontend/src/components/ExpressionBuilder/` and `/frontend/src/components/validation/`
- **Backend Engine:** `/backend/internal/services/validation_rule_engine.go`
- **Database Schema:** `/backend/db/migrations/2025_10_20_add_hierarchy_support.sql`
- **Full Documentation:** `FEATURE_STATUS_ADVANCED_VALIDATION.md`

---

**Status:** ✅ Complete and Ready for Production  
**Last Verified:** October 20, 2025
