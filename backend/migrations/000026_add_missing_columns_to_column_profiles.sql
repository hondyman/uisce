-- Migration: 000026_add_missing_columns_to_column_profiles.sql
-- Ensure sml.column_profiles has the columns expected by the profiler service.



CREATE SCHEMA IF NOT EXISTS sml;

ALTER TABLE sml.column_profiles
    ADD COLUMN IF NOT EXISTS datasource text,
    ADD COLUMN IF NOT EXISTS schema text,
    ADD COLUMN IF NOT EXISTS table_name text,
    ADD COLUMN IF NOT EXISTS column_name text;

-- Optional indexes to speed up lookups by schema/table.
CREATE INDEX IF NOT EXISTS idx_profiles_schema_table
    ON sml.column_profiles USING btree (schema, table_name);


