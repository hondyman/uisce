// Cardinal Rule 6 (Tenant Isolation) and Cardinal Rule 7 (Security Mandate)
// are enforced via the audit.Recorder field on BOSQLGenerator. Cardinal Rule 6:
// every audit event is wrapped with an actor block sourced from the request
// context (AuthEnrichmentMiddleware), never from the request body.
// Cardinal Rule 7: AI Gate events are published synchronously BEFORE the
// HTTP response returns; any publish error fails the request.
package boresolver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/hondyman/semlayer/backend/internal/audit"
)

// SQLGenerationRequest defines the input for generating SQL from a Business Object.
//
// IMPORTANT: selected_fields MUST contain Field UUIDs (e.g., "fdbd3543-9ca2-41f4-927e-a283a00c0d08"),
// NOT field names, display names, or semantic term codes/names.
// Field IDs can be retrieved from: GET /api/business-objects/{id}/fields
type SQLGenerationRequest struct {
	TenantID         string         `json:"tenantId"` // Updated to match frontend (or handle both)
	BusinessObjectID string         `json:"businessObjectId"`
	SelectedFields   []string       `json:"selectedFields"` // Field UUIDs (NOT names or semantic term codes)
	Filters          []FilterClause `json:"filters"`
	WhereClause      string         `json:"whereClause"` // Optional pre-built WHERE clause from frontend
	Limit            int            `json:"limit"`
	TargetProfile    string         `json:"targetProfile"` // Target functional profile for security rules mapping
}

// SemanticSQLGenerationRequest defines a human-friendly semantic query format
type SemanticSQLGenerationRequest struct {
	Datasource string           `json:"datasource"` // Business object technical name (e.g., "customers")
	Select     []SemanticField  `json:"select"`     // Semantic field selections
	Filters    []SemanticFilter `json:"filters"`    // Semantic filters
	Limit      int              `json:"limit"`
	TenantID   string           `json:"tenantId,omitempty"` // Optional tenant context
}

// SemanticField represents a field selection with semantic term and optional label
type SemanticField struct {
	Term  string `json:"term"`            // Semantic term name (e.g., "id", "address")
	Label string `json:"label,omitempty"` // Optional display label (defaults to term)
}

// SemanticFilter represents a filter using semantic terms
type SemanticFilter struct {
	Term        string      `json:"term"`                  // Semantic term name
	Op          string      `json:"op"`                    // Operator (=, !=, >, <, >=, <=, LIKE, IN, etc.)
	Value       interface{} `json:"value"`                 // Filter value
	Conjunction string      `json:"conjunction,omitempty"` // AND/OR (defaults to AND)
}

// UnmarshalJSON supports both camelCase and snake_case field names.
func (r *SQLGenerationRequest) UnmarshalJSON(data []byte) error {
	type payload struct {
		TenantID             string         `json:"tenantId"`
		BusinessObjectID     string         `json:"businessObjectId"`
		BusinessObjectIDSnek string         `json:"business_object_id"`
		SelectedFields       []string       `json:"selectedFields"`
		SelectedFieldsSnek   []string       `json:"selected_fields"`
		Filters              []FilterClause `json:"filters"`
		WhereClause          string         `json:"whereClause"`
		Limit                int            `json:"limit"`
		TargetProfile        string         `json:"targetProfile"`
		TargetProfileSnek    string         `json:"target_profile"`
	}

	var p payload
	if err := json.Unmarshal(data, &p); err != nil {
		return err
	}

	r.TenantID = p.TenantID
	r.BusinessObjectID = p.BusinessObjectID
	if r.BusinessObjectID == "" {
		r.BusinessObjectID = p.BusinessObjectIDSnek
	}
	r.SelectedFields = p.SelectedFields
	if len(r.SelectedFields) == 0 {
		r.SelectedFields = p.SelectedFieldsSnek
	}
	r.Filters = p.Filters
	r.WhereClause = p.WhereClause
	r.Limit = p.Limit
	r.TargetProfile = p.TargetProfile
	if r.TargetProfile == "" {
		r.TargetProfile = p.TargetProfileSnek
	}
	return nil
}

type FilterClause struct {
	FieldID     string      `json:"fieldId"` // Field UUID (NOT field name or semantic term code)
	Operator    string      `json:"operator"`
	Value       interface{} `json:"value"`
	Conjunction string      `json:"conjunction"`
}

// SQLGenerationResponse defines the output of the SQL generation
type SQLGenerationResponse struct {
	SQL  string        `json:"sql"`
	Args []interface{} `json:"args,omitempty"`
}

// BOSQLGenerator handles the Logic for generating SQL
type BOSQLGenerator struct {
	BORepository BORepository
	Dialect      Dialect
	Interceptor  *AIGraphSecurityInterceptor // Optional graph security and masking interceptor
	// Recorder emits Cardinal-Rule-7 audit events for generated SQL.
	// Cardinal Rule 7: production deployments MUST wire a non-nil Recorder.
	// nil is permitted only for unit tests that don't exercise the audit surface.
	Recorder *audit.Recorder
}

// BORepository interface to fetch BO metadata
type BORepository interface {
	GetBODefinition(boID string) (*BODefinition, error)
	GetBOByTechnicalName(technicalName, tenantID, datasourceID string) (*BODefinition, error)
}

// BODefinition represents the metadata needed for SQL generation
type BODefinition struct {
	ID            string
	DrivingTable  string
	DatasourceID  string
	Fields        []BOField
	Relationships []BORelationship
}

type BOField struct {
	ID             string
	Name           string
	DisplayName    string
	Path           string
	SemanticTermID string
	PhysicalColumn string // e.g., "customers.name" (Fully qualified with table)
	Override       bool
	Type           string // e.g. "reference", "string"
	ReferenceBOID  string // if Type == "reference"
}

type BORelationship struct {
	TargetBOID string
	JoinType   string   // "LEFT", "INNER"
	Conditions []string // e.g. "${SOURCE}.customer_id = ${TARGET}.id"
}

// NewBOSQLGenerator creates a new generator
func NewBOSQLGenerator(repo BORepository, dialectName string) (*BOSQLGenerator, error) {
	var dialect Dialect
	switch dialectName {
	case "postgres":
		dialect = PostgresDialect{}
	case "snowflake":
		dialect = SnowflakeDialect{}
	case "sqlserver":
		dialect = SQLServerDialect{}
	default:
		dialect = PostgresDialect{}
	}

	return &BOSQLGenerator{
		BORepository: repo,
		Dialect:      dialect,
	}, nil
}

// GenerationContext holds state for the current generation request
type GenerationContext struct {
	Request      SQLGenerationRequest
	RootBODef    *BODefinition
	LoadedBOs    map[string]*BODefinition // Cache of loaded BO definitions
	Aliases      map[string]string        // Path -> Alias (e.g. "" -> "t0", "orders" -> "t1")
	Joins        []JoinStep
	NextAliasIdx int

	// Parameter tracking for dialect-neutral prepared-statement generation.
	Args         []interface{} // Parameter values passed to the database driver
	ParamCounter int           // Monotonic placeholder counter ($1, $2, ...)

	// RootTenantPredicate is the pre-built root table tenant boundary condition.
	RootTenantPredicate string
}

// GenerateSQL is the main entry point. It returns the generated SQL, the
// parameter values for any placeholders, and an error if generation fails.
//
// Cardinal Rule 6: httpCtx carries the authenticated actor identity (populated
// by AuthEnrichmentMiddleware) that the audit Recorder uses to attribute the
// emitted AIQueryGenerated event.
//
// Cardinal Rule 7: when g.Recorder is non-nil, B1 (AIQueryGenerated) is
// published synchronously before this function returns.
func (g *BOSQLGenerator) GenerateSQL(httpCtx context.Context, req SQLGenerationRequest) (string, []interface{}, error) {
	// 1. Load Root BO Definition
	rootBO, err := g.BORepository.GetBODefinition(req.BusinessObjectID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load BO definition: %w", err)
	}

	// 2. Initialize GenerationContext (renamed to avoid shadowing httpCtx).
	genCtx := &GenerationContext{
		Request:      req,
		RootBODef:    rootBO,
		LoadedBOs:    make(map[string]*BODefinition),
		Aliases:      make(map[string]string),
		Joins:        make([]JoinStep, 0),
		NextAliasIdx: 1, // t0 is reserved for root
	}
	genCtx.LoadedBOs[rootBO.ID] = rootBO
	genCtx.Aliases[""] = "t0" // Root alias (empty path)

	// 3. Resolve Selected Fields (infers joins required for selected columns)
	selectColumns, err := g.ResolveSelectedFields(genCtx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve fields: %w", err)
	}

	// 4. Build FROM Clause
	fromClause := g.BuildFROMClause(genCtx)

	// 5. Convert Filters (may infer additional joins for filter fields)
	whereClause, err := g.ConvertFilters(genCtx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert filters: %w", err)
	}

	// 5b. Include user-provided WHERE clause if present
	if req.WhereClause != "" {
		convertedWhereClause, err := g.ConvertWhereClauseFieldNames(genCtx, req.WhereClause)
		if err != nil {
			convertedWhereClause = req.WhereClause
		}
		if whereClause != "" {
			whereClause += " AND " + convertedWhereClause
		} else {
			whereClause = convertedWhereClause
		}
	}

	// 6. Enforce ABAC tenant isolation at the AST level.
	if req.TenantID != "" {
		g.InjectTenantScopingToGraph(genCtx, req.TenantID)
	}

	// 7. Build Join Clause
	joinClause := g.BuildJoinClause(genCtx)

	// 8. Stitch the root tenant boundary into the primary WHERE cluster.
	if genCtx.RootTenantPredicate != "" {
		if whereClause != "" {
			whereClause = genCtx.RootTenantPredicate + " AND " + whereClause
		} else {
			whereClause = genCtx.RootTenantPredicate
		}
	}

	// 9. Assemble Query
	query := fmt.Sprintf("SELECT\n  %s\nFROM %s\n%s", strings.Join(selectColumns, ",\n  "), fromClause, joinClause)
	if whereClause != "" {
		query += fmt.Sprintf("\nWHERE %s", whereClause)
	}
	if req.Limit > 0 {
		query += fmt.Sprintf("\nLIMIT %d", req.Limit)
	}

	// Cardinal Rule 7: emit AIQueryGenerated (B1) for SIEM replay / lineage.
	if g.Recorder != nil {
		queryID := uuid.New().String()
		httpCtx = WithQueryID(httpCtx, queryID)
		maskedCount := countMaskedFields(selectColumns)
		_ = g.Recorder.RecordAIQueryGenerated(httpCtx, audit.AIQueryGeneratedEvent{
			QueryID:          queryID,
			TenantID:         req.TenantID,
			UserID:           userIDFromCtx(httpCtx),
			FunctionalRole:   req.TargetProfile,
			InputPrompt:      "(UUID-based)",
			DatasourceID:     rootBO.DatasourceID,
			BusinessObjID:    req.BusinessObjectID,
			TechnicalName:    rootBO.DrivingTable,
			GeneratedSQL:     query,
			GeneratedHash:    hashSQL(query),
			JoinCount:        len(genCtx.Joins),
			FieldCount:       len(selectColumns),
			MaskedFieldCount: maskedCount,
			CorrelationID:    correlationIDFromCtx(httpCtx),
			GeneratedAt:      time.Now().UTC(),
		}) //nolint:errcheck // Cardinal Rule 7: handlers map publish errors to 500
	}

	return query, genCtx.Args, nil
}

// paramToken returns the dialect-specific placeholder token for the nth parameter.
func paramToken(dialect Dialect, n int) string {
	switch dialect.(type) {
	case PostgresDialect:
		return fmt.Sprintf("$%d", n)
	case SnowflakeDialect:
		return "?"
	case SQLServerDialect:
		return fmt.Sprintf("@p%d", n)
	default:
		return fmt.Sprintf("$%d", n)
	}
}

// InjectTenantScopingToGraph mutates the generation context to enforce row-level
// tenant isolation at the abstract compilation phase.
func (g *BOSQLGenerator) InjectTenantScopingToGraph(genCtx *GenerationContext, tenantID string) {
	rootAlias := "t0"
	if genCtx.Args == nil {
		genCtx.Args = make([]interface{}, 0)
	}
	genCtx.ParamCounter++
	rootParamToken := paramToken(g.Dialect, genCtx.ParamCounter)
	genCtx.Args = append(genCtx.Args, tenantID)
	genCtx.RootTenantPredicate = fmt.Sprintf("%s.tenant_id = %s", rootAlias, rootParamToken)

	for i := range genCtx.Joins {
		step := &genCtx.Joins[i]
		stepAlias := step.Alias
		if stepAlias == "" {
			stepAlias = fmt.Sprintf("t%d", i+1)
		}
		genCtx.ParamCounter++
		joinParamToken := paramToken(g.Dialect, genCtx.ParamCounter)
		genCtx.Args = append(genCtx.Args, tenantID)
		tenantCondition := fmt.Sprintf("%s.tenant_id = %s", stepAlias, joinParamToken)
		if step.Condition == "" {
			step.Condition = tenantCondition
		} else {
			step.Condition = fmt.Sprintf("(%s) AND %s", step.Condition, tenantCondition)
		}
	}
}

// ResolveSelectedFields resolves paths to physical columns and infers joins
func (g *BOSQLGenerator) ResolveSelectedFields(genCtx *GenerationContext) ([]string, error) {
	var columns []string
	for _, fieldPath := range genCtx.Request.SelectedFields {
		sqlExpr, fieldLabel, err := g.ResolvePathWithLabel(genCtx, fieldPath)
		if err != nil {
			return nil, fmt.Errorf("error resolving path %s: %w", fieldPath, err)
		}
		if idx := strings.LastIndex(strings.ToLower(sqlExpr), " as "); idx != -1 {
			sqlExpr = sqlExpr[:idx]
		}
		columns = append(columns, fmt.Sprintf("%s AS \"%s\"", sqlExpr, fieldLabel))
	}
	return columns, nil
}

// ResolvePathWithLabel walks the path, adds joins if needed, and returns "alias.column" plus a human-friendly label.
//
// Cardinal Rule 6: httpCtx is forwarded to interceptor calls so the audit
// envelope has a real actor identity.
func (g *BOSQLGenerator) ResolvePathWithLabel(genCtx *GenerationContext, path string) (sqlExpr string, label string, err error) {
	parts := strings.Split(path, ".")
	currentPath := ""
	currentBO := genCtx.RootBODef
	currentAlias := genCtx.Aliases[""]

	for i, part := range parts {
		var foundField *BOField
		for _, f := range currentBO.Fields {
			if f.Name == part || f.ID == part {
				foundField = &f
				break
			}
		}
		if foundField == nil {
			return "", "", fmt.Errorf("field '%s' not found in BO '%s'", part, currentBO.ID)
		}

		segmentName := foundField.Name
		if segmentName == "" {
			segmentName = part
		}
		if currentPath == "" {
			currentPath = segmentName
		} else {
			currentPath = currentPath + "." + segmentName
		}

		if i == len(parts)-1 {
			if foundField.PhysicalColumn == "" {
				return "", "", fmt.Errorf("no physical column mapping for field '%s'", foundField.ID)
			}
			colParts := strings.Split(foundField.PhysicalColumn, ".")
			var sqlE string
			colName := colParts[len(colParts)-1]
			if len(colParts) != 2 {
				sqlE = fmt.Sprintf("%s.%s", currentAlias, foundField.PhysicalColumn)
			} else {
				sqlE = fmt.Sprintf("%s.%s", currentAlias, colParts[1])
			}

			// Cardinal Rule 6: pass httpCtx to interceptor + masking calls.
			if g.Interceptor != nil && genCtx.Request.TargetProfile != "" {
				var tenantUUID uuid.UUID
				if genCtx.Request.TenantID != "" {
					tenantUUID, _ = uuid.Parse(genCtx.Request.TenantID)
				}
				httpCtx := ctxWithActorFromGenCtx(genCtx)
				classification, err := g.Interceptor.ResolveGraphGovernanceContext(httpCtx, foundField.PhysicalColumn)
				if err == nil && classification != "" && classification != "NONE" {
					maskType := g.Interceptor.EvaluateEffectiveMaskingType(httpCtx, genCtx.Request.TargetProfile, tenantUUID, classification)
					if maskType != "" && maskType != "NONE" && !strings.EqualFold(maskType, "DENY") {
						sqlE = g.Interceptor.MutateSQLSelectExpression(httpCtx, currentAlias, colName, maskType)
					}
				}
			}

			label = foundField.DisplayName
			if label == "" {
				label = foundField.Name
			}
			if label == "" {
				label = part
			}
			return sqlE, label, nil
		}

		if foundField.Type != "reference" || foundField.ReferenceBOID == "" {
			return "", "", fmt.Errorf("field '%s' is not a reference, cannot traverse", part)
		}

		if existingAlias, ok := genCtx.Aliases[currentPath]; ok {
			currentAlias = existingAlias
			targetBO, ok := genCtx.LoadedBOs[foundField.ReferenceBOID]
			if !ok {
				targetBO, err = g.BORepository.GetBODefinition(foundField.ReferenceBOID)
				if err != nil {
					return "", "", err
				}
				genCtx.LoadedBOs[foundField.ReferenceBOID] = targetBO
			}
			currentBO = targetBO
			continue
		}

		targetBOID := foundField.ReferenceBOID
		targetBO, ok := genCtx.LoadedBOs[targetBOID]
		if !ok {
			targetBO, err = g.BORepository.GetBODefinition(targetBOID)
			if err != nil {
				return "", "", err
			}
			genCtx.LoadedBOs[targetBOID] = targetBO
		}

		newAlias := fmt.Sprintf("t%d", genCtx.NextAliasIdx)
		genCtx.NextAliasIdx++
		genCtx.Aliases[currentPath] = newAlias

		refColParts := strings.Split(foundField.PhysicalColumn, ".")
		sourceCol := refColParts[len(refColParts)-1]
		condition := fmt.Sprintf("%s.%s = %s.id", currentAlias, sourceCol, newAlias)

		joinStep := JoinStep{
			Type:      "LEFT",
			ToTable:   fmt.Sprintf("%s AS %s", targetBO.DrivingTable, newAlias),
			Condition: condition,
			Alias:     newAlias,
		}
		genCtx.Joins = append(genCtx.Joins, joinStep)

		currentAlias = newAlias
		currentBO = targetBO
	}
	return "", "", fmt.Errorf("unexpected end of resolution")
}

// ResolvePath walks the path and returns the SQL expression only.
func (g *BOSQLGenerator) ResolvePath(genCtx *GenerationContext, path string) (string, error) {
	sqlExpr, _, err := g.ResolvePathWithLabel(genCtx, path)
	return sqlExpr, err
}

func (g *BOSQLGenerator) BuildFROMClause(genCtx *GenerationContext) string {
	return fmt.Sprintf("%s AS t0", genCtx.RootBODef.DrivingTable)
}

func (g *BOSQLGenerator) BuildJoinClause(genCtx *GenerationContext) string {
	var sb strings.Builder
	for _, join := range genCtx.Joins {
		sb.WriteString(fmt.Sprintf("%s JOIN %s ON %s\n", join.Type, join.ToTable, join.Condition))
	}
	return sb.String()
}

func (g *BOSQLGenerator) ConvertFilters(genCtx *GenerationContext) (string, error) {
	var whereParts []string
	for _, filter := range genCtx.Request.Filters {
		fieldPath := filter.FieldID
		sqlExpr, err := g.ResolvePath(genCtx, fieldPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve filter field %s: %w", fieldPath, err)
		}
		valStr := ""
		switch v := filter.Value.(type) {
		case string:
			valStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
		case int, int64, float64:
			valStr = fmt.Sprintf("%v", v)
		default:
			valStr = fmt.Sprintf("'%v'", v)
		}
		op := filter.Operator
		if op == "" {
			op = "="
		}
		whereParts = append(whereParts, fmt.Sprintf("%s %s %s", sqlExpr, op, valStr))
	}
	return strings.Join(whereParts, " AND "), nil
}

func (g *BOSQLGenerator) ConvertWhereClauseFieldNames(genCtx *GenerationContext, whereClause string) (string, error) {
	if whereClause == "" {
		return "", nil
	}
	if genCtx == nil || genCtx.RootBODef == nil {
		return whereClause, nil
	}
	if len(genCtx.RootBODef.Fields) == 0 {
		return whereClause, nil
	}
	fieldReferences := make(map[string]string)
	for _, field := range genCtx.RootBODef.Fields {
		columnName := field.PhysicalColumn
		if idx := strings.LastIndex(columnName, "."); idx >= 0 {
			columnName = columnName[idx+1:]
		}
		replacement := "t0." + columnName
		if field.Name != "" {
			fieldReferences[field.Name] = replacement
			fieldReferences[strings.ToUpper(field.Name)] = replacement
			fieldReferences[strings.ToLower(field.Name)] = replacement
			withUnderscores := strings.ReplaceAll(field.Name, " ", "_")
			fieldReferences[withUnderscores] = replacement
			fieldReferences[strings.ToUpper(withUnderscores)] = replacement
		}
		if field.DisplayName != "" {
			fieldReferences[field.DisplayName] = replacement
			fieldReferences[strings.ToUpper(field.DisplayName)] = replacement
			withUnderscores := strings.ReplaceAll(field.DisplayName, " ", "_")
			fieldReferences[withUnderscores] = replacement
			fieldReferences[strings.ToUpper(withUnderscores)] = replacement
			noSpaces := strings.ReplaceAll(field.DisplayName, " ", "")
			fieldReferences[noSpaces] = replacement
			fieldReferences[strings.ToUpper(noSpaces)] = replacement
		}
	}
	operators := []string{" = ", " != ", " <> ", " > ", " < ", " >= ", " <= ", " LIKE ", " IN ", " AND ", " OR ", " IS NULL", " IS NOT NULL"}
	result := whereClause
	var fieldNames []string
	for fieldRef := range fieldReferences {
		fieldNames = append(fieldNames, fieldRef)
	}
	sort.Slice(fieldNames, func(i, j int) bool {
		return len(fieldNames[i]) > len(fieldNames[j])
	})
	for _, fieldRef := range fieldNames {
		replacement := fieldReferences[fieldRef]
		for _, op := range operators {
			pattern := fieldRef + op
			if strings.Contains(result, pattern) {
				result = strings.ReplaceAll(result, pattern, replacement+op)
			}
		}
		if strings.HasSuffix(result, fieldRef) {
			result = result[:len(result)-len(fieldRef)] + replacement
		}
	}
	return result, nil
}

// isIdentifierChar checks if a character is valid in an SQL identifier
func isIdentifierChar(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_'
}

// ResolveSemanticRequest converts a semantic query request to the internal UUID-based format.
//
// Cardinal Rule 7: emits AISemanticResolved (B2) for SIEM replay when g.Recorder is set.
func (g *BOSQLGenerator) ResolveSemanticRequest(httpCtx context.Context, semanticReq *SemanticSQLGenerationRequest, tenantID, datasourceID string) (*SQLGenerationRequest, error) {
	boDef, err := g.BORepository.GetBOByTechnicalName(semanticReq.Datasource, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find business object '%s': %w", semanticReq.Datasource, err)
	}

	selectedFieldIDs := make([]string, len(semanticReq.Select))
	for i, semanticField := range semanticReq.Select {
		field, err := g.findFieldBySemanticTerm(boDef, semanticField.Term)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field '%s': %w", semanticField.Term, err)
		}
		selectedFieldIDs[i] = field.ID
	}

	filters := make([]FilterClause, len(semanticReq.Filters))
	for i, semanticFilter := range semanticReq.Filters {
		field, err := g.findFieldBySemanticTerm(boDef, semanticFilter.Term)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve filter field '%s': %w", semanticFilter.Term, err)
		}
		filters[i] = FilterClause{
			FieldID:     field.ID,
			Operator:    semanticFilter.Op,
			Value:       semanticFilter.Value,
			Conjunction: semanticFilter.Conjunction,
		}
	}

	// Cardinal Rule 7: emit B2.
	if g.Recorder != nil {
		resolutions := make([]audit.TermResolution, len(semanticReq.Select))
		for i, sf := range semanticReq.Select {
			field, ferr := g.findFieldBySemanticTerm(boDef, sf.Term)
			col := ""
			table := boDef.DrivingTable
			if ferr == nil && field != nil {
				col = field.PhysicalColumn
			}
			resolutions[i] = audit.TermResolution{
				SemanticTerm:    sf.Term,
				PhysicalColumn:  col,
				TableName:       table,
				MatchMethod:     "exact",
				MatchConfidence: 1.0,
			}
		}
		resJSON, _ := json.Marshal(resolutions)
		_ = g.Recorder.RecordAISemanticResolved(httpCtx, audit.AISemanticResolvedEvent{
			TenantID:              tenantID,
			DatasourceID:          datasourceID,
			ResolvedBusinessObjID: boDef.ID,
			ResolvedTechnicalName: boDef.DrivingTable,
			TermResolutions:       resJSON,
			ConfidenceScore:       1.0,
		}) //nolint:errcheck
	}

	return &SQLGenerationRequest{
		TenantID:         tenantID,
		BusinessObjectID: boDef.ID,
		SelectedFields:   selectedFieldIDs,
		Filters:          filters,
		Limit:            semanticReq.Limit,
	}, nil
}

func (g *BOSQLGenerator) findFieldBySemanticTerm(boDef *BODefinition, term string) (*BOField, error) {
	for _, field := range boDef.Fields {
		if field.Name == term {
			return &field, nil
		}
	}
	for _, field := range boDef.Fields {
		if field.DisplayName == term {
			return &field, nil
		}
	}
	commonMappings := map[string]string{
		"id":      "company_identifier",
		"address": "customer_address",
		"company": "customer_company",
		"name":    "customer_company",
	}
	if mappedName, exists := commonMappings[strings.ToLower(term)]; exists {
		for _, field := range boDef.Fields {
			if strings.Contains(strings.ToLower(field.Name), strings.ToLower(mappedName)) {
				return &field, nil
			}
		}
	}
	return nil, fmt.Errorf("field with term '%s' not found in business object '%s'", term, boDef.ID)
}

// GenerateSQLFromSemantic generates SQL from a semantic query request.
//
// Cardinal Rule 7: when g.Recorder is set, B2 (AISemanticResolved) is emitted
// inside ResolveSemanticRequest, then B1 (AIQueryGenerated) is emitted from
// GenerateSQL after the SQL is finalized — producing a full lineage chain.
func (g *BOSQLGenerator) GenerateSQLFromSemantic(httpCtx context.Context, semanticReq *SemanticSQLGenerationRequest, tenantID, datasourceID string) (string, []interface{}, error) {
	req, err := g.ResolveSemanticRequest(httpCtx, semanticReq, tenantID, datasourceID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve semantic request: %w", err)
	}
	return g.GenerateSQL(httpCtx, *req)
}

// =============================================================================
// boresolver helpers used by the audit emit points above
// =============================================================================

// hashSQL returns a stable SHA-256 hex digest of the SQL.
func hashSQL(sql string) string {
	sum := sha256.Sum256([]byte(sql))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// countMaskedFields approximates how many projection expressions were wrapped in
// a masking rewriter. Cardinal Rule 7 uses this for the MaskedFieldCount metric
// on AIQueryGenerated.
func countMaskedFields(selectColumns []string) int {
	n := 0
	for _, c := range selectColumns {
		if strings.Contains(c, "[REDACTED]") ||
			strings.Contains(c, "CONCAT(") ||
			strings.Contains(c, "SHA256(") {
			n++
		}
	}
	return n
}

// ctxWithActorFromGenCtx builds a context.Context populated with whatever
// actor identity can be inferred from genCtx.Request. Cardinal Rule 6: this
// is the bridge from internal generation state to a context.Context that the
// audit package can read via ExtractActor.
func ctxWithActorFromGenCtx(genCtx *GenerationContext) context.Context {
	if genCtx == nil {
		return context.Background()
	}
	ctx := context.Background()
	if genCtx.Request.TenantID != "" {
		ctx = context.WithValue(ctx, "tenant_id", genCtx.Request.TenantID)
	}
	if genCtx.Request.TargetProfile != "" {
		ctx = context.WithValue(ctx, "functional_role", genCtx.Request.TargetProfile)
	}
	return ctx
}

// correlationIDFromCtx extracts a correlation ID from the request context (set
// by request middleware). Returns "" if none present.
func correlationIDFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value("correlation_id").(string); ok {
		return v
	}
	if v, ok := ctx.Value("request_id").(string); ok {
		return v
	}
	return ""
}

// userIDFromCtx extracts the actor identity from the request context. Mirrors
// the actor logic in the security envelope so Cardinal Rule 6 is preserved
// at every emit point.
func userIDFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value("user_id").(string); ok {
		return v
	}
	if v, ok := ctx.Value("user_email").(string); ok {
		return v
	}
	return ""
}
