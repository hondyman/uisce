# orm-oms-connector setup (live state as of 2026-09-20)

This file documents the **live** registered Debezium connector, NOT the original
setup steps. The original AGENTS.md narrative about a crims-targeted connector
was wrong — the live connector reads from the **alpha** database's `orm` schema.

## Live config (verified 2026-09-20 via `GET /connectors/orm-oms-connector/config`)

- `connector.class`: `io.debezium.connector.postgresql.PostgresConnector`
- `database.hostname`: `100.84.50.65` (the remote Postgres at `100.84.50.65`, NOT
  `host.docker.internal` — that was a leftover from the original setup doc)
- `database.dbname`: `alpha` (NOT crims)
- `database.user`: `postgres`, `database.password`: `postgres` (dev)
- `database.sslmode`: `verify-full` with the .pk8 cert dance per AGENTS.md
- `topic.prefix`: `orm_oms`
- `schema.include.list`: `orm`
- `table.include.list`: `orm.execution, orm."order", orm.placement, orm.order_allocation, orm.execution_allocation` (5 of 7 orm.* tables; excludes `account` and `broker`)
- `publication.name`: `orm_cdc_publication` (on alpha, NOT crims)
- `slot.name`: `orm_oms_slot`
- `snapshot.mode`: `initial`
- `tombstones.on.delete`: `false`

## Where the mTLS certs live (live state)

The certs are mounted into the `uisce-debezium` container (the Kafka Connect
worker in `docker-compose.remote.yml`) at:

- `/tmp/orm_ca.crt` — host `/home/eganpj/.uisce/certs/ca.crt`
- `/tmp/orm_client.crt` — host `/home/eganpj/.uisce/certs/postgres-client.crt`
- `/tmp/orm_client_der.pk8` — host `/tmp/orm_client_der.pk8` (copied because
  the original at `~/.uisce/certs/postgres-client.pk8` is owned by uid 1000
  but the Connect JVM runs as uid 1001 — see AGENTS.md operational notes)

**Operational landmine:** these certs live in the container's `/tmp/`, so they
vanish on container recreation. Re-mount from the host paths after any
`docker compose up -d --force-recreate uisce-debezium`.

## Register a new (or replacement) connector

The config is now in sync with the live state in `debezium/orm-oms-connector.json`.
To re-register:

```bash
curl -X POST http://100.84.50.65:8083/connectors \
  -H 'Content-Type: application/json' \
  -d @debezium/orm-oms-connector.json
```

## Verify

```bash
curl -s http://100.84.50.65:8083/connectors/orm-oms-connector/status
curl -s http://100.84.50.65:8083/connectors/orm-oms-connector/config | jq .
```

Status `tasks[].state` must be `"RUNNING"` (not `FAILED`) — see the
2026-09-13 incident in AGENTS.md for the original registration failure mode.
