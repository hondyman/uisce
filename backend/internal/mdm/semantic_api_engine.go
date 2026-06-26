package mdm

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// SemanticAPIBundle represents an auto-generated API contract for a Business Object.
type SemanticAPIBundle struct {
	ID          string                 `json:"id"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Endpoints   []SemanticEndpoint     `json:"endpoints"`
	Relations   []SemanticRelation     `json:"relations"`
	Schema      map[string]interface{} `json:"schema"`
	OpenAPISpec map[string]interface{} `json:"openapi_spec"`
}

type SemanticEndpoint struct {
	Path        string `json:"path"`
	Verb        string `json:"verb"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
}

type SemanticRelation struct {
	TargetBO     string `json:"target_bo"`
	Type         string `json:"type"`
	Multiplicity string `json:"multiplicity"`
}

// SemanticAPIEngine generates dynamic API contracts from the semantic graph.
type SemanticAPIEngine struct {
	graphService *analytics.SemanticGraphService
}

func NewSemanticAPIEngine(gs *analytics.SemanticGraphService) *SemanticAPIEngine {
	return &SemanticAPIEngine{graphService: gs}
}

// GenerateBundleForBO creates a semantic API bundle for a specific business object.
func (e *SemanticAPIEngine) GenerateBundleForBO(ctx context.Context, tenantID uuid.UUID, boID uuid.UUID) (*SemanticAPIBundle, error) {
	node, err := e.graphService.GetNodeByID(boID)
	if err != nil {
		return nil, err
	}
	if node == nil || node.NodeType != "business_object" {
		return nil, fmt.Errorf("node not found or not a business_object")
	}

	bundle := &SemanticAPIBundle{
		ID:          node.ID.String(),
		DisplayName: node.NodeName,
		Description: node.Description,
		Schema:      make(map[string]interface{}),
	}

	// 1. Discover basic endpoints
	bundle.Endpoints = []SemanticEndpoint{
		{
			Path:    fmt.Sprintf("/api/v1/%s", node.NodeName),
			Verb:    "GET",
			Summary: fmt.Sprintf("List %s records", node.NodeName),
		},
		{
			Path:    fmt.Sprintf("/api/v1/%s/{id}", node.NodeName),
			Verb:    "GET",
			Summary: fmt.Sprintf("Get single %s record", node.NodeName),
		},
	}

	// 2. Discover columns and relations via outgoing edges
	edges, err := e.graphService.GetOutgoingEdges(boID)
	if err == nil {
		for _, edge := range edges {
			// Handle columns -> build schema
			if edge.EdgeType == analytics.EdgeTypeBOHasColumn || edge.EdgeType == analytics.EdgeTypeBOHasAttribute {
				target, tErr := e.graphService.GetNodeByID(edge.TargetNodeID)
				if tErr == nil && target != nil {
					bundle.Schema[target.NodeName] = map[string]string{
						"type": "string", // Default, could be enriched from target properties
					}
				}
			}

			// Handle relations
			if edge.EdgeType == analytics.EdgeTypeBORelatesToBO || edge.EdgeType == "holds_security" {
				target, tErr := e.graphService.GetNodeByID(edge.TargetNodeID)
				if tErr == nil && target != nil {
					bundle.Relations = append(bundle.Relations, SemanticRelation{
						TargetBO: target.NodeName,
						Type:     string(edge.EdgeType),
					})

					// Add related endpoint
					bundle.Endpoints = append(bundle.Endpoints, SemanticEndpoint{
						Path:    fmt.Sprintf("/api/v1/%s/{id}/%s", node.NodeName, target.NodeName),
						Verb:    "GET",
						Summary: fmt.Sprintf("Get related %s", target.NodeName),
					})
				}
			}
		}
	}

	bundle.OpenAPISpec = e.constructOpenAPI(bundle)

	return bundle, nil
}

func (e *SemanticAPIEngine) constructOpenAPI(bundle *SemanticAPIBundle) map[string]interface{} {
	paths := make(map[string]interface{})
	for _, ep := range bundle.Endpoints {
		paths[ep.Path] = map[string]interface{}{
			strings.ToLower(ep.Verb): map[string]interface{}{
				"summary":     ep.Summary,
				"description": ep.Description,
				"responses": map[string]interface{}{
					"200": map[string]interface{}{
						"description": "Successful operation",
						"content": map[string]interface{}{
							"application/json": map[string]interface{}{
								"schema": map[string]interface{}{
									"type":       "object",
									"properties": bundle.Schema,
								},
							},
						},
					},
				},
			},
		}
	}

	return map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":       fmt.Sprintf("Semantic API: %s", bundle.DisplayName),
			"version":     "1.0.0",
			"description": bundle.Description,
		},
		"paths": paths,
		"components": map[string]interface{}{
			"schemas": map[string]interface{}{
				bundle.DisplayName: map[string]interface{}{
					"type":       "object",
					"properties": bundle.Schema,
				},
			},
		},
	}
}

// ListAllBundles returns bundles for all business objects in the graph.
func (e *SemanticAPIEngine) ListAllBundles(ctx context.Context, tenantID uuid.UUID) ([]SemanticAPIBundle, error) {
	nodes, err := e.graphService.GetNodesByType(analytics.NodeTypeBusinessObject, tenantID)
	if err != nil {
		return nil, err
	}

	var bundles []SemanticAPIBundle
	for _, node := range nodes {
		bundle, err := e.GenerateBundleForBO(ctx, tenantID, node.ID)
		if err == nil && bundle != nil {
			bundles = append(bundles, *bundle)
		}
	}

	return bundles, nil
}
