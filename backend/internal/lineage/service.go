package lineage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// LineageRepository defines the interface for lineage storage
type LineageRepository interface {
	UpsertNode(ctx context.Context, node LineageNode) error
	UpsertEdge(ctx context.Context, edge LineageEdge) error
	DeleteNode(ctx context.Context, id string) error
	DeleteEdge(ctx context.Context, fromID, toID, edgeType string) error
	FindDownstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error)
	FindUpstreamGraph(ctx context.Context, rootID string, depth int) (*Graph, error)
	FindBiDirectionalGraph(ctx context.Context, rootID string, depth int) (*Graph, error)
	FindGraphByDatasource(ctx context.Context, datasourceID string) (*Graph, error)
	SyncDatasource(ctx context.Context, datasourceID string) error
}

// LineageService orchestrates lineage operations
type LineageService struct {
	repo LineageRepository
}

// NewLineageService creates a new lineage service
func NewLineageService(repo LineageRepository) *LineageService {
	return &LineageService{repo: repo}
}

// Repo returns the underlying repository
func (s *LineageService) Repo() LineageRepository {
	return s.repo
}

// RebuildForObject rebuilds lineage for a semantic object
func (s *LineageService) RebuildForObject(ctx context.Context, objType, objID string, payload []byte) error {
	// Simplified parsing logic
	// In a real implementation, we would unmarshal payload into specific struct based on objType
	// and call IngestBusinessObject, etc.

	// Example stub:
	var meta map[string]interface{}
	if err := json.Unmarshal(payload, &meta); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	env, _ := meta["env"].(string)
	tenantID, _ := meta["tenant_id"].(string)
	name, _ := meta["name"].(string)

	boUUID, _ := uuid.Parse(objID)

	switch objType {
	case "bo":
		var deps []uuid.UUID
		if d, ok := meta["dependencies"].([]interface{}); ok {
			for _, dep := range d {
				if idStr, ok := dep.(string); ok {
					if id, err := uuid.Parse(idStr); err == nil {
						deps = append(deps, id)
					}
				}
			}
		}
		return s.IngestBusinessObject(ctx, boUUID, name, env, tenantID, deps, meta)
	case "api_endpoint":
		boName, _ := meta["bo_name"].(string)
		return s.IngestAPIEndpoint(ctx, boUUID, name, env, tenantID, boName)
	case "page_core":
		var apiDeps []string
		if db, ok := meta["data_bindings"].(map[string]interface{}); ok {
			if sources, ok := db["sources"].(map[string]interface{}); ok {
				for _, src := range sources {
					if sMap, ok := src.(map[string]interface{}); ok {
						if eid, ok := sMap["endpointId"].(string); ok {
							apiDeps = append(apiDeps, eid)
						}
					}
				}
			}
		}
		return s.IngestPage(ctx, boUUID, name, env, tenantID, apiDeps)
	case "page_overlay":
		parentIDStr, _ := meta["parent_id"].(string)
		if parentID, err := uuid.Parse(parentIDStr); err == nil {
			return s.IngestPageOverlay(ctx, boUUID, parentID, env, tenantID)
		}
	case "aso_opt":
		// ...
	}
	return nil
}

// IngestBusinessObject ingests a BO and its dependencies
func (s *LineageService) IngestBusinessObject(ctx context.Context, boID uuid.UUID, name, env, tenantID string, dependencies []uuid.UUID, meta map[string]interface{}) error {
	// Ensure source is set
	if meta == nil {
		meta = make(map[string]interface{})
	}
	meta["source"] = "ingestion"

	node := LineageNode{
		ID:       boID.String(),
		Type:     NodeBO,
		Name:     name,
		Env:      env,
		TenantID: &tenantID,
		Metadata: mustMarshal(meta),
	}

	if err := s.repo.UpsertNode(ctx, node); err != nil {
		return fmt.Errorf("failed to upsert BO node: %w", err)
	}

	for _, depID := range dependencies {
		edge := LineageEdge{
			FromID:   boID.String(),
			ToID:     depID.String(),
			Type:     EdgeDependsOn,
			Env:      env,
			TenantID: &tenantID,
		}
		if err := s.repo.UpsertEdge(ctx, edge); err != nil {
			log.Printf("Failed to upsert edge: %v", err)
		}
	}

	return nil
}

// IngestASOOptimization ingests an optimization and links it to targets
func (s *LineageService) IngestASOOptimization(ctx context.Context, optID uuid.UUID, targetID uuid.UUID, env, tenantID string) error {
	node := LineageNode{
		ID:       optID.String(),
		Type:     NodeASOOpt,
		Env:      env,
		TenantID: &tenantID,
	}
	if err := s.repo.UpsertNode(ctx, node); err != nil {
		return err
	}

	edge := LineageEdge{
		FromID:   targetID.String(),
		ToID:     optID.String(),
		Type:     EdgeOptimizedBy,
		Env:      env,
		TenantID: &tenantID,
	}

	return s.repo.UpsertEdge(ctx, edge)
}

// FindDownstreamASO finds all ASO optimizations that would be affected by a change to the given node
func (s *LineageService) FindDownstreamASO(ctx context.Context, nodeID string) ([]string, error) {
	// Get downstream graph looking for ASOOptimization nodes
	graph, err := s.repo.FindDownstreamGraph(ctx, nodeID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to find downstream graph: %w", err)
	}

	var asoIDs []string
	for _, node := range graph.Nodes {
		if node.Type == NodeASOOpt {
			asoIDs = append(asoIDs, node.ID)
		}
	}
	return asoIDs, nil
}

// IngestAPIEndpoint ingests an API endpoint and links it to its BO
func (s *LineageService) IngestAPIEndpoint(ctx context.Context, apiID uuid.UUID, name, env, tenantID, boName string) error {
	node := LineageNode{
		ID:       apiID.String(),
		Type:     NodeAPIEndpoint,
		Name:     name,
		Env:      env,
		TenantID: &tenantID,
	}
	if err := s.repo.UpsertNode(ctx, node); err != nil {
		return err
	}

	// Link to BO (by name for now as we don't have ID in APIEndpoint struct)
	// In a more robust system, we would resolve BO name to ID
	edge := LineageEdge{
		FromID:   apiID.String(),
		ToID:     "bo:" + boName, // Symbolic ID
		Type:     EdgeDependsOn,
		Env:      env,
		TenantID: &tenantID,
	}
	return s.repo.UpsertEdge(ctx, edge)
}

// IngestPage ingests a page and its API dependencies
func (s *LineageService) IngestPage(ctx context.Context, pageID uuid.UUID, name, env, tenantID string, apiEndpointIDs []string) error {
	node := LineageNode{
		ID:       pageID.String(),
		Type:     NodePage,
		Name:     name,
		Env:      env,
		TenantID: &tenantID,
	}
	if err := s.repo.UpsertNode(ctx, node); err != nil {
		return err
	}

	for _, apiID := range apiEndpointIDs {
		edge := LineageEdge{
			FromID:   pageID.String(),
			ToID:     apiID,
			Type:     EdgeDependsOn,
			Env:      env,
			TenantID: &tenantID,
		}
		if err := s.repo.UpsertEdge(ctx, edge); err != nil {
			log.Printf("Failed to upsert edge for page %s to api %s: %v", pageID, apiID, err)
		}
	}
	return nil
}

// IngestPageOverlay links an overlay to its parent core page
func (s *LineageService) IngestPageOverlay(ctx context.Context, overlayID, parentID uuid.UUID, env, tenantID string) error {
	node := LineageNode{
		ID:       overlayID.String(),
		Type:     NodePage,
		Env:      env,
		TenantID: &tenantID,
	}
	if err := s.repo.UpsertNode(ctx, node); err != nil {
		return err
	}

	edge := LineageEdge{
		FromID:   overlayID.String(),
		ToID:     parentID.String(),
		Type:     EdgeOverrides,
		Env:      env,
		TenantID: &tenantID,
	}
	return s.repo.UpsertEdge(ctx, edge)
}

// InvalidateASO marks ASO optimizations as stale when upstream nodes change
func (s *LineageService) InvalidateASO(ctx context.Context, nodeID string, asoInvalidator ASOInvalidator) error {
	optIDs, err := s.FindDownstreamASO(ctx, nodeID)
	if err != nil {
		return err
	}

	for _, id := range optIDs {
		optUUID, parseErr := uuid.Parse(id)
		if parseErr != nil {
			log.Printf("Invalid ASO optimization ID %s: %v", id, parseErr)
			continue
		}
		if err := asoInvalidator.MarkStale(ctx, optUUID); err != nil {
			log.Printf("Failed to mark ASO optimization %s as stale: %v", id, err)
		}
	}
	return nil
}

// ImpactOfNode returns a comprehensive impact analysis for a given node
func (s *LineageService) ImpactOfNode(ctx context.Context, nodeID string, depth int) (*ImpactReport, error) {
	downstream, err := s.repo.FindDownstreamGraph(ctx, nodeID, depth)
	if err != nil {
		return nil, err
	}

	report := &ImpactReport{
		NodeID: nodeID,
	}

	// Categorize downstream impacts
	for _, node := range downstream.Nodes {
		switch node.Type {
		case NodeBO:
			report.AffectedBOs = append(report.AffectedBOs, node)
		case NodePreAgg:
			report.AffectedPreAggs = append(report.AffectedPreAggs, node)
		case NodeEntitlement:
			report.AffectedEntitlements = append(report.AffectedEntitlements, node)
		case NodeASOOpt:
			report.AffectedASOOptimizations = append(report.AffectedASOOptimizations, node)
		case NodePage:
			report.AffectedPages = append(report.AffectedPages, node)
		case NodeAPIEndpoint:
			report.AffectedAPIEndpoints = append(report.AffectedAPIEndpoints, node)
		case NodeTenant:
			if node.TenantID != nil {
				report.AffectedTenants = append(report.AffectedTenants, *node.TenantID)
			}
		}
	}

	return report, nil
}

// ASOInvalidator is an interface for marking ASO optimizations as stale
type ASOInvalidator interface {
	MarkStale(ctx context.Context, optID uuid.UUID) error
}
