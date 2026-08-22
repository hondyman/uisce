package boresolver_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
)

func TestBODiscoveryService_TenantGuard(t *testing.T) {
	service := boresolver.NewBODiscoveryService(nil)

	req := boresolver.BindingContextRequest{
		TenantID:      uuid.Nil, // Should trigger Rule 7 violation
		BackendID:     uuid.New(),
		DrivingNodeID: uuid.New(),
	}

	_, err := service.DiscoverBindingContext(context.Background(), req)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error on nil tenant_id")
	}
}

func TestBOSaveService_AtomicSaveValidation(t *testing.T) {
	service := boresolver.NewBOSaveService(nil)
	tenantID := uuid.New()

	req := boresolver.AtomicSaveBORequest{
		TenantID: tenantID,
		ModelID:  uuid.New(),
		Publish:  true,
	}
	req.BusinessObject.BOKey = "customer"
	req.BusinessObject.BOName = "Customer"
	req.BusinessObject.BOType = "ENTITY"

	boID, err := service.SaveBusinessObjectAtomic(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	if boID == uuid.Nil {
		t.Errorf("expected valid generated boID")
	}
}
