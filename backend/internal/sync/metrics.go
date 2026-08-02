package sync

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	conflictsDetectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_sync_conflicts_detected_total",
			Help: "Total number of conflicts detected between external and internal events",
		},
		[]string{"type", "severity"},
	)

	conflictResolvedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_sync_conflicts_resolved_total",
			Help: "Total number of conflicts resolved",
		},
		[]string{"strategy"},
	)

	recurringEventsExpanded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_recurring_events_expanded_total",
			Help: "Total number of recurring events expanded",
		},
		[]string{"status"},
	)

	recurringEventInstances = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calendar_recurring_event_instances",
			Help:    "Number of instances per recurring event",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"freq"},
	)

	recurringEventExpansionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_recurring_event_expansion_errors_total",
			Help: "Total number of recurring event expansion errors",
		},
		[]string{"error_type"},
	)

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

	internalEventsReceivedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "calendar_sync_internal_events_received_total",
			Help: "Total number of internal events received for sync",
		},
		[]string{"event_type"},
	)

	pushToExternalDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calendar_sync_push_duration_seconds",
			Help:    "Duration of pushing events to external calendar",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)
)
