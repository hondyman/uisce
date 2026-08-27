package bo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestValidationService(t *testing.T) {
	svc := NewValidationService(nil)
	ctx := context.Background()
	tenantID := uuid.New()
	boID := uuid.New()

	report, err := svc.ValidateBusinessObjectForPublish(ctx, tenantID, boID)
	if err != nil {
		t.Fatalf("unexpected error validating BO: %v", err)
	}

	if report.Status != "READY_TO_PUBLISH" {
		t.Errorf("expected READY_TO_PUBLISH, got %s", report.Status)
	}
}
