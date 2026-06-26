package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// DriftSignal represents a detected change in a semantic term's behaviour
type DriftSignal struct {
	TermID         string    `json:"term_id"`
	TermName       string    `json:"term_name"`
	SignalType     string    `json:"signal_type"` // value_change, schema_change, coverage_drop, frequency_spike
	Magnitude      float64   `json:"magnitude"`   // 0.0 - 1.0
	DetectedAt     time.Time `json:"detected_at"`
	BusinessObject string    `json:"business_object"`
	Region         string    `json:"region"`
}

// DriftPrediction describes a predicted future drift event
type DriftPrediction struct {
	ID                uuid.UUID     `json:"id"`
	TenantID          uuid.UUID     `json:"tenant_id"`
	TermID            string        `json:"term_id"`
	TermName          string        `json:"term_name"`
	PredictedAt       time.Time     `json:"predicted_at"`
	ExpectedWindow    string        `json:"expected_window"` // "24h", "7d", "30d"
	Probability       float64       `json:"probability"`     // 0.0 - 1.0
	Severity          string        `json:"severity"`        // LOW, MEDIUM, HIGH, CRITICAL
	DriftType         string        `json:"drift_type"`
	AffectedObjects   []string      `json:"affected_objects"`
	MitigationSteps   []string      `json:"mitigation_steps"`
	HistoricalSignals []DriftSignal `json:"historical_signals"`
	Explanation       string        `json:"explanation"`
	ConfidenceScore   int           `json:"confidence_score"` // 0-100
}

// DriftHistory holds a recorded historical drift event for model training
type DriftHistory struct {
	ID             uuid.UUID              `json:"id"`
	TenantID       uuid.UUID              `json:"tenant_id"`
	TermID         string                 `json:"term_id"`
	DriftType      string                 `json:"drift_type"`
	Magnitude      float64                `json:"magnitude"`
	OccurredAt     time.Time              `json:"occurred_at"`
	ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
	BusinessImpact string                 `json:"business_impact"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// DriftPredictor analyses semantic drift patterns and predicts future drift
type DriftPredictor struct {
	db          *sql.DB
	llmProvider llm.LLMProvider
}

// NewDriftPredictor creates a new drift prediction engine
func NewDriftPredictor(db *sql.DB, llmProvider interface{}) *DriftPredictor {
	provider, _ := llmProvider.(llm.LLMProvider)
	return &DriftPredictor{db: db, llmProvider: provider}
}

// GetDriftPredictions analyses historical drift patterns and predicts future events
func (d *DriftPredictor) GetDriftPredictions(ctx context.Context, tenantID uuid.UUID, params DriftPredictionParams) ([]DriftPrediction, error) {
	// 1. Fetch historical drift signals from the training data store
	signals, err := d.fetchHistoricalSignals(ctx, tenantID, params)
	if err != nil {
		return nil, fmt.Errorf("fetch historical signals: %w", err)
	}

	// 2. Cluster signals by semantic term
	clusters := d.clusterSignalsByTerm(signals)

	// 3. For each term cluster, compute a prediction
	predictions := make([]DriftPrediction, 0, len(clusters))
	for termID, termSignals := range clusters {
		pred := d.predictForTerm(ctx, tenantID, termID, termSignals, params)
		if pred.Probability >= 0.15 { // Only surface meaningful predictions
			predictions = append(predictions, pred)
		}
	}

	// 4. If LLM is available, enrich the top predictions with natural-language explanation
	if d.llmProvider != nil {
		d.enrichWithLLM(ctx, predictions)
	}

	return predictions, nil
}

// DriftPredictionParams scopes a drift prediction query
type DriftPredictionParams struct {
	BusinessObject string    `json:"business_object"`
	SemanticTerm   string    `json:"semantic_term"`
	Region         string    `json:"region"`
	LookbackDays   int       `json:"lookback_days"` // how far back to scan (default 90)
	AsOf           time.Time `json:"as_of"`
}

// fetchHistoricalSignals retrieves drift events from ai_training_data
func (d *DriftPredictor) fetchHistoricalSignals(ctx context.Context, tenantID uuid.UUID, params DriftPredictionParams) ([]DriftSignal, error) {
	lookback := params.LookbackDays
	if lookback == 0 {
		lookback = 90
	}
	cutoff := time.Now().AddDate(0, 0, -lookback)

	query := `
		SELECT id, input, created_at
		FROM edm.ai_training_data
		WHERE tenant_id = $1
		  AND source = 'drift_observation'
		  AND created_at >= $2
		ORDER BY created_at DESC
		LIMIT 500`

	rows, err := d.db.QueryContext(ctx, query, tenantID, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var signals []DriftSignal
	for rows.Next() {
		var id uuid.UUID
		var inputJSON []byte
		var createdAt time.Time
		if err := rows.Scan(&id, &inputJSON, &createdAt); err != nil {
			continue
		}
		var signal DriftSignal
		if err := json.Unmarshal(inputJSON, &signal); err == nil {
			if signal.DetectedAt.IsZero() {
				signal.DetectedAt = createdAt
			}
			// Apply filters
			if params.BusinessObject != "" && signal.BusinessObject != params.BusinessObject {
				continue
			}
			if params.SemanticTerm != "" && signal.TermName != params.SemanticTerm {
				continue
			}
			if params.Region != "" && signal.Region != params.Region {
				continue
			}
			signals = append(signals, signal)
		}
	}
	return signals, nil
}

// clusterSignalsByTerm groups signals by their semantic term ID
func (d *DriftPredictor) clusterSignalsByTerm(signals []DriftSignal) map[string][]DriftSignal {
	clusters := map[string][]DriftSignal{}
	for _, s := range signals {
		key := s.TermID
		if key == "" {
			key = s.TermName
		}
		clusters[key] = append(clusters[key], s)
	}
	return clusters
}

// predictForTerm applies a simple exponential-weighted recency model to a set of signals
func (d *DriftPredictor) predictForTerm(ctx context.Context, tenantID uuid.UUID, termID string, signals []DriftSignal, params DriftPredictionParams) DriftPrediction {
	now := time.Now()

	// Compute weighted average magnitude, weighting recent signals more heavily
	var weightedMag, totalWeight float64
	driftTypes := map[string]int{}
	affectedBOs := map[string]struct{}{}

	for i, s := range signals {
		age := now.Sub(s.DetectedAt).Hours() / 24 // days
		weight := math.Exp(-0.05 * age)           // decay: λ=0.05
		// Boost weight for the most recent 10 signals
		if i < 10 {
			weight *= 1.5
		}
		weightedMag += s.Magnitude * weight
		totalWeight += weight
		driftTypes[s.SignalType]++
		if s.BusinessObject != "" {
			affectedBOs[s.BusinessObject] = struct{}{}
		}
	}

	probability := 0.0
	if totalWeight > 0 {
		probability = math.Min(weightedMag/totalWeight*1.2, 1.0)
	}

	// Dominant drift type
	dominantType := mostFrequent(driftTypes)

	// Collect affected business objects
	bos := make([]string, 0, len(affectedBOs))
	for bo := range affectedBOs {
		bos = append(bos, bo)
	}

	// Determine expected window based on frequency
	window := predictWindow(signals, now)

	// Mitigation steps by type
	mitigations := mitigationStepsFor(dominantType)

	termName := termID
	if len(signals) > 0 {
		termName = signals[0].TermName
	}

	return DriftPrediction{
		ID:                uuid.New(),
		TenantID:          tenantID,
		TermID:            termID,
		TermName:          termName,
		PredictedAt:       now,
		ExpectedWindow:    window,
		Probability:       probability,
		Severity:          severityFromProbability(probability),
		DriftType:         dominantType,
		AffectedObjects:   bos,
		MitigationSteps:   mitigations,
		HistoricalSignals: signals,
		Explanation:       fmt.Sprintf("Based on %d historical signals, drift probability is %.0f%% within %s.", len(signals), probability*100, window),
		ConfidenceScore:   confidenceScore(len(signals)),
	}
}

// enrichWithLLM appends LLM-generated explanation to the top 3 predictions
func (d *DriftPredictor) enrichWithLLM(ctx context.Context, predictions []DriftPrediction) {
	limit := 3
	if len(predictions) < limit {
		limit = len(predictions)
	}
	for i := 0; i < limit; i++ {
		p := &predictions[i]
		signalSummary, _ := json.Marshal(p.HistoricalSignals)
		prompt := fmt.Sprintf(`You are a semantic data quality expert. A term named "%s" shows %.0f%% drift probability within %s.
Dominant signal type: %s. Affected business objects: %v.
Historical signals (last %d): %s

In 2-3 sentences, explain the root cause and recommended action. Be specific and actionable.`,
			p.TermName, p.Probability*100, p.ExpectedWindow,
			p.DriftType, p.AffectedObjects, len(p.HistoricalSignals), string(signalSummary))

		if resp, err := d.llmProvider.GenerateResponse(ctx, prompt); err == nil && resp != "" {
			p.Explanation = cleanJSON(resp)
		}
	}
}

// RecordDriftSignal persists an observed drift event as training data
func (d *DriftPredictor) RecordDriftSignal(ctx context.Context, tenantID uuid.UUID, signal DriftSignal) error {
	signal.DetectedAt = time.Now()
	inputJSON, err := json.Marshal(signal)
	if err != nil {
		return err
	}
	_, err = d.db.ExecContext(ctx, `
		INSERT INTO edm.ai_training_data (id, tenant_id, source, input, output, explainability, created_at)
		VALUES (gen_random_uuid(), $1, 'drift_observation', $2, '{}', 0, NOW())
	`, tenantID, inputJSON)
	return err
}

// --- Helper functions ---

func mostFrequent(m map[string]int) string {
	best, max := "", 0
	for k, v := range m {
		if v > max {
			best, max = k, v
		}
	}
	return best
}

func predictWindow(signals []DriftSignal, now time.Time) string {
	if len(signals) == 0 {
		return "30d"
	}
	// Average interval between signals
	if len(signals) < 2 {
		return "7d"
	}
	earliest := signals[len(signals)-1].DetectedAt
	span := now.Sub(earliest).Hours() / 24
	avgInterval := span / float64(len(signals))
	switch {
	case avgInterval <= 1:
		return "24h"
	case avgInterval <= 7:
		return "7d"
	default:
		return "30d"
	}
}

func severityFromProbability(p float64) string {
	switch {
	case p >= 0.75:
		return "CRITICAL"
	case p >= 0.50:
		return "HIGH"
	case p >= 0.25:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

func confidenceScore(n int) int {
	// More signals → higher model confidence, capped at 95
	score := n * 5
	if score > 95 {
		score = 95
	}
	return score
}

func mitigationStepsFor(driftType string) []string {
	m := map[string][]string{
		"value_change": {
			"Review source system changes in the last 30 days",
			"Update semantic term validation rules to reflect new value ranges",
			"Notify downstream consumers of the semantic term",
		},
		"schema_change": {
			"Run schema diff report on the source system",
			"Update business object field mappings",
			"Trigger a re-validation of affected rules",
		},
		"coverage_drop": {
			"Check source system data completeness reports",
			"Enable fallback source preference for the affected region",
			"Flag affected records for manual review",
		},
		"frequency_spike": {
			"Investigate upstream event triggers for abnormal activity",
			"Apply rate-limiting rules to prevent data flooding",
			"Quarantine anomalous records pending review",
		},
	}
	if steps, ok := m[driftType]; ok {
		return steps
	}
	return []string{"Investigate the anomaly pattern", "Review semantic term governance rules"}
}
