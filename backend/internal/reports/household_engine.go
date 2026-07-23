package reports

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// HouseholdReportEngine generates reports from household semantic aggregations
type HouseholdReportEngine struct {
	db *gorm.DB
}

// NewHouseholdReportEngine creates a new household report engine
func NewHouseholdReportEngine(db *gorm.DB) *HouseholdReportEngine {
	return &HouseholdReportEngine{db: db}
}

// ============================================================================
// DATA STRUCTURES
// ============================================================================

// Household represents a top-level grouping (family, trust, entity)
type Household struct {
	ID                  uuid.UUID  `gorm:"primaryKey" json:"id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	HeadOfHouseholdName string     `json:"head_of_household_name"`
	HouseholdType       string     `json:"household_type"` // 'individual', 'family', 'trust', 'entity'
	LedgerID            *uuid.UUID `json:"ledger_id"`
	Status              string     `json:"status"` // 'active', 'inactive', 'archived'
	IsPublished         bool       `json:"is_published"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// HouseholdMember represents an ALT, SMA, advisor, or beneficiary
type HouseholdMember struct {
	ID             uuid.UUID  `gorm:"primaryKey" json:"id"`
	HouseholdID    uuid.UUID  `json:"household_id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	MemberType     string     `json:"member_type"` // 'alt', 'sma', 'advisor', 'beneficiary'
	MemberID       uuid.UUID  `json:"member_id"`
	MemberName     string     `json:"member_name"`
	LedgerEntityID *uuid.UUID `json:"ledger_entity_id"`
	IsPrimary      bool       `json:"is_primary"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

// HouseholdSemanticMapping maps semantic views to households with custom grouping
type HouseholdSemanticMapping struct {
	ID               uuid.UUID       `gorm:"primaryKey" json:"id"`
	HouseholdID      uuid.UUID       `json:"household_id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	SemanticViewID   uuid.UUID       `json:"semantic_view_id"`
	ViewName         string          `json:"view_name"`
	GroupByFields    json.RawMessage `json:"group_by_fields"`   // Custom grouping (JSON tags)
	FilterConditions json.RawMessage `json:"filter_conditions"` // Aggregate filters
	AllocationWeight float64         `json:"allocation_weight"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

// HouseholdReport represents a generated report with semantic cube
type HouseholdReport struct {
	ID               uuid.UUID       `gorm:"primaryKey" json:"id"`
	HouseholdID      uuid.UUID       `json:"household_id"`
	TenantID         uuid.UUID       `json:"tenant_id"`
	ReportName       string          `json:"report_name"`
	ReportType       string          `json:"report_type"`   // 'summary', 'detailed', 'performance', 'allocation'
	ReportConfig     json.RawMessage `json:"report_config"` // ParameterBuilder schema
	SemanticCubeID   *uuid.UUID      `json:"semantic_cube_id"`
	SemanticCubeData json.RawMessage `json:"semantic_cube_data"` // Cached cube
	PDFData          []byte          `json:"pdf_data"`
	PDFGeneratedAt   *time.Time      `json:"pdf_generated_at"`
	PDFFileName      string          `json:"pdf_file_name"`
	DrillPaths       json.RawMessage `json:"drill_paths"` // Drill-down navigation
	PageCount        int             `json:"page_count"`
	SectionCount     int             `json:"section_count"`
	Status           string          `json:"status"` // 'draft', 'generated', 'error'
	GenerationError  string          `json:"generation_error"`
	CreatedAt        time.Time       `json:"created_at"`
	GeneratedAt      *time.Time      `json:"generated_at"`
	ExpiresAt        *time.Time      `json:"expires_at"`
}

// SemanticCube represents aggregated data for a household (AI-generated)
type SemanticCube struct {
	ID          uuid.UUID           `json:"id"`
	HouseholdID uuid.UUID           `json:"household_id"`
	ViewName    string              `json:"view_name"`
	Dimensions  map[string][]string `json:"dimensions"` // Grouping axes
	Metrics     map[string]float64  `json:"metrics"`    // Aggregated values
	Entities    []Entity            `json:"entities"`   // Leaf data (holdings, positions)
	Summary     map[string]any      `json:"summary"`    // Totals, averages, counts
	GeneratedAt time.Time           `json:"generated_at"`
}

// Entity represents a single item in the semantic cube (holding, position, etc.)
type Entity struct {
	ID            string          `json:"id"`
	Type          string          `json:"type"` // 'holding', 'position', 'trade', etc.
	Name          string          `json:"name"`
	Value         float64         `json:"value"`
	Allocation    float64         `json:"allocation"`      // % of household
	Owner         string          `json:"owner"`           // ALT/SMA that holds it
	Attributes    json.RawMessage `json:"attributes"`      // Custom attributes
	DrillDownPath string          `json:"drill_down_path"` // For multi-page nav
}

// ReportPage represents a single PDF page section
type ReportPage struct {
	PageNum      int      `json:"page_num"`
	Title        string   `json:"title"`
	SectionType  string   `json:"section_type"` // 'summary', 'holdings', 'performance', 'allocation'
	Entities     []Entity `json:"entities"`
	Summary      any      `json:"summary"`
	DrillTargets []string `json:"drill_targets"` // Links to other pages
}

// ============================================================================
// HOUSEHOLD DATA RETRIEVAL
// ============================================================================

// GetHouseholdData retrieves all household data (members, mappings, active views)
func (hre *HouseholdReportEngine) GetHouseholdData(ctx context.Context, householdID, tenantID uuid.UUID) (*Household, []HouseholdMember, []HouseholdSemanticMapping, error) {
	var household Household
	var members []HouseholdMember
	var mappings []HouseholdSemanticMapping

	// Get household
	if err := hre.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", householdID, tenantID).
		First(&household).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("household not found: %w", err)
	}

	// Get active members
	if err := hre.db.WithContext(ctx).
		Where("household_id = ? AND tenant_id = ? AND is_active = true", householdID, tenantID).
		Find(&members).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get members: %w", err)
	}

	// Get active semantic mappings
	if err := hre.db.WithContext(ctx).
		Where("household_id = ? AND tenant_id = ? AND is_active = true", householdID, tenantID).
		Find(&mappings).Error; err != nil {
		return nil, nil, nil, fmt.Errorf("failed to get semantic mappings: %w", err)
	}

	return &household, members, mappings, nil
}

// ============================================================================
// SEMANTIC CUBE GENERATION
// ============================================================================

// GenerateSemanticCube creates an AI semantic cube from household holdings
// This aggregates data from semantic views and structures it for AI analysis
func (hre *HouseholdReportEngine) GenerateSemanticCube(ctx context.Context, householdID, tenantID uuid.UUID, mapping *HouseholdSemanticMapping) (*SemanticCube, error) {
	cube := &SemanticCube{
		ID:          uuid.New(),
		HouseholdID: householdID,
		ViewName:    mapping.ViewName,
		Dimensions:  make(map[string][]string),
		Metrics:     make(map[string]float64),
		Entities:    []Entity{},
		Summary:     make(map[string]any),
		GeneratedAt: time.Now(),
	}

	// Parse filter conditions
	var filters map[string]any
	if err := json.Unmarshal(mapping.FilterConditions, &filters); err != nil {
		return nil, fmt.Errorf("failed to parse filter conditions: %w", err)
	}

	// Parse grouping fields
	var groupBy map[string][]string
	if err := json.Unmarshal(mapping.GroupByFields, &groupBy); err != nil {
		return nil, fmt.Errorf("failed to parse group by fields: %w", err)
	}

	// Query semantic view with filters and aggregation
	// This would query Hasura GraphQL to get the semantic view data
	// For now, we structure the framework:

	// 1. Build dimension axes from groupBy
	for dimension, values := range groupBy {
		cube.Dimensions[dimension] = values
	}

	// 2. Query holdings/positions matching filters
	// TODO: Query semantic_views via Hasura with GraphQL
	// Query pattern:
	//   query GetSemanticViewData($filter: ...) {
	//     semantic_view_entities(where: $filter) {
	//       id, name, value, owner, ...
	//     }
	//   }

	// 3. Aggregate into semantic cube structure
	// For each entity, map to dimensions and sum metrics
	totalValue := 0.0
	for _, entity := range cube.Entities {
		totalValue += entity.Value
	}

	// 4. Calculate allocation percentages
	for i := range cube.Entities {
		if totalValue > 0 {
			cube.Entities[i].Allocation = (cube.Entities[i].Value / totalValue) * 100
		}
	}

	// 5. Create summary (totals, counts, averages)
	cube.Summary = map[string]any{
		"total_value":    totalValue,
		"entity_count":   len(cube.Entities),
		"dimension_keys": len(cube.Dimensions),
		"primary_member": "", // Will be populated from household_members
		"generated_at":   cube.GeneratedAt,
	}

	return cube, nil
}

// ============================================================================
// REPORT BUILDING
// ============================================================================

// BuildReportFromCube structures semantic cube data into report sections (pages)
func (hre *HouseholdReportEngine) BuildReportFromCube(ctx context.Context, cube *SemanticCube, reportType string) ([]ReportPage, error) {
	pages := []ReportPage{}
	pageNum := 1

	switch reportType {
	case "summary":
		pages = append(pages, hre.buildSummaryPage(cube, pageNum))
		pageNum++

	case "detailed":
		pages = append(pages, hre.buildSummaryPage(cube, pageNum))
		pageNum++
		pages = append(pages, hre.buildHoldingsPages(cube, pageNum)...)
		pageNum += len(pages)

	case "performance":
		pages = append(pages, hre.buildPerformancePage(cube, pageNum))
		pageNum++

	case "allocation":
		pages = append(pages, hre.buildAllocationPage(cube, pageNum))
		pageNum++

	default:
		return nil, fmt.Errorf("unknown report type: %s", reportType)
	}

	return pages, nil
}

// buildSummaryPage creates the executive summary page
func (hre *HouseholdReportEngine) buildSummaryPage(cube *SemanticCube, pageNum int) ReportPage {
	return ReportPage{
		PageNum:      pageNum,
		Title:        "Executive Summary",
		SectionType:  "summary",
		Entities:     cube.Entities[:min(len(cube.Entities), 10)], // Top 10 entities
		Summary:      cube.Summary,
		DrillTargets: []string{}, // Will add drill paths for multi-page nav
	}
}

// buildHoldingsPages creates paginated holdings pages (20 per page)
func (hre *HouseholdReportEngine) buildHoldingsPages(cube *SemanticCube, startPageNum int) []ReportPage {
	const pageSize = 20
	pages := []ReportPage{}

	for i := 0; i < len(cube.Entities); i += pageSize {
		end := i + pageSize
		if end > len(cube.Entities) {
			end = len(cube.Entities)
		}

		page := ReportPage{
			PageNum:     startPageNum + len(pages),
			Title:       fmt.Sprintf("Holdings (Page %d of %d)", len(pages)+1, (len(cube.Entities)+pageSize-1)/pageSize),
			SectionType: "holdings",
			Entities:    cube.Entities[i:end],
		}

		// Build drill paths for pagination
		if i > 0 {
			page.DrillTargets = append(page.DrillTargets, fmt.Sprintf("page_%d", startPageNum+len(pages)-1))
		}
		if end < len(cube.Entities) {
			page.DrillTargets = append(page.DrillTargets, fmt.Sprintf("page_%d", startPageNum+len(pages)+1))
		}

		pages = append(pages, page)
	}

	return pages
}

// buildPerformancePage creates a performance analysis page
func (hre *HouseholdReportEngine) buildPerformancePage(cube *SemanticCube, pageNum int) ReportPage {
	return ReportPage{
		PageNum:     pageNum,
		Title:       "Performance Analysis",
		SectionType: "performance",
		Entities:    cube.Entities,
		Summary: map[string]any{
			"total_value":        cube.Summary["total_value"],
			"ytd_return":         0.0, // Would be calculated from market data
			"performance_period": "Year to Date",
		},
	}
}

// buildAllocationPage creates an allocation breakdown page
func (hre *HouseholdReportEngine) buildAllocationPage(cube *SemanticCube, pageNum int) ReportPage {
	return ReportPage{
		PageNum:     pageNum,
		Title:       "Allocation Breakdown",
		SectionType: "allocation",
		Entities:    cube.Entities,
		Summary: map[string]any{
			"dimensions":  cube.Dimensions,
			"total_value": cube.Summary["total_value"],
		},
	}
}

// ============================================================================
// REPORT PERSISTENCE
// ============================================================================

// SaveReport persists a generated report to the database
func (hre *HouseholdReportEngine) SaveReport(ctx context.Context, householdID, tenantID uuid.UUID, reportName, reportType string, config json.RawMessage, cube *SemanticCube, pages []ReportPage) (*HouseholdReport, error) {
	cubeData, err := json.Marshal(cube)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal semantic cube: %w", err)
	}

	// Build drill paths from pages
	drillPaths := map[string][]string{
		"pages": make([]string, len(pages)),
	}
	for i, page := range pages {
		drillPaths["pages"][i] = fmt.Sprintf("page_%d", page.PageNum)
	}
	drillPathsJSON, _ := json.Marshal(drillPaths)

	report := &HouseholdReport{
		ID:               uuid.New(),
		HouseholdID:      householdID,
		TenantID:         tenantID,
		ReportName:       reportName,
		ReportType:       reportType,
		ReportConfig:     config,
		SemanticCubeID:   &cube.ID,
		SemanticCubeData: cubeData,
		DrillPaths:       drillPathsJSON,
		PageCount:        len(pages),
		SectionCount:     len(pages),
		Status:           "generated",
		GeneratedAt:      timePtr(time.Now()),
		ExpiresAt:        timePtr(time.Now().AddDate(0, 0, 90)), // 90-day retention
	}

	if err := hre.db.WithContext(ctx).Create(report).Error; err != nil {
		return nil, fmt.Errorf("failed to save report: %w", err)
	}

	return report, nil
}

// GetReport retrieves a report by ID
func (hre *HouseholdReportEngine) GetReport(ctx context.Context, reportID, tenantID uuid.UUID) (*HouseholdReport, error) {
	var report HouseholdReport
	if err := hre.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ?", reportID, tenantID).
		First(&report).Error; err != nil {
		return nil, fmt.Errorf("report not found: %w", err)
	}
	return &report, nil
}

// ListHouseholdReports lists all reports for a household
func (hre *HouseholdReportEngine) ListHouseholdReports(ctx context.Context, householdID, tenantID uuid.UUID) ([]HouseholdReport, error) {
	var reports []HouseholdReport
	if err := hre.db.WithContext(ctx).
		Where("household_id = ? AND tenant_id = ?", householdID, tenantID).
		Order("created_at DESC").
		Find(&reports).Error; err != nil {
		return nil, fmt.Errorf("failed to list reports: %w", err)
	}
	return reports, nil
}

// ============================================================================
// HELPERS
// ============================================================================

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func timePtr(t time.Time) *time.Time {
	return &t
}
