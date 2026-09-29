package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// RulePorter is the handler-facing surface of the rule porter.
// Declared as an interface so handler tests can stub it without a database.
type RulePorter interface {
	ExportRules(ctx context.Context, tenantID, boName, domain, originFilter string) (*models.RuleBundle, error)
	Preflight(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool) (*models.RuleImportReport, error)
	ImportRules(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool, idempotencyKeyHeader string) (*models.RuleImportReport, error)
}

func isBypassGovernance(r *http.Request) bool {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		return false
	}
	if claims.IsCoreAdmin || jwtmiddleware.HasRole(claims, "global_admin") || jwtmiddleware.HasRole(claims, "RULE_IMPORT_BYPASS_GOVERNANCE") {
		return true
	}
	return false
}

func decodeImportRequest(r *http.Request) (*models.RuleImportRequest, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, fmt.Errorf("empty request body")
	}

	contentType := r.Header.Get("Content-Type")
	isYAML := strings.Contains(contentType, "yaml") || strings.Contains(contentType, "yml")

	if !isYAML {
		var req models.RuleImportRequest
		if err := json.Unmarshal(body, &req); err == nil && (len(req.Bundle.Rules) > 0 || req.Bundle.Checksum != "" || req.Bundle.BundleVersion != "") {
			return &req, nil
		}
		var bundle models.RuleBundle
		if err := json.Unmarshal(body, &bundle); err == nil {
			return &models.RuleImportRequest{
				Bundle:          bundle,
				OverwritePolicy: models.ImportOverwriteFail,
			}, nil
		}
	}

	// Try YAML -> intermediate generic -> JSON -> struct
	var raw interface{}
	if err := yaml.Unmarshal(body, &raw); err == nil {
		jsonBytes, err := json.Marshal(raw)
		if err == nil {
			var req models.RuleImportRequest
			if err := json.Unmarshal(jsonBytes, &req); err == nil && (len(req.Bundle.Rules) > 0 || req.Bundle.Checksum != "" || req.Bundle.BundleVersion != "") {
				return &req, nil
			}
			var bundle models.RuleBundle
			if err := json.Unmarshal(jsonBytes, &bundle); err == nil {
				return &models.RuleImportRequest{
					Bundle:          bundle,
					OverwritePolicy: models.ImportOverwriteFail,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("could not parse import request or bundle from JSON or YAML payload")
}

// handleExportRules exports rules as a portable bundle (JSON or YAML).
func (h *ValidationRuleHandler) handleExportRules(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}
	boName := r.URL.Query().Get("bo_name")
	domain := r.URL.Query().Get("domain")
	origin := r.URL.Query().Get("origin")
	format := r.URL.Query().Get("format")

	bundle, err := h.porter.ExportRules(r.Context(), tenantID.String(), boName, domain, origin)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/export failed: %v", err)
		http.Error(w, "export failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if format == "yaml" || format == "yml" || r.Header.Get("Accept") == "application/x-yaml" || r.Header.Get("Accept") == "text/yaml" {
		w.Header().Set("Content-Type", "application/x-yaml")
		jsonBytes, err := json.Marshal(bundle)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to marshal bundle to json: %v", err)
			http.Error(w, "export failed", http.StatusInternalServerError)
			return
		}
		var raw interface{}
		_ = json.Unmarshal(jsonBytes, &raw)
		if err := yaml.NewEncoder(w).Encode(raw); err != nil {
			logging.GetLogger().Sugar().Errorf("failed to encode yaml export: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(bundle); err != nil {
		logging.GetLogger().Sugar().Errorf("failed to encode json export: %v", err)
	}
}

// handlePreviewImport runs a dry-run preflight check and returns diffs/errors.
func (h *ValidationRuleHandler) handlePreviewImport(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}

	bypass := isBypassGovernance(r)

	req, err := decodeImportRequest(r)
	if err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.TargetTenantID = tenantID.String()
	req.DryRun = true
	if qOverwrite := r.URL.Query().Get("overwrite_policy"); qOverwrite != "" {
		req.OverwritePolicy = qOverwrite
	}
	if qPreserve := r.URL.Query().Get("preserve_status"); qPreserve != "" {
		req.PreserveStatus = qPreserve == "true"
	}
	if qPrune := r.URL.Query().Get("prune_missing"); qPrune != "" {
		req.Prune = qPrune == "true"
	}

	report, err := h.porter.Preflight(r.Context(), *req, bypass)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/import/preview failed: %v", err)
		http.Error(w, "preflight failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !report.Success {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(report)
}

// handleImportRules executes atomic rule bundle import.
func (h *ValidationRuleHandler) handleImportRules(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := mustTenantID(r)
	if !ok {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}

	bypass := isBypassGovernance(r)

	req, err := decodeImportRequest(r)
	if err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	req.TargetTenantID = tenantID.String()
	if qDryRun := r.URL.Query().Get("dry_run"); qDryRun != "" {
		req.DryRun = qDryRun == "true"
	}
	if qOverwrite := r.URL.Query().Get("overwrite_policy"); qOverwrite != "" {
		req.OverwritePolicy = qOverwrite
	}
	if qPreserve := r.URL.Query().Get("preserve_status"); qPreserve != "" {
		req.PreserveStatus = qPreserve == "true"
	}
	if qPrune := r.URL.Query().Get("prune_missing"); qPrune != "" {
		req.Prune = qPrune == "true"
	}

	idempotencyHeader := r.Header.Get("X-Idempotency-Key")
	report, err := h.porter.ImportRules(r.Context(), *req, bypass, idempotencyHeader)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("validation-rule-nodes/import failed: %v", err)
		http.Error(w, "import failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !report.Success {
		w.WriteHeader(http.StatusUnprocessableEntity)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(report)
}
