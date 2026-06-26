# BP Validation HTTP API Reference

## Overview

The BP Validation system exposes RESTful endpoints for:
- Validating business process steps synchronously
- Queuing async validations
- Managing rules (CRUD)
- Monitoring validation metrics
- Subscribing to validation events

All endpoints require tenant scope: `X-Tenant-ID` header or `?tenant_id=` query parameter.

---

## Endpoints

### 1. Synchronous BP Validation

**Validate a business process step immediately**

```http
POST /api/validations/validate
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "bp_name": "ChangeMaritalStatus",
  "step_name": "Submit",
  "form_data": {
    "age": 25,
    "marital_status": "married"
  },
  "user_id": "user-456",
  "context_id": "ctx-789"
}
```

**Response (200 OK):**
```json
{
  "id": "val_1729258341234567890",
  "passed": true,
  "errors": [],
  "warnings": [],
  "actions_to_take": ["route:hr_updates.queue"],
  "details": {
    "bp_name": "ChangeMaritalStatus",
    "step_name": "Submit",
    "rule_count": 2,
    "evaluation_ms": 12
  },
  "timestamp": "2025-10-18T22:39:01Z"
}
```

**Response (400 Bad Request):**
```json
{
  "passed": false,
  "errors": [
    "Age must be at least 18 for married status"
  ],
  "actions_to_take": ["route:validation_errors.queue"],
  "id": "val_1729258341234567890",
  "timestamp": "2025-10-18T22:39:01Z"
}
```

---

### 2. Asynchronous BP Validation

**Queue validation for background processing**

```http
POST /api/validations/queue-async
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "bp_name": "ApproveTimesheet",
  "step_name": "Submit",
  "form_data": {
    "hours": 40,
    "department": "Engineering"
  },
  "user_id": "user-456",
  "context_id": "ctx-789"
}
```

**Response (202 Accepted):**
```json
{
  "validation_id": "bpval_1729258341234567890",
  "status": "queued",
  "message": "Validation queued for async processing",
  "check_result_url": "/api/validations/result/bpval_1729258341234567890"
}
```

---

### 3. Get Validation Result

**Retrieve result of async validation**

```http
GET /api/validations/result/{validation_id}
X-Tenant-ID: {tenant_id}
```

**Response (200 OK - Complete):**
```json
{
  "id": "bpval_1729258341234567890",
  "passed": true,
  "errors": [],
  "warnings": [],
  "actions_to_take": ["route:approvals.queue"],
  "status": "completed",
  "result_available_at": "2025-10-18T22:39:15Z"
}
```

**Response (202 Accepted - Still Processing):**
```json
{
  "id": "bpval_1729258341234567890",
  "status": "processing",
  "message": "Validation still being processed"
}
```

**Response (404 Not Found):**
```json
{
  "error": "Validation result not found",
  "id": "bpval_1729258341234567890"
}
```

---

### 4. Create/Store Rule

**Define a new validation rule**

```http
POST /api/rules
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "bp_name": "ChangeMaritalStatus",
  "step_name": "Submit",
  "condition_json": {
    "and": [
      {"field": "marital_status", "operator": "=", "value": "married"},
      {"field": "age", "operator": ">=", "value": 18}
    ]
  },
  "action_on_success": "route:hr_updates.queue",
  "action_on_failure": "route:validation_errors.queue",
  "error_message": "Age must be at least 18 for married status",
  "priority": 1,
  "enabled": true
}
```

**Response (201 Created):**
```json
{
  "id": "rule_1729258341234567890",
  "bp_name": "ChangeMaritalStatus",
  "step_name": "Submit",
  "created_at": "2025-10-18T22:39:01Z",
  "message": "Rule created successfully"
}
```

---

### 5. Get Rules for BP Step

**Fetch all rules for a business process step**

```http
GET /api/rules?bp_name=ChangeMaritalStatus&step_name=Submit
X-Tenant-ID: {tenant_id}
```

**Response (200 OK):**
```json
{
  "rules": [
    {
      "id": "rule_1729258341234567890",
      "bp_name": "ChangeMaritalStatus",
      "step_name": "Submit",
      "condition_json": {
        "and": [
          {"field": "marital_status", "operator": "=", "value": "married"},
          {"field": "age", "operator": ">=", "value": 18}
        ]
      },
      "action_on_success": "route:hr_updates.queue",
      "error_message": "Age must be at least 18 for married status",
      "priority": 1,
      "enabled": true,
      "created_at": "2025-10-18T22:39:01Z"
    }
  ],
  "count": 1
}
```

---

### 6. Update Rule

**Modify an existing rule**

```http
PUT /api/rules/{rule_id}
Content-Type: application/json
X-Tenant-ID: {tenant_id}

{
  "condition_json": {
    "and": [
      {"field": "marital_status", "operator": "=", "value": "married"},
      {"field": "age", "operator": ">=", "value": 21}
    ]
  },
  "error_message": "Age must be at least 21 for married status",
  "enabled": true
}
```

**Response (200 OK):**
```json
{
  "id": "rule_1729258341234567890",
  "message": "Rule updated successfully",
  "updated_at": "2025-10-18T22:40:00Z"
}
```

---

### 7. Delete Rule

**Remove a validation rule**

```http
DELETE /api/rules/{rule_id}
X-Tenant-ID: {tenant_id}
```

**Response (200 OK):**
```json
{
  "id": "rule_1729258341234567890",
  "message": "Rule deleted successfully"
}
```

**Response (404 Not Found):**
```json
{
  "error": "Rule not found",
  "id": "rule_1729258341234567890"
}
```

---

### 8. Get Validation Audit History

**Retrieve validation execution history for audit/compliance**

```http
GET /api/validations/history?bp_name=ChangeMaritalStatus&days=7
X-Tenant-ID: {tenant_id}
```

**Response (200 OK):**
```json
{
  "executions": [
    {
      "id": "exec_1729258341234567890",
      "rule_id": "rule_1729258341234567890",
      "bp_name": "ChangeMaritalStatus",
      "step_name": "Submit",
      "result_passed": true,
      "error_message": null,
      "action_taken": "route:hr_updates.queue",
      "executed_by": "user-456",
      "executed_at": "2025-10-18T22:39:01Z"
    }
  ],
  "count": 47,
  "period": "2025-10-11 to 2025-10-18"
}
```

---

### 9. Get Validation Metrics

**Retrieve validation performance and success metrics**

```http
GET /api/validations/metrics?period=day
X-Tenant-ID: {tenant_id}
```

**Response (200 OK):**
```json
{
  "period": "2025-10-18",
  "total_validations": 152,
  "passed": 134,
  "failed": 18,
  "success_rate": 88.16,
  "average_evaluation_time_ms": 12.4,
  "by_bp": {
    "ChangeMaritalStatus": {
      "total": 45,
      "passed": 43,
      "failed": 2
    },
    "ApproveTimesheet": {
      "total": 107,
      "passed": 91,
      "failed": 16
    }
  },
  "top_failures": [
    {
      "error": "Age must be at least 18 for married status",
      "count": 8
    },
    {
      "error": "Hours must be <= 50 per week",
      "count": 6
    }
  ]
}
```

---

### 10. Subscribe to Validation Events (WebSocket)

**Real-time validation event stream**

```http
GET /ws/validations?bp_name=ChangeMaritalStatus&step_name=Submit
X-Tenant-ID: {tenant_id}

Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: ...
```

**Messages (as JSON):**
```json
{
  "type": "validation_complete",
  "validation_id": "bpval_1729258341234567890",
  "bp_name": "ChangeMaritalStatus",
  "step_name": "Submit",
  "passed": true,
  "errors": [],
  "timestamp": "2025-10-18T22:39:15Z"
}
```

---

## Operators Reference

### Comparison Operators
- `=` / `==` : Equality
- `!=` / `<>` : Not equal
- `>` : Greater than
- `<` : Less than
- `>=` : Greater than or equal
- `<=` : Less than or equal

### String Operators
- `contains` : String contains substring
- `startsWith` : String starts with prefix
- `endsWith` : String ends with suffix
- `regex` : Matches regular expression pattern

### Collection Operators
- `in` : Value is in list (comma-separated or array)

### Null Operators
- `isEmpty` : Value is null, empty string, or 0
- `isNotEmpty` : Value is not null/empty

### Range Operators
- `between` : Value between min and max (requires `{"min": 0, "max": 100}`)

---

## Error Codes

| Code | Status | Meaning |
|------|--------|---------|
| 200 | OK | Validation completed successfully |
| 202 | Accepted | Async validation queued |
| 400 | Bad Request | Validation failed or invalid input |
| 401 | Unauthorized | Missing/invalid authentication |
| 403 | Forbidden | Tenant isolation violation |
| 404 | Not Found | Resource not found |
| 422 | Unprocessable Entity | Invalid rule condition JSON |
| 500 | Internal Server Error | Server error |

---

## Example: Complete Flow

### Step 1: Create Marital Status Validation Rule

```bash
curl -X POST http://localhost:8080/api/rules \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -d '{
    "bp_name": "ChangeMaritalStatus",
    "step_name": "Submit",
    "condition_json": {
      "and": [
        {"field": "marital_status", "operator": "=", "value": "married"},
        {"field": "age", "operator": ">=", "value": 18}
      ]
    },
    "action_on_success": "route:hr_updates.queue",
    "action_on_failure": "route:validation_errors.queue",
    "error_message": "Age must be at least 18 for married status",
    "priority": 1
  }'
```

### Step 2: User Submits Form (Sync Validation)

```bash
curl -X POST http://localhost:8080/api/validations/validate \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: tenant-123" \
  -d '{
    "bp_name": "ChangeMaritalStatus",
    "step_name": "Submit",
    "form_data": {"age": 25, "marital_status": "married"},
    "user_id": "user-456"
  }'
```

### Step 3: Response (Validation Passed, Action Routed)

```json
{
  "passed": true,
  "actions_to_take": ["route:hr_updates.queue"],
  "id": "val_1729258341234567890"
}
```

### Step 4: RabbitMQ Queue Receives Event

HR workflow triggered, HR team notified, business process continues.

---

## Implementation in Go Handler

```go
import "github.com/gin-gonic/gin"

func validateBPStep(c *gin.Context) {
    var req services.BPValidationRequest
    if err := c.BindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }

    req.TenantID = c.GetHeader("X-Tenant-ID")
    req.UserID = c.GetString("user_id")
    req.ReturnSync = true

    response, err := bpCoordinator.ValidateBPStep(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    if !response.Passed {
        c.JSON(400, response)
    } else {
        c.JSON(200, response)
    }
}

// Register routes
r.POST("/api/validations/validate", validateBPStep)
```

---

## Rate Limiting Recommendations

- **Per tenant, per minute**: 1,000 validation requests
- **Per user, per minute**: 100 validation requests
- **Async queue depth**: 10,000 pending tasks

---

## Monitoring Queries

```sql
-- Validation success rate by BP (last 7 days)
SELECT bp_name, 
       COUNT(*) as total,
       SUM(CASE WHEN result_passed THEN 1 ELSE 0 END) as passed,
       ROUND(100.0 * SUM(CASE WHEN result_passed THEN 1 ELSE 0 END) / COUNT(*), 2) as success_rate
FROM bp_validation_executions
WHERE executed_at > NOW() - INTERVAL '7 days'
GROUP BY bp_name
ORDER BY success_rate ASC;

-- Top failure reasons
SELECT error_message, COUNT(*) as count
FROM bp_validation_executions
WHERE result_passed = FALSE
  AND executed_at > NOW() - INTERVAL '1 day'
GROUP BY error_message
ORDER BY count DESC
LIMIT 10;

-- Average validation time by BP step
SELECT bp_name, step_name, ROUND(AVG(execution_time_ms), 2) as avg_ms
FROM bp_validation_executions
WHERE executed_at > NOW() - INTERVAL '24 hours'
GROUP BY bp_name, step_name
ORDER BY avg_ms DESC;
```

---

This API is ready for integration with your Workday-like low-code designer UI in Phase 5c.
