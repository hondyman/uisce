package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// Built-in profile keys seeded by
// db/migrations/20260930_001_seed_menu_abac_policies.up.sql.
const (
	profileBaseUser         = "BASE_USER"
	profilePlatformOperator = "PLATFORM_OPERATOR"
)

// operatorRoleNames are the signed JWT role claims that map to
// PLATFORM_OPERATOR. This mirrors the vocabulary AuthContextMiddleware
// already trusts in internal/middleware/auth_context.go — do not widen it
// without updating that middleware too, or the two will disagree.
var operatorRoleNames = map[string]struct{}{
	"global_admin":      {},
	"global_ops":        {},
	"platform_operator": {},
	"admin":             {},
	"core_admin":        {},
	"is_core_admin":     {},
}

// CapabilitiesHandler returns the effective ABAC capability map for the
// current user (tenant + profile) — the frontend uses this to decide which
// menu groups and items to render.
//
// The result merges the gold-copy baseline (NULL tenant_id policies) with
// tenant-specific overrides, evaluated against the *caller's own* profile.
// Deny always beats allow at equal priority. An empty map (not an error) is
// returned when no policies are configured yet.
type CapabilitiesHandler struct {
	db *sqlx.DB
}

func NewCapabilitiesHandler(db *sqlx.DB) *CapabilitiesHandler {
	return &CapabilitiesHandler{db: db}
}

func (h *CapabilitiesHandler) RegisterRoutes(r chi.Router) {
	r.Get("/capabilities", h.getCapabilities)
}

func (h *CapabilitiesHandler) getCapabilities(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetTenantIDFromContext(r)
	if tenantIDStr == "" {
		tenantIDStr = r.Header.Get("X-Tenant-ID")
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil || tenantID == uuid.Nil {
		http.Error(w, "tenant_id is required", http.StatusUnauthorized)
		return
	}

	// AuthInfo is populated by AuthContextMiddleware from the verified JWT.
	// Its roles and IsGlobalAdmin flag are server-derived, never client-supplied.
	auth, _ := security.AuthInfoFromContext(r.Context())
	profileKey := h.resolveProfileKey(r.Context(), auth, tenantID)

	gold := goldcopy.ResolveTenantID(r.Context(), h.db)

	type policyRow struct {
		ActionAttribute string `db:"action_attribute"`
		Effect          string `db:"effect"`
		Priority        int    `db:"priority"`
	}
	var rows []policyRow
	// The target_profile_key predicate is load-bearing: without it every
	// caller would receive every other profile's policies for the tenant, so
	// a BASE_USER would be told it holds PLATFORM_OPERATOR's menu grants.
	err = h.db.SelectContext(r.Context(), &rows, `
		SELECT action_attribute, effect, priority
		FROM studio.tenant_abac_policies
		WHERE enabled = true
		  AND target_profile_key = $1
		  AND (
		    tenant_id = $2
		    OR tenant_id = $3
		    OR tenant_id IS NULL
		  )
		ORDER BY priority DESC, effect DESC
	`, profileKey, tenantID, gold)
	if err != nil {
		// Table may not exist yet (migration pending) — return an empty
		// map so the frontend falls through to the platform-operator bypass.
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
		return
	}

	// Merge: deny wins; highest-priority first, so first-seen value wins.
	caps := make(map[string]bool, len(rows))
	for _, row := range rows {
		if _, seen := caps[row.ActionAttribute]; seen {
			// Deny is sticky — once set to false, do not overwrite
			if !caps[row.ActionAttribute] {
				continue
			}
		}
		caps[row.ActionAttribute] = row.Effect == "allow"
	}

	// The response header lets the frontend enforce Menu Designer
	// required_entitlement (a target_profile_key) without changing the body
	// shape, which stays a flat action_attribute -> bool map. It reflects the
	// same server-resolved profile used to scope the query above, so there is
	// exactly one source of truth for "who is this caller".
	w.Header().Set("X-Resolved-Profile", profileKey)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(caps)
}

// resolveProfileKey determines which target_profile_key the caller's
// capabilities are evaluated against.
//
// The authoritative source is the tenant's own IAM assignment
// (iam.user_roles -> iam.roles.role_name), which also lets a tenant mint
// profile keys beyond the two built-in ones. When no IAM row exists — the
// common case today, because those tables ship empty — the profile is
// derived from the verified JWT role claims.
//
// The profile is never read from a request header or query parameter: a
// client that could name its own profile would simply pick the one with the
// widest grants. Unknown callers fall back to BASE_USER, the most
// restrictive profile.
func (h *CapabilitiesHandler) resolveProfileKey(ctx context.Context, auth security.AuthInfo, tenantID uuid.UUID) string {
	if uid, err := uuid.Parse(strings.TrimSpace(auth.UserID)); err == nil {
		var key string
		err = h.db.GetContext(ctx, &key, `
			SELECT r.role_name
			FROM iam.user_roles ur
			JOIN iam.roles r ON r.role_id = ur.role_id
			WHERE ur.user_id = $1
			  AND r.tenant_id = $2
			  AND (ur.expires_at IS NULL OR ur.expires_at > now())
			ORDER BY r.is_global_admin DESC NULLS LAST, r.role_name
			LIMIT 1
		`, uid, tenantID)
		if err == nil {
			if k := strings.TrimSpace(key); k != "" {
				return k
			}
		}
		// ErrNoRows just means "no IAM assignment" — fall through to claims.
	}

	if auth.IsGlobalAdmin {
		return profilePlatformOperator
	}
	for _, role := range auth.Roles {
		if _, ok := operatorRoleNames[strings.ToLower(strings.TrimSpace(role))]; ok {
			return profilePlatformOperator
		}
	}
	return profileBaseUser
}
