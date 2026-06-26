# Phase 4 Feature 3: Async Bulk Operations - Implementation Plan

**Status**: 🚀 STARTING NOW  
**Date**: February 21, 2026  
**Estimated Time**: 2-3 hours  
**Predecessor**: Phase 4 Feature 2 ✅ (100% Complete)

---

## Executive Summary

Phase 4 Feature 3 transforms bulk operations from synchronous (must wait) to asynchronous (fire-and-forget with status tracking). This is critical for:

- **Large Imports**: 10,000+ templates can process in background without blocking user
- **User Experience**: Return immediately with job ID, user continues working
- **Status Tracking**: Poll API or webhook for completion updates
- **Error Recovery**: Automatically retry failed items
- **Notifications**: Email/Slack when batch completes

---

## Feature Overview

### Current Limitation (Feature 2)
```
User submits bulk operation
  ↓
API holds connection
  ↓
Processes 1000+ items (30-40 seconds)
  ↓ (User waits)
Returns results
```

### New Capability (Feature 3)
```
User submits bulk operation
  ↓
Returns immediately with job_id + status_url
  ↓
User continues working
  ↓
Background job processes in parallel
  ↓
Webhook notification on completion
  OR user polls status endpoint
```

---

## Core Endpoints

### 1. Async Bulk Create (New)
**Endpoint**: `POST /api/v1/templates/bulk-create/async`  
**Response**: Immediate (HTTP 202 Accepted)

```json
{
  "jobId": "job-uuid",
  "status": "queued",
  "statusUrl": "/api/v1/jobs/job-uuid",
  "estimatedTime": "45 seconds",
  "templates": {
    "total": 1000,
    "processed": 0,
    "succeeded": 0,
    "failed": 0
  }
}
```

### 2. Job Status Endpoint (New)
**Endpoint**: `GET /api/v1/jobs/{jobId}`  
**Response**: Current job status and progress

```json
{
  "jobId": "job-uuid",
  "status": "running",
  "operationType": "bulk-create",
  "progress": {
    "total": 1000,
    "processed": 427,
    "succeeded": 420,
    "failed": 7
  },
  "startedAt": "2026-02-21T02:00:00Z",
  "estimatedCompletion": "2026-02-21T02:01:00Z",
  "completedAt": null,
  "results": {
    "successIds": ["uuid1", "uuid2", ...],
    "failedItems": [
      {
        "name": "Template Name",
        "error": "Duplicate name"
      }
    ]
  }
}
```

### 3. Job List Endpoint (New)
**Endpoint**: `GET /api/v1/jobs?status=running`  
**Response**: List of jobs with filtering

```json
{
  "jobs": [
    {
      "jobId": "job-uuid",
      "operationType": "bulk-create",
      "status": "running",
      "progress": {
        "total": 1000,
        "processed": 500,
        "succeeded": 495,
        "failed": 5
      },
      "startedAt": "2026-02-21T02:00:00Z"
    }
  ],
  "totalCount": 1,
  "completedCount": 3,
  "failedCount": 0
}
```

### 4. Job Cancel Endpoint (New)
**Endpoint**: `POST /api/v1/jobs/{jobId}/cancel`  
**Response**: Confirmation of cancellation

```json
{
  "jobId": "job-uuid",
  "status": "cancelled",
  "processedBefore": 427,
  "message": "Bulk operation cancelled at user request"
}
```

---

## Database Schema

### Table: `edm.async_jobs`
```sql
CREATE TABLE edm.async_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50),
  status VARCHAR(20),  -- queued, running, completed, failed, cancelled
  payload JSONB,       -- Original request
  total_count INT,
  processed_count INT DEFAULT 0,
  succeeded_count INT DEFAULT 0,
  failed_count INT DEFAULT 0,
  result_ids UUID[],
  error_details JSONB,
  webhook_url TEXT,
  created_by UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  priority INT DEFAULT 0,
  retry_count INT DEFAULT 0,
  max_retries INT DEFAULT 3
);

CREATE INDEX idx_jobs_tenant_status
  ON edm.async_jobs(tenant_id, status);

CREATE INDEX idx_jobs_created
  ON edm.async_jobs(created_at DESC);

CREATE INDEX idx_jobs_priority
  ON edm.async_jobs(priority DESC, created_at)
  WHERE status = 'queued';
```

### Table: `edm.job_items`
```sql
CREATE TABLE edm.job_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id UUID NOT NULL REFERENCES edm.async_jobs(id) ON DELETE CASCADE,
  item_name VARCHAR(500),
  item_data JSONB,
  status VARCHAR(20),  -- pending, processing, succeeded, failed
  error_message TEXT,
  result_id UUID,
  processed_at TIMESTAMP,
  CONSTRAINT fk_job FOREIGN KEY (job_id) REFERENCES edm.async_jobs(id)
);

CREATE INDEX idx_job_items_job
  ON edm.job_items(job_id);

CREATE INDEX idx_job_items_status
  ON edm.job_items(status) WHERE status != 'succeeded';
```

---

## Implementation Architecture

### Components

**1. Job Queue Service**
- Stores jobs in database
- Enqueue/Dequeue operations
- Priority handling
- Retry logic

**2. Job Processor Worker**
- Background goroutine pool
- Processes queued jobs
- Updates progress in database
- Error handling and retries

**3. Webhook Notifier**
- Sends HTTP POST to webhook URL
- Retries on failure
- Tracks notification status

**4. Status APIs**
- Poll endpoint for job status
- Job list with filtering
- Job cancellation endpoint

### Job Flow
```
1. User submits bulk operation to async endpoint
   ↓
2. Job created in DB with status="queued"
   ↓ (Return immediately with job_id)
3. Background worker picks up job
   ↓
4. Update status to "running"
   ↓
5. Process each item:
   - Insert into job_items table
   - Try operation
   - Update job_items.status
   - Update async_jobs counters
   ↓
6. When done, update job status to "completed"
   ↓
7. If webhook provided, send notification
   ↓
8. Data ready for user to retrieve
```

---

## Code Structure

### New Files
1. **backend/internal/services/job_queue.go** (250 lines)
   - JobQueue interface
   - PostgreSQL-backed implementation
   - Enqueue/Dequeue logic

2. **backend/internal/services/job_processor.go** (300 lines)
   - JobProcessor goroutines
   - Item processing loop
   - Error handling and retries

3. **backend/internal/services/webhook_notifier.go** (150 lines)
   - Webhook sending logic
   - Retry mechanisms
   - Error handling

4. **backend/internal/handlers/job_handlers.go** (200 lines)
   - GET /api/v1/jobs/{jobId}
   - GET /api/v1/jobs (list)
   - POST /api/v1/jobs/{jobId}/cancel
   - WebSocket support for real-time updates

### Modified Files
1. **backend/internal/handlers/bulk_operations_handler.go**
   - Add async versions of existing endpoints
   - Keep sync versions for backward compatibility

2. **backend/cmd/semantic-rules-api/main.go**
   - Initialize job queue
   - Start worker goroutines
   - Register new routes

---

## Implementation Phases

### Phase 3a: Database & Models (30 min)
- Create migration 008_async_jobs.sql
- Define Job and JobItem structs
- Apply migration to database

### Phase 3b: Job Queue Service (45 min)
- Implement JobQueue interface
- Enqueue/Dequeue operations
- Priority handling

### Phase 3c: Job Processor (60 min)
- Worker goroutine pool
- Item processing loop
- Progress tracking
- Error handling

### Phase 3d: API Endpoints (30 min)
- Job status endpoint
- Job list endpoint
- Job cancel endpoint
- Webhook support

### Phase 3e: Testing (30 min)
- Unit tests for queue
- E2E tests for async operations
- Webhook testing

---

## Key Features

### 1. Automatic Retry
```go
if item.ProcessingFailed {
  if job.RetryCount < job.MaxRetries {
    job.RetryCount++
    job.Status = "queued"  // Re-queue
    return
  }
}
```

### 2. Webhook Notifications
```json
POST https://user-webhook.example.com/

{
  "event": "bulk_operation_completed",
  "jobId": "job-uuid",
  "status": "completed",
  "succeeded": 1000,
  "failed": 5,
  "timestamp": "2026-02-21T02:15:00Z"
}
```

### 3. Real-time Progress (WebSocket)
```javascript
// Frontend can connect to WebSocket for live updates
ws = new WebSocket('/api/v1/jobs/job-uuid/watch');
ws.onmessage = (event) => {
  const progress = JSON.parse(event.data);
  console.log(`${progress.processed}/${progress.total} completed`);
};
```

### 4. Priority Queue
```go
// Higher priority jobs process first
// System jobs: priority=100
// User priority: priority=10
// Low priority: priority=1
```

---

## Performance Optimization

### Batching Strategy
```go
// Process items in batches of 100
for batch := range chunks(items, 100) {
  tx := startTransaction()
  for item := range batch {
    processItem(tx, item)
  }
  tx.Commit()
}
```

### Worker Pool
```go
// Multiple workers processing in parallel
numWorkers := 4  // CPU cores
for i := 0; i < numWorkers; i++ {
  go jobProcessor.ProcessJobs()
}
```

### Memory Management
```go
// Stream large result sets instead of loading all in memory
for result := range resultStream {
  handleResult(result)
}
```

---

## Error Handling

### Transient Errors (Retry)
- Database connection timeout
- Temporary network issue
- Resource temporarily unavailable

```go
if isTransient(err) {
  job.RetryCount++
  job.Status = "queued"
}
```

### Permanent Errors (Skip)
- Validation failed
- Item already exists
- Permission denied

```go
if isPermanent(err) {
  item.Status = "failed"
  item.Error = err.Message
}
```

### Catastrophic Errors (Abort)
- Database down
- Out of disk space
- Critical configuration error

```go
if isCatastrophic(err) {
  job.Status = "failed"
  notifyAdmin(err)
}
```

---

## Configuration

### Environment Variables
```bash
# Job processing
JOB_WORKER_COUNT=4           # Number of parallel workers
JOB_BATCH_SIZE=100          # Items per transaction
JOB_MAX_RETRIES=3           # Retry failed items
JOB_TIMEOUT=300             # Seconds per job
JOB_POLL_INTERVAL=5         # Seconds between status checks

# Webhook
WEBHOOK_TIMEOUT=30          # Seconds for webhook HTTP call
WEBHOOK_RETRIES=3           # Retry on webhook failure
WEBHOOK_MAX_PAYLOAD=10485760 # 10MB max
```

---

## Testing Strategy

### Unit Tests
1. **Queue Operations**
   - Enqueue job
   - Dequeue job
   - Update job status
   - Retry logic

2. **Processor Logic**
   - Process single item
   - Batch processing
   - Error handling
   - Retry on failure

3. **Webhook Notifier**
   - Send webhook
   - Retry on timeout
   - Handle webhook failure

### E2E Tests
1. **Complete Workflow**
   - Submit async bulk-create
   - Poll status endpoint
   - Get job completion
   - Verify all items created

2. **Error Scenarios**
   - Partial failure with retry
   - Complete failure
   - Job cancellation
   - Webhook delivery

3. **Performance**
   - 1000-item job latency
   - Queue throughput
   - Memory usage under load

---

## Success Criteria

- [ ] All 4 new endpoints implemented
- [ ] Job queue operational
- [ ] Workers processing jobs in background
- [ ] Status polling working
- [ ] Progress tracking accurate
- [ ] Webhook notifications sent
- [ ] Retry logic functioning
- [ ] Job cancellation working
- [ ] Unit tests passing (>90% coverage)
- [ ] E2E tests passing
- [ ] Performance acceptable (<100ms response time)
- [ ] Documentation complete

---

## Migration Strategy

While async features are being added, all existing sync endpoints remain functional. Users can choose:

```
POST /api/v1/templates/bulk-create       # Sync (wait)
POST /api/v1/templates/bulk-create/async # Async (fire-and-forget)
```

---

## Future Enhancements

### High Priority
1. **Batch Result Export**
   - Export results to CSV/JSON file
   - Download via presigned URL

2. **Advanced Scheduling**
   - Schedule job for later
   - Recurring jobs
   - Cron expressions

### Medium Priority
1. **Real-time Notifications**
   - WebSocket for live progress
   - Server-sent events (SSE)
   - Desktop notifications

2. **Job Analytics**
   - Performance metrics
   - Success rate tracking
   - Timing analysis

### Nice to Have
1. **Distributed Processing**
   - Multiple servers processing
   - Redis-backed queue
   - Load balancing

2. **Advanced Debugging**
   - Job replay functionality
   - Detailed audit logs
   - Performance profiling

---

## Rollback Plan

If issues arise:
1. Disable async endpoints (keep sync)
2. Stop worker goroutines
3. Purge job queue
4. Revert migrations if needed
5. Sync endpoints still operational

---

## Monitoring & Alerts

### Metrics to Track
- Jobs completed/failed per hour
- Average processing time
- Queue depth
- Worker utilization
- Webhook delivery success rate
- Error rate by operation type

### Alerts
- Queue depth > 100
- Worker crashes
- Webhook delivery failures
- Job timeout exceeded
- Database connectivity issues

---

**Status**: 🚀 READY TO IMPLEMENT  
**Next Step**: Create database migration and start Phase 3a implementation
