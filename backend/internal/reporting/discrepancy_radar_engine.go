package reporting

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type DiscrepancyRadarEngine struct {
	db *sqlx.DB
}

func NewDiscrepancyRadarEngine(db *sqlx.DB) *DiscrepancyRadarEngine {
	return &DiscrepancyRadarEngine{db: db}
}

// EvaluateStatementBatch executes comparative variance analysis against baseline records
func (e *DiscrepancyRadarEngine) EvaluateStatementBatch(
	ctx context.Context,
	tenantID uuid.UUID,
	statementID string,
	portfolioID string,
	effectiveDate time.Time,
	makerIdentity string,
	baselineMetrics map[string]float64,
	currentMetrics map[string]float64,
	rules []VarianceRuleDef,
) (*StatementApprovalTicket, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	ticket := &StatementApprovalTicket{
		TicketID:      uuid.New(),
		TenantID:      tenantID,
		StatementID:   statementID,
		PortfolioID:   portfolioID,
		EffectiveDate: effectiveDate,
		MakerIdentity: makerIdentity,
		Status:        ReviewStatusAutoApproved,
		CreatedAt:     time.Now().UTC(),
		Differences:   make([]RedlineDifferenceItem, 0),
	}

	var compositeScore float64
	totalBreaches := 0
	criticalBreaches := 0

	for _, rule := range rules {
		baseVal, baseExists := baselineMetrics[rule.FieldKey]
		currVal, currExists := currentMetrics[rule.FieldKey]

		if !baseExists || !currExists {
			continue
		}

		diff := currVal - baseVal
		var diffPct, diffBps float64
		if baseVal != 0 {
			diffPct = (diff / math.Abs(baseVal)) * 100.0
			diffBps = (diff / math.Abs(baseVal)) * 10000.0
		}

		isBreached := false
		diagnostic := "Normal variance within tolerance"

		switch rule.ThresholdType {
		case "BPS":
			if math.Abs(diffBps) > rule.ThresholdVal {
				isBreached = true
				excess := math.Abs(diffBps) - rule.ThresholdVal
				compositeScore += rule.Weight * (excess / rule.ThresholdVal)
				diagnostic = fmt.Sprintf("Drift of %.1f bps exceeds tolerance threshold %.1f bps", diffBps, rule.ThresholdVal)
			}
		case "PERCENT":
			if math.Abs(diffPct) > rule.ThresholdVal {
				isBreached = true
				excess := math.Abs(diffPct) - rule.ThresholdVal
				compositeScore += rule.Weight * (excess / rule.ThresholdVal)
				diagnostic = fmt.Sprintf("Variance of %.2f%% exceeds tolerance threshold %.2f%%", diffPct, rule.ThresholdVal)
			}
		case "ABSOLUTE_DIFF":
			if math.Abs(diff) > rule.ThresholdVal {
				isBreached = true
				excess := math.Abs(diff) - rule.ThresholdVal
				compositeScore += rule.Weight * (excess / rule.ThresholdVal)
				diagnostic = fmt.Sprintf("Absolute shift of $%.2f exceeds limit $%.2f", diff, rule.ThresholdVal)
			}
		}

		if isBreached {
			totalBreaches++
			if rule.Severity == SeverityCritical {
				criticalBreaches++
			}
		}

		ticket.Differences = append(ticket.Differences, RedlineDifferenceItem{
			ItemKey:          rule.FieldKey,
			DisplayName:      rule.RuleName,
			BaselineValue:    baseVal,
			CurrentValue:     currVal,
			VarianceAmount:   diff,
			VarianceBps:      diffBps,
			VariancePct:      diffPct,
			Severity:         rule.Severity,
			BreachDiagnostic: diagnostic,
			IsBreached:       isBreached,
		})
	}

	ticket.AnomalyScore = compositeScore
	ticket.TotalBreaches = totalBreaches
	ticket.CriticalBreaches = criticalBreaches

	if compositeScore >= 50.0 || criticalBreaches > 0 {
		ticket.Status = ReviewStatusPendingApproval
	} else if compositeScore > 0 {
		ticket.Status = ReviewStatusPendingApproval
	}

	// Persist to relational approval queue if state requires Maker-Checker signoff
	if e.db != nil && ticket.Status == ReviewStatusPendingApproval {
		query := `
			INSERT INTO reporting.statement_approval_queue (
				ticket_id, tenant_id, statement_id, portfolio_id, effective_date,
				maker_identity, status, anomaly_score, total_breaches, critical_breaches, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);
		`
		_, err := e.db.ExecContext(ctx, query,
			ticket.TicketID, ticket.TenantID, ticket.StatementID, ticket.PortfolioID, ticket.EffectiveDate,
			ticket.MakerIdentity, ticket.Status, ticket.AnomalyScore, ticket.TotalBreaches, ticket.CriticalBreaches, ticket.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed staging statement approval ticket: %w", err)
		}
	}

	return ticket, nil
}

// ProcessCheckerDecision applies 4-eyes approval/rejection rules (Maker != Checker)
func (e *DiscrepancyRadarEngine) ProcessCheckerDecision(
	ctx context.Context,
	tenantID uuid.UUID,
	ticketID uuid.UUID,
	checkerIdentity string,
	decision ReviewStatus,
	notes string,
) (*StatementApprovalTicket, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	// 4-Eyes Compliance Guard: Checker cannot be the same user who initiated the batch run
	var makerIdentity string
	if e.db != nil {
		err := e.db.GetContext(ctx, &makerIdentity, `
			SELECT maker_identity FROM reporting.statement_approval_queue 
			WHERE ticket_id = $1 AND tenant_id = $2;
		`, ticketID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("ticket not found or unauthorized: %w", err)
		}

		if makerIdentity == checkerIdentity {
			return nil, fmt.Errorf("4-Eyes Rule Violation: Checker (%s) cannot be the same identity as Maker (%s)", checkerIdentity, makerIdentity)
		}

		now := time.Now().UTC()
		_, err = e.db.ExecContext(ctx, `
			UPDATE reporting.statement_approval_queue
			SET status = $1, checker_identity = $2, checker_notes = $3, decided_at = $4
			WHERE ticket_id = $5 AND tenant_id = $6;
		`, decision, checkerIdentity, notes, now, ticketID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("failed recording checker decision: %w", err)
		}
	}

	return &StatementApprovalTicket{
		TicketID:        ticketID,
		TenantID:        tenantID,
		CheckerIdentity: &checkerIdentity,
		Status:          decision,
		CheckerNotes:    notes,
	}, nil
}
