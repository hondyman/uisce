-- Seed Account MDM steward pipelines for the demo tenant.
-- Product path: Data Pipelines UI — not cmd/account-load.

DO $seed$
DECLARE
    _tenant uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _ingest_id uuid := 'a11c0001-0001-4000-8000-000000000001'::uuid;
    _master_id uuid := 'a11c0001-0001-4000-8000-000000000002'::uuid;
    _ingest_spec jsonb := '{
      "version": 1,
      "batch_size": 500,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "src",
          "type": "file_source",
          "label": "Account vendor file",
          "position": {"x": 80, "y": 120},
          "config": {
            "uri": "",
            "format": "csv",
            "has_header": true,
            "columns": [
              {"name": "account_cd", "type": "string"},
              {"name": "account_name", "type": "string"},
              {"name": "account_type_cd", "type": "string"},
              {"name": "status_cd", "type": "string"},
              {"name": "base_currency", "type": "string"},
              {"name": "domicile", "type": "string"},
              {"name": "source_system", "type": "string"}
            ]
          }
        },
        {
          "id": "map",
          "type": "map",
          "label": "Map to staging columns",
          "position": {"x": 360, "y": 120},
          "config": {
            "fields": [
              {"from": "account_cd", "to": "account_cd"},
              {"from": "account_name", "to": "account_name"},
              {"from": "account_type_cd", "to": "account_type_cd"},
              {"from": "status_cd", "to": "status_cd"},
              {"from": "base_currency", "to": "base_currency"},
              {"from": "domicile", "to": "domicile"}
            ]
          }
        },
        {
          "id": "stg",
          "type": "staging_sink",
          "label": "Load staging.account_data",
          "position": {"x": 640, "y": 120},
          "config": {
            "table": "staging.account_data",
            "source_cd": "GOLDENSOURCE",
            "domain": "ACCOUNT"
          }
        }
      ],
      "edges": [
        {"from": "src", "to": "map"},
        {"from": "map", "to": "stg"}
      ]
    }'::jsonb;
    _master_spec jsonb := '{
      "version": 1,
      "batch_size": 100,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "trigger",
          "type": "bo_source",
          "label": "Trigger (Account BO)",
          "position": {"x": 80, "y": 120},
          "config": {
            "bo_key": "account",
            "limit": 1
          }
        },
        {
          "id": "master",
          "type": "master_sink",
          "label": "Master Account from staging",
          "position": {"x": 420, "y": 120},
          "config": {
            "entity_type": "ACCOUNT",
            "staging_table": "staging.account_data",
            "batch_size": 100,
            "require_semantic_terms": true
          }
        }
      ],
      "edges": [
        {"from": "trigger", "to": "master"}
      ]
    }'::jsonb;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM public.tenants WHERE id = _tenant) THEN
        RAISE NOTICE 'seed account MDM pipelines: demo tenant missing — skip';
        RETURN;
    END IF;

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _ingest_id, _tenant,
        'Account Ingest → staging.account_data',
        'Steward path: vendor CSV → map → staging.account_data. Set source_cd on the staging sink to GOLDENSOURCE, MARKET_EDM, ASSET_CONTROL, or INTERNAL to match Survivorship priority. Upload a file on the source node before running. Product path — do not use cmd/account-load.',
        'loader', 'ACCOUNT', _ingest_spec, 500, 'skip_and_log', true, 'seed:account-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json,
        batch_size = EXCLUDED.batch_size,
        error_policy = EXCLUDED.error_policy,
        is_active = true,
        last_modified_at = now();

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _master_id, _tenant,
        'Account Master ← staging (survivorship)',
        'Steward path: promote pending staging.account_data into mdm.account_master using semantic-term source hierarchy (Build → Data → Survivorship). Trigger BO read is ignored; master_sink loads staging. Then use POST /api/v1/mdm/account-gold/build to materialise gold + optional Kafka publish.',
        'loader', 'ACCOUNT', _master_spec, 100, 'skip_and_log', true, 'seed:account-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json,
        batch_size = EXCLUDED.batch_size,
        error_policy = EXCLUDED.error_policy,
        is_active = true,
        last_modified_at = now();
END
$seed$;
