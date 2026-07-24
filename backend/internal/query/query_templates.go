package query

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// QueryTemplate represents a pre-built query template
type QueryTemplate struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Template    string              `json:"template"`
	Parameters  []TemplateParameter `json:"parameters"`
	Examples    []string            `json:"examples"`
	Tags        []string            `json:"tags"`
	UseCount    int                 `json:"use_count"`
	LastUsed    time.Time           `json:"last_used"`
	CreatedAt   time.Time           `json:"created_at"`
	Confidence  float64             `json:"confidence"`
}

// TemplateParameter represents a parameter in a query template
type TemplateParameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "metric", "dimension", "time_range", "filter"
	Required    bool     `json:"required"`
	Default     string   `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"`
	Description string   `json:"description"`
}

// QueryTemplateManager manages query templates
type QueryTemplateManager struct {
	templates  map[string]*QueryTemplate
	categories map[string][]*QueryTemplate
}

// NewQueryTemplateManager creates a new template manager
func NewQueryTemplateManager() *QueryTemplateManager {
	manager := &QueryTemplateManager{
		templates:  make(map[string]*QueryTemplate),
		categories: make(map[string][]*QueryTemplate),
	}
	manager.initializeDefaultTemplates()
	return manager
}

// initializeDefaultTemplates sets up common business query templates
func (qtm *QueryTemplateManager) initializeDefaultTemplates() {
	templates := []*QueryTemplate{
		{
			ID:          "sales_performance",
			Name:        "Sales Performance Analysis",
			Description: "Analyze sales performance across different dimensions",
			Category:    "Sales",
			Template:    "Show me {metric} by {dimension} {time_range}",
			Parameters: []TemplateParameter{
				{
					Name:        "metric",
					Type:        "metric",
					Required:    true,
					Options:     []string{"total_sales", "average_order_value", "order_count"},
					Description: "Sales metric to analyze",
				},
				{
					Name:        "dimension",
					Type:        "dimension",
					Required:    true,
					Options:     []string{"region", "product_category", "sales_rep"},
					Description: "Dimension to group by",
				},
				{
					Name:        "time_range",
					Type:        "time_range",
					Required:    false,
					Default:     "last month",
					Description: "Time period for analysis",
				},
			},
			Examples: []string{
				"Show me total sales by region last quarter",
				"Show me average order value by product category this month",
			},
			Tags:       []string{"sales", "performance", "analysis"},
			UseCount:   0,
			CreatedAt:  time.Now(),
			Confidence: 0.9,
		},
		{
			ID:          "customer_analysis",
			Name:        "Customer Analysis",
			Description: "Analyze customer behavior and segmentation",
			Category:    "Customer",
			Template:    "Show me {metric} for {segment} customers {time_range}",
			Parameters: []TemplateParameter{
				{
					Name:        "metric",
					Type:        "metric",
					Required:    true,
					Options:     []string{"customer_count", "average_lifetime_value", "churn_rate"},
					Description: "Customer metric to analyze",
				},
				{
					Name:        "segment",
					Type:        "dimension",
					Required:    false,
					Options:     []string{"high_value", "new", "returning", "at_risk"},
					Description: "Customer segment to focus on",
				},
				{
					Name:        "time_range",
					Type:        "time_range",
					Required:    false,
					Default:     "last 30 days",
					Description: "Time period for analysis",
				},
			},
			Examples: []string{
				"Show me customer count for high value customers last quarter",
				"Show me average lifetime value for new customers this month",
			},
			Tags:       []string{"customer", "segmentation", "behavior"},
			UseCount:   0,
			CreatedAt:  time.Now(),
			Confidence: 0.9,
		},
		{
			ID:          "inventory_status",
			Name:        "Inventory Status Report",
			Description: "Check inventory levels and stock status",
			Category:    "Inventory",
			Template:    "Show me {metric} by {dimension} where {condition}",
			Parameters: []TemplateParameter{
				{
					Name:        "metric",
					Type:        "metric",
					Required:    true,
					Options:     []string{"stock_level", "stock_value", "turnover_rate"},
					Description: "Inventory metric to analyze",
				},
				{
					Name:        "dimension",
					Type:        "dimension",
					Required:    true,
					Options:     []string{"product", "warehouse", "category"},
					Description: "Dimension to group by",
				},
				{
					Name:        "condition",
					Type:        "filter",
					Required:    false,
					Default:     "stock_level < reorder_point",
					Description: "Filter condition for low stock items",
				},
			},
			Examples: []string{
				"Show me stock level by product where stock level < reorder point",
				"Show me stock value by warehouse",
			},
			Tags:       []string{"inventory", "stock", "warehouse"},
			UseCount:   0,
			CreatedAt:  time.Now(),
			Confidence: 0.85,
		},
		{
			ID:          "financial_kpi",
			Name:        "Financial KPI Dashboard",
			Description: "Monitor key financial performance indicators",
			Category:    "Finance",
			Template:    "Compare {metric1} and {metric2} by {dimension} {time_range}",
			Parameters: []TemplateParameter{
				{
					Name:        "metric1",
					Type:        "metric",
					Required:    true,
					Options:     []string{"revenue", "profit", "margin_percentage"},
					Description: "First financial metric",
				},
				{
					Name:        "metric2",
					Type:        "metric",
					Required:    true,
					Options:     []string{"budget", "forecast", "previous_period"},
					Description: "Second financial metric for comparison",
				},
				{
					Name:        "dimension",
					Type:        "dimension",
					Required:    true,
					Options:     []string{"department", "business_unit", "month"},
					Description: "Dimension for grouping",
				},
				{
					Name:        "time_range",
					Type:        "time_range",
					Required:    false,
					Default:     "this year",
					Description: "Time period for analysis",
				},
			},
			Examples: []string{
				"Compare revenue and budget by department this quarter",
				"Compare profit and forecast by business unit this year",
			},
			Tags:       []string{"finance", "kpi", "budget", "forecast"},
			UseCount:   0,
			CreatedAt:  time.Now(),
			Confidence: 0.9,
		},
		{
			ID:          "trend_analysis",
			Name:        "Trend Analysis",
			Description: "Analyze trends over time periods",
			Category:    "Analytics",
			Template:    "Show trend of {metric} over {time_dimension} {time_range}",
			Parameters: []TemplateParameter{
				{
					Name:        "metric",
					Type:        "metric",
					Required:    true,
					Options:     []string{"sales", "users", "conversion_rate", "revenue"},
					Description: "Metric to analyze trends for",
				},
				{
					Name:        "time_dimension",
					Type:        "dimension",
					Required:    true,
					Options:     []string{"day", "week", "month", "quarter"},
					Description: "Time granularity for trend analysis",
				},
				{
					Name:        "time_range",
					Type:        "time_range",
					Required:    false,
					Default:     "last 6 months",
					Description: "Time period for trend analysis",
				},
			},
			Examples: []string{
				"Show trend of sales over month last 6 months",
				"Show trend of users over week last quarter",
			},
			Tags:       []string{"trend", "time", "analysis"},
			UseCount:   0,
			CreatedAt:  time.Now(),
			Confidence: 0.85,
		},
	}

	for _, template := range templates {
		qtm.AddTemplate(template)
	}
}

// AddTemplate adds a new query template
func (qtm *QueryTemplateManager) AddTemplate(template *QueryTemplate) {
	qtm.templates[template.ID] = template
	if qtm.categories[template.Category] == nil {
		qtm.categories[template.Category] = []*QueryTemplate{}
	}
	qtm.categories[template.Category] = append(qtm.categories[template.Category], template)
}

// GetTemplate retrieves a template by ID
func (qtm *QueryTemplateManager) GetTemplate(id string) (*QueryTemplate, error) {
	template, exists := qtm.templates[id]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	return template, nil
}

// FindMatchingTemplates finds templates that match a natural language query
func (qtm *QueryTemplateManager) FindMatchingTemplates(query string) []*QueryTemplate {
	var matches []*QueryTemplate
	queryLower := strings.ToLower(query)

	for _, template := range qtm.templates {
		// Check if query matches template keywords or examples
		if qtm.matchesTemplate(queryLower, template) {
			matches = append(matches, template)
		}
	}

	// Sort by confidence and usage
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Confidence != matches[j].Confidence {
			return matches[i].Confidence > matches[j].Confidence
		}
		return matches[i].UseCount > matches[j].UseCount
	})

	return matches
}

// matchesTemplate checks if a query matches a template
func (qtm *QueryTemplateManager) matchesTemplate(queryLower string, template *QueryTemplate) bool {
	// Check template name and description
	if strings.Contains(strings.ToLower(template.Name), queryLower) ||
		strings.Contains(strings.ToLower(template.Description), queryLower) {
		return true
	}

	// Check tags
	for _, tag := range template.Tags {
		if strings.Contains(queryLower, strings.ToLower(tag)) {
			return true
		}
	}

	// Check examples
	for _, example := range template.Examples {
		if strings.Contains(strings.ToLower(example), queryLower) {
			return true
		}
	}

	// Check category
	if strings.Contains(queryLower, strings.ToLower(template.Category)) {
		return true
	}

	return false
}

// GetTemplatesByCategory returns templates for a specific category
func (qtm *QueryTemplateManager) GetTemplatesByCategory(category string) []*QueryTemplate {
	return qtm.categories[category]
}

// GetAllCategories returns all available categories
func (qtm *QueryTemplateManager) GetAllCategories() []string {
	var categories []string
	for category := range qtm.categories {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

// RecordTemplateUsage records that a template was used
func (qtm *QueryTemplateManager) RecordTemplateUsage(templateID string) {
	if template, exists := qtm.templates[templateID]; exists {
		template.UseCount++
		template.LastUsed = time.Now()
	}
}

// GenerateQueryFromTemplate generates a query from a template with parameters
func (qtm *QueryTemplateManager) GenerateQueryFromTemplate(templateID string, parameters map[string]string) (string, error) {
	template, err := qtm.GetTemplate(templateID)
	if err != nil {
		return "", err
	}

	query := template.Template

	// Replace parameters in template
	for paramName, paramValue := range parameters {
		placeholder := fmt.Sprintf("{%s}", paramName)
		query = strings.ReplaceAll(query, placeholder, paramValue)
	}

	// Record usage
	qtm.RecordTemplateUsage(templateID)

	return query, nil
}

// GetTemplateSuggestions returns template suggestions based on query analysis
func (qtm *QueryTemplateManager) GetTemplateSuggestions(query string, intent *ParsedIntent) []*QueryTemplate {
	matches := qtm.FindMatchingTemplates(query)

	// If we have intent information, prioritize templates that match the intent
	if intent != nil && len(matches) > 3 {
		// Prioritize templates that have parameters matching the intent
		var prioritized []*QueryTemplate
		for _, template := range matches {
			if qtm.templateMatchesIntent(template, intent) {
				prioritized = append(prioritized, template)
			}
		}

		// Add remaining templates
		for _, template := range matches {
			found := false
			for _, p := range prioritized {
				if p.ID == template.ID {
					found = true
					break
				}
			}
			if !found {
				prioritized = append(prioritized, template)
			}
		}

		return prioritized[:min(5, len(prioritized))]
	}

	return matches[:min(5, len(matches))]
}

// templateMatchesIntent checks if a template matches the parsed intent
func (qtm *QueryTemplateManager) templateMatchesIntent(template *QueryTemplate, intent *ParsedIntent) bool {
	// Check if template has parameters that match intent metrics/dimensions
	for _, param := range template.Parameters {
		switch param.Type {
		case "metric":
			if len(intent.Metrics) > 0 {
				return true
			}
		case "dimension":
			if len(intent.Dimensions) > 0 {
				return true
			}
		case "time_range":
			if intent.TimeRange != nil {
				return true
			}
		}
	}
	return false
}

// GetPopularTemplates returns the most popular templates
func (qtm *QueryTemplateManager) GetPopularTemplates(limit int) []*QueryTemplate {
	var templates []*QueryTemplate
	for _, template := range qtm.templates {
		templates = append(templates, template)
	}

	// Sort by usage count
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].UseCount > templates[j].UseCount
	})

	if limit > 0 && limit < len(templates) {
		return templates[:limit]
	}
	return templates
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
