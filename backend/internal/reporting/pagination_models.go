package reporting

type Millimeter float64

type PageFormat string

const (
	PageFormatA4Portrait      PageFormat = "A4_PORTRAIT"
	PageFormatA4Landscape     PageFormat = "A4_LANDSCAPE"
	PageFormatLetterPortrait  PageFormat = "LETTER_PORTRAIT"
	PageFormatLetterLandscape PageFormat = "LETTER_LANDSCAPE"
)

type PageDimensions struct {
	WidthMM  Millimeter `json:"widthMm"`
	HeightMM Millimeter `json:"heightMm"`
}

var StandardDimensions = map[PageFormat]PageDimensions{
	PageFormatA4Portrait:      {WidthMM: 210.0, HeightMM: 297.0},
	PageFormatA4Landscape:     {WidthMM: 297.0, HeightMM: 210.0},
	PageFormatLetterPortrait:  {WidthMM: 215.9, HeightMM: 279.4},
	PageFormatLetterLandscape: {WidthMM: 279.4, HeightMM: 215.9},
}

type PageMargins struct {
	TopMM    Millimeter `json:"topMm"`
	BottomMM Millimeter `json:"bottomMm"`
	LeftMM   Millimeter `json:"leftMm"`
	RightMM  Millimeter `json:"rightMm"`
}

type PageBudgetConstraint struct {
	TargetMaxPages int         `json:"targetMaxPages"` // e.g., 1 for FactSheet, 2 for TearSheet, 0 for unbounded
	Format         PageFormat  `json:"format"`
	Margins        PageMargins `json:"margins"`
	DPI            int         `json:"dpi"` // Default: 300 for Print, 96 for Web
}

type LayoutSectionType string

const (
	SectionReportHeader LayoutSectionType = "REPORT_HEADER"
	SectionPageHeader   LayoutSectionType = "PAGE_HEADER"
	SectionGroupHeader  LayoutSectionType = "GROUP_HEADER"
	SectionBodyDetail   LayoutSectionType = "BODY_DETAIL"
	SectionGroupFooter  LayoutSectionType = "GROUP_FOOTER"
	SectionPageFooter   LayoutSectionType = "PAGE_FOOTER"
	SectionReportFooter LayoutSectionType = "REPORT_FOOTER"
)

type MeasuredSection struct {
	SectionID       string            `json:"sectionId"`
	Type            LayoutSectionType `json:"type"`
	HeightMM        Millimeter        `json:"heightMm"`
	KeepWithNext    bool              `json:"keepWithNext"`
	PageBreakBefore bool              `json:"pageBreakBefore"`
	PageBreakAfter  bool              `json:"pageBreakAfter"`
	RenderPayload   interface{}       `json:"renderPayload"`
}

type CompiledPage struct {
	PageNumber   int               `json:"pageNumber"`
	TotalPages   int               `json:"totalPages"`
	Sections     []MeasuredSection `json:"sections"`
	UsedHeightMM Millimeter        `json:"usedHeightMm"`
	MaxHeightMM  Millimeter        `json:"maxHeightMm"`
	IsOverflown  bool              `json:"isOverflown"`
}
