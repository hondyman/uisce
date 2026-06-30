-- ============================================================
-- Business Object Definition Layer (Driving Table Pattern)
-- ============================================================

-- Enable CITEXT extension for case-insensitive text
CREATE EXTENSION IF NOT EXISTS citext;

-- ============================================================
-- Business Object Definition (driving table)
-- ============================================================
CREATE TABLE IF NOT EXISTS business_object_def (
  bo_def_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  bo_key CITEXT NOT NULL,               -- e.g. 'customer', 'ips', 'proposal'
  name TEXT NOT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  description TEXT,
  
  -- Driving table reference (from catalog_node)
  driver_table_id UUID REFERENCES catalog_node(id) ON DELETE SET NULL,
  driver_table_name TEXT,
  
  status TEXT NOT NULL DEFAULT 'draft', -- draft/active/deprecated
  config JSONB NOT NULL DEFAULT '{}'::jsonb, -- icons, ui hints, governance
  
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT,
  
  UNIQUE (tenant_id, bo_key)
);

CREATE INDEX IF NOT EXISTS idx_bo_def_tenant ON business_object_def(tenant_id);
CREATE INDEX IF NOT EXISTS idx_bo_def_status ON business_object_def(status);

-- ============================================================
-- Subtypes (sub business object types) - DEFINED FIRST
-- ============================================================
CREATE TABLE IF NOT EXISTS bo_subtype_def (
  subtype_def_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  bo_def_id UUID NOT NULL REFERENCES business_object_def(bo_def_id) ON DELETE CASCADE,

  subtype_key CITEXT NOT NULL,          -- 'taxable', 'ira', 'trust', 'personal', 'corporate'
  name TEXT NOT NULL,
  display_name TEXT NOT NULL DEFAULT '',
  description TEXT,
  
  -- Optional: parent subtype for inheritance
  parent_subtype_id UUID REFERENCES bo_subtype_def(subtype_def_id) ON DELETE SET NULL,
  
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  
  UNIQUE (tenant_id, bo_def_id, subtype_key)
);

CREATE INDEX IF NOT EXISTS idx_bo_subtype_bo ON bo_subtype_def(tenant_id, bo_def_id);

-- ============================================================
-- Field Definitions - SCOPED TO SUBTYPE
-- ============================================================
CREATE TABLE IF NOT EXISTS bo_field_def (
  field_def_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  bo_def_id UUID NOT NULL REFERENCES business_object_def(bo_def_id) ON DELETE CASCADE,
  subtype_def_id UUID NOT NULL REFERENCES bo_subtype_def(subtype_def_id) ON DELETE CASCADE,

  field_key CITEXT NOT NULL,            -- stable key: 'name', 'risk_score'
  display_name TEXT NOT NULL,           -- 'Risk Score'
  technical_name TEXT,                  -- optional mapping hint
  field_type TEXT NOT NULL,             -- string/number/date/boolean/json/array
  
  is_required BOOLEAN NOT NULL DEFAULT FALSE,
  is_multi_value BOOLEAN NOT NULL DEFAULT FALSE,
  
  json_schema JSONB NOT NULL DEFAULT '{}'::jsonb, -- constraints, enums, formats
  
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  
  -- Each field is unique per subtype
  UNIQUE (tenant_id, subtype_def_id, field_key)
);

CREATE INDEX IF NOT EXISTS idx_bo_field_def_subtype ON bo_field_def(tenant_id, subtype_def_id);
CREATE INDEX IF NOT EXISTS idx_bo_field_def_bo ON bo_field_def(tenant_id, bo_def_id);

-- ============================================================
-- Instances (the actual records)
-- Mirrors the "BusinessObjectInstance coreFieldValues/customFieldValues" model
-- ============================================================
CREATE TABLE IF NOT EXISTS bo_instance (
  bo_instance_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

  bo_def_id UUID NOT NULL REFERENCES business_object_def(bo_def_id) ON DELETE CASCADE,
  subtype_def_id UUID REFERENCES bo_subtype_def(subtype_def_id) ON DELETE SET NULL,

  external_ref TEXT,                    -- optional (CRM id, etc.)
  title TEXT,                           -- optional human label

  core_field_values JSONB NOT NULL DEFAULT '{}'::jsonb,
  custom_field_values JSONB NOT NULL DEFAULT '{}'::jsonb,

  status TEXT NOT NULL DEFAULT 'active',
  
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_by TEXT,
  
  is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
  deleted_at TIMESTAMPTZ,

  UNIQUE (tenant_id, bo_def_id, external_ref)
);

CREATE INDEX IF NOT EXISTS idx_bo_instance_lookup ON bo_instance(tenant_id, bo_def_id, subtype_def_id);
CREATE INDEX IF NOT EXISTS idx_bo_instance_core_gin ON bo_instance USING GIN (core_field_values);
CREATE INDEX IF NOT EXISTS idx_bo_instance_active ON bo_instance(tenant_id, bo_def_id, is_deleted);

-- ============================================================
-- Related Objects (links between instances / hard tables)
-- Generic graph: instance -> instance or instance -> hard table
-- ============================================================
CREATE TABLE IF NOT EXISTS bo_relationship (
  relationship_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

  from_instance_id UUID NOT NULL REFERENCES bo_instance(bo_instance_id) ON DELETE CASCADE,
  to_instance_id UUID REFERENCES bo_instance(bo_instance_id) ON DELETE CASCADE,
  
  -- Optional: if linking to a hard table (e.g. household.household_id)
  to_hard_table_name TEXT,
  to_hard_table_id TEXT,

  relationship_type CITEXT NOT NULL,    -- 'owns','depends_on','member_of','generated_by'
  properties JSONB NOT NULL DEFAULT '{}'::jsonb,
  
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  created_by TEXT,

  UNIQUE (tenant_id, from_instance_id, to_instance_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_bo_relationship_from ON bo_relationship(tenant_id, from_instance_id);
CREATE INDEX IF NOT EXISTS idx_bo_relationship_to ON bo_relationship(tenant_id, to_instance_id);
CREATE INDEX IF NOT EXISTS idx_bo_relationship_type ON bo_relationship(tenant_id, relationship_type);

-- ============================================================
-- Audit Log (for compliance)
-- ============================================================
CREATE TABLE IF NOT EXISTS bo_audit_log (
  audit_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  
  entity_type TEXT NOT NULL,            -- 'business_object' | 'instance' | 'relationship'
  entity_id UUID NOT NULL,
  
  action TEXT NOT NULL,                 -- 'CREATE' | 'UPDATE' | 'DELETE'
  changes JSONB NOT NULL DEFAULT '{}'::jsonb,
  
  created_by TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bo_audit_entity ON bo_audit_log(tenant_id, entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_bo_audit_time ON bo_audit_log(tenant_id, created_at DESC);

-- ============================================================
-- Comments
-- ============================================================
COMMENT ON TABLE business_object_def IS 'Driving table for business object definitions; one row per BO type (Customer, Portfolio, etc.)';
COMMENT ON TABLE bo_field_def IS 'Field definitions for a business object; flexible schema support via JSON';
COMMENT ON TABLE bo_subtype_def IS 'Optional subtypes/variants of a business object (e.g., Account -> {Taxable, IRA})';
COMMENT ON TABLE bo_instance IS 'Actual business object records; values stored as JSON to allow schema evolution';
COMMENT ON TABLE bo_relationship IS 'Typed relationships between instances or to hard tables; enables rich data modeling';
COMMENT ON TABLE bo_audit_log IS 'Audit trail for compliance and governance';
