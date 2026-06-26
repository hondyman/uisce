package performance

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type PerformanceMetrics struct {
	RenderTimeP95 int     `json:"render_time_p95_ms"`
	APILatencyP95 int     `json:"api_latency_p95_ms"`
	APIFanout     int     `json:"api_fanout"`
	CacheHitRate  float64 `json:"cache_hit_rate"`
	PreAggHitRate float64 `json:"preagg_hit_rate"`
}

type RegressionResult struct {
	PageID         uuid.UUID          `json:"page_id"`
	Before         PerformanceMetrics `json:"before"`
	After          PerformanceMetrics `json:"after"`
	HasRegression  bool               `json:"has_regression"`
	RegressionDesc string             `json:"regression_desc,omitempty"`
}

type SyntheticRunner struct{}

func NewSyntheticRunner() *SyntheticRunner {
	return &SyntheticRunner{}
}

func (s *SyntheticRunner) Run(ctx context.Context, pageID uuid.UUID) (*PerformanceMetrics, error) {
	// Mock implementation
	// Real: Spin up preview env, apply synthetic load, collect metrics
	metrics := &PerformanceMetrics{
		RenderTimeP95: 850,
		APILatencyP95: 120,
		APIFanout:     3,
		CacheHitRate:  0.92,
		PreAggHitRate: 0.88,
	}
	return metrics, nil
}

func (s *SyntheticRunner) Compare(ctx context.Context, pageID uuid.UUID, before, after *PerformanceMetrics) (*RegressionResult, error) {
	result := &RegressionResult{
		PageID:        pageID,
		Before:        *before,
		After:         *after,
		HasRegression: false,
	}

	// Check for regressions
	if after.RenderTimeP95 > before.RenderTimeP95+100 {
		result.HasRegression = true
		result.RegressionDesc = fmt.Sprintf("Render time increased by %dms", after.RenderTimeP95-before.RenderTimeP95)
	}

	return result, nil
}
