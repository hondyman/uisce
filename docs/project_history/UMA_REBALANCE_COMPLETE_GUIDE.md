# UMA Rebalance System - Complete End-to-End Implementation

## Overview

A production-grade **Unified Managed Account (UMA) Rebalancing Platform** built on Temporal workflows, Redpanda/Kafka events, Hasura GraphQL, and ABAC policies. This system orchestrates intelligent, real-time portfolio rebalancing with tax-aware optimization and compliance delegation.

### Key Components

| Component | Purpose | Technology |
|-----------|---------|-----------|
| **UMA Rebalance Workflow** | Orchestrates rebalance lifecycle | Temporal SDK (Go) |
| **Drift Detection Rules** | Business logic enforcement | Go rules engine |
| **Event Bus** | Asynchronous communication | Redpanda / Kafka |
| **Real-Time Queries** | Live portfolio views | Hasura GraphQL |
| **REST API** | Workflow triggers & management | Gin framework |
| **UI Components** | Visual rebalance builder | React + ReactFlow |
| **ABAC Enforcement** | Role/location/time policies | Existing ABAC engine |

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         Frontend (React)                        │
│                    UMA Builder + Dashboard                      │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│                   Gin REST API (8087)                           │
│  • POST /uma/rebalance/request                                 │
│  • GET /uma/rebalance/:workflow_id/status                      │
│  • POST /uma/rebalance/plan/:plan_id/approve                   │
└────────────────────┬────────────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────────────┐
│          Temporal Workflow (UMARebalanceWorkflow)               │
│ ┌──────────────────────────────────────────────────────────┐   │
│ │ 1. ABAC Authorization Check                             │   │
│ │ 2. Load UMA Data (accounts, sleeves, holdings)          │   │
│ │ 3. Evaluate Business Rules                              │   │
│ │ 4. Generate Rebalance Trades (drift-based)              │   │
│ │ 5. Tax Harvest Simulation (xAI-augmented)              │   │
│ │ 6. Approval Check (thresholds)                          │   │
│ │ 7. Execute Trades (custodian integration)               │   │
│ │ 8. Update Hasura (live queries)                         │   │
│ │ 9. Emit Completion Event                                │   │
│ └──────────────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────────────┘
                     │
     ┌───────────────┼───────────────┐
     │               │               │
┌────▼────┐   ┌─────▼──────┐  ┌────▼────────┐
│PostgreSQL│   │  Redpanda (Kafka)  │  │  Hasura     │
│          │   │            │  │             │
│ UMA Data │   │  Events    │  │  GraphQL    │
│ Tables   │   │  Transport │  │  Subscr.    │
└──────────┘   └──────┬─────┘  └─────────────┘
                      │
           ┌──────────▼──────────┐
           │  Events Listener    │
           │  (RabbitMQ consumer)│
           │                     │
           │ • Drift Detected    │
           │ • Tax Simulated     │
           │ • Rebalance Done    │
           └─────────────────────┘
```

---

## Data Models

### UMA Account
```go
type UMAAccount struct {
    ID              string
    TenantID        string
    Name            string
    AUM             float64
    Status          string // active, inactive, archived
    TargetAllocation map[string]float64 // {"equities": 0.60, ...}
    LastRebalanced  *time.Time
}
```

### UMA Sleeve
```go
type UMASleeve struct {
    ID                string
    UMAAccountID      string
    SleeveType        string // "equities", "fixed_income", "alternatives"
    TargetAllocation  float64 // 0.60 = 60%
    CurrentAllocation float64
    Drift             float64 // current - target
    MinDriftThreshold float64 // 0.05 = 5%
}
```

### Rebalance Plan
```go
type UMARebalancePlan struct {
    ID              string
    UMAAccountID    string
    Trades          []UMARebalanceTrade
    TotalTaxImpact  float64
    TotalCost       float64
    Status          string // draft, pending_approval, approved, executing, completed
    ApprovedBy      string
    ApprovedAt      *time.Time
}
```

---

## Workflow Phases

### Phase 1: ABAC Authorization
- Validates user has `rebalance` permission on `uma` resource
- Checks temporal policies (e.g., "NY office only during 9-5 ET")
- Returns 403 if denied

### Phase 2: Load UMA Data
- Retrieves account, sleeves, holdings from PostgreSQL
- Enriches with market prices
- Calculates current allocations

### Phase 3: Evaluate Rules
- **Drift Detection**: Flags sleeves exceeding thresholds
- **Allocation Balance**: Ensures target = 100%
- **Trade Validation**: Min size, tax lot sufficiency
- **Wash Sale Risk**: Checks for 61-day violations

### Phase 4: Generate Trades
- Calculates rebalance trades for drifted sleeves
- Prioritizes by drift magnitude
- Aggregates by sleeve/security

### Phase 5: Tax Simulation
- Identifies tax-loss harvesting opportunities
- Estimates tax savings (assumes 25% marginal rate)
- Flags wash-sale candidates

### Phase 6: Approval Check
- **Auto-Approve** if: AUM < $5M AND trade cost < $100K
- **Require Approval** if: AUM > $5M OR cost > $100K OR harvest > $50K
- Waits for signal (24h timeout)

### Phase 7: Execute Trades
- Sends trades to custodian APIs (mocked in demo)
- Marks as executed with timestamp
- Tracks failures

### Phase 8: Update Hasura
- Inserts plan, trades, history into database
- Triggers subscriptions for live dashboards
- Updates rebalance metrics

### Phase 9: Emit Events
- `uma.rebalance.completed` event to RabbitMQ
- Consumed by listeners for analytics/notifications
- Audit trail in event log

---

## Rules Engine

### Drift Detection
```go
if sleeve.CurrentAllocation - sleeve.TargetAllocation > sleeve.MinDriftThreshold {
    violation := DriftThresholdExceeded
    // Trigger rebalance
}
```

### Allocation Balance
```go
if sum(targetAllocations) != 1.0 ± 1% {
    violation := AllocationBalanceInvalid
}
```

### Trade Size Validation
```go
if trade.GrossAmount < 1000 { // $1K min
    violation := TradeToSmall
}
```

### Tax Harvesting Eligibility
```go
if holding.UnrealizedGain < 0 && abs(gain) > 500 {
    opportunity := TaxLossToHarvest
}
```

---

## Event Types

All events implement the `DomainEvent` interface and are emitted to RabbitMQ:

| Event | Routing Key | When | Payload |
|-------|------------|------|---------|
| RebalanceRequested | `uma.rebalance.requested` | User initiates | requestID, umaID, type |
| PlanGenerated | `uma.rebalance.plan.generated` | Trades calculated | planID, tradeCount |
| PlanApproved | `uma.rebalance.plan.approved` | Approver signals | planID, approvedBy |
| ExecutionStarted | `uma.rebalance.execution.started` | Trades begin | planID |
| TradeExecuted | `uma.rebalance.trade.executed` | Trade completes | tradeID, status |
| Completed | `uma.rebalance.completed` | Workflow done | planID, completedCount |
| SleeveDriftDetected | `uma.sleeve.drift.detected` | Drift > threshold | sleeveID, driftPct |
| TaxHarvestSimulated | `uma.tax.harvest.simulated` | Tax optimization done | planID, savingsEst |

---

## API Endpoints

### 1. Request Rebalance
```bash
POST /uma/rebalance/request
Content-Type: application/json
X-Tenant-ID: tenant-123
X-Tenant-Datasource-ID: ds-456
X-User-ID: user-789

{
  "uma_account_id": "uma-001",
  "request_type": "drift",  // or "manual", "scheduled"
  "reason": "Quarterly rebalance",
  "initiated_by": "user-789"
}

Response (202 Accepted):
{
  "request_id": "req-uuid",
  "workflow_id": "uma-rebalance-uuid",
  "workflow_run_id": "run-uuid",
  "status": "pending",
  "message": "Rebalance workflow initiated"
}
```

### 2. Get Rebalance Status
```bash
GET /uma/rebalance/uma-rebalance-uuid/status

Response (200 OK):
{
  "workflow_id": "uma-rebalance-uuid",
  "status": "completed",
  "current_phase": "completion",
  "progress": {
    "plan_id": "plan-uuid",
    "trade_count": 5,
    "total_cost": 25000.00,
    "total_tax_impact": -5000.00,
    "execution_status": "completed"
  }
}
```

### 3. Approve Plan
```bash
POST /uma/rebalance/plan/plan-uuid/approve
Content-Type: application/json

{
  "plan_id": "plan-uuid",
  "approved_by": "advisor-123",
  "reason": "Approved for execution"
}

Response (200 OK):
{
  "message": "Plan approved",
  "plan_id": "plan-uuid",
  "approved_by": "advisor-123"
}
```

### 4. Reject Plan
```bash
POST /uma/rebalance/plan/plan-uuid/reject
Content-Type: application/json

{
  "rejected_by": "advisor-123",
  "reason": "Market conditions unfavorable"
}

Response (200 OK):
{
  "message": "Plan rejected",
  "reason": "Market conditions unfavorable"
}
```

---

## Database Schema

### UMA Accounts
```sql
CREATE TABLE uma_accounts (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    name VARCHAR(255),
    status VARCHAR(50),
    aum DECIMAL(19, 2),
    target_allocation JSONB,
    last_rebalanced TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

### UMA Sleeves
```sql
CREATE TABLE uma_sleeves (
    id UUID PRIMARY KEY,
    uma_account_id UUID NOT NULL,
    sleeve_type VARCHAR(100),
    target_allocation DECIMAL(5, 4),
    current_allocation DECIMAL(5, 4),
    drift DECIMAL(5, 4),
    min_drift_threshold DECIMAL(5, 4) DEFAULT 0.05,
    status VARCHAR(50),
    created_at TIMESTAMP DEFAULT NOW()
);
```

### UMA Rebalance Plans
```sql
CREATE TABLE uma_rebalance_plans (
    id UUID PRIMARY KEY,
    uma_account_id UUID NOT NULL,
    total_tax_impact DECIMAL(19, 2),
    total_cost DECIMAL(19, 2),
    trades JSONB NOT NULL,
    status VARCHAR(50),
    approved_by UUID,
    approved_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### UMA Rebalance History
```sql
CREATE TABLE uma_rebalance_history (
    id UUID PRIMARY KEY,
    plan_id UUID NOT NULL,
    uma_account_id UUID NOT NULL,
    completed_at TIMESTAMP,
    total_trade_count INT,
    success_count INT,
    failure_count INT,
    total_tax_impact DECIMAL(19, 2),
    pre_drift JSONB,
    post_drift JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## Local Deployment

### Prerequisites
- Docker & Docker Compose
- Go 1.21+
- Node.js 18+
- PostgreSQL client (psql)

### Quick Start

1. **Start services**:
```bash
docker-compose -f docker-compose.uma.yml up -d
```

2. **Run migrations**:
```bash
psql postgres://postgres:postgres@localhost:5432/alpha < backend/internal/migrations/001_uma_tables.sql
```

3. **Register Temporal workflows** (in separate terminal):
```bash
cd backend
go run cmd/temporal-worker/main.go
```

4. **Start frontend**:
```bash
cd frontend
npm install
npm run dev
```

5. **Access services**:
- **React Frontend**: http://localhost:5173
- **Hasura GraphQL**: http://localhost:8080
- **RabbitMQ Admin**: http://localhost:15672 (guest/guest)
- **Temporal Admin**: http://localhost:6500
- **UMA API**: http://localhost:8087

### Test End-to-End

```bash
# 1. Create a test UMA account
curl -X POST http://localhost:8087/uma/rebalance/request \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -H "X-Tenant-Datasource-ID: ds-456" \
  -H "X-User-ID: user-789" \
  -d '{
    "uma_account_id": "uma-001",
    "request_type": "manual",
    "reason": "Test rebalance",
    "initiated_by": "user-789"
  }'

# Expected: 202 Accepted with workflow_id

# 2. Check status
curl http://localhost:8087/uma/rebalance/uma-rebalance-uuid/status

# 3. Approve (when workflow requires approval)
curl -X POST http://localhost:8087/uma/rebalance/plan/plan-uuid/approve \
  -H "Content-Type: application/json" \
  -d '{
    "plan_id": "plan-uuid",
    "approved_by": "advisor-123"
  }'

# 4. Monitor RabbitMQ events
docker logs semlayer-uma-events-listener -f
```

---

## Tenant Scoping

All endpoints require tenant context:

**Headers**:
```
X-Tenant-ID: <tenant-uuid>
X-Tenant-Datasource-ID: <datasource-uuid>
X-User-ID: <user-uuid>
```

**Query Parameters** (fallback):
```
?tenant_id=<tenant-uuid>&datasource_id=<datasource-uuid>
```

The Gin middleware (`tenantContextMiddleware`) extracts and validates scope before processing.

---

## ABAC Integration

The workflow enforces ABAC authorization in Phase 1:

```go
abac.Evaluate(ctx, &abac.Request{
    Subject: userID,
    Action:  "rebalance",
    Resource: fmt.Sprintf("uma:%s", umaID),
    Context: map[string]string{
        "location": userLocation,
        "time":     now.Format(time.RFC3339),
    },
})
```

**Example Policy**:
```json
{
  "name": "uma_rebalance_policy",
  "subject": { "role": "portfolio_manager" },
  "resource": { "type": "uma" },
  "action": "rebalance",
  "conditions": [
    { "field": "aum", "op": ">", "value": 500000 },
    { "field": "location", "op": "in", "value": ["NY", "CA"] }
  ],
  "temporal": {
    "start": "09:00",
    "end": "17:00",
    "timezone": "America/New_York"
  }
}
```

---

## Event-Driven Architecture

### RabbitMQ Setup
```bash
# Topic exchange: uma.events
# Bindings:
#  - uma.rebalance.requested -> uma-events-queue
#  - uma.rebalance.completed -> uma-events-queue
#  - uma.sleeve.drift.detected -> uma-events-queue
#  - uma.tax.harvest.simulated -> uma-events-queue

# Consumer: uma-events-listener
#  - Listens on uma-events-queue
#  - Routes to appropriate handlers
#  - Updates database / triggers actions
```

### Event Flow
1. **REST API** receives rebalance request
2. **Workflow** runs through phases
3. **Activities** emit events to RabbitMQ
4. **Listener** consumes events
5. **Database** updated
6. **Dashboard** reflects changes via Hasura subscriptions

---

## Monitoring & Observability

### Temporal Dashboard
Visit http://localhost:6500 to:
- View running workflows
- Check workflow history
- Debug failed activities
- Inspect retry policies

### RabbitMQ Dashboard
Visit http://localhost:15672 (guest/guest) to:
- Monitor message queues
- View event routing
- Track dead-letter exchanges

### Logs
```bash
# Tail workflow logs
docker logs semlayer-uma-rebalance -f

# Tail event listener logs
docker logs semlayer-uma-events-listener -f

# Tail temporal worker logs
docker logs semlayer-temporal-worker -f
```

### Hasura Subscriptions
Query live rebalance status:
```graphql
subscription {
  uma_rebalance_plans(where: { uma_account_id: { _eq: "uma-001" } }) {
    id
    status
    created_at
    updated_at
    trades(limit: 5) {
      id
      execution_status
      gross_amount
    }
  }
}
```

---

## Performance Characteristics

| Metric | Target | Notes |
|--------|--------|-------|
| Rebalance Request → Plan | < 5s | Includes ABAC, rules, tax sim |
| Trade Execution | < 10s | 100 trades in batch |
| Approval Workflow | < 30min | Signals up to 24h timeout |
| Event Processing | < 500ms | Per-message in listener |
| Hasura Query | < 100ms | Live subscription on plan |

---

## Future Enhancements

1. **AI-Augmented Lot Selection**: Integrate xAI for smart tax-loss harvesting
2. **Multi-Custodian Sync**: RabbitMQ fanout for JPMorgan, Fidelity, etc.
3. **Real-Time Rebalancing**: Sub-second market data ingestion
4. **Benchmark Tracking**: Hasura federation for alts benchmarks
5. **Advisor Delegation**: ABAC temporal policies for offshore approvals
6. **Compliance Audit**: Event replay for forensic analysis
7. **ML-Based Optimization**: Drift prediction + auto-rebalance

---

## Troubleshooting

### Workflow Stuck in Approval
```bash
# Signal rejection to unblock
curl -X POST http://localhost:8087/uma/rebalance/plan/plan-uuid/reject \
  -H "Content-Type: application/json" \
  -d '{"rejected_by": "admin", "reason": "Unblock test"}'
```

### Events Not Processing
```bash
# Check RabbitMQ binding
docker exec semlayer-rabbitmq rabbitmqctl list_bindings

# Check listener logs
docker logs semlayer-uma-events-listener | grep "ERROR"
```

### PostgreSQL Connection Issues
```bash
# Test connection
psql postgres://postgres:postgres@localhost:5432/alpha -c "SELECT 1"

# Check migrations applied
psql postgres://postgres:postgres@localhost:5432/alpha -c "\dt"
```

---

## Contributing

1. Add new rules to `internal/rules/uma_rebalance_rules.go`
2. Extend workflow phases in `internal/workflows/uma_rebalance_workflow.go`
3. Add event types to `internal/events/event_types.go`
4. Update database schema in `internal/migrations/`

---

## License

Proprietary - Semlayer Inc.
