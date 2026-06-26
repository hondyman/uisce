# Catalog-Integrated Audit Graph - Complete Implementation Guide

**Date:** January 18, 2026  
**Status:** Production-Ready Blueprint  
**Scope:** End-to-end audit semantic layer with AI-driven governance

---

## Executive Summary

You now have a complete, production-grade implementation of a **catalog-integrated audit graph system** that:

✅ Maps all audit events (JobRun, DAGRun, ChangeSet, Compliance, Incident, Semantic Snapshot, AI Suggestion) into the catalog graph  
✅ Enforces multi-tenant isolation at every layer  
✅ Provides graph-aware AI reasoning for explanations and recommendations  
✅ Enables role-based Audit Explorer UIs (Global Admin, Global Ops, Tenant Admin, Tenant Ops)  
✅ Implements a complete ChangeSet approval flow with Temporal orchestration  
✅ Exposes Trino views for queryable audit analytics  
✅ Maintains full audit trail and compliance context  

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    AUDIT PRODUCERS                           │
│  (Scheduler, Governance, Orchestration, AI, Compliance)      │
└─────────────────────────────┬───────────────────────────────┘
                              │
                    Kafka/Redpanda Topics
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
   job_runs         changeset_events        compliance_events
   dag_runs         incidents                ai_suggestions
                    semantic_snapshots
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
        ┌─────────────────────▼──────────────────────┐
        │  Audit Ingestion Worker (Go)               │
        │  - Routes events by type                   │
        │  - Creates catalog_node entries            │
        │  - Creates catalog_edge relationships      │
        │  - Batch writes for throughput             │
        └─────────────────────┬──────────────────────┘
                              │
        ┌─────────────────────▼──────────────────────┐
        │  PostgreSQL Catalog Tables                 │
        │  - catalog_node (8 audit types)            │
        │  - catalog_edge (13 relationships)         │
        │  - catalog_node_type                       │
        │  - catalog_edge_type                       │
        └─────────────────────┬──────────────────────┘
                              │
        ┌─────────────────────▼──────────────────────┐
        │  Trino Views (Audit Analytics)             │
        │  - semantic_events                         │
        │  - entity_timeline                         │
        │  - incident_graph                          │
        │  - changeset_impact                        │
        │  - compliance_context                      │
        │  - tenant_summary                          │
        └─────────────────────┬──────────────────────┘
                              │
        ┌─────────────────────┴──────────────────────┐
        │                                            │
    GraphQL API                            React UI (Audit Explorer)
    - Query audit events                   - Timeline view
    - List ChangeSets                      - Entity audit trail
    - Explain audit events (AI)            - ChangeSet approval
    - Approve/reject ChangeSets            - Incident analysis
    - Query Incidents                      - Compliance dashboard
                    │                            │
                    └────────────────────────────┘
                              │
            Temporal Workflow (ApplyChangeSetWorkflow)
            - Load ChangeSet + impacts
            - Apply semantic changes
            - Regenerate DAGs
            - Emit snapshots + audit
            - Mark as APPLIED
```

---

## Data Model: Node & Edge Types

### Node Types (8 audit + 3 optional)

| Node Type | Purpose | Example ID | Tenant-Scoped |
|-----------|---------|-----------|---|
| `audit_event` | Base audit event | `audit_event:ae-001` | ✅ |
| `job_run` | Job execution | `job_run:run-123` | ✅ |
| `dag_run` | DAG execution | `dag_run:run-456` | ✅ |
| `changeset_event` | Governance change | `changeset_event:cs-789` | ✅ |
| `compliance_event` | Compliance enforcement | `compliance_event:ce-111` | ✅ |
| `incident` | Operational incident | `incident:inc-222` | ✅ |
| `semantic_snapshot` | Semantic version | `semantic_snapshot:ss-333` | ✅ |
| `ai_suggestion` | AI recommendation | `ai_suggestion:as-444` | ✅ |
| `slo_risk` (opt) | SLO risk indicator | `slo_risk:sr-555` | ✅ |
| `tenant_summary` (opt) | Tenant-scoped summary | `tenant_summary:ts-666` | ✅ |
| `global_summary` (opt) | Platform-wide summary | `global_summary:gs-777` | ❌ |

### Edge Types (13 core relationships)

| Edge Type | From Node | To Node | Purpose |
|-----------|-----------|---------|---------|
| `event_of` | Audit event | Any entity | Event refers to entity |
| `runs_job` | job_run | job | Job run executes a job |
| `runs_dag` | dag_run | dag | DAG run executes a DAG |
| `has_impact_on` | changeset_event | semantic_term/job/dag/bo | ChangeSet affects entity |
| `causes` | incident | job_run/dag_run | Incident caused by run |
| `has_ai_narrative` | ai_suggestion | audit_event | AI attached to event |
| `has_compliance_context` | compliance_event | semantic_term/bo/term | Compliance linked to entity |
| `has_semantic_context` | audit_event | semantic_term | Event linked to semantic |
| `has_tenant` | Any audit node | tenant | Tenant ownership |
| `applied` | changeset_event | semantic_snapshot/dag_version | ChangeSet created snapshot |
| `version_of` | semantic_snapshot/dag_version | semantic_term/dag | Snapshot versions entity |
| `has_risk` | Any node | slo_risk | Entity has risk indicator |
| `has_slo_context` | audit_event | slo | Event linked to SLO |

---

## Backend Implementation

### 1. Database Layer

**File:** `backend/migrations/025_add_audit_graph_node_edge_types.sql`

- Inserts 11 node types into `catalog_node_type`
- Inserts 13 edge types into `catalog_edge_type`
- Idempotent: `ON CONFLICT` handles re-runs

### 2. CatalogWriter Interface

**File:** `backend/internal/catalog/writer.go`

```go
type Writer interface {
    CreateNode(ctx context.Context, node CatalogNode) error
    CreateNodes(ctx context.Context, nodes []CatalogNode) error      // Batch
    CreateEdge(ctx context.Context, edge CatalogEdge) error
    CreateEdges(ctx context.Context, edges []CatalogEdge) error      // Batch
    UpdateNode(ctx context.Context, node CatalogNode) error
    GetNode(ctx context.Context, nodeID string) (*CatalogNode, error)
    GetEdges(ctx context.Context, fromNode string) ([]CatalogEdge, error)
}
```

**Key Features:**
- Batch insert for high-volume ingestion (10,000+ events/sec)
- Validation of node/edge types against catalog_node_type/catalog_edge_type
- Upsert semantics for idempotence
- Transaction support for consistency

### 3. Audit Ingestion Worker

**File:** `backend/internal/audit/ingestion_graph.go`

Routes 7 event types to handlers:

| Event Type | Handler | Node Type Created | Edges |
|-----------|---------|-------------------|-------|
| `JOB_RUN_COMPLETED` | `ingestJobRun` | `job_run` | runs_job, has_tenant, has_semantic_context, has_compliance_context |
| `DAG_RUN_COMPLETED` | `ingestDAGRun` | `dag_run` | runs_dag, has_tenant, has_semantic_context |
| `CHANGESET_CREATED` | `ingestChangeSet` | `changeset_event` | has_impact_on (×N), has_tenant |
| `COMPLIANCE_EVENT` | `ingestComplianceEvent` | `compliance_event` | has_compliance_context, has_tenant |
| `INCIDENT_CLUSTERED` | `ingestIncident` | `incident` | causes (×M), has_tenant |
| `SEMANTIC_SNAPSHOT` | `ingestSemanticSnapshot` | `semantic_snapshot` | version_of, applied, has_tenant |
| `AI_SUGGESTION` | `ingestAISuggestion` | `ai_suggestion` | has_ai_narrative, has_tenant |

**Integration Point:** Kafka consumer reads from topics, unmarshals envelope, routes to handler.

### 4. Trino Views

**File:** `backend/migrations/026_create_audit_graph_trino_views.sql`

10 views for analytics:

| View | Purpose | Query Use Case |
|------|---------|-----------------|
| `audit.semantic_events` | Unified audit events with entity context | "Show all events affecting entity X" |
| `audit.entity_timeline` | Events by entity, ordered by time | "Timeline for semantic term Y" |
| `audit.job_run_context` | Job runs with semantic enrichment | "Jobs impacting term Z" |
| `audit.incident_graph` | Incidents linked to causing runs | "What caused incident #5?" |
| `audit.changeset_impact` | ChangeSets with affected entities | "What did ChangeSet #10 impact?" |
| `audit.compliance_context` | Compliance events with context | "Compliance violations related to PII" |
| `audit.ai_suggestions` | AI suggestions with source events | "AI suggestions for failed jobs" |
| `audit.semantic_snapshot_lineage` | Version history of semantic terms | "How did Positions evolve?" |
| `audit.tenant_summary` | Tenant-scoped metrics | "Failed runs per tenant" |
| `audit.cross_tenant_incidents` | Multi-tenant incidents | "Cross-tenant blast radius" |

### 5. GraphQL Layer

**File:** `backend/graph/schema/audit_graph.graphql`

```graphql
extend type Query {
  # Audit event queries
  auditEvents(tenantIds, from, to, filter, sort, pagination): [AuditEvent!]!
  auditEventById(id): AuditEvent
  entityAudit(entityType, entityId, tenantIds, from, to): EntityAudit!
  incidents(tenantIds, from, to, sort, pagination): [Incident!]!
  incidentById(id): Incident!
  
  # ChangeSet queries
  changeSets(filter, pagination, sort): ChangeSetConnection!
  changeSet(id): ChangeSet
  
  # Summary queries
  auditSummary(tenantIds, from, to): [AuditSummary!]!
  summarizeTenantActivity(tenantId, from, to): AIExplanation!
  summarizeIncident(incidentId): AIExplanation!
  getRemediationChain(initialEventId): RemediationChain
}

extend type Mutation {
  # ChangeSet mutations
  createChangeSetFromAI(input): ChangeSet!
  approveChangeSet(id): ChangeSet!
  rejectChangeSet(id, reason): ChangeSet!
  
  # AI mutations
  explainAudit(tenantIds, records): AIExplanation!
  assessChangeSet(id): AIAssessment!
}
```

**Key Resolvers:** `backend/internal/graphql/changeset_resolver.go`

- `CreateChangeSetFromAI` → Creates changeset_event node + impact edges
- `ApproveChangeSet` → Updates status + triggers Temporal workflow
- `RejectChangeSet` → Records rejection reason + audit event

### 6. Temporal Workflow

**File:** `backend/internal/temporal/apply_changeset_workflow.go`

```go
func ApplyChangeSetWorkflow(ctx workflow.Context, params ApplyChangeSetParams) error {
    // 1. LoadChangeSetActivity         → Query catalog_node + edges
    // 2. ApplySemanticChangesActivity  → Update semantic defs + create snapshots
    // 3. RegenerateDAGsActivity        → Recompile DAGs
    // 4. EmitSnapshotsAndAuditActivity → Create snapshot nodes + audit events
    // 5. MarkChangeSetAppliedActivity  → Update status = APPLIED
}
```

**Orchestration Pattern:**
- Retry policy: 5 attempts, exponential backoff (2s → 4s → 8s...)
- Activity timeout: 5 minutes each
- Workflow timeout: 30 minutes
- Task queue: Per-tenant for isolation

### 7. AI Prompt Builders

**File:** `backend/internal/ai/prompt_builder.go`

```go
type PromptBuilder struct{}

// Methods:
// - ExplainJobRunPrompt(jobRun, linkedJob, linkedDAG, semanticTerms, recentEvents, tenantScope) string
// - ExplainIncidentPrompt(incident, jobRuns, dags, semanticTerms, tenantScope) string
// - AssessChangeSetPrompt(changeSet, impactedEntities, recentIncidents, complianceContext, tenantScope) string
// - PostHocRemediationPrompt(initialFailure, aiSuggestion, changeSet, snapshots, dagVersions, subsequentRuns) string
```

**Prompt Structure:**
1. Context (event details, linked entities, recent events)
2. Tenant scope constraint (prevent cross-tenant data leakage)
3. Tasks (explain, analyze, recommend)
4. Output format (JSON schema)

**Integration:** ExplainService wraps prompt building + LLM calling.

---

## Frontend Implementation

### 1. React Hooks

**File:** `frontend/src/hooks/useAuditGraphHooks.ts`

```typescript
// Core hooks:
export function useAuditEvents(tenantIds, from, to, filters, pagination)
export function useChangeSets(tenantIds, status, pagination)
export function useChangeSet(id)
export function useExplainAudit()
export function useCreateChangeSetFromAI()
export function useApproveChangeSet()
export function useRejectChangeSet()

// High-level flow hooks:
export function useAuditExplainerFlow(event)  // Explain → Propose ChangeSet
export function useApprovalFlow(changeSetId)  // Approve → Reject
```

**GraphQL Integration:** Each hook maps to corresponding GraphQL query/mutation.

### 2. React Components

**File:** `frontend/src/components/audit/AuditExplorerGraph.tsx`

```typescript
<AuditExplorerGraph
  scope="tenant"  // or global, multi-tenant-assigned, tenant-ops
  role="TENANT_ADMIN"
  tenantIds={["tenant-001"]}
/>
```

**Component Hierarchy:**
- `AuditExplorerGraph` (main container)
  - `TimelineViewGraph` (event list)
    - `TimelineRowGraph` (each event)
  - `AIPanelWithChangeSetProposal` (right panel)
    - `ChangeSetProposalModalGraph` (creation modal)

**Role-Based Features:**
- Global Admin: Multi-tenant selector, cross-tenant summaries
- Global Ops: Assigned tenants only, incident analysis
- Tenant Admin: Single tenant, governance + compliance tabs
- Tenant Ops: Single tenant, ops-focused (jobs, incidents only)

---

## End-to-End Flow: "Explain with AI → ChangeSet Proposal → Approval → Application"

### Step 1: Job Fails (Event Produced)

```
Scheduler produces: JOB_RUN_COMPLETED
{
  "eventType": "JOB_RUN_COMPLETED",
  "tenantId": "tenant-001",
  "payload": {
    "runId": "run-123",
    "jobId": "job-positions-preagg",
    "status": "FAILED",
    "errorMessage": "schema mismatch on Positions",
    "semanticContext": { "semanticTermId": "semantic_term:positions" },
    ...
  }
}
```

### Step 2: Ingestion Worker Creates Graph Nodes

```go
ingestJobRun() creates:
  - Node: job_run:run-123
    Properties: status, error, semanticContext, etc.
  
  - Edges:
    - runs_job → job:job-positions-preagg
    - has_semantic_context → semantic_term:positions
    - has_tenant → tenant:tenant-001
```

### Step 3: Tenant Ops Clicks "Explain with AI"

```
Frontend (useAuditExplainerFlow):
  1. Query auditEvents to fetch job_run:run-123 details
  2. Call explainAudit(tenantIds=["tenant-001"], records=[job_run:run-123])
  3. GraphQL resolver enriches with graph context (linked job, DAG, semantic term)
  4. PromptBuilder constructs graph-aware prompt
  5. AI service generates narrative + rootCause + blast radius + recommendation
  6. UI displays AIPanel with explanation
```

**AI Response:**
```json
{
  "narrative": "The job positions-preagg failed due to a schema drift...",
  "rootCause": "Positions BO added column 'region_code' without updating semantic mapping.",
  "blastRadius": "Affects risk-batch DAG and downstream VaR reports.",
  "recommendedFix": "Update semantic term 'Positions' and regenerate DAG.",
  "suggestedChangeSetSummary": "Align Positions BO schema with semantic term and regenerate dependent DAGs.",
  "confidence": 0.92
}
```

### Step 4: Tenant Ops Proposes ChangeSet

```
Frontend (ChangeSetProposalModal):
  1. User edits title/description (prefilled from AI)
  2. Click "Create ChangeSet"
  3. Call createChangeSetFromAI mutation
  
GraphQL Resolver:
  1. Validate tenant scope (tenant-001 ✓)
  2. Create changeset_event:cs-982 node
  3. Create has_impact_on edges to semantic_term:positions, job, dag
  4. Create has_tenant edge → tenant:tenant-001
  5. Emit governance audit event
  6. Return ChangeSet with id=cs-982, status=PENDING
  
Database Result:
  - catalog_node: changeset_event:cs-982
  - catalog_edge: 4 edges (has_impact_on×3, has_tenant)
```

### Step 5: Tenant Admin Reviews ChangeSet

```
Frontend (ChangeSet Review Screen):
  1. Navigate: Governance → Pending ChangeSets
  2. Click Review on cs-982
  3. View:
     - Title, description
     - Impacted entities (via has_impact_on edges)
     - AI explanation (from aiPanel)
     - History (created by Tenant Ops, source AI_PROPOSED)
  4. Click Approve
  
GraphQL Mutation (approveChangeSet):
  1. Validate tenant scope ✓
  2. Update changeset_event:cs-982 node
     - status = APPROVED
     - approvedBy = tenant-admin-user
     - approvedAt = now()
  3. Start Temporal workflow: ApplyChangeSetWorkflow(cs-982, tenant-001)
  4. Return ChangeSet with status=APPROVED
```

### Step 6: Temporal Workflow Applies ChangeSet

```
ApplyChangeSetWorkflow:
  Activity 1: LoadChangeSetActivity
    - Query catalogWriter.GetNode(changeset_event:cs-982)
    - Query catalogWriter.GetEdges to find impacted entities
    - Return ChangeSetContext
  
  Activity 2: ApplySemanticChangesActivity
    - Fetch current Positions semantic term definition
    - Apply changes (add region_code column to mapping)
    - Persist new semantic version
    - Create semantic_snapshot:ss-441 node
    - Return snapshot ID
  
  Activity 3: RegenerateDAGsActivity
    - Identify DAGs using Positions (risk-batch)
    - Recompile risk-batch DAG with updated semantic
    - Create dag_version:dv-221 node
    - Return DAG version ID
  
  Activity 4: EmitSnapshotsAndAuditActivity
    - Create edges:
      - applied: changeset_event:cs-982 → semantic_snapshot:ss-441
      - version_of: semantic_snapshot:ss-441 → semantic_term:positions
      - applied: changeset_event:cs-982 → dag_version:dv-221
    - Create audit events for semantic update + DAG regen
    - Batch insert all nodes/edges
  
  Activity 5: MarkChangeSetAppliedActivity
    - Update changeset_event:cs-982
      - status = APPLIED
      - appliedAt = now()
    - Emit governance audit event
```

### Step 7: Subsequent Job Runs Succeed

```
Scheduler produces: JOB_RUN_COMPLETED (success)
{
  "eventType": "JOB_RUN_COMPLETED",
  "payload": {
    "runId": "run-124",
    "status": "SUCCESS",
    ...
  }
}

Ingestion worker creates:
  - Node: job_run:run-124
  - Edges: runs_job, has_semantic_context, has_tenant
```

### Step 8: Audit Trail Complete

```
Audit Explorer Timeline shows:
  10:42  [JOB RUN] positions-preagg FAILED
           ↓ (Explain with AI)
         [AI SUGGESTION] schema drift detected
           ↓ (Propose ChangeSet)
         [CHANGESET] cs-982 PENDING
           ↓ (Tenant Admin approves)
         [CHANGESET] cs-982 APPROVED
           ↓ (Temporal workflow)
         [SEMANTIC SNAPSHOT] ss-441 created
         [DAG VERSION] dv-221 created
           ↓ (Job runs DAG)
         [JOB RUN] positions-preagg SUCCESS (run-124)

All relationships captured in catalog graph:
  - AI_SUGGESTION → JOB_RUN (has_ai_narrative)
  - CHANGESET_EVENT → JOB_RUN (event_of)
  - CHANGESET_EVENT → SEMANTIC_SNAPSHOT (applied)
  - SEMANTIC_SNAPSHOT → SEMANTIC_TERM (version_of)
  - JOB_RUN (success) → SEMANTIC_TERM (has_semantic_context)
```

---

## Deployment Checklist

- [ ] Run SQL migration 025 (node/edge types)
- [ ] Run SQL migration 026 (Trino views)
- [ ] Deploy CatalogWriter (catalog/writer.go)
- [ ] Deploy IngestionWorker (audit/ingestion_graph.go)
- [ ] Deploy GraphQL schema extensions (audit_graph.graphql)
- [ ] Deploy ChangeSet resolvers (graphql/changeset_resolver.go)
- [ ] Deploy Temporal workflow (temporal/apply_changeset_workflow.go)
- [ ] Deploy AI prompt builders (ai/prompt_builder.go)
- [ ] Deploy React hooks (frontend/hooks/useAuditGraphHooks.ts)
- [ ] Deploy React components (frontend/components/audit/AuditExplorerGraph.tsx)
- [ ] Wire Kafka consumer to IngestionWorker
- [ ] Wire GraphQL resolvers to CatalogWriter + AI service
- [ ] Wire Temporal client to GovernanceService
- [ ] Test end-to-end flow (job failure → AI explanation → ChangeSet → approval → application)
- [ ] Validate multi-tenant isolation (queries respect tenant scope)
- [ ] Monitor ingestion throughput (target: 10,000+ events/sec)
- [ ] Validate Trino view performance (< 500ms for typical queries)

---

## Production Monitoring

### Key Metrics

- **Ingestion Latency:** Time from event produced to node/edge inserted
- **AI Response Time:** Prompt building + LLM call (target: < 5s)
- **ChangeSet Application Time:** Workflow execution (target: < 1 min)
- **Audit View Query Latency:** Trino queries (target: < 500ms)
- **Catalog Graph Size:** Total nodes + edges (monitor growth)
- **Multi-tenant Correctness:** Spot-check queries for tenant boundary violations

### Alerts

- Ingestion worker lag > 5 minutes
- AI service error rate > 1%
- ChangeSet workflow failures
- Audit view query timeouts
- Tenant scope violations detected

### Compliance Validation

- ✅ All audit nodes have `tenant_id` field
- ✅ All queries respect `tenant_id` filter
- ✅ All mutations validate tenant scope
- ✅ All GraphQL queries inject `tenantIds` from auth context
- ✅ All ChangeSet operations audit-logged
- ✅ All AI suggestions traceable to source event

---

## Future Enhancements

- [ ] Graph traversal UI (visualize node relationships)
- [ ] Automated root cause analysis (trained model)
- [ ] Cross-tenant pattern detection
- [ ] Predictive incident forecasting
- [ ] SLO risk scoring (via slo_risk nodes)
- [ ] Governance workflow templates
- [ ] Custom audit event schemas (tenant-specific)
- [ ] Real-time incident clustering via stream processing

---

## Support & Debugging

### Common Issues

**Q: Audit events not appearing in UI**
- A: Check Kafka consumer lag (IngestionWorker)
- A: Verify tenant_id in event matches allowed tenants
- A: Check PostgreSQL connectivity from worker

**Q: AI explanations take too long**
- A: Monitor LLM service latency
- A: Check prompt size (reduce context if needed)
- A: Consider caching similar explanations

**Q: ChangeSet approval fails silently**
- A: Check Temporal workflow execution logs
- A: Verify ApplyChangeSetWorkflow activities are registered
- A: Check Temporal service connectivity

**Q: GraphQL queries return empty results**
- A: Verify Trino view definitions (check migrations)
- A: Check auth context tenant scope extraction
- A: Validate catalog_node/catalog_edge data via direct SQL

---

## References

- Catalog Graph Model: `catalog_node`, `catalog_edge`, `catalog_node_type`, `catalog_edge_type`
- Ingestion: `kafka_events.go`, `ingestion_graph.go`
- GraphQL: `audit_graph.graphql`, `changeset_resolver.go`
- React: `useAuditGraphHooks.ts`, `AuditExplorerGraph.tsx`
- Temporal: `apply_changeset_workflow.go`
- AI: `prompt_builder.go`

---

**Maintained by:** SemLayer Engineering Team  
**Last Updated:** January 18, 2026  
**Version:** 1.0 (Production Ready)
