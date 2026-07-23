package reports

import (
	"encoding/json"
	"fmt"
	"time"
)

// SSRSGenerator generates paginated SSRS-style reports with drill-down navigation
// Produces structured report data that can be rendered to PDF/HTML
type SSRSGenerator struct {
	pageSize         int // Lines per page (typically 20 for table rows)
	includeMetrics   bool
	includeDrillDown bool
}

// GeneratedReport represents a complete report ready for rendering
type GeneratedReport struct {
	CoverPage      CoverPageData   `json:"cover_page"`
	Pages          []GeneratedPage `json:"pages"`
	DrillDownPages []DrillDownPage `json:"drill_down_pages"`
	Metadata       ReportMetadata  `json:"metadata"`
}

// CoverPageData represents the cover page
type CoverPageData struct {
	Title         string  `json:"title"`
	HouseholdName string  `json:"household_name"`
	HouseholdType string  `json:"household_type"`
	Status        string  `json:"status"`
	TotalAssets   float64 `json:"total_assets"`
	EntityCount   int     `json:"entity_count"`
	GeneratedDate string  `json:"generated_date"`
	SemanticView  string  `json:"semantic_view"`
}

// GeneratedPage represents a single report page
type GeneratedPage struct {
	PageNum     int            `json:"page_num"`
	Title       string         `json:"title"`
	SectionType string         `json:"section_type"`
	TableData   TableData      `json:"table_data"`
	Summary     map[string]any `json:"summary"`
	DrillLinks  []DrillLink    `json:"drill_links"`
}

// TableData represents paginated table content
type TableData struct {
	Headers  []string   `json:"headers"`
	Rows     [][]string `json:"rows"`
	RowCount int        `json:"row_count"`
	PageNum  int        `json:"page_num"`
	PageSize int        `json:"page_size"`
}

// DrillLink represents a link to another page
type DrillLink struct {
	Label      string `json:"label"`
	TargetPage string `json:"target_page"`
	DrillType  string `json:"drill_type"` // 'next', 'prev', 'detail'
}

// DrillDownPage represents a detailed drill-down page
type DrillDownPage struct {
	PageName   string    `json:"page_name"`
	Title      string    `json:"title"`
	TableData  TableData `json:"table_data"`
	DetailText string    `json:"detail_text"`
	BackLink   DrillLink `json:"back_link"`
}

// ReportMetadata captures report generation metadata
type ReportMetadata struct {
	GeneratedAt   time.Time `json:"generated_at"`
	TotalPages    int       `json:"total_pages"`
	TotalEntities int       `json:"total_entities"`
	GenerationMS  int64     `json:"generation_ms"`
}

// NewSSRSGenerator creates a new SSRS report generator
func NewSSRSGenerator() *SSRSGenerator {
	return &SSRSGenerator{
		pageSize:         20,
		includeMetrics:   true,
		includeDrillDown: true,
	}
}

// ============================================================================
// REPORT GENERATION
// ============================================================================

// GenerateReportStructure creates the structured report data for rendering to PDF/HTML
func (sg *SSRSGenerator) GenerateReportStructure(household *Household, cube *SemanticCube, pages []ReportPage, drillPaths map[string][]string) (*GeneratedReport, error) {
	startTime := time.Now()

	// Generate cover page
	coverPage := sg.generateCoverPageData(household, cube)

	// Generate report pages
	generatedPages := sg.generateReportPageData(pages)

	// Generate drill-down pages if enabled
	drillDownPages := []DrillDownPage{}
	if len(drillPaths) > 0 && sg.includeDrillDown {
		drillDownPages = sg.generateDrillDownPageData(cube, pages)
	}

	// Create metadata
	metadata := ReportMetadata{
		GeneratedAt:   time.Now(),
		TotalPages:    len(generatedPages) + len(drillDownPages) + 1, // +1 for cover
		TotalEntities: len(cube.Entities),
		GenerationMS:  time.Since(startTime).Milliseconds(),
	}

	return &GeneratedReport{
		CoverPage:      coverPage,
		Pages:          generatedPages,
		DrillDownPages: drillDownPages,
		Metadata:       metadata,
	}, nil
}

// generateCoverPageData creates cover page structured data
func (sg *SSRSGenerator) generateCoverPageData(household *Household, cube *SemanticCube) CoverPageData {
	totalAssets := cube.Summary["total_value"].(float64)
	entityCount := int(cube.Summary["entity_count"].(float64))

	return CoverPageData{
		Title:         "Household Report",
		HouseholdName: household.Name,
		HouseholdType: household.HouseholdType,
		Status:        household.Status,
		TotalAssets:   totalAssets,
		EntityCount:   entityCount,
		GeneratedDate: time.Now().Format("January 2, 2006"),
		SemanticView:  cube.ViewName,
	}
}

// generateReportPageData converts ReportPages to GeneratedPages
func (sg *SSRSGenerator) generateReportPageData(pages []ReportPage) []GeneratedPage {
	generated := make([]GeneratedPage, 0, len(pages))

	for pageIdx, page := range pages {
		tableData := sg.entitiesToTableData(page.Entities, pageIdx)

		drillLinks := []DrillLink{}
		for _, target := range page.DrillTargets {
			drillType := "detail"
			if target == "next" {
				drillType = "next"
			} else if target == "prev" {
				drillType = "prev"
			}

			drillLinks = append(drillLinks, DrillLink{
				Label:      fmt.Sprintf("→ %s", target),
				TargetPage: target,
				DrillType:  drillType,
			})
		}

		genPage := GeneratedPage{
			PageNum:     pageIdx + 1,
			Title:       page.Title,
			SectionType: page.SectionType,
			TableData:   tableData,
			Summary:     toMapStringAny(page.Summary),
			DrillLinks:  drillLinks,
		}

		generated = append(generated, genPage)
	}

	return generated
}

// generateDrillDownPageData creates drill-down pages for detailed exploration
func (sg *SSRSGenerator) generateDrillDownPageData(cube *SemanticCube, pages []ReportPage) []DrillDownPage {
	drillDownPages := make([]DrillDownPage, 0)

	// For each major section, create a drill-down detail page
	for i := 0; i < len(pages); i++ {
		page := pages[i]
		if len(page.Entities) == 0 {
			continue
		}

		pageData := DrillDownPage{
			PageName:  fmt.Sprintf("drill_down_page_%d", i+1),
			Title:     fmt.Sprintf("Detailed: %s", page.Title),
			TableData: sg.entitiesToDetailTableData(page.Entities),
			DetailText: fmt.Sprintf(
				"Expanded view of %s with %d holdings aggregated from semantic view '%s'.",
				page.Title, len(page.Entities), cube.ViewName,
			),
			BackLink: DrillLink{
				Label:      "← Back to Main Report",
				TargetPage: fmt.Sprintf("page_%d", i+1),
				DrillType:  "prev",
			},
		}

		drillDownPages = append(drillDownPages, pageData)
	}

	return drillDownPages
}

// entitiesToTableData converts entities to table format
func (sg *SSRSGenerator) entitiesToTableData(entities []Entity, pageNum int) TableData {
	headers := []string{"Entity Name", "Type", "Value", "Allocation %", "Owner"}
	rows := make([][]string, 0, len(entities))

	for _, entity := range entities {
		row := []string{
			truncateString(entity.Name, 25),
			entity.Type,
			fmt.Sprintf("$%.2f", entity.Value),
			fmt.Sprintf("%.1f%%", entity.Allocation),
			truncateString(entity.Owner, 15),
		}
		rows = append(rows, row)
	}

	return TableData{
		Headers:  headers,
		Rows:     rows,
		RowCount: len(rows),
		PageNum:  pageNum + 1,
		PageSize: sg.pageSize,
	}
}

// entitiesToDetailTableData converts entities to expanded table format
func (sg *SSRSGenerator) entitiesToDetailTableData(entities []Entity) TableData {
	headers := []string{"Entity ID", "Name", "Value", "Allocation %", "Owner", "Type"}
	rows := make([][]string, 0, len(entities))

	for _, entity := range entities {
		row := []string{
			truncateString(entity.ID, 12),
			truncateString(entity.Name, 20),
			fmt.Sprintf("$%.2f", entity.Value),
			fmt.Sprintf("%.1f%%", entity.Allocation),
			truncateString(entity.Owner, 12),
			entity.Type,
		}
		rows = append(rows, row)
	}

	return TableData{
		Headers:  headers,
		Rows:     rows,
		RowCount: len(rows),
		PageNum:  1,
		PageSize: 50,
	}
}

// ============================================================================
// HELPERS
// ============================================================================

// truncateString shortens a string to maximum length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// MarshalGeneratedReport converts report to JSON
func (sg *SSRSGenerator) MarshalGeneratedReport(report *GeneratedReport) (json.RawMessage, error) {
	return json.Marshal(report)
}

// UnmarshalGeneratedReport converts JSON back to report
func (sg *SSRSGenerator) UnmarshalGeneratedReport(data json.RawMessage) (*GeneratedReport, error) {
	var report GeneratedReport
	err := json.Unmarshal(data, &report)
	return &report, err
}

// toMapStringAny safely converts an any type to map[string]any
func toMapStringAny(val any) map[string]any {
	if m, ok := val.(map[string]any); ok {
		return m
	}
	return make(map[string]any)
}
