package analytics

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// EnhancedSemanticMatcher provides advanced semantic term matching with
// profile data integration and abbreviation handling
type EnhancedSemanticMatcher struct {
	db               *sqlx.DB
	abbreviationSvc  *AbbreviationService
	abbreviationMap  map[string]string // Fallback for legacy support
	semanticPatterns map[string]*regexp.Regexp
}

// ColumnProfile represents enriched column profiling data
type ColumnProfile struct {
	DataSource       string    `db:"datasource" json:"datasource"`
	Schema           string    `db:"schema" json:"schema"`
	TableName        string    `db:"table_name" json:"table_name"`
	ColumnName       string    `db:"column_name" json:"column_name"`
	DataType         string    `db:"data_type" json:"data_type"`
	Cardinality      int64     `db:"cardinality" json:"cardinality"`
	MinLength        int       `db:"min_length" json:"min_length"`
	MaxLength        int       `db:"max_length" json:"max_length"`
	AvgLength        float64   `db:"avg_length" json:"avg_length"`
	MinValue         float64   `db:"min_value" json:"min_value"`
	MaxValue         float64   `db:"max_value" json:"max_value"`
	AvgValue         float64   `db:"avg_value" json:"avg_value"`
	StdDev           float64   `db:"std_dev" json:"std_dev"`
	FrequentValues   []string  `db:"frequent_values" json:"frequent_values"`
	InferredPatterns []string  `db:"inferred_patterns" json:"inferred_patterns"`
	BloomFilter      []byte    `db:"bloom_filter" json:"-"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	NullCount        int64     `json:"null_count,omitempty"`
	SampleSize       int       `json:"sample_size,omitempty"`
}

// EnhancedMatchResult includes profile-based confidence factors
type EnhancedMatchResult struct {
	SemanticTerm      string            `json:"semantic_term"`
	SemanticTermID    string            `json:"semantic_term_id,omitempty"`
	Confidence        float64           `json:"confidence"`
	NameConfidence    float64           `json:"name_confidence"`
	ProfileConfidence float64           `json:"profile_confidence"`
	TypeConfidence    float64           `json:"type_confidence"`
	MatchReason       string            `json:"match_reason"`
	IsNewTerm         bool              `json:"is_new_term"`
	ProfileMatch      *ProfileMatchInfo `json:"profile_match,omitempty"`
}

// ProfileMatchInfo provides details about profile-based matching
type ProfileMatchInfo struct {
	ValueOverlap     float64 `json:"value_overlap"`
	PatternOverlap   float64 `json:"pattern_overlap"`
	CardinalityMatch float64 `json:"cardinality_match"`
	DataTypeMatch    bool    `json:"data_type_match"`
	StatisticalMatch float64 `json:"statistical_match"`
	BloomMatch       bool    `json:"bloom_match"`
}

// NewEnhancedSemanticMatcher creates a new enhanced matcher
func NewEnhancedSemanticMatcher(db *sqlx.DB) *EnhancedSemanticMatcher {
	matcher := &EnhancedSemanticMatcher{
		db:               db,
		abbreviationMap:  initializeAbbreviationMap(),
		semanticPatterns: initializeSemanticPatterns(),
	}
	return matcher
}

// NewEnhancedSemanticMatcherWithAbbreviations creates a new enhanced matcher with abbreviation service
func NewEnhancedSemanticMatcherWithAbbreviations(db *sqlx.DB, abbreviationSvc *AbbreviationService) *EnhancedSemanticMatcher {
	matcher := &EnhancedSemanticMatcher{
		db:               db,
		abbreviationSvc:  abbreviationSvc,
		abbreviationMap:  initializeAbbreviationMap(), // Fallback
		semanticPatterns: initializeSemanticPatterns(),
	}
	return matcher
}

// initializeAbbreviationMap creates a mapping of common abbreviations to full terms
func initializeAbbreviationMap() map[string]string {
	return map[string]string{
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
		"CLS":  "CLASS",

		// Common prefixes/suffixes
		"DESC": "DESCRIPTION",
		"NM":   "NAME",
		"TTL":  "TOTAL",
		"AVG":  "AVERAGE",
		"MIN":  "MINIMUM",
		"MAX":  "MAXIMUM",
		"SUM":  "SUMMARY",
	}
}

// initializeSemanticPatterns creates regex patterns for semantic recognition
func initializeSemanticPatterns() map[string]*regexp.Regexp {
	patterns := make(map[string]*regexp.Regexp)

	// Email patterns
	patterns["EMAIL"] = regexp.MustCompile(`(?i)(email|e_mail|mail|email_addr)`)

	// Phone patterns
	patterns["PHONE"] = regexp.MustCompile(`(?i)(phone|tel|telephone|mobile|cell)`)

	// Address patterns
	patterns["ADDRESS"] = regexp.MustCompile(`(?i)(addr|address|street|avenue|blvd|boulevard)`)

	// Date patterns
	patterns["DATE"] = regexp.MustCompile(`(?i)(date|dt|day|created_at|updated_at|modified)`)

	// Amount/Financial patterns
	patterns["AMOUNT"] = regexp.MustCompile(`(?i)(amt|amount|price|cost|value|balance|total|sum)`)

	// Identifier patterns
	patterns["IDENTIFIER"] = regexp.MustCompile(`(?i)(_id|_key|_num|_no|_nbr|_code|_cd)$`)

	return patterns
}

// DatabaseColumn represents a column from the catalog (imported from semantic_mapping_service)
type DatabaseColumnRef = DatabaseColumn

// SemanticTerm represents a semantic term node (imported from semantic_mapping_service)
type SemanticTermRef = SemanticTerm

// EnhancedMatchColumn performs advanced semantic matching for a database column
func (m *EnhancedSemanticMatcher) EnhancedMatchColumn(
	ctx context.Context,
	column *DatabaseColumnRef,
	existingTerms []SemanticTermRef,
	tenantID, datasourceID string,
) ([]EnhancedMatchResult, error) {

	// Get column profile data
	profile, err := m.getColumnProfile(ctx, column, tenantID, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to get profile for column %s.%s.%s: %v",
			column.Schema, column.Table, column.Column, err)
	}

	var results []EnhancedMatchResult

	// Generate expanded column names (with abbreviation expansion)
	expandedNames := m.expandAbbreviations(column.Column)

	// Score against each existing semantic term
	for _, term := range existingTerms {
		bestMatch := m.calculateEnhancedConfidence(column, &term, profile, expandedNames)
		if bestMatch.Confidence > 0.3 { // Only include reasonable matches
			results = append(results, bestMatch)
		}
	}

	// If no good matches found, suggest creating a new semantic term
	if len(results) == 0 || (len(results) > 0 && results[0].Confidence < 0.75) {
		newTermSuggestion := m.generateNewTermSuggestion(column, profile, expandedNames)
		results = append(results, newTermSuggestion)
	}

	// Sort by confidence descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results, nil
}

// expandAbbreviations creates variations of the column name with expanded abbreviations
func (m *EnhancedSemanticMatcher) expandAbbreviations(columnName string) []string {
	// Try database-backed abbreviation service first
	if m.abbreviationSvc != nil {
		ctx := context.Background()
		if expansions, err := m.abbreviationSvc.ExpandAbbreviations(ctx, columnName); err == nil && len(expansions) > 0 {
			return expansions
		}
	}

	// Fallback to legacy hardcoded expansion
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
		if expansion, exists := m.abbreviationMap[token]; exists {
			tokenVariations = append(tokenVariations, expansion)
			hasExpansion = true
		}
		expandedTokens = append(expandedTokens, tokenVariations)
	}

	// Generate all combinations if we have expansions
	if hasExpansion {
		combinations := m.generateCombinations(expandedTokens)
		for _, combo := range combinations {
			variations = append(variations, strings.Join(combo, "_"))
		}
	}

	return variations
}

// generateCombinations creates all possible combinations from token variations
func (m *EnhancedSemanticMatcher) generateCombinations(tokenVariations [][]string) [][]string {
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
	restCombos := m.generateCombinations(tokenVariations[1:])

	for _, variation := range tokenVariations[0] {
		for _, restCombo := range restCombos {
			combo := append([]string{variation}, restCombo...)
			result = append(result, combo)
		}
	}

	return result
}

// calculateEnhancedConfidence computes confidence using name, profile, and type matching
func (m *EnhancedSemanticMatcher) calculateEnhancedConfidence(
	column *DatabaseColumn,
	term *SemanticTerm,
	profile *ColumnProfile,
	columnNameVariations []string,
) EnhancedMatchResult {

	// 1. Calculate name-based confidence with abbreviation support
	nameConfidence := m.calculateNameConfidenceWithAbbreviations(columnNameVariations, term.TermName)

	// 2. Calculate profile-based confidence if profile data is available
	profileConfidence := 0.0
	var profileMatch *ProfileMatchInfo
	if profile != nil {
		profileConfidence, profileMatch = m.calculateProfileConfidence(profile, term)
	}

	// 3. Calculate data type compatibility confidence
	typeConfidence := m.calculateDataTypeConfidence(column.DataType, term.DataType)

	// 4. Combine confidences with weighted average
	// Name matching is most important (50%), profile data adds significant value (35%),
	// data type compatibility is supportive (15%)
	finalConfidence := (nameConfidence * 0.50) + (profileConfidence * 0.35) + (typeConfidence * 0.15)

	// 5. Build match reason
	reason := m.buildMatchReason(nameConfidence, profileConfidence, typeConfidence, profileMatch)

	return EnhancedMatchResult{
		SemanticTerm:      term.TermName,
		SemanticTermID:    term.NodeID,
		Confidence:        finalConfidence,
		NameConfidence:    nameConfidence,
		ProfileConfidence: profileConfidence,
		TypeConfidence:    typeConfidence,
		MatchReason:       reason,
		IsNewTerm:         false,
		ProfileMatch:      profileMatch,
	}
}

// calculateNameConfidenceWithAbbreviations computes name similarity including abbreviation expansion
func (m *EnhancedSemanticMatcher) calculateNameConfidenceWithAbbreviations(
	columnNameVariations []string,
	termName string,
) float64 {
	normalizedTermName := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(termName, "_", ""), "-", ""))

	bestScore := 0.0

	for _, variation := range columnNameVariations {
		normalizedVariation := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(variation, "_", ""), "-", ""))

		// Exact match
		if normalizedVariation == normalizedTermName {
			bestScore = 1.0
			break
		}

		// Calculate similarity using multiple approaches

		// 1. Levenshtein distance
		levDistance := m.levenshteinDistance(normalizedVariation, normalizedTermName)
		maxLen := maxInt(len(normalizedVariation), len(normalizedTermName))
		levScore := 0.0
		if maxLen > 0 {
			levScore = 1.0 - (float64(levDistance) / float64(maxLen))
		}

		// 2. Jaccard similarity on tokens
		variationTokens := m.tokenizeSemanticTerm(variation)
		termTokens := m.tokenizeSemanticTerm(termName)
		jaccardScore := m.calculateJaccardSimilarity(variationTokens, termTokens)

		// 3. Substring matching bonus
		substringBonus := 0.0
		if strings.Contains(normalizedTermName, normalizedVariation) ||
			strings.Contains(normalizedVariation, normalizedTermName) {
			substringBonus = 0.2
		}

		// 4. Check semantic patterns
		patternBonus := m.calculatePatternBonus(variation, termName)

		// Combine scores
		combinedScore := (levScore * 0.4) + (jaccardScore * 0.4) + substringBonus + patternBonus
		if combinedScore > bestScore {
			bestScore = combinedScore
		}
	}

	return minFloat64(bestScore, 1.0)
}

// calculatePatternBonus gives bonus for matching semantic patterns
func (m *EnhancedSemanticMatcher) calculatePatternBonus(columnName, termName string) float64 {
	bonus := 0.0

	for _, pattern := range m.semanticPatterns {
		columnMatches := pattern.MatchString(columnName)
		termMatches := pattern.MatchString(termName)

		if columnMatches && termMatches {
			bonus += 0.15
		}
	}

	return minFloat64(bonus, 0.3) // Cap the bonus
}

// calculateProfileConfidence uses column profiling data to assess semantic similarity
func (m *EnhancedSemanticMatcher) calculateProfileConfidence(
	profile *ColumnProfile,
	term *SemanticTerm,
) (float64, *ProfileMatchInfo) {

	matchInfo := &ProfileMatchInfo{}
	confidence := 0.0

	// 1. Frequent values overlap
	if len(profile.FrequentValues) > 0 && len(term.ReferenceValues) > 0 {
		overlap := m.calculateSetOverlap(profile.FrequentValues, term.ReferenceValues)
		matchInfo.ValueOverlap = overlap
		confidence += overlap * 0.4 // Strong indicator
	}

	// 2. Inferred patterns overlap
	if len(profile.InferredPatterns) > 0 && len(term.ReferencePatterns) > 0 {
		overlap := m.calculateSetOverlap(profile.InferredPatterns, term.ReferencePatterns)
		matchInfo.PatternOverlap = overlap
		confidence += overlap * 0.3
	}

	// 3. Cardinality similarity
	if profile.Cardinality > 0 {
		// Get expected cardinality for the semantic term (could be stored as metadata)
		// For now, use heuristics based on term name
		expectedCardinality := m.estimateExpectedCardinality(term.TermName)
		if expectedCardinality > 0 {
			cardinalityRatio := float64(minInt(int(profile.Cardinality), expectedCardinality)) /
				float64(maxInt(int(profile.Cardinality), expectedCardinality))
			matchInfo.CardinalityMatch = cardinalityRatio
			confidence += cardinalityRatio * 0.2
		}
	}

	// 4. Data type compatibility
	dataTypeMatch := m.areDataTypesCompatible(profile.DataType, term.DataType)
	matchInfo.DataTypeMatch = dataTypeMatch
	if dataTypeMatch {
		confidence += 0.1
	}

	return minFloat64(confidence, 1.0), matchInfo
}

// estimateExpectedCardinality provides heuristic cardinality estimates based on semantic term names
func (m *EnhancedSemanticMatcher) estimateExpectedCardinality(termName string) int {
	termLower := strings.ToLower(termName)

	// High cardinality terms
	if strings.Contains(termLower, "id") || strings.Contains(termLower, "key") ||
		strings.Contains(termLower, "email") || strings.Contains(termLower, "phone") {
		return 100000
	}

	// Medium cardinality terms
	if strings.Contains(termLower, "name") || strings.Contains(termLower, "code") ||
		strings.Contains(termLower, "number") {
		return 1000
	}

	// Low cardinality terms
	if strings.Contains(termLower, "type") || strings.Contains(termLower, "status") ||
		strings.Contains(termLower, "category") || strings.Contains(termLower, "flag") {
		return 50
	}

	// Country/State level
	if strings.Contains(termLower, "country") || strings.Contains(termLower, "state") {
		return 250
	}

	return 0 // Unknown
}

// calculateDataTypeConfidence assesses data type compatibility
func (m *EnhancedSemanticMatcher) calculateDataTypeConfidence(columnType, termType string) float64 {
	if columnType == "" || termType == "" {
		return 0.5 // Neutral when types unknown
	}

	normalizedColumnType := m.normalizeDataType(columnType)
	normalizedTermType := m.normalizeDataType(termType)

	if normalizedColumnType == normalizedTermType {
		return 1.0
	}

	// Check for compatible types
	if m.areDataTypesCompatible(columnType, termType) {
		return 0.8
	}

	return 0.2 // Incompatible types
}

// generateNewTermSuggestion creates a suggestion for a new semantic term
func (m *EnhancedSemanticMatcher) generateNewTermSuggestion(
	column *DatabaseColumn,
	profile *ColumnProfile,
	expandedNames []string,
) EnhancedMatchResult {

	// Choose the best expanded name - prefer fully expanded versions over originals
	bestName := column.Column
	originalName := strings.ToUpper(column.Column)

	// Look for the best expanded version
	for _, name := range expandedNames {
		if name == originalName {
			continue // Skip the original name
		}

		// Prefer names with more expansions (more words/underscores typically means more expansion)
		nameWordCount := strings.Count(name, "_") + 1
		bestWordCount := strings.Count(bestName, "_") + 1

		if nameWordCount > bestWordCount ||
			(nameWordCount == bestWordCount && len(name) > len(bestName)) {
			bestName = name
		}
	}

	// If we didn't find a better expansion, use the longest one
	if bestName == column.Column {
		for _, name := range expandedNames {
			if len(name) > len(bestName) {
				bestName = name
			}
		}
	}

	// Confidence for new term based on how well we can understand the column
	confidence := 0.6 // Base confidence for new terms

	// Boost confidence if we have good profile data
	if profile != nil {
		if len(profile.InferredPatterns) > 0 {
			confidence += 0.1
		}
		if len(profile.FrequentValues) > 0 && profile.Cardinality > 0 {
			confidence += 0.1
		}
	}

	// Boost confidence if name was expanded from abbreviations
	if len(expandedNames) > 1 && bestName != strings.ToUpper(column.Column) {
		confidence += 0.15 // Higher boost for actual expansions
	}

	reason := fmt.Sprintf("New term suggestion with expanded name: %s", bestName)
	if bestName != strings.ToUpper(column.Column) {
		reason += fmt.Sprintf(" (expanded from: %s)", column.Column)
	}
	if profile != nil {
		reason += fmt.Sprintf(" (cardinality: %d)", profile.Cardinality)
	}

	return EnhancedMatchResult{
		SemanticTerm:      bestName,
		SemanticTermID:    "",                           // Will be generated when created
		Confidence:        minFloat64(confidence, 0.95), // Cap new term confidence
		NameConfidence:    0.8,
		ProfileConfidence: 0.0,
		TypeConfidence:    0.5,
		MatchReason:       reason,
		IsNewTerm:         true,
	}
}

// getColumnProfile retrieves profiling data for a column
func (m *EnhancedSemanticMatcher) getColumnProfile(
	ctx context.Context,
	column *DatabaseColumn,
	tenantID, datasourceID string,
) (*ColumnProfile, error) {

	query := `
		SELECT datasource, schema, table_name, column_name, data_type, cardinality,
		       min_length, max_length, avg_length, min_value, max_value, avg_value,
		       std_dev, frequent_values, inferred_patterns, bloom_filter, created_at
		FROM column_profiles 
		WHERE schema = $1 AND table_name = $2 AND column_name = $3
		ORDER BY created_at DESC 
		LIMIT 1
	`

	var profile ColumnProfile
	var frequentValuesJSON, inferredPatternsJSON []byte

	err := m.db.QueryRowContext(ctx, query,
		column.Schema, column.Table, column.Column).Scan(
		&profile.DataSource, &profile.Schema, &profile.TableName, &profile.ColumnName,
		&profile.DataType, &profile.Cardinality, &profile.MinLength, &profile.MaxLength,
		&profile.AvgLength, &profile.MinValue, &profile.MaxValue, &profile.AvgValue,
		&profile.StdDev, &frequentValuesJSON, &inferredPatternsJSON,
		&profile.BloomFilter, &profile.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No profile data available
		}
		return nil, fmt.Errorf("failed to query column profile: %w", err)
	}

	// Parse JSON arrays
	if len(frequentValuesJSON) > 0 {
		if err := json.Unmarshal(frequentValuesJSON, &profile.FrequentValues); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to unmarshal frequent values: %v", err)
		}
	}

	if len(inferredPatternsJSON) > 0 {
		if err := json.Unmarshal(inferredPatternsJSON, &profile.InferredPatterns); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to unmarshal inferred patterns: %v", err)
		}
	}

	return &profile, nil
}

// buildMatchReason creates a human-readable explanation of the match
func (m *EnhancedSemanticMatcher) buildMatchReason(
	nameConf, profileConf, typeConf float64,
	profileMatch *ProfileMatchInfo,
) string {
	var reasons []string

	if nameConf > 0.8 {
		reasons = append(reasons, "Strong name similarity")
	} else if nameConf > 0.6 {
		reasons = append(reasons, "Good name similarity")
	} else if nameConf > 0.4 {
		reasons = append(reasons, "Moderate name similarity")
	}

	if profileMatch != nil {
		if profileMatch.ValueOverlap > 0.5 {
			reasons = append(reasons, fmt.Sprintf("%.0f%% value overlap", profileMatch.ValueOverlap*100))
		}
		if profileMatch.PatternOverlap > 0.5 {
			reasons = append(reasons, fmt.Sprintf("%.0f%% pattern overlap", profileMatch.PatternOverlap*100))
		}
		if profileMatch.CardinalityMatch > 0.8 {
			reasons = append(reasons, "Similar cardinality")
		}
		if profileMatch.DataTypeMatch {
			reasons = append(reasons, "Compatible data types")
		}
	} else if typeConf > 0.8 {
		reasons = append(reasons, "Compatible data types")
	}

	if len(reasons) == 0 {
		return "Low confidence match"
	}

	return strings.Join(reasons, ", ")
}

// Utility methods (these can be shared with the original semantic mapping service)

func (m *EnhancedSemanticMatcher) calculateSetOverlap(set1, set2 []string) float64 {
	if len(set1) == 0 || len(set2) == 0 {
		return 0.0
	}

	// Convert to maps for faster lookup
	map1 := make(map[string]struct{})
	for _, item := range set1 {
		map1[strings.ToUpper(item)] = struct{}{}
	}

	intersection := 0
	for _, item := range set2 {
		if _, exists := map1[strings.ToUpper(item)]; exists {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (m *EnhancedSemanticMatcher) levenshteinDistance(s1, s2 string) int {
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

			matrix[i][j] = minInt3(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

func (m *EnhancedSemanticMatcher) tokenizeSemanticTerm(term string) []string {
	// Split on underscores, spaces, and camelCase boundaries
	term = strings.ReplaceAll(term, "_", " ")
	term = strings.ReplaceAll(term, "-", " ")

	// Handle camelCase
	camelRegex := regexp.MustCompile("([a-z])([A-Z])")
	term = camelRegex.ReplaceAllString(term, "$1 $2")

	tokens := strings.Fields(strings.ToUpper(term))

	// Filter out very short tokens
	var filtered []string
	for _, token := range tokens {
		if len(token) >= 2 {
			filtered = append(filtered, token)
		}
	}

	return filtered
}

func (m *EnhancedSemanticMatcher) calculateJaccardSimilarity(tokens1, tokens2 []string) float64 {
	if len(tokens1) == 0 && len(tokens2) == 0 {
		return 1.0
	}
	if len(tokens1) == 0 || len(tokens2) == 0 {
		return 0.0
	}

	set1 := make(map[string]struct{})
	for _, token := range tokens1 {
		set1[token] = struct{}{}
	}

	intersection := 0
	set2 := make(map[string]struct{})
	for _, token := range tokens2 {
		set2[token] = struct{}{}
		if _, exists := set1[token]; exists {
			intersection++
		}
	}

	union := len(set1) + len(set2) - intersection
	if union == 0 {
		return 0.0
	}

	return float64(intersection) / float64(union)
}

func (m *EnhancedSemanticMatcher) normalizeDataType(dataType string) string {
	lower := strings.ToLower(strings.TrimSpace(dataType))

	// Normalize common variations
	if strings.Contains(lower, "varchar") || strings.Contains(lower, "text") || strings.Contains(lower, "char") {
		return "text"
	}
	if strings.Contains(lower, "int") || strings.Contains(lower, "bigint") || strings.Contains(lower, "smallint") {
		return "integer"
	}
	if strings.Contains(lower, "float") || strings.Contains(lower, "double") || strings.Contains(lower, "decimal") || strings.Contains(lower, "numeric") {
		return "numeric"
	}
	if strings.Contains(lower, "timestamp") || strings.Contains(lower, "datetime") {
		return "timestamp"
	}
	if strings.Contains(lower, "date") {
		return "date"
	}
	if strings.Contains(lower, "bool") {
		return "boolean"
	}

	return lower
}

func (m *EnhancedSemanticMatcher) areDataTypesCompatible(type1, type2 string) bool {
	norm1 := m.normalizeDataType(type1)
	norm2 := m.normalizeDataType(type2)

	if norm1 == norm2 {
		return true
	}

	// Define compatible type groups
	numericTypes := map[string]bool{"integer": true, "numeric": true, "float": true}
	dateTypes := map[string]bool{"date": true, "timestamp": true}
	textTypes := map[string]bool{"text": true, "varchar": true, "char": true}

	if numericTypes[norm1] && numericTypes[norm2] {
		return true
	}
	if dateTypes[norm1] && dateTypes[norm2] {
		return true
	}
	if textTypes[norm1] && textTypes[norm2] {
		return true
	}

	return false
}

// Utility functions
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func minInt3(a, b, c int) int {
	if a < b && a < c {
		return a
	}
	if b < c {
		return b
	}
	return c
}
