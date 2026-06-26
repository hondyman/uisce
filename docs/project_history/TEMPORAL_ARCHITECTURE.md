# Temporal Governance Architecture Diagram

## High-Level System Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           FABRIC BUILDER PLATFORM                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                    FRONTEND (React + Vite)                          │   │
│  │                                                                      │   │
│  │  ┌─────────────────────────────────────────────────────────────┐    │   │
│  │  │       TemporalAdminDashboard                               │    │   │
│  │  │  • Workflow List with Filters                             │    │   │
│  │  │  • Saved Views (Failed, Pending, HighPriority)            │    │   │
│  │  │  • Search Attributes Panel                                │    │   │
│  │  │  • Inline Admin Actions (Signal/Cancel/Terminate/Reset)   │    │   │
│  │  │  • Workflow Details Sidebar                               │    │   │
│  │  │  • Action History Audit Trail                             │    │   │
│  │  └─────────────────────────────────────────────────────────────┘    │   │
│  │                              ↓ HTTP REST                            │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                   │                                         │
└───────────────────────────────────┼─────────────────────────────────────────┘
                                    │
                ┌───────────────────▼────────────────────┐
                │                                        │
                │  HTTP: /api/temporal/workflows/{id}/   │
                │   - /signal       (send signal)       │
                │   - /update       (update workflow)   │
                │   - /cancel       (graceful stop)     │
                │   - /terminate    (force stop)        │
                │   - /reset        (replay)            │
                │   - /history      (export)            │
                │   - /search-attributes (list)         │
                │                                        │
                └────────────────────┬───────────────────┘
                                     │
        ┌────────────────────────────▼────────────────────────────┐
        │                   BACKEND (Go)                          │
        │                                                          │
        │  ┌────────────────────────────────────────────────────┐ │
        │  │        API Handler Layer                           │ │
        │  │  (backend/internal/api/temporal_admin.go)          │ │
        │  └────────────────────────────────────────────────────┘ │
        │                      ↓                                    │
        │  ┌────────────────────────────────────────────────────┐ │
        │  │        Service Layer                               │ │
        │  │  ┌──────────────────────────────────────────────┐  │ │
        │  │  │ SearchAttributeInitializer                   │  │ │
        │  │  │  • 10 standard attributes                    │  │ │
        │  │  │  • Generate CLI setup script                 │  │ │
        │  │  └──────────────────────────────────────────────┘  │ │
        │  │  ┌──────────────────────────────────────────────┐  │ │
        │  │  │ WorkflowAdminService                         │  │ │
        │  │  │  • Signal, Update, Cancel, Terminate, Reset  │  │ │
        │  │  │  • Batch operations                          │  │ │
        │  │  │  • Audit logging                             │  │ │
        │  │  └──────────────────────────────────────────────┘  │ │
        │  │  ┌──────────────────────────────────────────────┐  │ │
        │  │  │ HistoryExportService                         │  │ │
        │  │  │  • Export full history as JSON               │  │ │
        │  │  │  • Flatten for analytics (SQL-ready)         │  │ │
        │  │  │  • Compliance audit trails                   │  │ │
        │  │  └──────────────────────────────────────────────┘  │ │
        │  └────────────────────────────────────────────────────┘ │
        │                      ↓ gRPC                              │
        └──────────────────────┬───────────────────────────────────┘
                               │
                    ┌──────────▼────────────┐
                    │  TEMPORAL SERVER      │
                    │  (Docker Container)   │
                    │                       │
                    │  • 7233: gRPC (SDK)   │
                    │  • 8233: Metrics      │
                    │                       │
                    │  Components:          │
                    │  • Workflow Exec      │
                    │  • Event History      │
                    │  • Search Index       │
                    │  • Worker Pool        │
                    │                       │
                    └──────┬────────┬───────┘
                           │        │
                    ┌──────▼──┐    │
                    │PostgreSQL   │
                    │(History)    │
                    └────────────┘
                           │
                ┌──────────┴────────────┐
                │                       │
        ┌───────▼─────────┐   ┌────────▼────────┐
        │   PROMETHEUS    │   │    GRAFANA      │
        │   (Port 9090)   │   │  (Port 3000)    │
        │                 │   │                 │
        │ • Metrics       │   │ • Dashboards    │
        │   Storage       │   │ • Real-time     │
        │ • Time Series   │   │   Monitoring    │
        │ • Alert Rules   │   │ • SLA Tracking  │
        │                 │   │ • KPI Displays  │
        └─────────────────┘   └─────────────────┘
              ↑                      ↓
              │                    User
              │             (Ops, Management)
              └──────────────────────┘
                   Metrics Scraping
                   (15 sec interval)
```

## Data Flow Diagrams

### 1. Admin Operation Flow

```
┌──────────────────┐
│  Ops User        │
│  (Dashboard)     │
└────────┬─────────┘
         │
         │ Selects workflow + action
         ▼
┌──────────────────────────────────────────┐
│  TemporalAdminDashboard (React)          │
│                                          │
│  • Filter → Find workflow                │
│  • Click Signal/Cancel/Terminate/Reset   │
│  • Modal: Enter reason + parameters      │
└────────┬─────────────────────────────────┘
         │ POST /api/temporal/workflows/{id}/{action}
         │ with { reason, input, ... }
         ▼
┌──────────────────────────────────────────┐
│  Backend API Handler                     │
│  (temporal_admin.go)                     │
│                                          │
│  • Validate request                      │
│  • Call WorkflowAdminService             │
│  • Log audit record                      │
└────────┬─────────────────────────────────┘
         │ client.SignalWorkflow() / 
         │ client.CancelWorkflow() / etc.
         ▼
┌──────────────────────────────────────────┐
│  Temporal Server (gRPC)                  │
│                                          │
│  • Signal: Queued for workflow           │
│  • Cancel: Graceful cleanup initiated    │
│  • Terminate: Force stop                 │
│  • Reset: Event history replay           │
└────────┬─────────────────────────────────┘
         │ Event recorded in history
         ▼
┌──────────────────────────────────────────┐
│  Workflow Execution                      │
│                                          │
│  • Receives signal → handler triggered   │
│  • Receives cancel → cleanup code runs   │
│  • Terminated → forced exit              │
│  • Reset → replay from decision point    │
└──────────────────────────────────────────┘
```

### 2. Search & Filter Flow

```
User Input
  ↓
Filters: status, businessUnit, priority, searchText
  ↓
Frontend: useMemo(applyFilters)
  ↓
Local filter (workflowList → filteredWorkflows)
  ↓
  ├─ If "Saved View" selected → Populate searchText
  │  (e.g., "status = 'failed' AND start_time > '-24h'")
  │
  ├─ Frontend lists filtered results
  │
  └─ User can export selected workflows
     → /api/temporal/workflows/batch-export
     → Temporal ListWorkflow API
     → JSON export
```

### 3. Metrics & Monitoring Flow

```
Temporal Server
  ↓ Metrics Endpoint
  │ (8233/metrics)
  │
  ├─ temporal_workflow_completed_total
  ├─ temporal_workflow_failed_total
  ├─ temporal_workflow_timedout_total
  ├─ temporal_workflow_running_total
  ├─ temporal_workflow_execution_latency_bucket
  ├─ temporal_activity_executed_total
  ├─ temporal_worker_task_queue_lag
  └─ ... (100+ metrics)
       ↓ Prometheus Scrape
       │ (every 15 seconds)
       ▼
   PROMETHEUS
     ↓
     ├─ Time series storage
     ├─ Alert rule evaluation
     └─ PromQL queries
          ↓
       GRAFANA
         ↓
         ├─ Panel 1: Workflow Executions (1h)
         ├─ Panel 2: Running Workflows (gauge)
         ├─ Panel 3: Execution Latency Percentiles
         ├─ Panel 4: Server Status
         ├─ Panel 5: Failed Workflows (1h)
         ├─ Panel 6: Task Queue Backlog
         └─ Panel 7: Success Rate (1h)
              ↓
           Dashboard
           (http://localhost:3000)
```

### 4. History Export Flow

```
User/Admin
  ↓
Request: /api/temporal/workflows/{id}/history
  ↓
HistoryExportService.ExportHistory()
  ↓
client.GetWorkflowHistory()
  ↓
Temporal Server: Fetch event stream
  ↓
┌────────────────────────────────────┐
│ Event Iterator                     │
│  • EventID, EventType              │
│  • Timestamp, Attributes           │
│  • Loop until HasNext() == false   │
└────────┬───────────────────────────┘
         │
         ├─ Convert to HistoryEvent objects
         ├─ Extract event attributes
         ├─ Calculate start/end times
         └─ Aggregate summary
              ↓
         Response: HistoryExportResponse
           {
             "status": "success",
             "workflow_id": "...",
             "events": [...],
             "summary": {
               "total_events": 1250,
               "start_time": "2024-10-22T10:00:00Z",
               "end_time": "2024-10-22T14:30:00Z",
               "duration": 16200,
               "status": "exported"
             }
           }
              ↓
         Frontend/Client
           ↓
           ├─ Download as JSON
           ├─ Import into BI tool
           ├─ Generate compliance report
           └─ Archive for audit
```

## Component Dependency Graph

```
Frontend Layer
  └─ TemporalAdminDashboard.tsx
      ├─ Depends: React, Lucide icons
      ├─ Calls: /api/temporal/workflows/{id}/*
      └─ Renders: Filters, Workflows, Details, Actions

Backend API Layer
  └─ temporal_admin.go
      ├─ HTTP handlers for all admin endpoints
      ├─ Depends: Chi router, JSON marshaling
      └─ Calls: WorkflowAdminService

Service Layer
  ├─ WorkflowAdminService
  │  ├─ Depends: Temporal client.Client
  │  ├─ Provides: Signal, Update, Cancel, Terminate, Reset
  │  └─ Returns: AdminActionResponse
  │
  ├─ SearchAttributeInitializer
  │  ├─ Depends: Temporal client.Client
  │  ├─ Provides: Attribute list, CLI script generation
  │  └─ Returns: SearchAttributeConfig array
  │
  └─ HistoryExportService
     ├─ Depends: Temporal client.Client
     ├─ Provides: Export, flatten, audit trail generation
     └─ Returns: HistoryExportResponse, audit records

Temporal Integration
  ├─ client.Client (SDK)
  │  ├─ SignalWorkflow()
  │  ├─ CancelWorkflow()
  │  ├─ TerminateWorkflow()
  │  ├─ GetWorkflowHistory()
  │  └─ ListWorkflow()
  │
  └─ Temporal Server
     ├─ Workflow Execution Engine
     ├─ Event History Storage
     ├─ Search Index
     ├─ Metrics Exporter
     └─ Worker Management

Observability Stack
  ├─ Prometheus
  │  ├─ Scrapes: Temporal metrics (port 8233)
  │  ├─ Stores: Time series in TSDB
  │  └─ Evaluates: Alert rules
  │
  └─ Grafana
     ├─ Datasource: Prometheus
     ├─ Dashboards: temporal-workflows.json
     └─ Panels: 7 pre-built visualizations
```

## Deployment Architecture

```
Docker Compose Services
│
├─ postgres (existing)
│  └─ Data: Temporal history, application DB
│
├─ graphql-engine (existing)
│  └─ API: Hasura GraphQL
│
├─ backend (existing → updated)
│  ├─ New routes: /api/temporal/*
│  ├─ New services: SearchAttr, WorkflowAdmin, HistoryExport
│  └─ Port: 8080
│
├─ temporal (existing)
│  ├─ Workflow execution engine
│  ├─ Metrics endpoint: 8233/metrics
│  └─ Port: 7233 (gRPC)
│
├─ prometheus (new)
│  ├─ Scrape config: temporal:8233
│  ├─ Alert rules: alert-rules.yml
│  └─ Port: 9090
│
├─ grafana (new)
│  ├─ Dashboard: temporal-workflows.json
│  ├─ Datasource: Prometheus
│  ├─ Port: 3000
│  └─ Default: admin/admin
│
├─ frontend (existing → updated)
│  ├─ New route: /temporal-admin
│  ├─ New component: TemporalAdminDashboard
│  └─ Port: 5173
│
└─ [other services...]
```

## Search Attributes Index

```
Temporal Server Search Index
│
├─ BusinessUnit (Keyword)
│  └─ Values: "Retail", "Wholesale", "Operations"
│
├─ SlaDeadline (Datetime)
│  └─ Used for: "deadline > now", "overdue"
│
├─ Priority (Int)
│  └─ Range: 1 (high) to 5 (low)
│
├─ ProcessOwner (Keyword)
│  └─ Values: User IDs, team names
│
├─ CustomerID (Keyword)
│  └─ Used for: Customer filtering
│
├─ ProcessStatus (Keyword)
│  └─ Values: "started", "approved", "rejected", "escalated"
│
├─ ComplianceRisk (Keyword)
│  └─ Values: "high-risk", "audit-required"
│
├─ EscalationLevel (Int)
│  └─ Range: 0 (normal) to 5+ (levels of escalation)
│
├─ StartTime (Datetime)
│  └─ Used for: "started last 24h", "long-running"
│
└─ TenantID (Keyword)
   └─ Used for: Multi-tenant isolation
```

---

**Architecture Version**: 1.0  
**Last Updated**: October 22, 2025  
**Status**: Production Ready
