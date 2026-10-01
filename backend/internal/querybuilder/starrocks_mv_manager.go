package querybuilder

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
)

// StarRocksMaterializationManager generates DDL and handles query routing / mvHit observability for StarRocks MVs.
type StarRocksMaterializationManager struct {
	targetStarRocksVersion string
}

// NewStarRocksMaterializationManager creates a new StarRocksMaterializationManager.
func NewStarRocksMaterializationManager(version string) *StarRocksMaterializationManager {
	if version == "" {
		version = "3.2.0"
	}
	return &StarRocksMaterializationManager{
		targetStarRocksVersion: version,
	}
}

// GeneratedMVDDL contains the content-hash-stable SQL DDL for a StarRocks Materialized View.
type GeneratedMVDDL struct {
	MVName      string
	DDL         string
	ContentHash string
}

// GenerateMVDDL produces deterministic, content-hash-stable DDL for an async StarRocks Materialized View.
// Tenant scoping is embedded into the name (mv_{tenant}_{bo}_{metric}) or gold copy name.
func (m *StarRocksMaterializationManager) GenerateMVDDL(tenantID string, isGoldCopy bool, metric MetricDefinition) (*GeneratedMVDDL, error) {
	if metric.BOID == "" || metric.Name == "" {
		return nil, fmt.Errorf("metric requires boid and name for MV DDL generation")
	}

	scopePrefix := "gold"
	if !isGoldCopy && tenantID != "" {
		scopePrefix = sanitizeIdentifier(tenantID)
	}
	mvName := fmt.Sprintf("mv_%s_%s_%s", scopePrefix, sanitizeIdentifier(metric.BOID), sanitizeIdentifier(metric.Name))

	// Sort grains for deterministic column selection & grouping
	sortedGrains := make([]string, len(metric.GrainAllowlist))
	copy(sortedGrains, metric.GrainAllowlist)
	sort.Strings(sortedGrains)

	var selectCols []string
	var groupByCols []string

	for _, g := range sortedGrains {
		sanitized := sanitizeIdentifier(g)
		selectCols = append(selectCols, sanitized)
		groupByCols = append(groupByCols, sanitized)
	}

	// Compile measure expression
	measureExpr := "SUM(notional)"
	if metric.Expression.Kind == "aggregation" {
		fn := strings.ToUpper(metric.Expression.Fn)
		term := sanitizeIdentifier(metric.Expression.TermNodeID)
		measureExpr = fmt.Sprintf("%s(%s)", fn, term)
	}
	selectCols = append(selectCols, fmt.Sprintf("%s AS metric_val", measureExpr))

	refreshSchedule := "ASYNC EVERY(INTERVAL 1 HOUR)"
	if metric.MaterializationConfig.RefreshSchedule != nil && *metric.MaterializationConfig.RefreshSchedule != "" {
		refreshSchedule = *metric.MaterializationConfig.RefreshSchedule
	}

	sourceTable := fmt.Sprintf("oms.%s", sanitizeIdentifier(metric.BOID))
	groupByClause := ""
	if len(groupByCols) > 0 {
		groupByClause = fmt.Sprintf("\nGROUP BY %s", strings.Join(groupByCols, ", "))
	}

	ddl := fmt.Sprintf(`CREATE MATERIALIZED VIEW %s
REFRESH %s
PROPERTIES (
  "replication_num" = "1",
  "storage_medium" = "HDD"
)
AS SELECT
  %s
FROM %s%s;`,
		mvName,
		refreshSchedule,
		strings.Join(selectCols, ",\n  "),
		sourceTable,
		groupByClause,
	)

	hash := sha256.Sum256([]byte(ddl))
	contentHash := hex.EncodeToString(hash[:])

	return &GeneratedMVDDL{
		MVName:      mvName,
		DDL:         ddl,
		ContentHash: contentHash,
	}, nil
}

// MVHitParseResult represents the outcome of parsing a StarRocks EXPLAIN plan for MV rewrite.
type MVHitParseResult struct {
	MVHit   *bool  `json:"mvHit"`   // true, false, or nil (if unknown/unrecognized plan format)
	MVName  string `json:"mvName,omitempty"`
	Warning string `json:"warning,omitempty"`
}

// ParseStarRocksExplainPlan parses the EXPLAIN output from StarRocks to identify whether
// the query was transparently rewritten to hit a Materialized View.
// Features version pinning and fallback to (mvHit: nil + warning) when plan structure is unrecognized.
func (m *StarRocksMaterializationManager) ParseStarRocksExplainPlan(explainOutput string) MVHitParseResult {
	if strings.TrimSpace(explainOutput) == "" {
		return MVHitParseResult{
			MVHit:   nil,
			Warning: "empty EXPLAIN plan output",
		}
	}

	// StarRocks native query rewrite indicator in EXPLAIN COST / EXPLAIN output
	// Format example:
	// "PREDICATES: ...\n MaterializedView: mv_gold_orders_revenue\n" or "rollup: mv_..." or "Materialized View: ..."
	lowerPlan := strings.ToLower(explainOutput)

	if strings.Contains(lowerPlan, "materializedview:") || strings.Contains(lowerPlan, "materialized view:") || strings.Contains(lowerPlan, "mv:") {
		// Extract MV name if possible
		var mvName string
		lines := strings.Split(explainOutput, "\n")
		for _, line := range lines {
			lowerLine := strings.ToLower(line)
			if strings.Contains(lowerLine, "materializedview:") || strings.Contains(lowerLine, "materialized view:") {
				parts := strings.Split(line, ":")
				if len(parts) >= 2 {
					mvName = strings.TrimSpace(parts[1])
				}
				break
			}
		}
		hit := true
		return MVHitParseResult{
			MVHit:  &hit,
			MVName: mvName,
		}
	}

	// Check if plan recognized standard OlapScanNode
	if strings.Contains(lowerPlan, "olapscannode") || strings.Contains(lowerPlan, "tablescan") || strings.Contains(lowerPlan, "scan") {
		hit := false
		return MVHitParseResult{
			MVHit: &hit,
		}
	}

	// Unrecognized plan structure -> graceful fallback
	return MVHitParseResult{
		MVHit:   nil,
		Warning: fmt.Sprintf("unrecognized StarRocks EXPLAIN plan shape for version %s", m.targetStarRocksVersion),
	}
}

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
