# Uisce Backend Package Analysis Report
## `backend/internal/` — 123 packages

**Module:** `github.com/hondyman/uisce/backend`  
**Total:** 123 top-level packages, 1,590 .go files, ~440,075 LOC  
**Reference:** `/Users/eganpj/GitHub/uisce/backend/internal/`

> **NOTE:** There is a separate snapshot/copy at `uisce_frontend/backend/internal/` (190 packages) — this analysis is based on the main `backend/internal/` only.

---

## 1. EXECUTIVE SUMMARY

The `backend/internal/` directory contains **123 top-level packages** with **1,590 .go files** (~440K LOC). The architecture is heavily hub-and-spoke:

- **5 HUB packages** (≥30 importers): `logging`, `models`, `services`, `analytics`, `security`
- **18 mid-hubs** (5-29 importers)
- **33 leaf packages** (0-1 importers) — many extraction candidates
- **2 cycles** in the dep graph: `wealth ↔ calcengine`, `api ↔ reports`
- **`internal/logging` is byte-identical to `libs/logging`** (md5 confirmed) — extraction is half done
- **`backend/internal/calc-engine/` is a thin integration shim** for the already-extracted `/calc-engine/` Go module (separate `go.mod` at repo root)

### Top 10 Hubs
| # | Package | Internal Imp | External Imp | Total | Files | LOC | Verdict |
|---|---------|--------------|--------------|-------|-------|-----|---------|
| 1 | `logging` | 98 | 4 | **102** | 1 | 42 | EXTRACT (duplicate) |
| 2 | `models` | 94 | 3 | 97 | 25 | 3,080 | Keep in monolith |
| 3 | `services` | 73 | 16 | 89 | **145** | 45,002 | Keep — needs decomposition |
| 4 | `analytics` | 40 | 4 | 44 | 70 | 29,829 | Keep |
| 5 | `security` | 35 | 6 | 41 | 24 | 4,857 | Keep — but check overlap with libs/auth, libs/abac-client |
| 6 | `audit` | 27 | 4 | 31 | 32 | 9,738 | Keep |
| 7 | `auth` | 22 | 0 | 22 | 1 | 73 | **EXTRACT** after refactor |
| 8 | `cube` | 18 | 4 | 22 | 14 | 10,174 | Keep |
| 9 | `metadata` | 17 | 4 | 21 | 24 | — | Keep |
| 10 | `events` | 17 | 3 | 20 | 12 | — | Keep |


---

## 2. ALL 123 PACKAGES (sorted by total importers, descending)

| # | Package | Files | Subs | Int Imp | Ext Imp | Out Deps | Hub/Leaf |
|---|---------|-------|------|---------|---------|----------|----------|
| 1 | `logging` | 1 | 0 | 98 | 4 | — | HUB |
| 2 | `models` | 25 | 0 | 94 | 3 | goldcopy | HUB |
| 3 | `services` | 145 | 1 | 73 | 16 | analytics, audit, bo, cube, cubeengine (+20) | HUB |
| 4 | `analytics` | 70 | 2 | 40 | 4 | audit, boresolver, calculation, cbo, cube (+10) | HUB |
| 5 | `security` | 24 | 1 | 35 | 6 | events, models | HUB |
| 6 | `audit` | 32 | 0 | 27 | 4 | auth, catalog, datafusion, events, logging (+3) | HUB |
| 7 | `auth` | 1 | 0 | 22 | 0 | models | Mid-hub |
| 8 | `cube` | 14 | 1 | 18 | 4 | — | Mid-hub |
| 9 | `metadata` | 24 | 0 | 17 | 4 | cube, db, events, lineage, logging (+5) | Mid-hub |
| 10 | `events` | 12 | 0 | 17 | 3 | — | Mid-hub |
| 11 | `api` | 270 | 2 | 3 | 12 | access, ai, altinv, altinvest, analytics (+64) | Mid-hub |
| 12 | `db` | 8 | 1 | 14 | 0 | logging | Mid-hub |
| 13 | `lineage` | 3 | 0 | 10 | 4 | — | Mid-hub |
| 14 | `temporal` | 18 | 3 | 12 | 1 | aso, audit, events, factor, metadata (+1) | Mid-hub |
| 15 | `wealth` | 33 | 7 | 7 | 6 | calcengine, logging, metadata, models, platform | Mid-hub |
| 16 | `identity` | 1 | 0 | 11 | 1 | — | Mid-hub |
| 17 | `rules` | 26 | 0 | 10 | 2 | models, starlib | Mid-hub |
| 18 | `handlers` | 152 | 1 | 7 | 4 | ai, analytics, apistudio, audit, auth (+37) | Mid-hub |
| 19 | `platform` | 2 | 0 | 10 | 1 | models | Mid-hub |
| 20 | `query` | 18 | 0 | 7 | 4 | domain | Mid-hub |
| 21 | `domain` | 15 | 0 | 9 | 1 | calcengine | Mid-hub |
| 22 | `observability` | 17 | 2 | 7 | 3 | cbo, logging | Mid-hub |
| 23 | `semantic` | 2 | 0 | 9 | 1 | — | Mid-hub |
| 24 | `boresolver` | 26 | 0 | 9 | 0 | bo | Mid |
| 25 | `cbo` | 12 | 0 | 8 | 1 | audit | Mid |
| 26 | `goldcopy` | 6 | 0 | 9 | 0 | — | Mid |
| 27 | `pagestudio` | 9 | 0 | 7 | 2 | analytics, apistudio, semantic | Mid |
| 28 | `repository` | 2 | 0 | 7 | 0 | graphql, models | Mid |
| 29 | `scheduler_intelligence` | 15 | 3 | 7 | 0 | analytics, calendar, compliance, events, logging | Mid |
| 30 | `validation` | 9 | 0 | 6 | 1 | rules | Mid |
| 31 | `workflows` | 28 | 3 | 2 | 5 | activities, audit, cbo, config, events (+9) | Mid |
| 32 | `aso` | 19 | 0 | 6 | 0 | — | Mid |
| 33 | `catalog` | 5 | 0 | 5 | 1 | — | Mid |
| 34 | `dynamic` | 2 | 0 | 3 | 3 | cube, query | Mid |
| 35 | `ml` | 5 | 6 | 6 | 0 | — | Mid |
| 36 | `tenant` | 3 | 0 | 5 | 1 | audit, config | Mid |
| 37 | `values` | 1 | 0 | 6 | 0 | — | Mid |
| 38 | `viewmodel` | 1 | 0 | 6 | 0 | — | Mid |
| 39 | `graphql` | 21 | 1 | 5 | 0 | audit, catalog | Mid |
| 40 | `store` | 3 | 0 | 4 | 1 | models, policy, simulation | Mid |
| 41 | `webhooks` | 1 | 0 | 5 | 0 | — | Mid |
| 42 | `billing` | 7 | 0 | 4 | 0 | — | Mid-leaf |
| 43 | `config` | 4 | 0 | 0 | 4 | — | Mid-leaf |
| 44 | `mdm` | 8 | 0 | 4 | 0 | analytics, goldcopy, models, portfoliomaster | Mid-leaf |
| 45 | `oauth` | 1 | 0 | 4 | 0 | security | Mid-leaf |
| 46 | `policy` | 2 | 0 | 3 | 1 | — | Mid-leaf |
| 47 | `profiler` | 2 | 1 | 3 | 1 | — | Mid-leaf |
| 48 | `rag` | 12 | 3 | 3 | 1 | config, tenant | Mid-leaf |
| 49 | `region` | 3 | 0 | 4 | 0 | — | Mid-leaf |
| 50 | `scanner` | 3 | 0 | 4 | 0 | cube, db, logging | Mid-leaf |
| 51 | `simulation` | 8 | 0 | 3 | 1 | policy, services | Mid-leaf |
| 52 | `sync` | 14 | 0 | 3 | 1 | audit, google, models, oauth, repository (+1) | Mid-leaf |
| 53 | `activities` | 2 | 0 | 2 | 1 | logging, security | Mid-leaf |
| 54 | `apistudio` | 13 | 0 | 3 | 0 | analytics, logging, region, semantic | Mid-leaf |
| 55 | `bp` | 11 | 2 | 2 | 1 | rules | Mid-leaf |
| 56 | `calcengine` | 14 | 1 | 3 | 0 | pricing, wealth | Mid-leaf |
| 57 | `compliance` | 4 | 4 | 2 | 1 | pagestudio | Mid-leaf |
| 58 | `household` | 2 | 0 | 3 | 0 | — | Mid-leaf |
| 59 | `middleware` | 13 | 0 | 3 | 0 | auth, identity, logging, models, requestcontext (+3) | Mid-leaf |
| 60 | `nba` | 7 | 1 | 1 | 2 | temporal | Mid-leaf |
| 61 | `notifications` | 5 | 0 | 3 | 0 | — | Mid-leaf |
| 62 | `optimizer` | 2 | 0 | 3 | 0 | — | Mid-leaf |
| 63 | `portfoliomaster` | 3 | 0 | 3 | 0 | — | Mid-leaf |
| 64 | `pricing` | 4 | 0 | 3 | 0 | — | Mid-leaf |
| 65 | `queue` | 1 | 0 | 3 | 0 | — | Mid-leaf |
| 66 | `reports` | 8 | 0 | 3 | 0 | api | Mid-leaf |
| 67 | `requestcontext` | 1 | 0 | 3 | 0 | — | Mid-leaf |
| 68 | `starlib` | 11 | 0 | 3 | 0 | — | Mid-leaf |
| 69 | `succession` | 2 | 0 | 3 | 0 | — | Mid-leaf |
| 70 | `trino` | 1 | 0 | 2 | 1 | — | Mid-leaf |
| 71 | `wasm` | 2 | 0 | 2 | 1 | audit, validation | Mid-leaf |
| 72 | `access` | 1 | 0 | 2 | 0 | — | Mid-leaf |
| 73 | `ai` | 14 | 2 | 2 | 0 | audit, catalog, indexing, rules, validation (+1) | Mid-leaf |
| 74 | `altinvest` | 2 | 0 | 2 | 0 | — | Mid-leaf |
| 75 | `bo` | 1 | 0 | 2 | 0 | — | Mid-leaf |
| 76 | `cache` | 5 | 0 | 0 | 2 | — | Mid-leaf |
| 77 | `calendar` | 3 | 0 | 1 | 1 | — | Mid-leaf |
| 78 | `cash` | 3 | 0 | 1 | 1 | goldcopy | Mid-leaf |
| 79 | `cubeengine` | 1 | 0 | 2 | 0 | cube | Mid-leaf |
| 80 | `google` | 1 | 0 | 2 | 0 | oauth | Mid-leaf |
| 81 | `guardrails` | 4 | 0 | 2 | 0 | audit | Mid-leaf |
| 82 | `indexing` | 3 | 0 | 2 | 0 | values | Mid-leaf |
| 83 | `infrastructure` | 1 | 0 | 2 | 0 | — | Mid-leaf |
| 84 | `jobs` | 4 | 0 | 2 | 0 | notifications, queue, services | Mid-leaf |
| 85 | `ops` | 36 | 0 | 1 | 1 | — | Mid-leaf |
| 86 | `preference` | 3 | 0 | 2 | 0 | — | Mid-leaf |
| 87 | `review` | 4 | 0 | 1 | 1 | apistudio, lineage, semantic | Mid-leaf |
| 88 | `risk` | 3 | 0 | 1 | 1 | — | Mid-leaf |
| 89 | `transaction` | 4 | 0 | 1 | 1 | goldcopy | Mid-leaf |
| 90 | `views` | 1 | 0 | 2 | 0 | cube | Mid-leaf |
| 91 | `altinv` | 9 | 0 | 1 | 0 | — | Leaf |
| 92 | `bundles` | 8 | 0 | 0 | 1 | logging | Leaf |
| 93 | `business_process` | 9 | 0 | 1 | 0 | — | Leaf |
| 94 | `calculation` | 1 | 0 | 1 | 0 | — | Leaf |
| 95 | `catalogsync` | 7 | 0 | 0 | 1 | catalog | Leaf |
| 96 | `crypto` | 1 | 0 | 1 | 0 | — | Leaf |
| 97 | `datafusion` | 1 | 0 | 1 | 0 | — | Leaf |
| 98 | `delegation` | 1 | 0 | 1 | 0 | — | Leaf |
| 99 | `factor` | 1 | 0 | 1 | 0 | — | Leaf |
| 100 | `feebilling` | 2 | 0 | 1 | 0 | — | Leaf |
| 101 | `financial` | 6 | 0 | 1 | 0 | — | Leaf |
| 102 | `forecasting` | 1 | 0 | 0 | 1 | policy, simulation | Leaf |
| 103 | `help` | 1 | 0 | 1 | 0 | — | Leaf |
| 104 | `ledger` | 1 | 0 | 1 | 0 | — | Leaf |
| 105 | `marketdata` | 2 | 0 | 0 | 1 | — | Leaf |
| 106 | `metrics` | 3 | 0 | 1 | 0 | — | Leaf |
| 107 | `nl_intelligence` | 8 | 0 | 1 | 0 | — | Leaf |
| 108 | `offboarding` | 1 | 0 | 1 | 0 | — | Leaf |
| 109 | `planner` | 8 | 0 | 1 | 0 | domain | Leaf |
| 110 | `position` | 4 | 0 | 1 | 0 | goldcopy | Leaf |
| 111 | `querybuilder` | 3 | 0 | 1 | 0 | boresolver, handlers, logging, security | Leaf |
| 112 | `rdl` | 3 | 0 | 1 | 0 | — | Leaf |
| 113 | `reporting` | 16 | 0 | 1 | 0 | services | Leaf |
| 114 | `scheduler` | 4 | 0 | 0 | 1 | audit, services | Leaf |
| 115 | `semanticviews` | 1 | 0 | 1 | 0 | — | Leaf |
| 116 | `taxplan` | 2 | 0 | 1 | 0 | — | Leaf |
| 117 | `tenantauto` | 2 | 0 | 0 | 1 | — | Leaf |
| 118 | `tests` | 1 | 0 | 0 | 1 | analytics, cbo, semantic | Leaf |
| 119 | `triggers` | 1 | 0 | 0 | 1 | workflows | Leaf |
| 120 | `viewgen` | 4 | 0 | 1 | 0 | cube, viewmodel | Leaf |
| 121 | `viewmerge` | 1 | 0 | 1 | 0 | viewmodel | Leaf |
| 122 | `workers` | 3 | 0 | 0 | 1 | activities, events, logging, security | Leaf |
| 123 | `calc-engine` | 0 | 6 | 0 | 0 | audit, logging, temporal | Leaf |

---

## 3. EXTRACTION CANDIDATES (Prioritized)

### Tier 1: TRIVIAL wins (no refactoring needed)

| Package | Files | Int Imp | Ext Imp | Deps | Action |
|---------|-------|---------|---------|------|--------|
| `logging` | 1 | 98 | 4 | none | **Already duplicate of `libs/logging`** — just swap imports in 102 files & delete |

**Effort: 1 day, Risk: ZERO** (md5 match confirmed: `673d33ed0171c108726baf6cc26df947`)

### Tier 2: EASY extractions (small refactor to break 1 dep)

| Package | Files | Int Imp | Ext Imp | Deps | Action |
|---------|-------|---------|---------|------|--------|
| `auth` | 1 | 22 | 0 | `models` | Extract after decoupling from `models.User` (or accept the dep) |
| `help` | 1 | 1 | 0 | none | Move to `libs/help` |
| `identity` | 1 | 11 | 1 | none | Move to `libs/identity` (merge with `requestcontext`) |
| `requestcontext` | 1 | 3 | 0 | none | Merge into `libs/identity` |
| `infrastructure` | 1 | 2 | 0 | none | Move to `libs/infrastructure` (verify generic) |
| `access` | 1 | 2 | 0 | none | Move to `libs/access` (small) |
| `metric`/`metrics` | 3 | 1 | 0 | none | Move to `libs/metrics` |

**Effort: 1-2 days each, Risk: LOW**

### Tier 3: MODERATE extractions (need refactoring)

| Package | Files | Int Imp | Deps | Action |
|---------|-------|---------|------|--------|
| `validation` | 9 | 6 | `rules` | Refactor: extract shared types from `rules` first; then move to `libs/validation` |
| `oauth` | 1 | 4 | `security` | Extract after security decoupling |
| `google` | 1 | 2 | `oauth` | Already OAuth-specific; part of `oauth` extraction |
| `nl_intelligence` | 8 | 1 | none | Move to `libs/nl-intelligence` — genuine generic DB-dialect engine |
| `guardrails` | 4 | 2 | `audit` | Could go to `libs/ai-sdk` (guardrails) |

**Effort: 1-2 weeks each, Risk: MEDIUM**

### Tier 4: HARD extractions (significant refactoring)

| Package | Files | Int Imp | Deps | Action |
|---------|-------|---------|------|--------|
| `services` | 145 | 73 | 24 | Decompose into ~10 focused service modules; cannot extract as-is |
| `analytics` | 70 | 40 | 14 | Keep in monolith; only extract sub-clusters (cube, cbo) |
| `events` | 12 | 17 | none | Could become `libs/events`, but uses 5+ types from `models` |

**Effort: 1-2 months each, Risk: HIGH**


---

## 4. DOMAIN CLUSTERS

### 4.1 Trading & Wealth Management
- **Packages (18):** `wealth`, `portfoliomaster`, `position`, `cash`, `transaction`, `pricing`, `household`, `succession`, `offboarding`, `taxplan`, `financial`, `planner`, `preference`, `delegation`, `optimizer`, `factor`, `altinv`, `altinvest`
- **Hub:** `wealth` (13 importers, 7 subpackages) — 33 files, includes the TradingWorkflowActivities
- **External connections:** Analytics (goldcopy, models), Calculation (calcengine), Observability (logging)
- **CYCLE WARNING:** `wealth ↔ calcengine` (mutual)
- **Verdict:** **KEEP in monolith** — tightly coupled to goldcopy (semantic layer) and wealth domain models. Consider this as the future `libs/wealth` extraction target, but only AFTER analytics/semantic layer stabilizes.

### 4.2 Analytics & Semantic Layer (largest cluster)
- **Packages (22):** `analytics`, `cube`, `cubeengine`, `cbo`, `boresolver`, `scanner`, `calculation`, `lineage`, `metadata`, `viewmodel`, `viewgen`, `viewmerge`, `views`, `semantic`, `semanticviews`, `values`, `indexing`, `bo`, `catalog`, `goldcopy`, `mdm`, `datafusion`
- **Hubs:** `analytics` (44 imp), `metadata` (21), `cube` (22)
- **Internal edges:** 8
- **External edges:** 32 (mostly to observability, security, events)
- **Verdict:** **KEEP in monolith** — this is the semantic/analytics core. `cube` and `cbo` are potential future extraction targets. `cube` already has 1 subpackage and is largely self-contained.

### 4.3 Security & Access Control
- **Packages (9):** `security`, `auth`, `oauth`, `policy`, `google`, `identity`, `guardrails`, `access`, `platform`
- **Hub:** `security` (41 imp), `auth` (22)
- **Internal edges:** 2
- **External edges:** 5
- **Cohesion:** MEDIUM (0.29) — auth→models, oauth→security, google→oauth, guardrails→audit
- **Existing libs overlap:** `libs/auth/` (JWT), `libs/jwt-middleware/`, `libs/abac-client/`
- **Verdict:** **PARTIAL EXTRACT recommended** — extract `auth` (context utils) and `identity` to `libs/`. Keep `security` and `platform` in monolith.

### 4.4 Workflow & Temporal Orchestration
- **Packages (11):** `temporal`, `workflows`, `bp`, `business_process`, `triggers`, `scheduler`, `scheduler_intelligence`, `nba`, `activities`, `workers`, `jobs`
- **Hub:** `temporal` (13 imp), `workflows` (7)
- **Existing libs overlap:** `libs/temporal-client/`
- **Verdict:** **KEEP in monolith** — too tightly coupled to analytics, events, audit. `business_process` and `triggers` look extractable but are deeply embedded.

### 4.5 Events & Messaging
- **Packages (4):** `events`, `queue`, `notifications`, `webhooks`
- **Hub:** `events` (20 imp)
- **Verdict:** `events` is a mid-hub (12 files). The other 3 are small leaves. `events` should stay because it provides the standard event bus used everywhere. `queue`, `notifications`, `webhooks` could be extracted IF a generic abstraction is created.

### 4.6 Calculation Engine
- **Packages (4):** `calcengine`, `calc-engine`, `simulation`, `forecasting`
- **Existing extraction:** `/calc-engine/` is ALREADY a separate Go module at repo root
- **`backend/internal/calc-engine/`** has 0 top-level importers, 6 subpackages (activities, workflows, trino, datafusion, internal, worker)
- **Verdict:** **Already partially done.** `backend/internal/calc-engine/` is a thin integration shim. `calcengine` (in monolith) is a different package — it has 3 importers (wealth, domain, pricing).

### 4.7 AI/ML & Intelligence
- **Packages (6):** `ai`, `ml`, `rag`, `nl_intelligence`, `starlib`, `nba`
- **Existing libs overlap:** `libs/ai-sdk/`
- **Verdict:** **KEEP, with FUTURE EXTRACTION** — `ai`, `ml`, `nba` are domain-specific. `nl_intelligence` is genuinely reusable (dialect engine). `rag` is mid-sized and could go to `libs/rag`.

### 4.8 API & HTTP Layer
- **Packages (9):** `api` (270 files), `handlers` (152), `graphql`, `middleware`, `requestcontext`, `pagestudio`, `apistudio`, `dynamic`, `region`
- **Hub:** `api` is a "reverse-hub" — heavily imports, lightly imported (15 total)
- **CYCLE:** `api ↔ reports` (mutual)
- **Verdict:** **KEEP in monolith** — this is THE API surface. Cannot be extracted. `api` + `handlers` alone = 422 files. This is the monolith.

### 4.9 Observability
- **Packages (4):** `logging` (1 file, 102 imp), `audit` (32 files, 31 imp), `metrics` (3 files, 1 imp), `observability` (17 files, 10 imp)
- **`logging` is a duplicate of `libs/logging`**
- **Verdict:** **EXTRACT `logging` immediately** (Tier 1). Keep `audit` in monolith (it's domain-specific). `observability` could partially extract. `metrics` is generic.

### 4.10 Data & Storage
- **Packages (5):** `db`, `query`, `querybuilder`, `repository`, `trino`
- **Hub:** `db` (14), `query` (11)
- **Verdict:** **KEEP** — DB infrastructure is monolith-specific.

### 4.11 Config & Multi-tenancy
- **Packages (5):** `config`, `tenant`, `tenantauto`, `infrastructure`, `ops`
- **Verdict:** `config` is interesting — 0 internal importers but 4 external (from cmd scripts). The `config` package is itself a giant `Config` struct (~hundreds of lines), used by main.go-like scripts. Could be a lib candidate if made generic.

### 4.12 Rules Engine (Hidden Hub)
- **Packages (3):** `rules` (26 files, 12 imp), `validation` (9, 7), `domain` (15, 10)
- **Verdict:** This is a SUBSYSTEM, not in `services` but deserves its own cluster. It implements the rule/policy engine. `validation` is the best extraction candidate (depends only on `rules`).

### 4.13 Reporting
- **Packages (3):** `reporting` (16, 1), `reports` (8, 3), `rdl` (3, 1)
- **CYCLE:** `api ↔ reports` (mutual dep)
- **Verdict:** **KEEP** — has a circular dep with `api` that must be broken before any extraction.


---

## 5. PACKAGES TO STAY IN MONOLITH

These packages should NOT be extracted because they're too interconnected:

| Package | Why it stays | Risk if extracted |
|---------|--------------|-------------------|
| `api` (270 files) | Imports 74+ packages, central API surface | CRITICAL — touches everything |
| `handlers` (152 files) | API handlers for 40+ domains | CRITICAL — touches everything |
| `services` (145 files) | 73 importers, 24 deps — kitchen sink | HIGH — 89 importers break |
| `analytics` (70 files) | Semantic layer core, 40 importers | HIGH — would force extract of all consumers |
| `models` (25 files) | Central schema, 94 importers | HIGH — almost every package depends on these types |
| `audit` (32 files) | 27 internal importers | MEDIUM — domain-specific (ImpersonationSession etc.) |
| `security` (24 files) | 35 importers, depends on models+events | MEDIUM |
| `metadata` (24 files) | 17 importers, deeply embedded in analytics | MEDIUM |
| `temporal` (18 files) | 13 importers, workflow core | MEDIUM |
| `events` (12 files) | 17 importers, base event bus | MEDIUM |
| `wealth` (33 files) | 7 importers, 7 subpackages, wealth domain | MEDIUM — has cycle with calcengine |
| `workflows` (28 files) | 7 importers, complex | MEDIUM |
| `rules` (26 files) | 10 importers, rule engine | MEDIUM |
| `domain` (15 files) | 9 importers, advanced policy engine | MEDIUM |
| `boresolver` (26 files) | 9 importers, BO resolver | MEDIUM |
| `cube` (14 files) | 18 importers, Cube.js abstraction | MEDIUM |

---

## 6. LIKELY DEAD / UNDERUSED CODE

These packages have 0-1 importers and warrant investigation:

| Package | Files | Int Imp | Ext Imp | Likely Verdict |
|---------|-------|---------|---------|----------------|
| `altinv` | 9 | 1 | 0 | Used by `api` only — domain-specific wealth code, KEEP |
| `bundles` | 8 | 0 | 1 | Only used by `backend/cmd/bundles/main.go` — could be moved to scripts/ or deleted |
| `business_process` | 9 | 1 | 0 | Used by `api` only — KEEP (workflow core) |
| `cache` | 5 | 0 | 2 | Used only by `backend/pkg/meta/` — could be moved to `pkg/` |
| `calc-engine` | 0 | 0 | 0 | NOT DEAD — 6 subpackages, used by `api/calc-engine_handlers.go` and `calcengine/multi_source_engine.go` |
| `calculation` | 1 | 1 | 0 | Used by `analytics` only — KEEP (BO field) |
| `catalogsync` | 7 | 0 | 1 | Only used by `backend/cmd/catalog-sync/main.go` — script-only |
| `crypto` | 1 | 1 | 0 | Used by `api` only — KEEP (crypto holding model) |
| `datafusion` | 1 | 1 | 0 | Used by `audit` only — DB-specific client |
| `delegation` | 1 | 1 | 0 | Used by `api` only — KEEP (domain model) |
| `factor` | 1 | 1 | 0 | Used by `temporal` only — KEEP (factor universe) |
| `feebilling` | 2 | 1 | 0 | Used by `api` only — domain-specific, KEEP |
| `financial` | 6 | 1 | 0 | Used by `services` only — domain-specific, KEEP |
| `forecasting` | 1 | 0 | 1 | Only used by `backend/cmd/simulate/main.go` — could be removed |
| `help` | 1 | 1 | 0 | **EXTRACT to libs/help** |
| `household` | 2 | 3 | 0 | Wealth domain — KEEP |
| `ledger` | 1 | 1 | 0 | Used by `analytics` only — KEEP (trade event model) |
| `marketdata` | 2 | 0 | 1 | Only used by `backend/cmd/shadow_mode/main.go` — script-only |
| `metrics` | 3 | 1 | 0 | **EXTRACT to libs/metrics** |
| `nl_intelligence` | 8 | 1 | 0 | Used by `handlers` only — **EXTRACT to libs/nl-intelligence** (genuinely reusable) |
| `offboarding` | 1 | 1 | 0 | Used by `api` only — KEEP (wealth flow) |
| `planner` | 8 | 1 | 0 | Used by `api` only — KEEP (semantic planner) |
| `position` | 4 | 1 | 0 | Used by `handlers` only — KEEP (wealth domain) |
| `querybuilder` | 3 | 1 | 0 | Used by `api` only — KEEP |
| `rdl` | 3 | 1 | 0 | Used by `api` only — KEEP (reporting) |
| `reporting` | 16 | 1 | 0 | Used by `api` only — KEEP |
| `scheduler` | 4 | 0 | 1 | Only used by `backend/cmd/semantic-rules-api/main.go` — script-only |
| `semanticviews` | 1 | 1 | 0 | Used by `api` only — KEEP |
| `starlib` | 11 | 3 | 0 | Used by `rules` and `services` only — domain rule lib, KEEP |
| `taxplan` | 2 | 1 | 0 | Used by `api` only — KEEP (wealth) |
| `tenantauto` | 2 | 0 | 1 | Only used by `backend/cmd/tenant_automation/main.go` — script-only |
| `tests` | 1 | 0 | 1 | Only used by `backend/cmd/worker/main.go` — script-only |
| `triggers` | 1 | 0 | 1 | Only used by `backend/cmd/triggers/main.go` — script-only |
| `viewgen` | 4 | 1 | 0 | Used by `analytics` only — KEEP |
| `viewmerge` | 1 | 1 | 0 | Used by `analytics` only — KEEP |
| `workers` | 3 | 0 | 1 | Only used by `backend/cmd/security-event-worker/main.go` — script-only |

**Summary:** Of 33 leaf packages, only ~5 are clearly extractable. The rest are either:
1. Domain-specific (wealth/trading/etc.) — should stay
2. Single-cmd-script utility — should move to scripts/ or be deleted
3. Trivial type definitions used by one or two packages


---

## 7. EFFORT & SEQUENCING RECOMMENDATION

### Recommended sequence (in order of execution):

| Phase | Task | Effort | Risk | Why this order |
|-------|------|--------|------|----------------|
| **1a** | Remove `internal/logging` (use `libs/logging`) | 1 day | NONE | Lowest risk, highest value (102 importers) |
| **1b** | Delete cmd-script-only packages (`bundles`, `marketdata`, `forecasting`, `tenantauto`, `tests`, `triggers`, `workers`, `scheduler`, `catalogsync`) — confirm with team first | 2 days | LOW | Cleans up dead code; reduces surface area |
| **2a** | Extract `auth` → `libs/auth` (refactor to decouple from `models`) | 1 week | LOW | 22 importers, 1 file, common pattern |
| **2b** | Extract `identity` + `requestcontext` → `libs/identity` | 2 days | LOW | 14 importers combined |
| **2c** | Extract `help` → `libs/help` | 1 day | LOW | 1 importer, pure utility |
| **2d** | Extract `metrics` → `libs/metrics` | 1 day | LOW | Generic metric definitions |
| **2e** | Extract `access` → `libs/access` | 1 day | LOW | 1 file, 2 importers |
| **2f** | Extract `infrastructure` → `libs/infrastructure` (verify first) | 1 day | LOW | 1 file, 2 importers |
| **3a** | Break `api ↔ reports` cycle (refactor `reports` to not import `api`) | 1-2 weeks | MEDIUM | Cycle must be broken before any extraction |
| **3b** | Break `wealth ↔ calcengine` cycle | 1 week | MEDIUM | Same |
| **3c** | Extract `validation` → `libs/validation` (after refactoring to decouple from `rules`) | 1 week | MEDIUM | 7 importers |
| **3d** | Extract `nl_intelligence` → `libs/nl-intelligence` | 2 weeks | MEDIUM | Genuinely reusable |
| **3e** | Extract `oauth` + `google` → `libs/oauth` (after security decoupling) | 2 weeks | MEDIUM | 4+2 importers |
| **3f** | Extract `guardrails` → `libs/ai-sdk` (or `libs/guardrails`) | 1 week | MEDIUM | AI guardrails are generic |
| **4a** | Extract `events` → `libs/events` (after refactoring to break model deps) | 2-3 weeks | HIGH | 17 importers, requires interface extraction |
| **4b** | Decompose `services` (145 files) into focused service modules | 2-3 months | HIGH | Largest cleanup, but unlocks many extractions |
| **4c** | Extract `cube` sub-cluster (cube, cubeengine, cbo, viewmodel) to `libs/cube` | 1 month | HIGH | 22+8+1+6 importers; 30+ files |
| **4d** | Extract wealth domain (18 packages) to `libs/wealth` | 2-3 months | HIGH | Largest domain extraction |

### Total estimated timeline: 4-6 months for full extraction

### Parallelization opportunities:
- Phase 2 extractions (2a-2f) can be done in parallel by different engineers — they're independent
- Phase 3 work on cycles (3a, 3b) can be done in parallel with Phase 2 (different file scopes)
- Phase 4 should be sequential because they all depend on `services` decomposition

---

## 8. KEY METRICS

| Metric | Value |
|--------|-------|
| Total top-level packages | 123 |
| Total .go files | 1,590 |
| Total LOC | ~440,075 |
| HUB packages (≥10 importers) | 23 |
| LEAF packages (≤1 importer) | 33 |
| Cycles in dep graph | 2 (wealth↔calcengine, api↔reports) |
| Already-extracted to libs/ | 1 (`logging` is duplicate) |
| Already-extracted as Go module | 1 (`/calc-engine/`) |
| Largest package by files | `api` (270 files, 84K LOC) |
| Largest package by importers | `logging` (102 importers) |
| Smallest packages | 47 packages with exactly 1 .go file |

### Top 10 packages by file count:
1. `api` — 270 files (84K LOC)
2. `handlers` — 152 files
3. `services` — 145 files (45K LOC)
4. `analytics` — 70 files (30K LOC)
5. `ops` — 36 files
6. `wealth` — 33 files
7. `audit` — 32 files
8. `workflows` — 28 files
9. `boresolver` — 26 files
10. `rules` — 26 files

These top 10 packages = **1,088 files (68% of total)**, confirming the heavy concentration of code in a few hubs.


---

## 9. EXISTING `libs/` STRUCTURE & GAPS

The repo already has these extracted libs (in `go.work`):

| Lib | Module | Files | What it does |
|-----|--------|-------|--------------|
| `abac-client` | `.../libs/abac-client` | 1 | ABAC policy client (overlap with internal/security?) |
| `ai-sdk` | (no go.mod) | 0 | Empty — likely TypeScript/Node only |
| `auth` | `.../libs/auth` | 6 | JWT/claims/middleware (overlap with internal/auth and internal/security) |
| `domain-types` | `.../libs/domain-types` | 3 | Alternative investments, crypto, upgrade types |
| `hasura-client` | `.../libs/hasura-client` | 1 | Hasura GraphQL wrapper |
| `jwt-middleware` | `.../libs/jwt-middleware` | 2 | JWT HTTP middleware (used by many internal packages) |
| `logging` | `.../libs/logging` | 1 | Logger (DUPLICATE of internal/logging) |
| `shared-types` | `.../libs/shared-types` | 1 | Permission types |
| `temporal-client` | `.../libs/temporal-client` | 2 | Temporal client wrapper |

### Overlap analysis (suggests further consolidation):
- `libs/auth` vs `internal/auth` — different concerns (JWT vs context), but `internal/security` may overlap with `libs/abac-client`
- `libs/logging` vs `internal/logging` — **exact duplicate** (md5 confirmed)
- `libs/temporal-client` vs `internal/temporal` — likely overlap
- `libs/hasura-client` vs `internal/graphql` — different (Hasura GQL vs internal GraphQL schema)

### Gaps where new libs should be created:
1. `libs/validation` — for `internal/validation` after refactor
2. `libs/identity` — for `internal/identity` + `internal/requestcontext`
3. `libs/help` — for `internal/help`
4. `libs/metrics` — for `internal/metrics`
5. `libs/access` — for `internal/access`
6. `libs/infrastructure` — for `internal/infrastructure` (after verification)
7. `libs/nl-intelligence` — for `internal/nl_intelligence` (genuinely reusable)
8. `libs/cube` (longer term) — for the cube/boresolver/cbo subcluster
9. `libs/wealth` (longer term) — for the wealth domain

