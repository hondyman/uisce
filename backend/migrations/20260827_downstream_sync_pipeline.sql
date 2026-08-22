-- 20260827_downstream_sync_pipeline.sql
-- Downstream Push Pipeline, Field Transformations, Code Translations & Delivery Audit

CREATE SCHEMA IF NOT EXISTS mdm_pipeline;

-- 1. Downstream Target Sync Configuration per BO Binding
CREATE TABLE IF NOT EXISTS mdm_pipeline.binding_sync_configs (
    config_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    target_name VARCHAR(100) NOT NULL,            -- "CRIMS_ORACLE", "DATAMART_POSTGRES", "SALESFORCE_REST"
    sync_mode VARCHAR(50) NOT NULL DEFAULT 'REALTIME_EVENT', -- REALTIME_EVENT, MICRO_BATCH, SCHEDULED
    delivery_channel VARCHAR(50) NOT NULL,        -- SQL_MERGE, BULK_STAGE, REST_API, EVENT_TOPIC
    api_endpoint_url TEXT,
    api_auth_secret_arn TEXT,
    batch_size INT DEFAULT 500,
    retry_max_attempts INT DEFAULT 5,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_binding_sync UNIQUE (tenant_id, bo_id, binding_id)
);

-- 2. Declarative Field Transformations & Output Expressions (Rule 1: Config-Before-Code)
CREATE TABLE IF NOT EXISTS mdm_pipeline.field_transformation_rules (
    rule_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    binding_id UUID NOT NULL REFERENCES public.business_object_bindings(id) ON DELETE CASCADE,
    source_field_name VARCHAR(100) NOT NULL,
    target_column_name VARCHAR(100) NOT NULL,
    transformation_type VARCHAR(50) NOT NULL,     -- DIRECT, EXPRESSION, CODE_TRANSLATION, WASM_FORMATTER
    transformation_expr TEXT,                     -- e.g. "UPPER(TRIM(source_val))" or "lookup_iso2(country)"
    target_data_type VARCHAR(50) NOT NULL,        -- VARCHAR, NUMERIC, TIMESTAMP, JSONB
    is_required BOOLEAN DEFAULT TRUE,
    null_fallback_value TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_field_trans_rule UNIQUE (tenant_id, binding_id, source_field_name)
);

-- 3. Discrete Enum / Code Translation Dictionaries (e.g. ISO Country <-> CRIMS Code)
CREATE TABLE IF NOT EXISTS mdm_pipeline.code_translation_dictionaries (
    dict_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    dictionary_name VARCHAR(100) NOT NULL,        -- "COUNTRY_ISO_TO_CRIMS", "ASSET_CLASS_TO_BLOOMBERG"
    source_code VARCHAR(100) NOT NULL,
    target_code VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_code_trans_entry UNIQUE (tenant_id, dictionary_name, source_code)
);

-- 4. Downstream Delivery Audit & Reconciliation Ledger (SEC Rule 17a-4)
CREATE TABLE IF NOT EXISTS mdm_pipeline.downstream_sync_logs (
    log_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL,
    bo_id UUID NOT NULL,
    binding_id UUID NOT NULL,
    entity_sid VARCHAR(100) NOT NULL,
    target_name VARCHAR(100) NOT NULL,
    payload_sha256 VARCHAR(64) NOT NULL,
    delivery_status VARCHAR(50) NOT NULL,         -- DELIVERED, FAILED, RETRYING, DEAD_LETTER
    target_acknowledgment_id TEXT,
    execution_duration_ms INT,
    error_detail TEXT,
    dispatched_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_downstream_sync_search 
ON mdm_pipeline.downstream_sync_logs (tenant_id, entity_sid, dispatched_at DESC);
