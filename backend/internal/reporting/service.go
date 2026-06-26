package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Service handles business logic for reporting
type Service struct {
	repo       *Repository
	cubeClient *CubeClient
	merger     *DefinitionMerger
	renderer   *Renderer
}

// NewService creates a new reporting service
func NewService(repo *Repository, cubeClient *CubeClient, renderer *Renderer) *Service {
	return &Service{
		repo:       repo,
		cubeClient: cubeClient,
		merger:     NewDefinitionMerger(),
		renderer:   renderer,
	}
}

// ============================================================================
// REPORT DEFINITIONS
// ============================================================================

// CreateDefinition creates a new report definition
func (s *Service) CreateDefinition(ctx context.Context, tenantID, datasourceID uuid.UUID, req CreateReportDefinitionRequest, userID *uuid.UUID) (*ReportDefinition, error) {
	// Check for duplicate key
	existing, err := s.repo.GetDefinitionByKey(ctx, tenantID, datasourceID, req.ReportKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing definition: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("report with key '%s' already exists", req.ReportKey)
	}

	def := &ReportDefinition{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceID: datasourceID,
		ReportKey:          req.ReportKey,
		DisplayName:        req.DisplayName,
		Description:        req.Description,
		Category:           req.Category,
		Tags:               req.Tags,
		ReportType:         req.ReportType,
		OutputFormats:      []string{"pdf", "html", "excel"},
		Definition:         req.Definition,
		Version:            1,
		IsCurrent:          true,
		IsCore:             req.IsCore,
		Status:             "draft",
		CreatedBy:          userID,
	}

	// Extract parameters from definition
	if req.Definition != nil && len(req.Definition.Parameters) > 0 {
		def.ParametersSchema = req.Definition.Parameters
	}

	if err := s.repo.CreateDefinition(ctx, def); err != nil {
		return nil, fmt.Errorf("failed to create definition: %w", err)
	}

	return def, nil
}

// GetDefinition retrieves a report definition
func (s *Service) GetDefinition(ctx context.Context, id uuid.UUID) (*ReportDefinition, error) {
	return s.repo.GetDefinition(ctx, id)
}

// GetDefinitionByKey retrieves a report definition by key
func (s *Service) GetDefinitionByKey(ctx context.Context, tenantID, datasourceID uuid.UUID, key string) (*ReportDefinition, error) {
	return s.repo.GetDefinitionByKey(ctx, tenantID, datasourceID, key)
}

// ListDefinitions lists report definitions
func (s *Service) ListDefinitions(ctx context.Context, tenantID, datasourceID uuid.UUID, filters map[string]interface{}) ([]ReportDefinition, error) {
	return s.repo.ListDefinitions(ctx, tenantID, datasourceID, filters)
}

// UpdateDefinition updates a report definition
func (s *Service) UpdateDefinition(ctx context.Context, def *ReportDefinition, userID *uuid.UUID) error {
	def.UpdatedBy = userID
	return s.repo.UpdateDefinition(ctx, def)
}

// PublishDefinition publishes a report definition
func (s *Service) PublishDefinition(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	return s.repo.PublishDefinition(ctx, id, userID)
}

// DeleteDefinition deletes a report definition
func (s *Service) DeleteDefinition(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteDefinition(ctx, id)
}

// ============================================================================
// REPORT EXTENSIONS
// ============================================================================

// CreateExtension creates a report extension
func (s *Service) CreateExtension(ctx context.Context, tenantID, datasourceID uuid.UUID, req CreateReportExtensionRequest, userID *uuid.UUID) (*ReportExtension, error) {
	// Verify base report exists
	baseReport, err := s.repo.GetDefinition(ctx, req.BaseReportID)
	if err != nil {
		return nil, fmt.Errorf("failed to get base report: %w", err)
	}
	if baseReport == nil {
		return nil, fmt.Errorf("base report not found")
	}

	ext := &ReportExtension{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceID: datasourceID,
		BaseReportID:       req.BaseReportID,
		ExtensionKey:       req.ExtensionKey,
		ExtensionName:      req.ExtensionName,
		Description:        req.Description,
		Overrides:          req.Overrides,
		Additions:          req.Additions,
		Removals:           req.Removals,
		ParameterDefaults:  req.ParameterDefaults,
		Version:            1,
		IsCurrent:          true,
		CoreVersionTarget:  &baseReport.Version,
		Status:             "draft",
		CreatedBy:          userID,
	}

	if err := s.repo.CreateExtension(ctx, ext); err != nil {
		return nil, fmt.Errorf("failed to create extension: %w", err)
	}

	return ext, nil
}

// GetExtension retrieves a report extension
func (s *Service) GetExtension(ctx context.Context, id uuid.UUID) (*ReportExtension, error) {
	return s.repo.GetExtension(ctx, id)
}

// ListExtensions lists extensions for a base report
func (s *Service) ListExtensions(ctx context.Context, tenantID, datasourceID, baseReportID uuid.UUID) ([]ReportExtension, error) {
	return s.repo.ListExtensions(ctx, tenantID, datasourceID, baseReportID)
}

// ListAllExtensions lists all extensions for a tenant
func (s *Service) ListAllExtensions(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]ReportExtension, error) {
	return s.repo.ListAllExtensions(ctx, tenantID, datasourceID)
}

// ============================================================================
// REPORT RENDERING
// ============================================================================

// RenderReport generates a report (synchronous)
func (s *Service) RenderReport(ctx context.Context, tenantID, datasourceID uuid.UUID, req RenderReportRequest, userID *uuid.UUID) (*ReportInstance, error) {
	// Get definition
	def, err := s.repo.GetDefinition(ctx, req.ReportDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get definition: %w", err)
	}
	if def == nil {
		return nil, fmt.Errorf("report definition not found")
	}

	// Get extension if specified
	var ext *ReportExtension
	if req.ReportExtensionID != nil {
		ext, err = s.repo.GetExtension(ctx, *req.ReportExtensionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get extension: %w", err)
		}
	}

	// Merge definition with extension
	mergedDef, err := s.merger.Merge(def.Definition, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to merge definition: %w", err)
	}

	mergedJSON, _ := json.Marshal(mergedDef)

	// Create instance record
	inst := &ReportInstance{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceID: datasourceID,
		ReportDefinitionID: req.ReportDefinitionID,
		ReportExtensionID:  req.ReportExtensionID,
		MergedDefinition:   mergedJSON,
		ContextType:        req.ContextType,
		ContextID:          req.ContextID,
		ContextName:        req.ContextName,
		Parameters:         req.Parameters,
		OutputFormat:       req.OutputFormat,
		Status:             "generating",
		RequestedBy:        userID,
		RequestedAt:        time.Now(),
	}

	if err := s.repo.CreateInstance(ctx, inst); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// Render the report
	startTime := time.Now()
	result, err := s.renderer.Render(ctx, tenantID, datasourceID, mergedDef, req.Parameters, req.OutputFormat)
	if err != nil {
		s.repo.UpdateInstanceStatus(ctx, inst.ID, "failed", err.Error())
		return nil, fmt.Errorf("failed to render report: %w", err)
	}

	// Update instance with result
	generationTimeMs := int(time.Since(startTime).Milliseconds())
	if err := s.repo.UpdateInstanceComplete(ctx, inst.ID, result.URL, result.Metadata, generationTimeMs); err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	// Reload to get updated fields
	return s.repo.GetInstance(ctx, inst.ID)
}

// RenderReportAsync queues a report for async generation
func (s *Service) RenderReportAsync(ctx context.Context, tenantID, datasourceID uuid.UUID, req RenderReportRequest, userID *uuid.UUID) (*ReportInstance, error) {
	// Get definition
	def, err := s.repo.GetDefinition(ctx, req.ReportDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get definition: %w", err)
	}
	if def == nil {
		return nil, fmt.Errorf("report definition not found")
	}

	// Get extension if specified
	var ext *ReportExtension
	if req.ReportExtensionID != nil {
		ext, err = s.repo.GetExtension(ctx, *req.ReportExtensionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get extension: %w", err)
		}
	}

	// Merge definition with extension
	mergedDef, err := s.merger.Merge(def.Definition, ext)
	if err != nil {
		return nil, fmt.Errorf("failed to merge definition: %w", err)
	}

	mergedJSON, _ := json.Marshal(mergedDef)

	// Create instance record
	inst := &ReportInstance{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		TenantDatasourceID: datasourceID,
		ReportDefinitionID: req.ReportDefinitionID,
		ReportExtensionID:  req.ReportExtensionID,
		MergedDefinition:   mergedJSON,
		ContextType:        req.ContextType,
		ContextID:          req.ContextID,
		ContextName:        req.ContextName,
		Parameters:         req.Parameters,
		OutputFormat:       req.OutputFormat,
		Status:             "pending",
		RequestedBy:        userID,
		RequestedAt:        time.Now(),
	}

	if err := s.repo.CreateInstance(ctx, inst); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	// TODO: Queue for async processing via Temporal/RabbitMQ

	return inst, nil
}

// GetInstance retrieves a report instance
func (s *Service) GetInstance(ctx context.Context, id uuid.UUID) (*ReportInstance, error) {
	return s.repo.GetInstance(ctx, id)
}

// ListInstances lists report instances
func (s *Service) ListInstances(ctx context.Context, tenantID, datasourceID uuid.UUID, limit int) ([]ReportInstance, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListInstances(ctx, tenantID, datasourceID, limit)
}

// ============================================================================
// SCHEDULING
// ============================================================================

// CreateSchedule creates a report schedule
func (s *Service) CreateSchedule(ctx context.Context, sched *ReportSchedule) error {
	// Calculate next run time
	nextRun, err := calculateNextRun(sched.CronExpression, sched.Timezone)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	sched.NextRunAt = nextRun

	return s.repo.CreateSchedule(ctx, sched)
}

// GetSchedule retrieves a schedule
func (s *Service) GetSchedule(ctx context.Context, id uuid.UUID) (*ReportSchedule, error) {
	return s.repo.GetSchedule(ctx, id)
}

// ListSchedules lists schedules
func (s *Service) ListSchedules(ctx context.Context, tenantID, datasourceID uuid.UUID) ([]ReportSchedule, error) {
	return s.repo.ListSchedules(ctx, tenantID, datasourceID)
}

// ProcessDueSchedules processes schedules that are due
func (s *Service) ProcessDueSchedules(ctx context.Context) error {
	schedules, err := s.repo.GetDueSchedules(ctx)
	if err != nil {
		return fmt.Errorf("failed to get due schedules: %w", err)
	}

	for _, sched := range schedules {
		// Queue report generation
		req := RenderReportRequest{
			ReportDefinitionID: sched.ReportDefinitionID,
			ReportExtensionID:  sched.ReportExtensionID,
			OutputFormat:       "pdf", // Default, should come from OutputFormats
			ContextType:        sched.ContextType,
			ContextID:          sched.FixedContextID,
			Parameters:         sched.ParametersTemplate,
		}

		_, err := s.RenderReportAsync(ctx, sched.TenantID, sched.TenantDatasourceID, req, nil)

		// Calculate next run
		nextRun, _ := calculateNextRun(sched.CronExpression, sched.Timezone)

		// Update schedule
		status := "success"
		errMsg := ""
		if err != nil {
			status = "failed"
			errMsg = err.Error()
		}

		s.repo.UpdateScheduleRun(ctx, sched.ID, status, errMsg, nextRun)
	}

	return nil
}

// ============================================================================
// TENANT PROVISIONING
// ============================================================================

// ProvisionReports provisions reports for a new tenant
func (s *Service) ProvisionReports(ctx context.Context, req ProvisionReportsRequest) (*ProvisionReportsResponse, error) {
	// Get package
	pkg, err := s.repo.GetPackage(ctx, req.PackageKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get package: %w", err)
	}
	if pkg == nil {
		return nil, fmt.Errorf("package '%s' not found", req.PackageKey)
	}

	resp := &ProvisionReportsResponse{
		Success:          true,
		CreatedReportIDs: make([]uuid.UUID, 0),
	}

	// Parse report definitions from package
	var reportDefs []struct {
		Key         string `json:"key"`
		DisplayName string `json:"displayName"`
	}
	if err := json.Unmarshal(pkg.ReportDefinitions, &reportDefs); err != nil {
		return nil, fmt.Errorf("failed to parse package definitions: %w", err)
	}

	// Create each report definition
	for _, defInfo := range reportDefs {
		// Create a basic definition (in production, would load full template from package)
		createReq := CreateReportDefinitionRequest{
			ReportKey:   defInfo.Key,
			DisplayName: defInfo.DisplayName,
			Description: fmt.Sprintf("Auto-provisioned from %s package", req.PackageKey),
			Category:    pkg.Category,
			ReportType:  "paginated",
			Definition:  createDefaultDefinition(defInfo.Key, defInfo.DisplayName),
			IsCore:      false,
		}

		def, err := s.CreateDefinition(ctx, req.TenantID, req.TenantDatasourceID, createReq, nil)
		if err != nil {
			resp.Errors = append(resp.Errors, fmt.Sprintf("failed to create %s: %v", defInfo.Key, err))
			continue
		}

		resp.CreatedReportIDs = append(resp.CreatedReportIDs, def.ID)
		resp.ReportsCreated++
	}

	if len(resp.Errors) > 0 {
		resp.Success = false
	}

	return resp, nil
}

// ListPackages lists available provisioning packages
func (s *Service) ListPackages(ctx context.Context) ([]ReportPackage, error) {
	return s.repo.ListPackages(ctx)
}

// ============================================================================
// HELPERS
// ============================================================================

func calculateNextRun(cronExpr, timezone string) (*time.Time, error) {
	// Simplified - in production use github.com/robfig/cron
	// For now, just add 1 hour
	t := time.Now().Add(time.Hour)
	return &t, nil
}

func createDefaultDefinition(key, displayName string) *ReportLayout {
	return &ReportLayout{
		Metadata: ReportMetadata{
			Key:         key,
			DisplayName: displayName,
		},
		Parameters: []Parameter{},
		DataBindings: map[string]DataBinding{
			"primary": {
				Cube:       "default",
				Measures:   []string{},
				Dimensions: []string{},
			},
		},
		Layout: Layout{
			PageSettings: PageSettings{
				Size:        "letter",
				Orientation: "portrait",
				Margins:     Margins{Top: 72, Right: 72, Bottom: 72, Left: 72},
			},
			Body: Body{
				Sections: []ReportSection{
					{
						ID:    "main",
						Type:  "summary",
						Title: displayName,
					},
				},
			},
		},
	}
}

// RenderReportRequest helper type for the service
type RenderReportRequestHelper struct {
	FixedContextID *uuid.UUID
}

// Add FixedContextID to the original RenderReportRequest via extension
func (r *RenderReportRequest) SetFixedContextID(id *uuid.UUID) {
	// This is handled in the schedule processing
}
