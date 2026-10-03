package querybuilder

import (
	"fmt"
	"strings"
	"time"
)

// Materialized-view routing policy.
//
// These predicates decide whether a query may be served from a materialized
// view or must fall back to the base table. They are called by CubeRouter
// (cube_router.go) and emit no SQL.
//
// The SQL-generating half of the old StarRocks MV path — StarRocksMaterializationManager,
// GenerateMVDDL and ParseStarRocksExplainPlan — was removed because it had zero
// production callers, and because GenerateMVDDL hardcoded SUM(notional), so a
// derived or formula metric reaching it would have materialized silently wrong.
// See ADR-028. Any future materialization path must route through CompileMetric
// or call ValidateMetricExpression first.

// EvaluateABACMVCompatibility evaluates whether row-level ABAC predicates allow routing to MV.
// If caller's ABAC context restricts columns below the MV grain (e.g. account-level restrictions
// when MV is region-level), query MUST fall back to base table.
func EvaluateABACMVCompatibility(mvGrains []string, userRestrictedGrains []string) (canRouteToMV bool, reason string) {
	mvGrainMap := make(map[string]bool)
	for _, g := range mvGrains {
		mvGrainMap[strings.ToLower(strings.TrimSpace(g))] = true
	}

	for _, userGrain := range userRestrictedGrains {
		norm := strings.ToLower(strings.TrimSpace(userGrain))
		if !mvGrainMap[norm] {
			return false, fmt.Sprintf("user has row-level ABAC predicate on sub-grain %q which is aggregated away by MV", userGrain)
		}
	}

	return true, ""
}

// EvaluateMVWatermarkStaleness checks if the Materialized View refresh timestamp is older than the BO watermark.
func EvaluateMVWatermarkStaleness(mvRefreshedAt time.Time, boWatermarkTimestamp time.Time) (isStale bool) {
	if mvRefreshedAt.IsZero() {
		return true
	}
	return mvRefreshedAt.Before(boWatermarkTimestamp)
}

// EvaluateStaleMVAction decides whether to serve the stale MV result with a warning flag or force raw table fallback.
func EvaluateStaleMVAction(isStale bool, policy string) (action string) {
	if !isStale {
		return "serve_fresh"
	}
	if strings.ToLower(strings.TrimSpace(policy)) == "force_raw_fallback" {
		return "fallback_raw"
	}
	return "serve_with_stale_flag"
}

// ResolveColdTierRoute resolves the federated StarRocks external catalog query target for Iceberg tables.
func ResolveColdTierRoute(metric MetricDefinition) (catalogTable string, routeTier string) {
	routeTier = "cold"
	target := fmt.Sprintf("lakekeeper_catalog.oms.iceberg_%s", sanitizeIdentifier(metric.BOID))
	if metric.MaterializationConfig.TargetTable != nil && *metric.MaterializationConfig.TargetTable != "" {
		target = *metric.MaterializationConfig.TargetTable
	}
	return target, routeTier
}
