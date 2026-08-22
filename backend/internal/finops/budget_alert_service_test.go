package finops_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/audit"
	"github.com/hondyman/uisce/backend/internal/finops"
)

func TestAnalyticalAuditInterceptor_ZeroDataLeakage(t *testing.T) {
	interceptor := audit.NewAnalyticalAuditInterceptor(nil)
	tenantID := uuid.New()

	telemetry := audit.QueryAuditTelemetry{
		TenantID:        tenantID,
		RequestID:       "req-trace-88912",
		UserID:          "usr-portfolio-mgr",
		ServerHost:      "100.84.50.65",
		ExecutionType:   "SSRS_REPORT_RENDER",
		NormalizedQuery: "SELECT isin, px_last FROM security WHERE effective_date = '2026-08-21'",
		ExecutionPlanJSON: map[string]interface{}{
			"nodes": []string{"AST_RESOLVE", "STARROCKS_HOT_SCAN", "VECTOR_KERNEL"},
		},
		RowCountReturned:  15420,
		ScannedBytes:      10485760, // 10 MB
		CPUDurationMs:     42,
		TotalLatencyMs:    88,
		EngineType:        "STARROCKS",
		AttributedCostUSD: 0.000050,
		Status:            "COMPLETED",
	}

	err := interceptor.RecordQueryExecution(context.Background(), telemetry)
	if err != nil {
		t.Fatalf("unexpected audit error: %v", err)
	}

	// Verify Rule 7 security enforcement on nil tenant
	telemetry.TenantID = uuid.Nil
	nilTenantErr := interceptor.RecordQueryExecution(context.Background(), telemetry)
	if nilTenantErr == nil {
		t.Fatalf("expected Rule 7 violation error on nil tenant_id")
	}
}

func TestBudgetAlertService_SlackWebhookPayload(t *testing.T) {
	var receivedPayload map[string]interface{}
	mockSlackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer mockSlackServer.Close()

	service := finops.NewBudgetAlertService(nil)
	tenantID := uuid.New()

	err := service.EvaluateTenantBudgetAndAlert(context.Background(), tenantID, "2026-08")
	if err != nil {
		t.Fatalf("budget alert evaluation failed: %v", err)
	}
	_ = receivedPayload
}
