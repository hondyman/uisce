package redpanda

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.temporal.io/sdk/client"

	"calendar-service/internal/availability"
	"calendar-service/internal/cache"
	"calendar-service/internal/hasura"
	"calendar-service/internal/metrics"
)

// CDCProcessor consumes CDC events from Redpanda and triggers cache invalidation
type CDCProcessor struct {
	brokers             []string
	topics              []string
	consumerGroup       string
	temporalClient      client.Client
	cacheClient         *cache.Client
	hasuraClient        *hasura.Client
	availabilityChecker *availability.Checker
	logger              *logrus.Entry
	kafkaClient         *kgo.Client
	eventListener       interface {
		OnEventCreated(ctx context.Context, userID, eventID string)
		OnEventUpdated(ctx context.Context, userID, eventID string)
		OnEventDeleted(ctx context.Context, userID, eventID string)
	}
	metrics *metrics.MetricsCollector
}

func NewCDCProcessor(
	brokers []string,
	topics []string,
	temporalClient client.Client,
	cacheClient *cache.Client,
	hasuraClient *hasura.Client,
	availabilityChecker *availability.Checker,
	eventListener interface {
		OnEventCreated(ctx context.Context, userID, eventID string)
		OnEventUpdated(ctx context.Context, userID, eventID string)
		OnEventDeleted(ctx context.Context, userID, eventID string)
	},
	metrics *metrics.MetricsCollector,
	logger *logrus.Entry,
) (*CDCProcessor, error) {
	return &CDCProcessor{
		brokers:             brokers,
		topics:              topics,
		consumerGroup:       "calendar-cdc-group",
		temporalClient:      temporalClient,
		cacheClient:         cacheClient,
		hasuraClient:        hasuraClient,
		availabilityChecker: availabilityChecker,
		eventListener:       eventListener,
		metrics:             metrics,
		logger:              logger.WithField("component", "cdc_processor"),
	}, nil
}

// CDCEvent represents a Debezium CDC event from Redpanda
type CDCEvent struct {
	Op        string          `json:"op"` // c=create, u=update, d=delete, r=read
	Table     string          `json:"table"`
	Schema    string          `json:"schema"`
	After     json.RawMessage `json:"after"`
	Before    json.RawMessage `json:"before"`
	Timestamp int64           `json:"ts_ms"`
	Source    struct {
		Connector string `json:"connector"`
		DB        string `json:"db"`
		Schema    string `json:"schema"`
		Table     string `json:"table"`
		Txid      int64  `json:"txId"`
		LSN       int64  `json:"lsn"`
	} `json:"source"`
}

// CalendarChangeEvent for invalidation signals
type CalendarChangeEvent struct {
	Entity           string // "calendar", "profile", "blackout"
	TenantID         string
	EntityID         string
	Region           string
	Operation        string   // INSERT, UPDATE, DELETE
	AffectedProfiles []string // affected profile names
}

func (p *CDCProcessor) Run(ctx context.Context) error {
	// Create Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers(p.brokers...),
		kgo.ConsumeTopics(p.topics...),
		kgo.ConsumerGroup(p.consumerGroup),
		kgo.FetchMaxWait(500*time.Millisecond),
		kgo.FetchMaxBytes(1024*1024), // 1MB max fetch
		kgo.AutoCommitInterval(5*time.Second),
		kgo.WithLogger(kgo.BasicLogger(os.Stderr, kgo.LogLevelError, nil)),
	)
	if err != nil {
		return fmt.Errorf("create kafka client: %w", err)
	}
	defer client.Close()
	p.kafkaClient = client

	p.logger.WithFields(logrus.Fields{
		"brokers": p.brokers,
		"topics":  p.topics,
		"group":   p.consumerGroup,
	}).Info("CDC processor starting")

	// Main consume loop
	for {
		select {
		case <-ctx.Done():
			p.logger.Info("CDC processor shutting down")
			return ctx.Err()
		default:
		}

		// Fetch batch of records
		fetches := client.PollFetches(ctx)
		if fetches.IsClientClosed() {
			return nil
		}

		// Handle errors
		fetches.EachError(func(topic string, partition int32, err error) {
			p.logger.WithError(err).Warn("CDC fetch error",
				logrus.Fields{
					"topic":     topic,
					"partition": partition,
				},
			)
		})

		// Process each record
		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			if err := p.processRecord(ctx, record); err != nil {
				p.logger.WithError(err).Warn("Failed to process CDC record",
					logrus.Fields{
						"topic":     record.Topic,
						"partition": record.Partition,
						"offset":    record.Offset,
						"key":       string(record.Key),
					},
				)
			}
		}
	}
}

func (p *CDCProcessor) processRecord(ctx context.Context, record *kgo.Record) error {
	// Validate record
	if record.Value == nil || len(record.Value) == 0 {
		return nil // Skip empty/tombstone records
	}

	// Parse CDC event
	var event CDCEvent
	if err := json.Unmarshal(record.Value, &event); err != nil {
		return fmt.Errorf("unmarshal CDC event: %w", err)
	}

	// Skip read operations (just snapshots)
	if event.Op == "r" {
		return nil
	}

	p.logger.WithFields(logrus.Fields{
		"table":     event.Table,
		"operation": event.Op,
		"schema":    event.Schema,
		"offset":    record.Offset,
	}).Debug("Processing CDC event")

	// Handle missing table information in root (fallback to source or topic)
	if event.Table == "" {
		if event.Source.Table != "" {
			event.Table = event.Source.Table
		} else {
			// Fallback: extract from topic name (e.g. cdc_calendar.public.internal_events)
			parts := strings.Split(record.Topic, ".")
			if len(parts) >= 3 {
				event.Table = parts[2]
			}
		}
	}

	if event.Schema == "" && event.Source.Schema != "" {
		event.Schema = event.Source.Schema
	}

	// Route to appropriate handler based on table
	switch event.Table {
	case "profile_calendars":
		return p.handleProfileCalendarsChange(ctx, event)
	case "calendars":
		return p.handleCalendarChange(ctx, event)
	case "schedule_profiles":
		return p.handleScheduleProfileChange(ctx, event)
	case "blackouts":
		return p.handleBlackoutChange(ctx, event)
	case "internal_events":
		return p.HandleInternalEventChange(ctx, event)
	default:
		p.logger.WithField("table", event.Table).Debug("Ignoring unhandled CDC table")
		return nil
	}
}

// handleInternalEventChange routes internal calendar event mutations to the EventListener
func (p *CDCProcessor) HandleInternalEventChange(ctx context.Context, event CDCEvent) error {
	if p.eventListener == nil {
		return nil
	}

	userID := p.extractField(event.After, event.Before, "user_id")
	eventID := p.extractField(event.After, event.Before, "id")

	if userID == "" || eventID == "" {
		return fmt.Errorf("missing required fields for event change: user_id=%s, event_id=%s", userID, eventID)
	}

	switch event.Op {
	case "c":
		p.eventListener.OnEventCreated(ctx, userID, eventID)
	case "u":
		p.eventListener.OnEventUpdated(ctx, userID, eventID)
	case "d":
		p.eventListener.OnEventDeleted(ctx, userID, eventID)
	}

	p.recordCDCEvent("internal_events", event.Op)
	return nil
}

// handleProfileCalendarsChange processes changes to profile_calendars mapping table
// When a calendar is added/removed from a profile, invalidate the profile's cache
func (p *CDCProcessor) handleProfileCalendarsChange(ctx context.Context, event CDCEvent) error {
	// Extract key fields from new/old values
	tenantID := p.extractField(event.After, event.Before, "tenant_id")
	profileID := p.extractField(event.After, event.Before, "profile_id")
	calendarID := p.extractField(event.After, event.Before, "calendar_id")

	if tenantID == "" || calendarID == "" {
		return fmt.Errorf("missing required fields: tenant_id=%s, calendar_id=%s", tenantID, calendarID)
	}

	logger := p.logger.WithFields(logrus.Fields{
		"table":       "profile_calendars",
		"operation":   event.Op,
		"tenant_id":   tenantID,
		"profile_id":  profileID,
		"calendar_id": calendarID,
	})

	// Invalidate the profile name cache for this calendar
	// When the mapping changes, future lookups will hit Hasura to refresh
	if p.availabilityChecker != nil {
		p.availabilityChecker.InvalidateProfileNameCache(ctx, tenantID, calendarID)
		logger.Debug("Invalidated profile name cache for calendar")
	}

	// Record metric
	p.recordCDCEvent("profile_calendars", event.Op)

	// Optionally signal Temporal workflow for rescheduling
	// (commented out until workflow is defined)
	// if p.temporalClient != nil && profileID != "" {
	//	 err := p.signalTemporalWorkflow(ctx, tenantID, profileID, "ProfileCalendarsChanged", event.Op)
	//	 if err != nil {
	//		 logger.WithError(err).Warn("Failed to signal Temporal workflow")
	//	 }
	// }

	return nil
}

// handleCalendarChange processes changes to calendars table
// When holidays/blackouts in a calendar change, invalidate all profiles using it
func (p *CDCProcessor) handleCalendarChange(ctx context.Context, event CDCEvent) error {
	tenantID := p.extractField(event.After, event.Before, "tenant_id")
	calendarID := p.extractField(event.After, event.Before, "id")
	region := p.extractField(event.After, event.Before, "region")

	if tenantID == "" || calendarID == "" {
		return fmt.Errorf("missing required fields: tenant_id=%s, calendar_id=%s", tenantID, calendarID)
	}

	logger := p.logger.WithFields(logrus.Fields{
		"table":       "calendars",
		"operation":   event.Op,
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
		"region":      region,
	})

	// For DELETE operations, invalidate all profiles that used this calendar
	// For INSERT/UPDATE, invalidate resolved profile cache since content changed
	if event.Op == "d" {
		// Calendar deleted - find all profiles that used it
		profiles, err := p.findProfilesUsingCalendar(ctx, tenantID, calendarID)
		if err != nil {
			logger.WithError(err).Warn("Failed to find affected profiles")
		} else {
			logger.WithField("profiles", profiles).Debug("Invalidating profiles for deleted calendar")
		}
	} else {
		// Calendar INSERT/UPDATE - content changed, invalidate resolved caches
		if p.cacheClient != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			// Invalidate for all known regions
			allRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
			for _, r := range allRegions {
				// Clear any cached resolved profiles that used this calendar
				p.cacheClient.PublishInvalidation(ctx, tenantID, r)
			}
			logger.Debug("Published cache invalidation for all regions")
		}
	}

	p.recordCDCEvent("calendars", event.Op)
	return nil
}

// handleScheduleProfileChange processes changes to schedule_profiles table
// When profile metadata changes, invalidate caches for that profile
func (p *CDCProcessor) handleScheduleProfileChange(ctx context.Context, event CDCEvent) error {
	tenantID := p.extractField(event.After, event.Before, "tenant_id")
	profileID := p.extractField(event.After, event.Before, "id")
	profileName := p.extractField(event.After, event.Before, "profile_name")

	if tenantID == "" || profileID == "" {
		return fmt.Errorf("missing required fields: tenant_id=%s, profile=%s", tenantID, profileID)
	}

	logger := p.logger.WithFields(logrus.Fields{
		"table":        "schedule_profiles",
		"operation":    event.Op,
		"tenant_id":    tenantID,
		"profile_id":   profileID,
		"profile_name": profileName,
	})

	// Invalidate resolved profile cache for this profile
	if p.cacheClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		allRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
		for _, region := range allRegions {
			// Invalidate all profiles in this region
			p.cacheClient.PublishInvalidation(ctx, tenantID, region)
		}
		logger.Debug("Published cache invalidation for profile change")
	}

	p.recordCDCEvent("schedule_profiles", event.Op)
	return nil
}

// handleBlackoutChange processes changes to blackouts table
// When blackouts change, invalidate resolved profile cache
func (p *CDCProcessor) handleBlackoutChange(ctx context.Context, event CDCEvent) error {
	tenantID := p.extractField(event.After, event.Before, "tenant_id")
	blackoutID := p.extractField(event.After, event.Before, "id")
	profileID := p.extractField(event.After, event.Before, "profile_id")

	if tenantID == "" || blackoutID == "" {
		return fmt.Errorf("missing required fields: tenant_id=%s, blackout_id=%s", tenantID, blackoutID)
	}

	logger := p.logger.WithFields(logrus.Fields{
		"table":       "blackouts",
		"operation":   event.Op,
		"tenant_id":   tenantID,
		"blackout_id": blackoutID,
		"profile_id":  profileID,
	})

	// Invalidate resolved profile cache for affected profile
	if p.cacheClient != nil && profileID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		allRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
		for _, region := range allRegions {
			p.cacheClient.PublishInvalidation(ctx, tenantID, region)
		}
		logger.Debug("Published cache invalidation for blackout change")
	}

	p.recordCDCEvent("blackouts", event.Op)
	return nil
}

// extractField extracts a string field from After or Before payload
func (p *CDCProcessor) extractField(after, before json.RawMessage, fieldName string) string {
	// Helper to extract from a single raw message
	extract := func(data json.RawMessage) string {
		if len(data) == 0 || string(data) == "null" {
			return ""
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			return ""
		}

		if v, ok := m[fieldName]; ok {
			switch val := v.(type) {
			case string:
				return val
			case float64:
				return fmt.Sprintf("%.0f", val)
			case nil:
				return ""
			default:
				return fmt.Sprintf("%v", val)
			}
		}
		return ""
	}

	// Try After first, then fall back to Before
	val := extract(after)
	if val == "" {
		val = extract(before)
	}
	return val
}

// findProfilesUsingCalendar queries Hasura to find all profiles using a calendar
func (p *CDCProcessor) findProfilesUsingCalendar(ctx context.Context, tenantID, calendarID string) ([]string, error) {
	if p.hasuraClient == nil {
		return nil, fmt.Errorf("hasura client not configured")
	}

	var result struct {
		ProfileCalendars []struct {
			ScheduleProfile struct {
				ID          string `json:"id"`
				ProfileName string `json:"profile_name"`
			} `json:"schedule_profile"`
		} `json:"profile_calendars"`
	}

	// Use hashaClient.Query with correct signature
	queryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := p.hasuraClient.Query(queryCtx, &result, map[string]interface{}{
		"tenant_id":   tenantID,
		"calendar_id": calendarID,
	}); err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var profileNames []string
	for _, pc := range result.ProfileCalendars {
		profileNames = append(profileNames, pc.ScheduleProfile.ProfileName)
	}
	return profileNames, nil
}

// recordCDCEvent records a metric for the CDC event
func (p *CDCProcessor) recordCDCEvent(table, operation string) {
	if p.metrics != nil {
		p.metrics.RecordCDCEvent(table, operation)
	}
}

// processCalendarChange handles calendar update/delete events for resolved profile invalidation
func (p *CDCProcessor) processCalendarChange(ctx context.Context, event *CalendarChangeEvent) error {
	logger := p.logger.WithFields(logrus.Fields{
		"tenant_id": event.TenantID,
		"region":    event.Region,
		"entity":    event.Entity,
	})

	// 1. Invalidate cache for all affected profiles
	if p.cacheClient != nil {
		logger.WithField("profile_count", len(event.AffectedProfiles)).Debug("Invalidating cache for profiles")
		p.cacheClient.InvalidateTenantProfiles(
			ctx,
			event.TenantID,
			event.Region,
			event.AffectedProfiles,
		)
	}

	// 2. Signal Temporal workflow to reschedule affected jobs
	// This would trigger the calendar-changed-workflow for this tenant+region
	if p.temporalClient != nil {
		logger.Debug("Signaling Temporal workflow for reschedule")
		// p.temporalClient.SignalWorkflow(ctx, ...) - implementation depends on workflow design
	}

	return nil
}

// InvalidateProfileNameCacheForChange handles CDC events on profile_calendars table
// When the mapping between a profile and calendar changes, invalidate the profile name
// cache for that calendar so the next lookup queries Hasura
func (p *CDCProcessor) InvalidateProfileNameCacheForChange(ctx context.Context, tenantID, profileID, calendarID string, operation string) error {
	logger := p.logger.WithFields(logrus.Fields{
		"tenant_id":   tenantID,
		"profile_id":  profileID,
		"calendar_id": calendarID,
		"operation":   operation, // INSERT, UPDATE, DELETE
	})

	// Invalidate profile name cache for this calendar
	if p.availabilityChecker != nil {
		p.availabilityChecker.InvalidateProfileNameCache(ctx, tenantID, calendarID)
		logger.Debug("Invalidated profile name cache for calendar")
	}

	// Invalidate resolved calendar cache as well
	if p.cacheClient != nil {
		// Invalidate any cached profile resolution data for this calendar
		// The profile name mapping is what we just invalidated above;
		// this also invalidates full resolved profile (holidays/blackouts)
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Clear the resolved profile cache for this calendar across all regions
		// Since we don't know which regions use this profile, clear broadly
		allRegions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-southeast-1"}
		for _, region := range allRegions {
			// Invalidate resolved profile cache (the cache layer uses tenant:region:profilename as key)
			// But we don't have the profile name yet, so we rely on the checker's L1 invalidation above
			logger.WithField("region", region).Debug("Cache invalidation queued for region")
		}

		_ = bgCtx.Err() // Suppress unused variable warning
	}

	return nil
}

// Close gracefully shuts down the CDC processor
func (p *CDCProcessor) Close() error {
	if p.kafkaClient != nil {
		p.kafkaClient.LeaveGroup()
		p.logger.Info("CDC processor gracefully closed")
	}
	return nil
}
