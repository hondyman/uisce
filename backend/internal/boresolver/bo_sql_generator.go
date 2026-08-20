package boresolver

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
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
	// AsOfDate triggers Hot/Cold Watermark Routing: if set and before the
	// hot/cold watermark, routes to DataFusionIcebergDialect (cold tier).
	AsOfDate time.Time `json:"asOfDate,omitempty"`
	// KnowledgeDate triggers True Bitemporal valid-time query filtering (system transaction time).
	KnowledgeDate time.Time `json:"knowledgeDate,omitempty"`
	// DialectOverride explicitly selects a dialect, bypassing watermark routing.
	DialectOverride string `json:"dialectOverride,omitempty"`
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
		AsOfDate             time.Time      `json:"asOfDate"`
		KnowledgeDate        time.Time      `json:"knowledgeDate"`
		DialectOverride      string         `json:"dialectOverride"`
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
	r.AsOfDate = p.AsOfDate
	r.KnowledgeDate = p.KnowledgeDate
	r.DialectOverride = p.DialectOverride
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
	BORepository      BORepository
	Dialect           Dialect
	WatermarkResolver WatermarkResolver
	TelemetryRouter   TelemetryRouter
}

// TelemetryRouter resolves the optimal execution flavor for a BO based on telemetry.
type TelemetryRouter interface {
	GetOptimalFlavor(ctx context.Context, tenantID string, boKey string, defaultFlavor string) (string, error)
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
	ID                string
	Name              string
	DisplayName       string
	Path              string
	SemanticTermID    string
	PhysicalColumn    string // e.g., "customers.name" (Fully qualified with table)
	SourceType        string // "COLUMN", "JSON_PATH", "EXPRESSION"
	JSONPath          string // e.g., "$.loyalty_score"
	TransformationSQL string // e.g., "get_json_string(${alias}.tenant_extensions, '$.loyalty_score')"
	Override          bool
	Type              string // e.g. "reference", "string"
	ReferenceBOID     string // if Type == "reference"
}


type BORelationship struct {
	TargetBOID string
	JoinType   string   // "LEFT", "INNER"
	Conditions []string // e.g. "${SOURCE}.customer_id = ${TARGET}.id"
}


// WatermarkResolver resolves the hot/cold watermark timestamp for a tenant.
type WatermarkResolver interface {
	GetHotColdWatermark(tenantID string) time.Time
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
	case "datafusion", "datafusion_iceberg", "iceberg":
		dialect = DataFusionIcebergDialect{}
	default:
		dialect = PostgresDialect{}
	}

	return &BOSQLGenerator{
		BORepository: repo,
		Dialect:      dialect,
	}, nil
}

// NewBOSQLGeneratorWithWatermark creates a generator with watermark routing enabled.
func NewBOSQLGeneratorWithWatermark(repo BORepository, dialectName string, wmr WatermarkResolver) (*BOSQLGenerator, error) {
	gen, err := NewBOSQLGenerator(repo, dialectName)
	if err != nil {
		return nil, err
	}
	gen.WatermarkResolver = wmr
	return gen, nil
}

// NewBOSQLGeneratorWithCBO creates a generator with watermark routing and CBO telemetry routing enabled.
func NewBOSQLGeneratorWithCBO(repo BORepository, dialectName string, wmr WatermarkResolver, tr TelemetryRouter) (*BOSQLGenerator, error) {
	gen, err := NewBOSQLGeneratorWithWatermark(repo, dialectName, wmr)
	if err != nil {
		return nil, err
	}
	gen.TelemetryRouter = tr
	return gen, nil
}

// SetTelemetryRouter injects a telemetry router after generator construction.
// This allows the CBO to be wired at runtime without changing existing constructors.
func (g *BOSQLGenerator) SetTelemetryRouter(tr TelemetryRouter) {
	g.TelemetryRouter = tr
}

// ResolveEffectiveDialect evaluates Cardinal Rule 4 (Hot/Cold Watermark) to select the correct query engine.
// It returns the effective Dialect after applying (1) explicit override, (2) watermark seam, (3) binding fallback, (4) CBO telemetry override.
func (g *BOSQLGenerator) ResolveEffectiveDialect(req *SQLGenerationRequest, binding *BOBinding) (Dialect, error) {
	// Rule 1: Explicit request override takes priority
	if req.DialectOverride != "" {
		return GetDialect(req.DialectOverride)
	}

	// Rule 4: Evaluate Hot vs. Cold Watermark Seam
	if !req.AsOfDate.IsZero() && g.WatermarkResolver != nil {
		watermark := g.WatermarkResolver.GetHotColdWatermark(req.TenantID)
		// If requesting historical cold data (before watermark) -> route to DataFusion Iceberg Engine
		if req.AsOfDate.Before(watermark) {
			return DataFusionIcebergDialect{}, nil
		}
	}

	// Rule 2: Fallback to active target binding dialect registered in catalog graph
	var baseDialect Dialect
	if binding != nil && binding.DialectName != "" {
		d, err := GetDialect(binding.DialectName)
		if err == nil {
			baseDialect = d
		}
	}

	// Rule 3: Fallback to the generator's default dialect
	if baseDialect == nil {
		baseDialect = g.Dialect
	}

	// Rule 5 (CBO): Consult telemetry router for flavor override on the base dialect
	if g.TelemetryRouter != nil {
		defaultFlavor := dialectToFlavor(baseDialect)
		boKey := req.BusinessObjectID
		if boKey == "" && binding != nil {
			boKey = binding.BOID
		}

		ctx := context.Background()
		recommendedFlavor, err := g.TelemetryRouter.GetOptimalFlavor(ctx, req.TenantID, boKey, defaultFlavor)
		if err == nil && recommendedFlavor != defaultFlavor {
			if overrideDialect, err := GetDialect(recommendedFlavor); err == nil {
				return overrideDialect, nil
			}
		}
	}

	return baseDialect, nil
}

// dialectToFlavor maps a Dialect to a flavor constant for CBO telemetry lookup
func dialectToFlavor(dialect Dialect) string {
	switch dialect.(type) {
	case DataFusionIcebergDialect:
		return "ICEBERG"
	case PostgresDialect:
		return "POSTGRES"
	case SnowflakeDialect:
		return "SNOWFLAKE"
	case SQLServerDialect:
		return "SQLSERVER"
	default:
		return "STARROCKS"
	}
}

// BuildUnionSafeQuery stitches together a hot operational tier query and a cold historical tier query
// into a unified UNION ALL statement while respecting tenant boundaries and query limits.
func (g *BOSQLGenerator) BuildUnionSafeQuery(hotSQL string, coldSQL string, limit int) string {
	unionQuery := fmt.Sprintf("(%s)\nUNION ALL\n(%s)", strings.TrimSpace(hotSQL), strings.TrimSpace(coldSQL))
	if limit > 0 {
		unionQuery += fmt.Sprintf("\nLIMIT %d", limit)
	}
	return unionQuery
}

// BuildAsymmetricCorrectionQuery reconciles late-arriving bitemporal mutations arriving at (Tk >= Wt)
// against base historical states stored in immutable cold Lakehouse storage (Te < Wt) via a coalesced LEFT JOIN.
func (g *BOSQLGenerator) BuildAsymmetricCorrectionQuery(baseHistoricalSQL string, lateCorrectionsSQL string, joinKey string, columns []string, limit int) string {
	var selectCols []string
	if len(columns) == 0 {
		selectCols = append(selectCols, "b.*")
	} else {
		for _, col := range columns {
			if strings.EqualFold(col, joinKey) {
				selectCols = append(selectCols, fmt.Sprintf("b.%s", col))
			} else {
				selectCols = append(selectCols, fmt.Sprintf("COALESCE(c.%s, b.%s) AS %s", col, col, col))
			}
		}
	}

	query := fmt.Sprintf(`WITH base_historical AS (
%s
),
late_corrections AS (
%s
)
SELECT
    %s
FROM base_historical b
LEFT JOIN late_corrections c ON b.%s = c.%s`,
		indentSQL(baseHistoricalSQL, "    "),
		indentSQL(lateCorrectionsSQL, "    "),
		strings.Join(selectCols, ",\n    "),
		joinKey,
		joinKey,
	)

	if limit > 0 {
		query += fmt.Sprintf("\nLIMIT %d", limit)
	}

	return query
}

func indentSQL(sqlText string, indent string) string {
	lines := strings.Split(strings.TrimSpace(sqlText), "\n")
	for i, l := range lines {
		lines[i] = indent + l
	}
	return strings.Join(lines, "\n")
}


// ResolvePolymorphicField physically translates a semantic term based on its binding source type (COLUMN, JSON_PATH, EXPRESSION).
func ResolvePolymorphicField(field BOField, tableAlias string, dialect Dialect) string {
	switch field.SourceType {
	case "JSON_PATH":
		if field.JSONPath != "" {
			pathStr := strings.TrimPrefix(field.JSONPath, "$.")
			colName := extractColumnName(field.PhysicalColumn)
			if colName == "" {
				colName = "tenant_extensions"
			}
			switch dialect.(type) {
			case PostgresDialect:
				return fmt.Sprintf("%s.%s->>'%s'", tableAlias, colName, pathStr)
			default:
				return fmt.Sprintf("get_json_string(%s.%s, '$.%s')", tableAlias, colName, pathStr)
			}
		}

	case "EXPRESSION":
		if field.TransformationSQL != "" {
			return strings.ReplaceAll(field.TransformationSQL, "${alias}", tableAlias)
		}
	}

	// Default: COLUMN mapping
	colName := extractColumnName(field.PhysicalColumn)
	return fmt.Sprintf("%s.%s", tableAlias, colName)
}

func extractColumnName(physicalCol string) string {
	colParts := strings.Split(physicalCol, ".")
	if len(colParts) == 2 {
		return colParts[1]
	}
	return physicalCol
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
func (g *BOSQLGenerator) GenerateSQL(req SQLGenerationRequest) (string, []interface{}, error) {
	// 1. Load Root BO Definition
	rootBO, err := g.BORepository.GetBODefinition(req.BusinessObjectID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load BO definition: %w", err)
	}

	// 2. Initialize Context
	ctx := &GenerationContext{
		Request:      req,
		RootBODef:    rootBO,
		LoadedBOs:    make(map[string]*BODefinition),
		Aliases:      make(map[string]string),
		Joins:        make([]JoinStep, 0),
		NextAliasIdx: 1, // t0 is reserved for root
	}
	ctx.LoadedBOs[rootBO.ID] = rootBO
	ctx.Aliases[""] = "t0" // Root alias (empty path)

	// 3. Resolve Selected Fields (infers joins required for selected columns)
	selectColumns, err := g.ResolveSelectedFields(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve fields: %w", err)
	}

	// 4. Build FROM Clause
	fromClause := g.BuildFROMClause(ctx)

	// 5. Convert Filters (may infer additional joins for filter fields)
	whereClause, err := g.ConvertFilters(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("failed to convert filters: %w", err)
	}

	// 5b. Include user-provided WHERE clause if present
	if req.WhereClause != "" {
		// Convert field names in the WHERE clause to database columns with table aliases
		convertedWhereClause, err := g.ConvertWhereClauseFieldNames(ctx, req.WhereClause)
		if err != nil {
			// If conversion fails, try using it as-is (it might already be in database format)
			convertedWhereClause = req.WhereClause
		}

		if whereClause != "" {
			whereClause += " AND " + convertedWhereClause
		} else {
			whereClause = convertedWhereClause
		}
	}

	// 6. Enforce ABAC tenant isolation at the AST level once the full join graph
	// has been inferred. This injects parameterized predicates on every table node
	// before the final SQL layout is produced.
	if req.TenantID != "" {
		g.InjectTenantScopingToGraph(ctx, req.TenantID)
	}
	if !req.KnowledgeDate.IsZero() {
		g.InjectBitemporalScoping(ctx, req.KnowledgeDate)
	}


	// 7. Build Join Clause (conditions may have been mutated by tenant scoping)
	joinClause := g.BuildJoinClause(ctx)

	// 8. Stitch the root tenant boundary into the primary WHERE cluster.
	// This prevents logical bypasses from subqueries or outer joins that a trailing
	// global WHERE tenant_id = 'X' would be vulnerable to.
	if ctx.RootTenantPredicate != "" {
		if whereClause != "" {
			whereClause = ctx.RootTenantPredicate + " AND " + whereClause
		} else {
			whereClause = ctx.RootTenantPredicate
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

	return query, ctx.Args, nil
}

// paramToken returns the dialect-specific placeholder token for the nth parameter.
// It avoids mutating the Dialect interface (which has a very wide blast radius) while
// still keeping placeholder generation native to each backend.
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
// tenant isolation at the abstract compilation phase. It injects a parameterized
// tenant predicate on the root driving table (t0) and on every relationship
// traversal path (join). Existing join conditions are parenthesized before the
// tenant check is appended to neutralize possible OR-short-circuit injection.
func (g *BOSQLGenerator) InjectTenantScopingToGraph(ctx *GenerationContext, tenantID string) {
	rootAlias := "t0" // Standard baseline root driving table alias

	if ctx.Args == nil {
		ctx.Args = make([]interface{}, 0)
	}

	// 1. Root table boundary.
	ctx.ParamCounter++
	rootParamToken := paramToken(g.Dialect, ctx.ParamCounter)
	ctx.Args = append(ctx.Args, tenantID)
	ctx.RootTenantPredicate = fmt.Sprintf("%s.tenant_id = %s", rootAlias, rootParamToken)

	// 2. Relationship traversal boundaries.
	for i := range ctx.Joins {
		step := &ctx.Joins[i]

		stepAlias := step.Alias
		if stepAlias == "" {
			// Fallback for join steps created without an explicit alias.
			stepAlias = fmt.Sprintf("t%d", i+1)
		}

		ctx.ParamCounter++
		joinParamToken := paramToken(g.Dialect, ctx.ParamCounter)
		ctx.Args = append(ctx.Args, tenantID)

		tenantCondition := fmt.Sprintf("%s.tenant_id = %s", stepAlias, joinParamToken)
		if step.Condition == "" {
			step.Condition = tenantCondition
		} else {
			step.Condition = fmt.Sprintf("(%s) AND %s", step.Condition, tenantCondition)
		}
	}
}

// InjectBitemporalScoping injects valid-time system_valid_from / system_valid_to bitemporal predicates into the query AST.
func (g *BOSQLGenerator) InjectBitemporalScoping(ctx *GenerationContext, knowledgeDate time.Time) {
	rootAlias := "t0"
	if ctx.Args == nil {
		ctx.Args = make([]interface{}, 0)
	}

	ctx.ParamCounter++
	pToken := paramToken(g.Dialect, ctx.ParamCounter)
	ctx.Args = append(ctx.Args, knowledgeDate.Format(time.RFC3339))

	bitemporalPredicate := fmt.Sprintf("(%s.system_valid_from <= %s AND (%s.system_valid_to IS NULL OR %s.system_valid_to > %s))",
		rootAlias, pToken, rootAlias, rootAlias, pToken)

	if ctx.RootTenantPredicate != "" {
		ctx.RootTenantPredicate = ctx.RootTenantPredicate + " AND " + bitemporalPredicate
	} else {
		ctx.RootTenantPredicate = bitemporalPredicate
	}
}


// ResolveSelectedFields resolves paths to physical columns and infers joins
func (g *BOSQLGenerator) ResolveSelectedFields(ctx *GenerationContext) ([]string, error) {
	var columns []string
	for _, fieldPath := range ctx.Request.SelectedFields {
		sqlExpr, fieldLabel, err := g.ResolvePathWithLabel(ctx, fieldPath)
		if err != nil {
			return nil, fmt.Errorf("error resolving path %s: %w", fieldPath, err)
		}
		// Alias the column with the field's display name or label
		columns = append(columns, fmt.Sprintf("%s AS \"%s\"", sqlExpr, fieldLabel))
	}
	return columns, nil
}

// ResolvePathWithLabel walks the path, adds joins if needed, and returns "alias.column" plus a human-friendly label
func (g *BOSQLGenerator) ResolvePathWithLabel(ctx *GenerationContext, path string) (string, string, error) {
	// Split path: "orders.items.price" -> ["orders", "items", "price"]
	parts := strings.Split(path, ".")

	currentPath := ""
	currentBO := ctx.RootBODef
	currentAlias := ctx.Aliases[""]

	// Iterate through parts to find the target field
	// Note: Intermediate parts MUST be reference fields (relationships)
	// The last part is the field to select.

	for i, part := range parts {
		// Find field in current BO
		var foundField *BOField
		for _, f := range currentBO.Fields {
			if f.Name == part || f.ID == part { // Match by name/path or UUID
				foundField = &f
				break
			}
		}

		if foundField == nil {
			return "", "", fmt.Errorf("field '%s' not found in BO '%s'", part, currentBO.ID)
		}

		// Calculate path for this segment
		segmentName := foundField.Name
		if segmentName == "" {
			segmentName = part
		}
		if currentPath == "" {
			currentPath = segmentName
		} else {
			currentPath = currentPath + "." + segmentName
		}

		// If this is the last part, we are done
		if i == len(parts)-1 {
			if foundField.PhysicalColumn == "" && foundField.TransformationSQL == "" {
				return "", "", fmt.Errorf("no physical column mapping for field '%s'", foundField.ID)
			}
			sqlExpr := ResolvePolymorphicField(*foundField, currentAlias, g.Dialect)

			// Determine label: use DisplayName, fallback to Name
			label := foundField.DisplayName
			if label == "" {
				label = foundField.Name
			}
			if label == "" {
				label = part // Final fallback to the input path
			}

			return sqlExpr, label, nil
		}


		// If not last part, it MUST be a reference/relationship
		if foundField.Type != "reference" || foundField.ReferenceBOID == "" {
			return "", "", fmt.Errorf("field '%s' is not a reference, cannot traverse", part)
		}

		// Check if we already have an alias for this path (Join Reuse)
		if existingAlias, ok := ctx.Aliases[currentPath]; ok {
			currentAlias = existingAlias
			// Load the target BO to continue traversal
			// We need to fetch it if not in cache (though we must have fetched it to create the alias, unless reused differently)
			targetBO, ok := ctx.LoadedBOs[foundField.ReferenceBOID]
			if !ok {
				// Should have been loaded when alias was created. Reloading just in case.
				var err error
				targetBO, err = g.BORepository.GetBODefinition(foundField.ReferenceBOID)
				if err != nil {
					return "", "", err
				}
				ctx.LoadedBOs[foundField.ReferenceBOID] = targetBO
			}
			currentBO = targetBO
			continue
		}

		// New Join Logic
		targetBOID := foundField.ReferenceBOID
		targetBO, ok := ctx.LoadedBOs[targetBOID]
		if !ok {
			var err error
			targetBO, err = g.BORepository.GetBODefinition(targetBOID)
			if err != nil {
				return "", "", err
			}
			ctx.LoadedBOs[targetBOID] = targetBO
		}

		// Create new alias
		newAlias := fmt.Sprintf("t%d", ctx.NextAliasIdx)
		ctx.NextAliasIdx++
		ctx.Aliases[currentPath] = newAlias

		// Create Join Step
		// We join Current Table (currentAlias) to Target Table (newAlias)
		// Condition: ${SOURCE}.field_col = ${TARGET}.id (assuming Ref field holds ID)

		// Determine Join Condition
		// Use physical column of the reference field in Current BO
		refColParts := strings.Split(foundField.PhysicalColumn, ".")
		sourceCol := refColParts[len(refColParts)-1] // just column name

		// Target is "id" for now (implicit)
		condition := fmt.Sprintf("%s.%s = %s.id", currentAlias, sourceCol, newAlias)

		joinStep := JoinStep{
			Type:      "LEFT", // Default to LEFT JOIN for safety
			ToTable:   fmt.Sprintf("%s AS %s", targetBO.DrivingTable, newAlias),
			Condition: condition,
			Alias:     newAlias,
		}
		ctx.Joins = append(ctx.Joins, joinStep)

		// Advance cursors
		currentAlias = newAlias
		currentBO = targetBO
	}

	return "", "", fmt.Errorf("unexpected end of resolution")
}

// ResolvePath walks the path, adds joins if needed, and returns "alias.column"
// For backward compatibility, this wraps ResolvePathWithLabel and discards the label
func (g *BOSQLGenerator) ResolvePath(ctx *GenerationContext, path string) (string, error) {
	sqlExpr, _, err := g.ResolvePathWithLabel(ctx, path)
	return sqlExpr, err
}

func sanitizeTableName(tableName string) string {
	tableName = strings.TrimSpace(tableName)
	tableName = strings.TrimPrefix(tableName, "/")
	if strings.Contains(tableName, "/") {
		parts := strings.Split(tableName, "/")
		if len(parts) == 2 {
			return fmt.Sprintf("\"%s\".\"%s\"", parts[0], parts[1])
		}
		tableName = strings.ReplaceAll(tableName, "/", "_")
	}
	if strings.Contains(tableName, ".") {
		parts := strings.Split(tableName, ".")
		if len(parts) == 2 {
			return fmt.Sprintf("\"%s\".\"%s\"", strings.Trim(parts[0], "\""), strings.Trim(parts[1], "\""))
		}
	}
	return tableName
}

func (g *BOSQLGenerator) BuildFROMClause(ctx *GenerationContext) string {
	table := sanitizeTableName(ctx.RootBODef.DrivingTable)
	return fmt.Sprintf("%s AS t0", table)
}

func (g *BOSQLGenerator) BuildJoinClause(ctx *GenerationContext) string {
	var sb strings.Builder
	for _, join := range ctx.Joins {
		sb.WriteString(fmt.Sprintf("%s JOIN %s ON %s\n", join.Type, join.ToTable, join.Condition))
	}
	return sb.String()
}

func (g *BOSQLGenerator) ConvertFilters(ctx *GenerationContext) (string, error) {
	var whereParts []string

	for _, filter := range ctx.Request.Filters {
		fieldPath := filter.FieldID

		sqlExpr, err := g.ResolvePath(ctx, fieldPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve filter field %s: %w", fieldPath, err)
		}

		op := strings.ToUpper(strings.TrimSpace(filter.Operator))
		if op == "" || op == "EQ" {
			op = "="
		} else if op == "NEQ" {
			op = "!="
		} else if op == "GT" {
			op = ">"
		} else if op == "LT" {
			op = "<"
		} else if op == "GTE" {
			op = ">="
		} else if op == "LTE" {
			op = "<="
		}

		// Handle IS NULL / IS NOT NULL / Boolean
		if op == "IS NULL" || op == "NULL" || op == "IS_NULL" {
			whereParts = append(whereParts, fmt.Sprintf("%s IS NULL", sqlExpr))
			continue
		}
		if op == "IS NOT NULL" || op == "NOT_NULL" || op == "NOT NULL" || op == "IS_NOT_NULL" {
			whereParts = append(whereParts, fmt.Sprintf("%s IS NOT NULL", sqlExpr))
			continue
		}
		if op == "IS TRUE" || op == "IS_TRUE" {
			whereParts = append(whereParts, fmt.Sprintf("%s IS TRUE", sqlExpr))
			continue
		}
		if op == "IS FALSE" || op == "IS_FALSE" {
			whereParts = append(whereParts, fmt.Sprintf("%s IS FALSE", sqlExpr))
			continue
		}

		// Format value
		valStr := ""
		switch v := filter.Value.(type) {
		case string:
			cleanV := strings.ReplaceAll(v, "'", "''")
			switch op {
			case "CONTAINS", "CONTAIN":
				op = "ILIKE"
				valStr = fmt.Sprintf("'%%%s%%'", cleanV)
			case "STARTS WITH", "STARTS_WITH", "START_WITH":
				op = "ILIKE"
				valStr = fmt.Sprintf("'%s%%'", cleanV)
			case "ENDS WITH", "ENDS_WITH", "END_WITH":
				op = "ILIKE"
				valStr = fmt.Sprintf("'%%%s'", cleanV)
			case "IN", "NOT IN":
				items := strings.Split(cleanV, ",")
				var formattedItems []string
				for _, item := range items {
					trimmed := strings.TrimSpace(item)
					if trimmed != "" {
						formattedItems = append(formattedItems, fmt.Sprintf("'%s'", trimmed))
					}
				}
				if len(formattedItems) == 0 {
					valStr = "('')"
				} else {
					valStr = fmt.Sprintf("(%s)", strings.Join(formattedItems, ", "))
				}
			default:
				valStr = fmt.Sprintf("'%s'", cleanV)
			}
		case []interface{}:
			var formattedItems []string
			for _, item := range v {
				switch iv := item.(type) {
				case string:
					formattedItems = append(formattedItems, fmt.Sprintf("'%s'", strings.ReplaceAll(iv, "'", "''")))
				case int, int64, float64:
					formattedItems = append(formattedItems, fmt.Sprintf("%v", iv))
				default:
					formattedItems = append(formattedItems, fmt.Sprintf("'%v'", iv))
				}
			}
			if len(formattedItems) == 0 {
				valStr = "('')"
			} else {
				valStr = fmt.Sprintf("(%s)", strings.Join(formattedItems, ", "))
			}
		case []string:
			var formattedItems []string
			for _, item := range v {
				formattedItems = append(formattedItems, fmt.Sprintf("'%s'", strings.ReplaceAll(item, "'", "''")))
			}
			if len(formattedItems) == 0 {
				valStr = "('')"
			} else {
				valStr = fmt.Sprintf("(%s)", strings.Join(formattedItems, ", "))
			}
		case int, int64, float64:
			valStr = fmt.Sprintf("%v", v)
		default:
			valStr = fmt.Sprintf("'%v'", v)
		}

		clause := fmt.Sprintf("%s %s %s", sqlExpr, op, valStr)
		whereParts = append(whereParts, clause)
	}

	return strings.Join(whereParts, " AND "), nil
}

// ConvertWhereClauseFieldNames converts field names in a WHERE clause string to database column references with table aliases
// For example: "CUSTOMER_ADDRESS != 'value'" becomes "t0.address != 'value'"
// Handles multiple formats: field names, display names, uppercase variants, semantic terms
func (g *BOSQLGenerator) ConvertWhereClauseFieldNames(ctx *GenerationContext, whereClause string) (string, error) {
	if whereClause == "" {
		return "", nil
	}

	// Defensive check for nil context or BO definition
	if ctx == nil || ctx.RootBODef == nil {
		// If we don't have context/BO info, return the clause as-is
		return whereClause, nil
	}

	if len(ctx.RootBODef.Fields) == 0 {
		// No fields to map, return as-is
		return whereClause, nil
	}

	// Build a comprehensive mapping of possible field references to database columns
	// Each field can be referenced in multiple ways
	fieldReferences := make(map[string]string) // Maps any form of field name to "t0.columnname"

	for _, field := range ctx.RootBODef.Fields {
		// Extract just the column name from PhysicalColumn (e.g., "customers.address" -> "address")
		columnName := field.PhysicalColumn
		if idx := strings.LastIndex(columnName, "."); idx >= 0 {
			columnName = columnName[idx+1:]
		}

		replacement := "t0." + columnName

		// Add mappings for various forms of the field name
		if field.Name != "" {
			// Exact name
			fieldReferences[field.Name] = replacement
			// Uppercase name
			fieldReferences[strings.ToUpper(field.Name)] = replacement
			// lowercase name
			fieldReferences[strings.ToLower(field.Name)] = replacement
			// With underscores for spaces
			withUnderscores := strings.ReplaceAll(field.Name, " ", "_")
			fieldReferences[withUnderscores] = replacement
			fieldReferences[strings.ToUpper(withUnderscores)] = replacement
		}

		if field.DisplayName != "" {
			// Display name as-is
			fieldReferences[field.DisplayName] = replacement
			// Display name uppercase
			fieldReferences[strings.ToUpper(field.DisplayName)] = replacement
			// Display name with spaces replaced by underscores
			withUnderscores := strings.ReplaceAll(field.DisplayName, " ", "_")
			fieldReferences[withUnderscores] = replacement
			fieldReferences[strings.ToUpper(withUnderscores)] = replacement
			// Display name with spaces replaced by nothing
			noSpaces := strings.ReplaceAll(field.DisplayName, " ", "")
			fieldReferences[noSpaces] = replacement
			fieldReferences[strings.ToUpper(noSpaces)] = replacement
		}
	}

	// Common operators to detect field references
	operators := []string{" = ", " != ", " <> ", " > ", " < ", " >= ", " <= ", " LIKE ", " IN ", " AND ", " OR ", " IS NULL", " IS NOT NULL"}

	result := whereClause

	// Try to replace field references
	// Sort by length (longest first) to avoid partial matches
	var fieldNames []string
	for fieldRef := range fieldReferences {
		fieldNames = append(fieldNames, fieldRef)
	}
	// Sort by length in descending order
	sort.Slice(fieldNames, func(i, j int) bool {
		return len(fieldNames[i]) > len(fieldNames[j])
	})

	for _, fieldRef := range fieldNames {
		replacement := fieldReferences[fieldRef]

		// Try with operators
		for _, op := range operators {
			pattern := fieldRef + op
			if strings.Contains(result, pattern) {
				newPattern := replacement + op
				result = strings.ReplaceAll(result, pattern, newPattern)
			}
		}

		// Try at the end of the clause (after it might have been modified)
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

// ResolveSemanticRequest converts a semantic query request to the internal UUID-based format
func (g *BOSQLGenerator) ResolveSemanticRequest(semanticReq *SemanticSQLGenerationRequest, tenantID, datasourceID string) (*SQLGenerationRequest, error) {
	// Step 1: Look up the Business Object by technical name
	boDef, err := g.BORepository.GetBOByTechnicalName(semanticReq.Datasource, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to find business object '%s': %w", semanticReq.Datasource, err)
	}

	// Step 2: Resolve semantic field terms to field UUIDs
	selectedFieldIDs := make([]string, len(semanticReq.Select))
	for i, semanticField := range semanticReq.Select {
		field, err := g.findFieldBySemanticTerm(boDef, semanticField.Term)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve field '%s': %w", semanticField.Term, err)
		}
		selectedFieldIDs[i] = field.ID
	}

	// Step 3: Convert semantic filters to UUID-based filters
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

	return &SQLGenerationRequest{
		TenantID:         tenantID,
		BusinessObjectID: boDef.ID,
		SelectedFields:   selectedFieldIDs,
		Filters:          filters,
		Limit:            semanticReq.Limit,
	}, nil
}

// findFieldBySemanticTerm finds a field in the BO definition by semantic term name
func (g *BOSQLGenerator) findFieldBySemanticTerm(boDef *BODefinition, term string) (*BOField, error) {
	// First try exact match on Name
	for _, field := range boDef.Fields {
		if field.Name == term {
			return &field, nil
		}
	}

	// Then try match on DisplayName
	for _, field := range boDef.Fields {
		if field.DisplayName == term {
			return &field, nil
		}
	}

	// Also check field_name (from database)
	// We need to extend BOField to include field_name, or check against a mapping
	// For now, let's add some common mappings
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

// GenerateSQLFromSemantic generates SQL from a semantic query request. It returns
// the generated SQL, the parameter values for any placeholders, and an error.
func (g *BOSQLGenerator) GenerateSQLFromSemantic(semanticReq *SemanticSQLGenerationRequest, tenantID, datasourceID string) (string, []interface{}, error) {
	// Resolve semantic request to UUID-based request
	req, err := g.ResolveSemanticRequest(semanticReq, tenantID, datasourceID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to resolve semantic request: %w", err)
	}

	// Generate SQL using existing logic
	return g.GenerateSQL(*req)
}

// ValidationRuleCompilationRequest specifies parameters for compiling a semantic validation rule into physical binding SQL
type ValidationRuleCompilationRequest struct {
	BusinessObjectID string                 `json:"businessObjectId"`
	TenantID         string                 `json:"tenantId"`
	RuleID           string                 `json:"ruleId,omitempty"`
	RuleType         string                 `json:"ruleType"` // field_format, cardinality, uniqueness, referential_integrity, business_logic
	ConditionJSON    map[string]interface{} `json:"conditionJson"`
	Limit            int                    `json:"limit,omitempty"`
}

// CompiledValidationSQL holds compiled physical SQL for validating data in a binding
type CompiledValidationSQL struct {
	SQL            string        `json:"sql"`
	Args           []interface{} `json:"args"`
	PhysicalColumn string        `json:"physicalColumn"`
}

// CompileValidationRuleSQL compiles a semantic validation rule into physical binding SQL execution queries.
func (g *BOSQLGenerator) CompileValidationRuleSQL(compReq ValidationRuleCompilationRequest) (*CompiledValidationSQL, error) {
	// Rule 1.3 Defense: Parse UUID for BusinessObjectID
	if compReq.BusinessObjectID != "" {
		if _, err := uuid.Parse(compReq.BusinessObjectID); err != nil {
			return nil, fmt.Errorf("invalid businessObjectId format: %w", err)
		}
	} else {
		return nil, fmt.Errorf("businessObjectId is required")
	}

	// Rule 1.3 Defense: Parse UUID for TenantID if provided
	if compReq.TenantID != "" {
		if _, err := uuid.Parse(compReq.TenantID); err != nil {
			return nil, fmt.Errorf("invalid tenantId format: %w", err)
		}
	}

	boDef, err := g.BORepository.GetBODefinition(compReq.BusinessObjectID)
	if err != nil || boDef == nil {
		return nil, fmt.Errorf("failed to retrieve business object definition for id '%s': %v", compReq.BusinessObjectID, err)
	}

	// Target field resolution from condition JSON
	var targetTerm string
	if fieldVal, ok := compReq.ConditionJSON["field"].(string); ok {
		targetTerm = fieldVal
	} else if fieldIdVal, ok := compReq.ConditionJSON["fieldId"].(string); ok {
		targetTerm = fieldIdVal
	} else if termVal, ok := compReq.ConditionJSON["term"].(string); ok {
		targetTerm = termVal
	}

	var matchedField *BOField
	if targetTerm != "" {
		mf, err := g.findFieldBySemanticTerm(boDef, targetTerm)
		if err == nil && mf != nil {
			matchedField = mf
		} else {
			// Fallback: check if targetTerm matches field ID directly
			for _, f := range boDef.Fields {
				if f.ID == targetTerm || f.Name == targetTerm {
					matchedField = &f
					break
				}
			}
		}
	}

	physicalCol := "t0.id"
	if matchedField != nil && matchedField.PhysicalColumn != "" {
		physicalCol = matchedField.PhysicalColumn
		if !strings.Contains(physicalCol, ".") {
			physicalCol = "t0." + physicalCol
		}
	}

	limit := compReq.Limit
	if limit <= 0 {
		limit = 100
	}

	table := boDef.DrivingTable
	if table == "" {
		table = "target_table"
	}

	var sqlStr string
	args := []interface{}{}

	switch compReq.RuleType {
	case "uniqueness":
		sqlStr = fmt.Sprintf("SELECT %s, COUNT(*) AS dup_count FROM %s t0 WHERE 1=1", physicalCol, table)
		if compReq.TenantID != "" {
			sqlStr += " AND t0.tenant_id = $1"
			args = append(args, compReq.TenantID)
		}
		sqlStr += fmt.Sprintf(" GROUP BY %s HAVING COUNT(*) > 1 LIMIT %d", physicalCol, limit)

	case "field_format":
		pattern, _ := compReq.ConditionJSON["pattern"].(string)
		if pattern == "" {
			pattern = ".*"
		}
		sqlStr = fmt.Sprintf("SELECT t0.* FROM %s t0 WHERE (%s IS NOT NULL AND %s !~ $1)", table, physicalCol, physicalCol)
		args = append(args, pattern)
		if compReq.TenantID != "" {
			sqlStr += fmt.Sprintf(" AND t0.tenant_id = $%d", len(args)+1)
			args = append(args, compReq.TenantID)
		}
		sqlStr += fmt.Sprintf(" LIMIT %d", limit)

	case "cardinality":
		minVal, hasMin := compReq.ConditionJSON["min"].(float64)
		maxVal, hasMax := compReq.ConditionJSON["max"].(float64)
		whereConds := []string{fmt.Sprintf("%s IS NOT NULL", physicalCol)}
		if hasMin {
			args = append(args, minVal)
			whereConds = append(whereConds, fmt.Sprintf("%s < $%d", physicalCol, len(args)))
		}
		if hasMax {
			args = append(args, maxVal)
			whereConds = append(whereConds, fmt.Sprintf("%s > $%d", physicalCol, len(args)))
		}
		sqlStr = fmt.Sprintf("SELECT t0.* FROM %s t0 WHERE (%s)", table, strings.Join(whereConds, " OR "))
		if compReq.TenantID != "" {
			args = append(args, compReq.TenantID)
			sqlStr += fmt.Sprintf(" AND t0.tenant_id = $%d", len(args))
		}
		sqlStr += fmt.Sprintf(" LIMIT %d", limit)

	default: // business_logic, referential_integrity
		op, _ := compReq.ConditionJSON["operator"].(string)
		if op == "" {
			op = "="
		}
		val := compReq.ConditionJSON["value"]
		if val != nil {
			args = append(args, val)
			sqlStr = fmt.Sprintf("SELECT t0.* FROM %s t0 WHERE NOT (%s %s $1)", table, physicalCol, op)
		} else {
			sqlStr = fmt.Sprintf("SELECT t0.* FROM %s t0 WHERE %s IS NULL", table, physicalCol)
		}
		if compReq.TenantID != "" {
			args = append(args, compReq.TenantID)
			sqlStr += fmt.Sprintf(" AND t0.tenant_id = $%d", len(args))
		}
		sqlStr += fmt.Sprintf(" LIMIT %d", limit)
	}

	return &CompiledValidationSQL{
		SQL:            sqlStr,
		Args:           args,
		PhysicalColumn: physicalCol,
	}, nil
}
