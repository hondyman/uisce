package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// SemanticGraphHandler handles semantic graph operations
type SemanticGraphHandler struct {
	GraphService *analytics.SemanticGraphService
	Resolver     *analytics.BOContextResolver
	CacheService *analytics.BOSQLCacheService
}

// NewSemanticGraphHandler creates a new handler
func NewSemanticGraphHandler(
	graphService *analytics.SemanticGraphService,
	resolver *analytics.BOContextResolver,
	cacheService *analytics.BOSQLCacheService,
) *SemanticGraphHandler {
	return &SemanticGraphHandler{
		GraphService: graphService,
		Resolver:     resolver,
		CacheService: cacheService,
	}
}

// RegisterRoutes registers semantic graph routes
func (h *SemanticGraphHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/graph/nodes", h.CreateNode)
	r.Get("/api/graph/nodes/{nodeType}/{nodeName}", h.GetNode)
	r.Post("/api/graph/edges", h.CreateEdge)
	r.Get("/api/graph/nodes/{nodeId}/edges", h.GetEdges)
	r.Post("/api/graph/resolve/term", h.ResolveTerm)
	r.Post("/api/graph/resolve/calculation", h.ResolveCalculation)
	r.Post("/api/graph/resolve/bo-sql", h.GenerateBOSQL)
	r.Get("/api/graph/bo/{boName}/calculations", h.GetBOCalculations)
	r.Get("/api/graph/bo/{boName}/terms", h.GetBOTerms)
	r.Post("/api/graph/bo/{boName}/assign-calculation", h.AssignCalculation)
	r.Post("/api/graph/bo/{boName}/assign-term", h.AssignTerm)
}

// CreateNodeRequest represents a node creation request
type CreateNodeRequest struct {
	NodeType     string                 `json:"node_type"`
	NodeName     string                 `json:"node_name"`
	Description  string                 `json:"description"`
	Properties   map[string]interface{} `json:"properties"`
	Config       map[string]interface{} `json:"config"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
}

// CreateNode creates a new node
// POST /api/graph/nodes
func (h *SemanticGraphHandler) CreateNode(w http.ResponseWriter, r *http.Request) {
	var req CreateNodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	nodeID, err := h.GraphService.CreateNode(
		analytics.NodeType(req.NodeType),
		req.NodeName,
		req.Description,
		req.Properties,
		req.Config,
		tenantID,
		datasourceID,
	)
	if err != nil {
		http.Error(w, "Failed to create node: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"node_id": nodeID.String()})
}

// GetNode retrieves a node by type and name
// GET /api/graph/nodes/{nodeType}/{nodeName}
func (h *SemanticGraphHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	nodeType := chi.URLParam(r, "nodeType")
	nodeName := chi.URLParam(r, "nodeName")
	datasourceID := r.URL.Query().Get("datasource_id")

	dsID, _ := uuid.Parse(datasourceID)

	node, err := h.GraphService.GetNodeByName(analytics.NodeType(nodeType), nodeName, dsID)
	if err != nil {
		http.Error(w, "Failed to get node: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if node == nil {
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(node)
}

// CreateEdgeRequest represents an edge creation request
type CreateEdgeRequest struct {
	SourceNodeID string                 `json:"source_node_id"`
	TargetNodeID string                 `json:"target_node_id"`
	EdgeType     string                 `json:"edge_type"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	Properties   map[string]interface{} `json:"properties"`
}

// CreateEdge creates an edge between nodes
// POST /api/graph/edges
func (h *SemanticGraphHandler) CreateEdge(w http.ResponseWriter, r *http.Request) {
	var req CreateEdgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sourceID, _ := uuid.Parse(req.SourceNodeID)
	targetID, _ := uuid.Parse(req.TargetNodeID)
	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	edgeID, err := h.GraphService.CreateEdge(
		sourceID,
		targetID,
		analytics.EdgeType(req.EdgeType),
		tenantID,
		datasourceID,
		req.Properties,
	)
	if err != nil {
		http.Error(w, "Failed to create edge: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"edge_id": edgeID.String()})
}

// GetEdges retrieves edges for a node
// GET /api/graph/nodes/{nodeId}/edges
func (h *SemanticGraphHandler) GetEdges(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := chi.URLParam(r, "nodeId")
	nodeID, _ := uuid.Parse(nodeIDStr)
	direction := r.URL.Query().Get("direction") // "outgoing" or "incoming"

	var edges []analytics.SemanticEdge
	var err error

	if direction == "incoming" {
		edges, err = h.GraphService.GetIncomingEdges(nodeID)
	} else {
		edges, err = h.GraphService.GetOutgoingEdges(nodeID)
	}

	if err != nil {
		http.Error(w, "Failed to get edges: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(edges)
}

// ResolveTermRequest represents a term resolution request
type ResolveTermRequest struct {
	TermName     string `json:"term_name"`
	BOName       string `json:"bo_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
	Dialect      string `json:"dialect"`
}

// ResolveTerm resolves a term using BO context
// POST /api/graph/resolve/term
func (h *SemanticGraphHandler) ResolveTerm(w http.ResponseWriter, r *http.Request) {
	var req ResolveTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	ctx, err := h.Resolver.GetBOContext(req.BOName, tenantID, datasourceID, req.Dialect)
	if err != nil {
		http.Error(w, "Failed to get BO context: "+err.Error(), http.StatusBadRequest)
		return
	}

	resolved, err := h.Resolver.ResolveTerm(req.TermName, *ctx)
	if err != nil {
		http.Error(w, "Failed to resolve term: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resolved)
}

// ResolveCalculationRequest represents a calculation resolution request
type ResolveCalculationRequest struct {
	CalcName     string `json:"calc_name"`
	BOName       string `json:"bo_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
	Dialect      string `json:"dialect"`
}

// ResolveCalculation resolves a calculation using BO context
// POST /api/graph/resolve/calculation
func (h *SemanticGraphHandler) ResolveCalculation(w http.ResponseWriter, r *http.Request) {
	var req ResolveCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	ctx, err := h.Resolver.GetBOContext(req.BOName, tenantID, datasourceID, req.Dialect)
	if err != nil {
		http.Error(w, "Failed to get BO context: "+err.Error(), http.StatusBadRequest)
		return
	}

	resolved, err := h.Resolver.ResolveCalculation(req.CalcName, *ctx)
	if err != nil {
		http.Error(w, "Failed to resolve calculation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resolved)
}

// GenerateBOSQLRequest represents a BO SQL generation request
type GenerateBOSQLRequest struct {
	BOName       string   `json:"bo_name"`
	Terms        []string `json:"terms"`
	Calculations []string `json:"calculations"`
	TenantID     string   `json:"tenant_id"`
	DatasourceID string   `json:"datasource_id"`
	Dialect      string   `json:"dialect"`
}

// GenerateBOSQL generates full SQL for a BO
// POST /api/graph/resolve/bo-sql
func (h *SemanticGraphHandler) GenerateBOSQL(w http.ResponseWriter, r *http.Request) {
	var req GenerateBOSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	// Use CacheService which wraps Resolver (GetOrGenerate)
	sql, err := h.CacheService.GetOrGenerateBOSQL(req.BOName, req.Terms, req.Calculations, tenantID, datasourceID, req.Dialect)
	if err != nil {
		http.Error(w, "Failed to generate SQL: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"sql": sql})
}

// GetBOCalculations retrieves calculations for a BO
// GET /api/graph/bo/{boName}/calculations
func (h *SemanticGraphHandler) GetBOCalculations(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	datasourceID := r.URL.Query().Get("datasource_id")
	tenantID := r.URL.Query().Get("tenant_id")

	tID, _ := uuid.Parse(tenantID)
	dsID, _ := uuid.Parse(datasourceID)

	ctx, err := h.Resolver.GetBOContext(boName, tID, dsID, "postgres")
	if err != nil {
		http.Error(w, "BO not found", http.StatusNotFound)
		return
	}

	calcs, err := h.Resolver.GetBOCalculations(ctx.BOID)
	if err != nil {
		http.Error(w, "Failed to get calculations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calcs)
}

// GetBOTerms retrieves terms for a BO
// GET /api/graph/bo/{boName}/terms
func (h *SemanticGraphHandler) GetBOTerms(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	datasourceID := r.URL.Query().Get("datasource_id")
	tenantID := r.URL.Query().Get("tenant_id")

	tID, _ := uuid.Parse(tenantID)
	dsID, _ := uuid.Parse(datasourceID)

	ctx, err := h.Resolver.GetBOContext(boName, tID, dsID, "postgres")
	if err != nil {
		http.Error(w, "BO not found", http.StatusNotFound)
		return
	}

	terms, err := h.Resolver.GetBOTerms(ctx.BOID)
	if err != nil {
		http.Error(w, "Failed to get terms: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(terms)
}

// AssignCalculationRequest represents a calculation assignment request
type AssignCalculationRequest struct {
	CalcName     string `json:"calc_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
}

// AssignCalculation assigns a calculation to a BO
// POST /api/graph/bo/{boName}/assign-calculation
func (h *SemanticGraphHandler) AssignCalculation(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	var req AssignCalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	// Get BO node
	boNode, err := h.GraphService.GetNodeByName(analytics.NodeTypeBusinessObject, boName, datasourceID)
	if err != nil || boNode == nil {
		http.Error(w, "BO not found", http.StatusNotFound)
		return
	}

	// Get calculation node
	calcNode, err := h.GraphService.GetNodeByName(analytics.NodeTypeCalculationTerm, req.CalcName, datasourceID)
	if err != nil || calcNode == nil {
		http.Error(w, "Calculation not found", http.StatusNotFound)
		return
	}

	// Create edge
	err = h.Resolver.AssignCalculationToBO(boNode.ID, calcNode.ID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, "Failed to assign calculation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// AssignTermRequest represents a term assignment request
type AssignTermRequest struct {
	TermName     string `json:"term_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
}

// AssignTerm assigns a term to a BO
// POST /api/graph/bo/{boName}/assign-term
func (h *SemanticGraphHandler) AssignTerm(w http.ResponseWriter, r *http.Request) {
	boName := chi.URLParam(r, "boName")
	var req AssignTermRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, _ := uuid.Parse(req.TenantID)
	datasourceID, _ := uuid.Parse(req.DatasourceID)

	// Get BO node
	boNode, err := h.GraphService.GetNodeByName(analytics.NodeTypeBusinessObject, boName, datasourceID)
	if err != nil || boNode == nil {
		http.Error(w, "BO not found", http.StatusNotFound)
		return
	}

	// Get term node
	termNode, err := h.GraphService.GetNodeByName(analytics.NodeTypeSemanticTerm, req.TermName, datasourceID)
	if err != nil || termNode == nil {
		http.Error(w, "Term not found", http.StatusNotFound)
		return
	}

	// Create edge
	err = h.Resolver.AssignTermToBO(boNode.ID, termNode.ID, tenantID, datasourceID)
	if err != nil {
		http.Error(w, "Failed to assign term: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
