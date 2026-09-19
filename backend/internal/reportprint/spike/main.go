// Package main is a throwaway spike (HANDOFF_REPORT_BUILDER_SPINE_PLAN.md
// ticket 0.3), not part of the running application. It answers one
// question: does a thin, hand-rolled band-layout engine over a low-level
// PDF primitives library (go-pdf/fpdf, the maintained successor to the
// archived jung-kurt/gofpdf) fit this project's report model better than
// a grid library like Maroto v2 that owns its own pagination?
//
// Research finding (WebSearch, this ticket): Maroto "automatically adds a
// new page when a row will extrapolate the useful area of a page" — it
// decides page breaks by row overflow, and its manual `AddPages` escape
// hatch can still be re-split by Maroto if the caller's page doesn't fit
// its own calculated useful area. Maroto also has no group-level
// repeating header/footer (only page-level), which the grouped/subtotaled
// report kind (Phase 4) needs. That's the "grid lib wants to own
// pagination, and pagination is the report engine's job" conflict
// predicted before this spike — confirmed by Maroto's own docs, not just
// guessed.
//
// This spike proves the alternative is simple, not just theoretically
// preferable: bands are rectangles with a known height: (1) place a band
// at the current Y; (2) advance Y by the band's height; (3) before
// placing a band, check whether it fits in what's left of the page and
// call fpdf.AddPage() if not, re-emitting sticky bands (page header) on
// the new page. That's the whole engine — no library-owned auto-pagination
// to fight, full control over group-header/group-footer/page-break-on-
// group-change semantics the report model actually needs.
//
// To run: this package needs the DejaVu font files go-pdf/fpdf bundles as
// its own examples, copied next to main.go (not committed here - binary
// assets, redistribute from the module cache instead):
//
//	cp "$(go env GOPATH)/pkg/mod/github.com/go-pdf/fpdf@v0.9.0"/font/DejaVuSansCondensed{,-Bold}.ttf .
//	go run .
package main

import (
	"fmt"
	"log"

	"github.com/go-pdf/fpdf"
)

// band is the same shape ReportLayout's future `bands` column (2.1) would
// carry: a fixed height and a list of field placements to draw within it.
// Real bands would carry many more field/style properties; this spike only
// needs enough to prove the page-fitting mechanics and font handling.
type band struct {
	heightMM float64
	draw     func(pdf *fpdf.Fpdf, topY float64)
}

// reportPrinter is the whole "engine": current cursor position, page
// geometry, and the one piece of state a real renderer would need beyond
// this spike — a sticky page-header band re-emitted on every new page.
type reportPrinter struct {
	pdf          *fpdf.Fpdf
	y            float64
	pageHeight   float64
	bottomMargin float64
	pageHeader   band
}

// newReportPrinterFromPDF takes an already-configured *fpdf.Fpdf (fonts
// registered) rather than constructing one, since fonts must be registered
// before the first band (the sticky page header) draws.
func newReportPrinterFromPDF(pdf *fpdf.Fpdf, pageHeader band) *reportPrinter {
	pdf.SetMargins(15, 15, 15)
	_, pageHeight := pdf.GetPageSize()
	rp := &reportPrinter{pdf: pdf, pageHeight: pageHeight, bottomMargin: 20, pageHeader: pageHeader}
	rp.newPage()
	return rp
}

func (rp *reportPrinter) newPage() {
	rp.pdf.AddPage()
	rp.y = 15
	rp.placeBand(rp.pageHeader)
}

// placeBand is the entire pagination decision: does this band fit in what's
// left of the page? If not, start a new page (which re-emits the sticky
// page header) before drawing. No library-owned row-overflow logic to
// fight — the report engine decides, band by band, exactly like SSRS/
// Crystal/Jasper's band model, because that model is what the whole spine
// plan's report kinds (Document/List/Grouped) are built around.
func (rp *reportPrinter) placeBand(b band) {
	if rp.y+b.heightMM > rp.pageHeight-rp.bottomMargin {
		rp.newPage()
		if b.heightMM == rp.pageHeader.heightMM {
			return // the page header IS what newPage() just placed
		}
	}
	b.draw(rp.pdf, rp.y)
	rp.y += b.heightMM
}

func main() {
	pageHeader := band{
		heightMM: 18,
		draw: func(pdf *fpdf.Fpdf, y float64) {
			pdf.SetY(y)
			pdf.SetFont("DejaVu", "B", 14)
			pdf.CellFormat(0, 8, "Order Invoice — Report Builder spine spike", "", 1, "L", false, 0, "")
			pdf.SetFont("DejaVu", "", 9)
			pdf.SetY(y + 9)
			pdf.CellFormat(0, 6, "Customer: Société Générale — Account: 4711-€UR", "", 1, "L", false, 0, "")
		},
	}

	// AddUTF8Font, not AddFont: this is the unicode-font check ticket 0.3
	// asked for. DejaVuSans ships as one of go-pdf/fpdf's bundled example
	// fonts; a real deployment would register the platform's actual
	// invoice font the same way. The € sign and "é" above are the point —
	// gofpdf's classic single-byte fonts render those as "tofu" (see
	// jung-kurt/gofpdf#250, found in this ticket's research); AddUTF8Font
	// is what fixes it. Must happen before the first band draws (the page
	// header renders immediately inside newReportPrinter), so it's
	// registered on the raw *fpdf.Fpdf before the printer is constructed.
	pdf := fpdf.New("P", "mm", "A4", ".")
	pdf.AddUTF8Font("DejaVu", "", "DejaVuSansCondensed.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "DejaVuSansCondensed-Bold.ttf")

	rp := newReportPrinterFromPDF(pdf, pageHeader)

	detailRow := func(label, qty, price string) band {
		return band{
			heightMM: 7,
			draw: func(pdf *fpdf.Fpdf, y float64) {
				pdf.SetY(y)
				pdf.SetFont("DejaVu", "", 9)
				pdf.CellFormat(100, 7, label, "B", 0, "L", false, 0, "")
				pdf.CellFormat(40, 7, qty, "B", 0, "R", false, 0, "")
				pdf.CellFormat(40, 7, price, "B", 1, "R", false, 0, "")
			},
		}
	}

	// Enough detail rows to force a real page break, proving placeBand's
	// fit-check + sticky-header re-emission actually fires, not just
	// renders a single page.
	for i := 1; i <= 45; i++ {
		rp.placeBand(detailRow(
			fmt.Sprintf("Line item %d — Allocation €%d.00", i, 100+i),
			fmt.Sprintf("%d", i),
			fmt.Sprintf("€%.2f", float64(i)*12.5),
		))
	}

	outPath := "/tmp/report_spine_band_spike.pdf"
	if err := rp.pdf.OutputFileAndClose(outPath); err != nil {
		log.Fatalf("spike failed: %v", err)
	}
	fmt.Printf("wrote %s (%d pages)\n", outPath, rp.pdf.PageCount())
}
