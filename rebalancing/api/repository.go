package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/db/queries"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository provides direct DB access for the rebalancing REST API,
// replacing the client-side Hasura GraphQL queries/subscriptions that
// used to serve the frontend dashboards directly.
type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Close() {
	r.pool.Close()
}

// PortfolioSummary mirrors the fields the dashboards previously read via
// the `portfolios` GraphQL subscription (RiskAlphaDashboard,
// AttributionAlphaDashboard, AISimulationDashboard, AIRebalancingDashboard).
type PortfolioSummary struct {
	ID                string                 `json:"id"`
	TenantID          string                 `json:"tenant_id"`
	Name              string                 `json:"name"`
	AUM               float64                `json:"aum"`
	Drift             float64                `json:"drift"`
	LastRebalance     *string                `json:"last_rebalance"`
	TargetModel       map[string]interface{} `json:"target_model"`
	Constraints       map[string]interface{} `json:"constraints"`
	RebalanceStatus   string                 `json:"rebalance_status"`
	RiskScore         float64                `json:"risk_score"`
	Alpha             float64                `json:"alpha"`
	SectorAttribution map[string]interface{} `json:"sector_attribution"`
	TaxSaved          float64                `json:"tax_saved"`
	PolicyDocument    *string                `json:"policy_document"`
	HoldingsCount     int                    `json:"holdings_count"`
}

// ListPortfolios returns all portfolios for a tenant, replacing the
// `PORTFOLIOS_SUB` / `PortfoliosRisk` / `PortfoliosAttribution` GraphQL
// subscriptions.
func (r *Repository) ListPortfolios(ctx context.Context, tenantID uuid.UUID) ([]PortfolioSummary, error) {
	rows, err := r.pool.Query(ctx, queries.ListPortfolios, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list portfolios: %w", err)
	}
	defer rows.Close()

	portfolios := []PortfolioSummary{}
	for rows.Next() {
		var p PortfolioSummary
		var targetModel, constraints, sectorAttribution []byte
		if err := rows.Scan(
			&p.ID, &p.TenantID, &p.Name, &p.AUM, &p.Drift, &p.LastRebalance,
			&targetModel, &constraints, &p.RebalanceStatus,
			&p.RiskScore, &p.Alpha, &sectorAttribution, &p.TaxSaved,
			&p.PolicyDocument, &p.HoldingsCount,
		); err != nil {
			return nil, fmt.Errorf("failed to scan portfolio: %w", err)
		}
		p.TargetModel = decodeJSONMap(targetModel)
		p.Constraints = decodeJSONMap(constraints)
		p.SectorAttribution = decodeJSONMap(sectorAttribution)
		portfolios = append(portfolios, p)
	}
	return portfolios, rows.Err()
}

// RebalancePlanSummary mirrors the fields read via the `rebalance_plans`
// GraphQL subscription in AIRebalancingDashboard.
type RebalancePlanSummary struct {
	ID             string  `json:"id"`
	PortfolioID    string  `json:"portfolio_id"`
	Timestamp      string  `json:"timestamp"`
	CurrentDrift   float64 `json:"current_drift"`
	ExpectedDrift  float64 `json:"expected_drift"`
	TaxSavings     float64 `json:"tax_savings"`
	Confidence     float64 `json:"confidence"`
	Status         string  `json:"status"`
	Rationale      string  `json:"rationale"`
	Summary        string  `json:"summary"`
	ProposedTrades string  `json:"proposed_trades"`
}

// ListRebalancePlans returns the most recent rebalance plans for a
// portfolio, replacing the `REBALANCE_PLANS_SUB` / `Plans` GraphQL
// subscription.
func (r *Repository) ListRebalancePlans(ctx context.Context, portfolioID uuid.UUID, limit int) ([]RebalancePlanSummary, error) {
	rows, err := r.pool.Query(ctx, queries.ListRebalancePlans, portfolioID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list rebalance plans: %w", err)
	}
	defer rows.Close()

	plans := []RebalancePlanSummary{}
	for rows.Next() {
		var p RebalancePlanSummary
		if err := rows.Scan(
			&p.ID, &p.PortfolioID, &p.Timestamp, &p.CurrentDrift, &p.ExpectedDrift,
			&p.TaxSavings, &p.Confidence, &p.Status, &p.Rationale, &p.Summary, &p.ProposedTrades,
		); err != nil {
			return nil, fmt.Errorf("failed to scan rebalance plan: %w", err)
		}
		plans = append(plans, p)
	}
	return plans, rows.Err()
}

func decodeJSONMap(raw []byte) map[string]interface{} {
	m := map[string]interface{}{}
	if len(raw) == 0 {
		return m
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}
