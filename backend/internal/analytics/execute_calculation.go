package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/models"
)

// ExecuteFormulaCalculation is THE single centralized run path for a
// formula-driven calculation (models.Calculation — not to be confused with
// the typed FinancialCalculation quant-model dispatcher below, a
// deliberately separate system — see ExecuteCalculation's doc comment).
// Every caller — handlers.CalculationHandler.Execute (HTTP,
// POST /calculations/{id}/execute) and pkg/workflows' ActivityCalculation
// (a Temporal workflow step) — calls this same method, so a calculation
// produces an identical result regardless of which system triggered it.
//
// It builds the calc's full dependency chain (calc-in-calc included, via
// BuildCalcGraph), compiles it via boresolver.CompileDeepCalculations, and
// dispatches by the tier that resolves to: pure SQL pushdown runs directly
// against the database; anything needing the host-runtime engine (e.g.
// xirr) runs through boresolver.HostRuntimeExecutor. tier is always
// returned ("pushdown" or "host_runtime") so callers can log/report which
// engine actually ran.
func (s *SemanticCalculationService) ExecuteFormulaCalculation(ctx context.Context, tenantID string, calc *models.Calculation) (results interface{}, tier string, err error) {
	if tenantID == "" {
		return nil, "", fmt.Errorf("tenantID is required")
	}
	if calc.DomainID == nil {
		return nil, "", fmt.Errorf("calculation has no domain_id (business object) to resolve base fields against")
	}

	rctx, err := NewResolutionContext(s.db, calc.DomainID.String())
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve business object: %w", err)
	}
	dialect := boresolver.PostgresDialect{}
	resolver := &boresolver.Resolver{
		BOID:         rctx.BOID,
		DrivingTable: rctx.DrivingTable,
		TermMappings: rctx.TermMappings,
		JoinPaths:    rctx.JoinPaths,
		Dialect:      dialect,
	}

	graph, err := s.BuildCalcGraph(calc)
	if err != nil {
		return nil, "", err
	}
	layers, err := graph.ResolveExecutionLayers()
	if err != nil {
		return nil, "", err
	}

	var baseFieldTerms []string
	for _, node := range graph.Nodes {
		if node.IsBaseField {
			baseFieldTerms = append(baseFieldTerms, node.TermKey)
		}
	}
	baseQuery, err := buildCalcBaseQuery(resolver, baseFieldTerms)
	if err != nil {
		return nil, "", err
	}

	gen := &boresolver.BOSQLGenerator{Dialect: dialect}
	sqlText, hostNodes, err := gen.CompileDeepCalculations(layers, baseQuery, []string{calc.Name})
	if err != nil {
		return nil, "", err
	}

	if len(hostNodes) == 0 {
		rows, execErr := s.executePushdownSQL(ctx, sqlText, tenantID)
		return rows, "pushdown", execErr
	}

	rows, execErr := s.executeHostRuntimeCalc(ctx, resolver, calc, hostNodes, tenantID)
	return rows, "host_runtime", execErr
}

// GetCalculationByID loads a calculation by its primary key.
func (s *SemanticCalculationService) GetCalculationByID(id string) (*models.Calculation, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	var calc models.Calculation
	if err := s.db.Get(&calc, "SELECT * FROM calculations WHERE id = $1", id); err != nil {
		return nil, err
	}
	return &calc, nil
}

func (s *SemanticCalculationService) executePushdownSQL(ctx context.Context, sqlText, tenantID string) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryxContext(ctx, sqlText, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		if err := rows.MapScan(row); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// executeHostRuntimeCalc runs the cut host-runtime nodes via
// boresolver.HostRuntimeExecutor, fetching rows with a tenant-scoped
// boresolver.SQLRowSource built from the same Resolver the pushdown path
// uses. entity_term/order_term (which field groups rows into entities and
// which orders each entity's series) come from calc.Arguments — a calc
// destined for host-runtime execution must declare them, since there's no
// way to infer "what does one row of this series mean" from the formula
// alone.
func (s *SemanticCalculationService) executeHostRuntimeCalc(ctx context.Context, resolver *boresolver.Resolver, calc *models.Calculation, hostNodes []*boresolver.CalcNode, tenantID string) ([]boresolver.HostRuntimeResult, error) {
	var execConfig struct {
		EntityTerm string `json:"entity_term"`
		OrderTerm  string `json:"order_term"`
	}
	if len(calc.Arguments) > 0 {
		_ = json.Unmarshal(calc.Arguments, &execConfig)
	}
	if execConfig.EntityTerm == "" || execConfig.OrderTerm == "" {
		return nil, fmt.Errorf("this calculation resolves to host-runtime tier (e.g. it uses xirr) and requires arguments.entity_term and arguments.order_term to be set — see boresolver.SQLRowSource")
	}

	source := &boresolver.SQLRowSource{
		DB:         s.db,
		Resolver:   resolver,
		EntityTerm: execConfig.EntityTerm,
		OrderTerm:  execConfig.OrderTerm,
	}
	executor := &boresolver.HostRuntimeExecutor{Rows: source}

	return executor.Execute(ctx, tenantID, hostNodes)
}

// buildCalcBaseQuery selects every base-field term needed anywhere in the
// calc chain, tenant-scoped. All base fields must resolve to the SAME
// physical table — cross-table base fields would need the join-injection
// machinery bo_sql_generator.go uses for Rule 7 tenant scoping
// (InjectTenantScopingToGraph), which this doesn't duplicate; it fails
// loudly rather than silently dropping a join.
func buildCalcBaseQuery(resolver *boresolver.Resolver, baseFieldTerms []string) (string, error) {
	if len(baseFieldTerms) == 0 {
		return "SELECT 1", nil
	}

	dialect := resolver.Dialect
	if dialect == nil {
		dialect = boresolver.PostgresDialect{}
	}

	var table string
	cols := make([]string, 0, len(baseFieldTerms))
	for _, term := range baseFieldTerms {
		mapping, _, err := resolver.ResolveTerm(term)
		if err != nil {
			return "", fmt.Errorf("failed to resolve base field %q: %w", term, err)
		}
		if table == "" {
			table = mapping.Table
		} else if table != mapping.Table {
			return "", fmt.Errorf("execute: base fields span multiple tables (%q and %q) — cross-table pushdown execution isn't wired yet", table, mapping.Table)
		}
		cols = append(cols, fmt.Sprintf("%s AS %s", dialect.QuoteIdent(mapping.Column), dialect.QuoteIdent(term)))
	}

	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s = $1",
		strings.Join(cols, ", "),
		dialect.QuoteIdent(table),
		dialect.QuoteIdent("tenant_id"),
	), nil
}
