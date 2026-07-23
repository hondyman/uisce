package sync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Conflict Metrics
	conflictsDetectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_sync_conflicts_detected_total",
			Help: "Total number of conflicts detected between Google and Internal events",
		},
		[]string{"type", "severity"},
	)

	conflictResolvedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_sync_conflicts_resolved_total",
			Help: "Total number of conflicts resolved",
		},
		[]string{"strategy"},
	)

	// Recurring Event Metrics
	recurringEventsExpanded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_recurring_events_expanded_total",
			Help: "Total number of recurring events expanded",
		},
		[]string{"status"},
	)

	recurringEventInstances = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "google_recurring_event_instances",
			Help:    "Number of instances per recurring event",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"freq"},
	)

	recurringEventExpansionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_recurring_event_expansion_errors_total",
			Help: "Total number of recurring event expansion errors",
		},
		[]string{"error_type"},
	)

	// Timezone Metrics
	timezoneConversions = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "timezone_conversions_total",
			Help: "Total number of timezone conversions",
		},
		[]string{"from_timezone", "to_timezone"},
	)

	timezoneConversionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "timezone_conversion_errors_total",
			Help: "Total number of timezone conversion errors",
		},
		[]string{"error_type"},
	)

	// Listener/Push Metrics
	internalEventsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "google_sync_internal_events_received_total",
			Help: "Total number of internal events received for sync",
		},
		[]string{"event_type"},
	)

	pushToGoogleDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "google_sync_push_duration_seconds",
			Help:    "Duration of pushing events to Google",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
)
