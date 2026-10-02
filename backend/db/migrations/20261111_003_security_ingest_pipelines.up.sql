-- Security ingest: staging_bindings + file/queue pipeline defs (Account mirror).
-- Survivorship is applied at master_sink / security-load (Bloomberg > Refinitiv > GoldenSource > Internal),
-- not at ingest — ingest stamps source_system via staging_sink.source_cd.

DO $seed$
DECLARE
    _tenant uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _bo uuid;
    _file_id uuid := 'a11c0001-0001-4000-8000-000000000011'::uuid;
    _queue_id uuid := 'a11c0001-0001-4000-8000-000000000012'::uuid;
    _fields jsonb := '{
      "security_id": "security_id",
      "primary_identifier": "primary_identifier",
      "isin": "isin",
      "cusip": "cusip",
      "sedol": "sedol",
      "figi": "figi",
      "ticker": "ticker",
      "security_name": "security_name",
      "asset_class": "asset_class",
      "currency": "currency",
      "status": "status",
      "esg_score": "custom_attributes->>''esg_score''",
      "sfdr_article": "custom_attributes->>''sfdr_article''",
      "mifid_target_market": "custom_attributes->>''mifid_target_market''",
      "lei": "custom_attributes->>''lei''",
      "cic_code": "custom_attributes->>''cic_code''",
      "primary_exchange_mic": "custom_attributes->>''primary_exchange_mic''",
      "lot_size": "custom_attributes->>''lot_size''",
      "tick_size": "custom_attributes->>''tick_size''",
      "trading_status": "custom_attributes->>''trading_status''",
      "gics_sector": "custom_attributes->>''gics_sector''",
      "gics_industry": "custom_attributes->>''gics_industry''",
      "coupon_type": "custom_attributes->>''coupon_type''",
      "day_count": "custom_attributes->>''day_count''",
      "payment_frequency": "custom_attributes->>''payment_frequency''",
      "callable_flag": "custom_attributes->>''callable_flag''",
      "puttable_flag": "custom_attributes->>''puttable_flag''",
      "underlying_security_id": "custom_attributes->>''underlying_security_id''",
      "option_style": "custom_attributes->>''option_style''",
      "strike_price": "custom_attributes->>''strike_price''",
      "contract_size": "custom_attributes->>''contract_size''",
      "multiplier": "custom_attributes->>''multiplier''",
      "figi_share_class": "custom_attributes->>''figi_share_class''",
      "bloomberg_unique_id": "custom_attributes->>''bloomberg_unique_id''",
      "country_of_incorporation": "custom_attributes->>''country_of_incorporation''"
    }'::jsonb;
    _file_spec jsonb := '{
      "version": 1,
      "batch_size": 500,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "src",
          "type": "file_source",
          "label": "Security vendor file",
          "position": {"x": 80, "y": 120},
          "config": {
            "uri": "",
            "format": "csv",
            "has_header": true,
            "columns": [
              {"name": "security_id", "type": "string"},
              {"name": "primary_identifier", "type": "string"},
              {"name": "isin", "type": "string"},
              {"name": "cusip", "type": "string"},
              {"name": "sedol", "type": "string"},
              {"name": "figi", "type": "string"},
              {"name": "ticker", "type": "string"},
              {"name": "security_name", "type": "string"},
              {"name": "asset_class", "type": "string"},
              {"name": "currency", "type": "string"},
              {"name": "status", "type": "string"},
              {"name": "esg_score", "type": "decimal"},
              {"name": "sfdr_article", "type": "string"},
              {"name": "lei", "type": "string"},
              {"name": "gics_sector", "type": "string"},
              {"name": "gics_industry", "type": "string"}
            ]
          }
        },
        {
          "id": "map",
          "type": "map",
          "label": "Map typed + keep customs",
          "position": {"x": 360, "y": 120},
          "config": {
            "keep_unmapped": true,
            "fields": [
              {"from": "security_id", "to": "security_id"},
              {"from": "primary_identifier", "to": "primary_identifier"},
              {"from": "isin", "to": "isin"},
              {"from": "cusip", "to": "cusip"},
              {"from": "sedol", "to": "sedol"},
              {"from": "figi", "to": "figi"},
              {"from": "ticker", "to": "ticker"},
              {"from": "security_name", "to": "security_name"},
              {"from": "asset_class", "to": "asset_class"},
              {"from": "currency", "to": "currency"},
              {"from": "status", "to": "status"}
            ]
          }
        },
        {
          "id": "stg",
          "type": "staging_sink",
          "label": "Load staging.security_data",
          "position": {"x": 640, "y": 120},
          "config": {
            "table": "staging.security_data",
            "source_cd": "BLOOMBERG",
            "domain": "SECURITY"
          }
        }
      ],
      "edges": [
        {"from": "src", "to": "map"},
        {"from": "map", "to": "stg"}
      ]
    }'::jsonb;
    _queue_spec jsonb := '{
      "version": 1,
      "batch_size": 500,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "src",
          "type": "queue_source",
          "label": "Security ingest topic",
          "position": {"x": 80, "y": 120},
          "config": {
            "broker": "redpanda",
            "topic_or_queue": "security.ingest.v1",
            "format": "json",
            "consumer_group": "uisce-security-ingest",
            "max_messages": 1000,
            "idle_timeout_ms": 3000,
            "brokers_env": "KAFKA_BROKERS"
          }
        },
        {
          "id": "map",
          "type": "map",
          "label": "Map typed + keep customs",
          "position": {"x": 360, "y": 120},
          "config": {
            "keep_unmapped": true,
            "fields": [
              {"from": "security_id", "to": "security_id"},
              {"from": "primary_identifier", "to": "primary_identifier"},
              {"from": "isin", "to": "isin"},
              {"from": "cusip", "to": "cusip"},
              {"from": "sedol", "to": "sedol"},
              {"from": "figi", "to": "figi"},
              {"from": "ticker", "to": "ticker"},
              {"from": "security_name", "to": "security_name"},
              {"from": "asset_class", "to": "asset_class"},
              {"from": "currency", "to": "currency"},
              {"from": "status", "to": "status"}
            ]
          }
        },
        {
          "id": "stg",
          "type": "staging_sink",
          "label": "Load staging.security_data",
          "position": {"x": 640, "y": 120},
          "config": {
            "table": "staging.security_data",
            "source_cd": "REFINITIV",
            "domain": "SECURITY"
          }
        }
      ],
      "edges": [
        {"from": "src", "to": "map"},
        {"from": "map", "to": "stg"}
      ]
    }'::jsonb;
BEGIN
    PERFORM set_config('uisce.current_tenant', _tenant::text, true);
    PERFORM set_config('app.current_tenant', _tenant::text, true);

    SELECT id INTO _bo FROM public.business_objects
     WHERE tenant_id = _tenant AND bo_key = 'security' LIMIT 1;
    IF _bo IS NULL THEN
        RAISE NOTICE 'security ingest seed: security BO missing — skip binding';
    ELSE
        INSERT INTO public.staging_bindings (
            tenant_id, bo_id, staging_table, fields, version, source_type
        ) VALUES (
            _tenant, _bo, 'staging.security_data', _fields, 1, 'JSON_PATH'
        )
        ON CONFLICT (tenant_id, bo_id, staging_table) DO UPDATE SET
            fields = EXCLUDED.fields,
            source_type = EXCLUDED.source_type,
            version = public.staging_bindings.version + 1,
            updated_at = now();
    END IF;

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _file_id, _tenant,
        'Security Ingest → staging.security_data (file)',
        'Vendor CSV → map → staging.security_data. Change staging_sink.source_cd per feed (BLOOMBERG|REFINITIV|GOLDENSOURCE|INTERNAL). Unmapped CSV columns fold into custom_attributes. Survivorship runs at Security Master.',
        'loader', 'SECURITY', _file_spec, 500, 'skip_and_log', true, 'seed:security-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name, description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json, is_active = true, last_modified_at = now();

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _queue_id, _tenant,
        'Security Ingest ← queue (Redpanda)',
        'Bounded poll from security.ingest.v1 → staging.security_data (default source_cd=REFINITIV; override per run). Set KAFKA_BROKERS. Master merge applies Bloomberg > Refinitiv > GoldenSource > Internal.',
        'loader', 'SECURITY', _queue_spec, 500, 'skip_and_log', true, 'seed:security-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name, description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json, is_active = true, last_modified_at = now();
END
$seed$;
