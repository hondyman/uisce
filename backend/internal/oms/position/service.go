package position

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(db *sql.DB) *Service {
	return &Service{
		repo: NewRepository(db),
	}
}

func (s *Service) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]PositionRecord, error) {
	return s.repo.List(ctx, tenantID, subtypeCode)
}

func (s *Service) Get(ctx context.Context, tenantID, id uuid.UUID) (*PositionRecord, error) {
	return s.repo.Get(ctx, tenantID, id)
}

func (s *Service) Create(ctx context.Context, tenantID uuid.UUID, rec *PositionRecord) error {
	if err := rec.Validate(); err != nil {
		return err
	}
	rec.TenantID = tenantID
	return s.repo.Create(ctx, rec)
}

func (s *Service) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.SoftDelete(ctx, tenantID, id)
}
