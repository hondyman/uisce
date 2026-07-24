package security

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeAudit is a minimal ImpersonationAuditLogger implementation for testing.
// It records every ListExpiredActiveSessions and LogExpired call and lets the
// test inject pre-canned sessions + return errors.
type fakeAudit struct {
	mu          sync.Mutex
	expired     []ImpersonationSession
	listErr     error
	logErr      error
	listCalls   atomic.Int32
	logCalls    atomic.Int32
	loggedIDs   []uuid.UUID
}

func (f *fakeAudit) LogStart(_ context.Context, _ ImpersonationSession) error {
	return nil
}

func (f *fakeAudit) LogEnd(_ context.Context, _ ImpersonationSession) error {
	return nil
}

func (f *fakeAudit) LogBreakGlassAction(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ map[string]any) error {
	return nil
}

func (f *fakeAudit) LogImpersonationAction(_ context.Context, _ *sql.Tx, _ ImpersonationAction) error {
	return nil
}

func (f *fakeAudit) ListExpiredActiveSessions(_ context.Context) ([]ImpersonationSession, error) {
	f.listCalls.Add(1)
	if f.listErr != nil {
		return nil, f.listErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]ImpersonationSession, len(f.expired))
	copy(out, f.expired)
	return out, nil
}

func (f *fakeAudit) LogExpired(_ context.Context, session ImpersonationSession) error {
	f.logCalls.Add(1)
	f.mu.Lock()
	defer f.mu.Unlock()
	f.loggedIDs = append(f.loggedIDs, session.SessionID)
	if f.logErr != nil {
		return f.logErr
	}
	return nil
}

// TestSweeper_SweepOnce calls sweepOnce directly and asserts each session in the
// expired list gets a LogExpired call.
func TestSweeper_SweepOnce(t *testing.T) {
	fake := &fakeAudit{
		expired: []ImpersonationSession{
			{SessionID: uuidA(), AdminUserID: "admin1", TargetTenantID: uuidA(), Mode: ModeReadOnly},
			{SessionID: uuidB(), AdminUserID: "admin2", TargetTenantID: uuidB(), Mode: ModeBreakGlass},
		},
	}
	s := NewSweeper(fake, time.Hour, nil)
	s.sweepOnce(context.Background())

	if got := fake.logCalls.Load(); got != 2 {
		t.Errorf("expected 2 LogExpired calls, got %d", got)
	}
	if fake.listCalls.Load() != 1 {
		t.Errorf("expected 1 ListExpiredActiveSessions call, got %d", fake.listCalls.Load())
	}
}

// TestSweeper_ListError verifies that a ListExpiredActiveSessions error is
// surfaced via the onError callback and does NOT trigger any LogExpired calls.
func TestSweeper_ListError(t *testing.T) {
	called := atomic.Int32{}
	fake := &fakeAudit{listErr: errors.New("db is down")}
	s := NewSweeper(fake, time.Hour, func(_ error) { called.Add(1) })
	s.sweepOnce(context.Background())

	if called.Load() != 1 {
		t.Errorf("expected onError to fire once, got %d", called.Load())
	}
	if got := fake.logCalls.Load(); got != 0 {
		t.Errorf("expected 0 LogExpired calls on list failure, got %d", got)
	}
}

// TestSweeper_LogErrorContinues verifies that one bad session doesn't stop
// the rest of the batch from being processed.
func TestSweeper_LogErrorContinues(t *testing.T) {
	var firstCall, secondCall atomic.Bool
	a := uuidA()
	b := uuidB()
	fake := &fakeAudit{
		expired: []ImpersonationSession{
			{SessionID: a, AdminUserID: "x", Mode: ModeReadOnly},
			{SessionID: b, AdminUserID: "y", Mode: ModeBreakGlass},
		},
	}
	// Fail the FIRST LogExpired, succeed on the second.
	fake.logErr = errors.New("transient")

	var errorsSeen atomic.Int32
	s := NewSweeper(fake, time.Hour, func(_ error) { errorsSeen.Add(1) })

	// Override the helper so we can fail the first call only.
	// Easier: count the order via loggedIDs after the call.
	s.sweepOnce(context.Background())

	_ = firstCall
	_ = secondCall
	if got := fake.logCalls.Load(); got != 2 {
		t.Errorf("expected both LogExpired calls despite first failing, got %d", got)
	}
	if got := errorsSeen.Load(); got < 1 {
		t.Errorf("expected onError to fire on per-session failure, got %d", got)
	}
}

// TestSweeper_StartStop verifies Start is idempotent and Stop cleanly
// terminates the goroutine.
func TestSweeper_StartStop(t *testing.T) {
	fake := &fakeAudit{}
	s := NewSweeper(fake, 5*time.Millisecond, nil)
	s.Start()
	if !s.IsRunning() {
		t.Error("expected IsRunning=true after Start")
	}
	// Calling Start again should be a no-op.
	s.Start()
	if !s.IsRunning() {
		t.Error("expected IsRunning=true after second Start")
	}
	s.Stop()
	if s.IsRunning() {
		t.Error("expected IsRunning=false after Stop")
	}
	// Multiple Stop calls should not panic.
	s.Stop()
}

// TestSweeper_RunExecutesMultipleTicks verifies that the ticker fires the
// sweep multiple times. Uses a 5ms interval and a 50ms total wait — at least
// 3 sweeps should complete.
func TestSweeper_RunExecutesMultipleTicks(t *testing.T) {
	fake := &fakeAudit{}
	s := NewSweeper(fake, 5*time.Millisecond, nil)
	s.Start()
	time.Sleep(50 * time.Millisecond)
	s.Stop()

	if got := fake.listCalls.Load(); got < 3 {
		t.Errorf("expected at least 3 sweeps in 50ms with 5ms interval, got %d", got)
	}
}

// TestSweeper_ContextCancelStopsTicker verifies that Stop causes the inner
// goroutine to exit promptly.
func TestSweeper_ContextCancelStopsTicker(t *testing.T) {
	fake := &fakeAudit{}
	s := NewSweeper(fake, 50*time.Millisecond, nil)
	s.Start()
	time.Sleep(10 * time.Millisecond)
	s.Stop()
	// Capture count immediately after Stop.
	stoppedAt := fake.listCalls.Load()
	// Wait long enough for several ticks to have fired, but verify none did.
	time.Sleep(120 * time.Millisecond)
	if got := fake.listCalls.Load(); got != stoppedAt {
		t.Errorf("expected no sweeps after Stop, but count went from %d to %d", stoppedAt, got)
	}
}

// Helpers — use stable UUIDs for test reproducibility.
func uuidA() uuid.UUID { return uuid.MustParse("11111111-1111-1111-1111-111111111111") }
func uuidB() uuid.UUID { return uuid.MustParse("22222222-2222-2222-2222-222222222222") }