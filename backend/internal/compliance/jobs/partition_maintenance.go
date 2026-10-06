package jobs

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// PartitionMaintenanceJob periodically ensures upcoming monthly partitions are created ahead of time
type PartitionMaintenanceJob struct {
	db           *sql.DB
	monthsAhead  int
	pollInterval time.Duration
}

// NewPartitionMaintenanceJob creates a new partition maintenance job instance
func NewPartitionMaintenanceJob(db *sql.DB, monthsAhead int, pollInterval time.Duration) *PartitionMaintenanceJob {
	if monthsAhead <= 0 {
		monthsAhead = 3
	}
	if pollInterval <= 0 {
		pollInterval = 24 * time.Hour
	}
	return &PartitionMaintenanceJob{
		db:           db,
		monthsAhead:  monthsAhead,
		pollInterval: pollInterval,
	}
}

// RunOnce executes the partition maintenance function on the database
func (j *PartitionMaintenanceJob) RunOnce(ctx context.Context) error {
	_, err := j.db.ExecContext(ctx, "SELECT compliance.ensure_evaluation_partitions($1)", j.monthsAhead)
	if err != nil {
		return fmt.Errorf("ensure evaluation partitions failed: %w", err)
	}
	return nil
}

// Start begins the scheduled partition maintenance loop
func (j *PartitionMaintenanceJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.pollInterval)
	defer ticker.Stop()

	// Initial run on boot
	_ = j.RunOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = j.RunOnce(ctx)
		}
	}
}
