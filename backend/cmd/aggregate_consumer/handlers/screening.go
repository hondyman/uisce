package handlers

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
)

// InvestmentScreening transposes the trigger `screen_investment_opportunity`.
//
// The original trigger ran BEFORE INSERT OR UPDATE on investment_opportunities,
// computed a screening score, set screening_passed/reasons/score on the NEW row,
// and auto-advanced stage INTAKE -> INITIAL_SCREEN if passed.
//
// Differences from the trigger:
//   - The trigger mutated NEW in-place before INSERT/UPDATE, so the database
//     always reflected post-screening state. The consumer is async, so the
//     screening fields land on the row after the CDC event has been emitted.
//     Downstream readers that need the screening fields must tolerate a brief
//     eventual-consistency lag (the same lag already accepted for cube_versions
//     and template_rbac).
//   - REPLICA IDENTITY FULL is required so we can read OLD.current_stage to
//     decide whether to auto-advance.
//
// Note: the trigger's RAISE EXCEPTION for bad data is NOT replicated here —
// that's enforcement at save time and now lives in CHECK constraints /
// business-object semantic-terms layer (see AGENTS.md).
type InvestmentScreening struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const screeningHandler = "investment_screening"

// screeningOwnedColumns are the columns that InvestmentScreening writes back
// to investment_opportunities. If every changed column in an event belongs
// exclusively to this set, the event originated from the handler's own write
// and must be skipped to prevent a feedback loop.
var screeningOwnedColumns = map[string]bool{
	"screening_passed":        true,
	"screening_reasons":       true,
	"screening_score":        true,
	"screening_completed_at":  true,
	"current_stage":          true,
	"stage_updated_at":       true,
	"stage_history":          true,
	"updated_at":             true,
}

func isScreeningSelfWrite(evt *Event) bool {
	if len(evt.ChangedColumns) == 0 {
		return false
	}
	for _, col := range evt.ChangedColumns {
		if !screeningOwnedColumns[col] {
			return false
		}
	}
	return true
}

func NewInvestmentScreening(pool *pgxpool.Pool, d *dedupe.Store) *InvestmentScreening {
	return &InvestmentScreening{pool: pool, dedupe: d}
}

func (h *InvestmentScreening) Topic() string { return "alpha_trg.public.investment_opportunities" }

func (h *InvestmentScreening) Handle(ctx context.Context, evt Event) error {
	if evt.Op != "c" && evt.Op != "u" {
		return nil
	}

	// Skip events that are the result of our own screening write (CDC echo).
	// Only skip if ALL changed columns are owned by this handler — any
	// external write to those columns (unlikely but possible) still needs
	// a fresh screening pass.
	if isScreeningSelfWrite(&evt) {
		return nil
	}

	if err := h.dedupe.MarkIfNew(ctx, screeningHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	clientID := stringOf(evt.After["client_id"])
	minimumCommitment := numericOf(evt.After["minimum_commitment"])
	if clientID == "" {
		return fmt.Errorf("screen: empty client_id")
	}

	// Read client's liquid assets from alternative_investments (legacy public table).
	// TODO: when altinv.alternative_investment STI is the source of truth, swap this.
	const liquidAssetsQuery = `
SELECT COALESCE(SUM(current_nav), 0)::numeric
FROM   alternative_investments
WHERE  client_id = $1::uuid
`
	var liquidAssets big.Float
	if err := h.pool.QueryRow(ctx, liquidAssetsQuery, clientID).Scan(&liquidAssets); err != nil {
		return fmt.Errorf("liquid assets for client %s: %w", clientID, err)
	}

	passed := true
	var reasons []string
	score := 100.0

	// Check 1: minimum_commitment vs 10% of AUM
	limit := new(big.Float).Mul(&liquidAssets, big.NewFloat(0.10))
	mm := big.NewFloat(minimumCommitment)
	if mm.Cmp(limit) > 0 {
		passed = false
		reasons = append(reasons, "Minimum commitment exceeds 10% of alternative AUM")
		score -= 20
	}

	// Check 2: vintage_year 2025-2028
	if v, ok := evt.After["vintage_year"].(float64); ok && (v < 2025 || v > 2028) {
		reasons = append(reasons, "Vintage year outside preferred 2025-2028 range")
		score -= 10
	}

	// Check 3: track_record_years_min >= 5
	if v, ok := evt.After["track_record_years_min"].(float64); ok && v < 5 {
		reasons = append(reasons, "Manager track record less than 5 years")
		score -= 15
	}

	// Check 4: target_irr_min <= 35
	if v, ok := evt.After["target_irr_min"].(float64); ok && v > 35 {
		reasons = append(reasons, "Target IRR appears unrealistically high")
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	// Determine stage transition: original logic only advanced INTAKE -> INITIAL_SCREEN
	// on insert. For updates, we preserve current_stage (no auto-advance).
	currentStage := stringOf(evt.After["current_stage"])
	newStage := currentStage
	var historyAppend string

	if evt.Op == "c" {
		if passed && currentStage == "INTAKE" {
			newStage = "INITIAL_SCREEN"
			historyAppend = `{"stage":"INITIAL_SCREEN","timestamp":"now","notes":"Automated screening passed"}`
		} else if !passed && currentStage == "INTAKE" {
			historyAppend = fmt.Sprintf(`{"stage":"INTAKE","timestamp":"now","notes":"Automated screening flagged issues: %s"}`, joinReasons(reasons))
		}
	}

	const applyScreening = `
UPDATE investment_opportunities
SET    screening_passed      = $2,
       screening_reasons     = $3,
       screening_score       = $4,
       screening_completed_at = NOW(),
       current_stage         = COALESCE(NULLIF($5, ''), current_stage),
       stage_updated_at      = CASE WHEN $5 <> '' THEN NOW() ELSE stage_updated_at END,
       stage_history         = COALESCE(stage_history, '[]'::jsonb) || COALESCE(NULLIF($6::jsonb, 'null'::jsonb), '[]'::jsonb),
       updated_at            = NOW()
WHERE  opportunity_id = $1::uuid
`
	var historyJSON string
	if historyAppend != "" {
		historyJSON = fmt.Sprintf("[%s]", historyAppend)
	}
	if _, err := h.pool.Exec(ctx, applyScreening,
		stringOf(evt.After["opportunity_id"]),
		passed,
		reasons,
		score,
		newStage,
		historyJSON,
	); err != nil {
		return fmt.Errorf("apply screening for opportunity %s: %w", evt.After["opportunity_id"], err)
	}
	return nil
}

func joinReasons(rs []string) string {
	out := ""
	for i, r := range rs {
		if i > 0 {
			out += "; "
		}
		out += r
	}
	return out
}
