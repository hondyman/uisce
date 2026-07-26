// Package notifications - Notification campaign orchestration.
//
// This file contains the canonical NotificationCampaignService that was
// extracted from internal/services/notification_campaign_service.go.
//
// Cardinal Rule 3 (no cycles): only depends on backend/models + stdlib.
// Cardinal Rule 7 (tenant security): tenantID required on all campaigns.
package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ============================================================================
// NOTIFICATION CAMPAIGN MODEL
// ============================================================================

// NotificationCampaign represents an orchestration of notifications targeting
// a specific audience with a schedule.
type NotificationCampaign struct {
	ID          uuid.UUID      `json:"id" db:"id"`
	TenantID    string         `json:"tenant_id" db:"tenant_id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Channel     Channel        `json:"channel" db:"channel"`
	Audience    pq.StringArray `json:"audience" db:"audience"`
	ScheduleAt  *time.Time     `json:"schedule_at,omitempty" db:"schedule_at"`
	Status      string         `json:"status" db:"status"`
	Metadata    []byte         `json:"metadata" db:"metadata"`
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// NOTIFICATION CAMPAIGN SERVICE
// ============================================================================

// NotificationCampaignService handles notification campaign orchestration
// including creation, scheduling, audience targeting, and execution.
type NotificationCampaignService struct {
	DB                  *sql.DB
	NotificationService *EngagementNotificationService
}

// NewNotificationCampaignService creates a new campaign service.
func NewNotificationCampaignService(db *sql.DB, notificationService *EngagementNotificationService) *NotificationCampaignService {
	return &NotificationCampaignService{
		DB:                  db,
		NotificationService: notificationService,
	}
}

// CreateCampaign creates a new notification campaign.
func (s *NotificationCampaignService) CreateCampaign(ctx context.Context, campaign NotificationCampaign) (*NotificationCampaign, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("notification campaign service: db is nil")
	}
	if campaign.ID == uuid.Nil {
		campaign.ID = uuid.New()
	}
	if campaign.CreatedAt.IsZero() {
		campaign.CreatedAt = time.Now()
	}
	campaign.UpdatedAt = time.Now()

	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO notification_campaigns (id, tenant_id, name, description, channel, audience,
		 schedule_at, status, metadata, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		campaign.ID, campaign.TenantID, campaign.Name, campaign.Description,
		campaign.Channel, campaign.Audience, campaign.ScheduleAt, campaign.Status,
		campaign.Metadata, campaign.CreatedAt, campaign.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create campaign: %w", err)
	}
	return &campaign, nil
}

// GetCampaign retrieves a campaign by ID.
func (s *NotificationCampaignService) GetCampaign(ctx context.Context, id uuid.UUID) (*NotificationCampaign, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("notification campaign service: db is nil")
	}
	var c NotificationCampaign
	err := s.DB.QueryRowContext(ctx,
		`SELECT id, tenant_id, name, description, channel, audience, schedule_at,
		        status, metadata, created_at, updated_at
		 FROM notification_campaigns WHERE id = $1`, id).Scan(
		&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Channel, &c.Audience,
		&c.ScheduleAt, &c.Status, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get campaign %s: %w", id, err)
	}
	return &c, nil
}

// ListCampaigns returns campaigns for a tenant, optionally filtered by status.
func (s *NotificationCampaignService) ListCampaigns(ctx context.Context, tenantID, status string) ([]NotificationCampaign, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("notification campaign service: db is nil")
	}
	query := `SELECT id, tenant_id, name, description, channel, audience, schedule_at,
	                 status, metadata, created_at, updated_at
	          FROM notification_campaigns WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}
	query += " ORDER BY created_at DESC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list campaigns: %w", err)
	}
	defer rows.Close()

	var campaigns []NotificationCampaign
	for rows.Next() {
		var c NotificationCampaign
		if err := rows.Scan(
			&c.ID, &c.TenantID, &c.Name, &c.Description, &c.Channel, &c.Audience,
			&c.ScheduleAt, &c.Status, &c.Metadata, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan campaign: %w", err)
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, rows.Err()
}

// UpdateCampaignStatus updates a campaign's status.
func (s *NotificationCampaignService) UpdateCampaignStatus(ctx context.Context, id uuid.UUID, status string) error {
	if s.DB == nil {
		return fmt.Errorf("notification campaign service: db is nil")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE notification_campaigns SET status = $1, updated_at = $2 WHERE id = $3`,
		status, time.Now(), id)
	return err
}

// DeleteCampaign removes a campaign by ID.
func (s *NotificationCampaignService) DeleteCampaign(ctx context.Context, id uuid.UUID) error {
	if s.DB == nil {
		return fmt.Errorf("notification campaign service: db is nil")
	}
	_, err := s.DB.ExecContext(ctx,
		`DELETE FROM notification_campaigns WHERE id = $1`, id)
	return err
}

// ============================================================================
// CAMPAIGN EXECUTION
// ============================================================================

// ExecuteCampaign runs a scheduled campaign by recording events for all
// audience members.
func (s *NotificationCampaignService) ExecuteCampaign(ctx context.Context, campaignID uuid.UUID) error {
	if s.DB == nil {
		return fmt.Errorf("notification campaign service: db is nil")
	}
	campaign, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return err
	}

	// Marshal metadata to JSON
	metadataJSON, err := json.Marshal(map[string]interface{}{
		"campaign_id":   campaign.ID,
		"campaign_name": campaign.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal campaign metadata: %w", err)
	}

	// For each audience member, record an event
	for _, userID := range campaign.Audience {
		if s.NotificationService != nil {
			event := NotificationEvent{
				ID:       uuid.New(),
				TenantID: campaign.TenantID,
				UserID:   userID,
				Title:    campaign.Name,
				Body:     campaign.Description,
				Channel:  string(campaign.Channel),
				Priority: 5,
				Metadata: metadataJSON,
			}
			if err := s.NotificationService.RecordEvent(ctx, event); err != nil {
				return fmt.Errorf("failed to record event for user %s: %w", userID, err)
			}
		}
	}

	// Mark campaign as completed
	return s.UpdateCampaignStatus(ctx, campaignID, "completed")
}

// Compile-time guard.
var _ = pq.StringArray{}