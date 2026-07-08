<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **uisce** (747080 symbols, 1072336 relationships, 300 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/uisce/context` | Codebase overview, check index freshness |
| `gitnexus://repo/uisce/clusters` | All functional areas |
| `gitnexus://repo/uisce/processes` | All execution flows |
| `gitnexus://repo/uisce/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->

## Core vs. Custom Inheritance Architecture Mandates

### Rule 1.3: Safe UUID Query Parameter Defenses (Config-Before-Code)
- Every query string or variable header that maps down to native Postgres `UUID` targets must pass through parsing sanity loops (`uuid.Parse`) prior to binding assembly.
- Any unpopulated, missing, or blank text parameter (`""`, `"null"`, `"undefined"`) must be caught and evaluated to a true variable `NULL` or result in a prompt HTTP exit block, completely preventing database `22P02` validation syntax crashes.

### Rule 6.2: Structural Dictionary Layer Separation (Semantic/OLTP Boundary)
- The structural data catalog defines semantic intent and functional topologies. Core shared definitions belong to the System Gold Copy workspace namespace (`00000000-0000-0000-0000-000000000000`), which is read-only and automatically inherited by active clients.

### Rule 7.4: Structural Multi-Tenant Union Enforcement (The Security Mandate)
- Do not build flat `WHERE tenant_id = ?` filters when looking up systemic structural types like Glossary nodes or Business Objects.
- The retrieval logic must aggregate global Core parameters and local Custom overrides using explicit `ROW_NUMBER() OVER (PARTITION BY asset_key ORDER BY precedence_rank DESC)` windows to ensure deterministic tenant shadowing.

### Rule 5.4: Compilation-Time Physical Abstraction (Graph-Driven Routing)
- The SQL generator engine (`BOSQLGenerator`) must never make raw assumptions about table paths or target names from structural models.
- All target extraction configurations (like the root source `FROM` clause or automated relational joins) must be compiled dynamically by looking up the active `bo_binding_id` properties inside the catalog layout.
- The entry-level abstraction mappings for semantic entities are strictly generated from the compilation paths of graph nodes and `MAPS_TO` parameters.

## Distributed Caching & Invalidation Rules

### Rule 8.1: Write-Before-Invalidate Order of Operations
- You MUST write to the database and successfully commit the transaction **before** publishing the invalidation event to the Redis pub/sub channel.
- This ensures concurrent readers do not race the database commit, fetch the old data from the DB, and re-cache it.

### Rule 8.2: Versioned Cache Keys for Atomic Swaps
- Cache keys should incorporate versions or hashes (e.g. `bo:{boId}:tenant:{tenantId}`) rather than simple flat deletions.
- This enables atomic swaps and check-before-refresh operations using version metrics checks.

### Rule 8.3: Synchronous Eviction via Memory-Speed Pub/Sub
- Real-time synchronous eviction across distributed service instances is achieved by subscribing to the `metadata:invalidation` Redis pub/sub channel, executing evictions in memory speed upon event arrival.



