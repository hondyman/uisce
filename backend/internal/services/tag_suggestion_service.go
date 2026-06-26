package services

import (
	"context"
	"database/sql"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// ============================================================================
// SEMANTIC TERM TAG SUGGESTION SERVICE
// Provides intelligent tag recommendations for semantic terms
// ============================================================================

// TagSuggestionService handles tag inference and recommendations for semantic terms
type TagSuggestionService struct {
	db TagSuggestionDB // Assume Database interface has query methods
}

// NewTagSuggestionService creates a new tag suggestion service
func NewTagSuggestionService(db TagSuggestionDB) *TagSuggestionService {
	return &TagSuggestionService{db: db}
}

// SuggestTagsForSemanticTerm generates tag suggestions based on semantic term characteristics
func (s *TagSuggestionService) SuggestTagsForSemanticTerm(ctx context.Context, req *models.TagSuggestionRequest) (*models.TagSuggestionResponse, error) {
	response := &models.TagSuggestionResponse{
		Suggestions: []models.TagSuggestion{},
		Reasons:     make(map[string]string),
	}

	// Infer tags from multiple sources
	suggestedByDataType := s.inferTagsFromDataType(req.DataType)
	suggestedByName := s.inferTagsFromName(req.NodeName, req.DisplayName)
	suggestedByDescription := s.inferTagsFromDescription(req.Description)
	suggestedByDomain := s.inferTagsFromDomain(req.Domain)
	suggestedByExpression := s.inferTagsFromExpression(req.Expression)
	suggestedByMapping := s.inferTagsFromPhysicalMapping(req.PhysicalMapping)

	// Combine suggestions (avoiding duplicates)
	allSuggestions := make(map[string]*models.TagSuggestion)

	// Merge all suggestions with confidence weighting
	s.mergeSuggestions(allSuggestions, suggestedByDataType, 0.9, "inferred_from_datatype", response)
	s.mergeSuggestions(allSuggestions, suggestedByName, 0.85, "inferred_from_name", response)
	s.mergeSuggestions(allSuggestions, suggestedByDescription, 0.8, "inferred_from_description", response)
	s.mergeSuggestions(allSuggestions, suggestedByDomain, 0.95, "inferred_from_domain", response)
	s.mergeSuggestions(allSuggestions, suggestedByExpression, 0.75, "inferred_from_expression", response)
	s.mergeSuggestions(allSuggestions, suggestedByMapping, 0.7, "inferred_from_mapping", response)

	// Convert map to sorted slice by confidence
	for _, suggestion := range allSuggestions {
		response.Suggestions = append(response.Suggestions, *suggestion)
	}

	// Sort by confidence descending
	sortSuggestionsByConfidence(response.Suggestions)

	// Filter out already assigned tags
	response.Suggestions = s.filterExistingTags(response.Suggestions, req.ExistingTags)

	return response, nil
}

// inferTagsFromDataType suggests tags based on the data type
func (s *TagSuggestionService) inferTagsFromDataType(dataType models.SemanticDataType) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}

	switch dataType {
	case models.DataTypeNumber:
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "numeric",
			TagLabel:         "Numeric",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.95,
			ColorCode:        "#FF6F00",
			IconName:         "sigma",
		})
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "measure",
			TagLabel:         "Measure",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.85,
			ColorCode:        "#388E3C",
			IconName:         "calculator",
		})

	case models.DataTypeDate, models.DataTypeDateTime:
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "date",
			TagLabel:         "Date",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.95,
			ColorCode:        "#7B1FA2",
			IconName:         "calendar",
		})
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "dimension",
			TagLabel:         "Dimension",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.8,
			ColorCode:        "#1976D2",
			IconName:         "layers",
		})

	case models.DataTypeString:
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "text",
			TagLabel:         "Text",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.95,
			ColorCode:        "#00838F",
			IconName:         "text",
		})
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "dimension",
			TagLabel:         "Dimension",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.85,
			ColorCode:        "#1976D2",
			IconName:         "layers",
		})

	case models.DataTypeBoolean:
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "boolean",
			TagLabel:         "Boolean",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.95,
			ColorCode:        "#00BCD4",
			IconName:         "toggle-on",
		})
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "categorical",
			TagLabel:         "Categorical",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_datatype",
			ConfidenceScore:  0.8,
			ColorCode:        "#5E35B1",
			IconName:         "list",
		})
	}

	return suggestions
}

// inferTagsFromName suggests tags based on the field name patterns
func (s *TagSuggestionService) inferTagsFromName(nodeName, displayName string) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}
	lowerName := strings.ToLower(nodeName + " " + displayName)

	// Business area keywords
	businessAreaTags := map[string]models.TagSuggestion{
		"sales": {
			TagKey:      "sales",
			TagLabel:    "Sales",
			TagCategory: "business_area",
			ColorCode:   "#2E7D32",
			IconName:    "trending-up",
		},
		"revenue": {
			TagKey:      "sales",
			TagLabel:    "Sales",
			TagCategory: "business_area",
			ColorCode:   "#2E7D32",
			IconName:    "trending-up",
		},
		"finance": {
			TagKey:      "finance",
			TagLabel:    "Finance",
			TagCategory: "business_area",
			ColorCode:   "#1565C0",
			IconName:    "dollar-sign",
		},
		"cost": {
			TagKey:      "finance",
			TagLabel:    "Finance",
			TagCategory: "business_area",
			ColorCode:   "#1565C0",
			IconName:    "dollar-sign",
		},
		"profit": {
			TagKey:      "finance",
			TagLabel:    "Finance",
			TagCategory: "business_area",
			ColorCode:   "#1565C0",
			IconName:    "dollar-sign",
		},
		"marketing": {
			TagKey:      "marketing",
			TagLabel:    "Marketing",
			TagCategory: "business_area",
			ColorCode:   "#C2185B",
			IconName:    "megaphone",
		},
		"campaign": {
			TagKey:      "marketing",
			TagLabel:    "Marketing",
			TagCategory: "business_area",
			ColorCode:   "#C2185B",
			IconName:    "megaphone",
		},
		"customer": {
			TagKey:      "customer",
			TagLabel:    "Customer",
			TagCategory: "business_area",
			ColorCode:   "#C62828",
			IconName:    "users",
		},
		"employee": {
			TagKey:      "hr",
			TagLabel:    "Human Resources",
			TagCategory: "business_area",
			ColorCode:   "#6A1B9A",
			IconName:    "users",
		},
		"product": {
			TagKey:      "product",
			TagLabel:    "Product",
			TagCategory: "business_area",
			ColorCode:   "#0097A7",
			IconName:    "package",
		},
	}

	// Check for keyword matches
	for keyword, tag := range businessAreaTags {
		if strings.Contains(lowerName, keyword) {
			tag.SuggestionReason = "inferred_from_name"
			tag.ConfidenceScore = 0.85
			suggestions = append(suggestions, tag)
			break // Take first match
		}
	}

	// Domain pattern suggestions
	if strings.Contains(lowerName, "amount") || strings.Contains(lowerName, "total") || strings.Contains(lowerName, "sum") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "measure",
			TagLabel:         "Measure",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_name",
			ConfidenceScore:  0.8,
			ColorCode:        "#388E3C",
			IconName:         "calculator",
		})
	}

	if strings.Contains(lowerName, "rate") || strings.Contains(lowerName, "percentage") || strings.Contains(lowerName, "percent") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "kpi",
			TagLabel:         "KPI",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_name",
			ConfidenceScore:  0.75,
			ColorCode:        "#F57F17",
			IconName:         "target",
		})
	}

	return suggestions
}

// inferTagsFromDescription suggests tags based on field description
func (s *TagSuggestionService) inferTagsFromDescription(description string) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}
	lowerDesc := strings.ToLower(description)

	// Domain keywords in description
	if strings.Contains(lowerDesc, "kpi") || strings.Contains(lowerDesc, "key performance") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "kpi",
			TagLabel:         "KPI",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_description",
			ConfidenceScore:  0.9,
			ColorCode:        "#F57F17",
			IconName:         "target",
		})
	}

	// Sensitivity keywords
	if strings.Contains(lowerDesc, "sensitive") || strings.Contains(lowerDesc, "confidential") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "confidential",
			TagLabel:         "Confidential",
			TagCategory:      "sensitivity",
			SuggestionReason: "inferred_from_description",
			ConfidenceScore:  0.85,
			ColorCode:        "#F57F17",
			IconName:         "lock",
		})
	}

	if strings.Contains(lowerDesc, "pii") || strings.Contains(lowerDesc, "personally identifiable") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "pii",
			TagLabel:         "PII",
			TagCategory:      "sensitivity",
			SuggestionReason: "inferred_from_description",
			ConfidenceScore:  0.95,
			ColorCode:        "#D32F2F",
			IconName:         "alert-circle",
		})
	}

	return suggestions
}

// inferTagsFromDomain suggests tags based on business domain
func (s *TagSuggestionService) inferTagsFromDomain(domain string) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}
	lowerDomain := strings.ToLower(domain)

	// Map domains to tags
	domainTags := map[string]models.TagSuggestion{
		"sales": {
			TagKey:      "sales",
			TagLabel:    "Sales",
			TagCategory: "business_area",
			ColorCode:   "#2E7D32",
			IconName:    "trending-up",
		},
		"finance": {
			TagKey:      "finance",
			TagLabel:    "Finance",
			TagCategory: "business_area",
			ColorCode:   "#1565C0",
			IconName:    "dollar-sign",
		},
		"marketing": {
			TagKey:      "marketing",
			TagLabel:    "Marketing",
			TagCategory: "business_area",
			ColorCode:   "#C2185B",
			IconName:    "megaphone",
		},
		"operations": {
			TagKey:      "operations",
			TagLabel:    "Operations",
			TagCategory: "business_area",
			ColorCode:   "#F57C00",
			IconName:    "gear",
		},
		"hr": {
			TagKey:      "hr",
			TagLabel:    "Human Resources",
			TagCategory: "business_area",
			ColorCode:   "#6A1B9A",
			IconName:    "users",
		},
	}

	if tag, exists := domainTags[lowerDomain]; exists {
		tag.SuggestionReason = "inferred_from_domain"
		tag.ConfidenceScore = 0.95
		suggestions = append(suggestions, tag)
	}

	return suggestions
}

// inferTagsFromExpression suggests tags based on the SQL/calculation expression
func (s *TagSuggestionService) inferTagsFromExpression(expression string) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}
	lowerExpr := strings.ToLower(expression)

	// If expression is not empty, it's likely a derived metric
	if expression != "" {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "derived_metric",
			TagLabel:         "Derived Metric",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_expression",
			ConfidenceScore:  0.9,
			ColorCode:        "#7B1FA2",
			IconName:         "function",
		})
	}

	// Detect calculation patterns
	if strings.Contains(lowerExpr, "sum") || strings.Contains(lowerExpr, "sum(") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "measure",
			TagLabel:         "Measure",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_expression",
			ConfidenceScore:  0.85,
			ColorCode:        "#388E3C",
			IconName:         "calculator",
		})
	}

	if strings.Contains(lowerExpr, "count(") || strings.Contains(lowerExpr, "count ") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "measure",
			TagLabel:         "Measure",
			TagCategory:      "domain",
			SuggestionReason: "inferred_from_expression",
			ConfidenceScore:  0.85,
			ColorCode:        "#388E3C",
			IconName:         "calculator",
		})
	}

	if strings.Contains(lowerExpr, "case when") || strings.Contains(lowerExpr, "if(") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "categorical",
			TagLabel:         "Categorical",
			TagCategory:      "data_type",
			SuggestionReason: "inferred_from_expression",
			ConfidenceScore:  0.75,
			ColorCode:        "#5E35B1",
			IconName:         "list",
		})
	}

	return suggestions
}

// inferTagsFromPhysicalMapping suggests tags based on database table/column mapping
func (s *TagSuggestionService) inferTagsFromPhysicalMapping(mapping *models.PhysicalMapping) []models.TagSuggestion {
	suggestions := []models.TagSuggestion{}

	if mapping == nil {
		return suggestions
	}

	lowerTable := strings.ToLower(mapping.Table)
	lowerColumn := strings.ToLower(mapping.Column)

	// Check if table suggests a domain
	if strings.Contains(lowerTable, "sales") || strings.Contains(lowerTable, "order") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "sales",
			TagLabel:         "Sales",
			TagCategory:      "business_area",
			SuggestionReason: "inferred_from_mapping",
			ConfidenceScore:  0.8,
			ColorCode:        "#2E7D32",
			IconName:         "trending-up",
		})
	}

	if strings.Contains(lowerTable, "customer") || strings.Contains(lowerColumn, "customer") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "customer",
			TagLabel:         "Customer",
			TagCategory:      "business_area",
			SuggestionReason: "inferred_from_mapping",
			ConfidenceScore:  0.85,
			ColorCode:        "#C62828",
			IconName:         "users",
		})
	}

	if strings.Contains(lowerTable, "financial") || strings.Contains(lowerTable, "ledger") || strings.Contains(lowerTable, "transaction") {
		suggestions = append(suggestions, models.TagSuggestion{
			TagKey:           "finance",
			TagLabel:         "Finance",
			TagCategory:      "business_area",
			SuggestionReason: "inferred_from_mapping",
			ConfidenceScore:  0.85,
			ColorCode:        "#1565C0",
			IconName:         "dollar-sign",
		})
	}

	return suggestions
}

// Helper function to merge suggestions avoiding duplicates
func (s *TagSuggestionService) mergeSuggestions(
	allSuggestions map[string]*models.TagSuggestion,
	newSuggestions []models.TagSuggestion,
	baseConfidence float64,
	reason string,
	response *models.TagSuggestionResponse,
) {
	for _, suggestion := range newSuggestions {
		suggestion.ConfidenceScore = baseConfidence
		suggestion.SuggestionReason = reason

		if existing, exists := allSuggestions[suggestion.TagKey]; exists {
			// Boost confidence if multiple sources suggest the same tag
			if suggestion.ConfidenceScore > existing.ConfidenceScore {
				allSuggestions[suggestion.TagKey] = &suggestion
				response.Reasons[suggestion.TagKey] = reason
			}
		} else {
			allSuggestions[suggestion.TagKey] = &suggestion
			response.Reasons[suggestion.TagKey] = reason
		}
	}
}

// Helper function to filter out already assigned tags
func (s *TagSuggestionService) filterExistingTags(suggestions []models.TagSuggestion, existingTags []string) []models.TagSuggestion {
	existingMap := make(map[string]bool)
	for _, tag := range existingTags {
		existingMap[tag] = true
	}

	filtered := []models.TagSuggestion{}
	for _, suggestion := range suggestions {
		if !existingMap[suggestion.TagKey] {
			filtered = append(filtered, suggestion)
		}
	}
	return filtered
}

// Helper function to sort suggestions by confidence
func sortSuggestionsByConfidence(suggestions []models.TagSuggestion) {
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[j].ConfidenceScore > suggestions[i].ConfidenceScore {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}
}

// TagSuggestionDB interface defines database operations needed by tag suggestion service
type TagSuggestionDB interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}
