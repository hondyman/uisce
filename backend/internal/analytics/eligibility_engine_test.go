package analytics

import (
	"testing"
)

func TestPickPreAgg_ExactMatch(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-1",
			Priority: 100,
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)", "SUM(revenue)"},
			Status:   "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country", "date"},
		Measures: []string{"COUNT(*)"},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg")
	}
	if result.ID != "preagg-1" {
		t.Errorf("Expected preagg-1, got %s", result.ID)
	}
}

func TestPickPreAgg_SubsetMatch(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-wide",
			Priority: 100,
			GroupBy:  []string{"country", "date", "region", "city"},
			Measures: []string{"COUNT(*)", "SUM(revenue)", "AVG(price)"},
			Status:   "active",
		},
	}

	// Query asks for subset of columns and measures
	q := &SemanticQuery{
		GroupBy:  []string{"country", "date"},
		Measures: []string{"SUM(revenue)"},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg for subset query")
	}
	if result.ID != "preagg-wide" {
		t.Errorf("Expected preagg-wide, got %s", result.ID)
	}
}

func TestPickPreAgg_NoMatch_MissingGroupBy(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-1",
			Priority: 100,
			GroupBy:  []string{"country"},
			Measures: []string{"COUNT(*)"},
			Status:   "active",
		},
	}

	// Query asks for a column that's not in the pre-agg
	q := &SemanticQuery{
		GroupBy:  []string{"country", "state"}, // "state" not in pre-agg
		Measures: []string{"COUNT(*)"},
	}

	result := PickPreAgg(q, metas)
	if result != nil {
		t.Error("Expected no match when query has column not in pre-agg")
	}
}

func TestPickPreAgg_NoMatch_MissingMeasure(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-1",
			Priority: 100,
			GroupBy:  []string{"country"},
			Measures: []string{"COUNT(*)"},
			Status:   "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country"},
		Measures: []string{"SUM(revenue)"}, // Not available in pre-agg
	}

	result := PickPreAgg(q, metas)
	if result != nil {
		t.Error("Expected no match when measure not in pre-agg")
	}
}

func TestPickPreAgg_FilterMatch(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:               "preagg-1",
			Priority:         100,
			GroupBy:          []string{"country", "date"},
			Measures:         []string{"COUNT(*)"},
			FiltersSupported: []string{"country", "date", "status"},
			Status:           "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country"},
		Measures: []string{"COUNT(*)"},
		Filters: []EligibilityFilter{
			{Field: "country", Op: "=", Value: "US"},
		},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg with supported filter")
	}
}

func TestPickPreAgg_FilterNoMatch(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:               "preagg-1",
			Priority:         100,
			GroupBy:          []string{"country", "date"},
			Measures:         []string{"COUNT(*)"},
			FiltersSupported: []string{"country"},
			Status:           "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country"},
		Measures: []string{"COUNT(*)"},
		Filters: []EligibilityFilter{
			{Field: "status", Op: "=", Value: "active"}, // "status" not supported
		},
	}

	result := PickPreAgg(q, metas)
	if result != nil {
		t.Error("Expected no match when filter field not supported")
	}
}

func TestPickPreAgg_PrefersMostSpecific(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-wide",
			Priority: 100,
			GroupBy:  []string{"country", "date", "region", "city"},
			Measures: []string{"COUNT(*)", "SUM(revenue)", "AVG(price)"},
			Status:   "active",
		},
		{
			ID:       "preagg-narrow",
			Priority: 100, // Same priority
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)", "SUM(revenue)"},
			Status:   "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country", "date"},
		Measures: []string{"COUNT(*)"},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg")
	}
	// Should prefer the narrower/more specific pre-agg
	if result.ID != "preagg-narrow" {
		t.Errorf("Expected preagg-narrow (more specific), got %s", result.ID)
	}
}

func TestPickPreAgg_PrefersHigherPriority(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-low",
			Priority: 50,
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)"},
			Status:   "active",
		},
		{
			ID:       "preagg-high",
			Priority: 200,
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)"},
			Status:   "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country"},
		Measures: []string{"COUNT(*)"},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg")
	}
	if result.ID != "preagg-high" {
		t.Errorf("Expected preagg-high (higher priority), got %s", result.ID)
	}
}

func TestPickPreAgg_SkipsInactive(t *testing.T) {
	metas := []PreAggMeta{
		{
			ID:       "preagg-draft",
			Priority: 100,
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)"},
			Status:   "draft", // Not active
		},
		{
			ID:       "preagg-active",
			Priority: 50,
			GroupBy:  []string{"country", "date"},
			Measures: []string{"COUNT(*)"},
			Status:   "active",
		},
	}

	q := &SemanticQuery{
		GroupBy:  []string{"country"},
		Measures: []string{"COUNT(*)"},
	}

	result := PickPreAgg(q, metas)
	if result == nil {
		t.Fatal("Expected to find eligible pre-agg")
	}
	// Should skip draft and pick active
	if result.ID != "preagg-active" {
		t.Errorf("Expected preagg-active, got %s", result.ID)
	}
}

func TestIsSubset(t *testing.T) {
	tests := []struct {
		needle   []string
		haystack []string
		expected bool
	}{
		{[]string{}, []string{"a", "b"}, true},
		{[]string{"a"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"a", "b"}, true},
		{[]string{"a", "b"}, []string{"a", "b", "c"}, true},
		{[]string{"a", "c"}, []string{"a", "b"}, false},
		{[]string{"x"}, []string{"a", "b"}, false},
		{[]string{"a"}, []string{}, false},
	}

	for _, tt := range tests {
		result := isSubset(tt.needle, tt.haystack)
		if result != tt.expected {
			t.Errorf("isSubset(%v, %v) = %v, expected %v", tt.needle, tt.haystack, result, tt.expected)
		}
	}
}

func TestNormalizeColumn(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"country", "country"},
		{"Country", "country"},
		{"  country  ", "country"},
		{"orders.country", "country"},
		{"public.orders.country", "country"},
	}

	for _, tt := range tests {
		result := normalizeColumn(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeColumn(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
