# RLS / MCP production binding (staged MCP-first)

**Status:** R0+R1 merged (#416); **R2 CLOSED** (SET ROLE, #418); **R2b CLOSED 2026-10-06** (direct `uisce-app-dsn` + Infisical `UISCE_APP_DSN` durability). R3 parked.  
**Date:** 2026-10-06  
**Depends on:** Gold-aware FORCE RLS (`20261020_001`+), grants (`20261020_002`), `OpenMCPAppDB` (`9701efac4`).  
**Branch:** `feat/rls-mcp-production-binding` (merged).

## Claim boundary

**In scope to claim:** MCP Server DB pool runs as `uisce_mcp_app` with FORCE RLS effective, secrets in Infisical, dated triple receipt.

**Out of scope / not claimed:** Full fleet `DATABASE_URL` flip to app role. BeginTx inventory (2026-10-06): 93 BeginTx / 82 unfenced / **66 fence-needed**. Fleet flip waits on fence-needed→0 (R3 parked).

## Modes (honest)

| Mode | Login (`session_user`) | Effective (`current_user`) | How |
|------|------------------------|----------------------------|-----|
| Direct DSN | `uisce_mcp_app` | `uisce_mcp_app` | `UISCE_APP_DSN` TCP login |
| SET ROLE fallback | `postgres` (or parent) | `uisce_mcp_app` | `DATABASE_URL` + `AfterConnect SET ROLE` |

Both are valid MCP bindings **if** receipts distinguish them. Prefer direct DSN when pg_hba allows.

## Fail-loud (R1 deliverable)

If `DATABASE_URL`/`POSTGRES_DSN` is set and `OpenMCPAppDB` fails, the process **must not** attach MCP handlers to the shared HTTP/postgres pool.

- Log `[mcp-cutover] OpenMCPAppDB failed — MCP endpoints refuse shared-pool degrade`
- Register `/mcp` as **503** (or omit tool execution), never `mcp.NewServer(sqlxDB)` on failure
- Receipt A includes **process PID** and **startup timestamp** so a receipt cannot be paired with stale logs from a prior process

Emergency only: `UISCE_MCP_ALLOW_SHARED_POOL=1` re-enables legacy degrade (logged as WARNING); forbidden for claiming production binding.

## Pre-receipt role check

Before A–C, once against the MCP pool connection:

```sql
SELECT current_user AS effective_user,
       session_user AS login_user,
       r.rolbypassrls
FROM pg_roles r
WHERE r.rolname = current_user;
```

Require `rolbypassrls = false`. Confirm MCP pool is dedicated (SET ROLE only on MCP pool connections opened by `OpenMCPAppDB`, never on the shared HTTP pool).

## Triple receipt (amended)

Capture with redacted secrets. Include PID + startup time from receipt A on every artifact.

### A — Startup mode

Log line: `[mcp-cutover] MCP DB pool mode=<uisce-app-dsn|set-role:uisce_mcp_app> pid=<N> started_at=<RFC3339>`

### B — Role identity (mode-sensitive)

MCP-side probe (startup or one-shot) records **both**:

- Direct DSN: `session_user=uisce_mcp_app`, `current_user=uisce_mcp_app`
- SET ROLE: `session_user=<parent>`, `current_user=uisce_mcp_app`

**Do not** use `pg_stat_activity.usename` alone as proof under SET ROLE — that column is the **login** role and will show `postgres`.

### C — IDOR with positive control

Same MCP path, same process:

1. **Positive:** tenant A JWT **can** read tenant A page/BO (or MCP tool equivalent).
2. **Negative:** tenant A JWT **cannot** read tenant B page/BO (not found / empty / deny).

Negative-only proof is insufficient (deny-all or broken pool would pass).

## Flip checklist (ordered)

1. Infisical: store `UISCE_APP_DSN` (and optional `UISCE_MCP_DB_ROLE`) in project `uisce` / env `dev` — **operator token required**.
2. `scripts/infisical-bootstrap.sh` / `START_BACKEND.sh` export the key when present.
3. Pre-receipt role check (`rolbypassrls=false`).
4. Restart backend; capture A (mode + pid + started_at).
5. Capture B (session_user + current_user).
6. Capture C (positive + negative).
7. Update `topics/uisce-rls.md` + observation; claim MCP-only binding with mode named.

## PR plan

| PR | Scope |
|----|--------|
| **R0+R1** | This design + fail-loud + probe log + Infisical bootstrap key + tripwire |
| **R2** | Ops restart + triple receipt (after Infisical gate) |
| **R3** | BeginTx fence waves (parked; prefer R3b BO HTTP writers later) |

## Secrets hygiene

Never echo DSN passwords, tokens, or full connection strings in chat, commits, or PR bodies. Receipts name paths and key names only.

## R2 receipt (SET ROLE, 2026-10-06)

**Claimed:** MCP Server pool effective identity `uisce_mcp_app` with FORCE RLS (`rolbypassrls=false`) under **SET ROLE** fallback.

**Not claimed:** `UISCE_APP_DSN` direct login; full-fleet `DATABASE_URL` flip; R3 BeginTx fencing.

| Receipt | Evidence |
|---------|----------|
| A | `[mcp-cutover] MCP DB pool mode=set-role:uisce_mcp_app pid=62768 started_at=2026-10-06T02:37:59.417759Z …` |
| B | `session_user=postgres` `current_user=uisce_mcp_app` `rolbypassrls=false` (do not use `pg_stat_activity.usename` alone under SET ROLE) |
| C+ | tenant `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa` JWT: `list_pages` includes A Page `eb23fc7a-66ef-4da5-8da4-1ea66570b530`; `get_page` → `found:true` |
| C− | same JWT: `list_pages` excludes B Page `0f3af13f-8ad0-4538-af25-fa38536bed15`; `get_page` → `found:false` |

Endpoint: `POST /api/mcp` (streamable, stateless). Local ops log: `/tmp/uisce-server-r2.log`. Redacted capture: `/tmp/r2-triple-receipt.md`.

## R2b receipt (direct DSN + Infisical, 2026-10-06)

**Claimed:** MCP pool TCP login `mode=uisce-app-dsn` with `session_user=current_user=uisce_mcp_app` and `rolbypassrls=false`, IDOR C+/C− on `/api/mcp`, and `UISCE_APP_DSN` durable in Infisical `uisce`/`dev`.

**Ops applied (alpha):** `GRANT ivy_tenant_apps TO uisce_mcp_app`; `GRANT CONNECT ON DATABASE alpha TO uisce_mcp_app`; password rotated (value not in git/chat).

**Infisical notes:** initial read-only token got 403; RW token stored the key. CLI `SECRET=@/path` stored the path literal — correct upsert used `infisical secrets set --file` (.env). Restart pulled value_len=165 matching the proven DSN.

| Receipt | Evidence |
|---------|----------|
| A (local export) | `mode=uisce-app-dsn pid=65410 started_at=2026-10-06T03:12:12.228606Z` |
| A/B (Infisical pull restart) | `mode=uisce-app-dsn pid=66086 started_at=2026-10-06T03:20:21.221815Z session_user=uisce_mcp_app current_user=uisce_mcp_app rolbypassrls=false` |
| C+ | tenant A JWT: `list_pages` includes A Page; `get_page` → `found:true` |
| C− | same JWT: B Page absent; `get_page` → `found:false` |

Local ops log: `/tmp/uisce-server-r2b.log`. Redacted captures: `/tmp/r2b-triple-receipt.md`, `/tmp/r2b-infisical-durability-receipt.md`.

