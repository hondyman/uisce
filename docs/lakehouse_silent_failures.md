# Lakehouse silent-failure inventory

Every place the lakehouse can be *wrong without being visibly broken*. Compiled
2026-10-07 while fixing the Debezium → Redpanda → StarRocks path.

The theme: all three entries return success while producing incorrect data. None
of them log an error, none raise, and none appear in a health check.

## 1. StarRocks stream load silently NULLs unconvertible values — FIXED

**Where:** `backend/cmd/stream_loader` → `_stream_load` into `oms.*`.

**Behaviour (measured on 3.3.22):** a `DECIMAL` column fed `"NOT_A_NUMBER"` is
stored as NULL, and the reply is `Status=Success` with `NumberLoadedRows: 1` and
`NumberFilteredRows: 0`. Nothing downstream can tell. Reading
`NumberFilteredRows` does **not** help — that counter only reflects strict-mode
rejections, which is why a first attempt at detection passed the test suite and
then failed live.

**Fix:** `strict_mode: true` by default (`STREAM_LOAD_STRICT_MODE=false` opts
out), plus `loadRows` bisection so a rejected batch is split to isolate the bad
row instead of blocking all 2,000. Bad rows go to the DLQ and increment
`rows_rejected_by_starrocks`. Transport and auth errors are not bisected.

## 2. Cube materialization NULLs the same way — OPEN

**Where:** `backend/internal/querybuilder/cube_materialize_cold.go` →
`CREATE TABLE ... AS SELECT` over the `pg_alpha` JDBC catalog, and the async
materialized views in `gold` / `tenant_<uuid>`.

**Why it matters more than entry 1:** this is the *aggregate read path*. A wrong
cube number looks plausible — there is no source row to compare it against, and
downstream consumers trust it. It has none of the protection entry 1 now has.

**Not yet done.** It needs the same treatment: verify whether StarRocks CTAS over
a JDBC catalog reports filtered rows, and if not, compare a materialized cube
against its source aggregate.

## 3. Iceberg compaction returns hardcoded results — OPEN, DELETE RATHER THAN FIX

**Where:** `backend/internal/metadata/lakehouse_maintenance.go`.

It builds a `CALL system.rewrite_data_files(...)` string, assigns it to
`_ = compactionSQL // Executed via ...`, and returns literal
`CompactedFilesCount: 42, BytesCompacted: 2147483648`. The SQL is never executed.

**Recommendation: delete it, not fix it.** A stub that reports success is worse
than a missing feature — anything consuming those numbers trusts them. Until the
Iceberg workstream lands with a real Spark/Glue job, the honest state is "not
implemented".

## Detection

`scripts/cdc_value_audit.sh` compares Postgres and StarRocks per value, not per
count. Run it after any change to the loader, the connector config, or the
`orm.*` schemas. It currently reports all 15 numeric columns matching
byte-for-byte.

Note the two audit methods that gave false results, so nobody re-introduces them:
row **counts** miss non-null-but-wrong values, and order-dependent **hashes**
across two engines produce false positives from collation differences.