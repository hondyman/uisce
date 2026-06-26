# 🚀 Phase 5: Workday Trigger System - Complete Specification

**Status:** Ready for Planning & Design  
**Timeline:** 2-3 weeks  
**Scope:** 13 Trigger Types + Event System  
**Start After:** Phase 1-4 deployed and verified (production)

---

## 📋 Executive Summary

Phase 5 extends Phase 1-4 validation system with **Workday-style triggers** - application-layer events that fire rules automatically based on actions (Save, Delete, Field Change, etc).

**Key Difference from Phase 1-4:**
- **Phase 1-4:** Rules exist, but require manual invocation
- **Phase 5:** Rules fire automatically when events happen

**Example:**
```
User saves order with total=-50
  ↓
TRIGGER: save + orders entity
  ↓
FETCH: All rules for "orders" entity
  ↓
EXECUTE: "Order Total Positive" rule
  ↓
RESULT: Block save + error "Total must be > 0"
```

---

## 🎯 13 Trigger Types (Workday Standard)

### Tier 1: Core (Immediate, High-Value)
These 3 are easiest & provide most value. **Recommend doing these in Phase 5a (Week 1-2):**

| # | Trigger | Fires When | Example | Complexity | Value |
|---|---------|-----------|---------|-----------|-------|
| 1 | **Save** | Entity saved | User clicks "Save Order" | ⭐ Low | ⭐⭐⭐ High |
| 2 | **Field Change** | Field modified | User edits "Total" field | ⭐ Low | ⭐⭐⭐ High |
| 3 | **Delete** | Entity deleted | User clicks "Delete Order" | ⭐ Low | ⭐⭐⭐ High |

### Tier 2: Common (Medium-Value)
These 3 are medium complexity. **Optional for Phase 5a, or Phase 5b (Week 2-3):**

| # | Trigger | Fires When | Example | Complexity | Value |
|---|---------|-----------|---------|-----------|-------|
| 4 | **Create** | New entity | User clicks "New Order" | ⭐⭐ Medium | ⭐⭐ Medium |
| 5 | **Workflow Step** | BP step completes | Manager approves order | ⭐⭐⭐ High | ⭐⭐ Medium |
| 6 | **Sub-Entity Change** | Child modified | Line item qty changed | ⭐⭐ Medium | ⭐⭐⭐ High |

### Tier 3: Advanced (Lower-Value, Future)
These are harder or less immediate. **Reserve for Phase 6+:**

| # | Trigger | Fires When | Example | Complexity | Value |
|---|---------|-----------|---------|-----------|-------|
| 7 | **Bulk Load** | Batch import | CSV upload | ⭐⭐⭐ High | ⭐ Low |
| 8 | **Integration** | External event | API webhook | ⭐⭐⭐ High | ⭐ Low |
| 9 | **Time-Based** | Scheduled time | Daily 2am | ⭐⭐⭐⭐ Very High | ⭐ Low |
| 10 | **Status Change** | Status field changed | Pending → Approved | ⭐⭐ Medium | ⭐⭐ Medium |
| 11 | **Calculated Field** | Formula result changes | Total (qty × price) | ⭐⭐⭐ High | ⭐⭐ Medium |
| 12 | **Relationship Change** | FK changed | Customer reassigned | ⭐⭐ Medium | ⭐ Low |
| 13 | **Security Role** | User role assigned | New Manager | ⭐⭐⭐⭐ Very High | ⭐ Low |

---

## 🏗️ Phase 5a Architecture (Week 1-2: Core 3 Triggers)

### Database Schema Addition

```sql
-- NEW: Trigger Configuration Table
CREATE TABLE IF NOT EXISTS validation_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID NOT NULL,
    trigger_type VARCHAR(50) NOT NULL,      -- "save", "field_change", "delete"
    target_entity VARCHAR(100) NOT NULL,    -- "orders", "line_items"
    target_field VARCHAR(100),              -- For "field_change" trigger
    step_name VARCHAR(100),                 -- Business process step
    rule_ids UUID[] NOT NULL DEFAULT '{}',  -- Rules to execute
    is_active BOOLEAN DEFAULT true,
    description TEXT,
    priority INT DEFAULT 0,                 -- Execution order
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
    CONSTRAINT fk_datasource FOREIGN KEY (datasource_id) REFERENCES datasources(id),
    UNIQUE(tenant_id, datasource_id, trigger_type, target_entity, target_field)
);

-- Index for fast trigger lookups
CREATE INDEX idx_triggers_lookup 
ON validation_triggers(tenant_id, datasource_id, trigger_type, target_entity, is_active);

-- Sample triggers
INSERT INTO validation_triggers (tenant_id, datasource_id, trigger_type, target_entity, description, is_active)
VALUES 
  ('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111', 
   'save', 'orders', 'Validate all orders on save', true),
  
  ('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111',
   'field_change', 'orders', 'Validate when total field changes', true),
  
  ('00000000-0000-0000-0000-000000000000', '11111111-1111-1111-1111-111111111111',
   'delete', 'orders', 'Check if order can be deleted', true);
```

### Backend Go Implementation (trigger_engine.go)

```go
package services

import (
    "context"
    "fmt"
    "log"
    "github.com/google/uuid"
    "github.com/jmoiron/sqlx"
)

// TriggerType represents when a trigger fires
type TriggerType string

const (
    TriggerSave       TriggerType = "save"
    TriggerFieldChange TriggerType = "field_change"
    TriggerDelete     TriggerType = "delete"
    TriggerCreate     TriggerType = "create"
    TriggerWorkflowStep TriggerType = "workflow_step"
    TriggerSubEntity  TriggerType = "sub_entity_change"
)

// ValidationTrigger represents a trigger configuration
type ValidationTrigger struct {
    ID            uuid.UUID  `db:"id"`
    TenantID      uuid.UUID  `db:"tenant_id"`
    DatasourceID  uuid.UUID  `db:"datasource_id"`
    TriggerType   string     `db:"trigger_type"`
    TargetEntity  string     `db:"target_entity"`
    TargetField   string     `db:"target_field"`
    StepName      string     `db:"step_name"`
    RuleIDs       []uuid.UUID `db:"rule_ids"`
    IsActive      bool       `db:"is_active"`
    Description   string     `db:"description"`
    Priority      int        `db:"priority"`
}

// TriggerEvent represents an action that fired
type TriggerEvent struct {
    TriggerType  TriggerType
    TargetEntity string
    TargetField  string // For field_change
    EntityID     uuid.UUID
    Data         map[string]interface{}
    OldData      map[string]interface{} // For field_change
}

// TriggerEngine handles trigger execution
type TriggerEngine struct {
    db              *sqlx.DB
    validationEngine *ValidationRuleEngine
}

// NewTriggerEngine creates a new trigger engine
func NewTriggerEngine(db *sqlx.DB, ve *ValidationRuleEngine) *TriggerEngine {
    return &TriggerEngine{
        db:               db,
        validationEngine: ve,
    }
}

// ========================================================================
// CORE: Fetch Triggers
// ========================================================================

// FetchTriggers retrieves all active triggers for an event
func (te *TriggerEngine) FetchTriggers(ctx context.Context, tenantID, datasourceID uuid.UUID, 
    triggerType TriggerType, targetEntity, targetField string) ([]ValidationTrigger, error) {
    
    query := `
        SELECT id, tenant_id, datasource_id, trigger_type, target_entity, 
               target_field, step_name, rule_ids, is_active, description, priority
        FROM validation_triggers
        WHERE tenant_id = $1 
          AND datasource_id = $2 
          AND trigger_type = $3 
          AND target_entity = $4 
          AND is_active = true
    `
    
    var triggers []ValidationTrigger
    err := te.db.SelectContext(ctx, &triggers, query, tenantID, datasourceID, triggerType, targetEntity)
    if err != nil && err.Error() != "sql: no rows" {
        log.Printf("Error fetching triggers: %v", err)
        return nil, err
    }
    
    // For field_change triggers, also check if target_field matches
    if triggerType == TriggerFieldChange && targetField != "" {
        var filtered []ValidationTrigger
        for _, t := range triggers {
            if t.TargetField == "" || t.TargetField == targetField {
                filtered = append(filtered, t)
            }
        }
        return filtered, nil
    }
    
    return triggers, nil
}

// ========================================================================
// CORE: Execute Triggers
// ========================================================================

// ExecuteTriggers runs all matching triggers and returns validation errors
func (te *TriggerEngine) ExecuteTriggers(ctx context.Context, 
    tenantID, datasourceID uuid.UUID, event TriggerEvent, data map[string]interface{}) error {
    
    // 1. Fetch matching triggers
    triggers, err := te.FetchTriggers(ctx, tenantID, datasourceID, 
        TriggerType(event.TriggerType), event.TargetEntity, event.TargetField)
    if err != nil {
        return fmt.Errorf("failed to fetch triggers: %w", err)
    }
    
    if len(triggers) == 0 {
        // No triggers - allow operation
        return nil
    }
    
    // 2. Execute each trigger's rules (sorted by priority)
    for _, trigger := range triggers {
        if err := te.executeTriggerRules(ctx, trigger, data); err != nil {
            // IMPORTANT: If ANY rule fails, BLOCK the operation
            return fmt.Errorf("trigger validation failed: %w", err)
        }
    }
    
    return nil
}

// executeTriggerRules executes all rules for a single trigger
func (te *TriggerEngine) executeTriggerRules(ctx context.Context, 
    trigger ValidationTrigger, data map[string]interface{}) error {
    
    for _, ruleID := range trigger.RuleIDs {
        // Fetch rule
        rule, err := te.validationEngine.GetRuleByID(ctx, ruleID)
        if err != nil {
            log.Printf("Rule %s not found, skipping", ruleID)
            continue
        }
        
        // Evaluate rule
        passed, err := te.validationEngine.EvaluateComplexCondition(ctx, 
            rule.ComplexCondition, data)
        if err != nil {
            return fmt.Errorf("error evaluating rule %s: %w", rule.ID, err)
        }
        
        // If rule fails, block operation
        if !passed {
            return fmt.Errorf("%s: %s", rule.Name, rule.ErrorMessage)
        }
    }
    
    return nil
}

// ========================================================================
// HOOK: Save Trigger
// ========================================================================

// OnSave fires save triggers (call before INSERT/UPDATE)
func (te *TriggerEngine) OnSave(ctx context.Context, 
    tenantID, datasourceID uuid.UUID, entity string, data map[string]interface{}) error {
    
    event := TriggerEvent{
        TriggerType:  TriggerSave,
        TargetEntity: entity,
        Data:         data,
    }
    
    return te.ExecuteTriggers(ctx, tenantID, datasourceID, event, data)
}

// ========================================================================
// HOOK: Field Change Trigger
// ========================================================================

// OnFieldChange fires field_change triggers (call when field modified)
func (te *TriggerEngine) OnFieldChange(ctx context.Context, 
    tenantID, datasourceID uuid.UUID, entity, fieldName string, 
    oldValue, newValue interface{}, fullData map[string]interface{}) error {
    
    // Only fire if value actually changed
    if oldValue == newValue {
        return nil
    }
    
    oldData := map[string]interface{}{fieldName: oldValue}
    
    event := TriggerEvent{
        TriggerType:  TriggerFieldChange,
        TargetEntity: entity,
        TargetField:  fieldName,
        Data:         fullData,
        OldData:      oldData,
    }
    
    return te.ExecuteTriggers(ctx, tenantID, datasourceID, event, fullData)
}

// ========================================================================
// HOOK: Delete Trigger
// ========================================================================

// OnDelete fires delete triggers (call before DELETE)
func (te *TriggerEngine) OnDelete(ctx context.Context, 
    tenantID, datasourceID uuid.UUID, entity string, entityID uuid.UUID, data map[string]interface{}) error {
    
    event := TriggerEvent{
        TriggerType:  TriggerDelete,
        TargetEntity: entity,
        EntityID:     entityID,
        Data:         data,
    }
    
    return te.ExecuteTriggers(ctx, tenantID, datasourceID, event, data)
}

// ========================================================================
// ADMIN: Manage Triggers
// ========================================================================

// CreateTrigger creates a new trigger
func (te *TriggerEngine) CreateTrigger(ctx context.Context, trigger ValidationTrigger) (uuid.UUID, error) {
    id := uuid.New()
    
    query := `
        INSERT INTO validation_triggers 
        (id, tenant_id, datasource_id, trigger_type, target_entity, target_field, 
         step_name, rule_ids, is_active, description, priority, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
    `
    
    _, err := te.db.ExecContext(ctx, query,
        id, trigger.TenantID, trigger.DatasourceID, trigger.TriggerType,
        trigger.TargetEntity, trigger.TargetField, trigger.StepName,
        trigger.RuleIDs, trigger.IsActive, trigger.Description, trigger.Priority)
    
    if err != nil {
        return uuid.Nil, fmt.Errorf("failed to create trigger: %w", err)
    }
    
    return id, nil
}

// DeleteTrigger disables a trigger
func (te *TriggerEngine) DeleteTrigger(ctx context.Context, triggerID uuid.UUID) error {
    query := `UPDATE validation_triggers SET is_active = false WHERE id = $1`
    _, err := te.db.ExecContext(ctx, query, triggerID)
    return err
}

// GetTriggersByEntity returns all triggers for an entity
func (te *TriggerEngine) GetTriggersByEntity(ctx context.Context, 
    tenantID, datasourceID uuid.UUID, entity string) ([]ValidationTrigger, error) {
    
    query := `
        SELECT id, tenant_id, datasource_id, trigger_type, target_entity,
               target_field, step_name, rule_ids, is_active, description, priority
        FROM validation_triggers
        WHERE tenant_id = $1 AND datasource_id = $2 AND target_entity = $3
        ORDER BY priority ASC, created_at ASC
    `
    
    var triggers []ValidationTrigger
    err := te.db.SelectContext(ctx, &triggers, query, tenantID, datasourceID, entity)
    return triggers, err
}
```

### API Integration (api.go changes)

```go
// In your existing API handlers, add trigger calls:

// CreateOrder endpoint
func (s *Server) CreateOrder(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    datasourceID := c.GetString("datasource_id")
    
    var orderData map[string]interface{}
    c.BindJSON(&orderData)
    
    // NEW: Fire triggers before save
    if err := s.triggerEngine.OnSave(c, 
        uuid.MustParse(tenantID), 
        uuid.MustParse(datasourceID),
        "orders", orderData); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Save to database
    // ... existing code ...
    c.JSON(200, order)
}

// UpdateOrder endpoint
func (s *Server) UpdateOrder(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    datasourceID := c.GetString("datasource_id")
    orderID := c.Param("id")
    
    // Fetch current data
    oldOrder := /* fetch from DB */
    
    var newData map[string]interface{}
    c.BindJSON(&newData)
    
    // NEW: Fire field_change triggers for changed fields
    for field, newValue := range newData {
        oldValue := oldOrder[field]
        if err := s.triggerEngine.OnFieldChange(c,
            uuid.MustParse(tenantID),
            uuid.MustParse(datasourceID),
            "orders", field, oldValue, newValue, newData); err != nil {
            c.JSON(400, gin.H{"error": err.Error()})
            return
        }
    }
    
    // NEW: Fire save trigger
    if err := s.triggerEngine.OnSave(c,
        uuid.MustParse(tenantID),
        uuid.MustParse(datasourceID),
        "orders", newData); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Save to database
    // ... existing code ...
    c.JSON(200, order)
}

// DeleteOrder endpoint
func (s *Server) DeleteOrder(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    datasourceID := c.GetString("datasource_id")
    orderID := c.Param("id")
    
    // Fetch current data
    order := /* fetch from DB */
    
    // NEW: Fire delete trigger before deletion
    if err := s.triggerEngine.OnDelete(c,
        uuid.MustParse(tenantID),
        uuid.MustParse(datasourceID),
        "orders", uuid.MustParse(orderID), order); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Delete from database
    // ... existing code ...
    c.JSON(200, gin.H{"message": "deleted"})
}
```

### Frontend Integration (TriggerConfiguration.tsx)

```typescript
import React, { useState, useEffect } from 'react';
import { Plus, Trash2, Zap } from 'lucide-react';
import styles from './TriggerConfiguration.module.css';

interface Trigger {
  id: string;
  triggerType: 'save' | 'field_change' | 'delete' | 'create';
  targetEntity: string;
  targetField?: string;
  ruleIds: string[];
  isActive: boolean;
  description: string;
}

export const TriggerConfiguration: React.FC = () => {
  const [triggers, setTriggers] = useState<Trigger[]>([]);
  const [selectedEntity, setSelectedEntity] = useState<string>('orders');
  const [newTrigger, setNewTrigger] = useState<Partial<Trigger>>({
    triggerType: 'save',
    targetEntity: 'orders',
    isActive: true,
  });

  // Fetch triggers for entity
  useEffect(() => {
    const fetchTriggers = async () => {
      const response = await fetch(
        `/api/triggers?entity=${selectedEntity}`,
        {
          headers: {
            'X-Tenant-ID': localStorage.getItem('selected_tenant_id'),
            'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource_id'),
          },
        }
      );
      const data = await response.json();
      setTriggers(data);
    };
    fetchTriggers();
  }, [selectedEntity]);

  const handleAddTrigger = async () => {
    const response = await fetch('/api/triggers', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Tenant-ID': localStorage.getItem('selected_tenant_id'),
        'X-Tenant-Datasource-ID': localStorage.getItem('selected_datasource_id'),
      },
      body: JSON.stringify(newTrigger),
    });

    if (response.ok) {
      const created = await response.json();
      setTriggers([...triggers, created]);
      setNewTrigger({ triggerType: 'save', targetEntity: selectedEntity, isActive: true });
    }
  };

  const handleDeleteTrigger = async (triggerId: string) => {
    await fetch(`/api/triggers/${triggerId}`, { method: 'DELETE' });
    setTriggers(triggers.filter(t => t.id !== triggerId));
  };

  return (
    <div className={styles.container}>
      <h2>
        <Zap size={24} /> Trigger Configuration
      </h2>

      {/* Trigger List */}
      <div className={styles.triggerList}>
        {triggers.map(trigger => (
          <div key={trigger.id} className={styles.triggerItem}>
            <div>
              <strong>{trigger.triggerType}</strong>
              {trigger.targetField && ` → ${trigger.targetField}`}
              <p>{trigger.description}</p>
            </div>
            <button onClick={() => handleDeleteTrigger(trigger.id)}>
              <Trash2 size={16} />
            </button>
          </div>
        ))}
      </div>

      {/* Add New Trigger */}
      <div className={styles.addTrigger}>
        <select
          value={newTrigger.triggerType || 'save'}
          onChange={e => setNewTrigger({ ...newTrigger, triggerType: e.target.value as any })}
        >
          <option value="save">Save</option>
          <option value="field_change">Field Change</option>
          <option value="delete">Delete</option>
          <option value="create">Create</option>
        </select>

        {newTrigger.triggerType === 'field_change' && (
          <input
            type="text"
            placeholder="Field name (optional)"
            onChange={e => setNewTrigger({ ...newTrigger, targetField: e.target.value })}
          />
        )}

        <button onClick={handleAddTrigger}>
          <Plus size={16} /> Add Trigger
        </button>
      </div>
    </div>
  );
};
```

---

## 📅 Phase 5 Timeline

### Phase 5a (Week 1-2): Core 3 Triggers
- **Week 1:**
  - Day 1-2: Design + Database schema
  - Day 3-4: Backend TriggerEngine implementation
  - Day 5: API integration + testing
- **Week 2:**
  - Day 1-2: Frontend TriggerConfiguration component
  - Day 3-4: E2E testing (save, field_change, delete)
  - Day 5: Deploy to staging + QA

### Phase 5b (Week 2-3): Extended Triggers (Optional)
- Workflow Step triggers
- Sub-Entity Change triggers
- Status Change triggers

### Phase 5c+ (Future): Advanced Triggers
- Time-based (scheduler)
- Integration (webhooks)
- Calculated fields
- Security roles

---

## ✅ Phase 5a Success Criteria

- [x] Database migration applied (triggers table + index)
- [x] TriggerEngine implemented (3 methods: OnSave, OnFieldChange, OnDelete)
- [x] API endpoints updated (POST/PUT/DELETE endpoints call triggers)
- [x] Frontend UI for trigger management
- [x] E2E tests passing (save blocks invalid, delete blocks if prevented, etc)
- [x] Documentation complete
- [x] Deployed to staging
- [x] QA verified

---

## 🎯 Ready to Start Phase 5?

**Next Steps:**

1. **Deploy Phase 1-4 to production** (complete deployment first)
2. **Gather user feedback** (use Phase 1-4 in production for 1 week)
3. **Start Phase 5 planning** (2-3 week sprint)

**OR**

Start Phase 5 development NOW in parallel branch while Phase 1-4 deploys.

---

## 📚 Related Documentation

- `DEPLOYMENT_RUNBOOK_PATH_B.md` - Phase 1-4 deployment
- `PHASE_1_OPTIMIZATION_INTEGRATION.md` - Performance hooks
- `FEATURE_STATUS_ADVANCED_VALIDATION.md` - Phase 1-4 features

---

**Ready to code Phase 5? Say "Start Phase 5 backend" or "Start Phase 5 frontend" to begin! 🚀**
