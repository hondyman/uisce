# Catalog Change Event Schemas and Sinks

## Avro schema (register in Schema Registry)
```json
{
  "type": "record",
  "name": "CatalogChangeEvent",
  "namespace": "com.wealthstream.catalog",
  "fields": [
    { "name": "eventId", "type": "string" },
    { "name": "entityType", "type": "string" },
    { "name": "changeType", "type": { "type": "enum", "name": "ChangeType", "symbols": ["insert", "update", "delete"] } },
    { "name": "tenantId", "type": "string" },
    { "name": "occurredAt", "type": { "type": "long", "logicalType": "timestamp-millis" } },
    { "name": "before", "type": ["null", { "type": "map", "values": "string" }], "default": null },
    { "name": "after",  "type": ["null", { "type": "map", "values": "string" }], "default": null },
    { "name": "source", "type": "string" }
  ]
}
```

## Iceberg table DDL (Trino syntax)
```sql
CREATE TABLE IF NOT EXISTS catalog_events (
  event_id      VARCHAR,
  entity_type   VARCHAR,
  change_type   VARCHAR,
  tenant_id     VARCHAR,
  occurred_at   TIMESTAMP(3) WITH TIME ZONE,
  before        MAP(VARCHAR, VARCHAR),
  after         MAP(VARCHAR, VARCHAR),
  source        VARCHAR
)
WITH (
  format = 'PARQUET',
  partitioning = ARRAY['entity_type', 'tenant_id', 'month(occurred_at)']
);
```

## Kafka Connect sink (Iceberg) — sketch
```json
{
  "name": "catalog-events-iceberg-sink",
  "config": {
    "connector.class": "org.apache.iceberg.connect.IcebergSinkConnector",
    "tasks.max": "1",
    "topics": "catalog-change-events",

    "iceberg.catalog.name": "ws_catalog",
    "iceberg.catalog.type": "hadoop",
    "iceberg.catalog.warehouse": "s3://your-bucket/iceberg",

    "key.converter": "org.apache.kafka.connect.storage.StringConverter",
    "value.converter": "io.confluent.connect.avro.AvroConverter",
    "value.converter.schema.registry.url": "http://schema-registry:8081",

    "iceberg.table.auto-create": "true",
    "iceberg.table.default.write.format": "PARQUET",
    "iceberg.tables": "catalog_events",

    "behavior.on.error": "fail",
    "errors.tolerance": "none"
  }
}
```

## Kafka Connect sink (alternative) — simple JSON → Iceberg
If you emit JSON instead of Avro:
```json
{
  "name": "catalog-events-json-iceberg-sink",
  "config": {
    "connector.class": "org.apache.iceberg.connect.IcebergSinkConnector",
    "tasks.max": "1",
    "topics": "catalog-change-events-json",

    "iceberg.catalog.name": "ws_catalog",
    "iceberg.catalog.type": "hadoop",
    "iceberg.catalog.warehouse": "s3://your-bucket/iceberg",

    "value.converter": "org.apache.kafka.connect.json.JsonConverter",
    "value.converter.schemas.enable": "true",

    "iceberg.table.auto-create": "true",
    "iceberg.table.default.write.format": "PARQUET",
    "iceberg.tables": "catalog_events"
  }
}
```

## Notes
- Avro on Kafka with Schema Registry is the recommended path; JSON is convenient for debugging.
- Partitioning by `entity_type`, `tenant_id`, and `month(occurred_at)` keeps Iceberg tables balanced and queryable for time travel.
- If you evolve schemas, rely on Schema Registry compatibility rules; Iceberg will accept additive changes when using Avro with schemas.
