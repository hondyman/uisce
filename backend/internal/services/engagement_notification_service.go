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

// EngagementNotificationService handles advanced notification engagement features
type EngagementNotificationService struct {
	db                    *sql.DB                               // Assuming you have a database connection
	broadcastFunc         func(userID string, message []byte)   // Function to broadcast to specific user
	broadcastAllFunc      func(message []byte)                  // Function to broadcast to all users
	broadcastAudienceFunc func(audience string, message []byte) // Function to broadcast to audience
}

// NewEngagementNotificationService creates a new engagement notification service
func NewEngagementNotificationService(db *sql.DB) *EngagementNotificationService {
	return &EngagementNotificationService{db: db}
}

// SetBroadcastFunctions sets the broadcasting functions for real-time delivery
func (s *EngagementNotificationService) SetBroadcastFunctions(
	broadcastFunc func(userID string, message []byte),
	broadcastAllFunc func(message []byte),
	broadcastAudienceFunc func(audience string, message []byte),
) {
	s.broadcastFunc = broadcastFunc
	s.broadcastAllFunc = broadcastAllFunc
	s.broadcastAudienceFunc = broadcastAudienceFunc
}

// CreateNotification creates a new engagement notification
func (s *EngagementNotificationService) CreateNotification(ctx context.Context, notification *models.EngagementNotification) error {
	notification.ID = uuid.New().String()
	notification.CreatedAt = time.Now()
	notification.UpdatedAt = time.Now()

	query := `
		INSERT INTO engagement_notifications (
			id, user_id, type, title, message, rich_content, priority, channels,
			status, scheduled_at, expires_at, created_by, engagement_score,
			user_segment, ab_test_variant, template_id, personalization, actions, cta
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`

	richContentJSON, _ := json.Marshal(notification.RichContent)
	channelsArray := pq.Array(notification.Channels)
	personalizationJSON, _ := json.Marshal(notification.Personalization)
	actionsJSON, _ := json.Marshal(notification.Actions)
	ctaJSON, _ := json.Marshal(notification.CTA)

	_, err := s.db.ExecContext(ctx, query,
		notification.ID, notification.UserID, notification.Type, notification.Title,
		notification.Message, richContentJSON, notification.Priority, channelsArray,
		notification.Status, notification.ScheduledAt, notification.ExpiresAt,
		notification.CreatedBy, notification.EngagementScore, notification.UserSegment,
		notification.ABTestVariant, notification.TemplateID, personalizationJSON,
		actionsJSON, ctaJSON,
	)

	return err
}

// SendNotification sends a notification through appropriate channels
func (s *EngagementNotificationService) SendNotification(ctx context.Context, notificationID string) error {
	// Get notification details
	notification, err := s.GetNotification(ctx, notificationID)
	if err != nil {
		return err
	}

	// Check user preferences
	preferences, err := s.GetUserPreferences(ctx, notification.UserID)
	if err != nil {
		log.Printf("Failed to get user preferences: %v", err)
		preferences = &models.UserNotificationPreferences{
			EmailEnabled: true,
			SMSEnabled:   false,
			PushEnabled:  true,
			InAppEnabled: true,
		}
	}

	// Send through enabled channels
	for _, channel := range notification.Channels {
		if s.isChannelEnabled(channel, preferences) {
			switch channel {
			case "email":
				go s.sendEmailNotification(ctx, notification)
			case "sms":
				go s.sendSMSNotification(ctx, notification)
			case "push":
				go s.sendPushNotification(ctx, notification)
			case "in_app":
				go s.sendInAppNotification(ctx, notification)
			}
		}
	}

	// Update notification status
	return s.updateNotificationStatus(ctx, notificationID, "sent", &time.Time{})
}

// GetNotification retrieves a notification by ID
func (s *EngagementNotificationService) GetNotification(ctx context.Context, notificationID string) (*models.EngagementNotification, error) {
	query := `
		SELECT id, user_id, type, title, message, rich_content, priority, channels,
			   status, scheduled_at, sent_at, read_at, clicked_at, dismissed_at,
			   expires_at, created_by, created_at, updated_at, engagement_score,
			   user_segment, ab_test_variant, template_id, personalization, actions, cta
		FROM engagement_notifications WHERE id = $1
	`

	var notification models.EngagementNotification
	var richContentJSON, personalizationJSON, actionsJSON, ctaJSON []byte
	var channelsArray pq.StringArray

	var createdBy sql.NullString
	err := s.db.QueryRowContext(ctx, query, notificationID).Scan(
		&notification.ID, &notification.UserID, &notification.Type, &notification.Title,
		&notification.Message, &richContentJSON, &notification.Priority, &channelsArray,
		&notification.Status, &notification.ScheduledAt, &notification.SentAt,
		&notification.ReadAt, &notification.ClickedAt, &notification.DismissedAt,
		&notification.ExpiresAt, &createdBy, &notification.CreatedAt,
		&notification.UpdatedAt, &notification.EngagementScore, &notification.UserSegment,
		&notification.ABTestVariant, &notification.TemplateID, &personalizationJSON,
		&actionsJSON, &ctaJSON,
	)

	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		notification.CreatedBy = createdBy.String
	} else {
		notification.CreatedBy = ""
	}

	notification.Channels = []string(channelsArray)
	json.Unmarshal(richContentJSON, &notification.RichContent)
	json.Unmarshal(personalizationJSON, &notification.Personalization)
	json.Unmarshal(actionsJSON, &notification.Actions)
	json.Unmarshal(ctaJSON, &notification.CTA)

	return &notification, nil
}

// GetUserNotifications retrieves notifications for a user
func (s *EngagementNotificationService) GetUserNotifications(ctx context.Context, userID string, limit, offset int) ([]*models.EngagementNotification, error) {
	query := `
		SELECT id, user_id, type, title, message, rich_content, priority, channels,
			   status, scheduled_at, sent_at, read_at, clicked_at, dismissed_at,
			   expires_at, created_by, created_at, updated_at, engagement_score,
			   user_segment, ab_test_variant, template_id, personalization, actions, cta
		FROM engagement_notifications
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*models.EngagementNotification
	for rows.Next() {
		var notification models.EngagementNotification
		var richContentJSON, personalizationJSON, actionsJSON, ctaJSON []byte
		var channelsArray pq.StringArray

		var createdBy sql.NullString
		err := rows.Scan(
			&notification.ID, &notification.UserID, &notification.Type, &notification.Title,
			&notification.Message, &richContentJSON, &notification.Priority, &channelsArray,
			&notification.Status, &notification.ScheduledAt, &notification.SentAt,
			&notification.ReadAt, &notification.ClickedAt, &notification.DismissedAt,
			&notification.ExpiresAt, &createdBy, &notification.CreatedAt,
			&notification.UpdatedAt, &notification.EngagementScore, &notification.UserSegment,
			&notification.ABTestVariant, &notification.TemplateID, &personalizationJSON,
			&actionsJSON, &ctaJSON,
		)
		if err != nil {
			return nil, err
		}

		notification.Channels = []string(channelsArray)
		json.Unmarshal(richContentJSON, &notification.RichContent)
		json.Unmarshal(personalizationJSON, &notification.Personalization)
		json.Unmarshal(actionsJSON, &notification.Actions)
		json.Unmarshal(ctaJSON, &notification.CTA)

		if createdBy.Valid {
			notification.CreatedBy = createdBy.String
		} else {
			notification.CreatedBy = ""
		}

		notifications = append(notifications, &notification)
	}

	return notifications, nil
}

// TrackEngagementEvent tracks user interaction with notifications
func (s *EngagementNotificationService) TrackEngagementEvent(ctx context.Context, event *models.NotificationAnalytics) error {
	event.ID = uuid.New().String()

	query := `
		INSERT INTO notification_analytics (
			id, notification_id, user_id, event_type, event_timestamp,
			user_agent, ip_address, device_type, location, session_id, additional_metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	metadataJSON, _ := json.Marshal(event.AdditionalMetadata)

	_, err := s.db.ExecContext(ctx, query,
		event.ID, event.NotificationID, event.UserID, event.EventType, event.EventTimestamp,
		event.UserAgent, event.IPAddress, event.DeviceType, event.Location,
		event.SessionID, metadataJSON,
	)

	if err != nil {
		return err
	}

	// Update user engagement profile
	return s.updateUserEngagementProfile(ctx, event.UserID)
}

// GetUserPreferences retrieves user notification preferences
func (s *EngagementNotificationService) GetUserPreferences(ctx context.Context, userID string) (*models.UserNotificationPreferences, error) {
	query := `
		SELECT user_id, email_enabled, sms_enabled, push_enabled, in_app_enabled,
			   quiet_hours_start, quiet_hours_end, timezone, channel_preferences,
			   type_preferences, frequency_preferences, created_at, updated_at
		FROM user_notification_preferences WHERE user_id = $1
	`

	var prefs models.UserNotificationPreferences
	var channelPrefsJSON, typePrefsJSON, freqPrefsJSON []byte

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&prefs.UserID, &prefs.EmailEnabled, &prefs.SMSEnabled, &prefs.PushEnabled,
		&prefs.InAppEnabled, &prefs.QuietHoursStart, &prefs.QuietHoursEnd,
		&prefs.Timezone, &channelPrefsJSON, &typePrefsJSON, &freqPrefsJSON,
		&prefs.CreatedAt, &prefs.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	json.Unmarshal(channelPrefsJSON, &prefs.ChannelPreferences)
	json.Unmarshal(typePrefsJSON, &prefs.TypePreferences)
	json.Unmarshal(freqPrefsJSON, &prefs.FrequencyPreferences)

	return &prefs, nil
}

// UpdateUserPreferences updates user notification preferences
func (s *EngagementNotificationService) UpdateUserPreferences(ctx context.Context, prefs *models.UserNotificationPreferences) error {
	prefs.UpdatedAt = time.Now()

	query := `
		INSERT INTO user_notification_preferences (
			user_id, email_enabled, sms_enabled, push_enabled, in_app_enabled,
			quiet_hours_start, quiet_hours_end, timezone, channel_preferences,
			type_preferences, frequency_preferences, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			push_enabled = EXCLUDED.push_enabled,
			in_app_enabled = EXCLUDED.in_app_enabled,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			timezone = EXCLUDED.timezone,
			channel_preferences = EXCLUDED.channel_preferences,
			type_preferences = EXCLUDED.type_preferences,
			frequency_preferences = EXCLUDED.frequency_preferences,
			updated_at = EXCLUDED.updated_at
	`

	channelPrefsJSON, _ := json.Marshal(prefs.ChannelPreferences)
	typePrefsJSON, _ := json.Marshal(prefs.TypePreferences)
	freqPrefsJSON, _ := json.Marshal(prefs.FrequencyPreferences)

	_, err := s.db.ExecContext(ctx, query,
		prefs.UserID, prefs.EmailEnabled, prefs.SMSEnabled, prefs.PushEnabled,
		prefs.InAppEnabled, prefs.QuietHoursStart, prefs.QuietHoursEnd,
		prefs.Timezone, channelPrefsJSON, typePrefsJSON, freqPrefsJSON, prefs.UpdatedAt,
	)

	return err
}

// CreateNotificationTemplate creates a reusable notification template
func (s *EngagementNotificationService) CreateNotificationTemplate(ctx context.Context, template *models.NotificationTemplate) error {
	template.ID = uuid.New().String()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	query := `
		INSERT INTO notification_templates (
			id, name, type, subject, title, message, rich_content, variables,
			channels, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	richContentJSON, _ := json.Marshal(template.RichContent)
	variablesArray := pq.Array(template.Variables)
	channelsArray := pq.Array(template.Channels)

	_, err := s.db.ExecContext(ctx, query,
		template.ID, template.Name, template.Type, template.Subject, template.Title,
		template.Message, richContentJSON, variablesArray, channelsArray, template.CreatedBy,
	)

	return err
}

// GetEngagementAnalytics retrieves engagement analytics for notifications
func (s *EngagementNotificationService) GetEngagementAnalytics(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_sent,
			COUNT(CASE WHEN event_type = 'opened' THEN 1 END) as total_opened,
			COUNT(CASE WHEN event_type = 'clicked' THEN 1 END) as total_clicked,
			AVG(CASE WHEN event_type = 'opened' THEN 1 ELSE 0 END) as avg_open_rate,
			AVG(CASE WHEN event_type = 'clicked' THEN 1 ELSE 0 END) as avg_click_rate
		FROM notification_analytics
		WHERE event_timestamp BETWEEN $1 AND $2
	`

	var totalSent, totalOpened, totalClicked int
	var avgOpenRate, avgClickRate float64

	err := s.db.QueryRowContext(ctx, query, startDate, endDate).Scan(
		&totalSent, &totalOpened, &totalClicked, &avgOpenRate, &avgClickRate,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total_sent":     totalSent,
		"total_opened":   totalOpened,
		"total_clicked":  totalClicked,
		"avg_open_rate":  avgOpenRate,
		"avg_click_rate": avgClickRate,
		"period_start":   startDate,
		"period_end":     endDate,
	}, nil
}

// Helper methods

func (s *EngagementNotificationService) isChannelEnabled(channel string, prefs *models.UserNotificationPreferences) bool {
	switch channel {
	case "email":
		return prefs.EmailEnabled
	case "sms":
		return prefs.SMSEnabled
	case "push":
		return prefs.PushEnabled
	case "in_app":
		return prefs.InAppEnabled
	default:
		return false
	}
}

func (s *EngagementNotificationService) updateNotificationStatus(ctx context.Context, notificationID, status string, timestamp *time.Time) error {
	query := `UPDATE engagement_notifications SET status = $1, sent_at = $2, updated_at = NOW() WHERE id = $3`
	_, err := s.db.ExecContext(ctx, query, status, timestamp, notificationID)
	return err
}

func (s *EngagementNotificationService) updateUserEngagementProfile(ctx context.Context, userID string) error {
	query := `
		INSERT INTO user_engagement_profiles (
			user_id, total_notifications, opened_notifications, clicked_notifications,
			dismissed_notifications, avg_open_rate, avg_click_rate, last_activity,
			engagement_score, updated_at
		)
		SELECT
			$1,
			COUNT(DISTINCT na.notification_id) as total_notifications,
			COUNT(DISTINCT CASE WHEN na.event_type = 'opened' THEN na.notification_id END) as opened_notifications,
			COUNT(DISTINCT CASE WHEN na.event_type = 'clicked' THEN na.notification_id END) as clicked_notifications,
			COUNT(DISTINCT CASE WHEN na.event_type = 'dismissed' THEN na.notification_id END) as dismissed_notifications,
			AVG(CASE WHEN na.event_type = 'opened' THEN 1 ELSE 0 END) as avg_open_rate,
			AVG(CASE WHEN na.event_type = 'clicked' THEN 1 ELSE 0 END) as avg_click_rate,
			MAX(na.event_timestamp) as last_activity,
			CASE
				WHEN COUNT(DISTINCT na.notification_id) = 0 THEN 0
				ELSE (COUNT(DISTINCT CASE WHEN na.event_type IN ('opened', 'clicked') THEN na.notification_id END)::float /
					  COUNT(DISTINCT na.notification_id)::float)
			END as engagement_score,
			NOW()
		FROM notification_analytics na
		WHERE na.user_id = $1
		ON CONFLICT (user_id) DO UPDATE SET
			total_notifications = EXCLUDED.total_notifications,
			opened_notifications = EXCLUDED.opened_notifications,
			clicked_notifications = EXCLUDED.clicked_notifications,
			dismissed_notifications = EXCLUDED.dismissed_notifications,
			avg_open_rate = EXCLUDED.avg_open_rate,
			click_rate = EXCLUDED.avg_click_rate,
			last_activity = EXCLUDED.last_activity,
			engagement_score = EXCLUDED.engagement_score,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

// Channel-specific sending methods (implement actual integrations)
func (s *EngagementNotificationService) sendEmailNotification(_ context.Context, notification *models.EngagementNotification) {
	log.Printf("[Email] Sending notification %s to user %s: %s", notification.ID, notification.UserID, notification.Title)
	// Implement actual email sending (SendGrid, AWS SES, etc.)
	// For now, log the email details
	to := notification.UserID + "@example.com" // Placeholder
	subject := notification.Title
	body := notification.Message
	fmt.Printf("Sending email to: %s\n", to)
	fmt.Printf("Subject: %s\n", subject)
	fmt.Printf("Body: %s\n", body)

	// TODO: Integration code would be:
	// - Initialize email client (SendGrid, SES, etc.)
	// - Build email message
	// - Send and handle response
	// - Track delivery status
}

func (s *EngagementNotificationService) sendSMSNotification(_ context.Context, notification *models.EngagementNotification) {
	log.Printf("[SMS] Sending notification %s to user %s: %s", notification.ID, notification.UserID, notification.Title)
	// Implement actual SMS sending (Twilio, AWS SNS, etc.)
	to := "+15551234567" // Placeholder
	body := notification.Message
	fmt.Printf("Sending SMS to: %s\n", to)
	fmt.Printf("Message: %s\n", body)

	// TODO: Integration code:
	// - Initialize SMS client (Twilio, SNS, etc.)
	// - Format phone number
	// - Send message and track status
}

func (s *EngagementNotificationService) sendPushNotification(_ context.Context, notification *models.EngagementNotification) {
	log.Printf("[Push] Sending notification %s to user %s: %s", notification.ID, notification.UserID, notification.Title)
	// Implement actual push notification sending (Firebase, etc.)
	deviceToken := "device_token_for_" + notification.UserID // Placeholder
	title := notification.Title
	body := notification.Message
	fmt.Printf("Sending push notification to device: %s\n", deviceToken)
	fmt.Printf("Title: %s\n", title)
	fmt.Printf("Body: %s\n", body)

	// TODO: Integration code:
	// - Initialize FCM/APNS client
	// - Build notification payload
	// - Send to device token
	// - Handle delivery receipts
}

func (s *EngagementNotificationService) sendInAppNotification(ctx context.Context, notification *models.EngagementNotification) {
	log.Printf("[In-App] Sending notification %s to user %s: %s", notification.ID, notification.UserID, notification.Title)
	// Broadcast via WebSocket for real-time delivery
	s.broadcastNotification(ctx, notification)
}

// WebSocket Broadcasting Methods

// broadcastNotification broadcasts a notification to WebSocket clients
func (s *EngagementNotificationService) broadcastNotification(ctx context.Context, notification *models.EngagementNotification) {
	// Create real-time message
	message := map[string]interface{}{
		"type": "notification",
		"data": map[string]interface{}{
			"id":               notification.ID,
			"user_id":          notification.UserID,
			"type":             notification.Type,
			"title":            notification.Title,
			"message":          notification.Message,
			"rich_content":     notification.RichContent,
			"priority":         notification.Priority,
			"channels":         notification.Channels,
			"status":           notification.Status,
			"scheduled_at":     notification.ScheduledAt,
			"expires_at":       notification.ExpiresAt,
			"created_by":       notification.CreatedBy,
			"engagement_score": notification.EngagementScore,
			"user_segment":     notification.UserSegment,
			"template_id":      notification.TemplateID,
			"personalization":  notification.Personalization,
			"actions":          notification.Actions,
			"cta":              notification.CTA,
			"created_at":       notification.CreatedAt,
			"updated_at":       notification.UpdatedAt,
		},
		"timestamp": time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("Failed to marshal notification message: %v", err)
		return
	}

	// Broadcast to specific user or all users based on the notification
	if notification.UserID != "" && s.broadcastFunc != nil {
		s.broadcastFunc(notification.UserID, messageBytes)
	} else if s.broadcastAllFunc != nil {
		s.broadcastAllFunc(messageBytes)
	}
}
