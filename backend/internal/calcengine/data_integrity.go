package calcengine

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// ============================================================================
// BULLETPROOF HOT/COLD DATA INTEGRITY
// ============================================================================
//
// PROBLEM: How do we prevent double-counting or missing data between hot/cold?
//
// RISKS:
// 1. Query spans boundary during migration → sees partial data in both tiers
// 2. UNION ALL of hot + cold could duplicate rows if boundary isn't exact
// 3. Migration fails midway → data in neither hot nor cold
// 4. Clock skew between systems → different "cutoff" interpretations
//
// SOLUTION: Water Mark Table + Atomic Boundary + Validation
//
// ============================================================================

// DataIntegrityManager ensures no double-counting or missing data between tiers
type DataIntegrityManager struct {
	db     *sql.DB
	mu     sync.RWMutex
	config *IntegrityConfig
}

// IntegrityConfig configures data integrity checks
type IntegrityConfig struct {
	// Grace period: data in BOTH tiers during migration window
	// Queries use hot ONLY for this period to avoid double-count
	GracePeriodDays int `yaml:"grace_period_days"` // Default: 7

	// Validation settings
	ValidateOnQuery   bool `yaml:"validate_on_query"`   // Check row counts match
	ValidateOnMigrate bool `yaml:"validate_on_migrate"` // Verify before/after migration

	// Alert thresholds
	RowCountMismatchThreshold float64 `yaml:"row_count_mismatch_threshold"` // 0.001 = 0.1% tolerance
}

// WaterMark tracks the authoritative boundary between hot and cold data
// This is the SINGLE SOURCE OF TRUTH for tier boundaries
type WaterMark struct {
	TableName    string `json:"table_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`

	// The AUTHORITATIVE cutoff date
	// Hot: as_of_date >= CutoffDate
	// Cold: as_of_date < CutoffDate
	CutoffDate time.Time `json:"cutoff_date"`

	// Migration state
	State            string     `json:"state"` // STABLE, MIGRATING, VALIDATING
	MigrationStarted *time.Time `json:"migration_started,omitempty"`
	MigrationEnded   *time.Time `json:"migration_ended,omitempty"`

	// Validation checksums
	HotRowCount     int64     `json:"hot_row_count"`
	ColdRowCount    int64     `json:"cold_row_count"`
	LastValidatedAt time.Time `json:"last_validated_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// NewDataIntegrityManager creates a new integrity manager
func NewDataIntegrityManager(db *sql.DB, config *IntegrityConfig) *DataIntegrityManager {
	if config.GracePeriodDays == 0 {
		config.GracePeriodDays = 7
	}
	if config.RowCountMismatchThreshold == 0 {
		config.RowCountMismatchThreshold = 0.001 // 0.1% tolerance
	}
	return &DataIntegrityManager{
		db:     db,
		config: config,
	}
}

// ============================================================================
// WATER MARK MANAGEMENT
// ============================================================================

// GetWaterMark returns the current authoritative cutoff for a table/tenant
func (m *DataIntegrityManager) GetWaterMark(ctx context.Context, tableName, tenantID, datasourceID string) (*WaterMark, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	query := `
		SELECT table_name, tenant_id, datasource_id, cutoff_date, state,
		       migration_started, migration_ended, hot_row_count, cold_row_count,
		       last_validated_at, updated_at
		FROM semantic_hot.tier_watermarks
		WHERE table_name = ? AND tenant_id = ? AND datasource_id = ?
	`

	var wm WaterMark
	err := m.db.QueryRowContext(ctx, query, tableName, tenantID, datasourceID).Scan(
		&wm.TableName, &wm.TenantID, &wm.DatasourceID, &wm.CutoffDate, &wm.State,
		&wm.MigrationStarted, &wm.MigrationEnded, &wm.HotRowCount, &wm.ColdRowCount,
		&wm.LastValidatedAt, &wm.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		// Return default: 90 days ago, STABLE state
		return &WaterMark{
			TableName:    tableName,
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			CutoffDate:   time.Now().AddDate(0, 0, -90),
			State:        "STABLE",
			UpdatedAt:    time.Now(),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get watermark: %w", err)
	}
	return &wm, nil
}

// ============================================================================
// BULLETPROOF QUERY BUILDER
// ============================================================================

// TierQueryMode determines how to query hot/cold tiers
type TierQueryMode string

const (
	// HotOnly - Query only hot tier (safest during migration)
	HotOnly TierQueryMode = "hot_only"

	// ColdOnly - Query only cold tier
	ColdOnly TierQueryMode = "cold_only"

	// UnionSafe - UNION hot + cold with EXACT boundary from watermark
	// This is the DEFAULT and SAFE mode for spanning queries
	UnionSafe TierQueryMode = "union_safe"

	// UnionUnsafe - UNION without boundary checks (DANGEROUS - for testing only)
	UnionUnsafe TierQueryMode = "union_unsafe"
)

// TierQuery represents a query that spans hot and/or cold tiers
type TierQuery struct {
	TableName     string
	TenantID      string
	DatasourceID  string
	DateColumn    string     // Column to use for tier boundary (e.g., "as_of_date")
	StartDate     *time.Time // Optional: query start date
	EndDate       *time.Time // Optional: query end date
	Mode          TierQueryMode
	SelectColumns string // Columns to select
	WhereClause   string // Additional WHERE conditions
	GroupByClause string // Optional GROUP BY
	OrderByClause string // Optional ORDER BY
}

// BuildSafeQuery builds a bulletproof UNION query that guarantees no overlap
func (m *DataIntegrityManager) BuildSafeQuery(ctx context.Context, q *TierQuery) (string, error) {
	// Get the authoritative watermark
	wm, err := m.GetWaterMark(ctx, q.TableName, q.TenantID, q.DatasourceID)
	if err != nil {
		return "", fmt.Errorf("failed to get watermark: %w", err)
	}

	// If migration is in progress, use HOT ONLY to be safe
	if wm.State == "MIGRATING" || wm.State == "VALIDATING" {
		return m.buildHotOnlyQuery(q), nil
	}

	cutoffDate := wm.CutoffDate.Format("2006-01-02")

	switch q.Mode {
	case HotOnly:
		return m.buildHotOnlyQuery(q), nil

	case ColdOnly:
		return m.buildColdOnlyQuery(q, cutoffDate), nil

	case UnionSafe:
		return m.buildUnionSafeQuery(q, cutoffDate), nil

	default:
		return "", fmt.Errorf("unsupported query mode: %s", q.Mode)
	}
}

func (m *DataIntegrityManager) buildHotOnlyQuery(q *TierQuery) string {
	return fmt.Sprintf(`
		SELECT %s
		FROM semantic_hot.%s
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  %s
		%s
		%s
	`, q.SelectColumns, q.TableName, q.TenantID, q.DatasourceID,
		m.buildDateFilter(q, ""),
		m.buildGroupBy(q),
		m.buildOrderBy(q))
}

func (m *DataIntegrityManager) buildColdOnlyQuery(q *TierQuery, cutoffDate string) string {
	return fmt.Sprintf(`
		SELECT %s
		FROM semantic_cold.%s
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND %s < '%s'
		  %s
		%s
		%s
	`, q.SelectColumns, q.TableName, q.TenantID, q.DatasourceID,
		q.DateColumn, cutoffDate,
		m.buildDateFilter(q, cutoffDate),
		m.buildGroupBy(q),
		m.buildOrderBy(q))
}

// buildUnionSafeQuery builds a UNION query with EXACT boundary enforcement
// KEY INSIGHT: We use EXPLICIT date filters on BOTH sides to prevent ANY overlap
func (m *DataIntegrityManager) buildUnionSafeQuery(q *TierQuery, cutoffDate string) string {
	hotDateFilter := m.buildHotDateFilter(q, cutoffDate)
	coldDateFilter := m.buildColdDateFilter(q, cutoffDate)

	return fmt.Sprintf(`
		-- HOT TIER: Data >= cutoff date (authoritative from watermark)
		SELECT %s, 'hot' as _data_tier
		FROM semantic_hot.%s
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND %s >= '%s'  -- CRITICAL: Explicit boundary
		  %s
		
		UNION ALL
		
		-- COLD TIER: Data < cutoff date (authoritative from watermark)
		SELECT %s, 'cold' as _data_tier
		FROM semantic_cold.%s
		WHERE tenant_id = '%s'
		  AND datasource_id = '%s'
		  AND %s < '%s'   -- CRITICAL: Explicit boundary (< not <=)
		  %s
		%s
		%s
	`,
		// Hot tier
		q.SelectColumns, q.TableName, q.TenantID, q.DatasourceID,
		q.DateColumn, cutoffDate, hotDateFilter,
		// Cold tier
		q.SelectColumns, q.TableName, q.TenantID, q.DatasourceID,
		q.DateColumn, cutoffDate, coldDateFilter,
		// Outer clauses
		m.buildGroupBy(q),
		m.buildOrderBy(q))
}

func (m *DataIntegrityManager) buildHotDateFilter(q *TierQuery, cutoffDate string) string {
	if q.StartDate == nil && q.EndDate == nil {
		return ""
	}

	var filter string
	if q.StartDate != nil {
		startDate := q.StartDate.Format("2006-01-02")
		// Use MAX of start date and cutoff for hot tier
		if q.StartDate.Before(parseDate(cutoffDate)) {
			filter += fmt.Sprintf(" AND %s >= '%s'", q.DateColumn, cutoffDate)
		} else {
			filter += fmt.Sprintf(" AND %s >= '%s'", q.DateColumn, startDate)
		}
	}
	if q.EndDate != nil {
		filter += fmt.Sprintf(" AND %s <= '%s'", q.DateColumn, q.EndDate.Format("2006-01-02"))
	}
	return filter
}

func (m *DataIntegrityManager) buildColdDateFilter(q *TierQuery, cutoffDate string) string {
	if q.StartDate == nil && q.EndDate == nil {
		return ""
	}

	var filter string
	if q.StartDate != nil {
		filter += fmt.Sprintf(" AND %s >= '%s'", q.DateColumn, q.StartDate.Format("2006-01-02"))
	}
	if q.EndDate != nil {
		endDate := q.EndDate.Format("2006-01-02")
		// Use MIN of end date and cutoff-1 for cold tier
		if q.EndDate.After(parseDate(cutoffDate)) {
			// End date is after cutoff, so cold should go up to cutoff-1day
			coldEnd := parseDate(cutoffDate).AddDate(0, 0, -1).Format("2006-01-02")
			filter += fmt.Sprintf(" AND %s <= '%s'", q.DateColumn, coldEnd)
		} else {
			filter += fmt.Sprintf(" AND %s <= '%s'", q.DateColumn, endDate)
		}
	}
	return filter
}

func (m *DataIntegrityManager) buildDateFilter(q *TierQuery, _ string) string {
	if q.StartDate == nil && q.EndDate == nil {
		return ""
	}
	var filter string
	if q.StartDate != nil {
		filter += fmt.Sprintf(" AND %s >= '%s'", q.DateColumn, q.StartDate.Format("2006-01-02"))
	}
	if q.EndDate != nil {
		filter += fmt.Sprintf(" AND %s <= '%s'", q.DateColumn, q.EndDate.Format("2006-01-02"))
	}
	return filter
}

func (m *DataIntegrityManager) buildGroupBy(q *TierQuery) string {
	if q.GroupByClause == "" {
		return ""
	}
	return "GROUP BY " + q.GroupByClause
}

func (m *DataIntegrityManager) buildOrderBy(q *TierQuery) string {
	if q.OrderByClause == "" {
		return ""
	}
	return "ORDER BY " + q.OrderByClause
}

func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

// ============================================================================
// MIGRATION WITH VALIDATION
// ============================================================================

// IntegrityMigrationResult tracks migration outcome with validation
// (Named differently to avoid conflict with MigrationResult in unified_engine.go)
type IntegrityMigrationResult struct {
	TableName    string `json:"table_name"`
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`

	OldCutoff time.Time `json:"old_cutoff"`
	NewCutoff time.Time `json:"new_cutoff"`

	RowsMigrated     int64 `json:"rows_migrated"`
	RowsInHotBefore  int64 `json:"rows_in_hot_before"`
	RowsInHotAfter   int64 `json:"rows_in_hot_after"`
	RowsInColdBefore int64 `json:"rows_in_cold_before"`
	RowsInColdAfter  int64 `json:"rows_in_cold_after"`

	ValidationPassed bool     `json:"validation_passed"`
	ValidationErrors []string `json:"validation_errors,omitempty"`

	Duration    time.Duration `json:"duration"`
	CompletedAt time.Time     `json:"completed_at"`
}

// MigrateWithValidation performs a BULLETPROOF migration:
// 1. Set state to MIGRATING (queries fall back to hot-only)
// 2. Export data to Parquet (cold tier)
// 3. Validate row counts match
// 4. Update watermark atomically
// 5. Delete from hot tier
// 6. Set state to STABLE
func (m *DataIntegrityManager) MigrateWithValidation(ctx context.Context, tableName, tenantID, datasourceID string, newCutoff time.Time) (*IntegrityMigrationResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := &IntegrityMigrationResult{
		TableName:    tableName,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		NewCutoff:    newCutoff,
	}
	startTime := time.Now()

	// Step 1: Get current watermark
	wm, err := m.getWaterMarkNoLock(ctx, tableName, tenantID, datasourceID)
	if err != nil {
		return nil, err
	}
	result.OldCutoff = wm.CutoffDate

	// Step 2: Set state to MIGRATING
	if err := m.setMigrationState(ctx, tableName, tenantID, datasourceID, "MIGRATING"); err != nil {
		return nil, fmt.Errorf("failed to set migration state: %w", err)
	}
	defer func() {
		// Always try to set back to STABLE on exit
		_ = m.setMigrationState(ctx, tableName, tenantID, datasourceID, "STABLE")
	}()

	// Step 3: Count rows BEFORE migration
	result.RowsInHotBefore, _ = m.countRows(ctx, "semantic_hot", tableName, tenantID, datasourceID, nil, nil)
	result.RowsInColdBefore, _ = m.countRows(ctx, "semantic_cold", tableName, tenantID, datasourceID, nil, nil)

	// Step 4: Count rows to migrate (between old and new cutoff)
	result.RowsMigrated, _ = m.countRows(ctx, "semantic_hot", tableName, tenantID, datasourceID, &wm.CutoffDate, &newCutoff)

	// Step 5: Export to Parquet (external table refresh happens automatically)
	// This is done by writing to S3/HDFS which external table points to
	if err := m.exportToParquet(ctx, tableName, tenantID, datasourceID, wm.CutoffDate, newCutoff); err != nil {
		return nil, fmt.Errorf("export to parquet failed: %w", err)
	}

	// Step 6: Validate data exists in cold tier
	if err := m.setMigrationState(ctx, tableName, tenantID, datasourceID, "VALIDATING"); err != nil {
		return nil, err
	}

	coldCount, err := m.countRows(ctx, "semantic_cold", tableName, tenantID, datasourceID, &wm.CutoffDate, &newCutoff)
	if err != nil {
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("failed to count cold rows: %v", err))
	}

	// Validate counts match within tolerance
	if !m.validateRowCounts(result.RowsMigrated, coldCount) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("row count mismatch: hot=%d, cold=%d", result.RowsMigrated, coldCount))
		return result, fmt.Errorf("validation failed: row count mismatch")
	}

	// Step 7: Update watermark ATOMICALLY
	if err := m.updateWaterMark(ctx, tableName, tenantID, datasourceID, newCutoff); err != nil {
		return nil, fmt.Errorf("failed to update watermark: %w", err)
	}

	// Step 8: Delete from hot tier (data now in cold)
	if err := m.deleteFromHot(ctx, tableName, tenantID, datasourceID, wm.CutoffDate, newCutoff); err != nil {
		// CRITICAL: Watermark already updated, so queries will go to cold
		// Data might be duplicated temporarily but NOT lost
		result.ValidationErrors = append(result.ValidationErrors, fmt.Sprintf("hot deletion failed: %v", err))
	}

	// Step 9: Final counts
	result.RowsInHotAfter, _ = m.countRows(ctx, "semantic_hot", tableName, tenantID, datasourceID, nil, nil)
	result.RowsInColdAfter, _ = m.countRows(ctx, "semantic_cold", tableName, tenantID, datasourceID, nil, nil)

	// Validate total row count unchanged
	totalBefore := result.RowsInHotBefore + result.RowsInColdBefore
	totalAfter := result.RowsInHotAfter + result.RowsInColdAfter
	if !m.validateRowCounts(totalBefore, totalAfter) {
		result.ValidationErrors = append(result.ValidationErrors,
			fmt.Sprintf("total row count changed: before=%d, after=%d", totalBefore, totalAfter))
	}

	result.ValidationPassed = len(result.ValidationErrors) == 0
	result.Duration = time.Since(startTime)
	result.CompletedAt = time.Now()

	return result, nil
}

func (m *DataIntegrityManager) getWaterMarkNoLock(ctx context.Context, tableName, tenantID, datasourceID string) (*WaterMark, error) {
	query := `
		SELECT cutoff_date, state FROM semantic_hot.tier_watermarks
		WHERE table_name = ? AND tenant_id = ? AND datasource_id = ?
	`
	var wm WaterMark
	wm.TableName = tableName
	wm.TenantID = tenantID
	wm.DatasourceID = datasourceID

	err := m.db.QueryRowContext(ctx, query, tableName, tenantID, datasourceID).Scan(&wm.CutoffDate, &wm.State)
	if err == sql.ErrNoRows {
		wm.CutoffDate = time.Now().AddDate(0, 0, -90)
		wm.State = "STABLE"
		return &wm, nil
	}
	return &wm, err
}

func (m *DataIntegrityManager) setMigrationState(ctx context.Context, tableName, tenantID, datasourceID, state string) error {
	query := `
		INSERT INTO semantic_hot.tier_watermarks (table_name, tenant_id, datasource_id, state, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE state = ?, updated_at = NOW()
	`
	_, err := m.db.ExecContext(ctx, query, tableName, tenantID, datasourceID, state, state)
	return err
}

func (m *DataIntegrityManager) updateWaterMark(ctx context.Context, tableName, tenantID, datasourceID string, cutoff time.Time) error {
	query := `
		INSERT INTO semantic_hot.tier_watermarks (table_name, tenant_id, datasource_id, cutoff_date, state, updated_at)
		VALUES (?, ?, ?, ?, 'STABLE', NOW())
		ON DUPLICATE KEY UPDATE cutoff_date = ?, state = 'STABLE', updated_at = NOW()
	`
	_, err := m.db.ExecContext(ctx, query, tableName, tenantID, datasourceID, cutoff, cutoff)
	return err
}

func (m *DataIntegrityManager) countRows(ctx context.Context, schema, tableName, tenantID, datasourceID string, startDate, endDate *time.Time) (int64, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s.%s WHERE tenant_id = ? AND datasource_id = ?", schema, tableName)
	args := []interface{}{tenantID, datasourceID}

	if startDate != nil {
		query += " AND as_of_date >= ?"
		args = append(args, startDate.Format("2006-01-02"))
	}
	if endDate != nil {
		query += " AND as_of_date < ?"
		args = append(args, endDate.Format("2006-01-02"))
	}

	var count int64
	err := m.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

func (m *DataIntegrityManager) validateRowCounts(expected, actual int64) bool {
	if expected == 0 && actual == 0 {
		return true
	}
	if expected == 0 || actual == 0 {
		return false
	}
	diff := float64(expected-actual) / float64(expected)
	if diff < 0 {
		diff = -diff
	}
	return diff <= m.config.RowCountMismatchThreshold
}

func (m *DataIntegrityManager) exportToParquet(ctx context.Context, tableName, tenantID, datasourceID string, startDate, endDate time.Time) error {
	// StarRocks EXPORT TABLE command to write Parquet to S3
	exportSQL := fmt.Sprintf(`
		EXPORT TABLE semantic_hot.%s
		WHERE tenant_id = '%s' AND datasource_id = '%s'
		  AND as_of_date >= '%s' AND as_of_date < '%s'
		TO 's3://your-bucket/semantic_cold/%s/%s/%s/%s/'
		PROPERTIES (
			"format" = "parquet",
			"column_separator" = ",",
			"max_file_size" = "256MB"
		)
	`, tableName, tenantID, datasourceID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"),
		tableName, tenantID, datasourceID, endDate.Format("2006"))

	_, err := m.db.ExecContext(ctx, exportSQL)
	return err
}

func (m *DataIntegrityManager) deleteFromHot(ctx context.Context, tableName, tenantID, datasourceID string, startDate, endDate time.Time) error {
	deleteSQL := fmt.Sprintf(`
		DELETE FROM semantic_hot.%s
		WHERE tenant_id = '%s' AND datasource_id = '%s'
		  AND as_of_date >= '%s' AND as_of_date < '%s'
	`, tableName, tenantID, datasourceID,
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	_, err := m.db.ExecContext(ctx, deleteSQL)
	return err
}

// ============================================================================
// INTEGRITY VALIDATION QUERIES
// ============================================================================

// ValidateNoOverlap checks there's no data overlap between hot and cold
func (m *DataIntegrityManager) ValidateNoOverlap(ctx context.Context, tableName, tenantID, datasourceID string) error {
	wm, err := m.GetWaterMark(ctx, tableName, tenantID, datasourceID)
	if err != nil {
		return err
	}

	// Check for rows in hot that should be in cold
	hotOverlapQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM semantic_hot.%s 
		WHERE tenant_id = ? AND datasource_id = ? AND as_of_date < ?
	`, tableName)

	var hotOverlap int64
	if err := m.db.QueryRowContext(ctx, hotOverlapQuery, tenantID, datasourceID, wm.CutoffDate).Scan(&hotOverlap); err != nil {
		return err
	}
	if hotOverlap > 0 {
		return fmt.Errorf("INTEGRITY ERROR: %d rows in hot tier should be in cold (before cutoff %s)",
			hotOverlap, wm.CutoffDate.Format("2006-01-02"))
	}

	// Check for rows in cold that should be in hot
	coldOverlapQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM semantic_cold.%s 
		WHERE tenant_id = ? AND datasource_id = ? AND as_of_date >= ?
	`, tableName)

	var coldOverlap int64
	if err := m.db.QueryRowContext(ctx, coldOverlapQuery, tenantID, datasourceID, wm.CutoffDate).Scan(&coldOverlap); err != nil {
		return err
	}
	if coldOverlap > 0 {
		return fmt.Errorf("INTEGRITY ERROR: %d rows in cold tier should be in hot (at/after cutoff %s)",
			coldOverlap, wm.CutoffDate.Format("2006-01-02"))
	}

	return nil
}

// ValidateTotalRowCount ensures no data was lost or duplicated
func (m *DataIntegrityManager) ValidateTotalRowCount(ctx context.Context, tableName, tenantID, datasourceID string, expectedTotal int64) error {
	hotCount, err := m.countRows(ctx, "semantic_hot", tableName, tenantID, datasourceID, nil, nil)
	if err != nil {
		return err
	}
	coldCount, err := m.countRows(ctx, "semantic_cold", tableName, tenantID, datasourceID, nil, nil)
	if err != nil {
		return err
	}

	actualTotal := hotCount + coldCount
	if !m.validateRowCounts(expectedTotal, actualTotal) {
		return fmt.Errorf("INTEGRITY ERROR: total row count mismatch - expected %d, got %d (hot=%d, cold=%d)",
			expectedTotal, actualTotal, hotCount, coldCount)
	}

	return nil
}
