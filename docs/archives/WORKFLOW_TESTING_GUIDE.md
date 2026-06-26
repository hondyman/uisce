# Testing & Integration Guide - Workflows & Screens

## Quick Test (5 minutes)

### 1. Verify Services Running

```bash
# Workflow Service Health
curl http://localhost:8082/health

# Screen Builder Health
curl http://localhost:8083/health

# Hasura GraphQL
curl http://localhost:8080/v1/graphql \
  -H "x-hasura-admin-secret: test-secret-key" \
  -d '{"query": "{ __typename }"}'
```

Expected responses:
```json
{"status":"ok"}
{"status":"ok"}
{"__typename":"Query"}
```

---

## Test Scenario 1: Order Processing Workflow (10 min)

### 1.1 Create Workflow Rule

```bash
curl -X POST http://localhost:8080/v1/graphql \
  -H "x-hasura-admin-secret: test-secret-key" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "mutation { insert_workflow_rules_one(object: {tenant_id: \"00000000-0000-0000-0000-000000000001\", workflow_name: \"OrderProcessing\", step_name: \"ApproveOrder\", step_order: 1, condition_json: {\"and\": [{\"field\": \"order_total\", \"operator\": \">=\", \"value\": 1000}]}, action_on_success: \"route:order_approved.queue\", action_on_failure: \"notify:manager\", error_message: \"Order total must be at least $1000\", is_active: true, created_by: \"test-user\"}) { id } }"
  }'
```

### 1.2 Trigger Workflow - Success Case

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

Expected:
```json
{
  "status": "success",
  "history_id": "...",
  "message": "Workflow step ApproveOrder completed successfully",
  "next_action": "route:order_approved.queue"
}
```

### 1.3 Trigger Workflow - Failure Case

```bash
curl -X POST http://localhost:8082/workflow/trigger \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "workflow_name": "OrderProcessing",
    "step_name": "ApproveOrder",
    "bo_type": "orders",
    "bo_id": "550e8400-e29b-41d4-a716-446655440001",
    "form_data": {"order_total": 500},
    "user_id": "00000000-0000-0000-0000-000000000002"
  }'
```

Expected:
```json
{
  "status": "failed",
  "history_id": "...",
  "error": "Order total must be at least $1000",
  "message": "Workflow step condition not satisfied"
}
```

### 1.4 Check Workflow History

```bash
curl http://localhost:8082/workflow/history/00000000-0000-0000-0000-000000000001/orders/550e8400-e29b-41d4-a716-446655440000 \
  | jq
```

Expected to see execution records.

---

## Test Scenario 2: Screen Builder (10 min)

### 2.1 Create Customer Screen

```bash
curl -X POST http://localhost:8083/screens \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "00000000-0000-0000-0000-000000000001",
    "bo_type": "customers",
    "screen_name": "CustomerDetails",
    "screen_type": "detail",
    "fields": [
      {"field": "company_name", "label": "Company", "type": "text", "order": 1, "required": true, "searchable": true, "editable": true},
      {"field": "contact_name", "label": "Contact", "type": "text", "order": 2, "required": true, "searchable": false, "editable": true},
      {"field": "address", "label": "Address", "type": "text", "order": 3, "required": false, "searchable": false, "editable": true},
      {"field": "phone", "label": "Phone", "type": "text", "order": 4, "required": false, "searchable": false, "editable": true}
    ],
    "filters": [
      {"field": "company_name", "label": "Search by Company", "type": "text"}
    ],
    "actions": ["save", "delete", "cancel"],
    "permissions": {"admin": ["save", "delete"], "user": ["save"]},
    "user_id": "00000000-0000-0000-0000-000000000002"
  }'
```

Expected:
```json
{
  "id": "...",
  "message": "Screen CustomerDetails created successfully"
}
```

### 2.2 List Screens

```bash
curl http://localhost:8083/screens/00000000-0000-0000-0000-000000000001/customers
```

Expected to see the created screen.

### 2.3 Get Screen Details

```bash
curl http://localhost:8083/screens/00000000-0000-0000-0000-000000000001/<screen_id>
```

### 2.4 Update Screen

```bash
curl -X PUT http://localhost:8083/screens/00000000-0000-0000-0000-000000000001/<screen_id> \
  -H "Content-Type: application/json" \
  -d '{
    "screen_name": "CustomerDetailsV2",
    "fields": [
      {"field": "company_name", "label": "Company Name", "type": "text", "order": 1},
      {"field": "email", "label": "Email", "type": "text", "order": 2}
    ]
  }'
```

### 2.5 Publish Screen

```bash
curl -X POST http://localhost:8083/screens/00000000-0000-0000-0000-000000000001/<screen_id>/publish
```

---

## Test Scenario 3: React UI (15 min)

### 3.1 Access Screen Builder

```bash
# Open in browser
http://localhost:3000/workflows/screen-builder
```

### 3.2 Create Screen via UI

1. Enter Screen Name: "OrderForm"
2. Select Type: "Create Form"
3. Drag fields from palette to preview
4. Check "save" and "approve" actions
5. Click "Save Screen"
6. Verify success message

### 3.3 Access Workflow Designer

```bash
http://localhost:3000/workflows/workflow-designer
```

### 3.4 Create Workflow via UI

1. Enter Step Name: "ValidateOrder"
2. Enter Step Order: 1
3. Select Field: "order_total"
4. Select Operator: ">="
5. Enter Value: "100"
6. Enter On Success: "route:order_validated.queue"
7. Enter On Failure: "notify:validation_failed"
8. Enter Error Message: "Order must be at least $100"
9. Click "Create Workflow Step"
10. Verify step appears in list

---

## Redpanda (Kafka) Event Testing (10 min)

> Note: RabbitMQ has been replaced by Redpanda in the local compose files. Some test helpers or code may still reference RabbitMQ and will need updating to use Kafka topics or a compatibility adapter.

### 3.1 Access Redpanda (Pandaproxy / broker)

Use `http://localhost:8082` for Pandaproxy (if enabled) and `localhost:9092` for the Kafka broker. Redpanda does not provide a RabbitMQ-style management UI.

```bash
http://localhost:15672
# Login: guest / guest
```

### 3.2 Create Test Queues

Via Management UI:
1. Go to Queues tab
2. Create queue: `order_approved.queue`
3. Create queue: `order_failed.queue`
4. Create queue: `manager_approval.queue`

### 3.3 Verify Events

1. Trigger workflow with success condition
2. Check `order_approved.queue` - should have message
3. Trigger workflow with failure condition
4. Check `order_failed.queue` - should have message

View message content in "Get Messages" section.

---

## End-to-End Workflow Test (30 min)

```
Setup:
  ├─ Create 3 workflow rules (ApproveOrder, ValidateOrder, NotifyShipping)
  ├─ Create 2 screens (OrderForm, OrderList)
  └─ Create 4 RabbitMQ queues

Execute:
  ├─ User fills OrderForm in React UI
  ├─ Form submits to /workflow/trigger
  ├─ Workflow Service validates and executes steps
  ├─ Events route to RabbitMQ queues
  ├─ Events consumed by downstream services
  └─ Check workflow_history table for audit trail

Verify:
  ├─ workflow_history table has records
  ├─ RabbitMQ queues have messages
  ├─ Screen appears in UI
  ├─ Workflow rules are evaluated correctly
  └─ Multi-tenant isolation works
```

### Commands

```bash
# 1. Setup
psql -c "SELECT COUNT(*) FROM workflow_rules WHERE is_active = true;"

# 2. Execute
curl -X POST http://localhost:8082/workflow/trigger ...

# 3. Verify
psql -c "SELECT COUNT(*) FROM workflow_history WHERE status = 'success';"
psql -c "SELECT * FROM workflow_history ORDER BY created_at DESC LIMIT 5;"
```

---

## Performance Testing (Optional)

### Load Test with Apache Bench

```bash
# Single workflow trigger
ab -n 100 -c 10 http://localhost:8082/health

# Create 100 workflow rules
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/graphql \
    -H "x-hasura-admin-secret: test-secret-key" \
    -H "Content-Type: application/json" \
    -d "{\"query\": \"mutation { insert_workflow_rules_one(...) { id } }\"}"
done
```

### Monitor Performance

```bash
# Check Postgres connections
psql -c "SELECT count(*) FROM pg_stat_activity;"

# Check RabbitMQ queue depth
curl -u guest:guest http://localhost:15672/api/queues
```

---

## Troubleshooting

### Workflow not triggering?

```bash
# 1. Check service logs
docker logs <workflow-service-container>

# 2. Verify GraphQL query
curl -H "x-hasura-admin-secret: test-secret-key" \
  http://localhost:8080/v1/graphql -d '{"query": "{ workflow_rules { id workflow_name } }"}'

# 3. Check workflow_history for errors
psql -c "SELECT * FROM workflow_history WHERE status = 'failure' ORDER BY created_at DESC LIMIT 1;"
```

### Screen not saving?

```bash
# 1. Check network tab in browser DevTools
# 2. Verify Hasura endpoint
curl http://localhost:8080/health

# 3. Check screen_configs table
psql -c "SELECT COUNT(*) FROM screen_configs;"
```

### RabbitMQ not routing events?

```bash
# 1. Verify connection
curl -u guest:guest http://localhost:15672/api/connections

# 2. Check queues
curl -u guest:guest http://localhost:15672/api/queues

# 3. Monitor broker
docker logs redpanda
```

---

## Success Checklist

- [ ] All services healthy
- [ ] Workflow rule created
- [ ] Workflow triggered successfully
- [ ] Workflow history recorded
- [ ] Screen created via API
- [ ] Screen appears in React UI
- [ ] Drag-drop screen builder works
- [ ] Workflow designer creates rules
- [ ] RabbitMQ receives events
- [ ] Multi-tenant isolation verified
- [ ] Tenant context applied correctly
- [ ] Error handling works

**✅ System Ready for Development!**
