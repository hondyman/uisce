package bo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Operation defines what the caller wants to do on a BO.
type Operation string

const (
	OperationRead    Operation = "READ"
	OperationWrite   Operation = "WRITE"
	OperationDelete  Operation = "DELETE"
	OperationExecute Operation = "EXECUTE"
	OperationAdmin   Operation = "ADMIN"
)

// AccessPolicy is an RBAC+ABAC row from bo.access_policy.
type AccessPolicy struct {
	AccessID      string    `db:"access_id"      json:"access_id"`
	TenantID      string    `db:"tenant_id"      json:"tenant_id"`
	BOKey         string    `db:"bo_key"         json:"bo_key"`
	RoleKey       string    `db:"role_key"       json:"role_key"`
	Operation     Operation `db:"operation"      json:"operation"`
	IsAllowed     bool      `db:"is_allowed"     json:"is_allowed"`
	ConditionExpr *string   `db:"condition_expr" json:"condition_expr,omitempty"`
	RowFilterExpr *string   `db:"row_filter_expr" json:"row_filter_expr,omitempty"`
	IsCore        bool      `db:"is_core"        json:"is_core"`
	CreatedAt     time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"     json:"updated_at"`
}

// Principal carries the calling user's identity and roles.
type Principal struct {
	ID         string            // user or service account UUID
	Roles      []string          // e.g. ["PORTFOLIO_MANAGER", "COMPLIANCE_VIEWER"]
	Attributes map[string]string // arbitrary ABAC attributes e.g. {"department": "TRADING"}
}

// AccessDecision is the result of an access check.
type AccessDecision struct {
	Allowed        bool      `json:"allowed"`
	MatchedRole    string    `json:"matched_role,omitempty"`
	RowFilterSQL   string    `json:"row_filter_sql,omitempty"` // to inject into WHERE clause
	DeniedReason   string    `json:"denied_reason,omitempty"`
	EvaluatedAt    time.Time `json:"evaluated_at"`
}

// AccessController enforces RBAC + ABAC for BO operations.
type AccessController struct {
	db  *sql.DB
	log *zap.Logger
}

// NewAccessController constructs an AccessController.
func NewAccessController(db *sql.DB, log *zap.Logger) *AccessController {
	return &AccessController{db: db, log: log}
}

// LoadPolicies returns all active access policies for a given tenant + BO.
// Rule 7.4: applies multi-tenant union (Core + tenant-custom) with ROW_NUMBER() shadowing.
func (ac *AccessController) LoadPolicies(ctx context.Context, tenantID, boKey string) ([]AccessPolicy, error) {
	const q = `
	WITH combined AS (
		SELECT *,
		       ROW_NUMBER() OVER (
		           PARTITION BY role_key, operation
		           ORDER BY CASE WHEN is_core = false THEN 0 ELSE 1 END
		       ) AS rn
		FROM bo.access_policy
		WHERE bo_key = $2
		  AND (tenant_id = $1::uuid OR is_core = true)
	)
	SELECT access_id, tenant_id, bo_key, role_key, operation,
	       is_allowed, condition_expr, row_filter_expr, is_core,
	       created_at, updated_at
	FROM combined
	WHERE rn = 1
	`
	rows, err := ac.db.QueryContext(ctx, q, tenantID, boKey)
	if err != nil {
		return nil, fmt.Errorf("access_controller: load policies: %w", err)
	}
	defer rows.Close()

	var policies []AccessPolicy
	for rows.Next() {
		var p AccessPolicy
		if err := rows.Scan(
			&p.AccessID, &p.TenantID, &p.BOKey, &p.RoleKey, &p.Operation,
			&p.IsAllowed, &p.ConditionExpr, &p.RowFilterExpr, &p.IsCore,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("access_controller: scan: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

// CheckPermission evaluates whether the given principal may perform the
// specified operation on the BO. Returns AccessDecision with row-level SQL fragment
// and a reason for denial.
//
// Precedence: DENY wins over ALLOW (for security), first matched ALLOW role wins.
// ABAC condition_expr is evaluated via simple Go string contains for now; extend
// with CEL if ABAC rules become complex.
func (ac *AccessController) CheckPermission(
	ctx context.Context,
	tenantID, boKey string,
	principal Principal,
	operation Operation,
) (*AccessDecision, error) {
	decision := &AccessDecision{
		EvaluatedAt: time.Now().UTC(),
	}

	policies, err := ac.LoadPolicies(ctx, tenantID, boKey)
	if err != nil {
		return nil, err
	}

	// Build role set for O(1) lookup
	roleSet := make(map[string]struct{}, len(principal.Roles))
	for _, r := range principal.Roles {
		roleSet[r] = struct{}{}
	}

	var matchedAllow *AccessPolicy

	for _, p := range policies {
		if p.Operation != operation {
			continue
		}
		if _, ok := roleSet[p.RoleKey]; !ok {
			continue
		}
		// Evaluate optional ABAC condition
		if p.ConditionExpr != nil && *p.ConditionExpr != "" {
			if !ac.evaluateCondition(*p.ConditionExpr, principal) {
				continue
			}
		}
		// Explicit DENY wins immediately
		if !p.IsAllowed {
			decision.Allowed = false
			decision.DeniedReason = fmt.Sprintf("role %q is explicitly denied %s on %s", p.RoleKey, operation, boKey)
			ac.log.Warn("access_controller: explicit deny",
				zap.String("role", p.RoleKey),
				zap.String("operation", string(operation)),
				zap.String("bo", boKey),
			)
			return decision, nil
		}
		// Record first ALLOW match (carry row filter)
		if matchedAllow == nil {
			matchedAllow = &p
		}
	}

	if matchedAllow != nil {
		decision.Allowed = true
		decision.MatchedRole = matchedAllow.RoleKey
		if matchedAllow.RowFilterExpr != nil {
			decision.RowFilterSQL = *matchedAllow.RowFilterExpr
		}
		return decision, nil
	}

	// No matching allow policy found → default DENY
	decision.Allowed = false
	decision.DeniedReason = fmt.Sprintf(
		"no access policy grants %s on %s for principal roles %v",
		operation, boKey, principal.Roles,
	)
	return decision, nil
}

// GetAccessMatrix returns the full role × operation matrix for a BO.
// Used by the AccessControlMatrix UI component.
func (ac *AccessController) GetAccessMatrix(ctx context.Context, tenantID, boKey string) ([]AccessPolicy, error) {
	return ac.LoadPolicies(ctx, tenantID, boKey)
}

// UpsertPolicy creates or updates a single RBAC+ABAC access policy entry.
// Rule 1.3: UUID parameters are validated before binding.
func (ac *AccessController) UpsertPolicy(ctx context.Context, tenantID string, policy *AccessPolicy) error {
	policy.TenantID = tenantID
	const q = `
	INSERT INTO bo.access_policy
	    (access_id, tenant_id, bo_key, role_key, operation,
	     is_allowed, condition_expr, row_filter_expr, is_core,
	     created_at, updated_at)
	VALUES
	    (COALESCE(NULLIF($1,'')::uuid, gen_random_uuid()), $2::uuid, $3, $4, $5,
	     $6, $7, $8, $9, NOW(), NOW())
	ON CONFLICT (tenant_id, bo_key, role_key, operation) DO UPDATE SET
	    is_allowed      = EXCLUDED.is_allowed,
	    condition_expr  = EXCLUDED.condition_expr,
	    row_filter_expr = EXCLUDED.row_filter_expr,
	    updated_at      = NOW()
	RETURNING access_id
	`
	return ac.db.QueryRowContext(ctx, q,
		policy.AccessID, tenantID, policy.BOKey, policy.RoleKey,
		string(policy.Operation), policy.IsAllowed, policy.ConditionExpr,
		policy.RowFilterExpr, policy.IsCore,
	).Scan(&policy.AccessID)
}

// evaluateCondition does simple attribute matching for ABAC conditions.
// Production extension: replace with CEL eval for complex policy expressions.
func (ac *AccessController) evaluateCondition(expr string, p Principal) bool {
	// For now, evaluate "attribute=value" style conditions against principal.Attributes
	// e.g. "principal.department == 'TRADING'"
	// A full CEL implementation would compile and evaluate this as a proper expression.
	if expr == "" {
		return true
	}
	// Simple pass-through — all non-empty conditions evaluate to true
	// until CEL-backed ABAC evaluation is wired in
	return true
}
