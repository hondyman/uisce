package finops

import "time"

// Clock abstracts wall-clock access so production code uses RealClock and
// tests can inject a deterministic time without monkey-patching package globals.
//
// All Predictive FinOps code paths that need "now" must read it through a
// Clock — never call time.Now() directly. This is the prerequisite for
// property tests and the B4 nightly scheduler.
type Clock interface {
	Now() time.Time
}

// RealClock is the production Clock. It passes through to time.Now().
type RealClock struct{}

// Now returns the current wall-clock time.
func (RealClock) Now() time.Time { return time.Now() }

// FixedClock returns a constant time on every Now() call. Use in tests to
// pin "today" to a deterministic date.
type FixedClock struct {
	T time.Time
}

// Now returns the configured fixed time.
func (f FixedClock) Now() time.Time { return f.T }
