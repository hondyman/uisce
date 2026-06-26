package ml

// ResourceAllocator determines the resource profile for syncing
type ResourceAllocator struct{}

// Allocate returns the resource profile
func (r *ResourceAllocator) Allocate(features *SyncFeatures) string {
	if features.TotalEvents > 1000 {
		return "performance"
	} else if features.TotalEvents < 100 {
		return "economy"
	}
	return "standard"
}
