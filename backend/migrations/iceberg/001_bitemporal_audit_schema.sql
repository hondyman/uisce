-- Bitemporal Audit Schema for Iceberg
-- This schema enables point-in-time recovery and compliance tracking

-- Create audit schema if it doesn't exist
CREATE SCHEMA IF NOT EXISTS iceberg.audit;

-- Tenant History Table
CREATE TABLE IF NOT EXISTS iceberg.audit.tenant_history (
  -- Primary identifiers
  tenant_id VARCHAR NOT NULL,
  version_id VARCHAR NOT NULL,
  
  -- Bitemporal timestamps
  valid_from TIMESTAMP(6) NOT NULL,
  valid_to TIMESTAMP(6),
  system_from TIMESTAMP(6) NOT NULL,
  system_to TIMESTAMP(6),
  
  -- Change metadata
  change_type VARCHAR(20) NOT NULL, -- INSERT, UPDATE, DELETE, RESTORE
  changed_by VARCHAR(255),
  change_reason VARCHAR(1000),
  
  -- Full entity snapshot as JSON
  entity_data VARCHAR NOT NULL,
  
  -- Computed flags
  is_current BOOLEAN NOT NULL,
  is_deleted BOOLEAN NOT NULL,
  
  -- Metadata
  created_at TIMESTAMP(6) NOT NULL DEFAULT current_timestamp
)
WITH (
  format = 'PARQUET',
  partitioning = ARRAY['day(system_from)']
);

-- Instance History Table
CREATE TABLE IF NOT EXISTS iceberg.audit.instance_history (
  -- Primary identifiers
  instance_id VARCHAR NOT NULL,
  tenant_id VARCHAR NOT NULL,
  version_id VARCHAR NOT NULL,
  
  -- Bitemporal timestamps
  valid_from TIMESTAMP(6) NOT NULL,
  valid_to TIMESTAMP(6),
  system_from TIMESTAMP(6) NOT NULL,
  system_to TIMESTAMP(6),
  
  -- Change metadata
  change_type VARCHAR(20) NOT NULL,
  changed_by VARCHAR(255),
  change_reason VARCHAR(1000),
  
  -- Full entity snapshot as JSON
  entity_data VARCHAR NOT NULL,
  
  -- Computed flags
  is_current BOOLEAN NOT NULL,
  is_deleted BOOLEAN NOT NULL,
  
  -- Metadata
  created_at TIMESTAMP(6) NOT NULL DEFAULT current_timestamp
)
WITH (
  format = 'PARQUET',
  partitioning = ARRAY['day(system_from)']
);

-- Connection History Table
CREATE TABLE IF NOT EXISTS iceberg.audit.connection_history (
  -- Primary identifiers
  connection_id VARCHAR NOT NULL,
  tenant_id VARCHAR NOT NULL,
  version_id VARCHAR NOT NULL,
  
  -- Bitemporal timestamps
  valid_from TIMESTAMP(6) NOT NULL,
  valid_to TIMESTAMP(6),
  system_from TIMESTAMP(6) NOT NULL,
  system_to TIMESTAMP(6),
  
  -- Change metadata
  change_type VARCHAR(20) NOT NULL,
  changed_by VARCHAR(255),
  change_reason VARCHAR(1000),
  
  -- Full entity snapshot as JSON
  entity_data VARCHAR NOT NULL,
  
  -- Computed flags
  is_current BOOLEAN NOT NULL,
  is_deleted BOOLEAN NOT NULL,
  
  -- Metadata
  created_at TIMESTAMP(6) NOT NULL DEFAULT current_timestamp
)
WITH (
  format = 'PARQUET',
  partitioning = ARRAY['day(system_from)']
);

-- Product History Table
CREATE TABLE IF NOT EXISTS iceberg.audit.product_history (
  -- Primary identifiers
  product_id VARCHAR NOT NULL,
  tenant_id VARCHAR NOT NULL,
  version_id VARCHAR NOT NULL,
  
  -- Bitemporal timestamps
  valid_from TIMESTAMP(6) NOT NULL,
  valid_to TIMESTAMP(6),
  system_from TIMESTAMP(6) NOT NULL,
  system_to TIMESTAMP(6),
  
  -- Change metadata
  change_type VARCHAR(20) NOT NULL,
  changed_by VARCHAR(255),
  change_reason VARCHAR(1000),
  
  -- Full entity snapshot as JSON
  entity_data VARCHAR NOT NULL,
  
  -- Computed flags
  is_current BOOLEAN NOT NULL,
  is_deleted BOOLEAN NOT NULL,
  
  -- Metadata
  created_at TIMESTAMP(6) NOT NULL DEFAULT current_timestamp
)
WITH (
  format = 'PARQUET',
  partitioning = ARRAY['day(system_from)']
);

-- Create indexes for common queries
-- Note: Iceberg doesn't support traditional indexes, but we can use metadata columns
-- and partitioning for efficient queries

-- Views for easier querying

-- Current state view for tenants
CREATE OR REPLACE VIEW iceberg.audit.tenant_current AS
SELECT 
  tenant_id,
  version_id,
  valid_from,
  system_from,
  change_type,
  changed_by,
  entity_data,
  is_deleted
FROM iceberg.audit.tenant_history
WHERE is_current = true;

-- Current state view for instances
CREATE OR REPLACE VIEW iceberg.audit.instance_current AS
SELECT 
  instance_id,
  tenant_id,
  version_id,
  valid_from,
  system_from,
  change_type,
  changed_by,
  entity_data,
  is_deleted
FROM iceberg.audit.instance_history
WHERE is_current = true;

-- Current state view for connections
CREATE OR REPLACE VIEW iceberg.audit.connection_current AS
SELECT 
  connection_id,
  tenant_id,
  version_id,
  valid_from,
  system_from,
  change_type,
  changed_by,
  entity_data,
  is_deleted
FROM iceberg.audit.connection_history
WHERE is_current = true;

-- Current state view for products
CREATE OR REPLACE VIEW iceberg.audit.product_current AS
SELECT 
  product_id,
  tenant_id,
  version_id,
  valid_from,
  system_from,
  change_type,
  changed_by,
  entity_data,
  is_deleted
FROM iceberg.audit.product_history
WHERE is_current = true;
