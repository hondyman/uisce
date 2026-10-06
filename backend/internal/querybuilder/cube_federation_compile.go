package querybuilder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// ErrCubeFederationCompile is returned when federation join SQL cannot be
// compiled from semantic term bindings (missing source, unbound term, etc.).
var ErrCubeFederationCompile = errors.New("cube federation join compile failed")

// FederationBindingResolver resolves a federation source's driving table and
// each join term ID to a physical column under that source's BO + binding hint.
//
// This is the semantic-layer spine: the cube stores term IDs only; physical
// columns come from bindings (same term on two BOs ⇒ joinable across DBs).
type FederationBindingResolver interface {
	ResolveDrivingTable(ctx context.Context, tenantID, boID, bindingHint string) (string, error)
	ResolveTermColumn(ctx context.Context, tenantID, boID, bindingHint, termID string) (string, error)
}

// MapFederationBindingResolver is a fixture/resolver for unit tests.
// Keys: boID (lower) → driving table; "boID|termID" (lower) → column.
type MapFederationBindingResolver struct {
	Tables  map[string]string // boId -> driving table
	Columns map[string]string // "boId|termId" -> physical column (bare name)
}

// ResolveDrivingTable implements FederationBindingResolver.
func (m MapFederationBindingResolver) ResolveDrivingTable(_ context.Context, _, boID, _ string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(boID))
	table, ok := m.Tables[key]
	if !ok || strings.TrimSpace(table) == "" {
		return "", fmt.Errorf("%w: no driving table for boId %q", ErrCubeFederationCompile, boID)
	}
	return table, nil
}

// ResolveTermColumn implements FederationBindingResolver.
func (m MapFederationBindingResolver) ResolveTermColumn(_ context.Context, _, boID, _, termID string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(boID)) + "|" + strings.ToLower(strings.TrimSpace(termID))
	col, ok := m.Columns[key]
	if !ok || strings.TrimSpace(col) == "" {
		return "", fmt.Errorf("%w: no binding for term %q on boId %q", ErrCubeFederationCompile, termID, boID)
	}
	return col, nil
}

// FederationSourcePlan is one resolved source in a compiled join.
type FederationSourcePlan struct {
	Alias        string `json:"alias"`
	BOID         string `json:"boId"`
	BindingHint  string `json:"bindingHint,omitempty"`
	DrivingTable string `json:"drivingTable"`
}

// FederationJoinPlan is one compiled join edge with ON predicates.
type FederationJoinPlan struct {
	LeftAlias    string   `json:"leftAlias"`
	RightAlias   string   `json:"rightAlias"`
	KeyKind      string   `json:"keyKind"`
	OnPredicates []string `json:"onPredicates"`
}

// CompiledFederationJoinSQL is the StarRocks FROM-clause join plan for a
// federated cube grain MV (CUBE-2.2).
type CompiledFederationJoinSQL struct {
	FromSQL string                 `json:"fromSql"`
	Sources []FederationSourcePlan `json:"sources"`
	Joins   []FederationJoinPlan   `json:"joins"`
	// TermExprs maps "alias|termId" → "alias.column" for grain/measure qualify.
	TermExprs map[string]string `json:"termExprs"`
}

// DimExpr returns a qualified SQL expression for a grain/dimension term,
// preferring the source whose BOID matches primaryBOID, then any source that
// bound the term. Falls back to the sanitized bare identifier.
func (c *CompiledFederationJoinSQL) DimExpr(termID, primaryBOID string) string {
	if c == nil {
		return sanitizeIdentifier(termID)
	}
	termKey := strings.ToLower(strings.TrimSpace(termID))
	primary := strings.ToLower(strings.TrimSpace(primaryBOID))
	// Prefer primary BO's alias.
	for _, src := range c.Sources {
		if primary != "" && strings.ToLower(src.BOID) != primary {
			continue
		}
		if expr, ok := c.TermExprs[src.Alias+"|"+termKey]; ok {
			return expr
		}
	}
	for aliasTerm, expr := range c.TermExprs {
		parts := strings.SplitN(aliasTerm, "|", 2)
		if len(parts) == 2 && parts[1] == termKey {
			return expr
		}
	}
	return sanitizeIdentifier(termID)
}

// DimExprsForGrain builds grain dim → qualified expr for DDL SELECT/GROUP BY.
func (c *CompiledFederationJoinSQL) DimExprsForGrain(grain []string, primaryBOID string) map[string]string {
	out := make(map[string]string, len(grain))
	for _, d := range grain {
		col := sanitizeIdentifier(d)
		if col == "" {
			continue
		}
		out[col] = c.DimExpr(d, primaryBOID)
	}
	return out
}

// EnrichGrainTerms resolves grain/dimension term IDs through the primary BO
// binding so SELECT/GROUP BY can qualify columns that were not join keys.
func (c *CompiledFederationJoinSQL) EnrichGrainTerms(
	ctx context.Context,
	tenantID string,
	primaryBOID string,
	grain []string,
	resolver FederationBindingResolver,
) error {
	if c == nil || resolver == nil {
		return nil
	}
	primary := strings.ToLower(strings.TrimSpace(primaryBOID))
	var src *FederationSourcePlan
	for i := range c.Sources {
		if primary != "" && strings.ToLower(c.Sources[i].BOID) == primary {
			src = &c.Sources[i]
			break
		}
	}
	if src == nil && len(c.Sources) > 0 {
		src = &c.Sources[0]
	}
	if src == nil {
		return nil
	}
	if c.TermExprs == nil {
		c.TermExprs = make(map[string]string)
	}
	for _, d := range grain {
		termKey := strings.ToLower(strings.TrimSpace(d))
		if termKey == "" {
			continue
		}
		mapKey := src.Alias + "|" + termKey
		if _, ok := c.TermExprs[mapKey]; ok {
			continue
		}
		col, err := resolver.ResolveTermColumn(ctx, tenantID, src.BOID, src.BindingHint, d)
		if err != nil {
			return err
		}
		col = barePhysicalColumn(col)
		if err := assertSafeSQLIdent(col); err != nil {
			return fmt.Errorf("%w: grain term %q: %v", ErrCubeFederationCompile, d, err)
		}
		c.TermExprs[mapKey] = src.Alias + "." + col
	}
	return nil
}

// CompileFederationJoinSQL resolves federation sources/joins through the
// binding resolver and emits a StarRocks INNER JOIN FROM clause.
//
// Common joins: ON left.alias.col = right.alias.col for each parallel term pair.
// Transform joins: resolve transformTermId on each side and equate those columns
// (deterministic term/calc — never a metric aggregate; gated earlier).
func CompileFederationJoinSQL(
	ctx context.Context,
	tenantID string,
	f CubeFederation,
	resolver FederationBindingResolver,
) (*CompiledFederationJoinSQL, error) {
	if err := ValidateCubeFederation(f); err != nil {
		return nil, err
	}
	if f.Empty() {
		return nil, fmt.Errorf("%w: federation is empty", ErrCubeFederationCompile)
	}
	if resolver == nil {
		return nil, fmt.Errorf("%w: binding resolver is required", ErrCubeFederationCompile)
	}

	byAlias := make(map[string]CubeFederationSource, len(f.Sources))
	sources := make([]FederationSourcePlan, 0, len(f.Sources))
	termExprs := make(map[string]string)

	for _, s := range f.Sources {
		alias := strings.ToLower(strings.TrimSpace(s.Alias))
		boID := strings.TrimSpace(s.BOID)
		hint := strings.ToLower(strings.TrimSpace(s.BindingHint))
		table, err := resolver.ResolveDrivingTable(ctx, tenantID, boID, hint)
		if err != nil {
			return nil, err
		}
		table = strings.TrimSpace(table)
		if err := assertSafeSQLIdent(table); err != nil {
			return nil, fmt.Errorf("%w: driving table for %s: %v", ErrCubeFederationCompile, alias, err)
		}
		if err := assertSafeSQLIdent(alias); err != nil {
			return nil, fmt.Errorf("%w: alias: %v", ErrCubeFederationCompile, err)
		}
		plan := FederationSourcePlan{
			Alias:        alias,
			BOID:         boID,
			BindingHint:  hint,
			DrivingTable: table,
		}
		sources = append(sources, plan)
		byAlias[alias] = s
		_ = hint
	}
	if len(sources) < 2 {
		return nil, fmt.Errorf("%w: need at least two sources", ErrCubeFederationCompile)
	}

	// Stable source order for FROM: first join's left, then append rights in join order.
	joined := make(map[string]bool)
	var fromParts []string
	joinPlans := make([]FederationJoinPlan, 0, len(f.Joins))

	resolveSide := func(alias, termID string) (string, error) {
		src, ok := byAlias[alias]
		if !ok {
			return "", fmt.Errorf("%w: unknown alias %q", ErrCubeFederationCompile, alias)
		}
		col, err := resolver.ResolveTermColumn(ctx, tenantID, src.BOID, src.BindingHint, termID)
		if err != nil {
			return "", err
		}
		col = barePhysicalColumn(col)
		if err := assertSafeSQLIdent(col); err != nil {
			return "", fmt.Errorf("%w: column for term %q on %s: %v", ErrCubeFederationCompile, termID, alias, err)
		}
		expr := alias + "." + col
		termExprs[alias+"|"+strings.ToLower(strings.TrimSpace(termID))] = expr
		return expr, nil
	}

	for i, jn := range f.Joins {
		left := strings.ToLower(strings.TrimSpace(jn.LeftAlias))
		right := strings.ToLower(strings.TrimSpace(jn.RightAlias))
		kind := strings.ToLower(strings.TrimSpace(jn.KeyKind))
		if kind == "" {
			kind = "common"
		}

		var preds []string
		switch kind {
		case "common":
			if len(jn.LeftTermIDs) == 0 || len(jn.LeftTermIDs) != len(jn.RightTermIDs) {
				return nil, fmt.Errorf("%w: joins[%d] common key needs parallel leftTermIds/rightTermIds", ErrCubeFederationCompile, i)
			}
			for k := range jn.LeftTermIDs {
				lExpr, err := resolveSide(left, jn.LeftTermIDs[k])
				if err != nil {
					return nil, err
				}
				rExpr, err := resolveSide(right, jn.RightTermIDs[k])
				if err != nil {
					return nil, err
				}
				preds = append(preds, fmt.Sprintf("%s = %s", lExpr, rExpr))
			}
		case "transform":
			tid := strings.TrimSpace(jn.TransformTermID)
			if tid == "" {
				return nil, fmt.Errorf("%w: joins[%d] transform missing transformTermId", ErrCubeFederationCompile, i)
			}
			lExpr, err := resolveSide(left, tid)
			if err != nil {
				return nil, err
			}
			rExpr, err := resolveSide(right, tid)
			if err != nil {
				return nil, err
			}
			preds = append(preds, fmt.Sprintf("%s = %s", lExpr, rExpr))
		default:
			return nil, fmt.Errorf("%w: joins[%d] unknown keyKind %q", ErrCubeFederationCompile, i, kind)
		}

		joinPlans = append(joinPlans, FederationJoinPlan{
			LeftAlias:    left,
			RightAlias:   right,
			KeyKind:      kind,
			OnPredicates: preds,
		})

		leftSrc := sourceByAlias(sources, left)
		rightSrc := sourceByAlias(sources, right)
		if leftSrc == nil || rightSrc == nil {
			return nil, fmt.Errorf("%w: joins[%d] missing source plan", ErrCubeFederationCompile, i)
		}

		if len(fromParts) == 0 {
			fromParts = append(fromParts, fmt.Sprintf("%s AS %s", leftSrc.DrivingTable, left))
			joined[left] = true
		}
		if !joined[left] {
			// Left not yet in FROM — prepend via joining onto existing tree is
			// unsupported in this slice; require join order to grow from first left.
			return nil, fmt.Errorf("%w: joins[%d] leftAlias %q is not yet in FROM; order joins from the primary source outward", ErrCubeFederationCompile, i, left)
		}
		if joined[right] {
			return nil, fmt.Errorf("%w: joins[%d] rightAlias %q already joined", ErrCubeFederationCompile, i, right)
		}
		on := strings.Join(preds, " AND ")
		fromParts = append(fromParts, fmt.Sprintf("INNER JOIN %s AS %s ON %s", rightSrc.DrivingTable, right, on))
		joined[right] = true
	}

	if len(fromParts) == 0 {
		return nil, fmt.Errorf("%w: no joins to compile", ErrCubeFederationCompile)
	}
	// Ensure every declared source appears (disconnected sources are invalid).
	for _, src := range sources {
		if !joined[src.Alias] {
			return nil, fmt.Errorf("%w: source %q is not reachable from joins", ErrCubeFederationCompile, src.Alias)
		}
	}

	return &CompiledFederationJoinSQL{
		FromSQL:   strings.Join(fromParts, "\n"),
		Sources:   sources,
		Joins:     joinPlans,
		TermExprs: termExprs,
	}, nil
}

func sourceByAlias(sources []FederationSourcePlan, alias string) *FederationSourcePlan {
	for i := range sources {
		if sources[i].Alias == alias {
			return &sources[i]
		}
	}
	return nil
}

func barePhysicalColumn(col string) string {
	col = strings.TrimSpace(col)
	if idx := strings.LastIndex(col, "."); idx >= 0 {
		col = col[idx+1:]
	}
	return col
}

func assertSafeSQLIdent(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("empty identifier")
	}
	parts := strings.Split(name, ".")
	for _, p := range parts {
		if p == "" || !identRe.MatchString(p) {
			return fmt.Errorf("unsafe identifier %q", name)
		}
	}
	return nil
}

// dbFederationBindingResolver resolves driving tables and term→column via
// business_objects + field_bindings / catalog (same spine as GetBOTerms).
type dbFederationBindingResolver struct {
	db *sqlx.DB
}

func (r dbFederationBindingResolver) ResolveDrivingTable(ctx context.Context, tenantID, boID, bindingHint string) (string, error) {
	boID = strings.TrimSpace(boID)
	if boID == "" {
		return "", fmt.Errorf("%w: boId required", ErrCubeFederationCompile)
	}
	hint := strings.ToLower(strings.TrimSpace(bindingHint))

	// Prefer an authored business_object_binding whose name/hint matches.
	if hint != "" {
		var table string
		err := r.db.GetContext(ctx, &table, `
			SELECT COALESCE(NULLIF(TRIM(bo.driver_table_name), ''), NULLIF(TRIM(bo.bo_key), ''), '')
			FROM public.business_objects bo
			JOIN public.business_object_binding b ON b.bo_id = bo.id
			WHERE (bo.tenant_id = $1::uuid OR bo.tenant_id IS NULL)
			  AND (bo.bo_key = $2 OR bo.id::text = $2)
			  AND (
			    lower(COALESCE(b.binding_name, '')) = $3
			    OR lower(COALESCE(b.bo_binding_id::text, '')) = $3
			  )
			ORDER BY b.is_default DESC
			LIMIT 1
		`, tenantID, boID, hint)
		if err == nil && strings.TrimSpace(table) != "" {
			return QualifyCubeSourceTable(table), nil
		}
	}

	var table string
	err := r.db.GetContext(ctx, &table, `
		SELECT COALESCE(NULLIF(TRIM(driver_table_name), ''), NULLIF(TRIM(bo_key), ''), '')
		FROM public.business_objects
		WHERE tenant_id = $1::uuid
		  AND (bo_key = $2 OR id::text = $2)
		LIMIT 1
	`, tenantID, boID)
	if err == sql.ErrNoRows {
		err = r.db.GetContext(ctx, &table, `
			SELECT COALESCE(NULLIF(TRIM(driver_table_name), ''), NULLIF(TRIM(bo_key), ''), '')
			FROM public.business_objects
			WHERE bo_key = $1 OR id::text = $1
			LIMIT 1
		`, boID)
	}
	if err != nil {
		return "", fmt.Errorf("%w: resolve driving table for %q: %v", ErrCubeFederationCompile, boID, err)
	}
	if strings.TrimSpace(table) == "" {
		return "", fmt.Errorf("%w: BO %q has no driver_table_name", ErrCubeFederationCompile, boID)
	}
	return QualifyCubeSourceTable(table), nil
}

func (r dbFederationBindingResolver) ResolveTermColumn(ctx context.Context, tenantID, boID, bindingHint, termID string) (string, error) {
	boID = strings.TrimSpace(boID)
	termID = strings.TrimSpace(termID)
	if boID == "" || termID == "" {
		return "", fmt.Errorf("%w: boId and termId required", ErrCubeFederationCompile)
	}
	hint := strings.ToLower(strings.TrimSpace(bindingHint))

	var bindingIDParam *string
	if hint != "" {
		var bid string
		_ = r.db.GetContext(ctx, &bid, `
			SELECT b.bo_binding_id::text
			FROM public.business_object_binding b
			JOIN public.business_objects bo ON bo.id = b.bo_id
			WHERE (bo.bo_key = $1 OR bo.id::text = $1)
			  AND (
			    lower(COALESCE(b.binding_name, '')) = $2
			    OR lower(b.bo_binding_id::text) = $2
			  )
			ORDER BY b.is_default DESC
			LIMIT 1
		`, boID, hint)
		if bid != "" {
			bindingIDParam = &bid
		}
	}

	var col string
	err := r.db.GetContext(ctx, &col, `
		SELECT cn.node_name
		FROM public.business_object_fields f
		JOIN public.business_objects bo
		  ON bo.id = f.bo_id
		 AND (bo.bo_key = $1 OR bo.id::text = $1)
		JOIN public.field_bindings fb
		  ON fb.field_id = f.id
		 AND fb.bo_id = f.bo_id
		 AND (fb.binding_id = $3::uuid OR $3::uuid IS NULL)
		JOIN catalog_node cn ON cn.id = fb.source_node_id
		WHERE (f.term_node_id::text = $2 OR lower(f.field_name) = lower($2) OR lower(f.technical_name) = lower($2))
		  AND fb.binding_status = 'RESOLVED'
		  AND fb.source_type = 'COLUMN'
		LIMIT 1
	`, boID, termID, bindingIDParam)
	if err == nil && strings.TrimSpace(col) != "" {
		return col, nil
	}

	// Fallback: treat term id / key as the physical column name when no
	// explicit field_bindings row exists (common for STI driver tables).
	fallback := sanitizeIdentifier(termID)
	if fallback == "" {
		return "", fmt.Errorf("%w: unbound term %q on boId %q", ErrCubeFederationCompile, termID, boID)
	}
	_ = tenantID
	return fallback, nil
}
