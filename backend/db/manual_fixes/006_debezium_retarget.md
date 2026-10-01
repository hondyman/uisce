-- 006_debezium_retarget.md — cutover step (DEBEZIUM_RETARGET blocker)
--
-- Pre-staged: crims publication orm_cdc_publication (5 tables), wal_level=logical
-- on both DBs, crims orm.* source tables verified present.
--
-- The flip is config-only. DO NOT execute until cutover step 4 of the locked
-- sequence (app already read-only; dual-write window armed).
--
-- Slot MUST be new: replication slots are per-database; crims_oms_slot ≠
-- orm_oms_slot (alpha's).

## Connector config diff (only keys that change)

| key                | alpha (current)          | crims (cutover)              |
|--------------------|--------------------------|------------------------------|
| database.hostname  | 100.84.50.65             | 100.84.50.65                 |
| database.dbname    | alpha                    | crims                        |
| slot.name          | orm_oms_slot             | crims_oms_slot               |
| publication.name   | orm_cdc_publication      | orm_cdc_publication          |
| everything else    | unchanged                | unchanged (certs, tables, tpc, snapshot) |

## Cutover commands (step 4–5 of locked sequence)

```bash
# 4a. stop alpha connector
curl -s -X DELETE http://localhost:8083/connectors/orm-oms-connector

# 4b. register crims connector (config below)
curl -s -X POST http://localhost:8083/connectors -H 'Content-Type: application/json' -d '{
  "name": "orm-oms-connector",
  "config": {
    "connector.class": "io.debezium.connector.postgresql.PostgresConnector",
    "topic.prefix": "orm_oms",
    "database.hostname": "100.84.50.65",
    "database.port": "5432",
    "database.user": "postgres",
    "database.password": "<from .env>",
    "database.dbname": "crims",
    "database.sslmode": "verify-full",
    "database.sslcert": "/tmp/orm_client.crt",
    "database.sslkey": "/tmp/orm_client_der.pk8",
    "database.sslrootcert": "/tmp/orm_ca.crt",
    "plugin.name": "pgoutput",
    "slot.name": "crims_oms_slot",
    "publication.name": "orm_cdc_publication",
    "publication.autocreate.mode": "disabled",
    "schema.include.list": "orm",
    "table.include.list": "orm.execution,orm.\"order\",orm.placement,orm.order_allocation,orm.execution_allocation",
    "snapshot.mode": "never",
    "tombstones.on.delete": "false",
    "decimal.handling.mode": "double",
    "time.precision.mode": "connect"
  }
}'

# 4c. verify all tasks RUNNING
curl -s http://localhost:8083/connectors/orm-oms-connector/status | jq '.tasks[].state'

# 5. smoke: INSERT into crims.orm.execution, confirm StarRocks row ~5s
```

## Guards

- crims publication pre-staged with exactly 5 tables (verified; was 0 before fix)
- crims_oms_slot is new — never reuse orm_oms_slot
- stream loaders are LastOffset consumers: INSERT a fresh row after flip,
  do not rely on snapshot backlog
- rollback: re-register alpha config with original slot orm_oms_slot (slot
  still exists on alpha while dual-write window is open)
