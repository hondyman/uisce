-- Migration: Add Audit Semantic Layer Node & Edge Types
-- Date: 2026-01-18
-- Purpose: Extend catalog_node_type and catalog_edge_type for audit graph integration
--
-- This migration adds:
-- 1. New node types for audit events (JOB_RUN, DAG_RUN, CHANGESET_EVENT, etc.)
-- 2. New edge types for audit relationships (RUNS_JOB, HAS_IMPACT_ON, CAUSES, etc.)
--
-- All audit events and relationships are stored in the existing catalog_node
-- and catalog_edge tables with these new type identifiers.

-- Phase 1: Insert Audit Node Types into catalog_node_type
-- These represent different kinds of audit events in the semantic graph

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'audit_event') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'audit_event', 'Generic audit event representing any system activity', true, '{"semantic_category": "audit", "icon": "📋", "queryable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'job_run') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'job_run', 'Execution record of a scheduler job with status and timing', true, '{"semantic_category": "audit", "icon": "⚙️", "queryable": true, "has_status": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'dag_run') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'dag_run', 'Execution record of a scheduler DAG (Directed Acyclic Graph)', true, '{"semantic_category": "audit", "icon": "🔀", "queryable": true, "has_status": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'changeset_event') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'changeset_event', 'Governance ChangeSet event affecting semantic or business objects', true, '{"semantic_category": "audit", "icon": "📝", "queryable": true, "has_status": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'compliance_event') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'compliance_event', 'Compliance violation or enforcement event', true, '{"semantic_category": "audit", "icon": "⚖️", "queryable": true, "has_severity": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'incident') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'incident', 'Clustered operational incident derived from audit events', true, '{"semantic_category": "audit", "icon": "🚨", "queryable": true, "has_status": true, "has_severity": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'semantic_snapshot') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'semantic_snapshot', 'Versioned snapshot of a semantic term at a point in time', true, '{"semantic_category": "audit", "icon": "📸", "queryable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;

  IF NOT EXISTS (SELECT 1 FROM public.catalog_node_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND catalog_type_name = 'ai_suggestion') THEN
    INSERT INTO public.catalog_node_type (id, tenant_id, catalog_type_name, description, is_active, config, created_at, updated_at)
    VALUES (gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'ai_suggestion', 'AI-generated narrative, recommendation, or root cause analysis', true, '{"semantic_category": "audit", "icon": "🤖", "queryable": true, "has_confidence": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
  END IF;
END$$;

-- Phase 2: Insert Audit Edge Types into catalog_edge_type
-- These represent relationships between audit nodes and other entities

DO $$
DECLARE
  has_singular boolean := EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_type');
  has_plural boolean := EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_types');
BEGIN
  IF has_singular THEN

    -- Use dynamic SQL to avoid compile-time dependency on catalog_edge_type
    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'event_of', 'Audit event refers to or documents a specific entity', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'event_of')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'runs_job', 'Job run executes a specific job', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'runs_job')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'runs_dag', 'DAG run executes a specific DAG', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'runs_dag')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_impact_on', 'ChangeSet impacts semantic terms, business objects, APIs, or pages', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true, "has_risk_score": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'has_impact_on')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'causes', 'Incident was caused by a job run, DAG run, or other event', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'causes')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_ai_narrative', 'AI suggestion (narrative, recommendation, analysis) attached to audit event', true, '{"semantic_category": "audit", "multiplicity": "one_to_one", "traversable": true, "has_confidence": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'has_ai_narrative')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_compliance_context', 'Compliance event linked to semantic term, business term, or regulation', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'has_compliance_context')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_semantic_context', 'Audit event linked to semantic term for context and lineage', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'has_semantic_context')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_type (id, tenant_id, predicate, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_tenant', 'Audit event associated with tenant for multi-tenant isolation', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true, "required": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_type WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND predicate = 'has_tenant')
    $exec$;

  ELSIF has_plural THEN

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'event_of', 'Audit event refers to or documents a specific entity', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'event_of')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'runs_job', 'Job run executes a specific job', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'runs_job')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'runs_dag', 'DAG run executes a specific DAG', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'runs_dag')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_impact_on', 'ChangeSet impacts semantic terms, business objects, APIs, or pages', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true, "has_risk_score": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'has_impact_on')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'causes', 'Incident was caused by a job run, DAG run, or other event', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'causes')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_ai_narrative', 'AI suggestion (narrative, recommendation, analysis) attached to audit event', true, '{"semantic_category": "audit", "multiplicity": "one_to_one", "traversable": true, "has_confidence": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'has_ai_narrative')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_compliance_context', 'Compliance event linked to semantic term, business term, or regulation', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'has_compliance_context')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_semantic_context', 'Audit event linked to semantic term for context and lineage', true, '{"semantic_category": "audit", "multiplicity": "many_to_many", "traversable": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'has_semantic_context')
    $exec$;

    EXECUTE $exec$
      INSERT INTO public.catalog_edge_types (id, tenant_id, edge_type_name, description, is_active, config, created_at, updated_at)
      SELECT gen_random_uuid(), '00000000-0000-0000-0000-000000000000', 'has_tenant', 'Audit event associated with tenant for multi-tenant isolation', true, '{"semantic_category": "audit", "multiplicity": "many_to_one", "traversable": true, "required": true}'::jsonb, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
      WHERE NOT EXISTS (SELECT 1 FROM public.catalog_edge_types WHERE tenant_id = '00000000-0000-0000-0000-000000000000' AND edge_type_name = 'has_tenant')
    $exec$;

  ELSE
    RAISE NOTICE 'no audit edge type table found, skipping';
  END IF;
END$$;

-- Phase 3: Create indexes for efficient audit graph traversal
CREATE INDEX IF NOT EXISTS idx_catalog_node_type_audit ON public.catalog_node_type(catalog_type_name)
  WHERE catalog_type_name IN (
    'audit_event', 'job_run', 'dag_run', 'changeset_event',
    'compliance_event', 'incident', 'semantic_snapshot', 'ai_suggestion'
  );

DO $$
DECLARE
  has_singular boolean := EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_type');
  has_plural boolean := EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'catalog_edge_types');
BEGIN
  IF has_singular THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_catalog_edge_type_audit ON public.catalog_edge_type(predicate) WHERE predicate IN (''event_of'', ''runs_job'', ''runs_dag'', ''has_impact_on'', ''causes'', ''has_ai_narrative'', ''has_compliance_context'', ''has_semantic_context'', ''has_tenant'')';
  ELSIF has_plural THEN
    EXECUTE 'CREATE INDEX IF NOT EXISTS idx_catalog_edge_types_audit ON public.catalog_edge_types(edge_type_name) WHERE edge_type_name IN (''event_of'', ''runs_job'', ''runs_dag'', ''has_impact_on'', ''causes'', ''has_ai_narrative'', ''has_compliance_context'', ''has_semantic_context'', ''has_tenant'')';
  ELSE
    RAISE NOTICE 'no audited edge type table found, skipping index creation';
  END IF;
END$$;

-- Phase 4: Add audit graph partition keys (for future sharding)
-- This enables efficient multi-tenant graph traversal
-- ALTER TABLE public.catalog_node ADD COLUMN IF NOT EXISTS partition_key VARCHAR(10);
-- ALTER TABLE public.catalog_edge ADD COLUMN IF NOT EXISTS partition_key VARCHAR(10);
-- (Uncomment in future phases when implementing sharding)

-- Phase 5: Create audit event materialized view base
-- This view unifies all audit events for efficient querying
-- (Created separately in Trino for cross-tenant performance)
