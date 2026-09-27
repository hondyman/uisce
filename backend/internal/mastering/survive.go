package mastering

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// Survivorship, in layers - declarative first, expressions only where a
// ranking can't say it, stewards for true exceptions:
//
//  1. the entity's source hierarchy (by field group) is every attribute's
//     default: no rule needed;
//  2. a survivorship rule overrides it per attribute: a ranking, most
//     recent, most frequent, most complete, min / max, with staleness and
//     an anomaly tolerance;
//  3. a rule may name a selection rule - a catalog rule on the one rule
//     engine - each candidate must satisfy to be chosen, evaluated against
//     the other candidates (e.g. "within 20% of the median of the other
//     sources"); ENFORCE excludes a failing candidate, FLAG only reports;
//  4. steward overrides and merges (outside this function).
//
// Every decision keeps its reason and every competing value, with whether
// each passed the selection rule.

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
	Strategy     string   // SOURCE_PRIORITY | PROVIDER_AUTHORITATIVE | MOST_RECENT | MOST_FREQUENT | MOST_COMPLETE | CONSERVATIVE_MIN | CONSERVATIVE_MAX
	Priority     []string // source system codes, best first (empty: the entity's hierarchy)
	TolerancePct float64  // numeric: a larger move than this is held for review
	StalenessSec int      // a source older than this loses to a fresh one
	// SelectionRuleID names the catalog rule (domain survivorship) a
	// candidate must satisfy to be chosen.
	SelectionRuleID string
	SelectionMode   string // ENFORCE (default) | FLAG
	OnNoneSelected  string // HOLD (default) | ALLOW
	// SelectionMinPeers: the selection rule applies only when at least this
	// many other sources have a value (no consensus against one peer).
	SelectionMinPeers int
}

// Candidate is one source's value for an attribute.
type Candidate struct {
	SourceID  string    `json:"source_id"`
	SourceCd  string    `json:"source"`
	SourceKey string    `json:"source_key"`
	Value     any       `json:"value"`
	AsOf      time.Time `json:"as_of"`
	Stale     bool      `json:"stale,omitempty"`
	// Selected is whether the candidate passed the attribute's selection
	// rule (absent: no selection rule); Note says why not.
	Selected *bool  `json:"selected,omitempty"`
	Note     string `json:"note,omitempty"`
	record   map[string]any
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
	Held       bool        `json:"held,omitempty"` // the previous value was kept (anomaly, or nothing selectable)
	Competing  []Candidate `json:"competing"`
}

// SelectionContext is what a selection rule reads for one candidate. As
// rule data: value, source, source_key, age_hours, stale, rank, peers (the
// other candidates, as rows), all (every candidate), previous (the current
// golden value), has_previous, record (the candidate's source record).
type SelectionContext struct {
	Attribute string
	Candidate Candidate
	Rank      int // 1-based position in the ranking; 0 when unranked
	Peers     []Candidate
	All       []Candidate
	Previous  any
	Now       time.Time
	Ranking   []string
}

// Data is the context as the rule engine reads it.
func (c SelectionContext) Data() map[string]any {
	row := func(x Candidate) map[string]any {
		return map[string]any{"value": x.Value, "source": x.SourceCd, "source_key": x.SourceKey,
			"age_hours": ageHours(x.AsOf, c.Now), "stale": x.Stale, "rank": float64(rankOf(c.Ranking, x.SourceCd))}
	}
	peers := make([]any, 0, len(c.Peers))
	for _, p := range c.Peers {
		peers = append(peers, row(p))
	}
	all := make([]any, 0, len(c.All))
	for _, p := range c.All {
		all = append(all, row(p))
	}
	d := row(c.Candidate)
	d["peers"] = peers
	d["all"] = all
	d["previous"] = c.Previous
	d["has_previous"] = c.Previous != nil
	d["record"] = c.Candidate.record
	d["attribute"] = c.Attribute
	return d
}

// SelectFunc evaluates an attribute's selection rule for one candidate.
// name is the rule's name (for the reason); an error means the rule could
// not be evaluated, which never counts as a pass.
type SelectFunc func(ruleID string, ctx SelectionContext) (pass bool, name string, err error)

// SurviveOptions carries what survivorship needs beyond the rules.
type SurviveOptions struct {
	// Hierarchy is the entity's source ranking for an attribute (by its
	// field group), used when the attribute has no rule or the rule no
	// ranking.
	Hierarchy func(attr string) []string
	// Select evaluates selection rules (nil: rules are not evaluated, and
	// an attribute with a selection rule is held for a steward).
	Select SelectFunc
}

// Issue codes survivorship raises.
const (
	IssueAnomaly       = "ANOMALY"
	IssueSelectionHold = "SELECTION_HOLD" // nothing passed the selection rule: previous kept, steward to review
	IssueSelectionFlag = "SELECTION_FLAG" // the winner failed the selection rule (FLAG mode), or no candidate passed (ALLOW)
)

// Survive picks every attribute's surviving value from the contributions.
// previous is the current golden attributes (for anomaly checks and
// selection rules); issues are what a steward should look at.
func Survive(contribs []Contribution, rules map[string]SurvivalRule, previous map[string]any, now time.Time, opts *SurviveOptions) (map[string]Decision, []Issue) {
	if opts == nil {
		opts = &SurviveOptions{}
	}
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
		d, is := surviveAttr(a, contribs, rules[a], previous[a], now, opts)
		out[a] = d
		issues = append(issues, is...)
	}
	sort.SliceStable(issues, func(i, j int) bool { return issues[i].Attribute < issues[j].Attribute })
	return out, issues
}

func surviveAttr(attr string, contribs []Contribution, rule SurvivalRule, previous any, now time.Time, opts *SurviveOptions) (Decision, []Issue) {
	var all []Candidate
	for _, c := range contribs {
		v, ok := c.Attrs[attr]
		if !ok || v == nil {
			continue
		}
		stale := rule.StalenessSec > 0 && !c.AsOf.IsZero() && now.Sub(c.AsOf) > time.Duration(rule.StalenessSec)*time.Second
		all = append(all, Candidate{SourceID: c.SourceID, SourceCd: c.SourceCd, SourceKey: c.SourceKey, Value: v, AsOf: c.AsOf, Stale: stale, record: c.Attrs})
	}

	// The ranking: the rule's own, else the entity's hierarchy.
	ranking := rule.Priority
	fromHierarchy := false
	if len(ranking) == 0 && opts.Hierarchy != nil {
		ranking, fromHierarchy = opts.Hierarchy(attr), true
	}
	strategy := rule.Strategy
	if strategy == "" {
		// No rule: the hierarchy decides; with no hierarchy, recency.
		if len(ranking) > 0 {
			strategy = "SOURCE_PRIORITY"
		} else {
			strategy = "MOST_RECENT"
		}
	}
	d := Decision{Attribute: attr, RuleID: rule.ID, Strategy: strategy}
	var issues []Issue

	// Selection rule: which candidates may be chosen at all.
	eligible := all
	selNote := ""
	if rule.SelectionRuleID != "" && len(all)-1 < rule.SelectionMinPeers {
		selNote = fmt.Sprintf("; selection rule not applied: %d other source(s), %d needed", len(all)-1, rule.SelectionMinPeers)
	} else if rule.SelectionRuleID != "" {
		var passed []Candidate
		var failed []string
		ruleName := rule.SelectionRuleID
		for i := range all {
			ok, name, err := false, ruleName, error(nil)
			if opts.Select == nil {
				err = fmt.Errorf("selection rules are not available")
			} else {
				peers := make([]Candidate, 0, len(all)-1)
				for j := range all {
					if j != i {
						peers = append(peers, all[j])
					}
				}
				ok, name, err = opts.Select(rule.SelectionRuleID, SelectionContext{Attribute: attr, Candidate: all[i],
					Rank: rankOf(ranking, all[i].SourceCd), Peers: peers, All: all, Previous: previous, Now: now, Ranking: ranking})
			}
			if name != "" {
				ruleName = name
			}
			pass := ok && err == nil
			all[i].Selected = &pass
			switch {
			case err != nil:
				all[i].Note = "selection rule could not be evaluated: " + err.Error()
			case !ok:
				all[i].Note = "failed selection rule " + ruleName
			}
			if pass {
				passed = append(passed, all[i])
			} else {
				failed = append(failed, all[i].SourceCd)
			}
		}
		if len(failed) > 0 {
			selNote = fmt.Sprintf("; selection rule %s excluded %s", ruleName, strings.Join(failed, ", "))
		}
		switch {
		case strings.EqualFold(rule.SelectionMode, "FLAG"):
			// Report only: everyone may still win.
			if len(failed) > 0 {
				selNote = fmt.Sprintf("; selection rule %s (flag only) failed for %s", ruleName, strings.Join(failed, ", "))
			}
		case len(passed) > 0:
			eligible = passed
		case strings.EqualFold(rule.OnNoneSelected, "ALLOW"):
			issues = append(issues, Issue{Code: IssueSelectionFlag, Severity: SevWarning, Attribute: attr, RuleID: rule.ID,
				Message: fmt.Sprintf("%s: no source passed selection rule %s; chosen from all sources", attr, ruleName)})
			selNote = fmt.Sprintf("; no source passed selection rule %s, chosen from all", ruleName)
		default: // HOLD
			d.Competing, d.Held, d.Value = all, true, previous
			d.Reason = fmt.Sprintf("held: no source passed selection rule %s", ruleName)
			if previous == nil {
				d.Reason += " and there is no previous value"
			}
			d.Winner = Candidate{SourceCd: "PREVIOUS", Value: previous}
			issues = append(issues, Issue{Code: IssueSelectionHold, Severity: SevWarning, Attribute: attr, RuleID: rule.ID,
				Message: fmt.Sprintf("%s: no source passed selection rule %s (%s); the previous value is kept for a steward", attr, ruleName, candidateNotes(all))})
			return d, issues
		}
	}

	// Fresh sources compete; only when every eligible source is stale does
	// a stale value survive (flagged), so an old feed never blanks a value.
	pool := make([]Candidate, 0, len(eligible))
	for _, c := range eligible {
		if !c.Stale {
			pool = append(pool, c)
		}
	}
	allStale := len(pool) == 0
	if allStale {
		pool = eligible
	}

	var reason string
	switch strategy {
	case "SOURCE_PRIORITY":
		d.Winner, reason = byPriority(pool, ranking, false)
	case "PROVIDER_AUTHORITATIVE":
		d.Winner, reason = byPriority(pool, ranking, true)
	case "CONSERVATIVE_MIN", "CONSERVATIVE_MAX":
		d.Winner, reason = byExtreme(pool, strategy == "CONSERVATIVE_MAX")
	case "MOST_FREQUENT":
		d.Winner, reason = mostFrequent(pool, ranking)
	case "MOST_COMPLETE":
		d.Winner, reason = mostComplete(pool, ranking)
	default:
		d.Winner, reason = mostRecent(pool), "most recent source"
	}
	if fromHierarchy && (strategy == "SOURCE_PRIORITY" || strategy == "PROVIDER_AUTHORITATIVE") {
		reason += " (entity hierarchy)"
	}
	if allStale && len(eligible) > 0 {
		reason += "; every source is stale"
	}
	d.Value = d.Winner.Value
	d.Reason = reason + selNote
	d.Competing = all
	d.Confidence = agreement(all, d.Value)
	if allStale {
		d.Confidence /= 2
	}
	if strings.EqualFold(rule.SelectionMode, "FLAG") && d.Winner.Selected != nil && !*d.Winner.Selected {
		issues = append(issues, Issue{Code: IssueSelectionFlag, Severity: SevWarning, Attribute: attr, RuleID: rule.ID,
			Message: fmt.Sprintf("%s: %s won but %s", attr, d.Winner.SourceCd, d.Winner.Note)})
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
					issues = append(issues, Issue{Code: IssueAnomaly, Severity: SevWarning, Attribute: attr, RuleID: rule.ID,
						Message: fmt.Sprintf("%s: %s reported %s, %.1f%% from %s (tolerance %.1f%%); the previous value is kept",
							attr, d.Winner.SourceCd, text(nw), pct, text(old), rule.TolerancePct)})
				}
			}
		}
	}
	return d, issues
}

func candidateNotes(cs []Candidate) string {
	var parts []string
	for _, c := range cs {
		parts = append(parts, fmt.Sprintf("%s %s", c.SourceCd, text(c.Value)))
	}
	return strings.Join(parts, ", ")
}

func rankOf(ranking []string, source string) int {
	for i, s := range ranking {
		if strings.EqualFold(s, source) {
			return i + 1
		}
	}
	return 0
}

func ageHours(asOf, now time.Time) float64 {
	if asOf.IsZero() {
		return 0
	}
	return math.Round(now.Sub(asOf).Hours()*100) / 100
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

// mostFrequent: the value most sources agree on; a tie goes to the best
// ranked source among the tied values, then the most recent.
func mostFrequent(pool []Candidate, ranking []string) (Candidate, string) {
	counts := map[string]int{}
	for _, c := range pool {
		counts[strings.ToUpper(strings.TrimSpace(text(c.Value)))]++
	}
	top := 0
	for _, n := range counts {
		if n > top {
			top = n
		}
	}
	var tied []Candidate
	for _, c := range pool {
		if counts[strings.ToUpper(strings.TrimSpace(text(c.Value)))] == top {
			tied = append(tied, c)
		}
	}
	w, _ := byPriority(tied, ranking, false)
	return w, fmt.Sprintf("most frequent: %d of %d sources agree", top, len(pool))
}

// mostComplete: the fullest value (longest text); a tie goes to the best
// ranked source, then the most recent.
func mostComplete(pool []Candidate, ranking []string) (Candidate, string) {
	best := 0
	for _, c := range pool {
		if n := len(strings.TrimSpace(text(c.Value))); n > best {
			best = n
		}
	}
	var tied []Candidate
	for _, c := range pool {
		if len(strings.TrimSpace(text(c.Value))) == best {
			tied = append(tied, c)
		}
	}
	w, _ := byPriority(tied, ranking, false)
	return w, "most complete value"
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
