-- Migration: Data Contracts & Semantic Versioning Schema
-- Date: 2026-07-30
-- Purpose: Schema for tracking Business Object data contracts and breaking change protection.

CREATE TABLE IF NOT EXISTS security.bo_data_contracts (
    contract_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    bo_name VARCHAR(128) NOT NULL,
    version VARCHAR(32) NOT NULL,
    schema_json JSONB NOT NULL,
    status VARCHAR(32) DEFAULT 'ACTIVE',
    breaking BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, bo_name, version)
);
