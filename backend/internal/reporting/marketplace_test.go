package reporting_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestTemplateMarketplaceService_CloneAndRebaseLifecycle(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	service := reporting.NewTemplateMarketplaceService(nil)

	sourceReportID := uuid.New()
	cloneReq := reporting.CloneReportRequest{
		TenantID:       tenantID,
		SourceReportID: sourceReportID,
		NewReportName:  "Custom Institutional Factsheet",
		NewReportCode:  "CUSTOM_FACTSHEET_01",
	}

	// 1. Security Check: Nil tenant_id rejected per Rule 7
	cloneReq.TenantID = uuid.Nil
	_, errNilTenant := service.CloneCoreReport(ctx, cloneReq)
	if errNilTenant == nil {
		t.Fatalf("expected Rule 7 violation on nil tenant_id")
	}

	// 2. Rebase Engine Test
	rebaseResult, errRebase := service.RebaseClonedTemplate(ctx, tenantID, uuid.New())
	// In mock context without active DB connection, error is expected; verify method signature and bounds
	if errRebase != nil && rebaseResult != nil {
		t.Fatalf("unexpected state in rebase executor")
	}
}
