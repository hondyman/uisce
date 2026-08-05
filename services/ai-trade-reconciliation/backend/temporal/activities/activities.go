package activities

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/services/ai-trade-reconciliation/backend/internal/ai"
	"github.com/hondyman/uisce/services/ai-trade-reconciliation/backend/internal/models"
)

// ActivityContext holds shared dependencies for activity functions
type ActivityContext struct {
	db *sql.DB
}

// NewActivityContext creates a new activity context
func NewActivityContext(db *sql.DB) *ActivityContext {
	return &ActivityContext{db: db}
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

// SaveResult saves the reconciliation result
func (ac *ActivityContext) SaveResult(ctx context.Context, output *ai.ReconcileOutput, modelVersion int) (uuid.UUID, error) {
	resultID := uuid.New()
	now := time.Now()
	discrepanciesJSON, _ := json.Marshal(output.Discrepancies)

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

// CreateReconciliationTask creates a task for high-severity discrepancies
func CreateReconciliationTask(ctx context.Context, db *sql.DB, resultID uuid.UUID, discrepancy models.Discrepancy, priority string) error {
	actCtx := &ActivityContext{db: db}
	return actCtx.CreateTask(ctx, resultID, discrepancy, priority)
}

// CreateTask creates a reconciliation task
func (ac *ActivityContext) CreateTask(ctx context.Context, resultID uuid.UUID, discrepancy models.Discrepancy, priority string) error {
	taskID := uuid.New()
	now := time.Now()

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

// LogAudit creates an audit log entry
func (ac *ActivityContext) LogAudit(ctx context.Context, resultID uuid.UUID, action string, details json.RawMessage) error {
	auditID := uuid.New()
	now := time.Now()

	_, err := ac.db.ExecContext(ctx, `
		INSERT INTO reconciliation_audit_logs
			(id, result_id, action, details, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`, auditID, resultID, action, details, now)

	return err
}
