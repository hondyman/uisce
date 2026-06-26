package ml

import (
	"time"
)

// SyncTimePredictor predicts the optimal time for the next sync
type SyncTimePredictor struct{}

// Predict returns the best time to sync
func (s *SyncTimePredictor) Predict(features *SyncFeatures) time.Time {
	now := time.Now()
	// Basic fallback prediction
	if features.BestHourOfDay > 0 {
		return time.Date(now.Year(), now.Month(), now.Day(), features.BestHourOfDay, 0, 0, 0, now.Location())
	}
	// Default to 2 AM
	return time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
}
