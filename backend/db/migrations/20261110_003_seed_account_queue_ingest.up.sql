-- Seed Account Ingest (queue) pipeline for the demo tenant.
-- Uses queue_source → map → staging_sink. Credentials via KAFKA_BROKERS env.

DO $seed$
DECLARE
    _tenant uuid := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    _id uuid := 'a11c0001-0001-4000-8000-000000000003'::uuid;
    _spec jsonb := '{
      "version": 1,
      "batch_size": 500,
      "error_policy": "skip_and_log",
      "nodes": [
        {
          "id": "src",
          "type": "queue_source",
          "label": "Account ingest topic",
          "position": {"x": 80, "y": 120},
          "config": {
            "broker": "redpanda",
            "topic_or_queue": "account.ingest.v1",
            "format": "json",
            "consumer_group": "uisce-account-ingest",
            "max_messages": 1000,
            "idle_timeout_ms": 3000,
            "brokers_env": "KAFKA_BROKERS"
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
            "source_cd": "MARKET_EDM",
            "domain": "ACCOUNT"
          }
        }
      ],
      "edges": [
        {"from": "src", "to": "map"},
        {"from": "map", "to": "stg"}
      ]
    }'::jsonb;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM public.tenants WHERE id = _tenant) THEN
        RAISE NOTICE 'seed account queue ingest: demo tenant missing — skip';
        RETURN;
    END IF;

    INSERT INTO public.data_pipeline_definitions (
        id, tenant_id, name, description, mode, target_entity, dag_json,
        batch_size, error_policy, is_active, created_by
    ) VALUES (
        _id, _tenant,
        'Account Ingest ← queue (Redpanda)',
        'Steward path: bounded poll from Kafka/Redpanda topic account.ingest.v1 → staging.account_data. Set KAFKA_BROKERS (e.g. uisce-redpanda:9092). Change source_cd on the staging sink to match Survivorship priority. Swap broker to aws_sqs or azure_servicebus when those credentials are in env.',
        'loader', 'ACCOUNT', _spec, 500, 'skip_and_log', true, 'seed:account-mdm'
    )
    ON CONFLICT (id) DO UPDATE SET
        name = EXCLUDED.name,
        description = EXCLUDED.description,
        dag_json = EXCLUDED.dag_json,
        is_active = true,
        last_modified_at = now();
END
$seed$;
