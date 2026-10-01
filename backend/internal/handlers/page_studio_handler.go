package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

func isUniqueViolation(err error) bool {
	pqErr, ok := err.(*pq.Error)
	return ok && pqErr.Code == "23505"
}

// PageStudioPage is the wire shape the Page Studio frontend
// (frontend/src/api/pageStudio.ts, CorePageDefinition) reads and writes.
// Backed by page_definitions, which existed with no handler at all before
// this - every save from the Page Studio UI 404'd.
type PageStudioPage struct {
	ID                 uuid.UUID        `json:"id" db:"id"`
	TenantID           uuid.UUID        `json:"-" db:"tenant_id"`
	Name               string           `json:"name" db:"name"`
	Slug               string           `json:"slug" db:"slug"`
	Description        string           `json:"description,omitempty" db:"description"`
	Layout             json.RawMessage  `json:"layout" db:"layout"`
	Tabs               json.RawMessage  `json:"tabs,omitempty" db:"tabs"`
	Components         json.RawMessage  `json:"components" db:"components"`
	DataSources        json.RawMessage  `json:"dataSources" db:"data_sources"`
	PresentationEvents json.RawMessage  `json:"presentationEvents,omitempty" db:"presentation_events"`
	FilterBar          json.RawMessage  `json:"filterBar,omitempty" db:"filter_bar"`
	App                *json.RawMessage `json:"app,omitempty" db:"app_model"` // page application model (variables, queries, tabs); opaque here, NULL on plain BO pages (pointer: RawMessage cannot scan NULL)
	Version            int              `json:"version" db:"version"`
	IsCore             bool             `json:"isCore" db:"is_core"`
	Status             string           `json:"status" db:"status"`
	CreatedAt          time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt          time.Time        `json:"updatedAt" db:"updated_at"`
	// Editable is computed: core pages are writable only by gold-copy admin.
	Editable bool `json:"editable" db:"-"`
	// Customization: on a core page seen from a tenant, how that tenant
	// uses it (page_studio_core.go). CanCustomize: the caller may extend,
	// clone or switch it off. ClonedFrom: this tenant page is a clone.
	Customization *PageCustomization `json:"customization,omitempty" db:"-"`
	CanCustomize  bool               `json:"canCustomize,omitempty" db:"-"`
	ClonedFrom    *PageCloneSource   `json:"clonedFrom,omitempty" db:"-"`
	// MenuPlacements: every menu entry (the tenant's own or inherited from
	// the gold copy) that opens this page. List responses only.
	MenuPlacements []PageMenuPlacement `json:"menuPlacements,omitempty" db:"-"`
}

// pageColumns is the select list every page read and RETURNING uses.
const pageColumns = `id, tenant_id, name, slug, COALESCE(description, '') AS description,
	       layout, tabs, components, data_sources,
	       COALESCE(presentation_events, '[]'::jsonb) AS presentation_events,
	       COALESCE(filter_bar, '{}'::jsonb) AS filter_bar,
	       app_model,
	       version, is_core, status, created_at, updated_at`

// PageAIField is one BO field passed to the AI page generator as grounding.
type PageAIField struct {
	Key         string
	DisplayName string
	DataType    string
	Role        string
}

// PageAIRelatedBO is one Business Object related to the page's primary BO
// (from the catalog relationship graph), offered to the generator as a
// candidate to pull onto the page as its own section.
type PageAIRelatedBO struct {
	BOID             string
	BOKey            string
	DisplayName      string
	RelationshipType string
	Cardinality      string
	JoinCondition    string
	Fields           []PageAIField
}

// PageAISection is one widget the generator wants placed on the page, and
// which Business Object (by key; "" = primary) it should bind to.
type PageAISection struct {
	BOKey string
	Type  string
	Title string
}

// PageAISpec is the generated page shape: a title, layout template, and a
// small section mix. Deliberately NOT a full CorePageDefinition - expanding
// this into an actual layout/components/dataSources tree is the frontend's
// job (fetchBusinessObjectBindings + the same DataSourceDefinition
// construction DataBindingsPanel.tsx already does), since widget data
// binding is resolved live from the page's Business Object data source at
// render time, not chosen by the generator.
type PageAISpec struct {
	Title          string
	PageKind       string
	LayoutTemplate string
	Sections       []PageAISection
	FilterBar      []PageAISection
}

// PageAIGenerateFunc calls out to whatever LLM gateway is configured (or is
// nil when none is). Defined here, not in internal/api, so this package
// doesn't have to import internal/api to accept it - internal/api already
// imports internal/handlers to wire up its routes, so the reverse import
// would cycle.
type PageAIGenerateFunc func(ctx context.Context, boName, boKey, description, pageKind string, fields []PageAIField, relatedBOs []PageAIRelatedBO) (*PageAISpec, error)

// boRelationshipsService is the one method of metadata.BusinessObjectService
// this handler needs - a narrow interface (rather than depending on the
// concrete struct) purely so a fake is easy to substitute in tests, matching
// the same seam internal/api/business_object_handlers.go already uses.
type boRelationshipsService interface {
	GetBusinessObjectRelationships(ctx context.Context, secCtx *security.Context, boID string) (*metadata.BORelationshipsResponse, error)
}

type PageStudioHandler struct {
	db            *sqlx.DB
	boRepo        *boresolver.PostgresBORepository
	relationships boRelationshipsService
	generateSpec  PageAIGenerateFunc
}

func NewPageStudioHandler(db *sqlx.DB, boRepo *boresolver.PostgresBORepository, relationships boRelationshipsService, generateSpec PageAIGenerateFunc) *PageStudioHandler {
	return &PageStudioHandler{db: db, boRepo: boRepo, relationships: relationships, generateSpec: generateSpec}
}

func (h *PageStudioHandler) RegisterRoutes(r chi.Router) {
	r.Route("/page-studio/pages", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Get("/slug/{slug}", h.getBySlug)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Put("/{id}/status", h.setStatus)
		// Tenant lifecycle of a core page (page_studio_core.go).
		r.Put("/{id}/extension", h.saveExtension)
		r.Put("/{id}/activation", h.setActivation)
		r.Post("/{id}/clone", h.cloneCore)
		r.Delete("/{id}/customization", h.revertToCore)
		r.Get("/{id}/compare", h.compare)
		r.Post("/{id}/upgrade", h.upgrade)
	})
	h.registerFragmentRoutes(r)
	h.registerBundleRoutes(r)
	r.Post("/page-studio/generate", h.generate)
}

type pageStudioGenerateRequest struct {
	BOID        string `json:"boId"`
	BOKey       string `json:"boKey"`
	BOName      string `json:"boName"`
	Description string `json:"description"`
	PageKind    string `json:"pageKind"`
}

type pageStudioGenerateSection struct {
	BOKey string `json:"boKey"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type pageStudioRelatedBO struct {
	BOID          string `json:"boId"`
	BOKey         string `json:"boKey"`
	DisplayName   string `json:"displayName"`
	Cardinality   string `json:"cardinality,omitempty"`
	JoinCondition string `json:"joinCondition,omitempty"`
}

type pageStudioGenerateResponse struct {
	Title                  string                      `json:"title"`
	PageKind               string                      `json:"pageKind"`
	LayoutTemplate         string                      `json:"layoutTemplate"`
	RelatedBusinessObjects []pageStudioRelatedBO       `json:"relatedBusinessObjects"`
	Sections               []pageStudioGenerateSection `json:"sections"`
	FilterBar              []pageStudioGenerateSection `json:"filterBar,omitempty"`
	Source                 string                      `json:"source"` // "ai" or "template", surfaced so the UI can be honest about which one ran
}

// maxGeneratedRelatedBOs caps how many related Business Objects are even
// offered as candidates - both to Gemini (keeps the prompt small) and to
// the deterministic fallback (keeps an untended page from ballooning).
const maxGeneratedRelatedBOs = 6

// resolveRelatedBOs loads the Business Objects related to boID (via the
// catalog relationship graph, already filtered to genuine structural
// relationships - see businessobject_service.go's GetBusinessObjectRelationships)
// along with each one's own resolved fields, so the generator can reason
// about "this BO's real business context" rather than only its own columns.
// Returns an empty slice (never an error) when relationships/boRepo aren't
// wired or the lookup fails - a page can always still be generated from the
// primary BO alone.
func (h *PageStudioHandler) resolveRelatedBOs(ctx context.Context, tenantID uuid.UUID, boID string) []PageAIRelatedBO {
	if h.relationships == nil {
		return nil
	}
	rels, err := h.relationships.GetBusinessObjectRelationships(ctx, &security.Context{TenantID: tenantID.String()}, boID)
	if err != nil || rels == nil {
		return nil
	}

	var out []PageAIRelatedBO
	seen := map[string]bool{}
	for _, rel := range rels.RelatedObjects {
		if len(out) >= maxGeneratedRelatedBOs {
			break
		}
		if rel.TargetObjectID == "" || seen[rel.TargetObjectID] {
			continue
		}
		var bo struct {
			ID          string `db:"id"`
			BOKey       string `db:"bo_key"`
			DisplayName string `db:"bo_name"`
		}
		// GetBusinessObjectRelationships's targetObjectId is the related
		// object's catalog_node/driver_table id (business_objects.driver_table_id),
		// not business_objects.id itself - matching on id here always missed
		// (confirmed: order's "order_allocation" relationship reports
		// targetObjectId a7b99b57-... but order_allocation's own row id is
		// 0b5d5b5b-...; a7b99b57-... is its driver_table_id). The relationship
		// graph's target isn't always another BO at all (some catalog edges
		// point at raw tables) - a miss here just means this candidate is
		// skipped, not an error for the whole request.
		if err := h.db.GetContext(ctx, &bo, `
			SELECT id, bo_key, bo_name FROM public.business_objects WHERE driver_table_id = $1::uuid AND tenant_id = $2::uuid
		`, rel.TargetObjectID, tenantID); err != nil {
			continue
		}
		if seen[bo.ID] {
			continue
		}
		seen[bo.ID] = true

		terms, err := h.boRepo.GetBOTerms(bo.ID, "")
		if err != nil {
			continue
		}
		fields := make([]PageAIField, len(terms))
		for i, t := range terms {
			fields[i] = PageAIField{Key: t.TermKey, DisplayName: t.DisplayName, DataType: t.DataType, Role: t.Role}
		}
		out = append(out, PageAIRelatedBO{
			BOID: bo.ID, BOKey: bo.BOKey, DisplayName: bo.DisplayName,
			RelationshipType: rel.RelationshipType, Cardinality: rel.Cardinality,
			JoinCondition: rel.JoinCondition,
			Fields:        fields,
		})
	}
	return out
}

func deterministicPageSpec(pageKind, boName string, measureCount, dimensionCount int, relatedBOs []PageAIRelatedBO) (title, layout string, sections, filterBar []pageStudioGenerateSection) {
	childTables := func() []pageStudioGenerateSection {
		out := []pageStudioGenerateSection{}
		for _, rel := range relatedBOs {
			if len(out) >= 2 {
				break
			}
			out = append(out, pageStudioGenerateSection{BOKey: rel.BOKey, Type: "Table", Title: rel.DisplayName})
		}
		return out
	}
	switch pageKind {
	case "list":
		title = boName + " List"
		layout = "single-column"
		sections = []pageStudioGenerateSection{{Type: "Table", Title: boName + " Records"}}
		if dimensionCount > 0 {
			filterBar = []pageStudioGenerateSection{{Type: "Slicer", Title: "Filters"}}
		}
	case "detail":
		title = boName + " Detail"
		sections = append([]pageStudioGenerateSection{{Type: "Form", Title: boName}}, childTables()...)
		if len(sections) > 1 {
			layout = "two-column"
		} else {
			layout = "single-column"
		}
	case "master-detail":
		title = boName
		layout = "master-detail"
		sections = append([]pageStudioGenerateSection{
			{Type: "Table", Title: boName + " List"},
			{Type: "Form", Title: boName},
		}, childTables()...)
	default:
		title = boName
		if measureCount > 0 {
			sections = append(sections, pageStudioGenerateSection{Type: "KPIGroup", Title: "Key Metrics"})
		}
		if measureCount > 0 && dimensionCount > 0 {
			sections = append(sections, pageStudioGenerateSection{Type: "LineChart", Title: boName + " Trend"})
		}
		sections = append(sections, pageStudioGenerateSection{Type: "Table", Title: boName + " Records"})
		sections = append(sections, childTables()...)
		if len(sections) > 2 {
			layout = "dashboard-grid"
		} else if measureCount > 0 && dimensionCount > 0 {
			layout = "two-column"
		} else {
			layout = "single-column"
		}
	}
	return title, layout, sections, filterBar
}

// generate picks a small section mix (KPI/Chart/Table/Slicer), each bound
// to either the primary Business Object or one of its real related
// Business Objects, grounded in real resolved fields on both sides
// (h.boRepo.GetBOTerms, h.resolveRelatedBOs) - via Gemini when configured,
// falling back to a deterministic template otherwise so "Generate with AI"
// still produces a real, working starting page rather than erroring out
// when no LLM gateway is wired up.
func (h *PageStudioHandler) generate(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req pageStudioGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.BOID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	terms, err := h.boRepo.GetBOTerms(req.BOID, "")
	if err != nil {
		http.Error(w, "failed to load business object fields: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if len(terms) == 0 {
		http.Error(w, "this business object has no resolved fields to generate a page from", http.StatusUnprocessableEntity)
		return
	}

	fields := make([]PageAIField, len(terms))
	measureCount := 0
	dimensionCount := 0
	for i, t := range terms {
		fields[i] = PageAIField{Key: t.TermKey, DisplayName: t.DisplayName, DataType: t.DataType, Role: t.Role}
		if t.Role == "MEASURE" || t.Role == "CALCULATED" {
			measureCount++
		} else {
			dimensionCount++
		}
	}

	boName := req.BOName
	if boName == "" {
		boName = req.BOID
	}

	relatedBOs := h.resolveRelatedBOs(r.Context(), tenantID, req.BOID)

	pageKind := req.PageKind
	switch pageKind {
	case "list", "detail", "master-detail", "dashboard":
	default:
		pageKind = "dashboard"
	}

	resp := pageStudioGenerateResponse{Source: "template", PageKind: pageKind}
	if h.generateSpec != nil {
		spec, aiErr := h.generateSpec(r.Context(), boName, req.BOKey, req.Description, pageKind, fields, relatedBOs)
		if aiErr != nil {
			log.Printf("page-studio generate: gemini call failed, falling back to template: %v", aiErr)
		} else {
			resp.Title = spec.Title
			resp.LayoutTemplate = spec.LayoutTemplate
			if spec.PageKind != "" {
				resp.PageKind = spec.PageKind
			}
			resp.Sections = make([]pageStudioGenerateSection, len(spec.Sections))
			for i, s := range spec.Sections {
				resp.Sections[i] = pageStudioGenerateSection{BOKey: s.BOKey, Type: s.Type, Title: s.Title}
			}
			resp.FilterBar = make([]pageStudioGenerateSection, len(spec.FilterBar))
			for i, s := range spec.FilterBar {
				resp.FilterBar[i] = pageStudioGenerateSection{BOKey: s.BOKey, Type: s.Type, Title: s.Title}
			}
			resp.Source = "ai"
		}
	}

	if len(resp.Sections) == 0 {
		resp.Title, resp.LayoutTemplate, resp.Sections, resp.FilterBar = deterministicPageSpec(pageKind, boName, measureCount, dimensionCount, relatedBOs)
	}

	sectionUses := func(boKey string) bool {
		for _, s := range resp.Sections {
			if s.BOKey == boKey {
				return true
			}
		}
		for _, s := range resp.FilterBar {
			if s.BOKey == boKey {
				return true
			}
		}
		return false
	}
	for _, rel := range relatedBOs {
		if sectionUses(rel.BOKey) {
			resp.RelatedBusinessObjects = append(resp.RelatedBusinessObjects, pageStudioRelatedBO{
				BOID: rel.BOID, BOKey: rel.BOKey, DisplayName: rel.DisplayName,
				Cardinality: rel.Cardinality, JoinCondition: rel.JoinCondition,
			})
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *PageStudioHandler) list(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var pages []PageStudioPage
	gold := h.goldCopyID(r.Context())
	err := h.db.SelectContext(r.Context(), &pages, `
		SELECT `+pageColumns+`
		FROM page_definitions
		WHERE tenant_id = $1
		   OR (is_core = true AND tenant_id = $2)
		ORDER BY updated_at DESC
	`, tenantID, gold)
	if err != nil {
		http.Error(w, "failed to list pages: "+err.Error(), http.StatusInternalServerError)
		return
	}
	var adoptions map[uuid.UUID]pageAdoption
	if gold != uuid.Nil && tenantID != gold {
		if adoptions, err = h.adoptions(r.Context(), tenantID); err != nil {
			http.Error(w, "failed to load core page customizations: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}
	canEdit := h.canEditCore(r, tenantID)
	canCustomize := adoptions != nil && h.canCustomize(r, tenantID)
	cores := map[uuid.UUID]PageStudioPage{}
	clones := map[uuid.UUID]uuid.UUID{} // clone page id -> core page id
	for id, a := range adoptions {
		if a.CloneObjectID.Valid {
			clones[a.CloneObjectID.UUID] = id
		}
	}
	out := make([]PageStudioPage, 0, len(pages))
	for _, p := range pages {
		if p.IsCore && p.TenantID != tenantID {
			var a *pageAdoption
			if found, ok := adoptions[p.ID]; ok {
				a = &found
			}
			presentCore(&p, a, canCustomize)
			cores[p.ID] = p
		} else {
			p.Editable = !p.IsCore || canEdit
		}
		out = append(out, p)
	}
	placements := map[string][]PageMenuPlacement{}
	if nodes, err := loadMenuNodes(r.Context(), h.db, tenantID); err != nil {
		log.Printf("page-studio list: menu placements unavailable: %v", err)
	} else {
		placements = menuPlacements(nodes)
	}
	for i := range out {
		out[i].MenuPlacements = placements[out[i].Slug]
		if coreID, ok := clones[out[i].ID]; ok {
			if c, ok := cores[coreID]; ok {
				out[i].ClonedFrom = &PageCloneSource{PageID: c.ID, Name: c.Name, Version: int(adoptions[coreID].BaseVersion.Int64)}
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// get is the studio's read: a core page comes back even when this tenant
// has switched it off, with its customization state.
func (h *PageStudioHandler) get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	page, err := h.getOne(r, "id = $1 AND tenant_id = $2", id, tenantID)
	if err == nil {
		page.Editable = !page.IsCore || h.canEditCore(r, tenantID)
		h.markClone(r.Context(), tenantID, page)
		writeJSON(w, http.StatusOK, page)
		return
	}
	if err == sql.ErrNoRows {
		page, err = h.getOne(r, "id = $1 AND is_core = true AND tenant_id = $2", id, h.goldCopyID(r.Context()))
	}
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a, err := h.adoption(r.Context(), tenantID, page.ID)
	if err != nil {
		http.Error(w, "failed to load customization: "+err.Error(), http.StatusInternalServerError)
		return
	}
	presentCore(page, a, h.canCustomize(r, tenantID))
	writeJSON(w, http.StatusOK, page)
}

// getBySlug is the runtime read (PageBrowser): the tenant's own page at
// the slug (its clone, if it cloned a core page) wins; otherwise the core
// page as the tenant uses it. A core page the tenant switched off is not
// served.
func (h *PageStudioHandler) getBySlug(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	slug := chi.URLParam(r, "slug")
	page, err := h.getOne(r, "slug = $1 AND tenant_id = $2", slug, tenantID)
	if err == nil {
		page.Editable = !page.IsCore || h.canEditCore(r, tenantID)
		writeJSON(w, http.StatusOK, page)
		return
	}
	if err == sql.ErrNoRows {
		page, err = h.getOne(r, "slug = $1 AND is_core = true AND tenant_id = $2", slug, h.goldCopyID(r.Context()))
	}
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a, err := h.adoption(r.Context(), tenantID, page.ID)
	if err != nil {
		http.Error(w, "failed to load customization: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if a != nil && !a.Active {
		http.Error(w, "this page is not active in your environment", http.StatusNotFound)
		return
	}
	presentCore(page, a, false)
	writeJSON(w, http.StatusOK, page)
}

// markClone sets ClonedFrom when a tenant page is the clone of a core page.
func (h *PageStudioHandler) markClone(ctx context.Context, tenantID uuid.UUID, page *PageStudioPage) {
	var src struct {
		ID          uuid.UUID     `db:"id"`
		Name        string        `db:"name"`
		BaseVersion sql.NullInt64 `db:"base_version"`
	}
	err := h.db.GetContext(ctx, &src, `
		SELECT p.id, p.name, a.base_version
		FROM core_object_adoption a JOIN page_definitions p ON p.id = a.core_object_id
		WHERE a.tenant_id = $1 AND a.object_type = $2 AND a.clone_object_id = $3
	`, tenantID, corePageObjectType, page.ID)
	if err == nil {
		page.ClonedFrom = &PageCloneSource{PageID: src.ID, Name: src.Name, Version: int(src.BaseVersion.Int64)}
	}
}

func (h *PageStudioHandler) getOne(r *http.Request, where string, args ...interface{}) (*PageStudioPage, error) {
	var page PageStudioPage
	query := `SELECT ` + pageColumns + ` FROM page_definitions WHERE ` + where
	if err := h.db.GetContext(r.Context(), &page, query, args...); err != nil {
		return nil, err
	}
	return &page, nil
}

type pageStudioUpsertRequest struct {
	Name               string          `json:"name"`
	Slug               string          `json:"slug"`
	Description        string          `json:"description,omitempty"`
	Layout             json.RawMessage `json:"layout"`
	Tabs               json.RawMessage `json:"tabs,omitempty"`
	Components         json.RawMessage `json:"components"`
	DataSources        json.RawMessage `json:"dataSources"`
	PresentationEvents json.RawMessage `json:"presentationEvents,omitempty"`
	FilterBar          json.RawMessage `json:"filterBar,omitempty"`
	App                json.RawMessage `json:"app,omitempty"`
	Version            int             `json:"version"`
	// IsCore: nil = the default - core when the gold-copy admin creates
	// it (everything the gold copy authors, every MDM page, is core) and
	// never otherwise.
	IsCore *bool  `json:"isCore,omitempty"`
	Status string `json:"status,omitempty"`
}

func (h *PageStudioHandler) create(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	var req pageStudioUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Slug == "" {
		http.Error(w, "name and slug are required", http.StatusBadRequest)
		return
	}
	normalizeUpsertDefaults(&req)

	id := uuid.New()
	if req.Status == "" {
		req.Status = "draft"
	}
	isCore := (req.IsCore == nil || *req.IsCore) && h.canEditCore(r, tenantID)
	var page PageStudioPage
	err := h.db.GetContext(r.Context(), &page, `
		INSERT INTO page_definitions (id, tenant_id, name, slug, description, layout, tabs, components, data_sources, presentation_events, filter_bar, version, is_core, status, app_model)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING `+pageColumns, id, tenantID, req.Name, req.Slug, req.Description, req.Layout, req.Tabs, req.Components, req.DataSources, req.PresentationEvents, req.FilterBar, max(req.Version, 1), isCore, req.Status, nullableJSON(req.App))
	if err != nil {
		if isUniqueViolation(err) {
			http.Error(w, "a page with this slug already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to create page: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, page)
}

func (h *PageStudioHandler) update(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req pageStudioUpsertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Slug == "" {
		http.Error(w, "name and slug are required", http.StatusBadRequest)
		return
	}
	normalizeUpsertDefaults(&req)

	var existing struct {
		IsCore bool `db:"is_core"`
	}
	if err := h.db.GetContext(r.Context(), &existing, `SELECT is_core FROM page_definitions WHERE id = $1`, id); err == nil && existing.IsCore {
		if !h.canEditCore(r, tenantID) {
			http.Error(w, "core pages can only be changed by the gold-copy tenant admin", http.StatusForbidden)
			return
		}
	}

	if req.Status == "" {
		req.Status = "draft"
	}
	var page PageStudioPage
	err = h.db.GetContext(r.Context(), &page, `
		UPDATE page_definitions
		SET name = $1, slug = $2, description = $3, layout = $4, tabs = $5, components = $6,
		    data_sources = $7, presentation_events = $8, filter_bar = $9, status = $10, app_model = $13, version = version + 1, updated_at = NOW()
		WHERE id = $11 AND tenant_id = $12
		RETURNING `+pageColumns, req.Name, req.Slug, req.Description, req.Layout, req.Tabs, req.Components, req.DataSources, req.PresentationEvents, req.FilterBar, req.Status, id, tenantID, nullableJSON(req.App))
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "page not found", http.StatusNotFound)
			return
		}
		if isUniqueViolation(err) {
			http.Error(w, "a page with this slug already exists", http.StatusConflict)
			return
		}
		http.Error(w, "failed to update page: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// setStatus publishes or unpublishes a page the caller owns. Only the
// status changes - not the version - so publishing a core page is not a
// new core version and tenants see no upgrade for it. Core pages need the
// gold-copy admin, like any other core change.
func (h *PageStudioHandler) setStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || (req.Status != "draft" && req.Status != "published") {
		http.Error(w, `status must be "draft" or "published"`, http.StatusBadRequest)
		return
	}
	var isCore bool
	err = h.db.GetContext(r.Context(), &isCore, `SELECT is_core FROM page_definitions WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err == sql.ErrNoRows {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if isCore && !h.canEditCore(r, tenantID) {
		http.Error(w, "core pages can only be published by the gold-copy tenant admin", http.StatusForbidden)
		return
	}
	var page PageStudioPage
	err = h.db.GetContext(r.Context(), &page, `
		UPDATE page_definitions SET status = $1, updated_at = NOW()
		WHERE id = $2 AND tenant_id = $3
		RETURNING `+pageColumns, req.Status, id, tenantID)
	if err != nil {
		http.Error(w, "failed to change status: "+err.Error(), http.StatusInternalServerError)
		return
	}
	page.Editable = true
	writeJSON(w, http.StatusOK, page)
}

func (h *PageStudioHandler) delete(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var existing struct {
		IsCore bool `db:"is_core"`
	}
	if err := h.db.GetContext(r.Context(), &existing, `SELECT is_core FROM page_definitions WHERE id = $1 AND tenant_id = $2`, id, tenantID); err == nil && existing.IsCore && !h.canEditCore(r, tenantID) {
		http.Error(w, "core pages can only be deleted by the gold-copy tenant admin", http.StatusForbidden)
		return
	}
	res, err := h.db.ExecContext(r.Context(), `DELETE FROM page_definitions WHERE id = $1 AND tenant_id = $2`, id, tenantID)
	if err != nil {
		http.Error(w, "failed to delete page: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		http.Error(w, "page not found", http.StatusNotFound)
		return
	}
	// Deleting a tenant's clone of a core page puts the core page back.
	if _, err := h.db.ExecContext(r.Context(), `DELETE FROM core_object_adoption WHERE tenant_id = $1 AND object_type = $2 AND clone_object_id = $3`, tenantID, corePageObjectType, id); err != nil {
		log.Printf("page-studio delete: clearing clone record for %s: %v", id, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

// emptyPageLayoutJSON is a well-formed, empty PageLayout ({root, nodes} -
// see types/pageStudio.ts) - the same shape PageEditor.tsx synthesizes
// client-side when a draft has no filter bar yet. NOT `{}`: `filterBar`'s
// default used to be the bare `{}` this handler and the filter_bar column
// (20261016_017_page_definitions_filter_bar.up.sql) both defaulted to -
// valid JSON, but missing PageLayout's required `root`/`nodes` fields, so
// LayoutCanvas.tsx's renderNode(layout.root) crashed the whole editor on
// any page saved with no explicit filter bar (the common case). Fixed here
// on the write path; PageEditor.tsx also validates shape, not just
// truthiness, on read, so already-saved `{}` rows self-heal without a data
// migration. See HANDOFF_REPORT_BUILDER_SPINE_PLAN.md "Order Detail crash
// fix". `layout`'s column default has the same `[]`-not-`{root,nodes}`
// defect (from before PageLayout's shape was pinned down) but is unfixed
// here deliberately - every reproduced case goes through `tabs`, not the
// legacy singular `layout` field, so fixing it blind without a reproduced
// case is out of scope for this pass.
var emptyPageLayoutJSON = json.RawMessage(`{"root":"filter_root","nodes":{"filter_root":{"id":"filter_root","type":"Row","children":[]}}}`)

// normalizeUpsertDefaults fills the jsonb columns with valid empty JSON
// when the client omits them, since page_definitions.layout/components/
// data_sources are all NOT NULL.
func normalizeUpsertDefaults(req *pageStudioUpsertRequest) {
	if len(req.Layout) == 0 {
		req.Layout = json.RawMessage(`[]`)
	}
	if len(req.Tabs) == 0 {
		req.Tabs = json.RawMessage(`[]`)
	}
	if len(req.Components) == 0 {
		req.Components = json.RawMessage(`[]`)
	}
	if len(req.DataSources) == 0 {
		req.DataSources = json.RawMessage(`[]`)
	}
	if len(req.PresentationEvents) == 0 {
		req.PresentationEvents = json.RawMessage(`[]`)
	}
	if len(req.FilterBar) == 0 {
		req.FilterBar = emptyPageLayoutJSON
	}
}

// nullableJSON stores an absent (or JSON null) app model as SQL NULL rather
// than the jsonb value null, so "no app model" has one representation.
func nullableJSON(raw json.RawMessage) interface{} {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return []byte(raw)
}

func (h *PageStudioHandler) goldCopyID(ctx context.Context) uuid.UUID {
	return goldcopy.ResolveTenantID(ctx, h.db)
}

func (h *PageStudioHandler) canEditCore(r *http.Request, tenantID uuid.UUID) bool {
	auth, ok := security.AuthInfoFromContext(r.Context())
	if !ok || !auth.IsGlobalAdmin {
		return false
	}
	gold := h.goldCopyID(r.Context())
	if gold == uuid.Nil {
		return true
	}
	return tenantID == gold
}
