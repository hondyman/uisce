package rules

import (
	"sort"
	"sync/atomic"
)

const BufferSize = 10000

type LatencyProfiler struct {
	samples [BufferSize]atomic.Int64
	idx     atomic.Uint64
}

func NewLatencyProfiler() *LatencyProfiler {
	return &LatencyProfiler{}
}

func (p *LatencyProfiler) RecordExecution(nanos int64) {
	i := p.idx.Add(1) % BufferSize
	p.samples[i].Store(nanos)
}

type LatencyReport struct {
	P50Ns int64  `json:"p50_ns"`
	P95Ns int64  `json:"p95_ns"`
	P99Ns int64  `json:"p99_ns"`
	Count uint64 `json:"sample_count"`
}

func (p *LatencyProfiler) GetDistribution() LatencyReport {
	copySamples := make([]int64, 0, BufferSize)
	for i := 0; i < BufferSize; i++ {
		val := p.samples[i].Load()
		if val > 0 {
			copySamples = append(copySamples, val)
		}
	}

	if len(copySamples) == 0 {
		return LatencyReport{}
	}

	sort.Slice(copySamples, func(i, j int) bool { return copySamples[i] < copySamples[j] })

	n := len(copySamples)
	return LatencyReport{
		P50Ns: copySamples[int(float64(n)*0.50)],
		P95Ns: copySamples[int(float64(n)*0.95)],
		P99Ns: copySamples[int(float64(n)*0.99)],
		Count: uint64(n),
	}
}
