package provisioning

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/migrations"
	"github.com/hondyman/uisce/backend/internal/security"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

// ProvisioningTaskQueue is the queue the provisioning saga runs on: the one cmd/worker polls.
// The handler used to start it on "tenant-provisioning", which no worker polls.
const ProvisioningTaskQueue = "bp_queue"

// provisioningExecutionTimeout bounds one run. It now includes applying tenant migrations and a
// probe, so it is longer than the 15 minutes the earlier saga fitted in.
const provisioningExecutionTimeout = 60 * time.Minute

// codePattern is the tenant code: it becomes part of a database name that is later used to build
// CREATE/DROP DATABASE, so it is a strict identifier, never free text. "tenant_" + code must fit
// in Postgres' 63 bytes.
var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,40}$`)

type ProvisioningHandler struct {
	temporalClient client.Client
	controlDB      *sql.DB
	workflowIDBase string
	logger         *zap.SugaredLogger
}

func NewProvisioningHandler(temporalClient client.Client, controlDB *sql.DB, logger *zap.SugaredLogger) *ProvisioningHandler {
	return &ProvisioningHandler{
		temporalClient: temporalClient,
		controlDB:      controlDB,
		workflowIDBase: "tenant-provisioning",
		logger:         logger,
	}
}

// RegisterAdminRoutes mounts the routes that are safe to expose: creating a tenant and reading a
// run's status, global admins only (each handler enforces that itself, so the protection does not
// depend on where this is mounted). Re-provisioning an existing instance is not mounted.
//
//	POST /system/tenants/provision
//	GET  /system/tenants/{tenantID}/provision/{workflow_id}
func (h *ProvisioningHandler) RegisterAdminRoutes(r chi.Router) {
	r.Post("/system/tenants/provision", h.ProvisionTenant)
	r.Get("/system/tenants/{tenantID}/provision/{workflow_id}", h.GetProvisioningStatus)
}

// admin returns the caller if they are a global admin; otherwise it has already answered.
func admin(w http.ResponseWriter, r *http.Request) (security.AuthInfo, bool) {
	auth, ok := security.RequireAuth(w, r)
	if !ok {
		return security.AuthInfo{}, false
	}
	if !auth.IsGlobalAdmin {
		http.Error(w, "global admin role required", http.StatusForbidden)
		return security.AuthInfo{}, false
	}
	return auth, true
}

func (h *ProvisioningHandler) RegisterRoutes(r chi.Router) {
	r.Post("/v1/tenants/provision", h.ProvisionTenant)
	r.Get("/v1/tenants/{tenant_id}/provision/{workflow_id}", h.GetProvisioningStatus)
	r.Post("/v1/tenants/{tenant_id}/instances/{instance_id}/provision", h.ProvisionInstance)
}

func (h *ProvisioningHandler) ProvisionTenant(w http.ResponseWriter, r *http.Request) {
	caller, ok := admin(w, r)
	if !ok {
		return
	}
	var req ProvisionTenantRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.App != "" {
		// Validate against the same rule the runner applies, so a bad app is a 400 here and
		// not a run that fails at the migration step.
		if err := (migrations.Target{TenantID: uuid.NewString(), App: req.App}).Validate(); err != nil {
			http.Error(w, "invalid app: must match ^[a-z][a-z0-9_]{0,31}$", http.StatusBadRequest)
			return
		}
	}
	if req.TemplateDatasourceID != "" {
		if req.App == "" {
			http.Error(w, "template_datasource_id requires app", http.StatusBadRequest)
			return
		}
		if _, err := uuid.Parse(req.TemplateDatasourceID); err != nil {
			http.Error(w, "invalid template_datasource_id: must be a uuid", http.StatusBadRequest)
			return
		}
	}
	if req.TenantCode != "" && !codePattern.MatchString(req.TenantCode) {
		http.Error(w, "invalid tenant_code: must match ^[a-z][a-z0-9_]{0,40}$", http.StatusBadRequest)
		return
	}

	if req.TenantName == "" {
		http.Error(w, "tenant_name is required", http.StatusBadRequest)
		return
	}
	if req.InstanceName == "" {
		http.Error(w, "instance_name is required", http.StatusBadRequest)
		return
	}

	if h.temporalClient == nil {
		http.Error(w, "Temporal client not configured", http.StatusServiceUnavailable)
		return
	}

	tenantCode := req.TenantCode
	if tenantCode == "" {
		tenantCode = GenerateTenantCode(req.TenantName)
		if existingCode := h.checkExistingTenantCode(tenantCode); existingCode != "" {
			tenantCode = tenantCode + "_" + strings.ToLower(uuid.New().String()[:8])
		}
	} else {
		if existingCode := h.checkExistingTenantCode(tenantCode); existingCode != "" {
			http.Error(w, fmt.Sprintf("tenant with code %s already exists", tenantCode), http.StatusConflict)
			return
		}
	}

	if existingName := h.checkExistingTenantName(req.TenantName); existingName != "" {
		http.Error(w, fmt.Sprintf("tenant with name %s already exists", req.TenantName), http.StatusConflict)
		return
	}

	goldCopyTenantID, goldCopyInstanceID, goldCopyDatabase, err := h.resolveGoldCopy()
	if err != nil {
		h.logger.Errorf("Failed to resolve gold copy: %v", err)
		http.Error(w, "failed to resolve gold copy configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if !codePattern.MatchString(tenantCode) {
		// A generated code can start with a digit or underscore; it must still be a safe name.
		http.Error(w, "could not derive a valid tenant_code from tenant_name; supply tenant_code", http.StatusBadRequest)
		return
	}
	databaseName := fmt.Sprintf("tenant_%s", tenantCode)
	namespace := tenantCode
	tenantID := uuid.New().String()
	instanceID := uuid.New().String()

	workflowID := fmt.Sprintf("%s-%s-%s", h.workflowIDBase, tenantCode, uuid.New().String()[:8])

	workflowInput := ProvisioningWorkflowInput{
		TenantID:           tenantID,
		TenantName:         req.TenantName,
		TenantCode:         tenantCode,
		InstanceID:         instanceID,
		InstanceName:       req.InstanceName,
		GoldCopyTenantID:   goldCopyTenantID,
		GoldCopyInstanceID: goldCopyInstanceID,
		GoldCopyDatabase:   goldCopyDatabase,
		DatabaseName:       databaseName,
		LakekeeperNS:      namespace,
		RequesterID:        caller.UserID,
		App:                req.App,
		TemplateDatasourceID: req.TemplateDatasourceID,
	}

	h.logger.Infof("Starting tenant provisioning workflow: %s", workflowID)

	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                ProvisioningTaskQueue,
		WorkflowExecutionTimeout:  provisioningExecutionTimeout,
		WorkflowTaskTimeout:       5 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}

	we, err := h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, "TenantInstanceProvisioningWorkflowFn", workflowInput)
	if err != nil {
		h.logger.Errorf("Failed to start workflow: %v", err)
		http.Error(w, "failed to start provisioning workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ProvisionTenantResponse{
		WorkflowID:   we.GetID(),
		TenantID:     tenantID,
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		LakekeeperNS: namespace,
		Status:       "provisioning",
		StartedAt:    time.Now(),
	}

	h.logger.Infof("Workflow started: %s (RunID: %s)", we.GetID(), we.GetRunID())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ProvisioningHandler) GetProvisioningStatus(w http.ResponseWriter, r *http.Request) {
	if _, ok := admin(w, r); !ok {
		return
	}
	workflowID := chi.URLParam(r, "workflow_id")
	if workflowID == "" {
		http.Error(w, "workflow_id is required", http.StatusBadRequest)
		return
	}
	// Only provisioning runs: this must not describe an arbitrary workflow by id.
	if !strings.HasPrefix(workflowID, h.workflowIDBase+"-") {
		http.Error(w, "not a provisioning workflow id", http.StatusBadRequest)
		return
	}
	if h.temporalClient == nil {
		http.Error(w, "Temporal client not configured", http.StatusServiceUnavailable)
		return
	}

	h.logger.Infof("Querying workflow status: %s", workflowID)

	describeResp, err := h.temporalClient.DescribeWorkflowExecution(r.Context(), workflowID, "")
	if err != nil {
		h.logger.Errorf("Failed to describe workflow: %v", err)
		http.Error(w, "failed to get workflow status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	statusStr := describeResp.WorkflowExecutionInfo.Status.String()
	var status string
	switch statusStr {
	case "Running":
		status = "provisioning"
	case "Completed":
		status = "completed"
	case "Failed":
		status = "failed"
	case "Canceled":
		status = "canceled"
	case "Terminated":
		status = "terminated"
	default:
		status = "unknown"
	}

	resp := ProvisioningStatus{
		WorkflowID: workflowID,
		Status:     status,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ProvisioningHandler) ProvisionInstance(w http.ResponseWriter, r *http.Request) {
	if _, ok := admin(w, r); !ok {
		return
	}
	tenantID := chi.URLParam(r, "tenant_id")
	instanceID := chi.URLParam(r, "instance_id")

	if tenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	if instanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}

	h.logger.Infof("Provisioning instance %s for tenant %s", instanceID, tenantID)

	tenant, err := h.getTenant(tenantID)
	if err != nil {
		http.Error(w, "tenant not found: "+err.Error(), http.StatusNotFound)
		return
	}

	instance, err := h.getInstance(instanceID)
	if err != nil {
		http.Error(w, "instance not found: "+err.Error(), http.StatusNotFound)
		return
	}

	goldCopyTenantID, goldCopyInstanceID, goldCopyDatabase, err := h.resolveGoldCopy()
	if err != nil {
		h.logger.Errorf("Failed to resolve gold copy: %v", err)
		http.Error(w, "failed to resolve gold copy: "+err.Error(), http.StatusInternalServerError)
		return
	}

	databaseName := fmt.Sprintf("tenant_%s", strings.ToLower(tenant.Code))
	namespace := tenant.Code

	workflowID := fmt.Sprintf("%s-%s-%s", h.workflowIDBase, tenant.Code, uuid.New().String()[:8])

	workflowInput := ProvisioningWorkflowInput{
		TenantID:           tenant.ID,
		TenantName:         tenant.Name,
		TenantCode:         tenant.Code,
		InstanceID:         instance.ID,
		InstanceName:       instance.InstanceName,
		GoldCopyTenantID:   goldCopyTenantID,
		GoldCopyInstanceID: goldCopyInstanceID,
		GoldCopyDatabase:   goldCopyDatabase,
		DatabaseName:       databaseName,
		LakekeeperNS:      namespace,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:                       workflowID,
		TaskQueue:                ProvisioningTaskQueue,
		WorkflowExecutionTimeout:  provisioningExecutionTimeout,
		WorkflowTaskTimeout:       5 * time.Minute,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}

	we, err := h.temporalClient.ExecuteWorkflow(r.Context(), workflowOptions, "TenantInstanceProvisioningWorkflowFn", workflowInput)
	if err != nil {
		h.logger.Errorf("Failed to start workflow: %v", err)
		http.Error(w, "failed to start provisioning workflow: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := ProvisionTenantResponse{
		WorkflowID:   we.GetID(),
		TenantID:     tenantID,
		InstanceID:   instanceID,
		DatabaseName: databaseName,
		LakekeeperNS: namespace,
		Status:       "provisioning",
		StartedAt:    time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *ProvisioningHandler) checkExistingTenantCode(code string) string {
	var exists string
	err := h.controlDB.QueryRowContext(context.Background(),
		`SELECT code FROM public.tenants WHERE code = $1 LIMIT 1`, code).Scan(&exists)
	if err == sql.ErrNoRows {
		return ""
	}
	return exists
}

func (h *ProvisioningHandler) checkExistingTenantName(name string) string {
	var exists string
	err := h.controlDB.QueryRowContext(context.Background(),
		`SELECT name FROM public.tenants WHERE LOWER(name) = LOWER($1) LIMIT 1`, name).Scan(&exists)
	if err == sql.ErrNoRows {
		return ""
	}
	return exists
}

func (h *ProvisioningHandler) resolveGoldCopy() (tenantID, instanceID, database string, err error) {
	// No fallback: a gold copy with no database_name used to resolve to "alpha", which would
	// have cloned the control plane's own schema into a new tenant's database.
	query := `
		SELECT t.id, ti.id, COALESCE(t.database_name, '')
		FROM public.tenants t
		JOIN public.tenant_instance ti ON ti.tenant_id = t.id
		WHERE t.gold_copy = true
		LIMIT 1
	`
	err = h.controlDB.QueryRowContext(context.Background(), query).Scan(&tenantID, &instanceID, &database)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to resolve gold copy: %w", err)
	}
	if database == "" || database == "alpha" {
		return "", "", "", fmt.Errorf("gold copy tenant %s has no dedicated database_name (got %q); refusing to clone from the control plane", tenantID, database)
	}
	return tenantID, instanceID, database, nil
}

type TenantInfo struct {
	ID   string
	Name string
	Code string
}

type InstanceInfo struct {
	ID           string
	InstanceName string
}

func (h *ProvisioningHandler) getTenant(tenantID string) (*TenantInfo, error) {
	var t TenantInfo
	err := h.controlDB.QueryRowContext(context.Background(),
		`SELECT id, name, COALESCE(code, '') FROM public.tenants WHERE id = $1`, tenantID).Scan(&t.ID, &t.Name, &t.Code)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (h *ProvisioningHandler) getInstance(instanceID string) (*InstanceInfo, error) {
	var i InstanceInfo
	err := h.controlDB.QueryRowContext(context.Background(),
		`SELECT id, instance_name FROM public.tenant_instance WHERE id = $1`, instanceID).Scan(&i.ID, &i.InstanceName)
	if err != nil {
		return nil, err
	}
	return &i, nil
}
