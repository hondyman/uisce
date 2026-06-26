# Risk & Compliance Console - Go Backend Implementation Guide

## Overview

The React console expects these Go endpoints. All responses are JSON.

---

## Dashboard Endpoints

### 1. GET `/api/dashboard/compliance`

**Purpose**: Compliance summary KPIs

**Query Parameters**:
```
tenant_id: string (required)
valuation_date: string (required, format: YYYY-MM-DD)
```

**Response**:
```json
{
  "total_rules": 125,
  "pass_rate": 0.92,
  "hard_breaches": 3,
  "soft_breaches": 12,
  "by_severity": {
    "HARD": 3,
    "SOFT": 12,
    "INFO": 5
  }
}
```

**Implementation Pattern** (chi router):
```go
func (h *DashboardHandler) ComplianceSummary(w http.ResponseWriter, r *http.Request) {
  tenantID := r.URL.Query().Get("tenant_id")
  valuationDate := r.URL.Query().Get("valuation_date")
  
  if tenantID == "" || valuationDate == "" {
    http.Error(w, "Missing parameters", http.StatusBadRequest)
    return
  }
  
  summary, err := h.DB.GetDashboardComplianceSummary(r.Context(), 
    db.GetDashboardComplianceSummaryParams{
      TenantID:      tenantID,
      ValuationDate: valuationDate,
    })
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(summary)
}
```

**Database Query** (sqlc):
```sql
-- queries/dashboard.sql
-- name: GetDashboardComplianceSummary :one
SELECT
  COUNT(DISTINCT r.rule_id) as total_rules,
  COALESCE(SUM(CASE WHEN re.status = 'PASS' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) as pass_rate,
  COALESCE(SUM(CASE WHEN re.status = 'FAIL' AND r.severity = 'HARD' THEN 1 ELSE 0 END), 0) as hard_breaches,
  COALESCE(SUM(CASE WHEN re.status = 'FAIL' AND r.severity = 'SOFT' THEN 1 ELSE 0 END), 0) as soft_breaches,
  JSONB_BUILD_OBJECT(
    'HARD', COALESCE(SUM(CASE WHEN re.status = 'FAIL' AND r.severity = 'HARD' THEN 1 ELSE 0 END), 0),
    'SOFT', COALESCE(SUM(CASE WHEN re.status = 'FAIL' AND r.severity = 'SOFT' THEN 1 ELSE 0 END), 0),
    'INFO', COALESCE(SUM(CASE WHEN re.status = 'FAIL' AND r.severity = 'INFO' THEN 1 ELSE 0 END), 0)
  ) as by_severity
FROM rules r
LEFT JOIN rule_evaluations re ON r.rule_id = re.rule_id
  AND re.tenant_id = @tenant_id
  AND DATE(re.evaluation_date) = @valuation_date::date
WHERE r.tenant_id = @tenant_id;
```

---

### 2. GET `/api/dashboard/risk`

**Purpose**: Risk metrics and VaR

**Query Parameters**:
```
tenant_id: string (required)
valuation_date: string (required, format: YYYY-MM-DD)
```

**Response**:
```json
{
  "avg_volatility": 0.1234,
  "avg_var_95": 125000.50,
  "avg_var_99": 185000.75,
  "worst_scenario": {
    "scenario_id": "equity-shock-20",
    "name": "Equity Market Down 20%",
    "pnl": -450000.25
  },
  "exposure_breakdown": {
    "equity": 0.45,
    "rates": 0.25,
    "credit": 0.20,
    "fx": 0.10
  }
}
```

**Implementation Pattern**:
```go
func (h *DashboardHandler) RiskSummary(w http.ResponseWriter, r *http.Request) {
  tenantID := r.URL.Query().Get("tenant_id")
  valuationDate := r.URL.Query().Get("valuation_date")
  
  summary, err := h.DB.GetDashboardRiskSummary(r.Context(), 
    db.GetDashboardRiskSummaryParams{
      TenantID:      tenantID,
      ValuationDate: valuationDate,
    })
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(summary)
}
```

---

### 3. GET `/api/dashboard/sparklines`

**Purpose**: 7-day historical sparkline data

**Query Parameters**:
```
tenant_id: string (required)
```

**Response**:
```json
{
  "pass_rate": [
    { "timestamp": "2024-01-09T00:00:00Z", "value": 0.88 },
    { "timestamp": "2024-01-10T00:00:00Z", "value": 0.89 },
    { "timestamp": "2024-01-11T00:00:00Z", "value": 0.90 },
    { "timestamp": "2024-01-12T00:00:00Z", "value": 0.91 },
    { "timestamp": "2024-01-13T00:00:00Z", "value": 0.91 },
    { "timestamp": "2024-01-14T00:00:00Z", "value": 0.92 },
    { "timestamp": "2024-01-15T00:00:00Z", "value": 0.92 }
  ],
  "hard_breaches": [
    { "timestamp": "2024-01-09T00:00:00Z", "value": 5 },
    { "timestamp": "2024-01-10T00:00:00Z", "value": 4 },
    { "timestamp": "2024-01-11T00:00:00Z", "value": 4 },
    { "timestamp": "2024-01-12T00:00:00Z", "value": 3 },
    { "timestamp": "2024-01-13T00:00:00Z", "value": 3 },
    { "timestamp": "2024-01-14T00:00:00Z", "value": 3 },
    { "timestamp": "2024-01-15T00:00:00Z", "value": 3 }
  ],
  "soft_breaches": [...],
  "volatility": [...],
  "etl_duration": [...]
}
```

**Database Pattern**: For each metric, select last 7 days with daily timestamp, aggregate by day

---

### 4. GET `/api/dashboard/etl-health`

**Purpose**: ETL run operational health

**Query Parameters**:
```
tenant_id: string (required)
```

**Response**:
```json
{
  "last_run": {
    "etl_run_id": "run-20240115-001",
    "status": "COMPLETED",
    "started_at": "2024-01-15T22:00:00Z",
    "completed_at": "2024-01-15T22:15:30Z",
    "duration_ms": 930000,
    "rules_evaluated": 125,
    "scenarios_evaluated": 45,
    "wasm_version": "1.2.3"
  },
  "success_rate": 0.98,
  "avg_duration_ms": 900000,
  "total_runs": 1000
}
```

---

### 5. GET `/api/dashboard/alerts`

**Purpose**: All active alerts, breaches, and failures

**Query Parameters**:
```
tenant_id: string (required)
valuation_date: string (required)
```

**Response**:
```json
{
  "hard_breaches": [
    {
      "rule_code": "LIQ_MIN_BUFFER",
      "metric_value": 0.05,
      "threshold_value": 0.10,
      "portfolio_id": "PORT-001",
      "severity": "HARD"
    }
  ],
  "soft_breaches": [
    {
      "rule_code": "ISSUER_CONCENTRATION",
      "metric_value": 0.25,
      "threshold_value": 0.20,
      "portfolio_id": "PORT-002",
      "severity": "SOFT"
    }
  ],
  "scenario_losses": [
    {
      "scenario_id": "equity-shock-20",
      "name": "Equity Market Down 20%",
      "pnl": -450000.25,
      "portfolio_id": "PORT-001"
    }
  ],
  "etl_failures": [
    {
      "etl_run_id": "run-20240114-002",
      "error_message": "Failed to compute WASM risk factors",
      "error_time": "2024-01-14T22:05:00Z"
    }
  ]
}
```

---

## ETL Endpoints

### 6. GET `/api/etl-runs`

**Query Parameters**:
```
tenant_id: string (required)
status: string (optional, QUEUED|RUNNING|COMPLETED|FAILED)
from: string (optional, YYYY-MM-DD)
to: string (optional, YYYY-MM-DD)
limit: int (default: 200)
```

**Response**:
```json
{
  "runs": [
    {
      "etl_run_id": "run-20240115-001",
      "tenant_id": "tenant-1",
      "valuation_date": "2024-01-15",
      "started_at": "2024-01-15T22:00:00Z",
      "completed_at": "2024-01-15T22:15:30Z",
      "status": "COMPLETED",
      "rules_evaluated": 125,
      "scenarios_evaluated": 45,
      "wasm_version": "1.2.3",
      "orchestrator_version": "2.0.1",
      "error_summary": null
    }
  ]
}
```

---

### 7. GET `/api/etl-runs/{id}`

**Response**:
```json
{
  "etl_run_id": "run-20240115-001",
  "tenant_id": "tenant-1",
  "valuation_date": "2024-01-15",
  "started_at": "2024-01-15T22:00:00Z",
  "completed_at": "2024-01-15T22:15:30Z",
  "status": "COMPLETED",
  "rules_evaluated": 125,
  "scenarios_evaluated": 45,
  "wasm_version": "1.2.3",
  "orchestrator_version": "2.0.1",
  "error_summary": null
}
```

---

## WASM Endpoints

### 8. GET `/api/wasm-versions`

**Query Parameters**:
```
module_name: string (required, e.g., "risk-engine", "compliance-checker")
```

**Response**:
```json
{
  "versions": [
    {
      "wasm_version_id": "wasm-1",
      "module_name": "risk-engine",
      "version": "1.2.3",
      "build_hash": "abc123def456",
      "build_time": "2024-01-15T10:00:00Z",
      "artifact_uri": "s3://bucket/wasm/risk-engine-1.2.3.wasm",
      "checksum_sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
      "is_active": true
    },
    {
      "wasm_version_id": "wasm-2",
      "module_name": "risk-engine",
      "version": "1.2.2",
      "build_hash": "xyz789uvw012",
      "build_time": "2024-01-14T10:00:00Z",
      "artifact_uri": "s3://bucket/wasm/risk-engine-1.2.2.wasm",
      "checksum_sha256": "5feceb66ffc86f38d952786c6d696c79c2dbc238c4cafb11f2271d7907e0f8aa",
      "is_active": false
    }
  ]
}
```

---

### 9. POST `/api/wasm-versions/{id}/activate`

**Request Body**: (none, or empty)

**Response**:
```json
{
  "wasm_version_id": "wasm-1",
  "module_name": "risk-engine",
  "version": "1.2.3",
  "is_active": true
}
```

**Implementation Pattern**:
```go
func (h *WASMHandler) ActivateVersion(w http.ResponseWriter, r *http.Request) {
  id := chi.URLParam(r, "id")
  
  // Update is_active = true for this version
  // Update is_active = false for all other versions of same module
  
  version, err := h.DB.ActivateWASMVersion(r.Context(), id)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  
  w.Header().Set("Content-Type", "application/json")
  json.NewEncoder(w).Encode(version)
}
```

---

## Lineage Endpoints

### 10. GET `/api/rules/{ruleId}/lineage`

**Path Parameters**:
```
ruleId: string (required, e.g., "MAX_ISSUER_5")
```

**Query Parameters**:
```
date_from: string (optional, YYYY-MM-DD)
date_to: string (optional, YYYY-MM-DD)
portfolio_id: string (optional)
```

**Response**:
```json
{
  "evaluations": [
    {
      "valuation_date": "2024-01-15",
      "portfolio_id": "PORT-001",
      "status": "PASS",
      "metric_value": 0.08,
      "threshold_value": 0.10,
      "etl_run_id": "run-20240115-001"
    },
    {
      "valuation_date": "2024-01-14",
      "portfolio_id": "PORT-001",
      "status": "FAIL",
      "metric_value": 0.12,
      "threshold_value": 0.10,
      "etl_run_id": "run-20240114-001"
    }
  ]
}
```

**Query Pattern**:
```sql
SELECT
  DATE(re.evaluation_date) as valuation_date,
  re.portfolio_id,
  re.status,
  re.metric_value,
  re.threshold_value,
  re.etl_run_id
FROM rule_evaluations re
WHERE re.rule_id = @rule_id
  AND (@portfolio_id IS NULL OR re.portfolio_id = @portfolio_id)
  AND (@date_from IS NULL OR DATE(re.evaluation_date) >= @date_from::date)
  AND (@date_to IS NULL OR DATE(re.evaluation_date) <= @date_to::date)
ORDER BY re.evaluation_date DESC
LIMIT 200;
```

---

### 11. GET `/api/scenarios/{scenarioId}/lineage`

**Path Parameters**:
```
scenarioId: string (required, e.g., "equity-shock-20")
```

**Query Parameters**:
```
date_from: string (optional, YYYY-MM-DD)
date_to: string (optional, YYYY-MM-DD)
portfolio_id: string (optional)
```

**Response**:
```json
{
  "results": [
    {
      "valuation_date": "2024-01-15",
      "portfolio_id": "PORT-001",
      "pnl": -450000.25,
      "etl_run_id": "run-20240115-001"
    },
    {
      "valuation_date": "2024-01-14",
      "portfolio_id": "PORT-001",
      "pnl": -425000.50,
      "etl_run_id": "run-20240114-001"
    }
  ]
}
```

---

## Router Setup (chi)

```go
// main.go or routes.go

func setupConsoleRoutes(r chi.Router, db *sql.DB) {
  dashboardHandler := &DashboardHandler{DB: db}
  etlHandler := &ETLHandler{DB: db}
  wasmHandler := &WASMHandler{DB: db}
  lineageHandler := &LineageHandler{DB: db}

  r.Route("/api", func(r chi.Router) {
    // Dashboard
    r.Get("/dashboard/compliance", dashboardHandler.ComplianceSummary)
    r.Get("/dashboard/risk", dashboardHandler.RiskSummary)
    r.Get("/dashboard/sparklines", dashboardHandler.Sparklines)
    r.Get("/dashboard/etl-health", dashboardHandler.ETLHealth)
    r.Get("/dashboard/alerts", dashboardHandler.Alerts)

    // ETL
    r.Get("/etl-runs", etlHandler.ListRuns)
    r.Get("/etl-runs/{id}", etlHandler.GetRun)

    // WASM
    r.Get("/wasm-versions", wasmHandler.ListVersions)
    r.Post("/wasm-versions/{id}/activate", wasmHandler.ActivateVersion)

    // Lineage
    r.Get("/rules/{ruleId}/lineage", lineageHandler.RuleLineage)
    r.Get("/scenarios/{scenarioId}/lineage", lineageHandler.ScenarioLineage)
  })
}
```

---

## Database Schema (Reference)

```sql
-- ETL Runs
CREATE TABLE etl_runs (
  etl_run_id UUID PRIMARY KEY,
  tenant_id VARCHAR(255),
  valuation_date DATE,
  started_at TIMESTAMP,
  completed_at TIMESTAMP,
  status VARCHAR(50), -- QUEUED, RUNNING, COMPLETED, FAILED
  rules_evaluated INT,
  scenarios_evaluated INT,
  wasm_version VARCHAR(100),
  orchestrator_version VARCHAR(100),
  error_summary TEXT,
  UNIQUE (tenant_id, valuation_date)
);

-- Rule Evaluations (for lineage)
CREATE TABLE rule_evaluations (
  rule_evaluation_id UUID PRIMARY KEY,
  tenant_id VARCHAR(255),
  rule_id VARCHAR(255),
  portfolio_id VARCHAR(255),
  evaluation_date TIMESTAMP,
  status VARCHAR(50), -- PASS, FAIL
  metric_value DECIMAL,
  threshold_value DECIMAL,
  etl_run_id UUID REFERENCES etl_runs(etl_run_id),
  FOREIGN KEY (etl_run_id) REFERENCES etl_runs(etl_run_id)
);

-- Scenario Results (for lineage)
CREATE TABLE scenario_results (
  scenario_result_id UUID PRIMARY KEY,
  tenant_id VARCHAR(255),
  scenario_id VARCHAR(255),
  portfolio_id VARCHAR(255),
  valuation_date DATE,
  pnl DECIMAL,
  etl_run_id UUID REFERENCES etl_runs(etl_run_id),
  FOREIGN KEY (etl_run_id) REFERENCES etl_runs(etl_run_id)
);

-- WASM Versions
CREATE TABLE wasm_versions (
  wasm_version_id UUID PRIMARY KEY,
  module_name VARCHAR(255),
  version VARCHAR(100),
  build_hash VARCHAR(255),
  build_time TIMESTAMP,
  artifact_uri TEXT,
  checksum_sha256 VARCHAR(255),
  is_active BOOLEAN,
  created_at TIMESTAMP,
  UNIQUE (module_name, version)
);
```

---

## Testing Your Endpoints

```bash
# Test dashboard compliance
curl http://localhost:8080/api/dashboard/compliance?tenant_id=tenant-1&valuation_date=2024-01-15

# Test ETL runs list
curl http://localhost:8080/api/etl-runs?tenant_id=tenant-1&limit=10

# Test WASM versions
curl http://localhost:8080/api/wasm-versions?module_name=risk-engine

# Test rule lineage
curl http://localhost:8080/api/rules/MAX_ISSUER_5/lineage

# Test activate WASM version
curl -X POST http://localhost:8080/api/wasm-versions/wasm-1/activate
```

---

**All endpoints return JSON. Make sure to set `Content-Type: application/json` in responses.**

All query parameters should be URL-encoded. Use standard HTTP status codes:
- 200 OK (success)
- 400 Bad Request (missing params)
- 404 Not Found (resource not found)
- 500 Internal Server Error (database error)
