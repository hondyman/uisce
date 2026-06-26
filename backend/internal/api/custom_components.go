package api

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CustomComponent represents a custom component configuration
type CustomComponent struct {
	ID           string                 `json:"id"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // web_component, iframe, api_integration, custom_widget, chart, custom_code
	Config       map[string]interface{} `json:"config"`
	Events       []ComponentEvent       `json:"events"`
	Filters      []ComponentFilter      `json:"filters"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	CreatedBy    *string                `json:"created_by,omitempty"`
	UpdatedBy    *string                `json:"updated_by,omitempty"`
	IsActive     bool                   `json:"is_active"`
	Description  *string                `json:"description,omitempty"`
}

type ComponentEvent struct {
	ID                string `json:"id"`
	EventName         string `json:"event_name"`
	Action            string `json:"action"` // refresh, filter, navigate, custom
	TargetComponentID string `json:"target_component_id,omitempty"`
	CustomScript      string `json:"custom_script,omitempty"`
}

type ComponentFilter struct {
	ID                string `json:"id"`
	Field             string `json:"field"`
	Operator          string `json:"operator"` // equals, contains, in, between
	ListenToComponent string `json:"listen_to_component,omitempty"`
}

// registerCustomComponentRoutes registers all custom component routes
func (s *Server) registerCustomComponentRoutes(r chi.Router) {
	r.Get("/custom-components", s.listCustomComponents)
	r.Post("/custom-components", s.createCustomComponent)
	r.Get("/custom-components/{id}", s.getCustomComponent)
	r.Put("/custom-components/{id}", s.updateCustomComponent)
	r.Delete("/custom-components/{id}", s.deleteCustomComponent)
	r.Post("/custom-components/test-api", s.testComponentAPI)
	r.Get("/custom-components/export", s.exportComponents)
	r.Post("/custom-components/import", s.importComponents)
}

// listCustomComponents lists all custom components for a tenant/datasource
func (s *Server) listCustomComponents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, name, type, config, events, filters,
		       created_at, updated_at, created_by, updated_by, is_active, description
		FROM custom_components
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY created_at DESC`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID, datasourceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to query components", "db_error", err.Error())
		return
	}
	defer rows.Close()

	var components []CustomComponent
	for rows.Next() {
		var comp CustomComponent
		var configJSON, eventsJSON, filtersJSON []byte
		var createdBy, updatedBy, description sql.NullString

		err := rows.Scan(
			&comp.ID, &comp.TenantID, &comp.DatasourceID, &comp.Name, &comp.Type,
			&configJSON, &eventsJSON, &filtersJSON,
			&comp.CreatedAt, &comp.UpdatedAt, &createdBy, &updatedBy,
			&comp.IsActive, &description,
		)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to scan component", "scan_error", err.Error())
			return
		}

		// Parse JSON fields
		comp.Config = make(map[string]interface{})
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &comp.Config)
		}

		comp.Events = []ComponentEvent{}
		if len(eventsJSON) > 0 {
			json.Unmarshal(eventsJSON, &comp.Events)
		}

		comp.Filters = []ComponentFilter{}
		if len(filtersJSON) > 0 {
			json.Unmarshal(filtersJSON, &comp.Filters)
		}

		if createdBy.Valid {
			comp.CreatedBy = &createdBy.String
		} else {
			comp.CreatedBy = nil
		}
		if updatedBy.Valid {
			comp.UpdatedBy = &updatedBy.String
		} else {
			comp.UpdatedBy = nil
		}
		if description.Valid {
			comp.Description = &description.String
		} else {
			comp.Description = nil
		}

		components = append(components, comp)
	}

	if components == nil {
		components = []CustomComponent{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(components)
}

// createCustomComponent creates a new custom component
func (s *Server) createCustomComponent(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	var component CustomComponent
	if err := json.NewDecoder(r.Body).Decode(&component); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "json_error", err.Error())
		return
	}

	// Validate required fields
	if component.Name == "" || component.Type == "" {
		writeJSONError(w, http.StatusBadRequest, "name and type are required", "validation_error", "")
		return
	}

	// Set server-controlled fields
	component.TenantID = tenantID
	component.DatasourceID = datasourceID
	component.CreatedAt = time.Now()
	component.UpdatedAt = time.Now()
	component.IsActive = true

	// Marshal JSON fields
	configJSON, _ := json.Marshal(component.Config)
	eventsJSON, _ := json.Marshal(component.Events)
	filtersJSON, _ := json.Marshal(component.Filters)

	// Insert into database
	query := `
		INSERT INTO custom_components
		(tenant_id, datasource_id, name, type, config, events, filters, created_at, updated_at, is_active, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`

	var id string
	err := s.DB.QueryRowContext(r.Context(), query,
		tenantID, datasourceID, component.Name, component.Type,
		configJSON, eventsJSON, filtersJSON,
		component.CreatedAt, component.UpdatedAt, component.IsActive, component.Description,
	).Scan(&id)

	if err != nil {
		if strings.Contains(err.Error(), "unique violation") {
			writeJSONError(w, http.StatusConflict, "Component name already exists in this scope", "duplicate_name", "")
		} else {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create component", "db_error", err.Error())
		}
		return
	}

	component.ID = id
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(component)
}

// getCustomComponent retrieves a single custom component
func (s *Server) getCustomComponent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	query := `
		SELECT id, tenant_id, datasource_id, name, type, config, events, filters,
		       created_at, updated_at, created_by, updated_by, is_active, description
		FROM custom_components
		WHERE id = $1 AND tenant_id = $2 AND datasource_id = $3 AND is_active = true`

	var component CustomComponent
	var configJSON, eventsJSON, filtersJSON []byte

	var createdBy, updatedBy, description sql.NullString
	err := s.DB.QueryRowContext(r.Context(), query, id, tenantID, datasourceID).Scan(
		&component.ID, &component.TenantID, &component.DatasourceID, &component.Name, &component.Type,
		&configJSON, &eventsJSON, &filtersJSON,
		&component.CreatedAt, &component.UpdatedAt, &createdBy, &updatedBy,
		&component.IsActive, &description,
	)

	if err == sql.ErrNoRows {
		writeJSONError(w, http.StatusNotFound, "Component not found", "not_found", "")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to retrieve component", "db_error", err.Error())
		return
	}

	// Parse JSON fields
	component.Config = make(map[string]interface{})
	if len(configJSON) > 0 {
		json.Unmarshal(configJSON, &component.Config)
	}

	component.Events = []ComponentEvent{}
	if len(eventsJSON) > 0 {
		json.Unmarshal(eventsJSON, &component.Events)
	}

	component.Filters = []ComponentFilter{}
	if len(filtersJSON) > 0 {
		json.Unmarshal(filtersJSON, &component.Filters)
	}

	if createdBy.Valid {
		component.CreatedBy = &createdBy.String
	} else {
		component.CreatedBy = nil
	}
	if updatedBy.Valid {
		component.UpdatedBy = &updatedBy.String
	} else {
		component.UpdatedBy = nil
	}
	if description.Valid {
		component.Description = &description.String
	} else {
		component.Description = nil
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(component)
}

// updateCustomComponent updates an existing custom component
func (s *Server) updateCustomComponent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	var component CustomComponent
	if err := json.NewDecoder(r.Body).Decode(&component); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "json_error", err.Error())
		return
	}

	component.ID = id
	component.TenantID = tenantID
	component.DatasourceID = datasourceID
	component.UpdatedAt = time.Now()

	// Marshal JSON fields
	configJSON, _ := json.Marshal(component.Config)
	eventsJSON, _ := json.Marshal(component.Events)
	filtersJSON, _ := json.Marshal(component.Filters)

	query := `
		UPDATE custom_components
		SET name = $1, type = $2, config = $3, events = $4, filters = $5,
		    updated_at = $6, description = $7
		WHERE id = $8 AND tenant_id = $9 AND datasource_id = $10 AND is_active = true`

	result, err := s.DB.ExecContext(r.Context(), query,
		component.Name, component.Type, configJSON, eventsJSON, filtersJSON,
		component.UpdatedAt, component.Description,
		id, tenantID, datasourceID,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to update component", "db_error", err.Error())
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get affected rows", "db_error", err.Error())
		return
	}

	if rows == 0 {
		writeJSONError(w, http.StatusNotFound, "Component not found", "not_found", "")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(component)
}

// deleteCustomComponent soft-deletes a custom component
func (s *Server) deleteCustomComponent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	query := `
		UPDATE custom_components
		SET is_active = false, updated_at = $1
		WHERE id = $2 AND tenant_id = $3 AND datasource_id = $4`

	result, err := s.DB.ExecContext(r.Context(), query,
		time.Now(), id, tenantID, datasourceID,
	)

	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete component", "db_error", err.Error())
		return
	}

	rows, err := result.RowsAffected()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to get affected rows", "db_error", err.Error())
		return
	}

	if rows == 0 {
		writeJSONError(w, http.StatusNotFound, "Component not found", "not_found", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// testComponentAPI tests connectivity to an API component
func (s *Server) testComponentAPI(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	var testRequest struct {
		URL     string                 `json:"url"`
		Method  string                 `json:"method"`
		Headers map[string]string      `json:"headers"`
		Body    map[string]interface{} `json:"body"`
	}

	if err := json.NewDecoder(r.Body).Decode(&testRequest); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body", "json_error", err.Error())
		return
	}

	if testRequest.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "URL is required", "validation_error", "")
		return
	}

	if testRequest.Method == "" {
		testRequest.Method = "GET"
	}

	// Make test request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Build request
	var reqBody io.Reader
	if len(testRequest.Body) > 0 {
		bodyJSON, _ := json.Marshal(testRequest.Body)
		reqBody = bytes.NewReader(bodyJSON)
	}

	req, err := http.NewRequestWithContext(r.Context(), testRequest.Method, testRequest.URL, reqBody)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid URL or request", "request_error", err.Error())
		return
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range testRequest.Headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		writeJSONError(w, http.StatusGatewayTimeout, "API test failed", "connection_error", err.Error())
		return
	}
	defer resp.Body.Close()

	// Read response
	body, _ := io.ReadAll(resp.Body)

	result := map[string]interface{}{
		"status_code": resp.StatusCode,
		"status":      resp.Status,
		"headers":     resp.Header,
		"body":        string(body),
		"success":     resp.StatusCode >= 200 && resp.StatusCode < 300,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// exportComponents exports all components as JSON or ZIP
func (s *Server) exportComponents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	format := r.URL.Query().Get("format") // "json" or "zip"

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	// Query all components
	query := `
		SELECT id, tenant_id, datasource_id, name, type, config, events, filters,
		       created_at, updated_at, created_by, updated_by, is_active, description
		FROM custom_components
		WHERE tenant_id = $1 AND datasource_id = $2 AND is_active = true
		ORDER BY name`

	rows, err := s.DB.QueryContext(r.Context(), query, tenantID, datasourceID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to query components", "db_error", err.Error())
		return
	}
	defer rows.Close()

	var components []CustomComponent
	for rows.Next() {
		var comp CustomComponent
		var configJSON, eventsJSON, filtersJSON []byte
		var createdBy, updatedBy, description sql.NullString

		err := rows.Scan(
			&comp.ID, &comp.TenantID, &comp.DatasourceID, &comp.Name, &comp.Type,
			&configJSON, &eventsJSON, &filtersJSON,
			&comp.CreatedAt, &comp.UpdatedAt, &createdBy, &updatedBy,
			&comp.IsActive, &description,
		)
		if err != nil {
			continue
		}

		// Parse JSON fields
		comp.Config = make(map[string]interface{})
		if len(configJSON) > 0 {
			json.Unmarshal(configJSON, &comp.Config)
		}

		comp.Events = []ComponentEvent{}
		if len(eventsJSON) > 0 {
			json.Unmarshal(eventsJSON, &comp.Events)
		}

		comp.Filters = []ComponentFilter{}
		if len(filtersJSON) > 0 {
			json.Unmarshal(filtersJSON, &comp.Filters)
		}

		if createdBy.Valid {
			comp.CreatedBy = &createdBy.String
		} else {
			comp.CreatedBy = nil
		}
		if updatedBy.Valid {
			comp.UpdatedBy = &updatedBy.String
		} else {
			comp.UpdatedBy = nil
		}
		if description.Valid {
			comp.Description = &description.String
		} else {
			comp.Description = nil
		}

		components = append(components, comp)
	}

	if format == "zip" {
		// Create ZIP file
		zipBuf := new(bytes.Buffer)
		zipWriter := zip.NewWriter(zipBuf)

		for _, comp := range components {
			compJSON, _ := json.MarshalIndent(comp, "", "  ")
			fileName := fmt.Sprintf("%s.json", comp.Name)

			w, _ := zipWriter.Create(fileName)
			w.Write(compJSON)
		}

		zipWriter.Close()

		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=components-%s.zip", datasourceID))
		w.Write(zipBuf.Bytes())
	} else {
		// JSON format
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=components-%s.json", datasourceID))
		json.NewEncoder(w).Encode(components)
	}
}

// importComponents imports components from JSON file
func (s *Server) importComponents(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	if tenantID == "" || datasourceID == "" {
		writeJSONError(w, http.StatusBadRequest, "tenant_id and datasource_id are required", "missing_params", "")
		return
	}

	// Verify tenant context headers
	headerTenantID := jwtmiddleware.GetClaimsFromContext(r).TenantID
	headerDatasourceID := r.Header.Get("X-Tenant-Datasource-ID")

	if headerTenantID != tenantID || headerDatasourceID != datasourceID {
		writeJSONError(w, http.StatusForbidden, "Tenant context mismatch", "context_mismatch", "")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		writeJSONError(w, http.StatusBadRequest, "Failed to parse form", "parse_error", err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "No file provided", "upload_error", err.Error())
		return
	}
	defer file.Close()

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read file", "read_error", err.Error())
		return
	}

	// Parse JSON
	var components []CustomComponent
	if err := json.Unmarshal(fileContent, &components); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid JSON file", "json_error", err.Error())
		return
	}

	// Import components
	var imported []CustomComponent
	for _, comp := range components {
		comp.ID = "" // Clear ID to get new one
		comp.TenantID = tenantID
		comp.DatasourceID = datasourceID
		comp.CreatedAt = time.Now()
		comp.UpdatedAt = time.Now()
		comp.IsActive = true

		configJSON, _ := json.Marshal(comp.Config)
		eventsJSON, _ := json.Marshal(comp.Events)
		filtersJSON, _ := json.Marshal(comp.Filters)

		query := `
			INSERT INTO custom_components
			(tenant_id, datasource_id, name, type, config, events, filters, created_at, updated_at, is_active, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING id`

		var newID string
		err := s.DB.QueryRowContext(r.Context(), query,
			tenantID, datasourceID, comp.Name, comp.Type,
			configJSON, eventsJSON, filtersJSON,
			comp.CreatedAt, comp.UpdatedAt, comp.IsActive, comp.Description,
		).Scan(&newID)

		if err != nil {
			// Skip duplicates or errors, continue importing others
			continue
		}

		comp.ID = newID
		imported = append(imported, comp)
	}

	response := map[string]interface{}{
		"imported":   len(imported),
		"total":      len(components),
		"components": imported,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}
