package discovery

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"

	"go.temporal.io/sdk/workflow"
)

// DiscoveryWorkflow orchestrates feature discovery across all sources
func DiscoveryWorkflow(ctx workflow.Context, config models.DiscoveryConfig) (*models.DiscoveryResult, error) {
	logger := workflow.GetLogger(ctx)

	result := &models.DiscoveryResult{
		RunID:          "discovery-run-" + fmt.Sprintf("%d", time.Now().Unix()),
		StartTime:      time.Now(),
		SourcesScanned: []string{},
		Candidates:     []models.FeatureCandidate{},
		Stats:          make(map[string]interface{}),
	}

	// Step 1: Scan Postgres schemas
	var postgresFields []FieldMetadata
	err := workflow.ExecuteActivity(
		ctx,
		ScanPostgresActivity,
		config,
	).Get(ctx, &postgresFields)

	if err != nil {
		logger.Error("Postgres scan failed", "error", err)
		result.RunStatus = "partial"
		// Continue with other sources
	} else {
		logger.Info("Postgres scan complete", "fields_found", len(postgresFields))
		result.SourcesScanned = append(result.SourcesScanned, "postgres")
	}

	// Step 2: Scan Trino warehouses (parallel with postgres)
	var trinoFields []FieldMetadata
	err = workflow.ExecuteActivity(
		ctx,
		ScanTrinoActivity,
		config,
	).Get(ctx, &trinoFields)

	if err != nil {
		logger.Error("Trino scan failed", "error", err)
	} else {
		logger.Info("Trino scan complete", "fields_found", len(trinoFields))
		result.SourcesScanned = append(result.SourcesScanned, "trino")
	}

	// Step 3: Parse application logs
	var logFields []ParsedLogField
	err = workflow.ExecuteActivity(
		ctx,
		ParseLogsActivity,
		config,
	).Get(ctx, &logFields)

	if err != nil {
		logger.Error("Log parsing failed", "error", err)
	} else {
		logger.Info("Log parsing complete", "fields_found", len(logFields))
		result.SourcesScanned = append(result.SourcesScanned, "logs")
	}

	// Step 4: Extract Prometheus metrics
	var metricInfo []MetricInfo
	err = workflow.ExecuteActivity(
		ctx,
		ExtractMetricsActivity,
		config,
	).Get(ctx, &metricInfo)

	if err != nil {
		logger.Error("Metric extraction failed", "error", err)
	} else {
		logger.Info("Metric extraction complete", "metrics_found", len(metricInfo))
		result.SourcesScanned = append(result.SourcesScanned, "prometheus")
	}

	// Step 5: Convert all discovered fields to candidate features
	allCandidates := []models.FeatureCandidate{}

	// Process Postgres fields
	postgresCandidates := convertFieldsToFeatureCandidates(postgresFields, "postgres")
	allCandidates = append(allCandidates, postgresCandidates...)

	// Process Trino fields
	trinoCandidates := convertFieldsToFeatureCandidates(trinoFields, "trino")
	allCandidates = append(allCandidates, trinoCandidates...)

	// Process log fields
	logCandidates := convertLogFieldsToFeatureCandidates(logFields)
	allCandidates = append(allCandidates, logCandidates...)

	// Process metrics
	metricCandidates := convertMetricsToFeatureCandidates(metricInfo)
	allCandidates = append(allCandidates, metricCandidates...)

	// Step 6: Rank all candidates
	var rankedCandidates []models.FeatureCandidate
	err = workflow.ExecuteActivity(
		ctx,
		RankCandidatesActivity,
		allCandidates,
		config.ScoringWeights,
	).Get(ctx, &rankedCandidates)

	if err != nil {
		logger.Error("Ranking failed", "error", err)
		rankedCandidates = allCandidates // Fallback to unranked
	}

	// Step 7: Generate derived features for top candidates
	var derivedCandidates []models.FeatureCandidate
	topCandidates := getTopCandidates(rankedCandidates, 50)
	err = workflow.ExecuteActivity(
		ctx,
		GenerateDerivedFeaturesActivity,
		topCandidates,
	).Get(ctx, &derivedCandidates)

	if err != nil {
		logger.Error("Derived feature generation failed", "error", err)
	} else {
		logger.Info("Derived features generated", "count", len(derivedCandidates))
	}

	// Combine all candidates
	allCandidates = append(rankedCandidates, derivedCandidates...)

	// Step 8: Persist discovery results
	err = workflow.ExecuteActivity(
		ctx,
		PersistDiscoveryResultsActivity,
		result.RunID,
		allCandidates,
		result.SourcesScanned,
	).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to persist results", "error", err)
	} else {
		logger.Info("Discovery results persisted")
	}

	// Step 9: Generate statistics
	statsMap := make(map[string]interface{})
	statsMap["total_candidates"] = len(allCandidates)
	statsMap["ranked_candidates"] = len(rankedCandidates)
	statsMap["derived_candidates"] = len(derivedCandidates)
	statsMap["sources_scanned"] = len(result.SourcesScanned)
	statsMap["candidates_from_postgres"] = len(postgresCandidates)
	statsMap["candidates_from_trino"] = len(trinoCandidates)
	statsMap["candidates_from_logs"] = len(logCandidates)
	statsMap["candidates_from_prometheus"] = len(metricCandidates)

	result.Candidates = allCandidates
	result.CandidatesFound = len(allCandidates)
	result.Stats = statsMap
	result.EndTime = time.Now()
	result.RunStatus = "success"

	logger.Info("Discovery workflow complete",
		"total_candidates", len(allCandidates),
		"runtime_seconds", result.EndTime.Sub(result.StartTime).Seconds(),
	)

	return result, nil
}

// Activity: Scan Postgres schemas
func ScanPostgresActivity(ctx context.Context, config models.DiscoveryConfig) ([]FieldMetadata, error) {
	// In real implementation, would connect to actual Postgres instances
	// For now, return simulated data for testing
	return []FieldMetadata{
		{
			DatabaseType:        "postgres",
			DatabaseName:        "semlayer",
			TableName:           "incidents",
			FieldName:           "severity",
			FieldType:           "string",
			IsNullable:          true,
			CardinalityEstimate: 5,
			LastScannedAt:       time.Now(),
			Frequency:           10000,
		},
		{
			DatabaseType:        "postgres",
			DatabaseName:        "semlayer",
			TableName:           "incidents",
			FieldName:           "duration_minutes",
			FieldType:           "integer",
			IsNullable:          false,
			CardinalityEstimate: 500,
			LastScannedAt:       time.Now(),
			Frequency:           10000,
		},
		{
			DatabaseType:        "postgres",
			DatabaseName:        "semlayer",
			TableName:           "action_history",
			FieldName:           "action_type",
			FieldType:           "string",
			IsNullable:          false,
			CardinalityEstimate: 5,
			LastScannedAt:       time.Now(),
			Frequency:           50000,
		},
	}, nil
}

// Activity: Scan Trino warehouses
func ScanTrinoActivity(ctx context.Context, config models.DiscoveryConfig) ([]FieldMetadata, error) {
	// Simulated Trino scan
	return []FieldMetadata{
		{
			DatabaseType:        "trino",
			DatabaseName:        "analytics",
			TableName:           "events_daily",
			FieldName:           "error_count",
			FieldType:           "bigint",
			IsNullable:          true,
			CardinalityEstimate: 100,
			LastScannedAt:       time.Now(),
		},
		{
			DatabaseType:        "trino",
			DatabaseName:        "analytics",
			TableName:           "events_daily",
			FieldName:           "avg_latency_ms",
			FieldType:           "double",
			IsNullable:          true,
			CardinalityEstimate: 1000,
			LastScannedAt:       time.Now(),
		},
	}, nil
}

// Activity: Parse application logs
func ParseLogsActivity(ctx context.Context, config models.DiscoveryConfig) ([]ParsedLogField, error) {
	// Simulated log parsing
	return []ParsedLogField{
		{
			FieldName:  "request_path",
			FieldType:  "string",
			SampleVal:  "/api/v1/incidents",
			Frequency:  5000,
			SourceType: "json_field",
			Confidence: 0.9,
		},
		{
			FieldName:  "response_status",
			FieldType:  "number",
			SampleVal:  200,
			Frequency:  4800,
			SourceType: "json_field",
			Confidence: 0.95,
		},
		{
			FieldName:  "processing_time_ms",
			FieldType:  "number",
			SampleVal:  123.45,
			Frequency:  4900,
			SourceType: "json_field",
			Confidence: 0.92,
		},
	}, nil
}

// Activity: Extract Prometheus metrics
func ExtractMetricsActivity(ctx context.Context, config models.DiscoveryConfig) ([]MetricInfo, error) {
	// Simulated metric extraction
	return []MetricInfo{
		{
			MetricName:    "http_requests_total",
			MetricType:    "counter",
			SampleValue:   50000,
			Cardinality:   500,
			DiscoveryTime: time.Now(),
		},
		{
			MetricName:    "http_request_duration_seconds",
			MetricType:    "histogram",
			SampleValue:   0.045,
			Cardinality:   100,
			DiscoveryTime: time.Now(),
		},
		{
			MetricName:    "active_workers",
			MetricType:    "gauge",
			SampleValue:   8,
			Cardinality:   1,
			DiscoveryTime: time.Now(),
		},
	}, nil
}

// Activity: Rank all candidates
func RankCandidatesActivity(ctx context.Context, candidates []models.FeatureCandidate, weights map[string]float64) ([]models.FeatureCandidate, error) {
	ranker := NewCandidateRanker(log.New(log.Writer(), "ACTIVITY_RANK: ", log.LstdFlags))

	// Convert weights to ScoringWeights struct
	sw := DefaultWeights()
	if len(weights) > 0 {
		if w, ok := weights["completeness"]; ok {
			sw.Completeness = w
		}
		// ... similar for other weights
	}

	ranked := ranker.RankCandidates(candidates, sw)
	return ranked, nil
}

// Activity: Generate derived features
func GenerateDerivedFeaturesActivity(ctx context.Context, candidates []models.FeatureCandidate) ([]models.FeatureCandidate, error) {
	gen := NewFeatureGenerator(log.New(log.Writer(), "ACTIVITY_DERIVE: ", log.LstdFlags))

	// Generate time-series features for top candidates
	timeSeries := gen.GenerateTimeSeriesFeatures(candidates)

	// Generate aggregations
	aggs := gen.GenerateAggregations(candidates)

	// Generate derived feature candidates
	derived := append(timeSeries, aggs...)
	candidates = gen.ExportAsFeatureCandidates(derived)

	return candidates, nil
}

// Activity: Persist discovery results to database
func PersistDiscoveryResultsActivity(ctx context.Context, runID string, candidates []models.FeatureCandidate, sourceScanned []string) error {
	logger := log.New(log.Writer(), "ACTIVITY_PERSIST: ", log.LstdFlags)

	// In real implementation, would persist to Postgres
	logger.Printf("Persisting %d candidates from discovery run %s\n", len(candidates), runID)

	// Simulated: would execute INSERT statements
	for i, candidate := range candidates {
		if i < 3 { // Log first 3
			logger.Printf("  - %s (score=%.2f)\n", candidate.Name, candidate.BusinessValue)
		}
	}

	return nil
}

// Helper: Convert field metadata to feature candidates
func convertFieldsToFeatureCandidates(fields []FieldMetadata, source string) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, len(fields))
	for i, f := range fields {
		candidates[i] = models.FeatureCandidate{
			Name:           f.FieldName,
			SourceDatabase: source,
			SourceSchema:   f.DatabaseName,
			SourceTable:    f.TableName,
			SourceField:    f.FieldName,
			DataType:       f.FieldType,
			Completeness:   float64(f.Frequency) / 10000.0, // Estimate
			Cardinality:    f.CardinalityEstimate,
			BusinessValue:  0.5, // Will be scored later
			TechnicalScore: 0.7,
			DiscoveredAt:   f.LastScannedAt,
			Status:         "candidate",
		}
	}
	return candidates
}

// Helper: Convert log fields to candidates
func convertLogFieldsToFeatureCandidates(fields []ParsedLogField) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, len(fields))
	for i, f := range fields {
		candidates[i] = models.FeatureCandidate{
			Name:           "log_" + f.FieldName,
			SourceDatabase: "logs",
			SourceField:    f.FieldName,
			DataType:       f.FieldType,
			Completeness:   0.8,
			Cardinality:    -1,
			BusinessValue:  0.5,
			TechnicalScore: f.Confidence,
			DiscoveredAt:   time.Now(),
			Status:         "candidate",
		}
	}
	return candidates
}

// Helper: Convert metrics to candidates
func convertMetricsToFeatureCandidates(metrics []MetricInfo) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, len(metrics))
	for i, m := range metrics {
		candidates[i] = models.FeatureCandidate{
			Name:           m.MetricName,
			SourceDatabase: "prometheus",
			SourceField:    m.MetricName,
			DataType:       "float",
			Completeness:   0.99,
			Cardinality:    m.Cardinality,
			BusinessValue:  0.5,
			TechnicalScore: 0.8,
			DiscoveredAt:   m.DiscoveryTime,
			Status:         "candidate",
		}
	}
	return candidates
}

// Helper: Get top N candidates
func getTopCandidates(candidates []models.FeatureCandidate, n int) []models.FeatureCandidate {
	if len(candidates) <= n {
		return candidates
	}
	return candidates[:n]
}

// ScheduleDiscoveryWorkflow schedules regular discovery runs
func ScheduleDiscoveryWorkflow(ctx context.Context, db *sql.DB, interval time.Duration) {
	logger := log.New(log.Writer(), "SCHEDULE_DISCOVERY: ", log.LstdFlags)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		logger.Printf("Triggering discovery workflow at %v\n", time.Now())
		// In real implementation, would:
		// 1. Connect to Temporal client
		// 2. Start new workflow execution
		// 3. Await completion
		// 4. Log results
	}
}
