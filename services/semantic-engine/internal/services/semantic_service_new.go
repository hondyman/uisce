package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	sharedtypes "github.com/hondyman/uisce/libs/shared-types"
	temporalclient "github.com/hondyman/uisce/libs/temporal-client"
)

// SemanticServiceConfig holds configuration for the semantic service
type SemanticServiceConfig struct {
	AIEndpoint         string
	GovernanceEndpoint string
	DB                 *sql.DB
	TemporalClient     *temporalclient.Client
}

// SemanticService provides semantic processing capabilities
type SemanticService struct {
	config SemanticServiceConfig
	db     *sql.DB
}

// NewSemanticService creates a new semantic service instance
func NewSemanticService(config SemanticServiceConfig) *SemanticService {
	return &SemanticService{
		config: config,
		db:     config.DB,
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
	sqlQuery := `
		SELECT id, source_field, target_field, mapping_type, confidence_score, created_at
		FROM semantic_mappings
		WHERE tenant_id = $1 AND datasource_id = $2
	`
	rows, err := s.db.QueryContext(ctx, sqlQuery, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic mappings: %w", err)
	}
	defer rows.Close()

	var mappings []sharedtypes.SemanticMapping
	for rows.Next() {
		var mapping sharedtypes.SemanticMapping
		if err := rows.Scan(&mapping.ID, &mapping.SourceField, &mapping.TargetField, &mapping.MappingType, &mapping.ConfidenceScore, &mapping.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan mapping row: %w", err)
		}
		mappings = append(mappings, mapping)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mapping rows: %w", err)
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
