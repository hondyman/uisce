package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
)

// PendingSemanticMapping represents a mapping awaiting approval
type PendingSemanticMapping struct {
	ID                    uuid.UUID    `json:"id" db:"id"`
	TenantID              uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	DatasourceID          uuid.UUID    `json:"datasource_id" db:"datasource_id"`
	ColumnID              uuid.UUID    `json:"column_id" db:"column_id"`
	ColumnName            string       `json:"column_name" db:"column_name"`
	ExpandedColumnName    string       `json:"expanded_column_name" db:"expanded_column_name"`
	SuggestedSemanticTerm string       `json:"suggested_semantic_term" db:"suggested_semantic_term"`
	SuggestedBusinessTerm string       `json:"suggested_business_term" db:"suggested_business_term"`
	SuggestedTitle        string       `json:"suggested_title" db:"suggested_title"`
	SuggestedDescription  string       `json:"suggested_description" db:"suggested_description"`
	SuggestedMeta         models.JSONB `json:"suggested_meta" db:"suggested_meta"`
	SuggestedFormat       string       `json:"suggested_format" db:"suggested_format"`

	Confidence    float64    `json:"confidence" db:"confidence"`
	Reasoning     string     `json:"reasoning" db:"reasoning"`
	LLMSuggestion *string    `json:"llm_suggestion,omitempty" db:"llm_suggestion"`
	Status        string     `json:"status" db:"status"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	ApprovedAt    *time.Time `json:"approved_at,omitempty" db:"approved_at"`
	ApprovedBy    *string    `json:"approved_by,omitempty" db:"approved_by"`
}

// GenerateMappingsRequest represents the request for generating mappings
type GenerateMappingsRequest struct {
	TenantID         string `json:"tenant_id"`
	DatasourceID     string `json:"datasource_id"`
	TenantInstanceID string `json:"datasource_id"`
}

// GenerateMappingsResponse represents the response with generated mappings
type GenerateMappingsResponse struct {
	TotalColumns      int                `json:"total_columns"`
	MappingsGenerated int                `json:"mappings_generated"`
	HighConfidence    int                `json:"high_confidence"`
	MediumConfidence  int                `json:"medium_confidence"`
	LowConfidence     int                `json:"low_confidence"`
	Mappings          []GeneratedMapping `json:"mappings"`
}

// GeneratedMapping represents a single generated mapping
type GeneratedMapping struct {
	ColumnID                 string              `json:"column_id"`
	ColumnName               string              `json:"column_name"`
	TableName                string              `json:"table_name"`
	ExpandedColumnName       string              `json:"expanded_column_name"`
	SuggestedSemanticTerm    string              `json:"suggested_semantic_term"`
	SuggestedBusinessTerm    string              `json:"suggested_business_term"`
	SuggestedTitle           string              `json:"suggested_title"`
	SuggestedDescription     string              `json:"suggested_description"`
	SuggestedMeta            models.JSONB        `json:"suggested_meta"`
	SuggestedFormat          string              `json:"suggested_format"`
	SemanticType             string              `json:"semantic_type"` // dimension, measure, time_dimension
	SuggestedValidationRules models.JSONB        `json:"suggested_validation_rules"`
	SuggestedEnrichment      *SemanticEnrichment `json:"suggested_enrichment,omitempty"`

	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
	WillAutoCreate bool    `json:"will_auto_create"`
	NeedsApproval  bool    `json:"needs_approval"`
}

// ApplyMappingsRequest represents the request to apply mappings
type ApplyMappingsRequest struct {
	TenantID            string  `json:"tenant_id"`
	DatasourceID        string  `json:"datasource_id"`
	TenantInstanceID    string  `json:"datasource_id"`
	AutoCreateThreshold float64 `json:"auto_create_threshold"` // Default 0.85
	ApprovalThreshold   float64 `json:"approval_threshold"`    // Default 0.60
}

// ApplyMappingsResponse represents the result of applying mappings
type ApplyMappingsResponse struct {
	AutoCreated     int `json:"auto_created"`
	PendingApproval int `json:"pending_approval"`
	Skipped         int `json:"skipped"`
	Errors          int `json:"errors"`
}

// GenerateMappingsWithAI scans columns and generates semantic term suggestions using AI
func (s *SemanticMappingService) GenerateMappingsWithAI(ctx context.Context, req *GenerateMappingsRequest) (*GenerateMappingsResponse, error) {
	logger := logging.GetLogger().Sugar()

	// Handle field alias
	if req.DatasourceID == "" && req.TenantInstanceID != "" {
		req.DatasourceID = req.TenantInstanceID
	}

	logger.Infof("Generating semantic mappings for tenant %s, datasource %s", req.TenantID, req.DatasourceID)

	// Resolve the correct datasource ID for catalog queries
	catalogDatasourceID, err := s.resolveCatalogDatasourceID(ctx, req.DatasourceID)
	if err != nil {
		logger.Warnf("Failed to resolve datasource ID: %v", err)
	}

	// 1. Load feedback stats (historical approvals/rejections)
	feedbackStats, err := s.loadFeedbackStats(ctx, req.TenantID, catalogDatasourceID)
	if err != nil {
		logger.Warnf("Failed to load feedback stats: %v", err)
		// Continue without feedback stats
		feedbackStats = &MappingFeedbackStats{
			ApprovedPatterns: make(map[string]int),
			RejectedPatterns: make(map[string]int),
			ColumnMappings:   make(map[string]string),
		}
	}

	// 2. Fetch all unmapped columns using the resolved catalogDatasourceID
	columns, err := s.fetchUnmappedColumns(ctx, req.TenantID, catalogDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch columns: %w", err)
	}

	response := &GenerateMappingsResponse{
		TotalColumns: len(columns),
		Mappings:     make([]GeneratedMapping, 0, len(columns)),
	}

	// 3. For each column, expand abbreviations and generate suggestions
	for _, col := range columns {
		logger.Infof("Processing column: %s, table: '%s', schema: '%s'", col.Column, col.Table, col.Schema)
		mapping, err := s.generateSingleMapping(ctx, &col, req.TenantID, catalogDatasourceID, feedbackStats)
		if err != nil {
			logger.Warnf("Failed to generate mapping for column %s: %v", col.Column, err)
			continue
		}

		response.Mappings = append(response.Mappings, *mapping)
		response.MappingsGenerated++

		// Categorize by confidence
		if mapping.Confidence >= 0.85 {
			response.HighConfidence++
			mapping.WillAutoCreate = true
		} else if mapping.Confidence >= 0.60 {
			response.MediumConfidence++
			mapping.NeedsApproval = true
		} else {
			response.LowConfidence++
		}
	}

	return response, nil
}

// Generic Attribute List - columns that need table context per spec
var genericAttributeList = map[string]bool{
	"id": true, "uuid": true, "key": true, "pk": true, "fk": true,
	"name": true, "title": true, "date": true, "dt": true, "type": true,
	"typ": true, "status": true, "cd": true, "code": true, "desc": true,
	"address": true, "description": true, "value": true, "num": true,
	"number": true, "flag": true, "indicator": true, "ind": true,
}

// isGenericAttribute checks if a column name is in the Generic Attribute List
func isGenericAttribute(columnName string) bool {
	upperName := strings.ToUpper(columnName)

	// Check if the full name is generic
	if genericAttributeList[strings.ToLower(columnName)] {
		return true
	}

	// Check if it ends with a generic suffix (e.g., USER_ID, ORDER_NAME)
	genericSuffixes := []string{"_ID", "_UUID", "_KEY", "_PK", "_FK", "_NAME", "_TITLE",
		"_DATE", "_DT", "_TYPE", "_TYP", "_STATUS", "_CD", "_CODE", "_DESC", "_ADDRESS"}
	for _, suffix := range genericSuffixes {
		if strings.HasSuffix(upperName, suffix) {
			return true
		}
	}

	// Very short names are ambiguous
	if len(columnName) <= 2 {
		return true
	}

	return false
}

// hasRedundantPrefix checks if table name is already present in the term to avoid CUSTOMER_CUSTOMER_ID
func hasRedundantPrefix(tableName, term string) bool {
	upperTable := strings.ToUpper(tableName)
	upperTerm := strings.ToUpper(term)

	// Check if term already starts with table name
	return strings.HasPrefix(upperTerm, upperTable+"_") || strings.HasPrefix(upperTerm, upperTable)
}

// toCamelCase converts SNAKE_CASE to camelCase for Cube.dev name property
func toCamelCase(s string) string {
	s = strings.ToLower(s)
	parts := strings.Split(s, "_")
	result := ""
	for i, part := range parts {
		if part == "" {
			continue
		}
		if i == 0 {
			result += part
		} else {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}

// inferFormat suggests a Cube.dev format based on column name patterns
func inferFormat(columnName string, semanticType string) string {
	upperCol := strings.ToUpper(columnName)

	// Currency patterns
	if strings.Contains(upperCol, "_AMT") || strings.Contains(upperCol, "_AMOUNT") ||
		strings.Contains(upperCol, "_BALANCE") || strings.Contains(upperCol, "_PRICE") ||
		strings.Contains(upperCol, "_COST") || strings.Contains(upperCol, "_REVENUE") ||
		strings.Contains(upperCol, "_TOTAL") && semanticType == "measure" {
		return "currency"
	}

	// Percentage patterns
	if strings.Contains(upperCol, "_PCT") || strings.Contains(upperCol, "_PERCENT") ||
		strings.Contains(upperCol, "_RATE") || strings.HasSuffix(upperCol, "_RATIO") {
		return "percent"
	}

	// Date/time patterns
	if semanticType == "time_dimension" {
		if strings.Contains(upperCol, "TIME") || strings.Contains(upperCol, "TS") ||
			strings.Contains(upperCol, "TIMESTAMP") {
			return "datetime"
		}
		return "date"
	}

	// ID patterns (for display as plain string)
	if strings.HasSuffix(upperCol, "_ID") || strings.HasSuffix(upperCol, "_KEY") ||
		strings.HasSuffix(upperCol, "_CODE") {
		return "id"
	}

	return "" // No special format needed
}

// stripTechnicalPrefix removes common data warehouse prefixes from table/column names
func stripTechnicalPrefix(name string) string {
	upperName := strings.ToUpper(name)
	prefixes := []string{
		"DIM_", "FCT_", "FACT_", "STG_", "RAW_", "SRC_",
		"TBL_", "VW_", "VIEW_", "MV_", "MAT_",
		"PK_", "FK_", "SK_", "BK_", "NK_",
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(upperName, prefix) {
			return name[len(prefix):]
		}
	}
	return name
}

// singularizeTableName converts plural table names to singular form
func singularizeTableName(name string) string {
	// Common irregular plurals
	irregulars := map[string]string{
		"PEOPLE": "PERSON", "CHILDREN": "CHILD", "MEN": "MAN", "WOMEN": "WOMAN",
		"MICE": "MOUSE", "GEESE": "GOOSE", "TEETH": "TOOTH", "FEET": "FOOT",
		"ANALYSES": "ANALYSIS", "CRISES": "CRISIS", "INDICES": "INDEX",
		"STATUSES": "STATUS", "ADDRESSES": "ADDRESS",
	}

	upperName := strings.ToUpper(name)
	if singular, ok := irregulars[upperName]; ok {
		// Preserve original case pattern
		if name == strings.ToLower(name) {
			return strings.ToLower(singular)
		}
		return singular
	}

	// Standard English pluralization rules (applied in reverse)
	if strings.HasSuffix(upperName, "IES") && len(name) > 3 {
		// CATEGORIES -> CATEGORY
		return name[:len(name)-3] + "Y"
	}
	if strings.HasSuffix(upperName, "ES") && len(name) > 2 {
		// Check for -SES, -XES, -CHES, -SHES endings
		if strings.HasSuffix(upperName, "SES") || strings.HasSuffix(upperName, "XES") ||
			strings.HasSuffix(upperName, "CHES") || strings.HasSuffix(upperName, "SHES") {
			return name[:len(name)-2]
		}
	}
	if strings.HasSuffix(upperName, "S") && len(name) > 1 &&
		!strings.HasSuffix(upperName, "SS") && !strings.HasSuffix(upperName, "US") {
		// ORDERS -> ORDER, CUSTOMERS -> CUSTOMER (but not ADDRESS, STATUS)
		return name[:len(name)-1]
	}

	return name
}

// inferSemanticType infers the semantic type (dimension/measure/time_dimension) from data type and column name
func inferSemanticType(dataType string, columnName string) string {
	lowerType := strings.ToLower(dataType)
	upperCol := strings.ToUpper(columnName)

	// Time dimensions - check column name patterns first
	timePatterns := []string{"_DT", "_DATE", "_TS", "_TIMESTAMP", "_TIME", "CREATED_AT", "UPDATED_AT", "DELETED_AT"}
	for _, pattern := range timePatterns {
		if strings.HasSuffix(upperCol, pattern) || strings.Contains(upperCol, pattern+"_") {
			return "time_dimension"
		}
	}
	// Also check data type
	if strings.Contains(lowerType, "date") || strings.Contains(lowerType, "time") ||
		strings.Contains(lowerType, "timestamp") {
		return "time_dimension"
	}

	// Measures - check column name patterns for aggregatable values
	measurePatterns := []string{"_AMT", "_AMOUNT", "_QTY", "_QUANTITY", "_COUNT", "_CNT",
		"_TOTAL", "_SUM", "_PRICE", "_COST", "_RATE", "_PCT", "_PERCENT", "_VALUE", "_BALANCE"}
	for _, pattern := range measurePatterns {
		if strings.HasSuffix(upperCol, pattern) || strings.Contains(upperCol, pattern+"_") {
			return "measure"
		}
	}
	// Check numeric data types for measures
	if strings.Contains(lowerType, "int") || strings.Contains(lowerType, "numeric") ||
		strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") ||
		strings.Contains(lowerType, "double") || strings.Contains(lowerType, "money") ||
		strings.Contains(lowerType, "real") {
		// Only treat as measure if it looks aggregatable (not IDs)
		if !strings.HasSuffix(upperCol, "_ID") && !strings.HasSuffix(upperCol, "_KEY") &&
			!strings.HasSuffix(upperCol, "_CODE") && !strings.HasSuffix(upperCol, "_NUM") {
			return "measure"
		}
	}

	// Default to dimension for everything else (strings, booleans, IDs, etc.)
	return "dimension"
}

// inferAggregationType suggests an aggregation type for measures
func inferAggregationType(columnName string) string {
	upperCol := strings.ToUpper(columnName)

	// Count-based columns
	if strings.Contains(upperCol, "COUNT") || strings.Contains(upperCol, "_CNT") ||
		strings.HasSuffix(upperCol, "_NUM") {
		return "count"
	}

	// Average/rate-based columns
	if strings.Contains(upperCol, "AVG") || strings.Contains(upperCol, "AVERAGE") ||
		strings.Contains(upperCol, "RATE") || strings.Contains(upperCol, "_PCT") ||
		strings.Contains(upperCol, "PERCENT") {
		return "avg"
	}

	// Default to sum for amounts, quantities, totals
	return "sum"
}

// formatSemanticTermName formats a semantic term in UPPERCASE_WITH_UNDERSCORES
func formatSemanticTermName(term string) string {
	// Remove leading underscores
	term = strings.TrimLeft(term, "_")

	// Convert to uppercase (keep underscores)
	term = strings.ToUpper(term)

	return term
}

// formatBusinessTermName formats a business term with Title Case and spaces
func formatBusinessTermName(term string) string {
	// Remove leading underscores
	term = strings.TrimLeft(term, "_")

	// Replace underscores with spaces
	term = strings.ReplaceAll(term, "_", " ")

	// Title case each word
	words := strings.Fields(term)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}

	return strings.Join(words, " ")
}

// generateSingleMapping generates a mapping for a single column
func (s *SemanticMappingService) generateSingleMapping(ctx context.Context, col *DatabaseColumn, tenantID, datasourceID string, feedbackStats *MappingFeedbackStats) (*GeneratedMapping, error) {
	// 0. Apply deterministic rules: strip technical prefixes from table name
	cleanTableName := stripTechnicalPrefix(col.Table)
	cleanTableName = singularizeTableName(cleanTableName)

	// 1. Strip prefixes from column name and expand abbreviations
	cleanColumnName := stripTechnicalPrefix(col.Column)
	expandedName := cleanColumnName
	if s.abbreviationSvc != nil {
		expanded, err := s.abbreviationSvc.ExpandToHumanReadable(ctx, cleanColumnName)
		if err == nil && expanded != "" {
			expandedName = expanded
		}
	}

	// 2. Start with the expanded column name as the base semantic term
	semanticTerm := expandedName

	// 3. Smart Contextualization: Check if column is a generic attribute needing table context
	if isGenericAttribute(cleanColumnName) {
		// Redundancy protection: avoid CUSTOMER_CUSTOMER_ID
		if !hasRedundantPrefix(cleanTableName, expandedName) {
			// Prepend cleaned and singularized table name for context
			semanticTerm = cleanTableName + "_" + expandedName
		}
		// 4. Use LLM to improve the suggestion if available
		var suggestion SemanticSuggestion
		if s.llmProvider != nil {
			improvedSuggestion, err := s.suggestSemanticTermWithLLM(ctx, col, expandedName, semanticTerm)
			if err == nil && improvedSuggestion.TermName != "" {
				semanticTerm = improvedSuggestion.TermName
				suggestion = improvedSuggestion
			}
		}

		// 5. Format the semantic term (uppercase with underscores)
		semanticTerm = formatSemanticTermName(semanticTerm)

		// 6. Generate and format business term (title case with spaces)
		businessTerm := s.generateBusinessTermName(expandedName, col.Table)
		businessTerm = formatBusinessTermName(businessTerm)

		// 7. Calculate confidence
		confidence, reasoning := s.calculateMappingConfidence(ctx, col, expandedName, semanticTerm, tenantID, datasourceID)

		// Boost confidence if LLM was used and returned a result
		if suggestion.TermName != "" {
			confidence += 0.15
			if confidence > 1.0 {
				confidence = 1.0
			}
			reasoning = fmt.Sprintf("%s; Enhanced by AI analysis. ", reasoning)
		}

		// 8. Apply feedback adjustments
		patternKey := fmt.Sprintf("%s:%s", strings.ToLower(col.Column), strings.ToLower(semanticTerm))

		// Heuristic: If rejected >= 3 times, heavily penalize or skip
		rejectionCount := feedbackStats.RejectedPatterns[patternKey]
		if rejectionCount >= 3 {
			// Penalize confidence significantly instead of skipping entirely
			confidence -= 0.5
			reasoning = fmt.Sprintf("%s; Penalized due to multiple rejections (%d times)", reasoning, rejectionCount)
			if confidence < 0 {
				confidence = 0.1 // Minimum confidence
			}
		} else if rejectionCount > 0 {
			// Slight penalty for 1-2 rejections
			confidence -= 0.1 * float64(rejectionCount)
			reasoning = fmt.Sprintf("%s; Penalized due to prior rejection", reasoning)
		}

		// Heuristic: If approved >= 3 times, boost
		// If approved < 3 times, still boost but less?
		approvalCount := feedbackStats.ApprovedPatterns[patternKey]
		if approvalCount > 0 {
			boost := 0.1 // Base boost
			if approvalCount >= 3 {
				boost = 0.2 // Higher boost for frequent approvals
			}
			confidence = confidence + boost
			if confidence > 1.0 {
				confidence = 1.0
			}
			reasoning = fmt.Sprintf("%s; Boosted by approval history (%d times)", reasoning, approvalCount)
		}

		var metaRaw models.JSONB
		if len(suggestion.Meta) > 0 {
			b, _ := json.Marshal(suggestion.Meta)
			metaRaw = models.JSONB(b)
		}

		// Determine semantic type - from LLM suggestion or infer from data type and column name
		semanticType := suggestion.SemanticType
		if semanticType == "" {
			semanticType = inferSemanticType(col.DataType, col.Column)
		}

		// Generate enrichment using the semantic enricher
		enricher := NewSemanticEnricher(s.db, s.llmProvider)
		enrichment := enricher.EnrichFromColumnData(
			col.Column,
			col.Table,
			col.DataType,
			true,  // isNullable - will be overridden by actual validation rules
			false, // isForeignKey - inferred from column name in enricher
			false, // isPrimaryKey - inferred from column name in enricher
		)

		return &GeneratedMapping{
			ColumnID:                 col.NodeID,
			ColumnName:               col.Column,
			TableName:                col.Table,
			ExpandedColumnName:       expandedName,
			SuggestedSemanticTerm:    semanticTerm,
			SuggestedBusinessTerm:    businessTerm,
			SuggestedTitle:           suggestion.Title,
			SuggestedDescription:     suggestion.Description,
			SuggestedMeta:            metaRaw,
			SuggestedFormat:          suggestion.Format,
			SemanticType:             semanticType,
			SuggestedValidationRules: col.SuggestedValidationRules,
			SuggestedEnrichment:      enrichment,
			Confidence:               confidence,
			Reasoning:                reasoning,
			WillAutoCreate:           false,
			NeedsApproval:            false,
		}, nil
	}
	return nil, fmt.Errorf("unexpected")
}
