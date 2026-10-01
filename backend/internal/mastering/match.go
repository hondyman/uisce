package mastering

import (
	"encoding/json"
	"math"
	"strings"
)

// MatchRule is one row of mdm.<prefix>_match_rule.
type MatchRule struct {
	ID                string          `db:"id"`
	RuleCd            string          `db:"rule_cd"`
	RawDeterministic  json.RawMessage `db:"deterministic_keys"`
	RawFuzzy          json.RawMessage `db:"fuzzy_keys"`
	ThresholdAuto     float64         `db:"threshold_auto_match"`
	ThresholdReview   float64         `db:"threshold_review"`
	Priority          int             `db:"priority"`
	DeterministicKeys []string        `db:"-"`
	FuzzyKeys         []FuzzyKey      `db:"-"`
}

// FuzzyKey is one weighted comparison. trigram compares text similarity
// (pg_trgm, 0..1); exact compares case-insensitively (1 or 0).
type FuzzyKey struct {
	Field  string  `json:"field"`
	Method string  `json:"method"`
	Weight float64 `json:"weight"`
}

func (m *MatchRule) decode() error {
	if len(m.RawDeterministic) > 0 {
		if err := json.Unmarshal(m.RawDeterministic, &m.DeterministicKeys); err != nil {
			return err
		}
	}
	if len(m.RawFuzzy) > 0 {
		if err := json.Unmarshal(m.RawFuzzy, &m.FuzzyKeys); err != nil {
			return err
		}
	}
	for _, k := range m.FuzzyKeys {
		if !ident.MatchString(k.Field) {
			return errBadName(k.Field)
		}
	}
	return nil
}

// Outcome of matching a record.
const (
	MatchXref          = "XREF"          // seen before: its cross-reference
	MatchDeterministic = "DETERMINISTIC" // a shared identifier
	MatchFuzzy         = "FUZZY"         // scored at or above auto-match
	MatchReview        = "REVIEW"        // scored between review and auto-match: new record + steward candidate
	MatchNew           = "NEW"           // nothing close: a new golden record
	MatchConflict      = "CONFLICT"      // identifiers point at different golden records
)

// FuzzyScore scores a record against a candidate golden record. sims holds
// trigram similarities computed by the database. A trigram comparison is
// the anchor of the score: if either side lacks it there is no match (two
// funds are not the same because they share a currency). Any other
// comparison either side lacks is left out and the weights renormalized, so
// a sparse source is neither rewarded nor punished for what it doesn't say.
func FuzzyScore(keys []FuzzyKey, rec, cand map[string]any, sims map[string]float64) float64 {
	var sum, weight float64
	for _, k := range keys {
		a, b := rec[k.Field], cand[k.Field]
		if a == nil || b == nil || text(a) == "" || text(b) == "" {
			if k.Method == "trigram" {
				return 0
			}
			continue
		}
		var s float64
		switch k.Method {
		case "trigram":
			s = sims[k.Field]
		default:
			if strings.EqualFold(strings.TrimSpace(text(a)), strings.TrimSpace(text(b))) {
				s = 1
			}
		}
		sum += k.Weight * s
		weight += k.Weight
	}
	if weight == 0 {
		return 0
	}
	return math.Round(sum/weight*10000) / 10000
}

// Classify turns a fuzzy score into an outcome.
func (m *MatchRule) Classify(score float64) string {
	switch {
	case score >= m.ThresholdAuto:
		return MatchFuzzy
	case score >= m.ThresholdReview:
		return MatchReview
	default:
		return MatchNew
	}
}
