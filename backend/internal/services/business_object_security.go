package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// AccessLevel represents the effective permission over a Business Object.
type AccessLevel string

const (
	AccessLevelNone  AccessLevel = "NONE"
	AccessLevelRead  AccessLevel = "READ"
	AccessLevelWrite AccessLevel = "WRITE"
)

// ErrForbidden is returned when a caller lacks the required permission.
var ErrForbidden = errors.New("forbidden")

// AccessDecision is the composed decision for a principal over a BO.
type AccessDecision struct {
	AccessLevel  AccessLevel
	RowPredicate string
	ColumnMasks  map[string]string
}

// Principal carries the resolved user identity and groups.
type Principal struct {
	UserID string
	Groups []string
}

// principalContextKey avoids collisions in context.
type principalContextKey struct{}

// WithPrincipal stores a Principal in context.
func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, p)
}

// PrincipalFromContext extracts a Principal if present.
func PrincipalFromContext(ctx context.Context) Principal {
	if ctx == nil {
		return Principal{}
	}
	if v := ctx.Value(principalContextKey{}); v != nil {
		if p, ok := v.(Principal); ok {
			return p
		}
	}
	return Principal{}
}

// AccessRule captures a single row from access_rule.
type AccessRule struct {
	RuleID           string          `db:"rule_id"`
	TenantID         string          `db:"tenant_id"`
	BusinessObjectID string          `db:"bo_id"`
	GroupDN          string          `db:"group_dn"`
	RowFilterDSL     sql.NullString  `db:"row_filter_dsl"`
	ColumnMasksRaw   json.RawMessage `db:"column_masks"`
	AccessLevel      AccessLevel     `db:"access_level"`
	Status           string          `db:"status"`
}

// columnMaskWire mirrors JSON structure stored in column_masks.
type columnMaskWire struct {
	Term     string `json:"term"`
	MaskType string `json:"mask_type"`
}

// AccessRuleRepository resolves decisions for BO access.
type AccessRuleRepository interface {
	ResolveDecision(ctx context.Context, tenantID, boID string, principal Principal) (*AccessDecision, error)
}

// pgAccessRuleRepository fetches rules from Postgres and caches decisions.
type pgAccessRuleRepository struct {
	db    *sqlx.DB
	ttl   time.Duration
	cache sync.Map // key -> *cachedDecision
}

type cachedDecision struct {
	decision *AccessDecision
	expires  time.Time
}

// atLeast returns true if current level meets the required level.
func (l AccessLevel) atLeast(required AccessLevel) bool {
	rank := map[AccessLevel]int{
		AccessLevelNone:  0,
		AccessLevelRead:  1,
		AccessLevelWrite: 2,
	}
	return rank[l] >= rank[required]
}

// resolveAccessDecision loads the decision for a BO using the attached repository.
func (s *BusinessObjectService) resolveAccessDecision(ctx context.Context, tenantID, boID string) (*AccessDecision, error) {
	// 1. Resolve Principal from context (Handler vs Service layer gap)
	principal := PrincipalFromContext(ctx)

	// If Principal is empty, try to derive from AuthInfo (Handler context)
	if principal.UserID == "" {
		if auth, ok := security.AuthInfoFromContext(ctx); ok {
			principal = Principal{
				UserID: auth.UserID,
				Groups: auth.Roles, // Map roles to groups for now
			}
		}
	}

	// 2. Global Admin / Global Ops Bypass (Root Access)
	for _, role := range principal.Groups {
		if role == "global_admin" || role == "global_ops" {
			// Full access, no row filters, no column masks
			return &AccessDecision{
				AccessLevel: AccessLevelWrite,
				ColumnMasks: map[string]string{},
			}, nil
		}
	}

	if s.rules == nil {
		logging.GetLogger().Sugar().Warn("[SECURITY] access rule repository not configured; allowing all access")
		return &AccessDecision{AccessLevel: AccessLevelWrite, ColumnMasks: map[string]string{}}, nil
	}

	return s.rules.ResolveDecision(ctx, tenantID, boID, principal)
}

// requireAccess enforces the required level and returns the decision for downstream use.
func (s *BusinessObjectService) requireAccess(ctx context.Context, tenantID, boID string, required AccessLevel) (*AccessDecision, error) {
	decision, err := s.resolveAccessDecision(ctx, tenantID, boID)
	if err != nil {
		return nil, err
	}

	if !decision.AccessLevel.atLeast(required) {
		return nil, ErrForbidden
	}

	if decision.ColumnMasks == nil {
		decision.ColumnMasks = make(map[string]string)
	}

	return decision, nil
}
