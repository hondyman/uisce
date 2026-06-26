package featurestore

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// FeatureStorePersistence handles database persistence for features
type FeatureStorePersistence struct {
	mu         sync.RWMutex
	db         *sql.DB
	batchSize  int
	flushTick  time.Ticker
	writeQueue chan *FeatureSnapshot
}

// NewFeatureStorePersistence creates a new persistence layer
func NewFeatureStorePersistence(db *sql.DB, batchSize int) *FeatureStorePersistence {
	return &FeatureStorePersistence{
		db:         db,
		batchSize:  batchSize,
		flushTick:  *time.NewTicker(5 * time.Second),
		writeQueue: make(chan *FeatureSnapshot, 100),
	}
}

// InitializeSchema creates required database tables
func (fsp *FeatureStorePersistence) InitializeSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS feature_definitions (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL UNIQUE,
		category VARCHAR(50),
		description TEXT,
		data_type VARCHAR(50),
		version VARCHAR(50),
		is_active BOOLEAN DEFAULT true,
		compute_latency_ms INT,
		availability FLOAT,
		freshness_seconds INT,
		metadata JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS feature_values (
		id SERIAL PRIMARY KEY,
		entity_id VARCHAR(255) NOT NULL,
		entity_type VARCHAR(50),
		feature_name VARCHAR(255) NOT NULL,
		value FLOAT NOT NULL,
		computed_at TIMESTAMP NOT NULL,
		valid_until TIMESTAMP,
		is_cached BOOLEAN DEFAULT false,
		region VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(entity_id, feature_name, computed_at)
	);

	CREATE TABLE IF NOT EXISTS feature_snapshots (
		id SERIAL PRIMARY KEY,
		entity_id VARCHAR(255) NOT NULL,
		snapshot_timestamp TIMESTAMP NOT NULL,
		features JSONB NOT NULL,
		region VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(entity_id, snapshot_timestamp)
	);

	CREATE TABLE IF NOT EXISTS feature_statistics (
		id SERIAL PRIMARY KEY,
		feature_name VARCHAR(255) NOT NULL,
		period_start TIMESTAMP NOT NULL,
		period_end TIMESTAMP NOT NULL,
		sample_count INT,
		mean FLOAT,
		stddev FLOAT,
		min FLOAT,
		q25 FLOAT,
		median FLOAT,
		q75 FLOAT,
		max FLOAT,
		region VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(feature_name, period_start)
	);

	CREATE TABLE IF NOT EXISTS feature_computations (
		id SERIAL PRIMARY KEY,
		entity_id VARCHAR(255) NOT NULL,
		feature_name VARCHAR(255) NOT NULL,
		computer_name VARCHAR(100),
		compute_time_ms INT,
		cache_hit BOOLEAN,
		error_message TEXT,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		region VARCHAR(50)
	);

	CREATE INDEX IF NOT EXISTS idx_feature_values_entity ON feature_values(entity_id);
	CREATE INDEX IF NOT EXISTS idx_feature_values_name ON feature_values(feature_name);
	CREATE INDEX IF NOT EXISTS idx_feature_snapshots_entity ON feature_snapshots(entity_id);
	CREATE INDEX IF NOT EXISTS idx_feature_stats_name ON feature_statistics(feature_name);
	`

	_, err := fsp.db.ExecContext(ctx, schema)
	return err
}

// StoreFeatureValue persists a computed feature value
func (fsp *FeatureStorePersistence) StoreFeatureValue(ctx context.Context, entityID string, feature *ComputedFeature, region string) error {
	query := `
	INSERT INTO feature_values (entity_id, feature_name, value, computed_at, valid_until, is_cached, region)
	VALUES ($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT (entity_id, feature_name, computed_at) DO UPDATE
	SET valid_until = EXCLUDED.valid_until, is_cached = EXCLUDED.is_cached
	`

	_, err := fsp.db.ExecContext(ctx, query,
		entityID,
		feature.FeatureName,
		feature.Value,
		feature.ComputedAt,
		feature.ValidUntil,
		feature.IsFromCache,
		region,
	)

	return err
}

// StoreFeatureSnapshot stores a point-in-time snapshot
func (fsp *FeatureStorePersistence) StoreFeatureSnapshot(ctx context.Context, snapshot *FeatureSnapshot, region string) error {
	// Convert features to JSON
	featureJSON := make(map[string]interface{})
	for featureName, value := range snapshot.Features {
		featureJSON[featureName] = value
	}

	query := `
	INSERT INTO feature_snapshots (entity_id, snapshot_timestamp, features, region)
	VALUES ($1, $2, $3, $4)
	ON CONFLICT (entity_id, snapshot_timestamp) DO NOTHING
	`

	_, err := fsp.db.ExecContext(ctx, query,
		snapshot.EntityID,
		snapshot.Timestamp,
		featureJSON,
		region,
	)

	return err
}

// GetFeatureHistory retrieves historical feature values
func (fsp *FeatureStorePersistence) GetFeatureHistory(ctx context.Context, entityID string, featureName string, since time.Time) ([]*ComputedFeature, error) {
	query := `
	SELECT feature_name, value, computed_at, valid_until, is_cached
	FROM feature_values
	WHERE entity_id = $1 AND feature_name = $2 AND computed_at >= $3
	ORDER BY computed_at DESC
	LIMIT 1000
	`

	rows, err := fsp.db.QueryContext(ctx, query, entityID, featureName, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []*ComputedFeature
	for rows.Next() {
		var f ComputedFeature
		if err := rows.Scan(&f.FeatureName, &f.Value, &f.ComputedAt, &f.ValidUntil, &f.IsFromCache); err != nil {
			return nil, err
		}
		features = append(features, &f)
	}

	return features, rows.Err()
}

// UpdateStatistics computes and stores feature statistics
func (fsp *FeatureStorePersistence) UpdateStatistics(ctx context.Context, stats *FeatureStatistics, region string) error {
	query := `
	INSERT INTO feature_statistics (feature_name, period_start, period_end, sample_count, mean, stddev, min, q25, median, q75, max, region)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	ON CONFLICT (feature_name, period_start) DO UPDATE
	SET sample_count = EXCLUDED.sample_count,
	    mean = EXCLUDED.mean,
	    stddev = EXCLUDED.stddev,
	    min = EXCLUDED.min,
	    q25 = EXCLUDED.q25,
	    median = EXCLUDED.median,
	    q75 = EXCLUDED.q75,
	    max = EXCLUDED.max
	`

	_, err := fsp.db.ExecContext(ctx, query,
		stats.FeatureName,
		time.Now().Add(-24*time.Hour),
		time.Now(),
		stats.Count,
		stats.Mean,
		stats.StdDev,
		stats.Min,
		stats.Q1,
		stats.Median,
		stats.Q3,
		stats.Max,
		region,
	)

	return err
}

// GetStatistics retrieves feature statistics
func (fsp *FeatureStorePersistence) GetStatistics(ctx context.Context, featureName string) (*FeatureStatistics, error) {
	query := `
	SELECT feature_name, sample_count, mean, stddev, min, q25, median, q75, max
	FROM feature_statistics
	WHERE feature_name = $1 AND period_end >= NOW() - INTERVAL '24 hours'
	ORDER BY period_end DESC
	LIMIT 1
	`

	var stats FeatureStatistics
	err := fsp.db.QueryRowContext(ctx, query, featureName).Scan(
		&stats.FeatureName,
		&stats.Count,
		&stats.Mean,
		&stats.StdDev,
		&stats.Min,
		&stats.Q1,
		&stats.Median,
		&stats.Q3,
		&stats.Max,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no statistics for feature %s", featureName)
	}

	return &stats, err
}

// LogComputation logs feature computation metrics
func (fsp *FeatureStorePersistence) LogComputation(ctx context.Context, entityID string, featureName string, computeTimeMs int, cacheHit bool, region string) error {
	query := `
	INSERT INTO feature_computations (entity_id, feature_name, compute_time_ms, cache_hit, region)
	VALUES ($1, $2, $3, $4, $5)
	`

	_, err := fsp.db.ExecContext(ctx, query, entityID, featureName, computeTimeMs, cacheHit, region)
	return err
}

// GetComputationStats retrieves computation statistics
func (fsp *FeatureStorePersistence) GetComputationStats(ctx context.Context, featureName string, hours int) (map[string]interface{}, error) {
	query := `
	SELECT
		COUNT(*) as total_computations,
		SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END) as cache_hits,
		AVG(compute_time_ms) as avg_compute_time,
		MAX(compute_time_ms) as max_compute_time,
		MIN(compute_time_ms) as min_compute_time,
		PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY compute_time_ms) as p95_latency
	FROM feature_computations
	WHERE feature_name = $1 AND timestamp >= NOW() - INTERVAL '1 hour' * $2
	`

	row := fsp.db.QueryRowContext(ctx, query, featureName, hours)

	stats := make(map[string]interface{})
	var totalComputations, cacheHits, avgComputeTime, maxComputeTime, minComputeTime, p95Latency sql.NullFloat64

	err := row.Scan(&totalComputations, &cacheHits, &avgComputeTime, &maxComputeTime, &minComputeTime, &p95Latency)
	if err != nil {
		return nil, err
	}

	if totalComputations.Valid {
		stats["total_computations"] = int(totalComputations.Float64)
	}
	if cacheHits.Valid {
		stats["cache_hits"] = int(cacheHits.Float64)
	}
	if avgComputeTime.Valid {
		stats["avg_compute_time_ms"] = avgComputeTime.Float64
	}
	if p95Latency.Valid {
		stats["p95_latency_ms"] = p95Latency.Float64
	}

	return stats, nil
}

// Close closes database connection
func (fsp *FeatureStorePersistence) Close() error {
	if fsp.db != nil {
		return fsp.db.Close()
	}
	return nil
}
