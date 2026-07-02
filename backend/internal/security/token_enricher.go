package security

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

// AppSubjectContext represents the enriched subject attributes.
type AppSubjectContext struct {
	UserID         string
	TenantID       uuid.UUID
	FunctionalRole string
	ClearanceLevel string
}

// TokenEnricher enriches authentication tokens with internal abstract profile traits.
type TokenEnricher struct {
	db *sql.DB
}

// NewTokenEnricher creates a new TokenEnricher.
func NewTokenEnricher(db *sql.DB) *TokenEnricher {
	return &TokenEnricher{db: db}
}

// EnrichSubjectAttributes maps raw group arrays from the IdP to internal functional roles and clearance levels.
func (e *TokenEnricher) EnrichSubjectAttributes(ctx context.Context, tenantID uuid.UUID, userID string, idpGroups []string) (*AppSubjectContext, error) {
	subjectCtx := &AppSubjectContext{
		UserID:   userID,
		TenantID: tenantID,
	}

	if len(idpGroups) == 0 {
		subjectCtx.FunctionalRole = "standard_guest"
		subjectCtx.ClearanceLevel = "L1"
		return subjectCtx, nil
	}

	// Query mapping table to translate AD/Keycloak group claim to role/clearance
	query := `
		SELECT functional_role, clearance_level 
		FROM security.identity_profile_mappings
		WHERE tenant_id = $1 AND idp_group_id = ANY($2)
		LIMIT 1;
	`

	var functionalRole, clearanceLevel string
	err := e.db.QueryRowContext(ctx, query, tenantID, idpGroups).Scan(&functionalRole, &clearanceLevel)
	if err == sql.ErrNoRows {
		subjectCtx.FunctionalRole = "unassigned_operator"
		subjectCtx.ClearanceLevel = "L1"
		return subjectCtx, nil
	} else if err != nil {
		return nil, fmt.Errorf("security infrastructure execution failure: %w", err)
	}

	subjectCtx.FunctionalRole = functionalRole
	subjectCtx.ClearanceLevel = clearanceLevel
	return subjectCtx, nil
}

// ProfileRepository handles standard SQL access to effective profiles for resolution.
type ProfileRepository struct {
	db *sql.DB
}

// NewProfileRepository creates a new ProfileRepository.
func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// FetchEffectiveProfile compiles the active security traits for an operational context.
func (r *ProfileRepository) FetchEffectiveProfile(ctx context.Context, tenantID uuid.UUID, profileKey string) (*ResolvedProfile, error) {
	query := `
		SELECT profile_id, tenant_id, parent_profile_id 
		FROM security.security_profiles
		WHERE profile_key = $1 AND (tenant_id IS NULL OR tenant_id = $2)
		ORDER BY tenant_id ASC NULLS FIRST;
	`

	rows, err := r.db.QueryContext(ctx, query, profileKey, tenantID)
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
