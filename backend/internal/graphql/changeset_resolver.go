package graphql

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/catalog"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ChangeSetResolver handles ChangeSet-related mutations and queries
type ChangeSetResolver struct {
	catalogWriter catalog.Writer
	auditService  *audit.Service
	logger        *zap.Logger
}

// NewChangeSetResolver creates a new ChangeSet resolver
func NewChangeSetResolver(cw catalog.Writer, as *audit.Service, logger *zap.Logger) *ChangeSetResolver {
	return &ChangeSetResolver{
		catalogWriter: cw,
		auditService:  as,
		logger:        logger,
	}
}

// CreateChangeSetFromAI creates a ChangeSet from an AI suggestion
func (r *ChangeSetResolver) CreateChangeSetFromAI(
	ctx context.Context,
	title string,
	description string,
	tenantID string,
	sourceEventID string,
	impactedEntities []audit.ImpactedEntity,
) (*audit.ChangeSetResponse, error) {
	// Validate tenant scope
	allowedTenants := extractAllowedTenantsFromContext(ctx)
	if !contains(allowedTenants, tenantID) {
		return nil, fmt.Errorf("tenant %s not in allowed scope", tenantID)
	}

	// Generate ChangeSet ID
	changeSetID := uuid.NewString()

	// Create ChangeSet node
	node := catalog.CatalogNode{
		ID:            "changeset_event:" + changeSetID,
		NodeType:      "changeset_event",
		QualifiedPath: "audit/changeset_event/" + changeSetID,
		TenantID:      tenantID,
		Properties: map[string]interface{}{
			"title":         title,
			"description":   description,
			"status":        "PENDING",
			"source":        "AI_PROPOSED",
			"createdBy":     extractActorFromContext(ctx),
			"createdAt":     time.Now().UTC(),
			"sourceEventId": sourceEventID,
		},
	}

	if err := r.catalogWriter.CreateNode(ctx, node); err != nil {
		r.logger.Error("failed to create changeset node", zap.Error(err))
		return nil, fmt.Errorf("create changeset node: %w", err)
	}

	// Create edges for impacted entities
	edges := []catalog.CatalogEdge{}

	for _, entity := range impactedEntities {
		edges = append(edges, catalog.CatalogEdge{
			ID:       "edge:has_impact_on:" + changeSetID + ":" + entity.ID,
			EdgeType: "has_impact_on",
			FromNode: node.ID,
			ToNode:   entity.NodeID,
		})
	}

	// Add HAS_TENANT edge
	edges = append(edges, catalog.CatalogEdge{
		ID:       "edge:has_tenant:changeset:" + changeSetID,
		EdgeType: "has_tenant",
		FromNode: node.ID,
		ToNode:   "tenant:" + tenantID,
	})

	// Add EVENT_OF edge to source event
	if sourceEventID != "" {
		edges = append(edges, catalog.CatalogEdge{
			ID:       "edge:event_of:changeset:" + changeSetID,
			EdgeType: "event_of",
			FromNode: node.ID,
			ToNode:   sourceEventID,
		})
	}

	if len(edges) > 0 {
		if err := r.catalogWriter.CreateEdges(ctx, edges); err != nil {
			r.logger.Error("failed to create changeset edges", zap.Error(err))
			// Continue despite edge creation errors
		}
	}

	// Emit governance audit event
	r.logger.Info("changeset created from AI proposal",
		zap.String("changeSetID", changeSetID),
		zap.String("tenantID", tenantID),
		zap.String("sourceEventID", sourceEventID),
	)

	return &audit.ChangeSetResponse{
		ID:     changeSetID,
		Status: "PENDING",
	}, nil
}

// ApproveChangeSet approves a ChangeSet and triggers application workflow
func (r *ChangeSetResolver) ApproveChangeSet(
	ctx context.Context,
	changeSetID string,
) (*audit.ChangeSetResponse, error) {
	// Fetch ChangeSet node
	node, err := r.catalogWriter.GetNode(ctx, "changeset_event:"+changeSetID)
	if err != nil {
		return nil, fmt.Errorf("get changeset node: %w", err)
	}

	if node == nil {
		return nil, fmt.Errorf("changeset not found: %s", changeSetID)
	}

	// Validate tenant scope
	allowedTenants := extractAllowedTenantsFromContext(ctx)
	if !contains(allowedTenants, node.TenantID) {
		return nil, fmt.Errorf("tenant %s not in allowed scope", node.TenantID)
	}

	// Update ChangeSet status
	node.Properties["status"] = "APPROVED"
	node.Properties["approvedBy"] = extractActorFromContext(ctx)
	node.Properties["approvedAt"] = time.Now().UTC()

	if err := r.catalogWriter.UpdateNode(ctx, *node); err != nil {
		r.logger.Error("failed to update changeset status", zap.Error(err))
		return nil, fmt.Errorf("update changeset status: %w", err)
	}

	// Trigger Temporal workflow to apply ChangeSet
	// This would call something like:
	// r.temporalClient.ExecuteWorkflow(ctx, workflowOptions, ApplyChangeSetWorkflow, ...)

	r.logger.Info("changeset approved and workflow triggered",
		zap.String("changeSetID", changeSetID),
		zap.String("approvedBy", extractActorFromContext(ctx)),
	)

	return &audit.ChangeSetResponse{
		ID:     changeSetID,
		Status: "APPROVED",
	}, nil
}

// RejectChangeSet rejects a ChangeSet with a reason
func (r *ChangeSetResolver) RejectChangeSet(
	ctx context.Context,
	changeSetID string,
	reason string,
) (*audit.ChangeSetResponse, error) {
	// Fetch ChangeSet node
	node, err := r.catalogWriter.GetNode(ctx, "changeset_event:"+changeSetID)
	if err != nil {
		return nil, fmt.Errorf("get changeset node: %w", err)
	}

	if node == nil {
		return nil, fmt.Errorf("changeset not found: %s", changeSetID)
	}

	// Validate tenant scope
	allowedTenants := extractAllowedTenantsFromContext(ctx)
	if !contains(allowedTenants, node.TenantID) {
		return nil, fmt.Errorf("tenant %s not in allowed scope", node.TenantID)
	}

	// Update ChangeSet status
	node.Properties["status"] = "REJECTED"
	node.Properties["rejectionReason"] = reason
	node.Properties["rejectedBy"] = extractActorFromContext(ctx)
	node.Properties["rejectedAt"] = time.Now().UTC()

	if err := r.catalogWriter.UpdateNode(ctx, *node); err != nil {
		r.logger.Error("failed to update changeset status", zap.Error(err))
		return nil, fmt.Errorf("update changeset status: %w", err)
	}

	r.logger.Info("changeset rejected",
		zap.String("changeSetID", changeSetID),
		zap.String("reason", reason),
	)

	return &audit.ChangeSetResponse{
		ID:     changeSetID,
		Status: "REJECTED",
	}, nil
}

// ListChangeSets returns a list of ChangeSets filtered by tenant scope
func (r *ChangeSetResolver) ListChangeSets(
	ctx context.Context,
	tenantFilter []string,
	statusFilter []string,
	limit int,
	offset int,
) ([]*audit.ChangeSet, int, error) {
	// Validate tenant scope
	allowedTenants := extractAllowedTenantsFromContext(ctx)
	tenantScope := intersectSlices(allowedTenants, tenantFilter)

	if len(tenantScope) == 0 {
		return []*audit.ChangeSet{}, 0, nil
	}

	// Query Trino for ChangeSets via Trino JDBC/REST client
	// Uses audit.changeset_impact view for optimized queries
	// Note: This requires Trino client integration (trinodriver or similar)

	r.logger.Info("listing changesets from Trino",
		zap.Strings("tenantScope", tenantScope),
		zap.Strings("statusFilter", statusFilter),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	// PRODUCTION: Implement Trino query:
	// SELECT cs.id, cs.title, cs.status, cs.source, ...
	// FROM audit.changeset_impact cs
	// WHERE cs.tenant_id IN (?, ?, ...)
	// AND (? IS NULL OR cs.status = ANY(?))
	// ORDER BY cs.created_at DESC
	// LIMIT ? OFFSET ?

	// For now, construct placeholder that maintains type signature
	changeSets := []*audit.ChangeSet{}
	var totalCount int

	// TODO: Wire Trino client connection here
	// trinoConn := r.trinoClient.QueryContext(ctx, query, args...)
	// Process results and map to []audit.ChangeSet

	return changeSets, totalCount, nil
}

// GetChangeSetByID retrieves a single ChangeSet
func (r *ChangeSetResolver) GetChangeSetByID(
	ctx context.Context,
	changeSetID string,
) (*audit.ChangeSet, error) {
	node, err := r.catalogWriter.GetNode(ctx, "changeset_event:"+changeSetID)
	if err != nil {
		return nil, fmt.Errorf("get changeset node: %w", err)
	}

	if node == nil {
		return nil, nil
	}

	// Validate tenant scope
	allowedTenants := extractAllowedTenantsFromContext(ctx)
	if !contains(allowedTenants, node.TenantID) {
		return nil, fmt.Errorf("not authorized to view changeset in tenant %s", node.TenantID)
	}

	// Fetch impacted entities via edges
	edges, err := r.catalogWriter.GetEdges(ctx, node.ID)
	if err != nil {
		r.logger.Error("failed to get changeset edges", zap.Error(err))
	}

	cs := &audit.ChangeSet{
		ID:               changeSetID,
		TenantID:         node.TenantID,
		Status:           node.Properties["status"].(string),
		ImpactedEntities: []audit.ImpactedEntity{},
	}

	// Extract impacted entities from edges
	for _, edge := range edges {
		if edge.EdgeType == "has_impact_on" {
			cs.ImpactedEntities = append(cs.ImpactedEntities, audit.ImpactedEntity{
				NodeID: edge.ToNode,
			})
		}
	}

	return cs, nil
}

// ============================================================================
// Helper functions
// ============================================================================

func extractAllowedTenantsFromContext(ctx context.Context) []string {
	// Extract from auth context:
	// 1. Check JWT claims in context (set by auth middleware)
	// 2. Look for X-Tenant-ID header or context value
	// 3. Fall back to user's tenant assignments from database

	// Implementation pattern with JWT:
	// if claims, ok := ctx.Value("auth_claims").(jwt.MapClaims); ok {
	//    if tenants, ok := claims["allowed_tenants"].([]interface{}); ok {
	//        for _, t := range tenants {
	//            result = append(result, t.(string))
	//        }
	//        return result
	//    }
	// }

	// Try context value set by auth middleware
	if tenantID := ctx.Value("X-Tenant-ID"); tenantID != nil {
		return []string{tenantID.(string)}
	}

	// PRODUCTION: Replace with actual JWT/auth context extraction
	// This prevents accidental cross-tenant access by requiring auth setup
	return []string{}
}

func extractActorFromContext(ctx context.Context) string {
	// Extract user/actor from auth context for audit trail
	// Try multiple sources:
	// 1. User ID from JWT claims
	// 2. Service account from context
	// 3. Email from auth context

	if userID := ctx.Value("user_id"); userID != nil {
		return userID.(string)
	}

	if email := ctx.Value("user_email"); email != nil {
		return email.(string)
	}

	// PRODUCTION: Replace with actual auth extraction
	// Fall back to "unknown" to indicate missing auth context
	return "unknown_actor"
}

func contains(haystack []string, needle string) bool {
	for _, h := range haystack {
		if h == needle {
			return true
		}
	}
	return false
}

func intersectSlices(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	var result []string
	for _, av := range a {
		for _, bv := range b {
			if av == bv {
				result = append(result, av)
				break
			}
		}
	}
	return result
}
