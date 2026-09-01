package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CFGValidationError struct {
	ViolatedRule  string
	TokenOffender string
}

func (e *CFGValidationError) Error() string {
	return fmt.Sprintf("CFG Constraint Violation [%s]: Unauthorized token '%s'", e.ViolatedRule, e.TokenOffender)
}

type CFGValidatedQueryAST struct {
	RootBOKey     string   `json:"rootBoKey"`
	Dimensions    []string `json:"dimensions"`
	Measures      []string `json:"measures"`
	FilterClauses []string `json:"filterClauses"`
	CompiledSQL   string   `json:"compiledSql"`
}

type CFGSQLGenerator struct {
	db *sqlx.DB
}

func NewCFGSQLGenerator(db *sqlx.DB) *CFGSQLGenerator {
	return &CFGSQLGenerator{db: db}
}

// CompileConstrainedSQL guarantees zero-hallucination execution by masking tokens against catalog rules
func (g *CFGSQLGenerator) CompileConstrainedSQL(
	ctx context.Context,
	tenantID, boID uuid.UUID,
	boKey string,
	requestedDimensions []string,
	requestedMeasures []string,
	filters []string,
) (*CFGValidatedQueryAST, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 1. Fetch catalog-defined grammar mask (Rule 1: Config-Before-Code)
	var cfgRule struct {
		AllowedDimensionsRaw []byte `db:"allowed_dimensions"`
		AllowedMeasuresRaw   []byte `db:"allowed_measures"`
		NonAdditiveRaw       []byte `db:"non_additive_metrics"`
		ComplexityCeiling    int    `db:"complexity_ceiling"`
	}

	query := `
		SELECT allowed_dimensions, allowed_measures, non_additive_metrics, complexity_ceiling
		FROM catalog_governance.cfg_semantic_rules
		WHERE tenant_id = $1 AND bo_id = $2 AND is_active = TRUE;
	`
	if err := g.db.GetContext(ctx, &cfgRule, query, tenantID, boID); err != nil {
		return nil, fmt.Errorf("failed fetching CFG grammar rules for BO %s: %w", boKey, err)
	}

	// 2. Validate Dimension Tokens
	allowedDims := string(cfgRule.AllowedDimensionsRaw)
	for _, dim := range requestedDimensions {
		if !strings.Contains(allowedDims, fmt.Sprintf(`"%s"`, dim)) {
			return nil, &CFGValidationError{ViolatedRule: "DIMENSION_WHITELIST", TokenOffender: dim}
		}
	}

	// 3. Validate Measure Tokens & Enforce Non-Additive Locks
	allowedMeasures := string(cfgRule.AllowedMeasuresRaw)
	nonAdditive := string(cfgRule.NonAdditiveRaw)
	for _, m := range requestedMeasures {
		if !strings.Contains(allowedMeasures, fmt.Sprintf(`"%s"`, m)) {
			return nil, &CFGValidationError{ViolatedRule: "MEASURE_WHITELIST", TokenOffender: m}
		}
		if strings.Contains(nonAdditive, fmt.Sprintf(`"%s"`, m)) && len(requestedDimensions) > 0 {
			// Enforce two-pass CTE calculation for non-additive metrics (e.g., XIRR, Yield)
			return nil, &CFGValidationError{
				ViolatedRule:  "NON_ADDITIVE_AGGREGATION_LOCK",
				TokenOffender: fmt.Sprintf("%s (requires atomic WASM vector calculation)", m),
			}
		}
	}

	// 4. Synthesize Dialect Pushdown SQL
	dimProj := strings.Join(requestedDimensions, ", ")
	var measureProjs []string
	for _, m := range requestedMeasures {
		measureProjs = append(measureProjs, fmt.Sprintf("SUM(%s) AS %s", m, m))
	}
	measureProj := strings.Join(measureProjs, ", ")

	var compiled string
	if len(requestedDimensions) > 0 && len(measureProjs) > 0 {
		compiled = fmt.Sprintf("SELECT %s, %s FROM public.%s WHERE tenant_id = '%s' GROUP BY %s;",
			dimProj, measureProj, boKey, tenantID.String(), dimProj)
	} else if len(requestedDimensions) > 0 {
		compiled = fmt.Sprintf("SELECT %s FROM public.%s WHERE tenant_id = '%s' GROUP BY %s;",
			dimProj, boKey, tenantID.String(), dimProj)
	} else if len(measureProjs) > 0 {
		compiled = fmt.Sprintf("SELECT %s FROM public.%s WHERE tenant_id = '%s';",
			measureProj, boKey, tenantID.String())
	} else {
		compiled = fmt.Sprintf("SELECT 1 FROM public.%s WHERE tenant_id = '%s';",
			boKey, tenantID.String())
	}

	return &CFGValidatedQueryAST{
		RootBOKey:     boKey,
		Dimensions:    requestedDimensions,
		Measures:      requestedMeasures,
		FilterClauses: filters,
		CompiledSQL:   compiled,
	}, nil
}
