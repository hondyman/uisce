# orm-oms-connector setup

Registers the Debezium Postgres connector that feeds the OMS CDC hot-tier
pipeline (`orm.*` tables -> Kafka -> `cmd/stream_loader` -> StarRocks
`oms.*`). These are one-time, host-side setup steps against the shared
`crims` database and the running `semlayer-debezium` Kafka Connect
instance - not something `docker compose up` recreates on its own, since
they touch data outside this compose stack.

## 1. Create the CDC publication (once, on `crims`)

```sql
CREATE PUBLICATION orm_cdc_publication FOR TABLES IN SCHEMA orm;
```

## 2. Copy the mTLS client cert material into the connector container

`crims` requires client-cert auth from the Docker bridge gateway IP.
The cert/key live in `tenant_product_datasource.config` (in the `alpha`
database) for the ORM datasource. The connector's bundled pgjdbc needs
the private key as **PKCS#8 DER** (not PEM) - see
`backend/cmd/stream_loader/main.go`'s header comment for the matching
Decimal-decoding note on the consumer side.

```bash
psql -U postgres -h localhost -d alpha -t -A \
  -c "SELECT config->>'ca_cert' FROM tenant_product_datasource WHERE id='441f62c9-aad1-481d-9aab-62943fa11cd3';" \
  > /tmp/orm_ca.crt
psql -U postgres -h localhost -d alpha -t -A \
  -c "SELECT config->>'client_cert' FROM tenant_product_datasource WHERE id='441f62c9-aad1-481d-9aab-62943fa11cd3';" \
  > /tmp/orm_client.crt
psql -U postgres -h localhost -d alpha -t -A \
  -c "SELECT config->>'private_key' FROM tenant_product_datasource WHERE id='441f62c9-aad1-481d-9aab-62943fa11cd3';" \
  > /tmp/orm_client.key

openssl pkcs8 -topk8 -nocrypt -in /tmp/orm_client.key -outform DER -out /tmp/orm_client_der.pk8

docker cp /tmp/orm_ca.crt semlayer-debezium:/tmp/orm_ca.crt
docker cp /tmp/orm_client.crt semlayer-debezium:/tmp/orm_client.crt
docker cp /tmp/orm_client_der.pk8 semlayer-debezium:/tmp/orm_client_der.pk8

# The Connect worker JVM runs as uid 1001 (kafka), not the uid docker cp
# leaves the files owned as - fix ownership or the connector fails with
# "Could not read SSL key file".
docker exec -u root semlayer-debezium chown kafka:kafka /tmp/orm_ca.crt /tmp/orm_client.crt /tmp/orm_client_der.pk8
docker exec -u root semlayer-debezium chmod 600 /tmp/orm_client_der.pk8

shred -u /tmp/orm_client.key /tmp/orm_client_der.pk8
```

## 3. Register the connector

`orm-oms-connector.json` ships with `database.password` as a placeholder -
substitute the real dev value (see `tenant_product_datasource.config` in
`alpha`, or `.env`) before posting, e.g.:

```bash
sed 's/REPLACE_WITH_POSTGRES_PASSWORD/postgres/' orm-oms-connector.json | \
  curl -X POST http://localhost:8083/connectors -H 'Content-Type: application/json' -d @-
```

## 4. Verify

```bash
curl -s http://localhost:8083/connectors/orm-oms-connector-v2/status
docker exec semlayer-redpanda rpk topic list | grep orm_oms
```

## 5. Two traps in this config (both caused a silent three-week outage)

### `table.include.list` must be the PLAIN form — do not add backslashes

The table is named `order` (lowercase; `pg_class.relname = 'order'`). Write it plainly:

```
orm.execution,orm.order,orm.placement,...
```

A previous deployment carried `orm.\"order\"`, on the theory that Debezium needs
SQL-keyword quoting. It does not here, and it is actively harmful: Debezium then looks
for a table whose name is literally `"order"` **including the quote characters**, matches
nothing, and silently captures zero rows from `orm."order"` while the other four tables
keep flowing. The symptom is a *partial* outage — `orm_oms.orm.placement` advances,
`orm_oms.orm.order` sits at its old high-water mark — which reads like a broken consumer
rather than a broken filter.

If you are tempted to add escaping at POST time, do not. Verified live 2026-10-07:
the plain `orm.order` form emitted all 187 rows; the escaped form emitted none.

`rpk topic describe -p` reports **only partition 0**. With more than one partition that
is a misleading spot-check — sum across all partitions instead:

```bash
docker exec uisce-redpanda rpk topic describe orm_oms.orm.order
```

### `decimal.handling.mode` must be `string`, not `double`

A double cannot hold a full `numeric(18,9)`. Measured end-to-end through
Postgres → Debezium → Redpanda → StarRocks on 2026-10-07:

| Postgres `numeric(18,9)` | via `double` | via `string` |
|---|---|---|
| `12345.123456789` | `12345.123456789` | `12345.123456789` |
| `1234567.123456789` | `1234567.123456789` | `1234567.123456789` |
| `12345678.123456789` | `12345678.123456790` (rounded) | exact |
| `999999999.999999999` | **`NULL`** | exact |

The last row is the dangerous one: the double rounds to `1e9`, which **overflows**
`DECIMAL(18,9)`, and StarRocks stores NULL rather than erroring. An amount column
silently becomes null with no DLQ entry and no loader error.

The stream loader needs no change for `string` mode — it passes the decimal through as
a JSON string and StarRocks casts it into the `DECIMAL` column.

### Preventing recurrence

Both failures were invisible to "no errors in the logs". A scheduled parity check
(row counts per table, Postgres vs StarRocks) would have caught the `order` table's
silent absence on day one.
