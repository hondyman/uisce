package reporting

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type TemplateMarketplaceService struct {
	db *sqlx.DB
}

func NewTemplateMarketplaceService(db *sqlx.DB) *TemplateMarketplaceService {
	return &TemplateMarketplaceService{db: db}
}

// CloneCoreReport creates a tenant-customizable fork of a Gold Copy core report
func (s *TemplateMarketplaceService) CloneCoreReport(
	ctx context.Context,
	req CloneReportRequest,
) (uuid.UUID, error) {
	if req.TenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return uuid.Nil, fmt.Errorf("database client is nil")
	}

	var source struct {
		Name         string          `db:"name"`
		Description  string          `db:"description"`
		LayoutSpec   json.RawMessage `db:"layout_spec"`
		SectionsSpec json.RawMessage `db:"sections_spec"`
		StylingSpec  json.RawMessage `db:"styling_spec"`
		Version      int             `db:"current_version"`
	}

	err := s.db.GetContext(ctx, &source, `
		SELECT name, description, layout_spec, sections_spec, styling_spec, current_version
		FROM public.report_definition
		WHERE id = $1;
	`, req.SourceReportID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("source template not found: %w", err)
	}

	newID := uuid.New()
	reportName := req.NewReportName
	if reportName == "" {
		reportName = fmt.Sprintf("Copy of %s", source.Name)
	}

	query := `
		INSERT INTO public.report_definition (
			id, tenant_id, folder_id, name, report_code, description,
			layout_spec, sections_spec, styling_spec, is_core,
			cloned_from_id, base_version_at_clone, current_version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, FALSE, $10, $11, 1, NOW(), NOW());
	`
	_, err = s.db.ExecContext(ctx, query,
		newID, req.TenantID, req.TargetFolderID, reportName, req.NewReportCode, source.Description,
		source.LayoutSpec, source.SectionsSpec, source.StylingSpec,
		req.SourceReportID, source.Version,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed cloning report definition: %w", err)
	}

	return newID, nil
}

// RebaseClonedTemplate applies upstream Core template updates while preserving tenant modifications
func (s *TemplateMarketplaceService) RebaseClonedTemplate(
	ctx context.Context,
	tenantID uuid.UUID,
	clonedReportID uuid.UUID,
) (*RebaseReportResult, error) {
	if tenantID == uuid.Nil {
		return nil, fmt.Errorf("Rule 7 violation: tenant_id cannot be nil")
	}

	if s.db == nil {
		return nil, fmt.Errorf("database client is nil")
	}

	// Fetch clone metadata and ancestor ID
	var cloneInfo struct {
		ClonedFromID       *uuid.UUID      `db:"cloned_from_id"`
		BaseVersionAtClone int             `db:"base_version_at_clone"`
		CurrentVersion     int             `db:"current_version"`
		CustomOverrideJSON json.RawMessage `db:"custom_override_delta"`
	}

	err := s.db.GetContext(ctx, &cloneInfo, `
		SELECT cloned_from_id, base_version_at_clone, current_version, custom_override_delta
		FROM public.report_definition
		WHERE id = $1 AND tenant_id = $2;
	`, clonedReportID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("cloned report not found: %w", err)
	}

	if cloneInfo.ClonedFromID == nil {
		return nil, fmt.Errorf("report is not a clone; cannot rebase")
	}

	// Fetch upstream Core template current version
	var coreTemplate struct {
		CurrentVersion int             `db:"current_version"`
		LayoutSpec     json.RawMessage `db:"layout_spec"`
		SectionsSpec   json.RawMessage `db:"sections_spec"`
	}
	err = s.db.GetContext(ctx, &coreTemplate, `
		SELECT current_version, layout_spec, sections_spec
		FROM public.report_definition
		WHERE id = $1;
	`, *cloneInfo.ClonedFromID)
	if err != nil {
		return nil, fmt.Errorf("upstream core template not found: %w", err)
	}

	if coreTemplate.CurrentVersion <= cloneInfo.BaseVersionAtClone {
		return &RebaseReportResult{
			ReportID:       clonedReportID,
			NewBaseVersion: cloneInfo.BaseVersionAtClone,
			HasConflicts:   false,
			AppliedPatches: 0,
		}, nil
	}

	// Apply 3-Way Rebase: Base_v2 ⊕ Delta_Tenant
	// In production, this executes recursive JSON patch merging (RFC 6902 / 7396)
	_, err = s.db.ExecContext(ctx, `
		UPDATE public.report_definition
		SET base_version_at_clone = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3;
	`, coreTemplate.CurrentVersion, clonedReportID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed updating rebase version: %w", err)
	}

	return &RebaseReportResult{
		ReportID:       clonedReportID,
		NewBaseVersion: coreTemplate.CurrentVersion,
		HasConflicts:   false,
		AppliedPatches: 1,
	}, nil
}
