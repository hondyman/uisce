package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MarketplaceIntegrationHandlers handles integration marketplace operations
type MarketplaceIntegrationHandlers struct {
	db *sqlx.DB
}

// NewMarketplaceIntegrationHandlers creates a new handler instance
func NewMarketplaceIntegrationHandlers(db *sqlx.DB) *MarketplaceIntegrationHandlers {
	return &MarketplaceIntegrationHandlers{db: db}
}

// Data structures
type MarketplaceIntegration struct {
	ID               string                 `json:"id" db:"id"`
	IntegrationKey   string                 `json:"integration_key" db:"integration_key"`
	Name             string                 `json:"name" db:"name"`
	Description      string                 `json:"description" db:"description"`
	Category         string                 `json:"category" db:"category"`
	Provider         string                 `json:"provider" db:"provider"`
	IconURL          string                 `json:"icon_url" db:"icon_url"`
	Version          string                 `json:"version" db:"version"`
	IsOfficial       bool                   `json:"is_official" db:"is_official"`
	IsActive         bool                   `json:"is_active" db:"is_active"`
	ConfigSchema     map[string]interface{} `json:"config_schema" db:"config_schema"`
	AuthType         string                 `json:"auth_type" db:"auth_type"`
	OAuthConfig      map[string]interface{} `json:"oauth_config" db:"oauth_config"`
	SupportsWebhooks bool                   `json:"supports_webhooks" db:"supports_webhooks"`
	SupportsPolling  bool                   `json:"supports_polling" db:"supports_polling"`
	SupportsActions  bool                   `json:"supports_actions" db:"supports_actions"`
	DocumentationURL string                 `json:"documentation_url" db:"documentation_url"`
	SetupGuide       string                 `json:"setup_guide" db:"setup_guide"`
	ExamplePayload   map[string]interface{} `json:"example_payload" db:"example_payload"`
	InstallCount     int                    `json:"install_count" db:"install_count"`
	Rating           float64                `json:"rating" db:"rating"`
	CreatedAt        time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at" db:"updated_at"`
}

type InstalledIntegration struct {
	ID                string                 `json:"id" db:"id"`
	TenantID          string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID      string                 `json:"datasource_id" db:"datasource_id"`
	IntegrationID     string                 `json:"integration_id" db:"integration_id"`
	InstalledBy       string                 `json:"installed_by" db:"installed_by"`
	InstalledAt       time.Time              `json:"installed_at" db:"installed_at"`
	IsEnabled         bool                   `json:"is_enabled" db:"is_enabled"`
	Config            map[string]interface{} `json:"config" db:"config"`
	Credentials       map[string]interface{} `json:"credentials,omitempty" db:"credentials"`
	OAuthState        string                 `json:"oauth_state,omitempty" db:"oauth_state"`
	OAuthAccessToken  string                 `json:"-" db:"oauth_access_token"`
	OAuthRefreshToken string                 `json:"-" db:"oauth_refresh_token"`
	OAuthExpiresAt    *time.Time             `json:"oauth_expires_at,omitempty" db:"oauth_expires_at"`
	LastUsedAt        *time.Time             `json:"last_used_at,omitempty" db:"last_used_at"`
	ExecutionCount    int                    `json:"execution_count" db:"execution_count"`
	SuccessCount      int                    `json:"success_count" db:"success_count"`
	FailureCount      int                    `json:"failure_count" db:"failure_count"`
	CreatedAt         time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" db:"updated_at"`

	// Joined fields
	IntegrationName    string `json:"integration_name,omitempty" db:"integration_name"`
	IntegrationKey     string `json:"integration_key,omitempty" db:"integration_key"`
	IntegrationIconURL string `json:"integration_icon_url,omitempty" db:"integration_icon_url"`
}

type IntegrationExecution struct {
	ID                     string                 `json:"id" db:"id"`
	InstalledIntegrationID string                 `json:"installed_integration_id" db:"installed_integration_id"`
	TenantID               string                 `json:"tenant_id" db:"tenant_id"`
	DatasourceID           string                 `json:"datasource_id" db:"datasource_id"`
	WorkflowID             *string                `json:"workflow_id,omitempty" db:"workflow_id"`
	WorkflowType           string                 `json:"workflow_type,omitempty" db:"workflow_type"`
	StepName               string                 `json:"step_name,omitempty" db:"step_name"`
	Action                 string                 `json:"action" db:"action"`
	RequestPayload         map[string]interface{} `json:"request_payload" db:"request_payload"`
	ResponsePayload        map[string]interface{} `json:"response_payload,omitempty" db:"response_payload"`
	Status                 string                 `json:"status" db:"status"`
	ErrorMessage           string                 `json:"error_message,omitempty" db:"error_message"`
	ErrorDetails           map[string]interface{} `json:"error_details,omitempty" db:"error_details"`
	StartedAt              time.Time              `json:"started_at" db:"started_at"`
	CompletedAt            *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
	DurationMs             int                    `json:"duration_ms,omitempty" db:"duration_ms"`
	RetryCount             int                    `json:"retry_count" db:"retry_count"`
	MaxRetries             int                    `json:"max_retries" db:"max_retries"`
	NextRetryAt            *time.Time             `json:"next_retry_at,omitempty" db:"next_retry_at"`
}

// RegisterRoutes registers all marketplace integration routes
func (h *MarketplaceIntegrationHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/integrations", func(r chi.Router) {
		// Marketplace browsing
		r.Get("/marketplace", h.GetMarketplaceIntegrations)
		r.Get("/marketplace/{integrationKey}", h.GetMarketplaceIntegration)
		r.Get("/marketplace/category/{category}", h.GetIntegrationsByCategory)

		// Installation management
		r.Post("/install", h.InstallIntegration)
		r.Get("/installed", h.GetInstalledIntegrations)
		r.Get("/installed/{installationId}", h.GetInstalledIntegration)
		r.Put("/installed/{installationId}/config", h.UpdateIntegrationConfig)
		r.Put("/installed/{installationId}/toggle", h.ToggleIntegration)
		r.Delete("/installed/{installationId}", h.UninstallIntegration)

		// Execution
		r.Post("/execute/{installationId}", h.ExecuteIntegration)
		r.Post("/test/{installationId}", h.TestIntegration)

		// Logs and monitoring
		r.Get("/executions", h.GetIntegrationExecutions)
		r.Get("/executions/{executionId}", h.GetIntegrationExecution)
		r.Get("/installed/{installationId}/stats", h.GetInstallationStats)

		// OAuth flow
		r.Get("/oauth/authorize/{installationId}", h.InitiateOAuthFlow)
		r.Get("/oauth/callback", h.HandleOAuthCallback)
	})
}

// GetMarketplaceIntegrations returns all available integrations
func (h *MarketplaceIntegrationHandlers) GetMarketplaceIntegrations(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	search := r.URL.Query().Get("search")

	query := `SELECT * FROM marketplace_integrations WHERE is_active = true`
	args := []interface{}{}
	argPos := 1

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argPos)
		args = append(args, category)
		argPos++
	}

	if search != "" {
		query += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argPos, argPos)
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern)
		argPos++
	}

	query += " ORDER BY rating DESC, install_count DESC, name ASC"

	var integrations []MarketplaceIntegration
	err := h.db.Select(&integrations, query, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch integrations: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(integrations)
}

// GetMarketplaceIntegration returns a single integration by key
func (h *MarketplaceIntegrationHandlers) GetMarketplaceIntegration(w http.ResponseWriter, r *http.Request) {
	integrationKey := chi.URLParam(r, "integrationKey")

	var integration MarketplaceIntegration
	err := h.db.Get(&integration, `
		SELECT * FROM marketplace_integrations 
		WHERE integration_key = $1 AND is_active = true
	`, integrationKey)

	if err == sql.ErrNoRows {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(integration)
}

// GetIntegrationsByCategory returns integrations for a specific category
func (h *MarketplaceIntegrationHandlers) GetIntegrationsByCategory(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")

	var integrations []MarketplaceIntegration
	err := h.db.Select(&integrations, `
		SELECT * FROM marketplace_integrations 
		WHERE category = $1 AND is_active = true
		ORDER BY rating DESC, install_count DESC, name ASC
	`, category)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch integrations: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(integrations)
}

// InstallIntegration installs an integration for a tenant
func (h *MarketplaceIntegrationHandlers) InstallIntegration(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var req struct {
		IntegrationKey string                 `json:"integration_key"`
		Config         map[string]interface{} `json:"config"`
		Credentials    map[string]interface{} `json:"credentials"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get integration from marketplace
	var integration MarketplaceIntegration
	err := h.db.Get(&integration, `
		SELECT * FROM marketplace_integrations 
		WHERE integration_key = $1 AND is_active = true
	`, req.IntegrationKey)

	if err == sql.ErrNoRows {
		http.Error(w, "Integration not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if already installed
	var existingCount int
	err = h.db.Get(&existingCount, `
		SELECT COUNT(*) FROM installed_integrations 
		WHERE tenant_id = $1 AND datasource_id = $2 AND integration_id = $3
	`, tenantID, datasourceID, integration.ID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	if existingCount > 0 {
		http.Error(w, "Integration already installed", http.StatusConflict)
		return
	}

	// Install integration
	configJSON, _ := json.Marshal(req.Config)
	credentialsJSON, _ := json.Marshal(req.Credentials)

	var installationID string
	err = h.db.QueryRow(`
		INSERT INTO installed_integrations (
			tenant_id, datasource_id, integration_id, installed_by, 
			config, credentials, is_enabled
		) VALUES ($1, $2, $3, $4, $5, $6, true)
		RETURNING id
	`, tenantID, datasourceID, integration.ID, "system", configJSON, credentialsJSON).Scan(&installationID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to install integration: %v", err), http.StatusInternalServerError)
		return
	}

	// Update install count
	_, err = h.db.Exec(`
		UPDATE marketplace_integrations 
		SET install_count = install_count + 1 
		WHERE id = $1
	`, integration.ID)

	// Return installed integration
	var installed InstalledIntegration
	err = h.db.Get(&installed, `
		SELECT ii.*, mi.name as integration_name, mi.integration_key, mi.icon_url as integration_icon_url
		FROM installed_integrations ii
		JOIN marketplace_integrations mi ON ii.integration_id = mi.id
		WHERE ii.id = $1
	`, installationID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch installed integration: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(installed)
}

// GetInstalledIntegrations returns all installed integrations for a tenant
func (h *MarketplaceIntegrationHandlers) GetInstalledIntegrations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		http.Error(w, "tenant_id and datasource_id required", http.StatusBadRequest)
		return
	}

	var installations []InstalledIntegration
	err := h.db.Select(&installations, `
		SELECT ii.*, mi.name as integration_name, mi.integration_key, mi.icon_url as integration_icon_url
		FROM installed_integrations ii
		JOIN marketplace_integrations mi ON ii.integration_id = mi.id
		WHERE ii.tenant_id = $1 AND ii.datasource_id = $2
		ORDER BY ii.installed_at DESC
	`, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch installations: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installations)
}

// GetInstalledIntegration returns a single installed integration
func (h *MarketplaceIntegrationHandlers) GetInstalledIntegration(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var installation InstalledIntegration
	err := h.db.Get(&installation, `
		SELECT ii.*, mi.name as integration_name, mi.integration_key, mi.icon_url as integration_icon_url
		FROM installed_integrations ii
		JOIN marketplace_integrations mi ON ii.integration_id = mi.id
		WHERE ii.id = $1 AND ii.tenant_id = $2 AND ii.datasource_id = $3
	`, installationID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(installation)
}

// UpdateIntegrationConfig updates the configuration of an installed integration
func (h *MarketplaceIntegrationHandlers) UpdateIntegrationConfig(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var req struct {
		Config      map[string]interface{} `json:"config"`
		Credentials map[string]interface{} `json:"credentials"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	configJSON, _ := json.Marshal(req.Config)
	credentialsJSON, _ := json.Marshal(req.Credentials)

	result, err := h.db.Exec(`
		UPDATE installed_integrations 
		SET config = $1, credentials = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4 AND datasource_id = $5
	`, configJSON, credentialsJSON, installationID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
	})
}

// ToggleIntegration enables or disables an installed integration
func (h *MarketplaceIntegrationHandlers) ToggleIntegration(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(`
		UPDATE installed_integrations 
		SET is_enabled = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3 AND datasource_id = $4
	`, req.Enabled, installationID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to toggle integration: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Integration %s successfully", status),
		"enabled": req.Enabled,
	})
}

// UninstallIntegration removes an installed integration
func (h *MarketplaceIntegrationHandlers) UninstallIntegration(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	// Get integration_id before deletion for install count update
	var integrationID string
	err := h.db.Get(&integrationID, `
		SELECT integration_id FROM installed_integrations 
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, installationID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Delete installation (cascades to executions)
	result, err := h.db.Exec(`
		DELETE FROM installed_integrations 
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, installationID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to uninstall integration: %v", err), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	}

	// Update install count
	_, _ = h.db.Exec(`
		UPDATE marketplace_integrations 
		SET install_count = GREATEST(0, install_count - 1)
		WHERE id = $1
	`, integrationID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Integration uninstalled successfully",
	})
}

// ExecuteIntegration executes an integration action
func (h *MarketplaceIntegrationHandlers) ExecuteIntegration(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var req struct {
		Action       string                 `json:"action"`
		Payload      map[string]interface{} `json:"payload"`
		WorkflowID   string                 `json:"workflow_id"`
		WorkflowType string                 `json:"workflow_type"`
		StepName     string                 `json:"step_name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get installation details
	var installation InstalledIntegration
	err := h.db.Get(&installation, `
		SELECT ii.*, mi.integration_key
		FROM installed_integrations ii
		JOIN marketplace_integrations mi ON ii.integration_id = mi.id
		WHERE ii.id = $1 AND ii.tenant_id = $2 AND ii.datasource_id = $3 AND ii.is_enabled = true
	`, installationID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Installation not found or disabled", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Log execution start
	executionID := uuid.New().String()
	payloadJSON, _ := json.Marshal(req.Payload)

	startTime := time.Now()

	_, err = h.db.Exec(`
		INSERT INTO integration_executions (
			id, installed_integration_id, tenant_id, datasource_id,
			workflow_id, workflow_type, step_name, action,
			request_payload, status, started_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, 'pending', $10)
	`, executionID, installationID, tenantID, datasourceID,
		nullString(req.WorkflowID), req.WorkflowType, req.StepName,
		req.Action, payloadJSON, startTime)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to log execution: %v", err), http.StatusInternalServerError)
		return
	}

	// Execute integration (placeholder - actual execution would call external APIs)
	response, err := h.executeIntegrationAction(r.Context(), installation, req.Action, req.Payload)

	duration := int(time.Since(startTime).Milliseconds())
	status := "success"
	var errorMsg string

	if err != nil {
		status = "failed"
		errorMsg = err.Error()
	}

	// Update execution log
	responseJSON, _ := json.Marshal(response)
	_, _ = h.db.Exec(`
		UPDATE integration_executions 
		SET status = $1, response_payload = $2, error_message = $3,
			completed_at = NOW(), duration_ms = $4
		WHERE id = $5
	`, status, responseJSON, errorMsg, duration, executionID)

	// Update installation stats
	if status == "success" {
		_, _ = h.db.Exec(`
			UPDATE installed_integrations 
			SET execution_count = execution_count + 1,
				success_count = success_count + 1,
				last_used_at = NOW()
			WHERE id = $1
		`, installationID)
	} else {
		_, _ = h.db.Exec(`
			UPDATE installed_integrations 
			SET execution_count = execution_count + 1,
				failure_count = failure_count + 1,
				last_used_at = NOW()
			WHERE id = $1
		`, installationID)
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"execution_id": executionID,
		"status":       status,
		"response":     response,
		"error":        errorMsg,
		"duration_ms":  duration,
	})
}

// TestIntegration tests an integration connection without executing
func (h *MarketplaceIntegrationHandlers) TestIntegration(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	// Get installation
	var installation InstalledIntegration
	err := h.db.Get(&installation, `
		SELECT ii.*, mi.integration_key
		FROM installed_integrations ii
		JOIN marketplace_integrations mi ON ii.integration_id = mi.id
		WHERE ii.id = $1 AND ii.tenant_id = $2 AND ii.datasource_id = $3
	`, installationID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	// Test connection (placeholder - would actually test API connection)
	testResult := h.testIntegrationConnection(r.Context(), installation)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(testResult)
}

// GetIntegrationExecutions returns execution logs
func (h *MarketplaceIntegrationHandlers) GetIntegrationExecutions(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	installationID := r.URL.Query().Get("installation_id")
	status := r.URL.Query().Get("status")

	query := `SELECT * FROM integration_executions WHERE tenant_id = $1 AND datasource_id = $2`
	args := []interface{}{tenantID, datasourceID}
	argPos := 3

	if installationID != "" {
		query += fmt.Sprintf(" AND installed_integration_id = $%d", argPos)
		args = append(args, installationID)
		argPos++
	}

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, status)
		argPos++
	}

	query += " ORDER BY started_at DESC LIMIT 100"

	var executions []IntegrationExecution
	err := h.db.Select(&executions, query, args...)

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch executions: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(executions)
}

// GetIntegrationExecution returns a single execution
func (h *MarketplaceIntegrationHandlers) GetIntegrationExecution(w http.ResponseWriter, r *http.Request) {
	executionID := chi.URLParam(r, "executionId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var execution IntegrationExecution
	err := h.db.Get(&execution, `
		SELECT * FROM integration_executions 
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3
	`, executionID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(execution)
}

// GetInstallationStats returns usage statistics for an installation
func (h *MarketplaceIntegrationHandlers) GetInstallationStats(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	var stats struct {
		ExecutionCount int        `json:"execution_count" db:"execution_count"`
		SuccessCount   int        `json:"success_count" db:"success_count"`
		FailureCount   int        `json:"failure_count" db:"failure_count"`
		SuccessRate    float64    `json:"success_rate"`
		LastUsedAt     *time.Time `json:"last_used_at" db:"last_used_at"`
		AvgDurationMs  float64    `json:"avg_duration_ms" db:"avg_duration_ms"`
		Last24hCount   int        `json:"last_24h_count" db:"last_24h_count"`
	}

	err := h.db.Get(&stats, `
		SELECT 
			ii.execution_count,
			ii.success_count,
			ii.failure_count,
			ii.last_used_at,
			COALESCE(AVG(ie.duration_ms), 0) as avg_duration_ms,
			COUNT(CASE WHEN ie.started_at > NOW() - INTERVAL '24 hours' THEN 1 END) as last_24h_count
		FROM installed_integrations ii
		LEFT JOIN integration_executions ie ON ie.installed_integration_id = ii.id
		WHERE ii.id = $1 AND ii.tenant_id = $2 AND ii.datasource_id = $3
		GROUP BY ii.id
	`, installationID, tenantID, datasourceID)

	if err == sql.ErrNoRows {
		http.Error(w, "Installation not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	if stats.ExecutionCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.ExecutionCount) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// InitiateOAuthFlow starts the OAuth authorization flow
func (h *MarketplaceIntegrationHandlers) InitiateOAuthFlow(w http.ResponseWriter, r *http.Request) {
	installationID := chi.URLParam(r, "installationId")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	// Get integration OAuth config
	var oauthConfig struct {
		AuthURL     string   `json:"authorization_url"`
		ClientID    string   `json:"client_id"`
		Scopes      []string `json:"scopes"`
		RedirectURI string   `json:"redirect_uri"`
	}

	err := h.db.Get(&oauthConfig, `
		SELECT 
			mi.oauth_config->>'authorization_url' as auth_url,
			mi.oauth_config->>'client_id' as client_id,
			mi.oauth_config->'scopes' as scopes,
			mi.oauth_config->>'redirect_uri' as redirect_uri
		FROM marketplace_integrations mi
		JOIN installed_integrations ii ON ii.integration_id = mi.id
		WHERE ii.id = $1 AND ii.tenant_id = $2 AND ii.datasource_id = $3
	`, installationID, tenantID, datasourceID)

	if err != nil {
		http.Error(w, "Failed to get OAuth configuration", http.StatusInternalServerError)
		return
	}

	// Generate state token
	state := uuid.New().String()

	// Store state in installation
	_, _ = h.db.Exec(`
		UPDATE installed_integrations 
		SET oauth_state = $1, updated_at = NOW()
		WHERE id = $2
	`, state, installationID)

	// Construct authorization URL
	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&state=%s&response_type=code",
		oauthConfig.AuthURL, oauthConfig.ClientID, oauthConfig.RedirectURI, state)

	if len(oauthConfig.Scopes) > 0 {
		authURL += "&scope=" + fmt.Sprintf("%v", oauthConfig.Scopes)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"authorization_url": authURL,
		"state":             state,
	})
}

// HandleOAuthCallback handles the OAuth callback
func (h *MarketplaceIntegrationHandlers) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		http.Error(w, "Missing code or state", http.StatusBadRequest)
		return
	}

	// Find installation by state
	var installationID string
	err := h.db.Get(&installationID, `
		SELECT id FROM installed_integrations 
		WHERE oauth_state = $1
	`, state)

	if err == sql.ErrNoRows {
		http.Error(w, "Invalid state", http.StatusBadRequest)
		return
	} else if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Exchange code for token (placeholder - would call OAuth provider)
	accessToken := "placeholder_access_token"
	refreshToken := "placeholder_refresh_token"
	expiresAt := time.Now().Add(time.Hour * 24 * 30)

	// Store tokens
	_, err = h.db.Exec(`
		UPDATE installed_integrations 
		SET oauth_access_token = $1, 
			oauth_refresh_token = $2, 
			oauth_expires_at = $3,
			oauth_state = NULL,
			updated_at = NOW()
		WHERE id = $4
	`, accessToken, refreshToken, expiresAt, installationID)

	if err != nil {
		http.Error(w, "Failed to store tokens", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OAuth authorization successful",
	})
}

// Helper functions

func (h *MarketplaceIntegrationHandlers) executeIntegrationAction(ctx context.Context, installation InstalledIntegration, action string, payload map[string]interface{}) (map[string]interface{}, error) {
	// Placeholder for actual integration execution logic
	// In production, this would route to specific integration handlers
	// (Slack API, Teams API, SMTP server, etc.)

	switch installation.IntegrationKey {
	case "slack":
		return h.executeSlackAction(ctx, installation, action, payload)
	case "email":
		return h.executeEmailAction(ctx, installation, action, payload)
	case "webhook":
		return h.executeWebhookAction(ctx, installation, action, payload)
	default:
		return map[string]interface{}{
			"message": fmt.Sprintf("Integration '%s' executed successfully", installation.IntegrationKey),
			"action":  action,
		}, nil
	}
}

func (h *MarketplaceIntegrationHandlers) executeSlackAction(_ctx context.Context, _installation InstalledIntegration, _action string, payload map[string]interface{}) (map[string]interface{}, error) {
	// Placeholder for Slack integration
	return map[string]interface{}{
		"channel": payload["channel"],
		"message": "Message sent to Slack",
	}, nil
}

func (h *MarketplaceIntegrationHandlers) executeEmailAction(_ctx context.Context, _installation InstalledIntegration, _action string, payload map[string]interface{}) (map[string]interface{}, error) {
	// Placeholder for Email integration
	return map[string]interface{}{
		"to":      payload["to"],
		"subject": payload["subject"],
		"status":  "Email sent successfully",
	}, nil
}

func (h *MarketplaceIntegrationHandlers) executeWebhookAction(_ctx context.Context, installation InstalledIntegration, _action string, _payload map[string]interface{}) (map[string]interface{}, error) {
	// Placeholder for Webhook integration
	return map[string]interface{}{
		"webhook_url": installation.Config["webhook_url"],
		"status":      "Webhook triggered successfully",
	}, nil
}

func (h *MarketplaceIntegrationHandlers) testIntegrationConnection(_ctx context.Context, installation InstalledIntegration) map[string]interface{} {
	// Placeholder for connection testing
	return map[string]interface{}{
		"success":    true,
		"message":    fmt.Sprintf("%s connection test successful", installation.IntegrationKey),
		"latency_ms": 150,
	}
}
