package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/lib/pq"
)

// NotificationCampaignService handles notification campaign orchestration
type NotificationCampaignService struct {
	db                  *sql.DB
	notificationService *EngagementNotificationService
}

// NewNotificationCampaignService creates a new campaign service
func NewNotificationCampaignService(db *sql.DB, notificationService *EngagementNotificationService) *NotificationCampaignService {
	return &NotificationCampaignService{
		db:                  db,
		notificationService: notificationService,
	}
}

// CreateCampaign creates a new notification campaign
func (s *NotificationCampaignService) CreateCampaign(ctx context.Context, campaign *models.NotificationCampaign) error {
	campaign.ID = uuid.New().String()
	campaign.CreatedAt = time.Now()
	campaign.UpdatedAt = time.Now()
	campaign.Status = "draft"

	query := `
		INSERT INTO notification_campaigns (
			id, name, description, type, status, target_users, user_segment, steps, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	targetUsersArray := pq.Array(campaign.TargetUsers)
	stepsJSON, _ := json.Marshal(campaign.Steps)

	_, err := s.db.ExecContext(ctx, query,
		campaign.ID, campaign.Name, campaign.Description, campaign.Type, campaign.Status,
		targetUsersArray, campaign.UserSegment, stepsJSON, campaign.CreatedBy,
	)

	return err
}

// LaunchCampaign launches a notification campaign
func (s *NotificationCampaignService) LaunchCampaign(ctx context.Context, campaignID string) error {
	// Get campaign details
	campaign, err := s.GetCampaign(ctx, campaignID)
	if err != nil {
		return err
	}

	// Update campaign status to active
	err = s.updateCampaignStatus(ctx, campaignID, "active")
	if err != nil {
		return err
	}

	// Get target users based on criteria
	targetUsers, err := s.getTargetUsers(ctx, campaign)
	if err != nil {
		log.Printf("Failed to get target users for campaign %s: %v", campaignID, err)
		return err
	}

	// Schedule notifications for each user
	for _, userID := range targetUsers {
		err = s.scheduleUserNotification(ctx, campaign, userID)
		if err != nil {
			log.Printf("Failed to schedule notification for user %s in campaign %s: %v", userID, campaignID, err)
			continue
		}
	}

	return nil
}

// GetCampaign retrieves a campaign by ID
func (s *NotificationCampaignService) GetCampaign(ctx context.Context, campaignID string) (*models.NotificationCampaign, error) {
	query := `
		SELECT id, name, description, type, status, target_users, user_segment, steps,
			   created_by, created_at, updated_at
		FROM notification_campaigns WHERE id = $1
	`

	var campaign models.NotificationCampaign
	var targetUsersArray pq.StringArray
	var stepsJSON []byte

	var createdBy sql.NullString
	err := s.db.QueryRowContext(ctx, query, campaignID).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Type, &campaign.Status,
		&targetUsersArray, &campaign.UserSegment, &stepsJSON, &createdBy,
		&campaign.CreatedAt, &campaign.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		campaign.CreatedBy = createdBy.String
	} else {
		campaign.CreatedBy = ""
	}

	campaign.TargetUsers = []string(targetUsersArray)
	json.Unmarshal(stepsJSON, &campaign.Steps)

	return &campaign, nil
}

// GetCampaignAnalytics retrieves analytics for a campaign
func (s *NotificationCampaignService) GetCampaignAnalytics(ctx context.Context, campaignID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(DISTINCT en.id) as total_notifications,
			COUNT(DISTINCT CASE WHEN en.status = 'sent' THEN en.id END) as sent_notifications,
			COUNT(DISTINCT CASE WHEN na.event_type = 'opened' THEN na.notification_id END) as opened_notifications,
			COUNT(DISTINCT CASE WHEN na.event_type = 'clicked' THEN na.notification_id END) as clicked_notifications,
			AVG(CASE WHEN na.event_type = 'opened' THEN 1 ELSE 0 END) as avg_open_rate,
			AVG(CASE WHEN na.event_type = 'clicked' THEN 1 ELSE 0 END) as avg_click_rate
		FROM engagement_notifications en
		LEFT JOIN notification_analytics na ON en.id = na.notification_id
		WHERE en.campaign_id = $1
	`

	var totalNotifications, sentNotifications, openedNotifications, clickedNotifications int
	var avgOpenRate, avgClickRate float64

	err := s.db.QueryRowContext(ctx, query, campaignID).Scan(
		&totalNotifications, &sentNotifications, &openedNotifications, &clickedNotifications,
		&avgOpenRate, &avgClickRate,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"campaign_id":           campaignID,
		"total_notifications":   totalNotifications,
		"sent_notifications":    sentNotifications,
		"opened_notifications":  openedNotifications,
		"clicked_notifications": clickedNotifications,
		"avg_open_rate":         avgOpenRate,
		"avg_click_rate":        avgClickRate,
	}, nil
}

// PauseCampaign pauses an active campaign
func (s *NotificationCampaignService) PauseCampaign(ctx context.Context, campaignID string) error {
	return s.updateCampaignStatus(ctx, campaignID, "paused")
}

// ResumeCampaign resumes a paused campaign
func (s *NotificationCampaignService) ResumeCampaign(ctx context.Context, campaignID string) error {
	return s.updateCampaignStatus(ctx, campaignID, "active")
}

// StopCampaign stops a campaign
func (s *NotificationCampaignService) StopCampaign(ctx context.Context, campaignID string) error {
	return s.updateCampaignStatus(ctx, campaignID, "stopped")
}

// GetActiveCampaigns retrieves all active campaigns
func (s *NotificationCampaignService) GetActiveCampaigns(ctx context.Context) ([]*models.NotificationCampaign, error) {
	query := `
		SELECT id, name, description, type, status, target_users, user_segment, steps,
			   created_by, created_at, updated_at
		FROM notification_campaigns
		WHERE status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []*models.NotificationCampaign
	for rows.Next() {
		var campaign models.NotificationCampaign
		var targetUsersArray pq.StringArray
		var stepsJSON []byte

		var createdBy sql.NullString
		err := rows.Scan(
			&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Type, &campaign.Status,
			&targetUsersArray, &campaign.UserSegment, &stepsJSON, &createdBy,
			&campaign.CreatedAt, &campaign.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if createdBy.Valid {
			campaign.CreatedBy = createdBy.String
		} else {
			campaign.CreatedBy = ""
		}

		campaign.TargetUsers = []string(targetUsersArray)
		json.Unmarshal(stepsJSON, &campaign.Steps)

		campaigns = append(campaigns, &campaign)
	}

	return campaigns, nil
}

// Helper methods

func (s *NotificationCampaignService) updateCampaignStatus(ctx context.Context, campaignID, status string) error {
	query := `UPDATE notification_campaigns SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, campaignID)
	return err
}

func (s *NotificationCampaignService) getTargetUsers(ctx context.Context, campaign *models.NotificationCampaign) ([]string, error) {
	// This is a simplified implementation - in a real system, you'd have complex
	// user segmentation logic based on the target criteria
	query := `
		SELECT DISTINCT u.id
		FROM users u
		WHERE u.id = ANY($1) OR $1 IS NULL
	`

	var userIDs []string
	if len(campaign.TargetUsers) > 0 {
		userIDs = campaign.TargetUsers
	}

	rows, err := s.db.QueryContext(ctx, query, pq.Array(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targetUsers []string
	for rows.Next() {
		var userID string
		err := rows.Scan(&userID)
		if err != nil {
			return nil, err
		}
		targetUsers = append(targetUsers, userID)
	}

	return targetUsers, nil
}

func (s *NotificationCampaignService) scheduleUserNotification(ctx context.Context, campaign *models.NotificationCampaign, userID string) error {
	// For now, use the first step's template. In a real implementation,
	// you'd handle multi-step campaigns with delays and triggers
	if len(campaign.Steps) == 0 {
		return fmt.Errorf("campaign has no steps defined")
	}

	firstStep := campaign.Steps[0]
	template, err := s.getNotificationTemplate(ctx, firstStep.TemplateID)
	if err != nil {
		return err
	}

	now := time.Now()
	notification := &models.EngagementNotification{
		UserID:      userID,
		Type:        template.Type,
		Title:       template.Title,
		Message:     template.Message,
		RichContent: template.RichContent,
		Priority:    2, // normal priority
		Channels:    template.Channels,
		Status:      "scheduled",
		ScheduledAt: &now,
		CreatedBy:   campaign.CreatedBy,
		UserSegment: campaign.UserSegment,
		TemplateID:  template.ID,
		Personalization: map[string]interface{}{
			"campaign_name": campaign.Name,
			"user_id":       userID,
		},
	}

	return s.notificationService.CreateNotification(ctx, notification)
}

func (s *NotificationCampaignService) getNotificationTemplate(ctx context.Context, templateID string) (*models.NotificationTemplate, error) {
	query := `
		SELECT id, name, type, subject, title, message, rich_content, variables,
			   channels, created_by, created_at, updated_at
		FROM notification_templates WHERE id = $1
	`

	var template models.NotificationTemplate
	var richContentJSON []byte
	var variablesArray, channelsArray pq.StringArray

	var templateCreatedBy sql.NullString
	err := s.db.QueryRowContext(ctx, query, templateID).Scan(
		&template.ID, &template.Name, &template.Type, &template.Subject, &template.Title,
		&template.Message, &richContentJSON, &variablesArray, &channelsArray,
		&templateCreatedBy, &template.CreatedAt, &template.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if templateCreatedBy.Valid {
		template.CreatedBy = templateCreatedBy.String
	} else {
		template.CreatedBy = ""
	}
	template.Variables = []string(variablesArray)
	template.Channels = []string(channelsArray)
	json.Unmarshal(richContentJSON, &template.RichContent)

	return &template, nil
}
