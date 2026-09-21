package metadata

import (
	"context"
	"time"

	"github.com/hondyman/uisce/backend/models"
)

// scanHeartbeatInterval is how long a scan may go without an event before the latest status is re-sent.
var scanHeartbeatInterval = 5 * time.Second

// scanPhaseRange is the slice of one datasource's 0-100 progress that a phase owns. Phases report their own
// 0-100 (or nothing), which used to leave the bar at 0% for the whole scan.
type scanPhaseRange struct{ lo, hi float64 }

var scanPhaseRanges = map[string]scanPhaseRange{
	"connecting":  {0, 5},
	"scanning":    {5, 55},
	"storing":     {55, 80},
	"enriching":   {80, 85},
	"calibrating": {85, 98},
}

// overallPercent maps a phase's own percent (0-100) into the datasource's 0-100. An unknown phase returns -1.
func overallPercent(phase string, pct float64) float64 {
	r, ok := scanPhaseRanges[phase]
	if !ok {
		return -1
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return r.lo + (r.hi-r.lo)*pct/100
}

// forwardScanProgress copies events from in to out until in is closed, and:
//   - re-maps each phase's percent into its own slice of [base, base+span], never moving backwards;
//   - while nothing arrives for `heartbeat`, re-sends the latest status marked Heartbeat with the elapsed time,
//     so a long query (or a dead stream) is visible instead of a frozen bar.
//
// Terminal events (complete, error) and events for unknown phases pass through with the percent kept.
func forwardScanProgress(ctx context.Context, in <-chan models.ScanProgress, out chan<- models.ScanProgress, base, span float64, heartbeat time.Duration) {
	start := time.Now()
	var last models.ScanProgress
	haveLast := false
	high := base

	// After a cancel or a dead consumer, keep draining so the scan (which sends without a select) never blocks
	// on a full channel; the caller closes `in` when the scan returns.
	drain := func() {
		for range in {
		}
	}
	send := func(p models.ScanProgress) bool {
		select {
		case out <- p:
			return true
		case <-ctx.Done():
			return false
		}
	}
	ticker := time.NewTicker(heartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			drain()
			return
		case p, ok := <-in:
			if !ok {
				return
			}
			if rel := overallPercent(p.Phase, p.Percent); rel >= 0 {
				p.Percent = base + span*rel/100
			}
			if p.Phase != "error" && p.Percent < high {
				p.Percent = high
			}
			if p.Percent > high {
				high = p.Percent
			}
			p.Heartbeat = false
			p.ElapsedSeconds = int(time.Since(start).Seconds())
			last, haveLast = p, true
			ticker.Reset(heartbeat)
			if !send(p) {
				drain()
				return
			}
		case <-ticker.C:
			if !haveLast || last.Phase == "complete" || last.Phase == "error" {
				continue
			}
			hb := last
			hb.Heartbeat = true
			hb.ElapsedSeconds = int(time.Since(start).Seconds())
			if !send(hb) {
				drain()
				return
			}
		}
	}
}
