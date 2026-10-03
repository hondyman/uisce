package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/lakehouse/registry"
	"github.com/hondyman/uisce/backend/internal/security"
)

// SystemLakehouseHandler is the System-area API for configuring and provisioning a
// tenant's one Iceberg warehouse (ADR-032): the per-tenant audit retention, which
// has no default, and the provisioning request. Global admins only. The target
// tenant is always explicit in the path and is checked to exist; it is never
// inferred from the caller.
//
// Mounted inside the /api router:
//
//	GET  /api/system/lakehouses                              list tenants + state
//	GET  /api/system/tenants/{tenantID}/lakehouse            one tenant's entry
//	PUT  /api/system/tenants/{tenantID}/lakehouse            set/extend retention
//	GET  /api/system/tenants/{tenantID}/lakehouse/audit      audit trail + chain check
//	POST /api/system/tenants/{tenantID}/lakehouse/provision  start provisioning
type SystemLakehouseHandler struct {
	reg  lakehouseRegistry
	prov LakehouseProvisioner
}

// lakehouseRegistry is the part of registry.Store the handler uses.
type lakehouseRegistry interface {
	Get(ctx context.Context, tenantID uuid.UUID) (*registry.Config, error)
	SetRetention(ctx context.Context, tenantID uuid.UUID, days int, actor registry.Actor) (*registry.Config, error)
	List(ctx context.Context, q string, limit, offset int) ([]registry.Config, int, error)
	Audit(ctx context.Context, tenantID uuid.UUID, limit int) ([]registry.AuditEntry, error)
	VerifyAudit(ctx context.Context, tenantID uuid.UUID) (*int64, error)
	Record(ctx context.Context, tenantID uuid.UUID, actor registry.Actor, action string, before, after any) error
}

// LakehouseProvisioner starts provisioning of a tenant's bucket, key, credential and
// warehouse. It may be nil, in which case provisioning answers 503: the
// configuration endpoints work without it.
type LakehouseProvisioner interface {
	// StartProvision must be idempotent per tenant and return the workflow id. It
	// returns ErrProvisionInProgress if one is already running.
	StartProvision(ctx context.Context, tenantID uuid.UUID, actor registry.Actor) (workflowID string, err error)
}

// ErrProvisionInProgress is returned by a LakehouseProvisioner when provisioning
// for the tenant is already running.
var ErrProvisionInProgress = errors.New("lakehouse provisioning already in progress")

func NewSystemLakehouseHandler(db *sql.DB, prov LakehouseProvisioner) *SystemLakehouseHandler {
	return &SystemLakehouseHandler{reg: registry.NewStore(db), prov: prov}
}

func (h *SystemLakehouseHandler) RegisterRoutes(r chi.Router) {
	r.Get("/system/lakehouses", h.list)
	r.Get("/system/tenants/{tenantID}/lakehouse", h.get)
	r.Put("/system/tenants/{tenantID}/lakehouse", h.put)
	r.Get("/system/tenants/{tenantID}/lakehouse/audit", h.audit)
	r.Post("/system/tenants/{tenantID}/lakehouse/provision", h.provision)
}

// admin returns the caller if they are a global admin, and has already written the
// failure response if not.
func (h *SystemLakehouseHandler) admin(w http.ResponseWriter, r *http.Request) (registry.Actor, bool) {
	auth, ok := security.RequireAuth(w, r)
	if !ok {
		return registry.Actor{}, false
	}
	if !auth.IsGlobalAdmin {
		writeLakehouseError(w, http.StatusForbidden, "forbidden", "global admin role required")
		return registry.Actor{}, false
	}
	role := "global_admin"
	if len(auth.Roles) > 0 {
		role = auth.Roles[0]
	}
	return registry.Actor{ID: auth.UserID, Role: role}, true
}

func tenantParam(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "tenantID"))
	if err != nil || id == uuid.Nil {
		writeLakehouseError(w, http.StatusBadRequest, "invalid_tenant_id", "tenant id must be a UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *SystemLakehouseHandler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.admin(w, r); !ok {
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	items, total, err := h.reg.List(r.Context(), q.Get("q"), limit, offset)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	writeLakehouseJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func (h *SystemLakehouseHandler) get(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.admin(w, r); !ok {
		return
	}
	id, ok := tenantParam(w, r)
	if !ok {
		return
	}
	cfg, err := h.reg.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeLakehouseJSON(w, http.StatusOK, cfg)
}

type putLakehouseRequest struct {
	AuditRetentionDays *int `json:"audit_retention_days"`
}

func (h *SystemLakehouseHandler) put(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.admin(w, r)
	if !ok {
		return
	}
	id, ok := tenantParam(w, r)
	if !ok {
		return
	}
	// Unknown fields are rejected so a client cannot try to supply the warehouse or
	// bucket name; those are derived from the tenant id.
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10))
	dec.DisallowUnknownFields()
	var req putLakehouseRequest
	if err := dec.Decode(&req); err != nil {
		writeLakehouseError(w, http.StatusBadRequest, "invalid_body", "body must be {\"audit_retention_days\": <integer>}")
		return
	}
	if req.AuditRetentionDays == nil {
		writeLakehouseError(w, http.StatusBadRequest, "retention_required", "audit_retention_days is required; there is no default")
		return
	}
	cfg, err := h.reg.SetRetention(r.Context(), id, *req.AuditRetentionDays, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeLakehouseJSON(w, http.StatusOK, cfg)
}

func (h *SystemLakehouseHandler) audit(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.admin(w, r); !ok {
		return
	}
	id, ok := tenantParam(w, r)
	if !ok {
		return
	}
	if _, err := h.reg.Get(r.Context(), id); err != nil { // existence check
		h.fail(w, r, err)
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.reg.Audit(r.Context(), id, limit)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	broken, err := h.reg.VerifyAudit(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if entries == nil {
		entries = []registry.AuditEntry{}
	}
	body := map[string]any{"entries": entries, "chain_intact": broken == nil}
	if broken != nil {
		body["first_broken_id"] = *broken
	}
	writeLakehouseJSON(w, http.StatusOK, body)
}

func (h *SystemLakehouseHandler) provision(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.admin(w, r)
	if !ok {
		return
	}
	id, ok := tenantParam(w, r)
	if !ok {
		return
	}
	cfg, err := h.reg.Get(r.Context(), id)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if !cfg.Configured || cfg.AuditRetentionDays == nil {
		writeLakehouseError(w, http.StatusConflict, "retention_required",
			"set the tenant's audit retention before provisioning; it cannot be shortened later")
		return
	}
	if cfg.Provisioned {
		writeLakehouseError(w, http.StatusConflict, "already_provisioned", "this tenant's warehouse is already provisioned")
		return
	}
	if h.prov == nil {
		writeLakehouseError(w, http.StatusServiceUnavailable, "provisioner_unavailable",
			"lakehouse provisioning is not configured on this server")
		return
	}

	// Record the request first: the audit says what was asked, whether or not the
	// workflow then starts.
	if err := h.reg.Record(r.Context(), id, actor, "provision_requested", nil,
		map[string]any{"audit_retention_days": *cfg.AuditRetentionDays}); err != nil {
		h.fail(w, r, err)
		return
	}
	wfID, err := h.prov.StartProvision(r.Context(), id, actor)
	if errors.Is(err, ErrProvisionInProgress) {
		writeLakehouseError(w, http.StatusConflict, "provision_in_progress", "provisioning is already running for this tenant")
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeLakehouseJSON(w, http.StatusAccepted, map[string]any{"workflow_id": wfID, "tenant_id": id.String()})
}

// fail maps a store error to a response. Internal errors are logged, not returned:
// the body must not leak SQL or infrastructure detail.
func (h *SystemLakehouseHandler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, registry.ErrTenantNotFound):
		writeLakehouseError(w, http.StatusNotFound, "tenant_not_found", "no such tenant")
	case errors.Is(err, registry.ErrInvalidRetention):
		writeLakehouseError(w, http.StatusBadRequest, "invalid_retention", err.Error())
	case errors.Is(err, registry.ErrRetentionLowered):
		writeLakehouseError(w, http.StatusConflict, "retention_lowered",
			"audit retention can only be extended once set; compliance retention cannot be shortened")
	default:
		slog.Error("system lakehouse request failed", "path", r.URL.Path, "error", err)
		writeLakehouseError(w, http.StatusInternalServerError, "internal_error", "internal error")
	}
}

func writeLakehouseJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeLakehouseError(w http.ResponseWriter, status int, code, msg string) {
	writeLakehouseJSON(w, status, map[string]string{"code": code, "error": msg})
}
