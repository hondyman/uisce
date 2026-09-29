-- 20261118_002_mdm_vendor_ingest_scoring_pipeline.up.sql
-- Pre-seeds the official MDM Multi-Vendor Ingestion, Iceberg Lakehouse, Staging, Validation, and Scoring Pipeline

DO $seed$
DECLARE
    _tenant uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _pipe_id uuid := 'a11c0001-0001-4000-8000-000000000099'::uuid;
    _spec jsonb := '{
      "version": 1,
      "batch_size": 500,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "src",
          "type": "file_source",
          "label": "Vendor Market Data Feed",
          "position": {"x": 60, "y": 140},
          "config": {
            "uri": "file://market_data/vendor_daily.csv",
            "format": "csv",
            "has_header": true,
            "columns": [
              {"name": "security_id", "type": "string"},
              {"name": "primary_identifier", "type": "string"},
              {"name": "isin", "type": "string"},
              {"name": "ticker", "type": "string"},
              {"name": "security_name", "type": "string"},
              {"name": "closing_price", "type": "decimal"},
              {"name": "composite_rating", "type": "string"},
              {"name": "country_of_risk", "type": "string"},
              {"name": "sanctions_flag", "type": "string"},
              {"name": "lei", "type": "string"},
              {"name": "gics_sector", "type": "string"},
              {"name": "market_cap", "type": "decimal"},
              {"name": "vendor_id", "type": "string"}
            ]
          }
        },
        {
          "id": "rules",
          "type": "rule_check",
          "label": "Centralized Validation Rules",
          "position": {"x": 280, "y": 140},
          "config": {
            "rule_ids": []
          }
        },
        {
          "id": "map",
          "type": "map",
          "label": "Attribute Normalization",
          "position": {"x": 480, "y": 140},
          "config": {
            "keep_unmapped": true,
            "fields": [
              {"from": "security_id", "to": "security_id"},
              {"from": "primary_identifier", "to": "primary_identifier"},
              {"from": "isin", "to": "isin"},
              {"from": "ticker", "to": "ticker"},
              {"from": "security_name", "to": "security_name"},
              {"from": "closing_price", "to": "closing_price", "transform": "to_number"},
              {"from": "composite_rating", "to": "composite_rating", "transform": "upper"},
              {"from": "country_of_risk", "to": "country_of_risk", "transform": "upper"},
              {"from": "sanctions_flag", "to": "sanctions_flag", "transform": "upper"},
              {"from": "lei", "to": "lei", "transform": "trim"},
              {"from": "gics_sector", "to": "gics_sector"}
            ]
          }
        },
        {
          "id": "ice",
          "type": "iceberg_sink",
          "label": "Iceberg Lakehouse (Parquet)",
          "position": {"x": 720, "y": 60},
          "config": {
            "namespace": "raw_market_data",
            "table": "source_attribute_value",
            "format": "parquet"
          }
        },
        {
          "id": "stg",
          "type": "staging_sink",
          "label": "Stage to staging.security_data",
          "position": {"x": 720, "y": 220},
          "config": {
            "table": "staging.security_data",
            "source_cd": "BLOOMBERG",
            "domain": "SECURITY"
          }
        },
        {
          "id": "mstr",
          "type": "master",
          "label": "Mastering & Golden Survivorship",
          "position": {"x": 960, "y": 220},
          "config": {
            "entity": "SECURITY"
          }
        },
        {
          "id": "score",
          "type": "vendor_scoring",
          "label": "Vendor Quality & Scorecard",
          "position": {"x": 720, "y": 380},
          "config": {
            "entity": "SECURITY",
            "vendor_id": "BLOOMBERG",
            "universe_size": 42000,
            "tolerance_check": true,
            "record_mart": true
          }
        }
      ],
      "edges": [
        {"from": "src", "to": "rules"},
        {"from": "rules", "to": "map"},
        {"from": "map", "to": "ice"},
        {"from": "map", "to": "stg"},
        {"from": "stg", "to": "mstr"},
        {"from": "map", "to": "score"}
      ]
    }'::jsonb;
BEGIN
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);
    PERFORM set_config('app.current_tenant', _tenant::text, true);

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _pipe_id, _tenant,
        'MDM Multi-Vendor Market Data Ingestion & Scoring',
        'Ingest raw vendor market files → Parquet Iceberg Lakehouse → Centralized Validation Engine → Normalization Map → Staging DB (staging.security_data) → MDM Survivorship Mastering → Vendor Quality & Displacement Scoring Mart.',
        'loader', 'SECURITY', _spec, 500, 'skip_and_log', true, 'seed:mdm-scoring'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json,
        is_active = true,
        last_modified_at = now();
END
$seed$;
