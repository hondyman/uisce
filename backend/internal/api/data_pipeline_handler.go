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
	"time"

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
	// catalog returns the platform as the caller's tenant sees it (nil:
	// structural checks only). assistant is nil when no LLM is configured.
	catalog   func(r *http.Request, tenant string) datapipeline.PlatformCatalog
	assistant *datapipeline.Assistant
	scheduler datapipeline.Scheduler // nil: schedules need Temporal
}

// WithScheduler enables pipeline schedules.
func (h *DataPipelineHandler) WithScheduler(s datapipeline.Scheduler) *DataPipelineHandler {
	h.scheduler = s
	return h
}

// WithGrounding enables grounded checks and, with an assistant, /assist.
func (h *DataPipelineHandler) WithGrounding(cat func(*http.Request, string) datapipeline.PlatformCatalog, a *datapipeline.Assistant) *DataPipelineHandler {
	h.catalog, h.assistant = cat, a
	return h
}

// check is structural validation plus grounding against the tenant's platform.
func (h *DataPipelineHandler) check(r *http.Request, tenant string, spec *datapipeline.Spec) []datapipeline.Issue {
	var cat datapipeline.PlatformCatalog
	if h.catalog != nil {
		cat = h.catalog(r, tenant)
	}
	issues := datapipeline.Check(r.Context(), cat, spec)
	if issues == nil {
		issues = []datapipeline.Issue{}
	}
	return issues
}

func NewDataPipelineHandler(store *datapipeline.Store, deps datapipeline.Deps, tc client.Client) *DataPipelineHandler {
	return &DataPipelineHandler{store: store, deps: deps, temporal: tc}
}

func (h *DataPipelineHandler) RegisterRoutes(r chi.Router) {
	r.Route("/data-pipelines", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Post("/validate", h.validate)
		r.Post("/assist", h.assist)
		r.Post("/preview", h.preview)
		r.Post("/suggest-mapping", h.suggestMapping)
		r.Get("/node-types", h.nodeTypes)
		r.Get("/staging-tables", h.stagingTables)
		r.Get("/files", h.listFiles)
		r.Post("/files/upload", h.uploadFile)
		r.Post("/files/profile", h.profileFile)
		r.Get("/runs/{runId}", h.getRun)
		r.Get("/{id}", h.get)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Post("/{id}/runs", h.startRun)
		r.Get("/{id}/runs", h.listRuns)
		r.Get("/{id}/schedule", h.getSchedule)
		r.Put("/{id}/schedule", h.putSchedule)
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
	id := chi.URLParam(r, "id")
	if err := h.store.Delete(r.Context(), t, id); err != nil {
		dpStoreError(w, err)
		return
	}
	if h.scheduler != nil {
		if err := h.scheduler.Apply(r.Context(), t, id, nil); err != nil {
			log.Printf("[data-pipelines] removing schedule of deleted pipeline %s: %v", id, err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *DataPipelineHandler) validate(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
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
	issues := h.check(r, t, &spec)
	dpJSON(w, http.StatusOK, map[string]any{"valid": len(issues) == 0, "issues": issues})
}

// --- runs --------------------------------------------------------------------

// errHasProblems carries grounded problems that stop a run or preview.
type errHasProblems struct{ issues []datapipeline.Issue }

func (e *errHasProblems) Error() string { return "the pipeline has problems to fix first" }

// startRunCore checks the saved pipeline and queues a run (Temporal, or
// in-process without it). Shared by HTTP and MCP.
func (h *DataPipelineHandler) startRunCore(r *http.Request, t, id string) (string, error) {
	d, err := h.store.Get(r.Context(), t, id)
	if err != nil {
		return "", err
	}
	if issues := h.check(r, t, &d.Spec); len(issues) > 0 {
		return "", &errHasProblems{issues}
	}
	runID, err := h.store.CreateRun(r.Context(), t, d)
	if err != nil {
		return "", err
	}
	in := datapipeline.RunInput{TenantID: t, RunID: runID}
	if h.temporal != nil {
		if _, err := h.temporal.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
			ID: "data-pipeline-run-" + runID, TaskQueue: datapipeline.TaskQueue,
		}, datapipeline.Workflow, in); err != nil {
			_ = h.store.FinishRun(context.WithoutCancel(r.Context()), t, runID, nil, fmt.Errorf("could not start: %w", err))
			return "", err
		}
		return runID, nil
	}
	// Dev without Temporal: run in the background.
	go func() {
		if _, err := h.store.Execute(context.Background(), h.deps, t, runID); err != nil {
			log.Printf("[data-pipelines] run %s: %v", runID, err)
		}
	}()
	return runID, nil
}

func (h *DataPipelineHandler) startRun(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	runID, err := h.startRunCore(r, t, chi.URLParam(r, "id"))
	var probs *errHasProblems
	switch {
	case errors.As(err, &probs):
		dpJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": probs.Error(), "issues": probs.issues})
	case err != nil:
		dpStoreError(w, err)
	default:
		dpJSON(w, http.StatusAccepted, map[string]string{"run_id": runID, "status": "queued"})
	}
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

// previewCore dry-runs a spec on its first rows. Shared by HTTP and MCP.
func (h *DataPipelineHandler) previewCore(r *http.Request, t string, spec datapipeline.Spec, rows, sampleRows int) (map[string]any, error) {
	if spec.Version == 0 {
		spec.Version = datapipeline.SpecVersion
	}
	if issues := h.check(r, t, &spec); len(issues) > 0 {
		return nil, &errHasProblems{issues}
	}
	if rows <= 0 || rows > previewMaxRows {
		rows = 100
	}
	if sampleRows <= 0 || sampleRows > 50 {
		sampleRows = 20
	}
	rec := &previewRecorder{}
	sum, err := datapipeline.Run(r.Context(), &spec, &datapipeline.RunContext{
		TenantID: t, RunID: "preview-" + uuid.NewString(), MaxRows: rows, SampleRows: sampleRows, DryRun: true,
	}, h.deps, rec)
	resp := map[string]any{"summary": sum, "rejects": rec.out}
	if err != nil {
		resp["error"] = err.Error()
	}
	return resp, nil
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
	resp, err := h.previewCore(r, t, b.Spec, b.Rows, b.SampleRows)
	var probs *errHasProblems
	if errors.As(err, &probs) {
		dpJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "fix these first", "issues": probs.issues})
		return
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

// stagingTables lists staging tables a pipeline can load (those with the
// load-tracking columns) and their loadable columns, as mapping targets.
func (h *DataPipelineHandler) stagingTables(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.tenant(w, r); !ok {
		return
	}
	if h.deps.StagingDB == nil {
		dpError(w, http.StatusServiceUnavailable, "the staging database is not configured for this environment")
		return
	}
	tables, err := datapipeline.StagingTables(r.Context(), h.deps.StagingDB)
	if err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusOK, tables)
}

// assist answers a question about, or proposes a change to, the pipeline.
// Proposals are grounded and checked before they are returned; nothing is
// saved - the analyst applies it on the canvas.
func (h *DataPipelineHandler) assist(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	if h.assistant == nil || h.catalog == nil {
		dpError(w, http.StatusServiceUnavailable, "the AI assistant is not configured for this environment (Admin > LLM Config)")
		return
	}
	var req datapipeline.AssistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	out, err := h.assistant.Respond(r.Context(), h.catalog(r, t), req)
	if err != nil {
		dpError(w, http.StatusBadGateway, err.Error())
		return
	}
	dpJSON(w, http.StatusOK, out)
}

// scheduleView is a schedule plus its next run times.
type scheduleView struct {
	Schedule *datapipeline.Schedule `json:"schedule"`
	NextRuns []time.Time            `json:"next_runs,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

func viewOf(sc *datapipeline.Schedule) scheduleView {
	v := scheduleView{Schedule: sc}
	if sc != nil && sc.Enabled {
		v.NextRuns, _ = sc.Next(5, time.Now())
	}
	return v
}

func (h *DataPipelineHandler) getSchedule(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	sc, err := h.store.GetSchedule(r.Context(), t, chi.URLParam(r, "id"))
	if err != nil {
		dpStoreError(w, err)
		return
	}
	dpJSON(w, http.StatusOK, viewOf(sc))
}

// setScheduleCore validates, applies the live Temporal schedule, then
// stores it. Shared by HTTP and MCP.
func (h *DataPipelineHandler) setScheduleCore(ctx context.Context, t, id string, sc datapipeline.Schedule) (scheduleView, error) {
	if _, err := h.store.Get(ctx, t, id); err != nil {
		return scheduleView{}, err
	}
	if sc.Enabled {
		if err := sc.Validate(); err != nil {
			return scheduleView{}, &errBadSchedule{err}
		}
		if h.scheduler == nil {
			return scheduleView{}, &errBadSchedule{fmt.Errorf("schedules need Temporal, which is not configured for this environment")}
		}
	}
	if h.scheduler != nil {
		if err := h.scheduler.Apply(ctx, t, id, &sc); err != nil {
			return scheduleView{}, fmt.Errorf("could not update the schedule: %w", err)
		}
	}
	if err := h.store.SetSchedule(ctx, t, id, &sc); err != nil {
		return scheduleView{}, err
	}
	return viewOf(&sc), nil
}

type errBadSchedule struct{ err error }

func (e *errBadSchedule) Error() string { return e.err.Error() }

func (h *DataPipelineHandler) putSchedule(w http.ResponseWriter, r *http.Request) {
	t, ok := h.tenant(w, r)
	if !ok {
		return
	}
	var sc datapipeline.Schedule
	if err := json.NewDecoder(r.Body).Decode(&sc); err != nil {
		dpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}
	v, err := h.setScheduleCore(r.Context(), t, chi.URLParam(r, "id"), sc)
	var bad *errBadSchedule
	switch {
	case errors.As(err, &bad):
		dpError(w, http.StatusBadRequest, bad.Error())
	case err != nil:
		dpStoreError(w, err)
	default:
		dpJSON(w, http.StatusOK, v)
	}
}
