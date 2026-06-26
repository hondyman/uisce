package middleware

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// NotificationMiddleware handles real-time notification delivery via WebSocket
type NotificationMiddleware struct {
	clients         map[string]*websocket.Conn // userID -> connection
	clientsMutex    sync.RWMutex
	notificationSvc *services.EngagementNotificationService
	upgrader        websocket.Upgrader
}

// NewNotificationMiddleware creates a new notification middleware
func NewNotificationMiddleware(notificationSvc *services.EngagementNotificationService) *NotificationMiddleware {
	return &NotificationMiddleware{
		clients:         make(map[string]*websocket.Conn),
		notificationSvc: notificationSvc,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
	}
}

// HandleWebSocket handles WebSocket connections for real-time notifications
func (nm *NotificationMiddleware) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from request (this would come from authentication middleware)
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		http.Error(w, "User ID required", http.StatusBadRequest)
		return
	}

	conn, err := nm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Register client
	nm.clientsMutex.Lock()
	nm.clients[userID] = conn
	nm.clientsMutex.Unlock()

	log.Printf("WebSocket connection established for user %s", userID)

	// Handle incoming messages
	go nm.handleClientMessages(userID, conn)

	// Send initial notifications
	nm.sendInitialNotifications(userID, conn)
}

// SendNotificationToUser sends a notification to a specific user via WebSocket
func (nm *NotificationMiddleware) SendNotificationToUser(userID string, notification *models.EngagementNotification) error {
	nm.clientsMutex.RLock()
	conn, exists := nm.clients[userID]
	nm.clientsMutex.RUnlock()

	if !exists {
		log.Printf("No active WebSocket connection for user %s", userID)
		return nil // User not connected, that's okay
	}

	message := map[string]interface{}{
		"type":         "notification",
		"notification": notification,
		"timestamp":    time.Now(),
	}

	return nm.sendMessage(conn, message)
}

// BroadcastNotification broadcasts a notification to all connected users
func (nm *NotificationMiddleware) BroadcastNotification(notification *models.EngagementNotification) error {
	message := map[string]interface{}{
		"type":         "notification",
		"notification": notification,
		"timestamp":    time.Now(),
	}

	nm.clientsMutex.RLock()
	defer nm.clientsMutex.RUnlock()

	for userID, conn := range nm.clients {
		if err := nm.sendMessage(conn, message); err != nil {
			log.Printf("Failed to send notification to user %s: %v", userID, err)
		}
	}

	return nil
}

// SendEngagementUpdate sends engagement analytics updates to a user
func (nm *NotificationMiddleware) SendEngagementUpdate(userID string, analytics map[string]interface{}) error {
	nm.clientsMutex.RLock()
	conn, exists := nm.clients[userID]
	nm.clientsMutex.RUnlock()

	if !exists {
		return nil
	}

	message := map[string]interface{}{
		"type":      "engagement_update",
		"analytics": analytics,
		"timestamp": time.Now(),
	}

	return nm.sendMessage(conn, message)
}

// RemoveClient removes a client connection
func (nm *NotificationMiddleware) RemoveClient(userID string) {
	nm.clientsMutex.Lock()
	if conn, exists := nm.clients[userID]; exists {
		conn.Close()
		delete(nm.clients, userID)
		log.Printf("WebSocket connection closed for user %s", userID)
	}
	nm.clientsMutex.Unlock()
}

// GetConnectedUsers returns a list of currently connected user IDs
func (nm *NotificationMiddleware) GetConnectedUsers() []string {
	nm.clientsMutex.RLock()
	defer nm.clientsMutex.RUnlock()

	users := make([]string, 0, len(nm.clients))
	for userID := range nm.clients {
		users = append(users, userID)
	}

	return users
}

// Helper methods

func (nm *NotificationMiddleware) handleClientMessages(userID string, conn *websocket.Conn) {
	defer nm.RemoveClient(userID)

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error for user %s: %v", userID, err)
			}
			break
		}

		// Handle client messages (e.g., mark as read, engagement events)
		nm.handleMessage(userID, msg)
	}
}

func (nm *NotificationMiddleware) handleMessage(userID string, msg map[string]interface{}) {
	msgType, ok := msg["type"].(string)
	if !ok {
		log.Printf("Invalid message type from user %s", userID)
		return
	}

	switch msgType {
	case "mark_read":
		nm.handleMarkAsRead(userID, msg)
	case "engagement_event":
		nm.handleEngagementEvent(userID, msg)
	case "ping":
		nm.handlePing(userID)
	default:
		log.Printf("Unknown message type: %s from user %s", msgType, userID)
	}
}

func (nm *NotificationMiddleware) handleMarkAsRead(userID string, msg map[string]interface{}) {
	notificationID, ok := msg["notification_id"].(string)
	if !ok {
		log.Printf("Invalid notification_id in mark_read message from user %s", userID)
		return
	}

	// Update notification status in database
	ctx := context.Background()
	err := nm.updateNotificationStatus(ctx, notificationID, "read", &time.Time{})
	if err != nil {
		log.Printf("Failed to mark notification %s as read: %v", notificationID, err)
	}

	// Track engagement event
	analytics := &models.NotificationAnalytics{
		NotificationID: notificationID,
		UserID:         userID,
		EventType:      "read",
		EventTimestamp: time.Now(),
	}
	nm.notificationSvc.TrackEngagementEvent(ctx, analytics)
}

func (nm *NotificationMiddleware) handleEngagementEvent(userID string, msg map[string]interface{}) {
	notificationID, ok := msg["notification_id"].(string)
	if !ok {
		log.Printf("Invalid notification_id in engagement_event message from user %s", userID)
		return
	}

	eventType, ok := msg["event_type"].(string)
	if !ok {
		log.Printf("Invalid event_type in engagement_event message from user %s", userID)
		return
	}

	// Track engagement event
	ctx := context.Background()
	analytics := &models.NotificationAnalytics{
		NotificationID:     notificationID,
		UserID:             userID,
		EventType:          eventType,
		EventTimestamp:     time.Now(),
		AdditionalMetadata: msg,
	}
	nm.notificationSvc.TrackEngagementEvent(ctx, analytics)
}

func (nm *NotificationMiddleware) handlePing(userID string) {
	nm.clientsMutex.RLock()
	conn, exists := nm.clients[userID]
	nm.clientsMutex.RUnlock()

	if exists {
		message := map[string]interface{}{
			"type":      "pong",
			"timestamp": time.Now(),
		}
		nm.sendMessage(conn, message)
	}
}

func (nm *NotificationMiddleware) sendInitialNotifications(userID string, conn *websocket.Conn) {
	ctx := context.Background()
	notifications, err := nm.notificationSvc.GetUserNotifications(ctx, userID, 10, 0)
	if err != nil {
		log.Printf("Failed to get initial notifications for user %s: %v", userID, err)
		return
	}

	message := map[string]interface{}{
		"type":          "initial_notifications",
		"notifications": notifications,
		"timestamp":     time.Now(),
	}

	nm.sendMessage(conn, message)
}

func (nm *NotificationMiddleware) sendMessage(conn *websocket.Conn, message interface{}) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(message)
}

func (nm *NotificationMiddleware) updateNotificationStatus(ctx context.Context, notificationID, status string, timestamp *time.Time) error {
	// For now, we'll just log the status update
	// In a real implementation, this would update the database
	log.Printf("Updating notification %s status to %s", notificationID, status)
	return nil
}
