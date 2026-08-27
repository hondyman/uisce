package reporting

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PaginationCompiler struct{}

func NewPaginationCompiler() *PaginationCompiler {
	return &PaginationCompiler{}
}

// CompilePhysicalPages partitions evaluated report elements into exact millimeter-bounded pages
func (c *PaginationCompiler) CompilePhysicalPages(
	ctx context.Context,
	tenantID uuid.UUID,
	constraint PageBudgetConstraint,
	reportHeader *MeasuredSection,
	pageHeader *MeasuredSection,
	bodySections []MeasuredSection,
	pageFooter *MeasuredSection,
	reportFooter *MeasuredSection,
) ([]CompiledPage, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	dims, exists := StandardDimensions[constraint.Format]
	if !exists {
		dims = StandardDimensions[PageFormatA4Portrait]
	}

	usableHeight := dims.HeightMM - (constraint.Margins.TopMM + constraint.Margins.BottomMM)
	if pageHeader != nil {
		usableHeight -= pageHeader.HeightMM
	}
	if pageFooter != nil {
		usableHeight -= pageFooter.HeightMM
	}

	if usableHeight <= 0 {
		return nil, fmt.Errorf("invalid margins: usable height is %f mm", usableHeight)
	}

	pages := make([]CompiledPage, 0)
	currentPage := CompiledPage{
		PageNumber:   1,
		Sections:     make([]MeasuredSection, 0),
		UsedHeightMM: 0,
		MaxHeightMM:  usableHeight,
	}

	// 1. Report Header on Page 1
	if reportHeader != nil {
		currentPage.Sections = append(currentPage.Sections, *reportHeader)
		currentPage.UsedHeightMM += reportHeader.HeightMM
	}

	// 2. Iterate Body Sections and Apply Break Rules
	for i, section := range bodySections {
		// Explicit PageBreakBefore
		if section.PageBreakBefore && len(currentPage.Sections) > 0 {
			pages = append(pages, currentPage)
			currentPage = CompiledPage{
				PageNumber:   len(pages) + 1,
				Sections:     make([]MeasuredSection, 0),
				UsedHeightMM: 0,
				MaxHeightMM:  usableHeight,
			}
		}

		// KeepWithNext lookahead check
		requiredSpace := section.HeightMM
		if section.KeepWithNext && i+1 < len(bodySections) {
			requiredSpace += bodySections[i+1].HeightMM
		}

		// Overflow check
		if currentPage.UsedHeightMM+requiredSpace > usableHeight && len(currentPage.Sections) > 0 {
			pages = append(pages, currentPage)
			currentPage = CompiledPage{
				PageNumber:   len(pages) + 1,
				Sections:     make([]MeasuredSection, 0),
				UsedHeightMM: 0,
				MaxHeightMM:  usableHeight,
			}
		}

		currentPage.Sections = append(currentPage.Sections, section)
		currentPage.UsedHeightMM += section.HeightMM

		// Explicit PageBreakAfter
		if section.PageBreakAfter {
			pages = append(pages, currentPage)
			currentPage = CompiledPage{
				PageNumber:   len(pages) + 1,
				Sections:     make([]MeasuredSection, 0),
				UsedHeightMM: 0,
				MaxHeightMM:  usableHeight,
			}
		}
	}

	// 3. Report Footer
	if reportFooter != nil {
		if currentPage.UsedHeightMM+reportFooter.HeightMM > usableHeight && len(currentPage.Sections) > 0 {
			pages = append(pages, currentPage)
			currentPage = CompiledPage{
				PageNumber:   len(pages) + 1,
				Sections:     make([]MeasuredSection, 0),
				UsedHeightMM: 0,
				MaxHeightMM:  usableHeight,
			}
		}
		currentPage.Sections = append(currentPage.Sections, *reportFooter)
		currentPage.UsedHeightMM += reportFooter.HeightMM
	}

	if len(currentPage.Sections) > 0 || len(pages) == 0 {
		pages = append(pages, currentPage)
	}

	// Set total page count across all compiled pages
	totalPages := len(pages)
	for i := range pages {
		pages[i].TotalPages = totalPages
		if pages[i].UsedHeightMM > usableHeight {
			pages[i].IsOverflown = true
		}
	}

	// Validate against strict page budget target (e.g. 1-page tear sheet)
	if constraint.TargetMaxPages > 0 && totalPages > constraint.TargetMaxPages {
		return pages, fmt.Errorf("page budget violation: generated %d pages, maximum allowed is %d", totalPages, constraint.TargetMaxPages)
	}

	return pages, nil
}
