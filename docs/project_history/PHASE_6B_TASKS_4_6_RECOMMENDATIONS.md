# Phase 6B Tasks 4-6: Implementation Recommendations

## 🎯 Overview

Tasks 1-3 (Infrastructure) are complete. Tasks 4-6 focus on **UI, Validation, and Testing** to ship a production-ready Business Process framework.

**Remaining Effort:** ~1,200 lines of code + tests
**Timeline:** 2-3 days for full delivery
**Impact:** 75% → 95%+ Workday Parity

---

## 📋 Task 4: React BP Builder UI (~400 lines TSX)

### What to Build
A **drag-and-drop visual workflow editor** enabling non-technical users to create Business Processes.

**File:** `frontend/src/pages/bundles/BPBuilder.tsx`

### Architecture

```
┌────────────────────────────────────────────────────────────┐
│                      BP Builder Main                        │
│  Orchestrates: Layout + Canvas + Editor + Preview          │
└────────────────────────────────────────────────────────────┘
         │                    │                    │
         ▼                    ▼                    ▼
    ┌────────────┐    ┌──────────────┐    ┌────────────────┐
    │ Palette    │    │ Canvas       │    │ Step Editor    │
    │ (Steps)    │    │ (Drop Zone)  │    │ (Form)         │
    └────────────┘    └──────────────┘    └────────────────┘
         │                    │                    │
         └────────────────────┴────────────────────┘
                      │
                      ▼
            ┌──────────────────────┐
            │ BP Preview & JSON    │
            │ (Real-time update)   │
            └──────────────────────┘
                      │
                      ▼
            ┌──────────────────────┐
            │ Save/Deploy          │
            │ (POST /api/bp)       │
            └──────────────────────┘
```

### Components to Create

#### 1. **BPBuilder.tsx** (Main Component)
```tsx
// State management
const [steps, setSteps] = useState<BPStep[]>([])
const [selectedStep, setSelectedStep] = useState<BPStep | null>(null)
const [bpName, setBPName] = useState('')
const [bpDescription, setBPDescription] = useState('')
const [isSaving, setIsSaving] = useState(false)

// Layout: 3-column: Palette | Canvas | Editor
// Drag-drop enabled with react-flow-renderer
```

**Key Features:**
- Drag steps from palette onto canvas
- Reorder steps (1→2→3→4)
- Delete steps
- Edit step properties
- Real-time JSON preview
- Save to backend via APICreateBusinessProcess

#### 2. **StepPalette.tsx** (Drag Source)
```tsx
// Available step types (draggable)
const STEP_TYPES = [
  { type: 'data_entry', label: 'Data Entry', icon: '📝' },
  { type: 'validate', label: 'Validation', icon: '✓' },
  { type: 'approve', label: 'Approval', icon: '👤' },
  { type: 'notify', label: 'Notification', icon: '📢' },
  { type: 'integrate', label: 'Integration', icon: '🔗' },
  { type: 'compute', label: 'Compute', icon: '⚙️' }
]

// Each type is draggable (react-dnd or native drag API)
```

#### 3. **BPCanvas.tsx** (Drop Zone + Visualization)
```tsx
// Uses react-flow-renderer to display workflow
// Nodes: steps (connected by edges)
// Supports conditional branching (if/then/else)
// Click node to edit in right panel
```

**Features:**
- Visualize step order (Step 1→2→3→4)
- Drag to reorder
- Double-click to edit
- Delete button on each node
- Conditional connectors (different colors for branches)

#### 4. **StepEditor.tsx** (Right Panel Form)
```tsx
// Form fields for selected step:
<TextField label="Step Name" value={step.step_name} />
<Select label="Step Type" value={step.step_type} options={STEP_TYPES} />
<TextField label="Assignee Role" value={step.assignee_role} />
<TextField label="Duration (hours)" type="number" value={step.duration_hours} />
<MultiSelect label="Triggers" value={step.trigger_ids} />
<JsonEditor label="Conditions (JSON)" value={step.condition_json} />
<Button onClick={() => saveStep(step)} label="Save" />
```

#### 5. **BPPreview.tsx** (Bottom Panel)
```tsx
// Real-time JSON preview (read-only)
// Shows the complete BP definition
// Enables copy-to-clipboard for API calls
{
  "process_name": "HireEmployee",
  "steps": [
    { "step_order": 1, "step_type": "data_entry", ... },
    { "step_order": 2, "step_type": "validate", ... },
    ...
  ]
}
```

#### 6. **BPActions.tsx** (Save/Deploy)
```tsx
// Buttons:
// - Save (POST /api/bp)
// - Publish (mark is_active = true)
// - Test (start execution with sample data)
// - Export (download JSON)
// - Delete
```

### Implementation Steps

**Step 1:** Set up component structure
```bash
cd frontend/src/pages/bundles
touch BPBuilder.tsx StepPalette.tsx BPCanvas.tsx StepEditor.tsx BPPreview.tsx BPActions.tsx
```

**Step 2:** Install dependencies
```bash
npm install react-flow-renderer react-dnd react-dnd-html5-backend
# or use native drag API (simpler, no extra deps)
```

**Step 3:** Implement drag-drop
```tsx
// Option A: Native drag API (simplest)
onDragStart={(e, stepType) => e.dataTransfer.setData('stepType', stepType)}
onDrop={(e) => {
  const stepType = e.dataTransfer.getData('stepType')
  addStep({ step_type: stepType, step_order: steps.length + 1 })
}}

// Option B: react-flow-renderer (more powerful)
// Provides visual workflow editor out of box
```

**Step 4:** Integrate with API
```tsx
// In BPActions:
const handleSave = async () => {
  const req = {
    process_name: bpName,
    description: bpDescription,
    steps: steps
  }
  
  const res = await fetch('/api/bp?tenant_id=...&datasource_id=...', {
    method: 'POST',
    headers: { 
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId
    },
    body: JSON.stringify(req)
  })
  
  if (res.ok) {
    notification.success('BP saved successfully')
  }
}
```

**Step 5:** Add routing
```tsx
// In frontend/src/App.tsx or router:
import BPBuilder from './pages/bundles/BPBuilder'

// Add route:
<Route path="/bp/builder" element={<BPBuilder />} />
```

### UI Layout (Recommended)

```
┌─────────────────────────────────────────────────────────────┐
│ Business Process Builder                    [Save] [Publish] │
├─────────────────┬──────────────────────┬───────────────────┤
│  Step Palette   │                      │  Step Editor      │
│                 │                      │  ────────────────│
│ ✎ Data Entry   │   Canvas:            │  Name: [____]    │
│ ✓ Validate     │                      │  Type: [____]    │
│ 👤 Approval    │   Step 1             │  Role: [____]    │
│ 📢 Notify      │    │                 │  Hours: [_48_]   │
│ 🔗 Integrate   │    ▼                 │  [Save] [Delete] │
│ ⚙️ Compute     │   Step 2             │                  │
│                 │    │                 │  Triggers:       │
│                 │    ▼                 │  [x] save        │
│                 │   Step 3             │  [x] validate    │
│                 │    │                 │                  │
│                 │    ▼                 │                  │
│                 │   Step 4             │                  │
├─────────────────┴──────────────────────┴───────────────────┤
│  JSON Preview (collapsible)                                 │
│  { "process_name": "HireEmployee", "steps": [...] }        │
└─────────────────────────────────────────────────────────────┘
```

### Testing Checklist (Task 4)

- [ ] Drag step from palette → appears on canvas
- [ ] Reorder steps (drag to change order)
- [ ] Edit step properties (name, duration, assignee)
- [ ] Delete step (confirm modal)
- [ ] JSON preview updates in real-time
- [ ] Save BP → calls POST /api/bp correctly
- [ ] Tenant scope applied (tenant_id + datasource_id)
- [ ] Error handling (network errors, validation)
- [ ] Loading state during save
- [ ] Success notification on save

### Estimated Breakdown
```
BPBuilder.tsx        100 lines
StepPalette.tsx      80 lines
BPCanvas.tsx         100 lines
StepEditor.tsx       80 lines
BPPreview.tsx        40 lines
BPActions.tsx        50 lines
───────────────────────────
Total               ~450 lines (higher due to styling + error handling)
```

---

## 📊 Task 5: HireEmployee E2E Demo (~200 lines)

### What to Build
**End-to-end validation** proving Phase 6A + 6B + 6C integration works.

### Demo Flow

1. **Create BP** (via API)
   ```bash
   curl -X POST http://localhost:8080/api/bp \
     -H "X-Tenant-ID: tenant-1" \
     -d '{"process_name": "HireEmployee", "steps": [...]}'
   # Response: { "id": "bp-123" }
   ```

2. **Start Execution**
   ```bash
   curl -X POST http://localhost:8080/api/bp/bp-123/start \
     -d '{"entity_id": "emp-456", "entity_type": "employee", ...}'
   # Response: { "instance_id": "inst-789" }
   ```

3. **Monitor Progress**
   ```bash
   curl http://localhost:8080/api/bp/instance/inst-789
   # Shows: Step 1 → 2 → 3 → 4 (with timestamps)
   ```

4. **Trigger Approval**
   ```bash
   curl -X POST http://localhost:8080/api/bp/instance/inst-789/approve \
     -d '{"decision": "approved"}'
   ```

5. **React UI** displays entire flow

### Implementation Approach

**Option A: Scripted Demo (Simpler)**
```bash
#!/bin/bash
# demo-hireemployee.sh

echo "1. Creating HireEmployee BP..."
BP_ID=$(curl -s -X POST http://localhost:8080/api/bp ... | jq -r '.id')
echo "   Created: $BP_ID"

echo "2. Starting execution..."
INSTANCE_ID=$(curl -s -X POST http://localhost:8080/api/bp/$BP_ID/start ... | jq -r '.instance_id')
echo "   Instance: $INSTANCE_ID"

echo "3. Monitoring steps..."
for i in {1..4}; do
  curl -s http://localhost:8080/api/bp/instance/$INSTANCE_ID | jq '.current_step'
  sleep 2
done

echo "4. Done!"
```

**Option B: React Demo Page (Better UX)**
```tsx
// frontend/src/pages/demo/HireEmployeeDemo.tsx

export default function HireEmployeeDemo() {
  const [bpId, setBpId] = useState<string | null>(null)
  const [instanceId, setInstanceId] = useState<string | null>(null)
  const [status, setStatus] = useState<BPInstanceResponse | null>(null)
  
  const handleCreateBP = async () => {
    // POST /api/bp with HireEmployee definition
    const res = await fetch('/api/bp?tenant_id=...', {
      method: 'POST',
      body: JSON.stringify({
        process_name: 'HireEmployee',
        steps: [
          { step_order: 1, step_type: 'data_entry', ... },
          { step_order: 2, step_type: 'validate', duration_hours: 24 },
          { step_order: 3, step_type: 'approve', duration_hours: 48 },
          { step_order: 4, step_type: 'notify', ... }
        ]
      })
    })
    const data = await res.json()
    setBpId(data.id)
  }
  
  const handleStartExecution = async () => {
    // POST /api/bp/:id/start
    const res = await fetch(`/api/bp/${bpId}/start?tenant_id=...`, {
      method: 'POST',
      body: JSON.stringify({
        entity_id: 'emp-12345',
        entity_type: 'employee',
        data: {
          first_name: 'John',
          last_name: 'Doe',
          email: 'john@company.com',
          position: 'Engineer'
        }
      })
    })
    const data = await res.json()
    setInstanceId(data.instance_id)
  }
  
  const handlePollStatus = async () => {
    // GET /api/bp/instance/:id
    const res = await fetch(`/api/bp/instance/${instanceId}?tenant_id=...`)
    const data = await res.json()
    setStatus(data)
  }
  
  const handleApprove = async () => {
    // POST /api/bp/instance/:id/approve
    await fetch(`/api/bp/instance/${instanceId}/approve?tenant_id=...`, {
      method: 'POST',
      body: JSON.stringify({
        decision: 'approved',
        comment: 'Candidate looks great'
      })
    })
    handlePollStatus() // Refresh status
  }
  
  return (
    <div style={{ padding: '20px' }}>
      <h1>HireEmployee BP E2E Demo</h1>
      
      <section>
        <h2>Step 1: Create BP</h2>
        <button onClick={handleCreateBP}>Create HireEmployee BP</button>
        {bpId && <p>✅ BP Created: {bpId}</p>}
      </section>
      
      <section>
        <h2>Step 2: Start Execution</h2>
        <button onClick={handleStartExecution} disabled={!bpId}>Start Execution</button>
        {instanceId && <p>✅ Instance Started: {instanceId}</p>}
      </section>
      
      <section>
        <h2>Step 3: Monitor Progress</h2>
        <button onClick={handlePollStatus} disabled={!instanceId}>Poll Status</button>
        {status && (
          <div>
            <p>Current Step: {status.current_step} / {status.process_name}</p>
            <p>Status: {status.status}</p>
            <p>Entity: {status.entity_id} ({status.entity_type})</p>
            <p>Started: {new Date(status.started_at).toLocaleString()}</p>
            <p>Due: {new Date(status.current_step_due_at).toLocaleString()}</p>
          </div>
        )}
      </section>
      
      <section>
        <h2>Step 4: Approve (if at approval step)</h2>
        <button onClick={handleApprove} disabled={status?.current_step !== 3}>
          Approve Step 3
        </button>
      </section>
      
      {/* Timeline visualization */}
      <StepTimeline status={status} />
      
      {/* Events/Logs */}
      <EventLog instanceId={instanceId} />
    </div>
  )
}
```

### Demo Components

1. **StepTimeline.tsx** — Visual progress bar
```tsx
// Shows: ✅ Step 1 → ✅ Step 2 → ⏳ Step 3 (48h due 10/30) → ⭕ Step 4

// Color coding:
// ✅ Complete (green)
// ⏳ In Progress (blue)
// ⭕ Pending (gray)
// ⚠️ Overdue (red)
```

2. **EventLog.tsx** — Real-time events
```tsx
// Subscribe to /api/bp/instance/:id/events (WebSocket or polling)
// Display:
// - Step started
// - Step completed
// - Approval pending
// - Timeout escalated
// - BP completed
```

### Testing Checklist (Task 5)

- [ ] BP creation succeeds via API
- [ ] Execution starts with entity_id + data
- [ ] current_step increments (1→2→3→4)
- [ ] Status shows correct step duration
- [ ] Due date calculated correctly (24h, 48h)
- [ ] Approval form works at step 3
- [ ] Timeout event fires at deadline
- [ ] Escalation to next manager triggered
- [ ] UI shows all steps completed
- [ ] No data leakage between tenants

---

## 🧪 Task 6: Tests & Documentation (~400 lines)

### 6A: Unit Tests

**File:** `backend/internal/temporal/bp_executor_test.go`

```go
// Test 1: ExecuteBusinessProcessWorkflow
func TestExecuteBusinessProcessWorkflow(t *testing.T) {
  // Mock DB
  // Mock activities
  // Execute workflow
  // Assert: workflow completes, steps logged
}

// Test 2: LoadBPInstanceActivity
func TestLoadBPInstanceActivity(t *testing.T) {
  // Mock: DB has instance
  // Execute activity
  // Assert: instance loaded correctly
}

// Test 3: ExecuteBPStepActivity
func TestExecuteBPStepActivity(t *testing.T) {
  // Test each step type: data_entry, validate, approve, notify
  // Assert: correct status returned
}

// Test 4: BranchingLogic
func TestBranchingLogic(t *testing.T) {
  // If condition_json is present
  // Assert: next_step determined correctly
}

// Test 5: TimeoutHandling
func TestTimeoutHandling(t *testing.T) {
  // Activity exceeds deadline
  // Assert: timeout event published
}

// Test 6: AuditLogging
func TestAuditLogging(t *testing.T) {
  // Step executes
  // Assert: entry written to bp_step_executions
}
```

**Setup Pattern:**
```go
import (
  "testing"
  "go.temporal.io/sdk/testsuite"
  "github.com/stretchr/testify/assert"
)

func TestBPWorkflow(t *testing.T) {
  ts := &testsuite.WorkflowTestSuite{}
  env := ts.NewTestWorkflowEnvironment()
  
  // Mock activities
  env.OnActivity(LoadBPInstanceActivity, mock.MatchedBy(func(ctx context.Context, instanceID string) bool {
    return true
  })).Return(&BPInstanceData{...}, nil)
  
  // Execute workflow
  env.ExecuteWorkflow(ExecuteBusinessProcessWorkflow, "instance-123")
  
  // Assert
  require.True(t, env.IsWorkflowCompleted())
  require.NoError(t, env.GetWorkflowError())
}
```

### 6B: Integration Test

**File:** `backend/internal/temporal/bp_integration_test.go`

```go
func TestHireEmployeeBPE2E(t *testing.T) {
  // Setup: Start real Temporal server (or Docker)
  // 1. Create HireEmployee BP via API
  // 2. Start execution
  // 3. Verify: Step 1 completes
  // 4. Verify: Step 2 starts
  // 5. Approve step 3
  // 6. Verify: BP completes
  // Assert: All 4 steps executed, audit trail logged
}
```

### 6C: API Curl Examples

**File:** `BUSINESS_PROCESS_API_EXAMPLES.md` (new)

```bash
# Create HireEmployee BP
curl -X POST http://localhost:8080/api/bp \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-1" \
  -H "X-Tenant-Datasource-ID: ds-1" \
  -d '{
    "process_name": "HireEmployee",
    "description": "End-to-end hiring workflow",
    "steps": [...]
  }'
# Response: { "id": "bp-123", "message": "BP created" }

# List BPs
curl http://localhost:8080/api/bp?tenant_id=tenant-1&datasource_id=ds-1

# Get BP details
curl http://localhost:8080/api/bp/bp-123?tenant_id=tenant-1&datasource_id=ds-1

# Start execution
curl -X POST http://localhost:8080/api/bp/bp-123/start?tenant_id=tenant-1&datasource_id=ds-1 \
  -H "Content-Type: application/json" \
  -d '{
    "entity_id": "emp-456",
    "entity_type": "employee",
    "data": {
      "first_name": "John",
      "last_name": "Doe",
      "email": "john@company.com"
    }
  }'
# Response: { "instance_id": "inst-789", "status": "started" }

# Check instance status
curl http://localhost:8080/api/bp/instance/inst-789?tenant_id=tenant-1&datasource_id=ds-1
# Response: { "current_step": 2, "status": "in_progress", "current_step_due_at": "...", ... }

# Approve step
curl -X POST http://localhost:8080/api/bp/instance/inst-789/approve?tenant_id=tenant-1 \
  -H "Content-Type: application/json" \
  -d '{
    "decision": "approved",
    "comment": "Candidate approved by CEO"
  }'
# Response: { "decision": "approved", "status": "updated" }
```

### 6D: Documentation

#### **BUSINESS_PROCESS_QUICK_START.md** (3 minutes)

```markdown
# Quick Start: Business Process Framework

## 1. Deploy Infrastructure
```bash
psql ... < migrations/business_processes.sql
go build -o bin/semlayer ./backend/cmd/...
# Register routes in api.go
# Register workflow in temporal worker
```

## 2. Create Your First BP
```bash
# Save to create-bp.json:
{
  "process_name": "OnboardEmployee",
  "steps": [
    {"step_order": 1, "step_type": "data_entry", "step_name": "Collect Info"},
    {"step_order": 2, "step_type": "validate", "step_name": "Background Check"},
    {"step_order": 3, "step_type": "approve", "step_name": "Manager Approval"},
    {"step_order": 4, "step_type": "notify", "step_name": "Send Welcome"}
  ]
}

curl -X POST http://localhost:8080/api/bp \
  -H "X-Tenant-ID: tenant-1" \
  -d @create-bp.json
```

## 3. Start Execution
```bash
curl -X POST http://localhost:8080/api/bp/BP_ID/start \
  -d '{"entity_id": "emp-123", "entity_type": "employee", ...}'
```

## 4. Monitor Progress
```bash
curl http://localhost:8080/api/bp/instance/INSTANCE_ID
```

**Done!** Your workflow is now running in Temporal.
```

#### **BUSINESS_PROCESS_ARCHITECTURE.md** (Overview)

Already exists → Reference in Task 6 docs.

### Testing Checklist (Task 6)

- [ ] Unit tests: 80%+ coverage for bp_executor.go
- [ ] Integration test: HireEmployee E2E passes
- [ ] API curl examples: All 6 endpoints work
- [ ] Temporal mock activities: Work correctly
- [ ] Database queries: Execute without errors
- [ ] Timeout logic: Tested explicitly
- [ ] Branching: Multiple paths tested
- [ ] Multi-tenant: Scoping verified

### Estimated Breakdown
```
bp_executor_test.go             150 lines
bp_integration_test.go          100 lines
BUSINESS_PROCESS_API_EXAMPLES.md 100 lines
BUSINESS_PROCESS_QUICK_START.md  50 lines
───────────────────────────────────────
Total                           ~400 lines
```

---

## 🚀 Implementation Roadmap

### Week 1 (Task 4: React UI)
**Day 1-2:**
- Scaffold BPBuilder.tsx + components
- Implement drag-drop (native or react-flow)
- Wire up StepEditor form

**Day 3:**
- Integrate with APICreateBusinessProcess
- Add error handling + loading states
- Test locally

**Deliverable:** React BP Builder UI complete, can create BP visually

### Week 2 (Task 5: E2E Demo)
**Day 1:**
- Implement HireEmployeeDemo.tsx
- Add StepTimeline + EventLog components
- Wire up all API calls

**Day 2:**
- Test end-to-end flow
- Validate Phase 6A + 6B + 6C integration
- Screenshot for documentation

**Deliverable:** E2E demo proves system works, Phase 6A/6B/6C integrate correctly

### Week 2-3 (Task 6: Tests)
**Day 1-2:**
- Write unit tests (bp_executor_test.go)
- Mock Temporal workflows
- Achieve 80%+ coverage

**Day 3:**
- Integration test (HireEmployee E2E)
- API curl examples
- Quick-start guide

**Deliverable:** 100% test coverage, production-ready

---

## 📊 Completion Criteria

### Task 4 ✅
- [ ] BPBuilder component loads
- [ ] Drag-drop works (palette → canvas)
- [ ] Step editor form saves changes
- [ ] JSON preview updates in real-time
- [ ] Save BP button works
- [ ] Multi-tenant scoping applied
- [ ] No TypeScript errors
- [ ] Responsive design (mobile-friendly optional)

### Task 5 ✅
- [ ] E2E demo script runs
- [ ] BP created successfully
- [ ] Execution starts with entity_id
- [ ] All 4 steps progress (Step 1→2→3→4)
- [ ] Approval step works
- [ ] Timeout escalation fires
- [ ] React UI displays progress
- [ ] No errors in console/logs

### Task 6 ✅
- [ ] Unit tests written (6+ tests)
- [ ] Integration test passes
- [ ] API curl examples all work
- [ ] Quick-start guide clear
- [ ] Code coverage 80%+
- [ ] Documentation complete
- [ ] Ready for production deployment

---

## 🎯 Success Metrics

**Code Quality:**
- No TypeScript errors in frontend
- No Go compilation errors in backend
- Test coverage ≥80%
- Proper error handling throughout

**Functionality:**
- BP creation via UI works
- BP execution via API works
- Monitoring shows correct status
- Phase 6A/6B/6C integration confirmed

**Documentation:**
- README clear enough for new developer
- Curl examples all work
- Quick-start < 5 minutes

**Performance:**
- BP creation < 200ms
- Execution polling < 100ms
- No N+1 queries
- Database indexes used

---

## ⚠️ Common Pitfalls to Avoid

1. **Not Validating Input**
   - Always validate step_order, duration_hours, etc.
   - Return clear error messages

2. **Forgetting Tenant Scope**
   - Prefix all queries with `WHERE tenant_id = $1 AND datasource_id = $2`
   - Double-check before pushing

3. **Blocking Approval Workflow**
   - Use async/await for long operations
   - Show loading spinner to user

4. **Not Testing Timeout Logic**
   - Unit test specifically: workflow exceeds deadline
   - Mock Temporal activity timeout

5. **Incomplete Error Handling**
   - Catch DB errors, network errors, validation errors
   - Display user-friendly messages

---

## 📞 Quick Reference

**Key Files to Reference:**
- `BUSINESS_PROCESS_DEPLOY.md` — Deployment steps
- `BUSINESS_PROCESS_DELIVERY.md` — Feature overview
- `bp_executor.go` — Temporal workflow (Task 2)
- `business_process_api.go` — REST API (Task 3)
- `business_processes.sql` — Database schema (Task 1)

**Dependencies to Install:**
```bash
# Frontend
npm install react-flow-renderer  # or use native drag API

# Backend
go get go.temporal.io/sdk@latest
go get github.com/stretchr/testify@latest
```

**Test Locally:**
```bash
# Terminal 1: Temporal server
temporal server started

# Terminal 2: Go backend
cd backend && go run ./cmd/...

# Terminal 3: React frontend
cd frontend && npm start

# Terminal 4: Test
curl -X POST http://localhost:8080/api/bp ...
```

---

## 🎉 Recommended Next Steps

1. **Start Task 4 Today**
   - Scaffold React component
   - Get drag-drop working
   - Celebrate first step!

2. **Parallel Work**
   - While building UI (Task 4), start writing tests (Task 6)
   - Tests help clarify expected behavior

3. **Demo Every Day**
   - Show progress to team
   - Get feedback early
   - Iterate quickly

4. **Ship Fast**
   - Tasks 4-6 are straightforward (mostly plumbing)
   - No complex logic left
   - Aim for 3-5 day completion

---

**Phase 6B → 95%+ Workday Parity in reach with Tasks 4-6! 🚀**
