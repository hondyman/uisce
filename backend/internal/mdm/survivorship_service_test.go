package mdm

import (
	"context"
	"testing"
	"time"
)

func TestMergeToGoldenRecord_SourcePriority(t *testing.T) {
	e := NewSurvivorshipEngine()
	now := time.Now()

	sources := []SourcePayload{
		{SourceID: "REFINITIV", Timestamp: now, Data: map[string]any{"price": 100.0}},
		{SourceID: "BLOOMBERG", Timestamp: now, Data: map[string]any{"price": 101.5}},
		{SourceID: "CRIMS", Timestamp: now, Data: map[string]any{"price": 99.0}},
	}
	rules := map[string]FieldRule{
		"price": {Strategy: "SOURCE_PRIORITY", PriorityOrder: []string{"BLOOMBERG", "REFINITIV"}},
	}

	golden, err := e.MergeToGoldenRecord(context.Background(), sources, rules, now)
	if err != nil {
		t.Fatal(err)
	}

	if got := golden["price"]; got != 101.5 {
		t.Errorf("expected BLOOMBERG price 101.5, got %v", got)
	}
}

func TestMergeToGoldenRecord_MostRecent(t *testing.T) {
	e := NewSurvivorshipEngine()
	base := time.Now()
	sources := []SourcePayload{
		{SourceID: "A", Timestamp: base.Add(-2 * time.Hour), Data: map[string]any{"x": 1.0}},
		{SourceID: "B", Timestamp: base.Add(-1 * time.Hour), Data: map[string]any{"x": 2.0}},
		{SourceID: "C", Timestamp: base, Data: map[string]any{"x": 3.0}},
	}
	rules := map[string]FieldRule{"x": {Strategy: "MOST_RECENT"}}

	golden, err := e.MergeToGoldenRecord(context.Background(), sources, rules, base)
	if err != nil {
		t.Fatal(err)
	}
	if got := golden["x"]; got != 3.0 {
		t.Errorf("expected most-recent x=3.0, got %v", got)
	}
}

func TestMergeToGoldenRecord_ConservativeMinMax(t *testing.T) {
	e := NewSurvivorshipEngine()
	now := time.Now()
	sources := []SourcePayload{
		{SourceID: "A", Timestamp: now, Data: map[string]any{"v": 10.0}},
		{SourceID: "B", Timestamp: now, Data: map[string]any{"v": 20.0}},
		{SourceID: "C", Timestamp: now, Data: map[string]any{"v": 30.0}},
	}
	rulesMin := map[string]FieldRule{"v": {Strategy: "CONSERVATIVE_MIN"}}
	rulesMax := map[string]FieldRule{"v": {Strategy: "CONSERVATIVE_MAX"}}

	g, err := e.MergeToGoldenRecord(context.Background(), sources, rulesMin, now)
	if err != nil {
		t.Fatal(err)
	}
	if g["v"] != 10.0 {
		t.Errorf("expected min v=10.0, got %v", g["v"])
	}

	g, err = e.MergeToGoldenRecord(context.Background(), sources, rulesMax, now)
	if err != nil {
		t.Fatal(err)
	}
	if g["v"] != 30.0 {
		t.Errorf("expected max v=30.0, got %v", g["v"])
	}
}

func TestMergeToGoldenRecord_DefaultStrategy(t *testing.T) {
	e := NewSurvivorshipEngine()
	now := time.Now()
	sources := []SourcePayload{
		{SourceID: "A", Timestamp: now.Add(-time.Minute), Data: map[string]any{"y": 1.0}},
		{SourceID: "B", Timestamp: now, Data: map[string]any{"y": 2.0}},
	}

	golden, err := e.MergeToGoldenRecord(context.Background(), sources, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if golden["y"] != 2.0 {
		t.Errorf("expected default MOST_RECENT to pick y=2.0, got %v", golden["y"])
	}
}

func TestMergeToGoldenRecord_StalenessFilter(t *testing.T) {
	e := NewSurvivorshipEngine()
	now := time.Now()
	sources := []SourcePayload{
		{SourceID: "A", Timestamp: now.Add(-10 * time.Minute), Data: map[string]any{"z": 1.0}},
		{SourceID: "B", Timestamp: now, Data: map[string]any{"z": 2.0}},
	}
	rules := map[string]FieldRule{
		"z": {Strategy: "MOST_RECENT", MaxStaleSeconds: 60},
	}

	golden, err := e.MergeToGoldenRecord(context.Background(), sources, rules, now)
	if err != nil {
		t.Fatal(err)
	}
	if golden["z"] != 2.0 {
		t.Errorf("expected stale source A to be filtered, got z=%v", golden["z"])
	}
}

func TestMergeToGoldenRecord_EmptySources(t *testing.T) {
	e := NewSurvivorshipEngine()
	golden, err := e.MergeToGoldenRecord(context.Background(), nil, nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if golden == nil {
		t.Fatal("expected non-nil empty map")
	}
	if len(golden) != 0 {
		t.Errorf("expected empty map, got %d entries", len(golden))
	}
}

func TestMergeToGoldenRecord_FieldOnlyInOneSource(t *testing.T) {
	e := NewSurvivorshipEngine()
	now := time.Now()
	sources := []SourcePayload{
		{SourceID: "A", Timestamp: now, Data: map[string]any{"a_only": 1.0, "shared": 10.0}},
		{SourceID: "B", Timestamp: now, Data: map[string]any{"b_only": 2.0, "shared": 20.0}},
	}

	golden, err := e.MergeToGoldenRecord(context.Background(), sources, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := golden["a_only"]; !ok {
		t.Error("a_only should be present")
	}
	if _, ok := golden["b_only"]; !ok {
		t.Error("b_only should be present")
	}
}

func TestMergeToGoldenRecord_ContextCancelled(t *testing.T) {
	e := NewSurvivorshipEngine()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := e.MergeToGoldenRecord(ctx, nil, nil, time.Now())
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
