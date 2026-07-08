# Cardinal Rules for Uisce Semantic OS

### 1. Config-Before-Code
Never hardcode business logic (fee math, reconciliation tolerances) in Go. Store as metadata/DSL in the graph. The execution engine must interpret the DSL, not the source code.

### 2. Graph-First
Topology dictates behavior. Every feature must introduce `catalog_node` or `catalog_edge` (`FEEDS_INTO`, `MAPS_TO_VALUE`) before implementation.

### 3. No Package Cycles
Domain ports (e.g., `ReconMatcher`) must be injected via interfaces. Adapters must never import rules directly; dependency inversion is mandatory.

### 4. Hot/Cold Watermarking
Historical balance lookups must flow through `GLBalanceResolver`, utilizing `buildUnionSafeQuery` to bridge Postgres and archival storage (Iceberg).

### 5. Graph-Driven Routing
Workflows are the engine; the graph is the steering wheel. Workflows must dynamically inspect graph edges to determine routing (e.g., "split" if `FEEDS_INTO` exists).

### 6. Semantic/OLTP Boundary
Identity and configuration live in the graph. Mutable financial state lives in OLTP tables. The UI joins these via `node_id`. Never embed amounts inside graph nodes.

### 7. The Security Mandate (Zero-Tolerance)
`RequireTenantOwnership` must be invoked on **every** resolved node during graph traversal. Cross-tenant edges are invisible by default. No exceptions.

### 8. Observability & Chargeback
Every generated query plan must be enriched with a complexity score and logged to `bo_governance_events` for tenant chargebacks.    