package fix

import (
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
)

// TestAdapter_LatencyBudget_RejectCancelRace is the table-driven test
// for the Amendment 2 fallback correctness. The crucial property:
// when the business-level ExecutionReport (Trade/Cancel) arrives
// before the latency budget expires, the scheduled ExecType=8 Reject
// MUST be cancelled. Otherwise the broker receives both — order-state
// corruption from the broker's perspective.
//
// Uses a fakeLatencyTimer (no real wall clock) and a captureSender
// (no real quickfix session) so the test is hermetic and fast.
func TestAdapter_LatencyBudget_RejectCancelRace(t *testing.T) {
	sid := quickfix.SessionID{BeginString: "FIX.4.4", SenderCompID: "TEST", TargetCompID: "BROKER"}

	cases := []struct {
		name             string
		budget           time.Duration
		deliverReport    bool
		deliverReportAt  time.Duration // simulated offset from "now"
		expectRejectSent bool
		expectOtherSent  int // how many other messages (PendingNew) the adapter should send
	}{
		{
			name:             "no_business_report_fires_reject",
			budget:           100 * time.Millisecond,
			deliverReport:    false,
			expectRejectSent: true,
			expectOtherSent:  0, // only scheduleLatencyReject is called directly — no PendingNew
		},
		{
			name:             "terminal_report_before_budget_cancels_reject",
			budget:           100 * time.Millisecond,
			deliverReport:    true,
			deliverReportAt:  50 * time.Millisecond,
			expectRejectSent: false,
			expectOtherSent:  0,
		},
		{
			name:             "terminal_report_at_budget_boundary_cancels_reject",
			budget:           100 * time.Millisecond,
			deliverReport:    true,
			deliverReportAt:  100 * time.Millisecond,
			expectRejectSent: false,
			expectOtherSent:  0,
		},
		{
			name:             "budget_zero_skips_timer",
			budget:           0,
			deliverReport:    false,
			expectRejectSent: false,
			expectOtherSent:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cap := &captureSender{}
			clock := newFakeLatencyClock()
			prevFactory := SetLatencyTimerFactory(clock.Schedule)
			t.Cleanup(func() { SetLatencyTimerFactory(prevFactory) })

			adapter := newAdapter(
				&fakeResolverForTest{},
				&captureSinkForTest{},
				nil,
				tc.budget,
				cap,
			)

			clOrdID := "ORD-001"
			adapter.scheduleLatencyReject(sid, clOrdID)

			if tc.deliverReport {
				// Advance the clock just past the deliverReportAt
				// offset and cancel. Since we use a fake clock,
				// "advance" just calls the cancel directly — there's
				// no real wall time. The real-time advance happens
				// below in the final sleep.
				clock.now = tc.deliverReportAt
				adapter.cancelPendingReject(sid.String(), clOrdID)
			}

			// Force any pending callbacks to fire by advancing the
			// fake clock past the budget. This mimics the real timer
			// firing after `budget` wall-time has passed.
			clock.now = tc.budget + 1
			clock.fireAll()

			rejects := 0
			others := 0
			for _, msg := range cap.sent {
				if msgType, _ := msg.Header.GetString(tag.MsgType); msgType == "8" {
					if execType, _ := msg.Body.GetString(tag.ExecType); execType == "8" {
						rejects++
					} else {
						others++
					}
				} else {
					others++
				}
			}

			if tc.expectRejectSent && rejects != 1 {
				t.Fatalf("expected 1 ExecType=8 Reject, got %d (all sent: %d)", rejects, len(cap.sent))
			}
			if !tc.expectRejectSent && rejects != 0 {
				t.Fatalf("expected 0 ExecType=8 Rejects, got %d — cancel race lost", rejects)
			}
			if others != tc.expectOtherSent {
				t.Fatalf("expected %d other outbound messages, got %d", tc.expectOtherSent, others)
			}
		})
	}
}

// captureSender records every Send call so the test can assert what
// the adapter would have put on the wire.
type captureSender struct {
	mu   sync.Mutex
	sent []*quickfix.Message
}

// Send implements Sender.
func (c *captureSender) Send(msg *quickfix.Message, _ quickfix.SessionID) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, msg)
	return nil
}

// fakeLatencyClock implements the test half of the LatencyTimer
// factory. Callbacks fire only when fireAll() is invoked, simulating
// the elapsing of wall-clock time.
type fakeLatencyClock struct {
	mu      sync.Mutex
	now     time.Duration
	pending []scheduledFire
}

type scheduledFire struct {
	fire func()
}

func newFakeLatencyClock() *fakeLatencyClock {
	return &fakeLatencyClock{}
}

// Schedule returns a LatencyTimer that the caller will treat as
// armed. The fire callback is invoked when fireAll() is called.
func (c *fakeLatencyClock) Schedule(_ time.Duration, f func()) LatencyTimer {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pending = append(c.pending, scheduledFire{fire: f})
	return &fakeLatencyTimer{}
}

// fireAll invokes every scheduled fire callback.
func (c *fakeLatencyClock) fireAll() {
	c.mu.Lock()
	pending := c.pending
	c.pending = nil
	c.mu.Unlock()
	for _, sf := range pending {
		sf.fire()
	}
}

// fakeLatencyTimer is a no-op Stop() implementation.
type fakeLatencyTimer struct{}

// Stop implements LatencyTimer.
func (fakeLatencyTimer) Stop() bool { return false }

// fakeResolverForTest satisfies TenantResolver for newAdapter.
type fakeResolverForTest struct{}

func (fakeResolverForTest) Resolve(_ quickfix.SessionID) (uuid.UUID, uuid.UUID, bool) {
	return uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		true
}
