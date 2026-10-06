package querybuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// GeneratedCubeDDL contains deterministic DDL for one cube materialization.
type GeneratedCubeDDL struct {
	// MaterializationName is the physical object name.
	MaterializationName string
	// DDL is the full statement.
	DDL string
	// ContentHash is SHA-256 over the DDL text.
	ContentHash string
	// MeasureColumns maps a cube metric ID to the physical column that carries
	// its aggregate. The router uses this to rewrite a measure reference onto
	// the materialization; it is also what makes the generated DDL auditable
	// against the metrics that produced it.
	MeasureColumns map[string]string
	// GroupByColumns are the physical key columns, in deterministic order.
	GroupByColumns []string
}

// CubeDDLGenerator renders StarRocks materialized views for cube grains.
//
// It deliberately does NOT reuse StarRocksMaterializationManager.GenerateMVDDL.
// That function hardcodes SUM(notional) as the default measure and an "oms."
// schema prefix for the source table, so it would emit a syntactically valid but
// semantically wrong MV for any real cube — silently, because the DDL would
// still parse. A cube must emit one measure column per declared metric, derived
// from the governed metric's own compiled expression.
type CubeDDLGenerator struct {
	compiler *MetricCompiler
	// defaultSchema is the schema physical objects live in when the source
	// table does not carry one explicitly.
	defaultSchema string
}

// SetTermGate installs the C2 PII gate used to clear every term a metric reads
// before it becomes a column in the generated MV. It must be called before
// GenerateCubeMaterializationDDL; see that method for why it is not optional.
func (g *CubeDDLGenerator) SetTermGate(gate MetricTermGate) {
	g.compiler = g.compiler.WithTermGate(gate)
}

// NewCubeDDLGenerator creates a generator bound to a metric compiler.
func NewCubeDDLGenerator(dialect string) *CubeDDLGenerator {
	if strings.TrimSpace(dialect) == "" {
		dialect = "starrocks"
	}
	// boresolver.GetDialect already maps starrocks/mysql onto the Postgres
	// dialect, which is the correct expression syntax for StarRocks. An
	// unrecognised dialect falls back to Postgres rather than failing here, so
	// DDL generation degrades to portable SQL.
	d, err := boresolver.GetDialect(dialect)
	if err != nil || d == nil {
		d = boresolver.PostgresDialect{}
	}
	return &CubeDDLGenerator{
		compiler:      NewMetricCompiler(d),
		defaultSchema: "uisce_cube",
	}
}

// SetDefaultSchema overrides the target schema for generated materializations.
func (g *CubeDDLGenerator) SetDefaultSchema(schema string) {
	if s := sanitizeIdentifier(schema); s != "" {
		g.defaultSchema = s
	}
}

// GenerateCubeMaterializationDDL renders the MV for one grain of one cube.
//
// sourceTable is the fully-qualified physical source (e.g. "oms.sales"). It is
// passed in rather than derived from the cube's BOID, because the cube is
// portable metadata that stores no physical information — the binding layer
// resolves the tenant's actual table.
//
// metricLookup supplies the governed metrics by ID. Every metric in the cube
// must be present; a missing one is an error rather than a skipped column,
// because a materialization that silently omits a measure would serve wrong
// totals with no signal.
func (g *CubeDDLGenerator) GenerateCubeMaterializationDDL(
	tenantID string,
	isGoldCopy bool,
	cube CubeDefinition,
	grain []string,
	sourceTable string,
	metricLookup map[string]MetricDefinition,
	varBindings map[string]interface{},
) (*GeneratedCubeDDL, error) {
	return g.generateCubeMaterializationDDL(tenantID, isGoldCopy, cube, grain, sourceTable, nil, metricLookup, varBindings)
}

// GenerateFederatedCubeMaterializationDDL renders the MV for a federated cube
// grain. fromSQL is the compiled JOIN expression (CUBE-2.2); dimExprs maps
// sanitized grain column names to qualified SQL expressions (alias.col) so
// multi-table FROM clauses stay unambiguous.
func (g *CubeDDLGenerator) GenerateFederatedCubeMaterializationDDL(
	tenantID string,
	isGoldCopy bool,
	cube CubeDefinition,
	grain []string,
	fromSQL string,
	dimExprs map[string]string,
	metricLookup map[string]MetricDefinition,
	varBindings map[string]interface{},
) (*GeneratedCubeDDL, error) {
	return g.generateCubeMaterializationDDL(tenantID, isGoldCopy, cube, grain, fromSQL, dimExprs, metricLookup, varBindings)
}

func (g *CubeDDLGenerator) generateCubeMaterializationDDL(
	tenantID string,
	isGoldCopy bool,
	cube CubeDefinition,
	grain []string,
	sourceTable string,
	dimExprs map[string]string,
	metricLookup map[string]MetricDefinition,
	varBindings map[string]interface{},
) (*GeneratedCubeDDL, error) {

	if err := ValidateCubeStructural(cube); err != nil {
		return nil, err
	}
	if len(grain) == 0 {
		return nil, ErrCubeNoGrains
	}
	if strings.TrimSpace(sourceTable) == "" {
		return nil, fmt.Errorf("cube materialization requires a resolved source table")
	}
	if len(cube.MetricIDs) == 0 {
		return nil, ErrCubeNoMetrics
	}

	// Fail closed. This method's output is not a query - it is a CREATE TABLE
	// MATERIALIZED VIEW, so a term that slips through here does not get masked
	// at read time, it gets aggregated into a stored artifact that every
	// downstream consumer of the MV inherits, masked or not. Query-time masking
	// tiers are not a defence here because the MV is read by things that never
	// pass through them.
	//
	// Generating DDL without a gate would make the C2 PII gate opt-in, and an
	// opt-in security control on a deploy path is the shape of defect this
	// gate exists to prevent.
	if g.compiler.termGate == nil {
		return nil, fmt.Errorf("cube materialization DDL requires a term gate: call SetTermGate before generating, so every metric term is cleared for reading before it becomes an MV column")
	}

	// Dimension columns, deduplicated and ordered deterministically so the DDL
	// is content-hash stable.
	keyCols := make([]string, 0, len(grain))
	seen := make(map[string]bool, len(grain))
	for _, d := range grain {
		col := sanitizeIdentifier(d)
		if col == "" || seen[col] {
			continue
		}
		seen[col] = true
		keyCols = append(keyCols, col)
	}
	if len(keyCols) == 0 {
		return nil, fmt.Errorf("cube grain %v contains no usable dimension columns", grain)
	}
	sort.Strings(keyCols)

	selectDims := make([]string, 0, len(keyCols))
	groupByDims := make([]string, 0, len(keyCols))
	for _, col := range keyCols {
		expr := col
		if dimExprs != nil {
			if q, ok := dimExprs[col]; ok && strings.TrimSpace(q) != "" {
				expr = strings.TrimSpace(q)
			}
		}
		if expr == col {
			selectDims = append(selectDims, col)
			groupByDims = append(groupByDims, col)
		} else {
			selectDims = append(selectDims, fmt.Sprintf("%s AS %s", expr, col))
			groupByDims = append(groupByDims, expr)
		}
	}

	// Federated FROM (JOIN …) cannot use StarRocks SYNC MVs (single-table only).
	// ASYNC + EVERY interval is required for JDBC/external multi-table grains.
	federated := dimExprs != nil && strings.Contains(strings.ToUpper(sourceTable), " JOIN ")
	drivingAlias := "t0"
	if federated {
		if a := firstSQLTableAlias(sourceTable); a != "" {
			drivingAlias = a
		}
	}

	// One measure column per governed metric, named from the metric ID so the
	// router can map a requested measure back to it unambiguously.
	measureExprs := make([]string, 0, len(keyCols)+len(cube.MetricIDs))
	measureExprs = append(measureExprs, selectDims...)
	measureColumns := make(map[string]string, len(cube.MetricIDs))

	// Iterate metric IDs in sorted order for determinism.
	ids := make([]string, len(cube.MetricIDs))
	copy(ids, cube.MetricIDs)
	sort.Strings(ids)

	for _, id := range ids {
		metric, ok := metricLookup[strings.ToLower(strings.TrimSpace(id))]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrCubeUnknownMetric, id)
		}
		compiled, err := g.compiler.CompileMetric(metric, varBindings, metricLookup)
		if err != nil {
			return nil, fmt.Errorf("compile metric %q for cube %q: %w", id, cube.Name, err)
		}
		// Guard against the exact bug class GenerateMVDDL embodies: a measure
		// that is really a hardcoded default would pass every other check here.
		if strings.TrimSpace(compiled.SQLExpr) == "" {
			return nil, fmt.Errorf("metric %q compiled to an empty expression", id)
		}
		// A materialized view is a stored statement, not a prepared one: it has
		// no argument list to bind against. A formula metric with a bound
		// variable compiles to positional placeholders ($1, $2, ...) that would
		// reach StarRocks unbound, so reject it here with a real reason instead
		// of letting the deploy fail on a syntax error. See ADR-025.
		if len(compiled.Args) > 0 {
			return nil, fmt.Errorf("metric %q is a parameterized formula (%d bound argument(s)); a cube measure must be a self-contained expression, so inline the value or use an aggregation",
				id, len(compiled.Args))
		}
		if isHardcodedDefaultMeasure(compiled.SQLExpr, metric) {
			return nil, fmt.Errorf("metric %q compiled to a hardcoded default measure (%q) rather than its own expression",
				id, compiled.SQLExpr)
		}

		col := sanitizeIdentifier(id)
		if col == "" {
			return nil, fmt.Errorf("metric id %q has no usable column name", id)
		}
		measureColumns[strings.ToLower(strings.TrimSpace(id))] = col
		sqlExpr := compiled.SQLExpr
		if federated && drivingAlias != "t0" {
			// MetricCompiler hardcodes t0.<term>; remap onto the federation driving alias.
			sqlExpr = rewriteMetricTableAlias(sqlExpr, "t0", drivingAlias)
		}
		measureExprs = append(measureExprs, fmt.Sprintf("%s AS %s", sqlExpr, col))
	}

	name := CubeMaterializationName(tenantID, isGoldCopy, cube, keyCols)
	// StarRocks SYNC MVs reject PARTITION BY. Only emit a partition clause when
	// the grain carries a recognizable time dimension (ASYNC-capable grains).
	partitionCol := timeColumnOf(keyCols)
	if partitionCol != "" && dimExprs != nil {
		if q, ok := dimExprs[partitionCol]; ok && strings.TrimSpace(q) != "" {
			partitionCol = strings.TrimSpace(q)
		}
	}
	partitionClause := ""
	if partitionCol != "" {
		partitionClause = fmt.Sprintf("\nPARTITION BY date_trunc('day', %s)", partitionCol)
	}
	refreshClause := ""
	if federated {
		// External/JDBC multi-table MVs require an explicit ASYNC refresh interval.
		refreshClause = "\nREFRESH ASYNC EVERY(INTERVAL 1 HOUR)"
	}
	ddl := fmt.Sprintf(`CREATE MATERIALIZED VIEW %s%s%s
PROPERTIES (
  "replication_num" = "1"
)
AS SELECT
  %s
FROM %s
GROUP BY %s;`,
		name,
		refreshClause,
		partitionClause,
		strings.Join(measureExprs, ",\n  "),
		strings.TrimSpace(sourceTable),
		strings.Join(groupByDims, ", "),
	)

	hash := sha256.Sum256([]byte(ddl))
	return &GeneratedCubeDDL{
		MaterializationName: name,
		DDL:                 ddl,
		ContentHash:         hex.EncodeToString(hash[:]),
		MeasureColumns:      measureColumns,
		GroupByColumns:      keyCols,
	}, nil
}

// isHardcodedDefaultMeasure reports whether a compiled expression is the
// SUM(notional) default that GenerateMVDDL falls back to, rather than the
// metric's own expression. A metric that genuinely aggregates "notional" is
// legitimate; the check therefore only fires when the metric's own expression
// does not name that term.
func isHardcodedDefaultMeasure(expr string, metric MetricDefinition) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(expr, " ", ""), "\n", ""))
	if normalized != "sum(notional)" {
		return false
	}
	// If the metric genuinely targets the notional term, this is its own
	// expression and is fine.
	if metric.Expression.Kind == "aggregation" &&
		strings.EqualFold(strings.TrimSpace(metric.Expression.TermNodeID), "notional") {
		return false
	}
	return true
}

// CubeMaterializationName derives the deterministic physical object name.
//
// Gold copy objects are shared by every vanilla adopting tenant, which is what
// preserves the multi-tenant economics: a tenant only gets its own object if
// its adoption actually changes the physical shape.
func CubeMaterializationName(tenantID string, isGoldCopy bool, cube CubeDefinition, keyCols []string) string {
	scope := "gold"
	if !isGoldCopy && strings.TrimSpace(tenantID) != "" {
		scope = sanitizeIdentifier(tenantID)
	}
	return fmt.Sprintf("cube_%s_%s_%s",
		scope,
		sanitizeIdentifier(cube.BOID),
		sanitizeIdentifier(cube.Name+"_"+strings.Join(keyCols, "_")),
	)
}

// timeColumnOf picks a recognizable time dimension for PARTITION BY.
// Returns empty when the grain has no time hint — StarRocks SYNC MVs cannot
// use PARTITION BY, so non-time grains omit the clause entirely.
func timeColumnOf(keyCols []string) string {
	timeHints := []string{"date", "day", "month", "year", "week", "ts", "time", "period"}
	for _, col := range keyCols {
		lower := strings.ToLower(col)
		for _, hint := range timeHints {
			if strings.Contains(lower, hint) {
				return col
			}
		}
	}
	return ""
}

// firstSQLTableAlias returns the first `AS <alias>` from a FROM/JOIN clause.
func firstSQLTableAlias(fromSQL string) string {
	upper := strings.ToUpper(fromSQL)
	idx := strings.Index(upper, " AS ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(fromSQL[idx+4:])
	end := len(rest)
	for i, r := range rest {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		end = i
		break
	}
	return sanitizeIdentifier(rest[:end])
}

// rewriteMetricTableAlias remaps MetricCompiler's hardcoded t0.<col> onto a
// federation driving alias (e.g. pos.<col>) so measures resolve in JOIN FROM.
func rewriteMetricTableAlias(expr, fromAlias, toAlias string) string {
	fromAlias = strings.TrimSpace(fromAlias)
	toAlias = strings.TrimSpace(toAlias)
	if fromAlias == "" || toAlias == "" || fromAlias == toAlias {
		return expr
	}
	return strings.ReplaceAll(expr, fromAlias+".", toAlias+".")
}
