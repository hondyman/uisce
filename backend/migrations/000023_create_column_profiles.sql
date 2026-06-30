-- Migration: 000023_create_column_profiles.sql
-- Create sml.column_profiles table used by the profiler and semantic mapping fallbacks



-- Ensure schema exists
CREATE SCHEMA IF NOT EXISTS sml;

-- CREATE TABLE IF NOT EXISTS CREATE TABLE IF NOT EXISTS sml.column_profiles (
    id uuid NOT NULL,
    tenant_datasource_id uuid NOT NULL,
    data_type text NULL,
    "cardinality" int8 NULL,
    min_length int4 NULL,
    max_length int4 NULL,
    avg_length float8 NULL,
    min_value float8 NULL,
    max_value float8 NULL,
    avg_value float8 NULL,
    std_dev float8 NULL,
    frequent_values _text NULL,
    inferred_patterns _text NULL,
    properties jsonb NULL,
    created_at timestamptz DEFAULT now() NULL,
    bloom_filter bytea NULL,
    tenant_id text NULL,
    datasource_id text NULL,
    CONSTRAINT column_profiles_pkey PRIMARY KEY (id)
);

-- Index to efficiently find profiles by tenant/datasource
CREATE INDEX IF NOT EXISTS idx_profiles_on_tenant_ds ON sml.column_profiles USING btree (tenant_datasource_id);

COMMENT ON TABLE sml.column_profiles IS 'Stores statistical and structural profiles of data columns from various sources. The id maps to the corresponding column node id in public.catalog_node.';
COMMENT ON COLUMN sml.column_profiles.frequent_values IS 'Stores top-K most frequent values as a text array.';
COMMENT ON COLUMN sml.column_profiles.bloom_filter IS 'Serialized Bloom filter data for the column.';
COMMENT ON COLUMN sml.column_profiles.properties IS 'Flexible JSON properties for profiler signals such as frequent_values and inferred_patterns.';


