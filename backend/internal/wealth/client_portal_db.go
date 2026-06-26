package wealth

import (
"context"
"crypto/aes"
"crypto/cipher"
"crypto/rand"
"encoding/base64"
"fmt"
"io"
"time"

"github.com/google/uuid"
)

// ============================================================================
// SECURE MESSAGING - DATABASE OPERATIONS
// ============================================================================

// SaveMessage persists a message to the database
func (s *ClientPortalService) SaveMessage(ctx context.Context, msg *Message) error {
	query := `
		INSERT INTO client_messages (
message_id, thread_id, family_id, sender_id, sender_type,
recipient_id, subject, body_encrypted, encrypted, priority,
read, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	// Encrypt the message body
	encryptedBody, err := s.encryptMessage(msg.Body)
	if err != nil {
		return fmt.Errorf("failed to encrypt message: %w", err)
	}

	_, err = s.db.Exec(ctx, query,
msg.MessageID, msg.ThreadID, msg.FamilyID, msg.SenderID, msg.SenderType,
msg.RecipientID, msg.Subject, encryptedBody, true, msg.Priority,
false, msg.CreatedAt,
)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Save attachments
	for _, att := range msg.Attachments {
		if err := s.saveMessageAttachment(ctx, msg.MessageID, att); err != nil {
			return err
		}
	}

	return nil
}

// saveMessageAttachment saves a message attachment
func (s *ClientPortalService) saveMessageAttachment(ctx context.Context, messageID string, att MessageAttachment) error {
	query := `
		INSERT INTO message_attachments (
attachment_id, message_id, file_name, file_size,
mime_type, storage_path, encrypted
) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.Exec(ctx, query,
att.AttachmentID, messageID, att.FileName, att.FileSize,
att.MimeType, att.StoragePath, att.Encrypted,
)
	return err
}

// GetMessagesForThread retrieves all messages in a thread
func (s *ClientPortalService) GetMessagesForThread(ctx context.Context, threadID string, userID string) ([]Message, error) {
	query := `
		SELECT 
			m.message_id, m.thread_id, m.family_id, m.sender_id, m.sender_type,
			m.recipient_id, m.subject, m.body_encrypted, m.priority, m.read,
			m.read_at, m.created_at
		FROM client_messages m
		WHERE m.thread_id = $1
		AND (m.sender_id = $2 OR m.recipient_id = $2)
		ORDER BY m.created_at ASC
	`

	rows, err := s.db.Query(ctx, query, threadID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	for rows.Next() {
		var msg Message
		var encryptedBody string
		err := rows.Scan(
&msg.MessageID, &msg.ThreadID, &msg.FamilyID, &msg.SenderID, &msg.SenderType,
			&msg.RecipientID, &msg.Subject, &encryptedBody, &msg.Priority, &msg.Read,
			&msg.ReadAt, &msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		// Decrypt the message body
		msg.Body, _ = s.decryptMessage(encryptedBody)
		msg.Encrypted = true

		// Load attachments
		msg.Attachments, _ = s.getMessageAttachments(ctx, msg.MessageID)

		messages = append(messages, msg)
	}

	// Mark messages as read
	if err := s.markMessagesRead(ctx, threadID, userID); err != nil {
		// Log but don't fail
}

return messages, nil
}

// markMessagesRead marks messages in a thread as read for a user
func (s *ClientPortalService) markMessagesRead(ctx context.Context, threadID string, userID string) error {
query := `
UPDATE client_messages 
SET read = true, read_at = $3
WHERE thread_id = $1 AND recipient_id = $2 AND read = false
`
_, err := s.db.Exec(ctx, query, threadID, userID, time.Now())
return err
}

// getMessageAttachments retrieves attachments for a message
func (s *ClientPortalService) getMessageAttachments(ctx context.Context, messageID string) ([]MessageAttachment, error) {
query := `
SELECT attachment_id, file_name, file_size, mime_type, storage_path, encrypted
FROM message_attachments WHERE message_id = $1
`

rows, err := s.db.Query(ctx, query, messageID)
if err != nil {
return nil, err
}
defer rows.Close()

attachments := []MessageAttachment{}
for rows.Next() {
var att MessageAttachment
if err := rows.Scan(
&att.AttachmentID, &att.FileName, &att.FileSize,
&att.MimeType, &att.StoragePath, &att.Encrypted,
); err != nil {
return nil, err
}
attachments = append(attachments, att)
}
return attachments, nil
}

// SendNotification sends a notification to a user
func (s *ClientPortalService) SendNotification(ctx context.Context, recipientID string, notificationType string, title string, body string) error {
query := `
INSERT INTO client_notifications (
notification_id, recipient_id, notification_type, title, body,
read, created_at
) VALUES ($1, $2, $3, $4, $5, false, $6)
`

_, err := s.db.Exec(ctx, query,
uuid.New().String(), recipientID, notificationType, title, body, time.Now(),
)
return err
}

// ============================================================================
// E-SIGNATURE - DATABASE OPERATIONS
// ============================================================================

// SaveSignatureRequest persists a signature request
func (s *ClientPortalService) SaveSignatureRequest(ctx context.Context, req *SignatureRequest) error {
query := `
INSERT INTO signature_requests (
request_id, family_id, document_name, document_type, document_url,
status, expires_at, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

_, err := s.db.Exec(ctx, query,
req.RequestID, req.FamilyID, req.DocumentName, req.DocumentType,
req.DocumentURL, req.Status, req.ExpiresAt, req.CreatedAt,
)
if err != nil {
return fmt.Errorf("failed to save signature request: %w", err)
}

// Save signers
for _, signer := range req.Signers {
if err := s.saveSigner(ctx, req.RequestID, signer); err != nil {
return err
}
}

return nil
}

// saveSigner saves a signer for a signature request
func (s *ClientPortalService) saveSigner(ctx context.Context, requestID string, signer Signer) error {
query := `
INSERT INTO signature_signers (
signer_id, request_id, member_id, name, email,
signing_order, status, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

_, err := s.db.Exec(ctx, query,
signer.SignerID, requestID, signer.MemberID, signer.Name, signer.Email,
signer.SigningOrder, signer.Status, time.Now(),
)
return err
}

// GetSignatureRequest retrieves a signature request
func (s *ClientPortalService) GetSignatureRequest(ctx context.Context, requestID string) (*SignatureRequest, error) {
query := `
SELECT request_id, family_id, document_name, document_type, document_url,
       status, expires_at, created_at, completed_at
FROM signature_requests WHERE request_id = $1
`

var req SignatureRequest
err := s.db.QueryRow(ctx, query, requestID).Scan(
&req.RequestID, &req.FamilyID, &req.DocumentName, &req.DocumentType,
&req.DocumentURL, &req.Status, &req.ExpiresAt, &req.CreatedAt, &req.CompletedAt,
)
if err != nil {
return nil, fmt.Errorf("signature request not found: %w", err)
}

// Load signers
req.Signers, _ = s.getSigners(ctx, requestID)

return &req, nil
}

// getSigners retrieves signers for a request
func (s *ClientPortalService) getSigners(ctx context.Context, requestID string) ([]Signer, error) {
query := `
SELECT signer_id, member_id, name, email, signing_order, status, signed_at, ip_address
FROM signature_signers WHERE request_id = $1 ORDER BY signing_order
`

rows, err := s.db.Query(ctx, query, requestID)
if err != nil {
return nil, err
}
defer rows.Close()

signers := []Signer{}
for rows.Next() {
var signer Signer
if err := rows.Scan(
&signer.SignerID, &signer.MemberID, &signer.Name, &signer.Email,
&signer.SigningOrder, &signer.Status, &signer.SignedAt, &signer.IPAddress,
); err != nil {
return nil, err
}
signers = append(signers, signer)
}
return signers, nil
}

// RecordSignature records a signature in the database
func (s *ClientPortalService) RecordSignature(ctx context.Context, requestID string, signerID string, signatureData string, ipAddress string) error {
// Verify signer exists and is authorized
var currentStatus string
var signingOrder int
checkQuery := `SELECT status, signing_order FROM signature_signers WHERE request_id = $1 AND signer_id = $2`
if err := s.db.QueryRow(ctx, checkQuery, requestID, signerID).Scan(&currentStatus, &signingOrder); err != nil {
return fmt.Errorf("signer not found: %w", err)
}

if currentStatus != "PENDING" {
return fmt.Errorf("signer has already signed or declined")
}

// Check signing order - verify all previous signers have signed
var pendingBefore int
orderQuery := `SELECT COUNT(*) FROM signature_signers WHERE request_id = $1 AND signing_order < $2 AND status != 'SIGNED'`
if err := s.db.QueryRow(ctx, orderQuery, requestID, signingOrder).Scan(&pendingBefore); err != nil {
return err
}
if pendingBefore > 0 {
return fmt.Errorf("waiting for previous signers")
}

// Record the signature
updateQuery := `
UPDATE signature_signers SET
status = 'SIGNED',
signed_at = $3,
ip_address = $4,
signature_data = $5
WHERE request_id = $1 AND signer_id = $2
`
if _, err := s.db.Exec(ctx, updateQuery, requestID, signerID, time.Now(), ipAddress, signatureData); err != nil {
return fmt.Errorf("failed to record signature: %w", err)
}

// Check if all signers have signed
var pendingSigners int
pendingQuery := `SELECT COUNT(*) FROM signature_signers WHERE request_id = $1 AND status = 'PENDING'`
if err := s.db.QueryRow(ctx, pendingQuery, requestID).Scan(&pendingSigners); err != nil {
return err
}

if pendingSigners == 0 {
// All signed - complete the request
completeQuery := `UPDATE signature_requests SET status = 'SIGNED', completed_at = $2 WHERE request_id = $1`
if _, err := s.db.Exec(ctx, completeQuery, requestID, time.Now()); err != nil {
return err
}
} else {
// Notify next signer
nextSignerQuery := `
SELECT name, email FROM signature_signers 
WHERE request_id = $1 AND status = 'PENDING' 
ORDER BY signing_order LIMIT 1
`
var nextName, nextEmail string
if err := s.db.QueryRow(ctx, nextSignerQuery, requestID).Scan(&nextName, &nextEmail); err == nil {
// Send notification to next signer
_ = s.SendNotification(ctx, nextEmail, "SIGNATURE_REQUIRED", 
"Document Ready for Signature", 
fmt.Sprintf("Dear %s, a document is ready for your signature.", nextName))
}
}

return nil
}

// ============================================================================
// MEETING SCHEDULER - DATABASE OPERATIONS
// ============================================================================

// SaveMeeting persists a meeting to the database
func (s *ClientPortalService) SaveMeeting(ctx context.Context, meeting *MeetingSchedule) error {
query := `
INSERT INTO client_meetings (
meeting_id, family_id, advisor_id, meeting_type, title, description,
scheduled_start, scheduled_end, time_zone, video_provider, meeting_url,
status, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
`

_, err := s.db.Exec(ctx, query,
meeting.MeetingID, meeting.FamilyID, meeting.AdvisorID, meeting.MeetingType,
meeting.Title, meeting.Description, meeting.ScheduledStart, meeting.ScheduledEnd,
meeting.TimeZone, meeting.VideoProvider, meeting.MeetingURL, meeting.Status,
meeting.CreatedAt,
)
if err != nil {
return fmt.Errorf("failed to save meeting: %w", err)
}

// Save participants
for _, p := range meeting.Participants {
if err := s.saveMeetingParticipant(ctx, meeting.MeetingID, p); err != nil {
return err
}
}

// Save agenda items
for _, item := range meeting.Agenda {
if err := s.saveMeetingAgendaItem(ctx, meeting.MeetingID, item); err != nil {
return err
}
}

// Save reminders
for _, reminder := range meeting.Reminders {
if err := s.saveMeetingReminder(ctx, meeting.MeetingID, reminder); err != nil {
return err
}
}

return nil
}

// saveMeetingParticipant saves a meeting participant
func (s *ClientPortalService) saveMeetingParticipant(ctx context.Context, meetingID string, p Participant) error {
query := `
INSERT INTO meeting_participants (
participant_id, meeting_id, member_id, name, email, role, status
) VALUES ($1, $2, $3, $4, $5, $6, $7)
`
_, err := s.db.Exec(ctx, query, uuid.New().String(), meetingID, p.MemberID, p.Name, p.Email, p.Role, "NO_RESPONSE")
return err
}

// saveMeetingAgendaItem saves a meeting agenda item
func (s *ClientPortalService) saveMeetingAgendaItem(ctx context.Context, meetingID string, item AgendaItem) error {
query := `
INSERT INTO meeting_agenda_items (
item_id, meeting_id, topic, duration, presenter, description
) VALUES ($1, $2, $3, $4, $5, $6)
`
_, err := s.db.Exec(ctx, query, uuid.New().String(), meetingID, item.Topic, item.Duration, item.Presenter, item.Description)
return err
}

// saveMeetingReminder saves a meeting reminder
func (s *ClientPortalService) saveMeetingReminder(ctx context.Context, meetingID string, reminder ReminderSettings) error {
query := `
INSERT INTO meeting_reminders (
reminder_id, meeting_id, time_before_meeting, reminder_type, sent
) VALUES ($1, $2, $3, $4, $5)
`
_, err := s.db.Exec(ctx, query, reminder.ReminderID, meetingID, reminder.TimeBeforeMeeting, reminder.ReminderType, false)
return err
}

// GetMeeting retrieves a meeting by ID
func (s *ClientPortalService) GetMeeting(ctx context.Context, meetingID string) (*MeetingSchedule, error) {
query := `
SELECT meeting_id, family_id, advisor_id, meeting_type, title, description,
       scheduled_start, scheduled_end, time_zone, video_provider, meeting_url,
       status, created_at
FROM client_meetings WHERE meeting_id = $1
`

var meeting MeetingSchedule
err := s.db.QueryRow(ctx, query, meetingID).Scan(
&meeting.MeetingID, &meeting.FamilyID, &meeting.AdvisorID, &meeting.MeetingType,
&meeting.Title, &meeting.Description, &meeting.ScheduledStart, &meeting.ScheduledEnd,
&meeting.TimeZone, &meeting.VideoProvider, &meeting.MeetingURL, &meeting.Status,
&meeting.CreatedAt,
)
if err != nil {
return nil, fmt.Errorf("meeting not found: %w", err)
}

// Load participants, agenda, reminders
meeting.Participants, _ = s.getMeetingParticipants(ctx, meetingID)
meeting.Agenda, _ = s.getMeetingAgenda(ctx, meetingID)
meeting.Reminders, _ = s.getMeetingReminders(ctx, meetingID)

return &meeting, nil
}

func (s *ClientPortalService) getMeetingParticipants(ctx context.Context, meetingID string) ([]Participant, error) {
query := `SELECT participant_id, member_id, name, email, role, status FROM meeting_participants WHERE meeting_id = $1`
rows, err := s.db.Query(ctx, query, meetingID)
if err != nil {
return nil, err
}
defer rows.Close()

participants := []Participant{}
for rows.Next() {
var p Participant
if err := rows.Scan(&p.ParticipantID, &p.MemberID, &p.Name, &p.Email, &p.Role, &p.Status); err != nil {
return nil, err
}
participants = append(participants, p)
}
return participants, nil
}

func (s *ClientPortalService) getMeetingAgenda(ctx context.Context, meetingID string) ([]AgendaItem, error) {
query := `SELECT item_id, topic, duration, presenter, description FROM meeting_agenda_items WHERE meeting_id = $1`
rows, err := s.db.Query(ctx, query, meetingID)
if err != nil {
return nil, err
}
defer rows.Close()

agenda := []AgendaItem{}
for rows.Next() {
var item AgendaItem
if err := rows.Scan(&item.ItemID, &item.Topic, &item.Duration, &item.Presenter, &item.Description); err != nil {
return nil, err
}
agenda = append(agenda, item)
}
return agenda, nil
}

func (s *ClientPortalService) getMeetingReminders(ctx context.Context, meetingID string) ([]ReminderSettings, error) {
query := `SELECT reminder_id, time_before_meeting, reminder_type, sent FROM meeting_reminders WHERE meeting_id = $1`
rows, err := s.db.Query(ctx, query, meetingID)
if err != nil {
return nil, err
}
defer rows.Close()

reminders := []ReminderSettings{}
for rows.Next() {
var r ReminderSettings
if err := rows.Scan(&r.ReminderID, &r.TimeBeforeMeeting, &r.ReminderType, &r.Sent); err != nil {
return nil, err
}
reminders = append(reminders, r)
}
return reminders, nil
}

// UpdateMeetingStatus updates a meeting's status
func (s *ClientPortalService) UpdateMeetingStatus(ctx context.Context, meetingID string, status string) error {
	query := `UPDATE client_meetings SET status = $2 WHERE meeting_id = $1`
	_, err := s.db.Exec(ctx, query, meetingID, status)
	return err
}

// ============================================================================
// ACTIVITY FEED - DATABASE OPERATIONS
// ============================================================================

// RecordActivity records an activity event
func (s *ClientPortalService) RecordActivity(ctx context.Context, event ActivityEvent) error {
	query := `
		INSERT INTO activity_events (
event_id, family_id, event_type, title, description,
actor_id, actor_name, metadata, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := s.db.Exec(ctx, query,
event.EventID, event.FamilyID, event.EventType, event.Title,
event.Description, event.ActorID, event.ActorName, event.Metadata,
event.CreatedAt,
)
	return err
}

// GetActivityFeedFromDB retrieves activity feed from database
func (s *ClientPortalService) GetActivityFeedFromDB(ctx context.Context, familyID string, limit int) ([]ActivityEvent, error) {
	query := `
		SELECT event_id, family_id, event_type, title, description,
		       actor_id, actor_name, metadata, created_at
		FROM activity_events
		WHERE family_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(ctx, query, familyID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	activities := []ActivityEvent{}
	for rows.Next() {
		var event ActivityEvent
		if err := rows.Scan(
&event.EventID, &event.FamilyID, &event.EventType, &event.Title,
			&event.Description, &event.ActorID, &event.ActorName, &event.Metadata,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		activities = append(activities, event)
	}
	return activities, nil
}

// ============================================================================
// NOTIFICATION PREFERENCES - DATABASE OPERATIONS
// ============================================================================

// GetNotificationPreferences retrieves notification preferences
func (s *ClientPortalService) GetNotificationPreferences(ctx context.Context, familyID string, memberID string) (*NotificationPreferences, error) {
	query := `
		SELECT family_id, member_id, email_enabled, sms_enabled, push_enabled,
		       event_preferences, quiet_hours_enabled, quiet_hours_start,
		       quiet_hours_end, quiet_hours_timezone, updated_at
		FROM notification_preferences
		WHERE family_id = $1 AND member_id = $2
	`

	var prefs NotificationPreferences
	var quietEnabled bool
	var quietStart, quietEnd, quietTZ *string

	err := s.db.QueryRow(ctx, query, familyID, memberID).Scan(
&prefs.FamilyID, &prefs.MemberID, &prefs.EmailEnabled, &prefs.SMSEnabled,
		&prefs.PushEnabled, &prefs.EventPreferences, &quietEnabled, &quietStart,
		&quietEnd, &quietTZ, &prefs.UpdatedAt,
	)
	if err != nil {
		// Return defaults if not found
		return &NotificationPreferences{
			FamilyID:     familyID,
			MemberID:     memberID,
			EmailEnabled: true,
			SMSEnabled:   false,
			PushEnabled:  true,
			EventPreferences: map[string]bool{
				"DOCUMENT_UPLOADED":   true,
				"MESSAGE_RECEIVED":    true,
				"MEETING_SCHEDULED":   true,
				"SIGNATURE_REQUIRED":  true,
				"PORTFOLIO_ALERT":     true,
			},
			UpdatedAt: time.Now(),
		}, nil
	}

	if quietEnabled {
		prefs.QuietHours = &QuietHoursSettings{
			Enabled:   true,
			StartTime: *quietStart,
			EndTime:   *quietEnd,
			TimeZone:  *quietTZ,
		}
	}

	return &prefs, nil
}

// SaveNotificationPreferences saves notification preferences
func (s *ClientPortalService) SaveNotificationPreferences(ctx context.Context, prefs NotificationPreferences) error {
	query := `
		INSERT INTO notification_preferences (
family_id, member_id, email_enabled, sms_enabled, push_enabled,
event_preferences, quiet_hours_enabled, quiet_hours_start,
quiet_hours_end, quiet_hours_timezone, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (family_id, member_id) DO UPDATE SET
			email_enabled = EXCLUDED.email_enabled,
			sms_enabled = EXCLUDED.sms_enabled,
			push_enabled = EXCLUDED.push_enabled,
			event_preferences = EXCLUDED.event_preferences,
			quiet_hours_enabled = EXCLUDED.quiet_hours_enabled,
			quiet_hours_start = EXCLUDED.quiet_hours_start,
			quiet_hours_end = EXCLUDED.quiet_hours_end,
			quiet_hours_timezone = EXCLUDED.quiet_hours_timezone,
			updated_at = EXCLUDED.updated_at
	`

	var quietEnabled bool
	var quietStart, quietEnd, quietTZ *string
	if prefs.QuietHours != nil {
		quietEnabled = prefs.QuietHours.Enabled
		quietStart = &prefs.QuietHours.StartTime
		quietEnd = &prefs.QuietHours.EndTime
		quietTZ = &prefs.QuietHours.TimeZone
	}

	_, err := s.db.Exec(ctx, query,
prefs.FamilyID, prefs.MemberID, prefs.EmailEnabled, prefs.SMSEnabled,
prefs.PushEnabled, prefs.EventPreferences, quietEnabled, quietStart,
quietEnd, quietTZ, time.Now(),
	)
	return err
}

// ============================================================================
// ENCRYPTION HELPERS
// ============================================================================

// encryptionKey should be loaded from secure configuration
var encryptionKey = []byte("32-byte-key-for-aes-256-encrypt!") // Replace with secure key management

// encryptMessage encrypts a message using AES-256-GCM
func (s *ClientPortalService) encryptMessage(plaintext string) (string, error) {
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptMessage decrypts a message using AES-256-GCM
func (s *ClientPortalService) decryptMessage(encrypted string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
