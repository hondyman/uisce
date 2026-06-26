package factor

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// FactorModel represents a collection of risk factors (e.g., Fama-French)
type FactorModel struct {
	ID          uuid.UUID `db:"model_id" json:"id"`
	Slug        string    `db:"slug" json:"slug"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// FactorDefinition represents a single risk factor (e.g., SMB)
type FactorDefinition struct {
	ID          uuid.UUID `db:"factor_id" json:"id"`
	ModelID     uuid.UUID `db:"model_id" json:"model_id"`
	Slug        string    `db:"slug" json:"slug"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// FactorReturn represents a single daily return observation
type FactorReturn struct {
	FactorID    uuid.UUID `db:"factor_id" json:"factor_id"`
	Date        time.Time `db:"date" json:"date"`
	ReturnValue float64   `db:"return_value" json:"return_value"`
}

type Service struct {
	db           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sql.DB) *Service {
	return &Service{db: sqlx.NewDb(db, "postgres")}
}

func NewServiceWithHasura(db *sql.DB, hasuraClient HasuraClient) *Service {
	return &Service{
		db:           sqlx.NewDb(db, "postgres"),
		hasuraClient: hasuraClient,
	}
}

// CreateModel creates a new factor model
func (s *Service) CreateModel(ctx context.Context, model *FactorModel) error {
	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	return s.createModelRecord(ctx, model)
}

// CreateFactor creates a new factor definition within a model
func (s *Service) CreateFactor(ctx context.Context, factor *FactorDefinition) error {
	if factor.ID == uuid.Nil {
		factor.ID = uuid.New()
	}
	return s.createFactorRecord(ctx, factor)
}

// IngestReturns bulk inserts factor returns
func (s *Service) IngestReturns(ctx context.Context, returns []FactorReturn) error {
	return s.ingestReturnsRecords(ctx, returns)
}

// GetModelBySlug retrieves a model and its factors by slug
func (s *Service) GetModelBySlug(ctx context.Context, slug string) (*FactorModel, []FactorDefinition, error) {
	model, err := s.getModelBySlugRecord(ctx, slug)
	if err != nil {
		return nil, nil, err
	}

	factors, err := s.getFactorsByModelIDRecords(ctx, model.ID)
	if err != nil {
		return nil, nil, err
	}

	return model, factors, nil
}

// GetFactorReturns retrieves returns for a specific factor within a date range
func (s *Service) GetFactorReturns(ctx context.Context, factorID uuid.UUID, startDate, endDate time.Time) ([]FactorReturn, error) {
	return s.getFactorReturnsRecords(ctx, factorID, startDate, endDate)
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (s *Service) createModelRecord(ctx context.Context, model *FactorModel) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for NamedExec
	query := `
		INSERT INTO factor_models (model_id, slug, name, description)
		VALUES (:model_id, :slug, :name, :description)
	`
	_, err := s.db.NamedExecContext(ctx, query, model)
	return err
}

func (s *Service) createFactorRecord(ctx context.Context, factor *FactorDefinition) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for NamedExec
	query := `
		INSERT INTO factor_definitions (factor_id, model_id, slug, name, description)
		VALUES (:factor_id, :model_id, :slug, :name, :description)
	`
	_, err := s.db.NamedExecContext(ctx, query, factor)
	return err
}

func (s *Service) ingestReturnsRecords(ctx context.Context, returns []FactorReturn) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for bulk INSERT with ON CONFLICT upsert
	query := `
		INSERT INTO factor_returns (factor_id, date, return_value)
		VALUES (:factor_id, :date, :return_value)
		ON CONFLICT (factor_id, date) DO UPDATE SET return_value = EXCLUDED.return_value
	`
	_, err := s.db.NamedExecContext(ctx, query, returns)
	return err
}

func (s *Service) getModelBySlugRecord(ctx context.Context, slug string) (*FactorModel, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT by slug
	var model FactorModel
	query := `SELECT * FROM factor_models WHERE slug = $1`
	err := s.db.GetContext(ctx, &model, query, slug)
	if err != nil {
		return nil, err
	}
	return &model, nil
}

func (s *Service) getFactorsByModelIDRecords(ctx context.Context, modelID uuid.UUID) ([]FactorDefinition, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for SELECT by model_id
	var factors []FactorDefinition
	query := `SELECT * FROM factor_definitions WHERE model_id = $1`
	err := s.db.SelectContext(ctx, &factors, query, modelID)
	return factors, err
}

func (s *Service) getFactorReturnsRecords(ctx context.Context, factorID uuid.UUID, startDate, endDate time.Time) ([]FactorReturn, error) {
	// TODO: Use HasuraClient for SELECT when available
	// For now, use SQL fallback for date range query with ORDER BY
	var returns []FactorReturn
	query := `
		SELECT * FROM factor_returns 
		WHERE factor_id = $1 AND date >= $2 AND date <= $3
		ORDER BY date ASC
	`
	err := s.db.SelectContext(ctx, &returns, query, factorID, startDate, endDate)
	return returns, err
}
