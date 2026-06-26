# Audit & Snapshot Plane - System Architecture

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         SEMLAYER PLATFORM SERVICES                          │
├─────────────────┬─────────────────┬─────────────────┬─────────────────────┤
│   Scheduler     │   Governance    │   Compliance    │   Semantic Engine   │
│   Engine        │   Service       │   Engine        │                     │
│                 │                 │                 │                     │
│ • Job Runs      │ • Policy Changes│ • Violations    │ • Term Updates      │
│ • DAG Execution │ • Access Reviews│ • Detections    │ • Schema Evolution  │
│ • Task Failures │ • Approvals     │ • Remediation   │ • Lineage Changes   │
└────────┬────────┴────────┬────────┴────────┬────────┴──────────┬──────────┘
         │                 │                 │                   │
         │ Publish Events  │                 │                   │
         ▼                 ▼                 ▼                   ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    KAFKA (REDPANDA) - Event Streaming Bus                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  Topics (6 partitions each, partitioned by tenant_id):                      │
│                                                                              │
│  • audit.scheduler.job_runs        • audit.scheduler.dag_runs               │
│  • audit.governance.changesets     • audit.semantic.snapshots               │
│  • audit.orchestration.events      • audit.compliance.violations            │
│  • audit.ai.suggestions            • audit.tenant.retention                 │
│                                                                              │
│  Retention: 7 days (then archived to Iceberg)                               │
│  Replication: 3x for durability                                             │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Consume
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      AUDIT SINK CONSUMER (Go Service)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Subscribes to all audit.* topics                                         │
│  • Routes events by topic to appropriate handlers                           │
│  • Converts JSON events to Parquet files                                    │
│  • Writes to MinIO S3 with Hive partitioning                                │
│  • Batch size: 100 records or 30 seconds                                    │
│  • Compression: ZSTD (3:1 ratio)                                            │
│  • Consumer Group: audit-sink-group                                         │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Write Parquet
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      MINIO (S3-Compatible Object Storage)                   │
├─────────────────────────────────────────────────────────────────────────────┤
│  Bucket: warehouse/audit/                                                   │
│                                                                              │
│  Structure (Hive Partitioning):                                             │
│  warehouse/audit/scheduler_job_runs/                                        │
│    ├── tenant_id=tenant-001/                                                │
│    │   ├── day=2026-01-15/                                                  │
│    │   │   ├── 00000000.parquet (ZSTD compressed)                           │
│    │   │   └── 00000001.parquet                                             │
│    │   └── day=2026-01-16/                                                  │
│    │       └── 00000000.parquet                                             │
│    └── tenant_id=tenant-002/                                                │
│        └── day=2026-01-15/                                                  │
│            └── 00000000.parquet                                             │
│                                                                              │
│  Total Storage: ~50GB per year per 100K events/day                          │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Register Metadata
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                     ICEBERG REST CATALOG (Metadata Service)                 │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Manages table metadata (schemas, partitions, snapshots)                  │
│  • Tracks Parquet file locations                                            │
│  • Provides transaction coordination                                        │
│  • Enables time travel (snapshot history)                                   │
│  • Schema evolution support                                                 │
│                                                                              │
│  Tables Managed:                                                            │
│  • iceberg.audit.scheduler_job_runs                                         │
│  • iceberg.audit.scheduler_dag_runs                                         │
│  • iceberg.audit.governance_changesets                                      │
│  • iceberg.audit.semantic_snapshots                                         │
│  • iceberg.audit.orchestration_events                                       │
│  • iceberg.audit.ai_suggestions                                             │
│  • iceberg.audit.compliance_violations                                      │
│  • iceberg.audit.tenant_retention_policies                                  │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Query Metadata
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                    TRINO (Distributed SQL Query Engine)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Coordinator: localhost:8090                                                │
│                                                                              │
│  Catalogs:                                                                  │
│  • iceberg (Iceberg REST connector)                                         │
│  • postgres (for real-time data)                                            │
│                                                                              │
│  Query Features:                                                            │
│  • Partition pruning (tenant_id, day)                                       │
│  • Columnar reading (Parquet)                                               │
│  • Predicate pushdown                                                       │
│  • Distributed aggregations                                                 │
│  • Time travel queries                                                      │
│                                                                              │
│  Materialized Views (Pre-aggregated):                                       │
│  • mv_tenant_scheduler_slo                                                  │
│  • mv_tenant_compliance_violations                                          │
│  • mv_tenant_governance_activity                                            │
│  • mv_semantic_drift_detection                                              │
│  • mv_platform_health                                                       │
│  • mv_tenant_audit_summary                                                  │
│  • mv_recent_job_runs                                                       │
│  • mv_recent_violations                                                     │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Query via SQL
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      TRINO AUDIT QUERIER (Go Service)                       │
├─────────────────────────────────────────────────────────────────────────────┤
│  • Connects to Trino via Trino Go driver                                    │
│  • Enforces multi-tenant scoping (MANDATORY tenant_id)                      │
│  • Provides typed query methods:                                            │
│    - QueryJobRuns(tenantID, filters)                                        │
│    - QueryChangeSets(tenantID, filters)                                     │
│    - QueryComplianceViolations(tenantID, filters)                           │
│    - QuerySemanticLineage(tenantID, termID)                                 │
│  • Returns structured Go types                                              │
│  • Connection pooling (max 10 connections)                                  │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ Serve Data
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      AUDIT API HANDLER (REST Endpoints)                     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Base Path: /api/audit                                                      │
│                                                                              │
│  Endpoints:                                                                 │
│  • GET /job-runs                - List job executions                       │
│  • GET /job-runs/:run_id        - Get job details                           │
│  • GET /dag-runs                - List DAG executions                       │
│  • GET /changesets              - List governance changes                   │
│  • GET /changesets/:id          - Get changeset details                     │
│  • GET /violations              - List compliance violations                │
│  • GET /violations/:id          - Get violation details                     │
│  • GET /semantic/:id/lineage    - Get semantic lineage                      │
│  • GET /semantic/:id/versions   - Get version history                       │
│  • GET /ai-narratives           - Get AI explanations                       │
│  • GET /dashboard/slo           - SLO dashboard data                        │
│  • GET /dashboard/compliance    - Compliance dashboard data                 │
│  • GET /dashboard/governance    - Governance dashboard data                 │
│                                                                              │
│  Middleware:                                                                │
│  • Tenant Scope Middleware (enforces X-Tenant-ID header)                    │
│  • JWT Auth (from parent router)                                            │
│  • CORS (configured in parent)                                              │
│  • Rate Limiting (100 req/min per tenant)                                   │
└──────────────────────────────────────┬──────────────────────────────────────┘
                                       │
                                       │ HTTP/JSON
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        AUDIT EXPLORER (React UI)                            │
├─────────────────────────────────────────────────────────────────────────────┤
│  Route: /audit                                                              │
│                                                                              │
│  Tabs:                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 1. JOB RUNS                                                         │   │
│  │    • Filter by job_id, status, date range                           │   │
│  │    • Table: run_id, job, status, duration, timestamp                │   │
│  │    • Detail panel with full context                                 │   │
│  │    • AI Explain button (generates narrative)                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 2. VIOLATIONS                                                       │   │
│  │    • Filter by severity, type, date range                           │   │
│  │    • Table: violation_id, type, severity, resource, timestamp       │   │
│  │    • Detail panel with compliance context                           │   │
│  │    • AI Explain button (root cause analysis)                        │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 3. CHANGESETS                                                       │   │
│  │    • Filter by change_type, actor, date range                       │   │
│  │    • Table: changeset_id, type, resource, actor, timestamp          │   │
│  │    • Detail panel with before/after diff                            │   │
│  │    • AI Explain button (impact analysis)                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │ 4. DASHBOARDS                                                       │   │
│  │    • SLO Metrics (success rate, P95 latency)                        │   │
│  │    • Compliance Trends (violations by severity over time)           │   │
│  │    • Governance Activity (changes by type)                          │   │
│  │    • Platform Health (job success rate)                             │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Features:                                                                  │
│  • Dark mode support                                                        │
│  • Tenant scoping (reads from localStorage)                                 │
│  • Real-time updates (polling every 30s)                                    │
│  • Export to CSV                                                            │
│  • Pagination (50 records per page)                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Data Flow Example: Scheduler Job Completion

```
1. Scheduler completes job
   └─> scheduler-engine/scheduler.go
       └─> calls auditPublisher.PublishJobRun(ctx, record)

2. Event published to Kafka
   └─> Topic: audit.scheduler.job_runs
       Partition: hash(tenant_id) % 6
       Message: {
         "run_id": "run-123",
         "tenant_id": "tenant-001",
         "job_id": "etl-daily",
         "status": "SUCCESS",
         "start_ts": "2026-01-17T10:00:00Z",
         "end_ts": "2026-01-17T10:15:00Z",
         "duration_ms": 900000,
         ...
       }

3. Audit Sink consumes event
   └─> audit-sink/main.go
       └─> IcebergSinkConsumer.consumeMessages()
           └─> Routes to handleJobRun()
               └─> Writes to Parquet buffer

4. Buffer flush (every 100 records or 30s)
   └─> IcebergWriter.WriteBatch()
       └─> Creates Parquet file
           └─> Uploads to MinIO: 
               s3://warehouse/audit/scheduler_job_runs/
                 tenant_id=tenant-001/day=2026-01-17/00000000.parquet

5. Iceberg REST Catalog updated
   └─> Registers new Parquet file in metadata
       └─> Updates table snapshot
           └─> Trino can now query the new data

6. User queries via UI
   └─> GET /api/audit/job-runs?tenant_id=tenant-001&limit=50
       └─> TrinoAuditQuerier.QueryJobRuns()
           └─> SQL: SELECT * FROM iceberg.audit.scheduler_job_runs 
                    WHERE tenant_id = 'tenant-001' 
                    ORDER BY start_ts DESC LIMIT 50
               └─> Trino:
                   • Reads Iceberg metadata
                   • Prunes partitions (tenant_id=tenant-001, day=2026-01-17)
                   • Scans Parquet file
                   • Returns results
                       └─> JSON response to UI
                           └─> Renders in Job Runs table
```

## Tenant Isolation

```
Multi-Tenant Scoping at Every Layer:

┌────────────────────────────────────────────────────────────────┐
│ Layer 1: UI (Frontend)                                         │
│ • Reads tenant from localStorage (selected_tenant)             │
│ • Adds X-Tenant-ID header to all API calls                     │
│ • Filters displayed data by tenant                             │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Layer 2: API (Backend)                                         │
│ • TenantScopeMiddleware validates X-Tenant-ID header           │
│ • Rejects requests without valid tenant                        │
│ • Passes tenant_id to query service                            │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Layer 3: Query Service (TrinoAuditQuerier)                     │
│ • REQUIRES tenant_id parameter                                 │
│ • Returns error if tenant_id is empty                          │
│ • Adds tenant_id to WHERE clause (mandatory)                   │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Layer 4: Trino (Query Engine)                                  │
│ • Partition pruning by tenant_id                               │
│ • Only scans Parquet files for specified tenant                │
│ • Cannot query across tenants                                  │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Layer 5: Storage (MinIO + Iceberg)                             │
│ • Hive partitioning: tenant_id=<value>/day=<date>/             │
│ • Physically separate Parquet files per tenant                 │
│ • Iceberg metadata tracks partition boundaries                 │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ Layer 6: Kafka (Event Bus)                                     │
│ • Partition key: tenant_id                                     │
│ • Ensures events for same tenant go to same partition          │
│ • Consumer can process tenant data in order                    │
└────────────────────────────────────────────────────────────────┘

Result: Complete isolation at every layer - no cross-tenant data leakage!
```

## Service Endpoints

```
┌─────────────────────────────────────────────────────────────────┐
│ SERVICE              │ PORT  │ URL                              │
├─────────────────────────────────────────────────────────────────┤
│ Backend API          │ 8080  │ http://localhost:8080            │
│ Frontend UI          │ 3000  │ http://localhost:3000            │
│ Redpanda (Kafka)     │ 19092 │ localhost:19092                  │
│ Redpanda Console     │ 8080  │ http://localhost:8080            │
│ Trino Coordinator    │ 8090  │ http://localhost:8090            │
│ Trino UI             │ 8090  │ http://localhost:8090/ui         │
│ Iceberg REST Catalog │ 8181  │ http://localhost:8181            │
│ MinIO API            │ 9000  │ http://localhost:9000            │
│ MinIO Console        │ 9001  │ http://localhost:9001            │
└─────────────────────────────────────────────────────────────────┘
```

## File Locations

```
semlayer/
├── backend/
│   ├── config.yaml                          # Added audit config section
│   ├── internal/
│   │   ├── api/
│   │   │   └── api.go                       # Wired up /api/audit/* routes
│   │   └── audit/
│   │       ├── models.go                    # 8 audit record types
│   │       ├── kafka_events.go              # Event schemas
│   │       ├── kafka_publisher.go           # Kafka publisher
│   │       ├── iceberg_schema.sql           # 8 Iceberg tables
│   │       ├── materialized_views.sql       # 8 dashboard views
│   │       ├── iceberg_sink.go              # Kafka→Parquet consumer
│   │       ├── trino_querier.go             # Trino query service
│   │       ├── api.go                       # 13 REST endpoints
│   │       ├── ai_narrative_service.go      # AI explanations
│   │       ├── compliance_reporter.go       # Compliance reports
│   │       ├── integration.go               # Helper functions
│   │       ├── README.md                    # Architecture docs
│   │       └── INTEGRATION.md               # Integration guide
│   ├── cmd/
│   │   └── audit-sink/
│   │       └── main.go                      # Consumer entry point
│   ├── scripts/
│   │   └── test-audit-event.go              # Test event publisher
│   └── audit-infrastructure/
│       ├── docker-compose.yml               # 7 services
│       ├── start.sh                         # Deployment script
│       ├── test.sh                          # Testing script
│       ├── stop.sh                          # Shutdown script
│       ├── Dockerfile.audit-sink            # Consumer image
│       ├── README.md                        # Quick start
│       └── trino/
│           ├── catalog/iceberg.properties   # Iceberg connector
│           └── config.properties            # Trino config
├── frontend/
│   └── src/
│       ├── App.tsx                          # (no changes)
│       ├── AppRoutes.tsx                    # Added /audit route
│       ├── components/
│       │   ├── MainNavigation.tsx           # Added Audit Plane link
│       │   └── audit/
│       │       └── AuditExplorer.tsx        # Full audit UI
├── AUDIT_PLANE_QUICKSTART.md               # Quick start guide
└── AUDIT_PLANE_COMPLETE.md                 # Complete summary
```

---

**System Status**: ✅ **FULLY INTEGRATED AND READY FOR DEPLOYMENT**

Next Step: `cd backend/audit-infrastructure && ./start.sh`
