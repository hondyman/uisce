package ai_routing

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// IntelligentRouter is the main routing engine combining multiple AI models
type IntelligentRouter struct {
	predictiveModel    *PredictiveRoutingModel
	reinforcementAgent *RLRoutingAgent
	sentimentAnalyzer  *SentimentClassifier
	ruleEngine         *HybridRuleEngine
	metricsCollector   *RoutingMetricsCollector
	modelTimeout       time.Duration
	decisionCache      map[string]*RoutingDecision
	cacheMutex         sync.RWMutex
}

// NewIntelligentRouter creates a new routing engine
func NewIntelligentRouter(
	predictiveModel *PredictiveRoutingModel,
	rlAgent *RLRoutingAgent,
	sentimentAnalyzer *SentimentClassifier,
	ruleEngine *HybridRuleEngine,
	metricsCollector *RoutingMetricsCollector,
) *IntelligentRouter {
	return &IntelligentRouter{
		predictiveModel:    predictiveModel,
		reinforcementAgent: rlAgent,
		sentimentAnalyzer:  sentimentAnalyzer,
		ruleEngine:         ruleEngine,
		metricsCollector:   metricsCollector,
		modelTimeout:       500 * time.Millisecond,
		decisionCache:      make(map[string]*RoutingDecision),
	}
}

// Route executes the main routing decision pipeline
func (r *IntelligentRouter) Route(ctx context.Context, req RoutingRequest) (*RoutingDecision, error) {
	startTime := time.Now()

	// Step 1: Feature extraction and enrichment
	features := r.extractFeatures(req)

	// Step 2: Run all AI models in parallel
	modelResults := r.runParallelModels(ctx, features, req)

	// Step 3: Ensemble decision making
	decision := r.ensembleDecision(modelResults, req)
	decision.Timestamp = time.Now()

	// Step 4: Validate decision against business rules
	if err := r.validateDecision(decision, req); err != nil {
		log.Printf("Decision validation failed, falling back: %v", err)
		decision = r.ruleEngine.FallbackRoute(req)
		decision.Reasoning = append(decision.Reasoning, "Fell back to rule-based routing: "+err.Error())
	}

	// Step 5: Log decision for RL training
	decisionLatency := time.Since(startTime)
	r.metricsCollector.LogDecision(decision, features, decisionLatency)

	// Cache decision
	r.cacheDecision(decision)

	return decision, nil
}

// runParallelModels executes all AI models concurrently
func (r *IntelligentRouter) runParallelModels(ctx context.Context, features Features, req RoutingRequest) []ModelResult {
	results := make(chan ModelResult, 4)
	var wg sync.WaitGroup

	// Model 1: Predictive Analytics
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		prediction := r.predictiveModel.Predict(features, req.AvailableBranches)
		results <- ModelResult{
			ModelName:   "predictive_analytics",
			BranchID:    prediction.BranchID,
			Score:       prediction.PredictedSuccessRate,
			Confidence:  prediction.Confidence,
			Explanation: fmt.Sprintf("Predicted %.1f%% success rate", prediction.PredictedSuccessRate*100),
			LatencyMs:   float64(time.Since(start).Milliseconds()),
		}
	}()

	// Model 2: Reinforcement Learning
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		rlDecision := r.reinforcementAgent.SelectAction(features, req.AvailableBranches)
		results <- ModelResult{
			ModelName:   "reinforcement_learning",
			BranchID:    rlDecision.BranchID,
			Score:       rlDecision.QValue,
			Confidence:  1.0 - rlDecision.Epsilon,
			Explanation: fmt.Sprintf("RL Q-value: %.3f (episodes: %d)", rlDecision.QValue, rlDecision.EpisodeCount),
			LatencyMs:   float64(time.Since(start).Milliseconds()),
		}
	}()

	// Model 3: Sentiment Analysis
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		if textData := r.extractTextFields(req.Data); len(textData) > 0 {
			sentiment := r.sentimentAnalyzer.AnalyzeBatch(textData)
			branchID := r.mapSentimentToBranch(sentiment, req.AvailableBranches)
			results <- ModelResult{
				ModelName:   "sentiment_analysis",
				BranchID:    branchID,
				Score:       sentiment.CompoundScore,
				Confidence:  sentiment.Confidence,
				Explanation: fmt.Sprintf("Sentiment: %.2f (%s)", sentiment.CompoundScore, sentiment.DominantEmotion),
				LatencyMs:   float64(time.Since(start).Milliseconds()),
			}
		}
	}()

	// Model 4: Load Balancing Optimizer
	wg.Add(1)
	go func() {
		defer wg.Done()
		start := time.Now()
		loadOptimal := r.findLoadOptimalBranch(req.AvailableBranches, req.Context.SystemLoad)
		results <- ModelResult{
			ModelName:   "load_balancer",
			BranchID:    loadOptimal.BranchID,
			Score:       loadOptimal.UtilizationScore,
			Confidence:  0.9,
			Explanation: fmt.Sprintf("Queue: %d, Wait: %.1fm", loadOptimal.QueueDepth, loadOptimal.EstimatedWaitMinutes),
			LatencyMs:   float64(time.Since(start).Milliseconds()),
		}
	}()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	var modelResults []ModelResult

modelLoop:
	for {
		select {
		case <-done:
			break modelLoop
		case <-time.After(r.modelTimeout):
			log.Printf("Model prediction timeout after collecting %d results", len(modelResults))
			return modelResults
		case result := <-results:
			if result.Score > 0 || result.ModelName == "load_balancer" {
				modelResults = append(modelResults, result)
			}
		}
	}

	// Drain remaining results
	for {
		select {
		case result := <-results:
			if result.Score > 0 || result.ModelName == "load_balancer" {
				modelResults = append(modelResults, result)
			}
		default:
			return modelResults
		}
	}
}

// ensembleDecision combines model results using weighted voting
func (r *IntelligentRouter) ensembleDecision(modelResults []ModelResult, req RoutingRequest) *RoutingDecision {
	modelWeights := map[string]float64{
		"predictive_analytics":   0.35,
		"reinforcement_learning": 0.30,
		"sentiment_analysis":     0.20,
		"load_balancer":          0.15,
	}

	branchScores := make(map[string]float64)
	branchReasons := make(map[string][]string)
	modelScores := make(map[string]float64)

	for _, result := range modelResults {
		weight := modelWeights[result.ModelName]
		weightedScore := result.Score * result.Confidence * weight
		branchScores[result.BranchID] += weightedScore
		modelScores[result.ModelName] = result.Score

		branchReasons[result.BranchID] = append(branchReasons[result.BranchID],
			fmt.Sprintf("[%s] %s (latency: %.1fms)", result.ModelName, result.Explanation, result.LatencyMs))
	}

	// Find winning branch
	var winningBranch string
	maxScore := -1.0
	for branchID, score := range branchScores {
		if score > maxScore {
			maxScore = score
			winningBranch = branchID
		}
	}

	// Calculate confidence (model agreement)
	confidence := r.calculateModelAgreement(modelResults, winningBranch)

	// Build alternative paths
	alternatives := r.buildAlternatives(branchScores, winningBranch)

	return &RoutingDecision{
		SelectedBranchID:  winningBranch,
		Confidence:        confidence,
		Reasoning:         branchReasons[winningBranch],
		AlternativePaths:  alternatives,
		ModelScores:       modelScores,
		ExecutionStrategy: r.determineStrategy(confidence, req.Context),
		DecisionID:        fmt.Sprintf("decision_%d", time.Now().UnixNano()),
	}
}

// calculateModelAgreement scores how much models agree on the selected branch
func (r *IntelligentRouter) calculateModelAgreement(results []ModelResult, selectedBranch string) float64 {
	if len(results) == 0 {
		return 0.0
	}

	agreementCount := 0
	for _, result := range results {
		if result.BranchID == selectedBranch {
			agreementCount++
		}
	}

	return float64(agreementCount) / float64(len(results))
}

// buildAlternatives creates ranked alternatives
func (r *IntelligentRouter) buildAlternatives(branchScores map[string]float64, selectedBranch string) []AlternativePath {
	var alternatives []AlternativePath
	var entries []struct {
		BranchID string
		Score    float64
	}

	for branchID, score := range branchScores {
		if branchID != selectedBranch {
			entries = append(entries, struct {
				BranchID string
				Score    float64
			}{branchID, score})
		}
	}

	// Sort by score (simple bubble sort for small lists)
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Score > entries[i].Score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	for i, entry := range entries {
		if i >= 3 { // Top 3 alternatives
			break
		}
		alternatives = append(alternatives, AlternativePath{
			BranchID:      entry.BranchID,
			Score:         entry.Score,
			Ranking:       i + 1,
			Justification: fmt.Sprintf("Alternative option with score %.3f", entry.Score),
		})
	}

	return alternatives
}

// Helper methods
func (r *IntelligentRouter) extractFeatures(req RoutingRequest) Features {
	return Features{
		OrderAmount:          r.getFloat64(req.Data, "order_amount", 0),
		CustomerLTV:          r.getFloat64(req.Data, "customer_ltv", 0),
		HistoricalOrderCount: r.getInt(req.Data, "order_count", 0),
		RiskScore:            r.getFloat64(req.Data, "risk_score", 0),
		CustomerPattern:      r.getString(req.Data, "customer_pattern", "new"),
		Timestamp:            time.Now(),
	}
}

func (r *IntelligentRouter) extractTextFields(data map[string]interface{}) []string {
	var textFields []string
	for key, val := range data {
		if str, ok := val.(string); ok && len(str) > 0 {
			// Skip structural fields
			if key != "workflow_id" && key != "tenant_id" {
				textFields = append(textFields, str)
			}
		}
	}
	return textFields
}

func (r *IntelligentRouter) mapSentimentToBranch(sentiment *SentimentResult, branches []Branch) string {
	if len(branches) == 0 {
		return ""
	}

	// Simple heuristic: negative sentiment -> priority handling
	if sentiment.CompoundScore < -0.1 {
		// Look for priority/escalation branch
		for _, branch := range branches {
			for _, spec := range branch.Specialties {
				if spec == "priority" || spec == "escalation" {
					return branch.ID
				}
			}
		}
	}

	return branches[0].ID
}

func (r *IntelligentRouter) findLoadOptimalBranch(branches []Branch, systemLoad SystemLoadMetrics) *LoadOptimalResult {
	var optimal *LoadOptimalResult
	minLoad := float64(1000)

	for _, branch := range branches {
		utilization := float64(branch.CurrentLoad) / float64(branch.Capacity)
		if utilization < minLoad && branch.Capacity > 0 {
			minLoad = utilization
			optimal = &LoadOptimalResult{
				BranchID:             branch.ID,
				UtilizationScore:     1.0 - utilization,
				QueueDepth:           branch.CurrentLoad,
				EstimatedWaitMinutes: float64(branch.CurrentLoad) * (branch.AvgDuration / 60.0),
			}
		}
	}

	if optimal == nil && len(branches) > 0 {
		optimal = &LoadOptimalResult{
			BranchID:             branches[0].ID,
			UtilizationScore:     0.5,
			QueueDepth:           0,
			EstimatedWaitMinutes: branches[0].AvgDuration / 60.0,
		}
	}

	return optimal
}

func (r *IntelligentRouter) validateDecision(decision *RoutingDecision, req RoutingRequest) error {
	// Check if selected branch exists
	for _, branch := range req.AvailableBranches {
		if branch.ID == decision.SelectedBranchID {
			return nil
		}
	}
	return fmt.Errorf("selected branch not in available branches")
}

func (r *IntelligentRouter) determineStrategy(confidence float64, ctx RoutingContext) string {
	if confidence > 0.8 {
		return "immediate"
	} else if confidence > 0.6 {
		return "conditional"
	}
	return "delayed"
}

func (r *IntelligentRouter) cacheDecision(decision *RoutingDecision) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.decisionCache[decision.DecisionID] = decision
}

func (r *IntelligentRouter) getFloat64(data map[string]interface{}, key string, defaultVal float64) float64 {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return defaultVal
}

func (r *IntelligentRouter) getInt(data map[string]interface{}, key string, defaultVal int) int {
	if val, ok := data[key]; ok {
		if i, ok := val.(int); ok {
			return i
		} else if f, ok := val.(float64); ok {
			return int(f)
		}
	}
	return defaultVal
}

func (r *IntelligentRouter) getString(data map[string]interface{}, key string, defaultVal string) string {
	if val, ok := data[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return defaultVal
}

// LoadOptimalResult represents the result of load optimization
type LoadOptimalResult struct {
	BranchID             string
	UtilizationScore     float64
	QueueDepth           int
	EstimatedWaitMinutes float64
}
