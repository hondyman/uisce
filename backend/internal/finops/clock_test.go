package finops

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestFixedClock_NowIsStable verifies that FixedClock returns the same instant
// on repeated Now() calls. This is the property tests and the B4 nightly scheduler
// depend on for deterministic behavior.
func TestFixedClock_NowIsStable(t *testing.T) {
	pinned := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)
	clock := FixedClock{T: pinned}

	first := clock.Now()
	second := clock.Now()
	third := clock.Now()

	assert.True(t, first.Equal(second))
	assert.True(t, second.Equal(third))
	assert.Equal(t, pinned, first)
}

// TestRealClock_NowAdvancesSanityCheck verifies that RealClock is a thin
// pass-through to time.Now() (different successive calls, monotonically
// non-decreasing). The clock interface contract doesn't *require* monotonic
// guarantees — only that RealClock does not invent time.
func TestRealClock_NowAdvancesSanityCheck(t *testing.T) {
	clock := RealClock{}
	first := clock.Now()
	time.Sleep(1 * time.Millisecond)
	second := clock.Now()

	assert.False(t, first.After(second), "RealClock must not return times earlier than a prior call")
}

// TestNewDemandForecasterWithClock_NilDefaultsToReal verifies that passing a nil
// Clock to NewDemandForecasterWithClock falls back to RealClock (no NPE).
func TestNewDemandForecasterWithClock_NilDefaultsToReal(t *testing.T) {
	f := NewDemandForecasterWithClock(nil, nil)
	assert.NotNil(t, f)
	assert.NotNil(t, f.clock)
	_, ok := f.clock.(RealClock)
	assert.True(t, ok, "nil clock must default to RealClock, got %T", f.clock)
}
