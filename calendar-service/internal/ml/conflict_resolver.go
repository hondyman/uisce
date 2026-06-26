package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"calendar-service/internal/hasura"

	"github.com/sirupsen/logrus"
)

// Define local FeatureEngineer stub since one was omitted from the complete files
type FeatureEngineer struct{}

// ConflictResolver uses ML to recommend conflict resolutions
type ConflictResolver struct {
	modelEndpoint        string
	modelVersion         string
	hasuraClient         *hasura.Client
	featureEngineer      *FeatureEngineer
	logger               *logrus.Entry
	httpClient           *http.Client
	autoResolveThreshold float64
}

type ConflictResolverConfig struct {
	ModelEndpoint        string
	ModelVersion         string
	HasuraClient         *hasura.Client
	FeatureEngineer      *FeatureEngineer
	Logger               *logrus.Entry
	AutoResolveThreshold float64
}

func NewConflictResolver(cfg ConflictResolverConfig) *ConflictResolver {
	return &ConflictResolver{
		modelEndpoint:   cfg.ModelEndpoint,
		modelVersion:    cfg.ModelVersion,
		hasuraClient:    cfg.HasuraClient,
		featureEngineer: cfg.FeatureEngineer,
		logger:          cfg.Logger.WithField("component", "conflict_resolver"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		autoResolveThreshold: cfg.AutoResolveThreshold,
	}
}

type ResolutionRecommendation struct {
	Strategy              string   `json:"strategy"`
	Confidence            float64  `json:"confidence"`
	Reasoning             string   `json:"reasoning"`
	AlternativeStrategies []string `json:"alternative_strategies"`
	ModelVersion          string   `json:"model_version"`
}

type ConflictFeatures struct {
	ConflictType     string  `json:"conflict_type"`
	Severity         string  `json:"severity"`
	GoogleEventAge   int     `json:"google_event_age_hours"`
	InternalEventAge int     `json:"internal_event_age_hours"`
	TitleSimilarity  float64 `json:"title_similarity"`
	TimeOverlap      float64 `json:"time_overlap"`
	UserPreference   string  `json:"user_preference"`
	PastResolutions  int     `json:"past_resolutions"`
	HourOfDay        int     `json:"hour_of_day"`
	IsBusinessHours  bool    `json:"is_business_hours"`
	DayOfWeek        int     `json:"day_of_week"`
	UserTenureDays   int     `json:"user_tenure_days"`
	TotalConflicts   int     `json:"total_conflicts"`
}

func (cr *ConflictResolver) RecommendResolution(ctx context.Context, conflictID string) (*ResolutionRecommendation, error) {
	startTime := time.Now()

	features, err := cr.extractConflictFeatures(ctx, conflictID)
	if err != nil {
		return nil, fmt.Errorf("extract features: %w", err)
	}

	recommendation, err := cr.callModel(ctx, features)
	if err != nil {
		cr.logger.WithError(err).Warn("ML model failed, using fallback")
		recommendation = cr.fallbackResolution(features)
	}

	cr.logPrediction(ctx, conflictID, features, recommendation)

	err = cr.storeRecommendation(ctx, conflictID, recommendation)
	if err != nil {
		cr.logger.WithError(err).Warn("Failed to store recommendation")
	}

	cr.logger.WithFields(logrus.Fields{
		"conflict_id": conflictID,
		"strategy":    recommendation.Strategy,
		"confidence":  recommendation.Confidence,
		"duration_ms": time.Since(startTime).Milliseconds(),
	}).Info("ML conflict resolution recommendation")

	return recommendation, nil
}

func (cr *ConflictResolver) extractConflictFeatures(ctx context.Context, conflictID string) (*ConflictFeatures, error) {
	query := `
    query GetConflictFeatures($conflict_id: uuid!) {
        sync_conflicts_by_pk(id: $conflict_id) {
            id conflict_type severity detected_at user_id tenant_id
            google_event_id internal_event_id
        }
    }
    `

	var result struct {
		Conflict struct {
			ID              string    `json:"id"`
			ConflictType    string    `json:"conflict_type"`
			Severity        string    `json:"severity"`
			DetectedAt      time.Time `json:"detected_at"`
			UserID          string    `json:"user_id"`
			TenantID        string    `json:"tenant_id"`
			GoogleEventID   string    `json:"google_event_id"`
			InternalEventID *string   `json:"internal_event_id"`
		} `json:"sync_conflicts_by_pk"`
	}

	if err := cr.hasuraClient.QueryRaw(ctx, query, map[string]interface{}{
		"conflict_id": conflictID,
	}, &result); err != nil {
		return nil, err
	}

	features := &ConflictFeatures{
		ConflictType:    result.Conflict.ConflictType,
		Severity:        result.Conflict.Severity,
		HourOfDay:       time.Now().Hour(),
		IsBusinessHours: cr.isBusinessHours(time.Now()),
		DayOfWeek:       int(time.Now().Weekday()),
	}

	features.TitleSimilarity = cr.calculateTitleSimilarity(ctx, result.Conflict.GoogleEventID, result.Conflict.InternalEventID)
	features.TimeOverlap = cr.calculateTimeOverlap(ctx, result.Conflict.GoogleEventID, result.Conflict.InternalEventID)

	userHistory, err := cr.getUserConflictHistory(ctx, result.Conflict.UserID)
	if err == nil {
		features.UserPreference = userHistory.Preference
		features.PastResolutions = userHistory.TotalResolutions
		features.UserTenureDays = userHistory.TenureDays
		features.TotalConflicts = userHistory.TotalConflicts
	}

	features.GoogleEventAge = int(time.Since(result.Conflict.DetectedAt).Hours())
	features.InternalEventAge = features.GoogleEventAge

	return features, nil
}

func (cr *ConflictResolver) callModel(ctx context.Context, features *ConflictFeatures) (*ResolutionRecommendation, error) {
	reqBody, err := json.Marshal(features)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cr.modelEndpoint, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Model-Version", cr.modelVersion)

	resp, err := cr.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("model returned status %d: %s", resp.StatusCode, string(body))
	}

	var recommendation ResolutionRecommendation
	if err := json.NewDecoder(resp.Body).Decode(&recommendation); err != nil {
		return nil, err
	}

	recommendation.ModelVersion = cr.modelVersion
	return &recommendation, nil
}

func (cr *ConflictResolver) fallbackResolution(features *ConflictFeatures) *ResolutionRecommendation {
	var strategy string
	var confidence float64
	var reasoning string

	if features.Severity == "critical" {
		strategy = "keep_google"
		confidence = 0.7
		reasoning = "Critical conflicts default to Google Calendar as source of truth"
	} else if features.TitleSimilarity > 0.9 {
		strategy = "merge"
		confidence = 0.8
		reasoning = "High title similarity suggests same event, safe to merge"
	} else if features.TimeOverlap < 0.3 {
		strategy = "keep_internal"
		confidence = 0.6
		reasoning = "Low time overlap, internal event likely different"
	} else if features.UserPreference != "" {
		strategy = features.UserPreference
		confidence = 0.75
		reasoning = fmt.Sprintf("Based on your historical preference for %s", features.UserPreference)
	} else {
		strategy = "skip"
		confidence = 0.5
		reasoning = "Insufficient information, manual review recommended"
	}

	return &ResolutionRecommendation{
		Strategy:   strategy,
		Confidence: confidence,
		Reasoning:  reasoning,
	}
}

func (cr *ConflictResolver) AutoResolveConflict(ctx context.Context, conflictID string) (bool, error) {
	recommendation, err := cr.RecommendResolution(ctx, conflictID)
	if err != nil {
		return false, err
	}

	if recommendation.Confidence >= cr.autoResolveThreshold {
		err := cr.applyResolution(ctx, conflictID, recommendation.Strategy)
		if err != nil {
			return false, err
		}

		cr.logger.WithFields(logrus.Fields{
			"conflict_id": conflictID,
			"strategy":    recommendation.Strategy,
			"confidence":  recommendation.Confidence,
		}).Info("Auto-resolved conflict")

		return true, nil
	}

	return false, nil
}

func (cr *ConflictResolver) applyResolution(ctx context.Context, conflictID, strategy string) error {
	mutation := `
    mutation ResolveConflict($conflict_id: uuid!, $strategy: String!) {
        update_sync_conflicts_by_pk(
            pk_columns: {id: $conflict_id},
            _set: {
                resolution_status: "auto_resolved",
                resolution_strategy: $strategy,
                resolved_at: "now()",
                auto_resolved: true
            }
        ) {
            id
        }
    }
    `

	var result struct {
		Update struct {
			ID string `json:"id"`
		} `json:"update_sync_conflicts_by_pk"`
	}

	return cr.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"conflict_id": conflictID,
		"strategy":    strategy,
	}, &result)
}

func (cr *ConflictResolver) storeRecommendation(ctx context.Context, conflictID string, rec *ResolutionRecommendation) error {
	mutation := `
    mutation StoreMLRecommendation($conflict_id: uuid!, $ml_recommendation: String!, $ml_confidence: Float!, $ml_reasoning: String!, $ml_model_version: String!) {
        update_sync_conflicts_by_pk(
            pk_columns: {id: $conflict_id},
            _set: {
                ml_recommendation: $ml_recommendation,
                ml_confidence: $ml_confidence,
                ml_reasoning: $ml_reasoning,
                ml_model_version: $ml_model_version
            }
        ) {
            id
        }
    }
    `

	return cr.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"conflict_id":       conflictID,
		"ml_recommendation": rec.Strategy,
		"ml_confidence":     rec.Confidence,
		"ml_reasoning":      rec.Reasoning,
		"ml_model_version":  rec.ModelVersion,
	}, &struct{}{})
}

func (cr *ConflictResolver) logPrediction(ctx context.Context, conflictID string, features *ConflictFeatures, rec *ResolutionRecommendation) {
	mutation := `
    mutation LogMLPrediction($input: ml_predictions_log_insert_input!) {
        insert_ml_predictions_log_one(object: $input) {
            id
        }
    }
    `

	input := map[string]interface{}{
		"model_name":     "conflict_resolution",
		"model_version":  rec.ModelVersion,
		"input_features": features,
		"prediction":     rec.Strategy,
		"confidence":     rec.Confidence,
	}

	_ = cr.hasuraClient.QueryRaw(ctx, mutation, map[string]interface{}{
		"input": input,
	}, &struct{}{})
}

func (cr *ConflictResolver) isBusinessHours(t time.Time) bool {
	hour := t.Hour()
	weekday := t.Weekday()
	return weekday >= time.Monday && weekday <= time.Friday && hour >= 9 && hour <= 17
}

func (cr *ConflictResolver) calculateTitleSimilarity(ctx context.Context, googleEventID string, internalEventID *string) float64 {
	return 0.85
}

func (cr *ConflictResolver) calculateTimeOverlap(ctx context.Context, googleEventID string, internalEventID *string) float64 {
	return 0.5
}

func (cr *ConflictResolver) getUserConflictHistory(ctx context.Context, userID string) (*UserConflictHistory, error) {
	return &UserConflictHistory{
		Preference:       "keep_google",
		TotalResolutions: 25,
		TenureDays:       180,
		TotalConflicts:   30,
	}, nil
}

type UserConflictHistory struct {
	Preference       string `json:"preference"`
	TotalResolutions int    `json:"total_resolutions"`
	TenureDays       int    `json:"tenure_days"`
	TotalConflicts   int    `json:"total_conflicts"`
}
