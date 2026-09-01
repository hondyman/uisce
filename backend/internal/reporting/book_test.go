package reporting_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

type MockSubReportRenderer struct{}

func (m *MockSubReportRenderer) RenderDocument(
	ctx context.Context,
	format, templateJSON string,
	clientData map[string]interface{},
) ([]byte, error) {
	// Emit 35KB mock vector PDF per page
	return make([]byte, 35000), nil
}

func TestReportBookCompiler_TwoPassTOCAndStitching(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	compiler := reporting.NewReportBookCompiler(&MockSubReportRenderer{})

	spec := reporting.ReportBookSpec{
		BookID:             uuid.New(),
		TenantID:           tenantID,
		BookName:           "Q2 2026 Executive Review Booklet",
		IncludeTOC:         true,
		TOCTitle:           "Table of Contents",
		PageNumberingStyle: "CONTINUOUS",
		PageFormat:         reporting.PageFormatA4Portrait,
		Sections: []reporting.BookSectionConfig{
			{
				SectionID:          uuid.New(),
				ReportDefinitionID: uuid.New(),
				SectionOrder:       1,
				ChapterTitle:       "Executive Summary",
				InsertDividerTab:   false,
			},
			{
				SectionID:          uuid.New(),
				ReportDefinitionID: uuid.New(),
				SectionOrder:       2,
				ChapterTitle:       "Holdings & Asset Allocation",
				InsertDividerTab:   true,
			},
		},
	}

	asOfDate := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	envelope := map[string]interface{}{
		"presentation_ccy": "USD",
		"benchmark_id":     "MSCI_WORLD",
	}

	booklet, err := compiler.CompileReportBooklet(ctx, tenantID, spec, "ACC_GLOBAL_01", asOfDate, envelope)
	if err != nil {
		t.Fatalf("booklet compilation failed: %v", err)
	}

	// Verify Cover (1) + TOC (1) + Section 1 (1) + Section 2 Divider (1) + Section 2 Page (1) = 5 Pages Total
	if booklet.TotalPages != 5 {
		t.Errorf("expected total pages = 5, got %d", booklet.TotalPages)
	}

	if len(booklet.TOCEntries) != 2 {
		t.Fatalf("expected 2 TOC entries, got %d", len(booklet.TOCEntries))
	}

	// Verify TOC Target Page Offsets
	if booklet.TOCEntries[0].StartPageNum != 3 {
		t.Errorf("expected Section 1 start page = 3, got %d", booklet.TOCEntries[0].StartPageNum)
	}

	if booklet.SHA256Checksum == "" {
		t.Errorf("expected valid SHA-256 checksum for compiled booklet")
	}
}
