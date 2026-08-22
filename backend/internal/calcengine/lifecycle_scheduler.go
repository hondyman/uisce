package calcengine

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// DATA LIFECYCLE: Hot → Cold Migration
// ============================================================================
// Runs on schedule (e.g., nightly at 2 AM)
// ============================================================================

// LifecycleScheduler manages hot-to-cold data migration
type LifecycleScheduler struct {
	engine  *UnifiedCalcEngine
	config  *LifecycleConfig
	running bool
	stopCh  chan struct{}
	mu      sync.Mutex
}

// LifecycleConfig configures the data lifecycle scheduler
type LifecycleConfig struct {
	// Schedule (cron-like)
	RunAt       string        `yaml:"run_at"`       // "02:00" for 2 AM
	RunInterval time.Duration `yaml:"run_interval"` // Alternative: every N hours

	// Tables to migrate
	Tables []string `yaml:"tables"`

	// Retention
	HotRetention time.Duration `yaml:"hot_retention"` // Default: 90 days

	// Migration settings
	BatchSize   int    `yaml:"batch_size"`   // Rows per batch
	ParquetPath string `yaml:"parquet_path"` // S3/HDFS path

	// Notifications
	SlackWebhook   string `yaml:"slack_webhook"`
	EmailRecipient string `yaml:"email_recipient"`
}

// NewLifecycleScheduler creates a new lifecycle scheduler
func NewLifecycleScheduler(engine *UnifiedCalcEngine, config *LifecycleConfig) *LifecycleScheduler {
	if config.HotRetention == 0 {
		config.HotRetention = 90 * 24 * time.Hour // 90 days
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100000
	}
	if len(config.Tables) == 0 {
		config.Tables = []string{
			"holdings",
			"portfolio_nav",
			"transactions",
			"prices",
		}
	}

	return &LifecycleScheduler{
		engine: engine,
		config: config,
		stopCh: make(chan struct{}),
	}
}

// Start begins the lifecycle scheduler
func (s *LifecycleScheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	go s.runScheduler()
	return nil
}

// Stop stops the lifecycle scheduler
func (s *LifecycleScheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		close(s.stopCh)
		s.running = false
	}
}

// RunNow triggers an immediate migration (for manual runs)
func (s *LifecycleScheduler) RunNow(ctx context.Context) (*MigrationResult, error) {
	return s.runMigration(ctx)
}

// runScheduler is the main scheduler loop
func (s *LifecycleScheduler) runScheduler() {
	// Calculate next run time
	nextRun := s.calculateNextRun()

	for {
		select {
		case <-s.stopCh:
			return
		case <-time.After(time.Until(nextRun)):
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Hour)
			result, err := s.runMigration(ctx)
			cancel()

			if err != nil {
				s.notifyError(err)
			} else {
				s.notifySuccess(result)
			}

			nextRun = s.calculateNextRun()
		}
	}
}

// runMigration executes the hot-to-cold migration
// Note: Cube.js coordination has been removed
func (s *LifecycleScheduler) runMigration(ctx context.Context) (*MigrationResult, error) {
	result := &MigrationResult{
		StartedAt: time.Now(),
		Tables:    make(map[string]TableMigrationResult),
	}

	// Step 1: Run migration for each table
	cutoffDate := time.Now().Add(-s.config.HotRetention)

	for _, table := range s.config.Tables {
		tableResult, err := s.migrateTable(ctx, table, cutoffDate)
		if err != nil {
			tableResult.Error = err.Error()
		}
		result.Tables[table] = tableResult
		result.TotalRowsMigrated += tableResult.RowsMigrated
	}

	result.CompletedAt = time.Now()
	result.Duration = result.CompletedAt.Sub(result.StartedAt)

	return result, nil
}

// migrateTable migrates a single table from hot to cold
func (s *LifecycleScheduler) migrateTable(ctx context.Context, table string, cutoffDate time.Time) (TableMigrationResult, error) {
	result := TableMigrationResult{}

	// Export to Parquet
	exportSQL := fmt.Sprintf(`
		EXPORT TABLE %s.%s
		WHERE as_of_date < '%s'
		TO '%s/%s/%s/'
		PROPERTIES (
			"format" = "parquet",
			"max_file_size" = "256MB",
			"column_separator" = ","
		)
	`, s.engine.config.HotDatabase, table, cutoffDate.Format("2006-01-02"),
		s.config.ParquetPath, table, cutoffDate.Format("2006"))

	if _, err := s.engine.starrocks.ExecContext(ctx, exportSQL); err != nil {
		return result, fmt.Errorf("export failed: %w", err)
	}

	// Count rows to migrate
	countSQL := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s.%s WHERE as_of_date < '%s'
	`, s.engine.config.HotDatabase, table, cutoffDate.Format("2006-01-02"))

	if err := s.engine.starrocks.QueryRowContext(ctx, countSQL).Scan(&result.RowsMigrated); err != nil {
		return result, fmt.Errorf("count failed: %w", err)
	}

	// Delete from hot table in batches to avoid long locks
	for {
		deleteSQL := fmt.Sprintf(`
			DELETE FROM %s.%s 
			WHERE as_of_date < '%s'
			LIMIT %d
		`, s.engine.config.HotDatabase, table, cutoffDate.Format("2006-01-02"), s.config.BatchSize)

		res, err := s.engine.starrocks.ExecContext(ctx, deleteSQL)
		if err != nil {
			return result, fmt.Errorf("delete failed: %w", err)
		}

		affected, _ := res.RowsAffected()
		result.RowsDeleted += affected

		if affected < int64(s.config.BatchSize) {
			break // No more rows to delete
		}

		// Brief pause between batches to allow queries
		time.Sleep(100 * time.Millisecond)
	}

	// Refresh external table metadata
	refreshSQL := fmt.Sprintf(`REFRESH EXTERNAL TABLE %s.%s`, s.engine.config.ColdDatabase, table)
	_, _ = s.engine.starrocks.ExecContext(ctx, refreshSQL)

	return result, nil
}

// calculateNextRun determines the next scheduled run time
func (s *LifecycleScheduler) calculateNextRun() time.Time {
	if s.config.RunInterval > 0 {
		return time.Now().Add(s.config.RunInterval)
	}

	// Parse RunAt time (e.g., "02:00")
	now := time.Now()
	runTime, err := time.Parse("15:04", s.config.RunAt)
	if err != nil {
		// Default to 2 AM
		runTime, _ = time.Parse("15:04", "02:00")
	}

	next := time.Date(now.Year(), now.Month(), now.Day(),
		runTime.Hour(), runTime.Minute(), 0, 0, now.Location())

	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	return next
}

// Notification helpers
func (s *LifecycleScheduler) notifyError(err error) {
	msg := fmt.Sprintf("🚨 Data lifecycle migration failed: %v", err)
	fmt.Println(msg)
	// TODO: Send to Slack/Email
}

func (s *LifecycleScheduler) notifySuccess(result *MigrationResult) {
	msg := fmt.Sprintf("Data lifecycle migration completed: %d rows migrated in %v",
		result.TotalRowsMigrated, result.Duration)
	fmt.Println(msg)
	// TODO: Send to Slack/Email
}

// ============================================================================
// STATUS API - Check Migration Status
// ============================================================================

// MigrationStatus represents the current migration status
type MigrationStatus struct {
	IsRunning        bool             `json:"is_running"`
	LastRun          *time.Time       `json:"last_run,omitempty"`
	LastResult       *MigrationResult `json:"last_result,omitempty"`
	NextScheduledRun time.Time        `json:"next_scheduled_run"`
	Config           *LifecycleConfig `json:"config"`
}

// GetStatus returns the current migration status
func (s *LifecycleScheduler) GetStatus() *MigrationStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return &MigrationStatus{
		IsRunning:        s.running,
		NextScheduledRun: s.calculateNextRun(),
		Config:           s.config,
	}
}
