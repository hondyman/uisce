package rules

import (
	"sync"
	"testing"
)

func TestLatencyProfiler_Empty(t *testing.T) {
	p := NewLatencyProfiler()
	r := p.GetDistribution()
	if r.Count != 0 {
		t.Errorf("expected count 0, got %d", r.Count)
	}
	if r.P50Ns != 0 || r.P95Ns != 0 || r.P99Ns != 0 {
		t.Errorf("expected zero percentiles on empty buffer, got %+v", r)
	}
}

func TestLatencyProfiler_SingleSample(t *testing.T) {
	p := NewLatencyProfiler()
	p.RecordExecution(42)
	r := p.GetDistribution()
	if r.Count != 1 {
		t.Errorf("expected count 1, got %d", r.Count)
	}
	if r.P50Ns != 42 || r.P95Ns != 42 || r.P99Ns != 42 {
		t.Errorf("expected all percentiles = 42, got %+v", r)
	}
}

func TestLatencyProfiler_PercentileOrdering(t *testing.T) {
	p := NewLatencyProfiler()
	for i := int64(1); i <= 100; i++ {
		p.RecordExecution(i * 10)
	}
	r := p.GetDistribution()
	if r.Count != 100 {
		t.Errorf("expected count 100, got %d", r.Count)
	}
	if !(r.P50Ns <= r.P95Ns && r.P95Ns <= r.P99Ns) {
		t.Errorf("expected p50 <= p95 <= p99, got p50=%d p95=%d p99=%d",
			r.P50Ns, r.P95Ns, r.P99Ns)
	}
}

func TestLatencyProfiler_Concurrent(t *testing.T) {
	p := NewLatencyProfiler()
	const goroutines = 16
	const perGoroutine = 500

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := int64(0); i < perGoroutine; i++ {
				p.RecordExecution(i + 1)
			}
		}()
	}
	wg.Wait()

	r := p.GetDistribution()
	// ring buffer wraps after BufferSize, so Count is capped at BufferSize
	if r.Count > BufferSize {
		t.Errorf("count exceeded BufferSize: %d", r.Count)
	}
	if r.Count == 0 {
		t.Error("expected non-zero count after concurrent writes")
	}
}

func TestLatencyProfiler_RingWraps(t *testing.T) {
	p := NewLatencyProfiler()
	for i := int64(0); i < BufferSize+1000; i++ {
		p.RecordExecution(i)
	}
	r := p.GetDistribution()
	if r.Count != BufferSize {
		t.Errorf("expected ring to retain BufferSize samples, got %d", r.Count)
	}
}
