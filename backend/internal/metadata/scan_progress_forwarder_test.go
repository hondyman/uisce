package metadata

import (
	"context"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/models"
)

func TestOverallPercent_PhasesOwnDisjointSlices(t *testing.T) {
	if got := overallPercent("scanning", 0); got != 5 {
		t.Errorf("scanning 0%% = %v; want 5", got)
	}
	if got := overallPercent("scanning", 100); got != 55 {
		t.Errorf("scanning 100%% = %v; want 55", got)
	}
	if got := overallPercent("storing", 50); got != 67.5 {
		t.Errorf("storing 50%% = %v; want 67.5", got)
	}
	if got := overallPercent("nonsense", 50); got != -1 {
		t.Errorf("unknown phase = %v; want -1", got)
	}
	if got := overallPercent("scanning", 250); got != 55 {
		t.Errorf("out-of-range percent not clamped: %v", got)
	}
}

func collect(out <-chan models.ScanProgress, n int, within time.Duration) []models.ScanProgress {
	var got []models.ScanProgress
	deadline := time.After(within)
	for len(got) < n {
		select {
		case p := <-out:
			got = append(got, p)
		case <-deadline:
			return got
		}
	}
	return got
}

// A phase that reports 0% (as "scanning" and "storing" used to) still moves the overall bar, and the bar never
// goes backwards when a later phase starts at 0.
func TestForwardScanProgress_MonotonicAndScaled(t *testing.T) {
	in, out := make(chan models.ScanProgress), make(chan models.ScanProgress, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go forwardScanProgress(ctx, in, out, 10, 40, time.Hour) // this datasource owns 10..50

	for _, p := range []models.ScanProgress{
		{Phase: "connecting", Percent: 0},
		{Phase: "scanning", Percent: 0},
		{Phase: "scanning", Percent: 50, Message: "half"},
		{Phase: "scanning", Percent: 20}, // a stale, lower report must not move the bar back
		{Phase: "storing", Percent: 0},
		{Phase: "calibrating", Percent: 100},
	} {
		in <- p
	}
	got := collect(out, 6, time.Second)
	if len(got) != 6 {
		t.Fatalf("got %d events; want 6", len(got))
	}
	prev := 0.0
	for i, p := range got {
		if p.Percent < prev {
			t.Errorf("event %d moved backwards: %v after %v", i, p.Percent, prev)
		}
		if p.Percent < 10 || p.Percent > 50 {
			t.Errorf("event %d percent %v outside this datasource's 10..50", i, p.Percent)
		}
		prev = p.Percent
	}
	if got[1].Percent <= got[0].Percent {
		t.Errorf("entering the scanning phase should already advance the bar: %v then %v", got[0].Percent, got[1].Percent)
	}
}

// While one step runs long, the latest status is re-sent as a heartbeat with the elapsed time.
func TestForwardScanProgress_HeartbeatWhileIdle(t *testing.T) {
	in, out := make(chan models.ScanProgress), make(chan models.ScanProgress, 16)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go forwardScanProgress(ctx, in, out, 0, 100, 20*time.Millisecond)

	in <- models.ScanProgress{Phase: "scanning", Percent: 40, Message: "Reading foreign keys"}
	got := collect(out, 4, time.Second)
	if len(got) < 3 {
		t.Fatalf("got %d events; want the event plus heartbeats", len(got))
	}
	if got[0].Heartbeat {
		t.Error("the real event must not be marked as a heartbeat")
	}
	for _, hb := range got[1:] {
		if !hb.Heartbeat || hb.Message != "Reading foreign keys" || hb.Percent != got[0].Percent {
			t.Errorf("heartbeat should repeat the latest status unchanged: %+v", hb)
		}
	}
}

func TestForwardScanProgress_NoHeartbeatAfterTerminalAndStopsWhenInputCloses(t *testing.T) {
	in, out := make(chan models.ScanProgress), make(chan models.ScanProgress, 16)
	done := make(chan struct{})
	go func() { forwardScanProgress(context.Background(), in, out, 0, 100, 10*time.Millisecond); close(done) }()

	in <- models.ScanProgress{Phase: "complete", Percent: 100}
	time.Sleep(60 * time.Millisecond)
	close(in)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("forwarder did not stop when its input closed")
	}
	got := collect(out, 5, 20*time.Millisecond)
	if len(got) != 1 || got[0].Heartbeat {
		t.Errorf("expected exactly the terminal event, got %+v", got)
	}
}
