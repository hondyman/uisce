package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// CollaborationService provides methods for comments and approvals.
type CollaborationService struct {
	db *sqlx.DB

	accessPolicyRepo AccessPolicyRepository
	accessPolicyMu   sync.RWMutex
	accessPolicies   map[uuid.UUID]models.AccessControlPolicy
}

// NewCollaborationService creates a new CollaborationService.
func NewCollaborationService(db *sqlx.DB) *CollaborationService {
	svc := &CollaborationService{
		db:               db,
		accessPolicyRepo: newAccessPolicyRepository(db),
		accessPolicies:   make(map[uuid.UUID]models.AccessControlPolicy),
	}
	svc.initializeAccessPolicies()
	return svc
}

func (s *CollaborationService) initializeAccessPolicies() {
	if s.accessPolicyRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		policies, err := s.accessPolicyRepo.List(ctx)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to load access policies from repository: %v", err)
		}

		if len(policies) == 0 {
			for _, policy := range defaultAccessPolicies(time.Now().UTC()) {
				if _, err := s.accessPolicyRepo.Create(ctx, &policy); err != nil {
					logging.GetLogger().Sugar().Warnf("failed to seed access policy %s: %v", policy.PolicyID, err)
				}
			}
			policies, err = s.accessPolicyRepo.List(ctx)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("failed to reload access policies after seeding: %v", err)
			}
		}

		if len(policies) > 0 {
			s.accessPolicyMu.Lock()
			for _, policy := range policies {
				s.accessPolicies[policy.ID] = policy
			}
			s.accessPolicyMu.Unlock()
			return
		}
	}

	s.seedMockAccessPolicies()
}

// seedMockAccessPolicies populates the in-memory access policy catalog with sample data
// so the UI has meaningful records to display on first load. In production this would
// query a durable store instead of using mock data.
func (s *CollaborationService) seedMockAccessPolicies() {
	s.accessPolicyMu.Lock()
	defer s.accessPolicyMu.Unlock()

	if len(s.accessPolicies) > 0 {
		return
	}

	for _, p := range defaultAccessPolicies(time.Now().UTC()) {
		s.accessPolicies[p.ID] = p
	}
}

func defaultAccessPolicies(now time.Time) []models.AccessControlPolicy {
	renewalCond1, _ := json.Marshal(map[string]any{"usage_within_days": 14, "review_required": true})
	renewalCond2, _ := json.Marshal(map[string]any{"usage_within_days": 30, "review_required": false})

	return []models.AccessControlPolicy{
		{
			ID:                    uuid.New(),
			PolicyID:              "finance_read_default",
			Scope:                 "domain:finance",
			Role:                  "analyst",
			Permissions:           []string{"read"},
			DurationDays:          90,
			RequiresCertification: true,
			MaxClaimsPerUser:      5,
			ApprovalThreshold:     1,
			RenewalConditions:     renewalCond1,
			CreatedAt:             now.Add(-30 * 24 * time.Hour),
			UpdatedAt:             now.Add(-15 * 24 * time.Hour),
		},
		{
			ID:                    uuid.New(),
			PolicyID:              "sales_temp_access",
			Scope:                 "domain:sales",
			Role:                  "sales_manager",
			Permissions:           []string{"read", "download"},
			DurationDays:          30,
			RequiresCertification: false,
			MaxClaimsPerUser:      10,
			ApprovalThreshold:     2,
			RenewalConditions:     renewalCond2,
			CreatedAt:             now.Add(-10 * 24 * time.Hour),
			UpdatedAt:             now.Add(-3 * 24 * time.Hour),
		},
	}
}

func sortPoliciesForUI(policies []models.AccessControlPolicy) []models.AccessControlPolicy {
	if len(policies) == 0 {
		return policies
	}

	out := make([]models.AccessControlPolicy, len(policies))
	copy(out, policies)

	sort.Slice(out, func(i, j int) bool {
		if strings.EqualFold(out[i].PolicyID, out[j].PolicyID) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].PolicyID < out[j].PolicyID
	})

	return out
}

func validateAccessPolicyPayload(policy *models.AccessControlPolicy) error {
	if policy == nil {
		return errors.New("policy payload is required")
	}

	policy.PolicyID = strings.TrimSpace(policy.PolicyID)
	policy.Scope = strings.TrimSpace(policy.Scope)
	policy.Role = strings.TrimSpace(policy.Role)

	if policy.PolicyID == "" {
		return errors.New("policy_id is required")
	}
	if policy.Scope == "" {
		return errors.New("scope is required")
	}
	if policy.Role == "" {
		return errors.New("role is required")
	}
	if policy.ApprovalThreshold < 0 {
		return errors.New("approval_threshold must be >= 0")
	}
	if policy.DurationDays < 0 {
		return errors.New("duration_days must be >= 0")
	}
	if policy.MaxClaimsPerUser < 0 {
		return errors.New("max_claims_per_user must be >= 0")
	}

	return nil
}

func normalizeAccessPolicyDefaults(policy *models.AccessControlPolicy) {
	if policy == nil {
		return
	}

	cleaned := make([]string, 0, len(policy.Permissions))
	for _, perm := range policy.Permissions {
		if trimmed := strings.TrimSpace(perm); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	policy.Permissions = cleaned

	if len(policy.RenewalConditions) == 0 {
		policy.RenewalConditions = json.RawMessage(`{}`)
	}
}

// ListComments retrieves comments for an asset. Mock implementation.
func (s *CollaborationService) ListComments(ctx context.Context, assetID uuid.UUID) ([]models.Comment, error) {
	// Mock data
	parentID := uuid.New()
	return []models.Comment{
		{
			ID:           parentID,
			AssetID:      assetID,
			AssetType:    "query",
			AuthorUserID: "patrick",
			Body:         "This looks great, but can we add a filter for the APAC region by default?",
			CreatedAt:    time.Now().Add(-2 * time.Hour),
			Resolved:     false,
		},
		{
			ID:           uuid.New(),
			AssetID:      assetID,
			AssetType:    "query",
			AuthorUserID: "data_team_lead",
			Body:         "Good point. I've added the filter. Ready for certification.",
			CreatedAt:    time.Now().Add(-1 * time.Hour),
			Resolved:     false,
			ParentID:     &parentID,
		},
	}, nil
}

// AddComment adds a new comment. Mock implementation.
func (s *CollaborationService) AddComment(ctx context.Context, comment models.Comment) (*models.Comment, error) {
	comment.ID = uuid.New()
	comment.CreatedAt = time.Now()
	// In a real implementation, this would insert into the explorer_comment table.
	return &comment, nil
}

// GetApprovalStatus retrieves the approval status for an asset. Mock implementation.
func (s *CollaborationService) GetApprovalStatus(ctx context.Context, assetID uuid.UUID) (*models.Approval, error) {
	// In a real implementation, this would query the explorer_approval table.
	// We'll cycle through statuses for demonstration purposes.
	switch assetID.String()[0] % 4 {
	case 0:
		return &models.Approval{Status: "draft"}, nil
	case 1:
		return &models.Approval{Status: "pending", RequestedBy: "patrick"}, nil
	case 2:
		reviewedBy := "data_team_lead"
		decisionAt := time.Now().Add(-24 * time.Hour)
		return &models.Approval{Status: "approved", RequestedBy: "patrick", ReviewedBy: &reviewedBy, DecisionAt: &decisionAt}, nil
	default:
		reviewedBy := "data_team_lead"
		decisionAt := time.Now().Add(-48 * time.Hour)
		notes := "Rejected due to incorrect metric definition. Please use `net_revenue` instead."
		return &models.Approval{Status: "rejected", RequestedBy: "patrick", ReviewedBy: &reviewedBy, DecisionAt: &decisionAt, Notes: &notes}, nil
	}
}

// RequestApproval changes an asset's status to 'pending'. Mock implementation.
func (s *CollaborationService) RequestApproval(ctx context.Context, assetID uuid.UUID, assetType, userID string) (*models.Approval, error) {
	// In a real implementation, this would create or update a record in explorer_approval.
	return &models.Approval{
		ID:          uuid.New(),
		AssetID:     assetID,
		AssetType:   assetType,
		Status:      "pending",
		RequestedBy: userID,
	}, nil
}

// UpdateApprovalStatus updates an approval record. Mock implementation.
func (s *CollaborationService) UpdateApprovalStatus(ctx context.Context, approvalID uuid.UUID, status, reviewerID, notes string) (*models.Approval, error) {
	// In a real implementation, this would update the record.
	return &models.Approval{
		ID:         approvalID,
		Status:     status,
		ReviewedBy: &reviewerID, // This was correct
		Notes:      &notes,
		DecisionAt: func() *time.Time { t := time.Now(); return &t }(),
	}, nil
}

// --- Semantic Model Access Requests ---

// RequestSemanticModelAccess creates a new access request. Mock implementation.
func (s *CollaborationService) RequestSemanticModelAccess(ctx context.Context, userID string, modelID uuid.UUID, permission, reason string) (*models.SemanticModelAccessRequest, error) {
	req := &models.SemanticModelAccessRequest{
		ID:                  uuid.New(),
		UserID:              userID,
		ModelID:             modelID,
		RequestedPermission: permission,
		Reason:              reason,
		Status:              "pending",
		RequestedAt:         time.Now(),
	}
	// In a real app, this would be inserted into the semantic_model_access_request table.
	return req, nil
}

// ListAccessRequests retrieves access requests. Mock implementation.
func (s *CollaborationService) ListAccessRequests(ctx context.Context, userID, reviewerID string) ([]models.SemanticModelAccessRequest, error) {
	// Mock data
	mockRequests := []models.SemanticModelAccessRequest{
		{
			ID:                  uuid.New(),
			UserID:              "patrick",
			ModelID:             uuid.New(),
			RequestedPermission: "read",
			Reason:              "Need to explore churn metrics for Q3 analysis.",
			Status:              "pending",
			RequestedAt:         time.Now().Add(-2 * time.Hour),
		},
		{
			ID:                  uuid.New(),
			UserID:              "patrick",
			ModelID:             uuid.New(),
			RequestedPermission: "read",
			Reason:              "Building a new sales dashboard.",
			Status:              "approved",
			ReviewerID:          &reviewerID,
			RequestedAt:         time.Now().Add(-48 * time.Hour),
			DecidedAt:           func() *time.Time { t := time.Now().Add(-24 * time.Hour); return &t }(),
		},
	}
	return mockRequests, nil
}

// ApproveAccessRequest approves a request and grants a claim. Mock implementation.
func (s *CollaborationService) ApproveAccessRequest(ctx context.Context, requestID uuid.UUID, reviewerID string) error {
	// In a real app:
	// 1. Start a transaction.
	// 2. Update the request status to 'approved'.
	// 3. Insert a new record into `semantic_model_claim`.
	// 4. Commit transaction.
	logging.GetLogger().Sugar().Infof("Reviewer %s approved request %s", reviewerID, requestID)
	return nil
}

// RejectAccessRequest rejects a request. Mock implementation.
func (s *CollaborationService) RejectAccessRequest(ctx context.Context, requestID uuid.UUID, reviewerID, notes string) error {
	// In a real app:
	// 1. Update the request status to 'rejected'.
	// 2. Store the decision_notes.
	logging.GetLogger().Sugar().Infof("Reviewer %s rejected request %s with notes: %s", reviewerID, requestID, notes)
	return nil
}

// ListRoleClaims retrieves all role-to-model permission mappings. Mock implementation.
func (s *CollaborationService) ListRoleClaims(ctx context.Context) ([]models.SemanticModelRoleClaim, error) {
	// Mock data
	modelID1, _ := uuid.NewUUID()
	modelID2, _ := uuid.NewUUID()
	modelID3, _ := uuid.NewUUID()
	return []models.SemanticModelRoleClaim{
		{ID: uuid.New(), Role: "analyst", ModelID: modelID1, Permissions: []string{"read"}, GrantedBy: "admin", GrantedAt: time.Now().Add(-10 * 24 * time.Hour)},
		{ID: uuid.New(), Role: "finance_team", ModelID: modelID2, Permissions: []string{"read"}, GrantedBy: "admin", GrantedAt: time.Now().Add(-10 * 24 * time.Hour)},
		{ID: uuid.New(), Role: "finance_team", ModelID: modelID3, Permissions: []string{"read", "write"}, GrantedBy: "admin", GrantedAt: time.Now().Add(-5 * 24 * time.Hour)},
	}, nil
}

// UpdateRoleClaim updates the permissions for a role on a model. Mock implementation.
func (s *CollaborationService) UpdateRoleClaim(ctx context.Context, role, modelID string, permissions []string, actorID string) (*models.SemanticModelRoleClaim, error) {
	// In a real app, this would perform an UPSERT on semantic_model_role_claim
	// and log the change to the audit log.
	logging.GetLogger().Sugar().Infof("Actor %s updated role %s on model %s with perms %v", actorID, role, modelID, permissions)
	parsedModelID, _ := uuid.Parse(modelID)
	return &models.SemanticModelRoleClaim{
		ID:          uuid.New(),
		Role:        role,
		ModelID:     parsedModelID,
		Permissions: permissions,
		GrantedBy:   actorID,
		GrantedAt:   time.Now(),
	}, nil
}

// GrantDirectClaim grants a direct permission to a user. Mock implementation.
func (s *CollaborationService) GrantDirectClaim(ctx context.Context, userID, modelID, permission, actorID string) (*models.SemanticModelClaim, error) {
	logging.GetLogger().Sugar().Infof("Actor %s granted user %s permission %s on model %s", actorID, userID, permission, modelID)
	parsedModelID, _ := uuid.Parse(modelID)
	return &models.SemanticModelClaim{
		ID:         uuid.New(),
		UserID:     userID,
		ModelID:    parsedModelID,
		Permission: permission,
		GrantedBy:  actorID,
		GrantedAt:  time.Now(),
	}, nil
}

// RevokeDirectClaim revokes a direct permission from a user. Mock implementation.
func (s *CollaborationService) RevokeDirectClaim(ctx context.Context, claimID uuid.UUID, actorID string) error {
	// In a real app, this would DELETE from semantic_model_claim and log the action.
	logging.GetLogger().Sugar().Infof("Actor %s revoked claim %s", actorID, claimID)
	return nil
}

// GetStewardDomainsForUser retrieves the domains a user is a steward for. Mock implementation.
func (s *CollaborationService) GetStewardDomainsForUser(ctx context.Context, userID string) ([]string, error) {
	// In a real app, this would query the `semantic_domain_steward` table.
	if userID == "data_steward" {
		return []string{"finance", "sales"}, nil
	}
	if userID == "admin" {
		// Admins might be stewards of all domains
		return []string{"finance", "sales", "marketing", "product"}, nil
	}
	return []string{}, nil
}

// ListAllRoles retrieves a list of all defined roles. Mock implementation.
func (s *CollaborationService) ListAllRoles(ctx context.Context) ([]string, error) {
	// In a real app, this would query a user/roles table.
	return []string{"analyst", "finance_team", "sales_manager", "admin"}, nil
}

// GetEffectiveClaimsForUser resolves all claims for a user, combining direct and role-based grants.
// Mock implementation.
func (s *CollaborationService) GetEffectiveClaimsForUser(ctx context.Context, userID string) ([]models.SemanticModelClaim, error) {
	// In a real app, this would involve multiple DB queries. We'll mock the logic.

	// 1. Mock user roles.
	userRoles := []string{}
	if userID == "patrick" {
		userRoles = append(userRoles, "analyst")
	}
	if userID == "ceo" {
		userRoles = append(userRoles, "finance_team")
	}

	// 2. Mock direct claims for the user.
	expiresIn5Days := time.Now().Add(5 * 24 * time.Hour)
	expiredYesterday := time.Now().Add(-1 * 24 * time.Hour)
	renewalRequested := true
	directClaims := []models.SemanticModelClaim{
		// Example: A special, direct grant for a specific model, with scope.
		// This implements Object-Level Security (OLS).
		// Patrick can see the model, but only specific dimensions and metrics.
		{
			UserID:     "patrick",
			TenantID:   uuid.New(),
			ModelID:    uuid.MustParse("d1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b"), // Corresponds to 'orders_view'
			Permission: "read",
			Scope:      []string{"dimension:order_id", "dimension:order_date", "metric:total_revenue"},
			GrantedBy:  "direct_grant_ols",
			Status:     "active",
		},
		{
			UserID:           "patrick",
			TenantID:         uuid.New(),
			ModelID:          uuid.MustParse("e1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b"), // Another model
			Permission:       "read",
			GrantedBy:        "direct_grant_expiring",
			ExpiresAt:        &expiresIn5Days,
			RenewalRequested: &renewalRequested, // User has already requested renewal
			Status:           "renewal_requested",
		},
		{
			UserID:     "patrick",
			TenantID:   uuid.New(),
			ModelID:    uuid.MustParse("f1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b"), // Expired model
			Permission: "read",
			GrantedBy:  "direct_grant_expired",
			ExpiresAt:  &expiredYesterday,
			Status:     "expired",
		},
	}

	// 3. Get all role-to-model mappings.
	allRoleClaims, _ := s.ListRoleClaims(ctx)

	// 4. Combine direct and role-based claims.
	effectiveClaims := make(map[string]models.SemanticModelClaim)

	// Process direct claims first.
	for _, claim := range directClaims {
		if claim.UserID == userID {
			key := fmt.Sprintf("%s-%s", claim.ModelID, claim.Permission)
			effectiveClaims[key] = claim
		}
	}

	// Process role-based claims, adding them only if a direct claim doesn't already exist.
	for _, roleClaim := range allRoleClaims {
		for _, userRole := range userRoles {
			if roleClaim.Role == userRole {
				for _, perm := range roleClaim.Permissions {
					key := fmt.Sprintf("%s-%s", roleClaim.ModelID, perm)
					if _, exists := effectiveClaims[key]; !exists {
						effectiveClaims[key] = models.SemanticModelClaim{UserID: userID, TenantID: uuid.New(), ModelID: roleClaim.ModelID, Permission: perm, GrantedBy: "role:" + roleClaim.Role, Status: "active"}
					}
				}
			}
		}
	}

	var result []models.SemanticModelClaim
	for _, claim := range effectiveClaims {
		result = append(result, claim)
	}

	return result, nil
}

// RequestClaimRenewal requests renewal for an expiring claim. Mock implementation.
func (s *CollaborationService) RequestClaimRenewal(ctx context.Context, claimID uuid.UUID, userID, reason string) error {
	// In a real app:
	// 1. Find the claim by ID.
	// 2. Check if it's eligible for renewal.
	// 3. Create a record in `semantic_claim_renewal_request`.
	// 4. Update the `renewal_requested` flag on the claim itself.
	logging.GetLogger().Sugar().Infof("User %s requested renewal for claim %s with reason: %s", userID, claimID, reason)
	return nil
}

// GetClaimLifecycleSnapshot retrieves a summary for the claim lifecycle dashboard.
func (s *CollaborationService) GetClaimLifecycleSnapshot(ctx context.Context) (*models.ClaimLifecycleSnapshot, error) {
	// In a real app, this would be a series of COUNT(*) queries on the semantic_model_claim table
	// grouped by status, and a SELECT from claim_lifecycle_event.

	// Mock recent events
	notes1 := "Approved for Q4 reporting."
	recentEvents := []models.ClaimLifecycleEvent{
		{
			ID:          uuid.New(),
			ClaimID:     uuid.New(),
			EventType:   "granted",
			ActorUserID: "admin",
			Timestamp:   time.Now().Add(-2 * time.Hour),
			Notes:       &notes1,
		},
		{
			ID:          uuid.New(),
			ClaimID:     uuid.New(),
			EventType:   "renewal_requested",
			ActorUserID: "patrick",
			Timestamp:   time.Now().Add(-8 * time.Hour),
		},
		{
			ID:          uuid.New(),
			ClaimID:     uuid.New(),
			EventType:   "expired",
			ActorUserID: "system",
			Timestamp:   time.Now().Add(-24 * time.Hour),
		},
	}

	snapshot := &models.ClaimLifecycleSnapshot{
		ActiveCount:           125,
		ExpiringSoonCount:     12,
		RenewalRequestedCount: 3,
		ExpiredCount:          45,
		RevokedCount:          22,
		RecentEvents:          recentEvents,
	}

	return snapshot, nil
}

// ListAccessPolicies retrieves all access control policies sorted by name for stable UI rendering.
func (s *CollaborationService) ListAccessPolicies(ctx context.Context) ([]models.AccessControlPolicy, error) {
	if s.accessPolicyRepo != nil {
		policies, err := s.accessPolicyRepo.List(ctx)
		if err != nil {
			return nil, err
		}
		return sortPoliciesForUI(policies), nil
	}

	s.accessPolicyMu.RLock()
	defer s.accessPolicyMu.RUnlock()

	policies := make([]models.AccessControlPolicy, 0, len(s.accessPolicies))
	for _, policy := range s.accessPolicies {
		policies = append(policies, policy)
	}

	return sortPoliciesForUI(policies), nil
}

// GetAccessPolicyByID retrieves a policy by its UUID.
func (s *CollaborationService) GetAccessPolicyByID(ctx context.Context, id uuid.UUID) (*models.AccessControlPolicy, error) {
	if s.accessPolicyRepo != nil {
		return s.accessPolicyRepo.GetByID(ctx, id)
	}

	s.accessPolicyMu.RLock()
	defer s.accessPolicyMu.RUnlock()

	policy, ok := s.accessPolicies[id]
	if !ok {
		return nil, fmt.Errorf("access policy %s not found", id)
	}
	clone := policy
	return &clone, nil
}

// GetAccessPolicyBySlug retrieves a policy by its human-readable policy_id field.
func (s *CollaborationService) GetAccessPolicyBySlug(ctx context.Context, slug string) (*models.AccessControlPolicy, error) {
	if s.accessPolicyRepo != nil {
		return s.accessPolicyRepo.GetByPolicyID(ctx, slug)
	}

	s.accessPolicyMu.RLock()
	defer s.accessPolicyMu.RUnlock()

	for _, policy := range s.accessPolicies {
		if strings.EqualFold(policy.PolicyID, slug) {
			clone := policy
			return &clone, nil
		}
	}
	return nil, fmt.Errorf("access policy %s not found", slug)
}

// CreateAccessPolicy stores a new access control policy.
func (s *CollaborationService) CreateAccessPolicy(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error) {
	if err := validateAccessPolicyPayload(policy); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	policy.CreatedAt = now
	policy.UpdatedAt = now
	normalizeAccessPolicyDefaults(policy)

	if s.accessPolicyRepo != nil {
		created, err := s.accessPolicyRepo.Create(ctx, policy)
		if err != nil {
			return nil, err
		}
		s.accessPolicyMu.Lock()
		s.accessPolicies[created.ID] = *created
		s.accessPolicyMu.Unlock()
		return created, nil
	}

	s.accessPolicyMu.Lock()
	defer s.accessPolicyMu.Unlock()

	for _, existing := range s.accessPolicies {
		if strings.EqualFold(existing.PolicyID, policy.PolicyID) {
			return nil, fmt.Errorf("policy with id %s already exists", policy.PolicyID)
		}
	}

	clone := *policy
	s.accessPolicies[policy.ID] = clone
	return &clone, nil
}

// UpdateAccessPolicy updates an existing access control policy.
func (s *CollaborationService) UpdateAccessPolicy(ctx context.Context, policy *models.AccessControlPolicy) (*models.AccessControlPolicy, error) {
	if policy == nil {
		return nil, errors.New("policy payload is required")
	}
	if policy.ID == uuid.Nil {
		return nil, errors.New("policy id is required for update")
	}
	if err := validateAccessPolicyPayload(policy); err != nil {
		return nil, err
	}
	normalizeAccessPolicyDefaults(policy)

	if s.accessPolicyRepo != nil {
		existing, err := s.accessPolicyRepo.GetByID(ctx, policy.ID)
		if err != nil {
			return nil, err
		}
		policy.CreatedAt = existing.CreatedAt
		policy.UpdatedAt = time.Now().UTC()
		updated, err := s.accessPolicyRepo.Update(ctx, policy)
		if err != nil {
			return nil, err
		}
		s.accessPolicyMu.Lock()
		s.accessPolicies[updated.ID] = *updated
		s.accessPolicyMu.Unlock()
		return updated, nil
	}

	s.accessPolicyMu.Lock()
	defer s.accessPolicyMu.Unlock()

	existing, ok := s.accessPolicies[policy.ID]
	if !ok {
		return nil, fmt.Errorf("policy %s not found", policy.ID)
	}
	for id, conflict := range s.accessPolicies {
		if id == policy.ID {
			continue
		}
		if strings.EqualFold(conflict.PolicyID, policy.PolicyID) {
			return nil, fmt.Errorf("policy with id %s already exists", policy.PolicyID)
		}
	}

	policy.CreatedAt = existing.CreatedAt
	policy.UpdatedAt = time.Now().UTC()

	clone := *policy
	s.accessPolicies[policy.ID] = clone
	return &clone, nil
}

// DeleteAccessPolicy removes a policy by UUID.
func (s *CollaborationService) DeleteAccessPolicy(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return errors.New("policy id is required for delete")
	}

	if s.accessPolicyRepo != nil {
		if err := s.accessPolicyRepo.Delete(ctx, id); err != nil {
			return err
		}
		s.accessPolicyMu.Lock()
		delete(s.accessPolicies, id)
		s.accessPolicyMu.Unlock()
		return nil
	}

	s.accessPolicyMu.Lock()
	defer s.accessPolicyMu.Unlock()

	if _, ok := s.accessPolicies[id]; !ok {
		return fmt.Errorf("policy %s not found", id)
	}

	delete(s.accessPolicies, id)
	return nil
}

// SimulatePolicyChange calculates the impact of a proposed policy change. Mock implementation.
func (s *CollaborationService) SimulatePolicyChange(ctx context.Context, policy models.AccessControlPolicy, actorID string) (*models.PolicySimulationResult, error) {
	// In a real app, this would be a complex process:
	// 1. Find all claims that would be affected by the policy change.
	// 2. Calculate the new state of those claims.
	// 3. Diff the old and new states to find added/removed/modified claims.
	// 4. Aggregate the affected users and assets.

	// Mocking a result for a policy change that adds a few claims.
	affectedClaimsJSON, _ := json.Marshal(map[string]int{"added": 5, "removed": 2, "modified": 10})

	return &models.PolicySimulationResult{
		ID:             uuid.New(),
		PolicyID:       policy.PolicyID,
		SimulatedBy:    actorID,
		SimulatedAt:    time.Now(),
		AffectedClaims: affectedClaimsJSON,
		AffectedUsers:  []string{"user1", "user2", "user3"},
		AffectedAssets: []string{uuid.NewString(), uuid.NewString()},
		RiskFlags:      []string{"Grants access to PII data"},
	}, nil
}

// GetClaimAwareLineage generates a lineage graph for an asset and decorates it with a user's access claims.
func (s *CollaborationService) GetClaimAwareLineage(ctx context.Context, assetID, userID string) (*models.ClaimAwareLineageGraphData, error) {
	// 1. Mock a base lineage graph for the asset.
	// In a real system, this would be fetched from a lineage registry.
	baseGraph := &models.LineageGraphData{
		Nodes: []models.LineageNode{
			{ID: "metric:avg_order_value", Type: "metric", Label: "Avg Order Value"},
			{ID: "metric:net_margin", Type: "metric", Label: "Net Margin", Data: map[string]any{"certified": true}},
			{ID: "view:orders_view", Type: "view", Label: "orders_view", Data: map[string]any{"certified": true}},
			{ID: "table:orders", Type: "table", Label: "orders"},
			{ID: "table:line_items", Type: "table", Label: "line_items"},
			{ID: "dashboard:sales_kpis", Type: "dashboard", Label: "Sales KPIs"},
		},
		Edges: []models.LineageEdge{
			{Source: "view:orders_view", Target: "metric:avg_order_value"},
			{Source: "view:orders_view", Target: "metric:net_margin"},
			{Source: "table:orders", Target: "view:orders_view"},
			{Source: "table:line_items", Target: "view:orders_view"},
			{Source: "metric:avg_order_value", Target: "dashboard:sales_kpis"},
		},
	}

	// 2. Get the user's effective claims.
	// This part is simplified for the mock. A real implementation would be more robust.

	// 3. Decorate nodes with visibility status based on the user.
	decoratedNodes := []models.ClaimAwareLineageNode{}
	for _, node := range baseGraph.Nodes {
		visibility := "none"
		reason := "No access claim found."

		// Mock logic: Patrick can see some things but not the certified finance metric.
		if userID == "patrick" {
			if node.ID == "metric:avg_order_value" || node.ID == "view:orders_view" || node.ID == "dashboard:sales_kpis" || node.ID == "table:orders" || node.ID == "table:line_items" {
				visibility = "full"
				reason = "Direct or role-based access granted."
			}
			if node.ID == "metric:net_margin" {
				visibility = "none"
				reason = "Access to this certified metric requires a specific 'finance' role."
			}
		} else {
			// Other users have full access for this mock.
			visibility = "full"
			reason = "Default access."
		}

		decoratedNodes = append(decoratedNodes, models.ClaimAwareLineageNode{
			LineageNode: node,
			Visibility:  visibility,
			Reason:      reason,
		})
	}

	return &models.ClaimAwareLineageGraphData{
		Nodes: decoratedNodes,
		Edges: baseGraph.Edges,
	}, nil
}

// ListNotificationsForUser retrieves recent notifications for a user. Mock implementation.
func (s *CollaborationService) ListNotificationsForUser(ctx context.Context, userID string) ([]models.SemanticNotification, error) {
	// In a real app, this would query the semantic_notification table for unread notifications for the user.
	return []models.SemanticNotification{
		{
			ID:            uuid.New(),
			EventType:     "certification_updated",
			AssetID:       uuid.New(),
			AssetType:     "metric",
			Message:       "Metric 'avg_order_value' is now certified.",
			RoutingRuleID: func() *string { s := "certification_change_alert"; return &s }(),
			TriggeredBy:   "admin",
			Status:        "sent",
			Timestamp:     time.Now().Add(-1 * time.Hour),
			IsRead:        false,
		},
		{
			ID:            uuid.New(),
			EventType:     "claim_granted",
			AssetID:       uuid.New(),
			AssetType:     "view",
			Message:       "You have been granted 'read' access to the 'orders_view' semantic model.",
			RoutingRuleID: func() *string { s := "claim_grant_alert"; return &s }(),
			TriggeredBy:   "data_steward",
			Status:        "sent",
			Timestamp:     time.Now().Add(-5 * time.Hour),
			IsRead:        false,
		},
		{
			ID:          uuid.New(),
			EventType:   "lineage_changed",
			AssetID:     uuid.New(),
			AssetType:   "dashboard",
			Message:     "Dashboard 'Sales KPIs' now depends on an uncertified view.",
			TriggeredBy: "system",
			Status:      "sent",
			Timestamp:   time.Now().Add(-2 * 24 * time.Hour),
			IsRead:      true,
		},
	}, nil
}

// MarkNotificationAsRead marks a single notification as read. Mock implementation.
func (s *CollaborationService) MarkNotificationAsRead(ctx context.Context, notificationID uuid.UUID, userID string) error {
	// In a real app, this would be an UPDATE query on semantic_notification.
	logging.GetLogger().Sugar().Infof("User %s marked notification %s as read.", userID, notificationID)
	return nil
}

// ListAccessAuditLogs retrieves the audit trail for governance actions. Mock implementation.
func (s *CollaborationService) ListAccessAuditLogs(ctx context.Context) ([]models.AccessControlAuditLog, error) {
	// Mock data
	details1, _ := json.Marshal(map[string]string{"model_id": uuid.New().String(), "permission": "read", "reason": "Need to explore churn metrics for Q3 analysis."})
	details2, _ := json.Marshal(map[string]string{"claim_id": uuid.New().String()})
	return []models.AccessControlAuditLog{
		{ID: uuid.New(), Timestamp: time.Now().Add(-1 * time.Hour), ActorUserID: "current_reviewer", Action: "request_approved", TargetType: "user", TargetID: "patrick", Details: details1},
		{ID: uuid.New(), Timestamp: time.Now().Add(-3 * time.Hour), ActorUserID: "admin", Action: "claim_revoked", TargetType: "user", TargetID: "old_employee", Details: details2},
		{ID: uuid.New(), Timestamp: time.Now().Add(-24 * time.Hour), ActorUserID: "admin", Action: "role_updated", TargetType: "role", TargetID: "analyst", Details: json.RawMessage(`{"model_id": "...", "permissions": ["read"]}`)},
	}, nil
}

// SimulateClaims calculates the impact of proposed claim changes.
// Mock implementation.
func (s *CollaborationService) SimulateClaims(ctx context.Context, req models.ClaimSimulationRequest, simulatedBy string) (*models.ClaimSimulationResult, error) {
	// In a real app, this would be a complex process:
	// 1. Get current effective claims for the user/role.
	// 2. Calculate new effective claims with proposed changes.
	// 3. Diff the two claim sets.
	// 4. Check metadata (e.g., certification status) for affected models.

	// Mocking the result for "analyst" role gaining "read" on a new model.
	affectedModelID := uuid.New()
	riskFlags := []string{}

	// Check for risky changes
	for _, claim := range req.ProposedClaims {
		// Let's pretend one of the models is certified and write access is proposed
		if claim.Permission == "write" && claim.ModelID.String()[0] < '8' { // Mock condition for certified model
			riskFlags = append(riskFlags, fmt.Sprintf("Proposed 'write' access to certified model %s", claim.ModelID.String()))
		}
	}

	proposedClaimsJSON, _ := json.Marshal(req.ProposedClaims)

	result := &models.ClaimSimulationResult{
		ID:             uuid.New(),
		SimulatedFor:   req.SimulateFor,
		SimulatedBy:    simulatedBy,
		ProposedClaims: proposedClaimsJSON,
		AffectedModels: []models.AffectedModel{
			{
				ModelID:   affectedModelID,
				ModelName: "orders_view_v2",
				Change:    "gained_read",
				Certified: true,
			},
		},
		RiskFlags:   riskFlags,
		SimulatedAt: time.Now(),
	}

	// In a real app, this result would be saved to the `claim_simulation_result` table.

	return result, nil
}

// ListClaimSimulations retrieves recent claim simulation results. Mock implementation.
func (s *CollaborationService) ListClaimSimulations(ctx context.Context) ([]models.ClaimSimulationResult, error) {
	// In a real app, this would SELECT from the `claim_simulation_result` table.
	// Mock data:
	proposedClaims1, _ := json.Marshal([]models.ProposedClaim{
		{ModelID: uuid.New(), Permission: "read"},
	})
	proposedClaims2, _ := json.Marshal([]models.ProposedClaim{
		{ModelID: uuid.New(), Permission: "write"},
	})

	return []models.ClaimSimulationResult{
		{
			ID:             uuid.New(),
			SimulatedFor:   "analyst_role",
			SimulatedBy:    "admin_user",
			ProposedClaims: proposedClaims1,
			AffectedModels: []models.AffectedModel{
				{ModelID: uuid.New(), ModelName: "orders_view_v2", Change: "gained_read", Certified: true},
			},
			RiskFlags:   []string{},
			SimulatedAt: time.Now().Add(-2 * time.Hour),
		},
		{
			ID:             uuid.New(),
			SimulatedFor:   "finance_user",
			SimulatedBy:    "admin_user",
			ProposedClaims: proposedClaims2,
			AffectedModels: []models.AffectedModel{
				{ModelID: uuid.New(), ModelName: "customer_pii_data", Change: "gained_write", Certified: true},
			},
			RiskFlags:   []string{"Proposed 'write' access to certified model customer_pii_data"},
			SimulatedAt: time.Now().Add(-5 * time.Hour),
		},
	}, nil
}

// LogAccessDeniedAttempt logs when a user tries to access an asset they don't have claims for.
func (s *CollaborationService) LogAccessDeniedAttempt(ctx context.Context, userID, assetType, assetID, reason string) error {
	// In a real app, this would insert into the audit_log table.
	details, _ := json.Marshal(map[string]string{
		"asset_id": assetID,
		"reason":   reason, // e.g., "User lacks 'read' claim for metrics on this model"
	})
	logEntry := models.AccessControlAuditLog{
		ID:          uuid.New(),
		Timestamp:   time.Now(),
		ActorUserID: userID,
		Action:      "access_denied",
		TargetType:  assetType, // e.g., 'semantic_model_scope'
		TargetID:    assetID,
		Details:     details,
	}
	logging.GetLogger().Sugar().Infof("Logged access denied attempt: %+v", logEntry)
	return nil
}

// --- Notification Routing Engine ---

// ListNotificationRules retrieves all notification routing rules. Mock implementation.
func (s *CollaborationService) ListNotificationRules(ctx context.Context) ([]models.NotificationRoutingRule, error) {
	// Mock data
	logic1, _ := json.Marshal(models.RoutingLogic{
		Notify: []string{"asset_owner", "domain_steward", "downstream_users"},
		SuppressIf: &models.SuppressionCondition{
			AssetCertified: func() *bool { b := false; return &b }(),
			RiskScoreLte:   func() *int { i := 20; return &i }(),
		},
		EscalateIf: &models.EscalationCondition{
			AssetCertified: func() *bool { b := true; return &b }(),
			ChangeType:     []string{"revocation"},
			RiskScoreGte:   func() *int { i := 80; return &i }(),
		},
		EscalateTo: []string{"governance_reviewers"},
	})
	logic2, _ := json.Marshal(models.RoutingLogic{
		Notify:  []string{"user"},
		Exclude: []string{"requestor"},
		SuppressIf: &models.SuppressionCondition{
			RiskScoreLte: func() *int { i := 10; return &i }(),
		},
	})

	return []models.NotificationRoutingRule{
		{
			ID:           uuid.New(),
			RuleID:       "certification_change_alert",
			Trigger:      "certification_updated",
			Scope:        "asset",
			AssetType:    "metric",
			RoutingLogic: logic1,
			UpdatedAt:    time.Now().Add(-5 * 24 * time.Hour),
			UpdatedBy:    "admin",
		},
		{
			ID:           uuid.New(),
			RuleID:       "claim_grant_alert",
			Trigger:      "claim_granted",
			Scope:        "asset",
			AssetType:    "view",
			RoutingLogic: logic2,
			UpdatedAt:    time.Now().Add(-10 * 24 * time.Hour),
			UpdatedBy:    "data_steward",
		},
	}, nil
}

// UpdateNotificationRule creates or updates a routing rule. Mock implementation.
func (s *CollaborationService) UpdateNotificationRule(ctx context.Context, rule models.NotificationRoutingRule, actorID string) (*models.NotificationRoutingRule, error) {
	if rule.ID == uuid.Nil {
		rule.ID = uuid.New()
		rule.CreatedAt = time.Now()
	}
	rule.UpdatedAt = time.Now()
	rule.UpdatedBy = actorID
	logging.GetLogger().Sugar().Infof("Actor %s updated rule %s", actorID, rule.RuleID)
	return &rule, nil
}

// PreviewRecipients simulates who would be notified for a given asset. Mock implementation.
func (s *CollaborationService) PreviewRecipients(ctx context.Context, assetID uuid.UUID) (map[string][]string, error) {
	// In a real app, this would run the resolution logic for the asset.
	return map[string][]string{
		"certification_updated": {"asset_owner_1", "domain_steward_finance", "downstream_user_1", "downstream_user_2"},
		"claim_granted":         {"user_patrick"},
		"lineage_changed":       {"downstream_user_1", "downstream_user_2"},
	}, nil
}

// --- Alert Suppression & Escalation ---

func (s *CollaborationService) calculateRiskScore(event models.SemanticChangeEvent) int {
	score := 0
	switch event.AssetSensitivity {
	case "medium":
		score += 25
	case "high":
		score += 50
	}
	switch event.ChangeType {
	case "certification_revoked":
		score += 40
	case "claim_grant":
		score += 15
	case "metric_updated":
		score += 5
	}
	// Add more complex logic here...
	return score
}

// EvaluateAlert classifies and routes a change event. Mock implementation.
func (s *CollaborationService) EvaluateAlert(ctx context.Context, event models.SemanticChangeEvent) (*models.SemanticNotification, error) {
	riskScore := s.calculateRiskScore(event)
	// Find a matching rule (mocking this part)
	rules, _ := s.ListNotificationRules(ctx)
	var matchedRule *models.NotificationRoutingRule
	for i, rule := range rules {
		if rule.Trigger == event.ChangeType || (rule.Trigger == "certification_updated" && event.ChangeType == "certification_revoked") {
			matchedRule = &rules[i]
			break
		}
	}

	notification := &models.SemanticNotification{
		ID:          uuid.New(),
		EventType:   event.ChangeType,
		AssetID:     event.AssetID,
		AssetType:   event.AssetType,
		Message:     fmt.Sprintf("Change '%s' on asset %s triggered by %s. Risk score: %d", event.ChangeType, event.AssetID, event.UserID, riskScore),
		TriggeredBy: event.UserID,
		Timestamp:   time.Now(),
	}

	if matchedRule != nil {
		notification.RoutingRuleID = &matchedRule.RuleID
		var logic models.RoutingLogic
		_ = json.Unmarshal(matchedRule.RoutingLogic, &logic)

		// 1. Check for suppression
		if logic.SuppressIf != nil && logic.SuppressIf.RiskScoreLte != nil && riskScore <= *logic.SuppressIf.RiskScoreLte {
			notification.Status = "suppressed"
			trace, _ := json.Marshal(map[string]interface{}{"reason": "Suppressed due to low risk score", "risk_score": riskScore, "rule_id": matchedRule.RuleID})
			notification.RoutingTrace = trace
			logging.GetLogger().Sugar().Infof("Alert suppressed: %+v", notification)
			return notification, nil
		}

		// 2. Check for escalation
		if logic.EscalateIf != nil && logic.EscalateIf.RiskScoreGte != nil && riskScore >= *logic.EscalateIf.RiskScoreGte {
			notification.Status = "escalated"
			trace, _ := json.Marshal(map[string]interface{}{"reason": "Escalated due to high risk score", "risk_score": riskScore, "rule_id": matchedRule.RuleID, "escalated_to": logic.EscalateTo})
			notification.RoutingTrace = trace
			logging.GetLogger().Sugar().Infof("Alert escalated: %+v", notification)
			return notification, nil
		}
	}

	// 3. Default to standard notification
	notification.Status = "sent"
	trace, _ := json.Marshal(map[string]interface{}{"reason": "Standard notification", "risk_score": riskScore})
	notification.RoutingTrace = trace
	logging.GetLogger().Sugar().Infof("Alert sent: %+v", notification)
	return notification, nil
}

// ListAlertsByStatus retrieves alerts with a given status. Mock implementation.
func (s *CollaborationService) ListAlertsByStatus(ctx context.Context, status string) ([]models.SemanticNotification, error) {
	// In a real app, this would be a SELECT ... WHERE status = ?
	// For mock, we'll just create some.
	if status == "suppressed" {
		return []models.SemanticNotification{
			{ID: uuid.New(), EventType: "metric_updated", Message: "Minor metric edit on non-certified asset.", Status: "suppressed", Timestamp: time.Now().Add(-2 * time.Hour)},
		}, nil
	}
	if status == "escalated" {
		return []models.SemanticNotification{
			{ID: uuid.New(), EventType: "certification_revoked", Message: "Certification revoked on critical 'revenue' metric.", Status: "escalated", Timestamp: time.Now().Add(-8 * time.Hour)},
		}, nil
	}
	return []models.SemanticNotification{}, nil
}

// OverrideAlertStatus changes the status of an alert. Mock implementation.
func (s *CollaborationService) OverrideAlertStatus(ctx context.Context, alertID uuid.UUID, newStatus string, actorID string) error {
	logging.GetLogger().Sugar().Infof("Actor %s overrode alert %s to status %s", actorID, alertID, newStatus)
	return nil
}

// --- Advanced Governance Features ---

// ListClaimSuggestions retrieves usage-aware claim suggestions. Mock implementation.
func (s *CollaborationService) ListClaimSuggestions(ctx context.Context, reviewerID string) ([]models.ClaimSuggestion, error) {
	// In a real app, this would query a suggestions table populated by a background job.
	evidence, _ := json.Marshal(map[string]interface{}{"query_count": 12, "last_queried": time.Now().Add(-24 * time.Hour)})
	return []models.ClaimSuggestion{
		{
			ID:                  uuid.New(),
			UserID:              "patrick",
			ModelID:             uuid.MustParse("d1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b"),
			SuggestedPermission: "read",
			Reason:              "High query frequency without a direct claim.",
			Evidence:            evidence,
			Status:              "new",
			CreatedAt:           time.Now().Add(-2 * time.Hour),
		},
	}, nil
}

// ListClaimBundles retrieves all claim bundles. Mock implementation.
func (s *CollaborationService) ListClaimBundles(ctx context.Context) ([]models.ClaimBundle, error) {
	return []models.ClaimBundle{
		{
			ID:          uuid.New(),
			Name:        "Marketing Analyst Bundle",
			Description: "Read access to core marketing and web analytics models.",
			CreatedBy:   "admin",
			UpdatedAt:   time.Now().Add(-5 * 24 * time.Hour),
		},
		{
			ID:          uuid.New(),
			Name:        "Finance Read-Only Bundle",
			Description: "Read access to all certified financial models.",
			CreatedBy:   "admin",
			UpdatedAt:   time.Now().Add(-10 * 24 * time.Hour),
		},
	}, nil
}

// DetectClaimDrift finds claims that haven't been used recently. Mock implementation.
func (s *CollaborationService) DetectClaimDrift(ctx context.Context) ([]models.SemanticModelClaim, error) {
	// In a real app, this would be a query like:
	// SELECT * FROM semantic_model_claim WHERE status = 'active' AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '60 days')
	lastUsed := time.Now().Add(-90 * 24 * time.Hour)
	return []models.SemanticModelClaim{
		{
			ID:         uuid.New(),
			UserID:     "old_employee",
			ModelID:    uuid.New(),
			Permission: "read",
			GrantedAt:  time.Now().Add(-120 * 24 * time.Hour),
			LastUsedAt: &lastUsed,
			Status:     "active",
		},
	}, nil
}

// GetGovernanceHeatmap generates a heatmap of governance metrics by domain. Mock implementation.
func (s *CollaborationService) GetGovernanceHeatmap(ctx context.Context) ([]models.GovernanceHeatmapDataPoint, error) {
	// In a real app, this would be a complex aggregation query across multiple tables.
	return []models.GovernanceHeatmapDataPoint{
		{Domain: "Finance", CertifiedModelPercent: 95.5, ClaimDensity: 150, RiskyClaimCount: 2, UnresolvedRequestCount: 1, ClaimDriftCount: 5},
		{Domain: "Sales", CertifiedModelPercent: 70.0, ClaimDensity: 250, RiskyClaimCount: 8, UnresolvedRequestCount: 5, ClaimDriftCount: 25},
		{Domain: "Marketing", CertifiedModelPercent: 45.0, ClaimDensity: 80, RiskyClaimCount: 1, UnresolvedRequestCount: 0, ClaimDriftCount: 15},
	}, nil
}

// DetectClaimConflicts finds overlapping or contradictory claims for a user. Mock implementation.
func (s *CollaborationService) DetectClaimConflicts(ctx context.Context, userID string) ([]models.ClaimConflict, error) {
	// In a real app, this would be a complex query or a background job.
	// We'll mock a scenario where a user has a 'read' claim from a role and a 'write' claim granted manually.
	if userID != "patrick" {
		return []models.ClaimConflict{}, nil
	}

	modelID := uuid.MustParse("d1b6a5e0-9a9a-4b1a-8b0a-1b1b1b1b1b1b")
	details, _ := json.Marshal(map[string]interface{}{
		"description": "User has 'read' access from 'analyst' role but was also granted 'write' access manually. This could be unintentional.",
		"conflicting_claims": []map[string]string{
			{"permission": "read", "source": "role:analyst"},
			{"permission": "write", "source": "direct_grant"},
		},
	})

	return []models.ClaimConflict{
		{
			ID:           uuid.New(),
			UserID:       "patrick",
			ModelID:      modelID,
			ConflictType: "contradiction",
			Details:      details,
			DetectedAt:   time.Now().Add(-1 * time.Hour),
			Status:       "new",
		},
	}, nil
}

// ResolveClaimConflict marks a conflict as resolved. Mock implementation.
func (s *CollaborationService) ResolveClaimConflict(ctx context.Context, conflictID uuid.UUID, action, actorID string) error {
	logging.GetLogger().Sugar().Infof("Actor %s resolved conflict %s with action: %s", actorID, conflictID, action)
	return nil
}
