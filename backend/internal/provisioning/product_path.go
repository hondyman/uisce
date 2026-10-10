package provisioning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdktemporal "go.temporal.io/sdk/temporal"

	"github.com/hondyman/uisce/backend/internal/security"
)

// The product path: "create tenant X in region R, register product P with label L". The request
// is checked by Describe (the same check the wizard prompts from), and the run it starts has ids
// derived from the request, so sending the same request twice finds the first run.

// DescribeParameters answers what a partly filled request still needs, and what a complete one would
// create. It creates nothing.
//
//	POST /system/tenants/provision/describe
func (h *ProvisioningHandler) DescribeParameters(w http.ResponseWriter, r *http.Request) {
	if _, ok := admin(w, r); !ok {
		return
	}
	var req ProvisionTenantRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	d, err := Describe(r.Context(), req, h.catalog)
	if err != nil {
		h.logger.Errorf("describe tenant parameters: %v", err)
		http.Error(w, "could not check the request", http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusOK, d)
}

func isWorkflowNotFound(err error) bool {
	var nf *serviceerror.NotFound
	return errors.As(err, &nf)
}

// tenantTaken checks the code and the name, and fails closed: a database error is an error, never
// "free".
func (h *ProvisioningHandler) tenantTaken(ctx context.Context, code, name string) (bool, error) {
	var n int
	if err := h.controlDB.QueryRowContext(ctx,
		`SELECT count(*) FROM public.tenants WHERE code = $1 OR LOWER(name) = LOWER($2)`, code, name).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (h *ProvisioningHandler) provisionProducts(w http.ResponseWriter, r *http.Request, caller security.AuthInfo, req ProvisionTenantRequest) {
	if req.App != "" || req.TemplateDatasourceID != "" || req.StructureFromGoldCopy {
		http.Error(w, "app and the structure options come from the products; do not set them with products", http.StatusBadRequest)
		return
	}
	if h.temporalClient == nil {
		http.Error(w, "Temporal client not configured", http.StatusServiceUnavailable)
		return
	}
	d, err := Describe(r.Context(), req, h.catalog)
	if err != nil {
		h.logger.Errorf("check provisioning request: %v", err)
		http.Error(w, "could not check the request", http.StatusInternalServerError)
		return
	}
	if !d.Complete {
		writeJSONStatus(w, http.StatusBadRequest, d)
		return
	}
	n, plan := *d.Normalized, *d.Plan
	db := plan.Databases[0]
	workflowID, tenantID, instanceID := deterministicIDs(h.workflowIDBase, n)
	resp := ProvisionTenantResponse{
		WorkflowID: workflowID, TenantID: tenantID, InstanceID: instanceID,
		DatabaseName: db.Database, LakekeeperNS: n.TenantCode, Status: "provisioning", StartedAt: time.Now(), Plan: &plan,
	}

	// The same request again is the same run: report it instead of starting another.
	desc, derr := h.temporalClient.DescribeWorkflowExecution(r.Context(), workflowID, "")
	switch {
	case derr == nil:
		switch desc.GetWorkflowExecutionInfo().GetStatus() {
		case enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING:
			writeJSONStatus(w, http.StatusOK, resp)
			return
		case enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED:
			http.Error(w, "this tenant was already provisioned by an earlier request", http.StatusConflict)
			return
		}
		// Failed, cancelled, terminated or timed out: the earlier run compensated, so a new one may start.
	case !isWorkflowNotFound(derr):
		h.logger.Errorf("look up provisioning run %s: %v", workflowID, derr)
		http.Error(w, "could not check for an earlier run", http.StatusBadGateway)
		return
	}

	taken, err := h.tenantTaken(r.Context(), n.TenantCode, n.TenantName)
	if err != nil {
		h.logger.Errorf("check for an existing tenant: %v", err)
		http.Error(w, "could not check for an existing tenant", http.StatusInternalServerError)
		return
	}
	if taken {
		http.Error(w, "a tenant with this code or name already exists", http.StatusConflict)
		return
	}
	goldTenant, goldInstance, goldDB, err := h.resolveGoldCopy()
	if err != nil {
		h.logger.Errorf("Failed to resolve gold copy: %v", err)
		http.Error(w, "failed to resolve gold copy configuration: "+err.Error(), http.StatusInternalServerError)
		return
	}

	in := ProvisioningWorkflowInput{
		TenantID: tenantID, TenantName: n.TenantName, TenantCode: n.TenantCode,
		InstanceID: instanceID, InstanceName: n.InstanceName,
		GoldCopyTenantID: goldTenant, GoldCopyInstanceID: goldInstance, GoldCopyDatabase: goldDB,
		DatabaseName: db.Database, LakekeeperNS: n.TenantCode, RequesterID: caller.UserID,
		// The product is the app, and its structure comes from the datasource the gold copy marks for it.
		App: db.Product, StructureFromGoldCopy: true,
		Region: plan.Region, ClusterHost: plan.Host, ClusterPort: plan.Port,
		ProductCodes: []string{db.Product}, Seed: true,
	}
	h.logger.Infof("Starting tenant provisioning workflow: %s (database %s, region %s)", workflowID, db.Database, plan.Region)

	_, err = h.temporalClient.ExecuteWorkflow(r.Context(), client.StartWorkflowOptions{
		ID: workflowID, TaskQueue: ProvisioningTaskQueue,
		WorkflowExecutionTimeout: provisioningExecutionTimeout, WorkflowTaskTimeout: 5 * time.Minute,
		WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		RetryPolicy: &sdktemporal.RetryPolicy{
			InitialInterval: time.Second, BackoffCoefficient: 2.0, MaximumInterval: 5 * time.Minute, MaximumAttempts: 3,
		},
	}, "TenantInstanceProvisioningWorkflowFn", in)
	var started *serviceerror.WorkflowExecutionAlreadyStarted
	if err != nil && !errors.As(err, &started) {
		h.logger.Errorf("Failed to start workflow: %v", err)
		http.Error(w, fmt.Sprintf("failed to start provisioning workflow: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSONStatus(w, http.StatusOK, resp)
}
