package apistudio

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// FusedQuery represents a merged query serving multiple original requests
type FusedQuery struct {
	CompositeRequest analytics.BOSQLRequest
	SourceIDs        []string
}

// QueryFusionEngine identifies and merges compatible queries
type QueryFusionEngine struct {
}

// NewQueryFusionEngine creates a new engine
func NewQueryFusionEngine() *QueryFusionEngine {
	return &QueryFusionEngine{}
}

// TryFuse takes a map of SourceID -> Request and attempts to merge them
func (e *QueryFusionEngine) TryFuse(requests map[string]analytics.BOSQLRequest) ([]FusedQuery, map[string]analytics.BOSQLRequest) {
	// Group by signature (BOName + Env + Tenant + FiltersHash)
	groups := make(map[string][]string)

	for id, req := range requests {
		sig := generateRequestSignature(req)
		groups[sig] = append(groups[sig], id)
	}

	var fused []FusedQuery
	remaining := make(map[string]analytics.BOSQLRequest)

	for _, ids := range groups {
		if len(ids) > 1 {
			// Candidates for measure merging
			firstReq := requests[ids[0]]
			mergedMeasures := make(map[string]bool)

			// Collect all unique measures
			for _, id := range ids {
				for _, m := range requests[id].Measures {
					mergedMeasures[m] = true
				}
			}

			// Convert back to slice
			var finalMeasures []string
			for m := range mergedMeasures {
				finalMeasures = append(finalMeasures, m)
			}
			sort.Strings(finalMeasures)

			// Create Fused Request
			composite := firstReq
			composite.Measures = finalMeasures
			// Reset EndpointID since this is synthetic/fused (or keep one for context?)
			// Keeping one might be misleading for logging, but required for resolver?
			// Resolver likely needs BOName primarily.

			fused = append(fused, FusedQuery{
				CompositeRequest: composite,
				SourceIDs:        ids,
			})
		} else {
			// Single item, no fusion
			id := ids[0]
			remaining[id] = requests[id]
		}
	}

	return fused, remaining
}

func generateRequestSignature(req analytics.BOSQLRequest) string {
	// Simple signature: Env:Tenant:BO:SortedFilters
	var filterKeys []string
	for k, v := range req.Filters {
		filterKeys = append(filterKeys, fmt.Sprintf("%s=%v", k, v))
	}
	sort.Strings(filterKeys)

	tenant := ""
	if req.TenantID != nil {
		tenant = req.TenantID.String()
	}

	return fmt.Sprintf("%s:%s:%s:%s", req.Env, tenant, req.BOName, strings.Join(filterKeys, ","))
}
