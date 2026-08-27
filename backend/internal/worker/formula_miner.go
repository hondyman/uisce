package worker

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

type FormulaMiner struct {
	DB         *sql.DB
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

func NewFormulaMiner(db *sql.DB) *FormulaMiner {
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	model := os.Getenv("AI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}

	return &FormulaMiner{
		DB:         db,
		APIKey:     os.Getenv("OPENAI_API_KEY"),
		BaseURL:    baseURL,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// StartNightlyExtraction begins the background ticker for telemetry harvesting
func (m *FormulaMiner) StartNightlyExtraction(ctx context.Context, interval time.Duration) {
	log.Printf("Starting Agentic Formula Miner (Interval: %v)", interval)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Formula Miner worker...")
			return
		case <-ticker.C:
			m.ExtractComplexMath(ctx)
		}
	}
}

func (m *FormulaMiner) ExtractComplexMath(ctx context.Context) {
	if m.DB == nil {
		return
	}

	query := `
		SELECT id, tenant_id, prompt, executed_query 
		FROM ai_query_telemetry 
		WHERE created_at >= NOW() - INTERVAL '24 hours'
		  AND was_edited = TRUE 
		  AND (prompt ILIKE '%margin%' OR prompt ILIKE '%yield%' OR prompt ILIKE '%ratio%' OR prompt ILIKE '%spread%')
	`
	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Formula Miner DB error: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var telemetryID, tenantID, prompt string
		var executedQuery []byte

		if err := rows.Scan(&telemetryID, &tenantID, &prompt, &executedQuery); err != nil {
			log.Printf("Failed to scan telemetry row: %v", err)
			continue
		}

		extractedFormula, termName, err := m.askLLMForFormula(ctx, prompt, executedQuery)
		if err != nil || extractedFormula == "" || termName == "" {
			continue
		}

		m.stageCalculatedMeasure(ctx, tenantID, termName, extractedFormula)
	}
}

func (m *FormulaMiner) askLLMForFormula(ctx context.Context, prompt string, executedQuery []byte) (string, string, error) {
	if m.APIKey == "" {
		return "", "", nil
	}

	systemPrompt := `You are a financial data architect extracting formulas from user behavior.
The user provided a natural language prompt, but the initial AI query did not match. The user then manually built and executed a query.
Analyze the user's original intent and the JSON payload of their manually executed query.
Identify the mathematical formula they constructed using the semantic field IDs.

Respond ONLY with a valid JSON object matching this schema:
{
  "extracted_term": "The name of the metric (e.g., Net Yield)",
  "formula": "The math expression using field IDs (e.g., (revenue - expenses) / total_valuation)"
}`

	userMessage := fmt.Sprintf("Original Prompt: \"%s\"\n\nExecuted Query JSON:\n%s", prompt, string(executedQuery))

	payload := map[string]interface{}{
		"model":       m.Model,
		"temperature": 0.0,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userMessage},
		},
		"response_format": map[string]string{"type": "json_object"},
	}

	bodyBytes, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", m.BaseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.APIKey)

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("LLM API error: %d", resp.StatusCode)
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil || len(chatResp.Choices) == 0 {
		return "", "", fmt.Errorf("failed to decode LLM response")
	}

	var result struct {
		ExtractedTerm string `json:"extracted_term"`
		Formula       string `json:"formula"`
	}

	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &result); err != nil {
		return "", "", err
	}

	return result.Formula, result.ExtractedTerm, nil
}

func (m *FormulaMiner) stageCalculatedMeasure(ctx context.Context, tenantID, term, expression string) {
	if m.DB == nil {
		return
	}
	query := `
		INSERT INTO ai_knowledge_candidates (
			tenant_id, type, term, expression, occurrences, confidence, status
		) VALUES ($1, 'calculated_measure', $2, $3, 1, 0.75, 'pending_review')
		ON CONFLICT (tenant_id, type, term) DO UPDATE
		SET occurrences = ai_knowledge_candidates.occurrences + 1,
		    confidence = GREATEST(0.000, LEAST(1.000, ai_knowledge_candidates.confidence + 0.10)),
		    updated_at = NOW()
	`
	_, err := m.DB.ExecContext(ctx, query, tenantID, term, expression)
	if err != nil {
		log.Printf("Failed to stage calculated measure: %v", err)
	} else {
		log.Printf("Staged new calculated measure for tenant %s: %s = %s", tenantID, term, expression)
	}
}
