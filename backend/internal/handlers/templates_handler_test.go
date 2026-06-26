package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Helper Functions
// ============================================================================

// wrapHandlerWithRouter wraps a handler to test it through the mux router
func wrapHandlerWithRouter(handler http.HandlerFunc, pattern string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		router := mux.NewRouter()
		router.HandleFunc(pattern, handler)
		router.ServeHTTP(w, r)
	}
}

func setupTestTemplateDB(t *testing.T) *sql.DB {
	// Create in-memory or test database connection
	// For now, use localhost postgres
	connStr := "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Test database not available: %v", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Test database connection failed: %v", err)
	}

	return db
}

func cleanupTestTemplates(t *testing.T, db *sql.DB, tenantID string) {
	_, err := db.Exec(`DELETE FROM edm.template_usage WHERE template_id IN (SELECT id FROM edm.rule_templates WHERE tenant_id = $1)`, tenantID)
	if err != nil {
		t.Logf("Cleanup template_usage failed: %v", err)
	}

	_, err = db.Exec(`DELETE FROM edm.rule_templates WHERE tenant_id = $1`, tenantID)
	if err != nil {
		t.Logf("Cleanup rule_templates failed: %v", err)
	}
}

func createTestTenantID() string {
	return uuid.New().String()
}

func createTestUserID() string {
	return uuid.New().String()
}

// ============================================================================
// Test Cases
// ============================================================================

// TestCreateTemplate tests template creation with valid parameters
func TestCreateTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	request := CreateTemplateRequest{
		BusinessObject: "calendar",
		Name:           "Weekend Override",
		Description:    "Override weekend classification",
		Category:       "weekend",
		BaseRuleSteps: []TemplateStep{
			{
				Priority: 1,
				Condition: map[string]interface{}{
					"semanticTerm": "IsBusinessDay",
					"operator":     "equals",
					"value":        false,
				},
				Action: map[string]interface{}{
					"useField":   "golden_record",
					"confidence": 90,
				},
				Description: "Check if not a business day",
			},
		},
		ParameterSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"regions": map[string]interface{}{
					"type":    "string",
					"pattern": "^[A-Z]{2}(,[A-Z]{2})*$",
				},
			},
			"required": []string{"regions"},
		},
		IsPublic: false,
	}

	body, _ := json.Marshal(request)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response RuleTemplate
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "calendar", response.BusinessObject)
	assert.Equal(t, "Weekend Override", response.Name)
	assert.Equal(t, "draft", response.Status)
	assert.Equal(t, tenantID, response.TenantID)
	assert.NotEmpty(t, response.ID)
}

// TestCreateTemplateValidation tests parameter validation
func TestCreateTemplateValidation(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()

	handler := &TemplateHandler{db: db}

	testCases := []struct {
		name       string
		request    CreateTemplateRequest
		statusCode int
	}{
		{
			name: "Missing business object",
			request: CreateTemplateRequest{
				Name:            "Test",
				Category:        "test",
				IsPublic:        false,
				BaseRuleSteps:   []TemplateStep{},
				ParameterSchema: map[string]interface{}{},
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "Missing name",
			request: CreateTemplateRequest{
				BusinessObject:  "calendar",
				Category:        "test",
				IsPublic:        false,
				BaseRuleSteps:   []TemplateStep{},
				ParameterSchema: map[string]interface{}{},
			},
			statusCode: http.StatusBadRequest,
		},
		{
			name: "Empty name",
			request: CreateTemplateRequest{
				BusinessObject:  "calendar",
				Name:            "",
				Category:        "test",
				IsPublic:        false,
				BaseRuleSteps:   []TemplateStep{},
				ParameterSchema: map[string]interface{}{},
			},
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.request)
			req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Tenant-ID", tenantID)
			req.Header.Set("X-User-ID", userID)

			w := httptest.NewRecorder()
			handler.CreateTemplate(w, req)

			assert.Equal(t, tc.statusCode, w.Code)
		})
	}
}

// TestGetTemplate tests fetching a template
func TestGetTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create a template first
	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Test Template",
		Description:     "Test",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Now fetch it
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/templates/%s", createdTemplate.ID), nil)
	getReq.Header.Set("X-Tenant-ID", tenantID)
	getReq.Header.Set("X-User-ID", userID)

	getW := httptest.NewRecorder()
	handler.GetTemplate(getW, getReq)

	assert.Equal(t, http.StatusOK, getW.Code)

	var fetchedTemplate RuleTemplate
	json.Unmarshal(getW.Body.Bytes(), &fetchedTemplate)

	assert.Equal(t, createdTemplate.ID, fetchedTemplate.ID)
	assert.Equal(t, "Test Template", fetchedTemplate.Name)
}

// TestGetTemplateNotFound tests 404 on non-existent template
func TestGetTemplateNotFound(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	handler := &TemplateHandler{db: db}

	fakeID := uuid.New().String()
	req := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/templates/%s", fakeID), nil)
	req.Header.Set("X-Tenant-ID", tenantID)

	w := httptest.NewRecorder()
	handler.GetTemplate(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestListTemplates tests template listing with pagination and filtering
func TestListTemplates(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create 3 templates
	for i := 1; i <= 3; i++ {
		createReq := CreateTemplateRequest{
			BusinessObject:  "calendar",
			Name:            fmt.Sprintf("Template %d", i),
			Description:     "Test",
			Category:        "weekend",
			BaseRuleSteps:   []TemplateStep{},
			ParameterSchema: map[string]interface{}{},
			IsPublic:        false,
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)

		w := httptest.NewRecorder()
		handler.CreateTemplate(w, req)
		require.Equal(t, http.StatusCreated, w.Code)
	}

	// List templates
	listReq := httptest.NewRequest("GET", "/api/v1/templates?businessObject=calendar", nil)
	listReq.Header.Set("X-Tenant-ID", tenantID)

	listW := httptest.NewRecorder()
	handler.ListTemplates(listW, listReq)

	assert.Equal(t, http.StatusOK, listW.Code)

	var templates []RuleTemplate
	json.Unmarshal(listW.Body.Bytes(), &templates)

	assert.GreaterOrEqual(t, len(templates), 3)
}

// TestUpdateTemplate tests template modification
func TestUpdateTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create a template
	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Original Name",
		Description:     "Original",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Update it
	updateReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Updated Name",
		Description:     "Updated description",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        true,
	}

	updateBody, _ := json.Marshal(updateReq)
	updateHTTPReq := httptest.NewRequest("PUT", fmt.Sprintf("/api/v1/templates/%s", createdTemplate.ID), bytes.NewReader(updateBody))
	updateHTTPReq.Header.Set("X-Tenant-ID", tenantID)
	updateHTTPReq.Header.Set("X-User-ID", userID)

	updateW := httptest.NewRecorder()
	handler.UpdateTemplate(updateW, updateHTTPReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	var updatedTemplate RuleTemplate
	json.Unmarshal(updateW.Body.Bytes(), &updatedTemplate)

	assert.Equal(t, "Updated Name", updatedTemplate.Name)
	assert.Equal(t, "Updated description", updatedTemplate.Description)
	assert.Equal(t, true, updatedTemplate.IsPublic)
}

// TestDeleteTemplate tests template deletion
func TestDeleteTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create a template
	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "To Delete",
		Description:     "Test",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Delete it
	deleteReq := httptest.NewRequest("DELETE", fmt.Sprintf("/api/v1/templates/%s", createdTemplate.ID), nil)
	deleteReq.Header.Set("X-Tenant-ID", tenantID)
	deleteReq.Header.Set("X-User-ID", userID)

	deleteW := httptest.NewRecorder()
	handler.DeleteTemplate(deleteW, deleteReq)

	assert.Equal(t, http.StatusOK, deleteW.Code)

	// Verify it's gone
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/api/v1/templates/%s", createdTemplate.ID), nil)
	getReq.Header.Set("X-Tenant-ID", tenantID)

	getW := httptest.NewRecorder()
	handler.GetTemplate(getW, getReq)

	assert.Equal(t, http.StatusNotFound, getW.Code)
}

// TestPreviewTemplate tests rule preview without creation
func TestPreviewTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create a template with parameters
	baseSteps := []TemplateStep{
		{
			Priority: 1,
			Condition: map[string]interface{}{
				"semanticTerm": "RegionCode",
				"operator":     "in",
				"value":        "{{regions}}",
			},
			Action: map[string]interface{}{
				"useField":   "region_system",
				"confidence": "{{confidence}}",
			},
			Description: "Region override",
		},
	}

	paramSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"regions": map[string]interface{}{
				"type": "string",
			},
			"confidence": map[string]interface{}{
				"type": "number",
			},
		},
	}

	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Region Override",
		Description:     "Test preview",
		Category:        "region",
		BaseRuleSteps:   baseSteps,
		ParameterSchema: paramSchema,
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Preview it
	previewBody := map[string]interface{}{
		"parameters": map[string]interface{}{
			"regions":    "US,GB",
			"confidence": 85,
		},
	}

	previewData, _ := json.Marshal(previewBody)
	previewReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/templates/%s/preview", createdTemplate.ID), bytes.NewReader(previewData))
	previewReq.Header.Set("X-Tenant-ID", tenantID)
	previewReq.Header.Set("X-User-ID", userID)

	previewW := httptest.NewRecorder()
	handler.GetTemplatePreview(previewW, previewReq)

	assert.Equal(t, http.StatusOK, previewW.Code)

	// Verify preview is returned correctly
	var preview map[string]interface{}
	json.Unmarshal(previewW.Body.Bytes(), &preview)

	assert.NotNil(t, preview["template"])
	assert.NotNil(t, preview["sampleParameters"])
	assert.NotNil(t, preview["previewSteps"])
}

// TestInstantiateTemplate tests creating a rule from template
func TestInstantiateTemplate(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(t, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create a template
	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Test Template",
		Description:     "Test instantiation",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenantID)
	req.Header.Set("X-User-ID", userID)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Instantiate a rule from it
	instantiateReq := InstantiateTemplateRequest{
		RuleName:   "My Rule from Template",
		Parameters: map[string]interface{}{},
	}

	instantiateBody, _ := json.Marshal(instantiateReq)
	instantiateHTTPReq := httptest.NewRequest("POST", fmt.Sprintf("/api/v1/templates/%s/create-rule", createdTemplate.ID), bytes.NewReader(instantiateBody))
	instantiateHTTPReq.Header.Set("X-Tenant-ID", tenantID)
	instantiateHTTPReq.Header.Set("X-User-ID", userID)

	instantiateW := httptest.NewRecorder()
	handler.InstantiateTemplate(instantiateW, instantiateHTTPReq)

	assert.Equal(t, http.StatusCreated, instantiateW.Code)

	var newRule map[string]interface{}
	json.Unmarshal(instantiateW.Body.Bytes(), &newRule)

	assert.Equal(t, "My Rule from Template", newRule["name"])
	assert.Equal(t, "calendar", newRule["businessObject"])
	assert.Equal(t, "draft", newRule["status"])
}

// TestTemplateRLSEnforcement tests that users can't access other tenant's templates
func TestTemplateRLSEnforcement(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenant1 := createTestTenantID()
	tenant2 := createTestTenantID()
	user1 := createTestUserID()

	defer cleanupTestTemplates(t, db, tenant1)
	defer cleanupTestTemplates(t, db, tenant2)

	handler := &TemplateHandler{db: db}

	// Create template in tenant 1
	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Tenant 1 Template",
		Description:     "Test RLS",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	body, _ := json.Marshal(createReq)
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
	req.Header.Set("X-Tenant-ID", tenant1)
	req.Header.Set("X-User-ID", user1)

	w := httptest.NewRecorder()
	handler.CreateTemplate(w, req)

	var createdTemplate RuleTemplate
	json.Unmarshal(w.Body.Bytes(), &createdTemplate)

	// Try to access from tenant 2 (should fail)
	getReq := httptest.NewRequest("GET", fmt.Sprintf("/templates/%s", createdTemplate.ID), nil)
	getReq.Header.Set("X-Tenant-ID", tenant2)

	getW := httptest.NewRecorder()

	// Wrap handler with router to properly extract path variables
	router := mux.NewRouter()
	router.HandleFunc("/templates/{templateId}", handler.GetTemplate).Methods("GET")
	router.ServeHTTP(getW, getReq)

	// Should return 404 (not found) due to RLS
	assert.Equal(t, http.StatusNotFound, getW.Code)
}

// TestTemplateStatusConstraints tests status validation
func TestTemplateStatusConstraints(t *testing.T) {
	db := setupTestTemplateDB(t)
	defer db.Close()

	tenantID := createTestTenantID()

	// Try to create template with invalid status
	invalidReq := map[string]interface{}{
		"businessObject":  "calendar",
		"name":            "Test",
		"category":        "test",
		"status":          "invalid_status",
		"baseRuleSteps":   []TemplateStep{},
		"parameterSchema": map[string]interface{}{},
		"isPublic":        false,
	}

	body, _ := json.Marshal(invalidReq)
	_ = string(body) // body not used, but keeping for consistency

	// Direct database test
	_, err := db.Exec(`
		INSERT INTO edm.rule_templates (tenant_id, business_object, name, status, base_rule_steps, parameter_schema, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, tenantID, "calendar", "Test", "invalid_status", "[]", "{}", createTestUserID())

	// Should fail due to CHECK constraint
	assert.Error(t, err)
}

// ============================================================================
// Benchmark Tests
// ============================================================================

// BenchmarkCreateTemplate measures template creation performance
func BenchmarkCreateTemplate(b *testing.B) {
	db := setupTestTemplateDB(&testing.T{})
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(&testing.T{}, db, tenantID)

	handler := &TemplateHandler{db: db}

	createReq := CreateTemplateRequest{
		BusinessObject:  "calendar",
		Name:            "Benchmark Template",
		Description:     "Test",
		Category:        "test",
		BaseRuleSteps:   []TemplateStep{},
		ParameterSchema: map[string]interface{}{},
		IsPublic:        false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)

		w := httptest.NewRecorder()
		handler.CreateTemplate(w, req)
	}
}

// BenchmarkListTemplates measures template listing performance
func BenchmarkListTemplates(b *testing.B) {
	db := setupTestTemplateDB(&testing.T{})
	defer db.Close()

	tenantID := createTestTenantID()
	userID := createTestUserID()
	defer cleanupTestTemplates(&testing.T{}, db, tenantID)

	handler := &TemplateHandler{db: db}

	// Create 10 templates
	for i := 0; i < 10; i++ {
		createReq := CreateTemplateRequest{
			BusinessObject:  "calendar",
			Name:            fmt.Sprintf("Template %d", i),
			Category:        "test",
			BaseRuleSteps:   []TemplateStep{},
			ParameterSchema: map[string]interface{}{},
			IsPublic:        false,
		}

		body, _ := json.Marshal(createReq)
		req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewReader(body))
		req.Header.Set("X-Tenant-ID", tenantID)
		req.Header.Set("X-User-ID", userID)

		w := httptest.NewRecorder()
		handler.CreateTemplate(w, req)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		listReq := httptest.NewRequest("GET", "/api/v1/templates?businessObject=calendar", nil)
		listReq.Header.Set("X-Tenant-ID", tenantID)

		w := httptest.NewRecorder()
		handler.ListTemplates(w, listReq)
	}
}
