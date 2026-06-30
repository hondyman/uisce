package security

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProfileService implements the core security profile repository and enrichment.
type ProfileService struct {
	db *sqlx.DB
}

// NewProfileService creates a new ProfileService.
func NewProfileService(db *sql.DB) *ProfileService {
	return &ProfileService{
		db: sqlx.NewDb(db, "postgres"),
	}
}

// FetchEffectiveProfile compiles the active security traits for an operational context.
// It resolves the global blueprint (tenant_id IS NULL) and overlays any tenant-specific customization.
func (s *ProfileService) FetchEffectiveProfile(ctx context.Context, tenantID uuid.UUID, profileKey string) (*ResolvedProfile, error) {
	query := `
		SELECT profile_id, tenant_id, parent_profile_id 
		FROM security.security_profiles
		WHERE profile_key = $1 AND (tenant_id IS NULL OR tenant_id = $2)
		ORDER BY tenant_id ASC NULLS FIRST;
	`

	rows, err := s.db.QueryContext(ctx, query, profileKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile lookup hierarchy: %w", err)
	}
	defer rows.Close()

	profile := &ResolvedProfile{
		ProfileKey: profileKey,
		Attributes: make(map[string]interface{}),
	}

	var totalFound int
	for rows.Next() {
		var pID uuid.UUID
		var tID *uuid.UUID
		var parentID *uuid.UUID

		if err := rows.Scan(&pID, &tID, &parentID); err != nil {
			return nil, err
		}

		if tID != nil {
			profile.IsCustomized = true
		}
		totalFound++
	}

	if totalFound == 0 {
		return nil, fmt.Errorf("denied: security profile key '%s' does not exist in platform blueprint", profileKey)
	}

	return profile, nil
}

// EnrichSubjectAttributes maps raw group arrays from IdP to internal abstract profiles and clearance levels.
func (s *ProfileService) EnrichSubjectAttributes(ctx context.Context, tenantID uuid.UUID, userID string, idpGroups []string) (string, string, error) {
	if len(idpGroups) == 0 {
		return "standard_guest", "L1", nil
	}

	query := `
		SELECT functional_role, clearance_level 
		FROM security.identity_profile_mappings
		WHERE tenant_id = $1 AND idp_group_claim = ANY($2)
		LIMIT 1;
	`

	var role, clearance string
	err := s.db.QueryRowContext(ctx, query, tenantID, idpGroups).Scan(&role, &clearance)
	if err == sql.ErrNoRows {
		return "unassigned_operator", "L1", nil
	} else if err != nil {
		return "", "", fmt.Errorf("security infrastructure execution failure: %w", err)
	}

	return role, clearance, nil
}

// ResolveTenantAndRole resolves the concrete tenant UUID and functional role for an external federated user
// using their Keycloak Client ID (azp) and IdP Group claims.
func (s *ProfileService) ResolveTenantAndRole(ctx context.Context, clientID string, idpGroups []string) (string, string, error) {
	if clientID == "" {
		return "", "", fmt.Errorf("client origin (azp) is required")
	}
	if len(idpGroups) == 0 {
		return "", "", fmt.Errorf("missing group claims")
	}

	query := `
		SELECT tenant_id::text, functional_role 
		FROM security.identity_profile_mappings 
		WHERE keycloak_client_id = $1 
		AND idp_group_claim = ANY($2) 
		LIMIT 1;
	`

	var tenantID, role string
	err := s.db.QueryRowContext(ctx, query, clientID, idpGroups).Scan(&tenantID, &role)
	return tenantID, role, err
}

// GetTenantIDByUser looks up the user's bound tenant ID in the database.
func (s *ProfileService) GetTenantIDByUser(ctx context.Context, userID string, email string) (string, error) {
	var tenantID sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT tenant_id FROM users WHERE id = $1 OR email = $2", userID, email).Scan(&tenantID)
	if err != nil {
		return "", err
	}
	if !tenantID.Valid {
		return "", fmt.Errorf("tenant ID is null")
	}
	return tenantID.String, nil
}

// VerifyStaffAssignment queries security.staff_tenant_assignments to check for active operator leases.
func (s *ProfileService) VerifyStaffAssignment(ctx context.Context, email string, tenantID uuid.UUID) (string, error) {
	var ticketRef string
	query := `
		SELECT ticket_reference 
		FROM security.staff_tenant_assignments 
		WHERE operator_email = $1 
		  AND target_tenant_id = $2 
		  AND expires_at > NOW();
	`
	err := s.db.GetContext(ctx, &ticketRef, query, email, tenantID)
	return ticketRef, err
}

// --- Profiles CRUD ---

func (s *ProfileService) CreateProfile(ctx context.Context, p *SecurityProfile) (*SecurityProfile, error) {
	if p.ProfileID == uuid.Nil {
		p.ProfileID = uuid.New()
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()

	query := `
		INSERT INTO security.security_profiles (profile_id, tenant_id, profile_key, profile_name, parent_profile_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING profile_id, tenant_id, profile_key, profile_name, parent_profile_id, created_at, updated_at
	`
	var newProfile SecurityProfile
	err := s.db.QueryRowxContext(ctx, query, p.ProfileID, p.TenantID, p.ProfileKey, p.ProfileName, p.ParentProfileID, p.CreatedAt, p.UpdatedAt).StructScan(&newProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to create security profile: %w", err)
	}
	return &newProfile, nil
}

func (s *ProfileService) ListProfiles(ctx context.Context, tenantID uuid.UUID) ([]SecurityProfile, error) {
	query := `
		SELECT profile_id, tenant_id, profile_key, profile_name, parent_profile_id, created_at, updated_at
		FROM security.security_profiles
		WHERE tenant_id IS NULL OR tenant_id = $1
		ORDER BY created_at DESC
	`
	var profiles []SecurityProfile
	err := s.db.SelectContext(ctx, &profiles, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list security profiles: %w", err)
	}
	return profiles, nil
}

func (s *ProfileService) GetProfile(ctx context.Context, id uuid.UUID) (*SecurityProfile, error) {
	query := `
		SELECT profile_id, tenant_id, profile_key, profile_name, parent_profile_id, created_at, updated_at
		FROM security.security_profiles
		WHERE profile_id = $1
	`
	var profile SecurityProfile
	err := s.db.GetContext(ctx, &profile, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get security profile: %w", err)
	}
	return &profile, nil
}

func (s *ProfileService) UpdateProfile(ctx context.Context, p *SecurityProfile) error {
	p.UpdatedAt = time.Now()
	query := `
		UPDATE security.security_profiles
		SET profile_name = $1, parent_profile_id = $2, updated_at = $3
		WHERE profile_id = $4 AND (tenant_id = $5 OR tenant_id IS NULL)
	`
	res, err := s.db.ExecContext(ctx, query, p.ProfileName, p.ParentProfileID, p.UpdatedAt, p.ProfileID, p.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update security profile: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("profile not found or unauthorized to update")
	}
	return nil
}

func (s *ProfileService) DeleteProfile(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		DELETE FROM security.security_profiles
		WHERE profile_id = $1 AND tenant_id = $2
	`
	res, err := s.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete security profile: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("profile not found or unauthorized to delete")
	}
	return nil
}

// --- Identity Profile Mappings CRUD ---

func (s *ProfileService) CreateMapping(ctx context.Context, m *IdentityProfileMapping) (*IdentityProfileMapping, error) {
	if m.MappingID == uuid.Nil {
		m.MappingID = uuid.New()
	}
	m.CreatedAt = time.Now()

	query := `
		INSERT INTO security.identity_profile_mappings (mapping_id, tenant_id, idp_group_claim, functional_role, clearance_level, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING mapping_id, tenant_id, idp_group_claim, functional_role, clearance_level, created_at
	`
	var newMapping IdentityProfileMapping
	err := s.db.QueryRowxContext(ctx, query, m.MappingID, m.TenantID, m.IDPGroupClaim, m.FunctionalRole, m.ClearanceLevel, m.CreatedAt).StructScan(&newMapping)
	if err != nil {
		return nil, fmt.Errorf("failed to create mapping: %w", err)
	}
	return &newMapping, nil
}

func (s *ProfileService) ListMappings(ctx context.Context, tenantID uuid.UUID) ([]IdentityProfileMapping, error) {
	query := `
		SELECT mapping_id, tenant_id, idp_group_claim, functional_role, clearance_level, created_at
		FROM security.identity_profile_mappings
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`
	var mappings []IdentityProfileMapping
	err := s.db.SelectContext(ctx, &mappings, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list mappings: %w", err)
	}
	return mappings, nil
}

func (s *ProfileService) GetMapping(ctx context.Context, id uuid.UUID) (*IdentityProfileMapping, error) {
	query := `
		SELECT mapping_id, tenant_id, idp_group_claim, functional_role, clearance_level, created_at
		FROM security.identity_profile_mappings
		WHERE mapping_id = $1
	`
	var mapping IdentityProfileMapping
	err := s.db.GetContext(ctx, &mapping, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get mapping: %w", err)
	}
	return &mapping, nil
}

func (s *ProfileService) UpdateMapping(ctx context.Context, m *IdentityProfileMapping) error {
	query := `
		UPDATE security.identity_profile_mappings
		SET idp_group_claim = $1, functional_role = $2, clearance_level = $3
		WHERE mapping_id = $4 AND tenant_id = $5
	`
	res, err := s.db.ExecContext(ctx, query, m.IDPGroupClaim, m.FunctionalRole, m.ClearanceLevel, m.MappingID, m.TenantID)
	if err != nil {
		return fmt.Errorf("failed to update mapping: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("mapping not found or unauthorized to update")
	}
	return nil
}

func (s *ProfileService) DeleteMapping(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) error {
	query := `
		DELETE FROM security.identity_profile_mappings
		WHERE mapping_id = $1 AND tenant_id = $2
	`
	res, err := s.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete mapping: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("mapping not found or unauthorized to delete")
	}
	return nil
}

// FetchInheritedRoles recursively traverses the security.security_profiles parent relationships
// to compile a list of all functional roles/profile keys that the user's role inherits from.
func (s *ProfileService) FetchInheritedRoles(ctx context.Context, tenantID uuid.UUID, profileKey string) ([]string, error) {
	query := `
		WITH RECURSIVE profile_hierarchy AS (
			SELECT profile_id, profile_key, parent_profile_id, tenant_id
			FROM security.security_profiles
			WHERE profile_key = $1 AND (tenant_id IS NULL OR tenant_id = $2)
			
			UNION ALL
			
			SELECT p.profile_id, p.profile_key, p.parent_profile_id, p.tenant_id
			FROM security.security_profiles p
			INNER JOIN profile_hierarchy h ON p.profile_id = h.parent_profile_id
		)
		SELECT DISTINCT profile_key FROM profile_hierarchy;
	`

	var roles []string
	err := s.db.SelectContext(ctx, &roles, query, profileKey, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile inheritance hierarchy: %w", err)
	}

	return roles, nil
}

