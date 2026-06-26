package ai_routing

import (
	"log"
	"strings"
	"time"
)

// SentimentClassifier analyzes text sentiment
type SentimentClassifier struct {
	positiveKeywords map[string]float64
	negativeKeywords map[string]float64
	neutralKeywords  map[string]float64
}

// SentimentResult represents sentiment analysis output
type SentimentResult struct {
	CompoundScore   float64
	Confidence      float64
	DominantEmotion string
	PositiveScore   float64
	NegativeScore   float64
	NeutralScore    float64
}

// NewSentimentClassifier initializes sentiment analyzer
func NewSentimentClassifier() *SentimentClassifier {
	return &SentimentClassifier{
		positiveKeywords: map[string]float64{
			"excellent": 0.8, "great": 0.7, "good": 0.6, "happy": 0.7, "satisfied": 0.6,
			"perfect": 0.9, "amazing": 0.8, "wonderful": 0.8, "fantastic": 0.8,
			"best": 0.8, "love": 0.9, "brilliant": 0.8,
		},
		negativeKeywords: map[string]float64{
			"bad": -0.6, "terrible": -0.8, "awful": -0.8, "horrible": -0.9,
			"poor": -0.7, "worse": -0.8, "hate": -0.9, "angry": -0.8,
			"frustrated": -0.7, "disappointed": -0.7, "useless": -0.8,
		},
		neutralKeywords: map[string]float64{
			"ok": 0.1, "fine": 0.2, "average": 0.0, "normal": 0.0,
		},
	}
}

// AnalyzeBatch analyzes multiple text samples
func (sc *SentimentClassifier) AnalyzeBatch(texts []string) *SentimentResult {
	totalCompound := 0.0
	sentimentCount := 0

	for _, text := range texts {
		sentiment := sc.analyzeSingle(text)
		totalCompound += sentiment.CompoundScore
		sentimentCount++
	}

	avgCompound := 0.0
	if sentimentCount > 0 {
		avgCompound = totalCompound / float64(sentimentCount)
	}

	// Determine dominant emotion
	dominantEmotion := "neutral"
	if avgCompound > 0.3 {
		dominantEmotion = "positive"
	} else if avgCompound < -0.3 {
		dominantEmotion = "negative"
	}

	return &SentimentResult{
		CompoundScore:   avgCompound,
		Confidence:      getConfidenceFromCompound(avgCompound),
		DominantEmotion: dominantEmotion,
	}
}

// analyzeSingle analyzes a single text string
func (sc *SentimentClassifier) analyzeSingle(text string) *SentimentResult {
	lower := strings.ToLower(text)
	words := strings.FieldsFunc(lower, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == '!' || r == '?'
	})

	positiveScore := 0.0
	negativeScore := 0.0
	neutralScore := 0.0

	for _, word := range words {
		if score, ok := sc.positiveKeywords[word]; ok {
			positiveScore += score
		} else if score, ok := sc.negativeKeywords[word]; ok {
			negativeScore += score
		} else if score, ok := sc.neutralKeywords[word]; ok {
			neutralScore += score
		}
	}

	compound := (positiveScore + negativeScore) / (float64(len(words)) + 1.0)

	dominantEmotion := "neutral"
	if positiveScore > negativeScore {
		dominantEmotion = "positive"
	} else if negativeScore > positiveScore {
		dominantEmotion = "negative"
	}

	return &SentimentResult{
		CompoundScore:   compound,
		Confidence:      getConfidenceFromCompound(compound),
		DominantEmotion: dominantEmotion,
		PositiveScore:   positiveScore,
		NegativeScore:   negativeScore,
		NeutralScore:    neutralScore,
	}
}

// HybridRuleEngine combines rule-based routing logic
type HybridRuleEngine struct {
	rules []RoutingRule
}

// RoutingRule represents a conditional routing rule
type RoutingRule struct {
	Name      string
	Priority  int
	Condition func(req RoutingRequest) bool
	BranchID  string
	Reason    string
}

// NewHybridRuleEngine creates rule engine with default rules
func NewHybridRuleEngine() *HybridRuleEngine {
	return &HybridRuleEngine{
		rules: []RoutingRule{
			{
				Name:     "VIP_Priority",
				Priority: 100,
				Condition: func(req RoutingRequest) bool {
					val, ok := req.Data["is_vip"]
					return ok && val == true
				},
				Reason: "VIP customer prioritized",
			},
			{
				Name:     "High_Value_Transaction",
				Priority: 90,
				Condition: func(req RoutingRequest) bool {
					val, ok := req.Data["order_amount"]
					if ok {
						if f, ok := val.(float64); ok {
							return f > 10000
						}
					}
					return false
				},
				Reason: "High-value transaction requires specialized handling",
			},
			{
				Name:     "Risk_Assessment",
				Priority: 80,
				Condition: func(req RoutingRequest) bool {
					val, ok := req.Data["risk_score"]
					if ok {
						if f, ok := val.(float64); ok {
							return f > 0.7
						}
					}
					return false
				},
				Reason: "High-risk transaction routed for review",
			},
		},
	}
}

// FallbackRoute returns a safe fallback routing decision
func (engine *HybridRuleEngine) FallbackRoute(req RoutingRequest) *RoutingDecision {
	// Sort rules by priority
	for i := 0; i < len(engine.rules); i++ {
		for j := i + 1; j < len(engine.rules); j++ {
			if engine.rules[j].Priority > engine.rules[i].Priority {
				engine.rules[i], engine.rules[j] = engine.rules[j], engine.rules[i]
			}
		}
	}

	// Apply rules in priority order
	for _, rule := range engine.rules {
		if rule.Condition(req) {
			// Find branch with this ID
			for _, branch := range req.AvailableBranches {
				if branch.ID == rule.BranchID {
					return &RoutingDecision{
						SelectedBranchID:  branch.ID,
						Confidence:        0.7,
						Reasoning:         []string{rule.Reason},
						ModelScores:       map[string]float64{"rule_engine": 1.0},
						ExecutionStrategy: "immediate",
					}
				}
			}
		}
	}

	// Default fallback: return first available branch
	if len(req.AvailableBranches) > 0 {
		return &RoutingDecision{
			SelectedBranchID:  req.AvailableBranches[0].ID,
			Confidence:        0.5,
			Reasoning:         []string{"Default fallback to first available branch"},
			ModelScores:       map[string]float64{"fallback": 1.0},
			ExecutionStrategy: "delayed",
		}
	}

	return nil
}

// AddRule adds a new routing rule
func (engine *HybridRuleEngine) AddRule(rule RoutingRule) {
	engine.rules = append(engine.rules, rule)
}

// RoutingMetricsCollector tracks routing performance
type RoutingMetricsCollector struct {
	decisions       []RoutingDecision
	outcomes        []WorkflowOutcome
	startTime       time.Time
	decisionsToday  int
	successToday    int
	totalLatencyMs  float64
	modelAgreements []float64
}

// NewRoutingMetricsCollector creates metric collector
func NewRoutingMetricsCollector() *RoutingMetricsCollector {
	return &RoutingMetricsCollector{
		decisions:       make([]RoutingDecision, 0),
		outcomes:        make([]WorkflowOutcome, 0),
		startTime:       time.Now(),
		modelAgreements: make([]float64, 0),
	}
}

// LogDecision records a routing decision
func (mc *RoutingMetricsCollector) LogDecision(decision *RoutingDecision, features Features, latency time.Duration) {
	mc.decisions = append(mc.decisions, *decision)
	mc.decisionsToday++
	mc.totalLatencyMs += float64(latency.Milliseconds())
	mc.modelAgreements = append(mc.modelAgreements, decision.Confidence)

	log.Printf("Routing Decision: branch=%s, confidence=%.2f, latency=%dms",
		decision.SelectedBranchID, decision.Confidence, latency.Milliseconds())
}

// LogOutcome records workflow completion
func (mc *RoutingMetricsCollector) LogOutcome(outcome WorkflowOutcome) {
	mc.outcomes = append(mc.outcomes, outcome)
	if outcome.Success {
		mc.successToday++
	}
}

// GetMetrics returns current metrics
func (mc *RoutingMetricsCollector) GetMetrics() RoutingMetrics {
	avgAgreement := 0.0
	if len(mc.modelAgreements) > 0 {
		sum := 0.0
		for _, agreement := range mc.modelAgreements {
			sum += agreement
		}
		avgAgreement = sum / float64(len(mc.modelAgreements))
	}

	avgLatency := 0.0
	if mc.decisionsToday > 0 {
		avgLatency = mc.totalLatencyMs / float64(mc.decisionsToday)
	}

	successRate := 0.0
	if mc.decisionsToday > 0 {
		successRate = float64(mc.successToday) / float64(mc.decisionsToday)
	}

	return RoutingMetrics{
		OverallAccuracy:      successRate,
		AvgDecisionTimeMs:    avgLatency,
		ModelAgreementRate:   avgAgreement,
		WorkflowsRoutedToday: mc.decisionsToday,
		LastRetrainTime:      time.Now(),
	}
}

// Helper function
func getConfidenceFromCompound(compound float64) float64 {
	absCompound := compound
	if absCompound < 0 {
		absCompound = -absCompound
	}
	// Higher confidence when compound score is more extreme
	return 0.3 + (absCompound * 0.7)
}
