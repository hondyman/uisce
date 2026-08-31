package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// StudioDefinitionsHandler persists Page Studio pages and API Studio endpoint
// definitions and fires the matching trigger_types events (page_save,
// api_endpoint_save) on every committed write, following the same
// tenant-scoped emit pattern as BOCRUDHandler.emitBORowEvent.
type StudioDefinitionsHandler struct {
	db      *sqlx.DB
	trigger *TriggerEngine
}

func NewStudioDefinitionsHandler(db *sqlx.DB, trigger *TriggerEngine) *StudioDefinitionsHandler {
	return &StudioDefinitionsHandler{db: db, trigger: trigger}
}

func (h *StudioDefinitionsHandler) emitStudioEvent(triggerKey string, tenantID uuid.UUID, targetEntity, entityID string, eventData map[string]interface{}) {
	if h.trigger == nil {
		return
	}
	go func() {
		_, err := h.trigger.EvaluateTriggers(context.Background(), &TriggerContext{
			TenantID:     tenantID.String(),
			TriggerKey:   triggerKey,
			TargetEntity: targetEntity,
			EntityID:     entityID,
			EventData:    eventData,
			RequestedAt:  time.Now(),
		})
		if err != nil {
			log.Printf("[WARN] Studio event %q for %s/%s failed: %v", triggerKey, targetEntity, entityID, err)
		}
	}()
}

func (h *StudioDefinitionsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/page-studio", func(r chi.Router) {
		r.Get("/pages", h.HandleListPages)
		r.Post("/pages", h.HandleSavePage)
		r.Get("/pages/{id}", h.HandleGetPage)
		r.Put("/pages/{id}", h.HandleUpdatePage)
		r.Delete("/pages/{id}", h.HandleDeletePage)
		r.Get("/pages/slug/{slug}", h.HandleGetPageBySlug)
	})
	r.Route("/api-studio", func(r chi.Router) {
		r.Get("/endpoints", h.HandleListEndpoints)
		r.Post("/endpoints", h.HandleSaveEndpoint)
		r.Get("/endpoints/{id}", h.HandleGetEndpoint)
		r.Post("/endpoints/{id}/deprecate", h.HandleDeprecateEndpoint)
		r.Post("/endpoints/{id}/retire", h.HandleRetireEndpoint)
	})
}

type pageDefinition struct {
	ID          string          `db:"id" json:"id"`
	TenantID    string          `db:"tenant_id" json:"-"`
	Name        string          `db:"name" json:"name"`
	Slug        string          `db:"slug" json:"slug"`
	Description *string         `db:"description" json:"description,omitempty"`
	Layout      json.RawMessage `db:"layout" json:"layout"`
	Components  json.RawMessage `db:"components" json:"components"`
	DataSources json.RawMessage `db:"data_sources" json:"dataSources"`
	Version     int             `db:"version" json:"version"`
	CreatedAt   time.Time       `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updatedAt"`
}

func (h *StudioDefinitionsHandler) HandleListPages(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	var pages []pageDefinition
	err := h.db.SelectContext(r.Context(), &pages,
		`SELECT id, tenant_id, name, slug, description, layout, components, data_sources, version, created_at, updated_at
		 FROM page_definitions WHERE tenant_id = $1 ORDER BY updated_at DESC`, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing pages: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, pages)
}

func (h *StudioDefinitionsHandler) HandleGetPage(w http.ResponseWriter, r *http.Request) {
	h.getPageBy(w, r, "id", chi.URLParam(r, "id"))
}

func (h *StudioDefinitionsHandler) HandleGetPageBySlug(w http.ResponseWriter, r *http.Request) {
	h.getPageBy(w, r, "slug", chi.URLParam(r, "slug"))
}

func (h *StudioDefinitionsHandler) getPageBy(w http.ResponseWriter, r *http.Request, column, value string) {
	tenantID := extractTenantUUIDFromRequest(r)
	var page pageDefinition
	err := h.db.GetContext(r.Context(), &page, fmt.Sprintf(
		`SELECT id, tenant_id, name, slug, description, layout, components, data_sources, version, created_at, updated_at
		 FROM page_definitions WHERE tenant_id = $1 AND %s = $2`, column), tenantID, value)
	if err != nil {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *StudioDefinitionsHandler) HandleSavePage(w http.ResponseWriter, r *http.Request) {
	h.upsertPage(w, r, "")
}

func (h *StudioDefinitionsHandler) HandleUpdatePage(w http.ResponseWriter, r *http.Request) {
	h.upsertPage(w, r, chi.URLParam(r, "id"))
}

func (h *StudioDefinitionsHandler) upsertPage(w http.ResponseWriter, r *http.Request, id string) {
	tenantID := extractTenantUUIDFromRequest(r)

	var in struct {
		ID          string          `json:"id"`
		Name        string          `json:"name"`
		Slug        string          `json:"slug"`
		Description *string         `json:"description"`
		Layout      json.RawMessage `json:"layout"`
		Components  json.RawMessage `json:"components"`
		DataSources json.RawMessage `json:"dataSources"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if id == "" {
		id = in.ID
	}
	if in.Layout == nil {
		in.Layout = json.RawMessage(`[]`)
	}
	if in.Components == nil {
		in.Components = json.RawMessage(`[]`)
	}
	if in.DataSources == nil {
		in.DataSources = json.RawMessage(`[]`)
	}

	var page pageDefinition
	var err error
	if id == "" {
		err = h.db.GetContext(r.Context(), &page, `
			INSERT INTO page_definitions (tenant_id, name, slug, description, layout, components, data_sources)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, tenant_id, name, slug, description, layout, components, data_sources, version, created_at, updated_at`,
			tenantID, in.Name, in.Slug, in.Description, in.Layout, in.Components, in.DataSources)
	} else {
		err = h.db.GetContext(r.Context(), &page, `
			UPDATE page_definitions
			SET name = $3, slug = $4, description = $5, layout = $6, components = $7, data_sources = $8,
			    version = version + 1, updated_at = now()
			WHERE tenant_id = $1 AND id = $2
			RETURNING id, tenant_id, name, slug, description, layout, components, data_sources, version, created_at, updated_at`,
			tenantID, id, in.Name, in.Slug, in.Description, in.Layout, in.Components, in.DataSources)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("failed saving page: %v", err), http.StatusInternalServerError)
		return
	}

	h.emitStudioEvent("page_save", tenantID, "page_definition", page.ID, map[string]interface{}{
		"id": page.ID, "name": page.Name, "slug": page.Slug, "version": page.Version,
	})

	writeJSON(w, http.StatusOK, page)
}

func (h *StudioDefinitionsHandler) HandleDeletePage(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	id := chi.URLParam(r, "id")
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM page_definitions WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed deleting page: %v", err), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type apiDefinition struct {
	ID                string          `db:"id" json:"id"`
	TenantID          string          `db:"tenant_id" json:"tenant_id"`
	Env               string          `db:"env" json:"env"`
	Name              string          `db:"name" json:"name"`
	Path              string          `db:"path" json:"path"`
	Method            string          `db:"method" json:"method"`
	Type              string          `db:"type" json:"type"`
	BOName            *string         `db:"bo_name" json:"bo_name,omitempty"`
	Fields            json.RawMessage `db:"fields" json:"fields"`
	Filters           json.RawMessage `db:"filters" json:"filters"`
	Pagination        json.RawMessage `db:"pagination" json:"pagination"`
	AuthPolicy        *string         `db:"auth_policy" json:"auth_policy,omitempty"`
	Version           int             `db:"version" json:"version"`
	IsActive          bool            `db:"is_active" json:"is_active"`
	Status            string          `db:"status" json:"status"`
	SemanticVersion   string          `db:"semantic_version" json:"semantic_version"`
	PreviousVersionID *string         `db:"previous_version_id" json:"previous_version_id,omitempty"`
	OwnerTeam         *string         `db:"owner_team" json:"owner_team,omitempty"`
	DeprecatedAt      *time.Time      `db:"deprecated_at" json:"deprecated_at,omitempty"`
	RetiredAt         *time.Time      `db:"retired_at" json:"retired_at,omitempty"`
	RequestSchemaID   *string         `db:"request_schema_id" json:"request_schema_id,omitempty"`
	ResponseSchemaID  *string         `db:"response_schema_id" json:"response_schema_id,omitempty"`
	CreatedAt         time.Time       `db:"created_at" json:"-"`
	UpdatedAt         time.Time       `db:"updated_at" json:"-"`
}

const apiDefinitionColumns = `id, tenant_id, env, name, path, method, type, bo_name, fields, filters, pagination,
	auth_policy, version, is_active, status, semantic_version, previous_version_id, owner_team,
	deprecated_at, retired_at, request_schema_id, response_schema_id, created_at, updated_at`

func (h *StudioDefinitionsHandler) HandleListEndpoints(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	env := r.URL.Query().Get("env")
	if env == "" {
		env = "default"
	}
	var endpoints []apiDefinition
	err := h.db.SelectContext(r.Context(), &endpoints, fmt.Sprintf(
		`SELECT %s FROM api_definitions WHERE tenant_id = $1 AND env = $2 ORDER BY updated_at DESC`, apiDefinitionColumns),
		tenantID, env)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing endpoints: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, endpoints)
}

func (h *StudioDefinitionsHandler) HandleGetEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	id := chi.URLParam(r, "id")
	var ep apiDefinition
	err := h.db.GetContext(r.Context(), &ep, fmt.Sprintf(
		`SELECT %s FROM api_definitions WHERE tenant_id = $1 AND id = $2`, apiDefinitionColumns), tenantID, id)
	if err != nil {
		http.Error(w, "endpoint not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func (h *StudioDefinitionsHandler) HandleSaveEndpoint(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)

	var in struct {
		ID         string          `json:"id"`
		Env        string          `json:"env"`
		Name       string          `json:"name"`
		Path       string          `json:"path"`
		Method     string          `json:"method"`
		Type       string          `json:"type"`
		BOName     *string         `json:"bo_name"`
		Fields     json.RawMessage `json:"fields"`
		Filters    json.RawMessage `json:"filters"`
		Pagination json.RawMessage `json:"pagination"`
		AuthPolicy *string         `json:"auth_policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if in.Env == "" {
		in.Env = "default"
	}
	if in.Method == "" {
		in.Method = "GET"
	}
	if in.Type == "" {
		in.Type = "rest"
	}
	if in.Fields == nil {
		in.Fields = json.RawMessage(`[]`)
	}
	if in.Filters == nil {
		in.Filters = json.RawMessage(`{}`)
	}
	if in.Pagination == nil {
		in.Pagination = json.RawMessage(`{"type":"offset","default_limit":50}`)
	}

	var ep apiDefinition
	var err error
	if in.ID == "" {
		err = h.db.GetContext(r.Context(), &ep, fmt.Sprintf(`
			INSERT INTO api_definitions (tenant_id, env, name, path, method, type, bo_name, fields, filters, pagination, auth_policy)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (tenant_id, env, path, method) DO UPDATE SET
				name = EXCLUDED.name, type = EXCLUDED.type, bo_name = EXCLUDED.bo_name,
				fields = EXCLUDED.fields, filters = EXCLUDED.filters, pagination = EXCLUDED.pagination,
				auth_policy = EXCLUDED.auth_policy, version = api_definitions.version + 1, updated_at = now()
			RETURNING %s`, apiDefinitionColumns),
			tenantID, in.Env, in.Name, in.Path, in.Method, in.Type, in.BOName, in.Fields, in.Filters, in.Pagination, in.AuthPolicy)
	} else {
		err = h.db.GetContext(r.Context(), &ep, fmt.Sprintf(`
			UPDATE api_definitions
			SET name = $3, path = $4, method = $5, type = $6, bo_name = $7, fields = $8, filters = $9,
			    pagination = $10, auth_policy = $11, version = version + 1, updated_at = now()
			WHERE tenant_id = $1 AND id = $2
			RETURNING %s`, apiDefinitionColumns),
			tenantID, in.ID, in.Name, in.Path, in.Method, in.Type, in.BOName, in.Fields, in.Filters, in.Pagination, in.AuthPolicy)
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("failed saving endpoint: %v", err), http.StatusInternalServerError)
		return
	}

	h.emitStudioEvent("api_endpoint_save", tenantID, "api_definition", ep.ID, map[string]interface{}{
		"id": ep.ID, "name": ep.Name, "path": ep.Path, "method": ep.Method, "version": ep.Version,
	})

	writeJSON(w, http.StatusOK, ep)
}

func (h *StudioDefinitionsHandler) HandleDeprecateEndpoint(w http.ResponseWriter, r *http.Request) {
	h.setEndpointStatus(w, r, "deprecated", "deprecated_at")
}

func (h *StudioDefinitionsHandler) HandleRetireEndpoint(w http.ResponseWriter, r *http.Request) {
	h.setEndpointStatus(w, r, "retired", "retired_at")
}

func (h *StudioDefinitionsHandler) setEndpointStatus(w http.ResponseWriter, r *http.Request, status, timestampColumn string) {
	tenantID := extractTenantUUIDFromRequest(r)
	id := chi.URLParam(r, "id")
	var ep apiDefinition
	err := h.db.GetContext(r.Context(), &ep, fmt.Sprintf(`
		UPDATE api_definitions SET status = $3, %s = now(), updated_at = now()
		WHERE tenant_id = $1 AND id = $2
		RETURNING %s`, timestampColumn, apiDefinitionColumns), tenantID, id, status)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed updating endpoint status: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, ep)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
