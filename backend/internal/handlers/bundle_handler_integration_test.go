package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/stretchr/testify/require"
)

// This test uses the in-memory bundle service to ensure the handler returns
// a bundle previously inserted into the store.
func TestGetBundleHandler_ReturnsSeededBundle(t *testing.T) {
	policySvc := services.NewPolicyService()
	bundleSvc, _ := services.NewBundleService(policySvc)

	// Insert a bundle directly into the in-memory store
	b := &models.DataBundle{
		ID:       "test-bundle-1",
		Name:     "Test Bundle 1",
		Version:  "1.0.0",
		Status:   models.StatusPublished,
		Audience: []string{"lp"},
	}
	require.NoError(t, services.InsertBundleForTesting(bundleSvc, b))

	h := NewBundleHandler(bundleSvc)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest(http.MethodGet, "/bundles/test-bundle-1/", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var got models.DataBundle
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, b.ID, got.ID)
	require.Equal(t, b.Name, got.Name)
}
