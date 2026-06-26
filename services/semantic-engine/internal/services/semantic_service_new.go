package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	sharedtypes "github.com/hondyman/semlayer/libs/shared-types"
	temporalclient "github.com/hondyman/semlayer/libs/temporal-client"
)

// SemanticServiceConfig holds configuration for the semantic service
type SemanticServiceConfig struct {
	AIEndpoint         string
	GovernanceEndpoint string
	HasuraClient       *hasuraclient.HasuraClient
	TemporalClient     *temporalclient.Client
}

// SemanticService provides semantic processing capabilities
type SemanticService struct {
	config SemanticServiceConfig
}

// NewSemanticService creates a new semantic service instance
func NewSemanticService(config SemanticServiceConfig) *SemanticService {
	return &SemanticService{
		config: config,
	}
}

// CalculateSemanticModel performs semantic calculation for a given model
func (s *SemanticService) CalculateSemanticModel(ctx context.Context, request sharedtypes.SemanticCalculationRequest) (*sharedtypes.SemanticCalculationResponse, error) {
	// Check permissions using governance service
	accessReq := sharedtypes.AccessEvaluationRequest{
		UserID:   request.UserID,
		Action:   "calculate",
		Resource: "semantic_model",
		Context: map[string]interface{}{
			"model_id":  request.ModelID,
			"tenant_id": request.TenantID,
		},
	}

	accessResp, err := s.evaluateAccess(ctx, accessReq)
	if err != nil {
		return nil, fmt.Errorf("access evaluation failed: %w", err)
	}

	if !accessResp.Allowed {
		return nil, fmt.Errorf("access denied: %s", accessResp.Reason)
	}

	// Use AI service for semantic processing
	aiRequest := map[string]interface{}{
		"model": "grok-beta",
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": fmt.Sprintf("Process semantic model %s for tenant %s", request.ModelID, request.TenantID),
			},
		},
		"temperature": 0.1,
		"max_tokens":  2000,
	}

	aiResponse, err := s.callAIService(aiRequest)
	if err != nil {
		return nil, fmt.Errorf("AI processing failed: %w", err)
	}

	// Store result in Hasura
	result := &sharedtypes.SemanticCalculationResponse{
		ModelID:     request.ModelID,
		Result:      s.extractAIResponse(aiResponse),
		ProcessedAt: time.Now(),
	}

	// TODO: Store in Hasura GraphQL

	return result, nil
}

// evaluateAccess calls the governance service to evaluate access permissions
func (s *SemanticService) evaluateAccess(ctx context.Context, req sharedtypes.AccessEvaluationRequest) (*sharedtypes.AccessEvaluationResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.GovernanceEndpoint+"/api/v1/policies/evaluate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call governance service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("governance service returned status %d", resp.StatusCode)
	}

	var accessResp sharedtypes.AccessEvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&accessResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &accessResp, nil
}

// GetSemanticMappings retrieves semantic mappings for a datasource
func (s *SemanticService) GetSemanticMappings(ctx context.Context, tenantID, datasourceID string) ([]sharedtypes.SemanticMapping, error) {
	// Query Hasura for semantic mappings
	query := `
		query GetSemanticMappings($tenantId: String!, $datasourceId: String!) {
			semantic_mappings(
				where: {
					tenant_id: { _eq: $tenantId }
					datasource_id: { _eq: $datasourceId }
				}
			) {
				id
				source_field
				target_field
				mapping_type
				confidence_score
				created_at
			}
		}
	`

	result, err := s.config.HasuraClient.Query(query, map[string]interface{}{
		"tenantId":     tenantID,
		"datasourceId": datasourceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic mappings: %w", err)
	}

	// Parse response
	mappings := make([]sharedtypes.SemanticMapping, 0)
	if data, ok := result["semantic_mappings"].([]interface{}); ok {
		for _, item := range data {
			if mappingData, ok := item.(map[string]interface{}); ok {
				mapping := sharedtypes.SemanticMapping{
					ID:              mappingData["id"].(string),
					SourceField:     mappingData["source_field"].(string),
					TargetField:     mappingData["target_field"].(string),
					MappingType:     mappingData["mapping_type"].(string),
					ConfidenceScore: mappingData["confidence_score"].(float64),
					CreatedAt:       mappingData["created_at"].(string),
				}
				mappings = append(mappings, mapping)
			}
		}
	}

	return mappings, nil
}

// callAIService makes an HTTP call to the AI service
func (s *SemanticService) callAIService(request map[string]interface{}) (map[string]interface{}, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal AI request: %w", err)
	}

	req, err := http.NewRequest("POST", s.config.AIEndpoint+"/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create AI request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call AI service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("AI service error: %s", resp.Status)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %w", err)
	}

	return response, nil
}

// extractAIResponse extracts the content from the AI service response
func (s *SemanticService) extractAIResponse(response map[string]interface{}) string {
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}
	return "No response content available"
}
