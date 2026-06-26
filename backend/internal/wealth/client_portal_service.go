package wealth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ClientPortalService handles client communication and portal features
type ClientPortalService struct {
	db *pgxpool.Pool
}

// NewClientPortalService creates a new client portal service
func NewClientPortalService(db *pgxpool.Pool) *ClientPortalService {
	return &ClientPortalService{
		db: db,
	}
}

// ============================================================================
// SECURE MESSAGING
// ============================================================================

// Message represents a secure message between client and advisor
type Message struct {
	MessageID   string              `json:"message_id"`
	ThreadID    string              `json:"thread_id"`
	FamilyID    string              `json:"family_id"`
	SenderID    string              `json:"sender_id"`
	SenderType  string              `json:"sender_type"` // CLIENT, ADVISOR, SYSTEM
	RecipientID string              `json:"recipient_id"`
	Subject     string              `json:"subject"`
	Body        string              `json:"body"`
	Encrypted   bool                `json:"encrypted"`
	Read        bool                `json:"read"`
	ReadAt      *time.Time          `json:"read_at,omitempty"`
	Priority    string              `json:"priority"` // LOW, NORMAL, HIGH, URGENT
	Attachments []MessageAttachment `json:"attachments"`
	CreatedAt   time.Time           `json:"created_at"`
}

// MessageAttachment represents a file attached to a message
type MessageAttachment struct {
	AttachmentID string `json:"attachment_id"`
	FileName     string `json:"file_name"`
	FileSize     int64  `json:"file_size"`
	MimeType     string `json:"mime_type"`
	StoragePath  string `json:"storage_path"`
	Encrypted    bool   `json:"encrypted"`
}

// SendMessage sends a secure message
func (s *ClientPortalService) SendMessage(
	ctx context.Context,
	familyID string,
	senderID string,
	senderType string,
	recipientID string,
	subject string,
	body string,
	priority string,
	attachments []MessageAttachment,
) (*Message, error) {
	message := &Message{
		MessageID:   uuid.New().String(),
		ThreadID:    uuid.New().String(), // New thread
		FamilyID:    familyID,
		SenderID:    senderID,
		SenderType:  senderType,
		RecipientID: recipientID,
		Subject:     subject,
		Body:        body,
		Encrypted:   true, // Always encrypt
		Read:        false,
		Priority:    priority,
		Attachments: attachments,
		CreatedAt:   time.Now(),
	}

	// Persist to database
	if err := s.SaveMessage(ctx, message); err != nil {
		return nil, err
	}

	// Send notification to recipient
	_ = s.SendNotification(ctx, recipientID, "MESSAGE_RECEIVED",
		"New Message: "+subject,
		fmt.Sprintf("You have received a new message from your advisor"))

	return message, nil
}

// GetMessageThread retrieves all messages in a thread
func (s *ClientPortalService) GetMessageThread(
	ctx context.Context,
	threadID string,
	userID string,
) ([]Message, error) {
	return s.GetMessagesForThread(ctx, threadID, userID)
}

// ============================================================================
// E-SIGNATURE WORKFLOW
// ============================================================================

// SignatureRequest represents a document requiring signature
type SignatureRequest struct {
	RequestID    string     `json:"request_id"`
	FamilyID     string     `json:"family_id"`
	DocumentName string     `json:"document_name"`
	DocumentType string     `json:"document_type"` // IPS, ACCOUNT_AGREEMENT, AMENDMENT, etc.
	DocumentURL  string     `json:"document_url"`
	Status       string     `json:"status"` // PENDING, SIGNED, REJECTED, EXPIRED
	Signers      []Signer   `json:"signers"`
	ExpiresAt    time.Time  `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// Signer represents a person who needs to sign
type Signer struct {
	SignerID      string     `json:"signer_id"`
	MemberID      string     `json:"member_id"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	SigningOrder  int        `json:"signing_order"` // 1, 2, 3...
	Status        string     `json:"status"`        // PENDING, SIGNED, DECLINED
	SignedAt      *time.Time `json:"signed_at,omitempty"`
	IPAddress     string     `json:"ip_address,omitempty"`
	SignatureData string     `json:"signature_data,omitempty"` // Base64 signature image
}

// CreateSignatureRequest creates a new e-signature request
func (s *ClientPortalService) CreateSignatureRequest(
	ctx context.Context,
	familyID string,
	documentName string,
	documentType string,
	documentURL string,
	signers []Signer,
	expirationDays int,
) (*SignatureRequest, error) {
	request := &SignatureRequest{
		RequestID:    uuid.New().String(),
		FamilyID:     familyID,
		DocumentName: documentName,
		DocumentType: documentType,
		DocumentURL:  documentURL,
		Status:       "PENDING",
		Signers:      signers,
		ExpiresAt:    time.Now().AddDate(0, 0, expirationDays),
		CreatedAt:    time.Now(),
	}

	// Set all signers to pending
	for i := range request.Signers {
		request.Signers[i].SignerID = uuid.New().String()
		request.Signers[i].Status = "PENDING"
	}

	// Persist to database
	if err := s.SaveSignatureRequest(ctx, request); err != nil {
		return nil, err
	}

	// Send email notifications to signers (first in signing order)
	for _, signer := range request.Signers {
		if signer.SigningOrder == 1 {
			_ = s.SendNotification(ctx, signer.MemberID, "SIGNATURE_REQUIRED",
				"Signature Required: "+documentName,
				fmt.Sprintf("Please sign document: %s", documentName))
		}
	}

	return request, nil
}

// SignDocument records a signature
func (s *ClientPortalService) SignDocument(
	ctx context.Context,
	requestID string,
	signerID string,
	signatureData string,
	ipAddress string,
) error {
	// Get the signature request to validate
	request, err := s.GetSignatureRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("failed to get signature request: %w", err)
	}

	// Check if request is still valid
	if request.Status != "PENDING" {
		return fmt.Errorf("signature request is not pending, current status: %s", request.Status)
	}

	if time.Now().After(request.ExpiresAt) {
		return fmt.Errorf("signature request has expired")
	}

	// Find the signer and validate
	var signer *Signer
	var signerIndex int
	for i := range request.Signers {
		if request.Signers[i].SignerID == signerID {
			signer = &request.Signers[i]
			signerIndex = i
			break
		}
	}
	if signer == nil {
		return fmt.Errorf("signer not found: %s", signerID)
	}

	if signer.Status == "SIGNED" {
		return fmt.Errorf("signer has already signed")
	}

	// Check signing order - all previous signers must have signed
	for _, s := range request.Signers {
		if s.SigningOrder < signer.SigningOrder && s.Status != "SIGNED" {
			return fmt.Errorf("previous signer (order %d) must sign first", s.SigningOrder)
		}
	}

	// Record signature with timestamp
	now := time.Now()
	request.Signers[signerIndex].Status = "SIGNED"
	request.Signers[signerIndex].SignedAt = &now
	request.Signers[signerIndex].IPAddress = ipAddress
	request.Signers[signerIndex].SignatureData = signatureData

	// Persist signature
	if err := s.RecordSignature(ctx, requestID, signerID, signatureData, ipAddress); err != nil {
		return fmt.Errorf("failed to record signature: %w", err)
	}

	// Check if all signers have signed
	allSigned := true
	nextSignerOrder := 0
	for _, s := range request.Signers {
		if s.Status != "SIGNED" {
			allSigned = false
			if nextSignerOrder == 0 || s.SigningOrder < nextSignerOrder {
				nextSignerOrder = s.SigningOrder
			}
		}
	}

	if allSigned {
		// Finalize document - update signature request status
		_, err := s.db.Exec(ctx,
			"UPDATE signature_requests SET status = 'SIGNED', completed_at = NOW() WHERE request_id = $1",
			requestID)
		if err != nil {
			// Log but don't fail
		}
		// Record activity
		_ = s.RecordActivity(ctx, ActivityEvent{
			EventID:     uuid.New().String(),
			FamilyID:    request.FamilyID,
			EventType:   "DOCUMENT_SIGNED",
			Title:       "Document Fully Signed",
			Description: fmt.Sprintf("All signers have signed: %s", request.DocumentName),
			Metadata:    map[string]interface{}{"request_id": requestID, "document_name": request.DocumentName},
			CreatedAt:   now,
		})
	} else {
		// Notify next signer
		for _, sig := range request.Signers {
			if sig.SigningOrder == nextSignerOrder {
				_ = s.SendNotification(ctx, sig.MemberID, "SIGNATURE_REQUIRED",
					"Signature Required: "+request.DocumentName,
					"It's your turn to sign the document")
				break
			}
		}
	}

	return nil
}

// ============================================================================
// VIDEO MEETING SCHEDULER
// ============================================================================

// MeetingSchedule represents a scheduled video meeting
type MeetingSchedule struct {
	MeetingID      string             `json:"meeting_id"`
	FamilyID       string             `json:"family_id"`
	AdvisorID      string             `json:"advisor_id"`
	MeetingType    string             `json:"meeting_type"` // QUARTERLY_REVIEW, ANNUAL_PLANNING, AD_HOC
	Title          string             `json:"title"`
	Description    string             `json:"description"`
	ScheduledStart time.Time          `json:"scheduled_start"`
	ScheduledEnd   time.Time          `json:"scheduled_end"`
	TimeZone       string             `json:"time_zone"`
	VideoProvider  string             `json:"video_provider"` // ZOOM, TEAMS, GOOGLE_MEET
	MeetingURL     string             `json:"meeting_url"`
	Participants   []Participant      `json:"participants"`
	Agenda         []AgendaItem       `json:"agenda"`
	Status         string             `json:"status"` // SCHEDULED, COMPLETED, CANCELLED, RESCHEDULED
	Reminders      []ReminderSettings `json:"reminders"`
	CreatedAt      time.Time          `json:"created_at"`
}

// Participant represents a meeting participant
type Participant struct {
	ParticipantID string `json:"participant_id"`
	MemberID      string `json:"member_id"`
	Name          string `json:"name"`
	Email         string `json:"email"`
	Role          string `json:"role"`   // ATTENDEE, ORGANIZER
	Status        string `json:"status"` // ACCEPTED, TENTATIVE, DECLINED, NO_RESPONSE
}

// AgendaItem represents a meeting agenda item
type AgendaItem struct {
	ItemID      string `json:"item_id"`
	Topic       string `json:"topic"`
	Duration    int    `json:"duration"` // Minutes
	Presenter   string `json:"presenter"`
	Description string `json:"description"`
}

// ReminderSettings represents meeting reminder configuration
type ReminderSettings struct {
	ReminderID        string `json:"reminder_id"`
	TimeBeforeMeeting int    `json:"time_before_meeting"` // Minutes
	ReminderType      string `json:"reminder_type"`       // EMAIL, SMS, PUSH
	Sent              bool   `json:"sent"`
}

// ScheduleMeeting schedules a new video meeting
func (s *ClientPortalService) ScheduleMeeting(
	ctx context.Context,
	familyID string,
	advisorID string,
	meetingType string,
	title string,
	scheduledStart time.Time,
	durationMinutes int,
	participants []Participant,
	agenda []AgendaItem,
) (*MeetingSchedule, error) {
	meeting := &MeetingSchedule{
		MeetingID:      uuid.New().String(),
		FamilyID:       familyID,
		AdvisorID:      advisorID,
		MeetingType:    meetingType,
		Title:          title,
		ScheduledStart: scheduledStart,
		ScheduledEnd:   scheduledStart.Add(time.Duration(durationMinutes) * time.Minute),
		TimeZone:       "America/New_York",
		VideoProvider:  "ZOOM", // Default to Zoom
		Participants:   participants,
		Agenda:         agenda,
		Status:         "SCHEDULED",
		Reminders: []ReminderSettings{
			{
				ReminderID:        uuid.New().String(),
				TimeBeforeMeeting: 1440, // 24 hours
				ReminderType:      "EMAIL",
				Sent:              false,
			},
			{
				ReminderID:        uuid.New().String(),
				TimeBeforeMeeting: 60, // 1 hour
				ReminderType:      "EMAIL",
				Sent:              false,
			},
		},
		CreatedAt: time.Now(),
	}

	// Generate meeting URL (placeholder for actual Zoom/Teams API integration)
	meeting.MeetingURL = "https://zoom.us/j/" + meeting.MeetingID

	// Persist to database
	if err := s.SaveMeeting(ctx, meeting); err != nil {
		return nil, fmt.Errorf("failed to save meeting: %w", err)
	}

	// Send notifications to participants
	for _, p := range participants {
		_ = s.SendNotification(ctx, p.MemberID, "MEETING_SCHEDULED",
			"Meeting Scheduled: "+title,
			fmt.Sprintf("Meeting scheduled for %s", scheduledStart.Format("Mon, Jan 2 at 3:04 PM")))
	}

	return meeting, nil
}

// CancelMeeting cancels a scheduled meeting
func (s *ClientPortalService) CancelMeeting(
	ctx context.Context,
	meetingID string,
	reason string,
) error {
	// Get the meeting to get participant info
	meeting, err := s.GetMeeting(ctx, meetingID)
	if err != nil {
		return fmt.Errorf("failed to get meeting: %w", err)
	}

	// Update meeting status to cancelled
	if err := s.UpdateMeetingStatus(ctx, meetingID, "CANCELLED"); err != nil {
		return fmt.Errorf("failed to update meeting status: %w", err)
	}

	// Send cancellation notifications to participants
	for _, p := range meeting.Participants {
		_ = s.SendNotification(ctx, p.MemberID, "MEETING_CANCELLED",
			"Meeting Cancelled: "+meeting.Title,
			fmt.Sprintf("The meeting has been cancelled. Reason: %s", reason))
	}

	// Record activity
	_ = s.RecordActivity(ctx, ActivityEvent{
		EventID:     uuid.New().String(),
		FamilyID:    meeting.FamilyID,
		EventType:   "MEETING_CANCELLED",
		Title:       "Meeting Cancelled",
		Description: fmt.Sprintf("Meeting '%s' was cancelled: %s", meeting.Title, reason),
		Metadata:    map[string]interface{}{"meeting_id": meetingID, "reason": reason},
		CreatedAt:   time.Now(),
	})

	return nil
}

// ============================================================================
// ACTIVITY FEED
// ============================================================================

// ActivityEvent represents an activity in the client portal
type ActivityEvent struct {
	EventID     string                 `json:"event_id"`
	FamilyID    string                 `json:"family_id"`
	EventType   string                 `json:"event_type"` // DOCUMENT_UPLOADED, MESSAGE_SENT, MEETING_SCHEDULED, etc.
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	ActorID     string                 `json:"actor_id"`
	ActorName   string                 `json:"actor_name"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
}

// GetActivityFeed retrieves recent activity for a family
func (s *ClientPortalService) GetActivityFeed(
	ctx context.Context,
	familyID string,
	limit int,
) ([]ActivityEvent, error) {
	return s.GetActivityFeedFromDB(ctx, familyID, limit)
}

// ============================================================================
// NOTIFICATION PREFERENCES
// ============================================================================

// NotificationPreferences represents client notification settings
type NotificationPreferences struct {
	FamilyID         string              `json:"family_id"`
	MemberID         string              `json:"member_id"`
	EmailEnabled     bool                `json:"email_enabled"`
	SMSEnabled       bool                `json:"sms_enabled"`
	PushEnabled      bool                `json:"push_enabled"`
	EventPreferences map[string]bool     `json:"event_preferences"` // Event type -> enabled
	QuietHours       *QuietHoursSettings `json:"quiet_hours,omitempty"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

// QuietHoursSettings defines when not to send notifications
type QuietHoursSettings struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	TimeZone  string `json:"time_zone"`
}

// UpdateNotificationPreferences updates client notification settings
func (s *ClientPortalService) UpdateNotificationPreferences(
	ctx context.Context,
	familyID string,
	memberID string,
	prefs NotificationPreferences,
) error {
	// Set IDs if not provided
	prefs.FamilyID = familyID
	prefs.MemberID = memberID
	prefs.UpdatedAt = time.Now()

	return s.SaveNotificationPreferences(ctx, prefs)
}
