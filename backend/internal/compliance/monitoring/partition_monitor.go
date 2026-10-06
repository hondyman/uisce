package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AlertHandler is invoked when default partition incidents are detected
type AlertHandler func(incidentCount int64)

// DefaultPartitionMonitor checks for rows in the default partition and raises alerts
type DefaultPartitionMonitor struct {
	db           *sql.DB
	onAlert      AlertHandler
	pollInterval time.Duration
}

// NewDefaultPartitionMonitor creates a new monitor instance
func NewDefaultPartitionMonitor(db *sql.DB, onAlert AlertHandler, pollInterval time.Duration) *DefaultPartitionMonitor {
	if pollInterval <= 0 {
		pollInterval = 1 * time.Minute
	}
	return &DefaultPartitionMonitor{
		db:           db,
		onAlert:      onAlert,
		pollInterval: pollInterval,
	}
}

// CheckIncident queries v_default_partition_incident and returns the count of incident rows
func (m *DefaultPartitionMonitor) CheckIncident(ctx context.Context) (int64, error) {
	var count int64
	err := m.db.QueryRowContext(ctx, "SELECT default_partition_incident_count FROM compliance.v_default_partition_incident").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("query default partition incident count: %w", err)
	}

	if count > 0 && m.onAlert != nil {
		m.onAlert(count)
	}

	return count, nil
}

// Start begins the background monitoring loop
func (m *DefaultPartitionMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	// Run initial check
	_, _ = m.CheckIncident(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_, _ = m.CheckIncident(ctx)
		}
	}
}
