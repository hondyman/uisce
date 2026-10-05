# Cube impact analysis + gated cascade

| Field | Value |
|-------|--------|
| **Status** | **Approved** — implementation starts at A1 |
| **Author** | Architecture (post unify Cubes PR0–PR8) |
| **Date** | 2026-10-05 |
| **Related** | `feat/cube-tables-phase0`; `DetectCubeContractBreaking`; CUBE-0.2 PATCH/versions; PR7 subject pins; program plan Track A |
| **Decision type** | Product + architecture |

---

## Overview

Operators need to **see what a cube is made of** and **where it is used** before archive, delete, or breaking contract change — then **cascade** deliberately so pins, schedules, pipelines, and physical grains stay coherent.

v1 bar (locked):

1. **Assess** — `GET/POST` dry-run impact + Studio Impact panel (catalog + designer).
2. **Apply** — `POST …/cascade` only after confirm; **fail closed** when blocking consumers remain unresolved (unless a documented cascade mode rewrites/disables them).

Hard physical DROP of StarRocks/Iceberg tables is out of scope for v1 (reconciler quarantine is enough).

---

## Problem

| Gap | Today |
|-----|--------|
| Composition view | Designer tabs show draft fields; no single “bill of materials” API |
| Consumer inventory | None across saved queries, pages, schedules, pipelines, reports |
| Breaking change | PATCH → 409 + “use POST /versions”; versions bump without consumer rewrite |
| Archive | Sets `status=archived` / `archived_at` without checking consumers |
| Orphans | Saved-query / page pins and `cube_refresh` / `cube_materialize` can point at archived cubes |

---

## Impact graph

### Composition (inbound)

From `data_explorer.cube_definition` (+ metric registry + preagg catalog):

```ts
type CubeComposition = {
  cubeId: string;
  name: string;
  status: string;
  contractVersion: number;
  contentHash: string;
  isCore: boolean;
  boId: string;
  dimensions: { termNodeId: string }[];
  timeDimension?: { termNodeId: string; defaultGrain?: string } | null;
  metrics: { id: string; name?: string; status?: string }[]; // join metric_definition
  grains: string[][];
  federation: {
    sources: { boId: string; alias: string; bindingHint?: string }[];
    joins: { leftAlias: string; rightAlias: string; keyKind: string;
             leftTermIds: string[]; rightTermIds: string[]; transformTermId?: string }[];
    orphanRateMaxPercent?: number;
  };
  materialization: Record<string, unknown>;
  physical?: {
    grains: {
      grainHash: string;
      grain: string[];
      lifecycleStatus: string;
      icebergTable?: string;
      dualCommitWatermark?: string;
      attemptId?: string;
    }[];
  };
};
```

### Consumers (outbound)

| Kind | Discovery |
|------|-----------|
| `saved_query` | `data_explorer.saved_query` where `source_kind='cube' AND source_id=:cubeId` (also inspect `query_state` subject for defense in depth) |
| `page_tile` | `page_definitions` JSON: `props.subject.cubeId = :id` OR `savedQueryId` ∈ cube saved queries |
| `schedule` | schedules with kind `cube_refresh` and target/params cube id |
| `pipeline` | pipeline defs with node kind `cube_materialize` and `config.cube_id` |
| `report` | `report_definitions.definition` subject pin `cubeId` (legacy `dataBindings.primary.cube` name → informational via migrate util; not auto-rewritten in A) |
| `semantic` | optional `semantic_cube_id` / related (secondary, non-blocking unless proven live) |

```ts
type CubeConsumer = {
  kind: 'saved_query' | 'page_tile' | 'schedule' | 'pipeline' | 'report' | 'semantic';
  id: string;
  label: string;          // name / slug / report_key
  href?: string;          // FE deep link when known
  pin?: { contractVersion?: number | 'latest' };
  severity: 'info' | 'warning' | 'blocking';
  blocking: boolean;      // true ⇒ fail_closed without cascade mode that clears it
  detail?: string;
};
```

**Blocking defaults**

| Consumer | Archive | Breaking publish |
|----------|---------|------------------|
| Active saved_query | blocking | blocking if pin ≠ new version and mode ≠ rewire |
| Published page tile | blocking | same |
| Enabled `cube_refresh` schedule | blocking | warning (will refresh new contract if left enabled) |
| Pipeline `cube_materialize` | blocking | warning |
| Draft page / disabled schedule | warning | info |
| Legacy report name-only | warning (unresolved name) / info if subject pin matches | info |

Tenant visibility: same gold-core rules as `CubeHandler` list/get (tenant sees own + core).

---

## Change class

Reuse **`DetectCubeContractBreaking(published, draft)`** — do not add a parallel detector.

| Class | Detection | Default |
|-------|-----------|---------|
| `non_breaking_patch` | empty break reasons; content may still change | Allow PATCH; impact informational |
| `breaking_contract` | non-empty break reasons | Require `POST /versions` path today; cascade A5 rewrites pins / rematerializes |
| `archive` | `status=archived` | Block if any `blocking` consumer unless cascade mode clears them |
| `hard_delete` | not exposed in v1 | — |

---

## API

### `GET /api/cubes/{id}/impact`

Query: `includePhysical=1` (default true).

Response:

```json
{
  "cubeId": "…",
  "composition": { … },
  "consumers": [ … ],
  "summary": {
    "consumerCount": 3,
    "blockingCount": 1,
    "warningCount": 1,
    "physicalGrainCount": 2
  }
}
```

Dry-run. 404 if cube not visible to tenant.

### `POST /api/cubes/{id}/impact/preview`

Body:

```json
{
  "action": "archive" | "patch" | "publish_version",
  "patch": { … }   // optional; same shape as PATCH/versions draft fields
}
```

Response extends impact with:

```json
{
  "changeClass": "archive" | "non_breaking_patch" | "breaking_contract",
  "breakReasons": [],
  "blockingCount": 1,
  "confirmToken": "opaque-signed-short-ttl",
  "allowedModes": ["fail_closed", "disable_consumers", "rewire_latest", "bump_and_refresh"],
  "recommendedMode": "fail_closed"
}
```

`confirmToken` binds `{cubeId, action, contentHash|patchHash, exp}` (HMAC with server secret). Cascade rejects mismatched tokens.

### `POST /api/cubes/{id}/cascade`

Body:

```json
{
  "action": "archive" | "publish_version",
  "confirmToken": "…",
  "mode": "fail_closed" | "disable_consumers" | "rewire_latest" | "bump_and_refresh",
  "patch": { … }
}
```

| Mode | Behavior |
|------|----------|
| `fail_closed` | If any blocking consumer → **409** + impact body; no writes |
| `disable_consumers` | Disable/pause schedules & pipeline steps referencing cube; archive cube; leave SQ/page pins (runtime already fail-closed on archived / pin mismatch) |
| `rewire_latest` | For breaking publish: bump contract; rewrite saved-query + page mirrored subjects to new `contractVersion` (numeric); then refresh grains |
| `bump_and_refresh` | Publish version (existing versions semantics) + start materialize for all grains; do **not** rewrite pins (pins on old version keep working until operator rewires; document drift risk) |

**Archive + `fail_closed`:** 409 when blockingCount > 0.  
**Archive + `disable_consumers`:** disable consumers then archive.  
**v1 does not** auto-delete saved queries or pages.

Receipt response: `{ cube, changeClass, consumersAffected: […], materializeStarts?: […] }`.

---

## UI

### Catalog (`cubes-catalog`)

- Row action **Impact** → drawer: composition summary + consumer table + counts.
- Archive flow: open preview → choose mode → Confirm → cascade.

### Designer (`cube-designer`)

- **Impact** tab or header action (always available).
- On Save: if validate/`DetectCubeContractBreaking` non-empty → block silent PATCH; show Impact preview with publish_version modes.
- On Archive: same as catalog.

Implementation: prefer small DomainComponent `cubes.ImpactPanel` + studio op `cubes.impact` / `cubes.impactPreview` / `cubes.cascade` (matches FederationEditor pattern). Avoid full-page DC.

---

## Integrity invariants

1. Cascade never substitutes a **different** cube id for a pin (no silent remap to another cube). Rewire only updates `contractVersion` (and optional `'latest'` → numeric on publish policy).
2. Archived cubes remain readable for audit; list/default scopes exclude them (`archived_at IS NULL`) as today.
3. Physical grains: on archive, mark/quarantine via existing reconciler patterns; do not DROP tables in A.
4. Confirm token single-use or short TTL (≤10m).
5. Gold-core: only gold tenant archives/modifies `is_core` cubes (existing rule).

---

## PR plan (Track A)

| PR | Scope | Gate |
|----|--------|------|
| **A0** | This doc | Approved |
| **A1** | Inventory service + `GET …/impact` + Go tests | **Done** — unit + alpha smoke (CUBE-2.5) |
| **A2** | `POST …/impact/preview` + changeClass + confirmToken | Break vs non-break tests |
| **A3** | `cubes.ImpactPanel` + catalog/designer wiring (dry-run) | Browser proof |
| **A4** | `POST …/cascade` archive + `fail_closed` / `disable_consumers` | 409 vs clean archive test |
| **A5** | Cascade `publish_version` + `rewire_latest` / `bump_and_refresh` | Designer Confirm + rematerialize receipt |

---

## Follow-on tracks (not this doc)

- **B** Lakekeeper OAuth refresh (ops; user go before catalog DDL).
- **C** extract-N→staging Temporal activities.
- **D** Delete `components/pagestudio` per PR8 archive plan.

---

## Non-goals

- Hard delete API / DROP cold+hot tables in v1.
- Auto-rewrite all legacy `report_definitions` cube **names**.
- MCP impact tools (optional after A1).
- Second break-detection algorithm beside `DetectCubeContractBreaking`.
