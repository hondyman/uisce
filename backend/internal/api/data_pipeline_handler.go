package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.temporal.io/sdk/client"

	"github.com/hondyman/uisce/backend/internal/datapipeline"
	"github.com/hondyman/uisce/backend/internal/security"
)

// DataPipelineHandler serves /data-pipelines: the visual loader's API.
// Every call is scoped to the caller's tenant (extractTenantUUIDFromRequest);
// there is no fallback tenant.
type DataPipelineHandler struct {
	store    *datapipeline.Store
	deps     datapipeline.Deps
	temporal client.Client // nil: runs execute in-process (dev)
}

func NewDataPipelineHandler(store *datapipeline.Store, deps datapipeline.Deps, tc client.Client) *DataPipelineHandler {
	return &DataPipelineHandler{store: store, deps: deps, temporal: tc}
}

func (h *DataPipelineHandler) RegisterRoutes(r chi.Router) {
	r.Route("/data-pipelines", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/validate", h.validate)
		r.Post("/preview", h.preview)
		r.Post("/suggest-mapping", h.suggestMapping)
		r.Get("/node-types", h.nodeTypes)
		r.Get("/files", h.listFiles)
		r.Post("/files/upload", h.uploadFile)
		r.Post("/files/profile", h.profileFile)
		r.Get("/runs/{runId}", h.getRun)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/runs", h.startRun)
		r.Get("/{id}/runs", h.listRuns)
	})
}

// --- helpers ---------------------------------------------------------------

func (h *DataPipelineHandler) tenant(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := extractTenantUUIDFromRequest(r)
	if err != nil {
		status := http.StatusUnauthorized
		var te *tenantResolutionError
		if errors.As(err, &te) {
			status = te.status
		}
		dpError(w, status, err.Error())
		return "", false
	}
	return id.String(), true
}

func dpUser(r *http.Request) string {
	if a, ok := security.AuthInfoFromContext(r.Context()); ok && a.UserID != "" {
		return a.UserID
	}
	return ""
}

func dpJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func dpError(w http.ResponseWriter, status int, msg string) {
	dpJSON(w, status, map[string]string{"error": msg})
}

func dpStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, datapipeline.ErrNotFound) {
		dpError(w, http.StatusNotFound, "not found")
		return
	}
	log.Printf("[data-pipelines] %v", err)
	dpError(w, http.StatusInternalServerError, "internal error")
}

// validationIssues turns Spec.Validate's errors into a list the canvas can
// attach to nodes: `node "x": ...` errors carry the node id.
type dpIssue struct {
	NodeID  string `json:"node_id,omitempty"`
	Message string `json:"message"`
}

func dpIssues(errs []error) []dpIssue {
	out := make([]dpIssue, 0, len(errs))
	for _, e := range errs {
		msg := e.Error()
		iss := dpIssue{Message: msg}
		if strings.HasPrefix(msg, `node "`) {
			if end := strings.Index(msg[6:], `"`); end >= 0 {
				iss.NodeID = msg[6 : 6+end]
				iss.Message = strings.TrimPrefix(msg[6+end+1:], ": ")
			}
		}
		out = append(out, iss)
	}
	return out
}

// --- definitions -------------------------------------------------------------

func (h *DataPipelineHandler) list(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	defs, err := h.store.List(r.Context(), t)
	if err != nil {
		dpStoreError(w, err)
		return
	}
	if defs == nil {
		defs = []datapipeline.Definition{}
	}
	dpJSON(w, http.StatusOK, defs)
}

func (h *DataPipelineHandler) get(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	d, err := h.store.Get(r.Context(), t, chi.URLParam(r, "id"))
	if err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusOK, d)
}

type definitionBody struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Spec        datapipeline.Spec `json:"spec"`
}

func decodeDefinition(w http.ResponseWriter, r *http.Request) (*definitionBody, bool) {
	var b definitionBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return nil, false
	}
	if strings.TrimSpace(b.Name) == "" {
		dpError(w, http.StatusBadRequest, "name is required")
		return nil, false
	}
	if b.Spec.Version == 0 {
		b.Spec.Version = datapipeline.SpecVersion
	}
	return &b, true
}

func (h *DataPipelineHandler) create(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	b, ok := decodeDefinition(w, r)
	if !ok {
		return
	}
	d := &datapipeline.Definition{Name: b.Name, Description: b.Description, Spec: b.Spec}
	if err := h.store.Save(r.Context(), t, dpUser(r), d); err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusCreated, d)
}

func (h *DataPipelineHandler) update(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		dpError(w, http.StatusNotFound, "not found")
		return
	}
	b, ok := decodeDefinition(w, r)
	if !ok {
		return
	}
	d := &datapipeline.Definition{ID: id, Name: b.Name, Description: b.Description, Spec: b.Spec}
	if err := h.store.Save(r.Context(), t, dpUser(r), d); err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusOK, d)
}

func (h *DataPipelineHandler) delete(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), t, chi.URLParam(r, "id")); err != nil {
		dpStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DataPipelineHandler) validate(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tenant(w, r); !ok {
		return
	}
	var spec datapipeline.Spec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if spec.Version == 0 {
		spec.Version = datapipeline.SpecVersion
	}
	issues := dpIssues(spec.Validate())
	dpJSON(w, http.StatusOK, map[string]any{"valid": len(issues) == 0, "issues": issues})
}

// --- runs --------------------------------------------------------------------

func (h *DataPipelineHandler) startRun(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	d, err := h.store.Get(r.Context(), t, chi.URLParam(r, "id"))
	if err != nil {
		dpStoreError(w, err)
		return
	}
	if issues := dpIssues(d.Spec.Validate()); len(issues) > 0 {
		dpJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "the pipeline has problems to fix before it can run", "issues": issues})
		return
	}
	runID, err := h.store.CreateRun(r.Context(), t, d)
	if err != nil {
		dpStoreError(w, err)
		return
	}
	in := datapipeline.RunInput{TenantID: t, RunID: runID}
	if h.temporal != nil {
		_, err = h.temporal.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
			ID: "data-pipeline-run-" + runID, TaskQueue: datapipeline.TaskQueue,
		}, datapipeline.Workflow, in)
		if err != nil {
			_ = h.store.FinishRun(context.WithoutCancel(r.Context()), t, runID, nil, fmt.Errorf("could not start: %w", err))
			dpStoreError(w, err)
			return
		}
	} else {
		// Dev without Temporal: run in the background.
		go func() {
			if _, err := h.store.Execute(context.Background(), h.deps, t, runID); err != nil {
				log.Printf("[data-pipelines] run %s: %v", runID, err)
			}
		}()
	}
	dpJSON(w, http.StatusAccepted, map[string]string{"run_id": runID, "status": "queued"})
}

func (h *DataPipelineHandler) listRuns(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	limit := 50
	if n, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && n > 0 && n <= 500 {
		limit = n
	}
	runs, err := h.store.ListRuns(r.Context(), t, chi.URLParam(r, "id"), limit)
	if err != nil {
		dpStoreError(w, err)
		return
	}
	if runs == nil {
		runs = []datapipeline.RunRecord{}
	}
	dpJSON(w, http.StatusOK, runs)
}

func (h *DataPipelineHandler) getRun(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	run, err := h.store.GetRun(r.Context(), t, chi.URLParam(r, "runId"))
	if err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusOK, run)
}

// --- design-time helpers -------------------------------------------------------

const previewMaxRows = 500

type previewBody struct {
	Spec       datapipeline.Spec `json:"spec"`
	Rows       int               `json:"rows"`
	SampleRows int               `json:"sample_rows"`
}

type previewReject struct {
	NodeID string `json:"node_id"`
	Row    int    `json:"row"`
	Field  string `json:"field,omitempty"`
	Reason string `json:"reason"`
	Kind   string `json:"kind"` // error | warning
}

type previewRecorder struct{ out []previewReject }

func (p *previewRecorder) NodeDone(context.Context, datapipeline.NodeStats) {}
func (p *previewRecorder) Rejected(_ context.Context, r datapipeline.Reject) {
	p.out = append(p.out, previewReject{r.NodeID, r.Row.Num, r.Field, r.Reason, "error"})
}
func (p *previewRecorder) Warned(_ context.Context, r datapipeline.Reject) {
	p.out = append(p.out, previewReject{r.NodeID, r.Row.Num, r.Field, r.Reason, "warning"})
}

// preview runs the spec on the first rows as a dry run: nothing is written,
// BO rows are still judged by the rule engine.
func (h *DataPipelineHandler) preview(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	var b previewBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	if b.Spec.Version == 0 {
		b.Spec.Version = datapipeline.SpecVersion
	}
	if issues := dpIssues(b.Spec.Validate()); len(issues) > 0 {
		dpJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "fix these first", "issues": issues})
		return
	}
	if b.Rows <= 0 || b.Rows > previewMaxRows {
		b.Rows = 100
	}
	if b.SampleRows <= 0 || b.SampleRows > 50 {
		b.SampleRows = 20
	}
	rec := &previewRecorder{}
	sum, err := datapipeline.Run(r.Context(), &b.Spec, &datapipeline.RunContext{
		TenantID: t, RunID: "preview-" + uuid.NewString(), MaxRows: b.Rows, SampleRows: b.SampleRows, DryRun: true,
	}, h.deps, rec)
	resp := map[string]any{"summary": sum, "rejects": rec.out}
	if err != nil {
		resp["error"] = err.Error()
	}
	dpJSON(w, http.StatusOK, resp)
}

func (h *DataPipelineHandler) suggestMapping(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tenant(w, r); !ok {
		return
	}
	var b struct {
		Source  []datapipeline.Column      `json:"source"`
		Targets []datapipeline.TargetField `json:"targets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	dpJSON(w, http.StatusOK, datapipeline.SuggestMapping(b.Source, b.Targets))
}

func (h *DataPipelineHandler) nodeTypes(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tenant(w, r); !ok {
		return
	}
	dpJSON(w, http.StatusOK, datapipeline.Palette(h.deps))
}

// --- files ---------------------------------------------------------------------

func (h *DataPipelineHandler) files(w http.ResponseWriter) bool {
	if h.deps.Files == nil {
		dpError(w, http.StatusServiceUnavailable, "the file engine is not configured for this environment")
		return false
	}
	return true
}

func (h *DataPipelineHandler) listFiles(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok || !h.files(w) {
		return
	}
	folder := r.URL.Query().Get("folder")
	if folder == "" {
		folder = "uploads"
	}
	p, err := datapipeline.TenantPath(t, folder)
	if err != nil {
		dpError(w, http.StatusBadRequest, err.Error())
		return
	}
	entries, err := h.deps.Files.List(r.Context(), p)
	if err != nil {
		dpError(w, http.StatusBadGateway, err.Error())
		return
	}
	for i := range entries {
		entries[i].Path = datapipeline.TenantRelative(t, entries[i].Path)
	}
	if entries == nil {
		entries = []datapipeline.FileEntry{}
	}
	dpJSON(w, http.StatusOK, entries)
}

// uploadFile streams the request body to uploads/<name> for the tenant.
func (h *DataPipelineHandler) uploadFile(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok || !h.files(w) {
		return
	}
	name := r.URL.Query().Get("name")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		dpError(w, http.StatusBadRequest, "name must be a plain file name")
		return
	}
	rel := "uploads/" + name
	p, err := datapipeline.TenantPath(t, rel)
	if err != nil {
		dpError(w, http.StatusBadRequest, err.Error())
		return
	}
	n, err := h.deps.Files.Upload(r.Context(), p, r.Body)
	if err != nil {
		dpError(w, http.StatusBadGateway, err.Error())
		return
	}
	dpJSON(w, http.StatusCreated, map[string]any{"uri": rel, "bytes": n})
}

type profileBody struct {
	URI       string `json:"uri"`
	Format    string `json:"format"`
	Delimiter string `json:"delimiter"`
	HasHeader *bool  `json:"has_header"`
	CountRows bool   `json:"count_rows"`
}

// profileFile previews a file and proposes a column contract for it.
func (h *DataPipelineHandler) profileFile(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok || !h.files(w) {
		return
	}
	var b profileBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	p, err := datapipeline.TenantPath(t, b.URI)
	if err != nil {
		dpError(w, http.StatusBadRequest, err.Error())
		return
	}
	if b.Format == "" {
		b.Format = datapipeline.GuessFormat(b.URI)
	}
	hasHeader := b.HasHeader == nil || *b.HasHeader
	prof, err := h.deps.Files.Profile(r.Context(), datapipeline.FileSpec{URI: p, Format: b.Format, Delimiter: b.Delimiter, HasHeader: hasHeader}, 20, b.CountRows)
	if err != nil {
		dpError(w, http.StatusBadRequest, err.Error())
		return
	}
	dpJSON(w, http.StatusOK, map[string]any{
		"format":    b.Format,
		"columns":   datapipeline.ContractFromProfile(prof),
		"sample":    prof.Sample,
		"row_count": prof.RowCount,
	})
}
