package bo

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestBusinessObjectService_TenantNilCheck(t *testing.T) {
	svc := NewBusinessObjectService(nil, nil)

	var req SaveBusinessObjectRequest
	req.TenantID = uuid.Nil

	_, err := svc.SaveBusinessObjectAtomic(context.Background(), "user-1", "admin", req)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error for nil tenant_id, got nil")
	}

	expected := "Rule 7 violation: tenant_id cannot be nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestDiscoveryService_TenantNilCheck(t *testing.T) {
	svc := NewDiscoveryService(nil)

	req := BindingContextRequest{
		TenantID: uuid.Nil,
	}

	_, err := svc.ResolveBindingContext(context.Background(), req)
	if err == nil {
		t.Fatalf("expected Rule 7 violation error for nil tenant_id, got nil")
	}

	expected := "Rule 7 violation: tenant_id cannot be nil"
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}
