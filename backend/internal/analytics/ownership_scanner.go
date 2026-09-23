package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// This file answers one question against a corpus of saved report/query
// column definitions, without needing a live database connection: does any
// selected column's resolved join path have shared (or unresolved) root
// ownership (see JoinPath.RootOwnership)? That's the structural half of
// the fan-out cardinality trace's exposure question - "is this a latent
// hazard, or is it producing inflated totals in a real report right now."
//
// It is deliberately NOT a row-level check. A "shared" path shape means
// two returned rows COULD carry the same terminal target row (e.g. two
// orders for the same region); whether they actually do, for any given
// tenant's data, is a fact about rows, not schema, and this scanner has
// no opinion on it. The row-level version of this question is a numeric
// test with seeded data, not a scan.
//
// It's also not, on its own, an answer to "has a wrong number already
// been served." That depends on two more facts this scanner captures but
// does not resolve on its own: the column's aggregation (MIN/MAX are
// exposed but safe; SUM/COUNT/AVG/COUNT DISTINCT are not - see AtRisk)
// and whether any consumer actually sums this column across rows at all.
// Recomputing historical tile totals against the true root grain is a
// different artifact from this one.

// ScannedColumn describes one selected column's resolved join path, as it
// would appear in a saved report/query definition. A corpus of these -
// one per selected column, across every saved definition - is
// ScanForSharedOwnership's input.
//
// Aggregation is the measure's aggregation function as the report
// definition stores it ("SUM", "COUNT", "MIN", "MAX", "AVG",
// "COUNT_DISTINCT", or "" for a bare, unaggregated dimension). It drives
// OwnershipFinding.AtRisk - see isRiskyAggregation.
type ScannedColumn struct {
	ReportID    string    `json:"report_id"`
	ColumnName  string    `json:"column_name"`
	Aggregation string    `json:"aggregation,omitempty"`
	Path        *JoinPath `json:"path"`
}

// OwnershipFinding is one ownership exposure: a column, in a specific
// report, whose path is not definitively "unique" - either "shared"
// (summing/averaging it across rows double-counts whenever two rows
// share the same terminal target row) or "unresolved" (the resolver
// couldn't classify a hop, so this must be treated with the same caution
// as "shared" until proven otherwise - see JoinPath.RootOwnership).
type OwnershipFinding struct {
	ReportID   string `json:"report_id"`
	ColumnName string `json:"column_name"`
	// Ownership is RootOwnership's own verdict for this column's path:
	// "shared" or "unresolved". Never "unique" - those don't produce a
	// finding.
	Ownership    string       `json:"ownership"`
	OffendingHop JoinPathStep `json:"offending_hop"`
	Path         *JoinPath    `json:"path"`
	Measure      string       `json:"measure,omitempty"`
	// AtRisk answers ONE predicate, not "is it safe to roll this column
	// up across rows": is this column's OWN per-row/per-group value
	// still CORRECT despite the ownership/fan-out hazard on its path?
	// MIN/MAX/COUNT(DISTINCT) are immune to row duplication, so their
	// per-row value is right even when Ownership/Cardinality above are
	// not "unique"/"one" - that's what AtRisk: false means here.
	//
	// It does NOT mean those columns are safe to further roll up across
	// MULTIPLE returned rows with a plain sum. `MIN(cost) GROUP BY
	// region`, summed across region rows, is a total of nothing; `COUNT(
	// DISTINCT sku_id) GROUP BY category`, summed across categories,
	// double-counts a multi-category sku. That's a THIRD, independent
	// predicate - aggregation LINEARITY (is `+` a meaningful way to
	// combine this measure's values across groups at all, regardless of
	// grain or ownership) - and it belongs to whatever eventually
	// decides "is it safe to sum this column across the rows a query
	// returned," not to AtRisk. Grain (Cardinality), ownership
	// (Ownership), and linearity are three separate checks; a consumer
	// that reads AtRisk: false as "safe to display a summed total" has
	// silently dropped the third one.
	AtRisk bool `json:"atRisk"`
	// Cardinality is TraversalCardinality's own verdict for this same
	// path: "one", "many", or "unresolved" - mirroring Ownership, not a
	// bool, for the same reason Ownership isn't a bool: "no fan-out" and
	// "don't know if there's fan-out" are different facts, and folding
	// the second into "false" tells a consumer a safety property this
	// scanner never established. An earlier round of this field was a
	// bool computed as "offending hop == M:M", which is wrong twice over
	// - it can't say "unresolved", and it treats fan-out as a special
	// case of M:M rather than the independent axis it actually is (see
	// below).
	//
	// This is a genuinely different axis from Ownership, not a special
	// case of "M:M": fan-out and shared-ownership are two independent
	// hazards that happen to share some shapes.
	//
	//   - Fan-out (this field, != "one"): a hop multiplies rows in the
	//     traversal direction. The ROOT's own rows duplicate, so the
	//     root's own measures are unsafe to sum within the query's OWN
	//     flat join, before any cross-row rollup is even considered.
	//     Fixed by phasing/decomposition (see the fan-out cardinality
	//     trace).
	//   - Shared ownership (Ownership field): the terminal isn't
	//     functionally determined by the root, independent of whether
	//     anything fans out. Fixed by not rolling the affected column up
	//     ACROSS returned rows.
	//
	// "M:M" is fan-out AND shared at once - not a third category. So is
	// shape C (root -(N:1)-> a -(1:N)-> b): the SECOND hop fans out in
	// the traversal direction exactly like "M:M" does, which is why this
	// field must read the same whole-path, direction-aware fold as
	// TraversalCardinality and not "was the offending hop specifically
	// M:M" - a classification I had wrong in an earlier round, conflating
	// "shared terminal" with "duplicated root" as if they were the same
	// failure. The only shape that separates from fan-out entirely is
	// shape A (shared, zero fan-out: two M:1 hops, nothing 1:N anywhere).
	//
	// A consumer rule written for "don't roll this column up across
	// rows" (the Ownership fix) does NOT fix a Cardinality != "one"
	// finding - that one needs the query itself restructured, not a rule
	// about what the caller does with the result. And per RootOwnership's
	// own doc comment: this is a PER-COLUMN fact. Whether a QUERY's
	// results are safe to roll up needs every selected column's
	// Cardinality folded together (a clean column can still ride next to
	// a fanning-out one in the same result) - see
	// savedQueryApi.ts's isRowGrainIntact for that fold.
	Cardinality string `json:"cardinality"`
}

// riskyAggregations are the aggregation functions that produce a WRONG
// number given a fan-out or shared-ownership hazard. SUM/COUNT(*)/COUNT(col)
// scale with row duplication; AVG is wrong from weighting even without
// duplication. MIN/MAX are excluded: max-of-maxes and min-of-mins are
// correct regardless. COUNT(DISTINCT) is also excluded, deliberately, not
// just conservatively omitted: row duplication multiplies ROWS, not
// distinct VALUES, so COUNT(DISTINCT x) is identical before and after a
// fan-out join, and it is the standard remedy for a SUM that fan-out or
// shared ownership made unsafe - flagging it as at-risk would report the
// fix as still broken. An empty aggregation (a bare, unaggregated
// dimension) is excluded too - this scanner has no evidence such a
// column is ever rolled up at all.
var riskyAggregations = map[string]bool{
	"SUM":   true,
	"COUNT": true,
	"AVG":   true,
}

func isRiskyAggregation(agg string) bool {
	return riskyAggregations[strings.ToUpper(strings.TrimSpace(agg))]
}

// ScanForSharedOwnership reports every column, across a corpus of saved
// report/query column definitions, whose path is not definitively
// "unique". It calls JoinPath.RootOwnership() rather than reimplementing
// the predicate, so a scan result and the runtime behavior it audits can
// never drift apart - one function, two call sites (this scanner, and
// wherever RootOwnership lands on QueryResultColumn). See
// ownership_scanner_test.go's structural (not just behavioral) test for
// what pins that.
//
// This returns ROWS, not a count, deliberately: report id, the path,
// which hop triggered it, the affected column, its measure, and whether
// that measure is actually at risk. A count answers "is there a
// problem"; the row list is the triage artifact that distinguishes one
// sandbox report's MAX(cost) (exposed, not at risk) from the standard
// customer-hierarchy template's SUM(revenue) (exposed and at risk) every
// tenant inherits - and that distinction is what decides whether this is
// a latent hazard or a live incident with a blast radius.
//
// A definition whose path can't be resolved to catalog relationships at
// all is the caller's problem, not this function's: it should still
// appear in the input as a ScannedColumn with Path.RootOwnership() ==
// "unresolved" (a JoinPath with an unclassifiable step), not be dropped
// before reaching this function. A caller that silently skips columns it
// couldn't resolve, rather than passing them through as unresolved,
// under-reports exposure precisely where visibility is already weakest.
func ScanForSharedOwnership(columns []ScannedColumn) []OwnershipFinding {
	var findings []OwnershipFinding
	for _, col := range columns {
		analysis := col.Path.Analyze()
		if analysis.Ownership == "unique" {
			continue
		}
		hop := offendingHop(col.Path, analysis.Ownership)
		findings = append(findings, OwnershipFinding{
			ReportID:     col.ReportID,
			ColumnName:   col.ColumnName,
			Ownership:    analysis.Ownership,
			OffendingHop: hop,
			Path:         col.Path,
			Measure:      col.Aggregation,
			AtRisk:       isRiskyAggregation(col.Aggregation),
			Cardinality:  analysis.Cardinality,
		})
	}
	return findings
}

// offendingHop returns the specific step that produced ownership's
// verdict, for the finding's diagnostic detail: the first "M:1"/"M:M"
// step for "shared", or the first unclassifiable step for "unresolved".
// Only called after RootOwnership() has already confirmed such a step
// exists.
func offendingHop(p *JoinPath, ownership string) JoinPathStep {
	for _, step := range p.Steps {
		switch {
		case ownership == "shared" && (step.Cardinality == "M:1" || step.Cardinality == "M:M"):
			return step
		case ownership == "unresolved" && step.Cardinality != "1:1" && step.Cardinality != "1:M" && step.Cardinality != "M:1" && step.Cardinality != "M:M":
			return step
		}
	}
	return JoinPathStep{}
}

// LoadScannedColumnsFromFixture reads a JSON fixture file (a []ScannedColumn
// array) and returns it, so ScanForSharedOwnership can run against a
// checked-in fixture - for review here, or in CI - instead of only a live
// database connection. See testdata/ownership_fixture_sample.json for the
// expected shape.
func LoadScannedColumnsFromFixture(path string) ([]ScannedColumn, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading ownership fixture %s: %w", path, err)
	}
	var columns []ScannedColumn
	if err := json.Unmarshal(data, &columns); err != nil {
		return nil, fmt.Errorf("parsing ownership fixture %s: %w", path, err)
	}
	return columns, nil
}
