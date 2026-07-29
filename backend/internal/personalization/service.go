package personalization

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Profile struct {
	ProfileID           uuid.UUID `db:"profile_id" json:"profile_id"`
	TenantID            uuid.UUID `db:"tenant_id" json:"tenant_id"`
	UserHash            string    `db:"user_hash" json:"user_hash"`
	PreferredDomain     string    `db:"preferred_domain" json:"preferred_domain"`
	PreferredCurrency   string    `db:"preferred_currency" json:"preferred_currency"`
	PreferredDialects   []string  `db:"preferred_dialects" json:"preferred_dialects"`
	FrequentBOKeys      []string  `db:"frequent_bo_keys" json:"frequent_bo_keys"`
	SavedFilterPresets  []byte    `db:"saved_filter_presets" json:"saved_filter_presets"`
	FeedbackScoreBias   float64   `db:"feedback_score_bias" json:"feedback_score_bias"`
	PinnedBOKeys        []string  `db:"pinned_bo_keys" json:"pinned_bo_keys"`
	QuickLaunchFilters  []byte    `db:"quick_launch_filters" json:"quick_launch_filters"`
	DefaultPageLayout   *string   `db:"default_page_layout" json:"default_page_layout"`
	DefaultDashboardDomain string `db:"default_dashboard_domain" json:"default_dashboard_domain"`
	UpdatedAt           time.Time `db:"updated_at" json:"updated_at"`
}

type Service struct {
	db *sqlx.DB
}

func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

func ComputeUserHash(userID, tenantID string) string {
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%s:uisce_salt_2026", userID, tenantID)))
	return hex.EncodeToString(hasher.Sum(nil))[:16]
}

func (s *Service) GetProfile(ctx context.Context, tenantID, userID string) (*Profile, error) {
	userHash := ComputeUserHash(userID, tenantID)

	var p Profile
	err := s.db.GetContext(ctx, &p, `
		SELECT profile_id, tenant_id, user_hash,
		       preferred_domain, preferred_currency, preferred_dialects,
		       frequent_bo_keys, saved_filter_presets, feedback_score_bias,
		       pinned_bo_keys, quick_launch_filters, default_page_layout,
		       default_dashboard_domain, updated_at
		FROM user_personalization_profiles
		WHERE tenant_id = $1 AND user_hash = $2
	`, tenantID, userHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get personalization profile: %w", err)
	}
	return &p, nil
}

func (s *Service) UpsertProfile(ctx context.Context, p *Profile) error {
	p.UpdatedAt = time.Now()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_personalization_profiles (
			profile_id, tenant_id, user_hash,
			preferred_domain, preferred_currency, preferred_dialects,
			frequent_bo_keys, saved_filter_presets, feedback_score_bias,
			pinned_bo_keys, quick_launch_filters, default_page_layout,
			default_dashboard_domain, updated_at
		) VALUES (
			$1, $2, $3,
			COALESCE($4, 'PORTFOLIO'), COALESCE($5, 'USD'), COALESCE($6, ARRAY['POSTGRES']::TEXT[]),
			COALESCE($7, ARRAY[]::TEXT[]), COALESCE($8, '{}'::jsonb), COALESCE($9, 1.0),
			COALESCE($10, ARRAY[]::TEXT[]), COALESCE($11, '{}'::jsonb), $12,
			COALESCE($13, 'PORTFOLIO'), $14
		)
		ON CONFLICT (tenant_id, user_hash) DO UPDATE SET
			preferred_domain = COALESCE(EXCLUDED.preferred_domain, user_personalization_profiles.preferred_domain),
			preferred_currency = COALESCE(EXCLUDED.preferred_currency, user_personalization_profiles.preferred_currency),
			preferred_dialects = COALESCE(EXCLUDED.preferred_dialects, user_personalization_profiles.preferred_dialects),
			frequent_bo_keys = EXCLUDED.frequent_bo_keys,
			saved_filter_presets = EXCLUDED.saved_filter_presets,
			feedback_score_bias = EXCLUDED.feedback_score_bias,
			pinned_bo_keys = EXCLUDED.pinned_bo_keys,
			quick_launch_filters = EXCLUDED.quick_launch_filters,
			default_page_layout = EXCLUDED.default_page_layout,
			default_dashboard_domain = EXCLUDED.default_dashboard_domain,
			updated_at = EXCLUDED.updated_at
	`, p.ProfileID, p.TenantID, p.UserHash,
		p.PreferredDomain, p.PreferredCurrency, p.PreferredDialects,
		p.FrequentBOKeys, p.SavedFilterPresets, p.FeedbackScoreBias,
		p.PinnedBOKeys, p.QuickLaunchFilters, p.DefaultPageLayout,
		p.DefaultDashboardDomain, p.UpdatedAt)
	return err
}

func (s *Service) PinBO(ctx context.Context, tenantID, userID, boKey string) error {
	userHash := ComputeUserHash(userID, tenantID)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_personalization_profiles (profile_id, tenant_id, user_hash, pinned_bo_keys)
		VALUES ($1, $2, $3, ARRAY[$4]::TEXT[])
		ON CONFLICT (tenant_id, user_hash) DO UPDATE SET
			pinned_bo_keys = array_distinct(
				COALESCE(user_personalization_profiles.pinned_bo_keys, ARRAY[]::TEXT[]) || ARRAY[$4]::TEXT[]
			),
			updated_at = NOW()
	`, uuid.New(), tenantID, userHash, boKey)
	return err
}

func (s *Service) UnpinBO(ctx context.Context, tenantID, userID, boKey string) error {
	userHash := ComputeUserHash(userID, tenantID)
	_, err := s.db.ExecContext(ctx, `
		UPDATE user_personalization_profiles
		SET pinned_bo_keys = array_remove(pinned_bo_keys, $4),
		    updated_at = NOW()
		WHERE tenant_id = $1 AND user_hash = $2
	`, tenantID, userHash, boKey)
	return err
}

func (s *Service) BumpBOFrequency(ctx context.Context, tenantID, userID, boKey string) error {
	userHash := ComputeUserHash(userID, tenantID)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_personalization_profiles (profile_id, tenant_id, user_hash, frequent_bo_keys)
		VALUES ($1, $2, $3, ARRAY[$4]::TEXT[])
		ON CONFLICT (tenant_id, user_hash) DO UPDATE SET
			frequent_bo_keys = (
				SELECT ARRAY_agg(key ORDER BY cnt DESC)
				FROM (
					SELECT unnest(COALESCE(user_personalization_profiles.frequent_bo_keys, ARRAY[]::TEXT[]) || ARRAY[$4]::TEXT[]) AS key,
					       count(*) AS cnt
					GROUP BY key
					ORDER BY cnt DESC
					LIMIT 10
				) sub
			),
			updated_at = NOW()
	`, uuid.New(), tenantID, userHash, boKey)
	return err
}

type CanonicalUserContext struct {
	UserID              string    `json:"user_id"`
	TenantID            string    `json:"tenant_id"`
	FunctionalRole      string    `json:"functional_role"`
	ClearanceLevel     string    `json:"clearance_level"`
	PreferredDomain     string    `json:"preferred_domain"`
	PreferredDialects   []string  `json:"preferred_dialects"`
	FrequentBOKeys      []string  `json:"frequent_bo_keys"`
	PinnedBOKeys        []string  `json:"pinned_bo_keys"`
	QuickLaunchFilters  map[string]interface{} `json:"quick_launch_filters"`
	DefaultPageLayout   *string   `json:"default_page_layout,omitempty"`
	DefaultDashboardDomain string `json:"default_dashboard_domain"`
}

func (s *Service) ResolveCanonicalContext(
	ctx context.Context,
	tenantID, userID, functionalRole, clearanceLevel string,
) (*CanonicalUserContext, error) {
	profile, err := s.GetProfile(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}

	ctx2 := &CanonicalUserContext{
		UserID:              userID,
		TenantID:            tenantID,
		FunctionalRole:      functionalRole,
		ClearanceLevel:      clearanceLevel,
		PreferredDomain:     "PORTFOLIO",
		PreferredDialects:   []string{"POSTGRES"},
		FrequentBOKeys:      []string{},
		PinnedBOKeys:        []string{},
		QuickLaunchFilters:  map[string]interface{}{},
		DefaultDashboardDomain: "PORTFOLIO",
	}

	if profile != nil {
		ctx2.PreferredDomain = profile.PreferredDomain
		ctx2.PreferredDialects = profile.PreferredDialects
		ctx2.FrequentBOKeys = profile.FrequentBOKeys
		ctx2.PinnedBOKeys = profile.PinnedBOKeys
		ctx2.DefaultPageLayout = profile.DefaultPageLayout
		ctx2.DefaultDashboardDomain = profile.DefaultDashboardDomain
		if len(profile.QuickLaunchFilters) > 0 {
			_ = parseJSON(profile.QuickLaunchFilters, &ctx2.QuickLaunchFilters)
		}
	}

	return ctx2, nil
}

func parseJSON(data []byte, out interface{}) error {
	if len(data) == 0 || string(data) == "{}" {
		return nil
	}
	return json.Unmarshal(data, out)
}
