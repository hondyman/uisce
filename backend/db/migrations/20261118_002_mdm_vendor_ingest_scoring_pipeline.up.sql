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
            "uri": "file://uploads/bbg_security_20260927.txt",
            "format": "csv",
            "delimiter": "|",
            "has_header": true,
            "columns": [
              {"name": "ID_BB_GLOBAL", "type": "string"},
              {"name": "ID_BB_UNIQUE", "type": "string"},
              {"name": "TICKER", "type": "string"},
              {"name": "EXCH_CODE", "type": "string"},
              {"name": "ID_ISIN", "type": "string"},
              {"name": "ID_CUSIP", "type": "string"},
              {"name": "ID_SEDOL1", "type": "string"},
              {"name": "NAME", "type": "string"},
              {"name": "SECURITY_DES", "type": "string"},
              {"name": "SECURITY_TYP", "type": "string"},
              {"name": "ASSET_CLASS", "type": "string"},
              {"name": "CRNCY", "type": "string"},
              {"name": "CNTRY_OF_DOMICILE", "type": "string"},
              {"name": "CNTRY_OF_RISK", "type": "string"},
              {"name": "GICS_SECTOR_NAME", "type": "string"},
              {"name": "GICS_INDUSTRY_NAME", "type": "string"},
              {"name": "ISSUE_DT", "type": "string"},
              {"name": "MATURITY", "type": "string"},
              {"name": "FIRST_TRADE_DT", "type": "string"},
              {"name": "MIC_PRIMARY", "type": "string"},
              {"name": "MARKET_STATUS", "type": "string"}
            ]
          }
        },
        {
          "id": "map",
          "type": "map",
          "label": "Attribute Normalization",
          "position": {"x": 280, "y": 140},
          "config": {
            "keep_unmapped": true,
            "fields": [
              {"from": "security_id", "to": "SecId"},
              {"from": "security_id", "to": "security_id"},
              {"from": "security_name", "to": "SecName"},
              {"from": "security_name", "to": "security_name"},
              {"from": "vendor_id", "to": "SecTypCd"},
              {"from": "primary_identifier", "to": "primary_identifier"},
              {"from": "isin", "to": "isin"},
              {"from": "ticker", "to": "ticker"},
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
          "id": "rules",
          "type": "rule_check",
          "label": "Centralized Validation Rules",
          "position": {"x": 480, "y": 140},
          "config": {
            "bo_key": "security",
            "rule_ids": ["7c88ad55-34d9-45db-b0b1-fb1e70cf9a44"]
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
            "domain": "SECURITY",
            "columns": {
              "figi": "figi",
              "isin": "isin",
              "cusip": "cusip",
              "sedol": "sedol",
              "status": "status",
              "ticker": "ticker",
              "currency": "currency",
              "asset_class": "asset_class",
              "security_id": "security_id",
              "security_name": "security_name",
              "primary_identifier": "primary_identifier"
            }
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
        {"from": "src", "to": "map"},
        {"from": "map", "to": "rules"},
        {"from": "rules", "to": "ice"},
        {"from": "rules", "to": "stg"},
        {"from": "stg", "to": "mstr"},
        {"from": "rules", "to": "score"}
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
