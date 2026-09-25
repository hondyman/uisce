package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
)

func dpRouter(deps datapipeline.Deps) chi.Router {
	r := chi.NewRouter()
	NewDataPipelineHandler(&datapipeline.Store{}, deps, nil).RegisterRoutes(r)
	return r
}

func TestDataPipelines_RefuseRequestsWithoutTenant(t *testing.T) {
	r := dpRouter(datapipeline.Deps{})
	for _, c := range []struct{ method, path string }{
		{"GET", "/data-pipelines/"}, {"POST", "/data-pipelines/"}, {"POST", "/data-pipelines/validate"},
		{"POST", "/data-pipelines/preview"}, {"GET", "/data-pipelines/files"}, {"GET", "/data-pipelines/runs/x"},
		{"POST", "/data-pipelines/00000000-0000-0000-0000-000000000009/runs"},
	} {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, bytes.NewBufferString("{}")))
		assert.Equal(t, http.StatusUnauthorized, rec.Code, "%s %s", c.method, c.path)
	}
}

func TestDataPipelines_ValidateAttachesIssuesToNodes(t *testing.T) {
	spec := `{"nodes":[{"id":"src","type":"file_source","config":{"uri":"","format":"xls"}}],"edges":[]}`
	req := withTestAuth(httptest.NewRequest("POST", "/data-pipelines/validate", bytes.NewBufferString(spec)), pipeTenant)
	rec := httptest.NewRecorder()
	dpRouter(datapipeline.Deps{}).ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var out struct {
		Valid  bool                 `json:"valid"`
		Issues []datapipeline.Issue `json:"issues"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.False(t, out.Valid)
	var onNode []string
	for _, i := range out.Issues {
		if i.NodeID == "src" {
			onNode = append(onNode, i.Message)
		}
	}
	assert.Contains(t, onNode, "uri is required")
	assert.Contains(t, onNode, "format must be csv, json or parquet")
}

func TestDataPipelines_UploadRejectsPathNames(t *testing.T) {
	r := dpRouter(datapipeline.Deps{Files: &datapipeline.HTTPFileEngine{BaseURL: "http://unused"}})
	for _, name := range []string{"", "../x.csv", "a/b.csv", ".hidden", `a\b.csv`} {
		req := withTestAuth(httptest.NewRequest("POST", "/data-pipelines/files/upload?name="+name, bytes.NewBufferString("x")), pipeTenant)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "name %q", name)
	}
}

func TestDataPipelines_PaletteReflectsConfiguration(t *testing.T) {
	req := withTestAuth(httptest.NewRequest("GET", "/data-pipelines/node-types", nil), pipeTenant)
	rec := httptest.NewRecorder()
	dpRouter(datapipeline.Deps{}).ServeHTTP(rec, req)
	var types []datapipeline.NodeType
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &types))
	byType := map[string]datapipeline.NodeType{}
	for _, nt := range types {
		byType[nt.Type] = nt
	}
	assert.True(t, byType["map"].Available)
	assert.False(t, byType["staging_sink"].Available)
	assert.Equal(t, "the staging database is not configured", byType["staging_sink"].Unavailable)
}

type stubCatalog struct{}

func (stubCatalog) BusinessObjects(context.Context) ([]datapipeline.BOInfo, error) {
	return []datapipeline.BOInfo{{Key: "fund", Label: "Fund"}}, nil
}
func (stubCatalog) BOFields(context.Context, string) ([]datapipeline.TargetField, error) {
	return []datapipeline.TargetField{{Name: "aum"}}, nil
}
func (stubCatalog) Rules(context.Context, string) ([]datapipeline.RuleInfo, error) { return nil, nil }
func (stubCatalog) Files(context.Context) ([]string, error)                        { return nil, nil }
func (stubCatalog) StagingTables(context.Context) ([]datapipeline.StagingTable, error) {
	return nil, nil
}

func TestDataPipelines_Assist(t *testing.T) {
	body := `{"message":"copy funds into fund","spec":{"nodes":[],"edges":[]}}`
	// Not configured: a clear refusal.
	rec := httptest.NewRecorder()
	dpRouter(datapipeline.Deps{}).ServeHTTP(rec, withTestAuth(httptest.NewRequest("POST", "/data-pipelines/assist", bytes.NewBufferString(body)), pipeTenant))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)

	var gotTenant string
	llm := func(context.Context, string) (string, error) {
		return `{"reply":"done","spec":{"version":1,"nodes":[
			{"id":"in","type":"bo_source","config":{"bo_key":"fund"}},
			{"id":"out","type":"bo_sink","config":{"bo_key":"funds"}}],"edges":[{"from":"in","to":"out"}]}}`, nil
	}
	r := chi.NewRouter()
	NewDataPipelineHandler(&datapipeline.Store{}, datapipeline.Deps{}, nil).WithGrounding(
		func(_ *http.Request, tenant string) datapipeline.PlatformCatalog {
			gotTenant = tenant
			return stubCatalog{}
		},
		&datapipeline.Assistant{LLM: llm, MaxAttempts: 1},
	).RegisterRoutes(r)
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, withTestAuth(httptest.NewRequest("POST", "/data-pipelines/assist", bytes.NewBufferString(body)), pipeTenant))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var out datapipeline.AssistResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, pipeTenant, gotTenant, "the catalog must be the caller's tenant")
	require.Len(t, out.Issues, 1)
	assert.Equal(t, `there is no business object "funds"`, out.Issues[0].Message)
}
