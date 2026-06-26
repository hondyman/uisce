package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/jmoiron/sqlx"
)

var ErrRoleNotFound = errors.New("role not found")
var errRoleRepositoryUnavailable = errors.New("role repository not configured")

// RoleService defines the interface for managing roles and their bundle assignments with full persistence.
type RoleService interface {
	CreateRole(ctx context.Context, user models.User, input RoleCreateInput) (*models.Role, error)
	GetRole(ctx context.Context, user models.User, name string) (*models.Role, error)
	ListRoles(ctx context.Context, user models.User) ([]*models.Role, error)
	UpdateRole(ctx context.Context, user models.User, name string, input RoleUpdateInput) (*models.Role, error)
	DeleteRole(ctx context.Context, user models.User, name string) error
	AssignBundleToRole(ctx context.Context, user models.User, roleName, bundleID string) error
	UnassignBundleFromRole(ctx context.Context, user models.User, roleName, bundleID string) error
	GetBundleIDsForRole(ctx context.Context, user models.User, roleName string) ([]string, error)
}

// RoleCreateInput captures the attributes required to create a new role.
type RoleCreateInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        models.RoleType   `json:"type,omitempty"`
	Scope       models.RoleScope  `json:"scope,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// RoleUpdateInput captures mutable role attributes for lifecycle and metadata updates.
type RoleUpdateInput struct {
	Description *string            `json:"description,omitempty"`
	Status      *models.RoleStatus `json:"status,omitempty"`
	Type        *models.RoleType   `json:"type,omitempty"`
	Scope       *models.RoleScope  `json:"scope,omitempty"`
	Tags        *[]string          `json:"tags,omitempty"`
	Attributes  *map[string]string `json:"attributes,omitempty"`
	Owner       *string            `json:"owner,omitempty"`
	Notes       string             `json:"notes,omitempty"`
}

type roleServiceImpl struct {
	repo          roleRepository
	policyService platform.PolicyService
	bundleAccess  BundleRoleManager
	policies      map[string][]models.Policy
}

type roleRepository interface {
	CreateRole(ctx context.Context, role *models.Role) error
	SaveRole(ctx context.Context, role *models.Role) error
	GetRoleByName(ctx context.Context, tenantID *uuid.UUID, normalized string) (*models.Role, error)
	ListRoles(ctx context.Context, tenantID *uuid.UUID) ([]*models.Role, error)
	RoleExists(ctx context.Context, tenantID *uuid.UUID, normalized string) (bool, error)
	DeleteRole(ctx context.Context, tenantID *uuid.UUID, normalized string) error
}

type sqlRoleRepository struct {
	db *sqlx.DB
}

type noopRoleRepository struct{}

func (noopRoleRepository) CreateRole(ctx context.Context, role *models.Role) error {
	return errRoleRepositoryUnavailable
}

func (noopRoleRepository) SaveRole(ctx context.Context, role *models.Role) error {
	return errRoleRepositoryUnavailable
}

func (noopRoleRepository) GetRoleByName(ctx context.Context, tenantID *uuid.UUID, normalized string) (*models.Role, error) {
	return nil, errRoleRepositoryUnavailable
}

func (noopRoleRepository) ListRoles(ctx context.Context, tenantID *uuid.UUID) ([]*models.Role, error) {
	return nil, errRoleRepositoryUnavailable
}

func (noopRoleRepository) RoleExists(ctx context.Context, tenantID *uuid.UUID, normalized string) (bool, error) {
	return false, errRoleRepositoryUnavailable
}

func (noopRoleRepository) DeleteRole(ctx context.Context, tenantID *uuid.UUID, normalized string) error {
	return errRoleRepositoryUnavailable
}

type roleRow struct {
	ID                   uuid.UUID       `db:"id"`
	TenantID             *uuid.UUID      `db:"tenant_id"`
	Name                 string          `db:"name"`
	NormalizedName       string          `db:"normalized_name"`
	DisplayName          string          `db:"display_name"`
	Description          sql.NullString  `db:"description"`
	Version              string          `db:"version"`
	Status               string          `db:"status"`
	RoleType             string          `db:"role_type"`
	Owner                string          `db:"owner"`
	Scope                string          `db:"scope"`
	Tags                 json.RawMessage `db:"tags"`
	Attributes           json.RawMessage `db:"attributes"`
	Policies             json.RawMessage `db:"policies"`
	Permissions          json.RawMessage `db:"permissions"`
	AttributeConstraints json.RawMessage `db:"attribute_constraints"`
	Members              json.RawMessage `db:"members"`
	BundleIDs            json.RawMessage `db:"bundle_ids"`
	AuditTrail           json.RawMessage `db:"audit_trail"`
	AuditMetadata        json.RawMessage `db:"audit_metadata"`
	Lifecycle            json.RawMessage `db:"lifecycle"`
	CreatedAt            time.Time       `db:"created_at"`
	UpdatedAt            time.Time       `db:"updated_at"`
}

const roleSelectBase = `SELECT id, tenant_id, name, normalized_name, display_name, description, version, status, role_type, owner, scope, tags, attributes, policies, permissions, attribute_constraints, members, bundle_ids, audit_trail, audit_metadata, lifecycle, created_at, updated_at FROM semantic_roles`

// NewRoleService creates a new instance of the role service backed by persistent storage.
func NewRoleService(db *sqlx.DB, policyService platform.PolicyService, bundleAccess BundleRoleManager) RoleService {
	var repo roleRepository
	if db != nil && db.DB != nil {
		repo = &sqlRoleRepository{db: db}
	} else {
		repo = noopRoleRepository{}
	}
	svc := &roleServiceImpl{
		repo:          repo,
		policyService: policyService,
		bundleAccess:  bundleAccess,
		policies: map[string][]models.Policy{
			"role": {
				{
					ID:          "role-create-steward",
					Effect:      "allow",
					Actions:     []string{"create", "list", "delete"},
					Resources:   []string{"role"},
					Description: "Stewards can create and list roles",
					Conditions: []models.AttributeCondition{
						{Attribute: "roles", Operator: "contains", Values: []string{"Steward"}},
					},
				},
				{
					ID:          "role-create-admin",
					Effect:      "allow",
					Actions:     []string{"create", "list", "delete"},
					Resources:   []string{"role"},
					Description: "Admins can create and list roles",
					Conditions: []models.AttributeCondition{
						{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
					},
				},
			},
			"role:*": {
				{
					ID:          "role-read-steward",
					Effect:      "allow",
					Actions:     []string{"read", "update", "delete"},
					Resources:   []string{"role:*"},
					Description: "Stewards can read and manage roles",
					Conditions: []models.AttributeCondition{
						{Attribute: "roles", Operator: "contains", Values: []string{"Steward"}},
					},
				},
				{
					ID:          "role-read-admin",
					Effect:      "allow",
					Actions:     []string{"read", "update", "delete"},
					Resources:   []string{"role:*"},
					Description: "Admins can read and manage roles",
					Conditions: []models.AttributeCondition{
						{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
					},
				},
			},
		},
	}
	if _, ok := repo.(*sqlRoleRepository); ok {
		svc.seedDefaults()
	}
	return svc
}

func (s *roleServiceImpl) CreateRole(ctx context.Context, user models.User, input RoleCreateInput) (*models.Role, error) {
	sanitizedName := strings.TrimSpace(input.Name)
	if sanitizedName == "" {
		return nil, fmt.Errorf("role name is required")
	}
	normalized := normalizeRoleKey(sanitizedName)

	if err := s.authorize(user, "create", "role"); err != nil {
		return nil, err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return nil, err
	}

	exists, err := s.repo.RoleExists(ctx, tenantUUID, normalized)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("role %s already exists", sanitizedName)
	}

	now := time.Now()
	roleType := input.Type
	if roleType == "" {
		roleType = models.RoleTypeBusiness
	}
	scope := input.Scope
	if scope == "" {
		scope = models.RoleScopeGlobal
	}

	description := strings.TrimSpace(input.Description)
	displayName := formatRoleDisplayName(sanitizedName)

	role := &models.Role{
		ID:          uuid.New().String(),
		Name:        sanitizedName,
		DisplayName: displayName,
		Description: description,
		Version:     "1.0.0",
		Status:      models.RoleStatusDraft,
		Type:        roleType,
		Owner:       user.ID,
		Scope:       scope,
		TenantID:    strings.TrimSpace(user.TenantID),
		Tags:        normalizeTags(input.Tags),
		Attributes:  sanitizeAttributes(input.Attributes),
		Permissions: []models.RolePermission{},
		Members:     []models.RoleMember{},
		BundleIDs:   []string{},
		AuditTrail:  []models.RoleChangeRecord{},
		AuditMetadata: &models.RoleAuditMetadata{
			CreatedBy: user.ID,
			CreatedAt: now,
		},
		Lifecycle: models.RoleLifecycle{
			CreatedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	recordRoleAudit(role, "create", "Role created", user.ID, now)

	if err := s.repo.CreateRole(ctx, role); err != nil {
		return nil, err
	}

	return cloneRole(role), nil
}

func (s *roleServiceImpl) GetRole(ctx context.Context, user models.User, name string) (*models.Role, error) {
	key := normalizeRoleKey(name)
	resource := fmt.Sprintf("role:%s", key)
	if err := s.authorize(user, "read", resource); err != nil {
		return nil, err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, key)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	return cloneRole(role), nil
}

func (s *roleServiceImpl) ListRoles(ctx context.Context, user models.User) ([]*models.Role, error) {
	if err := s.authorize(user, "list", "role"); err != nil {
		return nil, err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return nil, err
	}

	roles, err := s.repo.ListRoles(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}

	clones := make([]*models.Role, 0, len(roles))
	for _, role := range roles {
		clones = append(clones, cloneRole(role))
	}

	sort.Slice(clones, func(i, j int) bool {
		left := strings.ToLower(clones[i].DisplayName)
		right := strings.ToLower(clones[j].DisplayName)
		if left == right {
			return strings.ToLower(clones[i].Name) < strings.ToLower(clones[j].Name)
		}
		return left < right
	})

	return clones, nil
}

func (s *roleServiceImpl) UpdateRole(ctx context.Context, user models.User, name string, input RoleUpdateInput) (*models.Role, error) {
	key := normalizeRoleKey(name)
	resource := fmt.Sprintf("role:%s", key)
	if err := s.authorize(user, "update", resource); err != nil {
		return nil, err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, key)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	changed := false
	now := time.Now()

	if input.Description != nil {
		newDesc := strings.TrimSpace(*input.Description)
		if role.Description != newDesc {
			role.Description = newDesc
			changed = true
		}
	}

	if input.Type != nil && *input.Type != "" && role.Type != *input.Type {
		role.Type = *input.Type
		changed = true
	}

	if input.Scope != nil && *input.Scope != "" && role.Scope != *input.Scope {
		role.Scope = *input.Scope
		changed = true
	}

	if input.Owner != nil {
		newOwner := strings.TrimSpace(*input.Owner)
		if newOwner != "" && role.Owner != newOwner {
			role.Owner = newOwner
			changed = true
		}
	}

	if input.Tags != nil {
		normalizedTags := normalizeTags(*input.Tags)
		if !stringSlicesEqual(role.Tags, normalizedTags) {
			role.Tags = normalizedTags
			changed = true
		}
	}

	if input.Attributes != nil {
		sanitized := sanitizeAttributes(*input.Attributes)
		if !attributesEqual(role.Attributes, sanitized) {
			role.Attributes = sanitized
			changed = true
		}
	}

	if input.Status != nil && *input.Status != "" && role.Status != *input.Status {
		changed = true
		transitionRoleStatus(role, *input.Status, user.ID, input.Notes, now)
	}

	if !changed {
		return cloneRole(role), nil
	}

	if input.Status == nil || role.Status == *input.Status {
		note := input.Notes
		if strings.TrimSpace(note) == "" {
			note = "Role metadata updated"
		}
		recordRoleAudit(role, "update_metadata", note, user.ID, now)
	}

	if err := s.repo.SaveRole(ctx, role); err != nil {
		return nil, err
	}

	return cloneRole(role), nil
}

func (s *roleServiceImpl) DeleteRole(ctx context.Context, user models.User, name string) error {
	key := normalizeRoleKey(name)
	if key == "" {
		return fmt.Errorf("role name is required")
	}

	resource := fmt.Sprintf("role:%s", key)
	if err := s.authorize(user, "delete", resource); err != nil {
		return err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, key)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	if s.bundleAccess != nil {
		for _, bundleID := range role.BundleIDs {
			if err := s.bundleAccess.UnassignRoleFromBundle(role.Name, bundleID); err != nil {
				return err
			}
		}
	}

	if err := s.repo.DeleteRole(ctx, tenantUUID, key); err != nil {
		return err
	}

	return nil
}

func (s *roleServiceImpl) AssignBundleToRole(ctx context.Context, user models.User, roleName, bundleID string) error {
	roleKey := normalizeRoleKey(roleName)
	if roleKey == "" {
		return fmt.Errorf("role name is required")
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return fmt.Errorf("bundle id is required")
	}

	resource := fmt.Sprintf("role:%s", roleKey)
	if err := s.authorize(user, "update", resource); err != nil {
		return err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, roleKey)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}
	if role.Status == models.RoleStatusRetired {
		return fmt.Errorf("role %s is retired and cannot receive new bundles", roleName)
	}

	updatedBundles, changed := addBundleID(role.BundleIDs, bundleID)
	if !changed {
		return nil
	}

	if s.bundleAccess != nil {
		if err := s.bundleAccess.AssignRoleToBundle(role.Name, bundleID); err != nil {
			return err
		}
	}

	role.BundleIDs = updatedBundles
	now := time.Now()
	recordRoleAudit(role, "assign_bundle", fmt.Sprintf("Assigned bundle %s", bundleID), user.ID, now)

	if err := s.repo.SaveRole(ctx, role); err != nil {
		return err
	}

	return nil
}

func (s *roleServiceImpl) UnassignBundleFromRole(ctx context.Context, user models.User, roleName, bundleID string) error {
	roleKey := normalizeRoleKey(roleName)
	if roleKey == "" {
		return fmt.Errorf("role name is required")
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return fmt.Errorf("bundle id is required")
	}

	resource := fmt.Sprintf("role:%s", roleKey)
	if err := s.authorize(user, "update", resource); err != nil {
		return err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, roleKey)
	if err != nil {
		return err
	}
	if role == nil {
		return ErrRoleNotFound
	}

	updatedBundles, changed := removeBundleID(role.BundleIDs, bundleID)
	if !changed {
		return fmt.Errorf("bundle %s was not assigned to role %s", bundleID, roleName)
	}

	if s.bundleAccess != nil {
		if err := s.bundleAccess.UnassignRoleFromBundle(role.Name, bundleID); err != nil {
			return err
		}
	}

	role.BundleIDs = updatedBundles
	now := time.Now()
	recordRoleAudit(role, "unassign_bundle", fmt.Sprintf("Unassigned bundle %s", bundleID), user.ID, now)

	if err := s.repo.SaveRole(ctx, role); err != nil {
		return err
	}

	return nil
}

func (s *roleServiceImpl) GetBundleIDsForRole(ctx context.Context, user models.User, roleName string) ([]string, error) {
	roleKey := normalizeRoleKey(roleName)
	if roleKey == "" {
		return nil, fmt.Errorf("role name is required")
	}

	resource := fmt.Sprintf("role:%s", roleKey)
	if err := s.authorize(user, "read", resource); err != nil {
		return nil, err
	}

	tenantUUID, err := parseTenantUUID(user.TenantID)
	if err != nil {
		return nil, err
	}

	role, err := s.repo.GetRoleByName(ctx, tenantUUID, roleKey)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, ErrRoleNotFound
	}

	bundles := cloneStringSlice(role.BundleIDs)
	if bundles == nil {
		bundles = []string{}
	}
	return bundles, nil
}

func (s *roleServiceImpl) authorize(user models.User, action, resource string) error {
	if s.policyService == nil {
		return nil
	}
	policies := s.getPoliciesForResource(resource)
	allowed, err := s.policyService.Can(user, action, resource, policies)
	if err != nil {
		return err
	}
	if !allowed {
		principal := user.ID
		if principal == "" {
			principal = user.Email
		}
		if principal == "" {
			principal = "anonymous"
		}
		return fmt.Errorf("principal %s is not authorized to %s %s", principal, action, resource)
	}
	return nil
}

func (s *roleServiceImpl) getPoliciesForResource(resource string) []models.Policy {
	if s.policies == nil {
		return nil
	}
	var policies []models.Policy
	if p, ok := s.policies[resource]; ok {
		policies = append(policies, p...)
	}
	if resource != "role" {
		if p, ok := s.policies["role"]; ok {
			policies = append(policies, p...)
		}
	}
	if p, ok := s.policies["role:*"]; ok {
		policies = append(policies, p...)
	}
	return policies
}

func (s *roleServiceImpl) seedDefaults() {
	sqlRepo, ok := s.repo.(*sqlRoleRepository)
	if !ok || sqlRepo.db == nil || sqlRepo.db.DB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	seeds := []struct {
		name        string
		description string
		status      models.RoleStatus
		roleType    models.RoleType
		tags        []string
		bundles     []string
	}{
		{
			name:        "PortfolioManager",
			description: "Front office role for managing portfolios.",
			status:      models.RoleStatusActive,
			roleType:    models.RoleTypeBusiness,
			tags:        []string{"seed", "investment"},
			bundles:     []string{"front_office_performance_v1.3"},
		},
		{
			name:        "RiskOfficer",
			description: "Middle office role for monitoring risk.",
			status:      models.RoleStatusActive,
			roleType:    models.RoleTypeBusiness,
			tags:        []string{"seed", "risk"},
			bundles:     nil,
		},
		{
			name:        "DataSteward",
			description: "Data governance steward responsible for semantic definitions.",
			status:      models.RoleStatusActive,
			roleType:    models.RoleTypeTechnical,
			tags:        []string{"seed", "governance"},
			bundles:     nil,
		},
	}

	for _, seed := range seeds {
		normalized := normalizeRoleKey(seed.name)
		exists, err := s.repo.RoleExists(ctx, nil, normalized)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("role seed existence check failed for %s: %v", seed.name, err)
			continue
		}
		if exists {
			continue
		}

		role := newSeedRole(seed.name, seed.description, seed.status, seed.roleType, seed.tags, seed.bundles)
		if err := s.repo.CreateRole(ctx, role); err != nil {
			logging.GetLogger().Sugar().Warnf("role seed create failed for %s: %v", seed.name, err)
			continue
		}

		if s.bundleAccess != nil {
			for _, bundleID := range role.BundleIDs {
				if err := s.bundleAccess.AssignRoleToBundle(role.Name, bundleID); err != nil {
					logging.GetLogger().Sugar().Warnf("seed bundle assignment failed for role %s bundle %s: %v", role.Name, bundleID, err)
				}
			}
		}
	}
}

func newSeedRole(name, description string, status models.RoleStatus, roleType models.RoleType, tags []string, bundles []string) *models.Role {
	now := time.Now()
	lifecycle := models.RoleLifecycle{
		CreatedAt:  now,
		LastAction: "seed",
		LastActor:  "system",
		LastNotes:  "Role seeded",
	}

	switch status {
	case models.RoleStatusActive:
		lifecycle.ActivatedAt = timePtr(now)
	case models.RoleStatusSuspended:
		lifecycle.SuspendedAt = timePtr(now)
	case models.RoleStatusRetired:
		lifecycle.RetiredAt = timePtr(now)
	}

	normalizedTags := normalizeTags(append(tags, "seed"))
	normalizedBundles := normalizeBundleIDs(bundles)

	role := &models.Role{
		ID:          uuid.New().String(),
		Name:        name,
		DisplayName: formatRoleDisplayName(name),
		Description: description,
		Version:     "1.0.0",
		Status:      status,
		Type:        roleType,
		Owner:       "system",
		Scope:       models.RoleScopeGlobal,
		TenantID:    "",
		Tags:        normalizedTags,
		Attributes:  map[string]string{},
		Permissions: []models.RolePermission{},
		Members:     []models.RoleMember{},
		BundleIDs:   normalizedBundles,
		AuditTrail:  []models.RoleChangeRecord{},
		AuditMetadata: &models.RoleAuditMetadata{
			CreatedBy: "system",
			CreatedAt: now,
		},
		Lifecycle: lifecycle,
		CreatedAt: now,
		UpdatedAt: now,
	}

	recordRoleAudit(role, "seed", "Role seeded", "system", now)
	return role
}

func transitionRoleStatus(role *models.Role, status models.RoleStatus, actor string, notes string, ts time.Time) {
	if role.Status == status {
		return
	}

	switch status {
	case models.RoleStatusActive:
		role.Lifecycle.ActivatedAt = timePtr(ts)
	case models.RoleStatusSuspended:
		role.Lifecycle.SuspendedAt = timePtr(ts)
	case models.RoleStatusRetired:
		role.Lifecycle.RetiredAt = timePtr(ts)
	}

	role.Status = status
	message := fmt.Sprintf("Status changed to %s", status)
	if strings.TrimSpace(notes) != "" {
		message = fmt.Sprintf("%s - %s", message, strings.TrimSpace(notes))
	}
	recordRoleAudit(role, "status_change", message, actor, ts)
}

func recordRoleAudit(role *models.Role, action, notes, actorID string, ts time.Time) {
	ensureRoleAuditMetadata(role)
	role.AuditMetadata.LastModifiedBy = actorID
	role.AuditMetadata.LastModifiedAt = timePtr(ts)
	role.Lifecycle.LastAction = action
	role.Lifecycle.LastActor = actorID
	role.Lifecycle.LastNotes = notes
	role.UpdatedAt = ts
	role.AuditTrail = append(role.AuditTrail, models.RoleChangeRecord{
		Version:   role.Version,
		State:     string(role.Status),
		Action:    action,
		Actor:     actorID,
		Timestamp: ts,
		Notes:     notes,
	})
}

func ensureRoleAuditMetadata(role *models.Role) {
	if role.AuditMetadata == nil {
		role.AuditMetadata = &models.RoleAuditMetadata{
			CreatedBy: role.Owner,
			CreatedAt: role.CreatedAt,
		}
	}
}

func normalizeRoleKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Slice(normalized, func(i, j int) bool {
		return strings.ToLower(normalized[i]) < strings.ToLower(normalized[j])
	})
	return normalized
}

func sanitizeAttributes(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	cleaned := make(map[string]string, len(attrs))
	for key, value := range attrs {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		cleaned[trimmedKey] = strings.TrimSpace(value)
	}
	if len(cleaned) == 0 {
		return nil
	}
	return cleaned
}

func normalizeBundleIDs(bundleIDs []string) []string {
	if len(bundleIDs) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(bundleIDs))
	for _, id := range bundleIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}

func addBundleID(existing []string, bundleID string) ([]string, bool) {
	sanitized := strings.TrimSpace(bundleID)
	if sanitized == "" {
		return existing, false
	}
	for _, id := range existing {
		if id == sanitized {
			return existing, false
		}
	}
	updated := append(existing, sanitized)
	sort.Strings(updated)
	return updated, true
}

func removeBundleID(existing []string, bundleID string) ([]string, bool) {
	sanitized := strings.TrimSpace(bundleID)
	if sanitized == "" {
		return existing, false
	}
	idx := -1
	for i, id := range existing {
		if id == sanitized {
			idx = i
			break
		}
	}
	if idx == -1 {
		return existing, false
	}
	updated := append(existing[:idx], existing[idx+1:]...)
	if len(updated) == 0 {
		return nil, true
	}
	sort.Strings(updated)
	return updated, true
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func attributesEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if other, ok := b[key]; !ok || other != value {
			return false
		}
	}
	return true
}

func parseTenantUUID(raw string) (*uuid.UUID, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	id, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant id %s: %w", raw, err)
	}
	return &id, nil
}

func timePtr(t time.Time) *time.Time {
	tt := t
	return &tt
}

func copyTimePtr(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	tt := *t
	return &tt
}

func formatRoleDisplayName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	fields := strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == ' '
	})
	for i, field := range fields {
		if field == "" {
			continue
		}
		lower := strings.ToLower(field)
		fields[i] = strings.ToUpper(lower[:1]) + lower[1:]
	}
	return strings.Join(fields, " ")
}

func cloneRole(role *models.Role) *models.Role {
	if role == nil {
		return nil
	}
	clone := *role
	clone.Tags = cloneStringSlice(role.Tags)
	clone.Attributes = cloneStringMap(role.Attributes)
	clone.Policies = clonePolicies(role.Policies)
	clone.Permissions = cloneRolePermissions(role.Permissions)
	clone.AttributeConstraints = cloneAttributeConditions(role.AttributeConstraints)
	clone.Members = cloneRoleMembers(role.Members)
	clone.BundleIDs = cloneStringSlice(role.BundleIDs)
	clone.AuditTrail = cloneRoleChangeRecords(role.AuditTrail)
	clone.Lifecycle = cloneLifecycle(role.Lifecycle)

	if role.AuditMetadata != nil {
		metadataCopy := *role.AuditMetadata
		metadataCopy.LastModifiedAt = copyTimePtr(role.AuditMetadata.LastModifiedAt)
		metadataCopy.LastReviewedAt = copyTimePtr(role.AuditMetadata.LastReviewedAt)
		clone.AuditMetadata = &metadataCopy
	}

	return &clone
}

func clonePolicies(policies []models.Policy) []models.Policy {
	if len(policies) == 0 {
		return nil
	}
	copies := make([]models.Policy, len(policies))
	for i, policy := range policies {
		copies[i] = models.Policy{
			ID:          policy.ID,
			Effect:      policy.Effect,
			Actions:     cloneStringSlice(policy.Actions),
			Resources:   cloneStringSlice(policy.Resources),
			Description: policy.Description,
			Conditions:  cloneAttributeConditions(policy.Conditions),
		}
	}
	return copies
}

func cloneAttributeConditions(conditions []models.AttributeCondition) []models.AttributeCondition {
	if len(conditions) == 0 {
		return nil
	}
	copies := make([]models.AttributeCondition, len(conditions))
	for i, condition := range conditions {
		copies[i] = models.AttributeCondition{
			Attribute: condition.Attribute,
			Operator:  condition.Operator,
			Values:    cloneStringSlice(condition.Values),
		}
	}
	return copies
}

func cloneRolePermissions(perms []models.RolePermission) []models.RolePermission {
	if len(perms) == 0 {
		return nil
	}
	copies := make([]models.RolePermission, len(perms))
	for i, perm := range perms {
		copies[i] = models.RolePermission{
			Resource:    perm.Resource,
			Actions:     cloneStringSlice(perm.Actions),
			Effect:      perm.Effect,
			Description: perm.Description,
			Conditions:  cloneAttributeConditions(perm.Conditions),
		}
	}
	return copies
}

func cloneRoleMembers(members []models.RoleMember) []models.RoleMember {
	if len(members) == 0 {
		return nil
	}
	copies := make([]models.RoleMember, len(members))
	for i, member := range members {
		copies[i] = models.RoleMember{
			UserID:     member.UserID,
			AssignedAt: member.AssignedAt,
			Attributes: cloneStringMap(member.Attributes),
		}
	}
	return copies
}

func cloneRoleChangeRecords(records []models.RoleChangeRecord) []models.RoleChangeRecord {
	if len(records) == 0 {
		return nil
	}
	copies := make([]models.RoleChangeRecord, len(records))
	copy(copies, records)
	return copies
}

func cloneLifecycle(lifecycle models.RoleLifecycle) models.RoleLifecycle {
	clone := lifecycle
	clone.ActivatedAt = copyTimePtr(lifecycle.ActivatedAt)
	clone.SuspendedAt = copyTimePtr(lifecycle.SuspendedAt)
	clone.RetiredAt = copyTimePtr(lifecycle.RetiredAt)
	return clone
}

func cloneStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	copies := make([]string, len(values))
	copy(copies, values)
	return copies
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	copies := make(map[string]string, len(values))
	for key, value := range values {
		copies[key] = value
	}
	return copies
}

func marshalJSONOrDefault(value any, defaultJSON string) (json.RawMessage, error) {
	if value == nil {
		return json.RawMessage(defaultJSON), nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return json.RawMessage(defaultJSON), nil
	}
	return json.RawMessage(data), nil
}

func unmarshalJSONOrDefault(data json.RawMessage, defaultJSON string, dest any) error {
	raw := []byte(defaultJSON)
	if len(data) > 0 {
		raw = data
	}
	if string(raw) == "null" {
		raw = []byte(defaultJSON)
	}
	return json.Unmarshal(raw, dest)
}

func roleToRow(role *models.Role) (roleRow, error) {
	var row roleRow
	if role == nil {
		return row, fmt.Errorf("role cannot be nil")
	}

	var id uuid.UUID
	var err error
	if strings.TrimSpace(role.ID) == "" {
		id = uuid.New()
		role.ID = id.String()
	} else {
		id, err = uuid.Parse(role.ID)
		if err != nil {
			return row, fmt.Errorf("invalid role id %s: %w", role.ID, err)
		}
	}
	row.ID = id

	if strings.TrimSpace(role.TenantID) != "" {
		tenantID, err := uuid.Parse(strings.TrimSpace(role.TenantID))
		if err != nil {
			return row, fmt.Errorf("invalid tenant id %s: %w", role.TenantID, err)
		}
		row.TenantID = &tenantID
	}

	row.Name = role.Name
	row.NormalizedName = normalizeRoleKey(role.Name)
	row.DisplayName = role.DisplayName
	if strings.TrimSpace(role.Description) != "" {
		row.Description = sql.NullString{String: role.Description, Valid: true}
	}
	row.Version = role.Version
	row.Status = string(role.Status)
	row.RoleType = string(role.Type)
	row.Owner = role.Owner
	row.Scope = string(role.Scope)

	if row.Tags, err = marshalJSONOrDefault(role.Tags, "[]"); err != nil {
		return row, err
	}
	if row.Attributes, err = marshalJSONOrDefault(role.Attributes, "{}"); err != nil {
		return row, err
	}
	if row.Policies, err = marshalJSONOrDefault(role.Policies, "[]"); err != nil {
		return row, err
	}
	if row.Permissions, err = marshalJSONOrDefault(role.Permissions, "[]"); err != nil {
		return row, err
	}
	if row.AttributeConstraints, err = marshalJSONOrDefault(role.AttributeConstraints, "[]"); err != nil {
		return row, err
	}
	if row.Members, err = marshalJSONOrDefault(role.Members, "[]"); err != nil {
		return row, err
	}
	if row.BundleIDs, err = marshalJSONOrDefault(role.BundleIDs, "[]"); err != nil {
		return row, err
	}
	if row.AuditTrail, err = marshalJSONOrDefault(role.AuditTrail, "[]"); err != nil {
		return row, err
	}
	if row.AuditMetadata, err = marshalJSONOrDefault(role.AuditMetadata, "{}"); err != nil {
		return row, err
	}
	if row.Lifecycle, err = marshalJSONOrDefault(role.Lifecycle, "{}"); err != nil {
		return row, err
	}

	row.CreatedAt = role.CreatedAt
	row.UpdatedAt = role.UpdatedAt
	return row, nil
}

func roleRowToModel(row roleRow) (*models.Role, error) {
	role := &models.Role{
		ID:                   row.ID.String(),
		Name:                 row.Name,
		DisplayName:          row.DisplayName,
		Description:          "",
		Version:              row.Version,
		Status:               models.RoleStatus(row.Status),
		Type:                 models.RoleType(row.RoleType),
		Owner:                row.Owner,
		Scope:                models.RoleScope(row.Scope),
		Tags:                 []string{},
		Attributes:           map[string]string{},
		Policies:             []models.Policy{},
		Permissions:          []models.RolePermission{},
		AttributeConstraints: []models.AttributeCondition{},
		Members:              []models.RoleMember{},
		BundleIDs:            []string{},
		AuditTrail:           []models.RoleChangeRecord{},
		Lifecycle:            models.RoleLifecycle{},
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}

	if row.Description.Valid {
		role.Description = row.Description.String
	}
	if row.TenantID != nil {
		role.TenantID = row.TenantID.String()
	}

	if err := unmarshalJSONOrDefault(row.Tags, "[]", &role.Tags); err != nil {
		return nil, fmt.Errorf("failed to decode role tags: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.Attributes, "{}", &role.Attributes); err != nil {
		return nil, fmt.Errorf("failed to decode role attributes: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.Policies, "[]", &role.Policies); err != nil {
		return nil, fmt.Errorf("failed to decode role policies: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.Permissions, "[]", &role.Permissions); err != nil {
		return nil, fmt.Errorf("failed to decode role permissions: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.AttributeConstraints, "[]", &role.AttributeConstraints); err != nil {
		return nil, fmt.Errorf("failed to decode role attribute constraints: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.Members, "[]", &role.Members); err != nil {
		return nil, fmt.Errorf("failed to decode role members: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.BundleIDs, "[]", &role.BundleIDs); err != nil {
		return nil, fmt.Errorf("failed to decode role bundle ids: %w", err)
	}
	if err := unmarshalJSONOrDefault(row.AuditTrail, "[]", &role.AuditTrail); err != nil {
		return nil, fmt.Errorf("failed to decode role audit trail: %w", err)
	}
	var auditMetadata models.RoleAuditMetadata
	if err := unmarshalJSONOrDefault(row.AuditMetadata, "{}", &auditMetadata); err != nil {
		return nil, fmt.Errorf("failed to decode role audit metadata: %w", err)
	}
	role.AuditMetadata = &auditMetadata
	if err := unmarshalJSONOrDefault(row.Lifecycle, "{}", &role.Lifecycle); err != nil {
		return nil, fmt.Errorf("failed to decode role lifecycle: %w", err)
	}

	role.Tags = normalizeTags(role.Tags)
	role.BundleIDs = normalizeBundleIDs(role.BundleIDs)

	return role, nil
}

func (r *sqlRoleRepository) CreateRole(ctx context.Context, role *models.Role) error {
	row, err := roleToRow(role)
	if err != nil {
		return err
	}

	query := `INSERT INTO semantic_roles (
		id, tenant_id, name, normalized_name, display_name, description, version, status, role_type, owner, scope,
		tags, attributes, policies, permissions, attribute_constraints, members, bundle_ids, audit_trail, audit_metadata, lifecycle,
		created_at, updated_at
	) VALUES (
		:id, :tenant_id, :name, :normalized_name, :display_name, :description, :version, :status, :role_type, :owner, :scope,
		:tags, :attributes, :policies, :permissions, :attribute_constraints, :members, :bundle_ids, :audit_trail, :audit_metadata, :lifecycle,
		:created_at, :updated_at
	)`

	_, err = r.db.NamedExecContext(ctx, query, row)
	return err
}

func (r *sqlRoleRepository) SaveRole(ctx context.Context, role *models.Role) error {
	row, err := roleToRow(role)
	if err != nil {
		return err
	}

	query := `UPDATE semantic_roles SET
		tenant_id = :tenant_id,
		name = :name,
		normalized_name = :normalized_name,
		display_name = :display_name,
		description = :description,
		version = :version,
		status = :status,
		role_type = :role_type,
		owner = :owner,
		scope = :scope,
		tags = :tags,
		attributes = :attributes,
		policies = :policies,
		permissions = :permissions,
		attribute_constraints = :attribute_constraints,
		members = :members,
		bundle_ids = :bundle_ids,
		audit_trail = :audit_trail,
		audit_metadata = :audit_metadata,
		lifecycle = :lifecycle,
		updated_at = :updated_at
	WHERE id = :id`

	_, err = r.db.NamedExecContext(ctx, query, row)
	return err
}

func (r *sqlRoleRepository) RoleExists(ctx context.Context, tenantID *uuid.UUID, normalized string) (bool, error) {
	query := `SELECT 1 FROM semantic_roles WHERE normalized_name = $1 AND tenant_id IS NOT DISTINCT FROM $2 LIMIT 1`
	var marker int
	err := r.db.GetContext(ctx, &marker, query, normalized, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *sqlRoleRepository) GetRoleByName(ctx context.Context, tenantID *uuid.UUID, normalized string) (*models.Role, error) {
	var row roleRow
	var err error
	switch {
	case tenantID == nil:
		err = r.db.GetContext(ctx, &row, roleSelectBase+" WHERE normalized_name = $1 AND tenant_id IS NULL LIMIT 1", normalized)
	default:
		err = r.db.GetContext(ctx, &row, roleSelectBase+" WHERE normalized_name = $1 AND (tenant_id = $2 OR tenant_id IS NULL) ORDER BY CASE WHEN tenant_id = $2 THEN 0 ELSE 1 END LIMIT 1", normalized, tenantID)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return roleRowToModel(row)
}

func (r *sqlRoleRepository) ListRoles(ctx context.Context, tenantID *uuid.UUID) ([]*models.Role, error) {
	rows := []roleRow{}
	var err error
	if tenantID == nil {
		err = r.db.SelectContext(ctx, &rows, roleSelectBase+" WHERE tenant_id IS NULL ORDER BY display_name")
	} else {
		err = r.db.SelectContext(ctx, &rows, roleSelectBase+" WHERE tenant_id = $1 OR tenant_id IS NULL ORDER BY CASE WHEN tenant_id = $1 THEN 0 ELSE 1 END, display_name", tenantID)
	}
	if err != nil {
		return nil, err
	}

	roles := make([]*models.Role, 0, len(rows))
	for _, row := range rows {
		role, err := roleRowToModel(row)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *sqlRoleRepository) DeleteRole(ctx context.Context, tenantID *uuid.UUID, normalized string) error {
	query := `DELETE FROM semantic_roles WHERE normalized_name = $1 AND tenant_id IS NOT DISTINCT FROM $2`
	res, err := r.db.ExecContext(ctx, query, normalized, tenantID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrRoleNotFound
	}
	return nil
}
