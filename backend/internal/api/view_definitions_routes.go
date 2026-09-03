package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/graphview"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/lineage"
)

// ViewDefinitionsHandler serves tenant-configurable graph visualization definitions
// (ERD, semantic lineage, taxonomy lineage, tenant-custom views) and renders a
// normalized graph for a given view + root node, reusing the existing lineage
// traversal (FindUpstreamGraph/FindDownstreamGraph/FindBiDirectionalGraph) rather
// than reimplementing it.
type ViewDefinitionsHandler struct {
	db           *sql.DB
	lineageRepo  lineage.LineageRepository
	securityDeps handlers.SecurityContextDeps
}

func NewViewDefinitionsHandler(db *sql.DB, lineageRepo lineage.LineageRepository, securityDeps handlers.SecurityContextDeps) *ViewDefinitionsHandler {
	return &ViewDefinitionsHandler{db: db, lineageRepo: lineageRepo, securityDeps: securityDeps}
}

// ViewDefinition is a tenant-scoped graph visualization config. Config shape:
//
//	{
//	  "typePolicy": {"defaultInclude": true, "nodeTypes": {"column":"include"}, "edgeTypes": {}},
//	  "grouping": [{"childNodeType":"column","clusterLabel":"Columns","defaultCollapsed":true,"collapseThreshold":15}],
//	  "layout": {"algorithm":"dagre","direction":"LR"},
//	  "assignedAssetTypes": {"boSubtypes":[...], "catalogNodeTypes":[...]}
//	}
type ViewDefinition struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id"`
	ViewKey     string                 `json:"view_key"`
	DisplayName string                 `json:"display_name"`
	Description *string                `json:"description,omitempty"`
	IsCore      bool                   `json:"is_core"`
	IsActive    bool                   `json:"is_active"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type typePolicy struct {
	DefaultInclude *bool             `json:"defaultInclude"`
	NodeTypes      map[string]string `json:"nodeTypes"`
	EdgeTypes      map[string]string `json:"edgeTypes"`
}

type groupingRule struct {
	ChildNodeType     string `json:"childNodeType"`
	ParentRelation    string `json:"parentRelation"`
	ClusterLabel      string `json:"clusterLabel"`
	DefaultCollapsed  bool   `json:"defaultCollapsed"`
	CollapseThreshold int    `json:"collapseThreshold"`
}

func RegisterViewDefinitionsRoutes(r chi.Router, db *sql.DB, lineageRepo lineage.LineageRepository, securityDeps handlers.SecurityContextDeps) {
	h := NewViewDefinitionsHandler(db, lineageRepo, securityDeps)
	r.Get("/view-definitions", h.handleList)
	r.Post("/view-definitions", h.handleCreate)
	r.Get("/view-definitions/for-asset/{assetType}/{assetSubtype}", h.handleForAsset)
	r.Get("/view-definitions/{id}", h.handleGet)
	r.Patch("/view-definitions/{id}", h.handleUpdate)
	r.Delete("/view-definitions/{id}", h.handleDelete)
	r.Get("/view-definitions/{id}/graph/{rootNodeId}", h.handleGraph)
}

func (h *ViewDefinitionsHandler) tenantID(r *http.Request) (string, error) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		return "", err
	}
	return secCtx.TenantID, nil
}

func scanViewDefinition(scan func(dest ...interface{}) error) (*ViewDefinition, error) {
	var vd ViewDefinition
	var description sql.NullString
	var configJSON []byte
	if err := scan(&vd.ID, &vd.TenantID, &vd.ViewKey, &vd.DisplayName, &description,
		&vd.IsCore, &vd.IsActive, &configJSON, &vd.CreatedAt, &vd.UpdatedAt); err != nil {
		return nil, err
	}
	if description.Valid {
		vd.Description = &description.String
	}
	config, err := unmarshalConfigJSON(configJSON)
	if err != nil {
		return nil, err
	}
	vd.Config = config
	return &vd, nil
}

const viewDefinitionColumns = `id, tenant_id, view_key, display_name, description, is_core, is_active, config, created_at, updated_at`

// handleList returns the tenant's own views plus any is_core shipped views.
func (h *ViewDefinitionsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	query := `SELECT ` + viewDefinitionColumns + `
		FROM catalog_view_definitions
		WHERE is_active = true AND (tenant_id = $1 OR is_core = true)
		ORDER BY is_core ASC, display_name ASC`

	rows, err := h.db.Query(query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	views := []ViewDefinition{}
	for rows.Next() {
		vd, err := scanViewDefinition(rows.Scan)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		views = append(views, *vd)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(views)
}

// handleForAsset resolves which views apply to a given asset type/subtype via
// config.assignedAssetTypes, falling back to the core views if none match.
func (h *ViewDefinitionsHandler) handleForAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	assetType := chi.URLParam(r, "assetType")
	assetSubtype := chi.URLParam(r, "assetSubtype")

	query := `SELECT ` + viewDefinitionColumns + `
		FROM catalog_view_definitions
		WHERE is_active = true AND (tenant_id = $1 OR is_core = true)
		ORDER BY is_core ASC, display_name ASC`

	rows, err := h.db.Query(query, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	matched := []ViewDefinition{}
	core := []ViewDefinition{}
	for rows.Next() {
		vd, err := scanViewDefinition(rows.Scan)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if vd.IsCore {
			core = append(core, *vd)
		}
		if assignedAssetTypesMatch(vd.Config, assetType, assetSubtype) {
			matched = append(matched, *vd)
		}
	}

	if len(matched) == 0 {
		matched = core
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matched)
}

func assignedAssetTypesMatch(config map[string]interface{}, assetType, assetSubtype string) bool {
	assigned, ok := config["assignedAssetTypes"].(map[string]interface{})
	if !ok {
		return false
	}
	if list, ok := assigned["catalogNodeTypes"].([]interface{}); ok {
		for _, v := range list {
			if s, ok := v.(string); ok && strings.EqualFold(s, assetType) {
				return true
			}
		}
	}
	if list, ok := assigned["boSubtypes"].([]interface{}); ok {
		for _, v := range list {
			if s, ok := v.(string); ok && strings.EqualFold(s, assetSubtype) {
				return true
			}
		}
	}
	return false
}

func (h *ViewDefinitionsHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")

	query := `SELECT ` + viewDefinitionColumns + `
		FROM catalog_view_definitions WHERE id = $1 AND (tenant_id = $2 OR is_core = true)`

	vd, err := scanViewDefinition(func(dest ...interface{}) error {
		return h.db.QueryRow(query, id, tenantID).Scan(dest...)
	})
	if err == sql.ErrNoRows {
		http.Error(w, "View definition not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(vd)
}

func (h *ViewDefinitionsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var vd ViewDefinition
	if err := json.NewDecoder(r.Body).Decode(&vd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if vd.ViewKey == "" || vd.DisplayName == "" {
		http.Error(w, "view_key and display_name are required", http.StatusBadRequest)
		return
	}
	// is_core is server-protected: a tenant can never create a core view.
	vd.IsCore = false
	vd.TenantID = tenantID
	if vd.ID == "" {
		vd.ID = uuid.New().String()
	}
	if vd.Config == nil {
		vd.Config = make(map[string]interface{})
	}

	configJSON, err := json.Marshal(vd.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	query := `
		INSERT INTO catalog_view_definitions
			(id, tenant_id, view_key, display_name, description, is_core, is_active, config, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, false, true, $6, NOW(), NOW())
		RETURNING created_at, updated_at`

	err = h.db.QueryRow(query, vd.ID, vd.TenantID, vd.ViewKey, vd.DisplayName, vd.Description, configJSON).
		Scan(&vd.CreatedAt, &vd.UpdatedAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	vd.IsActive = true

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(vd)
}

func (h *ViewDefinitionsHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")

	var vd ViewDefinition
	if err := json.NewDecoder(r.Body).Decode(&vd); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if vd.Config == nil {
		vd.Config = make(map[string]interface{})
	}
	configJSON, err := json.Marshal(vd.Config)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// is_core rows are server-owned; a tenant may only update its own custom views.
	query := `
		UPDATE catalog_view_definitions
		SET display_name = $1, description = $2, is_active = $3, config = $4, updated_at = NOW()
		WHERE id = $5 AND tenant_id = $6 AND is_core = false
		RETURNING ` + viewDefinitionColumns

	updated, err := scanViewDefinition(func(dest ...interface{}) error {
		return h.db.QueryRow(query, vd.DisplayName, vd.Description, vd.IsActive, configJSON, id, tenantID).Scan(dest...)
	})
	if err == sql.ErrNoRows {
		http.Error(w, "View definition not found or not editable", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *ViewDefinitionsHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")

	result, err := h.db.Exec(`DELETE FROM catalog_view_definitions WHERE id = $1 AND tenant_id = $2 AND is_core = false`, id, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, "View definition not found or not editable", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleGraph loads a ViewDefinition, traverses via the existing lineage repository,
// then applies the view's typePolicy (include/exclude by type) and grouping rules
// (folding matching children into synthetic cluster nodes) before returning the
// normalized graphview.ResponseGraph.
func (h *ViewDefinitionsHandler) handleGraph(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	rootNodeID := chi.URLParam(r, "rootNodeId")

	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth <= 0 {
		depth = 3
	}

	vd, err := scanViewDefinition(func(dest ...interface{}) error {
		return h.db.QueryRow(`SELECT `+viewDefinitionColumns+`
			FROM catalog_view_definitions WHERE id = $1 AND (tenant_id = $2 OR is_core = true)`, id, tenantID).Scan(dest...)
	})
	if err == sql.ErrNoRows {
		http.Error(w, "View definition not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	direction := "both"
	if layoutCfg, ok := vd.Config["layout"].(map[string]interface{}); ok {
		if d, ok := layoutCfg["direction"].(string); ok && d != "" {
			direction = strings.ToLower(d)
		}
	}

	ctx := r.Context()
	var graph *lineage.Graph
	switch {
	case strings.Contains(direction, "up") && !strings.Contains(direction, "down"):
		graph, err = h.lineageRepo.FindUpstreamGraph(ctx, rootNodeID, depth)
	case strings.Contains(direction, "down") && !strings.Contains(direction, "up"):
		graph, err = h.lineageRepo.FindDownstreamGraph(ctx, rootNodeID, depth)
	default:
		graph, err = h.lineageRepo.FindBiDirectionalGraph(ctx, rootNodeID, depth)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	expandedClusters := map[string]bool{}
	if raw := r.URL.Query().Get("expandClusters"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			if id = strings.TrimSpace(id); id != "" {
				expandedClusters[id] = true
			}
		}
	}

	responseGraph := graphview.ConvertLineageGraph(graph)
	applyTypePolicy(responseGraph, vd.Config)
	applyGrouping(responseGraph, vd.Config, expandedClusters)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseGraph)
}

func parseTypePolicy(config map[string]interface{}) typePolicy {
	tp := typePolicy{NodeTypes: map[string]string{}, EdgeTypes: map[string]string{}}
	raw, ok := config["typePolicy"]
	if !ok {
		return tp
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return tp
	}
	_ = json.Unmarshal(b, &tp)
	if tp.NodeTypes == nil {
		tp.NodeTypes = map[string]string{}
	}
	if tp.EdgeTypes == nil {
		tp.EdgeTypes = map[string]string{}
	}
	return tp
}

// applyTypePolicy filters nodes/edges in-place per the view's typePolicy: a type
// is included unless explicitly excluded, or excluded by default when
// defaultInclude is false and the type has no explicit entry.
func applyTypePolicy(graph *graphview.ResponseGraph, config map[string]interface{}) {
	tp := parseTypePolicy(config)
	defaultInclude := true
	if tp.DefaultInclude != nil {
		defaultInclude = *tp.DefaultInclude
	}

	included := func(policyMap map[string]string, typeName string) bool {
		if v, ok := policyMap[typeName]; ok {
			return v == "include"
		}
		return defaultInclude
	}

	keptNodes := make(map[string]bool, len(graph.Nodes))
	filteredNodes := graph.Nodes[:0]
	for _, n := range graph.Nodes {
		if included(tp.NodeTypes, n.Type) {
			filteredNodes = append(filteredNodes, n)
			keptNodes[n.ID] = true
		}
	}
	graph.Nodes = filteredNodes

	filteredEdges := graph.Edges[:0]
	for _, e := range graph.Edges {
		if !included(tp.EdgeTypes, e.Type) {
			continue
		}
		if !keptNodes[e.Source] || !keptNodes[e.Target] {
			continue
		}
		filteredEdges = append(filteredEdges, e)
	}
	graph.Edges = filteredEdges
}

func parseGroupingRules(config map[string]interface{}) []groupingRule {
	raw, ok := config["grouping"]
	if !ok {
		return nil
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var rules []groupingRule
	_ = json.Unmarshal(b, &rules)
	return rules
}

// applyGrouping folds children matching a grouping rule's childNodeType (and,
// when the source edge carries a matching direct parent link, parentRelation)
// into one synthetic cluster node per parent, so a parent with hundreds of
// matching children renders as a single collapsed node instead of a flat fan-out.
// A cluster ID present in expandedClusters is left ungrouped instead, so a
// frontend "expand" action can request just that cluster's real member nodes
// via ?expandClusters=<id> rather than requiring the full member set upfront.
func applyGrouping(graph *graphview.ResponseGraph, config map[string]interface{}, expandedClusters map[string]bool) {
	rules := parseGroupingRules(config)
	if len(rules) == 0 {
		return
	}

	ruleByChildType := make(map[string]groupingRule, len(rules))
	for _, rule := range rules {
		ruleByChildType[rule.ChildNodeType] = rule
	}

	nodeByID := make(map[string]graphview.ResponseNode, len(graph.Nodes))
	for _, n := range graph.Nodes {
		nodeByID[n.ID] = n
	}

	// parent -> matching child IDs, discovered via edges whose target/source is a
	// grouped-type node and whose other end is the parent.
	childrenByParent := make(map[string][]string)
	childParentEdge := make(map[string]string) // childID -> edgeID consumed by grouping

	for _, e := range graph.Edges {
		src, srcOK := nodeByID[e.Source]
		tgt, tgtOK := nodeByID[e.Target]
		if !srcOK || !tgtOK {
			continue
		}
		if rule, ok := ruleByChildType[tgt.Type]; ok && (rule.ParentRelation == "" || strings.EqualFold(rule.ParentRelation, e.Type)) {
			childrenByParent[src.ID] = append(childrenByParent[src.ID], tgt.ID)
			childParentEdge[tgt.ID] = e.ID
			continue
		}
		if rule, ok := ruleByChildType[src.Type]; ok && (rule.ParentRelation == "" || strings.EqualFold(rule.ParentRelation, e.Type)) {
			childrenByParent[tgt.ID] = append(childrenByParent[tgt.ID], src.ID)
			childParentEdge[src.ID] = e.ID
		}
	}

	toRemoveNodes := make(map[string]bool)
	toRemoveEdges := make(map[string]bool)
	var clusterNodes []graphview.ResponseNode
	var clusterEdges []graphview.ResponseEdge

	for parentID, childIDs := range childrenByParent {
		if len(childIDs) == 0 {
			continue
		}
		childType := nodeByID[childIDs[0]].Type
		rule := ruleByChildType[childType]
		threshold := rule.CollapseThreshold
		if threshold <= 0 {
			threshold = 15
		}
		clusterID := "cluster:" + parentID + ":" + childType
		if len(childIDs) < threshold || expandedClusters[clusterID] {
			continue // small enough to render inline, or explicitly expanded by the caller
		}
		label := rule.ClusterLabel
		if label == "" {
			label = childType
		}

		for _, childID := range childIDs {
			toRemoveNodes[childID] = true
			if edgeID, ok := childParentEdge[childID]; ok {
				toRemoveEdges[edgeID] = true
			}
		}

		pid := parentID
		clusterNodes = append(clusterNodes, graphview.ResponseNode{
			ID:        clusterID,
			Type:      "cluster",
			Label:     label,
			ParentID:  &pid,
			IsCluster: true,
			MemberIDs: childIDs,
			Properties: map[string]interface{}{
				"clusterLabel":     label,
				"childNodeType":    childType,
				"memberCount":      len(childIDs),
				"defaultCollapsed": rule.DefaultCollapsed,
			},
		})
		clusterEdges = append(clusterEdges, graphview.ResponseEdge{
			ID:     "edge:" + parentID + "->" + clusterID,
			Source: parentID,
			Target: clusterID,
			Type:   rule.ParentRelation,
		})
	}

	if len(clusterNodes) == 0 {
		return
	}

	filteredNodes := graph.Nodes[:0]
	for _, n := range graph.Nodes {
		if !toRemoveNodes[n.ID] {
			filteredNodes = append(filteredNodes, n)
		}
	}
	graph.Nodes = append(filteredNodes, clusterNodes...)

	filteredEdges := graph.Edges[:0]
	for _, e := range graph.Edges {
		if !toRemoveEdges[e.ID] {
			filteredEdges = append(filteredEdges, e)
		}
	}
	graph.Edges = append(filteredEdges, clusterEdges...)
}
