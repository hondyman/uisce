package layout

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestLayoutResolverService(t *testing.T) {
	svc := NewLayoutResolverService(nil)

	res, err := svc.ResolvePageLayout(context.Background(), uuid.New(), uuid.New(), "business_object_detail")
	if err != nil {
		t.Fatalf("unexpected error resolving page layout: %v", err)
	}

	if len(res.Fields) == 0 {
		t.Errorf("expected hydrated fields")
	}

	if res.Capabilities["temporalStrategy"] != "BITEMPORAL_SEAM" {
		t.Errorf("expected BITEMPORAL_SEAM capability")
	}
}
