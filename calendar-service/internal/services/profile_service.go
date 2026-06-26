package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// scheduleProfile represents a schedule profile with multi-calendar support
type ScheduleProfile struct {
	ID                 string          `json:"id"`
	TenantID           string          `json:"tenant_id"`
	ProfileName        string          `json:"profile_name"`
	Description        string          `json:"description,omitempty"`
	Calendars          []string        `json:"calendars"`           // Array of calendar IDs
	ConflictResolution string          `json:"conflict_resolution"` // union, intersection, priority
	Timezone           string          `json:"timezone"`
	Rules              json.RawMessage `json:"rules,omitempty"`
	Active             bool            `json:"active"`
	ValidFrom          time.Time       `json:"valid_from"`
	ValidTo            *time.Time      `json:"valid_to,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	CreatedBy          string          `json:"created_by,omitempty"`
	UpdatedBy          string          `json:"updated_by,omitempty"`
}

// ProfileServiceTenantAware defines tenant-scoped profile management operations
type ProfileServiceTenantAware interface {
	// Create creates a new profile for a tenant with bitemporal versioning
	Create(ctx context.Context, tenantID string, input CreateProfileInput) (*ScheduleProfile, error)

	// GetByID retrieves a profile by ID with tenant verification
	GetByID(ctx context.Context, tenantID, profileID string) (*ScheduleProfile, error)

	// ListActive lists all active profiles for a tenant
	ListActive(ctx context.Context, tenantID string, limit, offset int) ([]ScheduleProfile, error)

	// Update updates a profile with bitemporal versioning (creates new version)
	Update(ctx context.Context, tenantID, profileID string, input UpdateProfileInput) (*ScheduleProfile, error)

	// Delete soft-deletes a profile (sets active = false, valid_to = now)
	Delete(ctx context.Context, tenantID, profileID string, actorID string) error

	// ListVersions lists all versions (including historical) of a profile
	ListVersions(ctx context.Context, tenantID, profileID string) ([]ScheduleProfile, error)
}

// CreateProfileInput defines input for creating a profile
type CreateProfileInput struct {
	ProfileName        string          `json:"profile_name"`
	Description        string          `json:"description,omitempty"`
	Calendars          []string        `json:"calendars"`
	ConflictResolution string          `json:"conflict_resolution"`
	Timezone           string          `json:"timezone"`
	Rules              json.RawMessage `json:"rules,omitempty"`
	ActorID            string          `json:"actor_id"`
}

// UpdateProfileInput defines input for updating a profile
type UpdateProfileInput struct {
	ProfileName        *string          `json:"profile_name,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Calendars          *[]string        `json:"calendars,omitempty"`
	ConflictResolution *string          `json:"conflict_resolution,omitempty"`
	Timezone           *string          `json:"timezone,omitempty"`
	Rules              *json.RawMessage `json:"rules,omitempty"`
	Active             *bool            `json:"active,omitempty"`
	ActorID            string           `json:"actor_id"`
}

// ProfileServiceImpl implements ProfileServiceTenantAware
type ProfileServiceImpl struct {
	repo         *RepositoryAdapter // Reverted type
	auditService AuditService       // Changed type
	logger       *logrus.Entry
}

// NewProfileService creates a new profile service instance
func NewProfileService(repo *RepositoryAdapter, auditService AuditService, logger *logrus.Entry) ProfileServiceTenantAware {
	return &ProfileServiceImpl{
		repo:         repo,
		auditService: auditService,
		logger:       logger.WithField("service", "profile"),
	}
}

// Create creates a new schedule profile
func (s *ProfileServiceImpl) Create(ctx context.Context, tenantID string, input CreateProfileInput) (*ScheduleProfile, error) {
	now := time.Now().UTC()

	// Validate conflict resolution
	validResolutions := map[string]bool{"union": true, "intersection": true, "priority": true}
	if !validResolutions[input.ConflictResolution] {
		input.ConflictResolution = "union" // default
	}

	// Validate timezone
	if input.Timezone == "" {
		input.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return nil, fmt.Errorf("invalid timezone: %w", err)
	}

	// Validate at least one calendar
	if len(input.Calendars) == 0 {
		return nil, fmt.Errorf("at least one calendar is required")
	}

	// Validate profile name
	if input.ProfileName == "" {
		return nil, fmt.Errorf("profile_name is required")
	}

	profileID := uuid.New().String()

	profile := &ScheduleProfile{
		ID:                 profileID,
		TenantID:           tenantID,
		ProfileName:        input.ProfileName,
		Description:        input.Description,
		Calendars:          input.Calendars,
		ConflictResolution: input.ConflictResolution,
		Timezone:           input.Timezone,
		Rules:              input.Rules,
		Active:             true,
		ValidFrom:          now,
		ValidTo:            nil,
		CreatedAt:          now,
		UpdatedAt:          now,
		CreatedBy:          input.ActorID,
		UpdatedBy:          input.ActorID,
	}

	// Store in repository (simulated)
	if err := s.repo.SaveProfile(ctx, profile); err != nil {
		s.logger.WithError(err).Error("Failed to save profile")
		return nil, fmt.Errorf("save profile: %w", err)
	}

	// Audit log
	_ = s.auditService.RecordCreate(ctx, tenantID, "profile", profileID, map[string]interface{}{
		"profile_name":        input.ProfileName,
		"calendars":           input.Calendars,
		"conflict_resolution": input.ConflictResolution,
		"timezone":            input.Timezone,
	}, input.ActorID)

	s.logger.WithField("profile_id", profileID).Info("Profile created")

	return profile, nil
}

// GetByID retrieves a profile by ID
func (s *ProfileServiceImpl) GetByID(ctx context.Context, tenantID, profileID string) (*ScheduleProfile, error) {
	profile, err := s.repo.GetProfile(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}

	// Verify tenant access
	if profile == nil || profile.TenantID != tenantID {
		return nil, fmt.Errorf("profile not found or access denied")
	}

	// Check if still valid
	if profile.ValidTo != nil && profile.ValidTo.Before(time.Now().UTC()) {
		return nil, fmt.Errorf("profile has been deprecated")
	}

	return profile, nil
}

// ListActive lists all active profiles for a tenant
func (s *ProfileServiceImpl) ListActive(ctx context.Context, tenantID string, limit, offset int) ([]ScheduleProfile, error) {
	profiles, err := s.repo.ListProfilesByTenant(ctx, tenantID, true)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list profiles")
		return nil, fmt.Errorf("list profiles: %w", err)
	}

	// Apply pagination
	if offset >= len(profiles) {
		return []ScheduleProfile{}, nil
	}

	end := offset + limit
	if end > len(profiles) {
		end = len(profiles)
	}

	return profiles[offset:end], nil
}

// Update updates a profile using bitemporal versioning
func (s *ProfileServiceImpl) Update(ctx context.Context, tenantID, profileID string, input UpdateProfileInput) (*ScheduleProfile, error) {
	now := time.Now().UTC()

	// 1. Fetch current active version
	current, err := s.GetByID(ctx, tenantID, profileID)
	if err != nil {
		return nil, fmt.Errorf("fetch current profile: %w", err)
	}

	s.logger.WithField("profile_id", profileID).Info("Updating profile with bitemporal versioning")

	// 2. Close the old version by setting valid_to
	current.ValidTo = &now
	if err := s.repo.SaveProfile(ctx, current); err != nil {
		s.logger.WithError(err).Error("Failed to close old profile version")
		return nil, fmt.Errorf("close old version: %w", err)
	}

	// 3. Build new version by merging updates
	newProfile := s.mergeProfileUpdates(current, input, now)

	// 4. Insert new version
	if err := s.repo.SaveProfile(ctx, newProfile); err != nil {
		s.logger.WithError(err).Error("Failed to insert new profile version")
		// Note: In production, would have compensation logic to reopen old version
		return nil, fmt.Errorf("insert new version: %w", err)
	}

	// 5. Audit log
	_ = s.auditService.RecordUpdate(ctx, tenantID, "profile", profileID,
		map[string]interface{}{
			"profile_name":        current.ProfileName,
			"calendars":           current.Calendars,
			"conflict_resolution": current.ConflictResolution,
			"timezone":            current.Timezone,
		},
		map[string]interface{}{
			"profile_name":        newProfile.ProfileName,
			"calendars":           newProfile.Calendars,
			"conflict_resolution": newProfile.ConflictResolution,
			"timezone":            newProfile.Timezone,
		},
		input.ActorID,
	)

	s.logger.WithField("profile_id", newProfile.ID).Info("Profile updated with new version")

	return newProfile, nil
}

// Delete soft-deletes a profile
func (s *ProfileServiceImpl) Delete(ctx context.Context, tenantID, profileID string, actorID string) error {
	current, err := s.GetByID(ctx, tenantID, profileID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	current.ValidTo = &now
	current.Active = false

	if err := s.repo.SaveProfile(ctx, current); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}

	// Audit
	_ = s.auditService.RecordDelete(ctx, tenantID, "profile", profileID,
		map[string]interface{}{
			"profile_name": current.ProfileName,
			"calendars":    current.Calendars,
		},
		actorID,
	)

	s.logger.WithField("profile_id", profileID).Info("Profile soft-deleted")
	return nil
}

// ListVersions lists all versions of a profile
func (s *ProfileServiceImpl) ListVersions(ctx context.Context, tenantID, profileID string) ([]ScheduleProfile, error) {
	profiles, err := s.repo.ListProfilesByID(ctx, profileID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}

	// Verify tenant access
	result := make([]ScheduleProfile, 0)
	for _, p := range profiles {
		if p.TenantID == tenantID {
			result = append(result, p)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no versions found or access denied")
	}

	return result, nil
}

// mergeProfileUpdates creates a new profile struct with applied updates
func (s *ProfileServiceImpl) mergeProfileUpdates(current *ScheduleProfile, input UpdateProfileInput, now time.Time) *ScheduleProfile {
	newProfile := &ScheduleProfile{
		ID:                 uuid.New().String(), // New ID for new version
		TenantID:           current.TenantID,
		ProfileName:        current.ProfileName,
		Description:        current.Description,
		Calendars:          current.Calendars,
		ConflictResolution: current.ConflictResolution,
		Timezone:           current.Timezone,
		Rules:              current.Rules,
		Active:             current.Active,
		ValidFrom:          now,
		ValidTo:            nil,
		CreatedAt:          now,
		UpdatedAt:          now,
		CreatedBy:          current.CreatedBy,
		UpdatedBy:          input.ActorID,
	}

	if input.ProfileName != nil {
		newProfile.ProfileName = *input.ProfileName
	}
	if input.Description != nil {
		newProfile.Description = *input.Description
	}
	if input.Calendars != nil {
		newProfile.Calendars = *input.Calendars
	}
	if input.ConflictResolution != nil {
		newProfile.ConflictResolution = *input.ConflictResolution
	}
	if input.Timezone != nil {
		newProfile.Timezone = *input.Timezone
	}
	if input.Rules != nil {
		newProfile.Rules = *input.Rules
	}
	if input.Active != nil {
		newProfile.Active = *input.Active
	}

	return newProfile
}
