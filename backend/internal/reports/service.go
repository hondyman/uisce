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

func (s *ReportService) Repo() *Repository {
	return s.repo
}

func (s *ReportService) DeleteTemplate(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteTemplate(ctx, id)
}

// Folder operations delegating to folder repository

func (s *ReportService) CreateFolder(ctx context.Context, folder *ReportFolder) error {
	return s.repo.CreateFolder(ctx, folder)
}

func (s *ReportService) RenameFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, newName string) error {
	return s.repo.RenameFolder(ctx, tenantID, userID, folderID, newName)
}

func (s *ReportService) MoveFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, newParentID *uuid.UUID) error {
	return s.repo.MoveFolder(ctx, tenantID, userID, folderID, newParentID)
}

func (s *ReportService) DeleteFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID) error {
	return s.repo.DeleteFolder(ctx, tenantID, userID, folderID)
}

func (s *ReportService) ListFolders(ctx context.Context, tenantID uuid.UUID, userID string) ([]ReportFolder, error) {
	return s.repo.ListFolders(ctx, tenantID, userID)
}

func (s *ReportService) AddReportToFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, templateID uuid.UUID) error {
	return s.repo.AddReportToFolder(ctx, tenantID, userID, folderID, templateID)
}

func (s *ReportService) RemoveReportFromFolder(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID, templateID uuid.UUID) error {
	return s.repo.RemoveReportFromFolder(ctx, tenantID, userID, folderID, templateID)
}

func (s *ReportService) ListFolderReportIDs(ctx context.Context, tenantID uuid.UUID, userID string, folderID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.ListFolderReportIDs(ctx, tenantID, userID, folderID)
}

// Schedule operations delegating to schedule repository

func (s *ReportService) CreateSchedule(ctx context.Context, tenantID uuid.UUID, callerUserID string, input CreateScheduleInput) (*ReportSchedule, error) {
	return s.repo.CreateSchedule(ctx, tenantID, callerUserID, input)
}

func (s *ReportService) ListSchedulesForTemplate(ctx context.Context, tenantID uuid.UUID, callerUserID string, templateID uuid.UUID) ([]ReportSchedule, error) {
	return s.repo.ListSchedulesForTemplate(ctx, tenantID, callerUserID, templateID)
}

func (s *ReportService) GetSchedule(ctx context.Context, tenantID uuid.UUID, scheduleID uuid.UUID) (*ReportSchedule, error) {
	return s.repo.GetSchedule(ctx, tenantID, scheduleID)
}

func (s *ReportService) DeleteSchedule(ctx context.Context, tenantID uuid.UUID, callerUserID string, isAdmin bool, scheduleID uuid.UUID) error {
	return s.repo.DeleteSchedule(ctx, tenantID, callerUserID, isAdmin, scheduleID)
}

func (s *ReportService) TriggerScheduleRun(ctx context.Context, tenantID uuid.UUID, callerUserID string, isAdmin bool, scheduleID uuid.UUID, executor ReportExecutor) (*ScheduleExecutionResult, error) {
	return s.repo.TriggerScheduleRun(ctx, tenantID, callerUserID, isAdmin, scheduleID, executor)
}


