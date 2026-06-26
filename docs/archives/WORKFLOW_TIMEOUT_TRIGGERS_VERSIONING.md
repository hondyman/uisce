# Workflow Timeout Triggers - Versioning & Enterprise Features

## Overview

This document describes the comprehensive versioning, approval workflow, and collaboration features added to the Workflow Timeout Triggers system. These features align with enterprise-grade validation rule management patterns.

## Architecture

### Core Components

#### 1. **Versioning System**
- Automatic version tracking on every update
- Complete change history with diff tracking
- Version snapshots stored in `workflow_timeout_trigger_versions` table
- Ability to restore previous versions
- Change summary and attribution to author

#### 2. **Approval Workflows**
- Multi-level approval chains via `workflow_timeout_trigger_approvals` table
- Approval request tracking with status (pending/approved/rejected)
- Rejection reason capture for audit compliance
- Timestamp tracking for each approval action

#### 3. **Collaboration**
- Team comments on triggers via `workflow_timeout_trigger_comments` table
- @mention support for user notifications
- Comment threading via parent comment references
- Comment-level audit trail

#### 4. **Testing & Validation**
- Test case management via `workflow_timeout_trigger_tests` table
- Test suite creation and execution tracking
- Pass rate calculation and performance metrics
- Test execution history

#### 5. **Analytics & Performance**
- Invocation tracking via `workflow_timeout_trigger_analytics` table
- Success rate monitoring
- Execution time metrics (avg, min, max)
- 30-day trend tracking

#### 6. **Audit Trail**
- Complete action logging via `workflow_timeout_trigger_audit` table
- Actor information capture (who made changes)
- Action type tracking (create, update, delete, restore, approve, reject)
- Detailed change information

## Database Schema

### Main Table Enhancement

```sql
ALTER TABLE workflow_timeout_triggers ADD:
  - version INT DEFAULT 1
  - status VARCHAR(20) CHECK (status IN ('draft', 'active', 'deprecated'))
  - created_by UUID
  - modified_by UUID
  - description TEXT
  - tags JSONB
  - metadata JSONB
```

### New Tables

#### Version History
```sql
CREATE TABLE workflow_timeout_trigger_versions (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  version INT
  workflow_name VARCHAR(100)
  step_name VARCHAR(100)
  due_hours INT
  trigger_percentages JSONB
  actions_json JSONB
  is_active BOOLEAN
  changes JSONB              -- Array of change descriptions
  change_summary TEXT        -- Human-readable summary
  author_id UUID
  author_email VARCHAR(255)
  author_name VARCHAR(100)
  created_at TIMESTAMP WITH TIME ZONE
  UNIQUE(trigger_id, version)
)
```

#### Approval Requests
```sql
CREATE TABLE workflow_timeout_trigger_approvals (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  version INT
  status VARCHAR(20)         -- pending, approved, rejected
  requested_by_id UUID
  requested_by_email VARCHAR(255)
  requested_by_name VARCHAR(100)
  requested_at TIMESTAMP WITH TIME ZONE
  approvers JSONB            -- Array: {id, email, name, status, timestamp}
  rejection_reason TEXT
  approved_at TIMESTAMP WITH TIME ZONE
  rejected_at TIMESTAMP WITH TIME ZONE
)
```

#### Comments
```sql
CREATE TABLE workflow_timeout_trigger_comments (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  content TEXT
  author_id UUID
  author_email VARCHAR(255)
  author_name VARCHAR(100)
  created_at TIMESTAMP WITH TIME ZONE
  updated_at TIMESTAMP WITH TIME ZONE
  parent_comment_id UUID REFERENCES comments
  mentioned_users JSONB      -- Array of user IDs
)
```

#### Tests
```sql
CREATE TABLE workflow_timeout_trigger_tests (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  test_case_name VARCHAR(255)
  input_data JSONB
  expected_result VARCHAR(10)   -- pass or fail
  actual_result VARCHAR(10)
  status VARCHAR(20)            -- pending, running, passed, failed
  error_message TEXT
  execution_time_ms INT
  run_at TIMESTAMP WITH TIME ZONE
  runner_id UUID
  runner_email VARCHAR(255)
)
```

#### Test Suites
```sql
CREATE TABLE workflow_timeout_trigger_test_suites (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  name VARCHAR(255)
  description TEXT
  total_tests INT
  passed_tests INT
  failed_tests INT
  pass_rate DECIMAL(5, 2)
  last_run_at TIMESTAMP WITH TIME ZONE
  last_run_duration_ms INT
  created_by_id UUID
  created_by_email VARCHAR(255)
)
```

#### Analytics
```sql
CREATE TABLE workflow_timeout_trigger_analytics (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  total_invocations BIGINT
  successful_invocations BIGINT
  failed_invocations BIGINT
  success_rate DECIMAL(5, 2)
  avg_execution_time_ms DECIMAL(10, 2)
  min_execution_time_ms INT
  max_execution_time_ms INT
  last_30_days_invocations BIGINT
  last_30_days_success_rate DECIMAL(5, 2)
  measured_at TIMESTAMP WITH TIME ZONE
)
```

#### Audit Log
```sql
CREATE TABLE workflow_timeout_trigger_audit (
  id UUID PRIMARY KEY
  trigger_id UUID FOREIGN KEY
  tenant_id UUID
  action VARCHAR(50)         -- create, update, delete, restore, approve, reject
  details JSONB
  actor_id UUID
  actor_email VARCHAR(255)
  actor_name VARCHAR(100)
  actor_role VARCHAR(50)
  created_at TIMESTAMP WITH TIME ZONE
)
```

## API Endpoints

### Versioning Endpoints

#### List Version History
```
GET /api/workflow-timeout-triggers/{triggerId}/versions
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response:
{
  "versions": [
    {
      "version": 3,
      "trigger": { ... },
      "changes": ["Updated due_hours from 24 to 48"],
      "change_summary": "Update: HireEmployee",
      "timestamp": "2025-10-21T10:30:00Z",
      "author": "john.doe@company.com",
      "author_name": "John Doe"
    }
  ]
}
```

#### Get Specific Version
```
GET /api/workflow-timeout-triggers/{triggerId}/versions/{version}
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response: TriggerVersion object
```

#### Restore Previous Version
```
POST /api/workflow-timeout-triggers/{triggerId}/versions/{version}/restore
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response:
{
  "message": "Restored version 2",
  "timestamp": "2025-10-21T10:35:00Z"
}
```

### Approval Endpoints

#### Request Approval
```
POST /api/workflow-timeout-triggers/{triggerId}/approvals/request
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Body:
{
  "version": 3,
  "reviewers": ["manager@company.com", "director@company.com"]
}

Response:
{
  "approval_id": "uuid",
  "status": "pending"
}
```

#### Get Approvals
```
GET /api/workflow-timeout-triggers/{triggerId}/approvals
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response: Array of ApprovalRequest objects
```

#### Approve Change
```
POST /api/workflow-timeout-triggers/approvals/{approvalId}/approve
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response:
{
  "message": "Change approved",
  "approval_id": "uuid"
}
```

#### Reject Change
```
POST /api/workflow-timeout-triggers/approvals/{approvalId}/reject
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Body:
{
  "reason": "Needs further discussion about escalation target"
}

Response:
{
  "message": "Change rejected",
  "approval_id": "uuid"
}
```

### Collaboration Endpoints

#### Get Comments
```
GET /api/workflow-timeout-triggers/{triggerId}/comments
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response: Array of Comment objects
```

#### Add Comment
```
POST /api/workflow-timeout-triggers/{triggerId}/comments
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Body:
{
  "content": "Should we also notify the finance team?",
  "mentioned_users": ["finance@company.com"]
}

Response:
{
  "comment_id": "uuid",
  "timestamp": "2025-10-21T10:40:00Z"
}
```

#### Delete Comment
```
DELETE /api/workflow-timeout-triggers/{triggerId}/comments/{commentId}
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response:
{
  "message": "Comment deleted"
}
```

### Testing Endpoints

#### List Tests
```
GET /api/workflow-timeout-triggers/{triggerId}/tests
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response: Array of TestCase objects
```

#### Run Tests
```
POST /api/workflow-timeout-triggers/{triggerId}/test
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Body:
{
  "test_cases": [
    {
      "name": "Test timeout at 80%",
      "input": { "elapsed_hours": 38.4 },
      "expected_result": "pass"
    }
  ]
}

Response:
{
  "results": [
    {
      "test_case": "Test timeout at 80%",
      "status": "passed",
      "execution_time_ms": 45
    }
  ],
  "pass_rate": 100.0
}
```

### Analytics Endpoints

#### Get Analytics
```
GET /api/workflow-timeout-triggers/{triggerId}/analytics
Headers:
  X-Tenant-ID: <tenant-id>
  X-Tenant-Datasource-ID: <datasource-id>

Response:
{
  "trigger_id": "uuid",
  "total_invocations": 15600,
  "successful_invocations": 14820,
  "failed_invocations": 780,
  "success_rate": 95.0,
  "avg_execution_time_ms": 24.5,
  "min_execution_time_ms": 12,
  "max_execution_time_ms": 89,
  "last_30_days_invocations": 5200,
  "last_30_days_success_rate": 96.2,
  "measured_at": "2025-10-21T10:45:00Z"
}
```

## Frontend Components

### Enhanced React Component

The `WorkflowTimeoutTriggersPageEnhanced.tsx` component includes:

#### Tab 1: Overview
- Display trigger metadata
- Show current version and status
- Display creator/modifier information

#### Tab 2: Version History
- Timeline view of all versions
- Author attribution
- Change descriptions
- "Restore to this version" button for previous versions

#### Tab 3: Approvals
- List all approval requests
- Show approval status for each reviewer
- Approve/Reject buttons with reason capture
- Approval timeline

#### Tab 4: Comments
- Comment thread display
- Add new comment form
- User avatars and timestamps
- @mention support

#### Tab 5: Analytics
- Total invocations counter
- Success rate percentage
- Average execution time
- 30-day trend statistics
- Performance bottleneck identification

## Workflow Examples

### Example 1: Create and Get Approval

```typescript
// 1. Create a new trigger
POST /api/workflow-timeout-triggers
{
  "workflow_name": "HireEmployee",
  "step_name": "ManagerApproval",
  "due_hours": 48,
  "actions": [...]
}
// Returns: trigger with version 1, status "active"

// 2. Update the trigger
PUT /api/workflow-timeout-triggers/{triggerId}
{
  "due_hours": 72,  // Extended from 48
  "actions": [...]
}
// Creates version 2 record

// 3. Request approval
POST /api/workflow-timeout-triggers/{triggerId}/approvals/request
{
  "version": 2,
  "reviewers": ["hr_director@company.com"]
}
// Creates approval request with status "pending"

// 4. Approve change
POST /api/workflow-timeout-triggers/approvals/{approvalId}/approve
// Approves the change

// 5. Get version history
GET /api/workflow-timeout-triggers/{triggerId}/versions
// Shows versions 1 and 2 with full history
```

### Example 2: Restore Previous Version

```typescript
// If version 3 has issues, restore to version 2
POST /api/workflow-timeout-triggers/{triggerId}/versions/2/restore
// Creates new version 4 as copy of version 2
// Logs audit trail
// Notifies team via comments
```

### Example 3: Collaboration Workflow

```typescript
// 1. Add comment to trigger
POST /api/workflow-timeout-triggers/{triggerId}/comments
{
  "content": "@john.doe Should we increase this timeout?",
  "mentioned_users": ["john.doe@company.com"]
}

// 2. Get all comments
GET /api/workflow-timeout-triggers/{triggerId}/comments
// Returns list of comments with author info

// 3. Notifications fired for mentioned users
// john.doe@company.com receives notification
```

## Implementation Checklist

### Database
- [x] Create migration file `2025_10_21_add_versioning_to_timeout_triggers.sql`
- [x] Add version columns to main table
- [x] Create version history table
- [x] Create approval table
- [x] Create comments table
- [x] Create tests table
- [x] Create test suites table
- [x] Create analytics table
- [x] Create audit table
- [x] Create indexes for performance

### Backend
- [x] Create versioned handler with all types
- [x] Implement versioning endpoints
- [x] Implement approval endpoints
- [x] Implement collaboration endpoints
- [x] Implement testing endpoints
- [x] Implement analytics endpoints
- [x] Add audit logging
- [x] Add version record creation on updates

### Frontend
- [x] Create enhanced React component
- [x] Add Overview tab
- [x] Add Version History tab with timeline
- [x] Add Approvals tab with workflow
- [x] Add Comments tab with threading
- [x] Add Analytics tab with metrics
- [x] Add CSS module for styling
- [x] Implement API calls for all features

## Best Practices

### Versioning
1. Always create a version record when trigger is updated
2. Include detailed change summaries for audit trail
3. Allow restoration to any previous version
4. Maintain immutable version snapshots

### Approvals
1. Support multi-level approval chains
2. Require approval for production changes
3. Capture rejection reasons for compliance
4. Provide audit trail of all approvals

### Collaboration
1. Support @mentions for team communication
2. Allow threaded comments for context
3. Maintain comment history for compliance
4. Notify mentioned users

### Testing
1. Create test suites before going live
2. Run tests on every change
3. Track pass rates over time
4. Identify flaky tests with execution history

### Analytics
1. Track all invocations for trending
2. Monitor success rates for reliability
3. Monitor execution time for performance
4. Alert on degradation

### Audit Trail
1. Log every action with actor information
2. Include detailed change information
3. Maintain immutable audit log
4. Support compliance reporting

## Migration Guide

To implement in existing system:

### Step 1: Apply Migration
```bash
cd backend
psql -U postgres -d alpha -f db/migrations/2025_10_21_add_versioning_to_timeout_triggers.sql
```

### Step 2: Update Backend Handler
```bash
# Replace old handler with versioned handler
cp internal/handlers/timeout_triggers_versioned_handler.go \
   internal/handlers/timeout_triggers_handler.go
```

### Step 3: Update Frontend Component
```bash
# Replace old component with enhanced component
cp frontend/src/pages/WorkflowTimeoutTriggersPageEnhanced.tsx \
   frontend/src/pages/WorkflowTimeoutTriggersPage.tsx
```

### Step 4: Update CSS
```bash
# Already includes new styles in module file
```

### Step 5: Rebuild and Deploy
```bash
# Backend
go build -o semlayer-api ./cmd/api

# Frontend
npm run build
npm run deploy
```

## Performance Considerations

### Indexes
- `idx_versions_trigger_id` - Fast version history lookup
- `idx_versions_tenant_id` - Fast tenant scoping
- `idx_approvals_status` - Fast approval workflow queries
- `idx_comments_trigger_id` - Fast comment retrieval
- `idx_audit_trigger_id` - Fast audit trail lookup

### Query Optimization
- Use cursor-based pagination for version history
- Cache analytics for frequently accessed triggers
- Archive old versions after 12 months
- Batch approve operations

### Caching Strategy
- Cache current trigger version (5 min TTL)
- Cache analytics data (15 min TTL)
- Cache version list (10 min TTL)
- Invalidate on any update

## Security Considerations

### Authorization
- Only admins can restore versions
- Only approval chain members can approve
- Users can only see own organization's triggers
- Audit trail accessible to compliance team only

### Data Protection
- Encrypt sensitive data in approvers array
- PII redaction for audit logs
- Soft-delete for comments (never truly deleted)
- Immutable version snapshots

## Conclusion

This comprehensive versioning system provides enterprise-grade governance, auditability, and collaboration features for workflow timeout trigger management. All changes are tracked, all approvals are recorded, and all team discussions are captured for compliance and operational transparency.
