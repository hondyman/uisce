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

func (s *ReportService) ListTemplates(ctx context.Context) ([]ReportTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *ReportService) UpdateTemplate(ctx context.Context, template *ReportTemplate) error {
	return s.repo.UpdateTemplate(ctx, template)
}

func (s *ReportService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTemplate(ctx, id)
}
