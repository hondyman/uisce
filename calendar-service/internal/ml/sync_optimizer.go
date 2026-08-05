package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"calendar-service/internal/database"

	"github.com/sirupsen/logrus"
)

type SyncOptimizer struct {
	modelEndpoint   string
	modelVersion    string
	dbClient        *database.Client
	featureEngineer *FeatureEngineer
	logger          *logrus.Entry
	httpClient      *http.Client
	costConfig      SyncCostConfig
}

type SyncOptimizerConfig struct {
	ModelEndpoint   string
	ModelVersion    string
	DBClient        *database.Client
	FeatureEngineer *FeatureEngineer
	Logger          *logrus.Entry
	CostConfig      SyncCostConfig
}

// SyncCostConfig holds cost calculation configuration
type SyncCostConfig struct {
	APICallCostCents     float64 `json:"api_call_cost_cents"`
	ComputeCostPerSecond float64 `json:"compute_cost_per_second"`
	StorageCostPerMB     float64 `json:"storage_cost_per_mb"`
	TransferCostPerMB    float64 `json:"transfer_cost_per_mb"`
}

// SyncRecommendation represents sync optimization recommendation
type SyncRecommendation struct {
	OptimalTime        time.Time `json:"optimal_time"`
	ExpectedDuration   float64   `json:"expected_duration_seconds"`
	BatchSize          int       `json:"batch_size"`
	ResourceProfile    string    `json:"resource_profile"` // standard, performance, economy
	Priority           string    `json:"priority"`         // high, normal, low
	PredictedCostCents int       `json:"predicted_cost_cents"`
	SavingsCents       int       `json:"savings_cents"`
	Confidence         float64   `json:"confidence"`
	Reasoning          string    `json:"reasoning"`
	ModelVersion       string    `json:"model_version"`
}

// SyncFeatures represents features for sync optimization
type SyncFeatures struct {
	// Time-based features
	HourOfDay       int  `json:"hour_of_day"`
	DayOfWeek       int  `json:"day_of_week"`
	IsWeekend       bool `json:"is_weekend"`
	IsBusinessHours bool `json:"is_business_hours"`

	// User features
	UserTenureDays  int     `json:"user_tenure_days"`
	TotalSyncs      int     `json:"total_syncs"`
	AvgSyncDuration float64 `json:"avg_sync_duration"`
	SuccessRate     float64 `json:"success_rate"`

	// Calendar features
	CalendarAgeDays int     `json:"calendar_age_days"`
	TotalEvents     int     `json:"total_events"`
	EventDensity    float64 `json:"event_density"`
	LastSyncAgo     int     `json:"last_sync_ago_hours"`

	// Historical performance
	BestHourOfDay     int     `json:"best_hour_of_day"`
	BestDayOfWeek     int     `json:"best_day_of_week"`
	PeakAPIQuotaUsage float64 `json:"peak_api_quota_usage"`

	// Context
	Provider      string `json:"provider"`
	SyncFrequency string `json:"sync_frequency"`
	Timezone      string `json:"timezone"`
}

// NewSyncOptimizer creates a new sync optimizer
func NewSyncOptimizer(cfg SyncOptimizerConfig) *SyncOptimizer {
	return &SyncOptimizer{
		modelEndpoint:   cfg.ModelEndpoint,
		modelVersion:    cfg.ModelVersion,
		dbClient:        cfg.DBClient,
		featureEngineer: cfg.FeatureEngineer,
		logger:          cfg.Logger.WithField("component", "sync_optimizer"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		costConfig: cfg.CostConfig,
	}
}

// RecommendSyncTime recommends optimal sync time for a calendar
func (so *SyncOptimizer) RecommendSyncTime(ctx context.Context, userID, calendarID string) (*SyncRecommendation, error) {
	startTime := time.Now()

	// Extract features
	features, err := so.extractSyncFeatures(ctx, userID, calendarID)
	if err != nil {
		return nil, fmt.Errorf("extract features: %w", err)
	}

	// Call ML model
	recommendation, err := so.callModel(ctx, features)
	if err != nil {
		so.logger.WithError(err).Warn("ML model failed, using fallback")
		recommendation = so.fallbackRecommendation(features)
	}

	// Calculate cost savings
	baselineCost := so.calculateBaselineCost(features)
	recommendation.SavingsCents = baselineCost - recommendation.PredictedCostCents

	// Store recommendation
	err = so.storeRecommendation(ctx, userID, calendarID, features, recommendation)
	if err != nil {
		so.logger.WithError(err).Warn("Failed to store recommendation")
	}

	so.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"calendar_id":   calendarID,
		"optimal_time":  recommendation.OptimalTime,
		"batch_size":    recommendation.BatchSize,
		"savings_cents": recommendation.SavingsCents,
		"confidence":    recommendation.Confidence,
		"duration_ms":   time.Since(startTime).Milliseconds(),
	}).Info("Sync optimization recommendation")

	return recommendation, nil
}

// extractSyncFeatures extracts features for sync optimization
func (so *SyncOptimizer) extractSyncFeatures(ctx context.Context, userID, calendarID string) (*SyncFeatures, error) {
	userQuery := `SELECT id, created_at, timezone FROM users WHERE id = $1`
	var user struct {
		ID        string
		CreatedAt time.Time
		Timezone  string
	}
	if err := so.dbClient.Pool().QueryRow(ctx, userQuery, userID).Scan(&user.ID, &user.CreatedAt, &user.Timezone); err != nil {
		return nil, err
	}

	calQuery := `SELECT id, created_at, provider, sync_frequency FROM calendars WHERE id = $1`
	var calendar struct {
		ID            string
		CreatedAt     time.Time
		Provider      string
		SyncFrequency string
	}
	if err := so.dbClient.Pool().QueryRow(ctx, calQuery, calendarID).Scan(&calendar.ID, &calendar.CreatedAt, &calendar.Provider, &calendar.SyncFrequency); err != nil {
		return nil, err
	}

	syncQuery := `SELECT COUNT(*), COALESCE(AVG(duration_seconds), 0) FROM sync_jobs WHERE user_id = $1 AND calendar_id = $2`
	var syncCount int
	var avgDuration float64
	so.dbClient.Pool().QueryRow(ctx, syncQuery, userID, calendarID).Scan(&syncCount, &avgDuration)

	now := time.Now()

	features := &SyncFeatures{
		HourOfDay:       now.Hour(),
		DayOfWeek:       int(now.Weekday()),
		IsWeekend:       now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
		IsBusinessHours: now.Hour() >= 9 && now.Hour() <= 17 && now.Weekday() >= time.Monday && now.Weekday() <= time.Friday,
		UserTenureDays:  int(now.Sub(user.CreatedAt).Hours() / 24),
		TotalSyncs:      syncCount,
		AvgSyncDuration: avgDuration,
		CalendarAgeDays: int(now.Sub(calendar.CreatedAt).Hours() / 24),
		Provider:        calendar.Provider,
		SyncFrequency:   calendar.SyncFrequency,
		Timezone:        result.User.Timezone,
	}

	// Calculate event density (events per day)
	features.EventDensity = float64(features.TotalSyncs) / float64(features.CalendarAgeDays+1)

	// Calculate success rate
	features.SuccessRate = so.calculateSuccessRate(ctx, userID, calendarID)

	// Get best performance times
	bestTime, err := so.getBestPerformanceTime(ctx, userID, calendarID)
	if err == nil {
		features.BestHourOfDay = bestTime.Hour
		features.BestDayOfWeek = bestTime.DayOfWeek
	}

	return features, nil
}

// callModel calls the ML model endpoint
func (so *SyncOptimizer) callModel(ctx context.Context, features *SyncFeatures) (*SyncRecommendation, error) {
	reqBody, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", so.modelEndpoint,
		ioutil.NopCloser(bytes.NewReader(reqBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Model-Version", so.modelVersion)

	resp, err := so.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("model returned status %d: %s", resp.StatusCode, string(body))
	}

	var recommendation SyncRecommendation
	if err := json.NewDecoder(resp.Body).Decode(&recommendation); err != nil {
		return nil, err
	}

	recommendation.ModelVersion = so.modelVersion

	return &recommendation, nil
}

// fallbackRecommendation provides rule-based fallback
func (so *SyncOptimizer) fallbackRecommendation(features *SyncFeatures) *SyncRecommendation {
	now := time.Now()

	// Rule-based optimal time
	var optimalTime time.Time
	if features.BestHourOfDay > 0 {
		optimalTime = time.Date(now.Year(), now.Month(), now.Day(), features.BestHourOfDay, 0, 0, 0, now.Location())
	} else {
		// Default to 2 AM (low API usage)
		optimalTime = time.Date(now.Year(), now.Month(), now.Day(), 2, 0, 0, 0, now.Location())
	}

	// Rule-based batch size
	batchSize := 100
	if features.EventDensity > 10 {
		batchSize = 500
	} else if features.EventDensity > 5 {
		batchSize = 200
	}

	// Rule-based resource profile
	resourceProfile := "standard"
	if features.TotalEvents > 1000 {
		resourceProfile = "performance"
	} else if features.TotalEvents < 100 {
		resourceProfile = "economy"
	}

	// Calculate predicted cost
	predictedCost := so.calculatePredictedCost(features, batchSize, resourceProfile)

	return &SyncRecommendation{
		OptimalTime:        optimalTime,
		ExpectedDuration:   features.AvgSyncDuration * 0.8, // Assume 20% improvement
		BatchSize:          batchSize,
		ResourceProfile:    resourceProfile,
		Priority:           "normal",
		PredictedCostCents: predictedCost,
		Confidence:         0.6,
		Reasoning:          "Rule-based optimization (ML model unavailable)",
	}
}

// calculateBaselineCost calculates cost without optimization
func (so *SyncOptimizer) calculateBaselineCost(features *SyncFeatures) int {
	// Default batch size and resource profile
	batchSize := 100
	resourceProfile := "standard"

	return so.calculatePredictedCost(features, batchSize, resourceProfile)
}

// calculatePredictedCost calculates predicted sync cost
func (so *SyncOptimizer) calculatePredictedCost(features *SyncFeatures, batchSize int, resourceProfile string) int {
	// Estimate API calls based on events
	apiCalls := features.TotalEvents / batchSize * 2 // 2 API calls per batch (list + get)

	// Estimate compute time
	computeSeconds := features.AvgSyncDuration
	if resourceProfile == "performance" {
		computeSeconds *= 0.7 // 30% faster
	} else if resourceProfile == "economy" {
		computeSeconds *= 1.3 // 30% slower
	}

	// Estimate storage and transfer
	storageMB := float64(features.TotalEvents) * 0.001 // 1KB per event
	transferMB := storageMB * 2                        // Request + response

	// Calculate total cost
	totalCents := int(
		float64(apiCalls)*so.costConfig.APICallCostCents +
			computeSeconds*so.costConfig.ComputeCostPerSecond +
			storageMB*so.costConfig.StorageCostPerMB +
			transferMB*so.costConfig.TransferCostPerMB,
	)

	return totalCents
}

// storeRecommendation stores optimization recommendation
func (so *SyncOptimizer) storeRecommendation(ctx context.Context, userID, calendarID string, features *SyncFeatures, rec *SyncRecommendation) error {
	featuresJSON, _ := json.Marshal(features)

	query := `
		INSERT INTO sync_optimization_recommendations (
			id, tenant_id, calendar_id, user_id, recommended_sync_time, recommended_batch_size,
			recommended_resource_profile, predicted_duration_seconds, predicted_cost_cents,
			model_version, confidence_score, features_used, created_at
		) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
	`

	_, err := so.dbClient.Pool().Exec(ctx, query,
		userID, calendarID, userID, rec.OptimalTime, rec.BatchSize,
		rec.ResourceProfile, rec.ExpectedDuration, rec.PredictedCostCents,
		rec.ModelVersion, rec.Confidence, featuresJSON,
	)
	return err
}

// ScheduleOptimizedSyncs schedules all syncs at optimal times
func (so *SyncOptimizer) ScheduleOptimizedSyncs(ctx context.Context) error {
	// Get all active calendars
	calendars, err := so.getActiveCalendars(ctx)
	if err != nil {
		return err
	}

	scheduled := 0
	for _, calendar := range calendars {
		rec, err := so.RecommendSyncTime(ctx, calendar.UserID, calendar.ID)
		if err != nil {
			so.logger.WithError(err).WithField("calendar_id", calendar.ID).Warn("Failed to get sync recommendation")
			continue
		}

		// Schedule sync at optimal time
		err = so.scheduleSyncAt(ctx, calendar.ID, rec.OptimalTime)
		if err != nil {
			so.logger.WithError(err).Warn("Failed to schedule sync")
			continue
		}

		scheduled++
	}

	so.logger.WithField("scheduled", scheduled).Info("Optimized syncs scheduled")
	return nil
}

// Helper functions
func (so *SyncOptimizer) calculateSuccessRate(ctx context.Context, userID, calendarID string) float64 {
	// Query success rate from database
	return 0.95 // Placeholder
}

func (so *SyncOptimizer) getBestPerformanceTime(ctx context.Context, userID, calendarID string) (*BestPerformanceTime, error) {
	// Query best performance time from baseline table
	return &BestPerformanceTime{
		Hour:      2,
		DayOfWeek: 1,
	}, nil
}

func (so *SyncOptimizer) getActiveCalendars(ctx context.Context) ([]CalendarInfo, error) {
	// Query active calendars from database
	return []CalendarInfo{}, nil
}

func (so *SyncOptimizer) scheduleSyncAt(ctx context.Context, calendarID string, scheduledTime time.Time) error {
	// Update calendar sync schedule
	return nil
}

// BestPerformanceTime represents best performance time
type BestPerformanceTime struct {
	Hour      int `json:"hour"`
	DayOfWeek int `json:"day_of_week"`
}

// CalendarInfo represents calendar information
type CalendarInfo struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}
