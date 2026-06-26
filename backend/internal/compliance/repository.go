package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ComplianceRepository defines persistence operations for the Compliance domain
type ComplianceRepository interface {
	ListComplianceRules(ctx context.Context, includeInactive bool) ([]ComplianceRule, error)
	ListComplianceEvaluations(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) ([]ComplianceEvaluation, error)
	ListComplianceBreaches(ctx context.Context, portfolioID uuid.UUID, status string) ([]ComplianceBreach, error)
	GetLineageForEvaluation(ctx context.Context, evaluationID uuid.UUID) ([]ComplianceLineage, error)
}

type pgComplianceRepo struct {
	db *sqlx.DB
}

// NewComplianceRepository creates a new Postgres-backed ComplianceRepository
func NewComplianceRepository(db *sqlx.DB) ComplianceRepository {
	return &pgComplianceRepo{db: db}
}

func (r *pgComplianceRepo) ListComplianceRules(ctx context.Context, includeInactive bool) ([]ComplianceRule, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.compliance_rule
		WHERE tenant_id = $1 AND valid_to = 'infinity'
	`
	if !includeInactive {
		query += ` AND status = 'ACTIVE'`
	}
	query += ` ORDER BY rule_code ASC`

	var rules []ComplianceRule
	err := r.db.SelectContext(ctx, &rules, query, tenantID)
	return rules, err
}

func (r *pgComplianceRepo) ListComplianceEvaluations(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) ([]ComplianceEvaluation, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.compliance_evaluation
		WHERE tenant_id = $1 AND portfolio_id = $2 AND valuation_date = $3
		ORDER BY evaluated_at DESC
	`

	var evals []ComplianceEvaluation
	err := r.db.SelectContext(ctx, &evals, query, tenantID, portfolioID, asOfDate)
	return evals, err
}

func (r *pgComplianceRepo) ListComplianceBreaches(ctx context.Context, portfolioID uuid.UUID, status string) ([]ComplianceBreach, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.compliance_breach
		WHERE tenant_id = $1 AND portfolio_id = $2
	`
	args := []interface{}{tenantID, portfolioID}

	if status != "" {
		query += ` AND status = $3`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC`

	var breaches []ComplianceBreach
	err := r.db.SelectContext(ctx, &breaches, query, args...)
	return breaches, err
}

func (r *pgComplianceRepo) GetLineageForEvaluation(ctx context.Context, evaluationID uuid.UUID) ([]ComplianceLineage, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `
		SELECT * FROM edm.compliance_lineage
		WHERE tenant_id = $1 AND evaluation_id = $2
		ORDER BY processed_at ASC
	`

	var lineages []ComplianceLineage
	err := r.db.SelectContext(ctx, &lineages, query, tenantID, evaluationID)
	return lineages, err
}
