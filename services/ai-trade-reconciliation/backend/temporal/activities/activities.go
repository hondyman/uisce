package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/ai"
	"github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/models"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// ActivityContext holds shared dependencies for activity functions
type ActivityContext struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewActivityContext creates a new activity context
func NewActivityContext(db *sql.DB) *ActivityContext {
	return &ActivityContext{db: db}
}

// NewActivityContextWithHasura creates a new activity context with Hasura support
func NewActivityContextWithHasura(db *sql.DB, hasura HasuraClient) *ActivityContext {
	return &ActivityContext{db: db, hasura: hasura}
}

// Activity functions for Temporal workflow

// FetchYesterdaysTrades fetches all trades from yesterday
func FetchYesterdaysTrades(ctx context.Context, db *sql.DB) ([]models.Trade, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { trades(
	//   where: {trade_date: {_gte: $start_date, _lt: $end_date}}
	//   order_by: {trade_date: desc}
	// ) { id portfolio_id symbol action shares price trade_date settle_date custodian status metadata }}
	yesterday := time.Now().AddDate(0, 0, -1)
	startOfDay := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location())
	endOfDay := startOfDay.AddDate(0, 0, 1)

	rows, err := db.QueryContext(ctx, `
		SELECT id, portfolio_id, symbol, action, shares, price, trade_date, settle_date, custodian, status, created_at, updated_at, metadata
		FROM trades
		WHERE trade_date >= $1 AND trade_date < $2
		ORDER BY trade_date DESC
	`, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	var trades []models.Trade
	for rows.Next() {
		var t models.Trade
		if err := rows.Scan(&t.ID, &t.PortfolioID, &t.Symbol, &t.Action, &t.Shares, &t.Price,
			&t.TradeDate, &t.SettleDate, &t.Custodian, &t.Status, &t.CreatedAt, &t.UpdatedAt, &t.Metadata); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return trades, nil
}

// FetchTradeConfirms fetches unprocessed trade confirmations
func FetchTradeConfirms(ctx context.Context, db *sql.DB) ([]models.TradeConfirm, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { trade_confirms(
	//   where: {received_at: {_gt: $since}}
	//   order_by: {received_at: desc}
	// ) { id source raw_data parsed received_at created_at }}
	// Fetch confirms received in last 48 hours
	since := time.Now().Add(-48 * time.Hour)

	rows, err := db.QueryContext(ctx, `
		SELECT id, source, raw_data, parsed, received_at, created_at
		FROM trade_confirms
		WHERE received_at > $1
		ORDER BY received_at DESC
	`, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query confirms: %w", err)
	}
	defer rows.Close()

	var confirms []models.TradeConfirm
	for rows.Next() {
		var c models.TradeConfirm
		if err := rows.Scan(&c.ID, &c.Source, &c.RawData, &c.Parsed, &c.ReceivedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		confirms = append(confirms, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return confirms, nil
}

// AIReconcile performs AI-driven reconciliation
func AIReconcile(ctx context.Context, trades []models.Trade, confirms []models.TradeConfirm) (*ai.ReconcileOutput, error) {
	reconciler := ai.NewReconciler()
	return reconciler.Reconcile(ctx, trades, confirms)
}

// SaveReconciliationResult saves the result to database
func SaveReconciliationResult(ctx context.Context, db *sql.DB, output *ai.ReconcileOutput, modelVersion int) (uuid.UUID, error) {
	actCtx := &ActivityContext{db: db}
	return actCtx.SaveResult(ctx, output, modelVersion)
}

// SaveResult saves the reconciliation result with Hasura-first approach
func (ac *ActivityContext) SaveResult(ctx context.Context, output *ai.ReconcileOutput, modelVersion int) (uuid.UUID, error) {
	resultID := uuid.New()
	now := time.Now()
	discrepanciesJSON, _ := json.Marshal(output.Discrepancies)

	if ac.hasura != nil {
		id, err := ac.saveResultWithHasura(ctx, resultID, now, output, discrepanciesJSON, modelVersion)
		if err == nil {
			return id, nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via saveResultWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See saveResultWithHasura() for the Hasura mutation: mutation SaveResult
	// SQL fallback
	err := ac.db.QueryRowContext(ctx, `
		INSERT INTO reconciliation_results 
			(id, run_date, match_rate, matched_count, unmatched_count, discrepancies, model_version, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`, resultID, now, output.MatchRate, len(output.Matched),
		len(output.UnmatchedTrades)+len(output.UnmatchedConfirms),
		string(discrepanciesJSON), modelVersion, "completed", now, now).Scan(&resultID)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to save result: %w", err)
	}

	return resultID, nil
}

func (ac *ActivityContext) saveResultWithHasura(ctx context.Context, resultID uuid.UUID, now time.Time, output *ai.ReconcileOutput, discrepanciesJSON []byte, modelVersion int) (uuid.UUID, error) {
	mutation := `
		mutation SaveResult($result: reconciliation_results_insert_input!) {
			insert_reconciliation_results_one(object: $result) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"result": map[string]interface{}{
			"id":              resultID.String(),
			"run_date":        now.Format(time.RFC3339),
			"match_rate":      output.MatchRate,
			"matched_count":   len(output.Matched),
			"unmatched_count": len(output.UnmatchedTrades) + len(output.UnmatchedConfirms),
			"discrepancies":   string(discrepanciesJSON),
			"model_version":   modelVersion,
			"status":          "completed",
			"created_at":      now.Format(time.RFC3339),
			"updated_at":      now.Format(time.RFC3339),
		},
	}

	resp, err := ac.hasura.Mutate(mutation, variables)
	if err != nil {
		return uuid.Nil, err
	}

	if result, ok := resp["insert_reconciliation_results_one"].(map[string]interface{}); ok {
		if idStr, ok := result["id"].(string); ok {
			if id, err := uuid.Parse(idStr); err == nil {
				return id, nil
			}
		}
	}

	return resultID, nil
}

// CreateReconciliationTask creates a task for high-severity discrepancies
func CreateReconciliationTask(ctx context.Context, db *sql.DB, resultID uuid.UUID, discrepancy models.Discrepancy, priority string) error {
	actCtx := &ActivityContext{db: db}
	return actCtx.CreateTask(ctx, resultID, discrepancy, priority)
}

// CreateTask creates a reconciliation task with Hasura-first approach
func (ac *ActivityContext) CreateTask(ctx context.Context, resultID uuid.UUID, discrepancy models.Discrepancy, priority string) error {
	taskID := uuid.New()
	now := time.Now()

	if ac.hasura != nil {
		err := ac.createTaskWithHasura(ctx, taskID, resultID, discrepancy.ID, priority, now)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via createTaskWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See createTaskWithHasura() for the Hasura mutation: mutation CreateTask
	// SQL fallback
	_, err := ac.db.ExecContext(ctx, `
		INSERT INTO reconciliation_tasks 
			(id, result_id, discrepancy_id, status, priority, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, taskID, resultID, discrepancy.ID, "open", priority, now, now)

	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	return nil
}

func (ac *ActivityContext) createTaskWithHasura(ctx context.Context, taskID, resultID, discrepancyID uuid.UUID, priority string, now time.Time) error {
	mutation := `
		mutation CreateTask($task: reconciliation_tasks_insert_input!) {
			insert_reconciliation_tasks_one(object: $task) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"task": map[string]interface{}{
			"id":             taskID.String(),
			"result_id":      resultID.String(),
			"discrepancy_id": discrepancyID.String(),
			"status":         "open",
			"priority":       priority,
			"created_at":     now.Format(time.RFC3339),
			"updated_at":     now.Format(time.RFC3339),
		},
	}

	_, err := ac.hasura.Mutate(mutation, variables)
	return err
}

// NotifyDiscrepancy sends a notification about a discrepancy
func NotifyDiscrepancy(ctx context.Context, discrepancy models.Discrepancy) error {
	// This would integrate with your notification system
	// For now, log it
	fmt.Printf("Discrepancy Alert: %s - Severity: %s\n", discrepancy.DiscrepType, discrepancy.Severity)
	return nil
}

// AutoResolveDiscrepancy marks low-severity discrepancies as resolved
func AutoResolveDiscrepancy(ctx context.Context, db *sql.DB, discrepancy models.Discrepancy) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { update_reconciliation_tasks(
	//   where: {discrepancy_id: {_eq: $discrepancy_id}}
	//   _set: {status: "resolved", resolved_at: $now, updated_at: $now}
	// ) { affected_rows }}
	now := time.Now()
	_, err := db.ExecContext(ctx, `
		UPDATE reconciliation_tasks 
		SET status = $1, resolved_at = $2, updated_at = $3
		WHERE discrepancy_id = $4
	`, "resolved", now, now, discrepancy.ID)

	return err
}

// LogReconciliationAudit creates an audit log entry
func LogReconciliationAudit(ctx context.Context, db *sql.DB, resultID uuid.UUID, action string, details json.RawMessage) error {
	actCtx := &ActivityContext{db: db}
	return actCtx.LogAudit(ctx, resultID, action, details)
}

// LogAudit creates an audit log entry with Hasura-first approach
func (ac *ActivityContext) LogAudit(ctx context.Context, resultID uuid.UUID, action string, details json.RawMessage) error {
	auditID := uuid.New()
	now := time.Now()

	if ac.hasura != nil {
		err := ac.logAuditWithHasura(ctx, auditID, resultID, action, details, now)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via logAuditWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See logAuditWithHasura() for the Hasura mutation: mutation LogAudit
	// SQL fallback
	_, err := ac.db.ExecContext(ctx, `
		INSERT INTO reconciliation_audit_logs 
			(id, result_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, auditID, resultID, action, details, now)

	return err
}

func (ac *ActivityContext) logAuditWithHasura(ctx context.Context, auditID, resultID uuid.UUID, action string, details json.RawMessage, now time.Time) error {
	mutation := `
		mutation LogAudit($log: reconciliation_audit_logs_insert_input!) {
			insert_reconciliation_audit_logs_one(object: $log) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"log": map[string]interface{}{
			"id":         auditID.String(),
			"result_id":  resultID.String(),
			"action":     action,
			"details":    string(details),
			"created_at": now.Format(time.RFC3339),
		},
	}

	_, err := ac.hasura.Mutate(mutation, variables)
	return err
}
