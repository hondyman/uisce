package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// reportBORelationshipsService is the one method of
// metadata.BusinessObjectService this handler needs - a narrow interface
// declared at point of use, the same seam page_studio_handler.go's own
// boRelationshipsService uses (that one lives in package handlers and
// can't be referenced here even though internal/api already imports
// internal/handlers, since it's unexported there).
type reportBORelationshipsService interface {
	GetBusinessObjectRelationships(ctx context.Context, secCtx *security.Context, boID string) (*metadata.BORelationshipsResponse, error)
}

// ReportGenerationHandler is a sibling of ReportHandler, not an extension
// of it: AI generation never reads or writes report_templates, it only
// returns a spec (exactly like Page Studio's PageStudioHandler.generate),
// so it gets its own small handler rather than growing ReportHandler's
// constructor - which has 12 existing test call sites that would
// otherwise all need updating for a dependency they don't use.
type ReportGenerationHandler struct {
	db            *sqlx.DB
	boRepo        *boresolver.PostgresBORepository
	relationships reportBORelationshipsService
	geminiClient  *GeminiClient
}

func NewReportGenerationHandler(db *sqlx.DB, boRepo *boresolver.PostgresBORepository, relationships reportBORelationshipsService, geminiClient *GeminiClient) *ReportGenerationHandler {
	return &ReportGenerationHandler{db: db, boRepo: boRepo, relationships: relationships, geminiClient: geminiClient}
}

func (h *ReportGenerationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/reports/generate", h.generate)
}

type reportGenerationRequest struct {
	BOID        string `json:"boId"`
	BOKey       string `json:"boKey"`
	BOName      string `json:"boName"`
	Description string `json:"description"`
	ReportKind  string `json:"reportKind"`
}

type reportGenerationElement struct {
	BOKey      string   `json:"boKey"`
	Type       string   `json:"type"`
	Title      string   `json:"title"`
	Dimensions []string `json:"dimensions,omitempty"`
	Measures   []string `json:"measures,omitempty"`
}

type reportGenerationRelatedBO struct {
	BOID          string `json:"boId"`
	BOKey         string `json:"boKey"`
	DisplayName   string `json:"displayName"`
	Cardinality   string `json:"cardinality,omitempty"`
	JoinCondition string `json:"joinCondition,omitempty"`
}

type reportGenerationField struct {
	TermNodeID  string `json:"termNodeId"`
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
	DataType    string `json:"dataType"`
	Role        string `json:"role"`
}

type reportGenerationResponse struct {
	Title                  string                      `json:"title"`
	ReportKind             string                      `json:"reportKind"`
	RelatedBusinessObjects []reportGenerationRelatedBO `json:"relatedBusinessObjects"`
	Elements               []reportGenerationElement   `json:"elements"`
	// Fields keyed by boKey ("" = primary) - the field metadata
	// (dataType/role/termNodeId) the frontend needs to build a real
	// dataBinding for each generated element, since the response only
	// says WHICH termNodeIds an element uses, not their type/role.
	Fields map[string][]reportGenerationField `json:"fields"`
	Source string                             `json:"source"` // "ai" or "template"
}

// maxGeneratedReportRelatedBOs mirrors Page Studio's maxGeneratedRelatedBOs.
const maxGeneratedReportRelatedBOs = 6

// resolveRelatedBOs duplicates (not imports) page_studio_handler.go's
// version of this lookup, including its documented driver_table_id vs
// business_objects.id quirk - small and self-contained, and this
// codebase already tolerates this class of duplication across the
// api/handlers package boundary (e.g. writeJSONError/writeJSON each have
// independent definitions in both packages today).
func (h *ReportGenerationHandler) resolveRelatedBOs(ctx context.Context, tenantID uuid.UUID, boID string) []reportGenerationRelatedBO {
	if h.relationships == nil {
		return nil
	}
	rels, err := h.relationships.GetBusinessObjectRelationships(ctx, &security.Context{TenantID: tenantID.String()}, boID)
	if err != nil || rels == nil {
		return nil
	}

	var out []reportGenerationRelatedBO
	seen := map[string]bool{}
	for _, rel := range rels.RelatedObjects {
		if len(out) >= maxGeneratedReportRelatedBOs {
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
		if err := h.db.GetContext(ctx, &bo, `
			SELECT id, bo_key, bo_name FROM public.business_objects WHERE driver_table_id = $1::uuid AND tenant_id = $2::uuid
		`, rel.TargetObjectID, tenantID); err != nil {
			continue
		}
		if seen[bo.ID] {
			continue
		}
		seen[bo.ID] = true
		out = append(out, reportGenerationRelatedBO{
			BOID: bo.ID, BOKey: bo.BOKey, DisplayName: bo.DisplayName,
			Cardinality: rel.Cardinality, JoinCondition: rel.JoinCondition,
		})
	}
	return out
}

// relatedBOFields resolves one related BO's own fields via GetBOTerms,
// used both to ground the Gemini prompt and to populate the response's
// per-BO Fields map the frontend needs.
func (h *ReportGenerationHandler) relatedBOFields(boID string) []reportGenerationField {
	terms, err := h.boRepo.GetBOTerms(boID, "")
	if err != nil {
		return nil
	}
	fields := make([]reportGenerationField, len(terms))
	for i, t := range terms {
		fields[i] = reportGenerationField{TermNodeID: t.TermNodeID, Key: t.TermKey, DisplayName: t.DisplayName, DataType: t.DataType, Role: t.Role}
	}
	return fields
}

// deterministicReportSpec is a zero-LLM-calls fallback, used whenever
// geminiClient is nil, errors, or returns zero usable elements - so
// "Generate with AI" always produces a real, working element set. Reuses
// the same "first 4 dims + first 2 measures" convention
// handleAddToolboxItem already uses client-side, so the fallback and the
// manual-drag path agree on what a default binding looks like.
//
// Honesty note (carried into any UI copy that describes this): "list" and
// "master-detail" produce structurally identical output today - there's
// no cross-filtering (masterFilter) for report elements yet, that's
// Phase 5, unbuilt. Not a regression this introduces; the manual
// drag-and-drop path has the same ceiling.
func deterministicReportSpec(reportKind, boName string, fields []reportGenerationField, relatedBOs []reportGenerationRelatedBO) (title string, elements []reportGenerationElement) {
	var dims, measures []string
	for _, f := range fields {
		if f.Role == "MEASURE" || f.Role == "CALCULATED" {
			if len(measures) < 2 {
				measures = append(measures, f.TermNodeID)
			}
		} else if len(dims) < 4 {
			dims = append(dims, f.TermNodeID)
		}
	}
	childTables := func() []reportGenerationElement {
		out := []reportGenerationElement{}
		for _, rel := range relatedBOs {
			if len(out) >= 2 {
				break
			}
			out = append(out, reportGenerationElement{BOKey: rel.BOKey, Type: "table", Title: rel.DisplayName})
		}
		return out
	}
	switch reportKind {
	case "list":
		title = boName + " List"
		elements = []reportGenerationElement{{Type: "table", Title: boName + " Records", Dimensions: dims, Measures: measures}}
		if len(dims) > 0 {
			elements = append(elements, reportGenerationElement{Type: "slicer", Title: "Filter", Dimensions: dims[:1]})
		}
		elements = append(elements, childTables()...)
	case "detail":
		title = boName + " Detail"
		elements = append([]reportGenerationElement{{Type: "form", Title: boName}}, childTables()...)
	case "master-detail":
		title = boName
		elements = append([]reportGenerationElement{
			{Type: "table", Title: boName + " List", Dimensions: dims, Measures: measures},
			{Type: "form", Title: boName},
		}, childTables()...)
	default:
		title = boName
		if len(measures) > 0 {
			elements = append(elements, reportGenerationElement{Type: "gauge", Title: "Key Metric", Measures: measures[:1]})
		}
		if len(measures) > 0 && len(dims) > 0 {
			elements = append(elements, reportGenerationElement{Type: "chart", Title: boName + " Trend", Dimensions: dims[:1], Measures: measures[:1]})
		}
		elements = append(elements, reportGenerationElement{Type: "table", Title: boName + " Records", Dimensions: dims, Measures: measures})
		elements = append(elements, childTables()...)
	}
	return title, elements
}

// generate picks a small widget mix with specific field bindings for a
// Business Object, grounded in real resolved fields on both the primary
// and related BOs (h.boRepo.GetBOTerms, h.resolveRelatedBOs) - via Gemini
// when configured, falling back to a deterministic template otherwise so
// generation always produces a real, working starting element set rather
// than erroring out when no LLM gateway is wired up.
func (h *ReportGenerationHandler) generate(w http.ResponseWriter, r *http.Request) {
	tenantID, _, _, err := resolveReportAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var req reportGenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.BOID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	fields := h.relatedBOFields(req.BOID)
	if len(fields) == 0 {
		http.Error(w, "this business object has no resolved fields to generate a report from", http.StatusUnprocessableEntity)
		return
	}

	boName := req.BOName
	if boName == "" {
		boName = req.BOID
	}

	relatedBOs := h.resolveRelatedBOs(r.Context(), tenantID, req.BOID)
	fieldsByBOKey := map[string][]reportGenerationField{"": fields}
	relatedFieldsForPrompt := make([]ReportGenerationRelatedBO, 0, len(relatedBOs))
	for _, rel := range relatedBOs {
		relFields := h.relatedBOFields(rel.BOID)
		fieldsByBOKey[rel.BOKey] = relFields
		promptFields := make([]ReportGenerationField, len(relFields))
		for i, f := range relFields {
			promptFields[i] = ReportGenerationField{TermNodeID: f.TermNodeID, Key: f.Key, DisplayName: f.DisplayName, DataType: f.DataType, Role: f.Role}
		}
		relatedFieldsForPrompt = append(relatedFieldsForPrompt, ReportGenerationRelatedBO{
			BOID: rel.BOID, BOKey: rel.BOKey, DisplayName: rel.DisplayName,
			Cardinality: rel.Cardinality, Fields: promptFields,
		})
	}

	reportKind := req.ReportKind
	switch reportKind {
	case "list", "detail", "master-detail", "dashboard":
	default:
		reportKind = "dashboard"
	}

	resp := reportGenerationResponse{Source: "template", ReportKind: reportKind}
	if h.geminiClient != nil {
		promptFields := make([]ReportGenerationField, len(fields))
		for i, f := range fields {
			promptFields[i] = ReportGenerationField{TermNodeID: f.TermNodeID, Key: f.Key, DisplayName: f.DisplayName, DataType: f.DataType, Role: f.Role}
		}
		spec, aiErr := h.geminiClient.GenerateReportSpec(r.Context(), boName, req.BOKey, req.Description, reportKind, promptFields, relatedFieldsForPrompt)
		if aiErr != nil {
			log.Printf("report generate: gemini call failed, falling back to template: %v", aiErr)
		} else {
			resp.Title = spec.Title
			if spec.ReportKind != "" {
				resp.ReportKind = spec.ReportKind
			}
			resp.Elements = make([]reportGenerationElement, len(spec.Elements))
			for i, e := range spec.Elements {
				resp.Elements[i] = reportGenerationElement{BOKey: e.BOKey, Type: e.Type, Title: e.Title, Dimensions: e.Dimensions, Measures: e.Measures}
			}
			resp.Source = "ai"
		}
	}

	if len(resp.Elements) == 0 {
		resp.Title, resp.Elements = deterministicReportSpec(reportKind, boName, fields, relatedBOs)
	}

	elementUses := func(boKey string) bool {
		for _, e := range resp.Elements {
			if e.BOKey == boKey {
				return true
			}
		}
		return false
	}
	usedFields := map[string][]reportGenerationField{"": fieldsByBOKey[""]}
	for _, rel := range relatedBOs {
		if elementUses(rel.BOKey) {
			resp.RelatedBusinessObjects = append(resp.RelatedBusinessObjects, rel)
			usedFields[rel.BOKey] = fieldsByBOKey[rel.BOKey]
		}
	}
	resp.Fields = usedFields

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
