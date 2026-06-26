# Workday-Inspired Workflow & Screen Builder System - Complete Implementation Guide

## 🎯 Project Overview

This implementation provides a **Workday-like low-code platform** for Northwind database using:
- **PostgreSQL**: Workflow rules, history, and screen configurations
- **Hasura**: GraphQL API for data access and mutations
- **Go Backend**: Workflow engine, screen builder, and event routing
- **React Frontend**: Drag-drop UI for screens and workflow configuration
- **RabbitMQ**: Asynchronous event processing
- **Temporal** (optional): Advanced workflow orchestration

---

## 📦 Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    FRONTEND (React)                         │
│  ┌─────────────────────┐  ┌──────────────────────────────┐ │
│  │  Screen Builder     │  │  Workflow Designer           │ │
│  │  (Drag-Drop UI)     │  │  (Low-Code Rules)            │ │
│  └─────────────────────┘  └──────────────────────────────┘ │
└────────────────┬──────────────────────────┬─────────────────┘
                 │                          │
         ┌───────▼─────────────────────────▼───────┐
         │     HASURA GraphQL API                   │
         │  (Tenant-scoped access control)          │
         └───────┬─────────────────────────┬───────┘
                 │                         │
   ┌─────────────▼──────┐  ┌──────────────▼────────────┐
   │  GO BACKEND SERVICES│  │  PostgreSQL Database      │
   │  ┌─────────────────┤  │  ┌────────────────────┐  │
   │  │ Workflow Service├──┤  │ workflow_rules     │  │
   │  │ Screen Builder  │  │  │ workflow_history   │  │
   │  │ Event Router    │  │  │ screen_configs     │  │
   │  └─────────────────┤  │  │ workflow_templates │  │
   └─────────────────────┘  │  └────────────────────┘  │
                            └──────────────────────────┘
         ┌──────────────────────┐
         │  RabbitMQ           │
         │  (Event Queues)     │
         └──────────────────────┘
```

---

## 🗂️ File Structure

```
semlayer/
├── backend/cmd/
│   ├── workflow-service/
│   │   ├── main.go                      # Workflow service API
│   │   └── workflows.go                 # Temporal workflow definitions
│   └── screen-builder-service/
│       └── main.go                      # Screen builder service API
├── db/migrations/
│   └── 005_workflows_screens.sql        # Database schema
├── frontend/src/pages/workflows/
│   ├── ScreenBuilder.tsx                # React: Drag-drop screen builder
│   ├── ScreenBuilder.css                # Screen builder styles
│   ├── WorkflowDesigner.tsx             # React: Workflow rule designer
│   └── WorkflowDesigner.css             # Workflow designer styles
├── WORKFLOW_IMPLEMENTATION.md           # This file
└── examples/
    └── workflow-triggers.ts             # Example usage
```

---

## 🚀 Quick Start (15 Minutes)

### Step 1: Setup Database (2 min)

```bash
# Run migrations
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable < db/migrations/005_workflows_screens.sql

# Verify tables created
psql postgres://postgres:postgres@host.docker.internal:5432/alpha?sslmode=disable \
  -c "SELECT tablename FROM pg_tables WHERE tablename LIKE 'workflow%' OR tablename LIKE 'screen%'"
```

### Step 2: Start Go Services (3 min)

```bash
# Terminal 1: Workflow Service
cd backend/cmd/workflow-service
go run main.go workflows.go

# Terminal 2: Screen Builder Service
cd backend/cmd/screen-builder-service
go run main.go

# Terminal 3: Main app
cd backend/cmd/server
go run main.go
```

### Step 3: Access React UI (3 min)

```bash
cd frontend
npm start
# Navigate to: http://localhost:3000/workflows
```

### Step 4: Create Your First Screen (7 min)

1. **Go to Screen Builder**:
   - http://localhost:3000/workflows/screen-builder
   
2. **Configure Screen**:
   - Name: "Customer Details"
   - Type: "Detail View"
   - Add fields by dragging from palette
   
3. **Save Screen**:
   - Click "Save Screen"
   - Screen is created in Hasura!

---

## 📊 Core APIs

### Workflow Service (Port 8082)

#### Trigger Workflow
```bash
curl -X POST http://localhost:8082/workflow/trigger \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "workflow_name": "OrderProcessing",
    "step_name": "ApproveOrder",
    "bo_type": "orders",
    "bo_id": "550e8400-e29b-41d4-a716-446655440000",
    "form_data": {"order_total": 1500},
    "user_id": "00000000-0000-0000-0000-000000000002"
  }'
```

#### Get Workflow History
```bash
curl http://localhost:8082/workflow/history/00000000-0000-0000-0000-000000000001/orders/550e8400-e29b-41d4-a716-446655440000
```

#### Get Available Workflows
```bash
curl http://localhost:8082/workflow/templates/00000000-0000-0000-0000-000000000001
```

### Screen Builder Service (Port 8083)

#### Create Screen
```bash
curl -X POST http://localhost:8083/screens \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "bo_type": "customers",
    "screen_name": "CustomerDetails",
    "screen_type": "detail",
    "fields": [
      {"field": "name", "label": "Company Name", "type": "text", "order": 1},
      {"field": "contact", "label": "Contact", "type": "text", "order": 2}
    ],
    "actions": ["save", "delete"],
    "user_id": "00000000-0000-0000-0000-000000000002"
  }'
```

#### List Screens
```bash
curl http://localhost:8083/screens/00000000-0000-0000-0000-000000000001/customers
```

---

## 🎨 React Component Usage

### Screen Builder

```tsx
import ScreenBuilder from './pages/workflows/ScreenBuilder';

function MyApp() {
  return (
    <ScreenBuilder
      tenantId="00000000-0000-0000-0000-000000000001"
      boType="orders"
      onScreenCreated={(screenId) => console.log('Created:', screenId)}
    />
  );
}
```

### Workflow Designer

```tsx
import WorkflowDesigner from './pages/workflows/WorkflowDesigner';

function MyApp() {
  return (
    <WorkflowDesigner
      tenantId="00000000-0000-0000-0000-000000000001"
      workflowName="OrderProcessing"
      onRuleCreated={(ruleId) => console.log('Created:', ruleId)}
    />
  );
}
```

---

## 🔄 Workflow Execution Flow

```
1. Frontend calls /workflow/trigger
   ↓
2. Workflow Service receives request
   ↓
3. Fetch workflow rules from Hasura
   ↓
4. Evaluate condition against form data
   ↓
5. If condition passes:
   → Record success in workflow_history
   → Route event to actionOnSuccess queue (RabbitMQ)
   → Return success response
   ↓
   If condition fails:
   → Record failure in workflow_history
   → Route event to actionOnFailure queue (RabbitMQ)
   → Return error response
   ↓
6. RabbitMQ processes async tasks
```

---

## 💾 Database Schema

### workflow_rules

```sql
CREATE TABLE workflow_rules (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(100),     -- e.g., "OrderProcessing"
    step_name VARCHAR(100),         -- e.g., "ApproveOrder"
    step_order INTEGER,             -- Sequence in workflow
    condition_json JSONB,           -- {"and": [{"field": "...", "operator": ">=", "value": 1000}]}
    action_on_success VARCHAR(255), -- "route:order_approved.queue"
    action_on_failure VARCHAR(255), -- "notify:manager"
    error_message TEXT,
    timeout_seconds INTEGER DEFAULT 3600,
    retry_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE
);
```

### workflow_history

```sql
CREATE TABLE workflow_history (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    workflow_name VARCHAR(100),
    step_name VARCHAR(100),
    bo_type VARCHAR(50),           -- "orders", "employees", "products"
    bo_id UUID,                    -- Business Object ID
    status VARCHAR(20),            -- "success", "failure", "pending"
    details JSONB,                 -- Execution details
    user_id UUID,
    temporal_workflow_id VARCHAR(255),
    temporal_run_id VARCHAR(255),
    created_at TIMESTAMP
);
```

### screen_configs

```sql
CREATE TABLE screen_configs (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL,
    bo_type VARCHAR(50),           -- "customers", "orders"
    screen_name VARCHAR(100),      -- "CustomerDetails"
    screen_type VARCHAR(50),       -- "detail", "list", "create", "edit"
    layout_json JSONB,             -- Field definitions
    filters_json JSONB,            -- Search/filter fields
    actions_json JSONB,            -- ["save", "delete"]
    permissions_json JSONB,        -- Role-based permissions
    is_published BOOLEAN
);
```

---

## 🧪 Testing Workflows

### Create a Test Workflow Rule

```typescript
// In frontend or API tester

const createTestRule = async () => {
  const response = await fetch('/graphql', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'x-hasura-admin-secret': 'test-secret-key',
    },
    body: JSON.stringify({
      query: `
        mutation {
          insert_workflow_rules_one(object: {
            tenant_id: "00000000-0000-0000-0000-000000000001"
            workflow_name: "OrderProcessing"
            step_name: "ApproveOrder"
            step_order: 1
            condition_json: {"and": [{"field": "order_total", "operator": ">=", "value": 1000}]}
            action_on_success: "route:order_approved.queue"
            action_on_failure: "notify:manager"
            error_message: "Order total must be at least $1000"
            is_active: true
            created_by: "test-user"
          }) {
            id
            workflow_name
            step_name
          }
        }
      `,
    }),
  });

  return response.json();
};
```

### Trigger Workflow Execution

```typescript
const triggerWorkflow = async () => {
  const response = await fetch('http://localhost:8082/workflow/trigger', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      tenant_id: '00000000-0000-0000-0000-000000000001',
      workflow_name: 'OrderProcessing',
      step_name: 'ApproveOrder',
      bo_type: 'orders',
      bo_id: '550e8400-e29b-41d4-a716-446655440000',
      form_data: { order_total: 1500 },
      user_id: '00000000-0000-0000-0000-000000000002',
    }),
  });

  return response.json();
};
```

---

## 🚢 Deployment

### Docker Compose Setup

```yaml
services:
  workflow-service:
    build: ./backend/cmd/workflow-service
    ports:
      - "8082:8082"
    environment:
      - HASURA_URL=http://hasura:8080
      - HASURA_ADMIN_SECRET=test-secret-key
      - RABBITMQ_URL=amqp://rabbitmq:5672
    depends_on:
      - hasura
      - rabbitmq

  screen-builder-service:
    build: ./backend/cmd/screen-builder-service
    ports:
      - "8083:8083"
    environment:
      - HASURA_URL=http://hasura:8080
      - HASURA_ADMIN_SECRET=test-secret-key
    depends_on:
      - hasura

  rabbitmq:
    image: rabbitmq:3.12-management
    ports:
      - "5672:5672"
      - "15672:15672"
    environment:
      - RABBITMQ_DEFAULT_USER=guest
      - RABBITMQ_DEFAULT_PASS=guest

  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=postgres12
      - DB_PORT=5432
      - POSTGRES_USER=temporal
      - POSTGRES_PASSWORD=temporal
```

---

## 🔐 Security & Multi-Tenancy

All endpoints enforce tenant scoping via Hasura RLS (Row-Level Security):

```sql
-- RLS Policy: Tenant isolation
CREATE POLICY workflow_rules_tenant ON workflow_rules
  USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

Set tenant context before queries:

```sql
SET LOCAL app.current_tenant_id = '00000000-0000-0000-0000-000000000001';
SELECT * FROM workflow_rules; -- Only this tenant's rules
```

---

## 📈 Extending the System

### Add New Workflow

1. **Create rule in database**:
   ```sql
   INSERT INTO workflow_rules (...)
   VALUES ('YourWorkflow', 'YourStep', ...);
   ```

2. **Trigger via API**:
   ```typescript
   POST /workflow/trigger {
     workflow_name: 'YourWorkflow',
     step_name: 'YourStep',
     ...
   }
   ```

### Add New Screen

1. **Use ScreenBuilder UI** or API:
   ```bash
   POST /screens {
     bo_type: 'your_entity',
     screen_name: 'YourScreen',
     fields: [...]
   }
   ```

2. **Access screen**:
   - Published screens auto-populate in UI
   - Render based on `screen_configs`

---

## 🐛 Troubleshooting

### Workflow not triggering?
- Check `workflow_history` table for failed attempts
- Verify `condition_json` format
- Ensure tenant_id matches

### Screen not saving?
- Verify Hasura GraphQL endpoint is accessible
- Check browser console for errors
- Confirm screen_configs table exists

### RabbitMQ events not processing?
- Check RabbitMQ management UI: http://localhost:15672
- Verify queue names match action_on_success values
- Ensure consumer is listening

---

## 📚 Advanced Topics

### Temporal Workflow Integration

For advanced orchestration, use Temporal workflows (requires SDK installation):

```go
go get go.temporal.io/sdk
```

See `workflows.go` for example implementations.

### Custom Conditions

Extend condition evaluation in `evaluateCondition()` to support custom logic:

```go
case "custom_rule":
    return evaluateCustomRule(val, target)
```

### Event Publishing

Modify `routeEvent()` to support additional destinations:

```go
case "webhook":
    // Call HTTP endpoint
case "email":
    // Send email notification
```

---

## 📞 Support

For issues or questions:
1. Check workflow_history table for execution logs
2. Review Go service logs on ports 8082-8083
3. Verify Hasura GraphQL at http://localhost:8080/console
4. Monitor RabbitMQ at http://localhost:15672 (guest:guest)

---

## ✅ Checklist: Deployment

- [ ] Database migrations applied
- [ ] Hasura GraphQL configured
- [ ] Workflow service running (port 8082)
- [ ] Screen builder service running (port 8083)
- [ ] RabbitMQ running (port 5672)
- [ ] React frontend built and running
- [ ] Tenant context set in localStorage
- [ ] Test workflow created
- [ ] Test screen created
- [ ] RabbitMQ queues consuming events

**🎉 You're ready to use Workday-style workflows!**
