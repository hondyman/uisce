package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"

	"github.com/go-chi/chi/v5"

	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/datapipeline"
	"github.com/hondyman/uisce/backend/internal/handlers"
	catalogmeta "github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/security"
)

// pipelineCatalog is the platform as one tenant sees it, for grounding
// pipelines (editor validation, the AI assistant, MCP). Every lookup goes
// through the same service or handler the rest of the platform uses, with the
// caller's own tenant.
type pipelineCatalog struct {
	r        *http.Request // the caller's request: carries its authentication
	tenant   string
	bos      *catalogmeta.BusinessObjectService
	resolver security.DatasourceResolver
	crud     *BOCRUDHandler
	rules    *analytics.ValidationRuleService
	deps     datapipeline.Deps
}

func (c *pipelineCatalog) BusinessObjects(ctx context.Context) ([]datapipeline.BOInfo, error) {
	if c.bos == nil {
		return nil, fmt.Errorf("business objects are not available")
	}
	// Same listing as GET /business-objects (core + custom, the tenant's
	// datasource), built from the caller's own request.
	secCtx, sctx, err := handlers.SecurityContextFromRequest(c.r.WithContext(ctx), "", "", handlers.SecurityContextDeps{Resolver: c.resolver})
	if err != nil {
		return nil, err
	}
	items, err := c.bos.ListBusinessObjectsComposed(sctx, secCtx)
	if err != nil {
		if items, err = c.bos.ListBusinessObjects(sctx, secCtx); err != nil {
			return nil, err
		}
	}
	out := make([]datapipeline.BOInfo, 0, len(items))
	for _, b := range items {
		if b == nil || b.Key == "" {
			continue
		}
		label := b.DisplayName
		if label == "" {
			label = b.Name
		}
		out = append(out, datapipeline.BOInfo{Key: b.Key, Label: label, Description: b.Description})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

// BOFields returns the BO's writable fields (physical column names: what the
// BO sink writes), via the /bo/{boKey}/schema handler itself.
func (c *pipelineCatalog) BOFields(ctx context.Context, boKey string) ([]datapipeline.TargetField, error) {
	fields, err := c.crud.schemaFields(c.r.WithContext(ctx), boKey)
	if err != nil {
		return nil, err
	}
	out := make([]datapipeline.TargetField, 0, len(fields))
	for _, f := range fields {
		name := f.PhysicalColumn
		if name == "" {
			name = f.Name
		}
		label := f.DisplayName
		if label == "" {
			label = f.Name
		}
		out = append(out, datapipeline.TargetField{Name: name, Label: label, Type: f.Type, Required: f.Required})
	}
	return out, nil
}

func (c *pipelineCatalog) Rules(ctx context.Context, boKey string) ([]datapipeline.RuleInfo, error) {
	list, err := c.rules.ListByBO(ctx, c.tenant, boKey, "")
	if err != nil {
		return nil, err
	}
	var out []datapipeline.RuleInfo
	for _, r := range list {
		if r.IsActive {
			out = append(out, datapipeline.RuleInfo{ID: r.ID.String(), Name: r.Name, Severity: r.Severity, Description: r.Description})
		}
	}
	return out, nil
}

func (c *pipelineCatalog) Files(ctx context.Context) ([]string, error) {
	if c.deps.Files == nil {
		return nil, fmt.Errorf("the file engine is not configured")
	}
	p, err := datapipeline.TenantPath(c.tenant, "uploads")
	if err != nil {
		return nil, err
	}
	entries, err := c.deps.Files.List(ctx, p)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(entries))
	for i, e := range entries {
		out[i] = datapipeline.TenantRelative(c.tenant, e.Path)
	}
	return out, nil
}

func (c *pipelineCatalog) StagingTables(ctx context.Context) ([]datapipeline.StagingTable, error) {
	if c.deps.StagingDB == nil {
		return nil, fmt.Errorf("the staging database is not configured")
	}
	return datapipeline.StagingTables(ctx, c.deps.StagingDB)
}

// schemaFields runs HandleGetBOSchema in-process for the caller's request,
// so field resolution (and its tenant scoping) is exactly the endpoint's.
func (h *BOCRUDHandler) schemaFields(r *http.Request, boKey string) ([]boSchemaField, error) {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("boKey", boKey)
	req := r.Clone(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	req.Method, req.Body = http.MethodGet, http.NoBody
	rec := httptest.NewRecorder()
	h.HandleGetBOSchema(rec, req)
	if rec.Code != http.StatusOK {
		return nil, fmt.Errorf("business object %q: %s", boKey, rec.Body.String())
	}
	var resp boSchemaResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		return nil, err
	}
	return resp.Fields, nil
}
