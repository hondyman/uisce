package roadmap

import (
	"context"

	"github.com/google/uuid"
)

type RoadmapItemType string

const (
	ItemTypeFeature          RoadmapItemType = "feature"
	ItemTypeComponent        RoadmapItemType = "component"
	ItemTypeWorkflow         RoadmapItemType = "workflow"
	ItemTypePreAgg           RoadmapItemType = "preagg"
	ItemTypeSLO              RoadmapItemType = "slo"
	ItemTypeComplianceBundle RoadmapItemType = "compliance_bundle"
	ItemTypeTheme            RoadmapItemType = "theme"
	ItemTypeGovernanceRule   RoadmapItemType = "governance_rule"
)

type RoadmapItem struct {
	ID               uuid.UUID       `json:"id"`
	Type             RoadmapItemType `json:"type"`
	Title            string          `json:"title"`
	Description      string          `json:"description"`
	Rationale        string          `json:"rationale"`
	Impact           int             `json:"impact"`        // 1-10
	Effort           int             `json:"effort"`        // 1-10
	Risk             int             `json:"risk"`          // 1-10
	TenantDemand     int             `json:"tenant_demand"` // Number of tenants requesting
	Priority         int             `json:"priority"`      // Calculated score
	Dependencies     []string        `json:"dependencies"`
	SLOImpact        string          `json:"slo_impact"`
	ComplianceImpact string          `json:"compliance_impact"`
}

type RoadmapGenerator struct{}

func NewRoadmapGenerator() *RoadmapGenerator {
	return &RoadmapGenerator{}
}

func (rg *RoadmapGenerator) GenerateRoadmap(ctx context.Context) ([]RoadmapItem, error) {
	// Mock: Generate roadmap items
	// Real: Analyze feature usage, page usage, workflow usage, API usage, tenant behavior, SLO pressure, drift patterns, compliance gaps

	items := []RoadmapItem{
		{
			ID:               uuid.New(),
			Type:             ItemTypeFeature,
			Title:            "Core Risk App",
			Description:      "Tenants frequently build risk dashboards; add a core Risk App template",
			Rationale:        "14 tenants have created custom risk dashboards with 80% overlap. Core app will reduce duplication and improve consistency.",
			Impact:           9,
			Effort:           7,
			Risk:             3,
			TenantDemand:     14,
			Priority:         85,
			Dependencies:     []string{},
			SLOImpact:        "Positive - reduces custom page complexity",
			ComplianceImpact: "Neutral",
		},
		{
			ID:               uuid.New(),
			Type:             ItemTypeComponent,
			Title:            "KPI Group Component",
			Description:      "Many tenants build KPI clusters; add a KPI Group component to component library",
			Rationale:        "22 tenants have created custom KPI cluster layouts. Standardized component will improve UX consistency.",
			Impact:           7,
			Effort:           4,
			Risk:             2,
			TenantDemand:     22,
			Priority:         82,
			Dependencies:     []string{},
			SLOImpact:        "Positive - optimized rendering",
			ComplianceImpact: "Neutral",
		},
		{
			ID:               uuid.New(),
			Type:             ItemTypePreAgg,
			Title:            "Global positions_by_region Pre-Agg",
			Description:      "12 tenants repeatedly query positions by region without pre-agg",
			Rationale:        "Global pre-agg will reduce query time by 70% and cost by $1,200/month.",
			Impact:           8,
			Effort:           3,
			Risk:             2,
			TenantDemand:     12,
			Priority:         88,
			Dependencies:     []string{},
			SLOImpact:        "Positive - 70% query time reduction",
			ComplianceImpact: "Neutral",
		},
		{
			ID:               uuid.New(),
			Type:             ItemTypeComplianceBundle,
			Title:            "MiFID II Compliance Bundle for EU Wealth Tenants",
			Description:      "EU wealth tenants need a MiFID II compliance bundle",
			Rationale:        "8 EU wealth tenants require MiFID II compliance. Standardized bundle will reduce compliance risk and implementation time.",
			Impact:           9,
			Effort:           6,
			Risk:             4,
			TenantDemand:     8,
			Priority:         80,
			Dependencies:     []string{"Workflow security enhancements"},
			SLOImpact:        "Neutral",
			ComplianceImpact: "Positive - reduces compliance risk",
		},
		{
			ID:               uuid.New(),
			Type:             ItemTypeGovernanceRule,
			Title:            "Stricter PII Masking Defaults",
			Description:      "PII exposure patterns suggest stricter masking defaults needed",
			Rationale:        "14 tenants have PII exposure patterns. Stricter defaults will reduce compliance risk.",
			Impact:           8,
			Effort:           5,
			Risk:             3,
			TenantDemand:     14,
			Priority:         78,
			Dependencies:     []string{},
			SLOImpact:        "Neutral",
			ComplianceImpact: "Positive - reduces PII exposure risk",
		},
	}

	return items, nil
}

func (rg *RoadmapGenerator) PrioritizeRoadmap(ctx context.Context, items []RoadmapItem) ([]RoadmapItem, error) {
	// Mock: Sort by priority
	// Real: Calculate priority score based on impact, effort, risk, tenant demand, SLO impact, compliance impact

	// Priority = (Impact * 2 + TenantDemand) - (Effort + Risk)
	// Already calculated in mock data

	// Sort by priority (descending)
	// In real implementation, use sort.Slice

	return items, nil
}
