package hierarchy

import "context"

// Service defines the methods implemented by hierarchy services.
type Service interface {
	ValidateHierarchy(ctx context.Context, tenantID, parentModelType, childModelType string) (*HierarchyValidationResult, error)
	GetHierarchyRules(ctx context.Context, tenantID string) ([]HierarchyRule, error)
	GetHierarchySummary(ctx context.Context, tenantID string) ([]HierarchySummary, error)
	GetEntityHierarchy(ctx context.Context, rootID string, maxDepth int) (*EntityHierarchyNode, error)
	GetHierarchyStats(ctx context.Context, tenantID string) (*HierarchyStats, error)
	CreateHierarchyRule(ctx context.Context, rule *HierarchyRule) error
	UpdateHierarchyRule(ctx context.Context, rule *HierarchyRule) error
	DeleteHierarchyRule(ctx context.Context, tenantID, parentType, childType string) error
	BulkCreateOperations(ctx context.Context, tenantID string, req *HierarchyBulkRequest) (*HierarchyBulkResponse, error)
	LogHierarchyAudit(ctx context.Context, log *HierarchyAuditLog) error
	GetHierarchyAuditLog(ctx context.Context, entityID string, limit int) ([]HierarchyAuditLog, error)
	ImportHierarchyRules(ctx context.Context, tenantID string, req *HierarchyImportRequest) (*HierarchyImportResponse, error)
	ValidateEntityConsistency(ctx context.Context, tenantID string) error
}
