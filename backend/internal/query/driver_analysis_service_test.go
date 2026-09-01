package query

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestDriverAnalysisService(t *testing.T) {
	svc := NewDriverAnalysisService(nil)
	ctx := context.Background()
	tenantID := uuid.New()

	report, err := svc.ExplainMetricVariance(ctx, tenantID, "net_fund_return", "2026-07", "2026-08")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.TopContributors) == 0 {
		t.Errorf("expected top contributors")
	}

	if report.CertifiedGoldenMatch == nil {
		t.Errorf("expected golden asset match")
	}

	if len(report.AnomaliesDetected) == 0 {
		t.Errorf("expected anomaly detection")
	}
}
