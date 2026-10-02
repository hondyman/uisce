package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"

	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type mockRulePorter struct {
	exportFunc    func(ctx context.Context, tenantID, boName, domain, originFilter string) (*models.RuleBundle, error)
	preflightFunc func(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool) (*models.RuleImportReport, error)
	importFunc    func(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool, idempotencyKeyHeader string) (*models.RuleImportReport, error)
}

func (m *mockRulePorter) ExportRules(ctx context.Context, tenantID, boName, domain, originFilter string) (*models.RuleBundle, error) {
	if m.exportFunc != nil {
		return m.exportFunc(ctx, tenantID, boName, domain, originFilter)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRulePorter) Preflight(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool) (*models.RuleImportReport, error) {
	if m.preflightFunc != nil {
		return m.preflightFunc(ctx, req, bypassGovernance)
	}
	return nil, errors.New("not implemented")
}

func (m *mockRulePorter) ImportRules(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool, idempotencyKeyHeader string) (*models.RuleImportReport, error) {
	if m.importFunc != nil {
		return m.importFunc(ctx, req, bypassGovernance, idempotencyKeyHeader)
	}
	return nil, errors.New("not implemented")
}

func newTestPorterRouter(porter RulePorter) http.Handler {
	r := chi.NewRouter()
	h := NewValidationRuleHandler(nil, nil, porter)
	h.RegisterRoutes(r)
	return r
}

func TestExportRulesEndpoint(t *testing.T) {
	mockBundle := &models.RuleBundle{
		BundleVersion: models.RuleBundleVersion,
		Checksum:      "sha256:abcd",
		Rules: []models.PortableRuleSpec{
			{
				RuleKey: "party.min_age",
				BOName:  "party",
				Name:    "Min Age",
			},
		},
	}

	porter := &mockRulePorter{
		exportFunc: func(ctx context.Context, tenantID, boName, domain, originFilter string) (*models.RuleBundle, error) {
			if tenantID != vrTenant {
				return nil, errors.New("unexpected tenant")
			}
			return mockBundle, nil
		},
	}

	r := newTestPorterRouter(porter)

	t.Run("Unauthorized if no tenant", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/validation-rule-nodes/export", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", w.Code)
		}
	})

	t.Run("JSON export default", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/validation-rule-nodes/export?bo_name=party", nil)
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{vrTenant}}))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
			t.Errorf("expected json content-type, got %s", ct)
		}

		var resp models.RuleBundle
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode json: %v", err)
		}
		if len(resp.Rules) != 1 || resp.Rules[0].RuleKey != "party.min_age" {
			t.Errorf("unexpected bundle: %+v", resp)
		}
	})

	t.Run("YAML export format=yaml", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/validation-rule-nodes/export?format=yaml", nil)
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{vrTenant}}))
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "yaml") {
			t.Errorf("expected yaml content-type, got %s", ct)
		}

		var resp models.RuleBundle
		if err := yaml.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode yaml: %v", err)
		}
		if len(resp.Rules) != 1 || resp.Rules[0].RuleKey != "party.min_age" {
			t.Errorf("unexpected bundle: %+v", resp)
		}
	})
}

func TestPreviewImportEndpoint(t *testing.T) {
	porter := &mockRulePorter{
		preflightFunc: func(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool) (*models.RuleImportReport, error) {
			if len(req.Bundle.Rules) == 0 {
				return &models.RuleImportReport{
					Success: false,
					Errors: []models.RuleImportErrorDetail{
						{Code: models.ImportErrInvalidField, Reason: "empty rules"},
					},
				}, nil
			}
			return &models.RuleImportReport{
				Success:    true,
				TotalRules: 1,
				DiffSummary: []models.RuleDiffSummary{
					{RuleKey: "party.min_age", Action: "CREATE"},
				},
			}, nil
		},
	}

	r := newTestPorterRouter(porter)

	t.Run("Valid JSON payload preview", func(t *testing.T) {
		body := `{"rules":[{"rule_key":"party.min_age","bo_name":"party","name":"Min Age"}]}`
		req := httptest.NewRequest("POST", "/validation-rule-nodes/import/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{vrTenant}}))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var report models.RuleImportReport
		json.Unmarshal(w.Body.Bytes(), &report)
		if !report.Success || len(report.DiffSummary) != 1 {
			t.Errorf("unexpected report: %+v", report)
		}
	})

	t.Run("Validation failure returns 422 with report", func(t *testing.T) {
		body := `{"rules":[]}`
		req := httptest.NewRequest("POST", "/validation-rule-nodes/import/preview", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{vrTenant}}))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Fatalf("expected 422, got %d: %s", w.Code, w.Body.String())
		}
		var report models.RuleImportReport
		json.Unmarshal(w.Body.Bytes(), &report)
		if report.Success || len(report.Errors) != 1 {
			t.Errorf("unexpected report: %+v", report)
		}
	})

	t.Run("Preflight endpoint alias returns identical preview", func(t *testing.T) {
		body := `{"rules":[{"rule_key":"party.min_age","bo_name":"party","name":"Min Age"}]}`
		req := httptest.NewRequest("POST", "/validation-rule-nodes/preflight", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(security.WithAuthInfo(req.Context(), security.AuthInfo{TenantIDs: []string{vrTenant}}))

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		var report models.RuleImportReport
		json.Unmarshal(w.Body.Bytes(), &report)
		if !report.Success || len(report.DiffSummary) != 1 {
			t.Errorf("unexpected report: %+v", report)
		}
	})
}

func TestImportRulesEndpoint(t *testing.T) {
	var capturedReq models.RuleImportRequest
	var capturedBypass bool
	var capturedIdemHeader string

	porter := &mockRulePorter{
		importFunc: func(ctx context.Context, req models.RuleImportRequest, bypassGovernance bool, idempotencyKeyHeader string) (*models.RuleImportReport, error) {
			capturedReq = req
			capturedBypass = bypassGovernance
			capturedIdemHeader = idempotencyKeyHeader
			return &models.RuleImportReport{
				Success:    true,
				TotalRules: 1,
				Created:    []string{"party.min_age"},
				DryRun:     req.DryRun,
			}, nil
		},
	}

	r := newTestPorterRouter(porter)

	t.Run("Import with query params override and JWT bypass", func(t *testing.T) {
		body := `{"rules":[{"rule_key":"party.min_age"}]}`
		req := httptest.NewRequest("POST", "/validation-rule-nodes/import?dry_run=true&overwrite_policy=overwrite&prune_missing=true", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Idempotency-Key", "custom-key-123")

		// Add claims with RULE_IMPORT_BYPASS_GOVERNANCE
		claims := &jwtmiddleware.JWTClaims{
			TenantID: vrTenant,
			Roles:    []string{"RULE_IMPORT_BYPASS_GOVERNANCE"},
		}
		ctx := context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims)
		ctx = security.WithAuthInfo(ctx, security.AuthInfo{TenantIDs: []string{vrTenant}})
		req = req.WithContext(ctx)

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
		if !capturedBypass {
			t.Errorf("expected bypassGovernance to be true for role")
		}
		if !capturedReq.DryRun {
			t.Errorf("expected dryRun to be true from query param")
		}
		if capturedReq.OverwritePolicy != models.ImportOverwriteOverwrite {
			t.Errorf("expected overwrite policy to be 'overwrite', got %s", capturedReq.OverwritePolicy)
		}
		if !capturedReq.Prune {
			t.Errorf("expected prune to be true")
		}
		if capturedReq.TargetTenantID != vrTenant {
			t.Errorf("expected targetTenantID %s, got %s", vrTenant, capturedReq.TargetTenantID)
		}
		if capturedIdemHeader != "custom-key-123" {
			t.Errorf("expected idempotency header 'custom-key-123', got %s", capturedIdemHeader)
		}
	})
}
