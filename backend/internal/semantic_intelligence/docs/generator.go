package docs

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

type ExportFormat string

const (
	FormatMarkdown ExportFormat = "markdown"
	FormatHTML     ExportFormat = "html"
	FormatPDF      ExportFormat = "pdf"
)

type BODocumentation struct {
	BOID          uuid.UUID         `json:"bo_id"`
	BOName        string            `json:"bo_name"`
	Overview      string            `json:"overview"`
	Fields        []FieldDoc        `json:"fields"`
	Relationships []RelationshipDoc `json:"relationships"`
	Lineage       LineageDoc        `json:"lineage"`
	Usage         UsageDoc          `json:"usage"`
	Governance    GovernanceDoc     `json:"governance"`
}

type FieldDoc struct {
	Name              string   `json:"name"`
	Type              string   `json:"type"`
	Description       string   `json:"description"`
	PIIClassification string   `json:"pii_classification,omitempty"`
	Examples          []string `json:"examples,omitempty"`
}

type RelationshipDoc struct {
	Type        string `json:"type"` // hasMany, belongsTo
	TargetBO    string `json:"target_bo"`
	Description string `json:"description"`
}

type LineageDoc struct {
	SourceSystems   []string `json:"source_systems"`
	Transformations []string `json:"transformations"`
}

type UsageDoc struct {
	TopPages     []string `json:"top_pages"`
	TopAPIs      []string `json:"top_apis"`
	TopWorkflows []string `json:"top_workflows"`
}

type GovernanceDoc struct {
	Policies         []string `json:"policies"`
	SLOs             []string `json:"slos"`
	DataQualityNotes string   `json:"data_quality_notes"`
}

type DocumentationGenerator struct{}

func NewDocumentationGenerator() *DocumentationGenerator {
	return &DocumentationGenerator{}
}

func (g *DocumentationGenerator) Generate(ctx context.Context, boID uuid.UUID) (*BODocumentation, error) {
	// Mock: Generate documentation
	// Real: Query semantic graph, lineage, usage stats, governance policies

	doc := &BODocumentation{
		BOID:     boID,
		BOName:   "Position",
		Overview: "Represents a financial position held in an account. Core entity for portfolio management and reporting.",
		Fields: []FieldDoc{
			{
				Name:              "position_id",
				Type:              "uuid",
				Description:       "Unique identifier for the position",
				PIIClassification: "",
			},
			{
				Name:              "account_id",
				Type:              "uuid",
				Description:       "Reference to the owning account",
				PIIClassification: "indirect",
			},
			{
				Name:        "market_value",
				Type:        "decimal",
				Description: "Current market value in base currency",
				Examples:    []string{"125000.50", "98765.43"},
			},
		},
		Relationships: []RelationshipDoc{
			{
				Type:        "belongsTo",
				TargetBO:    "Account",
				Description: "Position belongs to an Account",
			},
			{
				Type:        "hasMany",
				TargetBO:    "Transaction",
				Description: "Position has many Transactions",
			},
		},
		Lineage: LineageDoc{
			SourceSystems:   []string{"Trading System", "Custodian Feed"},
			Transformations: []string{"Currency conversion", "Price normalization"},
		},
		Usage: UsageDoc{
			TopPages:     []string{"Positions Dashboard", "Account Overview", "Portfolio Summary"},
			TopAPIs:      []string{"positions_api", "portfolio_api"},
			TopWorkflows: []string{"Rebalancing", "Tax Loss Harvesting"},
		},
		Governance: GovernanceDoc{
			Policies:         []string{"PII: Indirect (via account_id)", "Residency: Global"},
			SLOs:             []string{"Query latency < 100ms p95"},
			DataQualityNotes: "Market value updated real-time during market hours",
		},
	}

	return doc, nil
}

func (g *DocumentationGenerator) Export(ctx context.Context, doc *BODocumentation, format ExportFormat) (string, error) {
	switch format {
	case FormatMarkdown:
		return g.exportMarkdown(doc), nil
	case FormatHTML:
		return g.exportHTML(doc), nil
	case FormatPDF:
		return "PDF export not yet implemented", nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

func (g *DocumentationGenerator) exportMarkdown(doc *BODocumentation) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", doc.BOName))
	sb.WriteString(fmt.Sprintf("## Overview\n%s\n\n", doc.Overview))

	sb.WriteString("## Fields\n")
	for _, field := range doc.Fields {
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", field.Name, field.Type, field.Description))
	}
	sb.WriteString("\n")

	sb.WriteString("## Relationships\n")
	for _, rel := range doc.Relationships {
		sb.WriteString(fmt.Sprintf("- %s %s: %s\n", rel.Type, rel.TargetBO, rel.Description))
	}
	sb.WriteString("\n")

	sb.WriteString("## Lineage\n")
	sb.WriteString(fmt.Sprintf("**Source Systems**: %s\n\n", strings.Join(doc.Lineage.SourceSystems, ", ")))

	sb.WriteString("## Usage\n")
	sb.WriteString(fmt.Sprintf("**Top Pages**: %s\n\n", strings.Join(doc.Usage.TopPages, ", ")))

	sb.WriteString("## Governance\n")
	for _, policy := range doc.Governance.Policies {
		sb.WriteString(fmt.Sprintf("- %s\n", policy))
	}

	return sb.String()
}

func (g *DocumentationGenerator) exportHTML(doc *BODocumentation) string {
	// Simple HTML wrapper around markdown
	return fmt.Sprintf("<html><body><pre>%s</pre></body></html>", g.exportMarkdown(doc))
}
