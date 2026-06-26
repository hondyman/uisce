package services

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/jmoiron/sqlx"
)

// BundleService defines the interface for managing data bundles.
type BundleService interface {
	CreateBundle(user models.User, name, description string) (*models.DataBundle, error)
	GetBundle(user models.User, id string) (*models.DataBundle, error)
	ListBundles(user models.User) ([]*models.DataBundle, error)
	UpdateBundle(user models.User, id string, measures []models.SemanticObjectReference, dimensions []models.SemanticObjectReference) (*models.DataBundle, error)
	UpdateBundlePolicies(user models.User, id string, rowPolicies []models.BundleRowPolicy, columnPolicies []models.BundleColumnPolicy) (*models.DataBundle, error)
	CertifyBundle(user models.User, id string) (*models.DataBundle, error)
	PublishBundle(user models.User, id string) (*models.DataBundle, error)
	DeprecateBundle(user models.User, id string) (*models.DataBundle, error)
}

// BundleRoleManager exposes runtime methods for managing bundle ↔ role relationships.
type BundleRoleManager interface {
	AssignRoleToBundle(roleName, bundleID string) error
	UnassignRoleFromBundle(roleName, bundleID string) error
	GetBundleIDsForRole(roleName string) []string
	GetRolesForBundle(bundleID string) []string
}

// NewBundleService creates a new instance of the bundle service along with its role manager.
func NewBundleService(policyService platform.PolicyService) (BundleService, BundleRoleManager) {
	store := &inMemoryBundleStore{
		bundles:     make(map[string]*models.DataBundle),
		policies:    make(map[string][]models.Policy),
		roleBundles: make(map[string]map[string]struct{}),
		bundleRoles: make(map[string]map[string]struct{}),
	}

	// no hardcoded seed here; bundles should be persisted in the database and
	// optionally loaded into the in-memory store during server startup.

	// Baseline policies - in a production system these would come from policy store.
	store.policies["bundle"] = []models.Policy{
		{
			ID:          "bundle-create-steward",
			Effect:      "allow",
			Actions:     []string{"create"},
			Resources:   []string{"bundle"},
			Description: "Stewards can create bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Steward"}},
			},
		},
		{
			ID:          "bundle-create-admin",
			Effect:      "allow",
			Actions:     []string{"create"},
			Resources:   []string{"bundle"},
			Description: "Admins can create bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
			},
		},
	}
	store.policies["bundle:*"] = []models.Policy{
		{
			ID:          "bundle-read-steward",
			Effect:      "allow",
			Actions:     []string{"read"},
			Resources:   []string{"bundle:*"},
			Description: "Stewards can read all bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Steward"}},
			},
		},
		{
			ID:          "bundle-read-admin",
			Effect:      "allow",
			Actions:     []string{"read"},
			Resources:   []string{"bundle:*"},
			Description: "Admins can read all bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
			},
		},
		{
			ID:          "bundle-update-steward",
			Effect:      "allow",
			Actions:     []string{"update", "certify"},
			Resources:   []string{"bundle:*"},
			Description: "Stewards can update and certify bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Steward"}},
			},
		},
		{
			ID:          "bundle-update-admin",
			Effect:      "allow",
			Actions:     []string{"update", "certify"},
			Resources:   []string{"bundle:*"},
			Description: "Admins can update and certify bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
			},
		},
		{
			ID:          "bundle-publish-admin",
			Effect:      "allow",
			Actions:     []string{"publish", "deprecate"},
			Resources:   []string{"bundle:*"},
			Description: "Admins can publish and deprecate bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "contains", Values: []string{"Admin"}},
			},
		},
		{
			ID:          "bundle-publish-permission",
			Effect:      "allow",
			Actions:     []string{"publish", "deprecate"},
			Resources:   []string{"bundle:*"},
			Description: "Users with bundle:publish permission can publish/deprecate bundles",
			Conditions: []models.AttributeCondition{
				{Attribute: "permissions", Operator: "contains", Values: []string{"bundle:publish"}},
			},
		},
	}

	svc := &bundleServiceImpl{
		store:         store,
		policyService: policyService,
	}

	return svc, svc
}

// LoadBundlesIntoService is an exported helper that will try to load bundles from
// the provided sqlx.DB into the given BundleService if it is backed by the
// in-memory implementation. This avoids leaking concrete types across packages.
func LoadBundlesIntoService(svc BundleService, db *sqlx.DB) error {
	if db == nil {
		return nil
	}
	// try to cast to concrete type
	if impl, ok := svc.(*bundleServiceImpl); ok {
		return impl.LoadBundlesFromDB(db)
	}
	return nil
}

// InsertBundleForTesting inserts a DataBundle into the in-memory store when
// the provided BundleService is backed by the in-memory implementation.
// This is only intended for tests.
func InsertBundleForTesting(svc BundleService, b *models.DataBundle) error {
	if impl, ok := svc.(*bundleServiceImpl); ok {
		impl.store.mu.Lock()
		impl.store.bundles[b.ID] = b
		impl.store.mu.Unlock()
		return nil
	}
	return fmt.Errorf("service is not in-memory bundle service")
}

// inMemoryBundleStore simulates a database for bundles and their policies.
type inMemoryBundleStore struct {
	mu          sync.RWMutex
	bundles     map[string]*models.DataBundle
	policies    map[string][]models.Policy // Keyed by resource, e.g., "bundle" or "bundle:bundle-id"
	roleBundles map[string]map[string]struct{}
	bundleRoles map[string]map[string]struct{}
}

// bundleServiceImpl is the concrete implementation of the BundleService.
type bundleServiceImpl struct {
	store         *inMemoryBundleStore
	policyService platform.PolicyService
}

func (s *bundleServiceImpl) CreateBundle(user models.User, name, description string) (*models.DataBundle, error) {
	resource := "bundle"
	policies := s.store.policies[resource]

	allowed, err := s.policyService.Can(user, "create", resource, policies)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("user does not have permission to create bundles")
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	now := time.Now()
	effectiveFrom := now
	manifest := models.BundleManifest{
		Identifier: name,
		Summary:    description,
	}
	manifest.LastSyncedAt = &now
	if user.ID != "" || user.Name != "" || user.Email != "" {
		maintainer := models.BundleMaintainer{
			ID:    user.ID,
			Name:  user.Name,
			Email: strings.ToLower(strings.TrimSpace(user.Email)),
			Role:  user.Role,
		}
		manifest.Maintainers = append(manifest.Maintainers, maintainer)
	}
	auditMetadata := &models.BundleAuditMetadata{
		CreatedBy: user.ID,
		CreatedAt: now,
	}
	newBundle := &models.DataBundle{
		ID:             uuid.New().String(),
		Name:           name,
		Version:        "1.0.0",
		Status:         models.StatusDraft,
		Description:    description,
		Owner:          user.ID,
		EffectiveFrom:  &effectiveFrom,
		CreatedAt:      now,
		UpdatedAt:      now,
		Composition:    models.BundleComposition{},
		RowPolicies:    []models.BundleRowPolicy{},
		ColumnPolicies: []models.BundleColumnPolicy{},
		Lifecycle: models.BundleLifecycle{
			DraftedAt:  now,
			LastAction: "create",
			LastActor:  user.ID,
		},
		Manifest: manifest,
		AuditTrail: []models.BundleChangeRecord{{
			Version:   "1.0.0",
			State:     string(models.StatusDraft),
			Action:    "create",
			Actor:     user.ID,
			Timestamp: now,
			Notes:     "Bundle created",
		}},
		AuditMetadata: auditMetadata,
	}

	s.store.bundles[newBundle.ID] = newBundle

	defaultRoles := s.defaultRolesForUser(user)
	for _, role := range defaultRoles {
		_ = s.assignRoleToBundleLocked(role, newBundle.ID)
	}
	s.refreshBundleRoleMetadataLocked(newBundle.ID)
	return newBundle, nil
}

func (s *bundleServiceImpl) GetBundle(user models.User, id string) (*models.DataBundle, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	bundle, exists := s.store.bundles[id]
	if !exists {
		return nil, fmt.Errorf("bundle with id %s not found", id)
	}

	resource := fmt.Sprintf("bundle:%s", id)
	policies := s.getPoliciesForResource(resource)

	allowed, err := s.policyService.Can(user, "read", resource, policies)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("user does not have permission to read bundle %s", id)
	}

	return bundle, nil
}

func (s *bundleServiceImpl) ListBundles(user models.User) ([]*models.DataBundle, error) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()

	var accessibleBundles []*models.DataBundle
	for _, bundle := range s.store.bundles {
		resource := fmt.Sprintf("bundle:%s", bundle.ID)
		policies := s.getPoliciesForResource(resource)
		allowed, _ := s.policyService.Can(user, "read", resource, policies)
		if allowed {
			accessibleBundles = append(accessibleBundles, bundle)
		}
	}

	return accessibleBundles, nil
}

func (s *bundleServiceImpl) UpdateBundle(user models.User, id string, measures []models.SemanticObjectReference, dimensions []models.SemanticObjectReference) (*models.DataBundle, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	bundle, exists := s.store.bundles[id]
	if !exists {
		return nil, fmt.Errorf("bundle with id %s not found", id)
	}

	if bundle.Status != models.StatusDraft {
		return nil, fmt.Errorf("bundle can only be updated in Draft status")
	}

	resource := fmt.Sprintf("bundle:%s", id)
	policies := s.getPoliciesForResource(resource)

	allowed, err := s.policyService.Can(user, "update", resource, policies)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("user does not have permission to update bundle %s", id)
	}

	updateTime := time.Now()
	bundle.Measures = measures
	bundle.Dimensions = dimensions
	bundle.Composition = models.BundleComposition{
		Measures:   measures,
		Dimensions: dimensions,
	}
	bundle.UpdatedAt = updateTime
	bundle.Lifecycle.LastAction = "update"
	bundle.Lifecycle.LastActor = user.ID
	bundle.Lifecycle.LastNotes = "Updated bundle composition"
	if bundle.Manifest.LastSyncedAt == nil {
		bundle.Manifest.LastSyncedAt = &updateTime
	} else {
		*bundle.Manifest.LastSyncedAt = updateTime
	}
	if bundle.AuditMetadata == nil {
		bundle.AuditMetadata = &models.BundleAuditMetadata{CreatedBy: bundle.Owner, CreatedAt: bundle.CreatedAt}
	}
	bundle.AuditMetadata.LastModifiedBy = user.ID
	bundle.AuditMetadata.LastModifiedAt = &updateTime
	bundle.AuditTrail = append(bundle.AuditTrail, models.BundleChangeRecord{
		Version:   bundle.Version,
		State:     string(bundle.Status),
		Action:    "update",
		Actor:     user.ID,
		Timestamp: updateTime,
		Notes:     "Updated bundle composition",
	})

	s.store.bundles[id] = bundle
	return bundle, nil
}

func (s *bundleServiceImpl) UpdateBundlePolicies(user models.User, id string, rowPolicies []models.BundleRowPolicy, columnPolicies []models.BundleColumnPolicy) (*models.DataBundle, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	bundle, exists := s.store.bundles[id]
	if !exists {
		return nil, fmt.Errorf("bundle with id %s not found", id)
	}

	if bundle.Status == models.StatusDeprecated {
		return nil, fmt.Errorf("bundle %s is deprecated and cannot be modified", id)
	}

	resource := fmt.Sprintf("bundle:%s", id)
	policies := s.getPoliciesForResource(resource)

	allowed, err := s.policyService.Can(user, "update", resource, policies)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("user does not have permission to update policies for bundle %s", id)
	}

	sanitizedRows := sanitizeRowPolicies(rowPolicies)
	sanitizedColumns := sanitizeColumnPolicies(columnPolicies)

	if validationErrs := validateBundlePolicies(sanitizedRows, sanitizedColumns); len(validationErrs) > 0 {
		return nil, NewValidationError("bundle policy validation failed", validationErrs)
	}

	updateTime := time.Now()
	bundle.RowPolicies = sanitizedRows
	bundle.ColumnPolicies = sanitizedColumns
	bundle.UpdatedAt = updateTime
	bundle.Lifecycle.LastAction = "update_policies"
	bundle.Lifecycle.LastActor = user.ID
	bundle.Lifecycle.LastNotes = "Updated bundle policies"
	if bundle.Manifest.LastSyncedAt == nil {
		bundle.Manifest.LastSyncedAt = &updateTime
	} else {
		*bundle.Manifest.LastSyncedAt = updateTime
	}
	if bundle.AuditMetadata == nil {
		bundle.AuditMetadata = &models.BundleAuditMetadata{CreatedBy: bundle.Owner, CreatedAt: bundle.CreatedAt}
	}
	bundle.AuditMetadata.LastModifiedBy = user.ID
	bundle.AuditMetadata.LastModifiedAt = &updateTime
	bundle.AuditTrail = append(bundle.AuditTrail, models.BundleChangeRecord{
		Version:   bundle.Version,
		State:     string(bundle.Status),
		Action:    "update_policies",
		Actor:     user.ID,
		Timestamp: updateTime,
		Notes:     "Updated bundle policies",
	})

	s.store.bundles[id] = bundle
	return bundle, nil
}

// LoadBundlesFromDB loads private_markets_bundles rows into the in-memory store.
// This is used at server startup to populate the in-memory store with persisted bundles.
func (s *bundleServiceImpl) LoadBundlesFromDB(db *sqlx.DB) error {
	if db == nil {
		return nil
	}

	rows, err := db.Queryx(`
		SELECT bundle_id, name, audience, version, modules, metrics, governance, is_active, created_at, updated_at
		FROM private_markets_bundles
		WHERE is_active = true
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	temporary := make(map[string]*models.DataBundle)

	for rows.Next() {
		var (
			bundleID, name, audience, version string
			modules, metrics, governance      sql.NullString
			isActive                          bool
			createdAt, updatedAt              time.Time
		)

		if err := rows.Scan(&bundleID, &name, &audience, &version, &modules, &metrics, &governance, &isActive, &createdAt, &updatedAt); err != nil {
			return err
		}

		bundle, err := bundleFromDBRow(bundleID, name, audience, version, modules, metrics, governance, isActive, createdAt, updatedAt)
		if err != nil {
			return err
		}

		temporary[bundle.ID] = bundle
	}

	if err := rows.Err(); err != nil {
		return err
	}

	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	s.store.bundles = make(map[string]*models.DataBundle, len(temporary))
	for id, bundle := range temporary {
		s.store.bundles[id] = bundle
	}

	return nil
}

func (s *bundleServiceImpl) CertifyBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.updateBundleStatus(user, id, "certify", models.StatusDraft, models.StatusCertified)
}

func (s *bundleServiceImpl) PublishBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.updateBundleStatus(user, id, "publish", models.StatusCertified, models.StatusPublished)
}

func (s *bundleServiceImpl) DeprecateBundle(user models.User, id string) (*models.DataBundle, error) {
	return s.updateBundleStatus(user, id, "deprecate", models.StatusPublished, models.StatusDeprecated)
}

// updateBundleStatus is a helper to handle lifecycle transitions.
func (s *bundleServiceImpl) updateBundleStatus(user models.User, id, action string, fromStatus, toStatus models.BundleStatus) (*models.DataBundle, error) {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()

	bundle, exists := s.store.bundles[id]
	if !exists {
		return nil, fmt.Errorf("bundle with id %s not found", id)
	}

	if bundle.Status != fromStatus {
		return nil, fmt.Errorf("bundle must be in %s status to %s", fromStatus, action)
	}

	resource := fmt.Sprintf("bundle:%s", id)
	policies := s.getPoliciesForResource(resource)

	allowed, err := s.policyService.Can(user, action, resource, policies)
	if err != nil {
		return nil, err
	}
	if !allowed {
		return nil, fmt.Errorf("user does not have permission to %s bundle %s", action, id)
	}

	transitionTime := time.Now()
	bundle.Status = toStatus
	bundle.UpdatedAt = transitionTime
	bundle.Lifecycle.LastAction = action
	bundle.Lifecycle.LastActor = user.ID
	bundle.Lifecycle.LastNotes = fmt.Sprintf("Transitioned bundle to %s", toStatus)
	switch toStatus {
	case models.StatusCertified:
		bundle.Lifecycle.CertifiedAt = &transitionTime
	case models.StatusPublished:
		bundle.Lifecycle.PublishedAt = &transitionTime
		bundle.PublishedAt = &transitionTime
	case models.StatusDeprecated:
		bundle.Lifecycle.DeprecatedAt = &transitionTime
		bundle.DeprecatedAt = &transitionTime
	}
	if bundle.Manifest.LastSyncedAt == nil {
		bundle.Manifest.LastSyncedAt = &transitionTime
	} else {
		*bundle.Manifest.LastSyncedAt = transitionTime
	}
	if bundle.AuditMetadata == nil {
		bundle.AuditMetadata = &models.BundleAuditMetadata{CreatedBy: bundle.Owner, CreatedAt: bundle.CreatedAt}
	}
	bundle.AuditMetadata.LastModifiedBy = user.ID
	bundle.AuditMetadata.LastModifiedAt = &transitionTime
	if action == "certify" || action == "publish" {
		bundle.AuditMetadata.LastReviewedBy = user.ID
		bundle.AuditMetadata.LastReviewedAt = &transitionTime
	}
	bundle.AuditTrail = append(bundle.AuditTrail, models.BundleChangeRecord{
		Version:   bundle.Version,
		State:     string(toStatus),
		Action:    action,
		Actor:     user.ID,
		Timestamp: transitionTime,
		Notes:     bundle.Lifecycle.LastNotes,
	})

	s.store.bundles[id] = bundle
	return bundle, nil
}

// getPoliciesForResource retrieves all policies applicable to a resource.
func (s *bundleServiceImpl) getPoliciesForResource(resource string) []models.Policy {
	var policies []models.Policy
	// Add policies for the specific resource, e.g., "bundle:xyz"
	if p, ok := s.store.policies[resource]; ok {
		policies = append(policies, p...)
	}
	// Add wildcard policies, e.g., "bundle:*"
	if p, ok := s.store.policies["bundle:*"]; ok {
		policies = append(policies, p...)
	}
	return policies
}

// AssignRoleToBundle grants access to a bundle for the provided role.
func (s *bundleServiceImpl) AssignRoleToBundle(roleName, bundleID string) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	return s.assignRoleToBundleLocked(roleName, bundleID)
}

// UnassignRoleFromBundle revokes bundle access for the provided role.
func (s *bundleServiceImpl) UnassignRoleFromBundle(roleName, bundleID string) error {
	s.store.mu.Lock()
	defer s.store.mu.Unlock()
	return s.unassignRoleFromBundleLocked(roleName, bundleID)
}

// GetBundleIDsForRole returns the bundle IDs currently assigned to a role.
func (s *bundleServiceImpl) GetBundleIDsForRole(roleName string) []string {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	set, ok := s.store.roleBundles[roleName]
	if !ok || len(set) == 0 {
		return nil
	}
	ids := make([]string, 0, len(set))
	for id := range set {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// GetRolesForBundle lists roles that can access the bundle.
func (s *bundleServiceImpl) GetRolesForBundle(bundleID string) []string {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	return s.rolesForBundleLocked(bundleID)
}

func (s *bundleServiceImpl) defaultRolesForUser(user models.User) []string {
	roleSet := make(map[string]struct{})
	for _, role := range user.Roles {
		if role == "" {
			continue
		}
		roleSet[role] = struct{}{}
	}
	if user.Role != "" {
		roleSet[user.Role] = struct{}{}
	}
	if len(roleSet) == 0 {
		roleSet["Steward"] = struct{}{}
	}
	return sortedKeys(roleSet)
}

func sanitizeRowPolicies(policies []models.BundleRowPolicy) []models.BundleRowPolicy {
	if len(policies) == 0 {
		return nil
	}
	sanitized := make([]models.BundleRowPolicy, len(policies))
	for i, policy := range policies {
		sanitized[i] = models.BundleRowPolicy{
			ID:          strings.TrimSpace(policy.ID),
			Name:        strings.TrimSpace(policy.Name),
			Description: strings.TrimSpace(policy.Description),
			Member:      strings.TrimSpace(policy.Member),
			Operator:    strings.TrimSpace(policy.Operator),
			Values:      sanitizeStringList(policy.Values),
			Conditions:  sanitizeAttributeConditions(policy.Conditions),
		}
	}
	return sanitized
}

func sanitizeColumnPolicies(policies []models.BundleColumnPolicy) []models.BundleColumnPolicy {
	if len(policies) == 0 {
		return nil
	}
	sanitized := make([]models.BundleColumnPolicy, len(policies))
	for i, policy := range policies {
		sanitized[i] = models.BundleColumnPolicy{
			ID:          strings.TrimSpace(policy.ID),
			Name:        strings.TrimSpace(policy.Name),
			Description: strings.TrimSpace(policy.Description),
			Columns:     sanitizeStringList(policy.Columns),
			MaskType:    strings.TrimSpace(policy.MaskType),
			MaskValue:   strings.TrimSpace(policy.MaskValue),
			Conditions:  sanitizeAttributeConditions(policy.Conditions),
		}
	}
	return sanitized
}

func sanitizeAttributeConditions(conditions []models.AttributeCondition) []models.AttributeCondition {
	if len(conditions) == 0 {
		return nil
	}
	sanitized := make([]models.AttributeCondition, len(conditions))
	for i, condition := range conditions {
		sanitized[i] = models.AttributeCondition{
			Attribute: strings.TrimSpace(condition.Attribute),
			Operator:  strings.TrimSpace(condition.Operator),
			Values:    sanitizeStringList(condition.Values),
		}
	}
	return sanitized
}

func sanitizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	clean := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	if len(clean) == 0 {
		return nil
	}
	return clean
}

func validateBundlePolicies(rowPolicies []models.BundleRowPolicy, columnPolicies []models.BundleColumnPolicy) []FieldError {
	var errors []FieldError
	allowedRowOperators := map[string]struct{}{
		"equals":     {},
		"in":         {},
		"not_equals": {},
		"contains":   {},
	}
	allowedMaskTypes := map[string]struct{}{
		"redact": {},
		"null":   {},
		"hash":   {},
	}

	for i, policy := range rowPolicies {
		prefix := fmt.Sprintf("rowPolicies[%d]", i)
		if policy.Name == "" {
			errors = append(errors, FieldError{Field: prefix + ".name", Message: "Name is required"})
		}
		if policy.Member == "" {
			errors = append(errors, FieldError{Field: prefix + ".member", Message: "Member is required"})
		}
		if policy.Operator == "" {
			errors = append(errors, FieldError{Field: prefix + ".operator", Message: "Operator is required"})
		} else if _, ok := allowedRowOperators[policy.Operator]; !ok {
			errors = append(errors, FieldError{Field: prefix + ".operator", Message: "Operator is not supported"})
		}
		if len(policy.Values) == 0 {
			errors = append(errors, FieldError{Field: prefix + ".values", Message: "At least one value is required"})
		}
		for j, condition := range policy.Conditions {
			conditionPrefix := fmt.Sprintf("%s.conditions[%d]", prefix, j)
			if condition.Attribute == "" {
				errors = append(errors, FieldError{Field: conditionPrefix + ".attribute", Message: "Attribute is required"})
			}
			if condition.Operator == "" {
				errors = append(errors, FieldError{Field: conditionPrefix + ".operator", Message: "Operator is required"})
			} else if _, ok := allowedRowOperators[condition.Operator]; !ok {
				errors = append(errors, FieldError{Field: conditionPrefix + ".operator", Message: "Operator is not supported"})
			}
			if len(condition.Values) == 0 {
				errors = append(errors, FieldError{Field: conditionPrefix + ".values", Message: "At least one value is required"})
			}
		}
	}

	for i, policy := range columnPolicies {
		prefix := fmt.Sprintf("columnPolicies[%d]", i)
		if policy.Name == "" {
			errors = append(errors, FieldError{Field: prefix + ".name", Message: "Name is required"})
		}
		if len(policy.Columns) == 0 {
			errors = append(errors, FieldError{Field: prefix + ".columns", Message: "At least one column is required"})
		}
		if policy.MaskType == "" {
			errors = append(errors, FieldError{Field: prefix + ".maskType", Message: "Mask type is required"})
		} else if _, ok := allowedMaskTypes[policy.MaskType]; !ok {
			errors = append(errors, FieldError{Field: prefix + ".maskType", Message: "Mask type is not supported"})
		}
		for j, condition := range policy.Conditions {
			conditionPrefix := fmt.Sprintf("%s.conditions[%d]", prefix, j)
			if condition.Attribute == "" {
				errors = append(errors, FieldError{Field: conditionPrefix + ".attribute", Message: "Attribute is required"})
			}
			if condition.Operator == "" {
				errors = append(errors, FieldError{Field: conditionPrefix + ".operator", Message: "Operator is required"})
			} else if _, ok := allowedRowOperators[condition.Operator]; !ok {
				errors = append(errors, FieldError{Field: conditionPrefix + ".operator", Message: "Operator is not supported"})
			}
			if len(condition.Values) == 0 {
				errors = append(errors, FieldError{Field: conditionPrefix + ".values", Message: "At least one value is required"})
			}
		}
	}

	return errors
}

func (s *bundleServiceImpl) assignRoleToBundleLocked(roleName, bundleID string) error {
	if roleName == "" {
		return nil
	}
	if _, exists := s.store.bundles[bundleID]; !exists {
		return fmt.Errorf("bundle with id %s not found", bundleID)
	}
	if s.store.roleBundles[roleName] == nil {
		s.store.roleBundles[roleName] = make(map[string]struct{})
	}
	if s.store.bundleRoles[bundleID] == nil {
		s.store.bundleRoles[bundleID] = make(map[string]struct{})
	}
	if _, already := s.store.roleBundles[roleName][bundleID]; already {
		return nil
	}
	s.store.roleBundles[roleName][bundleID] = struct{}{}
	s.store.bundleRoles[bundleID][roleName] = struct{}{}
	s.refreshBundleRoleMetadataLocked(bundleID)
	return nil
}

func (s *bundleServiceImpl) unassignRoleFromBundleLocked(roleName, bundleID string) error {
	if roleName == "" {
		return nil
	}
	if _, exists := s.store.bundles[bundleID]; !exists {
		return fmt.Errorf("bundle with id %s not found", bundleID)
	}
	if assignments, ok := s.store.roleBundles[roleName]; ok {
		delete(assignments, bundleID)
		if len(assignments) == 0 {
			delete(s.store.roleBundles, roleName)
		}
	}
	if roles, ok := s.store.bundleRoles[bundleID]; ok {
		delete(roles, roleName)
		if len(roles) == 0 {
			delete(s.store.bundleRoles, bundleID)
		}
	}
	s.refreshBundleRoleMetadataLocked(bundleID)
	return nil
}

func (s *bundleServiceImpl) refreshBundleRoleMetadataLocked(bundleID string) {
	bundle, exists := s.store.bundles[bundleID]
	if !exists || bundle == nil {
		return
	}
	roles := s.rolesForBundleLocked(bundleID)
	bundle.AllowedRoles = roles
	s.store.bundles[bundleID] = bundle
	resource := fmt.Sprintf("bundle:%s", bundleID)
	s.store.policies[resource] = s.buildReadPoliciesForBundle(bundleID, roles)
}

func (s *bundleServiceImpl) rolesForBundleLocked(bundleID string) []string {
	roleSet, ok := s.store.bundleRoles[bundleID]
	if !ok || len(roleSet) == 0 {
		return nil
	}
	return sortedKeys(roleSet)
}

func (s *bundleServiceImpl) buildReadPoliciesForBundle(bundleID string, roles []string) []models.Policy {
	if len(roles) == 0 {
		return nil
	}
	resource := fmt.Sprintf("bundle:%s", bundleID)
	policyID := fmt.Sprintf("bundle-%s-read-assigned", bundleID)
	return []models.Policy{
		{
			ID:          policyID,
			Effect:      "allow",
			Actions:     []string{"read"},
			Resources:   []string{resource},
			Description: fmt.Sprintf("Assigned roles can read bundle %s", bundleID),
			Conditions: []models.AttributeCondition{
				{Attribute: "roles", Operator: "in", Values: roles},
			},
		},
	}
}

func sortedKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
