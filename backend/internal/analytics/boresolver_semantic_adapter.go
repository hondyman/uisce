package analytics

import (
	"context"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/cbo"
)

// BOResolverSemanticAdapter implements cbo.SemanticRepository on top of
// boresolver.BOSQLGenerator/PostgresBORepository — the resolver already
// used (and proven correct against live data) for AI query generation and
// the Query Builder, rather than BOContextResolver.GenerateBOSQL/ResolveTerm
// (SemanticRepoAdapter, cbo_adapter.go), which requires semantic-term
// catalog nodes to carry a config.physical_mappings payload that's never
// actually populated anywhere in this environment. Reusing the proven
// resolver here means API Studio benefits from every physical-column-drift
// fix already made in PostgresBORepository.resolveCatalogPhysicalColumn,
// instead of re-solving the same problem a second, redundant way.
type BOResolverSemanticAdapter struct {
	repo *boresolver.PostgresBORepository
	gen  *boresolver.BOSQLGenerator
}

// NewBOResolverSemanticAdapter creates the adapter.
func NewBOResolverSemanticAdapter(repo *boresolver.PostgresBORepository, gen *boresolver.BOSQLGenerator) *BOResolverSemanticAdapter {
	return &BOResolverSemanticAdapter{repo: repo, gen: gen}
}

// ResolveBaseSQL resolves pc.BOName (a bo_key, e.g. "northwind.customer") to
// its definition and generates SQL for pc.GroupBy+pc.Measures (or, when
// neither is set — a base "give me everything" query — every field the BO
// declares, matching the same "no explicit projection = select all"
// convention used by metadata.BusinessObjectService.QueryBORecords and
// analytics.BOContextResolver.GenerateBOSQL).
func (a *BOResolverSemanticAdapter) ResolveBaseSQL(ctx context.Context, pc cbo.PlanContext) (string, error) {
	if pc.TenantID == nil {
		return "", fmt.Errorf("tenant id required to resolve business object %q", pc.BOName)
	}

	datasourceID := ""
	if pc.DatasourceID != nil {
		datasourceID = pc.DatasourceID.String()
	}
	boDef, err := a.repo.GetBOByTechnicalName(pc.BOName, pc.TenantID.String(), datasourceID)
	if err != nil {
		return "", err
	}

	requested := append(append([]string{}, pc.GroupBy...), pc.Measures...)
	var fieldIDs []string
	if len(requested) == 0 {
		for _, f := range boDef.Fields {
			fieldIDs = append(fieldIDs, f.ID)
		}
	} else {
		byName := make(map[string]string, len(boDef.Fields))
		for _, f := range boDef.Fields {
			byName[f.Name] = f.ID
		}
		for _, name := range requested {
			if id, ok := byName[name]; ok {
				fieldIDs = append(fieldIDs, id)
			}
			// A requested field this BO doesn't declare is silently
			// dropped rather than erroring the whole query — matches
			// SemanticRepoAdapter/GenerateBOSQL's existing behavior for
			// an unresolvable term.
		}
	}

	sql, _, err := a.gen.GenerateSQL(boresolver.SQLGenerationRequest{
		TenantID:         pc.TenantID.String(),
		BusinessObjectID: boDef.ID,
		SelectedFields:   fieldIDs,
	})
	if err != nil {
		return "", err
	}

	return appendPlanFilters(sql, pc.Filters), nil
}

// ResolvePreAggSQL: this adapter doesn't yet integrate with a
// pre-aggregation catalog. Returning "" is not an error — cbo.Planner's
// generateCandidates already falls back to "SELECT * FROM <target_table>"
// when the SemanticRepository can't produce a pre-agg-specific query.
func (a *BOResolverSemanticAdapter) ResolvePreAggSQL(ctx context.Context, pc cbo.PlanContext, preAgg cbo.PreAggDescriptor) (string, error) {
	return "", nil
}

// appendPlanFilters adds pc.Filters as a WHERE/AND clause. Filter keys and
// values originate from untrusted callers (HTTP query params, GraphQL field
// args) and are embedded directly into the SQL text rather than passed as
// bind parameters — see SemanticRepoAdapter.appendFilters, which this
// shares logic with — so keys are restricted to a strict identifier
// whitelist and values have embedded quotes escaped.
func appendPlanFilters(sql string, filters map[string]interface{}) string {
	if len(filters) == 0 {
		return sql
	}

	conditions := []string{}
	for k, v := range filters {
		if !filterIdentifierPattern.MatchString(k) {
			continue
		}
		escaped := strings.ReplaceAll(fmt.Sprintf("%v", v), "'", "''")
		conditions = append(conditions, fmt.Sprintf("%q = '%s'", k, escaped))
	}
	if len(conditions) == 0 {
		return sql
	}

	joiner := " WHERE "
	if strings.Contains(strings.ToUpper(sql), " WHERE ") {
		joiner = " AND "
	}
	return sql + joiner + strings.Join(conditions, " AND ")
}
