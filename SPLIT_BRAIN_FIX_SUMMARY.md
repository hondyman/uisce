# Split-Brain 500 → 503 Defensive Refactor

**Date:** 2026-07-07
**Symptom:** `/api/catalog/nodes`, `/api/semantic/bundles/*`, and
`/api/node-types/*` returned **HTTP 500 Internal Server Error** when Trino /
Temporal / database handshake failed during backend startup. The FE could not
distinguish a real bug from a transient outage.

**Root cause:** Several handlers in `internal/api/` dereferenced
`*sql.DB`, `temporalclient.Client`, and `Server` fields without nil-guards.
When the `Server` was constructed with a `nil` DB (caused by the cascade of
Trino/Temporal init failures logged at startup), the handler panicked; the
chi middleware converted the panic into `500 Internal Server Error`.

## Fix summary

All metadata handlers now check for a nil `*sql.DB` and return a structured
**503 Service Unavailable** with the `db_unavailable` error code instead of
panicking. The central `respond()` helper additionally maps connection-class
errors (`sql.ErrConnDone`, `driver.ErrBadConn`, `connection refused`,
`context deadline exceeded`, `too many connections`, etc.) onto 503 so that
transient transport failures are correctly signalled to the FE.

### Files changed

| File | Change |
|---|---|
| `backend/internal/api/api.go` | Added `s.DB == nil` 503 guards to `handleListCatalogNodes` and `getSemanticBundle`. |
| `backend/internal/api/node_types_routes.go` | Added `requireDB(db, h)` wrapper applied to every `/api/node-types/*` route. |
| `backend/internal/api/helpers.go` | Added `isConnectionError(err)` and made `respond()` map connection errors onto 503. |
| `backend/internal/api/semantic_layer_handler.go` | Replaced plain-text 500 in `GetBundle` with a structured 503 JSON envelope. |
| `backend/internal/api/defensive_handlers_test.go` | **New.** Locks in the behavior of `requireDB`, `isConnectionError`, `respond()` and `handleListCatalogNodes` (14 sub-tests, all passing). |
| `backend/internal/metadata/businessobject_service.go` | **Already aligned** (per earlier diff): both `ListBusinessObjects` and `GetBusinessObject` read from `legacy_business_objects` with the same column map → no more "list has row, get returns 404" split-brain on this code path. |

### What the FE gets now

| Scenario | Before | After |
|---|---|---|
| `Server{DB: nil}` (Trino/Temporal init failed) | 500 plain-text panic | **503** `{error, code, errorCode: "db_unavailable"}` |
| Pool exhausted / connection refused / timeout | 500 | **503** `db_unavailable` (auto-detected by `respond`) |
| Real bug (bad SQL, etc.) | 500 | **500** `internal_error` (unchanged) |
| `/api/semantic/bundles/{domain}` with `h.server == nil` | 500 plain-text | **503** `server_unavailable` JSON envelope |
| Successful query | 200 JSON | **200** JSON (unchanged) |

### Cardinal rule alignment

| Rule | Status |
|---|---|
| 1 — Config-before-code | ✅ — no new `if feeSchedule.Type == "..."`-style hardcoding introduced; new guards use the existing `writeJSONError` envelope. |
| 2 — Graph-first | ✅ — handlers still driven by existing graph/DB endpoints; no business logic moved to Go. |
| 3 — No package cycles | ✅ — `helpers.go` is the same package; no new external imports. |
| 4 — Hot/cold watermark | ✅ — `*sql.DB` access is through a single helper; not relevant to HWM but the seam is preserved. |
| 5 — Graph-driven routing | ✅ — defensive guards do **not** alter routing behavior. |
| 6 — Semantic/OLTP boundary | ✅ — no change; financial state still lives in OLTP tables. |
| 7 — Security (tenant ownership) | ✅ — guards predate tenant resolution, so `RequireTenantOwnership` is unchanged. |

## Verification

```
$ go build ./...
exit 0

$ go vet ./internal/api/... ./internal/metadata/...
exit 0   (only third-party m1cpu warnings; no issues in our code)

$ go test ./internal/api/ -run 'TestRequireDB|TestIsConnectionError|TestRespond_|TestListCatalogNodes_NilDB' -v
... 14/14 PASS ...
exit 0
```

Pre-existing test failure (out of scope):
`TestListBusinessObjects_UsesNewDatasourceHeader` in
`internal/api/business_object_handlers_test.go` was already failing
before any of these changes (verified by stashing my work and re-running).
It tests `X-Datasource-Id` header extraction by the business-object
handlers — unrelated to split-brain / nil-DB symptoms.
