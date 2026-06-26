package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/jmoiron/sqlx"
)

// NLQService provides natural language Q&A over the catalog.
type NLQService struct {
	db              *sqlx.DB
	llmProvider     llm.LLMProvider
	searchService   *SearchService
	reasoningEngine *ReasoningEngine
	financialTools  *FinancialToolService
}

// NewNLQService creates a new NLQ service.
func NewNLQService(db *sqlx.DB, llmProvider llm.LLMProvider, searchService *SearchService, reasoningEngine *ReasoningEngine, financialTools *FinancialToolService) *NLQService {
	return &NLQService{
		db:              db,
		llmProvider:     llmProvider,
		searchService:   searchService,
		reasoningEngine: reasoningEngine,
		financialTools:  financialTools,
	}
}

// AskRequest represents the payload for a question.
type AskRequest struct {
	Question         string `json:"question"`
	TargetEntityPath string `json:"target_entity_path,omitempty"` // Optional - can be auto-discovered
}

// AskResponse is the structured answer from the service.
type AskResponse struct {
	Answer               string            `json:"answer"`
	CalculationBreakdown []CalculationStep `json:"calculation_breakdown,omitempty"`
	Sources              []SourceReference `json:"sources"`
	Confidence           string            `json:"confidence"`
	ResolvedEntityPath   string            `json:"resolved_entity_path,omitempty"`
	Caveats              []string          `json:"caveats,omitempty"`
	DataQuality          *DataQuality      `json:"data_quality,omitempty"`
	FactorBreakdown      []FactorExposure  `json:"factor_breakdown,omitempty"`
}

// FactorExposure represents a factor's contribution (from factors package)
type FactorExposure struct {
	Factor       string   `json:"factor"`
	Contribution float64  `json:"contribution"`
	Narrative    string   `json:"narrative"`
	Significance float64  `json:"significance"`
	Sources      []string `json:"sources"`
	PValue       float64  `json:"p_value"`
}

// CalculationStep represents a single step in a calculation breakdown.
type CalculationStep struct {
	Step        string                 `json:"step"`             // e.g., "Input", "Filter", "Aggregate", "Group"
	Source      string                 `json:"source,omitempty"` // Qualified path of source node
	Rule        string                 `json:"rule,omitempty"`   // Business rule or transformation logic
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Additional context
}

// SourceReference provides detailed information about a source used in the answer.
type SourceReference struct {
	Path        string                 `json:"path"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Ask processes a natural language question using hybrid retrieval (semantic + graph).
func (s *NLQService) Ask(ctx context.Context, secCtx *security.Context, req AskRequest) (*AskResponse, error) {
	if secCtx == nil {
		return nil, fmt.Errorf("security context is required")
	}
	// Step 1: If no target entity path provided, use semantic search to find the most relevant entity
	targetPath := req.TargetEntityPath
	if targetPath == "" {
		var err error
		targetPath, err = s.findRelevantEntity(ctx, req.Question, secCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to find relevant entity: %w", err)
		}
		if targetPath == "" {
			return nil, fmt.Errorf("no relevant entity found for question")
		}
	}

	// Step 2: Retrieve the calculation DAG using the existing Postgres function.
	var dagJSON []byte
	err := s.db.GetContext(ctx, &dagJSON, "SELECT get_calc_dag_with_metadata($1, $2)", targetPath, secCtx.TenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("target entity '%s' not found for tenant '%s'", targetPath, secCtx.TenantID)
		}
		return nil, fmt.Errorf("failed to retrieve calculation DAG: %w", err)
	}

	// Step 3: Use ReasoningEngine to synthesize the answer
	// We pass the DAG as context info
	contextInfo := fmt.Sprintf("Here is the dependency graph for the calculation/entity in question:\n%s", string(dagJSON))

	llmAnswer, err := s.reasoningEngine.Synthesize(ctx, req.Question, contextInfo)
	if err != nil {
		return nil, fmt.Errorf("ReasoningEngine failed to synthesize response: %w", err)
	}

	// Step 4: Parse the LLM's response and format it with enhanced metadata.
	return s.parseEnhancedLLMResponse(llmAnswer, dagJSON, targetPath)
}

// findRelevantEntity uses semantic search to find the most relevant catalog node for a question.
func (s *NLQService) findRelevantEntity(ctx context.Context, question string, secCtx *security.Context) (string, error) {
	if secCtx == nil {
		return "", fmt.Errorf("security context is required")
	}
	// Use SearchService for hybrid retrieval
	searchReq := models.SemanticSearchRequest{
		Query:        question,
		DatasourceID: secCtx.DatasourceID,
		Region:       secCtx.Region,
	}

	results, err := s.searchService.HybridSearch(ctx, searchReq, *secCtx)
	if err != nil {
		return "", fmt.Errorf("hybrid search failed: %w", err)
	}

	if len(results) == 0 {
		return "", nil
	}

	// Return the qualified path of the top result
	// Note: HybridSearch results are already sorted by score
	return results[0].QualifiedPath, nil
}
func (s *NLQService) buildPrompt(question, dagJSON string) (string, error) {
	// A sophisticated system prompt that instructs the LLM on how to behave.
	systemPrompt := `
SYSTEM: You are a helpful, precise data analyst assistant. Your role is to answer questions about our data catalog.
- Use ONLY the context provided below. Do not use any outside knowledge.
- If the context does not contain the answer, state that you cannot answer the question.
- Your answers must be grounded in the provided sources.
- Cite the 'path' for each node you reference in your answer.
- Start your answer with a direct, plain-language explanation.

CONTEXT:
---
Here is the dependency graph for the calculation in question.
%s
---

USER QUESTION:
%s

ASSISTANT RESPONSE:
`
	return fmt.Sprintf(systemPrompt, dagJSON, question), nil
}

func (s *NLQService) buildEnhancedPrompt(question, dagJSON string) (string, error) {
	// Enhanced system prompt with explicit instructions for calculation explanations
	systemPrompt := `
SYSTEM: You are a helpful, precise data analyst assistant. Your role is to answer questions about our data catalog and explain calculations.

INSTRUCTIONS:
- Use ONLY the context provided below. Do not use any outside knowledge.
- If the context does not contain the answer, state that you cannot answer the question.
- Your answers must be grounded in the provided sources.
- Cite the 'path' for each node you reference in your answer.
- Start your answer with a direct, plain-language explanation.
- For calculation questions, explain:
  1. What inputs are used (source tables/views)
  2. What transformations are applied (aggregations, filters, joins)
  3. What assumptions or business rules are embedded
  4. Any data quality considerations (freshness, null rates, SLAs)

CONTEXT:
---
Here is the dependency graph for the calculation/entity in question:
%s
---

USER QUESTION:
%s

Please provide a clear, structured answer that explains the calculation step-by-step if applicable.

ASSISTANT RESPONSE:
`
	return fmt.Sprintf(systemPrompt, dagJSON, question), nil
}

func (s *NLQService) parseLLMResponse(llmAnswer string, dagJSON []byte) (*AskResponse, error) {
	var dagData map[string]interface{}
	_ = json.Unmarshal(dagJSON, &dagData)

	var sources []SourceReference
	if nodes, ok := dagData["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if n, ok := node.(map[string]interface{}); ok {
				source := SourceReference{}
				if p, ok := n["path"].(string); ok {
					source.Path = p
				}
				if name, ok := n["name"].(string); ok {
					source.Name = name
				}
				if t, ok := n["type"].(string); ok {
					source.Type = t
				}
				sources = append(sources, source)
			}
		}
	}

	return &AskResponse{
		Answer:     strings.TrimSpace(llmAnswer),
		Sources:    sources,
		Confidence: "High", // Placeholder confidence
	}, nil
}

func (s *NLQService) parseEnhancedLLMResponse(llmAnswer string, dagJSON []byte, resolvedPath string) (*AskResponse, error) {
	var dagData map[string]interface{}
	_ = json.Unmarshal(dagJSON, &dagData)

	var sources []SourceReference
	var caveats []string

	if nodes, ok := dagData["nodes"].([]interface{}); ok {
		for _, node := range nodes {
			if n, ok := node.(map[string]interface{}); ok {
				source := SourceReference{
					Metadata: make(map[string]interface{}),
				}

				if p, ok := n["path"].(string); ok {
					source.Path = p
				}
				if name, ok := n["name"].(string); ok {
					source.Name = name
				}
				if t, ok := n["type"].(string); ok {
					source.Type = t
				}

				// Extract data quality contract info for caveats
				if dqc, ok := n["data_quality_contract"].(map[string]interface{}); ok && dqc != nil {
					source.Metadata["data_quality"] = dqc

					// Add caveats based on data quality
					if freshness, ok := dqc["freshness"].(string); ok && freshness != "" {
						caveats = append(caveats, fmt.Sprintf("Data freshness: %s", freshness))
					}
					if nullRate, ok := dqc["null_rate"].(string); ok && nullRate != "" {
						caveats = append(caveats, fmt.Sprintf("Null rate: %s", nullRate))
					}
				}

				// Extract SLA info
				if sla, ok := n["sla"].(map[string]interface{}); ok && sla != nil {
					source.Metadata["sla"] = sla
				}

				// Extract lineage
				if lineage, ok := n["lineage"].(map[string]interface{}); ok && lineage != nil {
					source.Metadata["lineage"] = lineage
				}

				sources = append(sources, source)
			}
		}
	}

	// Parse calculation breakdown from DAG edges/nodes into typed steps
	var calcBreakdown []CalculationStep
	if edges, ok := dagData["edges"].([]interface{}); ok && len(edges) > 0 {
		for _, edge := range edges {
			if edgeMap, ok := edge.(map[string]interface{}); ok {
				step := CalculationStep{
					Step:   "Dependency",
					Source: fmt.Sprintf("%v", edgeMap["source"]),
					Rule:   fmt.Sprintf("Relationship: %v", edgeMap["relationship"]),
				}
				calcBreakdown = append(calcBreakdown, step)
			}
		}
	}

	// If no edges, create a simple step from the node itself
	if len(calcBreakdown) == 0 {
		calcBreakdown = []CalculationStep{
			{
				Step:        "Direct Calculation",
				Source:      resolvedPath,
				Description: "No dependencies found",
			},
		}
	}

	return &AskResponse{
		Answer:               strings.TrimSpace(llmAnswer),
		Sources:              sources,
		CalculationBreakdown: calcBreakdown,
		Confidence:           "High",
		ResolvedEntityPath:   resolvedPath,
		Caveats:              caveats,
	}, nil
}
