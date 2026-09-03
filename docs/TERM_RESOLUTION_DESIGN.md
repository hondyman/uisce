# Term Resolution & Possessive Binding — Design

Status: Design signed off; engineering execution underway (see §9) — masking-enforcement and advisor-assignment owners still open
Owner: egan.patrick@gmail.com
Related: [SEMANTIC_LAYER_ARCHITECTURE.md](SEMANTIC_LAYER_ARCHITECTURE.md), [cue/semantic_terms.cue](../cue/semantic_terms.cue), [backend/internal/ai/vocabulary/resolver.go](../backend/internal/ai/vocabulary/resolver.go), [backend/internal/api/business_terms_handler.go](../backend/internal/api/business_terms_handler.go), [backend/db/migrations/20260821_3tier_taxonomy_and_classified_by.sql](../backend/db/migrations/20260821_3tier_taxonomy_and_classified_by.sql)

## 0. Empirical audit — read before anything else in this doc

Every earlier draft of this doc was written from code and migration files without checking what's actually live. That produced two rounds of invented mechanisms (`REALIZES`, `catalog_aliases`, `MAPS_TO_VALUE`, `CHILD_OF` taxonomy) that either duplicate or contradict working systems. This section replaces inference with a direct query against **alpha** (`$DATABASE_URL`, live via `.env`), run 2026-09-03.

**Method, in order of trust:** live `information_schema` + row counts (ground truth) > migration bookkeeping table (`schema_migrations`, found in the `vend` schema, **0 rows** — not used, meaning schema was built by hand-running SQL files, not a tracked migration tool) > `go:embed` grep (**no hits** — confirms no embedded migration runner) > migration files themselves (**lowest trust — several don't match live schema at all**, see below).

### Schema reality

`catalog_node_type` / `catalog_edge_type` (singular) are **views** over the real tables `catalog_node_types` / `catalog_edge_types` (plural) — both naming conventions seen in code are valid, this is not drift. Real columns: `catalog_node(id, tenant_id, node_type_id, node_name, description, properties jsonb, config jsonb, qualified_path, parent_id, is_active, governance_status, branch_id, ...)`, `catalog_edge(id, tenant_id, source_node_id, target_node_id, edge_type_id, relationship_type varchar, properties jsonb, is_active, governance_status, branch_id, ...)`. Two conventions for typing an edge **do** coexist on `catalog_edge` — a denormalized `relationship_type` text column and a normalized `edge_type_id` FK — and they are **not kept in sync** (see counts below); any new write must pick one and document it (this doc: **`edge_type_id`**, since it's the one with a registry and referential integrity).

`backend/db/migrations/20260821_3tier_taxonomy_and_classified_by.sql` references `source_id`/`target_id`/`edge_type_name` and `catalog_node_type.type_name` — **none of these column names exist on alpha.** That file did not run here, or ran against an entirely different database. `backend/db/migrations/` as a directory is not trustworthy as a description of live schema; treat it as historical/aspirational only.

`business_terms` (flat legacy table) and `catalog_aliases` (the table this doc's earlier draft designed around) — **neither exists on alpha**: `to_regclass()` returns NULL for both.

### What's registered vs. what has data

| Thing | Registered (type exists)? | Live data? |
|---|---|---|
| `business_term` catalog_node_type | Yes | **Yes — 88 nodes** |
| `Classification_L1`/`L2`/`L3`/`L4` catalog_node_types | Yes | **No seeded tree, no `CLASSIFIED_BY` edges** — types exist, taxonomy is empty |
| `ALIAS_OF`, `HAS_SYNONYM`, `MAPS_TO_SEMANTIC_TERM`, `CLASSIFIED_BY` edge types | **No — not even registered in `catalog_edge_types`** | **Zero rows**, by either `relationship_type` or `edge_type_id` |
| `MAPS_TO` edge type | Yes | **Yes — 234 edges (`edge_type_id`), 14 edges (`relationship_type`)** — the two typing columns disagree, confirming they drift |
| `HAS_BUSINESS_TERM` edge type | Yes | **Yes — 96 edges** |

`MAPS_TO` connects `column → semantic_term` (220 of the 234) and a handful of `semantic_term → column`/`semantic_term → table` (likely reverse-direction data-entry mistakes, not a second valid direction — worth a data-quality pass, not addressed by this doc). **This is the N:1 normalization edge** the earlier draft called `MAPS_TO_VALUE` — the real name is `MAPS_TO`.

`HAS_BUSINESS_TERM` connects `semantic_term → business_term` (86 of 96; the remaining 10 look like mismatched-direction noise). 86 edges against 88 business_term nodes is close to the 1:1 this doc wants — **this is the real semantic↔business bridge**, not the `REALIZES` or `MAPS_TO_SEMANTIC_TERM` names invented/found in earlier drafts.

**`backend/internal/ai/vocabulary/resolver.go` is deployed but functionally inert on alpha for three of its four resolution strategies.** `resolveViaAlias` and `resolveViaSynonym` query `ALIAS_OF`/`HAS_SYNONYM` edges that don't exist — always return zero rows. `resolveViaEmbedding` queries the `business_terms` table, which doesn't exist — the query errors and the code swallows the error (`return nil, nil` on error, `internal/ai/vocabulary/resolver.go:206-209`), silently contributing nothing. Only `resolveViaBusinessTermName` (exact, case-insensitive match against `business_term.node_name`) can return a result today. **This is a live correctness gap independent of this design doc** — worth its own fix regardless of what follows here, since today "account" or "PF" resolves to nothing even though the resolver *looks* like it handles synonyms.

### Possessive binding has no OLTP home either — a Rule 6 correction

Earlier drafts of §5 proposed an `entity_node --INSTANCE_OF--> semantic_term` catalog_edge, one per portfolio/fund/account row. **That's wrong and it's the same mistake Rule 6 exists to prevent** (the user's own framework, Feature 4: "a developer stores `break_amount` directly in the graph node... graph node only exists so the Studio UI can draw the break"). Stamping a `catalog_node` + edge per OLTP entity instance — potentially millions of rows across tenants — pulls mutable instance data into the graph, which Rule 6 explicitly reserves for identity/semantic configuration only.

Checked what actually exists: `portfolios` (OLTP table) has `client_id` — direct FK, no graph involvement needed for "my portfolios" from a **client's** perspective, it's a plain filtered query. There is **no existing table** for "which PM/advisor manages which portfolio" — `ownership_relationships` exists but models corporate/beneficial ownership (`percentage_ownership`, `voting_power`, `has_board_seat`) and is empty (0 rows); it is not an advisor-assignment table and should not be repurposed as one.

**Corrected design for §5**: possessive binding is resolved by a **config-driven OLTP filter**, not a graph traversal per instance — but the config itself must respect the business/semantic seam this whole doc is built on, not just avoid graph bloat. See §5 for the corrected split (business declaration on `business_term`, physical filter on `semantic_term`) and the safety/identity gaps that split surfaced.

## 1. Problem

The AI layer needs to resolve natural-language references — synonyms across data stores ("account" vs "fund" vs "portfolio"), abbreviations ("PF", "port"), and possessives ("my portfolio") — into concrete, tenant-scoped, physically-bound entities, without hardcoding any of that mapping in Go/TS. This must be config-in-the-graph for terminology (Rule 1), config-in-OLTP-metadata for possessives (Rule 1 + Rule 6, see §0), tenant-safe on every traversal (Rule 7), and routed through `buildUnionSafeQuery` for any historical/hot-cold lookup (Rule 4).

Two problems, two mechanisms:
- **Terminology equivalence** (synonym/alias/abbreviation) — static semantic configuration, lives in the graph as nodes/edges, edited ahead of time. **Partially built already** (§0) — `business_term` nodes and the `HAS_BUSINESS_TERM`/`MAPS_TO` bridge exist and are populated; the alias/synonym edges the resolver code expects do not.
- **Possessive binding** ("my") — runtime identity resolution; depends on who's asking, resolves to a *set* of entity instances via an OLTP filter driven by graph config, never a graph traversal per instance (§0). **Not built at all.**

## 2. Layering (as it actually exists on alpha, §0)

```
LANGUAGE LAYER      surface forms — NOT YET LIVE (edge types unregistered, zero data)
                          │  "account", "PF", "my portfolio"
                          ▼
BUSINESS LAYER       business_term (catalog_node_type, LIVE — 88 nodes)
                     taxonomy types registered (Classification_L1-4) but
                     UNSEEDED — no tree, no CLASSIFIED_BY edges
                          │  HAS_BUSINESS_TERM (LIVE — 86 edges, semantic_term → business_term)
                          ▼
SEMANTIC LAYER       semantic_term (catalog_node_type, LIVE)
                          │  MAPS_TO (LIVE — 220 edges, column → semantic_term)
              ┌───────────┼──────────────┬──────────────┐
              ▼           ▼              ▼              ▼
          physical columns across sources (per MAPS_TO source_node_id)
```

**Reuse, not reinvent — corrected names:**
- `business_term` — already a `catalog_node_type`, already has 88 live nodes, already has a CRUD API (`business_terms_handler.go`) and a glossary/compliance surface. Do not re-register it.
- `HAS_BUSINESS_TERM` — already the live semantic↔business bridge edge (`edge_type_id`-typed). This doc adopts it in place of the invented `REALIZES`/`MAPS_TO_SEMANTIC_TERM` names. It is not currently enforced 1:1 (86 edges / 88 nodes is close but not validated as a constraint) — §3 addresses that.
- `MAPS_TO` — already the live N:1 normalization edge (`column → semantic_term`), replacing the invented `MAPS_TO_VALUE` name.
- Taxonomy node types (`Classification_L1..L4`) — registered but empty. `backend/db/migrations/20260821_3tier_taxonomy_and_classified_by.sql` is the *intended* seed for this but does not match live column names (§0) and needs to be rewritten against real schema before it can run, not merely re-executed.

## 3. Business term layer — what to add, not what to build from scratch

No new `catalog_node_type` registration. `business_term` exists; extend its `properties`/`config` jsonb (the existing generic infra — no schema migration needed for these, they're additive keys in an already-jsonb column) with the fields the earlier draft proposed as new columns:

| Property (in `catalog_node.properties`) | Notes |
|---|---|
| `definition` | business-readable; fed into AI context pack |
| `sensitivity` | FK-by-convention to `data_classification_templates.name` (see below) |
| `pii`, `retention` | as before |
| `term_kind` | `dimension` / `measure` / `attribute` — new; gates possessive binding (only `dimension` resolves a "my X") |
| `status` | `draft`/`in_review`/`approved`/`published`/`deprecated` — check whether `business_term` nodes already carry a status convention via `governance_status` (the enum column found on `catalog_node` itself, §0) before adding a duplicate property; **open item**, not yet checked |

**Taxonomy**: adopt the registered `Classification_L1/L2/L3` types and a `CLASSIFIED_BY` edge from `business_term → Classification_L3`, matching the *intent* of `20260821_3tier_taxonomy_and_classified_by.sql` — but that file must be rewritten against real column names (`source_node_id`/`target_node_id`, `catalog_type_name` via the view, `edge_type_id`) before it can be the actual migration. This doc no longer proposes `CHILD_OF` between business_term nodes — the taxonomy is a separate node-type chain (L1→L2→L3) that business_term leaves attach to via `CLASSIFIED_BY`, which is a cleaner structure than what this doc originally invented and should be kept.

**1:1 semantic↔business, via existing `HAS_BUSINESS_TERM`**: not currently constrained. Add a unique index enforcing at most one `HAS_BUSINESS_TERM` edge per `(source_node_id)` where `edge_type_id` = the `HAS_BUSINESS_TERM` type id — this is additive (a constraint on an existing, populated edge type) and should be preceded by a duplicate-check query against the live 96 rows, the same discipline as any other constraint-over-existing-data change (see §8 migration notes).

Governance inheritance and the sensitivity-tightening rule (severity is not a total order — `restricted_pii`/`restricted_financial` share a rank and require set-inclusion, not linear comparison; formula and `data_classification_templates` citation unchanged from the prior draft) still apply, now anchored to `business_term.properties.sensitivity` and `MAPS_TO` edge overrides instead of the invented edge name. **Runtime consumer of `data_classification_templates` for enforcement is still unconfirmed** — the only reference found (`backend/internal/graphql/datasource_operations.go:416`) is an admin-listing query, not a masking enforcement path. This remains the doc's biggest liability and its own tracked, owned workstream (§8/§9) — that finding is unchanged by this rewrite.

## 4. Terminology equivalence layer — build on the graph, not a new table

**Adjudication, now decided on data instead of guesswork (per the review that asked for this):**

1. *"Does `HAS_SYNONYM` violate the hub-and-spoke rule?"* — Moot today: zero live `HAS_SYNONYM` edges exist, so there's no live transitive-resolution bug. But the **hazard is real in the code as written** — nothing stops a future `synonym`-type node from pointing at another `synonym` node instead of a `business_term`. Fix: **collapse `ALIAS_OF` and `HAS_SYNONYM` into one edge type.** They're semantically identical in the code (`surface_form_node --EDGE--> business_term`) and maintaining two names for the same relationship is exactly the kind of drift that produced the `relationship_type`/`edge_type_id` split in §0. Pick `ALIAS_OF`, register it in `catalog_edge_types` with `source_node_type_id`/`target_node_type_id` constraining `synonym → business_term` (the type table already supports this — `source_node_type_id`/`target_node_type_id` columns exist per §0's schema dump), which makes hub-and-spoke a DB-enforceable constraint instead of a convention.
2. *"Formalize or deprecate the legacy `business_terms` embedding table?"* — Neither: **it doesn't exist on alpha**, so there's nothing to formalize. Do not recreate it. Embeddings for fuzzy alias matching need a small dedicated table (not a full alias-store rebuild): `catalog_node_embedding(node_id UUID REFERENCES catalog_node(id), embedding vector(1536), embedding_model TEXT, status TEXT DEFAULT 'pending')`, generic enough to cover `synonym` nodes now and other node types later, populated async (same async/degrade-to-trigram reasoning as the earlier draft, unchanged).

**Mechanism**: a `synonym`-type `catalog_node` (register this node type — it does not exist yet, confirmed absent from the taxonomy/type list in §0) holds the surface form in `node_name`; an `ALIAS_OF` edge (`edge_type_id`-typed, per §0's convention decision) points at the target `business_term`. Confidence/status/store-hint — properties the earlier `catalog_aliases` design put in table columns — become **edge properties** on `ALIAS_OF` instead: `{"status": "published", "store_hint": [...], "match_priority": 100, "kind": "abbreviation"}`. This keeps everything on `catalog_node`/`catalog_edge`, no new tables beyond the embedding side-table above.

Resolution order (rewriting `resolver.go`'s dead paths to point at real data, not adding new strategies): exact `ALIAS_OF` match (published only) → direct `business_term.node_name` match (already works) → trigram → cosine similarity via `catalog_node_embedding` where `status = 'ready'`. `store_hint` filter and the "return candidates, never guess" below-threshold rule are unchanged from the earlier draft.

Conflict rule — same DB-constraint posture as before, now against the edge table: a partial unique index on `catalog_edge` scoped to `edge_type_id = <ALIAS_OF>` preventing two unscoped (no `store_hint`) `ALIAS_OF` edges from the same normalized surface form to different business terms. **Run a duplicate-detection query first** — trivial here since there's currently zero data to conflict with, but write the check into the migration anyway since this table will be populated by more than one path (Studio + `resolveViaBusinessTermName`-adjacent tooling) going forward.

## 5. Possessive binding — corrected, see §0

Not a term resolution — a scope resolution driven by the requester's identity, resolved as an **OLTP filter configured by graph metadata**, not a graph traversal over per-instance nodes (§0 explains why `INSTANCE_OF` was wrong).

### 5.1 The config must respect the business/semantic seam — corrected again

The first pass of this fix put `{"table": "portfolios", "owner_column": "client_id"}` directly on `business_term.properties`. That's a second layering violation, not a fix of the first one: it puts a **physical binding** — table and column name — on the layer this doc has consistently said holds only business judgments (PII, retention, taxonomy, definition). A steward editing a business term's definition or taxonomy placement should not be able to break a query path by fat-fingering a column name in the same JSON blob. And it duplicates what `MAPS_TO` already does at the semantic layer: point a concept at its physical realization.

Split it across the seam, matching every other governance/binding decision in this doc:

- **`business_term.properties.possessive`** — business declaration only: `{"enabled": true, "possessive_term": "my", "cardinality_hint": "many"}`. This is what a steward governs — whether the concept is possessable at all, and what the natural-language possessive maps to (i18n-ready).
- **`semantic_term.properties.scope_config`** (new, adjacent to the existing `MAPS_TO` bindings, same reasoning as "physical realization lives at the semantic layer") — the physical filter: `{"role": "client", "table": "portfolios", "owner_column": "client_id", "identity_key": "session.user.client_id"}`. Multiple roles can attach to one semantic term as a list (a `client` role now, an `advisor` role once that table exists).

`ScopeResolver` checks `business_term.possessive.enabled` first (does this concept support "mine" at all — this is also where `term_kind == dimension` gets checked, §3), then reads `scope_config` off the *linked* semantic term (via `HAS_BUSINESS_TERM`) for the actual table/column/identity-key to filter on. Business meaning and physical binding stay exactly as separated as they are everywhere else in this doc.

**One `scope_config` per semantic term, not one per `MAPS_TO` binding.** A semantic term can be physically realized in several sources — the live data confirms this is real, not hypothetical: e.g. `OrderId` on alpha has 10 distinct `MAPS_TO` source tables (Northwind sample data; no `Portfolio`/`Account`/`Fund` semantic term currently has *any* `MAPS_TO` binding yet, so this is unverified in the domain this doc actually cares about, but the fan-out pattern itself is confirmed live). `scope_config` is deliberately **one canonical assignment binding per semantic term**, not one per source — ownership/possession is an identity fact ("who this belongs to"), the same category as the `node_id` bridge Rule 6 already uses to unify a concept across OLTP tables, not a per-store fact that's allowed to fork the way raw balances or positions can differ hot vs. cold. If a future domain's data ever shows the true owning set actually differing by source (e.g. custodian says one PM, IBOR says another), that is a **reconciliation condition** to surface through the existing recon machinery (Rule 6's `recon_break` path) — not something `ScopeResolver` silently picks a winner on or forks its own ambiguity handling for. `ScopeResolver` never reads `scope_config` per-binding; it reads the one config on the semantic term.

### 5.2 Scope-config is one Studio edit from arbitrary SQL targets — needs guardrails, not just structure

A property that names a table and column, consumed by code that builds a filter query from it, is a SQL-injection-shaped hazard even when the values come from an internal config UI rather than user input — Studio access is not the same as "trusted to name any table in the database." Before this ships:

- **The referenced table/column must be independently resolvable through the catalog** — i.e. there must be an existing `catalog_node` (type `table`/`column`) with that qualified path, reachable the same way the query builder already trusts physical bindings from `MAPS_TO`. `ScopeResolver` rejects any `scope_config` whose `table`/`owner_column` doesn't resolve to a cataloged column. No free-text table name survives into a query untraced.
- **The filter is parameterized and tenant-scoped by construction, not by config author discipline** — the query builder always injects `tenant_id = $1` itself; `scope_config` supplies only the table/column names (used to build the SQL identifier via the catalog-validated allowlist above, never string-concatenated from the raw property value) and the parameter binding for the identity value. The config author cannot add, remove, or bypass the tenant predicate.
- **`RequireTenantOwnership` still runs on the result set** (Rule 7) — the config deciding *where* to filter doesn't substitute for verifying *what* came back. Defense in depth: catalog-validated table/column, tenant predicate injected by the builder, and a final ownership check on the returned rows.

### 5.3 The identity seam — previously unspecified, now the load-bearing question

Graph traversal had a self-evident identity story (authenticated identity node → traverse edges). The OLTP-filter version needs an explicit answer to "how does an authenticated session become a value in `owner_column`?" `scope_config.identity_key` names this: a dotted path resolved against the authenticated session/claims object (e.g. `session.user.client_id`), evaluated server-side in `ScopeResolver`, never client-supplied. It must resolve to an identifier that's (a) guaranteed unique per tenant, (b) present on every row `owner_column` is expected to filter, and (c) the same identifier the auth system already treats as the tenant-scoped principal — reuse whatever `RequireTenantOwnership` already keys on, don't invent a second identity representation.

- **The `client` role works today**: `portfolios.client_id` and the session's tenant-scoped client identifier are the same kind of value — this path is buildable now.
- **The `advisor` role has no backing table** (§0 — `ownership_relationships` is corporate/beneficial ownership, not advisor assignment, and is empty). This is not a parallel organizational blocker to note and move past — it's a **prerequisite feature** with its own design surface: a tenant-scoped, identity-keyed assignment table, whose schema needs to answer the same `identity_key` question this section just raised, and whose write path (PM reassignment) is exactly what the scope-token TTL (§6) was designed to bound the staleness of. Promoted from "open item" to its own tracked workstream in §8, with this identity-mapping question handed to whoever picks it up.
- **Multi-hop possessive ("my manager's portfolios")** stays out of scope until a real hierarchy table exists — same reasoning, don't design hop-cap mechanics against a table that isn't confirmed to exist.

Result is a scope descriptor (§6), not an enumerated set — the cap/`scope_token` mechanics are storage-agnostic and apply equally to an OLTP-filter-based resolution.

## 6. Resolver contract (domain port, Rule 3)

```go
package domain

type TermResolver interface {
    ResolveTerm(ctx context.Context, tenantID, surfaceForm string) (TermResolution, error)
}

type ScopeResolver interface {
    ResolveScope(ctx context.Context, tenantID, identityID string, term TermResolution) (ScopeResolution, error)
}

type TermResolution struct {
    BusinessTermID string
    Confidence     float64
    Ambiguity      *Ambiguity // nil when resolved unambiguously
}

type Ambiguity struct {
    Reason     string       // "no_match" | "below_threshold" | "multiple_unscoped_matches"
    Candidates []Candidate
}

type Candidate struct {
    BusinessTermID string
    Label          string
    Confidence     float64
}

type ScopeResolution struct {
    BoundVia    string // e.g. "portfolios.client_id" — the OLTP filter used, not an edge type
    Count       int
    Enumerated  bool
    NodeIDs     []string
    ScopeToken  string
}
```

`TermResolver`'s implementation is **not new code from scratch** — it's `backend/internal/ai/vocabulary/resolver.go`, fixed to query the real `ALIAS_OF`/`business_term` structure from §4 instead of its currently-dead `ALIAS_OF`/`HAS_SYNONYM`/`business_terms`-table paths, then wrapped to satisfy this interface (Rule 3: the port is satisfied by existing code, not reimplemented). `ScopeResolver` is genuinely net-new (§5).

Concrete implementations live in the catalog/vocabulary package; injected at `main.go` per Rule 3. Historical/balance-bearing lookups downstream of resolution go through `domain.GLBalanceResolver`/`buildUnionSafeQuery` per Rule 4 — the resolver only ever returns identifiers and physical bindings, never raw query results.

**Scope does not enumerate at scale.** Cap inline enumeration at ≤ 50 ids; above that, `Enumerated = false` and a `scope_token` (`scope:<identity>:<business_term>:<version>`) is returned instead, resolved server-side through `buildUnionSafeQuery`, never client/LLM-side. The token doubles as an audit artifact.

Output shape (AI context pack):

```json
{
  "business_term": {
    "id": "<business_term catalog_node id>",
    "term_kind": "dimension",
    "definition": "...",
    "taxonomy_path": ["Portfolio, Accounting & Custody", "Account Master", "Account Identification"],
    "governance": {"sensitivity": "internal", "pii": false, "retention": "7y"},
    "aka": ["fund", "account (custodian)", "PF", "port"]
  },
  "semantic_term": {
    "id": "<semantic_term catalog_node id>",
    "sources": ["<columns via MAPS_TO>"]
  },
  "scope": {"bound_via": "portfolios.client_id", "count": 3021, "enumerated": false, "scope_token": "scope:pm_jane:bt.portfolio:v7"},
  "ambiguity": null
}
```

Ambiguity payload, cache invalidation (config-version bump for term-layer changes, TTL bounding for identity/possessive-config drift, re-verification on resolve rather than trusting a materialized list), and the `resolution_feedback` write path (async, rate-capped, deduped on a sorted+versioned candidate-set hash) are unchanged in mechanism from the prior draft — none of that depended on the table-vs-edge question this rewrite resolved. Full detail:

```json
"ambiguity": {
  "reason": "multiple_unscoped_matches",
  "candidates": [
    {"business_term_id": "...", "label": "Portfolio", "confidence": 0.71},
    {"business_term_id": "...", "label": "Ledger Account", "confidence": 0.62}
  ]
}
```

`reason` is a closed enum: `no_match`, `below_threshold`, `multiple_unscoped_matches`, `not_a_dimension` (possessive against a `measure`/`attribute` term).

```sql
CREATE TABLE resolution_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    surface_form TEXT NOT NULL,
    normalized_surface_form TEXT NOT NULL,
    reason VARCHAR(30) NOT NULL,
    candidates JSONB,
    candidate_set_hash TEXT NOT NULL,  -- hash_v1: sha256 of candidate business_term ids sorted ascending, "v1:" prefix
    hit_count INT DEFAULT 1,
    first_seen_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ DEFAULT NOW(),
    resolved_alias_edge_id UUID,  -- catalog_edge.id once Inbox promotes it (was resolved_alias_id UUID REFERENCES catalog_aliases — table doesn't exist, see §0/§4)
    UNIQUE (tenant_id, normalized_surface_form, candidate_set_hash)
);
```

Upsert on `hit_count`/`last_seen_at`; async off the request path; hard per-tenant-per-window cap on new distinct rows (not upserts) to prevent an AI retry loop from write-amplifying this table. `surface_form` is a term guess, not user content, but still tenant-scoped and subject to the same retention policy as other catalog config.

**Note on the context pack**: `governance.sensitivity` is intentionally included — it's a label the AI needs to reason about masking, not the data itself — but if prompts are logged, a `restricted_pii` label now sits in those logs. Confirm with whoever owns prompt-log retention; not blocking, but should be a stated decision.

## 7. Studio UX (unchanged scope, still follow-up work)

Business Terms editor (now: an editor over the *existing* business_term nodes and their `properties`/`CLASSIFIED_BY` taxonomy placement, not a from-scratch node type) → Aliases (now: an editor over `ALIAS_OF` edges from `synonym` nodes, per §4) → Identity & Possessives (now: an editor spanning both `business_term.properties.possessive` — the enable/label toggle a steward sets — and the linked `semantic_term.properties.scope_config` — the table/column/identity-key binding, catalog-validated per §5.2, likely a data-engineer-facing sub-panel rather than the same form as the business declaration) → Resolution Inbox (`resolution_feedback`, §6) → Sandbox → Governance. No existing analog found in `backend/catalog-admin`; still separate feature work, `resolution_feedback` write path still lands with the resolver itself.

## 8. Open items / next steps

### 8.0 Root cause: no migration tooling — fix this before writing more schema

`schema_migrations` has 0 rows and no `go:embed` runner exists (§0). That single gap is the common cause of every drift finding in this doc: three incompatible migration directories, a taxonomy seed referencing columns that don't exist, and hand-run SQL as the only lineage. **Standing up a real runner (`goose` or `golang-migrate`) is a prerequisite, not a parallel task** — writing this doc's migration set into a fourth untracked directory just recreates the problem this whole investigation was spent diagnosing.

Concretely: pick one tool, embed its migrations directory in the Go binary (`//go:embed migrations/*.sql` or equivalent), and seed `schema_migrations` with a **baseline marker** representing current live state — do not attempt to replay years of hand-run history through it. From that point forward, every schema change in this doc's migration set (§8.2) goes through the tracked runner, starting with the taxonomy seed rewrite as its first real entry (a content fix with concrete column references — a good, low-risk exercise for the new tooling before anything riskier goes through it).

**Before the first migration runs against alpha: take a snapshot/backup, or clone to a scratch DB, and rehearse every migration against the clone first.** No tracked history plus hand-run SQL means there is currently no clean rollback path if a migration goes wrong — the new tooling's first act must not be an unrehearsed write against the only live copy of this data. "Done" for tooling setup is defined precisely: `migrate up` against a fresh clone of alpha's current state produces a schema identical to live alpha, verified with the same `information_schema` query used for the §0 audit (nice symmetry — the audit tool becomes the verification tool).

### 8.1 Remaining open items

- [x] Live schema audited directly against alpha (§0) — settles the `business_term`/alias/taxonomy/`INSTANCE_OF` questions that prior drafts guessed at.
- [x] Alias mechanism: graph edges (`ALIAS_OF`, collapsed from `ALIAS_OF`+`HAS_SYNONYM`), not a `catalog_aliases` table — table doesn't exist and shouldn't be created.
- [x] Possessive mechanism: OLTP filter driven by config, not `INSTANCE_OF` graph edges — corrects a Rule 6 violation in the earlier draft.
- [x] Possessive config split back across the business/semantic seam (§5.1) — the first fix recreated a layering violation by putting a physical binding on `business_term`; corrected to `business_term.possessive` (declaration) + `semantic_term.scope_config` (physical filter).
- [x] Scope-config safety clause specified: catalog-validated table/column, builder-injected tenant predicate, `RequireTenantOwnership` on results (§5.2).
- [x] Identity→assignment-column mapping specified as `scope_config.identity_key`, resolved server-side against session/claims, reusing whatever identifier `RequireTenantOwnership` already keys on (§5.3).
- [ ] **Stand up migration tooling before writing new schema** (§8.0) — this is now sequenced ahead of the migration set below, not alongside it.
- [ ] **Fix `resolver.go`'s dead code paths**, loudly: `resolveViaAlias`/`resolveViaSynonym` query edge types with zero data; `resolveViaEmbedding` queries a nonexistent table and silently swallows the error. The fix must (a) query the real `ALIAS_OF`/business_term structure from §4, and (b) **add a metric for resolution-path fallbacks/failures** — the current dead path shipped with zero signal and was only found by a schema audit; a metric turns the next one into a dashboard entry instead. Ship as its own standalone PR ahead of the rest of this doc's work, and **note in release notes that production resolution has been exact-match-only** — sets expectation that alias resolution visibly improving later is a feature landing, not a regression being fixed elsewhere.
- [ ] **Masking enforcement has no runtime consumer** (§3) — still the doc's biggest liability, still needs an owner and milestone, separate workstream from this migration set.
- [ ] **Advisor/PM → portfolio assignment — promoted from blocker to its own workstream** (§5.3): a tenant-scoped, identity-keyed assignment table is a prerequisite feature, not a parallel org task. Its design must answer the same `identity_key` question §5.3 raised, and its write path (reassignment) is exactly what the scope-token TTL (§6) exists to bound. "My portfolios" only works today for the `client` role via `portfolios.client_id`.
- [ ] Confirm whether `business_term` nodes already carry a status/lifecycle convention via `catalog_node.governance_status` before adding a redundant `status` property (§3).
- [ ] `MAPS_TO`/`HAS_BUSINESS_TERM` edges pointing the "wrong" direction (`semantic_term → column`, `semantic_term → semantic_term`, `table → table` under `HAS_BUSINESS_TERM`) — small counts, likely data-entry noise, worth a cleanup pass but not blocking.
- [ ] Locate/confirm which package should own the fixed `TermResolver`/new `ScopeResolver` — likely `backend/internal/ai/vocabulary` given `resolver.go` already lives there.
- [ ] Confirm scope-token TTL against the operational SLA for portfolio/account reassignment — business/ops decision, now directly informed by whatever cadence the advisor-assignment workstream lands on.

### 8.2 Migration set — sequenced behind the tooling fix (§8.0)

1. **Stand up the migration runner** (§8.0) — prerequisite, not step 1 of the schema work.
2. **Rewrite and run the 3-tier taxonomy seed** against real column names (§3) — first entry through the new runner; a content fix with concrete references, low risk, exercises the tooling.
3. **Register `synonym` catalog_node_type + `ALIAS_OF` edge type** (with `source_node_type_id`/`target_node_type_id` constraining `synonym → business_term`, making hub-and-spoke a DB constraint). Additive, zero live data to conflict with.
4. **`catalog_node_embedding` table** (§4): `embedding vector(1536)`, `embedding_model TEXT`, `status TEXT DEFAULT 'pending'` — carried over verbatim from the earlier review's decisions (async population, trigram degradation while pending), not re-litigated by the storage-location redesign.
5. **1:1 unique index on `HAS_BUSINESS_TERM`** — **88 business_term nodes vs. 86 edges means 2 unlinked terms**; reconcile those first (same detect-then-decide discipline as the alias pre-flight check — find out *why* they're unlinked, don't just force a placeholder edge) before the unique index goes on.
6. **`business_term.properties` additive keys** (`term_kind`, `sensitivity`, `possessive` — declaration only, §5.1) and **`semantic_term.properties.scope_config`** additive key (physical filter, §5.1) — no schema migration needed for either (jsonb), just a populate/backfill pass over the 88 existing business_term nodes and their linked semantic terms.
7. **`resolution_feedback` table** — trivial, no dependencies.
8. **Sensitivity-tightening trigger on `MAPS_TO`** — depends on `data_classification_templates.rank`/flag columns (unchanged from prior draft) and on confirming a runtime masking consumer exists (open item above) before it means anything.

No `INSTANCE_OF` backfill, no `catalog_aliases` extension, no new `business_term` node type registration — all three were the largest/riskiest items in the previous migration plan and all three are gone because the investigation in §0 found either that the work was already done or that the design was wrong.

**Guardrail: import-boundary test in CI.** Assert the resolver package only reads `ALIAS_OF` edges scoped to `synonym → business_term` (the registered type constraint in step 3 backs this at the DB level too — defense in depth, not either/or).

### 8.3 Data prerequisite: `Portfolio`/`Account`/`Fund` have zero `MAPS_TO` bindings today

§0 found this and it needs restating as a real work item, not a footnote: **no semantic term in the `Portfolio`/`Account`/`Fund` family has any live `MAPS_TO` edge.** Every piece of §8.2's schema work can ship correctly and "my portfolio" still resolves to a business term with no physical realization and no scope — this is the data prerequisite for the design's actual purpose, not optional polish after the schema lands. It's a catalog-population task (likely partially scriptable from existing column metadata rather than fully manual), scheduled **alongside** the advisor-assignment workstream, not queued behind it — both are needed for the same first end-to-end resolution.

### 8.4 First milestone — the thing all of this is actually for

Not "migrations done," which is internal plumbing invisible to anyone outside engineering. The milestone that proves the design: **a user types "my PF" in the Sandbox (§7) and gets back a resolved business term, its `aka` set, a scope resolved from a real assignment table, and physical bindings — against live alpha data.** Every item in §8.1–8.3 exists to make that sentence true; anything that doesn't move that sentence closer to true is lower priority than something that does.

## 9. Sign-off checklist

- [x] Live schema empirically verified against alpha before any migration was written (§0) — the review's own "let the live schema answer, not the migration files" instruction, executed
- [x] Alias mechanism reconciled to existing `ALIAS_OF`/`business_term` graph structure, hub-and-spoke enforced as a DB type constraint
- [x] Possessive binding corrected from a Rule 6 violation (`INSTANCE_OF` per-instance nodes) to config-driven OLTP filtering
- [x] Possessive config re-split across the business/semantic seam after the first fix violated it (§5.1) — caught before ship, not after
- [x] Scope-config guarded against becoming a SQL-injection-shaped hazard: catalog-validated references, builder-injected tenant predicate, `RequireTenantOwnership` on results (§5.2)
- [x] Identity seam named explicitly (`scope_config.identity_key`) rather than left implicit (§5.3)
- [x] Both adjudications from the prior review resolved with data, not guesses (`HAS_SYNONYM` hazard is latent/moot today; legacy `business_terms` table doesn't exist, not "half-wired")
- [x] Sensitivity tightening check specified as a total, unambiguous rule (§3, formula unchanged from prior draft)
- [x] Migration set re-sequenced and shrunk to reflect what's actually net-new, and sequenced behind a migration-tooling fix instead of alongside it
- [x] Migration safety net specified: backup/clone alpha and rehearse every migration against the clone before it touches live data (§8.0)
- [x] Resolver fix scoped as a standalone deploy, now with fallback/failure metrics and a release-note callout, not just a query fix
- [x] `scope_config` multi-store question resolved: one canonical assignment binding per semantic term, not one per `MAPS_TO` source; divergence across sources is a recon condition, not a resolver ambiguity case (§5.1). No `Portfolio`/`Account`/`Fund` semantic term has a live `MAPS_TO` binding yet, so this is architectural, not yet empirically exercised in-domain.
- [x] Data prerequisite named explicitly: `Portfolio`/`Account`/`Fund` `MAPS_TO` population is required before any end-to-end resolution is possible, scheduled alongside advisor-assignment, not after it (§8.3)
- [x] First milestone defined concretely and independent of internal-plumbing completion (§8.4)
- [x] Migration tooling decision owner (`goose` vs `golang-migrate`, baseline-seed, and the alpha backup/rehearsal step) — **egan.patrick@gmail.com** (engineering execution — legitimately fillable by an individual contributor)
- [ ] **Owner for the masking-enforcement workstream — not yet named.** This needs a human with authority over the query-execution path and security posture; it cannot be satisfied by naming the engineer driving the rest of this doc's execution. Do not backfill this line with a name just to close the checklist — an incorrect owner here is worse than an open line, since it would make governance enforcement look assigned when it isn't.
- [ ] **Owner for the advisor-assignment data model — not yet named.** Needs someone who can talk to the business about how PMs and clients are actually associated — a data-model/product decision, not an engineering one. Same caveat: leave open rather than mis-assign.

**Status: design signed off; two organizational owners still open (see above).** Engineering execution can and should start now — none of items 1–3 below are blocked on those two names. Recommended order: (1) ship the `resolver.go` fix standalone, today — it's a live bug, not a design question, and establishes that this project fixes what's broken before building what's new; (2) stand up migration tooling same-day, in parallel, including the alpha backup/clone-rehearsal step (§8.0) — half a day, unblocks §8.2; (3) run the §8.2 migration set through the new runner, taxonomy seed first; (4) start the §8.3 `MAPS_TO` data population and the advisor-assignment workstream **in parallel with each other**, not sequentially, since both gate the same first milestone (§8.4); (5) Studio surfaces (§7) last, once real node types/edges/traffic exist to build against. The two open owner lines above should be named before workstream (4) needs to start in earnest, not before engineering execution begins.
