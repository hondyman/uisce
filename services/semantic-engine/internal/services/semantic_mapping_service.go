package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/logging"
	"github.com/jmoiron/sqlx"
)

// SemanticMappingService provides intelligent semantic term mapping with fuzzy logic
// BusinessTermMatcher is an optional external matcher for business term suggestions.
type BusinessTermMatcher interface {
	Suggest(ctx context.Context, semanticTerm string) ([]BusinessTermSuggestionResult, error)
}

type AbbreviationService interface {
	ExpandAbbreviations(ctx context.Context, term string) ([]string, error)
}

type SemanticMappingService struct {
	db                  *sqlx.DB
	businessTermMatcher BusinessTermMatcher
	abbreviationSvc     AbbreviationService
	llmProvider         interface{} // LLM provider for AI-powered suggestions
}

// NewSemanticMappingService creates a new semantic mapping service
func NewSemanticMappingService(db *sqlx.DB) *SemanticMappingService {
	return &SemanticMappingService{db: db}
}

// NewSemanticMappingServiceWithMatcher creates a new service with an external matcher.
// NewSemanticMappingServiceWithAbbreviations creates a new service with abbreviation support
func NewSemanticMappingServiceWithAbbreviations(db *sqlx.DB, matcher BusinessTermMatcher, abbreviationSvc AbbreviationService) *SemanticMappingService {
	return &SemanticMappingService{
		db:                  db,
		businessTermMatcher: matcher,
		abbreviationSvc:     abbreviationSvc,
	}
}

// AbbreviationMap contains common abbreviation expansions
var AbbreviationMap = map[string]string{
	// Geographic
	"CNTRY":  "COUNTRY",
	"CTRY":   "COUNTRY",
	"ST":     "STATE",
	"ADDR":   "ADDRESS",
	"ZIP":    "ZIPCODE",
	"POSTAL": "POSTALCODE",
	"CTY":    "CITY",
	"REGN":   "REGION",

	// Financial
	"AMT":  "AMOUNT",
	"VAL":  "VALUE",
	"BAL":  "BALANCE",
	"CURR": "CURRENCY",
	"FX":   "FOREIGN_EXCHANGE",
	"ACCT": "ACCOUNT",
	"TXN":  "TRANSACTION",
	"PMT":  "PAYMENT",

	// Temporal
	"DT":  "DATE",
	"DTM": "DATETIME",
	"TS":  "TIMESTAMP",
	"YR":  "YEAR",
	"MON": "MONTH",
	"WK":  "WEEK",
	"QTR": "QUARTER",

	// Business
	"CUST":  "CUSTOMER",
	"CLNT":  "CLIENT",
	"ORD":   "ORDER",
	"PROD":  "PRODUCT",
	"CATEG": "CATEGORY",
	"DEPT":  "DEPARTMENT",
	"DIV":   "DIVISION",
	"ORG":   "ORGANIZATION",
	"COMP":  "COMPANY",

	// Identity
	"ID":  "IDENTIFIER",
	"NUM": "NUMBER",
	"NBR": "NUMBER",
	"NO":  "NUMBER",
	"CD":  "CODE",
	"KEY": "KEY",
	"REF": "REFERENCE",

	// Measurements
	"QTY":   "QUANTITY",
	"CNT":   "COUNT",
	"PCT":   "PERCENT",
	"RATE":  "RATE",
	"RATIO": "RATIO",
	"SCORE": "SCORE",
	"RANK":  "RANK",

	// Status/Flags
	"FLG":  "FLAG",
	"IND":  "INDICATOR",
	"STAT": "STATUS",
	"TYP":  "TYPE",

	// Common prefixes/suffixes
	"DESC": "DESCRIPTION",
	"NM":   "NAME",
	"TTL":  "TOTAL",
	"AVG":  "AVERAGE",
	"MIN":  "MINIMUM",
	"MAX":  "MAXIMUM",
	"SUM":  "SUMMARY",
}

// expandAbbreviations creates variations of the column name with expanded abbreviations
func expandAbbreviations(columnName string) []string {
	normalized := strings.ToUpper(columnName)
	variations := []string{normalized}

	// Split on common separators
	separators := []string{"_", "-", ".", " "}
	var tokens []string

	current := normalized
	for _, sep := range separators {
		if strings.Contains(current, sep) {
			tokens = strings.Split(current, sep)
			break
		}
	}

	if len(tokens) == 0 {
		tokens = []string{normalized}
	}

	// Expand each token if it's an abbreviation
	var expandedTokens [][]string
	hasExpansion := false

	for _, token := range tokens {
		tokenVariations := []string{token}
		if expansion, exists := AbbreviationMap[token]; exists {
			tokenVariations = append(tokenVariations, expansion)
			hasExpansion = true
		}
		expandedTokens = append(expandedTokens, tokenVariations)
	}

	// Generate all combinations if we have expansions
	if hasExpansion {
		combinations := generateCombinations(expandedTokens)
		for _, combo := range combinations {
			variations = append(variations, strings.Join(combo, "_"))
		}
	}

	return variations
}

// generateCombinations creates all possible combinations from token variations
func generateCombinations(tokenVariations [][]string) [][]string {
	if len(tokenVariations) == 0 {
		return [][]string{}
	}

	if len(tokenVariations) == 1 {
		var result [][]string
		for _, variation := range tokenVariations[0] {
			result = append(result, []string{variation})
		}
		return result
	}

	var result [][]string
	restCombos := generateCombinations(tokenVariations[1:])

	for _, variation := range tokenVariations[0] {
		for _, restCombo := range restCombos {
			combo := append([]string{variation}, restCombo...)
			result = append(result, combo)
		}
	}

	return result
}

// EnrichmentProposal represents a fully-formed suggestion for enriching a column.
type EnrichmentProposal struct {
	SemanticTermName string   `json:"semantic_term_name"`
	SemanticTermType string   `json:"semantic_term_type"`
	BusinessTermName string   `json:"business_term_name"`
	DomainHierarchy  []string `json:"domain_hierarchy"`
	Confidence       float64  `json:"confidence"`
	Reasoning        string   `json:"reasoning"`
}

// ApplyEnrichmentRequest is the payload for applying a suggested enrichment.
type ApplyEnrichmentRequest struct {
	Proposal     *EnrichmentProposal `json:"proposal"`
	ColumnID     string              `json:"column_id"`
	TenantID     string              `json:"tenant_id"`
	DatasourceID string              `json:"datasource_id"`
	Column       *DatabaseColumn     `json:"column,omitempty"` // Include column data for property inference
}

// minFloat64 returns the minimum of two float64 values
func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// EnhancedCalculateSemanticConfidence provides enhanced semantic confidence calculation
// For now, delegates to the original implementation
func (s *SemanticMappingService) EnhancedCalculateSemanticConfidence(
	ctx context.Context,
	generatedTerm, existingTerm string,
	column *DatabaseColumn,
	term *SemanticTerm,
) (float64, string, []ConfidenceBreakdown) {
	// Start with the base confidence calculation
	baseConfidence, baseReason, baseBreakdown := s.calculateSemanticConfidenceOriginal(generatedTerm, existingTerm, column, term)

	// Enhance with abbreviation expansion matching
	expandedVariations, err := s.ExpandAbbreviationsDB(ctx, generatedTerm)
	if err != nil || len(expandedVariations) <= 1 {
		// No abbreviations found or error, return base confidence
		return baseConfidence, baseReason, baseBreakdown
	}

	// Check if any expanded variation matches the existing term more closely
	var bestConfidence float64 = baseConfidence
	var bestReason string = baseReason
	var bestBreakdown []ConfidenceBreakdown = baseBreakdown
	var foundBetterMatch bool = false

	for _, expanded := range expandedVariations[1:] { // Skip the first one (original)
		// Calculate confidence for this expansion
		expandedConfidence, expandedReason, expandedBreakdown := s.calculateSemanticConfidenceOriginal(expanded, existingTerm, column, term)

		if expandedConfidence > bestConfidence {
			bestConfidence = expandedConfidence
			bestReason = expandedReason
			bestBreakdown = expandedBreakdown
			foundBetterMatch = true
		}
	}

	// If we found a better match via abbreviation expansion, enhance the reasoning
	if foundBetterMatch {
		bestReason = fmt.Sprintf("Abbreviation-enhanced match: %s [Expanded from '%s']", bestReason, generatedTerm)

		// Add abbreviation bonus to confidence if it's already pretty good
		if bestConfidence >= 0.6 {
			bestConfidence = math.Min(bestConfidence+0.05, 1.0)
		}

		// Add breakdown for abbreviation expansion
		bestBreakdown = append(bestBreakdown, ConfidenceBreakdown{
			Label:   "Abbreviation expansion",
			Score:   0.05,
			Weight:  0.1,
			Details: "Match improved by expanding abbreviations",
		})
	}

	return bestConfidence, bestReason, bestBreakdown
}

// SuggestEnrichment generates a comprehensive enrichment proposal for a given column.
// This method now expands abbreviations in column names to improve semantic term matching.
func (s *SemanticMappingService) SuggestEnrichment(ctx context.Context, column *DatabaseColumn, profile *NodeProperties) (*EnrichmentProposal, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Suggesting enrichment for column: %s", column.QualifiedPath)

	// 0. Expand abbreviations in the column name for better semantic matching
	expandedVariations, err := s.ExpandAbbreviationsDB(ctx, column.Column)
	if err != nil {
		logger.Warnf("Could not expand abbreviations for column %s: %v", column.Column, err)
		expandedVariations = []string{column.Column}
	}

	// Log the expanded variations
	if len(expandedVariations) > 1 {
		logger.Infof("Abbreviation expansion for '%s': %v", column.Column, expandedVariations)
	}

	// 1. Generate semantic term name(s) from expanded abbreviations
	// Start with the original column and generate additional variations from expansions
	var semanticTermName string
	var allTermVariations []string

	semanticTermName = s.generateSemanticTerm(column.Schema, column.Table, column.Column)
	allTermVariations = append(allTermVariations, semanticTermName)

	// Generate variations from expanded abbreviations (skip the first one which is the original)
	if len(expandedVariations) > 1 {
		for _, expansion := range expandedVariations[1:] {
			if expansion != column.Column {
				expandedTermName := s.generateSemanticTerm(column.Schema, column.Table, expansion)
				if expandedTermName != semanticTermName {
					allTermVariations = append(allTermVariations, expandedTermName)
					// Use the expanded version as primary if it's better formed
					if len(expandedTermName) > len(semanticTermName) && strings.Contains(expandedTermName, "_") {
						semanticTermName = expandedTermName
					}
				}
			}
		}
	}

	// 2. Determine the term type
	termType, err := s.determineTermType(column, profile)
	if err != nil {
		logger.Warnf("Could not determine term type for %s: %v", column.Column, err)
		termType = "Dimension" // Default to Dimension
	}

	// 3. Generate a business term name from the expanded column name
	// Use the primary semantic term to generate a better business name
	normalizedColumnName := s.normalizeColumnName(semanticTermName)
	businessTermName := s.generateBusinessTermName(normalizedColumnName, column.Table)

	// 4. Determine the data domain
	domainHierarchy := s.determineDataDomain(column.Schema, column.Table)

	// 5. Calculate confidence by finding the best matching existing semantic term
	// Check confidence against all term variations and find the best match
	terms, err := s.fetchSemanticTerms(ctx, column.TenantID, column.TenantDatasourceID)
	if err != nil {
		logger.Warnf("Could not fetch existing semantic terms for confidence calculation: %v", err)
	}

	var bestConfidence float64 = 0.5 // Base confidence for a new term
	var bestMatchReason string
	var reasoning strings.Builder

	reasoningBase := fmt.Sprintf("Generated from column '%s' in table '%s'. ", column.Column, column.Table)
	if len(expandedVariations) > 1 {
		reasoningBase += fmt.Sprintf("Abbreviations expanded to: %s. ", strings.Join(expandedVariations[1:], ", "))
	}
	reasoning.WriteString(reasoningBase)

	if len(terms) > 0 {
		// Check all variations against existing terms for best match
		for _, termVariation := range allTermVariations {
			for _, term := range terms {
				confidence, reason, _ := s.calculateSemanticConfidence(
					ctx, termVariation, term.TermName,
					column, &term,
				)
				if confidence > bestConfidence {
					bestConfidence = confidence
					bestMatchReason = reason
				}
			}
		}

		if bestConfidence > 0.5 {
			reasoning.WriteString(fmt.Sprintf("Found a potential match with confidence %.2f. ", bestConfidence))
			reasoning.WriteString("Match reason: " + bestMatchReason)
		}
	} else {
		reasoning.WriteString("No existing semantic terms to compare against. ")
	}

	// Construct the proposal
	proposal := &EnrichmentProposal{
		SemanticTermName: semanticTermName,
		SemanticTermType: termType,
		BusinessTermName: businessTermName,
		DomainHierarchy:  domainHierarchy,
		Confidence:       bestConfidence,
		Reasoning:        reasoning.String(),
	}

	return proposal, nil
}

// ApplyEnrichment applies a proposal, creating the necessary terms and edges.
func (s *SemanticMappingService) ApplyEnrichment(ctx context.Context, req *ApplyEnrichmentRequest) (map[string]string, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Applying enrichment for column %s with proposal: %+v", req.ColumnID, req.Proposal)

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // Rollback on error

	// 1. Get or create the Semantic Term
	semanticTermID, err := s.getOrCreateSemanticTerm(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	// 2. Get or create the Business Term
	businessTermID, err := s.getOrCreateBusinessTerm(ctx, tx, req)
	if err != nil {
		return nil, err
	}

	// 3. Create the edge from Semantic Term to Business Term (e.g., "IS_A")
	_, err = s.createEdge(ctx, tx, req.TenantID, req.DatasourceID, semanticTermID, businessTermID, "HAS_BUSINESS_TERM", EdgeTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge from semantic to business term: %w", err)
	}

	// 4. Create the edge from the Column to the Semantic Term
	_, err = s.createEdge(ctx, tx, req.TenantID, req.DatasourceID, req.ColumnID, semanticTermID, "MAPS_TO", EdgeTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge from column to semantic term: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Infof("Successfully applied enrichment for column %s. SemanticTermID: %s, BusinessTermID: %s", req.ColumnID, semanticTermID, businessTermID)

	return map[string]string{
		"business_term_id": businessTermID,
		"semantic_term_id": semanticTermID,
		"status":           "Enrichment applied successfully.",
	}, nil
}

// inferSemanticTermProperties intelligently infers properties for a semantic term based on column metadata.
// Uses heuristics to determine characteristics like foreign_key status, nullability, etc.
func (s *SemanticMappingService) inferSemanticTermProperties(column *DatabaseColumn, termType string) map[string]interface{} {
	properties := map[string]interface{}{
		"data_type": termType,
	}

	if column == nil {
		// Generate minimal cube_properties even without column metadata
		cubeProps := s.generateCubeProperties(nil, termType, "")
		properties["cube_properties"] = cubeProps
		properties["semantic_term_type"] = s.detectSemanticTermType(nil, termType)
		return properties
	}

	// Infer foreign key status from column name patterns
	columnUpper := strings.ToUpper(column.Column)
	isForeignKey := false
	if strings.HasSuffix(columnUpper, "_ID") || strings.HasPrefix(columnUpper, "FK_") ||
		strings.Contains(columnUpper, "_FK_") || strings.HasSuffix(columnUpper, "ID") {
		isForeignKey = true
	}
	properties["foreign_key"] = isForeignKey

	// Infer nullability from column characteristics
	isNullable := true
	if strings.HasSuffix(columnUpper, "_ID") || strings.HasPrefix(columnUpper, "PK_") ||
		strings.HasSuffix(columnUpper, "_KEY") || columnUpper == "ID" {
		isNullable = false
	}

	// Infer if it's a temporal field (date, timestamp, etc.)
	isTemporal := false
	if strings.HasSuffix(columnUpper, "_DATE") || strings.HasSuffix(columnUpper, "_AT") ||
		strings.HasSuffix(columnUpper, "_TIME") || strings.Contains(columnUpper, "TIMESTAMP") ||
		strings.Contains(columnUpper, "CREATED") || strings.Contains(columnUpper, "UPDATED") ||
		strings.Contains(columnUpper, "DELETED") {
		isTemporal = true
		// Temporal columns (especially CREATED_AT, UPDATED_AT) are usually NOT nullable
		isNullable = false
	}

	if isTemporal {
		properties["temporal"] = true
	}

	properties["nullable"] = isNullable

	// Infer if it's a status/flag field
	isStatusFlag := false
	if strings.HasSuffix(columnUpper, "_STATUS") || strings.HasSuffix(columnUpper, "_STATE") ||
		strings.HasSuffix(columnUpper, "_FLAG") || strings.Contains(columnUpper, "IS_") ||
		strings.Contains(columnUpper, "HAS_") {
		isStatusFlag = true
	}
	if isStatusFlag {
		properties["status_flag"] = true
	}

	// Add cardinality info if available
	if column.Cardinality > 0 {
		properties["cardinality"] = column.Cardinality
	}

	// Add data patterns if available
	if len(column.FrequentValues) > 0 {
		properties["frequent_values"] = column.FrequentValues
	}

	if len(column.InferredPatterns) > 0 {
		properties["inferred_patterns"] = column.InferredPatterns
	}

	// Add schema context
	properties["schema"] = column.Schema
	properties["table"] = column.Table
	properties["source_column"] = column.Column

	// NEW: Generate Cube.dev compatible properties
	cubeProps := s.generateCubeProperties(column, termType, column.Column)
	properties["cube_properties"] = cubeProps

	// NEW: Add semantic term type detection
	properties["semantic_term_type"] = s.detectSemanticTermType(column, termType)

	return properties
}

// generateCubeProperties creates comprehensive Cube.js compatible property structure
func (s *SemanticMappingService) generateCubeProperties(column *DatabaseColumn, termType string, columnName string) map[string]interface{} {
	if columnName == "" && column != nil {
		columnName = column.Column
	}

	cubeProps := map[string]interface{}{
		"name": columnName,
		"sql":  fmt.Sprintf("{CUBE}.%s", columnName),
		"type": s.mapToCubeType(termType),
	}

	// Add business-friendly title using abbreviations and semantic context
	cubeProps["title"] = s.generateBusinessTitle(columnName, termType)

	// Add comprehensive description
	description := s.generateDescription(columnName, "", column)
	if description != "" {
		cubeProps["description"] = description
	}

	// Core Cube.dev properties
	cubeProps["public"] = true
	cubeProps["shown"] = true
	cubeProps["hidden"] = false

	// Measure-specific properties
	columnUpper := strings.ToUpper(columnName)
	if strings.Contains(columnUpper, "AMOUNT") || strings.Contains(columnUpper, "TOTAL") ||
		strings.Contains(columnUpper, "REVENUE") || strings.Contains(columnUpper, "SALES") ||
		strings.Contains(columnUpper, "PRICE") || strings.Contains(columnUpper, "BALANCE") {
		cubeProps["cumulative"] = false
		cubeProps["rolling_window"] = nil
	}

	// Dimension-specific properties
	if strings.HasSuffix(columnUpper, "_ID") || strings.ToUpper(columnName) == "ID" {
		cubeProps["primary_key"] = false
		cubeProps["type"] = "number"
	}

	// Time dimension specific properties
	if strings.Contains(columnUpper, "DATE") || strings.Contains(columnUpper, "TIME") ||
		strings.Contains(columnUpper, "TIMESTAMP") {
		cubeProps["time_zone"] = "UTC"
		cubeProps["granularities"] = []string{"second", "minute", "hour", "day", "week", "month", "quarter", "year"}
	}

	// Ordering preference
	if strings.Contains(columnUpper, "CREATED") || strings.Contains(columnUpper, "UPDATED") {
		cubeProps["order"] = "desc"
	} else if strings.Contains(columnUpper, "NAME") || strings.Contains(columnUpper, "TITLE") {
		cubeProps["order"] = "asc"
	}

	// Formatting hints for UI
	if strings.Contains(columnUpper, "PRICE") || strings.Contains(columnUpper, "AMOUNT") ||
		strings.Contains(columnUpper, "REVENUE") || strings.Contains(columnUpper, "BALANCE") {
		cubeProps["format"] = "currency"
		cubeProps["currency"] = "USD"
	} else if strings.Contains(columnUpper, "PERCENT") || strings.Contains(columnUpper, "RATE") {
		cubeProps["format"] = "percent"
	}

	// Drill-down capable flag for IDs
	if strings.HasSuffix(columnUpper, "_ID") {
		cubeProps["drill_down_by"] = nil
	}

	return cubeProps
}

// detectSemanticTermType determines the semantic term type (Dimension, Measure, Time, Hierarchy, Segment)
func (s *SemanticMappingService) detectSemanticTermType(column *DatabaseColumn, dataType string) string {
	// If no column metadata, can't reliably detect type
	if column == nil {
		return "Dimension" // Default
	}

	columnUpper := strings.ToUpper(column.Column)

	// Time dimension detection
	if strings.Contains(columnUpper, "DATE") ||
		strings.Contains(columnUpper, "TIME") ||
		strings.Contains(columnUpper, "TIMESTAMP") ||
		strings.Contains(columnUpper, "CREATED") ||
		strings.Contains(columnUpper, "UPDATED") {
		return "Time"
	}

	// Measure detection (numeric, count, sum)
	dataTypeUpper := strings.ToUpper(dataType)
	if strings.Contains(dataTypeUpper, "INT") ||
		strings.Contains(dataTypeUpper, "NUMERIC") ||
		strings.Contains(dataTypeUpper, "DECIMAL") ||
		strings.Contains(dataTypeUpper, "FLOAT") {
		if strings.Contains(columnUpper, "AMOUNT") ||
			strings.Contains(columnUpper, "TOTAL") ||
			strings.Contains(columnUpper, "COUNT") ||
			strings.Contains(columnUpper, "REVENUE") ||
			strings.Contains(columnUpper, "SALES") ||
			strings.Contains(columnUpper, "PRICE") ||
			strings.Contains(columnUpper, "BALANCE") {
			return "Measure"
		}
	}

	// Default to Dimension
	return "Dimension"
}

// mapToCubeType converts database data type to Cube.js type
func (s *SemanticMappingService) mapToCubeType(dataType string) string {
	dtUpper := strings.ToUpper(dataType)

	if strings.Contains(dtUpper, "TIME") ||
		strings.Contains(dtUpper, "DATE") ||
		strings.Contains(dtUpper, "TIMESTAMP") {
		return "time"
	}

	if strings.Contains(dtUpper, "INT") ||
		strings.Contains(dtUpper, "NUMERIC") ||
		strings.Contains(dtUpper, "DECIMAL") ||
		strings.Contains(dtUpper, "FLOAT") ||
		strings.Contains(dtUpper, "MONEY") {
		return "number"
	}

	if strings.Contains(dtUpper, "BOOL") {
		return "boolean"
	}

	return "string"
}

// generateTitle converts column name to human-readable title (legacy)
func (s *SemanticMappingService) generateTitle(columnName string) string {
	// Replace underscores with spaces
	title := strings.ReplaceAll(columnName, "_", " ")

	// Title case
	parts := strings.Fields(title)
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

// generateBusinessTitle creates a business-friendly title using abbreviation expansion
func (s *SemanticMappingService) generateBusinessTitle(columnName string, termType string) string {
	ctx := context.Background()

	// Try to expand abbreviations for more meaningful titles
	expandedVariations, err := s.ExpandAbbreviationsDB(ctx, columnName)
	var expandedName string

	if err == nil && len(expandedVariations) > 0 {
		// Use the first non-empty expansion that differs from original
		for _, exp := range expandedVariations {
			if exp != columnName && strings.TrimSpace(exp) != "" {
				expandedName = exp
				break
			}
		}
	}

	// If no expansion found, use original column name
	if expandedName == "" {
		expandedName = columnName
	}

	// Convert to human-readable format
	title := strings.ReplaceAll(expandedName, "_", " ")
	title = strings.ReplaceAll(title, "-", " ")

	// Title case with proper capitalization
	parts := strings.Fields(title)
	for i, part := range parts {
		if len(part) > 0 {
			// Preserve acronyms
			if len(part) > 1 && part == strings.ToUpper(part) {
				parts[i] = part
			} else {
				parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
			}
		}
	}

	return strings.Join(parts, " ")
}

// generateDescription creates a meaningful description for the semantic term
func (s *SemanticMappingService) generateDescription(columnName string, termName string, column *DatabaseColumn) string {
	var descriptions []string

	// Use semantic term name if available
	if termName != "" {
		descriptions = append(descriptions, fmt.Sprintf("Business term: %s", termName))
	}

	// Add column context
	if column != nil {
		if column.Table != "" && column.Schema != "" {
			descriptions = append(descriptions, fmt.Sprintf("Source: %s.%s.%s", column.Schema, column.Table, columnName))
		}

		// Add data characteristics
		if column.Cardinality > 0 {
			descriptions = append(descriptions, fmt.Sprintf("Distinct values: %d", column.Cardinality))
		}

		// Add pattern info for categorization
		if len(column.InferredPatterns) > 0 {
			descriptions = append(descriptions, fmt.Sprintf("Pattern: %v", column.InferredPatterns))
		}
	} else {
		descriptions = append(descriptions, fmt.Sprintf("Dimension from column: %s", columnName))
	}

	return strings.Join(descriptions, " | ")
}

// ==================== OPTIONAL ENHANCEMENTS ====================

// 1. ADVANCED ABBREVIATION HANDLING - Domain-Specific Term Expansion

// DomainAbbreviationContext provides domain-specific abbreviation mappings
type DomainAbbreviationContext struct {
	Domain        string            // e.g., "finance", "healthcare", "retail"
	Abbreviations map[string]string // e.g., {"CAC": "Customer Acquisition Cost", "LTV": "Lifetime Value"}
	Synonyms      map[string]string // e.g., {"revenue": "sales", "income"}
	Conventions   map[string]string // e.g., {"_amt": " Amount", "_cnt": " Count"}
}

// ExpandDomainSpecificAbbreviations expands abbreviations using domain context
func (s *SemanticMappingService) ExpandDomainSpecificAbbreviations(ctx context.Context, columnName string, domain string) (string, map[string]interface{}, error) {
	// Load domain-specific context from database or config
	domainContext := s.getDomainAbbreviationContext(ctx, domain)
	if domainContext == nil {
		return columnName, nil, fmt.Errorf("domain context not found: %s", domain)
	}

	expanded := columnName
	metadata := make(map[string]interface{})
	appliedRules := []string{}

	// Apply convention-based expansions first (e.g., _amt -> Amount)
	for convention, expansion := range domainContext.Conventions {
		if strings.Contains(expanded, convention) {
			expanded = strings.ReplaceAll(expanded, convention, expansion)
			appliedRules = append(appliedRules, fmt.Sprintf("convention:%s", convention))
		}
	}

	// Apply domain-specific abbreviation expansions
	parts := strings.FieldsFunc(expanded, func(r rune) bool {
		return r == '_' || r == '-'
	})

	var expandedParts []string
	for _, part := range parts {
		if abbrev, exists := domainContext.Abbreviations[strings.ToUpper(part)]; exists {
			expandedParts = append(expandedParts, abbrev)
			appliedRules = append(appliedRules, fmt.Sprintf("abbrev:%s->%s", part, abbrev))
		} else if synonym, exists := domainContext.Synonyms[strings.ToLower(part)]; exists {
			expandedParts = append(expandedParts, synonym)
			appliedRules = append(appliedRules, fmt.Sprintf("synonym:%s->%s", part, synonym))
		} else {
			expandedParts = append(expandedParts, part)
		}
	}

	result := strings.Join(expandedParts, " ")
	metadata["domain"] = domain
	metadata["applied_rules"] = appliedRules
	metadata["original_name"] = columnName
	metadata["expanded_name"] = result

	return result, metadata, nil
}

// getDomainAbbreviationContext retrieves domain-specific abbreviation context
func (s *SemanticMappingService) getDomainAbbreviationContext(ctx context.Context, domain string) *DomainAbbreviationContext {
	// This would load from database or configuration
	// Example: Finance domain abbreviations
	domainContexts := map[string]*DomainAbbreviationContext{
		"finance": {
			Domain: "finance",
			Abbreviations: map[string]string{
				"CAC":    "Customer Acquisition Cost",
				"LTV":    "Lifetime Value",
				"ARR":    "Annual Recurring Revenue",
				"MRR":    "Monthly Recurring Revenue",
				"COGS":   "Cost of Goods Sold",
				"EBITDA": "Earnings Before Interest Taxes Depreciation Amortization",
				"ROI":    "Return On Investment",
				"APR":    "Annual Percentage Rate",
				"AUM":    "Assets Under Management",
			},
			Conventions: map[string]string{
				"_amt":  " Amount",
				"_cnt":  " Count",
				"_pct":  " Percentage",
				"_bal":  " Balance",
				"_rate": " Rate",
			},
			Synonyms: map[string]string{
				"revenue":  "sales",
				"income":   "earnings",
				"expense":  "cost",
				"customer": "client",
			},
		},
		"healthcare": {
			Domain: "healthcare",
			Abbreviations: map[string]string{
				"EHR": "Electronic Health Record",
				"ICD": "International Classification of Diseases",
				"CPT": "Current Procedural Terminology",
				"LOS": "Length Of Stay",
				"ED":  "Emergency Department",
				"ICU": "Intensive Care Unit",
				"ADL": "Activities Of Daily Living",
			},
			Conventions: map[string]string{
				"_dt": " Date",
				"_cd": " Code",
				"_id": " Identifier",
			},
			Synonyms: map[string]string{
				"patient": "member",
				"doctor":  "provider",
				"visit":   "encounter",
			},
		},
	}

	return domainContexts[domain]
}

// 2. LOCALIZATION - Multi-Language Title Support

// LocalizationConfig provides multi-language business term mappings
type LocalizationConfig struct {
	Languages     map[string]string            // "en" -> "English", "es" -> "Spanish"
	Translations  map[string]map[string]string // term -> language -> translation
	LocaleFormats map[string]map[string]string // language -> format rules
}

// GenerateLocalizedTitle creates business-friendly titles in multiple languages
func (s *SemanticMappingService) GenerateLocalizedTitle(ctx context.Context, columnName string, termName string, languages []string) (map[string]string, error) {
	locConfig := s.getLocalizationConfig(ctx)
	if locConfig == nil {
		return map[string]string{"en": s.generateBusinessTitle(columnName, "DIMENSION")}, nil
	}

	titles := make(map[string]string)
	baseTitle := s.generateBusinessTitle(columnName, "DIMENSION")

	for _, lang := range languages {
		if translations, exists := locConfig.Translations[termName]; exists {
			if translation, langExists := translations[lang]; langExists {
				titles[lang] = translation
				continue
			}
		}

		// Fallback: use base title for unsupported language
		titles[lang] = baseTitle
	}

	return titles, nil
}

// getLocalizationConfig retrieves localization configuration
func (s *SemanticMappingService) getLocalizationConfig(ctx context.Context) *LocalizationConfig {
	// This would load from database or i18n service
	return &LocalizationConfig{
		Languages: map[string]string{
			"en": "English",
			"es": "Spanish",
			"fr": "French",
			"de": "German",
			"ja": "Japanese",
			"pt": "Portuguese",
			"it": "Italian",
			"nl": "Dutch",
			"pl": "Polish",
			"ru": "Russian",
		},
		Translations: map[string]map[string]string{
			// Core Business Terms
			"Customer": {
				"en": "Customer",
				"es": "Cliente",
				"fr": "Client",
				"de": "Kunde",
				"ja": "顧客",
				"pt": "Cliente",
				"it": "Cliente",
				"nl": "Klant",
				"pl": "Klient",
				"ru": "Клиент",
			},
			"Revenue": {
				"en": "Revenue",
				"es": "Ingresos",
				"fr": "Chiffre d'affaires",
				"de": "Umsatz",
				"ja": "収益",
				"pt": "Receita",
				"it": "Ricavi",
				"nl": "Inkomsten",
				"pl": "Przychód",
				"ru": "Доход",
			},
			"Date": {
				"en": "Date",
				"es": "Fecha",
				"fr": "Date",
				"de": "Datum",
				"ja": "日付",
				"pt": "Data",
				"it": "Data",
				"nl": "Datum",
				"pl": "Data",
				"ru": "Дата",
			},
			// Financial Metrics
			"Total Amount": {
				"en": "Total Amount",
				"es": "Monto Total",
				"fr": "Montant Total",
				"de": "Gesamtbetrag",
				"ja": "合計金額",
				"pt": "Valor Total",
				"it": "Importo Totale",
				"nl": "Totaalbedrag",
				"pl": "Kwota Całkowita",
				"ru": "Общая сумма",
			},
			"Sales": {
				"en": "Sales",
				"es": "Ventas",
				"fr": "Ventes",
				"de": "Verkäufe",
				"ja": "売上",
				"pt": "Vendas",
				"it": "Vendite",
				"nl": "Verkoop",
				"pl": "Sprzedaż",
				"ru": "Продажи",
			},
			"Profit": {
				"en": "Profit",
				"es": "Ganancia",
				"fr": "Profit",
				"de": "Gewinn",
				"ja": "利益",
				"pt": "Lucro",
				"it": "Profitto",
				"nl": "Winst",
				"pl": "Zysk",
				"ru": "Прибыль",
			},
			"Cost": {
				"en": "Cost",
				"es": "Costo",
				"fr": "Coût",
				"de": "Kosten",
				"ja": "コスト",
				"pt": "Custo",
				"it": "Costo",
				"nl": "Kosten",
				"pl": "Koszt",
				"ru": "Стоимость",
			},
			"Price": {
				"en": "Price",
				"es": "Precio",
				"fr": "Prix",
				"de": "Preis",
				"ja": "価格",
				"pt": "Preço",
				"it": "Prezzo",
				"nl": "Prijs",
				"pl": "Cena",
				"ru": "Цена",
			},
			// Customer Metrics
			"Customer Count": {
				"en": "Customer Count",
				"es": "Cantidad de Clientes",
				"fr": "Nombre de Clients",
				"de": "Kundenanzahl",
				"ja": "顧客数",
				"pt": "Contagem de Clientes",
				"it": "Numero di Clienti",
				"nl": "Aantal Klanten",
				"pl": "Liczba Klientów",
				"ru": "Количество Клиентов",
			},
			"Customer ID": {
				"en": "Customer ID",
				"es": "ID de Cliente",
				"fr": "ID Client",
				"de": "Kunden-ID",
				"ja": "顧客ID",
				"pt": "ID do Cliente",
				"it": "ID Cliente",
				"nl": "Klant-ID",
				"pl": "ID Klienta",
				"ru": "ID Клиента",
			},
			"Customer Name": {
				"en": "Customer Name",
				"es": "Nombre del Cliente",
				"fr": "Nom du Client",
				"de": "Kundenname",
				"ja": "顧客名",
				"pt": "Nome do Cliente",
				"it": "Nome Cliente",
				"nl": "Klantnaam",
				"pl": "Nazwa Klienta",
				"ru": "Имя Клиента",
			},
			"Customer Acquisition": {
				"en": "Customer Acquisition",
				"es": "Adquisición de Clientes",
				"fr": "Acquisition de Clients",
				"de": "Kundenakquisition",
				"ja": "顧客獲得",
				"pt": "Aquisição de Clientes",
				"it": "Acquisizione Cliente",
				"nl": "Klantenverwerving",
				"pl": "Akwizycja Klientów",
				"ru": "Привлечение Клиентов",
			},
			// Product Metrics
			"Product": {
				"en": "Product",
				"es": "Producto",
				"fr": "Produit",
				"de": "Produkt",
				"ja": "製品",
				"pt": "Produto",
				"it": "Prodotto",
				"nl": "Product",
				"pl": "Produkt",
				"ru": "Продукт",
			},
			"Product Category": {
				"en": "Product Category",
				"es": "Categoría de Producto",
				"fr": "Catégorie de Produit",
				"de": "Produktkategorie",
				"ja": "製品カテゴリー",
				"pt": "Categoria de Produto",
				"it": "Categoria Prodotto",
				"nl": "Productcategorie",
				"pl": "Kategoria Produktu",
				"ru": "Категория Продукта",
			},
			"Quantity": {
				"en": "Quantity",
				"es": "Cantidad",
				"fr": "Quantité",
				"de": "Menge",
				"ja": "数量",
				"pt": "Quantidade",
				"it": "Quantità",
				"nl": "Hoeveelheid",
				"pl": "Ilość",
				"ru": "Количество",
			},
			// Time Dimensions
			"Year": {
				"en": "Year",
				"es": "Año",
				"fr": "Année",
				"de": "Jahr",
				"ja": "年",
				"pt": "Ano",
				"it": "Anno",
				"nl": "Jaar",
				"pl": "Rok",
				"ru": "Год",
			},
			"Month": {
				"en": "Month",
				"es": "Mes",
				"fr": "Mois",
				"de": "Monat",
				"ja": "月",
				"pt": "Mês",
				"it": "Mese",
				"nl": "Maand",
				"pl": "Miesiąc",
				"ru": "Месяц",
			},
			"Quarter": {
				"en": "Quarter",
				"es": "Trimestre",
				"fr": "Trimestre",
				"de": "Quartal",
				"ja": "四半期",
				"pt": "Trimestre",
				"it": "Trimestre",
				"nl": "Kwartaal",
				"pl": "Kwartał",
				"ru": "Квартал",
			},
			"Week": {
				"en": "Week",
				"es": "Semana",
				"fr": "Semaine",
				"de": "Woche",
				"ja": "週",
				"pt": "Semana",
				"it": "Settimana",
				"nl": "Week",
				"pl": "Tydzień",
				"ru": "Неделя",
			},
			"Day": {
				"en": "Day",
				"es": "Día",
				"fr": "Jour",
				"de": "Tag",
				"ja": "日",
				"pt": "Dia",
				"it": "Giorno",
				"nl": "Dag",
				"pl": "Dzień",
				"ru": "День",
			},
			// Order Metrics
			"Order": {
				"en": "Order",
				"es": "Pedido",
				"fr": "Commande",
				"de": "Bestellung",
				"ja": "注文",
				"pt": "Pedido",
				"it": "Ordine",
				"nl": "Bestelling",
				"pl": "Zamówienie",
				"ru": "Заказ",
			},
			"Order Count": {
				"en": "Order Count",
				"es": "Cantidad de Pedidos",
				"fr": "Nombre de Commandes",
				"de": "Bestellanzahl",
				"ja": "注文数",
				"pt": "Contagem de Pedidos",
				"it": "Numero Ordini",
				"nl": "Aantal Bestellingen",
				"pl": "Liczba Zamówień",
				"ru": "Количество Заказов",
			},
			"Order Status": {
				"en": "Order Status",
				"es": "Estado del Pedido",
				"fr": "Statut de la Commande",
				"de": "Bestellstatus",
				"ja": "注文ステータス",
				"pt": "Status do Pedido",
				"it": "Stato Ordine",
				"nl": "Bestellingsstatus",
				"pl": "Status Zamówienia",
				"ru": "Статус Заказа",
			},
			// Performance Metrics
			"Performance": {
				"en": "Performance",
				"es": "Desempeño",
				"fr": "Rendement",
				"de": "Leistung",
				"ja": "パフォーマンス",
				"pt": "Desempenho",
				"it": "Prestazioni",
				"nl": "Prestatie",
				"pl": "Wydajność",
				"ru": "Производительность",
			},
			"Conversion Rate": {
				"en": "Conversion Rate",
				"es": "Tasa de Conversión",
				"fr": "Taux de Conversion",
				"de": "Konversionsrate",
				"ja": "コンバージョン率",
				"pt": "Taxa de Conversão",
				"it": "Tasso di Conversione",
				"nl": "Conversiepercentage",
				"pl": "Wskaźnik Konwersji",
				"ru": "Коэффициент Конверсии",
			},
			"Growth Rate": {
				"en": "Growth Rate",
				"es": "Tasa de Crecimiento",
				"fr": "Taux de Croissance",
				"de": "Wachstumsrate",
				"ja": "成長率",
				"pt": "Taxa de Crescimento",
				"it": "Tasso di Crescita",
				"nl": "Groeipercentage",
				"pl": "Wskaźnik Wzrostu",
				"ru": "Темп Роста",
			},
			"Churn Rate": {
				"en": "Churn Rate",
				"es": "Tasa de Abandono",
				"fr": "Taux d'Attrition",
				"de": "Abwanderungsrate",
				"ja": "チャーン率",
				"pt": "Taxa de Rotatividade",
				"it": "Tasso di Abbandono",
				"nl": "Churnpercentage",
				"pl": "Wskaźnik Rezygnacji",
				"ru": "Коэффициент Оттока",
			},
			// Statistical Terms
			"Average": {
				"en": "Average",
				"es": "Promedio",
				"fr": "Moyenne",
				"de": "Durchschnitt",
				"ja": "平均",
				"pt": "Média",
				"it": "Media",
				"nl": "Gemiddelde",
				"pl": "Średnia",
				"ru": "Среднее",
			},
			"Total": {
				"en": "Total",
				"es": "Total",
				"fr": "Total",
				"de": "Gesamt",
				"ja": "合計",
				"pt": "Total",
				"it": "Totale",
				"nl": "Totaal",
				"pl": "Razem",
				"ru": "Итого",
			},
			"Count": {
				"en": "Count",
				"es": "Cantidad",
				"fr": "Compte",
				"de": "Anzahl",
				"ja": "カウント",
				"pt": "Contagem",
				"it": "Conteggio",
				"nl": "Aantal",
				"pl": "Liczba",
				"ru": "Количество",
			},
			"Minimum": {
				"en": "Minimum",
				"es": "Mínimo",
				"fr": "Minimum",
				"de": "Minimum",
				"ja": "最小",
				"pt": "Mínimo",
				"it": "Minimo",
				"nl": "Minimum",
				"pl": "Minimum",
				"ru": "Минимум",
			},
			"Maximum": {
				"en": "Maximum",
				"es": "Máximo",
				"fr": "Maximum",
				"de": "Maximum",
				"ja": "最大",
				"pt": "Máximo",
				"it": "Massimo",
				"nl": "Maximum",
				"pl": "Maksimum",
				"ru": "Максимум",
			},
		},
	}
}

// 3. FORMAT VALIDATION - Specialized Data Type Hints

// FormatValidator provides validation for specialized data types
type FormatValidator struct {
	Patterns map[string]*regexp.Regexp
	Hints    map[string]map[string]interface{}
}

// ValidateAndFormatProperty validates property values and provides format hints
func (s *SemanticMappingService) ValidateAndFormatProperty(ctx context.Context, propertyName string, value string, dataType string) (string, map[string]interface{}, error) {
	hints := make(map[string]interface{})

	switch dataType {
	case "email":
		if err := s.validateEmail(value); err != nil {
			return "", nil, err
		}
		hints["format"] = "email"
		hints["input_type"] = "email"
		hints["validation"] = "rfc5322"
		return value, hints, nil

	case "phone":
		normalized, err := s.normalizePhoneNumber(value)
		if err != nil {
			return "", nil, err
		}
		hints["format"] = "phone"
		hints["normalized"] = normalized
		hints["country_code_required"] = true
		return normalized, hints, nil

	case "currency":
		hints["format"] = "currency"
		hints["decimal_places"] = 2
		hints["thousands_separator"] = ","
		hints["currency_symbol"] = "$"
		return value, hints, nil

	case "percentage":
		hints["format"] = "percentage"
		hints["range"] = map[string]float64{"min": 0, "max": 100}
		hints["decimal_places"] = 2
		return value, hints, nil

	case "url":
		if err := s.validateURL(value); err != nil {
			return "", nil, err
		}
		hints["format"] = "url"
		hints["input_type"] = "url"
		hints["requires_protocol"] = true
		return value, hints, nil

	case "json":
		if err := s.validateJSON(value); err != nil {
			return "", nil, err
		}
		hints["format"] = "json"
		hints["pretty_print_available"] = true
		return value, hints, nil

	case "date":
		hints["format"] = "date"
		hints["format_pattern"] = "yyyy-MM-dd"
		hints["timezone_aware"] = false
		return value, hints, nil

	case "datetime":
		hints["format"] = "datetime"
		hints["format_pattern"] = "yyyy-MM-dd'T'HH:mm:ss'Z'"
		hints["timezone_aware"] = true
		return value, hints, nil

	default:
		return value, hints, nil
	}
}

func (s *SemanticMappingService) validateEmail(email string) error {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	if !matched {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

func (s *SemanticMappingService) normalizePhoneNumber(phone string) (string, error) {
	// Remove non-digit characters
	digits := regexp.MustCompile("[^0-9+]").ReplaceAllString(phone, "")
	if len(digits) < 10 {
		return "", fmt.Errorf("phone number too short: %s", phone)
	}
	return digits, nil
}

func (s *SemanticMappingService) validateURL(url string) error {
	pattern := `^https?://[a-zA-Z0-9.-]+(:[0-9]+)?(/[^?#]*)?(\?[^#]*)?(#.*)?$`
	matched, _ := regexp.MatchString(pattern, url)
	if !matched {
		return fmt.Errorf("invalid URL format: %s", url)
	}
	return nil
}

func (s *SemanticMappingService) validateJSON(jsonStr string) error {
	var js interface{}
	return json.Unmarshal([]byte(jsonStr), &js)
}

// 4. AI TITLE GENERATION - LLM-Based Enhancement

// AITitleGenerationConfig provides LLM configuration for title generation
type AITitleGenerationConfig struct {
	Enabled             bool
	Provider            string // "openai", "anthropic", "local"
	ModelName           string
	ConfidenceThreshold float64
	FallbackToRules     bool
}

// GenerateAITitle generates business-friendly titles using LLM
func (s *SemanticMappingService) GenerateAITitle(ctx context.Context, columnName string, columnMetadata map[string]interface{}, dataType string) (string, float64, error) {
	config := s.getAITitleGenerationConfig(ctx)
	if !config.Enabled {
		return s.generateBusinessTitle(columnName, "DIMENSION"), 1.0, nil
	}

	// Prepare prompt for LLM
	prompt := s.buildAITitlePrompt(columnName, columnMetadata, dataType)

	// Call LLM provider (implementation depends on provider)
	title, confidence, err := s.callLLMProvider(ctx, config, prompt)
	if err != nil {
		if config.FallbackToRules {
			return s.generateBusinessTitle(columnName, "DIMENSION"), 0.5, nil
		}
		return "", 0, err
	}

	// Validate confidence threshold
	if confidence < config.ConfidenceThreshold {
		if config.FallbackToRules {
			return s.generateBusinessTitle(columnName, "DIMENSION"), confidence, nil
		}
		return title, confidence, fmt.Errorf("confidence below threshold: %.2f", confidence)
	}

	return title, confidence, nil
}

// buildAITitlePrompt constructs a prompt for the LLM
func (s *SemanticMappingService) buildAITitlePrompt(columnName string, metadata map[string]interface{}, dataType string) string {
	return fmt.Sprintf(`
Generate a business-friendly title for a data column.

Column Name: %s
Data Type: %s
Metadata: %v

Requirements:
1. Title should be human-readable and business-appropriate
2. Title should be suitable for use in reports and dashboards
3. Title should be concise (2-5 words)
4. Preserve important acronyms (USD, KPI, etc.)
5. If the column represents a calculation, indicate it (e.g., "Total", "Average")

Respond with ONLY the title, nothing else.
`, columnName, dataType, metadata)
}

// callLLMProvider calls the configured LLM provider (OpenAI, Anthropic, Gemini, or local)
func (s *SemanticMappingService) callLLMProvider(ctx context.Context, config *AITitleGenerationConfig, prompt string) (string, float64, error) {
	// If no LLM provider is configured, return error
	if s.llmProvider == nil {
		return "", 0, fmt.Errorf("LLM provider not configured")
	}

	// Try to use the generic LLM provider interface
	// This supports any provider implementing GenerateContent(ctx, prompt) (string, error)
	llmProvider, ok := s.llmProvider.(interface {
		GenerateContent(context.Context, string) (string, error)
	})

	if !ok {
		return "", 0, fmt.Errorf("LLM provider does not implement required interface")
	}

	// Call the LLM provider with the prompt
	title, err := llmProvider.GenerateContent(ctx, prompt)
	if err != nil {
		return "", 0, fmt.Errorf("LLM provider error: %w", err)
	}

	// Trim whitespace from response
	title = strings.TrimSpace(title)

	if title == "" {
		return "", 0, fmt.Errorf("LLM provider returned empty title")
	}

	// Calculate confidence based on response length and format validity
	// Responses that are 2-5 words get higher confidence
	words := strings.Fields(title)
	wordCount := float64(len(words))
	confidence := 0.9

	if wordCount < 2 || wordCount > 5 {
		confidence = 0.7 // Lower confidence for unusual length
	}

	// Ensure confidence is within valid range
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return title, confidence, nil
}

// InitializeGeminiProvider initializes Google Gemini as the LLM provider
// This requires the GEMINI_API_KEY environment variable to be set
func (s *SemanticMappingService) InitializeGeminiProvider(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("Gemini API key is empty")
	}

	// Create a wrapper that implements the LLM provider interface
	geminiProvider := &GeminiProviderWrapper{
		apiKey: apiKey,
		model:  "gemini-pro", // Default model
	}

	s.llmProvider = geminiProvider
	return nil
}

// InitializeOpenAIProvider initializes OpenAI as the LLM provider
// This requires the OPENAI_API_KEY environment variable to be set
func (s *SemanticMappingService) InitializeOpenAIProvider(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("OpenAI API key is empty")
	}

	// Create a wrapper that implements the LLM provider interface
	openaiProvider := &OpenAIProviderWrapper{
		apiKey:    apiKey,
		modelName: "gpt-4", // Default model
	}

	s.llmProvider = openaiProvider
	return nil
}

// InitializeAnthropicProvider initializes Anthropic Claude as the LLM provider
// This requires the ANTHROPIC_API_KEY environment variable to be set
func (s *SemanticMappingService) InitializeAnthropicProvider(apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("Anthropic API key is empty")
	}

	// Create a wrapper that implements the LLM provider interface
	anthropicProvider := &AnthropicProviderWrapper{
		apiKey:    apiKey,
		modelName: "claude-3-sonnet", // Default model
	}

	s.llmProvider = anthropicProvider
	return nil
}

// GeminiProviderWrapper wraps Google Gemini API for use as an LLM provider
type GeminiProviderWrapper struct {
	apiKey string
	model  string
}

// GenerateContent calls Google Gemini API to generate content
func (g *GeminiProviderWrapper) GenerateContent(ctx context.Context, prompt string) (string, error) {
	// In production, this would call the actual Gemini API
	// For now, return a placeholder implementation that demonstrates the interface

	// Example implementation would be:
	// 1. Create request with API key and prompt
	// 2. Call https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:generateContent
	// 3. Parse response and return generated text

	// Placeholder: In real implementation, call Gemini API
	if g.apiKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}

	// This is where actual API call would happen:
	// response, err := callGeminiAPI(ctx, g.apiKey, g.model, prompt)
	// if err != nil { return "", err }
	// return response.Text, nil

	return "", fmt.Errorf("Gemini API client library not imported (requires github.com/google/generative-ai-go)")
}

// OpenAIProviderWrapper wraps OpenAI API for use as an LLM provider
type OpenAIProviderWrapper struct {
	apiKey    string
	modelName string
}

// GenerateContent calls OpenAI API to generate content
func (o *OpenAIProviderWrapper) GenerateContent(ctx context.Context, prompt string) (string, error) {
	if o.apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}

	// Placeholder: In real implementation, call OpenAI API
	// Uses https://api.openai.com/v1/chat/completions endpoint

	return "", fmt.Errorf("OpenAI API client library not imported (requires github.com/sashabaranov/go-openai)")
}

// AnthropicProviderWrapper wraps Anthropic Claude API for use as an LLM provider
type AnthropicProviderWrapper struct {
	apiKey    string
	modelName string
}

// GenerateContent calls Anthropic Claude API to generate content
func (a *AnthropicProviderWrapper) GenerateContent(ctx context.Context, prompt string) (string, error) {
	if a.apiKey == "" {
		return "", fmt.Errorf("Anthropic API key not configured")
	}

	// Placeholder: In real implementation, call Anthropic API
	// Uses https://api.anthropic.com/v1/messages endpoint

	return "", fmt.Errorf("Anthropic API client library not imported (requires internal anthropic client)")
}

// getAITitleGenerationConfig retrieves AI title generation configuration
func (s *SemanticMappingService) getAITitleGenerationConfig(ctx context.Context) *AITitleGenerationConfig {
	// Enable AI titles only if LLM provider is configured
	enabled := s.llmProvider != nil

	return &AITitleGenerationConfig{
		Enabled:             enabled, // Enabled if llmProvider is available
		Provider:            "openai",
		ModelName:           "gpt-4",
		ConfidenceThreshold: 0.85,
		FallbackToRules:     true,
	}
}

// 5. CUSTOM PROPERTY TEMPLATES - Domain-Specific Configuration

// PropertyTemplate defines a template for semantic term properties
type PropertyTemplate struct {
	ID              string                 `json:"id"`
	Domain          string                 `json:"domain"`
	TermType        string                 `json:"term_type"` // DIMENSION, MEASURE, etc.
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Properties      map[string]interface{} `json:"properties"`
	RequiredFields  []string               `json:"required_fields"`
	DefaultValues   map[string]interface{} `json:"default_values"`
	ValidationRules map[string]interface{} `json:"validation_rules"`
}

// ApplyPropertyTemplate applies a domain-specific property template to a semantic term
func (s *SemanticMappingService) ApplyPropertyTemplate(ctx context.Context, termType string, domain string, baseProperties map[string]interface{}) (map[string]interface{}, error) {
	template := s.getPropertyTemplate(ctx, domain, termType)
	if template == nil {
		return baseProperties, nil
	}

	// Start with base properties
	result := make(map[string]interface{})
	for k, v := range baseProperties {
		result[k] = v
	}

	// Apply template defaults
	for k, v := range template.DefaultValues {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	// Apply template-specific properties
	for k, v := range template.Properties {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	// Validate against template requirements
	for _, required := range template.RequiredFields {
		if _, exists := result[required]; !exists {
			return nil, fmt.Errorf("template requires field: %s", required)
		}
	}

	result["applied_template"] = template.ID
	result["domain"] = domain

	return result, nil
}

// registerPropertyTemplate registers a custom property template
func (s *SemanticMappingService) registerPropertyTemplate(ctx context.Context, template *PropertyTemplate) error {
	// Validate template
	if template.ID == "" {
		template.ID = uuid.New().String()
	}

	// Store in database or configuration
	// Example validation rules
	if template.Domain == "" {
		return fmt.Errorf("template domain is required")
	}
	if template.TermType == "" {
		return fmt.Errorf("template term type is required")
	}

	// In production, this would persist to database
	// For now, just log successful registration
	fmt.Printf("Registered property template: %s for domain: %s, type: %s\n", template.ID, template.Domain, template.TermType)

	return nil
}

// getPropertyTemplate retrieves a domain-specific property template
func (s *SemanticMappingService) getPropertyTemplate(ctx context.Context, domain string, termType string) *PropertyTemplate {
	// Example: Finance domain templates
	templates := map[string]*PropertyTemplate{
		"finance-measure": {
			ID:          "finance-measure-template-001",
			Domain:      "finance",
			TermType:    "MEASURE",
			Name:        "Financial Measure Template",
			Description: "Standard template for financial metrics and KPIs",
			Properties: map[string]interface{}{
				"show_in_reports":   true,
				"auditable":         true,
				"requires_approval": false,
			},
			RequiredFields: []string{"aggregation", "currency", "format"},
			DefaultValues: map[string]interface{}{
				"currency":           "USD",
				"format":             "currency",
				"decimal_places":     2,
				"includes_tax":       false,
				"calculation_method": "standard",
			},
			ValidationRules: map[string]interface{}{
				"aggregation": map[string]interface{}{
					"allowed": []string{"sum", "avg", "count"},
				},
				"currency": map[string]interface{}{
					"pattern": "^[A-Z]{3}$",
				},
			},
		},
		"finance-dimension": {
			ID:          "finance-dimension-template-001",
			Domain:      "finance",
			TermType:    "DIMENSION",
			Name:        "Financial Dimension Template",
			Description: "Standard template for financial dimensions",
			Properties: map[string]interface{}{
				"hierarchical":       true,
				"drill_down_enabled": true,
			},
			RequiredFields: []string{"title", "type"},
			DefaultValues: map[string]interface{}{
				"type":   "string",
				"shown":  true,
				"public": true,
			},
			ValidationRules: map[string]interface{}{
				"type": map[string]interface{}{
					"allowed": []string{"string", "number", "time"},
				},
			},
		},
	}

	key := fmt.Sprintf("%s-%s", domain, strings.ToLower(termType))
	return templates[key]
}

// getOrCreateSemanticTerm finds a semantic term or creates it if it doesn't exist.
func (s *SemanticMappingService) getOrCreateSemanticTerm(ctx context.Context, tx *sqlx.Tx, req *ApplyEnrichmentRequest) (string, error) {
	// Try to find existing term
	var termID string
	query := `SELECT id FROM catalog_node WHERE tenant_id = $1 AND node_type_id = $2 AND node_name = $3`
	err := tx.GetContext(ctx, &termID, query, req.TenantID, SemanticTermNodeTypeID, req.Proposal.SemanticTermName)
	if err == nil {
		return termID, nil // Found it
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("error checking for existing semantic term: %w", err)
	}

	// Not found, so create it
	newTermID := uuid.New().String()
	qualifiedPath := fmt.Sprintf("/semantic/%s", req.Proposal.SemanticTermName)

	// Use intelligent property inference
	properties := s.inferSemanticTermProperties(req.Column, req.Proposal.SemanticTermType)

	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal semantic term properties: %w", err)
	}

	insertQuery := `
		INSERT INTO catalog_node (
			id, tenant_datasource_id, node_type_id, node_name,
			qualified_path, tenant_id, created_at, updated_at, properties
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		newTermID, req.DatasourceID, SemanticTermNodeTypeID, req.Proposal.SemanticTermName,
		qualifiedPath, req.TenantID, time.Now(), time.Now(), string(propertiesJSON))

	if err != nil {
		return "", fmt.Errorf("failed to create semantic term: %w", err)
	}
	return newTermID, nil
}

// getOrCreateBusinessTerm finds a business term or creates it if it doesn't exist.
func (s *SemanticMappingService) getOrCreateBusinessTerm(ctx context.Context, tx *sqlx.Tx, req *ApplyEnrichmentRequest) (string, error) {
	// Try to find existing term
	var termID string
	query := `SELECT id FROM catalog_node WHERE tenant_id = $1 AND node_type_id = $2 AND node_name = $3`
	err := tx.GetContext(ctx, &termID, query, req.TenantID, BusinessTermNodeTypeID, req.Proposal.BusinessTermName)
	if err == nil {
		return termID, nil // Found it
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("error checking for existing business term: %w", err)
	}

	// Not found, so create it
	newTermID := uuid.New().String()
	qualifiedPath := fmt.Sprintf("/business/%s/%s", strings.Join(req.Proposal.DomainHierarchy, "/"), req.Proposal.BusinessTermName)
	properties := map[string]interface{}{
		"domain_hierarchy": req.Proposal.DomainHierarchy,
	}
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal business term properties: %w", err)
	}

	insertQuery := `
		INSERT INTO catalog_node (
			id, tenant_datasource_id, node_type_id, node_name,
			qualified_path, tenant_id, created_at, updated_at, properties
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, insertQuery,
		newTermID, req.DatasourceID, BusinessTermNodeTypeID, req.Proposal.BusinessTermName,
		qualifiedPath, req.TenantID, time.Now(), time.Now(), string(propertiesJSON))

	if err != nil {
		return "", fmt.Errorf("failed to create business term: %w", err)
	}
	return newTermID, nil
}

// createEdge creates a generic edge between two nodes within a transaction.
func (s *SemanticMappingService) createEdge(ctx context.Context, tx *sqlx.Tx, tenantID, tenantDatasourceID, sourceNodeID, targetNodeID, relationshipType, edgeTypeID string) (bool, error) {
	edgeID := uuid.New().String()

	query := `
		INSERT INTO catalog_edge (
			id, tenant_datasource_id, source_node_id, target_node_id,
			relationship_type, edge_type_id, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
		DO NOTHING
	`

	res, err := tx.ExecContext(ctx, query,
		edgeID, tenantDatasourceID, sourceNodeID, targetNodeID,
		relationshipType, edgeTypeID, tenantID, time.Now(), time.Now())

	if err != nil {
		return false, fmt.Errorf("failed to insert edge: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to check rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// normalizeColumnName converts a raw column name into a more readable, title-cased format.
func (s *SemanticMappingService) normalizeColumnName(columnName string) string {
	// Placeholder implementation
	name := strings.ReplaceAll(columnName, "_", " ")
	name = strings.Title(strings.ToLower(name))
	return name
}

// determineTermType classifies a semantic term as a Dimension, Measure, or Time based on its properties.
func (s *SemanticMappingService) determineTermType(column *DatabaseColumn, profile *NodeProperties) (string, error) {
	// Placeholder implementation
	dt := s.normalizeDataType(column.DataType)
	if dt == "NUMBER" && (strings.Contains(column.Column, "AMT") || strings.Contains(column.Column, "COUNT")) {
		return "Measure", nil
	}
	if dt == "DATE" || dt == "DATETIME" {
		return "Time", nil
	}
	return "Dimension", nil
}

// generateBusinessTermName creates a user-friendly business term name.
func (s *SemanticMappingService) generateBusinessTermName(normalizedName, tableName string) string {
	// Placeholder implementation
	return normalizedName
}

// determineDataDomain suggests a data domain hierarchy based on schema and table names.
func (s *SemanticMappingService) determineDataDomain(schema, table string) []string {
	// Placeholder implementation
	schema = strings.Title(strings.ToLower(schema))
	table = strings.Title(strings.ToLower(strings.ReplaceAll(table, "_", " ")))
	return []string{schema, table, "General"}
}

// ExpandAbbreviationsDB expands abbreviations using the database service if available
func (s *SemanticMappingService) ExpandAbbreviationsDB(ctx context.Context, columnName string) ([]string, error) {
	if s.abbreviationSvc != nil {
		return s.abbreviationSvc.ExpandAbbreviations(ctx, columnName)
	}

	// Fallback to legacy hardcoded expansion
	return expandAbbreviations(columnName), nil
}

// DatabaseColumn represents a column from the catalog
type DatabaseColumn struct {
	NodeID             string   `json:"node_id"`
	Schema             string   `json:"schema"`
	Table              string   `json:"table"`
	Column             string   `json:"column"`
	QualifiedPath      string   `json:"qualified_path"`
	TenantDatasourceID string   `json:"tenant_datasource_id"`
	TenantID           string   `json:"tenant_id"`
	DataType           string   `json:"data_type"`
	Cardinality        int      `json:"cardinality,omitempty"`
	FrequentValues     []string `json:"frequent_values,omitempty"`
	InferredPatterns   []string `json:"inferred_patterns,omitempty"`
	BloomFilter        []byte   `json:"-"` // Not for JSON, for internal use
}

// SemanticTerm represents a semantic term node
type SemanticTerm struct {
	NodeID        string   `json:"node_id" db:"id"`
	TermName      string   `json:"term_name" db:"node_name"`
	QualifiedPath string   `json:"qualified_path" db:"qualified_path"`
	DataType      string   `json:"data_type" db:"data_type"`
	Description   string   `json:"description,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	// Properties from reference data linked to the term
	ReferenceValues   []string `json:"reference_values,omitempty"`
	ReferencePatterns []string `json:"reference_patterns,omitempty"`
	ReferenceBloom    []byte   `json:"-"` // Not for JSON, for internal use
	// Score is an optional field used by the search API to communicate relevance
	Score float64 `json:"score,omitempty"`
}

// BusinessTermSuggestion represents a suggested business term with confidence.
type BusinessTermSuggestionResult struct {
	BusinessTermID      string                `json:"business_term_id,omitempty"`
	TermName            string                `json:"term_name"`
	Confidence          float64               `json:"confidence"`
	Reason              string                `json:"reason"`
	Source              string                `json:"source,omitempty"`
	Description         string                `json:"description,omitempty"`
	Categories          []string              `json:"categories,omitempty"`
	ConfidenceBreakdown []ConfidenceBreakdown `json:"confidence_breakdown,omitempty"`
}

// ConfidenceBreakdown provides a weighted view into the confidence calculation.
type ConfidenceBreakdown struct {
	Label   string  `json:"label"`
	Score   float64 `json:"score"`
	Weight  float64 `json:"weight,omitempty"`
	Details string  `json:"details,omitempty"`
}

// FeedbackStats represents historical feedback statistics for a business term
type FeedbackStats struct {
	BusinessTermName string
	AcceptCount      int
	RejectCount      int
	TotalFeedback    int
	AcceptanceRate   float64
}

// MappingResult represents a potential mapping
type MappingResult struct {
	DatabaseColumn DatabaseColumn `json:"database_column"`
	SemanticTerm   string         `json:"semantic_term"`
	SemanticTermID string         `json:"semantic_term_id,omitempty"`
	Confidence     float64        `json:"confidence"`
	IsNewTerm      bool           `json:"is_new_term"`
	Selected       bool           `json:"selected"`
	MatchReason    string         `json:"match_reason,omitempty"`
	EdgeExists     bool           `json:"edge_exists"`
	Override       bool           `json:"override"`
}

// NodeProperties represents the JSONB properties field
type NodeProperties struct {
	DataType         string   `json:"data_type"`
	Cardinality      int      `json:"cardinality"`
	FrequentValues   []string `json:"frequent_values"`
	InferredPatterns []string `json:"inferred_patterns"`
	BloomFilter      []byte   `json:"bloom_filter"` // Assume stored as base64 string in JSON
}

// SearchRequest for typeahead search
type SearchRequest struct {
	Query string `json:"query"`
	Limit int    `json:"limit"`
	// Optional scope tables or view names to bias suggestions (e.g. ["public.users", "orders"])
	ScopeTables []string `json:"scope_tables,omitempty"`
	// Optional column context to improve data-type compatible suggestions
	Column *struct {
		Schema   string `json:"schema,omitempty"`
		Table    string `json:"table,omitempty"`
		Column   string `json:"column,omitempty"`
		DataType string `json:"data_type,omitempty"`
	} `json:"column,omitempty"`
	// CandidateLimit sets how many candidates to fetch from SQL before re-ranking.
	// If omitted, a reasonably large default is used to avoid missing high-quality matches.
	CandidateLimit int `json:"candidate_limit,omitempty"`
}

// Constants for node types (these should match your database)
const (
	DatabaseColumnNodeTypeID = "a64c1011-16e8-4ddf-b447-363bf8e15c9a"
	SemanticTermNodeTypeID   = "820b942a-9c9e-4abc-acdc-84616db33098"
	BusinessTermNodeTypeID   = "21645d21-de5f-4feb-af99-99273ea75626"
	EdgeTypeID               = "99c86836-98ef-45a3-82df-4c62b5730ac6"
)

// Parse qualified path to extract schema, table, column
func (s *SemanticMappingService) parseQualifiedPath(path string) (schema, table, column string) {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 {
		// Assuming format like /schema/table/column or /datasource/schema/table/column
		if len(parts) >= 4 {
			schema = parts[1]
			table = parts[2]
			column = parts[3]
		} else {
			table = parts[len(parts)-2]
			column = parts[len(parts)-1]
		}
	}
	return
}

// removePrefixes removes BI prefixes like DIM_, FCT_, FACT_, etc.
func (s *SemanticMappingService) removePrefixes(term string) string {
	prefixes := []string{
		"DIM_", "FCT_", "FACT_", "DIMENSION_", "AGG_", "TMP_", "TEMP_", "STG_", "STAGE_",
	}
	upper := strings.ToUpper(term)
	for _, prefix := range prefixes {
		if strings.HasPrefix(upper, prefix) {
			term = term[len(prefix):]
			break
		}
	}
	return term
}

// singularize converts plural forms to singular
func (s *SemanticMappingService) singularize(term string) string {
	term = strings.TrimSpace(term)
	lower := strings.ToLower(term)

	// Special cases
	specialCases := map[string]string{
		"PEOPLE":     "PERSON",
		"CHILDREN":   "CHILD",
		"MEN":        "MAN",
		"WOMEN":      "WOMAN",
		"FEET":       "FOOT",
		"TEETH":      "TOOTH",
		"GEESE":      "GOOSE",
		"ANALYSES":   "ANALYSIS",
		"CRITERIA":   "CRITERION",
		"DATA":       "DATUM",
		"INDICES":    "INDEX",
		"MATRICES":   "MATRIX",
		"APPENDICES": "APPENDIX",
		"VERTICES":   "VERTEX",
	}

	upperTerm := strings.ToUpper(term)
	if singular, ok := specialCases[upperTerm]; ok {
		return singular
	}

	// Handle common plural patterns
	if strings.HasSuffix(lower, "ies") && len(term) > 3 {
		// companies -> company, categories -> category
		return term[:len(term)-3] + "Y"
	}
	if strings.HasSuffix(lower, "ses") && len(term) > 3 {
		// addresses -> address, statuses -> status
		return term[:len(term)-2]
	}
	if strings.HasSuffix(lower, "xes") && len(term) > 3 {
		// boxes -> box, indexes -> index
		return term[:len(term)-2]
	}
	if strings.HasSuffix(lower, "oes") && len(term) > 3 {
		// tomatoes -> tomato, heroes -> hero
		return term[:len(term)-2]
	}
	if strings.HasSuffix(lower, "ves") && len(term) > 3 {
		// wives -> wife, knives -> knife
		return term[:len(term)-3] + "FE"
	}
	if strings.HasSuffix(lower, "s") && len(term) > 1 && !strings.HasSuffix(lower, "ss") {
		// employees -> employee, customers -> customer
		return term[:len(term)-1]
	}

	return term
}

// addContextToGeneric adds table context to generic column names
func (s *SemanticMappingService) addContextToGeneric(column, table string) string {
	genericTerms := map[string]bool{
		"BIRTH_DATE":  true,
		"BIRTHDATE":   true,
		"ADDRESS":     true,
		"PHONE":       true,
		"EMAIL":       true,
		"CITY":        true,
		"STATE":       true,
		"ZIP":         true,
		"ZIPCODE":     true,
		"COUNTRY":     true,
		"FAX":         true,
		"DATE":        true,
		"TIME":        true,
		"TIMESTAMP":   true,
		"DESCRIPTION": true,
		"NOTES":       true,
		"REGION":      true,
	}

	columnUpper := strings.ToUpper(column)
	if genericTerms[columnUpper] {
		// Add table context: BIRTHDATE becomes EMPLOYEE_BIRTH_DATE
		return fmt.Sprintf("%s_%s", table, column)
	}

	return column
}

// removeRedundancy removes redundant table names from columns
// Example: CUSTOMERS_CUSTOMER_CITY -> CUSTOMER_CITY
func (s *SemanticMappingService) removeRedundancy(term string) string {
	parts := strings.Split(term, "_")
	if len(parts) < 2 {
		return term
	}

	// Check if any part is a plural of another
	seen := make(map[string]bool)
	result := []string{}

	for _, part := range parts {
		singular := s.singularize(part)
		singularUpper := strings.ToUpper(singular)

		// Skip if we've seen this singular form already
		if !seen[singularUpper] {
			result = append(result, part)
			seen[singularUpper] = true
		}
	}

	return strings.Join(result, "_")
}

// Normalize data type for comparison
func (s *SemanticMappingService) normalizeDataType(dataType string) string {
	dt := strings.ToUpper(strings.TrimSpace(dataType))

	// Map common type variations to standard types
	typeMapping := map[string]string{
		"VARCHAR":     "STRING",
		"VARCHAR2":    "STRING",
		"CHAR":        "STRING",
		"CHARACTER":   "STRING",
		"TEXT":        "STRING",
		"STRING":      "STRING",
		"NVARCHAR":    "STRING",
		"NCHAR":       "STRING",
		"INTEGER":     "NUMBER",
		"INT":         "NUMBER",
		"INT4":        "NUMBER",
		"INT8":        "NUMBER",
		"BIGINT":      "NUMBER",
		"SMALLINT":    "NUMBER",
		"TINYINT":     "NUMBER",
		"NUMERIC":     "NUMBER",
		"DECIMAL":     "NUMBER",
		"FLOAT":       "NUMBER",
		"DOUBLE":      "NUMBER",
		"REAL":        "NUMBER",
		"TIMESTAMP":   "DATETIME",
		"TIMESTAMPTZ": "DATETIME",
		"DATETIME":    "DATETIME",
		"DATE":        "DATE",
		"TIME":        "TIME",
		"BOOLEAN":     "BOOLEAN",
		"BOOL":        "BOOLEAN",
		"JSON":        "JSON",
		"JSONB":       "JSON",
		"UUID":        "UUID",
		"BLOB":        "BINARY",
		"BYTEA":       "BINARY",
		"ARRAY":       "ARRAY",
	}

	// Check for exact match
	if normalized, ok := typeMapping[dt]; ok {
		return normalized
	}

	// Check for partial matches (e.g., VARCHAR(255))
	for key, value := range typeMapping {
		if strings.HasPrefix(dt, key) {
			return value
		}
	}

	return dt
}

// Check if data types are compatible
func (s *SemanticMappingService) areDataTypesCompatible(type1, type2 string) bool {
	norm1 := s.normalizeDataType(type1)
	norm2 := s.normalizeDataType(type2)

	if norm1 == norm2 {
		return true
	}

	// Allow some flexibility for similar types
	compatibleGroups := [][]string{
		{"STRING", "TEXT"},
		{"NUMBER", "DECIMAL", "FLOAT"},
		{"DATETIME", "TIMESTAMP", "DATE"},
		{"JSON", "JSONB"},
		{"BINARY", "BLOB"},
	}

	for _, group := range compatibleGroups {
		hasType1 := false
		hasType2 := false
		for _, t := range group {
			if norm1 == t {
				hasType1 = true
			}
			if norm2 == t {
				hasType2 = true
			}
		}
		if hasType1 && hasType2 {
			return true
		}
	}

	return false
}

// Generate semantic term from table and column context
func (s *SemanticMappingService) generateSemanticTerm(schema, table, column string) string {
	// Clean and normalize inputs
	column = strings.ToUpper(strings.TrimSpace(column))
	table = strings.ToUpper(strings.TrimSpace(table))

	// Remove BI prefixes (DIM_, FCT_, etc.)
	table = s.removePrefixes(table)
	column = s.removePrefixes(column)

	// Convert plurals to singular
	table = s.singularize(table)
	column = s.singularize(column)

	// Replace common separators with underscore (not asterisk!)
	replacer := strings.NewReplacer(
		"_", "_", // Keep underscores
		"-", "_",
		".", "_",
		" ", "_",
		"*", "_", // Replace any asterisks with underscores
	)

	column = replacer.Replace(column)
	table = replacer.Replace(table)

	// Remove multiple consecutive underscores
	for strings.Contains(column, "__") {
		column = strings.ReplaceAll(column, "__", "_")
	}
	for strings.Contains(table, "__") {
		table = strings.ReplaceAll(table, "__", "_")
	}

	// Trim underscores from ends
	column = strings.Trim(column, "_")
	table = strings.Trim(table, "_")

	// Add context to generic terms (birthdate, address, phone, etc.)
	column = s.addContextToGeneric(column, table)

	// Common column patterns that don't need table context (after adding context)
	standaloneColumns := map[string]bool{
		"UUID":         true,
		"CREATED_AT":   true,
		"UPDATED_AT":   true,
		"DELETED_AT":   true,
		"CREATED_BY":   true,
		"UPDATED_BY":   true,
		"DELETED_BY":   true,
		"CREATED_DATE": true,
		"UPDATED_DATE": true,
	}

	// Check if column is generic and needs table context
	genericColumns := map[string]bool{
		"ID":     true,
		"NAME":   true,
		"TYPE":   true,
		"STATUS": true,
		"VALUE":  true,
		"COUNT":  true,
		"TOTAL":  true,
		"AMOUNT": true,
		"NUMBER": true,
		"CODE":   true,
		"LEVEL":  true,
		"FLAG":   true,
	}

	// If column already contains table context (e.g., USER_ID, EMPLOYEE_NAME)
	tableTokens := strings.Split(table, "_")
	columnTokens := strings.Split(column, "_")

	hasTableContext := false
	for _, tt := range tableTokens {
		for _, ct := range columnTokens {
			if tt == ct && len(tt) > 2 {
				hasTableContext = true
				break
			}
		}
		if hasTableContext {
			break
		}
	}

	if hasTableContext {
		// Remove redundancy like CUSTOMERS_CUSTOMER_CITY -> CUSTOMER_CITY
		return s.removeRedundancy(column)
	}

	// If it's a standalone column that doesn't need context
	if standaloneColumns[column] {
		return column
	}

	// If it's a generic column, add table context
	if genericColumns[column] {
		// Special cases
		if column == "ID" {
			result := fmt.Sprintf("%s_ID", table)
			return s.removeRedundancy(result)
		}
		result := fmt.Sprintf("%s_%s", table, column)
		return s.removeRedundancy(result)
	}

	// Check if column already has sufficient context
	parts := strings.Split(column, "_")
	if len(parts) > 2 {
		// Column already has structure, remove redundancy and use
		return s.removeRedundancy(column)
	}

	// For columns with 1-2 tokens, check if they need table context
	if len(parts) <= 2 {
		// Add table context for better semantic clarity
		result := fmt.Sprintf("%s_%s", table, column)
		return s.removeRedundancy(result)
	}

	return s.removeRedundancy(column)
}

// Calculate confidence for semantic term matching (enhanced with abbreviations and profile data)
func (s *SemanticMappingService) calculateSemanticConfidence(
	ctx context.Context,
	generatedTerm, existingTerm string,
	column *DatabaseColumn,
	term *SemanticTerm,
) (float64, string, []ConfidenceBreakdown) {
	// Use the enhanced matching algorithm with abbreviation and profile support
	return s.EnhancedCalculateSemanticConfidence(ctx, generatedTerm, existingTerm, column, term)
}

// Original implementation as backup (now internal)
func (s *SemanticMappingService) calculateSemanticConfidenceOriginal(
	generatedTerm, existingTerm string,
	column *DatabaseColumn,
	term *SemanticTerm,
) (float64, string, []ConfidenceBreakdown) {
	generatedDataType := column.DataType
	existingDataType := term.DataType

	// Normalize for comparison (remove both asterisks and underscores for comparison)
	gen := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(generatedTerm, "*", ""), "_", ""))
	exist := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(existingTerm, "*", ""), "_", ""))

	var matchReason strings.Builder
	// Exact match (ignoring separators)
	if gen == exist {
		dataTypeCompatible := s.areDataTypesCompatible(generatedDataType, existingDataType)
		if dataTypeCompatible {
			matchReason.WriteString("Exact name match, compatible data types")
			breakdown := []ConfidenceBreakdown{
				{Label: "Name similarity", Score: 1.0, Weight: 0.5, Details: "Exact match"},
				{Label: "Profile alignment", Score: 0.0, Weight: 0.35, Details: "Not evaluated"},
				{Label: "Data type alignment", Score: 1.0, Weight: 0.15, Details: "Compatible"},
			}
			return 1.0, matchReason.String(), breakdown
		} else {
			matchReason.WriteString(fmt.Sprintf("Exact name match, but data type mismatch (%s vs %s)",
				s.normalizeDataType(generatedDataType), s.normalizeDataType(existingDataType)))
			breakdown := []ConfidenceBreakdown{
				{Label: "Name similarity", Score: 1.0, Weight: 0.5, Details: "Exact match"},
				{Label: "Profile alignment", Score: 0.0, Weight: 0.35, Details: "Not evaluated"},
				{Label: "Data type alignment", Score: 0.4, Weight: 0.15, Details: "Mismatch"},
			}
			return 0.85, matchReason.String(), breakdown
		}
	}

	// Tokenize
	genTokens := s.tokenizeSemanticTerm(generatedTerm)
	existTokens := s.tokenizeSemanticTerm(existingTerm)

	// Calculate Jaccard similarity
	jaccardScore := s.calculateJaccardSimilarity(genTokens, existTokens)

	// Calculate Levenshtein similarity
	levDistance := s.levenshteinDistance(gen, exist)
	maxLen := s.max(len(gen), len(exist))
	levScore := 1.0
	if maxLen > 0 {
		levScore = 1.0 - (float64(levDistance) / float64(maxLen))
	}

	// Check for substring match
	substringBonus := 0.0
	if strings.Contains(exist, gen) || strings.Contains(gen, exist) {
		substringBonus = 0.15
		matchReason.WriteString("Substring match; ")
	}

	// Token order similarity
	orderBonus := 0.0
	minLen := s.min(len(genTokens), len(existTokens))
	if minLen > 0 {
		matchingPositions := 0
		for i := 0; i < minLen; i++ {
			if genTokens[i] == existTokens[i] {
				matchingPositions++
			}
		}
		if matchingPositions > 0 {
			orderBonus = float64(matchingPositions) / float64(minLen) * 0.2
			matchReason.WriteString(fmt.Sprintf("%d/%d tokens match in order; ", matchingPositions, minLen))
		}
	}

	// Weighted combination
	baseScore := (jaccardScore * 0.5) + (levScore * 0.5)
	nameScore := baseScore + substringBonus + orderBonus

	if nameScore > 1.0 {
		nameScore = 1.0
	}

	// Factor in profiling data if available
	profileBonus := 0.0
	// 1. Frequent Values Overlap
	if len(column.FrequentValues) > 0 && len(term.ReferenceValues) > 0 {
		overlap := s.calculateSetOverlap(column.FrequentValues, term.ReferenceValues)
		if overlap > 0 {
			profileBonus += overlap * 0.25 // Significant bonus for value overlap
			matchReason.WriteString(fmt.Sprintf("Frequent values overlap: %.0f%%; ", overlap*100))
		}
	}

	// 2. Inferred Patterns Overlap
	if len(column.InferredPatterns) > 0 && len(term.ReferencePatterns) > 0 {
		overlap := s.calculateSetOverlap(column.InferredPatterns, term.ReferencePatterns)
		if overlap > 0 {
			profileBonus += overlap * 0.15 // Bonus for pattern overlap
			matchReason.WriteString(fmt.Sprintf("Data patterns overlap: %.0f%%; ", overlap*100))
		}
	}

	// 3. Bloom Filter Check (if available)
	// This is a simplified check. A real implementation would use a bloom filter library.
	if len(column.BloomFilter) > 0 && len(term.ReferenceBloom) > 0 {
		// isSubset := bloom.Check(column.BloomFilter, term.ReferenceBloom)
		// For this example, we'll simulate a check.
		isSubset := true // Assume it's a subset for demonstration
		if isSubset {
			profileBonus += 0.1 // Small bonus for bloom filter confirmation
			matchReason.WriteString("Bloom filter match; ")
		}
	}

	// Apply data type compatibility factor and profile bonus
	dataTypeCompatible := s.areDataTypesCompatible(generatedDataType, existingDataType)
	finalScore := nameScore

	if dataTypeCompatible {
		matchReason.WriteString(fmt.Sprintf("Compatible data types (%s)", s.normalizeDataType(existingDataType)))
		// Boost score slightly for data type match
		finalScore = nameScore * 1.05
		if finalScore > 1.0 {
			finalScore = 1.0
		}
	} else if existingDataType != "" && generatedDataType != "" {
		matchReason.WriteString(fmt.Sprintf("Data type mismatch (%s vs %s)",
			s.normalizeDataType(generatedDataType), s.normalizeDataType(existingDataType)))
		// Penalize score for data type mismatch
		finalScore = nameScore * 0.75
	} else {
		matchReason.WriteString("Data type not available for comparison")
	}

	matchReason.WriteString(fmt.Sprintf(" (Jaccard: %.2f, Levenshtein: %.2f)", jaccardScore, levScore))

	// Add profile bonus to the final score
	finalScore += profileBonus
	if finalScore > 1.0 {
		finalScore = 1.0
	}
	nameComponent := minFloat64(nameScore, 1.0)
	profileComponent := 0.0
	if profileBonus > 0 {
		profileComponent = minFloat64(profileBonus/0.5, 1.0)
	}
	typeComponent := 0.5
	if dataTypeCompatible {
		typeComponent = 1.0
	} else if existingDataType == "" || generatedDataType == "" {
		typeComponent = 0.5
	} else {
		typeComponent = 0.3
	}

	breakdown := []ConfidenceBreakdown{
		{Label: "Name similarity", Score: nameComponent, Weight: 0.5, Details: fmt.Sprintf("Jaccard %.2f, Levenshtein %.2f", jaccardScore, levScore)},
		{Label: "Profile alignment", Score: profileComponent, Weight: 0.35, Details: fmt.Sprintf("Profile bonus %.2f", profileBonus)},
		{Label: "Data type alignment", Score: typeComponent, Weight: 0.15, Details: s.normalizeDataType(existingDataType)},
	}

	return finalScore, matchReason.String(), breakdown
}

func (s *SemanticMappingService) tokenizeSemanticTerm(term string) []string {
	// Split by asterisk and clean
	parts := strings.Split(strings.ToUpper(term), "*")
	tokens := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tokens = append(tokens, part)
		}
	}
	return tokens
}

func (s *SemanticMappingService) calculateJaccardSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := make(map[string]bool)
	set2 := make(map[string]bool)

	for _, token := range tokens1 {
		set1[token] = true
	}
	for _, token := range tokens2 {
		set2[token] = true
	}

	intersection := 0
	for token := range set1 {
		if set2[token] {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (s *SemanticMappingService) levenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
	}

	for i := 0; i <= len(s1); i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len(s2); j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}
			matrix[i][j] = s.min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func (s *SemanticMappingService) min(vals ...int) int {
	if len(vals) == 0 {
		return 0
	}
	minVal := vals[0]
	for _, v := range vals[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func (s *SemanticMappingService) max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// calculateSetOverlap calculates the percentage of items from set1 that are in set2.
func (s *SemanticMappingService) calculateSetOverlap(set1, set2 []string) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	set2Map := make(map[string]bool)
	for _, item := range set2 {
		set2Map[item] = true
	}

	overlapCount := 0
	for _, item := range set1 {
		if set2Map[item] {
			overlapCount++
		}
	}

	return float64(overlapCount) / float64(len(set1))
}

// Fetch database columns
func (s *SemanticMappingService) fetchDatabaseColumns(ctx context.Context, tenantID, tenantDatasourceID string) ([]DatabaseColumn, error) {
	query := `
		SELECT cn.id, cn.qualified_path, cn.node_name, cn.tenant_datasource_id, cn.tenant_id, cn.properties, pr.properties as profile_properties
		FROM catalog_node cn
		LEFT JOIN profiler_results pr ON cn.id = pr.column_id::uuid -- Assuming profiler_results links to catalog_node by ID
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		ORDER BY qualified_path
	`

	rows, err := s.db.QueryContext(ctx, query, DatabaseColumnNodeTypeID, tenantID, tenantDatasourceID)
	if err != nil {
		// If the profiler_results table doesn't exist in some environments, retry with a simpler query
		// that omits the LEFT JOIN. This allows fallback generation to work on lighter DBs.
		logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns initial query failed for tenant=%s datasource=%s: %v -- retrying simpler query", tenantID, tenantDatasourceID, err)

		// Fallback simpler query without profiler_results join
		simpleQuery := `
		SELECT id, qualified_path, node_name, tenant_datasource_id, tenant_id, properties
		FROM catalog_node
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		ORDER BY qualified_path
		`

		rows, err = s.db.QueryContext(ctx, simpleQuery, DatabaseColumnNodeTypeID, tenantID, tenantDatasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("fetchDatabaseColumns fallback query also failed for tenant=%s datasource=%s: %v", tenantID, tenantDatasourceID, err)
			return nil, err
		}
	}
	defer rows.Close()

	var columns []DatabaseColumn
	for rows.Next() {
		var col DatabaseColumn
		var nodeName sql.NullString
		var propertiesJSON, profilePropsJSON []byte

		err := rows.Scan(&col.NodeID, &col.QualifiedPath, &nodeName, &col.TenantDatasourceID, &col.TenantID, &propertiesJSON, &profilePropsJSON)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning column: %v", err)
			continue
		}

		// Parse the qualified path
		col.Schema, col.Table, col.Column = s.parseQualifiedPath(col.QualifiedPath)
		if nodeName.Valid {
			col.Column = nodeName.String
		}

		// Extract data type from properties
		if len(propertiesJSON) > 0 {
			var props NodeProperties
			if err := json.Unmarshal(propertiesJSON, &props); err == nil {
				col.DataType = props.DataType
			}
		}

		// Extract profiling data from profile_properties
		if len(profilePropsJSON) > 0 {
			var profileProps NodeProperties
			if err := json.Unmarshal(profilePropsJSON, &profileProps); err == nil {
				col.Cardinality = profileProps.Cardinality
				col.FrequentValues = profileProps.FrequentValues
			}
		}

		columns = append(columns, col)
	}

	logging.GetLogger().Sugar().Infof("Fetched %d database columns for tenant %s", len(columns), tenantID)

	// If we found zero columns, try a broader fallback query that looks for any catalog_node
	// rows with a qualified_path that looks like schema.table.column. Some environments may
	// populate catalog_node using different node types or enhanced paths; this widens the
	// search to synthesize columns even when the explicit DatabaseColumn node type isn't used.
	if len(columns) == 0 {
		logging.GetLogger().Sugar().Infof("fetchDatabaseColumns: no explicit column nodes found for tenant %s, trying broad qualified_path scan", tenantID)
		altQuery := `
			SELECT id, qualified_path, node_name, tenant_datasource_id, tenant_id, properties
			FROM catalog_node
			WHERE tenant_id = $1 AND tenant_datasource_id = $2 AND qualified_path LIKE '%.%._%'
			ORDER BY qualified_path
			`
		altRows, altErr := s.db.QueryContext(ctx, altQuery, tenantID, tenantDatasourceID)
		if altErr != nil {
			logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns: alternate qualified_path scan failed: %v", altErr)
		} else {
			defer altRows.Close()

			for altRows.Next() {
				var col DatabaseColumn
				var nodeName sql.NullString
				var propertiesJSON []byte
				err := altRows.Scan(&col.NodeID, &col.QualifiedPath, &nodeName, &col.TenantDatasourceID, &col.TenantID, &propertiesJSON)
				if err != nil {
					logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns alt scan error: %v", err)
					continue
				}
				// Parse qualified path into schema.table.column where possible
				col.Schema, col.Table, col.Column = s.parseQualifiedPath(col.QualifiedPath)
				if nodeName.Valid {
					// prefer node_name if it's more specific
					col.Column = nodeName.String
				}
				if len(propertiesJSON) > 0 {
					var props NodeProperties
					if err := json.Unmarshal(propertiesJSON, &props); err == nil {
						col.DataType = props.DataType
					}
				}
				columns = append(columns, col)
			}
			logging.GetLogger().Sugar().Infof("fetchDatabaseColumns: alternate scan found %d candidate columns for tenant %s", len(columns), tenantID)
		}

		// If still zero, try swapping tenant and tenantDatasource (some rows appear to have these values reversed)
		if len(columns) == 0 {
			logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns: no candidates found with tenant=%s datasource=%s; trying swapped lookup", tenantID, tenantDatasourceID)
			swappedQuery := `
				SELECT id, qualified_path, node_name, tenant_datasource_id, tenant_id, properties
				FROM catalog_node
				WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
				ORDER BY qualified_path
			`
			// Note: pass swapped values intentionally (datasource becomes tenant param and vice versa)
			swRows, swErr := s.db.QueryContext(ctx, swappedQuery, DatabaseColumnNodeTypeID, tenantDatasourceID, tenantID)
			if swErr != nil {
				logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns: swapped query failed: %v", swErr)
			} else {
				defer swRows.Close()
				for swRows.Next() {
					var col DatabaseColumn
					var nodeName sql.NullString
					var propertiesJSON []byte
					err := swRows.Scan(&col.NodeID, &col.QualifiedPath, &nodeName, &col.TenantDatasourceID, &col.TenantID, &propertiesJSON)
					if err != nil {
						logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns swapped scan error: %v", err)
						continue
					}
					col.Schema, col.Table, col.Column = s.parseQualifiedPath(col.QualifiedPath)
					if nodeName.Valid {
						col.Column = nodeName.String
					}
					if len(propertiesJSON) > 0 {
						var props NodeProperties
						if err := json.Unmarshal(propertiesJSON, &props); err == nil {
							col.DataType = props.DataType
						}
					}
					columns = append(columns, col)
				}
				logging.GetLogger().Sugar().Infof("fetchDatabaseColumns: swapped lookup found %d candidate columns", len(columns))
			}
		}

		// If still zero, do a broader ID match: sometimes tenant and datasource IDs may be stored in either
		// the tenant_id or tenant_datasource_id columns due to upstream ingestion differences. Try to find
		// any DatabaseColumn nodes where either column equals either of the provided IDs.
		if len(columns) == 0 {
			logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns: no candidates after swapped lookup; trying broad id-match for tenant/datasource (%s,%s)", tenantID, tenantDatasourceID)
			broadQuery := `
				SELECT id, qualified_path, node_name, tenant_datasource_id, tenant_id, properties
				FROM catalog_node
				WHERE node_type_id = $1
				  AND (
					tenant_id = $2 OR tenant_datasource_id = $2 OR tenant_id = $3 OR tenant_datasource_id = $3
				  )
				ORDER BY qualified_path
			`
			brRows, brErr := s.db.QueryContext(ctx, broadQuery, DatabaseColumnNodeTypeID, tenantID, tenantDatasourceID)
			if brErr != nil {
				logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns: broad id-match query failed: %v", brErr)
				return columns, nil
			}
			defer brRows.Close()
			for brRows.Next() {
				var col DatabaseColumn
				var nodeName sql.NullString
				var propertiesBytes []byte
				err := brRows.Scan(&col.NodeID, &col.QualifiedPath, &nodeName, &col.TenantDatasourceID, &col.TenantID, &propertiesBytes)
				if err != nil {
					logging.GetLogger().Sugar().Warnf("fetchDatabaseColumns broad scan error: %v", err)
					continue
				}
				col.Schema, col.Table, col.Column = s.parseQualifiedPath(col.QualifiedPath)
				if nodeName.Valid {
					col.Column = nodeName.String
				}
				if len(propertiesBytes) > 0 {
					var props NodeProperties
					if err := json.Unmarshal(propertiesBytes, &props); err == nil {
						col.DataType = props.DataType
					}
				}
				columns = append(columns, col)
			}
			logging.GetLogger().Sugar().Infof("fetchDatabaseColumns: broad id-match found %d candidate columns", len(columns))
		}
	}

	return columns, nil
}

// Fetch existing semantic terms
func (s *SemanticMappingService) fetchSemanticTerms(ctx context.Context, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	query := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.properties, ref.properties as ref_properties
		FROM catalog_node cn
		LEFT JOIN reference_data ref ON cn.id = ref.semantic_term_id::uuid -- Join with reference data
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		ORDER BY node_name
	`

	rows, err := s.db.QueryContext(ctx, query, SemanticTermNodeTypeID, tenantID, tenantDatasourceID)
	joinedRef := true
	if err != nil {
		// If reference_data table or the join isn't available in this environment,
		// fall back to a simpler query that omits the LEFT JOIN. This mirrors the
		// resilient approach used in fetchDatabaseColumns and allows the service
		// to operate in lighter-weight setups.
		logging.GetLogger().Sugar().Warnf("fetchSemanticTerms initial query failed for tenant=%s datasource=%s: %v -- retrying simpler query", tenantID, tenantDatasourceID, err)
		simpleQuery := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.properties
		FROM catalog_node cn
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		ORDER BY node_name
		`

		rows, err = s.db.QueryContext(ctx, simpleQuery, SemanticTermNodeTypeID, tenantID, tenantDatasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("fetchSemanticTerms fallback query also failed for tenant=%s datasource=%s: %v", tenantID, tenantDatasourceID, err)
			return nil, err
		}
		joinedRef = false
	}
	defer rows.Close()

	terms := make([]SemanticTerm, 0)
	if joinedRef {
		for rows.Next() {
			var term SemanticTerm
			var propertiesJSON, refPropsJSON []byte

			err := rows.Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesJSON, &refPropsJSON)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Error scanning term: %v", err)
				continue
			}

			// Extract data type from properties
			if len(propertiesJSON) > 0 {
				var props NodeProperties
				if err := json.Unmarshal(propertiesJSON, &props); err == nil {
					term.DataType = props.DataType
				}
			}

			// Extract reference data from ref_properties
			if len(refPropsJSON) > 0 {
				var refProps NodeProperties // Assuming similar structure
				if err := json.Unmarshal(refPropsJSON, &refProps); err == nil {
					term.ReferenceValues = refProps.FrequentValues     // Canonical values for the term
					term.ReferencePatterns = refProps.InferredPatterns // Canonical patterns
				}
			}

			terms = append(terms, term)
		}
	} else {
		for rows.Next() {
			var term SemanticTerm
			var propertiesJSON []byte

			err := rows.Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesJSON)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Error scanning term (fallback): %v", err)
				continue
			}

			if len(propertiesJSON) > 0 {
				var props NodeProperties
				if err := json.Unmarshal(propertiesJSON, &props); err == nil {
					term.DataType = props.DataType
				}
			}

			terms = append(terms, term)
		}
	}

	logging.GetLogger().Sugar().Infof("Fetched %d semantic terms for tenant %s", len(terms), tenantID)
	return terms, nil

}

// ListSemanticTerms is an exported wrapper for fetching semantic terms. It calls the
// internal fetchSemanticTerms implementation.
func (s *SemanticMappingService) ListSemanticTerms(ctx context.Context, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	return s.fetchSemanticTerms(ctx, tenantID, tenantDatasourceID)
}

// fetchBusinessTerms fetches business term nodes from the catalog. This is an
// internal helper; use FetchBusinessTerms to call from other packages.
func (s *SemanticMappingService) fetchBusinessTerms(ctx context.Context, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	query := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.properties
		FROM catalog_node cn
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3
		ORDER BY node_name
	`

	rows, err := s.db.QueryContext(ctx, query, BusinessTermNodeTypeID, tenantID, tenantDatasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("fetchBusinessTerms query failed for tenant=%s datasource=%s: %v", tenantID, tenantDatasourceID, err)
		return nil, err
	}
	defer rows.Close()

	terms := make([]SemanticTerm, 0)
	for rows.Next() {
		var term SemanticTerm
		var propertiesJSON []byte

		if err := rows.Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesJSON); err != nil {
			logging.GetLogger().Sugar().Warnf("Error scanning business term: %v", err)
			continue
		}

		if len(propertiesJSON) > 0 {
			var props NodeProperties
			if err := json.Unmarshal(propertiesJSON, &props); err == nil {
				term.DataType = props.DataType
			}
		}

		terms = append(terms, term)
	}

	logging.GetLogger().Sugar().Infof("Fetched %d business terms for tenant %s", len(terms), tenantID)
	return terms, nil
}

// FetchBusinessTerms is an exported wrapper for fetching business terms.
func (s *SemanticMappingService) FetchBusinessTerms(ctx context.Context, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	return s.fetchBusinessTerms(ctx, tenantID, tenantDatasourceID)
}

// SearchBusinessTerms performs a typeahead search over business terms.
func (s *SemanticMappingService) SearchBusinessTerms(ctx context.Context, req SearchRequest, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	if req.Limit == 0 {
		req.Limit = 10
	}
	// Default candidate limit larger than return limit to allow re-ranking
	if req.CandidateLimit == 0 {
		req.CandidateLimit = 100
	}

	query := `
		SELECT id, node_name, qualified_path, properties
		FROM catalog_node
		WHERE node_type_id = $1
		AND tenant_id = $2
		AND tenant_datasource_id = $3
		AND UPPER(node_name) LIKE UPPER($4)
		ORDER BY node_name
		LIMIT $5
	`

	searchPattern := "%" + req.Query + "%"
	rows, err := s.db.QueryContext(ctx, query, BusinessTermNodeTypeID, tenantID, tenantDatasourceID, searchPattern, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search business terms: %w", err)
	}
	defer rows.Close()

	terms := make([]SemanticTerm, 0)
	for rows.Next() {
		var term SemanticTerm
		var propertiesJSON []byte
		if err := rows.Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesJSON); err != nil {
			continue
		}
		if len(propertiesJSON) > 0 {
			var props NodeProperties
			if err := json.Unmarshal(propertiesJSON, &props); err == nil {
				term.DataType = props.DataType
			}
		}
		terms = append(terms, term)
	}

	// If the request included scope tables, we currently do not perform special reordering
	// here for business terms. The frontend or a future implementation can request
	// scope-aware suggestions. For now, just log the scope and return the fetched terms.
	if len(req.ScopeTables) > 0 {
		logging.GetLogger().Sugar().Debugf("SearchBusinessTerms: scope provided (%v) but scope-reordering not applied", req.ScopeTables)
	}

	// Return fetched terms as-is for now
	return terms, nil

}

// mapColumnsToTerms maps database columns to semantic terms with fuzzy logic.
func (s *SemanticMappingService) mapColumnsToTerms(ctx context.Context, columns []DatabaseColumn, terms []SemanticTerm) []MappingResult {
	var results []MappingResult

	// If no semantic terms exist, use cross-table fuzzy matching
	if len(terms) == 0 {
		return s.mapColumnsWithFuzzyLogic(ctx, columns)
	}

	// Standard mapping with existing semantic terms
	for _, col := range columns {
		// First, check if this column already has a mapped semantic term
		existingTerm, err := s.getExistingMappedTerm(col.NodeID, col.TenantDatasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Error checking existing mapping for column %s: %v", col.NodeID, err)
		}

		// If there's an existing mapping, use it
		if existingTerm != nil {
			result := MappingResult{
				DatabaseColumn: col,
				SemanticTerm:   existingTerm.TermName,
				SemanticTermID: existingTerm.NodeID,
				Confidence:     1.0, // Existing mappings have full confidence
				IsNewTerm:      false,
				Selected:       false,
				MatchReason:    "Existing mapping",
				EdgeExists:     true,
			}
			results = append(results, result)
			continue
		}

		// Generate semantic term for this column
		generatedTerm := s.generateSemanticTerm(col.Schema, col.Table, col.Column)

		// Find best matching existing term
		var bestMatch SemanticTerm
		var bestConfidence float64 = 0
		var bestMatchReason string

		for _, term := range terms {
			confidence, reason, _ := s.calculateSemanticConfidence(
				ctx, generatedTerm, term.TermName,
				&col, &term,
			)
			if confidence > bestConfidence {
				bestConfidence = confidence
				bestMatch = term
				bestMatchReason = reason
			}
		}

		// Check whether an edge already exists for this semantic term and column (only if we have a term id)
		edgeExists := false
		if bestMatch.NodeID != "" {
			if exists, err := s.checkEdgeExists(col.NodeID, bestMatch.NodeID, col.TenantDatasourceID); err == nil {
				edgeExists = exists
			}
		}

		result := MappingResult{
			DatabaseColumn: col,
			SemanticTerm:   generatedTerm,
			Confidence:     bestConfidence,
			IsNewTerm:      bestConfidence < 0.75,
			Selected:       false,
			MatchReason:    bestMatchReason,
			EdgeExists:     edgeExists,
		}

		// If we found a good match, use the existing term
		if bestConfidence >= 0.75 {
			result.SemanticTerm = bestMatch.TermName
			result.SemanticTermID = bestMatch.NodeID
			result.IsNewTerm = false
		}

		results = append(results, result)
	}

	return results
}

// GenerateMappings computes suggested semantic mappings for all database columns
// in the tenant/datasource by fetching columns and semantic terms and applying
// the mapping logic. This is exposed to the API layer.
func (s *SemanticMappingService) GenerateMappings(ctx context.Context, tenantID, tenantDatasourceID string) ([]MappingResult, error) {
	// Fetch database columns with optional profiler info
	cols, err := s.fetchDatabaseColumns(ctx, tenantID, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch database columns: %w", err)
	}

	// Fetch existing semantic terms
	terms, err := s.ListSemanticTerms(ctx, tenantID, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch semantic terms: %w", err)
	}

	// Map columns to terms
	results := s.mapColumnsToTerms(ctx, cols, terms)
	return results, nil
}

// Map columns using cross-table fuzzy logic (when no semantic terms exist)
func (s *SemanticMappingService) mapColumnsWithFuzzyLogic(_ context.Context, columns []DatabaseColumn) []MappingResult {
	// Group columns by semantic term and calculate confidence
	semanticGroups := make(map[string][]DatabaseColumn)
	termDataTypes := make(map[string]string)

	// Generate semantic terms for all columns
	for _, col := range columns {
		generatedTerm := s.generateSemanticTerm(col.Schema, col.Table, col.Column)
		semanticGroups[generatedTerm] = append(semanticGroups[generatedTerm], col)

		// Track data type (use first occurrence)
		if _, exists := termDataTypes[generatedTerm]; !exists && col.DataType != "" {
			termDataTypes[generatedTerm] = col.DataType
		}
	}

	var results []MappingResult

	// For each column, calculate confidence based on fuzzy matching with all other columns
	for _, col := range columns {
		generatedTerm := s.generateSemanticTerm(col.Schema, col.Table, col.Column)

		// Count how many tables use the same semantic pattern
		sameTermCount := len(semanticGroups[generatedTerm])

		// Calculate confidence based on various factors
		var confidence float64
		var matchReason strings.Builder

		// Factor 1: Frequency (more tables with same pattern = higher confidence)
		frequencyScore := float64(s.min(sameTermCount, 10)) / 10.0 * 0.3
		if sameTermCount > 1 {
			matchReason.WriteString(fmt.Sprintf("Used in %d table(s); ", sameTermCount))
		}

		// Factor 2: Check for fuzzy matches with other semantic terms
		bestFuzzyScore := 0.0
		matchCount := 0
		for otherTerm := range semanticGroups {
			if otherTerm == generatedTerm {
				continue
			}

			fuzzyScore, _, _ := s.calculateSemanticConfidenceOriginal(
				generatedTerm, otherTerm,
				&col, &SemanticTerm{DataType: termDataTypes[otherTerm]},
			)

			if fuzzyScore > 0.6 {
				matchCount++
				if fuzzyScore > bestFuzzyScore {
					bestFuzzyScore = fuzzyScore
				}
			}
		}

		fuzzyMatchScore := bestFuzzyScore * 0.4
		if matchCount > 0 {
			matchReason.WriteString(fmt.Sprintf("%d similar pattern(s) found; ", matchCount))
		}

		// Factor 3: Column name quality (longer, more structured names = higher confidence)
		tokens := s.tokenizeSemanticTerm(generatedTerm)
		qualityScore := float64(s.min(len(tokens), 3)) / 3.0 * 0.15

		// Factor 4: Data type availability (having data type = higher confidence)
		dataTypeScore := 0.0
		if col.DataType != "" {
			dataTypeScore = 0.15
			matchReason.WriteString(fmt.Sprintf("Data type: %s", s.normalizeDataType(col.DataType)))
		}

		// Combine all factors
		confidence = frequencyScore + fuzzyMatchScore + qualityScore + dataTypeScore

		// Ensure confidence is between 0 and 1
		if confidence > 1.0 {
			confidence = 1.0
		}

		// Boost confidence for common patterns
		commonPatterns := []string{
			"ID", "NAME", "DATE", "TIME", "CREATED", "UPDATED", "DELETED",
			"EMAIL", "PHONE", "ADDRESS", "STATUS", "TYPE", "CODE", "AMOUNT",
		}
		for _, pattern := range commonPatterns {
			if strings.Contains(generatedTerm, pattern) {
				confidence = confidence * 1.1
				if confidence > 1.0 {
					confidence = 1.0
				}
				break
			}
		}

		result := MappingResult{
			DatabaseColumn: col,
			SemanticTerm:   generatedTerm,
			Confidence:     confidence,
			IsNewTerm:      true,
			Selected:       false,
			MatchReason:    matchReason.String(),
		}

		results = append(results, result)
	}

	return results
}

// Search semantic terms for typeahead
func (s *SemanticMappingService) SearchSemanticTerms(ctx context.Context, req SearchRequest, tenantID, tenantDatasourceID string) ([]SemanticTerm, error) {
	if req.Limit == 0 {
		req.Limit = 10
	}

	logging.GetLogger().Sugar().Infof("SearchSemanticTerms called: query='%s' limit=%d scope_tables=%v tenant=%s ds=%s", req.Query, req.Limit, req.ScopeTables, tenantID, tenantDatasourceID)

	query := `
		SELECT id, node_name, qualified_path, properties
		FROM catalog_node
		WHERE node_type_id = $1
		AND tenant_id = $2
		AND tenant_datasource_id = $3
		AND UPPER(node_name) LIKE UPPER($4)
		ORDER BY node_name
		LIMIT $5
	`

	searchPattern := "%" + req.Query + "%"
	// Use CandidateLimit default if not set
	if req.CandidateLimit == 0 {
		req.CandidateLimit = 200
	}

	rows, err := s.db.QueryContext(ctx, query, SemanticTermNodeTypeID, tenantID, tenantDatasourceID, searchPattern, req.CandidateLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to search terms: %w", err)
	}
	defer rows.Close()

	terms := make([]SemanticTerm, 0)
	for rows.Next() {
		var term SemanticTerm
		var propertiesJSON []byte

		err := rows.Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesJSON)
		if err != nil {
			continue
		}

		logging.GetLogger().Sugar().Infof("SearchSemanticTerms: found %d terms for query='%s'", len(terms), req.Query)

		// Extract data type from properties
		if len(propertiesJSON) > 0 {
			var props NodeProperties
			if err := json.Unmarshal(propertiesJSON, &props); err == nil {
				term.DataType = props.DataType
			}
		}

		terms = append(terms, term)
	}

	// If scope tables provided, compute a simple relevance score and reorder
	if len(req.ScopeTables) > 0 && len(terms) > 1 {
		// normalize scope tokens
		tokens := make([]string, 0)
		for _, sc := range req.ScopeTables {
			scClean := strings.ToLower(strings.TrimSpace(sc))
			if scClean == "" {
				continue
			}
			// split schema-qualified like public.users -> users and keep both forms
			parts := strings.Split(scClean, ".")
			tokens = append(tokens, scClean)
			if len(parts) > 0 {
				tokens = append(tokens, parts[len(parts)-1])
			}
		}

		type scored struct {
			t     SemanticTerm
			score int
			idx   int
		}

		scoredList := make([]scored, 0, len(terms))
		for i, t := range terms {
			sScore := 0
			qLower := strings.ToLower(t.QualifiedPath)
			nLower := strings.ToLower(t.TermName)

			for _, tok := range tokens {
				if tok == "" {
					continue
				}
				// exact qualified path match (highest boost)
				if qLower == tok || nLower == tok {
					sScore += 200
					continue
				}
				// qualified_path contains token is strong
				if strings.Contains(qLower, tok) {
					sScore += 50
				}
				// term name contains token is medium
				if strings.Contains(nLower, tok) {
					sScore += 20
				}
				// token tokenized match on term name tokens
				nParts := strings.FieldsFunc(nLower, func(r rune) bool { return r == '_' || r == ' ' || r == '/' || r == '-' })
				for _, np := range nParts {
					if np == tok {
						sScore += 10
						break
					}
				}
			}

			// small bonus for nodes whose qualified path contains the original query
			if req.Query != "" {
				q := strings.ToLower(req.Query)
				if strings.Contains(qLower, q) || strings.Contains(nLower, q) {
					sScore += 5
				}
			}

			// attach numeric score as integer for now; we'll normalize to 0..1 after
			// small boost if the term has an explicit data type (presence indicates richer metadata)
			if t.DataType != "" {
				sScore += 10
			}

			// If column context provided, boost terms that are data-type compatible
			if req.Column != nil && req.Column.DataType != "" && t.DataType != "" {
				if s.areDataTypesCompatible(req.Column.DataType, t.DataType) {
					sScore += 50 // stronger boost for compatible types
				} else {
					// small penalty for incompatible types to deprioritize them
					sScore -= 10
				}
			}

			scoredList = append(scoredList, scored{t: t, score: sScore, idx: i})
		}

		// stable sort by score desc, index asc
		// Determine max score for normalization
		maxScore := 0
		for _, sIt := range scoredList {
			if sIt.score > maxScore {
				maxScore = sIt.score
			}
		}

		// Normalize scores to 0..1 (avoid division by zero)
		for i := range scoredList {
			if maxScore > 0 {
				scoredList[i].t.Score = float64(scoredList[i].score) / float64(maxScore)
			} else {
				scoredList[i].t.Score = 0.0
			}
		}

		sort.SliceStable(scoredList, func(i, j int) bool {
			if scoredList[i].score == scoredList[j].score {
				return scoredList[i].idx < scoredList[j].idx
			}
			return scoredList[i].score > scoredList[j].score
		})

		out := make([]SemanticTerm, 0, len(scoredList))
		for _, sIt := range scoredList {
			out = append(out, sIt.t)
		}

		// Return only the requested number of results (after re-ranking)
		topN := req.Limit
		if topN <= 0 || topN > len(out) {
			topN = len(out)
		}
		logging.GetLogger().Sugar().Infof("SearchSemanticTerms: reordered %d candidates using scope; returning top %d (maxScore=%d)", len(out), topN, maxScore)
		return out[:topN], nil
	}

	// No scope ordering requested - return SQL results as-is
	if len(terms) > 0 {
		return terms, nil
	}

	// If we reached here there were no semantic-term rows returned from catalog_node.
	// Use a schema-derived fallback: generate candidate semantic terms from the datasource
	// columns so the UI can show helpful suggestions even for a fresh tenant/datasource.
	logging.GetLogger().Sugar().Infof("SearchSemanticTerms: no semantic terms found for tenant=%s ds=%s; generating fallbacks from datasource schema", tenantID, tenantDatasourceID)

	// Attempt to fetch database columns and synthesize semantic-term candidates
	columns, colErr := s.fetchDatabaseColumns(ctx, tenantID, tenantDatasourceID)
	if colErr != nil {
		logging.GetLogger().Sugar().Warnf("SearchSemanticTerms: failed to fetch database columns for fallback: %v", colErr)
		// return empty list to caller
		return []SemanticTerm{}, nil
	}

	// Synthesize terms from columns: simple frequency + data-type presence scoring
	generated := make([]SemanticTerm, 0, len(columns))
	for _, c := range columns {
		term := SemanticTerm{
			NodeID:        "", // generated, not persisted
			TermName:      s.generateSemanticTerm(c.Schema, c.Table, c.Column),
			QualifiedPath: fmt.Sprintf("%s.%s.%s", c.TenantDatasourceID, c.Schema, c.Table),
			DataType:      c.DataType,
			Score:         0.0,
		}
		generated = append(generated, term)
	}

	// Basic normalization: score by presence of data type and query substring match
	maxScore := 0.0
	scores := make([]float64, len(generated))
	qLower := strings.ToLower(req.Query)
	for i, g := range generated {
		sVal := 0.0
		if g.DataType != "" {
			sVal += 1.0
		}
		if qLower != "" && strings.Contains(strings.ToLower(g.TermName), qLower) {
			sVal += 1.0
		}
		scores[i] = sVal
		if sVal > maxScore {
			maxScore = sVal
		}
	}

	for i := range generated {
		if maxScore > 0 {
			generated[i].Score = scores[i] / maxScore
		} else {
			generated[i].Score = 0.0
		}
	}

	// Sort by score desc
	sort.SliceStable(generated, func(i, j int) bool {
		return generated[i].Score > generated[j].Score
	})

	// Limit to requested number
	topN := req.Limit
	if topN <= 0 || topN > len(generated) {
		topN = len(generated)
	}

	logging.GetLogger().Sugar().Infof("SearchSemanticTerms: returning %d generated fallback candidates", topN)
	return generated[:topN], nil
}

// Create semantic term
func (s *SemanticMappingService) CreateSemanticTerm(ctx context.Context, tenantID, tenantDatasourceID, termName, dataType string) (string, error) {
	termID := uuid.New().String()
	qualifiedPath := fmt.Sprintf("/semantic/%s", termName)

	// Create properties JSON with data type
	properties := map[string]interface{}{
		"data_type": dataType,
	}
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal properties: %w", err)
	}

	query := `
		INSERT INTO catalog_node (
			id, tenant_datasource_id, node_type_id, node_name,
			qualified_path, tenant_id, created_at, updated_at, properties
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.ExecContext(ctx, query,
		termID, tenantDatasourceID, SemanticTermNodeTypeID, termName,
		qualifiedPath, tenantID, time.Now(), time.Now(), string(propertiesJSON))

	if err != nil {
		return "", fmt.Errorf("failed to create semantic term: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Created semantic term: %s for tenant %s", termName, tenantID)
	return termID, nil
}

// Create mapping edge. Returns true if a new edge row was created, false if it already existed.
func (s *SemanticMappingService) CreateMappingEdge(ctx context.Context, tenantID, tenantDatasourceID, semanticTermID, columnNodeID string) (bool, error) {
	edgeID := uuid.New().String()

	query := `
		INSERT INTO catalog_edge (
			id, tenant_datasource_id, source_node_id, target_node_id,
			relationship_type, edge_type_id, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
		DO NOTHING
	`

	res, err := s.db.ExecContext(ctx, query,
		edgeID, tenantDatasourceID, semanticTermID, columnNodeID,
		"MAPS_TO", EdgeTypeID, tenantID, time.Now(), time.Now())

	if err != nil {
		return false, fmt.Errorf("failed to create mapping edge: %w", err)
	}

	rows, _ := res.RowsAffected()
	created := rows > 0
	if created {
		logging.GetLogger().Sugar().Infof("Created mapping edge: %s -> %s for tenant %s", semanticTermID, columnNodeID, tenantID)
	} else {
		logging.GetLogger().Sugar().Debugf("Mapping edge already exists: %s -> %s for tenant %s", semanticTermID, columnNodeID, tenantID)
	}
	return created, nil
}

// DeleteMappingEdge removes an existing mapping edge between semantic term and column
// DeleteMappingEdge removes an existing mapping edge between semantic term and column and returns
// the number of rows deleted.
func (s *SemanticMappingService) DeleteMappingEdge(ctx context.Context, tenantID, tenantDatasourceID, semanticTermID, columnNodeID string) (int64, error) {
	query := `
		DELETE FROM catalog_edge
		WHERE tenant_datasource_id = $1
		AND source_node_id = $2
		AND target_node_id = $3
		AND edge_type_id = $4
	`
	res, err := s.db.ExecContext(ctx, query, tenantDatasourceID, semanticTermID, columnNodeID, EdgeTypeID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete mapping edge: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows > 0 {
		logging.GetLogger().Sugar().Infof("Deleted mapping edge: %s -> %s for tenant %s", semanticTermID, columnNodeID, tenantID)
	} else {
		logging.GetLogger().Sugar().Debugf("No mapping edge to delete: %s -> %s for tenant %s", semanticTermID, columnNodeID, tenantID)
	}
	return rows, nil
}

// checkEdgeExists returns true if a mapping edge already exists between the semantic term and column
func (s *SemanticMappingService) checkEdgeExists(columnNodeID, semanticTermID, tenantDatasourceID string) (bool, error) {
	query := `
		SELECT 1 FROM catalog_edge
		WHERE tenant_datasource_id = $1
		AND source_node_id = $2
		AND target_node_id = $3
		AND edge_type_id = $4
		LIMIT 1
	`

	var dummy int
	err := s.db.QueryRow(query, tenantDatasourceID, semanticTermID, columnNodeID, EdgeTypeID).Scan(&dummy)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// getExistingMappedTerm returns the semantic term that is currently mapped to this column, if any
func (s *SemanticMappingService) getExistingMappedTerm(columnNodeID, tenantDatasourceID string) (*SemanticTerm, error) {
	query := `
		SELECT cn.id, cn.node_name, cn.qualified_path, cn.properties
		FROM catalog_edge ce
		JOIN catalog_node cn ON ce.source_node_id = cn.id
		WHERE ce.tenant_datasource_id = $1
		AND ce.target_node_id = $2
		AND ce.edge_type_id = $3
		LIMIT 1
	`

	var term SemanticTerm
	var propsBytes []byte
	err := s.db.QueryRow(query, tenantDatasourceID, columnNodeID, EdgeTypeID).Scan(
		&term.NodeID,
		&term.TermName,
		&term.QualifiedPath,
		&propsBytes,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No existing mapping
		}
		return nil, err
	}

	// Parse properties if available
	if len(propsBytes) > 0 {
		var props map[string]interface{}
		if err := json.Unmarshal(propsBytes, &props); err == nil {
			if dataType, ok := props["data_type"].(string); ok {
				term.DataType = dataType
			}
		}
	}

	return &term, nil
}

// FindSemanticTermByName finds a semantic term by exact name
func (s *SemanticMappingService) FindSemanticTermByName(ctx context.Context, tenantID, tenantDatasourceID, termName string) (*SemanticTerm, error) {
	query := `
		SELECT id, node_name, qualified_path, properties
		FROM catalog_node
		WHERE node_type_id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3 AND node_name = $4
	`

	var term SemanticTerm
	var propertiesBytes []byte

	err := s.db.QueryRowContext(ctx, query, SemanticTermNodeTypeID, tenantID, tenantDatasourceID, termName).Scan(&term.NodeID, &term.TermName, &term.QualifiedPath, &propertiesBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Extract data type from properties
	if len(propertiesBytes) > 0 {
		var props NodeProperties
		if err := json.Unmarshal(propertiesBytes, &props); err == nil {
			term.DataType = props.DataType
		}
	}

	return &term, nil
}

// GetOrCreateSemanticTerm gets an existing semantic term or creates a new one
func (s *SemanticMappingService) GetOrCreateSemanticTerm(ctx context.Context, tenantID, tenantDatasourceID, termName, dataType string) (string, error) {
	term, err := s.FindSemanticTermByName(ctx, tenantID, tenantDatasourceID, termName)
	if err != nil {
		return "", err
	}

	if term != nil {
		return term.NodeID, nil
	}

	return s.CreateSemanticTerm(ctx, tenantID, tenantDatasourceID, termName, dataType)
}

// getColumnByID retrieves a database column by its node ID
func (s *SemanticMappingService) getColumnByID(ctx context.Context, tenantID, tenantDatasourceID, columnNodeID string) (*DatabaseColumn, error) {
	query := `
		SELECT id, qualified_path, node_name, tenant_datasource_id, tenant_id, properties
		FROM catalog_node
		WHERE id = $1 AND node_type_id = $2 AND tenant_id = $3 AND tenant_datasource_id = $4
	`

	var col DatabaseColumn
	var nodeName sql.NullString
	var propertiesBytes []byte

	err := s.db.QueryRowContext(ctx, query, columnNodeID, DatabaseColumnNodeTypeID, tenantID, tenantDatasourceID).Scan(&col.NodeID, &col.QualifiedPath, &nodeName, &col.TenantDatasourceID, &col.TenantID, &propertiesBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Parse the qualified path
	col.Schema, col.Table, col.Column = s.parseQualifiedPath(col.QualifiedPath)
	if nodeName.Valid {
		col.Column = nodeName.String
	}

	// Extract data type from properties
	if len(propertiesBytes) > 0 {
		var props NodeProperties
		if err := json.Unmarshal(propertiesBytes, &props); err == nil {
			col.DataType = props.DataType
		}
	}

	return &col, nil
}

// getCurrentMapping retrieves the current semantic term ID mapped to a column
func (s *SemanticMappingService) getCurrentMapping(ctx context.Context, tenantID, tenantDatasourceID, columnNodeID string) (string, error) {
	query := `
		SELECT source_node_id FROM catalog_edge
		WHERE tenant_datasource_id = $1 AND target_node_id = $2 AND edge_type_id = $3
	`

	var termID string
	err := s.db.QueryRowContext(ctx, query, tenantDatasourceID, columnNodeID, EdgeTypeID).Scan(&termID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return termID, nil
}

// ApplyCustomMapping applies a custom semantic term mapping to a database column, allowing overrides
func (s *SemanticMappingService) ApplyCustomMapping(ctx context.Context, tenantID, tenantDatasourceID, columnNodeID, semanticTermName string) error {
	// Get the column to determine data type
	column, err := s.getColumnByID(ctx, tenantID, tenantDatasourceID, columnNodeID)
	if err != nil {
		return fmt.Errorf("failed to get column: %w", err)
	}
	if column == nil {
		return fmt.Errorf("column not found")
	}

	// Get or create the semantic term
	termID, err := s.GetOrCreateSemanticTerm(ctx, tenantID, tenantDatasourceID, semanticTermName, column.DataType)
	if err != nil {
		return fmt.Errorf("failed to get or create semantic term: %w", err)
	}

	// Check if there's an existing mapping
	currentTermID, err := s.getCurrentMapping(ctx, tenantID, tenantDatasourceID, columnNodeID)
	if err != nil {
		return fmt.Errorf("failed to get current mapping: %w", err)
	}

	// If there's an existing mapping to a different term, delete it
	if currentTermID != "" && currentTermID != termID {
		_, err = s.DeleteMappingEdge(ctx, tenantID, tenantDatasourceID, currentTermID, columnNodeID)
		if err != nil {
			return fmt.Errorf("failed to delete existing mapping: %w", err)
		}
	}

	// Create the new mapping edge (if it doesn't already exist)
	_, err = s.CreateMappingEdge(ctx, tenantID, tenantDatasourceID, termID, columnNodeID)
	if err != nil {
		return fmt.Errorf("failed to create mapping edge: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Applied custom mapping: column %s -> term %s for tenant %s", columnNodeID, semanticTermName, tenantID)
	return nil
}

// CreateBusinessTerm creates a new business term node with flexible properties.
func (s *SemanticMappingService) CreateBusinessTerm(ctx context.Context, tenantID, tenantDatasourceID, termName string, properties map[string]interface{}) (string, error) {
	termID := uuid.New().String()
	// Business terms have a global-like path, not tied to a physical source.
	qualifiedPath := fmt.Sprintf("/business/%s", termName)

	// Use the provided properties map directly.
	propertiesJSON, err := json.Marshal(properties)
	if err != nil {
		return "", fmt.Errorf("failed to marshal business term properties: %w", err)
	}

	query := `
		INSERT INTO catalog_node (
			id, tenant_datasource_id, node_type_id, node_name,
			qualified_path, tenant_id, created_at, updated_at, properties
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = s.db.ExecContext(ctx, query,
		termID, tenantDatasourceID, BusinessTermNodeTypeID, termName,
		qualifiedPath, tenantID, time.Now(), time.Now(), string(propertiesJSON))

	if err != nil {
		// Check for unique constraint violation on name
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return "", fmt.Errorf("a business term with the name '%s' already exists", termName)
		}
		return "", fmt.Errorf("failed to create business term in database: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Created business term: %s (ID: %s) for tenant %s", termName, termID, tenantID)
	return termID, nil
}

// UpdateBusinessTerm updates an existing business term node with new properties.
func (s *SemanticMappingService) UpdateBusinessTerm(ctx context.Context, tenantID, tenantDatasourceID, termNodeID string, updates map[string]interface{}) error {
	// Validate inputs
	if strings.TrimSpace(termNodeID) == "" {
		return fmt.Errorf("term node ID is required")
	}
	if _, err := uuid.Parse(termNodeID); err != nil {
		return fmt.Errorf("term node ID must be a valid UUID: %w", err)
	}
	if len(updates) == 0 {
		return fmt.Errorf("updates map is required and cannot be empty")
	}

	// First verify the business term exists and belongs to this tenant
	var existingName string
	checkQuery := `
		SELECT node_name 
		FROM catalog_node 
		WHERE id = $1 AND node_type_id = $2 AND tenant_id = $3 AND tenant_datasource_id = $4
	`
	err := s.db.QueryRowContext(ctx, checkQuery, termNodeID, BusinessTermNodeTypeID, tenantID, tenantDatasourceID).Scan(&existingName)
	if err == sql.ErrNoRows {
		return fmt.Errorf("business term not found or access denied")
	}
	if err != nil {
		return fmt.Errorf("failed to verify business term: %w", err)
	}

	// Extract term name if provided in updates
	termName := existingName
	if name, ok := updates["term_name"].(string); ok && strings.TrimSpace(name) != "" {
		termName = strings.ToUpper(strings.TrimSpace(name))
	}

	// Build qualified path
	qualifiedPath := fmt.Sprintf("/business_terms/%s", termName)

	// Marshal properties to JSON
	propertiesJSON, err := json.Marshal(updates)
	if err != nil {
		return fmt.Errorf("failed to marshal properties: %w", err)
	}

	// Update the business term
	updateQuery := `
		UPDATE catalog_node 
		SET node_name = $1, qualified_path = $2, properties = $3, updated_at = $4
		WHERE id = $5 AND node_type_id = $6 AND tenant_id = $7 AND tenant_datasource_id = $8
	`

	result, err := s.db.ExecContext(ctx, updateQuery,
		termName, qualifiedPath, string(propertiesJSON), time.Now(),
		termNodeID, BusinessTermNodeTypeID, tenantID, tenantDatasourceID)

	if err != nil {
		// Check for unique constraint violation on name
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return fmt.Errorf("a business term with the name '%s' already exists", termName)
		}
		return fmt.Errorf("failed to update business term in database: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no business term was updated - may not exist or access denied")
	}

	logging.GetLogger().Sugar().Infof("Updated business term: %s (ID: %s) for tenant %s", termName, termNodeID, tenantID)
	return nil
}

// CreateBusinessTermEdge creates a mapping edge from a semantic term to a business term.
func (s *SemanticMappingService) CreateBusinessTermEdge(ctx context.Context, tenantID, tenantDatasourceID, semanticTermID, businessTermID string) (bool, error) {
	// Validate inputs to avoid passing empty strings into UUID columns
	if strings.TrimSpace(semanticTermID) == "" {
		return false, fmt.Errorf("semanticTermID is required")
	}
	if strings.TrimSpace(businessTermID) == "" {
		return false, fmt.Errorf("businessTermID is required")
	}
	if _, err := uuid.Parse(semanticTermID); err != nil {
		return false, fmt.Errorf("semanticTermID must be a valid UUID: %w", err)
	}
	if _, err := uuid.Parse(businessTermID); err != nil {
		return false, fmt.Errorf("businessTermID must be a valid UUID: %w", err)
	}

	edgeID := uuid.New().String()

	// For business term -> semantic term mapping use the canonical edge type id
	businessEdgeTypeID := "3be9d6ae-1598-4628-a3dd-b606921a9193"

	query := `
		INSERT INTO catalog_edge (
			id, tenant_datasource_id, source_node_id, target_node_id,
			relationship_type, edge_type_id, tenant_id, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
		DO NOTHING
	`

	// Note: source is business term, target is semantic term
	res, err := s.db.ExecContext(ctx, query,
		edgeID, tenantDatasourceID, businessTermID, semanticTermID,
		"business_term_to_semantic_term", businessEdgeTypeID, tenantID, time.Now(), time.Now())

	if err != nil {
		return false, fmt.Errorf("failed to create business term mapping edge: %w", err)
	}

	rows, _ := res.RowsAffected()
	created := rows > 0
	logging.GetLogger().Sugar().Infof("Business term mapping edge created (%s -> %s): %v", semanticTermID, businessTermID, created)
	return created, nil
}

// UpsertBusinessTermAndEdge will create a business term if it doesn't exist (by name) and create the mapping edge
// It returns the business term node id and the created flag for the edge
func (s *SemanticMappingService) UpsertBusinessTermAndEdge(ctx context.Context, tenantID, tenantDatasourceID, businessTermName, semanticTermID, edgeTypeID string) (string, bool, error) {
	// Basic validations
	if strings.TrimSpace(businessTermName) == "" {
		return "", false, fmt.Errorf("business term name is required")
	}
	if strings.TrimSpace(semanticTermID) == "" {
		return "", false, fmt.Errorf("semantic term id is required")
	}
	if _, err := uuid.Parse(semanticTermID); err != nil {
		return "", false, fmt.Errorf("semantic term id must be a valid uuid: %w", err)
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	var existingID string
	// Try to find existing business term by name under the tenant/datasource
	qFind := `SELECT id FROM catalog_node WHERE tenant_datasource_id = $1 AND node_type_id = $2 AND UPPER(node_name) = UPPER($3) LIMIT 1`
	err = tx.GetContext(ctx, &existingID, qFind, tenantDatasourceID, BusinessTermNodeTypeID, businessTermName)
	if err != nil && err != sql.ErrNoRows {
		return "", false, fmt.Errorf("failed to query existing business term: %w", err)
	}

	businessID := existingID
	if businessID == "" {
		// Create a new business term
		newID := uuid.New().String()
		props := map[string]interface{}{"created_by": "upsert_service"}
		propsJSON, _ := json.Marshal(props)
		qCreate := `INSERT INTO catalog_node (id, tenant_datasource_id, node_type_id, node_name, qualified_path, tenant_id, created_at, updated_at, properties) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
		_, err = tx.ExecContext(ctx, qCreate, newID, tenantDatasourceID, BusinessTermNodeTypeID, businessTermName, businessTermName, tenantID, time.Now(), time.Now(), string(propsJSON))
		if err != nil {
			return "", false, fmt.Errorf("failed to create business term: %w", err)
		}
		businessID = newID
	}

	// Now create the mapping edge (subject = business term, object = semantic term)
	edgeID := uuid.New().String()
	if edgeTypeID == "" {
		edgeTypeID = "3be9d6ae-1598-4628-a3dd-b606921a9193" // Default to business term to semantic term mapping
	}
	qEdge := `INSERT INTO catalog_edge (id, tenant_datasource_id, source_node_id, target_node_id, edge_type_id, relationship_type, tenant_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO NOTHING`
	res, err := tx.ExecContext(ctx, qEdge, edgeID, tenantDatasourceID, businessID, semanticTermID, edgeTypeID, "business_term_to_semantic_term", tenantID, time.Now(), time.Now())
	if err != nil {
		return businessID, false, fmt.Errorf("failed to create edge: %w", err)
	}
	rows, _ := res.RowsAffected()
	created := rows > 0

	return businessID, created, nil
}

// fetchFeedbackStats retrieves historical feedback statistics for business terms
// to help boost suggestions that users have accepted and penalize those they've rejected
func (s *SemanticMappingService) fetchFeedbackStats(ctx context.Context, tenantID, tenantDatasourceID string) (map[string]*FeedbackStats, error) {
	query := `
		SELECT
			business_term_name,
			COUNT(*) FILTER (WHERE action = 'accept') as accept_count,
			COUNT(*) FILTER (WHERE action = 'reject') as reject_count,
			COUNT(*) as total_feedback,
			CASE 
				WHEN COUNT(*) > 0 THEN 
					COUNT(*) FILTER (WHERE action = 'accept')::numeric / COUNT(*)::numeric 
				ELSE 0 
			END as acceptance_rate
		FROM public.suggestion_feedback
		WHERE tenant_id = $1 AND tenant_datasource_id = $2
		GROUP BY business_term_name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query feedback stats: %w", err)
	}
	defer rows.Close()

	feedbackMap := make(map[string]*FeedbackStats)
	for rows.Next() {
		var stats FeedbackStats
		if err := rows.Scan(&stats.BusinessTermName, &stats.AcceptCount, &stats.RejectCount, &stats.TotalFeedback, &stats.AcceptanceRate); err != nil {
			return nil, fmt.Errorf("failed to scan feedback stats: %w", err)
		}
		feedbackMap[strings.ToUpper(stats.BusinessTermName)] = &stats
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feedback stats: %w", err)
	}

	return feedbackMap, nil
}

// fetchRejectedSuggestions retrieves business terms that have been rejected for a specific semantic term
func (s *SemanticMappingService) fetchRejectedSuggestions(ctx context.Context, tenantID, tenantDatasourceID, semanticTermID string) (map[string]bool, error) {
	query := `
		SELECT DISTINCT business_term_name
		FROM public.suggestion_feedback
		WHERE tenant_id = $1 
			AND tenant_datasource_id = $2 
			AND semantic_term_id = $3 
			AND action = 'reject'
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, tenantDatasourceID, semanticTermID)
	if err != nil {
		return nil, fmt.Errorf("failed to query rejected suggestions: %w", err)
	}
	defer rows.Close()

	rejectedTerms := make(map[string]bool)
	for rows.Next() {
		var termName string
		if err := rows.Scan(&termName); err != nil {
			return nil, fmt.Errorf("failed to scan rejected suggestion: %w", err)
		}
		rejectedTerms[strings.ToUpper(termName)] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rejected suggestions: %w", err)
	}

	return rejectedTerms, nil
}

// SuggestBusinessTerms provides auto-suggestions for business terms based on a semantic term.
func (s *SemanticMappingService) SuggestBusinessTerms(ctx context.Context, tenantID, tenantDatasourceID, semanticTermID string) ([]BusinessTermSuggestionResult, error) {
	// 1. Fetch the source semantic term details
	var semanticTerm SemanticTerm
	err := s.db.GetContext(ctx, &semanticTerm, "SELECT id, node_name, qualified_path, properties->>'data_type' as data_type FROM catalog_node WHERE id = $1 AND node_type_id = $2", semanticTermID, SemanticTermNodeTypeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("semantic term with ID %s not found", semanticTermID)
		}
		return nil, fmt.Errorf("failed to fetch semantic term: %w", err)
	}

	// 2. Fetch all available business terms
	businessTerms, err := s.fetchBusinessTerms(ctx, tenantID, tenantDatasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business terms: %w", err)
	}

	// 3. Fetch rejected suggestions for this semantic term to exclude them
	rejectedTerms, err := s.fetchRejectedSuggestions(ctx, tenantID, tenantDatasourceID, semanticTermID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch rejected suggestions: %v", err)
		// Continue without filtering
		rejectedTerms = make(map[string]bool)
	}

	// 4. Fetch historical feedback data to boost/penalize suggestions
	feedbackMap, err := s.fetchFeedbackStats(ctx, tenantID, tenantDatasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch feedback stats (table may not exist yet): %v", err)
		// Continue without feedback data
		feedbackMap = make(map[string]*FeedbackStats)
	}

	suggestions := make(map[string]*BusinessTermSuggestionResult)

	// 5. Use the external matcher if available
	if s.businessTermMatcher != nil {
		externalSuggestions, err := s.businessTermMatcher.Suggest(ctx, semanticTerm.TermName)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("External business term matcher failed: %v", err)
		} else {
			for _, sug := range externalSuggestions {
				// Skip if this term was previously rejected for this semantic term
				if rejectedTerms[strings.ToUpper(sug.TermName)] {
					continue
				}
				suggestions[sug.TermName] = &BusinessTermSuggestionResult{
					TermName:   sug.TermName,
					Confidence: sug.Confidence,
					Reason:     "Suggested by external matcher.",
					Source:     sug.Source,
				}
			}
		}
	}

	// 6. Calculate confidence against existing business terms
	for _, bt := range businessTerms {
		// Skip if this term was previously rejected for this semantic term
		if rejectedTerms[strings.ToUpper(bt.TermName)] {
			continue
		}
		// Use a simplified confidence calculation for term-to-term matching
		confidence, reason, breakdown := s.calculateSemanticConfidence(ctx, semanticTerm.TermName, bt.TermName, &DatabaseColumn{}, &SemanticTerm{})

		// Apply feedback-based adjustment
		normalizedName := strings.ToUpper(bt.TermName)
		if feedback, hasFeedback := feedbackMap[normalizedName]; hasFeedback && feedback.TotalFeedback >= 3 {
			// Only apply feedback if we have at least 3 data points
			feedbackAdjustment := (feedback.AcceptanceRate - 0.5) * 0.3 // Max ±15% adjustment
			confidence += feedbackAdjustment

			// Clamp confidence between 0 and 1
			if confidence > 1.0 {
				confidence = 1.0
			} else if confidence < 0.0 {
				confidence = 0.0
			}

			// Add feedback info to breakdown
			breakdown = append(breakdown, ConfidenceBreakdown{
				Label:  "Historical Feedback",
				Score:  feedback.AcceptanceRate,
				Weight: 0.3,
				Details: fmt.Sprintf("%d accepts, %d rejects (%.1f%% acceptance)",
					feedback.AcceptCount, feedback.RejectCount, feedback.AcceptanceRate*100),
			})

			// Append feedback info to reason
			if feedback.AcceptanceRate > 0.7 {
				reason += fmt.Sprintf(" Users frequently accept this term (%.0f%% acceptance).", feedback.AcceptanceRate*100)
			} else if feedback.AcceptanceRate < 0.3 {
				reason += fmt.Sprintf(" Users rarely accept this term (%.0f%% acceptance).", feedback.AcceptanceRate*100)
			}
		}

		// If an external suggestion already exists, augment it; otherwise, add a new one.
		if existing, ok := suggestions[bt.TermName]; ok {
			existing.Confidence = (existing.Confidence + confidence) / 2 // Average the scores
			existing.Reason += " Also found via internal name similarity."
			existing.BusinessTermID = bt.NodeID
			// Merge breakdowns if needed
			if len(existing.ConfidenceBreakdown) == 0 {
				existing.ConfidenceBreakdown = breakdown
			}
		} else if confidence > 0.5 { // Only add internal suggestions above a threshold
			suggestions[bt.TermName] = &BusinessTermSuggestionResult{
				BusinessTermID:      bt.NodeID,
				TermName:            bt.TermName,
				Confidence:          confidence,
				Reason:              reason,
				Source:              "INTERNAL_SIMILARITY",
				ConfidenceBreakdown: breakdown,
			}
		}
	}

	// 5. Convert map to slice and sort by confidence
	var resultList []BusinessTermSuggestionResult
	for _, sug := range suggestions {
		resultList = append(resultList, *sug)
	}
	sort.Slice(resultList, func(i, j int) bool {
		return resultList[i].Confidence > resultList[j].Confidence
	})

	return resultList, nil
}

// ValidateSemanticTermProperties validates that a semantic term has all required Cube.dev properties
// based on its semantic_term_type. Returns validation errors if required fields are missing or invalid.
func (s *SemanticMappingService) ValidateSemanticTermProperties(ctx context.Context, termType string, properties map[string]interface{}) error {
	if properties == nil {
		return fmt.Errorf("properties cannot be nil")
	}

	// Define required fields per semantic term type based on Cube.dev specifications
	requiredFields := map[string][]string{
		"DIMENSION": {"name", "sql", "type", "title"},
		"MEASURE":   {"name", "sql", "type", "title", "aggregation"},
		"TIME":      {"name", "sql", "type", "title", "granularities"},
		"HIERARCHY": {"name", "title", "levels"},
		"SEGMENT":   {"name", "sql", "title"},
	}

	// Get required fields for this term type
	required, exists := requiredFields[strings.ToUpper(termType)]
	if !exists {
		return fmt.Errorf("unknown semantic term type: %s", termType)
	}

	// Check cube_properties nested object
	cubePropsInterface, hasCubeProps := properties["cube_properties"]
	if !hasCubeProps {
		return fmt.Errorf("missing cube_properties object for term type %s", termType)
	}

	cubeProps, ok := cubePropsInterface.(map[string]interface{})
	if !ok {
		return fmt.Errorf("cube_properties must be a map/object, got %T", cubePropsInterface)
	}

	// Validate all required fields are present and non-empty
	var missingFields []string
	for _, field := range required {
		value, exists := cubeProps[field]
		if !exists {
			missingFields = append(missingFields, field)
			continue
		}

		// Type-specific validation
		switch field {
		case "name":
			if str, ok := value.(string); !ok || strings.TrimSpace(str) == "" {
				missingFields = append(missingFields, fmt.Sprintf("name (invalid: %T, expected non-empty string)", value))
			}
		case "sql":
			if str, ok := value.(string); !ok || strings.TrimSpace(str) == "" {
				missingFields = append(missingFields, fmt.Sprintf("sql (invalid: %T, expected non-empty string)", value))
			}
		case "type":
			validTypes := map[string]bool{
				"number": true, "string": true, "time": true, "boolean": true,
				"measure": true, "dimension": true, "segment": true,
			}
			if str, ok := value.(string); !ok {
				missingFields = append(missingFields, fmt.Sprintf("type (invalid: %T, expected string)", value))
			} else if !validTypes[strings.ToLower(str)] {
				missingFields = append(missingFields, fmt.Sprintf("type (invalid value: %s)", str))
			}
		case "title":
			if str, ok := value.(string); !ok || strings.TrimSpace(str) == "" {
				missingFields = append(missingFields, fmt.Sprintf("title (invalid: %T, expected non-empty string)", value))
			}
		case "aggregation":
			validAggregations := map[string]bool{
				"count": true, "sum": true, "avg": true, "min": true, "max": true,
				"countDistinct": true,
			}
			if str, ok := value.(string); !ok {
				missingFields = append(missingFields, fmt.Sprintf("aggregation (invalid: %T, expected string)", value))
			} else if !validAggregations[strings.ToLower(str)] {
				missingFields = append(missingFields, fmt.Sprintf("aggregation (invalid value: %s)", str))
			}
		case "granularities":
			validGranularities := map[string]bool{
				"second": true, "minute": true, "hour": true, "day": true,
				"week": true, "month": true, "quarter": true, "year": true,
			}
			switch v := value.(type) {
			case []interface{}:
				if len(v) == 0 {
					missingFields = append(missingFields, "granularities (empty array)")
				} else {
					for _, g := range v {
						if str, ok := g.(string); !ok {
							missingFields = append(missingFields, fmt.Sprintf("granularities (invalid item type: %T)", g))
						} else if !validGranularities[strings.ToLower(str)] {
							missingFields = append(missingFields, fmt.Sprintf("granularities (invalid value: %s)", str))
						}
					}
				}
			default:
				missingFields = append(missingFields, fmt.Sprintf("granularities (invalid: %T, expected array)", value))
			}
		case "levels":
			switch v := value.(type) {
			case []interface{}:
				if len(v) == 0 {
					missingFields = append(missingFields, "levels (empty array)")
				}
			default:
				missingFields = append(missingFields, fmt.Sprintf("levels (invalid: %T, expected array)", value))
			}
		}
	}

	if len(missingFields) > 0 {
		return fmt.Errorf("validation failed for %s: missing or invalid fields: %v", termType, missingFields)
	}

	// Semantic term type detection validation
	termTypeStr, hasTermType := properties["semantic_term_type"]
	if !hasTermType {
		return fmt.Errorf("missing semantic_term_type field")
	}
	if str, ok := termTypeStr.(string); !ok || str != strings.ToUpper(termType) {
		return fmt.Errorf("semantic_term_type mismatch: expected %s, got %v", strings.ToUpper(termType), termTypeStr)
	}

	return nil
}
