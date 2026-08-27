package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
)

type TelemetryPayload struct {
	TenantID            string                    `json:"tenantId"`
	UserID              string                    `json:"userId"`
	UserRole            string                    `json:"userRole"`
	Prompt              string                    `json:"prompt"`
	GeneratedQuery      AIExplorerQueryDefinition `json:"generatedQuery"`
	ExecutedQuery       AIExplorerQueryDefinition `json:"executedQuery"`
	WasEdited           bool                      `json:"wasEdited"`
	WasSaved            bool                      `json:"wasSaved"`
	WasExported         bool                      `json:"wasExported"`
	ClonedToReport      bool                      `json:"clonedToReport"`
	Rating              int                       `json:"rating"` // 1 = Up, -1 = Down, 0 = Neutral
	FeedbackNotes       string                    `json:"feedbackNotes"`
	ExecutionDurationMs int                       `json:"executionDurationMs"`
}

type KnowledgeMiner struct {
	DB *sql.DB
}

func NewKnowledgeMiner(db *sql.DB) *KnowledgeMiner {
	return &KnowledgeMiner{DB: db}
}

// Built-in institutional finance dictionary for pattern extraction
var defaultFinancialAcronyms = map[string]string{
	"aum":        "total_valuation",
	"nav":        "total_valuation",
	"bps":        "basis_points",
	"pnl":        "realized_pnl",
	"unrealized": "unrealized_pnl",
	"irr":        "internal_rate_of_return",
	"twrr":       "time_weighted_return",
	"trades":     "trade_count",
	"vol":        "trade_volume",
	"cash":       "settled_cash_balance",
}

// IngestTelemetryAndHarvest processes incoming query metrics and extracts candidates
func (m *KnowledgeMiner) IngestTelemetryAndHarvest(ctx context.Context, payload TelemetryPayload) error {
	if m.DB == nil {
		return nil
	}

	genJSON, err := json.Marshal(payload.GeneratedQuery)
	if err != nil {
		return err
	}
	execJSON, err := json.Marshal(payload.ExecutedQuery)
	if err != nil {
		return err
	}

	// 1. Persist raw interaction telemetry
	query := `
		INSERT INTO ai_query_telemetry (
			tenant_id, user_id, user_role, prompt, generated_query, executed_query,
			was_edited, was_saved, was_exported, cloned_to_report, rating, feedback_notes, execution_duration_ms
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	_, err = m.DB.ExecContext(ctx, query,
		payload.TenantID, payload.UserID, payload.UserRole, payload.Prompt, genJSON, execJSON,
		payload.WasEdited, payload.WasSaved, payload.WasExported, payload.ClonedToReport,
		payload.Rating, payload.FeedbackNotes, payload.ExecutionDurationMs,
	)
	if err != nil {
		return fmt.Errorf("failed inserting telemetry: %w", err)
	}

	// 2. Auto-promote high-conviction interactions into Few-Shot Golden Queries
	if payload.Rating > 0 || payload.ClonedToReport || (payload.WasSaved && !payload.WasEdited) {
		upsertGolden := `
			INSERT INTO ai_golden_queries (tenant_id, prompt_pattern, verified_query, score)
			VALUES ($1, $2, $3, 1)
			ON CONFLICT (id) DO UPDATE
			SET score = ai_golden_queries.score + 1, updated_at = NOW()
		`
		_, _ = m.DB.ExecContext(ctx, upsertGolden, payload.TenantID, payload.Prompt, execJSON)
	}

	// 3. Extract candidate aliases and acronyms from natural language prompt
	m.extractCandidatesFromPrompt(ctx, payload)

	return nil
}

func (m *KnowledgeMiner) extractCandidatesFromPrompt(ctx context.Context, p TelemetryPayload) {
	if m.DB == nil {
		return
	}
	lowerPrompt := strings.ToLower(p.Prompt)

	for acronym, targetFieldID := range defaultFinancialAcronyms {
		re := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, acronym))
		if re.MatchString(lowerPrompt) {
			for _, meas := range p.ExecutedQuery.Measures {
				if meas.FieldID == targetFieldID || strings.Contains(meas.FieldID, targetFieldID) {
					m.upsertCandidate(ctx, p.TenantID, "alias", strings.ToUpper(acronym), targetFieldID, "", p.Rating)
				}
			}
		}
	}
}

func (m *KnowledgeMiner) upsertCandidate(ctx context.Context, tenantID, kind, term, targetField, expr string, rating int) {
	if m.DB == nil {
		return
	}
	baseDelta := 0.05
	if rating > 0 {
		baseDelta = 0.10
	} else if rating < 0 {
		baseDelta = -0.20
	}

	query := `
		INSERT INTO ai_knowledge_candidates (
			tenant_id, type, term, target_field_id, expression, occurrences, confidence, status
		) VALUES ($1, $2, $3, $4, $5, 1, $6, 'pending_review')
		ON CONFLICT (tenant_id, type, term) DO UPDATE
		SET occurrences = ai_knowledge_candidates.occurrences + 1,
		    confidence = GREATEST(0.000, LEAST(1.000, ai_knowledge_candidates.confidence + $7)),
		    updated_at = NOW()
	`
	initialConfidence := math.Max(0.1, math.Min(1.0, 0.5+baseDelta))
	_, _ = m.DB.ExecContext(ctx, query, tenantID, kind, term, targetField, expr, initialConfidence, baseDelta)
}

// ApproveCandidate promotes a learned pattern directly into active semantic resolution
func (m *KnowledgeMiner) ApproveCandidate(ctx context.Context, candidateID string) error {
	if m.DB == nil {
		return nil
	}
	tx, err := m.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var tenantID, kind, term, targetField string
	err = tx.QueryRowContext(ctx, `
		SELECT tenant_id, type, term, COALESCE(target_field_id, '') 
		FROM ai_knowledge_candidates WHERE id = $1
	`, candidateID).Scan(&tenantID, &kind, &term, &targetField)
	if err != nil {
		return err
	}

	if kind == "alias" && targetField != "" {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO semantic_field_aliases (tenant_id, alias_term, target_field_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (tenant_id, alias_term) DO UPDATE SET target_field_id = $3
		`, tenantID, term, targetField)
		if err != nil {
			return err
		}
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE ai_knowledge_candidates 
		SET status = 'approved', updated_at = NOW() 
		WHERE id = $1
	`, candidateID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (m *KnowledgeMiner) ListCandidates(ctx context.Context, tenantID string) ([]map[string]interface{}, error) {
	if m.DB == nil {
		return []map[string]interface{}{}, nil
	}
	rows, err := m.DB.QueryContext(ctx, `
		SELECT id, type, term, COALESCE(target_field_id, ''), COALESCE(expression, ''), occurrences, confidence, status
		FROM ai_knowledge_candidates
		WHERE tenant_id = $1 OR tenant_id = 'default'
		ORDER BY occurrences DESC, confidence DESC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id, cType, term, targetField, expr, status string
		var occurrences int
		var confidence float64
		if err := rows.Scan(&id, &cType, &term, &targetField, &expr, &occurrences, &confidence, &status); err == nil {
			list = append(list, map[string]interface{}{
				"id":              id,
				"type":            cType,
				"term":            term,
				"target_field_id": targetField,
				"expression":      expr,
				"occurrences":     occurrences,
				"confidence":      confidence,
				"status":          status,
			})
		}
	}
	return list, nil
}
