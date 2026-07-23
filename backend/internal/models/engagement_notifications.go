package models

import (
	"time"
)

// EngagementNotification extends the basic notification with engagement features
type EngagementNotification struct {
	ID          string                 `json:"id" db:"id"`
	UserID      string                 `json:"user_id" db:"user_id"`
	Type        string                 `json:"type" db:"type"` // welcome, feature, recommendation, alert, etc.
	Title       string                 `json:"title" db:"title"`
	Message     string                 `json:"message" db:"message"`
	RichContent map[string]interface{} `json:"rich_content,omitempty" db:"rich_content"`
	Priority    int                    `json:"priority" db:"priority"` // 1=low, 2=normal, 3=high, 4=critical
	Channels    []string               `json:"channels" db:"channels"` // email, sms, push, in_app
	Status      string                 `json:"status" db:"status"`     // draft, scheduled, sent, delivered, read, clicked, dismissed
	ScheduledAt *time.Time             `json:"scheduled_at,omitempty" db:"scheduled_at"`
	SentAt      *time.Time             `json:"sent_at,omitempty" db:"sent_at"`
	ReadAt      *time.Time             `json:"read_at,omitempty" db:"read_at"`
	ClickedAt   *time.Time             `json:"clicked_at,omitempty" db:"clicked_at"`
	DismissedAt *time.Time             `json:"dismissed_at,omitempty" db:"dismissed_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty" db:"expires_at"`
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`

	// Engagement tracking
	EngagementScore float64 `json:"engagement_score,omitempty" db:"engagement_score"`
	UserSegment     string  `json:"user_segment,omitempty" db:"user_segment"`
	ABTestVariant   string  `json:"ab_test_variant,omitempty" db:"ab_test_variant"`

	// Template and personalization
	TemplateID      string                 `json:"template_id,omitempty" db:"template_id"`
	Personalization map[string]interface{} `json:"personalization,omitempty" db:"personalization"`

	// Actions and CTAs
	Actions []NotificationAction `json:"actions,omitempty" db:"actions"`
	CTA     *CallToAction        `json:"cta,omitempty" db:"cta"`
}

// NotificationAction represents interactive elements in notifications
type NotificationAction struct {
	ID      string                 `json:"id" db:"id"`
	Label   string                 `json:"label" db:"label"`
	Type    string                 `json:"type" db:"type"` // button, link, dismiss
	URL     string                 `json:"url,omitempty" db:"url"`
	Payload map[string]interface{} `json:"payload,omitempty" db:"payload"`
	Primary bool                   `json:"primary" db:"primary"`
}

// CallToAction represents the primary action for a notification
type CallToAction struct {
	Text     string `json:"text" db:"text"`
	URL      string `json:"url" db:"url"`
	Type     string `json:"type" db:"type"` // internal, external
	Tracking string `json:"tracking,omitempty" db:"tracking"`
}

// NotificationTemplate for reusable notification templates
type NotificationTemplate struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Type        string                 `json:"type" db:"type"`
	Subject     string                 `json:"subject" db:"subject"`
	Title       string                 `json:"title" db:"title"`
	Message     string                 `json:"message" db:"message"`
	RichContent map[string]interface{} `json:"rich_content,omitempty" db:"rich_content"`
	Variables   []string               `json:"variables" db:"variables"`
	Channels    []string               `json:"channels" db:"channels"`
	CreatedBy   string                 `json:"created_by" db:"created_by"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// UserNotificationPreferences stores user preferences for notifications
type UserNotificationPreferences struct {
	UserID               string            `json:"user_id" db:"user_id"`
	EmailEnabled         bool              `json:"email_enabled" db:"email_enabled"`
	SMSEnabled           bool              `json:"sms_enabled" db:"sms_enabled"`
	PushEnabled          bool              `json:"push_enabled" db:"push_enabled"`
	InAppEnabled         bool              `json:"in_app_enabled" db:"in_app_enabled"`
	QuietHoursStart      *time.Time        `json:"quiet_hours_start,omitempty" db:"quiet_hours_start"`
	QuietHoursEnd        *time.Time        `json:"quiet_hours_end,omitempty" db:"quiet_hours_end"`
	Timezone             string            `json:"timezone" db:"timezone"`
	ChannelPreferences   map[string]bool   `json:"channel_preferences" db:"channel_preferences"`
	TypePreferences      map[string]bool   `json:"type_preferences" db:"type_preferences"`
	FrequencyPreferences map[string]string `json:"frequency_preferences" db:"frequency_preferences"` // immediate, daily, weekly
	CreatedAt            time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at" db:"updated_at"`
}

// NotificationCampaign for automated notification sequences
type NotificationCampaign struct {
	ID          string                     `json:"id" db:"id"`
	Name        string                     `json:"name" db:"name"`
	Description string                     `json:"description" db:"description"`
	Type        string                     `json:"type" db:"type"`     // onboarding, feature_adoption, re_engagement
	Status      string                     `json:"status" db:"status"` // draft, active, paused, completed
	TargetUsers []string                   `json:"target_users" db:"target_users"`
	UserSegment string                     `json:"user_segment" db:"user_segment"`
	Steps       []NotificationCampaignStep `json:"steps" db:"steps"`
	CreatedBy   string                     `json:"created_by" db:"created_by"`
	CreatedAt   time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time                  `json:"updated_at" db:"updated_at"`
}

// NotificationCampaignStep defines a step in a notification campaign
type NotificationCampaignStep struct {
	ID           string  `json:"id" db:"id"`
	StepNumber   int     `json:"step_number" db:"step_number"`
	TemplateID   string  `json:"template_id" db:"template_id"`
	DelayHours   int     `json:"delay_hours" db:"delay_hours"`
	TriggerEvent string  `json:"trigger_event,omitempty" db:"trigger_event"`
	Condition    string  `json:"condition,omitempty" db:"condition"`
	SentCount    int     `json:"sent_count" db:"sent_count"`
	OpenRate     float64 `json:"open_rate" db:"open_rate"`
	ClickRate    float64 `json:"click_rate" db:"click_rate"`
}

// NotificationAnalytics tracks engagement metrics
type NotificationAnalytics struct {
	ID                 string                 `json:"id" db:"id"`
	NotificationID     string                 `json:"notification_id" db:"notification_id"`
	UserID             string                 `json:"user_id" db:"user_id"`
	EventType          string                 `json:"event_type" db:"event_type"` // sent, delivered, opened, clicked, dismissed
	EventTimestamp     time.Time              `json:"event_timestamp" db:"event_timestamp"`
	UserAgent          string                 `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress          string                 `json:"ip_address,omitempty" db:"ip_address"`
	DeviceType         string                 `json:"device_type,omitempty" db:"device_type"`
	Location           string                 `json:"location,omitempty" db:"location"`
	SessionID          string                 `json:"session_id,omitempty" db:"session_id"`
	AdditionalMetadata map[string]interface{} `json:"additional_metadata,omitempty" db:"additional_metadata"`
}

// UserEngagementProfile tracks overall user engagement
type UserEngagementProfile struct {
	UserID                 string    `json:"user_id" db:"user_id"`
	TotalNotifications     int       `json:"total_notifications" db:"total_notifications"`
	OpenedNotifications    int       `json:"opened_notifications" db:"opened_notifications"`
	ClickedNotifications   int       `json:"clicked_notifications" db:"clicked_notifications"`
	DismissedNotifications int       `json:"dismissed_notifications" db:"dismissed_notifications"`
	AvgOpenRate            float64   `json:"avg_open_rate" db:"avg_open_rate"`
	AvgClickRate           float64   `json:"avg_click_rate" db:"avg_click_rate"`
	LastActivity           time.Time `json:"last_activity" db:"last_activity"`
	EngagementScore        float64   `json:"engagement_score" db:"engagement_score"`
	Segment                string    `json:"segment" db:"segment"` // highly_engaged, moderately_engaged, low_engaged, inactive
	PreferredChannels      []string  `json:"preferred_channels" db:"preferred_channels"`
	PreferredTimes         []string  `json:"preferred_times" db:"preferred_times"`
	CreatedAt              time.Time `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time `json:"updated_at" db:"updated_at"`
}
