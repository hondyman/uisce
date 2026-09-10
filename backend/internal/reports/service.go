package reports

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// ReportService manages report templates.
type ReportService struct {
	repo *Repository
}

// NewReportService creates a new ReportService.
func NewReportService(db *sql.DB) *ReportService {
	return &ReportService{
		repo: NewRepository(db),
	}
}

func (s *ReportService) CreateTemplate(ctx context.Context, template *ReportTemplate) error {
	return s.repo.CreateTemplate(ctx, template)
}

func (s *ReportService) GetTemplate(ctx context.Context, id uuid.UUID) (*ReportTemplate, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *ReportService) ResolveGoldCopyTenantID(ctx context.Context) (uuid.UUID, error) {
	return s.repo.ResolveGoldCopyTenantID(ctx)
}


func (s *ReportService) ListTemplatesScoped(ctx context.Context, tenantID uuid.UUID, callerUserID string) ([]ReportTemplate, error) {
	return s.repo.ListTemplatesScoped(ctx, tenantID, callerUserID)
}


func (s *ReportService) SetFavorite(ctx context.Context, tenantID uuid.UUID, userID string, templateID uuid.UUID) error {
	return s.repo.SetFavorite(ctx, tenantID, userID, templateID)
}

func (s *ReportService) RemoveFavorite(ctx context.Context, tenantID uuid.UUID, userID string, templateID uuid.UUID) error {
	return s.repo.RemoveFavorite(ctx, tenantID, userID, templateID)
}

func (s *ReportService) UpdateTemplate(ctx context.Context, template *ReportTemplate) error {
	return s.repo.UpdateTemplate(ctx, template)
}

func (s *ReportService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTemplate(ctx, id)
}

