package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Close() {
	r.pool.Close()
}

type Portfolio struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Name            string          `json:"name"`
	AUM             float64         `json:"aum"`
	Drift           float64         `json:"drift"`
	LastRebalance   *string         `json:"last_rebalance"`
	TargetModel     json.RawMessage `json:"target_model"`
	Constraints     json.RawMessage `json:"constraints"`
	Holdings        []Holding       `json:"holdings"`
}

type Holding struct {
	ID            uuid.UUID `json:"id"`
	Symbol        string    `json:"symbol"`
	Shares        float64   `json:"shares"`
	CurrentPrice  float64   `json:"current_price"`
	CostBasis     float64   `json:"cost_basis"`
	PurchaseDate  string    `json:"purchase_date"`
	TaxLotID      string    `json:"tax_lot_id"`
	Sector        string    `json:"sector"`
}

func (r *Repository) SetTenant(ctx context.Context, tenantID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "SELECT set_config('app.current_tenant', $1, false)", tenantID.String())
	return err
}

func (r *Repository) GetPortfolio(ctx context.Context, id uuid.UUID) (*Portfolio, error) {
	var p Portfolio
	err := r.pool.QueryRow(ctx, `
		SELECT id, tenant_id, name, aum, drift, last_rebalance::text, target_model, constraints
		FROM portfolios WHERE id = $1
	`, id).Scan(&p.ID, &p.TenantID, &p.Name, &p.AUM, &p.Drift, &p.LastRebalance, &p.TargetModel, &p.Constraints)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("portfolio not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get portfolio: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, symbol, shares, current_price, cost_basis, purchase_date, tax_lot_id, sector
		FROM portfolio_holdings WHERE portfolio_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get holdings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var h Holding
		if err := rows.Scan(&h.ID, &h.Symbol, &h.Shares, &h.CurrentPrice, &h.CostBasis, &h.PurchaseDate, &h.TaxLotID, &h.Sector); err != nil {
			return nil, fmt.Errorf("failed to scan holding: %w", err)
		}
		p.Holdings = append(p.Holdings, h)
	}

	return &p, nil
}

func (r *Repository) UpdatePortfolioState(ctx context.Context, id uuid.UUID, drift float64, taxSaved float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE portfolios
		SET drift = $2,
			tax_saved = $3,
			last_rebalance = NOW(),
			rebalance_status = 'completed',
			updated_at = NOW()
		WHERE id = $1
	`, id, drift, taxSaved)
	return err
}

type RebalancePlan struct {
	ID             uuid.UUID `json:"id"`
	PortfolioID    uuid.UUID `json:"portfolio_id"`
	Timestamp      string    `json:"timestamp"`
	CurrentDrift   float64   `json:"current_drift"`
	ExpectedDrift  float64   `json:"expected_drift"`
	TaxSavings     float64   `json:"tax_savings"`
	Confidence     float64   `json:"confidence"`
	Status         string    `json:"status"`
	Rationale      string    `json:"rationale"`
	ProposedTrades string    `json:"proposed_trades"`
	TaxAnalysis    string    `json:"tax_analysis"`
}

func (r *Repository) InsertRebalancePlan(ctx context.Context, plan *RebalancePlan) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO rebalance_plans (
			portfolio_id, timestamp, current_drift, expected_drift,
			tax_savings, confidence, status, rationale, proposed_trades, tax_analysis
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, plan.PortfolioID, plan.Timestamp, plan.CurrentDrift, plan.ExpectedDrift,
		plan.TaxSavings, plan.Confidence, plan.Status, plan.Rationale,
		plan.ProposedTrades, plan.TaxAnalysis)
	return err
}

func (r *Repository) UpdatePlanSummary(ctx context.Context, id uuid.UUID, summary string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE rebalance_plans SET summary = $2, updated_at = NOW() WHERE id = $1
	`, id, summary)
	return err
}

func (r *Repository) InsertAuditLog(ctx context.Context, userID, tenantID uuid.UUID, action, resource, resourceID string, allowed bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, user_id, action, resource, resource_id, allowed)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, tenantID, userID, action, resource, resourceID, allowed)
	return err
}

func (r *Repository) WithTransaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
