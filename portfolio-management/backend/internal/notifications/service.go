package notifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"net/smtp"

	"github.com/lib/pq"
	"github.com/twilio/twilio-go"
	twilioApi "github.com/twilio/twilio-go/rest/api/v2010"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// Notification types
type Notification struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	PortfolioID *string         `json:"portfolio_id"`
	Type        string          `json:"type"`
	Priority    string          `json:"priority"`
	Subject     string          `json:"subject"`
	Message     string          `json:"message"`
	Channels    []string        `json:"channels"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
}

type NotificationDelivery struct {
	ID             string     `json:"id"`
	NotificationID string     `json:"notification_id"`
	Channel        string     `json:"channel"`
	Recipient      string     `json:"recipient"`
	Status         string     `json:"status"`
	RetryCount     int        `json:"retry_count"`
	MaxRetries     int        `json:"max_retries"`
	ErrorMessage   *string    `json:"error_message"`
	SentAt         *time.Time `json:"sent_at"`
	FailedAt       *time.Time `json:"failed_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// Notification service
type NotificationService struct {
	db             *sql.DB
	hasuraClient   HasuraClient
	mailFrom       string
	smtpHost       string
	smtpPort       string
	smtpPassword   string
	twilioClient   *twilio.RestClient
	twilioPhoneNum string
	pusherAppID    string
	pusherKey      string
	pusherSecret   string
	pusherCluster  string
	queue          chan *Notification
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewNotificationService creates and initializes the notification service
func NewNotificationService(db *sql.DB) (*NotificationService, error) {
	// Initialize Twilio client
	twilioClient := twilio.NewRestClient()

	ctx, cancel := context.WithCancel(context.Background())

	service := &NotificationService{
		db:             db,
		mailFrom:       os.Getenv("SMTP_FROM_EMAIL"),
		smtpHost:       os.Getenv("SMTP_HOST"),
		smtpPort:       os.Getenv("SMTP_PORT"),
		smtpPassword:   os.Getenv("SMTP_PASSWORD"),
		twilioClient:   twilioClient,
		twilioPhoneNum: os.Getenv("TWILIO_PHONE_NUMBER"),
		pusherAppID:    os.Getenv("PUSHER_APP_ID"),
		pusherKey:      os.Getenv("PUSHER_KEY"),
		pusherSecret:   os.Getenv("PUSHER_SECRET"),
		pusherCluster:  os.Getenv("PUSHER_CLUSTER"),
		queue:          make(chan *Notification, 1000),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start worker goroutines
	for i := 0; i < 5; i++ {
		service.wg.Add(1)
		go service.processQueue()
	}

	return service, nil
}

// NewNotificationServiceWithHasura creates a service with Hasura support
func NewNotificationServiceWithHasura(db *sql.DB, hasuraClient HasuraClient) (*NotificationService, error) {
	twilioClient := twilio.NewRestClient()
	ctx, cancel := context.WithCancel(context.Background())

	service := &NotificationService{
		db:             db,
		hasuraClient:   hasuraClient,
		mailFrom:       os.Getenv("SMTP_FROM_EMAIL"),
		smtpHost:       os.Getenv("SMTP_HOST"),
		smtpPort:       os.Getenv("SMTP_PORT"),
		smtpPassword:   os.Getenv("SMTP_PASSWORD"),
		twilioClient:   twilioClient,
		twilioPhoneNum: os.Getenv("TWILIO_PHONE_NUMBER"),
		pusherAppID:    os.Getenv("PUSHER_APP_ID"),
		pusherKey:      os.Getenv("PUSHER_KEY"),
		pusherSecret:   os.Getenv("PUSHER_SECRET"),
		pusherCluster:  os.Getenv("PUSHER_CLUSTER"),
		queue:          make(chan *Notification, 1000),
		ctx:            ctx,
		cancel:         cancel,
	}

	for i := 0; i < 5; i++ {
		service.wg.Add(1)
		go service.processQueue()
	}

	return service, nil
}

// Enqueue adds a notification to the processing queue
func (ns *NotificationService) Enqueue(notification *Notification) error {
	select {
	case ns.queue <- notification:
		return nil
	case <-ns.ctx.Done():
		return fmt.Errorf("notification service is shutting down")
	default:
		return fmt.Errorf("notification queue is full")
	}
}

// processQueue processes notifications from the queue
func (ns *NotificationService) processQueue() {
	defer ns.wg.Done()

	for {
		select {
		case notification := <-ns.queue:
			if notification == nil {
				return
			}
			ns.processNotification(notification)
		case <-ns.ctx.Done():
			return
		}
	}
}

// processNotification handles a single notification
func (ns *NotificationService) processNotification(notification *Notification) {
	log.Printf("Processing notification: %s (type: %s, priority: %s)", notification.ID, notification.Type, notification.Priority)

	for _, channel := range notification.Channels {
		go func(ch string) {
			var recipient string
			var err error

			switch ch {
			case "email":
				recipient, err = ns.getUserEmail(notification.UserID)
				if err != nil {
					log.Printf("Failed to get email for user %s: %v", notification.UserID, err)
					ns.logDeliveryFailure(notification.ID, ch, "", err.Error())
					return
				}
				err = ns.sendEmail(recipient, notification.Subject, notification.Message)
			case "sms":
				recipient, err = ns.getUserPhoneNumber(notification.UserID)
				if err != nil {
					log.Printf("Failed to get phone for user %s: %v", notification.UserID, err)
					ns.logDeliveryFailure(notification.ID, ch, "", err.Error())
					return
				}
				err = ns.sendSMS(recipient, notification.Message)
			case "push":
				recipient = notification.UserID
				err = ns.sendPushNotification(notification.UserID, notification.Subject, notification.Message)
			case "in_app":
				err = ns.createInAppNotification(notification.ID, notification.UserID)
			}

			if err != nil {
				log.Printf("Failed to send %s notification %s: %v", ch, notification.ID, err)
				ns.logDeliveryFailure(notification.ID, ch, recipient, err.Error())
				ns.retryNotification(notification.ID, ch)
			} else {
				log.Printf("Successfully sent %s notification %s to %s", ch, notification.ID, recipient)
				ns.logDeliverySuccess(notification.ID, ch, recipient)
			}
		}(channel)
	}
}

// sendEmail sends an email notification
func (ns *NotificationService) sendEmail(to, subject, body string) error {
	if ns.smtpHost == "" || ns.mailFrom == "" {
		return fmt.Errorf("SMTP not configured")
	}

	auth := smtp.PlainAuth("", os.Getenv("SMTP_USERNAME"), ns.smtpPassword, ns.smtpHost)
	addr := ns.smtpHost + ":" + ns.smtpPort

	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s", ns.mailFrom, to, subject, body)
	err := smtp.SendMail(addr, auth, ns.mailFrom, []string{to}, []byte(message))

	return err
}

// sendSMS sends an SMS notification via Twilio
func (ns *NotificationService) sendSMS(to, message string) error {
	if ns.twilioClient == nil || ns.twilioPhoneNum == "" {
		return fmt.Errorf("Twilio not configured")
	}

	params := &twilioApi.CreateMessageParams{}
	params.SetBody(message)
	params.SetFrom(ns.twilioPhoneNum)
	params.SetTo(to)

	resp, err := ns.twilioClient.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	if resp.Sid == nil {
		return fmt.Errorf("SMS send returned no SID")
	}

	return nil
}

// sendPushNotification sends a push notification
func (ns *NotificationService) sendPushNotification(userID, title, message string) error {
	// TODO: Implement Pusher integration
	log.Printf("Push notification for user %s: %s - %s", userID, title, message)
	return nil
}

// createInAppNotification creates an in-app notification
func (ns *NotificationService) createInAppNotification(notificationID, userID string) error {
	// Already created in database, just mark as delivered
	return nil
}

// getUserEmail retrieves user email from database
func (ns *NotificationService) getUserEmail(userID string) (string, error) {
	return ns.getUserEmailRecord(userID)
}

// getUserPhoneNumber retrieves user phone number from database
func (ns *NotificationService) getUserPhoneNumber(userID string) (string, error) {
	return ns.getUserPhoneNumberRecord(userID)
}

// logDeliverySuccess logs a successful notification delivery
func (ns *NotificationService) logDeliverySuccess(notificationID, channel, recipient string) error {
	return ns.logDeliverySuccessRecord(notificationID, channel, time.Now())
}

// logDeliveryFailure logs a failed notification delivery
func (ns *NotificationService) logDeliveryFailure(notificationID, channel, recipient, errorMsg string) error {
	return ns.logDeliveryFailureRecord(notificationID, channel, recipient, errorMsg)
}

// retryNotification retries a failed notification with exponential backoff
func (ns *NotificationService) retryNotification(notificationID, channel string) {
	var retryCount int
	var maxRetries int

	err := ns.db.QueryRow(`
		SELECT retry_count, max_retries FROM notification_deliveries
		WHERE notification_id = $1 AND channel = $2 AND status = 'failed'
	`, notificationID, channel).Scan(&retryCount, &maxRetries)

	if err == sql.ErrNoRows {
		return
	}

	if err != nil {
		log.Printf("Error checking retry count: %v", err)
		return
	}

	if retryCount >= maxRetries {
		log.Printf("Max retries exceeded for notification %s on channel %s", notificationID, channel)
		return
	}

	// Exponential backoff: wait (2^retryCount) seconds
	backoffDuration := time.Duration(1<<uint(retryCount)) * time.Second
	log.Printf("Retrying notification %s on %s after %v (attempt %d/%d)",
		notificationID, channel, backoffDuration, retryCount+1, maxRetries)

	time.AfterFunc(backoffDuration, func() {
		// Update retry count and mark as pending
		_, updateErr := ns.db.Exec(`
			UPDATE notification_deliveries
			SET retry_count = retry_count + 1, status = 'pending'
			WHERE notification_id = $1 AND channel = $2
		`, notificationID, channel)

		if updateErr != nil {
			log.Printf("Error updating retry count: %v", updateErr)
		}

		// Fetch and retry
		var notification Notification
		scanErr := ns.db.QueryRow(`
			SELECT n.id, n.user_id, n.portfolio_id, n.type, n.priority, n.subject, n.message, n.channels, n.metadata, n.created_at
			FROM notifications n
			WHERE n.id = $1
		`, notificationID).Scan(
			&notification.ID, &notification.UserID, &notification.PortfolioID,
			&notification.Type, &notification.Priority, &notification.Subject,
			&notification.Message, pq.Array(&notification.Channels),
			&notification.Metadata, &notification.CreatedAt,
		)

		if scanErr != nil {
			log.Printf("Error fetching notification for retry: %v", scanErr)
			return
		}

		if enqueueErr := ns.Enqueue(&notification); enqueueErr != nil {
			log.Printf("Error re-enqueueing notification: %v", enqueueErr)
		}
	})
}

// ListenForNotifications listens on PostgreSQL for new notifications
func (ns *NotificationService) ListenForNotifications() error {
	listener := pq.NewListener(
		fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Printf("Listener error: %v", err)
			}
		},
	)
	defer listener.Close()

	err := listener.Listen("notifications")
	if err != nil {
		return fmt.Errorf("failed to listen on notifications channel: %w", err)
	}

	log.Println("Listening for new notifications...")

	for {
		select {
		case notification := <-listener.Notify:
			if notification != nil {
				var n Notification
				if err := json.Unmarshal([]byte(notification.Extra), &n); err == nil {
					if err := ns.Enqueue(&n); err != nil {
						log.Printf("Failed to enqueue notification: %v", err)
					}
				}
			}
		case <-ns.ctx.Done():
			return ns.ctx.Err()
		}
	}
}

// Shutdown gracefully shuts down the notification service
func (ns *NotificationService) Shutdown(timeout time.Duration) error {
	ns.cancel()
	close(ns.queue)

	done := make(chan struct{})
	go func() {
		ns.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("notification service shutdown timeout exceeded")
	}
}

// Health check endpoint
func (ns *NotificationService) HealthCheck() map[string]interface{} {
	return map[string]interface{}{
		"status":     "ok",
		"timestamp":  time.Now(),
		"queue_size": len(ns.queue),
		"db_status":  ns.db.Ping() == nil,
	}
}

// Helper methods for Hasura/SQL operations

func (ns *NotificationService) getUserEmailRecord(userID string) (string, error) {
	// Note: Using SQL fallback for simple SELECT
	var email string
	err := ns.db.QueryRow(
		"SELECT email FROM users WHERE id = $1",
		userID,
	).Scan(&email)
	return email, err
}

func (ns *NotificationService) getUserPhoneNumberRecord(userID string) (string, error) {
	// Note: Using SQL fallback for simple SELECT
	var phone string
	err := ns.db.QueryRow(
		"SELECT phone_number FROM user_profiles WHERE user_id = $1",
		userID,
	).Scan(&phone)
	return phone, err
}

func (ns *NotificationService) logDeliverySuccessRecord(notificationID, channel string, sentAt time.Time) error {
	// Note: Using SQL fallback for simple UPDATE
	_, err := ns.db.Exec(`
		UPDATE notification_deliveries
		SET status = 'sent', sent_at = $1
		WHERE notification_id = $2 AND channel = $3
	`, sentAt, notificationID, channel)
	return err
}

func (ns *NotificationService) logDeliveryFailureRecord(notificationID, channel, recipient, errorMsg string) error {
	// Note: Using SQL fallback for simple INSERT
	_, err := ns.db.Exec(`
		INSERT INTO notification_deliveries 
		(notification_id, channel, recipient, status, error_message)
		VALUES ($1, $2, $3, 'failed', $4)
	`, notificationID, channel, recipient, errorMsg)
	return err
}
