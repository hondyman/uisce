package ops

import (
	"context"
	"fmt"
	"time"
)

// HeatmapBuilder builds latency heatmap visualizations
type HeatmapBuilder struct {
	store    Store
	timeline *TimelineService
}

// NewHeatmapBuilder creates a new heatmap builder
func NewHeatmapBuilder(store Store) *HeatmapBuilder {
	return &HeatmapBuilder{store: store, timeline: nil}
}

// NewHeatmapBuilderWithTimeline creates a new heatmap builder with timeline service
func NewHeatmapBuilderWithTimeline(store Store, timeline *TimelineService) *HeatmapBuilder {
	return &HeatmapBuilder{store: store, timeline: timeline}
}

// BuildHeatmap builds a heatmap grouped by dimension over a time window
func (h *HeatmapBuilder) BuildHeatmap(ctx context.Context, dimensionType string, bucketSize, window time.Duration, limit int) (*Heatmap, error) {
	// Fetch all series for this dimension type
	series, err := h.store.GetHeatmapSeries(ctx, dimensionType, limit, bucketSize, window)
	if err != nil {
		return nil, fmt.Errorf("get heatmap series: %w", err)
	}

	// Extract unique buckets from series
	buckets := extractBuckets(series)

	return &Heatmap{
		Buckets: buckets,
		Series:  series,
	}, nil
}

// BuildRegionHeatmap builds a heatmap grouped by region
func (h *HeatmapBuilder) BuildRegionHeatmap(ctx context.Context) (*Heatmap, error) {
	// 5-minute buckets over last 24 hours
	return h.BuildHeatmap(ctx, "region", 5*time.Minute, 24*time.Hour, 10)
}

// BuildTenantHeatmap builds a heatmap grouped by tenant
func (h *HeatmapBuilder) BuildTenantHeatmap(ctx context.Context) (*Heatmap, error) {
	// 5-minute buckets over last 6 hours
	return h.BuildHeatmap(ctx, "tenant", 5*time.Minute, 6*time.Hour, 20)
}

// BuildEndpointHeatmap builds a heatmap grouped by endpoint
func (h *HeatmapBuilder) BuildEndpointHeatmap(ctx context.Context) (*Heatmap, error) {
	// 5-minute buckets over last 6 hours
	return h.BuildHeatmap(ctx, "endpoint", 5*time.Minute, 6*time.Hour, 20)
}

// extractBuckets extracts unique time buckets from series
func extractBuckets(series []HeatmapSeries) []time.Time {
	buckets := make([]time.Time, 0)
	seen := make(map[int64]bool)

	for _, s := range series {
		for _, v := range s.Values {
			ts := v.Time.Unix()
			if !seen[ts] {
				buckets = append(buckets, v.Time)
				seen[ts] = true
			}
		}
	}

	// Sort buckets in ascending order
	for i := 0; i < len(buckets)-1; i++ {
		for j := i + 1; j < len(buckets); j++ {
			if buckets[j].Before(buckets[i]) {
				buckets[i], buckets[j] = buckets[j], buckets[i]
			}
		}
	}

	return buckets
}

// DetectAnomalies detects latency spikes and records timeline events
func (h *HeatmapBuilder) DetectAnomalies(ctx context.Context, dimensionType, dimensionValue string) error {
	if h.timeline == nil {
		return nil // Timeline service not available
	}

	// Get recent latency data
	data, err := h.store.GetHeatmapData(ctx, dimensionType, dimensionValue, time.Minute, 2*time.Hour)
	if err != nil {
		return fmt.Errorf("get heatmap data: %w", err)
	}

	if len(data) < 2 {
		return nil // Need at least 2 data points for comparison
	}

	// Simple anomaly detection: if latest p95 is 2x higher than average of previous
	latest := data[len(data)-1]
	var prevSum int
	for i := 0; i < len(data)-1; i++ {
		prevSum += data[i].P95MS
	}
	baseline := prevSum / (len(data) - 1)

	// If p95 > 2x baseline, it's an anomaly
	if latest.P95MS > baseline*2 && baseline > 0 {
		_ = h.timeline.RecordLatencyAnomaly(ctx, dimensionValue, latest.P95MS, baseline)
	}

	return nil
}
