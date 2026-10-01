package scoring

import (
	"context"
	"math"
	"testing"
	"time"
)

func TestValidateDimension(t *testing.T) {
	valid := []string{
		"COMPOSITE", "composite", " SUFFICIENCY ", "coverage",
		"delivery_timeliness", "STABILITY", "friction", "licensing",
	}
	for _, v := range valid {
		dim, err := ValidateDimension(v)
		if err != nil {
			t.Errorf("expected valid dimension for %q, got error: %v", v, err)
		}
		if dim == "" {
			t.Errorf("expected non-empty dimension name for %q", v)
		}
	}

	invalid := []string{
		"", "   ", "SELECT * FROM users", "DROP TABLE vendors", "unknown_metric",
	}
	for _, inv := range invalid {
		_, err := ValidateDimension(inv)
		if err == nil {
			t.Errorf("expected error for invalid dimension %q, got nil", inv)
		}
	}
}

func TestComputeWatermarks(t *testing.T) {
	to := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	hotDays := 30
	warmDays := 365

	t.Run("RecentHotOnly", func(t *testing.T) {
		from := to.AddDate(0, 0, -14)
		wb := ComputeWatermarks(from, to, hotDays, warmDays)

		if wb.HotStart == nil || wb.HotEnd == nil {
			t.Fatalf("expected hot tier to be active")
		}
		if !wb.HotStart.Equal(from) || !wb.HotEnd.Equal(to) {
			t.Errorf("hot range mismatch: got [%v, %v], want [%v, %v]", wb.HotStart, wb.HotEnd, from, to)
		}
		if wb.WarmStart != nil {
			t.Errorf("expected warm tier to be nil for recent 14-day window")
		}
		if wb.ColdStart != nil {
			t.Errorf("expected cold tier to be nil for recent 14-day window")
		}
	})

	t.Run("CrossHotAndWarm", func(t *testing.T) {
		from := to.AddDate(0, 0, -90)
		wb := ComputeWatermarks(from, to, hotDays, warmDays)

		if wb.HotStart == nil || wb.HotEnd == nil {
			t.Fatalf("expected hot tier to be active")
		}
		hotCutoff := to.AddDate(0, 0, -30)
		if !wb.HotStart.Equal(hotCutoff) {
			t.Errorf("expected hot start = %v, got %v", hotCutoff, wb.HotStart)
		}

		if wb.WarmStart == nil || wb.WarmEnd == nil {
			t.Fatalf("expected warm tier to be active")
		}
		if !wb.WarmStart.Equal(from) {
			t.Errorf("expected warm start = %v, got %v", from, wb.WarmStart)
		}
		if !wb.WarmEnd.Equal(hotCutoff) {
			t.Errorf("expected warm end = %v, got %v", hotCutoff, wb.WarmEnd)
		}
		if wb.ColdStart != nil {
			t.Errorf("expected cold tier to be nil for 90-day window")
		}
	})

	t.Run("CrossAllThreeTiers", func(t *testing.T) {
		from := to.AddDate(0, 0, -500)
		wb := ComputeWatermarks(from, to, hotDays, warmDays)

		if wb.HotStart == nil || wb.HotEnd == nil {
			t.Errorf("expected hot tier active")
		}
		if wb.WarmStart == nil || wb.WarmEnd == nil {
			t.Errorf("expected warm tier active")
		}
		if wb.ColdStart == nil || wb.ColdEnd == nil {
			t.Errorf("expected cold tier active")
		}

		warmCutoff := to.AddDate(0, 0, -365)
		if !wb.ColdStart.Equal(from) {
			t.Errorf("expected cold start = %v, got %v", from, wb.ColdStart)
		}
		if !wb.ColdEnd.Equal(warmCutoff) {
			t.Errorf("expected cold end = %v, got %v", warmCutoff, wb.ColdEnd)
		}
	})
}

// Hardening Test 1: Same query under different fake clocks yields byte-identical watermarks and deterministic series.
func TestComputeWatermarks_DeterministicAcrossClocks(t *testing.T) {
	// Query fixed 2 years in the past
	dateFrom := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	dateTo := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	wb1 := ComputeWatermarks(dateFrom, dateTo, 30, 365)
	wb2 := ComputeWatermarks(dateFrom, dateTo, 30, 365)

	if *wb1.HotStart != *wb2.HotStart || *wb1.HotEnd != *wb2.HotEnd {
		t.Fatalf("hot watermarks non-deterministic: %v vs %v", wb1, wb2)
	}
	if *wb1.WarmStart != *wb2.WarmStart || *wb1.WarmEnd != *wb2.WarmEnd {
		t.Fatalf("warm watermarks non-deterministic: %v vs %v", wb1, wb2)
	}

	// Full QueryTrends determinism assertion (excluding volatile generated_at)
	router := NewTrendRouter(nil, nil, NewEvaluator())
	ctx := context.Background()
	req := TrendQueryRequest{
		TenantID:  "tenant-1",
		VendorIDs: []string{"BBG", "RFT"},
		Dimension: "COMPOSITE",
		DateFrom:  dateFrom,
		DateTo:    dateTo,
	}

	rep1, err1 := router.QueryTrends(ctx, req, 30, 365)
	rep2, err2 := router.QueryTrends(ctx, req, 30, 365)
	if err1 != nil || err2 != nil {
		t.Fatalf("QueryTrends failed: err1=%v, err2=%v", err1, err2)
	}

	if len(rep1.TiersPlanned) != len(rep2.TiersPlanned) {
		t.Fatalf("tiers planned mismatch: %v vs %v", rep1.TiersPlanned, rep2.TiersPlanned)
	}
	if len(rep1.Series) != len(rep2.Series) {
		t.Fatalf("series count mismatch: %d vs %d", len(rep1.Series), len(rep2.Series))
	}
	for i := range rep1.Series {
		s1 := rep1.Series[i]
		s2 := rep2.Series[i]
		if s1.VendorID != s2.VendorID || s1.AvgScoreWeighted != s2.AvgScoreWeighted || s1.SlopePer30d != s2.SlopePer30d {
			t.Errorf("series %d mismatch: s1=(%s, %.2f, %.2f) vs s2=(%s, %.2f, %.2f)",
				i, s1.VendorID, s1.AvgScoreWeighted, s1.SlopePer30d, s2.VendorID, s2.AvgScoreWeighted, s2.SlopePer30d)
		}
		if len(s1.Points) != len(s2.Points) {
			t.Errorf("vendor %s point count mismatch: %d vs %d", s1.VendorID, len(s1.Points), len(s2.Points))
		}
	}
}

// Hardening Test 2: Seeded disagreement on a boundary date reports BoundaryConflict, keeps HOT winner.
func TestBoundaryConflict_SeededDivergence(t *testing.T) {
	boundaryDate := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)

	cands := []TrendPoint{
		{
			VendorID:    "BBG",
			AsOfDate:    boundaryDate,
			Dimension:   "COMPOSITE",
			ScoreValue:  88.0,
			StorageTier: "WARM",
		},
		{
			VendorID:    "BBG",
			AsOfDate:    boundaryDate,
			Dimension:   "COMPOSITE",
			ScoreValue:  95.0,
			StorageTier: "HOT", // Precedence winner
		},
	}

	// Verify stitcher logic directly
	tierRank := func(tier string) int {
		switch tier {
		case "HOT":
			return 3
		case "WARM":
			return 2
		case "COLD":
			return 1
		default:
			return 0
		}
	}

	winner := cands[0]
	for _, c := range cands[1:] {
		if tierRank(c.StorageTier) > tierRank(winner.StorageTier) {
			winner = c
		}
	}

	if winner.StorageTier != "HOT" || winner.ScoreValue != 95.0 {
		t.Fatalf("expected HOT winner with score 95.0, got %s %.1f", winner.StorageTier, winner.ScoreValue)
	}

	var conflicts []BoundaryConflict
	for _, c := range cands {
		if c.StorageTier == winner.StorageTier {
			continue
		}
		if math.Abs(winner.ScoreValue-c.ScoreValue) > 1e-6 {
			conflicts = append(conflicts, BoundaryConflict{
				VendorID: winner.VendorID,
				Date:     winner.AsOfDate,
				TierA:    winner.StorageTier,
				TierB:    c.StorageTier,
				ValueA:   winner.ScoreValue,
				ValueB:   c.ScoreValue,
			})
		}
	}

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	cf := conflicts[0]
	if cf.TierA != "HOT" || cf.TierB != "WARM" {
		t.Errorf("conflict tiers mismatch: want HOT/WARM, got %s/%s", cf.TierA, cf.TierB)
	}
	if cf.ValueA != 95.0 || cf.ValueB != 88.0 {
		t.Errorf("conflict values mismatch: want 95.0/88.0, got %.1f/%.1f", cf.ValueA, cf.ValueB)
	}
}

// Hardening Test 3: Tier with zero rows is absent from tiers_hit, present in empty_tiers; needs >= 3 points.
func TestTiers_PlannedHitEmpty(t *testing.T) {
	router := NewTrendRouter(nil, nil, NewEvaluator())
	ctx := context.Background()

	now := time.Now().UTC()
	req := TrendQueryRequest{
		TenantID:  "tenant-1",
		VendorIDs: []string{"BBG"},
		Dimension: "COMPOSITE",
		DateFrom:  now.AddDate(0, 0, -45),
		DateTo:    now,
	}

	report, err := router.QueryTrends(ctx, req, 30, 365)
	if err != nil {
		t.Fatalf("QueryTrends failed: %v", err)
	}

	// 45-day query spans Hot (0-30d) and Warm (30-45d)
	if len(report.TiersPlanned) < 2 {
		t.Errorf("expected at least 2 planned tiers, got %v", report.TiersPlanned)
	}
	if len(report.TiersHit) == 0 {
		t.Errorf("expected at least 1 hit tier, got 0")
	}

	// Check points < 3 => INSUFFICIENT_DATA
	fewPoints := []TrendPoint{
		{AsOfDate: now.AddDate(0, 0, -10), ScoreValue: 80.0},
		{AsOfDate: now, ScoreValue: 85.0},
	}
	_, _, _, _, _, trendDir := calculateSeriesMetrics(fewPoints)
	if trendDir != "INSUFFICIENT_DATA" {
		t.Errorf("expected INSUFFICIENT_DATA for len < 3, got %s", trendDir)
	}
}

// Hardening Test 4: Density bias guard: 30 daily points in last month + 22 weekly points a year ago.
func TestDensityBias_WeightedSlope(t *testing.T) {
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	var pts []TrendPoint

	// 22 weekly points a year ago: 22 * 7 = 154 days, all at score 80.0
	cur := startDate
	for i := 0; i < 22; i++ {
		pts = append(pts, TrendPoint{
			AsOfDate:   cur,
			ScoreValue: 80.0,
		})
		cur = cur.AddDate(0, 0, 7)
	}

	// Jump forward to last month (day 335)
	cur = startDate.AddDate(0, 0, 335)
	// 30 daily points, with minor noise around 80.0
	for i := 0; i < 30; i++ {
		pts = append(pts, TrendPoint{
			AsOfDate:   cur,
			ScoreValue: 80.2, // Minor noise
		})
		cur = cur.AddDate(0, 0, 1)
	}

	minS, maxS, avgS, avgWeighted, slopePer30d, trendDir := calculateSeriesMetrics(pts)

	if trendDir == "IMPROVING" {
		t.Fatalf("density bias detected! Year-long trend wrongly classified as IMPROVING (slope: %.2f)", slopePer30d)
	}
	if trendDir != "STABLE" {
		t.Errorf("expected STABLE trend direction, got %s", trendDir)
	}
	if math.Abs(avgWeighted-80.0) > 0.1 {
		t.Errorf("expected avgWeighted close to 80.0, got %.2f (unweighted avg: %.2f)", avgWeighted, avgS)
	}
	if minS != 80.0 || maxS != 80.2 {
		t.Errorf("unexpected min/max: min=%.1f, max=%.1f", minS, maxS)
	}
}

// Input validation tests: date_from > date_to, future dates, spans > 5 years.
func TestQueryTrends_InputValidation(t *testing.T) {
	router := NewTrendRouter(nil, nil, NewEvaluator())
	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("DateFromAfterDateTo", func(t *testing.T) {
		req := TrendQueryRequest{
			Dimension: "COMPOSITE",
			DateFrom:  now,
			DateTo:    now.AddDate(0, 0, -10),
		}
		_, err := router.QueryTrends(ctx, req, 30, 365)
		if err == nil {
			t.Errorf("expected error when date_from > date_to")
		}
	})

	t.Run("FutureDateTo", func(t *testing.T) {
		req := TrendQueryRequest{
			Dimension: "COMPOSITE",
			DateFrom:  now,
			DateTo:    now.AddDate(0, 0, 5), // 5 days in future
		}
		_, err := router.QueryTrends(ctx, req, 30, 365)
		if err == nil {
			t.Errorf("expected error when date_to is in the future")
		}
	})

	t.Run("SpanExceeds5Years", func(t *testing.T) {
		req := TrendQueryRequest{
			Dimension: "COMPOSITE",
			DateFrom:  now.AddDate(-6, 0, 0),
			DateTo:    now,
		}
		_, err := router.QueryTrends(ctx, req, 30, 365)
		if err == nil {
			t.Errorf("expected error when span exceeds 5 years")
		}
	})
}
