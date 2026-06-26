package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type ValidationFunc func(raw string) bool

type AIOrchestrator struct {
	db       *sql.DB
	llm      llm.LLMProvider
	registry map[string]AIStrategy
}

func NewAIOrchestrator(db *sql.DB, llm llm.LLMProvider) *AIOrchestrator {
	return &AIOrchestrator{
		db:       db,
		llm:      llm,
		registry: make(map[string]AIStrategy),
	}
}

func (o *AIOrchestrator) RegisterStrategy(reqType string, strategy AIStrategy) {
	o.registry[reqType] = strategy
}

// Enqueue adds a new request to the queue
func (o *AIOrchestrator) Enqueue(ctx context.Context, reqType string, payload any) (string, error) {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	id := uuid.New().String()
	_, err = o.db.ExecContext(ctx, `
		INSERT INTO ai_requests (id, type, payload, status)
		VALUES ($1, $2, $3, $4)
	`, id, reqType, bytes, StatusPending)

	return id, err
}

// StartWorker begins the background polling loop
func (o *AIOrchestrator) StartWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.processNext(context.Background())
			}
		}
	}()
}

func (o *AIOrchestrator) processNext(ctx context.Context) {
	// Simple optimistic locking / fetch
	// In production, use SELECT FOR UPDATE SKIP LOCKED
	var req AIRequest
	err := o.db.QueryRowContext(ctx, `
		SELECT id, type, payload, attempts
		FROM ai_requests
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, StatusPending).Scan(&req.ID, &req.Type, &req.Payload, &req.Attempts)

	if err == sql.ErrNoRows {
		return
	} else if err != nil {
		log.Printf("[AI Orchestrator] Error fetching next request: %v", err)
		return
	}

	// Mark as running
	_, _ = o.db.ExecContext(ctx, "UPDATE ai_requests SET status = $1 WHERE id = $2", StatusRunning, req.ID)

	// Execute
	err = o.executeRequest(ctx, &req)

	// Update final status
	status := StatusSuccess
	errMsg := ""
	if err != nil {
		status = StatusFailed
		errMsg = err.Error()
		// Simple retry logic could go here (e.g. check attempts < max)
		if req.Attempts < 3 {
			status = StatusPending // Retry
		}
	}

	_, _ = o.db.ExecContext(ctx, `
		UPDATE ai_requests 
		SET status = $1, error = $2, attempts = attempts + 1, updated_at = now()
		WHERE id = $3
	`, status, errMsg, req.ID)
}

func (o *AIOrchestrator) executeRequest(ctx context.Context, req *AIRequest) error {
	strategy, ok := o.registry[req.Type]
	if !ok {
		return fmt.Errorf("no strategy registered for type: %s", req.Type)
	}

	systemPrompt, userPrompt := strategy.BuildPrompt(req.Payload)
	fullPrompt := fmt.Sprintf("%s\n\n%s", systemPrompt, userPrompt)

	// Call LLM
	raw, err := o.llm.GenerateResponse(ctx, fullPrompt)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Validate (simple auto-correction attempt could go here)
	if !strategy.Validate(raw) {
		return fmt.Errorf("output validation failed for type %s", req.Type)
	}

	// Parse
	parsed, err := strategy.Parse(raw)
	if err != nil {
		return fmt.Errorf("parsing failed: %w", err)
	}

	// Save Output
	outputBytes, _ := json.Marshal(parsed)
	_, _ = o.db.ExecContext(ctx, "UPDATE ai_requests SET output = $1 WHERE id = $2", outputBytes, req.ID)

	// Apply side effects
	if err := strategy.Apply(ctx, parsed); err != nil {
		return fmt.Errorf("apply failed: %w", err)
	}

	return nil
}
