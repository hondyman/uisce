# Phase 6C: Timeout Triggers API Integration Guide

## Quick Integration Checklist

This guide provides step-by-step instructions to fully integrate the Workday timeout triggers into the Semlayer platform.

### What You'll Complete
- ✅ Backend: Start TimeoutMonitor service on server startup
- ✅ Backend: Implement 5 REST API endpoints for trigger CRUD + test
- ✅ Frontend: Connect UI component to backend API
- ✅ Testing: Verify end-to-end timeout escalation

**Estimated Time: 45 minutes**

---

## Part 1: Backend Integration (15 minutes)

### Step 1A: Start TimeoutMonitor Service

**File:** `backend/cmd/server/main.go`

Find the database initialization section and add:

```go
// After database connection is established
db, err := initDB()
if err != nil {
    logger.Fatal("Database connection failed:", err)
}
defer db.Close()

// ADD THESE LINES:
timeout := temporal.NewTimeoutMonitor(db)
go timeout.Start(context.Background())
logger.Info("Timeout monitor service started successfully")

// Continue with rest of startup...
```

**Expected Output:**
```
INFO: Timeout monitor service started successfully
```

---

### Step 1B: Add API Endpoints

**File:** `backend/internal/api/api.go`

Add these handlers to your API struct:

```go
import (
    "database/sql"
    "encoding/json"
    "github.com/gin-gonic/gin"
    "github.com/lib/pq"
    "time"
)

// TimeoutTrigger represents a workflow timeout trigger
type TimeoutTrigger struct {
    ID                    string                 `json:"id"`
    TenantID              string                 `json:"tenant_id"`
    WorkflowName          string                 `json:"workflow_name" binding:"required"`
    StepName              string                 `json:"step_name" binding:"required"`
    DueHours              int                    `json:"due_hours" binding:"required,min=1,max=999"`
    TriggerPercentages    []int                  `json:"trigger_percentages"`
    Actions               []map[string]interface{} `json:"actions" binding:"required"`
    IsActive              bool                   `json:"is_active"`
    CreatedAt             time.Time              `json:"created_at"`
    UpdatedAt             time.Time              `json:"updated_at"`
}

// Register routes
func (api *API) RegisterTimeoutTriggerRoutes(r *gin.Engine) {
    group := r.Group("/api/workflow-timeout-triggers")
    
    group.GET("", api.listTimeoutTriggers)
    group.POST("", api.createTimeoutTrigger)
    group.PUT("/:id", api.updateTimeoutTrigger)
    group.DELETE("/:id", api.deleteTimeoutTrigger)
    group.POST("/:id/test", api.testTimeoutTrigger)
}

// GET /api/workflow-timeout-triggers
func (api *API) listTimeoutTriggers(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    
    query := `
        SELECT id, tenant_id, workflow_name, step_name, due_hours,
               trigger_percentages, actions_json, is_active,
               created_at, updated_at
        FROM workflow_timeout_triggers
        WHERE tenant_id = $1 AND is_active = true
        ORDER BY workflow_name, step_name
    `
    
    rows, err := api.db.Query(query, tenantID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to list triggers"})
        return
    }
    defer rows.Close()
    
    triggers := []TimeoutTrigger{}
    for rows.Next() {
        var t TimeoutTrigger
        var percentages []byte
        var actions []byte
        
        err := rows.Scan(
            &t.ID, &t.TenantID, &t.WorkflowName, &t.StepName,
            &t.DueHours, &percentages, &actions, &t.IsActive,
            &t.CreatedAt, &t.UpdatedAt,
        )
        if err != nil {
            continue
        }
        
        json.Unmarshal(percentages, &t.TriggerPercentages)
        json.Unmarshal(actions, &t.Actions)
        
        triggers = append(triggers, t)
    }
    
    c.JSON(200, triggers)
}

// POST /api/workflow-timeout-triggers
func (api *API) createTimeoutTrigger(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    
    var trigger TimeoutTrigger
    if err := c.BindJSON(&trigger); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request body"})
        return
    }
    
    trigger.TenantID = tenantID
    trigger.IsActive = true
    trigger.CreatedAt = time.Now()
    trigger.UpdatedAt = time.Now()
    
    if trigger.TriggerPercentages == nil {
        trigger.TriggerPercentages = []int{80, 100}
    }
    
    actionsJSON, _ := json.Marshal(trigger.Actions)
    percentsJSON, _ := json.Marshal(trigger.TriggerPercentages)
    
    query := `
        INSERT INTO workflow_timeout_triggers
        (tenant_id, workflow_name, step_name, due_hours,
         trigger_percentages, actions_json, is_active, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `
    
    var id string
    err := api.db.QueryRow(
        query,
        tenantID, trigger.WorkflowName, trigger.StepName, trigger.DueHours,
        percentsJSON, actionsJSON, true, time.Now(), time.Now(),
    ).Scan(&id)
    
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to create trigger"})
        return
    }
    
    trigger.ID = id
    c.JSON(201, trigger)
}

// PUT /api/workflow-timeout-triggers/:id
func (api *API) updateTimeoutTrigger(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    id := c.Param("id")
    
    var trigger TimeoutTrigger
    if err := c.BindJSON(&trigger); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request body"})
        return
    }
    
    trigger.UpdatedAt = time.Now()
    
    actionsJSON, _ := json.Marshal(trigger.Actions)
    percentsJSON, _ := json.Marshal(trigger.TriggerPercentages)
    
    query := `
        UPDATE workflow_timeout_triggers
        SET workflow_name = $1, step_name = $2, due_hours = $3,
            trigger_percentages = $4, actions_json = $5,
            is_active = $6, updated_at = $7
        WHERE id = $8 AND tenant_id = $9
    `
    
    result, err := api.db.Exec(
        query,
        trigger.WorkflowName, trigger.StepName, trigger.DueHours,
        percentsJSON, actionsJSON, trigger.IsActive,
        time.Now(), id, tenantID,
    )
    
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to update trigger"})
        return
    }
    
    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(404, gin.H{"error": "Trigger not found"})
        return
    }
    
    c.JSON(200, trigger)
}

// DELETE /api/workflow-timeout-triggers/:id
func (api *API) deleteTimeoutTrigger(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    id := c.Param("id")
    
    query := `
        UPDATE workflow_timeout_triggers
        SET is_active = false, updated_at = $1
        WHERE id = $2 AND tenant_id = $3
    `
    
    result, err := api.db.Exec(query, time.Now(), id, tenantID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to delete trigger"})
        return
    }
    
    rows, _ := result.RowsAffected()
    if rows == 0 {
        c.JSON(404, gin.H{"error": "Trigger not found"})
        return
    }
    
    c.JSON(200, gin.H{"message": "Trigger deleted successfully"})
}

// POST /api/workflow-timeout-triggers/:id/test
func (api *API) testTimeoutTrigger(c *gin.Context) {
    tenantID := c.GetString("tenant_id")
    id := c.Param("id")
    
    // Get the trigger
    query := `
        SELECT actions_json FROM workflow_timeout_triggers
        WHERE id = $1 AND tenant_id = $2
    `
    
    var actionsJSON []byte
    err := api.db.QueryRow(query, id, tenantID).Scan(&actionsJSON)
    if err == sql.ErrNoRows {
        c.JSON(404, gin.H{"error": "Trigger not found"})
        return
    }
    if err != nil {
        c.JSON(500, gin.H{"error": "Database error"})
        return
    }
    
    var actions []map[string]interface{}
    json.Unmarshal(actionsJSON, &actions)
    
    // Log test execution
    for _, action := range actions {
        actionType := action["type"].(string)
        target := action["target"].(string)
        
        // Record in audit log
        auditQuery := `
            INSERT INTO audit_events (tenant_id, action, details, created_at)
            VALUES ($1, 'TIMEOUT_TRIGGER_TEST', $2, $3)
        `
        
        details := map[string]string{
            "trigger_id": id,
            "type":       actionType,
            "target":     target,
        }
        
        detailsJSON, _ := json.Marshal(details)
        api.db.Exec(auditQuery, tenantID, detailsJSON, time.Now())
    }
    
    c.JSON(200, gin.H{
        "message": "Test executed successfully",
        "actions": len(actions),
    })
}
```

**Register in Router:**

Find your router initialization and add:

```go
// In your main route setup
api := NewAPI(db)
api.RegisterTimeoutTriggerRoutes(router)
```

---

## Part 2: Frontend API Integration (20 minutes)

### Step 2A: Update API Calls

**File:** `frontend/src/pages/WorkflowTimeoutTriggersPage.tsx`

Replace the mock data section with actual API calls:

```tsx
const fetchTriggers = async () => {
  setLoading(true);
  try {
    const tenantId = localStorage.getItem('selected_tenant');
    const datasourceId = localStorage.getItem('selected_datasource');
    
    if (!tenantId || !datasourceId) {
      message.error('Please select a tenant first');
      return;
    }
    
    const response = await fetch('/api/workflow-timeout-triggers', {
      headers: {
        'X-Tenant-ID': JSON.parse(tenantId).id,
        'X-Tenant-Datasource-ID': JSON.parse(datasourceId).id,
      },
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    const data = await response.json();
    setTriggers(data || []);
  } catch (error) {
    console.error('Failed to load triggers:', error);
    message.error('Failed to load timeout triggers');
  } finally {
    setLoading(false);
  }
};

const handleSave = async () => {
  try {
    const values = await form.validateFields();
    
    const tenantId = localStorage.getItem('selected_tenant');
    const datasourceId = localStorage.getItem('selected_datasource');
    
    if (!tenantId || !datasourceId) {
      message.error('Please select a tenant first');
      return;
    }
    
    const newTrigger: TimeoutTrigger = {
      workflow_name: values.workflow,
      step_name: values.step,
      due_hours: values.due_hours,
      actions: actions,
      is_active: true,
    };

    setLoading(true);
    
    const headers = {
      'Content-Type': 'application/json',
      'X-Tenant-ID': JSON.parse(tenantId).id,
      'X-Tenant-Datasource-ID': JSON.parse(datasourceId).id,
    };

    if (editing?.id) {
      const response = await fetch(
        `/api/workflow-timeout-triggers/${editing.id}`,
        {
          method: 'PUT',
          headers,
          body: JSON.stringify(newTrigger),
        }
      );

      if (!response.ok) throw new Error('Update failed');
      
      setTriggers(triggers.map(t => t.id === editing.id ? { ...newTrigger, id: editing.id } : t));
      message.success('Timeout trigger updated');
    } else {
      const response = await fetch('/api/workflow-timeout-triggers', {
        method: 'POST',
        headers,
        body: JSON.stringify(newTrigger),
      });

      if (!response.ok) throw new Error('Create failed');
      
      const created = await response.json();
      setTriggers([...triggers, created]);
      message.success('Timeout trigger created');
    }

    form.resetFields();
    setActions([
      { percent: 80, type: 'notify', target: 'assignee', message: '' },
      { percent: 100, type: 'escalate', target: 'hr_director', message: '' },
    ]);
    setEditing(null);
  } catch (error) {
    console.error('Save error:', error);
    message.error('Failed to save timeout trigger');
  } finally {
    setLoading(false);
  }
};

const handleDelete = (id?: string) => {
  Modal.confirm({
    title: 'Delete Timeout Trigger',
    content: 'Are you sure you want to delete this timeout trigger?',
    okText: 'Yes',
    cancelText: 'No',
    onOk: async () => {
      try {
        const tenantId = localStorage.getItem('selected_tenant');
        const datasourceId = localStorage.getItem('selected_datasource');
        
        const response = await fetch(`/api/workflow-timeout-triggers/${id}`, {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': JSON.parse(tenantId).id,
            'X-Tenant-Datasource-ID': JSON.parse(datasourceId).id,
          },
        });

        if (!response.ok) throw new Error('Delete failed');
        
        setTriggers(triggers.filter(t => t.id !== id));
        message.success('Timeout trigger deleted');
      } catch (error) {
        console.error('Delete error:', error);
        message.error('Failed to delete timeout trigger');
      }
    },
  });
};

const handleTestTrigger = (trigger: TimeoutTrigger) => {
  Modal.confirm({
    title: 'Test Timeout Trigger',
    content: `This will simulate a timeout for ${trigger.workflow_name}.${trigger.step_name}. Continue?`,
    okText: 'Yes',
    cancelText: 'No',
    onOk: async () => {
      try {
        const tenantId = localStorage.getItem('selected_tenant');
        const datasourceId = localStorage.getItem('selected_datasource');
        
        const response = await fetch(
          `/api/workflow-timeout-triggers/${trigger.id}/test`,
          {
            method: 'POST',
            headers: {
              'X-Tenant-ID': JSON.parse(tenantId).id,
              'X-Tenant-Datasource-ID': JSON.parse(datasourceId).id,
            },
          }
        );

        if (!response.ok) throw new Error('Test failed');
        
        const result = await response.json();
        message.success(
          `Timeout trigger tested: ${trigger.actions.map(a => a.type).join(', ')}`
        );
      } catch (error) {
        console.error('Test error:', error);
        message.error('Failed to test timeout trigger');
      }
    },
  });
};
```

---

## Part 3: Testing (10 minutes)

### Step 3A: Test API Endpoints

```bash
# 1. Test GET (list triggers)
curl -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
     "http://localhost:8080/api/workflow-timeout-triggers"

# Expected Response:
# [
#   {
#     "id": "...",
#     "workflow_name": "HireEmployee",
#     "step_name": "ManagerApproval",
#     "due_hours": 48,
#     "actions": [...]
#   },
#   ...
# ]

# 2. Test POST (create trigger)
curl -X POST \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "TestWorkflow",
    "step_name": "TestStep",
    "due_hours": 24,
    "actions": [
      {"percent": 100, "type": "log", "target": "audit", "message": "Test"}
    ]
  }' \
  "http://localhost:8080/api/workflow-timeout-triggers"

# 3. Test PUT (update trigger)
curl -X PUT \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "Content-Type: application/json" \
  -d '{
    "workflow_name": "TestWorkflow",
    "step_name": "TestStep",
    "due_hours": 48,
    "actions": [...]
  }' \
  "http://localhost:8080/api/workflow-timeout-triggers/[TRIGGER_ID]"

# 4. Test DELETE
curl -X DELETE \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  "http://localhost:8080/api/workflow-timeout-triggers/[TRIGGER_ID]"

# 5. Test timeout trigger execution
curl -X POST \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  "http://localhost:8080/api/workflow-timeout-triggers/[TRIGGER_ID]/test"
```

### Step 3B: E2E Timeout Test

```sql
-- 1. Create a test workflow instance
INSERT INTO workflow_instances (
  id, tenant_id, workflow_name, step_name, assignee, status, step_start, created_at
) VALUES (
  'test-instance-123',
  '00000000-0000-0000-0000-000000000000',
  'HireEmployee',
  'ManagerApproval',
  'manager@example.com',
  'in_progress',
  NOW() - INTERVAL '48 hours',  -- Started 48 hours ago
  NOW()
);

-- 2. Verify TimeoutMonitor will pick it up
SELECT elapsed_hours, due_hours, 
       (elapsed_hours / due_hours * 100)::int as percent
FROM (
  SELECT EXTRACT(EPOCH FROM (NOW() - step_start)) / 3600 as elapsed_hours,
         48 as due_hours
) sub;
-- Expected: elapsed_hours ≈ 48, percent = 100

-- 3. Check timeout trigger configuration
SELECT * FROM workflow_timeout_triggers 
WHERE workflow_name = 'HireEmployee' 
AND step_name = 'ManagerApproval';

-- 4. After TimeoutMonitor runs, verify escalation
SELECT * FROM audit_events 
WHERE tenant_id = '00000000-0000-0000-0000-000000000000'
ORDER BY created_at DESC LIMIT 5;

-- 5. Verify workflow reassigned
SELECT id, workflow_name, step_name, assignee, status, updated_at
FROM workflow_instances
WHERE id = 'test-instance-123';
-- Expected: assignee changed to 'hr_director', updated_at = NOW()
```

---

## Part 4: Deployment Verification

### Pre-Production Checklist

- [ ] Backend builds without errors: `go build ./cmd/server`
- [ ] Frontend builds without errors: `npm run build`
- [ ] Database migration executed: `3 rows inserted` verified
- [ ] TimeoutMonitor starts on server startup
- [ ] All 5 API endpoints respond with correct status codes
- [ ] Frontend can create timeout trigger
- [ ] Frontend can list triggers
- [ ] Frontend can update trigger
- [ ] Frontend can delete trigger
- [ ] Frontend can test trigger
- [ ] Audit events recorded for timeout executions

### Production Deployment

```bash
# 1. Backup database
pg_dump alpha > backup_$(date +%Y%m%d).sql

# 2. Run migration (already done, but verify)
psql -f backend/db/migrations/2025_10_20_workflow_timeout_triggers.sql

# 3. Build and deploy backend
cd backend && go build -o semlayer-server ./cmd/server
systemctl restart semlayer-backend

# 4. Build and deploy frontend
cd frontend && npm run build
cp -r dist/* /var/www/semlayer/

# 5. Monitor logs
journalctl -u semlayer-backend -f

# 6. Verify API is up
curl http://localhost:8080/api/workflow-timeout-triggers
```

---

## Troubleshooting

### Issue: "Timeout trigger not executing"

**Debug Steps:**
1. Check if TimeoutMonitor is running: `ps aux | grep timeout`
2. Verify database has triggers: `SELECT COUNT(*) FROM workflow_timeout_triggers;`
3. Check audit log: `SELECT * FROM audit_events WHERE action LIKE '%TIMEOUT%';`
4. Manually trigger check: Call `CheckAndExecuteTimeouts()` directly

### Issue: "API returns 404"

**Debug Steps:**
1. Verify routes are registered: Check `RegisterTimeoutTriggerRoutes()` called in main.go
2. Check server startup logs: `journalctl -u semlayer-backend -n 50`
3. Test with curl: `curl http://localhost:8080/api/workflow-timeout-triggers`

### Issue: "Frontend cannot save trigger"

**Debug Steps:**
1. Open browser console (F12) to see error
2. Verify tenant ID is set: `localStorage.getItem('selected_tenant')`
3. Check network tab for API response
4. Verify headers are being sent: `X-Tenant-ID`, `X-Tenant-Datasource-ID`

---

## Summary

Congratulations! You've successfully integrated Workday Step Timeout Triggers into Semlayer:

✅ Database: Timeout triggers configured  
✅ Backend: Service running hourly escalation checks  
✅ Frontend: UI for managing triggers  
✅ API: Full CRUD + test endpoints  
✅ Testing: End-to-end workflow escalation verified

The system now automatically escalates overdue workflow steps to supervisors, improving business process efficiency and reducing manual follow-ups.

**Next Steps:**
- Deploy to production
- Configure timeout rules for your workflows
- Monitor escalation metrics
- Gather feedback for enhancements

---

*Integration Guide for Phase 6C: Workday Step Timeout Triggers*  
*Last Updated: October 20, 2024*
