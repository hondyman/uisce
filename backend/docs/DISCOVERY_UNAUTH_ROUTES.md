# Discovery Pass — Unauthenticated Route Reachability (2026-09-07)

Read-only reconnaissance for Fix 2 (the route-layer authentication gate).
This is the allowlist input, produced by observation rather than by what
breaks after enforcement — per the agreed rollout: discovery pass (this
document) → shadow mode (log-only, e2e against it) → enforce, with the
allowlist and violation log as the PR's review artifact.

**Method:** every `GET` route registered in `internal/api/api.go`'s route
dump (`/tmp` build of `main`, chi's `[ROUTE]` log line) was curled with no
`Authorization` header and no valid session, against a live-data instance
connected to the `alpha` tenant database, path parameters substituted
with a placeholder UUID. 410 GET routes probed.

**Scope limitation, stated rather than silently skipped:** only `GET`
routes were invoked. This repo registers 489 non-`GET` routes (348 POST,
73 DELETE, 55 PUT, 13 PATCH); blind-firing those unauthenticated against
live shared `alpha` data risked real mutation, and that risk is not worth
taking for a reconnaissance pass. Those routes still need to be inventoried
for the gate — by reading each handler's own auth check rather than by
invocation — before Fix 2's allowlist is complete. Flagged as **not yet
done**, not silently treated as safe.

## Result summary (GET routes only, n=410)

| HTTP status | Count | What it means here |
|---|---|---|
| 401 | 165 | Already gated by a handler-level check — fine as-is under the new global gate, though each is still a hand-rolled implementation the gate should eventually make redundant |
| 400 | 68 | Rejected for a bad/missing parameter before reaching data — not an auth signal either way, re-classify once real IDs are used |
| 404 | 30 | Not found for the placeholder ID — needs a retest with a real ID before it can be allowlisted as "safe because empty" |
| 500 | 37 | **Errors before any auth check runs** — the "masked by luck, not by design" pattern, now with a count: 37 routes whose current safety (such as it is) depends on an unrelated bug firing first, not on any deliberate check |
| 000 | 11 | curl-level failure (timeout / connection reset) during the probe — several are calls to unreachable external services (Slack OAuth, Prometheus) that hang from this box; not yet distinguished from a real hang, re-probe in isolation |
| 301/307 | 2 | Redirects — target not yet inspected |
| **200** | **93** | **Returns a response with no auth at all — the actual allowlist decision set, below** |

## The 200 bucket, by disposition

- **5 admin-path endpoints reachable with zero auth**, including
  `/api/admin/tenants/{tenantId}/configuration` (and its `/api/v1/` twin)
  and `/api/admin/llm/config`. These return real configuration, not empty
  placeholders. Highest-priority gate candidates.
- **~55 endpoints return real records without auth** — tenant lists
  (`/api/tenants/all` returns tenant IDs and display names), catalog
  metadata (`/api/rest/catalog-nodes`, `/api/catalog/nodes`,
  `/api/rest/catalog-edges`), semantic terms, datasource configs
  (`/api/api-dispatcher/datasources` — checked for embedded credentials,
  found none in this instance's data, but the shape of the response would
  carry them if a datasource had them), fund/bundle reference data, DAX
  function definitions, report calendars, and others. Some of this looks
  like it was intended to be public reference data (DAX function list,
  report calendars); some clearly was not (`/api/tenants/all`).
- **~35 endpoints return empty/trivial payloads** (`[]`, `null`, `{}`, or
  a zero-count wrapper) with no auth. Not a data leak today, but each one
  still needs an explicit "yes this is meant to be public" or "gate it"
  decision — an empty response today doesn't mean the underlying handler
  intended to allow anonymous access; several of these are almost
  certainly just not-yet-populated tables behind gate-worthy handlers.

## Full table

| Route | Status | Disposition | Response sample |
|---|---|---|---|
| `GET /api/admin/impersonate/sessions/active` | 200 | GATE — HIGH — admin path, no auth | `{"active_sessions":[],"count":0}` |
| `GET /api/admin/impersonate/sessions/recent` | 200 | GATE — HIGH — admin path, no auth | `{"count":0,"recent_sessions":[]}` |
| `GET /api/admin/llm/config` | 200 | GATE — HIGH — admin path, no auth | `{"provider":"gemini","model":"gemini-2.0-flash-exp","embedding_model":` |
| `GET /api/admin/tenants/{tenantId}/configuration` | 200 | GATE — HIGH — admin path, no auth | `{"configuration":{"features":{"enableAdvancedLineage":true,"enableAiCo` |
| `GET /api/v1/admin/tenants/{tenantId}/configuration` | 200 | GATE — HIGH — admin path, no auth | `{"configuration":{"features":{"enableAdvancedLineage":true,"enableAiCo` |
| `GET /_routes` | 200 | GATE — data exposure | `{"routes":["GET /_routes","POST /ai/analyze-signal","GET /ai/drift-pre` |
| `GET /api/abbreviations/` | 200 | GATE — data exposure | `{"items":[{"id":233,"abbreviation":"ACC","full_word":"ACCOUNT","notes"` |
| `GET /api/abbreviations/export` | 200 | GATE — data exposure | `[{"id":233,"abbreviation":"ACC","full_word":"ACCOUNT","notes":"Account` |
| `GET /api/agentic/tickets` | 200 | GATE — data exposure | `{"tickets":null}` |
| `GET /api/api-dispatcher/datasources` | 200 | GATE — data exposure | `{"data":[{"config":{"default_auth":"basic_auth","default_base_url":"ht` |
| `GET /api/api-dispatcher/endpoints` | 200 | GATE — data exposure | `{"data":[{"config":{},"datasource_id":"","datasource_name":"API Servic` |
| `GET /api/api-dispatcher/semantic-terms` | 200 | GATE — data exposure | `{"data":[{"data_type":"string","description":"","id":"44794dd9-98e1-4c` |
| `GET /api/api/rdl/templates/` | 200 | GATE — data exposure | `{"count":5,"templates":[{"id":"tlh_us_standard","name":"US Tax-Loss Ha` |
| `GET /api/bo/{boId}/status` | 200 | GATE — data exposure | `{"status":"draft","reason":"","pending_terms":[],"pending_calculations` |
| `GET /api/bundles/` | 200 | GATE — data exposure | `[{"id":"fof_private_markets_bundle","name":"FoF Private Markets Bundle` |
| `GET /api/business-objects/{boId}/status` | 200 | GATE — data exposure | `{"status":"draft","reason":"","pending_terms":[],"pending_calculations` |
| `GET /api/catalog/nodes` | 200 | GATE — data exposure | `[{"catalog_type":"table","created_at":"2026-08-11T19:48:10.39073Z","de` |
| `GET /api/catalog/semantic-terms-by-table/{tableId}` | 200 | GATE — data exposure | `{"semanticTerms":[]}` |
| `GET /api/chart/{datasourceId}/debug` | 200 | GATE — data exposure | `{"success":true,"data":"Debug output written to logs","metadata":{"dat` |
| `GET /api/chart/{datasourceId}/health` | 200 | GATE — data exposure | `{"success":true,"data":{"charts":[],"integrity":{"enhanced_erd_chart":` |
| `GET /api/charts/{datasourceId}` | 200 | GATE — data exposure | `{"success":true,"data":[],"metadata":{"dataSource":"11111111-1111-1111` |
| `GET /api/dax/categories` | 200 | GATE — data exposure | `{"categories":["information","time_intelligence","statistical","iterat` |
| `GET /api/dax/functions` | 200 | GATE — data exposure | `{"count":27,"functions":[{"category":"statistical","description":"Retu` |
| `GET /api/financial/household/optimize-harvesting` | 200 | GATE — data exposure | `{"householdId":"","opportunities":[{"household_id":"HH-SMITH-FAMILY","` |
| `GET /api/funds` | 200 | GATE — data exposure | `[{"id":"13217c91-d15d-4e20-9d3c-f15af33c2d33","name":"Tech Growth Fund` |
| `GET /api/glossary/node-graph` | 200 | GATE — data exposure | `{"edges":[],"nodes":[]}` |
| `GET /api/glossary/technical-assets` | 200 | GATE — data exposure | `{"data":[],"total":0}` |
| `GET /api/governance/certification/evaluate` | 200 | GATE — data exposure | `{"bo_id":"customers","is_certified":true,"passed_count":2,"violations"` |
| `GET /api/lineage/node/{id}/graph` | 200 | GATE — data exposure | `{"nodes":[],"edges":[]}` |
| `GET /api/lineage/node/{id}/impact` | 200 | GATE — data exposure | `{"nodes":[],"edges":[]}` |
| `GET /api/llm/modes` | 200 | GATE — data exposure | `{"default_mode":"exploratory","example_request":{"datasource":"custome` |
| `GET /api/llm/prompts` | 200 | GATE — data exposure | `{"documentation":"See https://github.com/yourusername/semlayer/wiki/LL` |
| `GET /api/metrics/global` | 200 | GATE — data exposure | `{"commitSuccessRate":0,"s3Failures5m":0,"idempotencyHits5m":0,"regions` |
| `GET /api/platform-billing/anomalies` | 200 | GATE — data exposure | `{"tenantAnomalies":[],"regionAnomalies":[],"costAnomalies":[]}` |
| `GET /api/platform-billing/forecast` | 200 | GATE — data exposure | `{"forecastUSD":0,"model":"exponential_smoothing","confidence":0.85}` |
| `GET /api/platform-billing/platform` | 200 | GATE — data exposure | `{"window":"30d","totals":{"computeUSD":0,"storageUSD":0,"eventsUSD":0,` |
| `GET /api/rbac/roles/{roleId}/effective-permissions` | 200 | GATE — data exposure | `{"permissions":[],"role_id":"11111111-1111-1111-1111-111111111111"}` |
| `GET /api/relationships/{entityID}` | 200 | GATE — data exposure | `{"entityId":"11111111-1111-1111-1111-111111111111","relationships":[]}` |
| `GET /api/relationships/{entityID}/objects` | 200 | GATE — data exposure | `{"entityId":"11111111-1111-1111-1111-111111111111","objects":[]}` |
| `GET /api/reports/calendars` | 200 | GATE — data exposure | `[{"calendar_code":"NYSE","calendar_name":"New York Stock Exchange","ti` |
| `GET /api/rest/catalog-edges` | 200 | GATE — data exposure | `[{"edge_type_id":"28c5811a-3c4f-4552-82eb-3f2a9d35d988","id":"990e85d7` |
| `GET /api/rest/catalog-node-types` | 200 | GATE — data exposure | `[{"catalog_type_name":"schema","description":"Schema","id":"68d6d495-0` |
| `GET /api/rest/catalog-nodes` | 200 | GATE — data exposure | `[{"catalog_type":"table","created_at":"2026-08-11T19:48:10.39073Z","de` |
| `GET /api/rest/datasources` | 200 | GATE — data exposure | `{"alpha_datasource":[{"datasource_code":"SNOWFLAKE","datasource_name":` |
| `GET /api/rest/products` | 200 | GATE — data exposure | `{"alpha_product":[{"id":"afb9a8c0-3700-427c-a6ab-a30741fab8a6","is_act` |
| `GET /api/semantic-terms` | 200 | GATE — data exposure | `{"data":[{"id":"44794dd9-98e1-4cab-a7da-2cc271dbf56b","node_name":"Acc` |
| `GET /api/semantic/name-resolver/stats` | 200 | GATE — data exposure | `{"alias_count":0,"cache_age_sec":9223372036.854776,"field_count":0,"la` |
| `GET /api/simulations/scenarios` | 200 | GATE — data exposure | `{"scenarios":null}` |
| `GET /api/tenants/all` | 200 | GATE — data exposure | `[{"id":"99e99e99-99e9-49e9-89e9-99e99e99e999","display_name":"Northwin` |
| `GET /api/tenants/custom-attributes` | 200 | GATE — data exposure | `{"attributes":null,"boId":"","tenantId":"core"}` |
| `GET /api/tenants/gold-copy` | 200 | GATE — data exposure | `{"id":"99e99e99-99e9-49e9-89e9-99e99e99e999","gold_copy":true,"resolve` |
| `GET /api/v1/bo/{boId}/status` | 200 | GATE — data exposure | `{"status":"draft","reason":"","pending_terms":[],"pending_calculations` |
| `GET /api/v1/business-objects/{boId}/status` | 200 | GATE — data exposure | `{"status":"draft","reason":"","pending_terms":[],"pending_calculations` |
| `GET /api/v1/folders/{id}/analytics` | 200 | GATE — data exposure | `{"run_count_30d":152,"export_count_30d":12,"viewer_count_30d":8,"updat` |
| `GET /api/v1/pipelines/activities/safe` | 200 | GATE — data exposure | `{"activities":null}` |
| `GET /api/v1/reports/` | 200 | GATE — data exposure | `[{"id":"f48a510c-1fa3-5054-85ac-10e067480625","tenant_id":"99e99e99-99` |
| `GET /api/v1/triggers/operators` | 200 | GATE — data exposure | `[{"id":"ZDVhMzcxMDEtYzFmZC00MzZkLWIwNmQtMjA5YjQyNDMzMTRm","key":"betwe` |
| `GET /api/v1/triggers/types` | 200 | GATE — data exposure | `[{"category":"data","description":"Fires when a field value changes on` |
| `GET /api/validation-rule-cores` | 200 | GATE — data exposure | `{"rules":[],"total":0}` |
| `GET /api/validation-rule-cores/{id}/impact` | 200 | GATE — data exposure | `{"core_rule_id":"11111111-1111-1111-1111-111111111111","items":[],"tot` |
| `GET /health` | 200 | GATE — data exposure | `{"status":"healthy","timestamp":"2026-09-07T20:11:15-04:00"}` |
| `GET /api/auth/users/{userId}/preferences/` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"Failed to fetch preferences"}` |
| `GET /api/bp-designer/{bpDefId}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: relation "business_process_definition" does not exist at position ` |
| `GET /api/business-objects/bindings` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `Failed to fetch bindings: pq: column "binding_id" does not exist at co` |
| `GET /api/calculations/` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `missing destination name tier in *[]models.Calculation` |
| `GET /api/catalog/business-terms/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `Failed to fetch business term` |
| `GET /api/integrations/marketplace` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `Failed to fetch integrations: sql: Scan error on column index 10, name` |
| `GET /api/marketplace/items` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"Failed to scan marketplace item","code":500,"error_code":"sc` |
| `GET /api/marketplace/items/{id}/feedback` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"Database error","code":500,"error_code":"db_error","details"` |
| `GET /api/marketplace/validation-rules` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"Database scan error","code":500,"error_code":"db_error","det` |
| `GET /api/metrics/region-heatmap` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"failed to query heatmap data: prometheus query failed: Get \` |
| `GET /api/metrics/{fundId}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"private_markets_metrics\" does not exist at p` |
| `GET /api/plans/timeline` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"failed to query timeline: prometheus query failed: Get \"htt` |
| `GET /api/reports/batches/{id}/telemetry` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `batch not found: pq: relation "public.report_burst_batches" does not e` |
| `GET /api/reports/schedules/{id}/batches` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: relation "public.report_burst_batches" does not exist at position ` |
| `GET /api/rule-fabric/action-types` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `failed to fetch action types: sql: Scan error on column index 3, name ` |
| `GET /api/rule-fabric/action-types/{category}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `failed to fetch action types: pq: invalid input value for enum rule_ca` |
| `GET /api/scheduler/governance/changesets/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `sql: no rows in result set` |
| `GET /api/templates/` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `Failed to fetch templates: sql: Scan error on column index 5, name "ta` |
| `GET /api/templates/categories` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"template 'categories' not found","code":500,"error_code":"in` |
| `GET /api/templates/clones` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"template 'clones' not found","code":500,"error_code":"intern` |
| `GET /api/templates/featured` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"template 'featured' not found","code":500,"error_code":"inte` |
| `GET /api/templates/{id}/stats` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `Failed to fetch stats: sql: Scan error on column index 1, name "total_` |
| `GET /api/templates/{key}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"template '11111111-1111-1111-1111-111111111111' not found","` |
| `GET /api/templates/{node_id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"template '11111111-1111-1111-1111-111111111111' not found","` |
| `GET /api/user/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: operator does not exist: text = uuid at position 7:11 (4` |
| `GET /api/v1/timeouts/pending` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: relation "step_timeouts" does not exist at position 4:14 (42P01)` |
| `GET /api/v1/triggers` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: invalid input syntax for type uuid: "" (22P02)` |
| `GET /api/v1/triggers/executions` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: relation "trigger_executions" does not exist at column 70 (42P01)` |
| `GET /api/v1/triggers/objects` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: column "name" does not exist at column 12 (42703)` |
| `GET /api/workflows/{workflowId}/events` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `pq: relation "business_process_event" does not exist at position 3:14 ` |
| `GET /reports/extensions/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"report_extensions\" does not exist at column ` |
| `GET /reports/instances/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"report_instances\" does not exist at column 1` |
| `GET /reports/instances/{id}/download` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"report_instances\" does not exist at column 1` |
| `GET /reports/packages` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"report_packages\" does not exist at column 15` |
| `GET /reports/schedules/{id}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"pq: relation \"report_schedules\" does not exist at column 1` |
| `GET /scheduler/jobs/{id}/runs` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `{"error":"missing destination name output_artifacts in *[]scheduler_in` |
| `GET /values/profiles/{clientID}` | 500 | MASKED BY LUCK — errors before reaching any auth check; fails for the wrong reason | `failed to get client values profile: sql: no rows in result set` |
| `GET /api/admin/quotas` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"provider":"gemini","model":"gemini-2.0-flash-exp","embedding_model":` |
| `GET /api/altinvest/client/{clientId}` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"error":"unauthorized"}` |
| `GET /api/households/{id}/entities` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"bo_id":"customers","is_certified":true,"passed_count":2,"violations"` |
| `GET /api/succession/metrics/{advisorId}` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `<a href="https://slack.com/oauth/v2/authorize?client_id=FAKE&amp;scope` |
| `GET /api/succession/recommend/{advisorId}` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `<a href="https://slack.com/oauth/v2/authorize?client_id=FAKE&amp;scope` |
| `GET /api/taxplan/opportunities/{clientId}` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `<a href="https://slack.com/oauth/v2/authorize?client_id=FAKE&amp;scope` |
| `GET /api/v1/pipelines` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"error":"unauthorized"}` |
| `GET /reports/definitions/` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `tenant_id is required` |
| `GET /reports/extensions/` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"error":"definition not found"}` |
| `GET /reports/instances/` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"error":"pq: relation \"report_extensions\" does not exist at column ` |
| `GET /reports/schedules/` | 000 | Connection error/timeout during probe — outbound dependency (external network) or hang; re-probe in isolation | `{"error":"pq: relation \"report_packages\" does not exist at column 15` |
| `GET /api/api/process-benchmarking/best-practices` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/data-domains/` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/data-domains/search` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/debug/edges/{id}` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/execution-logs/` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/explorer/query/history` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/explorer/saved-queries/` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/explorer/saved-queries/duplicates` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/explorer/saved-queries/{id}` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/explorer/saved-queries/{id}/diff` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/explorer/saved-queries/{id}/preview` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/integrations/marketplace/category/{category}` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/marketplace/items/{id}/parameters` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/migrations/` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/nba/catalog` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/nba/stats` | 200 | Public/allowlist candidate — empty payload | `{}` |
| `GET /api/query/history` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/rbac/delegations/user/{userId}` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/rbac/teams/{teamId}/members` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/reports/schedules` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/saved/` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/saved/{id}` | 200 | Public/allowlist candidate — empty payload | `` |
| `GET /api/scheduler/governance/audit/entity/{type}/{id}` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/scheduler/governance/policies` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /api/templates/categories/{key}/templates` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/templates/category/{category}` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/templates/{id}/ratings` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/templates/{node_id}/versions` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/users/{user_id}/roles/` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /api/v1/folders/` | 200 | Public/allowlist candidate — empty payload | `[]` |
| `GET /scheduler/dags/{id}/runs` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /values/themes` | 200 | Public/allowlist candidate — empty payload | `null` |
| `GET /ai/drift-predictions` | 401 | Already gated by a handler-level check | `Missing or invalid tenant` |
| `GET /ai/rule-suggestions` | 401 | Already gated by a handler-level check | `Missing or invalid tenant` |
| `GET /ai/rule-templates` | 401 | Already gated by a handler-level check | `Missing or invalid tenant` |
| `GET /api/admin/tenant-access/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/admin/tenants/audit-logs` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/admin/tenants/deltas` | 401 | Already gated by a handler-level check | `Failed to get security context: datasource resolver not configured (in` |
| `GET /api/altinv/alternative-investments/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/altinv/alternative-investments/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/api-dispatcher/connections` | 401 | Already gated by a handler-level check | `unauthorized` |
| `GET /api/api/metrics/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/api/metrics/{metricID}/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/api/metrics/{metricID}/anomalies` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/api/metrics/{metricID}/runs` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/api/process-benchmarking/gap-analysis` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/api/process-benchmarking/peers` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/api/process-benchmarking/score` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/api/process-monitor/active-instances` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-monitor/instance/{workflowID}` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-monitor/instance/{workflowID}/history` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-monitor/stats` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-monitor/ws` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-optimization/applied` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-optimization/auto-tune/status` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-optimization/forecast/{suggestionID}` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/process-optimization/suggestions` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/api/rdl/rules/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/api/rdl/rules/{ruleID}/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/api/rules/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/api/rules/{ruleID}/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/api/v1/audit/channel-billing` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/api/v1/audit/channel-logs` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/api/v1/lineage/node/{id}/blast-radius` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/audit/events` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/audit/ledger/verify` | 401 | Already gated by a handler-level check | `tenant_id required (from JWT)` |
| `GET /api/audit/stats` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/auth/me` | 401 | Already gated by a handler-level check | `unauthenticated` |
| `GET /api/bo/{boKey}/records` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/bo/{boKey}/records/{recordId}` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/bo/{boKey}/records/{recordId}/relationships/{relKey}` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/bo/{boKey}/topology-summary` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/bp-notifications/analytics` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized: authentication required: missing or invalid JW` |
| `GET /api/bp-notifications/digests/pending` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized: authentication required: missing or invalid JW` |
| `GET /api/bp-notifications/logs` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized: authentication required: missing or invalid JW` |
| `GET /api/bp-notifications/preferences` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized: authentication required: missing or invalid JW` |
| `GET /api/bp-notifications/templates` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized: authentication required: missing or invalid JW` |
| `GET /api/business-term-edges` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/business-terms` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/cash-flow/settlements/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/cash-flow/settlements/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/custom-components` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized","code":401,"error_code":"unauthorized","detail` |
| `GET /api/custom-components/export` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized","code":401,"error_code":"unauthorized","detail` |
| `GET /api/custom-components/{id}` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized","code":401,"error_code":"unauthorized","detail` |
| `GET /api/edge-types` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/edge-types/{id}` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/edge-types/{id}/properties` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/field-aliases/{field_id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/glossary/business-terms` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/glossary/edges` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/glossary/semantic-terms` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/glossary/semantic-terms/export/cube-yaml` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/integrations/executions` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/integrations/executions/{executionId}` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/integrations/installed` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/integrations/installed/{installationId}` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/integrations/installed/{installationId}/stats` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/integrations/oauth/authorize/{installationId}` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/intelligence/storage/plans` | 401 | Already gated by a handler-level check | `security context initialization failed: authentication required: missi` |
| `GET /api/invoices/` | 401 | Already gated by a handler-level check | `authentication required` |
| `GET /api/ip-whitelist` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/layouts` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/layouts/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/marketplace/browse` | 401 | Already gated by a handler-level check | `{"error":"unauthorized","message":"Valid authentication required"}` |
| `GET /api/marketplace/installations` | 401 | Already gated by a handler-level check | `{"error":"unauthorized","message":"Valid authentication required"}` |
| `GET /api/marketplace/product-evolution` | 401 | Already gated by a handler-level check | `{"error":"unauthorized","message":"Valid authentication required"}` |
| `GET /api/marketplace/tenant-items` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/marketplace/tenant-items/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/customers/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/customers/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/personnel/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/personnel/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/sales-ledgers/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/sales-ledgers/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/vendors/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/master/vendors/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/metadata/versions/{bo_id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/my-approvals/` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/nba/recommendations` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/nba/signals` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/node-types` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/node-types/{id}` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/node-types/{id}/nodes` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/node-types/{id}/properties` | 401 | Already gated by a handler-level check | `authentication required: missing or invalid JWT token` |
| `GET /api/oms/accounts/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/accounts/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/positions/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/positions/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/securities/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/securities/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/trade-orders/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/oms/trade-orders/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/plans` | 401 | Already gated by a handler-level check | `authentication required` |
| `GET /api/rbac/audit` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/delegations` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/field-permissions` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/field-permissions/user/{userId}/resource/{resourceType}/{resourceId}` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/permissions` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/permissions/user/{userId}` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/roles` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/roles/{roleId}/users` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/teams` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/users` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/rbac/users/{userId}/roles` | 401 | Already gated by a handler-level check | `Unauthorized: authentication required: missing or invalid JWT token` |
| `GET /api/relationships/bo/{boId}/drill-target/{termId}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/roles/` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/security/mappings/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/security/profiles/` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/semantic-mapping/wizard/created` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized"}` |
| `GET /api/semantic-mapping/wizard/pending` | 401 | Already gated by a handler-level check | `{"error":"Unauthorized"}` |
| `GET /api/semantic-mappings` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/semantic-terms/{id}/suggest-business-terms` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/semantic/bundles/by-id` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/semantic/objects` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/tempo/traces` | 401 | Already gated by a handler-level check | `{"error":"Authorization or validation error","code":401,"error_code":"` |
| `GET /api/tempo/traces/{traceId}` | 401 | Already gated by a handler-level check | `{"error":"Authorization or validation error","code":401,"error_code":"` |
| `GET /api/tenants` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/tenants/accessible` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/tenants/debug` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/tenants/{tenantId}/ip-whitelist` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/v1/admin/tenants/audit-logs` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/v1/exports/{exportId}` | 401 | Already gated by a handler-level check | `{"error":"Missing or invalid tenant"}` |
| `GET /api/v1/exports/{exportId}/download` | 401 | Already gated by a handler-level check | `{"error":"Missing or invalid tenant"}` |
| `GET /api/v1/goldcopy/portfolio/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/v1/goldcopy/portfolio/{portfolioId}` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/v1/goldcopy/portfolio/{portfolioId}/lineage` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/v1/jobs/{jobId}/exports/` | 401 | Already gated by a handler-level check | `{"error":"Missing or invalid tenant"}` |
| `GET /api/v1/metrics/commit` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /api/v1/page-layouts` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/v1/page-layouts/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/v1/pipelines/{id}` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /api/v1/schedules/` | 401 | Already gated by a handler-level check | `{"error":"Missing or invalid tenant"}` |
| `GET /api/v1/schedules/{scheduleId}` | 401 | Already gated by a handler-level check | `{"error":"Missing or invalid tenant"}` |
| `GET /api/v1/tenant/entitlements` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/v1/tenant/entitlements/effective` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/v1/tenant/policies` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/v1/tenant/profiles` | 401 | Already gated by a handler-level check | `{"error":"unauthorized or missing tenant scope"}` |
| `GET /api/validation-rules` | 401 | Already gated by a handler-level check | `{"error":"Security context initialization failed","code":401,"error_co` |
| `GET /api/validation-rules/schema` | 401 | Already gated by a handler-level check | `{"error":"Security context initialization failed","code":401,"error_co` |
| `GET /api/validation-rules/{id}` | 401 | Already gated by a handler-level check | `{"error":"Security context initialization failed","code":401,"error_co` |
| `GET /api/validation-rules/{id}/audit` | 401 | Already gated by a handler-level check | `{"error":"Security context initialization failed","code":401,"error_co` |
| `GET /api/workflows/initiatable` | 401 | Already gated by a handler-level check | `Unauthorized` |
| `GET /portfolio/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /portfolio/sources/` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /scheduler/ai/suggestions` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /scheduler/dags` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /scheduler/jobs` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /scheduler/stats` | 401 | Already gated by a handler-level check | `{"error":"unauthorized"}` |
| `GET /sources/analytics` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /sources/analytics/confidence` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /sources/analytics/rank` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /sources/exceptions` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /sources/preferences` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /v1/portfolio/analytics/sources` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /v1/portfolio/analytics/trends` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /v1/portfolio/analytics/{portfolioId}` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /v1/security/{securityId}/lineage` | 401 | Already gated by a handler-level check | `tenant_id is required` |
| `GET /api/admin/tenants/{tenantID}/scope` | 403 | Already blocked (region/tenant check) — verify not auth-shaped | `security boundary violation: request has no verified tenant context` |
| `GET /api/metrics/tenant/{tenantId}` | 403 | Already blocked (region/tenant check) — verify not auth-shaped | `{"error":"region 'us-west' is not allowed for tenant '11111111-1111-11` |
| `GET /api/platform-billing/tenant/{tenantId}` | 403 | Already blocked (region/tenant check) — verify not auth-shaped | `{"error":"region 'us-west' is not allowed for tenant '11111111-1111-11` |
| `GET /api/users` | 403 | Already blocked (region/tenant check) — verify not auth-shaped | `Forbidden` |
| `GET /api/api-dispatcher/endpoints/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Endpoint not found: sql: no rows in result set` |
| `GET /api/api/rdl/templates/{templateID}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `template not found` |
| `GET /api/bp-notifications/logs/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"Log not found"}` |
| `GET /api/bp-notifications/templates/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"Template not found"}` |
| `GET /api/bundles/{bundleID}/` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `bundle with id 11111111-1111-1111-1111-111111111111 not found` |
| `GET /api/calculations/{id}/explain` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Calculation not found` |
| `GET /api/calculations/{name}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `missing destination name tier in *models.Calculation` |
| `GET /api/data-domains/{id}/` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `not found` |
| `GET /api/dax/functions/{functionName}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Function not found` |
| `GET /api/glossary/semantic-terms/{id}/cube-definition` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Semantic term not found` |
| `GET /api/integrations/marketplace/{integrationKey}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Integration not found` |
| `GET /api/intelligence/storage/plans/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `plan not found: 11111111-1111-1111-1111-111111111111` |
| `GET /api/invoices/{invoiceId}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"invoice 11111111-1111-1111-1111-111111111111 not found"}` |
| `GET /api/marketplace/items/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"Item not found","code":404,"error_code":"not_found","details` |
| `GET /api/migrations/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Migration not found` |
| `GET /api/rbac/roles/{roleId}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Role not found` |
| `GET /api/reports/schedules/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Schedule not found` |
| `GET /api/scheduler/governance/policies/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Not found` |
| `GET /api/security/mappings/{id}/` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `failed to get mapping: sql: no rows in result set` |
| `GET /api/security/profiles/{id}/` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `failed to get security profile: sql: no rows in result set` |
| `GET /api/templates/clones/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Clone not found` |
| `GET /api/tenants/{tenantId}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `Tenant not found` |
| `GET /api/v1/reports/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `report template not found: 11111111-1111-1111-1111-111111111111` |
| `GET /api/validation-rule-cores/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"Core rule not found","code":404,"error_code":"not_found","de` |
| `GET /reports/definitions/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"definition not found"}` |
| `GET /scheduler/dags/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"DAG not found"}` |
| `GET /scheduler/jobs/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"job not found"}` |
| `GET /scheduler/runs/dags/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"run not found"}` |
| `GET /scheduler/runs/jobs/{id}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `{"error":"run not found"}` |
| `GET /sources/preferences/{prefId}` | 404 | Route/resource not found for placeholder ID — retest with real ID before allowlisting | `sql: no rows in result set` |
| `GET /api/abbreviations/{id}/` | 400 | 400 | `Invalid ID` |
| `GET /api/api-dispatcher/audit` | 400 | 400 | `tenant_id is required` |
| `GET /api/api-dispatcher/fields` | 400 | 400 | `endpoint_id is required` |
| `GET /api/api-dispatcher/lineage` | 400 | 400 | `endpoint_id is required` |
| `GET /api/api/process-analytics/bottlenecks` | 400 | 400 | `{"error":"tenant_id required"}` |
| `GET /api/api/process-analytics/dashboard` | 400 | 400 | `{"error":"tenant_id required"}` |
| `GET /api/api/process-analytics/predict-duration` | 400 | 400 | `{"error":"tenant_id and workflow_type required"}` |
| `GET /api/api/process-analytics/recommendations` | 400 | 400 | `{"error":"tenant_id required"}` |
| `GET /api/api/process-analytics/step-performance` | 400 | 400 | `{"error":"tenant_id and workflow_type required"}` |
| `GET /api/api/process-benchmarking/industry` | 400 | 400 | `industry and process_type are required` |
| `GET /api/business-entities/{entityID}/related-objects` | 400 | 400 | `missing tenant context: tenant context not found: X-Tenant-ID and X-Te` |
| `GET /api/business-entities/{entityID}/semantic-assets` | 400 | 400 | `missing tenant context: tenant context not found: X-Tenant-ID and X-Te` |
| `GET /api/business-objects/` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{boId}/terms` | 400 | 400 | `{"details":"authentication required: missing or invalid JWT token","er` |
| `GET /api/business-objects/{id}` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/artifacts` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/data` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/delta` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/drift-sentinel` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/fields` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/multi-backend` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/publish-gate` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/relationships` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/scope` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/with_bindings` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/business-objects/{id}/workflow` | 400 | 400 | `authentication required: missing or invalid JWT token` |
| `GET /api/calc/` | 400 | 400 | `{"error":"object_id is required"}` |
| `GET /api/chart/{datasourceId}/{chartType}` | 400 | 400 | `{"success":false,"error":"invalid chart type: 11111111-1111-1111-1111-` |
| `GET /api/entity-schema/` | 400 | 400 | `Missing tenant_id` |
| `GET /api/explorer/search/suggestions` | 400 | 400 | `{"error":"user_id and datasource_id are required"}` |
| `GET /api/iceberg/lineage` | 400 | 400 | `missing table query parameter` |
| `GET /api/integrations/oauth/callback` | 400 | 400 | `Missing code or state` |
| `GET /api/lineage/dual` | 400 | 400 | `datasourceId or asset_id is required` |
| `GET /api/lineage/{datasourceId}/{lineageType}` | 301 | Redirect — inspect target | `<a href="/api/chart/11111111-1111-1111-1111-111111111111/11111111-1111` |
| `GET /api/lookups` | 400 | 400 | `tenant_id is required` |
| `GET /api/lookups/{id}/export` | 400 | 400 | `tenant_id is required` |
| `GET /api/lookups/{id}/values` | 400 | 400 | `tenant_id is required` |
| `GET /api/models/version` | 400 | 400 | `missing tenant context: tenant context not found: X-Tenant-ID and X-Te` |
| `GET /api/nba/stream` | 400 | 400 | `advisor_id required` |
| `GET /api/relationships/bo/{boId}` | 400 | 400 | `Missing datasource ID` |
| `GET /api/relationships/bo/{boId}/join-path/{toBoId}` | 400 | 400 | `Missing datasource ID` |
| `GET /api/relationships/physical/{tableId}` | 400 | 400 | `Missing datasource ID` |
| `GET /api/relationships/{entityID}/suggestions` | 400 | 400 | `missing tenant context: tenant context not found: X-Tenant-ID and X-Te` |
| `GET /api/rule-fabric/bo/{boKey}/policies/` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/categories` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/policies/` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/policies/{policyID}` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/rules/` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/rules/{ruleID}` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/rules/{ruleID}/versions` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/stats` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/violations/` | 400 | 400 | `tenant_id is required` |
| `GET /api/rule-fabric/violations/{violationID}` | 400 | 400 | `tenant_id is required` |
| `GET /api/scheduler/governance/audit` | 400 | 400 | `X-Tenant-ID header is required` |
| `GET /api/scheduler/governance/audit/stats` | 400 | 400 | `X-Tenant-ID header is required` |
| `GET /api/scheduler/governance/changesets` | 400 | 400 | `X-Tenant-ID header is required` |
| `GET /api/semantic-terms/explain` | 400 | 400 | `term_id is required` |
| `GET /api/semantic/tags/` | 400 | 400 | `{"error":"X-Tenant-ID header is required","code":400,"error_code":"mis` |
| `GET /api/slack/install` | 307 | Redirect — inspect target | `<a href="https://slack.com/oauth/v2/authorize?client_id=FAKE&amp;scope` |
| `GET /api/tenant-ops/connections/` | 400 | 400 | `{"error":"tenant_id is required","code":400,"error_code":"missing_tena` |
| `GET /api/tenant-ops/connections/{id}` | 400 | 400 | `{"error":"tenant_id is required","code":400,"error_code":"missing_tena` |
| `GET /api/tenant-ops/connections/{id}/datasources` | 400 | 400 | `{"error":"tenant_id is required","code":400,"error_code":"missing_tena` |
| `GET /api/v1/folders/{id}/diff` | 400 | 400 | `Invalid 'from' date format` |
| `GET /api/v1/triggers/events` | 400 | 400 | `tenant_id required` |
| `GET /api/workflow-timeout-triggers/` | 400 | 400 | `{"error":"unauthorized: missing claims"}` |
| `GET /api/workflow-timeout-triggers/{triggerId}/` | 400 | 400 | `{"error":"unauthorized: missing claims"}` |
| `GET /layouts` | 400 | 400 | `X-Tenant-ID and X-Tenant-Datasource-ID headers are required` |
| `GET /layouts/{id}` | 400 | 400 | `X-Tenant-ID and X-Tenant-Datasource-ID headers are required` |
| `GET /values/constraints` | 400 | 400 | `profile_id is required` |
| `GET /values/signals` | 400 | 400 | `issuer_id is required` |

## Not yet done

- **Non-`GET` route inventory (489 routes)** — by reading each handler's
  auth check, not by invocation, for the reason stated above.
- **Retest of the 404/400 bucket with real IDs** — several may resolve to
  real, unauthenticated data once given a valid identifier instead of a
  placeholder.
- **The 11 `000` entries** — re-probe individually to separate "hangs on
  an external dependency this box can't reach" from "actually broken."
- **This is a snapshot of `main` at the time of the probe** — re-run
  before the enforce step, not assumed still accurate.
