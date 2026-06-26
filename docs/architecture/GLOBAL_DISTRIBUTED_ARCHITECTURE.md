# Semlayer: Global Distributed Architecture

## Overview

This document describes the architecture for running Semlayer as a globally distributed platform using:
- **Temporal** - Unified scheduler and workflow orchestration
- **Azure Cosmos DB for PostgreSQL (Citus)** - Distributed metadata store
- **PostgreSQL Full-Text Search** - No Elasticsearch needed

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                            Global Control Plane                                  │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐             │
│  │    Temporal     │    │  Azure Front    │    │   Azure AD      │             │
│  │    (Schedules   │    │     Door        │    │   (Identity)    │             │
│  │   & Workflows)  │    │   (Global LB)   │    │                 │             │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘             │
└───────────┼──────────────────────┼──────────────────────┼──────────────────────┘
            │                      │                      │
┌───────────┼──────────────────────┼──────────────────────┼──────────────────────┐
│           │                      │                      │                      │
│  ┌────────▼────────┐    ┌────────▼────────┐    ┌───────▼────────┐             │
│  │    US East      │    │    EU West      │    │   Asia Pacific  │             │
│  │    Region       │    │    Region       │    │     Region      │             │
│  │                 │    │                 │    │                 │             │
│  │ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │             │
│  │ │  AKS        │ │    │ │  AKS        │ │    │ │  AKS        │ │             │
│  │ │  Cluster    │ │    │ │  Cluster    │ │    │ │  Cluster    │ │             │
│  │ │             │ │    │ │             │ │    │ │             │ │             │
│  │ │ - API GW    │ │    │ │ - API GW    │ │    │ │ - API GW    │ │             │
│  │ │ - Semantic  │ │    │ │ - Semantic  │ │    │ │ - Semantic  │ │             │
│  │ │ - Rules     │ │    │ │ - Rules     │ │    │ │ - Rules     │ │             │
│  │ │ - Cube.js   │ │    │ │ - Cube.js   │ │    │ │ - Cube.js   │ │             │
│  │ └──────┬──────┘ │    │ └──────┬──────┘ │    │ └──────┬──────┘ │             │
│  │        │        │    │        │        │    │        │        │             │
│  │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │    │ ┌──────▼──────┐ │             │
│  │ │ Redis       │ │    │ │ Redis       │ │    │ │ Redis       │ │             │
│  │ │ (Cache)     │ │    │ │ (Cache)     │ │    │ │ (Cache)     │ │             │
│  │ └─────────────┘ │    │ └─────────────┘ │    │ └─────────────┘ │             │
│  └────────┬────────┘    └────────┬────────┘    └────────┬────────┘             │
│           │                      │                      │                      │
└───────────┼──────────────────────┼──────────────────────┼──────────────────────┘
            │                      │                      │
            └──────────────────────┼──────────────────────┘
                                   │
                    ┌──────────────▼──────────────┐
                    │  Azure Cosmos DB for        │
                    │  PostgreSQL (Citus)         │
                    │                             │
                    │  ┌───────┐ ┌───────┐       │
                    │  │Coord. │ │Coord. │       │
                    │  │ Node  │ │ Node  │       │
                    │  └───┬───┘ └───┬───┘       │
                    │      │         │           │
                    │  ┌───▼─────────▼───┐       │
                    │  │  Worker Nodes   │       │
                    │  │  (Shards by     │       │
                    │  │   tenant_id)    │       │
                    │  └─────────────────┘       │
                    └─────────────────────────────┘
```

## 1. Scheduling: Temporal as Single Scheduler

### Why Temporal for Everything

| Requirement | Temporal Solution |
|-------------|-------------------|
| Cron jobs | `ScheduleClient` with cron expressions |
| Distributed | Runs across regions, deduplication built-in |
| Retries | Configurable retry policies |
| Visibility | Web UI, history, search |
| Multi-language | Your global scheduler works with Go, Python, TS |
| Workflows | Complex multi-step orchestration |
| Exactly-once | Guaranteed by Temporal's design |

### Migration from K8s CronJobs

**Before (K8s CronJobs):**
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: preagg-builder
spec:
  schedule: "0 */4 * * *"
  jobTemplate:
    spec:
      containers:
        - command: ["/app/jobs", "preagg-build"]
```

**After (Temporal Schedule):**
```go
client.ScheduleClient().Create(ctx, client.ScheduleOptions{
    ID: "preagg-builder",
    Spec: client.ScheduleSpec{
        CronExpressions: []string{"0 */4 * * *"},
    },
    Action: &client.ScheduleWorkflowAction{
        Workflow:  PreAggBuildWorkflow,
        TaskQueue: "analytics-tasks",
    },
})
```

### Benefits of Consolidation

1. **Single Pane of Glass** - All schedules visible in Temporal Web UI
2. **No More Missed Jobs** - Temporal handles catchup automatically
3. **Better Debugging** - Full execution history
4. **Global Deduplication** - Won't run same job twice across regions
5. **Language Agnostic** - Your "multi-lingual global scheduler" works perfectly

### All Semlayer Scheduled Jobs

| Category | Jobs | Frequency |
|----------|------|-----------|
| Analytics | Pre-agg build, refresh stale | 4h, 30m |
| Cache | Warm, invalidation sweep | 15m, 1h |
| Catalog | Full sync, incremental sync | Daily, 30m |
| Search | Index optimize, rebuild | Daily, Weekly |
| Compliance | Validation, policy sweep | Weekly, 6h |
| Reports | Daily usage, SLO, billing | Daily, Weekly, Monthly |
| Cleanup | Audit archival, soft-delete purge | Monthly, Weekly |
| Health | System check, metrics aggregation | 5m, 10m |
| Replication | Cross-region metadata sync | 5m |

## 2. Database: Azure Cosmos DB for PostgreSQL (Citus)

### Why Cosmos DB for PostgreSQL

| Feature | Benefit for Semlayer |
|---------|---------------------|
| **Horizontal scaling** | Add nodes as tenants grow |
| **Tenant colocation** | All tenant data on same shard |
| **Reference tables** | Small tables replicated everywhere |
| **PostgreSQL compatible** | Existing code works |
| **Global distribution** | Multi-region with Azure |
| **Managed service** | No Citus cluster management |

### Sharding Strategy

```sql
-- Reference tables (small, replicated to all nodes)
SELECT create_reference_table('tenants');
SELECT create_reference_table('compliance_frameworks');
SELECT create_reference_table('object_types');

-- Distributed tables (sharded by tenant_id)
SELECT create_distributed_table('datasources', 'tenant_id');
SELECT create_distributed_table('semantic_objects', 'tenant_id', 
    colocate_with => 'datasources');
SELECT create_distributed_table('bundles', 'tenant_id', 
    colocate_with => 'datasources');
SELECT create_distributed_table('policies', 'tenant_id', 
    colocate_with => 'datasources');
SELECT create_distributed_table('audit_logs', 'tenant_id', 
    colocate_with => 'datasources');
```

### Key Concepts

**Reference Tables:**
- Small tables (tenants, lookup data)
- Replicated to every node
- Always available for joins
- No shard key needed

**Distributed Tables:**
- Large tables (semantic objects, audit logs)
- Sharded by `tenant_id`
- Colocated = same tenant data on same node
- Enables efficient joins within tenant

**Coordinator Node:**
- Receives all queries
- Routes to appropriate shards
- Aggregates results
- Handles distributed transactions

### Query Patterns

**✅ Efficient (Single-shard):**
```sql
-- Always include tenant_id in WHERE clause
SELECT * FROM semantic_objects 
WHERE tenant_id = $1 AND datasource_id = $2;

-- Joins between colocated tables
SELECT so.*, b.name as bundle_name
FROM semantic_objects so
JOIN bundle_items bi ON so.id = bi.semantic_object_id 
    AND so.tenant_id = bi.tenant_id
JOIN bundles b ON bi.bundle_id = b.id 
    AND bi.tenant_id = b.tenant_id
WHERE so.tenant_id = $1;
```

**⚠️ Less Efficient (Cross-shard):**
```sql
-- Avoid: queries without tenant_id hit all shards
SELECT COUNT(*) FROM semantic_objects WHERE object_type = 'measure';

-- Better: include tenant filter
SELECT COUNT(*) FROM semantic_objects 
WHERE tenant_id = $1 AND object_type = 'measure';
```

### Multi-Region Setup

```
Primary Write Region: US East
├── Coordinator Node (Primary)
├── Worker Node 1 (Shards 1-100)
├── Worker Node 2 (Shards 101-200)
└── Worker Node 3 (Shards 201-300)

Read Replica: EU West  
├── Coordinator Node (Read-only)
├── Worker Node 1 (Shards 1-100 replica)
├── Worker Node 2 (Shards 101-200 replica)
└── Worker Node 3 (Shards 201-300 replica)

Read Replica: Asia Pacific
├── (Same structure)
```

### Connection Configuration

```go
// backend/config/database.go
type DatabaseConfig struct {
    // Primary (writes)
    PrimaryHost     string `env:"DB_PRIMARY_HOST"`
    PrimaryPort     int    `env:"DB_PRIMARY_PORT" envDefault:"5432"`
    
    // Read replicas (reads)
    ReadReplicaHosts []string `env:"DB_READ_REPLICA_HOSTS" envSeparator:","`
    
    // Common settings
    Database        string `env:"DB_NAME" envDefault:"semlayer"`
    Username        string `env:"DB_USER"`
    Password        string `env:"DB_PASSWORD"`
    SSLMode         string `env:"DB_SSL_MODE" envDefault:"require"`
    MaxConnections  int    `env:"DB_MAX_CONNECTIONS" envDefault:"100"`
    
    // Citus-specific
    UseColocatedJoins bool `env:"DB_USE_COLOCATED_JOINS" envDefault:"true"`
}

// Connection string for primary (writes)
func (c *DatabaseConfig) PrimaryDSN() string {
    return fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        c.PrimaryHost, c.PrimaryPort, c.Database, c.Username, c.Password, c.SSLMode,
    )
}

// Connection string for read replica (region-aware)
func (c *DatabaseConfig) ReadReplicaDSN(region string) string {
    // Select closest replica based on region
    host := c.selectReplicaForRegion(region)
    return fmt.Sprintf(
        "host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
        host, c.PrimaryPort, c.Database, c.Username, c.Password, c.SSLMode,
    )
}
```

## 3. Search: PostgreSQL Native (No Elasticsearch)

With Cosmos DB for PostgreSQL, full-text search still works:

```sql
-- Search is distributed - runs on each shard in parallel
SELECT * FROM search_tenant_objects(
    p_tenant_id := '...'::UUID,
    p_query := 'customer revenue',
    p_limit := 50
);
```

The function runs on the coordinator, which:
1. Routes query to correct shard (by tenant_id)
2. Executes search on that single node
3. Returns results directly (no cross-shard aggregation needed)

## 4. Implementation Files

| File | Purpose |
|------|---------|
| `backend/internal/scheduler/temporal_schedules.go` | All Temporal schedule definitions |
| `backend/internal/database/migrations/20241201_cosmos_db_citus_schema.sql` | Citus distributed schema |
| `backend/internal/search/postgres_search.go` | Search service (works with Citus) |

## 5. Migration Path

### Phase 1: Temporal Schedules (Week 1)
1. Deploy Temporal schedules alongside K8s CronJobs
2. Monitor both for 1 week
3. Disable K8s CronJobs
4. Remove K8s CronJob manifests

### Phase 2: Cosmos DB for PostgreSQL (Week 2-4)
1. Provision Cosmos DB for PostgreSQL cluster
2. Run Citus migration scripts
3. Dual-write to both databases
4. Migrate reads to Cosmos DB
5. Cut over writes
6. Decommission old database

### Phase 3: Multi-Region (Week 5-6)
1. Add read replicas in EU and Asia
2. Configure region-aware connection routing
3. Test failover scenarios
4. Enable cross-region metadata sync via Temporal

## 6. Cost Comparison

| Component | Before | After | Monthly Savings |
|-----------|--------|-------|-----------------|
| Scheduling | K8s CronJobs + pg_cron | Temporal (existing) | $0 (already have) |
| Search | Elasticsearch cluster | PostgreSQL native | ~$1,000 |
| Database | Single PostgreSQL | Cosmos DB for PG | +$500 (but scales) |
| **Net** | | | ~$500 saved + better scale |

## 7. Operational Benefits

1. **Fewer Moving Parts**: Temporal handles all scheduling
2. **Single Database Tech**: PostgreSQL everywhere (no Elasticsearch)
3. **Automatic Scaling**: Cosmos DB scales with tenant growth
4. **Better Observability**: Temporal Web UI for all jobs
5. **Global by Default**: Multi-region built into architecture
