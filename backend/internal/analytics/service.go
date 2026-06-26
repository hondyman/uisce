package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/hondyman/semlayer/backend/internal/ledger"
)

// AnalyticsService handles data pumping to OLAP storage (StarRocks via Iceberg)
type AnalyticsService struct {
	db *sql.DB
}

func NewAnalyticsService() *AnalyticsService {
	// Build DSN from environment variables
	host := os.Getenv("STARROCKS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	port := os.Getenv("STARROCKS_PORT")
	if port == "" {
		port = "9030"
	}
	user := os.Getenv("STARROCKS_USER")
	if user == "" {
		user = "root"
	}
	password := os.Getenv("STARROCKS_PASSWORD")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/wealth_analytics?parseTime=true", user, password, host, port)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("ERROR: Failed to connect to StarRocks: %v\n", err)
		return &AnalyticsService{}
	}

	// Configure connection pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Verify connection
	if err := db.Ping(); err != nil {
		fmt.Printf("WARNING: StarRocks connection failed ping: %v\n", err)
	}

	return &AnalyticsService{db: db}
}

// PublishToAnalytics is a Temporal Activity that pushes finalized ledger entries to StarRocks Iceberg table
func (s *AnalyticsService) PublishToAnalytics(ctx context.Context, entry ledger.LedgerEntry) error {
	if s.db == nil {
		return fmt.Errorf("starrocks connection is not initialized")
	}

	// Insert into Iceberg table via StarRocks external catalog
	query := `
		INSERT INTO iceberg_catalog.wealth.ledger_stream (
			event_time, entry_id, tenant_id, basis_id, 
			account_id, asset_id, quantity, 
			valid_from, valid_to, transaction_ref
		) VALUES (
			?, ?, ?, ?, 
			?, ?, ?, 
			?, ?, ?
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		time.Now(),
		entry.ID,
		entry.TenantID,
		entry.BasisID,
		entry.AccountID,
		entry.AssetID,
		entry.Quantity,
		entry.ValidFrom,
		entry.ValidTo,
		entry.TransactionRef,
	)

	if err != nil {
		return fmt.Errorf("failed to insert into StarRocks/Iceberg: %w", err)
	}

	return nil
}
