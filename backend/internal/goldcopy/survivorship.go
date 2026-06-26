package goldcopy

import (
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Survivorship resolves field values across competing source records.
type Survivorship struct {
	// rules: field_name → ordered list of survivorship rules (by priority)
	rules map[string][]*SurvivorshipRule
}

// NewSurvivorship builds a Survivorship resolver from the given rules.
func NewSurvivorship(rules []*SurvivorshipRule) *Survivorship {
	m := make(map[string][]*SurvivorshipRule)
	for _, r := range rules {
		m[r.FieldName] = append(m[r.FieldName], r)
	}
	// Sort each field's rules by priority
	for k := range m {
		rs := m[k]
		sort.Slice(rs, func(i, j int) bool { return rs[i].Priority < rs[j].Priority })
		m[k] = rs
	}
	return &Survivorship{rules: m}
}

// Resolve picks the winning value for fieldName across rawRecords.
// Falls back to the source with the highest QualityScore if no rule matches.
func (s *Survivorship) Resolve(fieldName string, rawRecords []*RawPortfolioRecord) SurvivorshipResult {
	fieldRules, ok := s.rules[fieldName]
	if !ok || len(fieldRules) == 0 {
		// No rule: pick highest quality score source that has a non-empty value.
		return s.highestQualityFallback(fieldName, rawRecords)
	}

	rule := fieldRules[0] // primary rule (lowest priority number = highest precedence)

	switch rule.Strategy {
	case "prefer_source":
		return s.preferSource(fieldName, rule, rawRecords)
	case "earliest_non_null":
		return s.earliestNonNull(fieldName, rule, rawRecords)
	case "latest_by":
		return s.latestBy(fieldName, rule, rawRecords)
	case "highest_quality":
		return s.highestQualityFallback(fieldName, rawRecords)
	default:
		return s.highestQualityFallback(fieldName, rawRecords)
	}
}

// ─── Strategy implementations ─────────────────────────────────────────────────

func (s *Survivorship) preferSource(fieldName string, rule *SurvivorshipRule, recs []*RawPortfolioRecord) SurvivorshipResult {
	// Build source → record map (keep only non-empty values)
	bySource := make(map[string]*RawPortfolioRecord)
	for _, r := range recs {
		if v := r.Fields[fieldName]; v != "" {
			bySource[r.SourceSystem] = r
		}
	}

	var rejected []RejectedSource

	for _, preferred := range rule.PreferredSources {
		rec, found := bySource[preferred]
		if !found {
			continue
		}
		// Collect rejected alternatives
		for _, other := range recs {
			if other.SourceSystem == preferred {
				continue
			}
			if v := other.Fields[fieldName]; v != "" {
				rejected = append(rejected, RejectedSource{
					Source: other.SourceSystem,
					Value:  v,
					Reason: fmt.Sprintf("lower priority than %s", preferred),
				})
			}
		}
		return SurvivorshipResult{
			FieldName:       fieldName,
			ChosenValue:     rec.Fields[fieldName],
			ChosenSource:    preferred,
			Strategy:        "prefer_source",
			RejectedSources: rejected,
		}
	}

	// None of the preferred sources had a value; fall back
	return s.highestQualityFallback(fieldName, recs)
}

func (s *Survivorship) earliestNonNull(fieldName string, rule *SurvivorshipRule, recs []*RawPortfolioRecord) SurvivorshipResult {
	var (
		best     *RawPortfolioRecord
		bestTime time.Time
		rejected []RejectedSource
	)
	for _, r := range recs {
		v := r.Fields[fieldName]
		if v == "" {
			continue
		}
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			// Not a date; try the record effective date as proxy
			t = r.EffectiveDate
		}
		if best == nil || t.Before(bestTime) {
			if best != nil {
				rejected = append(rejected, RejectedSource{
					Source: best.SourceSystem,
					Value:  best.Fields[fieldName],
					Reason: "later date than chosen record",
				})
			}
			best = r
			bestTime = t
		} else {
			ts := strconv.FormatInt(t.UnixMilli(), 10)
			rejected = append(rejected, RejectedSource{
				Source: r.SourceSystem,
				Value:  v,
				Reason: "later date (" + ts + ")",
			})
		}
	}
	if best == nil {
		return SurvivorshipResult{FieldName: fieldName, Strategy: "earliest_non_null"}
	}
	return SurvivorshipResult{
		FieldName:       fieldName,
		ChosenValue:     best.Fields[fieldName],
		ChosenSource:    best.SourceSystem,
		Strategy:        "earliest_non_null",
		RejectedSources: rejected,
	}
}

func (s *Survivorship) latestBy(fieldName string, rule *SurvivorshipRule, recs []*RawPortfolioRecord) SurvivorshipResult {
	var (
		best     *RawPortfolioRecord
		bestTime time.Time
		rejected []RejectedSource
	)
	for _, r := range recs {
		v := r.Fields[fieldName]
		if v == "" {
			continue
		}
		t := r.EffectiveDate
		if best == nil || t.After(bestTime) {
			if best != nil {
				rejected = append(rejected, RejectedSource{
					Source: best.SourceSystem,
					Value:  best.Fields[fieldName],
					Reason: "earlier effective date",
				})
			}
			best = r
			bestTime = t
		} else {
			rejected = append(rejected, RejectedSource{
				Source: r.SourceSystem,
				Value:  v,
				Reason: "earlier effective date",
			})
		}
	}
	if best == nil {
		return SurvivorshipResult{FieldName: fieldName, Strategy: "latest_by"}
	}
	return SurvivorshipResult{
		FieldName:       fieldName,
		ChosenValue:     best.Fields[fieldName],
		ChosenSource:    best.SourceSystem,
		Strategy:        "latest_by",
		RejectedSources: rejected,
	}
}

func (s *Survivorship) highestQualityFallback(fieldName string, recs []*RawPortfolioRecord) SurvivorshipResult {
	var (
		best     *RawPortfolioRecord
		rejected []RejectedSource
	)
	for _, r := range recs {
		if r.Fields[fieldName] == "" {
			continue
		}
		if best == nil || r.QualityScore > best.QualityScore {
			if best != nil {
				rejected = append(rejected, RejectedSource{
					Source: best.SourceSystem,
					Value:  best.Fields[fieldName],
					Reason: fmt.Sprintf("lower quality score (%d)", best.QualityScore),
				})
			}
			best = r
		} else {
			rejected = append(rejected, RejectedSource{
				Source: r.SourceSystem,
				Value:  r.Fields[fieldName],
				Reason: fmt.Sprintf("lower quality score (%d)", r.QualityScore),
			})
		}
	}
	if best == nil {
		return SurvivorshipResult{FieldName: fieldName, Strategy: "highest_quality"}
	}
	return SurvivorshipResult{
		FieldName:       fieldName,
		ChosenValue:     best.Fields[fieldName],
		ChosenSource:    best.SourceSystem,
		Strategy:        "highest_quality",
		RejectedSources: rejected,
	}
}
