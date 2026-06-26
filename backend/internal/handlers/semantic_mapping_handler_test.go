package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestHandleGenerateMappings_RequiresNewHeader(t *testing.T) {
	h := NewSemanticMappingHandler(nil)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/semantic-mapping/generate", strings.NewReader("{}"))
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}

func TestHandleApplyMappings_RequiresNewHeader(t *testing.T) {
	h := NewSemanticMappingHandler(nil)
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	req := httptest.NewRequest("POST", "/semantic-mapping/apply", strings.NewReader("{}"))
	req.Header.Set("X-Tenant-ID", "t1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Result().StatusCode)
}
