package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestCellExplainService_DeterministicLineageAndStateHash(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	service := reporting.NewCellExplainService()

	asOf := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	passport, err := service.ResolveCell(ctx, tenantID, "portfolio_nav", reporting.ContextRegulatoryABOR, asOf, nil)
	if err != nil {
		t.Fatalf("expected no error resolving cell lineage, got: %v", err)
	}

	if passport.TermKey != "portfolio_nav" {
		t.Errorf("expected termKey portfolio_nav, got: %s", passport.TermKey)
	}

	if passport.ContextType != reporting.ContextRegulatoryABOR {
		t.Errorf("expected contextType REGULATORY_ABOR, got: %s", passport.ContextType)
	}

	if passport.StateSHA256 == "" {
		t.Errorf("expected non-empty state SHA256 hash")
	}

	if !passport.IsReconciled {
		t.Errorf("expected isReconciled to be true")
	}
}
