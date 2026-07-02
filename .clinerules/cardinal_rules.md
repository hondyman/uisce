### Rule 1: Config-Before-Code
**Compliance:** Strong. Allocation percentages, fee math (HWM/Hurdle), match logic tolerances, and report XPath mappings are all stored as metadata/DSL in the graph.
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 2 (Fees):** A developer hardcodes `if feeSchedule.Type == "PERFORMANCE"` in the Go workflow to decide *how* to evaluate the DSL. **Enforcement:** The workflow must *only* evaluate the AST string. The distinction between a management fee and a performance fee must purely be a property on the node that the DSL reads, or a UI rendering hint, never an `if` statement in the accrual engine.
*   **Feature 3 (TA):** A developer hardcodes the disposal method (FIFO/LIFO) in the Go lot selector. **Enforcement:** The disposal method must be a property on the `HOLDS_SHARES_IN` edge or the `investor` node, passed dynamically into the lot selector adapter.
*   **Feature 4 (Recon):** A developer hardcodes `0.01` as a tolerance in Go. **Enforcement:** Tolerance *must* be read from the `reconciliation_rule` node properties.

### Rule 2: Graph-First
**Compliance:** Strong. Every feature introduces a `catalog_node` or `catalog_edge` (`FEEDS_INTO`, `MAPS_TO_VALUE`) *before* execution logic. Topology dictates behavior.
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 1 (Allocations):** A developer queries a hardcoded list of "known feeder IDs" from a config file instead of traversing the `FEEDS_INTO` edges from the Master node. **Enforcement:** The `FeederResolver` must start at a given `nodeID` and do a graph traverse. It must never fetch feeders by tenant ID alone.

### Rule 3: No Package Cycles
**Compliance:** Strong. I explicitly carved out narrow domain ports (`FeederResolver`, `ReconMatcher`, `InvestorCostBasisQuerier`, `ReportRenderer`) to be injected into Temporal workflows.
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 5 (Reporting):** The `ReportRenderer` adapter imports `mdm/rules` to evaluate the DSL, which imports `mdm/something_else`, causing a cycle. **Enforcement:** The renderer must depend on the `domain.DSLEvaluator` interface, not the concrete `mdm/rules` package. The `mdm/rules` package is injected at the root main.go.

### Rule 4: Hot/Cold Watermark
**Compliance:** Mixed. I explicitly called out using `DataIntegrityManager.buildUnionSafeQuery` in Features 3.7, 4.5, and 5.5.
**⚠️ Fix Required (Feature 2):** I missed explicitly stating this for Feature 2 (Fee Accruals). When the DSL evaluator fetches `NAV` or `CAPITAL_BALANCE` for a historical HWM calculation, it must use `buildUnionSafeQuery`. If a developer writes a raw `SELECT balance FROM ibor.position WHERE date = ?`, they will break StarRocks/Iceberg seam.
**Enforcement:** The domain port `domain.GLBalanceResolver` (introduced in 5.5) must also be used by the Fee DSL engine. All historical balance lookups must flow through it.

### Rule 5: Graph-Driven Routing
**Compliance:** Strong. Workflows inspect the graph to decide what to do (e.g., "Does this entity have `FEEDS_INTO` edges? Let's split." "Does this template have `MAPS_TO_VALUE` edges? Let's render.").
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 1 (Allocations):** The trade workflow is hardcoded to *always* run the feeder split step, even for standalone funds. **Enforcement:** The workflow must dynamically check for the existence of outbound `FEEDS_INTO` edges. If none exist, the step is skipped. The graph dictates the routing; the workflow is just the engine.

### Rule 6: The Semantic/OLTP Boundary
**Compliance:** Strong. I strictly separated high-throughput data (journal entries, tax lots, recon break records, HWM snapshots) into OLTP tables with a `node_id` bridge.
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 3 (TA):** A developer decides to store the investor's *cost basis number* as a property on the `investor` graph node to "make UI queries easier." **Enforcement:** The graph holds *identity* and *semantic configuration* (tax jurisdiction, share class). The IBOR/OLTP tables hold the *mutable financial state* (shares, cost basis). The UI must join via `node_id`, not stuff numbers into the graph.
*   **Feature 4 (Recon):** A developer stores the `break_amount` directly in the `recon_break_instance` graph node properties. **Enforcement:** Break amounts belong in `accounting.recon_break`. The graph node only exists so the Studio UI can draw the break and link it to a resolution workflow. Financial data in OLTP; topology in Graph.

### Rule 7: The Security Mandate (Zero-Tolerance Law)
**Compliance:** Strong. I explicitly appended security steps to every feature group.
**⚠️ Violation Risks (Watch Out For):**
*   **Feature 1 (Allocations):** Tenant A queries their Master Fund. The system traverses the `FEEDS_INTO` edges and finds Feeder X. Feeder X was just migrated and belongs to Tenant B (assuming multi-tenant entity structures for fund admins). The system returns Tenant B's feeder node to Tenant A. **Enforcement:** When traversing edges, `RequireTenantOwnership` must be called on *every* resolved node, not just the starting node. Alternatively, the graph query itself must be strictly scoped to the requesting tenant ID so cross-tenant edges are invisible.
*   **Feature 5 (Reporting):** A developer builds the report download endpoint and only checks if the user has "Report Download" permissions, forgetting to check if the *specific* `report_output` belongs to their tenant. **Enforcement:** `RequireTenantOwnership(ctx, pool, requestingTenantID, reportOutputID, "report_output")` on every single fetch.

---
