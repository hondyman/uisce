package api

import (
	"net/http"

	models "github.com/hondyman/semlayer/services/ai-trade-reconciliation/backend/internal/reports"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ReportHandlers holds report-related HTTP handlers
type ReportHandlers struct {
	engine *models.ReportEngine
}

// NewReportHandlers creates new report handlers
func NewReportHandlers(engine *models.ReportEngine) *ReportHandlers {
	return &ReportHandlers{engine: engine}
}

// GetSemanticViews returns all semantic views for a tenant
func (h *ReportHandlers) GetSemanticViews(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	views, err := h.engine.GetSemanticViews(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, views)
}

// GetViewEntities returns all draggable entities from a semantic view
func (h *ReportHandlers) GetViewEntities(c *gin.Context) {
	viewID := c.Param("view_id")

	entities, err := h.engine.GetEntitiesFromView(c.Request.Context(), viewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"view_id":  viewID,
		"entities": entities,
		"count":    len(entities),
	})
}

// GetViewRelationships returns entity relationships within a view
func (h *ReportHandlers) GetViewRelationships(c *gin.Context) {
	viewID := c.Param("view_id")

	relationships, err := h.engine.GetEntityRelationships(c.Request.Context(), viewID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"view_id":       viewID,
		"relationships": relationships,
		"count":         len(relationships),
	})
}

// CreateReportTemplate creates a new report template
func (h *ReportHandlers) CreateReportTemplate(c *gin.Context) {
	tenantID := c.GetString("tenant_id")
	userID := c.GetString("user_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id required"})
		return
	}

	var req struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		RefreshInterval int    `json:"refresh_interval"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	template := &models.ReportTemplate{
		Name:            req.Name,
		Description:     req.Description,
		TenantID:        uuid.MustParse(tenantID),
		CreatedBy:       uuid.MustParse(userID),
		RefreshInterval: req.RefreshInterval,
		IsActive:        true,
		Sections:        []models.ReportSection{},
		Filters:         []models.ReportFilter{},
		Rules:           []models.ReportRule{},
	}

	if err := h.engine.CreateReportTemplate(c.Request.Context(), template); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, template)
}

// GetReportTemplate retrieves a report template
func (h *ReportHandlers) GetReportTemplate(c *gin.Context) {
	templateID := c.Param("template_id")

	template, err := h.engine.GetReportTemplate(c.Request.Context(), templateID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, template)
}

// AddSectionToTemplate adds a new section to a template
func (h *ReportHandlers) AddSectionToTemplate(c *gin.Context) {
	templateID := c.Param("template_id")

	var req struct {
		Title             string                    `json:"title"`
		SectionType       string                    `json:"section_type"`
		DroppedEntities   []models.DragDropEntity   `json:"dropped_entities"`
		Visualization     models.VisualizationSpec  `json:"visualization"`
		GroupByFields     []string                  `json:"group_by_fields"`
		SortByFields      []models.SortField        `json:"sort_by_fields"`
		AggregationFields []models.AggregationField `json:"aggregation_fields"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	section := models.ReportSection{
		Title:             req.Title,
		SectionType:       req.SectionType,
		DroppedEntities:   req.DroppedEntities,
		Visualization:     req.Visualization,
		GroupByFields:     req.GroupByFields,
		SortByFields:      req.SortByFields,
		AggregationFields: req.AggregationFields,
	}

	if err := h.engine.AddSectionToTemplate(c.Request.Context(), templateID, section); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Section added", "section": section})
}

// ApplyFilterToTemplate applies a filter to a template
func (h *ReportHandlers) ApplyFilterToTemplate(c *gin.Context) {
	templateID := c.Param("template_id")

	var req struct {
		FilterType      string      `json:"filter_type"`
		EntityID        string      `json:"entity_id"`
		EntityName      string      `json:"entity_name"`
		Operator        string      `json:"operator"`
		Value           interface{} `json:"value"`
		SecondValue     interface{} `json:"second_value"`
		ApplyToSections []string    `json:"apply_to_sections"`
		DroppedFrom     string      `json:"dropped_from"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filter := models.ReportFilter{
		FilterType:      req.FilterType,
		EntityID:        req.EntityID,
		EntityName:      req.EntityName,
		Operator:        req.Operator,
		FilterValue:     req.Value,
		SecondValue:     req.SecondValue,
		ApplyToSections: req.ApplyToSections,
		DroppedFrom:     req.DroppedFrom,
	}

	if err := h.engine.ApplyFilterToTemplate(c.Request.Context(), templateID, filter); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Filter applied", "filter": filter})
}

// ApplyRuleToTemplate applies a business rule to a template
func (h *ReportHandlers) ApplyRuleToTemplate(c *gin.Context) {
	templateID := c.Param("template_id")

	var req struct {
		Name             string                  `json:"name"`
		Description      string                  `json:"description"`
		Condition        string                  `json:"condition"`
		Action           string                  `json:"action"`
		Priority         int                     `json:"priority"`
		EntitiesInvolved []string                `json:"entities_involved"`
		CreatedFrom      []models.DragDropEntity `json:"created_from"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule := models.ReportRule{
		Name:             req.Name,
		Description:      req.Description,
		Condition:        req.Condition,
		Action:           req.Action,
		Priority:         req.Priority,
		EntitiesInvolved: req.EntitiesInvolved,
		CreatedFrom:      req.CreatedFrom,
		IsActive:         true,
	}

	if err := h.engine.ApplyRuleToTemplate(c.Request.Context(), templateID, rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule applied", "rule": rule})
}

// GenerateReport generates a report from a template
func (h *ReportHandlers) GenerateReport(c *gin.Context) {
	templateID := c.Param("template_id")

	var req struct {
		AdditionalFilters []models.ReportFilter `json:"additional_filters"`
	}
	c.BindJSON(&req)

	generation, err := h.engine.GenerateReportFromTemplate(c.Request.Context(), templateID, req.AdditionalFilters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, generation)
}

// ValidateDragDrop validates a drag-drop operation
func (h *ReportHandlers) ValidateDragDrop(c *gin.Context) {
	var state models.DragDropState

	if err := c.BindJSON(&state); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	valid, err := h.engine.ValidateDragDrop(c.Request.Context(), state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":           valid,
		"allowed_actions": state.AllowedActions,
	})
}

// RegisterReportRoutes registers all report-related routes
func RegisterReportRoutes(router *gin.Engine, handlers *ReportHandlers) {
	api := router.Group("/api/reporting")
	{
		// Semantic views
		api.GET("/views", handlers.GetSemanticViews)
		api.GET("/views/:view_id/entities", handlers.GetViewEntities)
		api.GET("/views/:view_id/relationships", handlers.GetViewRelationships)

		// Report templates
		api.POST("/templates", handlers.CreateReportTemplate)
		api.GET("/templates/:template_id", handlers.GetReportTemplate)

		// Add sections, filters, rules to templates
		api.POST("/templates/:template_id/sections", handlers.AddSectionToTemplate)
		api.POST("/templates/:template_id/filters", handlers.ApplyFilterToTemplate)
		api.POST("/templates/:template_id/rules", handlers.ApplyRuleToTemplate)

		// Generate reports
		api.POST("/templates/:template_id/generate", handlers.GenerateReport)

		// Drag-drop validation
		api.POST("/validate-drag-drop", handlers.ValidateDragDrop)
	}
}
