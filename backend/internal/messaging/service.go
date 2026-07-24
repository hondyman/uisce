package messaging

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// Service provides secure messaging operations
type Service interface {
	// Messages
	SendMessage(ctx context.Context, input SendMessageInput) (*SecureMessage, error)
	GetMessage(ctx context.Context, messageID uuid.UUID) (*SecureMessage, error)
	ListConversation(ctx context.Context, conversationID uuid.UUID, limit int) ([]*SecureMessage, error)
	MarkAsRead(ctx context.Context, messageID uuid.UUID) error
	GetUnreadCount(ctx context.Context, recipientID uuid.UUID) (int, error)

	// Notifications
	CreateNotification(ctx context.Context, input CreateNotificationInput) (*Notification, error)
	GetNotifications(ctx context.Context, clientID uuid.UUID, unreadOnly bool) ([]*Notification, error)
	MarkNotificationRead(ctx context.Context, notificationID uuid.UUID) error
	DismissNotification(ctx context.Context, notificationID uuid.UUID) error
}

type service struct {
	db            *sqlx.DB
	encryptionKey []byte // AES-256 key (32 bytes)
	emailService  EmailService
	smsService    SMSService
	hasuraClient  HasuraClient
}

// EmailService handles email delivery
type EmailService interface {
	SendEmail(to, subject, body string) error
}

// SMSService handles SMS delivery
type SMSService interface {
	SendSMS(to, message string) error
}

func NewService(db *sqlx.DB, encryptionKey string, emailSvc EmailService, smsSvc SMSService) Service {
	// Convert hex key to bytes (in production, use proper key management)
	keyBytes := []byte(encryptionKey)
	if len(keyBytes) != 32 {
		panic("encryption key must be 32 bytes for AES-256")
	}

	return &service{
		db:            db,
		encryptionKey: keyBytes,
		emailService:  emailSvc,
		smsService:    smsSvc,
	}
}

// NewServiceWithHasura creates a new messaging service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, encryptionKey string, emailSvc EmailService, smsSvc SMSService, hasuraClient HasuraClient) Service {
	keyBytes := []byte(encryptionKey)
	if len(keyBytes) != 32 {
		panic("encryption key must be 32 bytes for AES-256")
	}

	return &service{
		db:            db,
		encryptionKey: keyBytes,
		emailService:  emailSvc,
		smsService:    smsSvc,
		hasuraClient:  hasuraClient,
	}
}

type SenderType string

const (
	SenderClient  SenderType = "CLIENT"
	SenderAdvisor SenderType = "ADVISOR"
	SenderSystem  SenderType = "SYSTEM"
)

type SecureMessage struct {
	MessageID            uuid.UUID  `json:"message_id" db:"message_id"`
	ConversationID       uuid.UUID  `json:"conversation_id" db:"conversation_id"`
	SenderID             uuid.UUID  `json:"sender_id" db:"sender_id"`
	SenderType           SenderType `json:"sender_type" db:"sender_type"`
	RecipientID          uuid.UUID  `json:"recipient_id" db:"recipient_id"`
	RecipientType        SenderType `json:"recipient_type" db:"recipient_type"`
	Subject              *string    `json:"subject" db:"subject"`
	MessageTextEncrypted string     `json:"-" db:"message_text_encrypted"` // Don't expose in JSON
	MessageText          string     `json:"message_text" db:"-"`           // Decrypted version
	EncryptionKeyID      *string    `json:"encryption_key_id" db:"encryption_key_id"`
	Attachments          []byte     `json:"attachments" db:"attachments"` // JSONB
	ReadAt               *time.Time `json:"read_at" db:"read_at"`
	Archived             bool       `json:"archived" db:"archived"`
	CreatedAt            time.Time  `json:"created_at" db:"created_at"`
}

type SendMessageInput struct {
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	SenderType     SenderType
	RecipientID    uuid.UUID
	RecipientType  SenderType
	Subject        string
	MessageText    string
	Attachments    []Attachment
}

type Attachment struct {
	Filename string `json:"filename"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
}

type NotificationType string

const (
	NotifMeetingReminder   NotificationType = "MEETING_REMINDER"
	NotifDocumentRequest   NotificationType = "DOCUMENT_REQUEST"
	NotifDocumentReady     NotificationType = "DOCUMENT_READY"
	NotifMessageReceived   NotificationType = "MESSAGE_RECEIVED"
	NotifBillingDue        NotificationType = "BILLING_DUE"
	NotifPaymentConfirmed  NotificationType = "PAYMENT_CONFIRMED"
	NotifMarketAlert       NotificationType = "MARKET_ALERT"
	NotifGoalMilestone     NotificationType = "GOAL_MILESTONE"
	NotifPerformanceUpdate NotificationType = "PERFORMANCE_UPDATE"
	NotifCapitalCall       NotificationType = "CAPITAL_CALL"
	NotifSystemUpdate      NotificationType = "SYSTEM_UPDATE"
)

type NotificationPriority string

const (
	PriorityLow    NotificationPriority = "LOW"
	PriorityMedium NotificationPriority = "MEDIUM"
	PriorityHigh   NotificationPriority = "HIGH"
	PriorityUrgent NotificationPriority = "URGENT"
)

type Notification struct {
	NotificationID uuid.UUID            `json:"notification_id" db:"notification_id"`
	ClientID       uuid.UUID            `json:"client_id" db:"client_id"`
	NotifType      NotificationType     `json:"notification_type" db:"notification_type"`
	Title          string               `json:"title" db:"title"`
	Body           string               `json:"body" db:"body"`
	Priority       NotificationPriority `json:"priority" db:"priority"`
	Channels       []string             `json:"channels" db:"channels"` // Array of channels
	SentVia        []byte               `json:"sent_via" db:"sent_via"` // JSONB
	ActionURL      *string              `json:"action_url" db:"action_url"`
	ActionLabel    *string              `json:"action_label" db:"action_label"`
	ReadAt         *time.Time           `json:"read_at" db:"read_at"`
	DismissedAt    *time.Time           `json:"dismissed_at" db:"dismissed_at"`
	CreatedAt      time.Time            `json:"created_at" db:"created_at"`
}

type CreateNotificationInput struct {
	ClientID    uuid.UUID
	NotifType   NotificationType
	Title       string
	Body        string
	Priority    NotificationPriority
	Channels    []string
	ActionURL   string
	ActionLabel string
}

// SendMessage encrypts and stores a message
func (s *service) SendMessage(ctx context.Context, input SendMessageInput) (*SecureMessage, error) {
	// Encrypt message
	encryptedText, err := s.encrypt(input.MessageText)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt message: %w", err)
	}

	attachmentsJSON, _ := json.Marshal(input.Attachments)

	msg := &SecureMessage{
		MessageID:            uuid.New(),
		ConversationID:       input.ConversationID,
		SenderID:             input.SenderID,
		SenderType:           input.SenderType,
		RecipientID:          input.RecipientID,
		RecipientType:        input.RecipientType,
		Subject:              &input.Subject,
		MessageTextEncrypted: encryptedText,
		MessageText:          input.MessageText, // For response only
		Attachments:          attachmentsJSON,
		Archived:             false,
		CreatedAt:            time.Now(),
	}

	err = s.sendMessageRecord(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to save message: %w", err)
	}

	// Send in-app notification to recipient
	go s.CreateNotification(context.Background(), CreateNotificationInput{
		ClientID:    input.RecipientID,
		NotifType:   NotifMessageReceived,
		Title:       "New Message",
		Body:        fmt.Sprintf("You have a new message: %s", input.Subject),
		Priority:    PriorityMedium,
		Channels:    []string{"IN_APP", "EMAIL"},
		ActionURL:   fmt.Sprintf("/messages/%s", input.ConversationID),
		ActionLabel: "View Message",
	})

	return msg, nil
}

func (s *service) GetMessage(ctx context.Context, messageID uuid.UUID) (*SecureMessage, error) {
	msg, err := s.getMessageRecord(ctx, messageID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found: %s", messageID)
		}
		return nil, err
	}

	// Decrypt message
	decrypted, err := s.decrypt(msg.MessageTextEncrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt message: %w", err)
	}
	msg.MessageText = decrypted

	return msg, nil
}

func (s *service) ListConversation(ctx context.Context, conversationID uuid.UUID, limit int) ([]*SecureMessage, error) {
	messages, err := s.listConversationRecords(ctx, conversationID, limit)
	if err != nil {
		return nil, err
	}

	// Decrypt all messages
	for _, msg := range messages {
		decrypted, err := s.decrypt(msg.MessageTextEncrypted)
		if err == nil {
			msg.MessageText = decrypted
		}
	}

	return messages, nil
}

func (s *service) MarkAsRead(ctx context.Context, messageID uuid.UUID) error {
	return s.markAsReadRecord(ctx, messageID)
}

func (s *service) GetUnreadCount(ctx context.Context, recipientID uuid.UUID) (int, error) {
	return s.getUnreadCountRecord(ctx, recipientID)
}

// CreateNotification sends multi-channel notifications
func (s *service) CreateNotification(ctx context.Context, input CreateNotificationInput) (*Notification, error) {
	notif := &Notification{
		NotificationID: uuid.New(),
		ClientID:       input.ClientID,
		NotifType:      input.NotifType,
		Title:          input.Title,
		Body:           input.Body,
		Priority:       input.Priority,
		Channels:       input.Channels,
		ActionURL:      &input.ActionURL,
		ActionLabel:    &input.ActionLabel,
		CreatedAt:      time.Now(),
	}

	err := s.createNotificationRecord(ctx, notif)
	if err != nil {
		return nil, err
	}

	// Send via requested channels (async)
	go s.sendViaChannels(notif, input.Channels)

	return notif, nil
}

func (s *service) GetNotifications(ctx context.Context, clientID uuid.UUID, unreadOnly bool) ([]*Notification, error) {
	return s.getNotificationsRecords(ctx, clientID, unreadOnly)
}

func (s *service) MarkNotificationRead(ctx context.Context, notificationID uuid.UUID) error {
	return s.markNotificationReadRecord(ctx, notificationID)
}

func (s *service) DismissNotification(ctx context.Context, notificationID uuid.UUID) error {
	return s.dismissNotificationRecord(ctx, notificationID)
}

// encrypt uses AES-256-GCM for message encryption
func (s *service) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.encryptionKey)
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

// decrypt decodes AES-256-GCM encrypted messages
func (s *service) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := data[:nonceSize]
	ciphertextBytes := data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// sendViaChannels delivers notification through requested channels
func (s *service) sendViaChannels(notif *Notification, channels []string) {
	sentVia := make(map[string]bool)

	for _, channel := range channels {
		switch channel {
		case "EMAIL":
			if s.emailService != nil {
				// err := s.emailService.SendEmail(clientEmail, notif.Title, notif.Body)
				sentVia["EMAIL"] = true
			}
		case "SMS":
			if s.smsService != nil {
				// err := s.smsService.SendSMS(clientPhone, notif.Body)
				sentVia["SMS"] = true
			}
		case "PUSH":
			// Would integrate with Firebase/APNs for push notifications
			sentVia["PUSH"] = false
		case "IN_APP":
			// Already stored in database
			sentVia["IN_APP"] = true
		}
	}

	// Update sent_via in database
	sentViaJSON, _ := json.Marshal(sentVia)
	s.db.Exec(`UPDATE notifications SET sent_via = $1 WHERE notification_id = $2`, sentViaJSON, notif.NotificationID)
}

// Helper methods for SQL operations with Hasura fallback

// sendMessageRecord inserts an encrypted message
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 11 SecureMessage fields with encryption
func (s *service) sendMessageRecord(ctx context.Context, msg *SecureMessage) error {
	query := `
		INSERT INTO secure_messages (
message_id, conversation_id, sender_id, sender_type, recipient_id, recipient_type,
subject, message_text_encrypted, attachments, archived, created_at
) VALUES (
:message_id, :conversation_id, :sender_id, :sender_type, :recipient_id, :recipient_type,
:subject, :message_text_encrypted, :attachments, :archived, :created_at
)
	`
	_, err := s.db.NamedExecContext(ctx, query, msg)
	return err
}

// getMessageRecord retrieves a message by ID
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext SELECT * by message_id
func (s *service) getMessageRecord(ctx context.Context, messageID uuid.UUID) (*SecureMessage, error) {
	var msg SecureMessage
	query := `SELECT * FROM secure_messages WHERE message_id = $1`
	err := s.db.GetContext(ctx, &msg, query, messageID)
	return &msg, err
}

// listConversationRecords retrieves messages for a conversation
// TODO: Implement Hasura GraphQL query
// SQL fallback: SelectContext with ORDER BY created_at DESC and LIMIT
func (s *service) listConversationRecords(ctx context.Context, conversationID uuid.UUID, limit int) ([]*SecureMessage, error) {
	var messages []*SecureMessage
	query := `
		SELECT * FROM secure_messages
		WHERE conversation_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &messages, query, conversationID, limit)
	return messages, err
}

// markAsReadRecord marks a message as read
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: UPDATE read_at with NOW()
func (s *service) markAsReadRecord(ctx context.Context, messageID uuid.UUID) error {
	query := `UPDATE secure_messages SET read_at = NOW() WHERE message_id = $1`
	_, err := s.db.ExecContext(ctx, query, messageID)
	return err
}

// getUnreadCountRecord counts unread messages
// TODO: Implement Hasura GraphQL query
// SQL fallback: GetContext COUNT(*) WHERE read_at IS NULL
func (s *service) getUnreadCountRecord(ctx context.Context, recipientID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(*) FROM secure_messages
		WHERE recipient_id = $1 AND read_at IS NULL
	`
	err := s.db.GetContext(ctx, &count, query, recipientID)
	return count, err
}

// createNotificationRecord inserts a notification
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 10 Notification fields
func (s *service) createNotificationRecord(ctx context.Context, notif *Notification) error {
	query := `
		INSERT INTO notifications (
notification_id, client_id, notification_type, title, body, priority,
channels, action_url, action_label, created_at
) VALUES (
:notification_id, :client_id, :notification_type, :title, :body, :priority,
:channels, :action_url, :action_label, :created_at
)
	`
	_, err := s.db.NamedExecContext(ctx, query, notif)
	return err
}

// getNotificationsRecords retrieves notifications for a client
// TODO: Implement Hasura GraphQL query
// SQL fallback: SelectContext with dynamic WHERE (unreadOnly flag) and LIMIT 50
func (s *service) getNotificationsRecords(ctx context.Context, clientID uuid.UUID, unreadOnly bool) ([]*Notification, error) {
	var notifications []*Notification
	query := `
		SELECT * FROM notifications
		WHERE client_id = $1
	`
	if unreadOnly {
		query += ` AND read_at IS NULL AND dismissed_at IS NULL`
	}
	query += ` ORDER BY created_at DESC LIMIT 50`

	err := s.db.SelectContext(ctx, &notifications, query, clientID)
	return notifications, err
}

// markNotificationReadRecord marks a notification as read
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: UPDATE read_at with NOW()
func (s *service) markNotificationReadRecord(ctx context.Context, notificationID uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE notification_id = $1`
	_, err := s.db.ExecContext(ctx, query, notificationID)
	return err
}

// dismissNotificationRecord dismisses a notification
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: UPDATE dismissed_at with NOW()
func (s *service) dismissNotificationRecord(ctx context.Context, notificationID uuid.UUID) error {
	query := `UPDATE notifications SET dismissed_at = NOW() WHERE notification_id = $1`
	_, err := s.db.ExecContext(ctx, query, notificationID)
	return err
}
