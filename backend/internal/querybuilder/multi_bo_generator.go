package querybuilder

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// This file generates SQL for QueryDefs that span more than one Business
// Object (QueryContext.RelatedBOIDs). It is deliberately separate from
// boresolver.BOSQLGenerator.GenerateSQLFromSemantic, which only ever
// resolves fields against a single BODefinition and has no join concept —
// extending it in place would have meant threading multi-table alias
// resolution through code that many single-table callers already depend on.
// This generator instead builds directly on the analytics package's
// server-resolved join paths (never client-supplied join SQL) and validates
// every identifier before interpolating it, since identifiers can't be bind
// parameters.

// identRe allows only plain SQL identifier characters. Every table, column,
// and alias name passed to this generator must match it before being
// concatenated into SQL text.
var identRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteIdent(name string) (string, error) {
	// Table names may be schema-qualified ("public.customers").
	parts := strings.Split(name, ".")
	for _, p := range parts {
		if !identRe.MatchString(p) {
			return "", fmt.Errorf("unsafe identifier: %q", name)
		}
	}
	return name, nil
}

// joinedBO carries the resolved join for one related BO relative to the
// primary BO in a multi-BO QueryDef.
type joinedBO struct {
	BOID        string
	BODef       *boresolver.BODefinition
	Path        *analytics.JoinPath
	Cardinality string // "one" | "many", from Path.TraversalCardinality()
}

// bareColumn strips any "table." qualifier a field's physical column may
// already carry, so it can be re-qualified with this query's own alias.
func bareColumn(f *boresolver.BOField) string {
	col := f.PhysicalColumn
	if col == "" {
		col = f.Name
	}
	if idx := strings.LastIndex(col, "."); idx >= 0 {
		col = col[idx+1:]
	}
	return col
}

// aliasAllocator assigns a stable, unique table alias per physical table
// name encountered while stitching together join paths for potentially
// several related BOs, so two paths that happen to pass through the same
// table (a shared lookup table, say) reuse one join instead of joining it
// twice.
type aliasAllocator struct {
	next    int
	byTable map[string]string
}

func newAliasAllocator() *aliasAllocator {
	return &aliasAllocator{byTable: make(map[string]string)}
}

func (a *aliasAllocator) allocFor(table string) (alias string, isNew bool) {
	if al, ok := a.byTable[table]; ok {
		return al, false
	}
	al := fmt.Sprintf("t%d", a.next)
	a.next++
	a.byTable[table] = al
	return al, true
}

type multiBOJoinClause struct {
	joinType    string
	table       string
	alias       string
	leftAlias   string
	leftColumn  string
	rightColumn string
}

// buildMultiBOSQL assembles a tenant-scoped, parameterized SELECT that joins
// the primary BO's driving table to one or more related BOs via
// server-resolved join paths. Each selected column's cardinality relative
// to the primary BO is reported on the result (QueryResultColumn.Cardinality)
// and, when a query mixes "one"-side and "many"-side selections, is used
// here to GROUP BY the one-side columns and auto-aggregate the many-side
// ones - see the fan-out guard below - rather than leaving that decision
// to callers, which previously left the flat LEFT JOIN free to fan out
// rows whenever a "many" relationship was queried without hand-written
// aggregation.
func buildMultiBOSQL(
	generator *boresolver.BOSQLGenerator,
	primary *boresolver.BODefinition,
	related []joinedBO,
	qd *boresolver.QueryDef,
	tenantID string,
) (string, []interface{}, []boresolver.QueryResultColumn, error) {
	if primary == nil {
		return "", nil, nil, fmt.Errorf("primary BO definition is nil")
	}

	baseTable, err := quoteIdent(primary.DrivingTable)
	if err != nil {
		return "", nil, nil, err
	}

	aliases := newAliasAllocator()
	baseAlias, _ := aliases.allocFor(primary.DrivingTable)

	boAlias := map[string]string{qd.Context.BOID: baseAlias}
	boCardinality := map[string]string{qd.Context.BOID: ""}
	// boOwnership mirrors boCardinality but reads RootOwnership instead
	// of TraversalCardinality - see analytics.JoinPath.RootOwnership.
	// The primary BO's own rows are trivially their own unique owner.
	boOwnership := map[string]string{qd.Context.BOID: "unique"}
	boDefByID := map[string]*boresolver.BODefinition{qd.Context.BOID: primary}

	var joinClauses []multiBOJoinClause

	for _, rel := range related {
		boDefByID[rel.BOID] = rel.BODef
		boCardinality[rel.BOID] = rel.Cardinality
		boOwnership[rel.BOID] = rel.Path.RootOwnership()

		if rel.Path == nil || len(rel.Path.Steps) == 0 {
			// Same driving table as primary (join path resolver returns an
			// empty path when from==to); nothing further to join.
			boAlias[rel.BOID] = baseAlias
			continue
		}

		for _, step := range rel.Path.Steps {
			leftTable, err := quoteIdent(step.LeftTable)
			if err != nil {
				return "", nil, nil, err
			}
			rightTable, err := quoteIdent(step.RightTable)
			if err != nil {
				return "", nil, nil, err
			}
			leftCol, err := quoteIdent(step.LeftColumn)
			if err != nil {
				return "", nil, nil, err
			}
			rightCol, err := quoteIdent(step.RightColumn)
			if err != nil {
				return "", nil, nil, err
			}

			leftAlias, ok := aliases.byTable[leftTable]
			if !ok {
				// The first hop's left side must be the base table; anything
				// else means the resolved path didn't start where we expect.
				return "", nil, nil, fmt.Errorf("join path for BO %s starts at unexpected table %q", rel.BOID, step.LeftTable)
			}

			rightAlias, isNew := aliases.allocFor(rightTable)
			if isNew {
				joinType := strings.ToUpper(step.JoinType)
				if joinType == "" {
					joinType = "LEFT"
				}
				joinClauses = append(joinClauses, multiBOJoinClause{
					joinType:    joinType,
					table:       rightTable,
					alias:       rightAlias,
					leftAlias:   leftAlias,
					leftColumn:  leftCol,
					rightColumn: rightCol,
				})
			}
		}

		targetTable, err := quoteIdent(rel.BODef.DrivingTable)
		if err != nil {
			return "", nil, nil, err
		}
		finalAlias, ok := aliases.byTable[targetTable]
		if !ok {
			return "", nil, nil, fmt.Errorf("join path for BO %s did not reach its driving table %q", rel.BOID, rel.BODef.DrivingTable)
		}
		boAlias[rel.BOID] = finalAlias
	}

	resolveField := func(boID, termNodeID string) (*boresolver.BOField, string, error) {
		if boID == "" {
			boID = qd.Context.BOID
		}
		def, ok := boDefByID[boID]
		if !ok {
			return nil, "", fmt.Errorf("boId %q is not the primary BO or one of relatedBoIds", boID)
		}
		field, err := resolveTermToField(def, termNodeID)
		if err != nil {
			return nil, "", err
		}
		return field, boAlias[boID], nil
	}

	// selectedColumn carries one selected dimension/measure through to a
	// second pass, deferred until every column is known, because whether
	// a "many"-side column needs auto-aggregation depends on whether the
	// query *also* selects any "one"-side column (see comment above
	// groupByOneSideExprs below) - that can't be decided per-column as
	// they're added.
	type selectedColumn struct {
		expr        string // "alias.column", unwrapped
		outLabel    string
		fieldType   string
		boID        string
		cardinality string
		hasAgg      bool
	}
	var selected []selectedColumn
	var columns []boresolver.QueryResultColumn

	addSelect := func(boID, termNodeID, label, aggWrap string) error {
		field, alias, err := resolveField(boID, termNodeID)
		if err != nil {
			return err
		}
		col := bareColumn(field)
		if !identRe.MatchString(col) {
			return fmt.Errorf("unsafe column identifier: %q", col)
		}
		outLabel := label
		if outLabel == "" {
			outLabel = field.Name
		}
		cardinality := cardinalityOrDefault(boCardinality, boID)
		ownership := ownershipOrDefault(boOwnership, boID)
		expr := fmt.Sprintf("%s.%s", alias, col)
		if aggWrap != "" {
			expr = fmt.Sprintf("%s(%s)", aggWrap, expr)
		}
		selected = append(selected, selectedColumn{
			expr: expr, outLabel: outLabel, fieldType: field.Type,
			boID: boID, cardinality: cardinality, hasAgg: aggWrap != "",
		})
		columns = append(columns, boresolver.QueryResultColumn{
			Name: outLabel, Type: field.Type, BOID: boID, Cardinality: cardinality,
			RootOwnership: ownership,
		})
		return nil
	}

	for _, dim := range qd.Query.Dimensions {
		if err := addSelect(dim.BOID, dim.TermNodeID, dim.Alias, ""); err != nil {
			return "", nil, nil, err
		}
	}
	for _, m := range qd.Query.Measures {
		agg := strings.ToUpper(m.Aggregation)
		if agg == "" || agg == "NONE" {
			agg = ""
		}
		if err := addSelect(m.BOID, m.TermNodeID, m.Alias, agg); err != nil {
			return "", nil, nil, err
		}
	}
	if len(selected) == 0 {
		return "", nil, nil, fmt.Errorf("query must select at least one dimension or measure")
	}

	// Fan-out guard: joining to a "many"-cardinality related BO (one order
	// has many allocations, say) and then selecting a bare column from
	// BOTH the "one" side and the "many" side repeats every one-side value
	// once per many-side row - not a listing, just duplicated data. A
	// query that selects ONLY many-side columns (e.g. "list this order's
	// allocations") is a legitimate one-row-per-child listing and is left
	// alone; the fan-out only exists once a one-side grain is mixed in.
	// When that happens, GROUP BY the one-side columns and aggregate every
	// unaggregated many-side column instead of leaving it as a bare
	// column repeated across the group (SUM for numeric types, since
	// that's almost always what "total allocated quantity per order"
	// means; STRING_AGG of distinct values otherwise, since concatenating
	// unrelated numbers is never useful but concatenating repeated text/
	// id values into a de-duplicated list is a normal way to surface a
	// many-side attribute at the one-side grain).
	hasOneSide := false
	hasManySide := false
	manyBOIDSet := map[string]bool{}
	for _, c := range selected {
		if c.cardinality == "many" {
			hasManySide = true
			manyBOIDSet[c.boID] = true
		} else {
			hasOneSide = true
		}
	}
	needsAggregation := hasOneSide && hasManySide

	// Containment for the case this generator cannot yet handle correctly:
	// selecting from two or more INDEPENDENTLY "many"-cardinality related
	// BOs at once. The GROUP BY + auto-aggregate approach below is only
	// correct when there is at most one expanding branch - it assumes the
	// flat JOIN produces one row per (root, many-side-child), so summing
	// or counting within a GROUP BY on the root key recovers the right
	// total. With two expanding branches joined in the same flat query,
	// that assumption breaks: order O with 3 line_items and 2 shipments
	// joined together produces 3*2=6 rows for O before any aggregation
	// runs, so SUM(line_items.amount) counts every line item once per
	// shipment (double- or triple-counted) instead of once. There is no
	// single flat SELECT that computes both branches correctly; emitting
	// one anyway would be a wrong number with nothing to show it's wrong.
	// See ErrUnsupportedFanOut.
	if len(manyBOIDSet) >= 2 {
		names := make([]string, 0, len(manyBOIDSet))
		for id := range manyBOIDSet {
			names = append(names, id)
		}
		sort.Strings(names)
		return "", nil, nil, &ErrUnsupportedFanOut{BOIDs: names}
	}

	var selectClauses []string
	var groupByExprs []string
	for _, c := range selected {
		expr := c.expr
		if needsAggregation && c.cardinality == "many" && !c.hasAgg {
			if isNumericFieldType(c.fieldType) {
				expr = fmt.Sprintf("SUM(%s)", expr)
			} else {
				expr = fmt.Sprintf("STRING_AGG(DISTINCT %s::text, ', ')", expr)
			}
		} else if needsAggregation && c.cardinality != "many" && !c.hasAgg {
			groupByExprs = append(groupByExprs, expr)
		}
		selectClauses = append(selectClauses, fmt.Sprintf("%s AS %q", expr, c.outLabel))
	}

	// Filter predicates are compiled through the existing, hardened
	// CompileFilterPredicate (operator normalization incl. "eq"/"neq"/...,
	// LIKE/ILIKE wrapping for contains/starts_with/ends_with, IN-list and
	// NULL handling, BETWEEN) rather than reimplementing that vocabulary
	// here — it already never interpolates values directly into SQL text.
	genCtx := &boresolver.GenerationContext{}
	var whereClauses []string
	for _, f := range qd.Query.Filters {
		field, alias, err := resolveField(f.BOID, f.TermNodeID)
		if err != nil {
			return "", nil, nil, err
		}
		col := bareColumn(field)
		if !identRe.MatchString(col) {
			return "", nil, nil, fmt.Errorf("unsafe column identifier: %q", col)
		}
		sqlExpr := fmt.Sprintf("%s.%s", alias, col)
		predicate, err := boresolver.CompileFilterPredicate(generator, genCtx, sqlExpr, boresolver.FilterClause{
			Operator: f.Operator,
			Value:    f.Value,
		})
		if err != nil {
			return "", nil, nil, fmt.Errorf("failed to compile filter on %s: %w", f.TermNodeID, err)
		}
		whereClauses = append(whereClauses, predicate)
	}

	// nextParam records a value into genCtx.Args (the SAME pending slice
	// CompileFilterPredicate above appended into) and returns a sentinel,
	// never a rendered token - see boresolver/params.go. CompileFilterPredicate
	// already emits sentinels internally (via its own nextParam), so this
	// generator's own params must use the identical mechanism: rendering a
	// literal "$N" here, as before, would (a) hardcode Postgres regardless
	// of the target dialect and (b) not match the textual position its
	// value ends up at once tenant scoping is spliced in front of the
	// filter clause below, which silently swaps args under any
	// positional ("?") dialect.
	nextParam := func(v interface{}) string {
		idx := len(genCtx.Args)
		genCtx.Args = append(genCtx.Args, v)
		return boresolver.ParamSentinel(boresolver.EnsureParamNonce(genCtx), idx)
	}

	// Tenant scoping on every joined table, mirroring
	// BOSQLGenerator.InjectTenantScopingToGraph's model: every physical
	// table this query touches carries its own tenant_id predicate rather
	// than trusting a single top-level check.
	if tenantID != "" {
		whereClauses = append([]string{fmt.Sprintf("%s.tenant_id = %s", baseAlias, nextParam(tenantID))}, whereClauses...)
		for _, jc := range joinClauses {
			whereClauses = append(whereClauses, fmt.Sprintf("%s.tenant_id = %s", jc.alias, nextParam(tenantID)))
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "SELECT\n  %s\nFROM %s AS %s", strings.Join(selectClauses, ",\n  "), baseTable, baseAlias)
	for _, jc := range joinClauses {
		fmt.Fprintf(&sb, "\n%s JOIN %s AS %s ON %s.%s = %s.%s",
			jc.joinType, jc.table, jc.alias, jc.leftAlias, jc.leftColumn, jc.alias, jc.rightColumn)
	}
	if len(whereClauses) > 0 {
		fmt.Fprintf(&sb, "\nWHERE %s", strings.Join(whereClauses, " AND "))
	}
	if len(groupByExprs) > 0 {
		fmt.Fprintf(&sb, "\nGROUP BY %s", strings.Join(groupByExprs, ", "))
	}
	if qd.Query.Limit > 0 {
		fmt.Fprintf(&sb, "\nLIMIT %d", qd.Query.Limit)
	}

	// One final pass turns every sentinel (from CompileFilterPredicate and
	// nextParam above) into this dialect's real placeholder, numbered by
	// where it actually lands in the assembled text - see
	// boresolver/params.go for why creation order and textual order can
	// differ here (tenant scoping is spliced in front of the filter
	// clause it was compiled after).
	finalSQL, finalArgs, err := boresolver.RenumberParams(sb.String(), generator.Dialect, genCtx.Args, boresolver.EnsureParamNonce(genCtx))
	if err != nil {
		return "", nil, nil, err
	}
	return finalSQL, finalArgs, columns, nil
}

// ErrUnsupportedFanOut is returned when a multi-BO query selects measures
// or dimensions from two or more independently "many"-cardinality related
// BOs at once. See the containment check in buildMultiBOSQL for why: there
// is no single flat query that combines them without cross-multiplying
// rows first. Callers must issue one request per business object instead.
type ErrUnsupportedFanOut struct {
	BOIDs []string
}

func (e *ErrUnsupportedFanOut) Error() string {
	return fmt.Sprintf(
		"query selects measures/dimensions from multiple independently-expanding related business objects (%s) in one request - combining them would double-count; issue separate requests per business object instead",
		strings.Join(e.BOIDs, ", "),
	)
}

func cardinalityOrDefault(m map[string]string, boID string) string {
	if boID == "" {
		return ""
	}
	return m[boID]
}

// ownershipOrDefault mirrors cardinalityOrDefault, reading boOwnership
// instead of boCardinality. An empty boID (the primary BO's own column)
// returns "" - QueryResultColumn.RootOwnership documents empty as
// "unique" by convention, matching Cardinality's own empty-means-primary
// convention.
func ownershipOrDefault(m map[string]string, boID string) string {
	if boID == "" {
		return ""
	}
	return m[boID]
}

// isNumericFieldType reports whether a BOField.Type value denotes a
// summable numeric column, per the business_object_fields.data_type
// vocabulary ("number", "integer", "numeric", "decimal", "float", ...).
func isNumericFieldType(fieldType string) bool {
	switch strings.ToLower(fieldType) {
	case "number", "integer", "numeric", "decimal", "float", "double", "int", "bigint":
		return true
	default:
		return false
	}
}
