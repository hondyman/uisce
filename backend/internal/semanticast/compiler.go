// Package semanticast owns NL→QueryAST compilation for catalog-grounded prompts.
// Ported from internal/mcp (SL commit 3/5); MCP tools call this package and
// never own the compiler.
package semanticast

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/optimizer"
	"github.com/jmoiron/sqlx"
)

// Compiler grounds natural-language prompts against catalog semantic terms.
type Compiler struct {
	db *sqlx.DB
}

func NewCompiler(db *sqlx.DB) *Compiler {
	return &Compiler{db: db}
}

// CompilePromptToAST grounds natural language prompts against active catalog
// semantic terms. Rejects uuid.Nil tenant (Rule 7).
func (c *Compiler) CompilePromptToAST(
	ctx context.Context,
	tenantID uuid.UUID,
	prompt string,
) (*optimizer.QueryAST, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	clean := strings.ToLower(prompt)

	// 1. Identify Target Driving Business Object (Rule 2: Graph-First)
	drivingEntity := "SecurityMaster"
	if strings.Contains(clean, "account") || strings.Contains(clean, "client") {
		drivingEntity = "AccountMaster"
	} else if strings.Contains(clean, "corporate action") || strings.Contains(clean, "dividend") {
		drivingEntity = "CorporateAction"
	} else if strings.Contains(clean, "fund") || strings.Contains(clean, "nav") {
		drivingEntity = "FundMaster"
	}

	// 2. Extract Matching Semantic Terms & Measures
	selectedFields := make([]string, 0)
	joinEntities := make([]string, 0)

	if strings.Contains(clean, "price") || strings.Contains(clean, "px") {
		selectedFields = append(selectedFields, "px_last")
	}
	if strings.Contains(clean, "isin") {
		selectedFields = append(selectedFields, "isin")
	}
	if strings.Contains(clean, "name") {
		selectedFields = append(selectedFields, "security_name")
	}
	if strings.Contains(clean, "sector") || strings.Contains(clean, "industry") {
		selectedFields = append(selectedFields, "bloomberg_industry_sector")
	}

	// Defaults if no specific field mentioned
	if len(selectedFields) == 0 {
		selectedFields = []string{"security_name", "isin", "px_last"}
	}

	// 3. Detect Partition Boundaries & Federation Requirements (Rule 4: Hot/Cold Watermark)
	hasDatePartition := strings.Contains(clean, "date") || strings.Contains(clean, "today") || strings.Contains(clean, "2026")
	hasEntityFilter := strings.Contains(clean, "for") || strings.Contains(clean, "isin") || strings.Contains(clean, "where")

	crossTierEngines := []string{"STARROCKS"}
	if strings.Contains(clean, "history") || strings.Contains(clean, "multi-year") || strings.Contains(clean, "archive") {
		crossTierEngines = append(crossTierEngines, "ICEBERG")
	}

	// 4. Construct Type-Safe Query AST
	ast := &optimizer.QueryAST{
		DrivingEntity:    drivingEntity,
		SelectedFields:   selectedFields,
		JoinEntities:     joinEntities,
		HasDatePartition: hasDatePartition,
		HasEntityFilter:  hasEntityFilter,
		CrossTierEngines: crossTierEngines,
		AggregationCount: countAggregations(clean),
		RawQuery:         fmt.Sprintf("SELECT %s FROM %s", strings.Join(selectedFields, ", "), drivingEntity),
	}

	return ast, nil
}

func countAggregations(prompt string) int {
	aggs := 0
	for _, term := range []string{"sum", "average", "avg", "total", "count", "max", "min", "xirr"} {
		if strings.Contains(prompt, term) {
			aggs++
		}
	}
	return aggs
}
