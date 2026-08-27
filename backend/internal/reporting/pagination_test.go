package reporting_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

func TestPaginationCompiler_StrictOnePageBudget(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	compiler := reporting.NewPaginationCompiler()

	constraint := reporting.PageBudgetConstraint{
		TargetMaxPages: 1,
		Format:         reporting.PageFormatA4Portrait,
		Margins:        reporting.PageMargins{TopMM: 15, BottomMM: 15, LeftMM: 15, RightMM: 15},
		DPI:            300,
	}

	reportHeader := &reporting.MeasuredSection{
		SectionID: "hdr_1",
		Type:      reporting.SectionReportHeader,
		HeightMM:  30.0,
	}

	bodySections := []reporting.MeasuredSection{
		{SectionID: "body_1", Type: reporting.SectionBodyDetail, HeightMM: 50.0},
		{SectionID: "body_2", Type: reporting.SectionBodyDetail, HeightMM: 60.0},
		{SectionID: "body_3", Type: reporting.SectionBodyDetail, HeightMM: 80.0},
	}

	// 1. Valid Execution: Total height (30 + 50 + 60 + 80 = 220mm) <= 267mm usable
	pages, err := compiler.CompilePhysicalPages(ctx, tenantID, constraint, reportHeader, nil, bodySections, nil, nil)
	if err != nil {
		t.Fatalf("expected compilation to succeed within 1-page budget, got: %v", err)
	}

	if len(pages) != 1 {
		t.Fatalf("expected exactly 1 page, got %d", len(pages))
	}

	if pages[0].IsOverflown {
		t.Errorf("expected page to not be marked as overflown")
	}

	// 2. Budget Overflow Execution: Adding an extra 70mm section triggers a page budget violation
	overflowSections := append(bodySections, reporting.MeasuredSection{
		SectionID: "body_4",
		Type:      reporting.SectionBodyDetail,
		HeightMM:  70.0,
	})

	_, errOverflow := compiler.CompilePhysicalPages(ctx, tenantID, constraint, reportHeader, nil, overflowSections, nil, nil)
	if errOverflow == nil {
		t.Fatalf("expected page budget violation error, but compilation succeeded")
	}
}
