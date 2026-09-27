package mastering

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Contribution is one source record's current canonical attributes, linked
// to the golden record through the cross-reference.
type Contribution struct {
	SourceID  string
	SourceCd  string
	SourceKey string
	AsOf      time.Time
	Attrs     map[string]any
}

// SurvivalRule is one attribute's survivorship rule (crims
// mdm.survivorship_rule, keyed by golden attribute).
type SurvivalRule struct {
	ID           string
	Attribute    string
	Strategy     string   // SOURCE_PRIORITY | PROVIDER_AUTHORITATIVE | MOST_RECENT | CONSERVATIVE_MIN | CONSERVATIVE_MAX
	Priority     []string // source system codes, best first
	TolerancePct float64  // numeric: a larger move than this is held for review
	StalenessSec int      // a source older than this loses to a fresh one
}

// Candidate is one source's value for an attribute.
type Candidate struct {
	SourceID  string    `json:"source_id"`
	SourceCd  string    `json:"source"`
	SourceKey string    `json:"source_key"`
	Value     any       `json:"value"`
	AsOf      time.Time `json:"as_of"`
	Stale     bool      `json:"stale,omitempty"`
}

// Decision is the surviving value of one attribute and why: the provenance
// kept per golden version (golden_field + survivorship_log).
type Decision struct {
	Attribute  string      `json:"attribute"`
	Value      any         `json:"value"`
	Winner     Candidate   `json:"winner"`
	RuleID     string      `json:"rule_id,omitempty"`
	Strategy   string      `json:"strategy"`
	Reason     string      `json:"reason"`
	Confidence float64     `json:"confidence"`
	Held       bool        `json:"held,omitempty"` // anomaly: the previous value was kept
	Competing  []Candidate `json:"competing"`
}

// Survive picks every attribute's surviving value from the contributions.
// previous is the current golden attributes (for anomaly checks); issues
// are the anomalies held for review.
func Survive(contribs []Contribution, rules map[string]SurvivalRule, previous map[string]any, now time.Time) (map[string]Decision, []Issue) {
	// Deterministic order whatever order the sources were read in.
	sort.SliceStable(contribs, func(i, j int) bool {
		if contribs[i].SourceCd != contribs[j].SourceCd {
			return contribs[i].SourceCd < contribs[j].SourceCd
		}
		return contribs[i].SourceKey < contribs[j].SourceKey
	})
	attrs := map[string]bool{}
	for _, c := range contribs {
		for a, v := range c.Attrs {
			if v != nil {
				attrs[a] = true
			}
		}
	}
	out := map[string]Decision{}
	var issues []Issue
	for a := range attrs {
		d, issue := surviveAttr(a, contribs, rules[a], previous[a], now)
		out[a] = d
		if issue != nil {
			issues = append(issues, *issue)
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Attribute < issues[j].Attribute })
	return out, issues
}

func surviveAttr(attr string, contribs []Contribution, rule SurvivalRule, previous any, now time.Time) (Decision, *Issue) {
	var all []Candidate
	for _, c := range contribs {
		v, ok := c.Attrs[attr]
		if !ok || v == nil {
			continue
		}
		stale := rule.StalenessSec > 0 && !c.AsOf.IsZero() && now.Sub(c.AsOf) > time.Duration(rule.StalenessSec)*time.Second
		all = append(all, Candidate{SourceID: c.SourceID, SourceCd: c.SourceCd, SourceKey: c.SourceKey, Value: v, AsOf: c.AsOf, Stale: stale})
	}
	// Fresh sources compete; only when every source is stale does a stale
	// value survive (flagged), so an old feed never blanks a golden value.
	pool := make([]Candidate, 0, len(all))
	for _, c := range all {
		if !c.Stale {
			pool = append(pool, c)
		}
	}
	allStale := len(pool) == 0
	if allStale {
		pool = all
	}

	strategy := rule.Strategy
	if strategy == "" {
		strategy = "MOST_RECENT"
	}
	d := Decision{Attribute: attr, RuleID: rule.ID, Strategy: strategy, Competing: all}
	var reason string
	switch strategy {
	case "SOURCE_PRIORITY":
		d.Winner, reason = byPriority(pool, rule.Priority, false)
	case "PROVIDER_AUTHORITATIVE":
		d.Winner, reason = byPriority(pool, rule.Priority, true)
	case "CONSERVATIVE_MIN", "CONSERVATIVE_MAX":
		d.Winner, reason = byExtreme(pool, strategy == "CONSERVATIVE_MAX")
	default:
		d.Winner, reason = mostRecent(pool), "most recent source"
	}
	if allStale && len(all) > 0 {
		reason += "; every source is stale"
	}
	d.Value = d.Winner.Value
	d.Reason = reason
	d.Confidence = agreement(all, d.Value)
	if allStale {
		d.Confidence /= 2
	}

	// A numeric move beyond the tolerance keeps the previous value and asks
	// a steward (a 25% AUM jump is more often a unit error than real).
	if rule.TolerancePct > 0 && previous != nil {
		if old, ok1 := number(previous); ok1 {
			if nw, ok2 := number(d.Value); ok2 && old != 0 {
				if pct := math.Abs(nw-old) / math.Abs(old) * 100; pct > rule.TolerancePct {
					d.Held = true
					d.Value = previous
					d.Reason = fmt.Sprintf("held: %s moved %.1f%% (tolerance %.1f%%)", d.Winner.SourceCd, pct, rule.TolerancePct)
					return d, &Issue{Code: "ANOMALY", Severity: SevWarning, Attribute: attr, RuleID: rule.ID,
						Message: fmt.Sprintf("%s: %s reported %s, %.1f%% from %s (tolerance %.1f%%); the previous value is kept",
							attr, d.Winner.SourceCd, text(nw), pct, text(old), rule.TolerancePct)}
				}
			}
		}
	}
	return d, nil
}

func byPriority(pool []Candidate, priority []string, authoritativeOnly bool) (Candidate, string) {
	for i, src := range priority {
		for _, c := range pool {
			if strings.EqualFold(c.SourceCd, src) {
				return c, fmt.Sprintf("source priority: %s is #%d", c.SourceCd, i+1)
			}
		}
	}
	if authoritativeOnly {
		return mostRecent(pool), "no authoritative source supplied it; most recent source"
	}
	return mostRecent(pool), "no ranked source supplied it; most recent source"
}

func mostRecent(pool []Candidate) Candidate {
	best := pool[0]
	for _, c := range pool[1:] {
		if c.AsOf.After(best.AsOf) {
			best = c
		}
	}
	return best
}

func byExtreme(pool []Candidate, max bool) (Candidate, string) {
	var best *Candidate
	var bv float64
	for i := range pool {
		v, ok := number(pool[i].Value)
		if !ok {
			continue
		}
		if best == nil || (max && v > bv) || (!max && v < bv) {
			best, bv = &pool[i], v
		}
	}
	if best == nil {
		return mostRecent(pool), "no numeric value; most recent source"
	}
	if max {
		return *best, "highest value"
	}
	return *best, "lowest value"
}

// agreement is the share of sources that agree with the surviving value.
func agreement(all []Candidate, v any) float64 {
	if len(all) == 0 {
		return 0
	}
	n := 0
	for _, c := range all {
		if sameValue(c.Value, v) {
			n++
		}
	}
	return math.Round(float64(n)/float64(len(all))*10000) / 10000
}

func sameValue(a, b any) bool {
	if x, ok := number(a); ok {
		if y, ok := number(b); ok {
			return x == y
		}
	}
	return strings.EqualFold(strings.TrimSpace(text(a)), strings.TrimSpace(text(b)))
}

func number(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}
