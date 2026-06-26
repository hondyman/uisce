package wealth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// DeliveryService handles the delivery of feed items to clients via their preferred channels
type DeliveryService struct {
	db *sql.DB
}

// NewDeliveryService creates a new DeliveryService
func NewDeliveryService(db *sql.DB) *DeliveryService {
	return &DeliveryService{
		db: db,
	}
}

// DeliverItem attempts to deliver a feed item to a client
func (s *DeliveryService) DeliverItem(ctx context.Context, clientID string, item FeedItem) error {
	// 1. Fetch Engagement Preferences
	prefs, err := s.fetchPreferences(ctx, clientID)
	if err != nil {
		return fmt.Errorf("failed to fetch preferences: %v", err)
	}

	// 2. Check Quiet Hours
	if s.inQuietHours(prefs) {
		logging.GetLogger().Sugar().Infof("Skipping delivery for client %s due to quiet hours", clientID)
		return nil // Or queue for later
	}

	// 3. Select Channel
	channel := prefs.PreferredChannel
	
	// 4. Simulate Delivery
	logging.GetLogger().Sugar().Infof("Delivering item %s to client %s via %s: %s", item.ID, clientID, channel, item.Title)

	// 5. Log Delivery Event (UAR)
	// In a real system, this would write to the audit log
	s.logDelivery(clientID, item.ID, channel)

	return nil
}

// Helper structs and methods

type EngagementPreferences struct {
	ClientID         string
	PreferredChannel string // MOBILE_PUSH, EMAIL, WEB
	QuietHoursStart  string // "22:00"
	QuietHoursEnd    string // "07:00"
}

func (s *DeliveryService) fetchPreferences(ctx context.Context, clientID string) (*EngagementPreferences, error) {
	// Placeholder: Fetch from DB or BO Service
	return &EngagementPreferences{
		ClientID:         clientID,
		PreferredChannel: "MOBILE_PUSH",
		QuietHoursStart:  "22:00",
		QuietHoursEnd:    "07:00",
	}, nil
}

func (s *DeliveryService) inQuietHours(prefs *EngagementPreferences) bool {
	// Simplified check
	now := time.Now()
	currentHour := now.Hour()
	
	// Parse start/end (assuming HH:00 format for simplicity)
	var start, end int
	fmt.Sscanf(prefs.QuietHoursStart, "%d:00", &start)
	fmt.Sscanf(prefs.QuietHoursEnd, "%d:00", &end)

	if start > end {
		// Spans midnight (e.g. 22:00 to 07:00)
		return currentHour >= start || currentHour < end
	}
	return currentHour >= start && currentHour < end
}

func (s *DeliveryService) logDelivery(clientID, itemID, channel string) {
	// Placeholder for UAR logging
	logging.GetLogger().Sugar().Debugf("[AUDIT] Delivery: Client=%s, Item=%s, Channel=%s, Time=%s", clientID, itemID, channel, time.Now().Format(time.RFC3339))
}
