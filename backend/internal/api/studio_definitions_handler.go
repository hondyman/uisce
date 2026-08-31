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
	"github.com/hondyman/uisce/backend/internal/semantic"
	"github.com/jmoiron/sqlx"
)

// StudioDefinitionsHandler persists Page Studio page definitions and fires
// the page_save trigger_types event on every committed write, following the
// same tenant-scoped emit pattern as BOCRUDHandler.emitBORowEvent. Every
// save also records a "page_definition" version through the same
// semantic.SemanticVersionStore that apistudio.Service already uses for
// "api_endpoint" — this is the tier-3 metadata-change audit trail
// (backend/internal/semantic/version_store.go), not a new mechanism.
//
// API Studio endpoint definitions are handled separately by
// APIStudioHandler (apistudio_handler.go), which wires the existing
// backend/internal/apistudio package (Repository/Service against the
// semantic.api_endpoints table) rather than a parallel table — see that
// file for why.
type StudioDefinitionsHandler struct {
	db       *sqlx.DB
	trigger  *TriggerEngine
	versions *semantic.SemanticVersionStore
}

func NewStudioDefinitionsHandler(db *sqlx.DB, trigger *TriggerEngine) *StudioDefinitionsHandler {
	return &StudioDefinitionsHandler{db: db, trigger: trigger, versions: semantic.NewSemanticVersionStore(db)}
}

// recordPageVersion records the saved page as a new "page_definition"
// version. Failures are logged, not surfaced to the caller: metadata
// versioning must never block a page save from succeeding.
func (h *StudioDefinitionsHandler) recordPageVersion(ctx context.Context, tenantID uuid.UUID, page pageDefinition, actor string) {
	payload, err := json.Marshal(page)
	if err != nil {
		log.Printf("[WARN] failed marshaling page %s for versioning: %v", page.ID, err)
		return
	}
	tenantIDStr := tenantID.String()
	err = h.versions.SaveObject(ctx, semantic.SemanticObject{
		ID:       page.ID,
		Env:      "default",
		TenantID: &tenantIDStr,
		Type:     "page_definition",
		Payload:  payload,
	}, actor)
	if err != nil {
		log.Printf("[WARN] failed recording page_definition version for %s: %v", page.ID, err)
	}
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
	h.recordPageVersion(r.Context(), tenantID, page, actorFromRequest(r))

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

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
