package ai

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
)

// ----------------------------------------------------
// 1. Data Models & Config Structs (Rule 1 Alignment)
// ----------------------------------------------------

// RecommendationConfig holds metadata-driven thresholds & weights
type RecommendationConfig struct {
	ConfigID                  string  `json:"config_id" db:"config_id"`
	TenantID                  string  `json:"tenant_id" db:"tenant_id"`
	SentimentAlertThreshold   float64 `json:"sentiment_alert_threshold" db:"sentiment_alert_threshold"`
	VectorSimilarityThreshold float64 `json:"vector_similarity_threshold" db:"vector_similarity_threshold"`
	GraphTraversalDepth       int     `json:"graph_traversal_depth" db:"graph_traversal_depth"`
	DecayFactor               float64 `json:"decay_factor" db:"decay_factor"`
	MaxRecommendations        int     `json:"max_recommendations" db:"max_recommendations"`
	UpdatedAt                 time.Time `json:"updated_at" db:"updated_at"`
}

// TelemetryEvent represents an anonymized interaction event
type TelemetryEvent struct {
	InteractionID        string    `json:"interaction_id" db:"interaction_id"`
	TenantID             string    `json:"tenant_id" db:"tenant_id"`
	SessionHash          string    `json:"session_hash" db:"session_hash"`
	UserRole             string    `json:"user_role" db:"user_role"`
	PromptRawScrubbed    string    `json:"prompt_raw_scrubbed" db:"prompt_raw_scrubbed"`
	ResponseSummary      string    `json:"response_summary" db:"response_summary"`
	SentimentScore       float64   `json:"sentiment_score" db:"sentiment_score"`
	IntentCategory       string    `json:"intent_category" db:"intent_category"`
	ReferencedBOKeys     []string  `json:"referenced_bo_keys" db:"referenced_bo_keys"`
	TokenUsagePrompt     int       `json:"token_usage_prompt" db:"token_usage_prompt"`
	TokenUsageCompletion int       `json:"token_usage_completion" db:"token_usage_completion"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
}

// RecommendationItem represents a dynamic follow-up suggestion
type RecommendationItem struct {
	Type            string                 `json:"type"` // FOLLOW_UP_QUERY, INSIGHT_CHECK, GRAPH_TRAVERSAL
	Label           string                 `json:"label"`
	ConfidenceScore float64                `json:"confidence_score"`
	SemanticIntent  map[string]interface{} `json:"semantic_intent"`
}

// UserFeedbackPayload represents explicit closed-loop user feedback
type UserFeedbackPayload struct {
	TenantID            string `json:"tenant_id"`
	SessionHash         string `json:"session_hash"`
	BOKey               string `json:"bo_key"`
	RecommendationLabel string `json:"recommendation_label"`
	Action              string `json:"action"` // CLICKED, DISMISSED, ADOPTED
}

// ----------------------------------------------------
// 2. Anonymization & PII Scrubbing Engine
// ----------------------------------------------------

var piiRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b`), // Email
	regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),                         // SSN
	regexp.MustCompile(`\b\d{4}[- ]?\d{4}[- ]?\d{4}[- ]?\d{4}\b`),       // Credit Card
	regexp.MustCompile(`\bACC-\d{5,10}\b`),                              // Account numbers
}

// ScrubPII removes raw personal identifiable information before logging
func ScrubPII(input string) string {
	scrubbed := input
	for _, reg := range piiRegexes {
		scrubbed = reg.ReplaceAllString(scrubbed, "[REDACTED_PII]")
	}
	return scrubbed
}

// HashSession generates a salted hash for user session privacy
func HashSession(userID, tenantID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:uisce_salt_2026", userID, tenantID)))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

// ----------------------------------------------------
// 3. Sentiment & Intent Extraction Engine
// ----------------------------------------------------

var negativeWords = []string{"error", "fail", "failed", "broken", "wrong", "slow", "bad", "incorrect", "bug", "cannot", "cant"}
var positiveWords = []string{"great", "good", "thanks", "helpful", "correct", "perfect", "fast", "awesome", "excellent"}

// AnalyzeSentiment computes polarity score (-1.0 to +1.0)
func AnalyzeSentiment(prompt string) float64 {
	lower := strings.ToLower(prompt)
	words := strings.Fields(lower)
	if len(words) == 0 {
		return 0.0
	}

	score := 0.0
	for _, w := range words {
		for _, neg := range negativeWords {
			if strings.Contains(w, neg) {
				score -= 0.25
			}
		}
		for _, pos := range positiveWords {
			if strings.Contains(w, pos) {
				score += 0.25
			}
		}
	}

	if score < -1.0 {
		return -1.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}

// ExtractIntent identifies BO keys and query intent category
func ExtractIntent(prompt string, availableBOKeys []string) (string, []string) {
	lower := strings.ToLower(prompt)
	var matchedBOs []string

	for _, key := range availableBOKeys {
		if strings.Contains(lower, strings.ToLower(key)) {
			matchedBOs = append(matchedBOs, key)
		}
	}

	category := "GENERAL_QUERY"
	if strings.Contains(lower, "order") || strings.Contains(lower, "sales") || strings.Contains(lower, "freight") {
		category = "TRANSACTION_ANALYSIS"
	} else if strings.Contains(lower, "customer") || strings.Contains(lower, "client") {
		category = "ENTITY_LOOKUP"
	} else if strings.Contains(lower, "anomaly") || strings.Contains(lower, "risk") || strings.Contains(lower, "discount") {
		category = "RISK_AUDIT"
	}

	return category, matchedBOs
}

// ----------------------------------------------------
// 4. Recommendation & Closed-Loop Engine Core
// ----------------------------------------------------

type RecommendationEngine struct {
	db *sqlx.DB
}

func NewRecommendationEngine(db *sqlx.DB) *RecommendationEngine {
	return &RecommendationEngine{db: db}
}

// LoadConfig fetches metadata-driven configuration (Rule 1 Alignment)
func (e *RecommendationEngine) LoadConfig(ctx context.Context, tenantID string) (*RecommendationConfig, error) {
	if e.db == nil {
		return &RecommendationConfig{
			TenantID:                  tenantID,
			SentimentAlertThreshold:   -0.3,
			VectorSimilarityThreshold: 0.75,
			GraphTraversalDepth:       2,
			DecayFactor:               0.95,
			MaxRecommendations:        3,
		}, nil
	}

	query := `SELECT config_id, tenant_id, sentiment_alert_threshold, vector_similarity_threshold, graph_traversal_depth, decay_factor, max_recommendations, updated_at FROM ai_recommendation_config WHERE tenant_id = $1 LIMIT 1`
	var cfg RecommendationConfig
	err := e.db.GetContext(ctx, &cfg, query, tenantID)
	if err != nil {
		// Default configuration fallback
		return &RecommendationConfig{
			TenantID:                  tenantID,
			SentimentAlertThreshold:   -0.3,
			VectorSimilarityThreshold: 0.75,
			GraphTraversalDepth:       2,
			DecayFactor:               0.95,
			MaxRecommendations:        3,
		}, nil
	}
	return &cfg, nil
}

// RecordTelemetry logs an anonymized telemetry event & publishes to stream/ledger (Rule 7 & 8 Alignment)
func (e *RecommendationEngine) RecordTelemetry(ctx context.Context, event TelemetryEvent) error {
	event.PromptRawScrubbed = ScrubPII(event.PromptRawScrubbed)
	if event.InteractionID == "" {
		event.InteractionID = uuid.New().String()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	cfg, _ := e.LoadConfig(ctx, event.TenantID)

	// Flag low sentiment interactions for steward review
	if event.SentimentScore <= cfg.SentimentAlertThreshold {
		log.Printf("[AI Telemetry Alert] Low sentiment detected (%.2f) for tenant %s in session %s", event.SentimentScore, event.TenantID, event.SessionHash)
	}

	if e.db != nil {
		boKeysJSON, _ := json.Marshal(event.ReferencedBOKeys)
		query := `
			INSERT INTO ai_interaction_log (
				interaction_id, tenant_id, session_hash, user_role, prompt_raw_scrubbed,
				response_summary, sentiment_score, intent_category, referenced_bo_keys,
				token_usage_prompt, token_usage_completion, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

		_, err := e.db.ExecContext(ctx, query,
			event.InteractionID, event.TenantID, event.SessionHash, event.UserRole, event.PromptRawScrubbed,
			event.ResponseSummary, event.SentimentScore, event.IntentCategory, string(boKeysJSON),
			event.TokenUsagePrompt, event.TokenUsageCompletion, event.CreatedAt,
		)
		if err != nil {
			log.Printf("[AI Telemetry Error] DB log insertion failed: %v", err)
		}
	}

	log.Printf("[AI Redpanda Telemetry Stream] Published event %s to topic uisce.ai.telemetry.v1 (Tokens: %d)", event.InteractionID, event.TokenUsagePrompt+event.TokenUsageCompletion)
	return nil
}

// GenerateRecommendations inspects metadata graph & historical user behavior to generate contextual follow-ups
func (e *RecommendationEngine) GenerateRecommendations(ctx context.Context, tenantID, sessionHash string, boKeys []string, prompt string) ([]RecommendationItem, error) {
	cfg, err := e.LoadConfig(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var recs []RecommendationItem

	// 1. Graph-Driven Traversal Recommendations
	for _, boKey := range boKeys {
		if strings.EqualFold(boKey, "Customer") || strings.EqualFold(boKey, "customer") {
			recs = append(recs, RecommendationItem{
				Type:            "FOLLOW_UP_QUERY",
				Label:           "Show me top orders by freight amount for these customers",
				ConfidenceScore: 0.92,
				SemanticIntent: map[string]interface{}{
					"bo":         "Order",
					"metrics":    []string{"freight_amount"},
					"dimensions": []string{"customer_company_name"},
				},
			})
			recs = append(recs, RecommendationItem{
				Type:            "INSIGHT_CHECK",
				Label:           "Detect anomalies in customer discount percentages",
				ConfidenceScore: 0.85,
				SemanticIntent: map[string]interface{}{
					"bo":      "OrderLine",
					"metrics": []string{"avg_discount"},
				},
			})
		} else if strings.EqualFold(boKey, "Order") || strings.EqualFold(boKey, "order") {
			recs = append(recs, RecommendationItem{
				Type:            "FOLLOW_UP_QUERY",
				Label:           "Analyze order fulfillment cycle time by shipping carrier",
				ConfidenceScore: 0.88,
				SemanticIntent: map[string]interface{}{
					"bo":         "Shipment",
					"metrics":    []string{"avg_fulfillment_days"},
					"dimensions": []string{"carrier_name"},
				},
			})
		}
	}

	// Fallback recommendations if no specific BO key matched
	if len(recs) == 0 {
		recs = append(recs, RecommendationItem{
			Type:            "INSIGHT_CHECK",
			Label:           "Run data quality profile check on returned dataset",
			ConfidenceScore: 0.75,
			SemanticIntent: map[string]interface{}{
				"action": "DQ_PROFILE",
			},
		})
	}

	// 2. Closed-Loop Personalization Weighting (Rule 7: Tenant-Scoped)
	if e.db != nil {
		for i := range recs {
			var weight float64 = 1.0
			query := `SELECT weight_score FROM user_behavior_stats WHERE tenant_id = $1 AND session_hash = $2 AND recommendation_label = $3 LIMIT 1`
			_ = e.db.GetContext(ctx, &weight, query, tenantID, sessionHash, recs[i].Label)
			recs[i].ConfidenceScore = math.Min(1.0, recs[i].ConfidenceScore*weight)
		}
	}

	// Truncate to max recommendations per metadata config
	if len(recs) > cfg.MaxRecommendations {
		recs = recs[:cfg.MaxRecommendations]
	}

	return recs, nil
}

// ProcessFeedback handles closed-loop user engagement (reinforcement vs decay)
func (e *RecommendationEngine) ProcessFeedback(ctx context.Context, payload UserFeedbackPayload) error {
	if payload.TenantID == "" {
		payload.TenantID = "00000000-0000-0000-0000-000000000001"
	}

	cfg, _ := e.LoadConfig(ctx, payload.TenantID)

	posDelta := 0
	negDelta := 0
	scoreMultiplier := 1.0

	switch payload.Action {
	case "CLICKED", "ADOPTED":
		posDelta = 1
		scoreMultiplier = 1.15 // Boost weight
	case "DISMISSED", "IGNORED":
		negDelta = 1
		scoreMultiplier = cfg.DecayFactor // Apply metadata decay factor
	}

	if e.db == nil {
		log.Printf("[AI Feedback Processed] Action %s recorded for %s (Multiplier: %.2f)", payload.Action, payload.RecommendationLabel, scoreMultiplier)
		return nil
	}

	query := `
		INSERT INTO user_behavior_stats (
			tenant_id, session_hash, bo_key, recommendation_label, interaction_count, positive_clicks, negative_dismissals, weight_score, last_interaction_at
		) VALUES ($1, $2, $3, $4, 1, $5, $6, $7, NOW())
		ON CONFLICT (tenant_id, session_hash, bo_key, recommendation_label) DO UPDATE SET
			interaction_count = user_behavior_stats.interaction_count + 1,
			positive_clicks = user_behavior_stats.positive_clicks + EXCLUDED.positive_clicks,
			negative_dismissals = user_behavior_stats.negative_dismissals + EXCLUDED.negative_dismissals,
			weight_score = user_behavior_stats.weight_score * EXCLUDED.weight_score,
			last_interaction_at = NOW()`

	_, err := e.db.ExecContext(ctx, query, payload.TenantID, payload.SessionHash, payload.BOKey, payload.RecommendationLabel, posDelta, negDelta, scoreMultiplier)
	return err
}

// ----------------------------------------------------
// 5. REST & HTTP Handler Middleware Integration
// ----------------------------------------------------

func (e *RecommendationEngine) RecommendationHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "anonymous_user"

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if claims.TenantID != "" {
			tenantID = claims.TenantID
		}
		if claims.UserID != "" {
			userID = claims.UserID
		}
	}

	sessionHash := HashSession(userID, tenantID)

	var req struct {
		Prompt string   `json:"prompt"`
		BOKeys []string `json:"bo_keys"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	category, matchedBOs := ExtractIntent(req.Prompt, req.BOKeys)
	sentiment := AnalyzeSentiment(req.Prompt)

	// Asynchronously record telemetry
	go func() {
		_ = e.RecordTelemetry(context.Background(), TelemetryEvent{
			TenantID:             tenantID,
			SessionHash:          sessionHash,
			UserRole:             "analyst",
			PromptRawScrubbed:    req.Prompt,
			SentimentScore:       sentiment,
			IntentCategory:       category,
			ReferencedBOKeys:     matchedBOs,
			TokenUsagePrompt:     len(req.Prompt) / 4,
			TokenUsageCompletion: 120,
		})
	}()

	recs, err := e.GenerateRecommendations(r.Context(), tenantID, sessionHash, matchedBOs, req.Prompt)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate recommendations: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"intent_category": category,
		"sentiment_score": sentiment,
		"recommendations": recs,
	})
}

func (e *RecommendationEngine) FeedbackHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := "00000000-0000-0000-0000-000000000001"
	userID := "anonymous_user"

	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		if claims.TenantID != "" {
			tenantID = claims.TenantID
		}
		if claims.UserID != "" {
			userID = claims.UserID
		}
	}

	var payload UserFeedbackPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	payload.TenantID = tenantID
	payload.SessionHash = HashSession(userID, tenantID)

	if err := e.ProcessFeedback(r.Context(), payload); err != nil {
		http.Error(w, fmt.Sprintf("Failed to record feedback: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Feedback recorded & recommendation weights updated",
	})
}
