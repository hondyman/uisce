package security

import (
	"context"
	"sync"
	"time"
)

// Sweeper periodically scans platform_admin_audit for active impersonation
// sessions whose expires_at has passed, and writes an IMPERSONATION_EXPIRED row
// for each. This closes the audit gap when a client goes offline without
// calling DELETE /api/admin/impersonate/{sessionId}.
//
// Design:
//   - One Sweeper instance per process. Construct with NewSweeper and call Start.
//   - Stop() cleanly cancels the underlying context. Safe to call multiple times.
//   - Runs every interval (default 60s). Each tick is independent; a slow DB
//     call does not block the next tick.
//   - Errors are logged via the optional onError callback (defaults to silent
//     because expired-session reconciliation is a best-effort background task
//     and a transient DB hiccup should not page anyone).
//
// Concurrency: a mutex protects Stop/start. The goroutine itself holds no
// state between ticks beyond the cancellation flag.
type Sweeper struct {
	audit    ImpersonationAuditLogger
	interval time.Duration
	onError  func(error)

	mu     sync.Mutex
	cancel context.CancelFunc
	running bool
}

// NewSweeper builds a sweeper. interval may be 0 (default 60s). onError may be
// nil (silent error handling).
func NewSweeper(audit ImpersonationAuditLogger, interval time.Duration, onError func(error)) *Sweeper {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	if onError == nil {
		onError = func(err error) { /* silent */ }
	}
	return &Sweeper{audit: audit, interval: interval, onError: onError}
}

// Start launches the background goroutine. Idempotent — calling Start on an
// already-running sweeper is a no-op.
func (s *Sweeper) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running = true
	go s.run(ctx)
}

// Stop signals the goroutine to exit and waits for it to finish (bounded by the
// goroutine exiting at the next select check). Safe to call multiple times.
func (s *Sweeper) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
}

// IsRunning reports whether the sweeper goroutine is active.
func (s *Sweeper) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Sweeper) run(ctx context.Context) {
	// Run an initial sweep immediately on Start so a freshly-restarted server
	// catches up on any sessions that expired while it was down. Subsequent
	// sweeps run on the interval.
	s.sweepOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepOnce(ctx)
		}
	}
}

// sweepOnce performs a single sweep. Extracted for testability.
func (s *Sweeper) sweepOnce(parent context.Context) {
	// Use a bounded timeout so a stuck DB query doesn't keep the goroutine
	// busy indefinitely. 30s is generous; typical sweeps complete in <100ms.
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	sessions, err := s.audit.ListExpiredActiveSessions(ctx)
	if err != nil {
		s.onError(err)
		return
	}

	for _, session := range sessions {
		// Don't cascade: a per-session error should not stop the rest of the batch.
		if err := s.audit.LogExpired(ctx, session); err != nil {
			s.onError(err)
		}
	}
}