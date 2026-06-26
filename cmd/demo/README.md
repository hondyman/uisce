# Temporal Rebalancing Agent Demo

This demo showcases the complete autonomous rebalancing workflow with:
- Real drift calculation (StarRocks or mock)
- AI-generated trade proposals (Gemini Pro or fallback)
- Policy-based validation
- Human-in-the-loop approval
- Saga-based trade execution
- Universal Audit Record (UAR) persistence

## Quick Start

### 1. Prerequisites
```bash
# Start Temporal dev server (in a separate terminal)
temporal server start-dev
```

### 2. Run Demo (All Mocks)
```bash
cd /Users/eganpj/GitHub/semlayer
go run ./cmd/demo
```

Expected output:
```
✅ Using in-memory UAR store
ℹ️  GOOGLE_GENERATIVE_AI_API_KEY not set – using mock AI proposals
ℹ️  STARROCKS_DSN not set – using mock drift data
✅ Temporal worker started
🚀 workflow started – ID=rebalancer-demo-XXXXXX run=...
✅ auto-approval signal sent
🛑 worker stopped
```

## Environment Variables (Optional)

### Real Gemini AI Proposals
```bash
export GOOGLE_GENERATIVE_AI_API_KEY="your-api-key-here"
go run ./cmd/demo
```

### Real StarRocks Drift Calculation
```bash
export STARROCKS_DSN="root:@tcp(localhost:9030)/"
go run ./cmd/demo
```

### Postgres UAR Persistence
```bash
export DATABASE_DSN="postgres://ws:ws_pass@localhost:5432/wealthstream_dev?sslmode=disable"
go run ./cmd/demo
```

### Full Production Mode
```bash
export GOOGLE_GENERATIVE_AI_API_KEY="your-api-key"
export STARROCKS_DSN="root:@tcp(localhost:9030)/"
export DATABASE_DSN="postgres://ws:ws_pass@localhost:5432/wealthstream_dev?sslmode=disable"
go run ./cmd/demo
```

## Workflow Steps

The demo executes this flow:

1. **Drift Check** (`CheckDriftActivity`)
   - Queries StarRocks for portfolio positions
   - Calculates drift percentage per asset class
   - Falls back to mock data if StarRocks unavailable

2. **AI Proposal Generation** (`GenerateAIProposalActivity`)
   - Calls Gemini Pro with drift report
   - Receives structured `TradeProposal` JSON
   - Falls back to mock proposal if Gemini unavailable

3. **Policy Validation** (`PolicyCheckActivity`)
   - Checks confidence threshold (must be ≥ 0.6)
   - Validates trade rules (placeholder for OPA/Rego)

4. **Advisor Notification & Approval** (`NotifyAdvisorActivity`)
   - Notifies advisor (mock or real GenUI POST)
   - Waits for approval signal (auto-sent after 30s in demo)
   - Timeout: 2 minutes → escalates to supervisor

5. **Trade Execution** (`ExecuteTradeSagaActivity`)
   - Executes trades using saga pattern
   - Simulates compensation on failure
   - Returns execution details

6. **UAR Persistence** (`PersistUARActivity`)
   - Writes complete audit trail
   - Stores in Postgres or in-memory

## Monitoring

### View Workflow in Temporal UI
```
http://localhost:8233
```

Look for workflow ID: `rebalancer-demo-XXXXXX`

### Check UAR Records (if using Postgres)
```sql
SELECT * FROM universal_audit_records 
WHERE tenant_id = 'demo_tenant' 
ORDER BY created_at DESC;
```

## Architecture

```
┌─────────────────┐
│  Temporal       │
│  Workflow       │
└────────┬────────┘
         │
    ┌────┴────────────────────────┐
    │                             │
┌───▼──────┐              ┌───────▼────────┐
│ Drift    │              │ Gemini AI      │
│Calculator│              │ Client         │
└───┬──────┘              └───────┬────────┘
    │                             │
┌───▼──────┐              ┌───────▼────────┐
│StarRocks │              │ Google Gemini  │
│ (or mock)│              │ Pro (or mock)  │
└──────────┘              └────────────────┘
                                   │
                          ┌────────▼────────┐
                          │ UAR Store       │
                          │ (Postgres or    │
                          │  in-memory)     │
                          └─────────────────┘
```

## Files

- [cmd/demo/main.go](file:///Users/eganpj/GitHub/semlayer/cmd/demo/main.go) - Demo runner
- [internal/workflows/rebalance_workflow.go](file:///Users/eganpj/GitHub/semlayer/internal/workflows/rebalance_workflow.go) - Workflow logic
- [internal/activities/activities.go](file:///Users/eganpj/GitHub/semlayer/internal/activities/activities.go) - Activity implementations
- [internal/drift/calculator.go](file:///Users/eganpj/GitHub/semlayer/internal/drift/calculator.go) - Drift calculation
- [internal/ai/gemini_client.go](file:///Users/eganpj/GitHub/semlayer/internal/ai/gemini_client.go) - Gemini integration
- [internal/uar/store.go](file:///Users/eganpj/GitHub/semlayer/internal/uar/store.go) - Audit trail stores

## Next Steps

1. Run the demo and verify workflow completes
2. Set `GOOGLE_GENERATIVE_AI_API_KEY` to test real AI proposals
3. Implement real `CheckDriftActivity` with StarRocks queries
4. Connect `NotifyAdvisorActivity` to GenUI frontend
5. Integrate OPA/Rego for policy checks
6. Wire `ExecuteTradeSagaActivity` to custodian APIs
