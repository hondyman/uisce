package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/jmoiron/sqlx"
)

type ComplianceRepository interface {
	ListComplianceRules(ctx context.Context, includeInactive bool) ([]ComplianceRule, error)
	ListComplianceEvaluations(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) ([]ComplianceEvaluation, error)
	ListComplianceBreaches(ctx context.Context, portfolioID uuid.UUID, status string) ([]ComplianceBreach, error)
	GetLineageForEvaluation(ctx context.Context, evaluationID uuid.UUID) ([]ComplianceLineage, error)
}

type pgComplianceRepo struct {
	*db.BitemporalRepository[ComplianceRule]
	sqlxDB *sqlx.DB
}

func NewComplianceRepository(sqlxDB *sqlx.DB) ComplianceRepository {
	return &pgComplianceRepo{
		BitemporalRepository: db.NewBitemporalRepository[ComplianceRule](sqlxDB, "edm.compliance_rule", "rule_id"),
		sqlxDB:              sqlxDB,
	}
}

func (r *pgComplianceRepo) ListComplianceRules(ctx context.Context, includeInactive bool) ([]ComplianceRule, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	opts := []db.QueryOption{db.WithOrderBy("rule_code", "ASC")}
	if !includeInactive {
		opts = append(opts, db.WithFilter("status", "ACTIVE"))
	}

	records, err := r.BitemporalRepository.ListCurrent(ctx, tenantID, opts...)
	if err != nil {
		return nil, err
	}

	result := make([]ComplianceRule, len(records))
	for i, rec := range records {
		result[i] = *rec
	}
	return result, nil
}

func (r *pgComplianceRepo) ListComplianceEvaluations(ctx context.Context, portfolioID uuid.UUID, asOfDate time.Time) ([]ComplianceEvaluation, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `SELECT * FROM edm.compliance_evaluation WHERE tenant_id = $1 AND portfolio_id = $2 AND valuation_date = $3 ORDER BY evaluated_at DESC`

	var evals []ComplianceEvaluation
	err := r.sqlxDB.SelectContext(ctx, &evals, query, tenantID, portfolioID, asOfDate)
	return evals, err
}

func (r *pgComplianceRepo) ListComplianceBreaches(ctx context.Context, portfolioID uuid.UUID, status string) ([]ComplianceBreach, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `SELECT * FROM edm.compliance_breach WHERE tenant_id = $1 AND portfolio_id = $2`
	args := []interface{}{tenantID, portfolioID}

	if status != "" {
		query += ` AND status = $3`
		args = append(args, status)
	}

	query += ` ORDER BY created_at DESC`

	var breaches []ComplianceBreach
	err := r.sqlxDB.SelectContext(ctx, &breaches, query, args...)
	return breaches, err
}

func (r *pgComplianceRepo) GetLineageForEvaluation(ctx context.Context, evaluationID uuid.UUID) ([]ComplianceLineage, error) {
	tenantID, ok := ctx.Value("tenant_id").(uuid.UUID)
	if !ok {
		return nil, fmt.Errorf("tenant_id not found in context")
	}

	query := `SELECT * FROM edm.compliance_lineage WHERE tenant_id = $1 AND evaluation_id = $2 ORDER BY processed_at ASC`

	var lineages []ComplianceLineage
	err := r.sqlxDB.SelectContext(ctx, &lineages, query, tenantID, evaluationID)
	return lineages, err
}
