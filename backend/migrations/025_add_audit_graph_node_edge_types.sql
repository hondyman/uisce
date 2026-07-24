-- Migration: Add audit graph node and edge types to catalog
-- Purpose: Establish catalog_node_type and catalog_edge_type entries for audit graph integration
-- Timestamp: 2026-01-18

-- Insert new node types for audit graph
INSERT INTO catalog_node_type (tenant_id, catalog_type_name, description, created_at, updated_at)
VALUES
  ('00000000-0000-0000-0000-000000000000', 'AUDIT_EVENT', 'Generic audit event', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'JOB_RUN', 'Execution of a scheduler job', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'DAG_RUN', 'Execution of a scheduler DAG', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'CHANGESET_EVENT', 'Governance ChangeSet event', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'COMPLIANCE_EVENT', 'Compliance violation or enforcement', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'INCIDENT', 'Clustered operational incident', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'SEMANTIC_SNAPSHOT', 'Versioned semantic term snapshot', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'AI_SUGGESTION', 'AI-generated narrative or recommendation', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'SLO_RISK', 'Service level objective risk indicator', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'TENANT_SUMMARY', 'Tenant-scoped aggregate summary', NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000000', 'GLOBAL_SUMMARY', 'Platform-wide aggregate summary', NOW(), NOW())
ON CONFLICT (tenant_id, catalog_type_name) DO UPDATE SET
  description = EXCLUDED.description,
  updated_at = NOW();

-- Insert new edge types for audit graph
INSERT INTO catalog_edge_type (code, label)
VALUES
  ('event_of', 'EVENT_OF'),
  ('runs_job', 'RUNS_JOB'),
  ('runs_dag', 'RUNS_DAG'),
  ('has_impact_on', 'HAS_IMPACT_ON'),
  ('causes', 'CAUSES'),
  ('has_ai_narrative', 'HAS_AI_NARRATIVE'),
  ('has_compliance_context', 'HAS_COMPLIANCE_CONTEXT'),
  ('has_semantic_context', 'HAS_SEMANTIC_CONTEXT'),
  ('has_tenant', 'HAS_TENANT'),
  ('applied', 'APPLIED'),
  ('version_of', 'VERSION_OF'),
  ('has_risk', 'HAS_RISK'),
  ('has_slo_context', 'HAS_SLO_CONTEXT')
ON CONFLICT (code) DO UPDATE SET
  label = EXCLUDED.label;
