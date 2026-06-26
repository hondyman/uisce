package featurestore

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// FeatureStore manages feature computation, versioning, and caching
type FeatureStore struct {
	mu             sync.RWMutex
	features       map[string]*FeatureDefinition
	cache          map[string]*ComputedFeature
	cacheExpiry    time.Duration
	computeTimeout time.Duration
	registry       map[string]FeatureComputer
	maxCacheSize   int
}

// FeatureDefinition represents a feature definition
type FeatureDefinition struct {
	Name         string                 `json:"name"`
	Category     string                 `json:"category"` // "numerical", "categorical", "temporal"
	Description  string                 `json:"description"`
	DataType     string                 `json:"data_type"`
	Version      string                 `json:"version"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ComputeFn    string                 `json:"compute_function"`
	Dependencies []string               `json:"dependencies"`
	SLO          FeatureSLO             `json:"slo"`
	Metadata     map[string]interface{} `json:"metadata"`
	IsActive     bool                   `json:"is_active"`
}

// FeatureSLO defines quality expectations
type FeatureSLO struct {
	ComputeLatencyMs int     `json:"compute_latency_ms"`
	Availability     float64 `json:"availability"`      // 0-1
	Freshness        int     `json:"freshness_seconds"` // Max staleness
	Completeness     float64 `json:"completeness"`      // 0-1
}

// ComputedFeature represents a computed feature value with metadata
type ComputedFeature struct {
	FeatureName string                 `json:"feature_name"`
	Value       float64                `json:"value"`
	ComputedAt  time.Time              `json:"computed_at"`
	ValidUntil  time.Time              `json:"valid_until"`
	IsFromCache bool                   `json:"is_from_cache"`
	Metadata    map[string]interface{} `json:"metadata"`
	Version     string                 `json:"version"`
	IsStale     bool                   `json:"is_stale"`
}

// FeatureRequest groups features to compute
type FeatureRequest struct {
	EntityID        string     `json:"entity_id"`
	EntityType      string     `json:"entity_type"` // "chain", "tenant", "region"
	FeatureNames    []string   `json:"feature_names"`
	TimeReference   *time.Time `json:"time_reference,omitempty"` // For time-travel
	IncludeMetadata bool       `json:"include_metadata"`
}

// FeatureBatch holds multiple computed features
type FeatureBatch struct {
	EntityID    string
	Features    map[string]*ComputedFeature
	ComputedAt  time.Time
	CacheMisses int
	Latency     time.Duration
}

// FeatureComputer interface for custom feature computation
type FeatureComputer interface {
	Compute(ctx context.Context, entityID string, timeRef *time.Time) (float64, error)
	Name() string
}

// FeatureSnapshot represents historical feature values
type FeatureSnapshot struct {
	EntityID      string
	Timestamp     time.Time
	Features      map[string]float64
	ComputeTimeMs int64
}

// FeatureStatistics holds distribution statistics
type FeatureStatistics struct {
	FeatureName string    `json:"feature_name"`
	Count       int64     `json:"count"`
	Mean        float64   `json:"mean"`
	StdDev      float64   `json:"std_dev"`
	Min         float64   `json:"min"`
	Q1          float64   `json:"q1"`
	Median      float64   `json:"median"`
	Q3          float64   `json:"q3"`
	Max         float64   `json:"max"`
	Missing     int64     `json:"missing_count"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewFeatureStore creates a new feature store
func NewFeatureStore(cacheExpiry time.Duration, maxCacheSize int) *FeatureStore {
	return &FeatureStore{
		features:       make(map[string]*FeatureDefinition),
		cache:          make(map[string]*ComputedFeature),
		registry:       make(map[string]FeatureComputer),
		cacheExpiry:    cacheExpiry,
		computeTimeout: 5 * time.Second,
		maxCacheSize:   maxCacheSize,
	}
}

// RegisterFeature registers a feature definition
func (fs *FeatureStore) RegisterFeature(ctx context.Context, def *FeatureDefinition) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if def.Name == "" {
		return fmt.Errorf("feature name cannot be empty")
	}

	def.CreatedAt = time.Now()
	def.UpdatedAt = time.Now()
	fs.features[def.Name] = def

	return nil
}

// RegisterComputer registers a feature computer
func (fs *FeatureStore) RegisterComputer(ctx context.Context, computer FeatureComputer) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.registry[computer.Name()] = computer
	return nil
}

// ComputeFeatures computes requested features with caching
func (fs *FeatureStore) ComputeFeatures(ctx context.Context, request *FeatureRequest) (*FeatureBatch, error) {
	start := time.Now()
	batch := &FeatureBatch{
		EntityID:   request.EntityID,
		Features:   make(map[string]*ComputedFeature),
		ComputedAt: time.Now(),
	}

	fs.mu.RLock()
	featureNames := request.FeatureNames
	if len(featureNames) == 0 {
		// Get all active features
		for name, def := range fs.features {
			if def.IsActive {
				featureNames = append(featureNames, name)
			}
		}
	}
	fs.mu.RUnlock()

	for _, featureName := range featureNames {
		// Try cache first
		cacheKey := fmt.Sprintf("%s:%s", request.EntityID, featureName)
		if computed := fs.getCachedFeature(cacheKey); computed != nil && !computed.IsStale {
			batch.Features[featureName] = computed
			continue
		}

		// Cache miss - compute
		batch.CacheMisses++
		computed, err := fs.computeFeature(ctx, request.EntityID, featureName, request.TimeReference)
		if err != nil {
			// Continue on error, don't fail entire batch
			continue
		}

		batch.Features[featureName] = computed
		fs.cacheFeature(cacheKey, computed)
	}

	batch.Latency = time.Since(start)
	return batch, nil
}

// GetFeatureSnapshot retrieves historical feature values (time-travel)
func (fs *FeatureStore) GetFeatureSnapshot(ctx context.Context, entityID string, timestamp time.Time) (*FeatureSnapshot, error) {
	// In production, would query feature store database
	// For now, return mock snapshot

	snapshot := &FeatureSnapshot{
		EntityID:      entityID,
		Timestamp:     timestamp,
		Features:      make(map[string]float64),
		ComputeTimeMs: 150,
	}

	// Simulate time-travel computation
	daysSinceSnapshot := time.Since(timestamp).Hours() / 24
	healthDegradation := daysSinceSnapshot * 0.01 // 1% per day

	snapshot.Features["health_score"] = 0.85 - healthDegradation
	snapshot.Features["active_conflicts"] = float64(5) + daysSinceSnapshot        // More conflicts over time
	snapshot.Features["p99_latency_ms"] = float64(200) + (daysSinceSnapshot * 50) // Increasing latency

	return snapshot, nil
}

// GetFeatureStatistics returns distribution statistics for a feature
func (fs *FeatureStore) GetFeatureStatistics(ctx context.Context, featureName string) (*FeatureStatistics, error) {
	fs.mu.RLock()
	_, exists := fs.features[featureName]
	fs.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("feature %s not found", featureName)
	}

	// In production, compute from feature store
	stats := &FeatureStatistics{
		FeatureName: featureName,
		Count:       1000000,
		Timestamp:   time.Now(),
	}

	// Mock statistics based on feature type
	switch featureName {
	case "health_score":
		stats.Mean = 0.75
		stats.StdDev = 0.15
		stats.Min = 0.0
		stats.Q1 = 0.65
		stats.Median = 0.78
		stats.Q3 = 0.87
		stats.Max = 1.0
	case "active_conflicts":
		stats.Mean = 5.0
		stats.StdDev = 8.0
		stats.Min = 0
		stats.Q1 = 1
		stats.Median = 3
		stats.Q3 = 7
		stats.Max = 100
	case "p99_latency_ms":
		stats.Mean = 250
		stats.StdDev = 400
		stats.Min = 10
		stats.Q1 = 100
		stats.Median = 200
		stats.Q3 = 400
		stats.Max = 5000
	}

	return stats, nil
}

// ListFeatures returns all registered features
func (fs *FeatureStore) ListFeatures(ctx context.Context, onlyActive bool) ([]*FeatureDefinition, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []*FeatureDefinition
	for _, def := range fs.features {
		if !onlyActive || def.IsActive {
			result = append(result, def)
		}
	}

	return result, nil
}

// GetFeatureDefinition retrieves a feature definition
func (fs *FeatureStore) GetFeatureDefinition(ctx context.Context, name string) (*FeatureDefinition, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	def, exists := fs.features[name]
	if !exists {
		return nil, fmt.Errorf("feature %s not found", name)
	}

	return def, nil
}

// InvalidateCache clears cache for an entity
func (fs *FeatureStore) InvalidateCache(ctx context.Context, entityID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for key := range fs.cache {
		if key[:len(entityID)] == entityID {
			delete(fs.cache, key)
		}
	}

	return nil
}

// ClearCache completely clears the cache
func (fs *FeatureStore) ClearCache(ctx context.Context) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	fs.cache = make(map[string]*ComputedFeature)
	return nil
}

// ComputeFeatureImportance ranks features by importance for a given prediction
func (fs *FeatureStore) ComputeFeatureImportance(ctx context.Context, entityID string, shapeValues map[string]float64) (map[string]float64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	importance := make(map[string]float64)
	totalAbsShap := 0.0

	for feature, shapValue := range shapeValues {
		importance[feature] = shapValue
		if shapValue < 0 {
			totalAbsShap += -shapValue
		} else {
			totalAbsShap += shapValue
		}
	}

	// Normalize to 0-1
	if totalAbsShap > 0 {
		for feature := range importance {
			if importance[feature] < 0 {
				importance[feature] = -importance[feature] / totalAbsShap
			} else {
				importance[feature] = importance[feature] / totalAbsShap
			}
		}
	}

	return importance, nil
}

// computeFeature computes a single feature value
func (fs *FeatureStore) computeFeature(ctx context.Context, entityID string, featureName string, timeRef *time.Time) (*ComputedFeature, error) {
	fs.mu.RLock()
	computer, exists := fs.registry[featureName]
	fs.mu.RUnlock()

	if !exists {
		// Use mock computation
		return &ComputedFeature{
			FeatureName: featureName,
			Value:       0.5,
			ComputedAt:  time.Now(),
			ValidUntil:  time.Now().Add(fs.cacheExpiry),
			IsStale:     false,
		}, nil
	}

	computeCtx, cancel := context.WithTimeout(ctx, fs.computeTimeout)
	defer cancel()

	value, err := computer.Compute(computeCtx, entityID, timeRef)
	if err != nil {
		return nil, err
	}

	return &ComputedFeature{
		FeatureName: featureName,
		Value:       value,
		ComputedAt:  time.Now(),
		ValidUntil:  time.Now().Add(fs.cacheExpiry),
		IsStale:     false,
	}, nil
}

// getCachedFeature retrieves a feature from cache
func (fs *FeatureStore) getCachedFeature(key string) *ComputedFeature {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	computed, exists := fs.cache[key]
	if !exists {
		return nil
	}

	// Check if stale
	if time.Now().After(computed.ValidUntil) {
		computed.IsStale = true
	}

	return computed
}

// cacheFeature stores a feature in cache
func (fs *FeatureStore) cacheFeature(key string, computed *ComputedFeature) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Evict oldest if cache full
	if len(fs.cache) >= fs.maxCacheSize {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range fs.cache {
			if oldestTime.IsZero() || v.ComputedAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.ComputedAt
			}
		}
		if oldestKey != "" {
			delete(fs.cache, oldestKey)
		}
	}

	fs.cache[key] = computed
}
