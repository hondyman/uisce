-- ============================================================================
-- WORKFLOW & SCREEN CONFIGURATION SCHEMA
-- Workday-Inspired Business Process Model for Northwind
-- ============================================================================

-- ============================================================================
-- 1. WORKFLOW RULES TABLE (Low-Code Workflow Configuration)
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    workflow_name VARCHAR(100) NOT NULL,     -- e.g., "OrderProcessing", "EmployeeHire"
    step_name VARCHAR(100) NOT NULL,         -- e.g., "ApproveOrder", "BackgroundCheck"
    step_order INTEGER NOT NULL DEFAULT 0,   -- Sequence in workflow
    condition_json JSONB NOT NULL,           -- e.g., {"and": [{"field": "order_total", "operator": ">=", "value": 1000}]}
    action_on_success VARCHAR(255),          -- e.g., "route:order_approved.queue"
    action_on_failure VARCHAR(255),          -- e.g., "notify:manager"
    error_message TEXT,                      -- User-friendly error for UI
    timeout_seconds INTEGER DEFAULT 3600,    -- Step timeout
    retry_count INTEGER DEFAULT 0,           -- Max retries
    is_active BOOLEAN DEFAULT TRUE,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT unique_workflow_step 
        UNIQUE (tenant_id, workflow_name, step_name),
    CONSTRAINT valid_timeout 
        CHECK (timeout_seconds > 0)
);

CREATE INDEX idx_workflow_rules_lookup 
    ON workflow_rules(tenant_id, workflow_name, step_name);
CREATE INDEX idx_workflow_rules_active 
    ON workflow_rules(tenant_id, is_active) WHERE is_active = TRUE;


-- ============================================================================
-- 2. WORKFLOW HISTORY TABLE (Audit Trail & Execution Tracking)
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(100) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    bo_type VARCHAR(50) NOT NULL,           -- e.g., "orders", "employees", "products"
    bo_id UUID NOT NULL,                    -- Business Object ID
    status VARCHAR(20) NOT NULL             -- "pending", "success", "failure", "skipped"
        CHECK (status IN ('pending', 'success', 'failure', 'skipped', 'timeout')),
    details JSONB,                          -- Execution details: {"old_value": "...", "new_value": "...", "error": "..."}
    user_id UUID NOT NULL,
    temporal_workflow_id VARCHAR(255),      -- Link to Temporal workflow
    temporal_run_id VARCHAR(255),           -- Link to Temporal run
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE
);

CREATE INDEX idx_workflow_history_bo 
    ON workflow_history(bo_type, bo_id);
CREATE INDEX idx_workflow_history_workflow 
    ON workflow_history(tenant_id, workflow_name, created_at DESC);
CREATE INDEX idx_workflow_history_status 
    ON workflow_history(status) WHERE status IN ('pending', 'failure');


-- ============================================================================
-- 3. SCREEN CONFIGURATION TABLE (Workday-Style Screen Builder)
-- ============================================================================
CREATE TABLE IF NOT EXISTS screen_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bo_type VARCHAR(50) NOT NULL,           -- "customers", "orders", "employees", "products"
    screen_name VARCHAR(100) NOT NULL,      -- e.g., "CustomerDetails", "OrderCreation"
    screen_type VARCHAR(50) DEFAULT 'detail' 
        CHECK (screen_type IN ('list', 'detail', 'create', 'edit', 'summary')),
    layout_json JSONB NOT NULL,             -- [{"field": "name", "label": "Name", "type": "text", "order": 1}]
    filters_json JSONB DEFAULT '[]',        -- [{"field": "status", "operator": "=", "value": "active"}]
    actions_json JSONB DEFAULT '["save", "delete", "cancel"]',  -- Available buttons
    permissions_json JSONB DEFAULT '{}',    -- {"admin": ["save", "delete"], "user": ["view", "save"]}
    is_published BOOLEAN DEFAULT FALSE,
    created_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    CONSTRAINT unique_screen_name UNIQUE (tenant_id, bo_type, screen_name)
);

CREATE INDEX idx_screen_configs_bo 
    ON screen_configs(tenant_id, bo_type, is_published);


-- ============================================================================
-- 4. WORKFLOW TEMPLATE TABLE (Predefined Workflow Blueprints)
-- ============================================================================
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,                         -- NULL = global template
    workflow_name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    bo_type VARCHAR(50) NOT NULL,           -- Which BO triggers this workflow
    trigger_event VARCHAR(100) NOT NULL,    -- e.g., "order.created", "employee.hired"
    workflow_json JSONB NOT NULL,           -- Full workflow definition with steps
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_workflow_templates_bo 
    ON workflow_templates(bo_type, trigger_event);


-- ============================================================================
-- 5. UPDATE TIMESTAMP TRIGGERS (Auto-Update Timestamps)
-- ============================================================================
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_workflow_rules_ts ON workflow_rules;
CREATE TRIGGER update_workflow_rules_ts
BEFORE UPDATE ON workflow_rules
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_workflow_history_ts ON workflow_history;
CREATE TRIGGER update_workflow_history_ts
BEFORE UPDATE ON workflow_history
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_screen_configs_ts ON screen_configs;
CREATE TRIGGER update_screen_configs_ts
BEFORE UPDATE ON screen_configs
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();

DROP TRIGGER IF EXISTS update_workflow_templates_ts ON workflow_templates;
CREATE TRIGGER update_workflow_templates_ts
BEFORE UPDATE ON workflow_templates
FOR EACH ROW
EXECUTE FUNCTION update_timestamp();


-- ============================================================================
-- 6. SEED DATA (Sample Workflows for Northwind)
-- ============================================================================

-- Helper function to get or create system tenant (adjust UUID as needed)
DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    v_tenant_id := '00000000-0000-0000-0000-000000000001'::UUID;
    
    -- Insert only if not exists (safe for idempotent runs)
    IF NOT EXISTS (SELECT 1 FROM tenants WHERE tenant_id = v_tenant_id) THEN
        INSERT INTO tenants (tenant_id, tenant_name) 
        VALUES (v_tenant_id, 'Default System Tenant');
    END IF;
    
    -- Order Processing Workflow - Step 1: Validate Order
    INSERT INTO workflow_rules 
        (tenant_id, workflow_name, step_name, step_order, condition_json, action_on_success, action_on_failure, error_message, created_by)
    VALUES 
        (v_tenant_id, 'OrderProcessing', 'ValidateOrder', 1,
         '{"and": [{"field": "order_total", "operator": ">=", "value": 1}]}'::jsonb,
         'route:order_validated.queue', 'notify:order_failed.queue',
         'Order total must be at least $1', '00000000-0000-0000-0000-000000000002'::UUID)
    ON CONFLICT (tenant_id, workflow_name, step_name) DO NOTHING;
    
    -- Order Processing Workflow - Step 2: Approve Order
    INSERT INTO workflow_rules 
        (tenant_id, workflow_name, step_name, step_order, condition_json, action_on_success, action_on_failure, error_message, created_by)
    VALUES 
        (v_tenant_id, 'OrderProcessing', 'ApproveOrder', 2,
         '{"and": [{"field": "order_total", "operator": ">=", "value": 1000}]}'::jsonb,
         'route:order_approved.queue', 'notify:manager_approval_needed.queue',
         'Order total must be at least $1000 for auto-approval', '00000000-0000-0000-0000-000000000002'::UUID)
    ON CONFLICT (tenant_id, workflow_name, step_name) DO NOTHING;
    
    -- Employee Hire Workflow - Step 1: Background Check
    INSERT INTO workflow_rules 
        (tenant_id, workflow_name, step_name, step_order, condition_json, action_on_success, action_on_failure, error_message, created_by)
    VALUES 
        (v_tenant_id, 'EmployeeHire', 'BackgroundCheck', 1,
         '{"and": [{"field": "hire_date", "operator": "not_null"}]}'::jsonb,
         'route:employee_background_check.queue', 'notify:hr_check_failed.queue',
         'Hire date is required', '00000000-0000-0000-0000-000000000002'::UUID)
    ON CONFLICT (tenant_id, workflow_name, step_name) DO NOTHING;
    
    -- Product Inventory Update - Step 1: Check Stock Levels
    INSERT INTO workflow_rules 
        (tenant_id, workflow_name, step_name, step_order, condition_json, action_on_success, action_on_failure, error_message, created_by)
    VALUES 
        (v_tenant_id, 'ProductInventoryUpdate', 'CheckStockLevels', 1,
         '{"and": [{"field": "stock_change", "operator": ">", "value": -999999}]}'::jsonb,
         'route:inventory_updated.queue', 'notify:inventory_error.queue',
         'Invalid stock adjustment', '00000000-0000-0000-0000-000000000002'::UUID)
    ON CONFLICT (tenant_id, workflow_name, step_name) DO NOTHING;

END
$$;

-- ============================================================================
-- 7. ENABLE RLS (Row-Level Security) FOR MULTI-TENANT ISOLATION
-- ============================================================================

ALTER TABLE workflow_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_history ENABLE ROW LEVEL SECURITY;
ALTER TABLE screen_configs ENABLE ROW LEVEL SECURITY;
ALTER TABLE workflow_templates ENABLE ROW LEVEL SECURITY;

-- RLS Policy: Users see only their tenant's workflows
CREATE POLICY workflow_rules_tenant_isolation ON workflow_rules
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY workflow_history_tenant_isolation ON workflow_history
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY screen_configs_tenant_isolation ON screen_configs
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY workflow_templates_public_or_tenant ON workflow_templates
    USING (tenant_id IS NULL OR tenant_id = current_setting('app.current_tenant_id', true)::UUID)
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
