package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/auth"
	coremodels "github.com/hondyman/semlayer/backend/models"
)

// CollaborationServiceAPI defines the minimal methods used by the policy routes.
// This allows the API routing to accept fakes in unit tests.
type CollaborationServiceAPI interface {
	ListAccessPolicies(ctx context.Context) ([]coremodels.AccessControlPolicy, error)
	GetAccessPolicyByID(ctx context.Context, id uuid.UUID) (*coremodels.AccessControlPolicy, error)
	GetAccessPolicyBySlug(ctx context.Context, slug string) (*coremodels.AccessControlPolicy, error)
	DeleteAccessPolicy(ctx context.Context, id uuid.UUID) error
	CreateAccessPolicy(ctx context.Context, policy *coremodels.AccessControlPolicy) (*coremodels.AccessControlPolicy, error)
	UpdateAccessPolicy(ctx context.Context, policy *coremodels.AccessControlPolicy) (*coremodels.AccessControlPolicy, error)
}

// RegisterPolicyRoutes mounts policy-related endpoints on the provided router.
func RegisterPolicyRoutes(r chi.Router, srv *Server, collabService CollaborationServiceAPI) {
	r.Get("/policies", func(w http.ResponseWriter, r *http.Request) {
		policies, err := collabService.ListAccessPolicies(r.Context())
		respond(w, r, policies, err)
	})

	r.Get("/policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		if strings.TrimSpace(idStr) == "" {
			http.Error(w, "policy id required", http.StatusBadRequest)
			return
		}

		var (
			policy *coremodels.AccessControlPolicy
			err    error
		)

		if parsedID, parseErr := uuid.Parse(idStr); parseErr == nil {
			policy, err = collabService.GetAccessPolicyByID(r.Context(), parsedID)
		} else {
			policy, err = collabService.GetAccessPolicyBySlug(r.Context(), idStr)
		}
		if err != nil {
			respond(w, r, nil, err)
			return
		}

		if srv.AuditSvc != nil {
			actorID := ""
			if u, ok := auth.GetUserFromContext(r.Context()); ok {
				actorID = u.ID
			} else if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_POLICIES", "true")) == "true" {
				actorID = auth.FallbackUser().ID
			}
			_ = srv.AuditSvc.LogDataAccess(r.Context(), actorID, "", "", "access_policy", policy.ID.String(), "read", map[string]any{
				"policy_id": policy.PolicyID,
				"scope":     policy.Scope,
				"role":      policy.Role,
			})
		}

		respond(w, r, policy, nil)
	})

	r.Delete("/policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		if strings.TrimSpace(idStr) == "" {
			http.Error(w, "policy id required", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		var (
			targetPolicy *coremodels.AccessControlPolicy
			err          error
		)

		if parsedID, parseErr := uuid.Parse(idStr); parseErr == nil {
			targetPolicy, err = collabService.GetAccessPolicyByID(ctx, parsedID)
		} else {
			targetPolicy, err = collabService.GetAccessPolicyBySlug(ctx, idStr)
		}
		if err != nil {
			respond(w, r, nil, err)
			return
		}

		user, ok := auth.GetUserFromContext(ctx)
		if !ok {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_POLICIES", "false")) == "true" {
				user = auth.FallbackUser()
			} else {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
		}

		if err := collabService.DeleteAccessPolicy(ctx, targetPolicy.ID); err != nil {
			respond(w, r, nil, err)
			return
		}

		if srv.AuditSvc != nil {
			_ = srv.AuditSvc.LogDataModification(ctx, user.ID, "", "", "access_policy", targetPolicy.ID.String(), "delete", map[string]any{
				"policy_id": targetPolicy.PolicyID,
			}, nil)
		}

		respond(w, r, map[string]any{"status": "deleted", "deleted_by": user.ID}, nil)
	})

	r.Post("/policies", func(w http.ResponseWriter, r *http.Request) {
		var policy coremodels.AccessControlPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			http.Error(w, "invalid policy payload: failed to decode JSON", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(policy.PolicyID) == "" {
			http.Error(w, "invalid policy payload: 'policy_id' is required", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(policy.Scope) == "" {
			http.Error(w, "invalid policy payload: 'scope' is required and must identify a data domain (e.g. 'domain:finance')", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(policy.Role) == "" {
			http.Error(w, "invalid policy payload: 'role' is required", http.StatusBadRequest)
			return
		}
		if policy.ApprovalThreshold < 0 {
			http.Error(w, "invalid policy payload: 'approval_threshold' must be >= 0", http.StatusBadRequest)
			return
		}
		if policy.DurationDays < 0 {
			http.Error(w, "invalid policy payload: 'duration_days' must be >= 0", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		user, ok := auth.GetUserFromContext(ctx)
		if !ok {
			if strings.ToLower(getEnv("DEV_ALLOW_UNAUTH_POLICIES", "false")) == "true" {
				user = auth.FallbackUser()
			} else {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
		}

		var (
			result   *coremodels.AccessControlPolicy
			action   string
			auditOld map[string]any
		)

		if policy.ID != uuid.Nil {
			existing, err := collabService.GetAccessPolicyByID(ctx, policy.ID)
			if err != nil {
				respond(w, r, nil, err)
				return
			}
			if existing != nil {
				b, _ := json.Marshal(existing)
				_ = json.Unmarshal(b, &auditOld)
			}

			result, err = collabService.UpdateAccessPolicy(ctx, &policy)
			if err != nil {
				respond(w, r, nil, err)
				return
			}
			action = "update"
		} else {
			created, err := collabService.CreateAccessPolicy(ctx, &policy)
			if err != nil {
				respond(w, r, nil, err)
				return
			}
			result = created
			action = "create"
		}

		if result == nil {
			respond(w, r, nil, fmt.Errorf("policy operation failed"))
			return
		}

		if srv.AuditSvc != nil {
			var auditNew map[string]any
			b, _ := json.Marshal(result)
			_ = json.Unmarshal(b, &auditNew)
			_ = srv.AuditSvc.LogDataModification(ctx, user.ID, "", "", "access_policy", result.ID.String(), action, auditOld, auditNew)
		}

		resp := map[string]any{
			"policy":       result,
			"status":       action,
			"performed_by": user.ID,
		}
		respond(w, r, resp, nil)
	})

	r.Post("/policies/simulate", func(w http.ResponseWriter, r *http.Request) {
		var policy coremodels.AccessControlPolicy
		if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
			http.Error(w, "invalid policy payload: failed to decode JSON", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(policy.Role) == "" {
			http.Error(w, "invalid policy payload: 'role' is required for simulation", http.StatusBadRequest)
			return
		}
		if len(policy.Permissions) == 0 {
			http.Error(w, "invalid policy payload: 'permissions' must be a non-empty array", http.StatusBadRequest)
			return
		}

		// For the simple simulation we just echo back the input with a mocked result
		resp := map[string]any{
			"simulated": true,
			"policy":    policy,
			"result": map[string]any{
				"estimated_approvals": 1,
			},
		}
		respond(w, r, resp, nil)
	})
}
