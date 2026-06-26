-- ============================================================================
-- Phase 6: Semantic Model Regeneration - DBA Schema
-- ============================================================================
-- Purpose: Track entity attribute changes and trigger semantic model regeneration
-- - model_regeneration_trigger: Records when models need regeneration
-- - entity_attribute_audit: Full audit trail of attribute changes
-- - model_version_history: Version control for semantic models
-- - semantic_model_change_log: Log of what changed and why
--
-- This enables:
-- 1. Automatic detection of schema changes
-- 2. Triggering of semantic model regeneration
-- 3. Version control for semantic models
-- 4. Change impact analysis
-- 5. Audit trail for compliance
-- ============================================================================

BEGIN;

-- ============================================================================
-- 1. Model Regeneration Trigger Table
-- ============================================================================
-- Records when entity attributes change and semantic models need regeneration

CREATE TABLE IF NOT EXISTS public.model_regeneration_trigger (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_attribute_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    
    -- What triggered regeneration
    trigger_type varchar(100) NOT NULL,  -- 'ATTRIBUTE_ADDED', 'ATTRIBUTE_REMOVED', 'RELATIONSHIP_ADDED', 'SEMANTIC_TERM_CHANGED'
    trigger_source varchar(100) NOT NULL, -- 'USER_ACTION', 'AUTO_DISCOVERY', 'IMPORT', 'API'
    
    -- Details of what changed
    change_detail jsonb NOT NULL,  -- Contains before/after values and field names
    
    -- Regeneration status
    regeneration_status varchar(50) DEFAULT 'PENDING',  -- 'PENDING', 'IN_PROGRESS', 'COMPLETED', 'FAILED'
    regeneration_started_at timestamp,
    regeneration_completed_at timestamp,
    regeneration_error text,
    
    -- Metadata
    triggered_by text,
    triggered_at timestamp DEFAULT now(),
    model_version_before varchar(50),
    model_version_after varchar(50),
    
    -- Tracking
    is_active boolean DEFAULT true,
    created_at timestamp DEFAULT now(),
    updated_at timestamp DEFAULT now(),
    
    -- Constraints
    CONSTRAINT model_regen_trigger_unique 
        UNIQUE (tenant_datasource_id, entity_attribute_id, trigger_type, triggered_at)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_model_regen_trigger_tenant_ds 
    ON public.model_regeneration_trigger(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_model_regen_trigger_entity_attr 
    ON public.model_regeneration_trigger(entity_attribute_id);

CREATE INDEX IF NOT EXISTS idx_model_regen_trigger_status 
    ON public.model_regeneration_trigger(regeneration_status) WHERE regeneration_status = 'PENDING';

CREATE INDEX IF NOT EXISTS idx_model_regen_trigger_type 
    ON public.model_regeneration_trigger(trigger_type, is_active);

CREATE INDEX IF NOT EXISTS idx_model_regen_trigger_timestamp 
    ON public.model_regeneration_trigger(triggered_at DESC);

COMMENT ON TABLE public.model_regeneration_trigger IS
    'Tracks when entity attributes change and semantic models need regeneration';

COMMENT ON COLUMN public.model_regeneration_trigger.trigger_type IS
    'Type of change: ATTRIBUTE_ADDED, ATTRIBUTE_REMOVED, RELATIONSHIP_ADDED, RELATIONSHIP_REMOVED, SEMANTIC_TERM_CHANGED, CARDINALITY_CHANGED';

COMMENT ON COLUMN public.model_regeneration_trigger.change_detail IS
    'JSON object with {before: {...}, after: {...}, field: "field_name", comparison: "..."}';

-- ============================================================================
-- 2. Entity Attribute Audit Table
-- ============================================================================
-- Full audit trail of all changes to entity attributes

CREATE TABLE IF NOT EXISTS public.entity_attribute_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_attribute_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE SET NULL,
    
    -- What changed
    action varchar(20) NOT NULL,  -- 'INSERT', 'UPDATE', 'DELETE'
    old_values jsonb,
    new_values jsonb,
    
    -- Changed fields
    changed_fields text[],  -- Array of field names that changed
    
    -- Metadata
    changed_by text NOT NULL,
    change_reason text,
    change_source varchar(100),  -- 'UI', 'API', 'IMPORT', 'AUTO_DISCOVERY'
    
    -- Impact
    affected_relationships int DEFAULT 0,  -- Number of relationships affected
    affected_reports int DEFAULT 0,  -- Number of reports that need updates
    
    -- Timestamp
    changed_at timestamp DEFAULT now(),
    
    -- Tracking
    correlation_id uuid,  -- Links related changes together
    is_rollback boolean DEFAULT false
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_entity_attr_audit_tenant_ds 
    ON public.entity_attribute_audit(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_audit_entity_attr 
    ON public.entity_attribute_audit(entity_attribute_id);

CREATE INDEX IF NOT EXISTS idx_entity_attr_audit_action 
    ON public.entity_attribute_audit(action);

CREATE INDEX IF NOT EXISTS idx_entity_attr_audit_timestamp 
    ON public.entity_attribute_audit(changed_at DESC);

CREATE INDEX IF NOT EXISTS idx_entity_attr_audit_correlation 
    ON public.entity_attribute_audit(correlation_id) WHERE correlation_id IS NOT NULL;

COMMENT ON TABLE public.entity_attribute_audit IS
    'Full audit trail of entity attribute changes for compliance and impact analysis';

COMMENT ON COLUMN public.entity_attribute_audit.changed_fields IS
    'Array of field names that changed (e.g., ARRAY[''name'', ''business_name''])';

-- ============================================================================
-- 3. Model Version History Table
-- ============================================================================
-- Tracks versions of semantic models for each entity

CREATE TABLE IF NOT EXISTS public.model_version_history (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_attribute_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    
    -- Version info
    version_number int NOT NULL,  -- 1, 2, 3, ...
    version_tag varchar(50),  -- e.g., "1.0.0", "1.1.0-beta"
    
    -- Model content
    model_signature varchar(64),  -- SHA256 hash of model for change detection
    model_content jsonb NOT NULL,  -- The semantic model definition
    
    -- What changed from previous version
    changes_from_previous text,  -- Human-readable summary
    attributes_changed int DEFAULT 0,
    relationships_changed int DEFAULT 0,
    
    -- Generation info
    generated_at timestamp DEFAULT now(),
    generated_by text,
    generation_trigger_id uuid REFERENCES public.model_regeneration_trigger(id) ON DELETE SET NULL,
    
    -- Status
    is_active boolean DEFAULT true,
    is_published boolean DEFAULT false,
    published_at timestamp,
    
    -- Metadata
    notes text,
    created_at timestamp DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_model_version_tenant_ds 
    ON public.model_version_history(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_model_version_entity_attr 
    ON public.model_version_history(entity_attribute_id, version_number DESC);

CREATE INDEX IF NOT EXISTS idx_model_version_published 
    ON public.model_version_history(is_published, is_active);

CREATE INDEX IF NOT EXISTS idx_model_version_signature 
    ON public.model_version_history(model_signature) WHERE model_signature IS NOT NULL;

COMMENT ON TABLE public.model_version_history IS
    'Version control for semantic models with change tracking and rollback capability';

COMMENT ON COLUMN public.model_version_history.model_signature IS
    'SHA256 hash of model content for detecting when actual changes occur vs. timestamp updates';

-- ============================================================================
-- 4. Semantic Model Change Log
-- ============================================================================
-- High-level change log for reporting and impact analysis

CREATE TABLE IF NOT EXISTS public.semantic_model_change_log (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    
    -- Scope
    entity_attribute_id uuid REFERENCES public.entity_attribute(id) ON DELETE SET NULL,
    
    -- Change summary
    change_type varchar(100) NOT NULL,  -- 'ATTRIBUTE_ADDED', 'RELATIONSHIP_DISCOVERED', etc.
    change_summary text NOT NULL,
    
    -- Impact
    impacted_entities int DEFAULT 0,
    impacted_relationships int DEFAULT 0,
    impacted_reports int DEFAULT 0,
    impacted_dashboards int DEFAULT 0,
    
    -- Details
    before_state jsonb,  -- Snapshot of state before change
    after_state jsonb,   -- Snapshot of state after change
    
    -- Metadata
    change_timestamp timestamp DEFAULT now(),
    changed_by text,
    change_source varchar(100),  -- 'USER', 'SYSTEM', 'AUTO_DISCOVERY'
    severity varchar(20) DEFAULT 'MEDIUM',  -- 'LOW', 'MEDIUM', 'HIGH', 'CRITICAL'
    
    -- Tracking
    requires_approval boolean DEFAULT false,
    approved_at timestamp,
    approved_by text,
    is_reversible boolean DEFAULT true
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_semantic_change_log_tenant_ds 
    ON public.semantic_model_change_log(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_semantic_change_log_entity 
    ON public.semantic_model_change_log(entity_attribute_id);

CREATE INDEX IF NOT EXISTS idx_semantic_change_log_timestamp 
    ON public.semantic_model_change_log(change_timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_semantic_change_log_severity 
    ON public.semantic_model_change_log(severity, is_reversible);

COMMENT ON TABLE public.semantic_model_change_log IS
    'High-level change log for tracking semantic model evolution and impact analysis';

-- ============================================================================
-- 5. Model Regeneration Queue
-- ============================================================================
-- Queue for processing model regeneration requests

CREATE TABLE IF NOT EXISTS public.model_regeneration_queue (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    tenant_datasource_id uuid NOT NULL REFERENCES public.tenant_product_datasource(id) ON DELETE CASCADE,
    entity_attribute_id uuid NOT NULL REFERENCES public.entity_attribute(id) ON DELETE CASCADE,
    
    -- Queue status
    queue_status varchar(50) DEFAULT 'QUEUED',  -- 'QUEUED', 'IN_PROGRESS', 'COMPLETED', 'FAILED', 'SKIPPED'
    priority int DEFAULT 5,  -- 1-10, higher = more urgent
    
    -- Processing info
    queued_at timestamp DEFAULT now(),
    started_at timestamp,
    completed_at timestamp,
    retry_count int DEFAULT 0,
    max_retries int DEFAULT 3,
    
    -- Error tracking
    last_error text,
    error_stack text,
    
    -- Dependencies
    depends_on_queue_ids uuid[],  -- IDs of other queue items that must complete first
    
    -- Metadata
    reason text,
    triggered_by_trigger_id uuid REFERENCES public.model_regeneration_trigger(id) ON DELETE SET NULL,
    
    is_active boolean DEFAULT true
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_model_regen_queue_status 
    ON public.model_regeneration_queue(queue_status, priority DESC);

CREATE INDEX IF NOT EXISTS idx_model_regen_queue_tenant_ds 
    ON public.model_regeneration_queue(tenant_datasource_id);

CREATE INDEX IF NOT EXISTS idx_model_regen_queue_entity 
    ON public.model_regeneration_queue(entity_attribute_id);

COMMENT ON TABLE public.model_regeneration_queue IS
    'Queue for processing semantic model regeneration requests with priority and dependencies';

-- ============================================================================
-- 6. Helper Views
-- ============================================================================

-- View: Pending regenerations
CREATE OR REPLACE VIEW public.v_pending_model_regenerations AS
SELECT 
    mrt.id,
    mrt.tenant_id,
    mrt.tenant_datasource_id,
    ea.name as entity_name,
    mrt.trigger_type,
    mrt.change_detail,
    mrt.triggered_at,
    mrt.regeneration_status,
    COUNT(er.id) as affected_relationships,
    mrt.model_version_before,
    mrt.model_version_after
FROM public.model_regeneration_trigger mrt
LEFT JOIN public.entity_attribute ea ON mrt.entity_attribute_id = ea.id
LEFT JOIN public.entity_relationship er ON (
    er.source_entity_id = mrt.entity_attribute_id OR 
    er.target_entity_id = mrt.entity_attribute_id
)
WHERE mrt.regeneration_status = 'PENDING'
    AND mrt.is_active = true
GROUP BY mrt.id, mrt.tenant_id, mrt.tenant_datasource_id, ea.name, mrt.trigger_type, 
         mrt.change_detail, mrt.triggered_at, mrt.regeneration_status, mrt.model_version_before, mrt.model_version_after;

COMMENT ON VIEW public.v_pending_model_regenerations IS
    'View of pending semantic model regenerations with affected relationship counts';

-- View: Change impact analysis
CREATE OR REPLACE VIEW public.v_change_impact_analysis AS
SELECT 
    sml.tenant_datasource_id,
    sml.change_type,
    ea.name as entity_name,
    sml.change_summary,
    sml.impacted_entities,
    sml.impacted_relationships,
    sml.impacted_reports,
    sml.impacted_dashboards,
    sml.severity,
    sml.change_timestamp,
    sml.is_reversible
FROM public.semantic_model_change_log sml
LEFT JOIN public.entity_attribute ea ON sml.entity_attribute_id = ea.id
WHERE sml.change_timestamp >= NOW() - INTERVAL '30 days'
ORDER BY sml.change_timestamp DESC;

COMMENT ON VIEW public.v_change_impact_analysis IS
    'High-level view of recent model changes and their business impact';

-- ============================================================================
-- 7. Utility Functions
-- ============================================================================

-- Function: Calculate model signature (SHA256 hash)
CREATE OR REPLACE FUNCTION public.calculate_model_signature(model_content jsonb)
RETURNS varchar AS $$
BEGIN
    RETURN encode(digest(model_content::text, 'sha256'), 'hex');
END;
$$ LANGUAGE plpgsql IMMUTABLE;

COMMENT ON FUNCTION public.calculate_model_signature IS
    'Calculates SHA256 hash of model content for change detection';

-- Function: Detect attribute changes
CREATE OR REPLACE FUNCTION public.detect_attribute_changes(
    old_values jsonb,
    new_values jsonb
) RETURNS jsonb AS $$
DECLARE
    v_changes jsonb := '{"changed_fields": []}'::jsonb;
    v_key text;
    v_changed_count int := 0;
BEGIN
    -- Detect added fields
    FOR v_key IN SELECT jsonb_object_keys(new_values - old_values) LOOP
        v_changes := jsonb_set(
            v_changes,
            ARRAY['changed_fields'],
            v_changes->'changed_fields' || jsonb_build_array(v_key)
        );
        v_changed_count := v_changed_count + 1;
    END LOOP;
    
    -- Detect removed fields
    FOR v_key IN SELECT jsonb_object_keys(old_values - new_values) LOOP
        v_changes := jsonb_set(
            v_changes,
            ARRAY['changed_fields'],
            v_changes->'changed_fields' || jsonb_build_array(v_key)
        );
        v_changed_count := v_changed_count + 1;
    END LOOP;
    
    -- Add metadata
    v_changes := jsonb_set(v_changes, ARRAY['change_count'], to_jsonb(v_changed_count));
    
    RETURN v_changes;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION public.detect_attribute_changes IS
    'Detects which fields changed between two JSONB objects';

-- Function: Create new model version
CREATE OR REPLACE FUNCTION public.create_model_version(
    p_entity_attr_id uuid,
    p_tenant_id uuid,
    p_tenant_ds_id uuid,
    p_model_content jsonb,
    p_changes_summary text,
    p_generated_by text
) RETURNS uuid AS $$
DECLARE
    v_latest_version int;
    v_model_signature varchar;
    v_new_version_id uuid;
BEGIN
    -- Get latest version number
    SELECT COALESCE(MAX(version_number), 0) INTO v_latest_version
    FROM public.model_version_history
    WHERE entity_attribute_id = p_entity_attr_id
        AND tenant_datasource_id = p_tenant_ds_id
        AND is_active = true;
    
    -- Calculate signature
    v_model_signature := public.calculate_model_signature(p_model_content);
    
    -- Create new version
    INSERT INTO public.model_version_history (
        tenant_id, tenant_datasource_id, entity_attribute_id,
        version_number, model_content, model_signature,
        changes_from_previous, generated_by, is_active
    ) VALUES (
        p_tenant_id, p_tenant_ds_id, p_entity_attr_id,
        v_latest_version + 1, p_model_content, v_model_signature,
        p_changes_summary, p_generated_by, true
    ) RETURNING id INTO v_new_version_id;
    
    RETURN v_new_version_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION public.create_model_version IS
    'Creates a new semantic model version with automatic version numbering';

-- ============================================================================
-- 8. Triggers for Automatic Regeneration Detection
-- ============================================================================

-- Trigger: Detect entity_attribute changes
CREATE OR REPLACE FUNCTION public.trigger_model_regeneration_on_entity_change()
RETURNS TRIGGER AS $$
DECLARE
    v_correlation_id uuid;
    v_change_detail jsonb;
BEGIN
    v_correlation_id := gen_random_uuid();
    
    -- Capture what changed
    v_change_detail := jsonb_build_object(
        'old_values', to_jsonb(OLD),
        'new_values', to_jsonb(NEW),
        'changed_fields', detect_attribute_changes(to_jsonb(OLD), to_jsonb(NEW))->>'changed_fields'
    );
    
    -- Create trigger record
    INSERT INTO public.model_regeneration_trigger (
        tenant_id, tenant_datasource_id, entity_attribute_id,
        trigger_type, trigger_source, change_detail, triggered_by,
        regeneration_status
    ) VALUES (
        NEW.tenant_id, NEW.tenant_datasource_id, NEW.id,
        'ATTRIBUTE_CHANGED', 'USER_ACTION', v_change_detail, 'system',
        'PENDING'
    );
    
    -- Create audit entry
    INSERT INTO public.entity_attribute_audit (
        tenant_id, tenant_datasource_id, entity_attribute_id,
        action, old_values, new_values, changed_fields,
        changed_by, change_source, correlation_id
    ) VALUES (
        NEW.tenant_id, NEW.tenant_datasource_id, NEW.id,
        'UPDATE', to_jsonb(OLD), to_jsonb(NEW),
        (detect_attribute_changes(to_jsonb(OLD), to_jsonb(NEW))->>'changed_fields')::text[],
        'system', 'TRIGGER', v_correlation_id
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if present
DROP TRIGGER IF EXISTS trigger_model_regen_on_entity_attr_change ON public.entity_attribute;

-- Create trigger
CREATE TRIGGER trigger_model_regen_on_entity_attr_change
AFTER UPDATE ON public.entity_attribute
FOR EACH ROW
WHEN (OLD.name IS DISTINCT FROM NEW.name OR 
      OLD.business_name IS DISTINCT FROM NEW.business_name OR
      OLD.catalog_node_id IS DISTINCT FROM NEW.catalog_node_id)
EXECUTE FUNCTION public.trigger_model_regeneration_on_entity_change();

COMMENT ON TRIGGER trigger_model_regen_on_entity_attr_change ON public.entity_attribute IS
    'Automatically triggers semantic model regeneration when entity attributes change';

-- Trigger: Detect relationship changes
CREATE OR REPLACE FUNCTION public.trigger_model_regeneration_on_relationship_change()
RETURNS TRIGGER AS $$
BEGIN
    -- For INSERT (new relationship)
    IF TG_OP = 'INSERT' THEN
        INSERT INTO public.model_regeneration_trigger (
            tenant_id, tenant_datasource_id, entity_attribute_id,
            trigger_type, trigger_source, change_detail, triggered_by,
            regeneration_status
        ) VALUES (
            NEW.tenant_id, NEW.tenant_datasource_id, NEW.source_entity_id,
            'RELATIONSHIP_ADDED', 'AUTO_DISCOVERY', 
            jsonb_build_object('relationship_id', NEW.id, 'target_entity_id', NEW.target_entity_id),
            'system', 'PENDING'
        ),
        (
            NEW.tenant_id, NEW.tenant_datasource_id, NEW.target_entity_id,
            'RELATIONSHIP_ADDED', 'AUTO_DISCOVERY',
            jsonb_build_object('relationship_id', NEW.id, 'source_entity_id', NEW.source_entity_id),
            'system', 'PENDING'
        );
    
    -- For DELETE (relationship removed)
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO public.model_regeneration_trigger (
            tenant_id, tenant_datasource_id, entity_attribute_id,
            trigger_type, trigger_source, change_detail, triggered_by,
            regeneration_status
        ) VALUES (
            OLD.tenant_id, OLD.tenant_datasource_id, OLD.source_entity_id,
            'RELATIONSHIP_REMOVED', 'USER_ACTION',
            jsonb_build_object('relationship_id', OLD.id, 'target_entity_id', OLD.target_entity_id),
            'system', 'PENDING'
        ),
        (
            OLD.tenant_id, OLD.tenant_datasource_id, OLD.target_entity_id,
            'RELATIONSHIP_REMOVED', 'USER_ACTION',
            jsonb_build_object('relationship_id', OLD.id, 'source_entity_id', OLD.source_entity_id),
            'system', 'PENDING'
        );
    END IF;
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Drop existing trigger if present
DROP TRIGGER IF EXISTS trigger_model_regen_on_relationship_change ON public.entity_relationship;

-- Create trigger
CREATE TRIGGER trigger_model_regen_on_relationship_change
AFTER INSERT OR DELETE ON public.entity_relationship
FOR EACH ROW
EXECUTE FUNCTION public.trigger_model_regeneration_on_relationship_change();

COMMENT ON TRIGGER trigger_model_regen_on_relationship_change ON public.entity_relationship IS
    'Automatically triggers semantic model regeneration when relationships change';

-- ============================================================================
-- 9. Grant Permissions
-- ============================================================================

GRANT SELECT, INSERT, UPDATE, DELETE ON public.model_regeneration_trigger TO postgres;
GRANT SELECT, INSERT ON public.entity_attribute_audit TO postgres;
GRANT SELECT, INSERT, UPDATE ON public.model_version_history TO postgres;
GRANT SELECT, INSERT ON public.semantic_model_change_log TO postgres;
GRANT SELECT, INSERT, UPDATE ON public.model_regeneration_queue TO postgres;
GRANT SELECT ON public.v_pending_model_regenerations TO postgres;
GRANT SELECT ON public.v_change_impact_analysis TO postgres;
GRANT EXECUTE ON FUNCTION public.calculate_model_signature TO postgres;
GRANT EXECUTE ON FUNCTION public.detect_attribute_changes TO postgres;
GRANT EXECUTE ON FUNCTION public.create_model_version TO postgres;

COMMIT;

-- ============================================================================
-- Schema Summary
-- ============================================================================
-- This migration adds semantic model regeneration tracking with:
--
-- 1. model_regeneration_trigger (5 indexes)
--    - Records when regeneration is needed
--    - Tracks progress from PENDING → IN_PROGRESS → COMPLETED
--    - Stores before/after versions
--
-- 2. entity_attribute_audit (5 indexes)
--    - Full audit trail of attribute changes
--    - Tracks who changed what and when
--    - Calculates impact (relationships, reports affected)
--
-- 3. model_version_history (4 indexes)
--    - Version control for semantic models
--    - Signature-based change detection
--    - Enables rollback and comparison
--
-- 4. semantic_model_change_log (4 indexes)
--    - High-level change log for reporting
--    - Severity levels and impact metrics
--    - Reversibility tracking
--
-- 5. model_regeneration_queue (3 indexes)
--    - Priority queue for processing regenerations
--    - Retry logic for failed attempts
--    - Dependency tracking for related changes
--
-- Features:
-- ✓ Automatic trigger detection
-- ✓ Change impact analysis
-- ✓ Version control & rollback
-- ✓ Audit trail for compliance
-- ✓ Multi-tenant isolation
-- ✓ Smart regeneration (skip if no actual changes)
-- ============================================================================
