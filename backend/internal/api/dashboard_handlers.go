package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// DashboardAPIHandlers handles dashboard-related API endpoints
type DashboardAPIHandlers struct {
	db *sql.DB
}

// NewDashboardAPIHandlers creates new dashboard API handlers
func NewDashboardAPIHandlers(db *sql.DB) *DashboardAPIHandlers {
	return &DashboardAPIHandlers{db: db}
}

// GetUserDashboards retrieves all dashboards for a user
func (h *DashboardAPIHandlers) GetUserDashboards(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "userId")
	if userID == "" {
		respondWithError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	query := `
		SELECT id, name, description, widgets, layout, theme, is_public,
			   created_by, created_at, updated_at
		FROM dashboards
		WHERE created_by = $1
		ORDER BY updated_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query, userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve dashboards")
		return
	}
	defer rows.Close()

	var dashboards []models.Dashboard
	for rows.Next() {
		var dashboard models.Dashboard
		var widgetsJSON []byte

		err := rows.Scan(
			&dashboard.ID, &dashboard.Name, &dashboard.Description, &widgetsJSON,
			&dashboard.Layout, &dashboard.Theme, &dashboard.IsPublic,
			&dashboard.CreatedBy, &dashboard.CreatedAt, &dashboard.UpdatedAt,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan dashboard")
			return
		}

		json.Unmarshal(widgetsJSON, &dashboard.Widgets)
		dashboards = append(dashboards, dashboard)
	}

	respond(w, r, dashboards, nil)
}

// GetDashboard retrieves a specific dashboard by ID
func (h *DashboardAPIHandlers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")
	if dashboardID == "" {
		respondWithError(w, http.StatusBadRequest, "Dashboard ID is required")
		return
	}

	query := `
		SELECT id, name, description, widgets, layout, theme, is_public,
			   created_by, created_at, updated_at
		FROM dashboards
		WHERE id = $1
	`

	var dashboard models.Dashboard
	var widgetsJSON []byte

	err := h.db.QueryRowContext(r.Context(), query, dashboardID).Scan(
		&dashboard.ID, &dashboard.Name, &dashboard.Description, &widgetsJSON,
		&dashboard.Layout, &dashboard.Theme, &dashboard.IsPublic,
		&dashboard.CreatedBy, &dashboard.CreatedAt, &dashboard.UpdatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Dashboard not found")
		return
	}

	json.Unmarshal(widgetsJSON, &dashboard.Widgets)
	respond(w, r, dashboard, nil)
}

// CreateDashboard creates a new dashboard
func (h *DashboardAPIHandlers) CreateDashboard(w http.ResponseWriter, r *http.Request) {
	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Generate ID if not provided
	if dashboard.ID == "" {
		dashboard.ID = generateID()
	}

	dashboard.CreatedAt = time.Now()
	dashboard.UpdatedAt = time.Now()

	widgetsJSON, _ := json.Marshal(dashboard.Widgets)

	query := `
		INSERT INTO dashboards (
			id, name, description, widgets, layout, theme, is_public,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := h.db.ExecContext(r.Context(), query,
		dashboard.ID, dashboard.Name, dashboard.Description, widgetsJSON,
		dashboard.Layout, dashboard.Theme, dashboard.IsPublic,
		dashboard.CreatedBy, dashboard.CreatedAt, dashboard.UpdatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to create dashboard")
		return
	}

	respond(w, r, dashboard, nil)
}

// UpdateDashboard updates an existing dashboard
func (h *DashboardAPIHandlers) UpdateDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")
	if dashboardID == "" {
		respondWithError(w, http.StatusBadRequest, "Dashboard ID is required")
		return
	}

	var dashboard models.Dashboard
	if err := json.NewDecoder(r.Body).Decode(&dashboard); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	dashboard.ID = dashboardID
	dashboard.UpdatedAt = time.Now()

	widgetsJSON, _ := json.Marshal(dashboard.Widgets)

	query := `
		UPDATE dashboards
		SET name = $1, description = $2, widgets = $3, layout = $4,
			theme = $5, is_public = $6, updated_at = $7
		WHERE id = $8
	`

	result, err := h.db.ExecContext(r.Context(), query,
		dashboard.Name, dashboard.Description, widgetsJSON,
		dashboard.Layout, dashboard.Theme, dashboard.IsPublic,
		dashboard.UpdatedAt, dashboard.ID,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to update dashboard")
		return
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Dashboard not found")
		return
	}

	respond(w, r, dashboard, nil)
}

// DeleteDashboard deletes a dashboard
func (h *DashboardAPIHandlers) DeleteDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")
	if dashboardID == "" {
		respondWithError(w, http.StatusBadRequest, "Dashboard ID is required")
		return
	}

	query := `DELETE FROM dashboards WHERE id = $1`

	result, err := h.db.ExecContext(r.Context(), query, dashboardID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to delete dashboard")
		return
	}

	if rowsAffected, _ := result.RowsAffected(); rowsAffected == 0 {
		respondWithError(w, http.StatusNotFound, "Dashboard not found")
		return
	}

	respond(w, r, map[string]string{"status": "deleted"}, nil)
}

// GetPublicDashboards retrieves all public dashboards
func (h *DashboardAPIHandlers) GetPublicDashboards(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT id, name, description, widgets, layout, theme, is_public,
			   created_by, created_at, updated_at
		FROM dashboards
		WHERE is_public = true
		ORDER BY updated_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to retrieve public dashboards")
		return
	}
	defer rows.Close()

	var dashboards []models.Dashboard
	for rows.Next() {
		var dashboard models.Dashboard
		var widgetsJSON []byte

		err := rows.Scan(
			&dashboard.ID, &dashboard.Name, &dashboard.Description, &widgetsJSON,
			&dashboard.Layout, &dashboard.Theme, &dashboard.IsPublic,
			&dashboard.CreatedBy, &dashboard.CreatedAt, &dashboard.UpdatedAt,
		)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to scan dashboard")
			return
		}

		json.Unmarshal(widgetsJSON, &dashboard.Widgets)
		dashboards = append(dashboards, dashboard)
	}

	respond(w, r, dashboards, nil)
}

// GetDashboardTemplates retrieves available dashboard templates
func (h *DashboardAPIHandlers) GetDashboardTemplates(w http.ResponseWriter, r *http.Request) {
	// For now, return some default templates
	templates := []models.DashboardTemplate{
		{
			ID:          "template-analytics",
			Name:        "Analytics Dashboard",
			Description: "Comprehensive analytics with charts and KPIs",
			Category:    "Analytics",
			IsDefault:   true,
			Widgets: []models.DashboardWidget{
				{
					Type:   "kpi-card",
					Title:  "Total Revenue",
					Size:   models.WidgetSize{Width: 2, Height: 2},
					Config: map[string]interface{}{"metric": "revenue", "format": "currency"},
				},
				{
					Type:   "bar-chart",
					Title:  "Monthly Sales",
					Size:   models.WidgetSize{Width: 4, Height: 3},
					Config: map[string]interface{}{"dataSource": "sales", "xAxis": "month", "yAxis": "amount"},
				},
			},
		},
		{
			ID:          "template-portfolio",
			Name:        "Portfolio Dashboard",
			Description: "Portfolio performance and risk analysis",
			Category:    "Finance",
			IsDefault:   true,
			Widgets: []models.DashboardWidget{
				{
					Type:   "portfolio-summary",
					Title:  "Portfolio Overview",
					Size:   models.WidgetSize{Width: 4, Height: 3},
					Config: map[string]interface{}{"portfolioId": "main", "showReturns": true, "showRisk": true},
				},
				{
					Type:   "line-chart",
					Title:  "Portfolio Value Over Time",
					Size:   models.WidgetSize{Width: 4, Height: 3},
					Config: map[string]interface{}{"dataSource": "portfolio", "xAxis": "date", "yAxis": "value"},
				},
			},
		},
	}

	respond(w, r, templates, nil)
}

// DuplicateDashboard creates a copy of an existing dashboard
func (h *DashboardAPIHandlers) DuplicateDashboard(w http.ResponseWriter, r *http.Request) {
	dashboardID := chi.URLParam(r, "id")
	if dashboardID == "" {
		respondWithError(w, http.StatusBadRequest, "Dashboard ID is required")
		return
	}

	var requestBody struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Get original dashboard
	query := `
		SELECT name, description, widgets, layout, theme, is_public, created_by
		FROM dashboards WHERE id = $1
	`

	var original models.Dashboard
	var widgetsJSON []byte

	err := h.db.QueryRowContext(r.Context(), query, dashboardID).Scan(
		&original.Name, &original.Description, &widgetsJSON,
		&original.Layout, &original.Theme, &original.IsPublic, &original.CreatedBy,
	)

	if err != nil {
		respondWithError(w, http.StatusNotFound, "Original dashboard not found")
		return
	}

	json.Unmarshal(widgetsJSON, &original.Widgets)

	// Create duplicate
	newDashboard := models.Dashboard{
		ID:          generateID(),
		Name:        requestBody.Name,
		Description: original.Description,
		Widgets:     original.Widgets,
		Layout:      original.Layout,
		Theme:       original.Theme,
		IsPublic:    false, // Duplicates are private by default
		CreatedBy:   original.CreatedBy,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	newWidgetsJSON, _ := json.Marshal(newDashboard.Widgets)

	insertQuery := `
		INSERT INTO dashboards (
			id, name, description, widgets, layout, theme, is_public,
			created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err = h.db.ExecContext(r.Context(), insertQuery,
		newDashboard.ID, newDashboard.Name, newDashboard.Description, newWidgetsJSON,
		newDashboard.Layout, newDashboard.Theme, newDashboard.IsPublic,
		newDashboard.CreatedBy, newDashboard.CreatedAt, newDashboard.UpdatedAt,
	)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to duplicate dashboard")
		return
	}

	respond(w, r, newDashboard, nil)
}

// Helper function to generate unique IDs
func generateID() string {
	return "dashboard-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
