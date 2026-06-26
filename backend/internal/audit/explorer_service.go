package audit

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hondyman/semlayer/backend/internal/auth"
)

// Service provides audit exploration and analysis operations
type ExplorerService struct {
	repo Repository
	ai   AIClient // Interface for AI explanations
}

// NewExplorerService creates a new audit explorer service
func NewExplorerService(repo Repository, ai AIClient) *ExplorerService {
	return &ExplorerService{
		repo: repo,
		ai:   ai,
	}
}

// ListEvents retrieves audit events with role-based tenant scoping
func (s *ExplorerService) ListEvents(ctx context.Context, req *ListEventsRequest) (*ListEventsResponse, error) {
	// Extract allowed tenants from context
	scope := auth.AllowedTenantsFromContext(ctx)

	// Enforce tenant scope intersection - filter to accessible tenants only
	for i, tenant := range req.TenantFilter {
		found := false
		for _, allowedTenant := range scope {
			if tenant == allowedTenant {
				found = true
				break
			}
		}
		if !found {
			req.TenantFilter = append(req.TenantFilter[:i], req.TenantFilter[i+1:]...)
		}
	}

	if len(req.TenantFilter) == 0 {
		return nil, fmt.Errorf("no accessible tenants for this request")
	}

	// Build query filters
	filters := QueryFilters{
		TimeRange: TimeRange{
			From: req.TimeRange.From,
			To:   req.TimeRange.To,
		},
		ArtifactTypes: req.ArtifactTypes,
		Statuses:      req.Statuses,
		RiskLevels:    req.RiskLevels,
		Actors:        req.Actors,
		Limit:         req.Limit,
		Offset:        req.Offset,
	}

	if filters.Limit == 0 {
		filters.Limit = 50
	}

	// Query repository
	events, total, err := s.repo.ListEvents(ctx, TenantScope(req.TenantFilter), filters)
	if err != nil {
		return nil, err
	}

	return &ListEventsResponse{
		Events:  events,
		Total:   total,
		Limit:   filters.Limit,
		Offset:  filters.Offset,
		HasMore: filters.Offset+filters.Limit < total,
	}, nil
}

// GetEntityAudit retrieves all audit events for a specific entity
func (s *ExplorerService) GetEntityAudit(ctx context.Context, entityType, entityID string, from, to time.Time) (*EntityAudit, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	return s.repo.GetEntityAudit(ctx, scope, entityType, entityID, from, to, 1000, 0)
}

// ListIncidents retrieves incident clusters
func (s *ExplorerService) ListIncidents(ctx context.Context, from, to time.Time, limit, offset int) ([]IncidentCluster, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	if limit == 0 {
		limit = 50
	}

	return s.repo.ListIncidents(ctx, scope, from, to, limit, offset)
}

// GetIncident retrieves a single incident with full details
func (s *ExplorerService) GetIncident(ctx context.Context, incidentID string) (*IncidentCluster, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	return s.repo.GetIncident(ctx, scope, incidentID)
}

// ListComplianceEvents retrieves compliance-related audit events
func (s *ExplorerService) ListComplianceEvents(ctx context.Context, from, to time.Time, violationTypes []string, limit, offset int) ([]ComplianceEvent, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	if limit == 0 {
		limit = 50
	}

	return s.repo.ListComplianceEvents(ctx, scope, from, to, violationTypes, limit, offset)
}

// ExplainAuditEvent generates AI explanation for audit events
func (s *ExplorerService) ExplainAuditEvent(ctx context.Context, req *ExplainRequest) (*ExplainResponse, error) {
	// Extract allowed tenants from context
	scope := auth.AllowedTenantsFromContext(ctx)

	// Enforce tenant scope intersection - filter to accessible tenants only
	for i, tenant := range req.TenantScope {
		found := false
		for _, allowedTenant := range scope {
			if tenant == allowedTenant {
				found = true
				break
			}
		}
		if !found {
			req.TenantScope = append(req.TenantScope[:i], req.TenantScope[i+1:]...)
		}
	}

	if len(req.TenantScope) == 0 {
		return nil, fmt.Errorf("no accessible tenants for this explanation request")
	}

	// Build AI prompt with tenant scope enforcement
	prompt := buildExplainPrompt(req)

	// Call AI service
	response, err := s.ai.GenerateExplanation(ctx, prompt, req.TenantScope)
	if err != nil {
		log.Printf("AI explanation failed: %v", err)
		return nil, err
	}

	return response, nil
}

// GetGlobalAdminDashboard returns platform-wide metrics (Global Admin only)
func (s *ExplorerService) GetGlobalAdminDashboard(ctx context.Context, from, to time.Time) (*GlobalAdminDashboard, error) {
	scope := auth.AllowedTenantsFromContext(ctx)
	// Check if user has global scope (empty means all, or has special "global" tenant)
	isGlobal := len(scope) == 0 || (len(scope) == 1 && scope[0] == "*")
	if !isGlobal {
		return nil, fmt.Errorf("insufficient permissions for global dashboard")
	}

	return s.repo.GetGlobalAdminDashboard(ctx, from, to)
}

// GetGlobalOpsDashboard returns multi-tenant ops metrics (Global Ops only)
func (s *ExplorerService) GetGlobalOpsDashboard(ctx context.Context, from, to time.Time) (*GlobalOpsDashboard, error) {
	scope := auth.AllowedTenantsFromContext(ctx)
	// Global Ops should have specific assigned tenants, not full global
	// This is enforced at auth middleware level
	if len(scope) == 0 {
		return nil, fmt.Errorf("insufficient scope for ops dashboard")
	}

	return s.repo.GetGlobalOpsDashboard(ctx, scope, from, to)
}

// GetTenantAdminDashboard returns tenant-specific metrics (Tenant Admin only)
func (s *ExplorerService) GetTenantAdminDashboard(ctx context.Context, tenantID string) (*TenantAdminDashboard, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	// Tenant Admin can only access their own tenant
	found := false
	for _, allowedTenant := range scope {
		if allowedTenant == tenantID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("access denied for tenant %s", tenantID)
	}

	from := time.Now().AddDate(0, 0, -30) // Last 30 days
	to := time.Now()

	return s.repo.GetTenantAdminDashboard(ctx, tenantID, from, to)
}

// GetTenantOpsDashboard returns tenant ops metrics (Tenant Ops only)
func (s *ExplorerService) GetTenantOpsDashboard(ctx context.Context, tenantID string) (*TenantOpsDashboard, error) {
	scope := auth.AllowedTenantsFromContext(ctx)

	// Tenant Ops can only access their own tenant
	found := false
	for _, allowedTenant := range scope {
		if allowedTenant == tenantID {
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("access denied for tenant %s", tenantID)
	}

	from := time.Now().AddDate(0, 0, -1) // Last 24 hours
	to := time.Now()

	return s.repo.GetTenantOpsDashboard(ctx, tenantID, from, to)
}

// Helper function to build AI explanation prompt
func buildExplainPrompt(req *ExplainRequest) string {
	prompt := fmt.Sprintf(`
You are an audit explanation assistant for a multi-tenant data platform.
You have access to the following audit events and should provide a clear, actionable explanation.

TENANT SCOPE (you can only reference these tenants):
%v

AUDIT RECORDS:
`, req.TenantScope)

	for _, record := range req.AuditRecords {
		prompt += fmt.Sprintf(`
- Type: %s
  ID: %s
  Tenant: %s
  Timestamp: %s
  Status: %s
  Title: %s
`, record.Type, record.ID, record.TenantID, record.Timestamp, record.Status, record.Title)
	}

	if len(req.AuditRecords) > 0 {
		prompt += fmt.Sprintf(`
SEMANTIC CONTEXT:
%v

COMPLIANCE CONTEXT:
%v

Please provide:
1. A clear narrative explaining what happened
2. The root cause
3. The blast radius (which jobs, DAGs, semantic terms are affected)
4. A recommended fix or remediation step
5. A summary of a suggested ChangeSet (if applicable)

Format your response as JSON with keys: narrative, rootCause, blastRadius, recommendedFix, suggestedChangeSetSummary
`, req.SemanticContext, req.ComplianceContext)
	}

	return prompt
}

// AIClient interface for AI explanation generation
type AIClient interface {
	GenerateExplanation(ctx context.Context, prompt string, tenantScope []string) (*ExplainResponse, error)
}
