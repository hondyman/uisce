package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// AIService is the primary entry point for all semantic AI capabilities.
// It coordinates specialized engines and ensures grounding in the semantic graph.
type AIService struct {
	db                 *sql.DB
	llmProvider        llm.LLMProvider
	ruleAnalyzer       *RuleAnalyzer
	driftPredictor     *DriftPredictor
	chatEngine         *ChatEngine
	dqAnalyzer         *DataQualityAnalyzer
	trainingData       *TrainingDataStore
	feedbackLoop       *FeedbackProcessor
	explainability     *ExplainabilityEngine
	scenarioSvc        *rules.ScenarioService
	ruleTemplateEngine *RuleTemplateEngine
}

// NewAIService creates a new unified AI service hub
func NewAIService(db *sql.DB, llmProvider llm.LLMProvider, scenarioSvc *rules.ScenarioService) *AIService {
	return &AIService{
		db:                 db,
		llmProvider:        llmProvider,
		ruleAnalyzer:       NewRuleAnalyzer(db, llmProvider, scenarioSvc),
		driftPredictor:     NewDriftPredictor(db, llmProvider),
		chatEngine:         NewChatEngine(db, llmProvider),
		dqAnalyzer:         NewDataQualityAnalyzer(db),
		trainingData:       NewTrainingDataStore(db),
		feedbackLoop:       NewFeedbackProcessor(db),
		explainability:     NewExplainabilityEngine(),
		scenarioSvc:        scenarioSvc,
		ruleTemplateEngine: NewRuleTemplateEngine(db, llmProvider),
	}
}

// RuleSuggestion represents an AI-generated suggestion for a semantic rule
type RuleSuggestion struct {
	ID                  uuid.UUID              `json:"id"`
	TenantID            uuid.UUID              `json:"tenant_id"`
	RuleType            string                 `json:"rule_type"`
	SuggestedName       string                 `json:"suggested_name"`
	Title               string                 `json:"title"`
	Description         string                 `json:"description"`
	ConditionJSON       map[string]interface{} `json:"condition_json"`
	Parameters          map[string]interface{} `json:"parameters"`
	Confidence          float64                `json:"confidence"`
	SemanticPath        []string               `json:"semantic_path"`
	ExplainabilityScore int                    `json:"explainability_score"`
	GovernanceContext   map[string]interface{} `json:"governance_context"`
	Impact              map[string]interface{} `json:"impact"`
	BasePattern         *Pattern               `json:"base_pattern"`
	Rationale           string                 `json:"rationale"`
	Status              string                 `json:"status"` // pending, accepted, dismissed
	CreatedAt           time.Time              `json:"created_at"`
}

// GetRuleSuggestions analyzes historical patterns and returns grounded suggestions
func (s *AIService) GetRuleSuggestions(ctx context.Context, tenantID uuid.UUID, businessObject string) ([]RuleSuggestion, error) {
	// 1. Analyze historical patterns
	patterns, err := s.ruleAnalyzer.AnalyzePatterns(ctx, tenantID, businessObject)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze patterns: %w", err)
	}

	// 2. Generate suggestions based on patterns
	suggestions, err := s.ruleAnalyzer.GenerateSuggestions(ctx, patterns)
	if err != nil {
		return nil, fmt.Errorf("failed to generate suggestions: %w", err)
	}

	// 3. Enhance with explainability
	for i := range suggestions {
		suggestions[i].ExplainabilityScore = s.explainability.CalculateScore(suggestions[i])
		suggestions[i].SemanticPath = s.explainability.TraceSemanticPath(suggestions[i])
	}

	return suggestions, nil
}

// GetDriftPredictions analyses historical signals and returns predictions (Feature 2)
func (s *AIService) GetDriftPredictions(ctx context.Context, tenantID uuid.UUID, params DriftPredictionParams) ([]DriftPrediction, error) {
	return s.driftPredictor.GetDriftPredictions(ctx, tenantID, params)
}

// SuggestRuleTemplates generates reusable rule templates from usage clusters (Feature 3)
func (s *AIService) SuggestRuleTemplates(ctx context.Context, tenantID uuid.UUID) ([]RuleTemplateSuggestion, error) {
	return s.ruleTemplateEngine.SuggestTemplates(ctx, tenantID)
}

// SubmitFeedback processes user feedback on a rule suggestion and updates training data
func (s *AIService) SubmitFeedback(ctx context.Context, feedback UserFeedback) (*FeedbackResponse, error) {
	return s.feedbackLoop.ProcessFeedback(ctx, feedback)
}

// SemanticNode represents a node in the semantic impact graph
type SemanticNode struct {
	ID           string `json:"id"`
	Type         string `json:"type"` // semantic_term, bo_field, business_object, rule
	Label        string `json:"label"`
	ImpactWeight int    `json:"impact_weight"` // 0-100
}

// SemanticEdge represents a relationship in the semantic impact graph
type SemanticEdge struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Relationship string `json:"relationship"` // used_in, belongs_to, governed_by
}

// ImpactReport represents the results of a semantic impact analysis
type ImpactReport struct {
	SuggestionID      uuid.UUID              `json:"suggestion_id"`
	AnalysisTimestamp time.Time              `json:"analysis_timestamp"`
	ImpactedCount     int                    `json:"impacted_count"`
	SeverityProfile   map[string]int         `json:"severity_profile"`
	RiskAssessment    string                 `json:"risk_assessment"`
	SemanticPath      []string               `json:"semantic_path"`
	ImpactGraph       []SemanticNode         `json:"impact_graph"`
	ImpactEdges       []SemanticEdge         `json:"impact_edges"`
	Propagation       map[string]int         `json:"propagation"` // node_id -> impact_score
	Details           map[string]interface{} `json:"details"`
}

// PerformImpactAnalysis runs a "what-if" simulation for a suggested rule
// and traces semantic lineage through the rule graph (semantic design.pdf §3.4)
func (s *AIService) PerformImpactAnalysis(ctx context.Context, tenantID uuid.UUID, suggestion RuleSuggestion) (*ImpactReport, error) {
	// 1. Create a draft scenario for what-if simulation
	scenario, err := s.scenarioSvc.CreateRuleScenario(ctx, tenantID.String(), nil, "AI Suggestion Impact: "+suggestion.Title, suggestion.Description, "AI_SYSTEM")
	if err != nil {
		return nil, fmt.Errorf("failed to create scenario: %w", err)
	}

	// 2. Wrap condition into scenario version
	condBytes, _ := json.Marshal(suggestion.ConditionJSON)
	_, err = s.scenarioSvc.SaveScenarioVersion(ctx, scenario.ID, condBytes, "AI_SYSTEM")
	if err != nil {
		return nil, fmt.Errorf("failed to save scenario version: %w", err)
	}

	// 3. Build semantic lineage graph from suggestion context
	graph, edges := s.buildSemanticImpactGraph(suggestion)

	// 4. Trace semantic lineage path
	semanticPath := s.traceSemanticLineage(graph)

	// 5. Calculate impact propagation through the graph
	propagation := s.calculateImpactPropagation(graph, edges)

	// 6. Determine risk based on propagation depth and severity
	risks := s.assessRisk(propagation)

	return &ImpactReport{
		SuggestionID:      suggestion.ID,
		AnalysisTimestamp: time.Now(),
		ImpactedCount:     len(propagation),
		SeverityProfile:   risks,
		RiskAssessment:    s.riskRating(risks),
		SemanticPath:      semanticPath,
		ImpactGraph:       graph,
		ImpactEdges:       edges,
		Propagation:       propagation,
		Details: map[string]interface{}{
			"scenario_id": scenario.ID,
			"note":        "Semantic lineage traced from rule through BO fields and terms",
		},
	}, nil
}

// buildSemanticImpactGraph constructs the impact graph from a suggestion
func (s *AIService) buildSemanticImpactGraph(suggestion RuleSuggestion) ([]SemanticNode, []SemanticEdge) {
	nodes := []SemanticNode{
		{ID: suggestion.ID.String(), Type: "rule", Label: suggestion.Title, ImpactWeight: 100},
	}
	edges := []SemanticEdge{}

	// Attach semantic path nodes
	prev := suggestion.ID.String()
	for i, step := range suggestion.SemanticPath {
		nodeType := "semantic_term"
		if i == len(suggestion.SemanticPath)-1 {
			nodeType = "business_object"
		}
		nodeID := fmt.Sprintf("%s-%d", step, i)
		nodes = append(nodes, SemanticNode{
			ID:           nodeID,
			Type:         nodeType,
			Label:        step,
			ImpactWeight: 100 - (i * 15),
		})
		edges = append(edges, SemanticEdge{From: prev, To: nodeID, Relationship: "governed_by"})
		prev = nodeID
	}

	return nodes, edges
}

// traceSemanticLineage extracts the ordered path labels from the graph
func (s *AIService) traceSemanticLineage(nodes []SemanticNode) []string {
	path := make([]string, 0, len(nodes))
	for _, n := range nodes {
		path = append(path, n.Label)
	}
	return path
}

// calculateImpactPropagation scores each node's propagated impact
func (s *AIService) calculateImpactPropagation(nodes []SemanticNode, edges []SemanticEdge) map[string]int {
	propagation := map[string]int{}
	for _, n := range nodes {
		propagation[n.ID] = n.ImpactWeight
	}
	// Dampen impact across edges
	for _, e := range edges {
		if from, ok := propagation[e.From]; ok {
			if _, exists := propagation[e.To]; !exists {
				propagation[e.To] = from / 2
			}
		}
	}
	return propagation
}

// assessRisk builds a severity profile from propagation scores
func (s *AIService) assessRisk(propagation map[string]int) map[string]int {
	profile := map[string]int{"critical": 0, "warning": 0, "info": 0}
	for _, score := range propagation {
		switch {
		case score >= 75:
			profile["critical"]++
		case score >= 40:
			profile["warning"]++
		default:
			profile["info"]++
		}
	}
	return profile
}

// riskRating converts a severity profile to a HIGH/MEDIUM/LOW string
func (s *AIService) riskRating(profile map[string]int) string {
	switch {
	case profile["critical"] > 2:
		return "HIGH"
	case profile["critical"] > 0 || profile["warning"] > 5:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// Placeholder for DataQualityAnalyzer
type DataQualityAnalyzer struct {
	db *sql.DB
}

func NewDataQualityAnalyzer(db *sql.DB) *DataQualityAnalyzer {
	return &DataQualityAnalyzer{db: db}
}

// Placeholder for ExplainabilityEngine
type ExplainabilityEngine struct{}

func NewExplainabilityEngine() *ExplainabilityEngine {
	return &ExplainabilityEngine{}
}

func (e *ExplainabilityEngine) CalculateScore(s RuleSuggestion) int {
	// Simple heuristic for now
	return 85
}

func (e *ExplainabilityEngine) TraceSemanticPath(s RuleSuggestion) []string {
	return []string{"BusinessObject", "SemanticTerm", "Rule"}
}

// GenerateJSON builds a prompt and extracts JSON from the LLM
func (s *AIService) GenerateJSON(ctx context.Context, prompt string, schema string) (string, error) {
	// Implementation for prompt_builder.go
	response, err := s.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return "", err
	}
	return cleanJSON(response), nil
}

func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}
