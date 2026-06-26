package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/stretchr/testify/require"
)

type bundleServiceMock struct {
	updateBundlePoliciesCalled bool
	lastUser                   models.User
	lastID                     string
	lastRowPolicies            []models.BundleRowPolicy
	lastColumnPolicies         []models.BundleColumnPolicy
	response                   *models.DataBundle
	err                        error
}

func (m *bundleServiceMock) CreateBundle(models.User, string, string) (*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) GetBundle(models.User, string) (*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) ListBundles(models.User) ([]*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) UpdateBundle(models.User, string, []models.SemanticObjectReference, []models.SemanticObjectReference) (*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) UpdateBundlePolicies(user models.User, id string, rowPolicies []models.BundleRowPolicy, columnPolicies []models.BundleColumnPolicy) (*models.DataBundle, error) {
	m.updateBundlePoliciesCalled = true
	m.lastUser = user
	m.lastID = id
	m.lastRowPolicies = rowPolicies
	m.lastColumnPolicies = columnPolicies
	return m.response, m.err
}

func (m *bundleServiceMock) CertifyBundle(models.User, string) (*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) PublishBundle(models.User, string) (*models.DataBundle, error) {
	return nil, nil
}

func (m *bundleServiceMock) DeprecateBundle(models.User, string) (*models.DataBundle, error) {
	return nil, nil
}

func TestUpdateBundlePoliciesHandler(t *testing.T) {
	mockService := &bundleServiceMock{
		response: &models.DataBundle{
			ID: "bundle-123",
			RowPolicies: []models.BundleRowPolicy{
				{
					ID:       "row-1",
					Name:     "Tenant Filter",
					Member:   "orders.tenant_id",
					Operator: "equals",
					Values:   []string{"tenant-a"},
				},
			},
			ColumnPolicies: []models.BundleColumnPolicy{
				{
					ID:       "col-1",
					Name:     "Mask PII",
					Columns:  []string{"ssn"},
					MaskType: "redact",
				},
			},
		},
	}

	handler := NewBundleHandler(mockService)
	router := chi.NewRouter()
	router.Route("/api", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	requestPayload := map[string]any{
		"rowPolicies": []models.BundleRowPolicy{
			{
				ID:       "row-1",
				Name:     "Tenant Filter",
				Member:   "orders.tenant_id",
				Operator: "equals",
				Values:   []string{"tenant-a"},
			},
		},
		"columnPolicies": []models.BundleColumnPolicy{
			{
				ID:       "col-1",
				Name:     "Mask PII",
				Columns:  []string{"ssn"},
				MaskType: "redact",
			},
		},
	}

	body, err := json.Marshal(requestPayload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/bundles/bundle-123/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.True(t, mockService.updateBundlePoliciesCalled)
	require.Equal(t, "bundle-123", mockService.lastID)
	require.Equal(t, "user-steward-1", mockService.lastUser.ID)

	require.Len(t, mockService.lastRowPolicies, 1)
	require.Len(t, mockService.lastColumnPolicies, 1)
	require.Equal(t, requestPayload["rowPolicies"].([]models.BundleRowPolicy)[0], mockService.lastRowPolicies[0])
	require.Equal(t, requestPayload["columnPolicies"].([]models.BundleColumnPolicy)[0], mockService.lastColumnPolicies[0])

	var response models.DataBundle
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, mockService.response.ID, response.ID)
	require.Len(t, response.RowPolicies, 1)
	require.Len(t, response.ColumnPolicies, 1)
}

func TestUpdateBundlePoliciesHandlerValidationError(t *testing.T) {
	mockService := &bundleServiceMock{
		err: services.NewValidationError("bundle policy validation failed", []services.FieldError{{
			Field:   "rowPolicies[0].name",
			Message: "Name is required",
		}}),
	}

	handler := NewBundleHandler(mockService)
	router := chi.NewRouter()
	router.Route("/api", func(r chi.Router) {
		handler.RegisterRoutes(r)
	})

	requestPayload := map[string]any{
		"rowPolicies": []models.BundleRowPolicy{{}},
	}

	body, err := json.Marshal(requestPayload)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPut, "/api/bundles/bundle-123/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "validation_failed", payload["error"])
	require.Equal(t, "bundle policy validation failed", payload["message"])
	if details, ok := payload["details"].([]any); ok {
		require.Len(t, details, 1)
	} else {
		t.Fatalf("expected details array, got %T", payload["details"])
	}
}
