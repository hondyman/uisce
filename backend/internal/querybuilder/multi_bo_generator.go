package querybuilder

import (
	"fmt"
	"regexp"
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
// server-resolved join paths, and reports each selected column's
// cardinality relative to the primary BO so callers can decide whether to
// flatten (one) or nest/aggregate (many) it in the UI.
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
	boDefByID := map[string]*boresolver.BODefinition{qd.Context.BOID: primary}

	var joinClauses []multiBOJoinClause

	for _, rel := range related {
		boDefByID[rel.BOID] = rel.BODef
		boCardinality[rel.BOID] = rel.Cardinality

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

	var selectClauses []string
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
		expr := fmt.Sprintf("%s.%s", alias, col)
		if aggWrap != "" {
			expr = fmt.Sprintf("%s(%s)", aggWrap, expr)
		}
		outLabel := label
		if outLabel == "" {
			outLabel = field.Name
		}
		selectClauses = append(selectClauses, fmt.Sprintf("%s AS %q", expr, outLabel))
		columns = append(columns, boresolver.QueryResultColumn{
			Name:        outLabel,
			Type:        field.Type,
			BOID:        boID,
			Cardinality: cardinalityOrDefault(boCardinality, boID),
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
	if len(selectClauses) == 0 {
		return "", nil, nil, fmt.Errorf("query must select at least one dimension or measure")
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

	args := append([]interface{}{}, genCtx.Args...)
	paramN := genCtx.ParamCounter
	nextParam := func(v interface{}) string {
		paramN++
		args = append(args, v)
		return fmt.Sprintf("$%d", paramN)
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
	if qd.Query.Limit > 0 {
		fmt.Fprintf(&sb, "\nLIMIT %d", qd.Query.Limit)
	}

	return sb.String(), args, columns, nil
}

func cardinalityOrDefault(m map[string]string, boID string) string {
	if boID == "" {
		return ""
	}
	return m[boID]
}
