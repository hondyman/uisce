package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

// suggestSemanticTermWithLLM uses LLM to suggest a better semantic term name and properties
func (s *SemanticMappingService) suggestSemanticTermWithLLM(ctx context.Context, col *DatabaseColumn, expandedName, currentSuggestion string) (SemanticSuggestion, error) {
	emptySuggestion := SemanticSuggestion{}
	if s.llmProvider == nil {
		return emptySuggestion, fmt.Errorf("LLM provider not available")
	}

	// Format sample values for prompt (limit to 10)
	sampleVals := col.SampleValues
	if len(sampleVals) > 10 {
		sampleVals = sampleVals[:10]
	}
	sampleStr := strings.Join(sampleVals, ", ")
	if sampleStr == "" {
		sampleStr = "(none)"
	}

	prompt := fmt.Sprintf(`Given a database column with the following information:
- Expanded column name: %s
- Table name: %s
- Data Type: %s
- Statistics: Unique Values: %d, Total Rows: %d
- Sample Values: [%s]
- Current suggestion: %s

You are generating semantic metadata for a Cube.dev semantic layer. Suggest properties following these rules:

1. Term Name: Use UPPERCASE with underscores (e.g., CUSTOMER_ACCOUNT_BALANCE). Be descriptive and business-friendly. Avoid redundant prefixes.
2. Title: A human-readable title for reports and dashboards (e.g., "Customer Account Balance").
3. Description: A business-friendly description of what this data represents and how it should be used.
4. Semantic Type: Choose one based on the column's PURPOSE and STATS:
   - "dimension": For categorical/descriptive attributes (names, IDs, codes, categories, status flags). Low cardinality relative to total count often implies dimension.
   - "measure": For numeric values that should be aggregated (amounts, totals, quantities, counts). High cardinality numeric often (but not always) implies measure.
   - "time_dimension": For date/time columns used for time-based analysis
5. Meta: Cube.dev-compatible metadata properties:
   - "type": The Cube.dev data type: "string", "number", "time", or "boolean"
   - "primaryKey": true if this appears to be a primary key column (Unique count == Total count, and name suggests ID)
   - "shown": true/false - whether to show by default in explorers
   - "drillMembers": array of related dimension names for drill-down (if applicable)
6. Format: Display format for BI tools ("currency", "percent", "date", "datetime", or empty for default)

Return strictly valid JSON matching this structure:
{
  "term_name": "...",
  "title": "...",
  "description": "...",
  "semantic_type": "dimension",
  "meta": {"type": "string", "primaryKey": false, "shown": true},
  "format": ""
}
Return ONLY the raw JSON string, no markdown formatting.`, expandedName, col.Table, col.DataType, col.UniqueCount, col.TotalCount, sampleStr, currentSuggestion)

	// Type assert to LLM provider interface
	llmProvider, ok := s.llmProvider.(interface {
		GenerateContent(context.Context, string) (string, error)
	})
	if !ok {
		return emptySuggestion, fmt.Errorf("invalid LLM provider type")
	}

	result, err := llmProvider.GenerateContent(ctx, prompt)
	if err != nil {
		return emptySuggestion, err
	}

	// Clean up the result (remove potential markdown code blocks)
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var suggestion SemanticSuggestion
	if err := json.Unmarshal([]byte(result), &suggestion); err != nil {
		// Fallback: treat entire result as term name if JSON parsing fails
		logging.GetLogger().Sugar().Warnf("Failed to parse LLM JSON: %v. Raw: %s", err, result)
		suggestion.TermName = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(result), " ", "_"))
	} else {
		suggestion.TermName = strings.ToUpper(strings.ReplaceAll(suggestion.TermName, " ", "_"))
	}

	return suggestion, nil
}

// calculateMappingConfidence calculates confidence score for a mapping
func (s *SemanticMappingService) calculateMappingConfidence(ctx context.Context, col *DatabaseColumn, expandedName, semanticTerm, tenantID, datasourceID string) (float64, string) {
	var reasoning strings.Builder
	totalScore := 0.0

	// 1. Abbreviation expansion confidence (0.1 weight)
	expansionScore := 0.5
	if expandedName != col.Column {
		expansionScore = 0.9
		reasoning.WriteString(fmt.Sprintf("Expanded '%s' to '%s'. ", col.Column, expandedName))
	}
	totalScore += expansionScore * 0.1

	// 2. Fuzzy match with existing terms (0.2 weight)
	terms, err := s.fetchSemanticTerms(ctx, tenantID, datasourceID)
	fuzzyScore := 0.3
	if err == nil && len(terms) > 0 {
		for _, term := range terms {
			similarity := s.calculateStringSimilarity(semanticTerm, term.TermName)
			if similarity > fuzzyScore {
				fuzzyScore = similarity
				if similarity > 0.8 {
					reasoning.WriteString(fmt.Sprintf("High similarity (%.2f) with existing term '%s'. ", similarity, term.TermName))
				}
			}
		}
	}
	totalScore += fuzzyScore * 0.2

	// 3. Data type consistency (0.1 weight)
	dataTypeScore := 0.7
	if col.DataType != "" {
		dataTypeScore = 0.8
		reasoning.WriteString(fmt.Sprintf("Data type: %s. ", col.DataType))
	}
	totalScore += dataTypeScore * 0.1

	// 4. Column name quality (0.1 weight)
	qualityScore := s.assessColumnNameQuality(col.Column)
	totalScore += qualityScore * 0.1

	// 5. Lookup matching (0.2 weight)
	lookupScore := 0.0
	if col.SampleValues != nil && len(col.SampleValues) > 0 {
		matchedLookup, matchRate := s.matchAgainstLookups(ctx, tenantID, col.SampleValues)
		if matchRate > 0.5 {
			lookupScore = matchRate
			reasoning.WriteString(fmt.Sprintf("Values match lookup '%s' (%.0f%% match). ", matchedLookup, matchRate*100))
			col.MatchedLookup = matchedLookup
		}
	}
	totalScore += lookupScore * 0.2

	// 6. Data Profile Confidence (0.3 weight) - NEW!
	profileScore := 0.0
	if col.TotalCount > 0 {
		profileScore += 0.5
		if col.UniqueCount > 0 {
			profileScore += 0.3
		}
		if len(col.SampleValues) > 0 {
			profileScore += 0.2
		}
		reasoning.WriteString(fmt.Sprintf(" verified by data profile (%d rows). ", col.TotalCount))
	} else {
		reasoning.WriteString(" (no data profile). ")
	}
	totalScore += profileScore * 0.3

	// Cap score at 1.0
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore, reasoning.String()
}

// matchAgainstLookups compares sample values against lookup_values table
func (s *SemanticMappingService) matchAgainstLookups(ctx context.Context, tenantID string, sampleValues []string) (string, float64) {
	if len(sampleValues) == 0 {
		return "", 0
	}

	// Get all lookups for this tenant
	type lookupInfo struct {
		ID     string   `db:"id"`
		Name   string   `db:"lookup_name"`
		Values []string // to be populated
	}

	lookupsQuery := `
		SELECT l.id, l.lookup_name 
		FROM lookups l 
		WHERE l.tenant_id = $1
	`
	rows, err := s.db.QueryContext(ctx, lookupsQuery, tenantID)
	if err != nil {
		return "", 0
	}
	defer rows.Close()

	var lookups []lookupInfo
	for rows.Next() {
		var l lookupInfo
		if err := rows.Scan(&l.ID, &l.Name); err != nil {
			continue
		}
		lookups = append(lookups, l)
	}

	// For each lookup, get values and check overlap
	bestMatch := ""
	bestRate := 0.0

	for _, lookup := range lookups {
		valuesQuery := `SELECT value FROM lookup_values WHERE lookup_id = $1`
		valRows, err := s.db.QueryContext(ctx, valuesQuery, lookup.ID)
		if err != nil {
			continue
		}

		lookupVals := make(map[string]bool)
		for valRows.Next() {
			var v string
			if err := valRows.Scan(&v); err == nil {
				lookupVals[strings.ToLower(v)] = true
			}
		}
		valRows.Close()

		if len(lookupVals) == 0 {
			continue
		}

		matchCount := 0
		for _, sv := range sampleValues {
			if lookupVals[strings.ToLower(sv)] {
				matchCount++
			}
		}

		rate := float64(matchCount) / float64(len(sampleValues))
		if rate > bestRate {
			bestRate = rate
			bestMatch = lookup.Name
		}
	}

	return bestMatch, bestRate
}

// findCrossDatasourceTerm checks if the same column name is mapped to a term in another datasource
func (s *SemanticMappingService) findCrossDatasourceTerm(ctx context.Context, tenantID, columnName string) (string, bool) {
	// Query for existing mappings of the same column name across all datasources for this tenant
	query := `
		SELECT cn_term.node_name as term_name
		FROM catalog_edge ce
		JOIN catalog_node cn_col ON ce.source_node_id = cn_col.id
		JOIN catalog_node cn_term ON ce.target_node_id = cn_term.id
		WHERE cn_col.tenant_id = $1
		  AND LOWER(cn_col.node_name) = LOWER($2)
		  AND ce.edge_type = 'MAPS_TO'
		  AND cn_term.node_type_id = $3
		LIMIT 1
	`
	var termName string
	err := s.db.GetContext(ctx, &termName, query, tenantID, columnName, SemanticTermNodeTypeID)
	if err == nil && termName != "" {
		return termName, true
	}
	return "", false
}

// getForeignKeyTargetHint extracts semantic hint from FK target table name
func getForeignKeyTargetHint(columnProps map[string]interface{}) string {
	// Check if column is a foreign key
	if isFk, ok := columnProps["is_foreign_key"].(bool); !ok || !isFk {
		return ""
	}

	// Get target table name
	if targetTable, ok := columnProps["foreign_key_target_table"].(string); ok && targetTable != "" {
		// Extract base name: "schema.table" -> "table"
		parts := strings.Split(targetTable, ".")
		tableName := parts[len(parts)-1]

		// Singularize and clean
		cleaned := strings.TrimSuffix(strings.ToLower(tableName), "s")
		cleaned = strings.ReplaceAll(cleaned, "_", " ")

		return strings.Title(cleaned)
	}

	return ""
}

// getColumnNamingHint extracts semantic hints from column naming conventions
func getColumnNamingHint(columnName string) string {
	upper := strings.ToUpper(columnName)

	// Common suffix patterns that give semantic hints
	suffixHints := map[string]string{
		"_CD":     "Code",
		"_CODE":   "Code",
		"_NM":     "Name",
		"_NAME":   "Name",
		"_DESC":   "Description",
		"_DT":     "Date",
		"_DATE":   "Date",
		"_TS":     "Timestamp",
		"_AMT":    "Amount",
		"_AMOUNT": "Amount",
		"_QTY":    "Quantity",
		"_CNT":    "Count",
		"_NUM":    "Number",
		"_PCT":    "Percentage",
		"_RATE":   "Rate",
		"_ADDR":   "Address",
		"_TEL":    "Telephone",
		"_EMAIL":  "Email",
		"_URL":    "URL",
		"_FLAG":   "Flag",
		"_IND":    "Indicator",
		"_TYPE":   "Type",
		"_STATUS": "Status",
		"_CLASS":  "Class",
		"_CAT":    "Category",
	}

	for suffix, hint := range suffixHints {
		if strings.HasSuffix(upper, suffix) {
			return hint
		}
	}

	return ""
}

// assessColumnNameQuality assesses how well-formed a column name is
func (s *SemanticMappingService) assessColumnNameQuality(columnName string) float64 {
	score := 0.5

	// Penalize single-letter columns
	if len(columnName) <= 2 {
		return 0.2
	}

	// Reward descriptive names
	if len(columnName) >= 5 {
		score += 0.2
	}

	// Reward underscores (structured naming)
	if strings.Contains(columnName, "_") {
		score += 0.2
	}

	// Penalize all caps with no separators
	if columnName == strings.ToUpper(columnName) && !strings.Contains(columnName, "_") {
		score -= 0.1
	}

	return min(1.0, max(0.0, score))
}

// calculateStringSimilarity calculates similarity between two strings (simple implementation)
func (s *SemanticMappingService) calculateStringSimilarity(s1, s2 string) float64 {
	s1 = strings.ToUpper(s1)
	s2 = strings.ToUpper(s2)

	if s1 == s2 {
		return 1.0
	}

	// Simple Levenshtein-based similarity
	longer := s1
	shorter := s2
	if len(s1) < len(s2) {
		longer = s2
		shorter = s1
	}

	if len(longer) == 0 {
		return 1.0
	}

	// Count matching characters
	matches := 0
	for i := 0; i < len(shorter); i++ {
		if strings.Contains(longer, string(shorter[i])) {
			matches++
		}
	}

	return float64(matches) / float64(len(longer))
}

// fetchUnmappedColumns fetches columns that don't have semantic mappings yet
func (s *SemanticMappingService) fetchUnmappedColumns(ctx context.Context, tenantID, datasourceID string) ([]DatabaseColumn, error) {
	query := `
		SELECT 
			cn.id as node_id,
			COALESCE(parent_table.properties->>'schema', '') as schema,
			COALESCE(parent_table.node_name, '') as table,
			cn.node_name as column,
			cn.qualified_path,
			cn.tenant_datasource_id,
			cn.tenant_id,
			COALESCE(cn.properties->>'data_type', '') as data_type,
			COALESCE(cn.properties->'suggested_validation_rules', '{}'::jsonb) as suggested_validation_rules,
			COALESCE((cn.properties->>'unique_count')::bigint, 0) as unique_count,
			COALESCE((cn.properties->>'total_count')::bigint, 0) as total_count,
			COALESCE(cn.properties->'sample_values', '[]'::jsonb) as sample_values_json
		FROM catalog_node cn
		LEFT JOIN catalog_node parent_table ON parent_table.id = cn.parent_id
		WHERE cn.tenant_id = $1
			AND cn.tenant_datasource_id = $2
			AND cn.node_type_id = $3
		ORDER BY parent_table.node_name, cn.node_name
		LIMIT 100
	`

	logger := logging.GetLogger().Sugar()
	logger.Infof("Fetching unmapped columns for tenant=%s, datasource=%s, nodeTypeID=%s", tenantID, datasourceID, DatabaseColumnNodeTypeID)

	var columns []DatabaseColumn
	err := s.db.SelectContext(ctx, &columns, query, tenantID, datasourceID, DatabaseColumnNodeTypeID)

	// Fallback to parsing qualified path if table/schema are still missing
	if err == nil {
		for i := range columns {
			// Populate SampleValues from JSON
			if len(columns[i].SampleValuesJSON) > 0 {
				_ = json.Unmarshal(columns[i].SampleValuesJSON, &columns[i].SampleValues)
			}

			if columns[i].Table == "" || columns[i].Schema == "" {
				sKey, tKey, _ := s.parseQualifiedPath(columns[i].QualifiedPath)
				if columns[i].Table == "" {
					columns[i].Table = tKey
				}
				if columns[i].Schema == "" {
					columns[i].Schema = sKey
				}
			}
		}
	}

	return columns, err
}

// ApplyMappingsBatch applies mappings based on confidence thresholds
func (s *SemanticMappingService) ApplyMappingsBatch(ctx context.Context, req *ApplyMappingsRequest, mappings []GeneratedMapping) (*ApplyMappingsResponse, error) {
	logger := logging.GetLogger().Sugar()

	// Handle field alias
	if req.DatasourceID == "" && req.TenantInstanceID != "" {
		req.DatasourceID = req.TenantInstanceID
	}

	response := &ApplyMappingsResponse{}

	// Resolve datasource ID to ensure we create edges in the correct (physical) datasource
	catalogDatasourceID, _ := s.resolveCatalogDatasourceID(ctx, req.DatasourceID)

	for _, mapping := range mappings {
		if mapping.Confidence >= req.AutoCreateThreshold {
			// Auto-create
			_, err := s.autoCreateMapping(ctx, req.TenantID, catalogDatasourceID, &mapping)
			if err != nil {
				logger.Errorf("Failed to auto-create mapping for %s: %v", mapping.ColumnName, err)
				response.Errors++
			} else {
				response.AutoCreated++
			}
		} else if mapping.Confidence >= req.ApprovalThreshold {
			// Queue for approval
			// Note: For pending approvals, we keep the original ID so the UI can find them
			// The resolution will happen again during approval.
			err := s.queueForApproval(ctx, req.TenantID, req.DatasourceID, &mapping)
			if err != nil {
				logger.Errorf("Failed to queue mapping for %s: %v", mapping.ColumnName, err)
				response.Errors++
			} else {
				response.PendingApproval++
			}
		} else {
			response.Skipped++
		}
	}

	return response, nil
}

// autoCreateMapping creates the semantic term and mapping automatically
func (s *SemanticMappingService) autoCreateMapping(ctx context.Context, tenantID, datasourceID string, mapping *GeneratedMapping) (string, error) {
	logger := logging.GetLogger().Sugar()

	// Parse meta to avoid double JSON encoding
	var metaMap map[string]interface{}
	if len(mapping.SuggestedMeta) > 0 {
		_ = json.Unmarshal(mapping.SuggestedMeta, &metaMap)
	}
	if metaMap == nil {
		metaMap = make(map[string]interface{})
	}

	// Extract semantic_type from meta if not explicitly provided
	semanticType := mapping.SemanticType
	if semanticType == "" {
		if st, ok := metaMap["semantic_type"].(string); ok {
			semanticType = st
		}
	}
	if semanticType == "" {
		semanticType = "dimension" // Default
	}

	// Infer Cube.dev data type from semantic type
	cubeDataType := "string"
	switch semanticType {
	case "measure":
		cubeDataType = "number"
	case "time_dimension":
		cubeDataType = "time"
	}

	// Generate camelCase name for Cube.dev API
	camelCaseName := toCamelCase(mapping.SuggestedSemanticTerm)

	// Infer format if not provided by LLM
	format := mapping.SuggestedFormat
	if format == "" {
		format = inferFormat(mapping.ColumnName, semanticType)
	}

	// Detect primary key columns
	isPrimaryKey := false
	upperCol := strings.ToUpper(mapping.ColumnName)
	if upperCol == "ID" || strings.HasSuffix(upperCol, "_ID") ||
		strings.HasPrefix(upperCol, "PK_") || metaMap["primaryKey"] == true {
		isPrimaryKey = true
	}

	// Generate business description if not provided
	description := mapping.SuggestedDescription

	// Generate human-readable title using abbreviation expansion
	title := mapping.SuggestedTitle
	if title == "" {
		// Use abbreviation service to expand term to human-readable format
		if s.abbreviationSvc != nil {
			expandedTitle, err := s.abbreviationSvc.ExpandToHumanReadable(ctx, mapping.SuggestedSemanticTerm)
			if err == nil && expandedTitle != "" {
				title = expandedTitle
			}
		}
		// Fallback to basic formatting if abbreviation expansion fails
		if title == "" {
			title = formatBusinessTermName(mapping.SuggestedSemanticTerm)
		}
	}

	// title_short is the abbreviated form of the title in Title Case
	titleShort := title // Default fallback
	if s.abbreviationSvc != nil {
		abbreviated, err := s.abbreviationSvc.AbbreviateToShort(ctx, title)
		if err == nil && abbreviated != "" {
			titleShort = abbreviated
		}
	}

	if description == "" {
		description = fmt.Sprintf("The %s.", title)
	}

	// Ensure we have table name for physical mapping
	tableName := mapping.TableName
	if tableName == "" {
		// Try to fetch from database using ColumnID if available
		if mapping.ColumnID != "" {
			var fetchedTable string
			err := s.db.GetContext(ctx, &fetchedTable, `
				SELECT parent.node_name 
				FROM catalog_node col
				JOIN catalog_node parent ON col.parent_id = parent.id
				WHERE col.id = $1
			`, mapping.ColumnID)
			if err == nil && fetchedTable != "" {
				tableName = fetchedTable
			}
		}
	}

	// Create the properties map with Cube.dev-compatible structure
	termProperties := map[string]interface{}{
		"name":          camelCaseName, // camelCase for Cube.dev API
		"data_type":     cubeDataType,
		"title":         title,
		"title_short":   titleShort,
		"description":   description,
		"meta":          metaMap,
		"format":        format,
		"semantic_type": semanticType,
		"type":          semanticType, // Cube.dev uses 'type'
		"sql":           fmt.Sprintf("${CUBE}.%s", mapping.ColumnName),
		"primaryKey":    isPrimaryKey,
		"public":        true, // Default to visible per spec
		"shown":         true,
	}

	// Add Physical Mapping for Semantic Term model compatibility
	if tableName != "" && mapping.ColumnName != "" {
		termProperties["physical_mapping"] = map[string]string{
			"table":  tableName,
			"column": mapping.ColumnName,
		}
	}

	// Merge enrichment properties if available
	if mapping.SuggestedEnrichment != nil {
		termProperties["ui_component"] = mapping.SuggestedEnrichment.UIComponent
		termProperties["ui_props"] = mapping.SuggestedEnrichment.UIProps
		termProperties["validation_rules"] = mapping.SuggestedEnrichment.ValidationRules
		termProperties["display_hints"] = mapping.SuggestedEnrichment.DisplayHints
		if mapping.SuggestedEnrichment.WealthDomain != "" {
			termProperties["wealth_domain"] = mapping.SuggestedEnrichment.WealthDomain
		}
		if mapping.SuggestedEnrichment.BOSubtypeHint != "" {
			termProperties["bo_subtype_hint"] = mapping.SuggestedEnrichment.BOSubtypeHint
		}
		if len(mapping.SuggestedEnrichment.ConstraintTemplate) > 0 {
			termProperties["constraint_template"] = mapping.SuggestedEnrichment.ConstraintTemplate
		}
	}

	// Add aggregation for measures
	if semanticType == "measure" {
		aggType := inferAggregationType(mapping.ColumnName)
		termProperties["aggregation"] = aggType
	}

	// Create the term node
	termID := uuid.New().String()
	termQuery := `
		INSERT INTO catalog_node (
			id, node_name, node_type_id, tenant_id, tenant_datasource_id, 
			qualified_path, properties, parent_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (tenant_datasource_id, node_type_id, qualified_path) DO UPDATE SET
			node_name = EXCLUDED.node_name,
			properties = catalog_node.properties || EXCLUDED.properties
		RETURNING id
	`

	propsJSON, _ := json.Marshal(termProperties)
	qualifiedPath := fmt.Sprintf("semantic.%s", mapping.SuggestedSemanticTerm)

	err := s.db.QueryRowContext(ctx, termQuery,
		termID, mapping.SuggestedSemanticTerm, SemanticTermNodeTypeID, tenantID, datasourceID,
		qualifiedPath, propsJSON, nil).Scan(&termID)

	if err != nil {
		return "", fmt.Errorf("failed to create semantic term: %v", err)
	}

	logger.Infof("[autoCreateMapping] Term INSERT returned id=%s for term=%s, datasource=%s", termID, mapping.SuggestedSemanticTerm, datasourceID)

	// Verify the term actually exists in the database
	var verifyCount int
	verifyErr := s.db.GetContext(ctx, &verifyCount, "SELECT COUNT(*) FROM catalog_node WHERE id = $1", termID)
	if verifyErr != nil || verifyCount == 0 {
		logger.Errorf("[autoCreateMapping] CRITICAL: Term id=%s NOT found! verifyErr=%v, count=%d", termID, verifyErr, verifyCount)
	}

	// Now create the mapping from column to semantic term (Create EDGE)
	// Use the 'MAPS_TO' edge type which maps database columns to semantic terms

	// Validate ColumnID before edge creation
	if mapping.ColumnID == "" {
		logger.Errorf("Cannot create edge: mapping.ColumnID is empty for term %s", mapping.SuggestedSemanticTerm)
		return "", fmt.Errorf("cannot create edge: column ID is empty for term %s", mapping.SuggestedSemanticTerm)
	}

	logger.Infof("Creating edge: source=%s (column), target=%s (term %s), edgeType=has_context",
		mapping.ColumnID, termID, mapping.SuggestedSemanticTerm)

	edgeProperties := map[string]interface{}{
		"confidence": mapping.Confidence,
		"reasoning":  mapping.Reasoning,
	}

	// Add suggested validation rules if present
	if len(mapping.SuggestedValidationRules) > 0 {
		// "conditionjson" is the expected key for validation rules on the edge
		edgeProperties["conditionjson"] = mapping.SuggestedValidationRules
	}

	edgePropsJSON, _ := json.Marshal(edgeProperties)

	edgeQuery := `
		INSERT INTO catalog_edge (
			id, source_node_id, target_node_id, edge_type, edge_type_id,
			tenant_id, tenant_datasource_id, properties, relationship_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id) DO UPDATE SET
			properties = catalog_edge.properties || EXCLUDED.properties,
			updated_at = NOW()
	`
	edgeID := uuid.New().String()
	_, err = s.db.ExecContext(ctx, edgeQuery,
		edgeID, mapping.ColumnID, termID, "has_context", HasContextEdgeTypeID,
		tenantID, datasourceID, edgePropsJSON, "has_context")

	if err != nil {
		logger.Errorf("Failed to create edge source=%s target=%s: %v", mapping.ColumnID, termID, err)
		return "", fmt.Errorf("failed to create semantic mapping edge: %v", err)
	}

	logger.Infof("Successfully created edge id=%s between column %s and term %s", edgeID, mapping.ColumnID, termID)

	// Publish term created event
	if s.publisher != nil {
		meta := map[string]interface{}{
			"semantic_type": mapping.SemanticType,
			"source_table":  mapping.TableName,
			"source_column": mapping.ColumnName,
			"auto_created":  true,
		}
		// Use "system" as user for auto-creation
		_ = s.publisher.PublishTermEvent(ctx, tenantID, "system", "term_created", termID, mapping.SuggestedSemanticTerm, meta)
	}

	// Auto-create Not Null validation rule if the source column is NOT NULL
	// Query the column's properties for validation rule generation
	type columnProps struct {
		IsNullable bool   `db:"is_nullable"`
		DataType   string `db:"data_type"`
		MaxLength  *int   `db:"max_length"`
	}

	var props columnProps
	propsQuery := `
		SELECT 
			COALESCE((properties->>'is_nullable')::boolean, true) as is_nullable,
			COALESCE(properties->>'data_type', '') as data_type,
			(properties->>'max_length')::int as max_length
		FROM catalog_node 
		WHERE id = $1
	`
	propsErr := s.db.GetContext(ctx, &props, propsQuery, mapping.ColumnID)

	if propsErr == nil {
		// Not Null Rule
		if !props.IsNullable {
			ruleErr := s.createNotNullValidationRule(ctx, tenantID, datasourceID, mapping.ColumnName, mapping.TableName, mapping.SuggestedSemanticTerm)
			if ruleErr != nil {
				logger.Warnf("Failed to auto-create Not Null rule for %s: %v", mapping.ColumnName, ruleErr)
			} else {
				logger.Infof("Auto-created Not Null validation rule for column %s", mapping.ColumnName)
			}
		}

		// Type Format Rule
		if props.DataType != "" {
			ruleErr := s.createTypeFormatValidationRule(ctx, tenantID, datasourceID, mapping.ColumnName, mapping.TableName, mapping.SuggestedSemanticTerm, props.DataType)
			if ruleErr != nil {
				logger.Warnf("Failed to auto-create Type Format rule for %s: %v", mapping.ColumnName, ruleErr)
			} else {
				logger.Infof("Auto-created Type Format validation rule for column %s (type: %s)", mapping.ColumnName, props.DataType)
			}
		}

		// Max Length Rule
		if props.MaxLength != nil && *props.MaxLength > 0 {
			ruleErr := s.createMaxLengthValidationRule(ctx, tenantID, datasourceID, mapping.ColumnName, mapping.TableName, mapping.SuggestedSemanticTerm, *props.MaxLength)
			if ruleErr != nil {
				logger.Warnf("Failed to auto-create Max Length rule for %s: %v", mapping.ColumnName, ruleErr)
			} else {
				logger.Infof("Auto-created Max Length validation rule for column %s (max: %d)", mapping.ColumnName, *props.MaxLength)
			}
		}
	} else {
		logger.Warnf("Could not query column properties for %s: %v", mapping.ColumnID, propsErr)
	}

	return termID, nil
}

// createNotNullValidationRule creates a "Not Null Check" validation rule for a column
func (s *SemanticMappingService) createNotNullValidationRule(ctx context.Context, tenantID, datasourceID, columnName, tableName, termName string) error {
	ruleID := uuid.New().String()
	ruleName := fmt.Sprintf("[%s] Not Null Check", termName)
	if termName == "" {
		ruleName = fmt.Sprintf("[%s] Not Null Check", columnName)
	}

	conditionJSON := map[string]interface{}{
		"field":    columnName,
		"operator": "not_empty",
	}
	conditionBytes, _ := json.Marshal(conditionJSON)

	targetEntity := termName
	if targetEntity == "" {
		targetEntity = tableName // Fallback to table name if term not available
		if targetEntity == "" {
			targetEntity = "global"
		}
	}

	now := time.Now()

	query := `
		INSERT INTO catalog_validation_rules (
			id, tenant_id, datasource_id, rule_name, rule_type, description,
			target_entity, condition_json, severity, is_active, is_core, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (tenant_id, datasource_id, rule_name) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		ruleID, tenantID, datasourceID, ruleName, "field_format",
		fmt.Sprintf("Auto-generated rule: %s must not be null", columnName),
		targetEntity, conditionBytes, "error", true, false, now, now,
	)

	return err
}

// createTypeFormatValidationRule creates a type format validation rule based on data type
func (s *SemanticMappingService) createTypeFormatValidationRule(ctx context.Context, tenantID, datasourceID, columnName, tableName, termName, dataType string) error {
	// Map data types to validation operators
	var operator string
	var severity string = "warning"

	switch strings.ToLower(dataType) {
	case "integer", "int", "int4", "int8", "bigint", "smallint":
		operator = "is_integer"
	case "numeric", "decimal", "real", "double precision", "float", "float4", "float8":
		operator = "is_number"
	case "date":
		operator = "is_date"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		operator = "is_datetime"
	case "boolean", "bool":
		operator = "is_boolean"
	case "uuid":
		operator = "is_uuid"
	case "json", "jsonb":
		operator = "is_json"
	default:
		// For text/varchar and other types, skip type format rule
		return nil
	}

	ruleID := uuid.New().String()
	ruleName := fmt.Sprintf("[%s] Type Format Check", termName)
	if termName == "" {
		ruleName = fmt.Sprintf("[%s] Type Format Check", columnName)
	}

	conditionJSON := map[string]interface{}{
		"field":     columnName,
		"operator":  operator,
		"data_type": dataType,
	}
	conditionBytes, _ := json.Marshal(conditionJSON)

	targetEntity := termName
	if targetEntity == "" {
		targetEntity = tableName
		if targetEntity == "" {
			targetEntity = "global"
		}
	}

	now := time.Now()

	query := `
		INSERT INTO catalog_validation_rules (
			id, tenant_id, datasource_id, rule_name, rule_type, description,
			target_entity, condition_json, severity, is_active, is_core, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (tenant_id, datasource_id, rule_name) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		ruleID, tenantID, datasourceID, ruleName, "field_format",
		fmt.Sprintf("Auto-generated rule: %s must be a valid %s", columnName, dataType),
		targetEntity, conditionBytes, severity, true, false, now, now,
	)

	return err
}

// createMaxLengthValidationRule creates a max length validation rule for varchar/text columns
func (s *SemanticMappingService) createMaxLengthValidationRule(ctx context.Context, tenantID, datasourceID, columnName, tableName, termName string, maxLength int) error {
	if maxLength <= 0 {
		return nil // Skip if no max length defined
	}

	ruleID := uuid.New().String()
	ruleName := fmt.Sprintf("[%s] Max Length Check", termName)
	if termName == "" {
		ruleName = fmt.Sprintf("[%s] Max Length Check", columnName)
	}

	conditionJSON := map[string]interface{}{
		"field":      columnName,
		"operator":   "max_length",
		"max_length": maxLength,
	}
	conditionBytes, _ := json.Marshal(conditionJSON)

	targetEntity := termName
	if targetEntity == "" {
		targetEntity = tableName
		if targetEntity == "" {
			targetEntity = "global"
		}
	}

	now := time.Now()

	query := `
		INSERT INTO catalog_validation_rules (
			id, tenant_id, datasource_id, rule_name, rule_type, description,
			target_entity, condition_json, severity, is_active, is_core, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (tenant_id, datasource_id, rule_name) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		ruleID, tenantID, datasourceID, ruleName, "field_format",
		fmt.Sprintf("Auto-generated rule: %s must not exceed %d characters", columnName, maxLength),
		targetEntity, conditionBytes, "warning", true, false, now, now,
	)

	return err
}

// queueForApproval adds mapping to pending approvals table
func (s *SemanticMappingService) queueForApproval(ctx context.Context, tenantID, datasourceID string, mapping *GeneratedMapping) error {
	query := `
		INSERT INTO pending_semantic_mappings (
			tenant_id, datasource_id, column_id, column_name, expanded_column_name,
			suggested_semantic_term, suggested_business_term, 
			suggested_title, suggested_description, suggested_meta, suggested_format,
			confidence, reasoning
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := s.db.ExecContext(ctx, query,
		tenantID, datasourceID, mapping.ColumnID, mapping.ColumnName, mapping.ExpandedColumnName,
		mapping.SuggestedSemanticTerm, mapping.SuggestedBusinessTerm,
		mapping.SuggestedTitle, mapping.SuggestedDescription, mapping.SuggestedMeta, mapping.SuggestedFormat,
		mapping.Confidence, mapping.Reasoning)

	return err
}

// GetPendingApprovals retrieves pending mappings for approval
func (s *SemanticMappingService) GetPendingApprovals(ctx context.Context, tenantID, datasourceID string) ([]PendingSemanticMapping, error) {
	query := `
		SELECT * FROM pending_semantic_mappings
		WHERE tenant_id = $1 AND datasource_id = $2 AND status = 'pending'
		ORDER BY confidence DESC, created_at DESC
	`

	var pending []PendingSemanticMapping
	err := s.db.SelectContext(ctx, &pending, query, tenantID, datasourceID)
	return pending, err
}

// ApprovePendingMapping approves or rejects a pending mapping
func (s *SemanticMappingService) ApprovePendingMapping(ctx context.Context, mappingID string, approved bool, userID string) (string, error) {
	// Fetch the pending mapping details for the event
	var pending PendingSemanticMapping
	err := s.db.GetContext(ctx, &pending, "SELECT * FROM pending_semantic_mappings WHERE id = $1", mappingID)
	if err != nil {
		return "", err
	}

	if !approved {
		// Just mark as rejected
		_, err := s.db.ExecContext(ctx,
			"UPDATE pending_semantic_mappings SET status = 'rejected', approved_at = NOW(), approved_by = $1 WHERE id = $2",
			userID, mappingID)
		if err != nil {
			return "", err
		}

		// Publish rejection event
		if s.publisher != nil {
			meta := map[string]interface{}{
				"reasoning":      pending.Reasoning,
				"confidence":     pending.Confidence,
				"suggested_term": pending.SuggestedSemanticTerm,
			}
			_ = s.publisher.PublishTermEvent(ctx, pending.TenantID.String(), userID, "term_rejected", mappingID, pending.SuggestedSemanticTerm, meta)
		}

		return "", nil
	}

	// Create the mapping
	mapping := &GeneratedMapping{
		ColumnID:              pending.ColumnID.String(),
		ColumnName:            pending.ColumnName,
		ExpandedColumnName:    pending.ExpandedColumnName,
		SuggestedSemanticTerm: pending.SuggestedSemanticTerm,
		SuggestedBusinessTerm: pending.SuggestedBusinessTerm,
		SuggestedTitle:        pending.SuggestedTitle,
		SuggestedDescription:  pending.SuggestedDescription,
		SuggestedMeta:         pending.SuggestedMeta,
		SuggestedFormat:       pending.SuggestedFormat,
		Confidence:            pending.Confidence,
		Reasoning:             pending.Reasoning,
	}

	// Resolve datasource ID
	catalogDatasourceID, _ := s.resolveCatalogDatasourceID(ctx, pending.DatasourceID.String())

	termID, err := s.autoCreateMapping(ctx, pending.TenantID.String(), catalogDatasourceID, mapping)
	if err != nil {
		return "", err
	}

	// Mark as approved
	_, err = s.db.ExecContext(ctx,
		"UPDATE pending_semantic_mappings SET status = 'approved', approved_at = NOW(), approved_by = $1 WHERE id = $2",
		userID, mappingID)
	if err != nil {
		return "", err
	}

	// Publish approval event
	if s.publisher != nil {
		eventID := mappingID // Use mapping ID for correlation
		meta := map[string]interface{}{
			"confidence":    mapping.Confidence,
			"semantic_type": mapping.SemanticType,
		}
		_ = s.publisher.PublishTermEvent(ctx, pending.TenantID.String(), userID, "term_approved", eventID, mapping.SuggestedSemanticTerm, meta)
	}

	return termID, nil
}

// resolveCatalogDatasourceID maps logical datasources (like datamart) to physical ones (like alpha_dwh)
func (s *SemanticMappingService) resolveCatalogDatasourceID(ctx context.Context, datasourceID string) (string, error) {
	logger := logging.GetLogger().Sugar()
	catalogDatasourceID := datasourceID

	var datasourceName string
	err := s.db.GetContext(ctx, &datasourceName, "SELECT datasource_name FROM alpha_datasource WHERE id = $1", datasourceID)
	if err == nil {
		if strings.EqualFold(datasourceName, "datamart") {
			var alphaDwhID string
			err = s.db.GetContext(ctx, &alphaDwhID, "SELECT id FROM alpha_datasource WHERE datasource_name = 'alpha_dwh'")
			if err == nil && alphaDwhID != "" {
				logger.Infof("Mapped 'datamart' datasource to 'alpha_dwh' ID: %s", alphaDwhID)
				catalogDatasourceID = alphaDwhID
			} else {
				logger.Warnf("Could not find 'alpha_dwh' datasource ID to map 'datamart'")
			}
		}
	} else {
		return datasourceID, err
	}

	return catalogDatasourceID, nil
}
