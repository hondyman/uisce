package ml

// BatchOptimizer determines the optimal batch size for syncing
type BatchOptimizer struct{}

// Optimize returns the optimal batch size
func (b *BatchOptimizer) Optimize(features *SyncFeatures) int {
	if features.EventDensity > 10 {
		return 500
	} else if features.EventDensity > 5 {
		return 200
	}
	return 100
}
