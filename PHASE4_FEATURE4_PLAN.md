# Phase 4 Feature 4: Advanced Async Features - Implementation Plan

**Status**: 🚀 STARTING NOW  
**Date**: February 21, 2026  
**Estimated Time**: 3-4 hours  
**Predecessor**: Phase 4 Feature 3 ✅ (100% Complete - Async Bulk Operations)

---

## Executive Summary

Phase 4 Feature 4 extends async capabilities with three major feature sets:

1. **Result Export & Download** - Export job results to CSV/JSON with presigned URLs
2. **Advanced Scheduling** - Schedule jobs for later execution or recurring intervals  
3. **Real-time Notifications** - WebSocket and Server-Sent Events for live progress updates

These features enable enterprise workflows like scheduled bulk operations, real-time dashboards, and data export pipelines.

---

## Feature 1: Result Export & Download

### Use Cases
- Export 10,000 template IDs to CSV for bulk import to external system
- Download failed items report in JSON for manual review
- Generate audit trail CSV for compliance
- Batch export to data warehouse

### New Endpoints

#### 1a. Export Job Results
**Endpoint**: `POST /api/v1/jobs/{jobId}/export`  
**Response**: Immediate (HTTP 202 Accepted)

```json
{
  "jobId": "job-uuid",
  "exportId": "export-uuid",
  "status": "processing",
  "format": "csv",
  "downloadUrl": "/api/v1/exports/export-uuid/download",
  "expiresAt": "2026-02-21T03:30:00Z",
  "estimatedSize": "2.5 MB"
}
```

#### 1b. Get Export Status
**Endpoint**: `GET /api/v1/exports/{exportId}`  
**Response**: Current export status and download URL

```json
{
  "exportId": "export-uuid",
  "jobId": "job-uuid",
  "status": "completed",
  "format": "csv",
  "fileSize": 2621440,
  "recordCount": 5000,
  "downloadUrl": "/api/v1/exports/export-uuid/download",
  "expiresAt": "2026-02-21T03:30:00Z",
  "createdAt": "2026-02-21T01:30:00Z",
  "completedAt": "2026-02-21T01:32:15Z"
}
```

#### 1c. Download Export File
**Endpoint**: `GET /api/v1/exports/{exportId}/download`  
**Response**: File download (CSV/JSON)

```bash
# Response headers
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="job-results-export.csv"
Content-Length: 2621440
```

#### 1d. List Exports
**Endpoint**: `GET /api/v1/exports?jobId=job-uuid&limit=20`  
**Response**: List of exports

```json
{
  "exports": [
    {
      "exportId": "export-uuid-1",
      "jobId": "job-uuid",
      "format": "csv",
      "status": "completed",
      "fileSize": 2621440,
      "createdAt": "2026-02-21T01:30:00Z"
    }
  ],
  "totalCount": 1
}
```

### Database Schema for Exports

#### Table: `edm.job_exports`
```sql
CREATE TABLE edm.job_exports (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  job_id UUID NOT NULL REFERENCES edm.async_jobs(id) ON DELETE CASCADE,
  tenant_id UUID NOT NULL,
  export_format VARCHAR(20),  -- csv, json, parquet
  status VARCHAR(20),  -- queued, processing, completed, failed
  file_location TEXT,  -- S3 path or local path
  file_size BIGINT DEFAULT 0,
  record_count INT DEFAULT 0,
  presigned_url TEXT,
  presigned_url_expires TIMESTAMP,
  filter_criteria JSONB,  -- what records to include
  created_by UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  expires_at TIMESTAMP  -- When file will be deleted
);

CREATE INDEX idx_exports_job ON edm.job_exports(job_id);
CREATE INDEX idx_exports_tenant ON edm.job_exports(tenant_id);
CREATE INDEX idx_exports_status ON edm.job_exports(status) WHERE status != 'completed';
```

### Implementation

#### Models (models/export.go - 200 lines)
```go
type ExportFormat string
const (
  ExportFormatCSV     ExportFormat = "csv"
  ExportFormatJSON    ExportFormat = "json"
  ExportFormatParquet ExportFormat = "parquet"
)

type JobExport struct {
  ID                  string
  JobID               string
  TenantID            string
  Format              ExportFormat
  Status              string
  FileLocation        string
  FileSize            int64
  RecordCount         int
  PresignedURL        string
  PresignedURLExpires *time.Time
  FilterCriteria      json.RawMessage
  CreatedBy           string
  CreatedAt           time.Time
  StartedAt           *time.Time
  CompletedAt         *time.Time
  ExpiresAt           *time.Time
}

type ExportCreateRequest struct {
  Format         ExportFormat   `json:"format"`
  FilterCriteria json.RawMessage `json:"filterCriteria,omitempty"`
  IncludeErrors  bool            `json:"includeErrors"`
}
```

#### Export Service (services/export_service.go - 350 lines)
```go
type ExportService interface {
  // CreateExport queues a new export
  CreateExport(ctx context.Context, jobID, format string, opts ...ExportOption) (*JobExport, error)
  
  // GetExportStatus returns current export status
  GetExportStatus(ctx context.Context, exportID string) (*JobExport, error)
  
  // GeneratePresignedURL creates a temporary download URL
  GeneratePresignedURL(ctx context.Context, exportID string, duration time.Duration) (string, error)
  
  // ProcessExport handles actual export generation
  ProcessExport(ctx context.Context, export *JobExport) error
  
  // ListExports returns exports for a job
  ListExports(ctx context.Context, jobID string, limit int) ([]*JobExport, error)
  
  // CleanupExpiredExports deletes old exports
  CleanupExpiredExports(ctx context.Context) error
}
```

#### Export Formats (services/exporters.go - 300 lines)
```go
// CSVExporter generates CSV from job results
type CSVExporter struct {
  db *sql.DB
}

func (e *CSVExporter) Export(ctx context.Context, jobID string, out io.Writer) (*ExportStats, error) {
  // Query job_items for job
  // Convert to CSV format
  // Stream to writer
  // Return record count and file size
}

// JSONExporter generates JSON from job results
type JSONExporter struct {
  db *sql.DB
}

func (e *JSONExporter) Export(ctx context.Context, jobID string, out io.Writer) (*ExportStats, error) {
  // Query job_items for job
  // Convert to JSONL format (one JSON per line)
  // Stream to writer
}
```

---

## Feature 2: Advanced Scheduling

### Use Cases
- Schedule bulk template creation for 2 AM (off-peak)
- Run daily sync of external templates every night at 1 AM
- Schedule template cleanup for Sundays
- Run quarterly bulk promotions

### New Endpoints

#### 2a. Create Scheduled Job
**Endpoint**: `POST /api/v1/jobs/schedule`  
**Response**: HTTP 201 Created

```json
{
  "scheduleId": "schedule-uuid",
  "jobTemplate": {
    "operationType": "bulk-create",
    "items": [...],
    "priority": 5
  },
  "schedule": {
    "type": "once",  // once, recurring, cron
    "startTime": "2026-02-21T02:00:00Z",
    "timezone": "America/New_York"
  },
  "status": "active",
  "nextRun": "2026-02-21T02:00:00Z"
}
```

#### 2b. List Scheduled Jobs
**Endpoint**: `GET /api/v1/jobs/schedule?status=active`  
**Response**: List of scheduled jobs

#### 2c. Update Schedule
**Endpoint**: `PUT /api/v1/jobs/schedule/{scheduleId}`  
**Response**: Updated schedule

#### 2d. Pause/Resume Schedule
**Endpoint**: `POST /api/v1/jobs/schedule/{scheduleId}/pause`  
**Response**: Updated status

### Database Schema for Scheduling

#### Table: `edm.scheduled_jobs`
```sql
CREATE TABLE edm.scheduled_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL,
  operation_type VARCHAR(50) NOT NULL,
  job_template JSONB NOT NULL,  -- Payload template
  
  -- Schedule configuration
  schedule_type VARCHAR(20),  -- once, recurring, cron
  start_time TIMESTAMP NOT NULL,
  end_time TIMESTAMP,  -- When to stop recurring
  cron_expression VARCHAR(100),  -- "0 2 * * *" for 2 AM daily
  timezone VARCHAR(50) DEFAULT 'UTC',
  
  -- Status
  status VARCHAR(20) DEFAULT 'active',  -- active, paused, completed, failed
  was_successful BOOLEAN,
  last_run_at TIMESTAMP,
  last_run_status VARCHAR(20),
  next_run_at TIMESTAMP,
  run_count INT DEFAULT 0,
  
  -- Metadata
  created_by UUID NOT NULL,
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_scheduled_next_run ON edm.scheduled_jobs(next_run_at)
  WHERE status = 'active';
CREATE INDEX idx_scheduled_tenant ON edm.scheduled_jobs(tenant_id);
```

#### Table: `edm.scheduled_job_runs`
```sql
CREATE TABLE edm.scheduled_job_runs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  schedule_id UUID NOT NULL REFERENCES edm.scheduled_jobs(id),
  job_id UUID REFERENCES edm.async_jobs(id),
  scheduled_time TIMESTAMP NOT NULL,
  actual_start_time TIMESTAMP,
  actual_end_time TIMESTAMP,
  status VARCHAR(20),  -- pending, running, completed, failed
  error_message TEXT,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_runs_schedule ON edm.scheduled_job_runs(schedule_id);
CREATE INDEX idx_runs_scheduled_time ON edm.scheduled_job_runs(scheduled_time);
```

### Implementation

#### Scheduler Service (services/scheduler.go - 350 lines)
```go
type SchedulerService struct {
  db    *sql.DB
  queue JobQueue
  cron  *cron.Cron
}

func (s *SchedulerService) CreateSchedule(ctx context.Context, schedule *ScheduledJob) error {
  // Parse schedule (cron, once, recurring)
  // Calculate next run time
  // Store in database
  // Register with cron if active
}

func (s *SchedulerService) ExecuteScheduledJobs(ctx context.Context) error {
  // Query for jobs due to run
  // Enqueue them to job queue
  // Update next_run_at
  // Track execution
}

func (s *SchedulerService) Start(ctx context.Context) error {
  // Start background scheduler loop
  // Check for due jobs every minute
}
```

#### Cron Parser (services/cron_parser.go - 150 lines)
```go
// Support cron expressions like:
// "0 2 * * *"     - 2 AM every day
// "0 */4 * * *"   - Every 4 hours
// "0 0 * * MON"   - Monday at midnight
// "0 0 1 * *"     - First day of month

func ParseCronExpression(expr string) (nextRun time.Time, err error) {
  // Parse using cron library
  // Calculate next run time
  // Handle timezone conversions
}
```

---

## Feature 3: Real-time Notifications

### Use Cases
- Live progress bar showing items processed in real-time
- Real-time error notifications as items fail
- Desktop notifications when job completes
- Dashboard showing live job queue status

### Technology Options

#### Option A: WebSocket (Full Duplex)
```
Client connects: ws://localhost:8080/ws/jobs/{jobId}
  ↓
Server sends progress updates every second
  ↓
Client updates UI in real-time
  ↓
Client can send commands back (pause, cancel)
```

**Pros**: 
- Full duplex communication
- Lowest latency
- Can send commands bidirectionally

**Cons**: 
- More complex to implement
- Connection state management
- Reconnection handling needed

#### Option B: Server-Sent Events (SSE)
```
Client opens: /sse/jobs/{jobId}
  ↓
Server streams updates as text/event-stream
  ↓
Client receives in browser with EventSource API
  ↓
Browser auto-reconnects if connection drops
```

**Pros**:
- Built-in browser API (EventSource)
- Automatic reconnection
- Single direction (simpler)
- Works over HTTP

**Cons**:
- One-way communication only
- Limited to ~6 concurrent connections per domain

#### Option C: Polling with Exponential Backoff
```
Current approach: GET /api/v1/jobs/{jobId} every 5 seconds
  ↓
Improved: Shorter initial interval, back off as job progresses
```

**Recommended**: Implement **Option B (SSE)** first - easiest for browser integration

### New Endpoints

#### 3a. Server-Sent Events Stream
**Endpoint**: `GET /api/v1/jobs/{jobId}/stream` (Server-Sent Events)  
**Content-Type**: `text/event-stream`

```
data: {"event":"progress","data":{"processed":100,"succeeded":95,"failed":5,"percentage":50}}

data: {"event":"item-completed","data":{"itemIndex":99,"resultId":"uuid","status":"succeeded"}}

data: {"event":"item-failed","data":{"itemIndex":45,"error":"Validation failed"}}

data: {"event":"job-completed","data":{"status":"completed","succeeded":950,"failed":50}}
```

#### 3b. WebSocket Endpoint (for Advanced Clients)
**Endpoint**: `ws://localhost:8080/ws/jobs/{jobId}`

```javascript
// Client-side
const ws = new WebSocket('ws://localhost:8080/ws/jobs/job-id');
ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`Progress: ${data.percentage}%`);
};
```

### Implementation

#### SSE Handler (handlers/sse_handler.go - 200 lines)
```go
func (h *Handler) StreamJobProgress(w http.ResponseWriter, r *http.Request) {
  jobID := mux.Vars(r)["jobId"]
  
  // Set SSE headers
  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")
  
  // Create channel for updates
  updates := make(chan interface{}, 100)
  
  // Subscribe to job updates
  h.subscriptionManager.Subscribe(jobID, updates)
  defer h.subscriptionManager.Unsubscribe(jobID, updates)
  
  // Stream updates
  for update := range updates {
    data, _ := json.Marshal(update)
    fmt.Fprintf(w, "data: %s\n\n", string(data))
    w.(http.Flusher).Flush()
  }
}
```

#### Subscription Manager (services/subscription_manager.go - 150 lines)
```go
type SubscriptionManager struct {
  subscriptions map[string][]chan interface{}
  mu            sync.RWMutex
}

func (sm *SubscriptionManager) Subscribe(jobID string, ch chan interface{}) {
  sm.mu.Lock()
  defer sm.mu.Unlock()
  sm.subscriptions[jobID] = append(sm.subscriptions[jobID], ch)
}

func (sm *SubscriptionManager) Broadcast(jobID string, message interface{}) {
  sm.mu.RLock()
  defer sm.mu.RUnlock()
  for _, ch := range sm.subscriptions[jobID] {
    select {
    case ch <- message:
    default:
      // Channel full, skip
    }
  }
}
```

#### Frontend Integration (React Component - 200 lines)
```typescript
// useJobProgress.ts
export function useJobProgress(jobId: string) {
  const [progress, setProgress] = useState({
    percentage: 0,
    processed: 0,
    succeeded: 0,
    failed: 0,
  });

  useEffect(() => {
    const eventSource = new EventSource(
      `/api/v1/jobs/${jobId}/stream`
    );

    eventSource.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.event === 'progress') {
        setProgress(data.data);
      }
    };

    eventSource.onerror = () => {
      eventSource.close();
    };

    return () => eventSource.close();
  }, [jobId]);

  return progress;
}

// In component:
function JobProgressMonitor({ jobId }) {
  const progress = useJobProgress(jobId);
  
  return (
    <div>
      <progress value={progress.percentage} max={100} />
      <p>{progress.percentage}% - {progress.processed}/{progress.succeeded}</p>
    </div>
  );
}
```

---

## Implementation Roadmap

### Phase 4a: Result Export (1.5 hours)
1. Create job_exports table and indexes (15 min)
2. Build ExportService interface (30 min)
3. Implement CSV and JSON exporters (30 min)
4. Create export API handlers (30 min)
5. Test export functionality (15 min)

### Phase 4b: Advanced Scheduling (1.5 hours)
1. Create scheduled_jobs and scheduled_job_runs tables (15 min)
2. Build SchedulerService with cron parser (45 min)
3. Create scheduling API handlers (30 min)
4. Implement scheduler background loop (15 min)
5. Test scheduling functionality (15 min)

### Phase 4c: Real-time Notifications (1 hour)
1. Build SubscriptionManager (30 min)
2. Implement SSE handler (20 min)
3. Integrate with job processor (10 min)

---

## Database Migrations

### Migration: 009_exports_and_scheduling.sql (250 lines)
```sql
-- Export tracking
CREATE TABLE edm.job_exports (...)
CREATE INDEX idx_exports_job ON edm.job_exports(job_id);
...

-- Scheduling
CREATE TABLE edm.scheduled_jobs (...)
CREATE TABLE edm.scheduled_job_runs (...)
...

-- Grant permissions
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA edm TO postgres;
```

---

## Configuration

### Environment Variables
```bash
# Export storage
EXPORT_STORAGE_TYPE=local              # local, s3, gcs
EXPORT_STORAGE_PATH=/tmp/exports
EXPORT_RETENTION_DAYS=7                # Delete exports after 7 days
EXPORT_MAX_FILE_SIZE=1073741824        # 1GB max

# Scheduling
SCHEDULER_CHECK_INTERVAL=60            # Check for due jobs every 60 seconds
SCHEDULER_TIMEZONE=America/New_York    # Default timezone
SCHEDULER_MAX_PARALLEL=4               # Max parallel scheduled job executions

# WebSocket/SSE
SERVER_SENT_EVENTS_ENABLED=true
WEBSOCKET_ENABLED=false                # Enable if needed

# Presigned URLs
PRESIGNED_URL_DURATION=3600            # 1 hour
```

---

## Success Criteria

- [ ] All export endpoints implemented
- [ ] CSV and JSON export formats working
- [ ] Presigned URLs generating correctly
- [ ] Export files downloadable
- [ ] All scheduling endpoints implemented
- [ ] Cron expressions parsed correctly
- [ ] Scheduled jobs executing on time
- [ ] SSE stream generating updates
- [ ] Real-time progress visible in browser
- [ ] Unit tests passing (>90% coverage)
- [ ] E2E tests passing
- [ ] Load tested (100+ concurrent streams)
- [ ] Documentation complete

---

## Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Export 10,000 items | <5 seconds | CSV streaming |
| Schedule creation | <100ms | Database write |
| Scheduled job execution | <1 second | Queue enqueue |
| SSE update latency | <100ms | Per-second updates |
| Concurrent SSE streams | 100+ | Per server |

---

## Testing Strategy

### Unit Tests
- Export service tests
- Scheduler tests
- Cron parser tests
- Subscription manager tests

### Integration Tests
- End-to-end export workflow
- Scheduled job execution
- Real-time updates delivery
- Multi-tenant isolation

### Performance Tests
- Export large datasets (10,000+ items)
- 100 concurrent SSE streams
- Scheduled job throughput
- Memory usage under load

---

## Next Steps After Feature 4

1. **Frontend UI Components**
   - Export dialog with format selection
   - Scheduled job management interface
   - Real-time job progress dashboard

2. **Advanced Configurations**
   - Export filtering and sorting
   - Conditional scheduling
   - Notification preferences

3. **Integration**
   - Slack notifications on completion
   - Email reports
   - Webhook callbacks

---

## Deliverables

**By End of Feature 4**:
- ✅ 7 new API endpoints
- ✅ 2 new database tables (+ indexes)
- ✅ 3 service implementations
- ✅ Real-time event streaming
- ✅ Comprehensive documentation
- ✅ Unit & integration tests
- ✅ Production-ready code

---

**Status**: Ready to implement  
**Next Action**: Begin Phase 4a (Result Export)
