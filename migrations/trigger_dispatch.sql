-- ============================================================================
-- PHASE 6A: TRIGGER DISPATCH SYSTEM - DATABASE SCHEMA
-- ============================================================================
-- This migration defines the validation_triggers table that powers the
-- Workday-style application-layer trigger dispatch system.
--
-- A "trigger" is a mapping between:
--   - When something happens: save, create, delete, field_change, etc.
--   - What entity it happens to: orders, customers, etc.
--   - What validation rules to run: array of rule IDs
--
-- Unlike database triggers, these are APPLICATION-LAYER events that fire
-- before data persistence, allowing Workday-style "no stalls" validation.
--
-- Workday's 13 Trigger Types (8 implemented here):
--   1. Save         ✅
--   2. Field Change ✅
--   3. Delete       ✅
--   4. Create       ✅
--   5. Workflow Step ✅
--   6. Sub-Entity   ✅
--   7. Relationship ✅
--   8. Status Change ✅
--   9. Bulk Load    🔄
--  10. Integration  🔄
--  11. Time-Based   🔄
--  12. Calculated   🔄
--  13. Security Role 🔄
-- ============================================================================

-- Main trigger configuration table
CREATE TABLE IF NOT EXISTS validation_triggers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    -- Trigger type: save, create, delete, field_change, workflow_step, etc.
    trigger_type VARCHAR(50) NOT NULL,
    
    -- Target entity: orders, customers, employees, etc.
    target_entity VARCHAR(50) NOT NULL,
    
    -- Optional step name for workflow_step triggers
    step_name VARCHAR(100),
    
    -- Array of validation rule IDs to execute for this trigger
    rule_ids UUID[] NOT NULL DEFAULT '{}',
    
    -- Optional metadata (trigger-specific config)
    meta JSONB,
    
    -- Active/inactive flag for soft disable
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    
    -- Audit fields
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255),
    updated_by VARCHAR(255),
    
    CONSTRAINT fk_validation_triggers_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id)
);

-- Optimized indexes for fast trigger lookup
CREATE INDEX IF NOT EXISTS idx_validation_triggers_lookup 
    ON validation_triggers(tenant_id, trigger_type, target_entity, is_active)
    WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_validation_triggers_step 
    ON validation_triggers(tenant_id, trigger_type, target_entity, step_name)
    WHERE step_name IS NOT NULL AND is_active = true;

CREATE INDEX IF NOT EXISTS idx_validation_triggers_tenant 
    ON validation_triggers(tenant_id, is_active);

-- Trigger event audit log (optional, for compliance/debugging)
CREATE TABLE IF NOT EXISTS trigger_dispatch_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    trigger_id UUID,
    entity VARCHAR(50),
    action VARCHAR(50),
    status VARCHAR(20), -- passed, failed, blocked
    error_message TEXT,
    execution_time_ms INT,
    request_id VARCHAR(255),
    user_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    
    CONSTRAINT fk_trigger_events_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id),
    CONSTRAINT fk_trigger_events_trigger FOREIGN KEY (trigger_id) REFERENCES validation_triggers(id)
);

CREATE INDEX IF NOT EXISTS idx_trigger_events_lookup 
    ON trigger_dispatch_events(tenant_id, created_at DESC, status);

-- ============================================================================
-- SAMPLE DATA: 5 Real-World Trigger Configurations
-- ============================================================================

-- Trigger 1: Order Total Validation (CREATE + SAVE)
-- Ensures all new and updated orders have positive totals
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'create', 
    'orders', 
    ARRAY['rule-positive-total'::UUID, 'rule-valid-customer'::UUID],
    jsonb_build_object('description', 'Validates new order totals must be positive'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 2: Order Total Check on SAVE
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'save', 
    'orders', 
    ARRAY['rule-positive-total'::UUID],
    jsonb_build_object('description', 'Validates updated order totals'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 3: Delete Protection (cannot delete completed orders)
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'delete', 
    'orders', 
    ARRAY['rule-cannot-delete-shipped'::UUID],
    jsonb_build_object('description', 'Prevent deletion of shipped orders'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 4: Status Transition Validation (pending → approved → completed)
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, step_name, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'status_change', 
    'orders', 
    'approval', 
    ARRAY['rule-manager-approval'::UUID, 'rule-limit-5k'::UUID],
    jsonb_build_object('description', 'Only managers can approve orders < $5000'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 5: Field Change Validation (total field onChange)
-- Quick validation for form UIs
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'field_change', 
    'orders', 
    ARRAY['rule-positive-total'::UUID],
    jsonb_build_object('description', 'Validate order total field on user input'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 6: Sub-Entity Validation (line items in orders)
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'sub_entity_change', 
    'order_items', 
    ARRAY['rule-item-positive-price'::UUID, 'rule-item-qty-gt-zero'::UUID],
    jsonb_build_object('description', 'Validate line items in orders'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- Trigger 7: Relationship Change (customer reassignment)
INSERT INTO validation_triggers 
(tenant_id, trigger_type, target_entity, rule_ids, meta, is_active, created_by)
SELECT 
    tenant_id, 
    'relationship_change', 
    'orders', 
    ARRAY['rule-customer-must-exist'::UUID],
    jsonb_build_object('description', 'Validate customer exists when reassigning order'),
    true,
    'system'
FROM tenants LIMIT 1
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- HELPER VIEW: Active Triggers (for debugging)
-- ============================================================================
CREATE OR REPLACE VIEW v_active_triggers AS
SELECT 
    vt.id,
    vt.tenant_id,
    vt.trigger_type,
    vt.target_entity,
    vt.step_name,
    array_length(vt.rule_ids, 1) as rule_count,
    vt.meta,
    vt.created_at,
    vt.updated_at
FROM validation_triggers vt
WHERE vt.is_active = true
ORDER BY vt.trigger_type, vt.target_entity;

-- ============================================================================
-- HELPER VIEW: Recent Trigger Events (for audit trail)
-- ============================================================================
CREATE OR REPLACE VIEW v_trigger_events_summary AS
SELECT 
    DATE_TRUNC('hour', tde.created_at) as hour,
    tde.tenant_id,
    tde.entity,
    tde.action,
    tde.status,
    COUNT(*) as event_count,
    AVG(tde.execution_time_ms) as avg_execution_time_ms
FROM trigger_dispatch_events tde
WHERE tde.created_at >= NOW() - INTERVAL '24 hours'
GROUP BY 1, 2, 3, 4, 5
ORDER BY 1 DESC, event_count DESC;

-- ============================================================================
-- QUERIES FOR COMMON OPERATIONS
-- ============================================================================
-- Find all triggers for an entity/action:
-- SELECT * FROM validation_triggers 
-- WHERE tenant_id = '...' 
--   AND trigger_type = 'create' 
--   AND target_entity = 'orders'
--   AND is_active = true;

-- Disable a trigger (soft delete):
-- UPDATE validation_triggers 
-- SET is_active = false, updated_at = NOW() 
-- WHERE id = '...';

-- View recent events:
-- SELECT * FROM trigger_dispatch_events 
-- WHERE tenant_id = '...' 
-- ORDER BY created_at DESC LIMIT 20;

-- Count triggers by entity:
-- SELECT target_entity, COUNT(*) as trigger_count
-- FROM validation_triggers 
-- WHERE tenant_id = '...' AND is_active = true
-- GROUP BY target_entity;
