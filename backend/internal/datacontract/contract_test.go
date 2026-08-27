package datacontract_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/datacontract"
)

func TestDataContractService_Rule7TenantScoping(t *testing.T) {
	ctx := context.Background()
	service := datacontract.NewDataContractService(nil)

	// Security Verification: Nil tenant_id must be rejected immediately per Rule 7
	_, err := service.CompileContractFromBO(ctx, uuid.Nil, uuid.New(), "wealth_team", "1.0.0")
	if err == nil {
		t.Fatalf("expected Rule 7 violation error when tenant_id is nil, got nil")
	}
}
