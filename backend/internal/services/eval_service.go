package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// EvalService runs evaluation suites against the NLQ engine.
type EvalService struct {
	db         *sqlx.DB
	nlqService *NLQService
}

// NewEvalService creates a new EvalService.
func NewEvalService(db *sqlx.DB, nlqService *NLQService) *EvalService {
	return &EvalService{
		db:         db,
		nlqService: nlqService,
	}
}

// EvalCase represents a test case from the database.
type EvalCase struct {
	ID             uuid.UUID `db:"id"`
	TenantID       string    `db:"tenant_id"`
	Question       string    `db:"question"`
	ExpectedAnswer string    `db:"expected_answer"`
}

// RunEval executes all test cases for a tenant and stores the results.
func (s *EvalService) RunEval(ctx context.Context, tenantID string) (uuid.UUID, error) {
	runID := uuid.New()

	// 1. Fetch test cases
	var cases []EvalCase
	err := s.db.SelectContext(ctx, &cases, "SELECT id, tenant_id, question, expected_answer FROM nlq_eval_cases WHERE tenant_id = $1", tenantID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to fetch eval cases: %w", err)
	}

	if len(cases) == 0 {
		return uuid.Nil, fmt.Errorf("no eval cases found for tenant %s", tenantID)
	}

	// 2. Run each case
	for _, c := range cases {
		start := time.Now()

		// Call NLQ Service
		req := AskRequest{
			Question: c.Question,
		}
		secCtx := &security.Context{
			TenantID:     c.TenantID,
			DatasourceID: "default",
			Region:       "default",
		}

		resp, err := s.nlqService.Ask(ctx, secCtx, req)

		latency := time.Since(start).Milliseconds()
		var actualAnswer string
		var errorMsg string
		var isCorrect bool

		if err != nil {
			errorMsg = err.Error()
			isCorrect = false
		} else {
			actualAnswer = resp.Answer
			// Simple exact match for now - in production use semantic similarity
			isCorrect = (actualAnswer == c.ExpectedAnswer)
		}

		// 3. Store result
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO nlq_eval_results (run_id, case_id, actual_answer, is_correct, latency_ms, error_message)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, runID, c.ID, actualAnswer, isCorrect, latency, errorMsg)

		if err != nil {
			// Log error but continue
			fmt.Printf("Failed to save result for case %s: %v\n", c.ID, err)
		}
	}

	return runID, nil
}
