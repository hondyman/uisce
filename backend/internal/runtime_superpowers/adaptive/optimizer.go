package adaptive

import (
	"context"
)

type RenderingMode string

const (
	ModeFull     RenderingMode = "full"
	ModeCompact  RenderingMode = "compact"
	ModeSkeleton RenderingMode = "skeleton"
	ModeDeferred RenderingMode = "deferred"
)

type DeviceProfile struct {
	DeviceClass string `json:"device_class"` // high, mid, low
	NetworkType string `json:"network_type"` // 4g, 3g, wifi
	RTT         int    `json:"rtt"`          // ms
	SaveData    bool   `json:"save_data"`
}

type OptimizationResult struct {
	Mode        RenderingMode `json:"mode"`
	RowLimit    int           `json:"row_limit"`
	DeferCharts bool          `json:"defer_charts"`
}

type Optimizer struct{}

func NewOptimizer() *Optimizer {
	return &Optimizer{}
}

func (o *Optimizer) Optimize(ctx context.Context, profile DeviceProfile) (*OptimizationResult, error) {
	result := &OptimizationResult{
		Mode:        ModeFull,
		RowLimit:    100,
		DeferCharts: false,
	}

	// 1. Check Network
	if profile.NetworkType == "3g" || profile.RTT > 500 || profile.SaveData {
		result.Mode = ModeCompact
		result.RowLimit = 20
		result.DeferCharts = true
	}

	// 2. Check Device Class
	if profile.DeviceClass == "low" {
		result.DeferCharts = true
		// Maybe force skeleton first
		if result.Mode == ModeFull {
			result.Mode = ModeSkeleton
		}
	}

	return result, nil
}
