package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	httpapi "github.com/hondyman/uisce/backend/internal/api"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/security"
)

func TestSchedulerMutationsRetired(t *testing.T) {
	h := httpapi.NewSchedulerHandlers(nil, nil, zap.NewNop(), handlers.SecurityContextDeps{})
	r := chi.NewRouter()
	h.RegisterRoutes(r)

	tenant := uuid.New().String()
	withAuth := func(req *http.Request) *http.Request {
		ctx := security.WithAuthInfo(req.Context(), security.AuthInfo{
			UserID: "u1", TenantIDs: []string{tenant}, Roles: []string{"user"},
		})
		return req.WithContext(ctx)
	}

	paths := []struct {
		method, path string
	}{
		{http.MethodPost, "/scheduler/jobs"},
		{http.MethodPatch, "/scheduler/jobs/" + uuid.New().String()},
		{http.MethodDelete, "/scheduler/jobs/" + uuid.New().String()},
		{http.MethodPost, "/scheduler/jobs/" + uuid.New().String() + "/run"},
		{http.MethodPost, "/scheduler/dags"},
		{http.MethodPatch, "/scheduler/dags/" + uuid.New().String()},
		{http.MethodDelete, "/scheduler/dags/" + uuid.New().String()},
		{http.MethodPost, "/scheduler/dags/" + uuid.New().String() + "/run"},
		{http.MethodPost, "/scheduler/ai/suggestions/" + uuid.New().String() + "/accept"},
		{http.MethodPost, "/scheduler/ai/suggestions/" + uuid.New().String() + "/dismiss"},
	}

	for _, p := range paths {
		t.Run(p.method+" "+p.path, func(t *testing.T) {
			req := withAuth(httptest.NewRequest(p.method, p.path, nil))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Equal(t, http.StatusGone, w.Code)
			var body map[string]string
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, "scheduler_intelligence_mutations_retired", body["error"])
			assert.Equal(t, "</api/schedules>; rel=\"successor-version\"", w.Header().Get("Link"))
		})
	}

	t.Run("unauthenticated POST is 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/scheduler/jobs", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
