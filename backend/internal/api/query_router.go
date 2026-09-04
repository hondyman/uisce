package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/optimizer"
)

type QueryRouter struct {
	auditor *QueryExecutionAuditor
}

func NewQueryRouter(auditor *QueryExecutionAuditor) *QueryRouter {
	return &QueryRouter{auditor: auditor}
}

// HandleSemanticQuery processes analytical queries through the zero-leak audit wrapper
func (r *QueryRouter) HandleSemanticQuery(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tenantID, err := uuid.Parse(getSecureTenantID(req))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context (Rule 7)"}`, http.StatusBadRequest)
		return
	}

	requestID := req.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = uuid.New().String()
	}
	userID := req.Header.Get("X-User-ID")
	if userID == "" {
		userID = "anonymous_user"
	}

	var payload struct {
		AST         optimizer.QueryAST `json:"ast"`
		CompiledSQL string             `json:"compiledSql"`
		EngineType  string             `json:"engineType"` // STARROCKS, POSTGRES_ALPHA, ICEBERG
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid query payload"}`, http.StatusBadRequest)
		return
	}

	if payload.EngineType == "" {
		payload.EngineType = "STARROCKS"
	}

	execCtx := ExecutionContextRequest{
		TenantID:      tenantID,
		RequestID:     requestID,
		UserID:        userID,
		ExecutionType: "SEMANTIC_QUERY",
		CompiledSQL:   payload.CompiledSQL,
		AST:           payload.AST,
		EngineType:    payload.EngineType,
	}

	// Mock physical execution engine callback
	mockEngineRunner := func(ctx context.Context, sql string) ([]map[string]interface{}, int64, int, error) {
		mockData := []map[string]interface{}{
			{"isin": "US0378331005", "security_name": "Apple Inc.", "px_last": 185.50},
			{"isin": "US5949181045", "security_name": "Microsoft Corp.", "px_last": 410.20},
		}
		return mockData, 1048576, 12, nil // 1MB scanned, 12ms CPU
	}

	result, err := r.auditor.ExecuteAndAudit(req.Context(), execCtx, mockEngineRunner)
	if err != nil {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

// HandleSSRSReportRender logs report generation telemetry while preserving confidential data
func (r *QueryRouter) HandleSSRSReportRender(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tenantID, err := uuid.Parse(getSecureTenantID(req))
	if err != nil {
		http.Error(w, `{"error":"invalid tenant context (Rule 7)"}`, http.StatusBadRequest)
		return
	}

	var payload struct {
		ReportTemplateID string             `json:"reportTemplateId"`
		AST              optimizer.QueryAST `json:"ast"`
		CompiledSQL      string             `json:"compiledSql"`
	}
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid report payload"}`, http.StatusBadRequest)
		return
	}

	execCtx := ExecutionContextRequest{
		TenantID:      tenantID,
		RequestID:     uuid.New().String(),
		UserID:        req.Header.Get("X-User-ID"),
		ExecutionType: "SSRS_REPORT_RENDER",
		CompiledSQL:   payload.CompiledSQL,
		AST:           payload.AST,
		EngineType:    "POSTGRES_ALPHA",
	}

	mockReportRunner := func(ctx context.Context, sql string) ([]map[string]interface{}, int64, int, error) {
		mockRows := make([]map[string]interface{}, 500)
		return mockRows, 5242880, 45, nil // 5MB scanned, 45ms CPU
	}

	result, err := r.auditor.ExecuteAndAudit(req.Context(), execCtx, mockReportRunner)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}
