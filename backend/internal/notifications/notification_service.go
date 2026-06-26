package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ENTERPRISE NOTIFICATION SERVICE
// ============================================================================
// Features:
// - Multi-channel delivery (email, SMS, push, in-app)
// - Template-based notifications
// - Priority-based routing
// - Delivery tracking & retry
// - User preferences
// - Quiet hours
// - Digest/batching
// - Calendar integration
// ============================================================================

// Channel represents a notification delivery channel
type Channel string

const (
	ChannelEmail   Channel = "EMAIL"
	ChannelSMS     Channel = "SMS"
	ChannelPush    Channel = "PUSH"
	ChannelInApp   Channel = "IN_APP"
	ChannelSlack   Channel = "SLACK"
	ChannelWebhook Channel = "WEBHOOK"
)

// Priority represents notification priority
type Priority string

const (
	PriorityCritical Priority = "CRITICAL"
	PriorityHigh     Priority = "HIGH"
	PriorityMedium   Priority = "MEDIUM"
	PriorityLow      Priority = "LOW"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	TypeOpportunityStageChange NotificationType = "OPPORTUNITY_STAGE_CHANGE"
	TypeTaskAssigned           NotificationType = "TASK_ASSIGNED"
	TypeTaskDueSoon            NotificationType = "TASK_DUE_SOON"
	TypeTaskOverdue            NotificationType = "TASK_OVERDUE"
	TypeCapitalCallNotice      NotificationType = "CAPITAL_CALL_NOTICE"
	TypeCapitalCallDue         NotificationType = "CAPITAL_CALL_DUE"
	TypeDistributionReceived   NotificationType = "DISTRIBUTION_RECEIVED"
	TypeRebalanceTrigger       NotificationType = "REBALANCE_TRIGGER"
	TypeRiskFlag               NotificationType = "RISK_FLAG"
	TypeComplianceDeadline     NotificationType = "COMPLIANCE_DEADLINE"
	TypeDocumentReady          NotificationType = "DOCUMENT_READY"
	TypeSignatureRequired      NotificationType = "SIGNATURE_REQUIRED"
	TypeMeetingReminder        NotificationType = "MEETING_REMINDER"
	TypeQuarterlyReview        NotificationType = "QUARTERLY_REVIEW"
	TypeEscalation             NotificationType = "ESCALATION"
	TypeSLAWarning             NotificationType = "SLA_WARNING"
	TypeSystemAlert            NotificationType = "SYSTEM_ALERT"
)

// Notification represents a notification to be sent
type Notification struct {
	ID             uuid.UUID              `json:"id"`
	Type           NotificationType       `json:"type"`
	Priority       Priority               `json:"priority"`
	RecipientID    uuid.UUID              `json:"recipient_id"`
	RecipientEmail string                 `json:"recipient_email,omitempty"`
	RecipientPhone string                 `json:"recipient_phone,omitempty"`
	Title          string                 `json:"title"`
	Body           string                 `json:"body"`
	Data           map[string]interface{} `json:"data"`
	Channels       []Channel              `json:"channels"`
	EntityType     string                 `json:"entity_type,omitempty"`
	EntityID       uuid.UUID              `json:"entity_id,omitempty"`
	ActionURL      string                 `json:"action_url,omitempty"`
	ScheduledFor   time.Time              `json:"scheduled_for"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	TemplateID     uuid.UUID              `json:"template_id,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// EnterpriseTemplate represents a notification template
type EnterpriseTemplate struct {
	ID               uuid.UUID        `json:"id"`
	Code             string           `json:"code"`
	Name             string           `json:"name"`
	Type             NotificationType `json:"type"`
	DefaultChannels  []Channel        `json:"default_channels"`
	DefaultPriority  Priority         `json:"default_priority"`
	EmailSubjectTmpl string           `json:"email_subject_template"`
	EmailBodyTmpl    string           `json:"email_body_template"`
	SMSTmpl          string           `json:"sms_template"`
	PushTmpl         string           `json:"push_template"`
	InAppTmpl        string           `json:"in_app_template"`
	SlackTmpl        string           `json:"slack_template"`
	IsActive         bool             `json:"is_active"`
}

// UserPreferences represents user notification preferences
type UserPreferences struct {
	UserID              uuid.UUID                           `json:"user_id"`
	EmailEnabled        bool                                `json:"email_enabled"`
	SMSEnabled          bool                                `json:"sms_enabled"`
	PushEnabled         bool                                `json:"push_enabled"`
	InAppEnabled        bool                                `json:"in_app_enabled"`
	QuietHoursEnabled   bool                                `json:"quiet_hours_enabled"`
	QuietHoursStart     string                              `json:"quiet_hours_start"` // "22:00"
	QuietHoursEnd       string                              `json:"quiet_hours_end"`   // "07:00"
	QuietHoursTimezone  string                              `json:"quiet_hours_timezone"`
	TypePreferences     map[NotificationType]TypePreference `json:"type_preferences"`
	DailyDigestEnabled  bool                                `json:"daily_digest_enabled"`
	DailyDigestTime     string                              `json:"daily_digest_time"`
	WeeklyDigestEnabled bool                                `json:"weekly_digest_enabled"`
	WeeklyDigestDay     int                                 `json:"weekly_digest_day"` // 0=Sunday
}

// TypePreference represents preferences for a specific notification type
type TypePreference struct {
	Enabled           bool      `json:"enabled"`
	Channels          []Channel `json:"channels"`
	PriorityThreshold Priority  `json:"priority_threshold"`
}

// DeliveryStatus tracks delivery status per channel
type DeliveryStatus struct {
	Channel       Channel    `json:"channel"`
	Status        string     `json:"status"` // PENDING, SENT, DELIVERED, FAILED, BOUNCED
	SentAt        *time.Time `json:"sent_at,omitempty"`
	DeliveredAt   *time.Time `json:"delivered_at,omitempty"`
	OpenedAt      *time.Time `json:"opened_at,omitempty"`
	ClickedAt     *time.Time `json:"clicked_at,omitempty"`
	FailedAt      *time.Time `json:"failed_at,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	RetryCount    int        `json:"retry_count"`
	ExternalID    string     `json:"external_id,omitempty"` // Provider's message ID
}

// NotificationRecord represents a sent notification with delivery tracking
type NotificationRecord struct {
	Notification
	DeliveryStatus map[Channel]*DeliveryStatus `json:"delivery_status"`
	ReadAt         *time.Time                  `json:"read_at,omitempty"`
	DismissedAt    *time.Time                  `json:"dismissed_at,omitempty"`
	ActionedAt     *time.Time                  `json:"actioned_at,omitempty"`
	ActionTaken    string                      `json:"action_taken,omitempty"`
}

// CalendarEvent represents a calendar event for scheduling
type CalendarEvent struct {
	ID                 uuid.UUID              `json:"id"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	StartTime          time.Time              `json:"start_time"`
	EndTime            time.Time              `json:"end_time"`
	Location           string                 `json:"location,omitempty"`
	VideoConferenceURL string                 `json:"video_conference_url,omitempty"`
	OrganizerID        uuid.UUID              `json:"organizer_id"`
	Attendees          []CalendarAttendee     `json:"attendees"`
	Reminders          []CalendarReminder     `json:"reminders"`
	RelatedEntityType  string                 `json:"related_entity_type,omitempty"`
	RelatedEntityID    uuid.UUID              `json:"related_entity_id,omitempty"`
	ExternalCalendarID string                 `json:"external_calendar_id,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// CalendarAttendee represents an event attendee
type CalendarAttendee struct {
	UserID         uuid.UUID `json:"user_id"`
	Email          string    `json:"email"`
	Name           string    `json:"name"`
	Required       bool      `json:"required"`
	ResponseStatus string    `json:"response_status"` // PENDING, ACCEPTED, DECLINED, TENTATIVE
}

// CalendarReminder represents an event reminder
type CalendarReminder struct {
	MinutesBefore int     `json:"minutes_before"`
	Method        Channel `json:"method"`
}

// EnterpriseNotificationService handles all notification operations
type EnterpriseNotificationService struct {
	templates        map[string]*EnterpriseTemplate
	preferences      map[uuid.UUID]*UserPreferences
	queue            chan *Notification
	providers        map[Channel]DeliveryProvider
	calendarProvider CalendarProvider
	mu               sync.RWMutex
	config           NotificationConfig
}

// NotificationConfig holds service configuration
type NotificationConfig struct {
	QueueSize           int           `json:"queue_size"`
	MaxRetries          int           `json:"max_retries"`
	RetryBackoff        time.Duration `json:"retry_backoff"`
	BatchSize           int           `json:"batch_size"`
	BatchInterval       time.Duration `json:"batch_interval"`
	DefaultExpiryHours  int           `json:"default_expiry_hours"`
	EnableDeduplication bool          `json:"enable_deduplication"`
	DeduplicationWindow time.Duration `json:"deduplication_window"`
}

// DeliveryProvider interface for channel-specific delivery
type DeliveryProvider interface {
	Send(ctx context.Context, notification *Notification) (*DeliveryStatus, error)
	GetDeliveryStatus(ctx context.Context, externalID string) (*DeliveryStatus, error)
	ValidateRecipient(ctx context.Context, recipient string) (bool, error)
}

// CalendarProvider interface for calendar integration
type CalendarProvider interface {
	CreateEvent(ctx context.Context, event *CalendarEvent) (string, error)
	UpdateEvent(ctx context.Context, event *CalendarEvent) error
	DeleteEvent(ctx context.Context, externalID string) error
	GetAvailability(ctx context.Context, userID uuid.UUID, start, end time.Time) ([]TimeSlot, error)
	SendInvites(ctx context.Context, event *CalendarEvent) error
}

// TimeSlot represents an available time slot
type TimeSlot struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	IsBusy bool      `json:"is_busy"`
}

// NewEnterpriseService creates a new notification service
func NewEnterpriseService(config NotificationConfig) *EnterpriseNotificationService {
	if config.QueueSize == 0 {
		config.QueueSize = 10000
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 5 * time.Second
	}
	if config.DefaultExpiryHours == 0 {
		config.DefaultExpiryHours = 168 // 1 week
	}

	return &EnterpriseNotificationService{
		templates:   make(map[string]*EnterpriseTemplate),
		preferences: make(map[uuid.UUID]*UserPreferences),
		queue:       make(chan *Notification, config.QueueSize),
		providers:   make(map[Channel]DeliveryProvider),
		config:      config,
	}
}

// RegisterProvider registers a delivery provider for a channel
func (s *EnterpriseNotificationService) RegisterProvider(channel Channel, provider DeliveryProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[channel] = provider
}

// RegisterCalendarProvider registers the calendar provider
func (s *EnterpriseNotificationService) RegisterCalendarProvider(provider CalendarProvider) {
	s.calendarProvider = provider
}

// LoadTemplate loads a notification template
func (s *EnterpriseNotificationService) LoadTemplate(tmpl *EnterpriseTemplate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.templates[tmpl.Code] = tmpl
}

// LoadUserPreferences loads user preferences
func (s *EnterpriseNotificationService) LoadUserPreferences(prefs *UserPreferences) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preferences[prefs.UserID] = prefs
}

// Send sends a notification
func (s *EnterpriseNotificationService) Send(ctx context.Context, notification *Notification) error {
	// Validate
	if notification.ID == uuid.Nil {
		notification.ID = uuid.New()
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}
	if notification.ScheduledFor.IsZero() {
		notification.ScheduledFor = time.Now()
	}

	// Apply user preferences
	notification = s.applyUserPreferences(notification)
	if len(notification.Channels) == 0 {
		return nil // User has disabled all channels for this type
	}

	// Check quiet hours
	if s.isInQuietHours(notification.RecipientID) && notification.Priority != PriorityCritical {
		// Delay until quiet hours end
		notification.ScheduledFor = s.getQuietHoursEnd(notification.RecipientID)
	}

	// Queue for delivery
	select {
	case s.queue <- notification:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("notification queue full")
	}
}

// SendFromTemplate sends a notification using a template
func (s *EnterpriseNotificationService) SendFromTemplate(
	ctx context.Context,
	templateCode string,
	recipientID uuid.UUID,
	recipientEmail string,
	data map[string]interface{},
) error {
	s.mu.RLock()
	tmpl, exists := s.templates[templateCode]
	s.mu.RUnlock()

	if !exists || !tmpl.IsActive {
		return fmt.Errorf("template not found or inactive: %s", templateCode)
	}

	// Render templates
	title, err := s.renderTemplate(tmpl.EmailSubjectTmpl, data)
	if err != nil {
		return fmt.Errorf("failed to render subject: %w", err)
	}

	body, err := s.renderTemplate(tmpl.EmailBodyTmpl, data)
	if err != nil {
		return fmt.Errorf("failed to render body: %w", err)
	}

	notification := &Notification{
		ID:             uuid.New(),
		Type:           tmpl.Type,
		Priority:       tmpl.DefaultPriority,
		RecipientID:    recipientID,
		RecipientEmail: recipientEmail,
		Title:          title,
		Body:           body,
		Data:           data,
		Channels:       tmpl.DefaultChannels,
		TemplateID:     tmpl.ID,
		CreatedAt:      time.Now(),
		ScheduledFor:   time.Now(),
	}

	return s.Send(ctx, notification)
}

// SendBulk sends notifications to multiple recipients
func (s *EnterpriseNotificationService) SendBulk(ctx context.Context, notifications []*Notification) error {
	for _, n := range notifications {
		if err := s.Send(ctx, n); err != nil {
			// Log error but continue
			continue
		}
	}
	return nil
}

// ProcessQueue processes the notification queue (run in a goroutine)
func (s *EnterpriseNotificationService) ProcessQueue(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-s.queue:
			s.processNotification(ctx, notification)
		}
	}
}

func (s *EnterpriseNotificationService) processNotification(ctx context.Context, notification *Notification) {
	// Check if scheduled for future
	if notification.ScheduledFor.After(time.Now()) {
		// Re-queue (in production, use a proper scheduler)
		time.AfterFunc(time.Until(notification.ScheduledFor), func() {
			s.processNotification(ctx, notification)
		})
		return
	}

	// Check expiry
	if notification.ExpiresAt != nil && notification.ExpiresAt.Before(time.Now()) {
		return
	}

	// Deliver to each channel
	for _, channel := range notification.Channels {
		s.mu.RLock()
		provider, exists := s.providers[channel]
		s.mu.RUnlock()

		if !exists {
			continue
		}

		// Deliver with retry
		s.deliverWithRetry(ctx, provider, notification, channel)
	}
}

func (s *EnterpriseNotificationService) deliverWithRetry(
	ctx context.Context,
	provider DeliveryProvider,
	notification *Notification,
	channel Channel,
) {
	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := s.config.RetryBackoff * time.Duration(attempt)
			time.Sleep(backoff)
		}

		_, err := provider.Send(ctx, notification)
		if err == nil {
			return
		}

		lastErr = err
	}

	// Log final failure
	_ = lastErr
}

func (s *EnterpriseNotificationService) applyUserPreferences(notification *Notification) *Notification {
	s.mu.RLock()
	prefs, exists := s.preferences[notification.RecipientID]
	s.mu.RUnlock()

	if !exists {
		return notification
	}

	// Filter channels based on preferences
	var enabledChannels []Channel
	for _, ch := range notification.Channels {
		switch ch {
		case ChannelEmail:
			if prefs.EmailEnabled {
				enabledChannels = append(enabledChannels, ch)
			}
		case ChannelSMS:
			if prefs.SMSEnabled {
				enabledChannels = append(enabledChannels, ch)
			}
		case ChannelPush:
			if prefs.PushEnabled {
				enabledChannels = append(enabledChannels, ch)
			}
		case ChannelInApp:
			if prefs.InAppEnabled {
				enabledChannels = append(enabledChannels, ch)
			}
		default:
			enabledChannels = append(enabledChannels, ch)
		}
	}

	// Check type-specific preferences
	if typePref, exists := prefs.TypePreferences[notification.Type]; exists {
		if !typePref.Enabled {
			return &Notification{} // Return empty to skip
		}
		if len(typePref.Channels) > 0 {
			enabledChannels = typePref.Channels
		}
		// Check priority threshold
		if !s.meetsPriorityThreshold(notification.Priority, typePref.PriorityThreshold) {
			return &Notification{}
		}
	}

	notification.Channels = enabledChannels
	return notification
}

func (s *EnterpriseNotificationService) isInQuietHours(userID uuid.UUID) bool {
	s.mu.RLock()
	prefs, exists := s.preferences[userID]
	s.mu.RUnlock()

	if !exists || !prefs.QuietHoursEnabled {
		return false
	}

	// Parse quiet hours
	now := time.Now()
	start, _ := time.Parse("15:04", prefs.QuietHoursStart)
	end, _ := time.Parse("15:04", prefs.QuietHoursEnd)

	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := start.Hour()*60 + start.Minute()
	endMinutes := end.Hour()*60 + end.Minute()

	if startMinutes < endMinutes {
		return currentMinutes >= startMinutes && currentMinutes < endMinutes
	}
	// Spans midnight
	return currentMinutes >= startMinutes || currentMinutes < endMinutes
}

func (s *EnterpriseNotificationService) getQuietHoursEnd(userID uuid.UUID) time.Time {
	s.mu.RLock()
	prefs, exists := s.preferences[userID]
	s.mu.RUnlock()

	if !exists {
		return time.Now()
	}

	end, _ := time.Parse("15:04", prefs.QuietHoursEnd)
	now := time.Now()
	result := time.Date(now.Year(), now.Month(), now.Day(), end.Hour(), end.Minute(), 0, 0, now.Location())

	if result.Before(now) {
		result = result.Add(24 * time.Hour)
	}

	return result
}

func (s *EnterpriseNotificationService) meetsPriorityThreshold(actual, threshold Priority) bool {
	priorities := map[Priority]int{
		PriorityCritical: 4,
		PriorityHigh:     3,
		PriorityMedium:   2,
		PriorityLow:      1,
	}
	return priorities[actual] >= priorities[threshold]
}

func (s *EnterpriseNotificationService) renderTemplate(tmplStr string, data map[string]interface{}) (string, error) {
	if tmplStr == "" {
		return "", nil
	}

	tmpl, err := template.New("notification").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// ============================================================================
// CALENDAR INTEGRATION
// ============================================================================

// ScheduleMeeting schedules a meeting with notifications
func (s *EnterpriseNotificationService) ScheduleMeeting(ctx context.Context, event *CalendarEvent) error {
	if s.calendarProvider == nil {
		return fmt.Errorf("calendar provider not configured")
	}

	// Create calendar event
	externalID, err := s.calendarProvider.CreateEvent(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to create calendar event: %w", err)
	}
	event.ExternalCalendarID = externalID

	// Send invitations
	if err := s.calendarProvider.SendInvites(ctx, event); err != nil {
		// Log but don't fail
	}

	// Schedule reminders
	for _, reminder := range event.Reminders {
		reminderTime := event.StartTime.Add(-time.Duration(reminder.MinutesBefore) * time.Minute)

		notification := &Notification{
			ID:           uuid.New(),
			Type:         TypeMeetingReminder,
			Priority:     PriorityMedium,
			RecipientID:  event.OrganizerID,
			Title:        fmt.Sprintf("Meeting Reminder: %s", event.Title),
			Body:         fmt.Sprintf("Your meeting '%s' starts in %d minutes", event.Title, reminder.MinutesBefore),
			Channels:     []Channel{reminder.Method},
			ScheduledFor: reminderTime,
			EntityType:   "calendar_event",
			EntityID:     event.ID,
			Data: map[string]interface{}{
				"event_id":       event.ID.String(),
				"event_title":    event.Title,
				"start_time":     event.StartTime.Format(time.RFC3339),
				"video_url":      event.VideoConferenceURL,
				"minutes_before": reminder.MinutesBefore,
			},
		}

		if err := s.Send(ctx, notification); err != nil {
			continue
		}
	}

	// Send reminders to attendees
	for _, attendee := range event.Attendees {
		for _, reminder := range event.Reminders {
			reminderTime := event.StartTime.Add(-time.Duration(reminder.MinutesBefore) * time.Minute)

			notification := &Notification{
				ID:             uuid.New(),
				Type:           TypeMeetingReminder,
				Priority:       PriorityMedium,
				RecipientID:    attendee.UserID,
				RecipientEmail: attendee.Email,
				Title:          fmt.Sprintf("Meeting Reminder: %s", event.Title),
				Body:           fmt.Sprintf("Meeting '%s' starts in %d minutes", event.Title, reminder.MinutesBefore),
				Channels:       []Channel{reminder.Method},
				ScheduledFor:   reminderTime,
				EntityType:     "calendar_event",
				EntityID:       event.ID,
			}

			if err := s.Send(ctx, notification); err != nil {
				continue
			}
		}
	}

	return nil
}

// FindAvailableSlots finds available meeting slots for a group
func (s *EnterpriseNotificationService) FindAvailableSlots(
	ctx context.Context,
	userIDs []uuid.UUID,
	start, end time.Time,
	duration time.Duration,
) ([]TimeSlot, error) {
	if s.calendarProvider == nil {
		return nil, fmt.Errorf("calendar provider not configured")
	}

	// Get availability for all users
	var allSlots [][]TimeSlot
	for _, userID := range userIDs {
		slots, err := s.calendarProvider.GetAvailability(ctx, userID, start, end)
		if err != nil {
			continue
		}
		allSlots = append(allSlots, slots)
	}

	// Find common available slots
	return findCommonAvailability(allSlots, duration), nil
}

func findCommonAvailability(allSlots [][]TimeSlot, minDuration time.Duration) []TimeSlot {
	if len(allSlots) == 0 {
		return nil
	}

	// Simple implementation: find slots where all users are available
	var common []TimeSlot

	for _, slot := range allSlots[0] {
		if slot.IsBusy {
			continue
		}

		isCommon := true
		for _, userSlots := range allSlots[1:] {
			found := false
			for _, s := range userSlots {
				if !s.IsBusy && s.Start.Equal(slot.Start) && s.End.Equal(slot.End) {
					found = true
					break
				}
			}
			if !found {
				isCommon = false
				break
			}
		}

		if isCommon && slot.End.Sub(slot.Start) >= minDuration {
			common = append(common, slot)
		}
	}

	return common
}

// ============================================================================
// NOTIFICATION BUILDERS
// ============================================================================

// NotificationBuilder provides a fluent interface for building notifications
type NotificationBuilder struct {
	notification *Notification
	service      *EnterpriseNotificationService
}

// NewNotification creates a new notification builder
func (s *EnterpriseNotificationService) NewNotification(notifType NotificationType) *NotificationBuilder {
	return &NotificationBuilder{
		notification: &Notification{
			ID:        uuid.New(),
			Type:      notifType,
			Priority:  PriorityMedium,
			Channels:  []Channel{ChannelInApp, ChannelEmail},
			CreatedAt: time.Now(),
		},
		service: s,
	}
}

// WithPriority sets the priority
func (b *NotificationBuilder) WithPriority(p Priority) *NotificationBuilder {
	b.notification.Priority = p
	return b
}

// WithRecipient sets the recipient
func (b *NotificationBuilder) WithRecipient(userID uuid.UUID, email string) *NotificationBuilder {
	b.notification.RecipientID = userID
	b.notification.RecipientEmail = email
	return b
}

// WithTitle sets the title
func (b *NotificationBuilder) WithTitle(title string) *NotificationBuilder {
	b.notification.Title = title
	return b
}

// WithBody sets the body
func (b *NotificationBuilder) WithBody(body string) *NotificationBuilder {
	b.notification.Body = body
	return b
}

// WithData sets the data payload
func (b *NotificationBuilder) WithData(data map[string]interface{}) *NotificationBuilder {
	b.notification.Data = data
	return b
}

// WithChannels sets the delivery channels
func (b *NotificationBuilder) WithChannels(channels ...Channel) *NotificationBuilder {
	b.notification.Channels = channels
	return b
}

// WithEntity links to a related entity
func (b *NotificationBuilder) WithEntity(entityType string, entityID uuid.UUID) *NotificationBuilder {
	b.notification.EntityType = entityType
	b.notification.EntityID = entityID
	return b
}

// WithActionURL sets the action URL
func (b *NotificationBuilder) WithActionURL(url string) *NotificationBuilder {
	b.notification.ActionURL = url
	return b
}

// ScheduleFor schedules the notification for a future time
func (b *NotificationBuilder) ScheduleFor(t time.Time) *NotificationBuilder {
	b.notification.ScheduledFor = t
	return b
}

// ExpiresAt sets the expiration time
func (b *NotificationBuilder) ExpiresAt(t time.Time) *NotificationBuilder {
	b.notification.ExpiresAt = &t
	return b
}

// Send sends the notification
func (b *NotificationBuilder) Send(ctx context.Context) error {
	return b.service.Send(ctx, b.notification)
}

// Build returns the notification without sending
func (b *NotificationBuilder) Build() *Notification {
	return b.notification
}

// ============================================================================
// PREDEFINED NOTIFICATION HELPERS
// ============================================================================

// NotifyCapitalCall sends a capital call notification
func (s *EnterpriseNotificationService) NotifyCapitalCall(
	ctx context.Context,
	recipientID uuid.UUID,
	recipientEmail string,
	fundName string,
	amount float64,
	dueDate time.Time,
) error {
	return s.NewNotification(TypeCapitalCallNotice).
		WithPriority(PriorityHigh).
		WithRecipient(recipientID, recipientEmail).
		WithTitle(fmt.Sprintf("Capital Call Notice: %s", fundName)).
		WithBody(fmt.Sprintf("A capital call of $%.2f has been issued for %s. Due date: %s",
			amount, fundName, dueDate.Format("January 2, 2006"))).
		WithData(map[string]interface{}{
			"fund_name": fundName,
			"amount":    amount,
			"due_date":  dueDate.Format(time.RFC3339),
		}).
		WithChannels(ChannelEmail, ChannelInApp, ChannelSMS).
		Send(ctx)
}

// NotifyRiskFlag sends a risk flag alert
func (s *EnterpriseNotificationService) NotifyRiskFlag(
	ctx context.Context,
	recipientID uuid.UUID,
	recipientEmail string,
	clientName string,
	flagType string,
	severity string,
	description string,
) error {
	priority := PriorityMedium
	if severity == "CRITICAL" || severity == "HIGH" {
		priority = PriorityHigh
	}

	return s.NewNotification(TypeRiskFlag).
		WithPriority(priority).
		WithRecipient(recipientID, recipientEmail).
		WithTitle(fmt.Sprintf("⚠️ Risk Flag: %s for %s", flagType, clientName)).
		WithBody(description).
		WithData(map[string]interface{}{
			"client_name": clientName,
			"flag_type":   flagType,
			"severity":    severity,
			"description": description,
		}).
		WithChannels(ChannelEmail, ChannelInApp).
		Send(ctx)
}

// NotifyEscalation sends an escalation notification
func (s *EnterpriseNotificationService) NotifyEscalation(
	ctx context.Context,
	recipientID uuid.UUID,
	recipientEmail string,
	opportunityName string,
	reason string,
	originalAssignee string,
) error {
	return s.NewNotification(TypeEscalation).
		WithPriority(PriorityHigh).
		WithRecipient(recipientID, recipientEmail).
		WithTitle(fmt.Sprintf("🔔 Escalation: %s", opportunityName)).
		WithBody(fmt.Sprintf("This item has been escalated to you. Reason: %s. Originally assigned to: %s",
			reason, originalAssignee)).
		WithData(map[string]interface{}{
			"opportunity_name":  opportunityName,
			"reason":            reason,
			"original_assignee": originalAssignee,
		}).
		WithChannels(ChannelEmail, ChannelInApp, ChannelPush).
		Send(ctx)
}

// NotifyTaskDue sends a task due reminder
func (s *EnterpriseNotificationService) NotifyTaskDue(
	ctx context.Context,
	recipientID uuid.UUID,
	recipientEmail string,
	taskTitle string,
	dueDate time.Time,
	isOverdue bool,
) error {
	notifType := TypeTaskDueSoon
	title := fmt.Sprintf("Task Due Soon: %s", taskTitle)
	priority := PriorityMedium

	if isOverdue {
		notifType = TypeTaskOverdue
		title = fmt.Sprintf("⚠️ Task Overdue: %s", taskTitle)
		priority = PriorityHigh
	}

	return s.NewNotification(notifType).
		WithPriority(priority).
		WithRecipient(recipientID, recipientEmail).
		WithTitle(title).
		WithBody(fmt.Sprintf("Task '%s' is due on %s", taskTitle, dueDate.Format("January 2, 2006"))).
		WithData(map[string]interface{}{
			"task_title": taskTitle,
			"due_date":   dueDate.Format(time.RFC3339),
			"is_overdue": isOverdue,
		}).
		Send(ctx)
}

// ============================================================================
// DEFAULT TEMPLATES
// ============================================================================

// GetDefaultTemplates returns default notification templates
func GetDefaultTemplates() []*EnterpriseTemplate {
	return []*EnterpriseTemplate{
		{
			ID:               uuid.New(),
			Code:             "capital_call_notice",
			Name:             "Capital Call Notice",
			Type:             TypeCapitalCallNotice,
			DefaultChannels:  []Channel{ChannelEmail, ChannelInApp, ChannelSMS},
			DefaultPriority:  PriorityHigh,
			EmailSubjectTmpl: "Capital Call Notice: {{.FundName}} - ${{.Amount}}",
			EmailBodyTmpl: `
Dear {{.RecipientName}},

A capital call has been issued for {{.FundName}}.

Amount: ${{.Amount}}
Due Date: {{.DueDate}}

Please ensure sufficient funds are available in your designated funding account.

{{if .Notes}}Notes: {{.Notes}}{{end}}

If you have any questions, please contact your advisor.

Best regards,
The Investment Team
`,
			SMSTmpl:   "Capital Call: {{.FundName}} - ${{.Amount}} due {{.DueDate}}. Log in to view details.",
			InAppTmpl: "Capital Call: {{.FundName}} - ${{.Amount}} due {{.DueDate}}",
			IsActive:  true,
		},
		{
			ID:               uuid.New(),
			Code:             "task_assigned",
			Name:             "Task Assigned",
			Type:             TypeTaskAssigned,
			DefaultChannels:  []Channel{ChannelEmail, ChannelInApp},
			DefaultPriority:  PriorityMedium,
			EmailSubjectTmpl: "New Task Assigned: {{.TaskTitle}}",
			EmailBodyTmpl: `
You have been assigned a new task:

Task: {{.TaskTitle}}
Due Date: {{.DueDate}}
Priority: {{.Priority}}

{{if .Description}}Description: {{.Description}}{{end}}

Click here to view the task: {{.ActionURL}}
`,
			InAppTmpl: "New task: {{.TaskTitle}}",
			IsActive:  true,
		},
		{
			ID:               uuid.New(),
			Code:             "opportunity_stage_change",
			Name:             "Opportunity Stage Change",
			Type:             TypeOpportunityStageChange,
			DefaultChannels:  []Channel{ChannelInApp, ChannelEmail},
			DefaultPriority:  PriorityMedium,
			EmailSubjectTmpl: "{{.OpportunityName}} moved to {{.NewStage}}",
			EmailBodyTmpl: `
The opportunity {{.OpportunityName}} has progressed to a new stage.

Previous Stage: {{.OldStage}}
New Stage: {{.NewStage}}

{{if .Notes}}Notes: {{.Notes}}{{end}}

{{if .RequiresAction}}Action Required: Please review and take necessary action.{{end}}
`,
			InAppTmpl: "{{.OpportunityName}} → {{.NewStage}}",
			IsActive:  true,
		},
	}
}

// SerializeNotification serializes a notification to JSON
func SerializeNotification(n *Notification) ([]byte, error) {
	return json.Marshal(n)
}

// DeserializeNotification deserializes a notification from JSON
func DeserializeNotification(data []byte) (*Notification, error) {
	var n Notification
	if err := json.Unmarshal(data, &n); err != nil {
		return nil, err
	}
	return &n, nil
}
