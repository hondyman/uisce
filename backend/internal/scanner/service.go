package scanner

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/db"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Scanner is an interface that defines the behavior of a metadata scanner.
type Scanner interface {
	ExtractMetadata() ([]*models.CatalogNode, []models.CatalogEdge, error)
}

// ScannerService provides scanning operations.
type ScannerService struct {
	DB *sqlx.DB
}

// NewScannerService creates a new service.
func NewScannerService(database *sqlx.DB) *ScannerService {
	return &ScannerService{DB: database}
}

// ScanDatasource orchestrates the scanning of a given datasource.
func (s *ScannerService) ScanDatasource(datasourceID string) error {
	logging.GetLogger().Sugar().Infof("Scanning datasource with ID: %s", datasourceID)

	// 1. Fetch datasource config from semlayer DB using the updated query.
	datasource, err := db.GetTenantProductDatasource(s.DB, datasourceID)
	if err != nil {
		return fmt.Errorf("failed to get datasource: %w", err)
	}

	var dbConfig db.DBConfig
	if err := json.Unmarshal(datasource.Config, &dbConfig); err != nil {
		return fmt.Errorf("failed to unmarshal db config: %w", err)
	}

	// New logic to handle gold copy mapping
	// The definition of a "gold copy" is now based on the tenant's `gold_copy` flag.
	// This assumes that the `datasource` object, fetched from `db.GetTenantProductDatasource`,
	// now includes a `TenantGoldCopy` field populated from the `tenants` table.
	isGoldCopy := datasource.TenantGoldCopy
	if !isGoldCopy {
		logging.GetLogger().Sugar().Infof("Source '%s' is not a gold copy. Attempting to find corresponding gold copy using AlphaProductID: %s and AlphaDatasourceID: %s",
			datasource.Name, datasource.AlphaProductID, datasource.AlphaDatasourceID)
		// TODO: This service is likely obsolete and needs refactoring to use the GraphQL client.
		// The function db.GetCatalogNodeMapForGoldCopy now requires a GraphQL client, which this service doesn't have.
		// Temporarily disabling this logic to allow compilation.
		logging.GetLogger().Sugar().Warn("Gold copy mapping is temporarily disabled in obsolete scanner.service.go.")
	} else {
		logging.GetLogger().Sugar().Infof("Source '%s' is a gold copy. No core_id mapping needed.", datasource.Name)
	}

	// Schema hash is not currently used for triggering ERD build,
	// but could be reinstated if conditional chart generation is needed.

	// 2. Initialize and run the appropriate scanner.
	var scanner Scanner
	var conn *sql.DB
	var connStr string

	// The switch now correctly uses the datasource's type code.
	switch datasource.DatasourceCode {
	case "postgresql", "postgres":
		sslmode := dbConfig.SSLMode
		if sslmode == "" {
			sslmode = "disable" // Default to 'disable' for flexibility
		}
		connStr = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			dbConfig.Host, dbConfig.Port, dbConfig.Auth.Basic.Username, dbConfig.Auth.Basic.Password, dbConfig.Database, sslmode)

		conn, err = sql.Open("pgx", connStr)
		if err != nil {
			return fmt.Errorf("failed to open postgres connection: %w", err)
		}
		defer conn.Close()

		if err = conn.Ping(); err != nil {
			return fmt.Errorf("could not ping source database: %w", err)
		}

		// The scanner is initialized with the specific datasource ID and its name.
		scanner, err = NewAnsiScanner(conn, datasource.TenantID, datasource.ID, datasource.Name, nil, isGoldCopy, nil)
		if err != nil {
			return fmt.Errorf("failed to initialize ansi scanner for postgres: %w", err)
		}
	default: // Fallback for unsupported types
		return fmt.Errorf("unsupported datasource type: %s", datasource.DatasourceCode)
	}

	// Log the DSN with the password redacted.
	re := regexp.MustCompile(`password=([^\s]+)`)
	redactedConnStr := re.ReplaceAllString(connStr, "password=********")
	logging.GetLogger().Sugar().Infof("Scanning datasource: %s (%s)", datasource.Name, redactedConnStr)

	nodes, edges, err := scanner.ExtractMetadata()
	if err != nil {
		return fmt.Errorf("metadata extraction failed: %w", err)
	}

	logging.GetLogger().Sugar().Infof("Extracted %d nodes and %d edges", len(nodes), len(edges))

	// 3. Save the results in a transaction.
	tx, err := s.DB.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback on error

	if err := db.InsertTempCatalogNodes(context.Background(), tx, nodes, nil); err != nil {
		return fmt.Errorf("failed to save nodes: %w", err)
	}

	if err := db.InsertTempCatalogEdges(context.Background(), tx, edges); err != nil {
		return fmt.Errorf("failed to save edges: %w", err)
	}

	// Merging data is specific to the datasource that was just scanned.
	if _, _, _, err := db.MergeCatalogData(tx, datasource.ID); err != nil {
		return fmt.Errorf("failed to merge data: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 4. Build all charts. Base charts are built first, then copied for lineage.
	logging.GetLogger().Sugar().Info("Building ERD chart...")
	if err := db.BuildERDChart(context.Background(), s.DB.DB, datasource.ID.String(), datasource.TenantGoldCopy); err != nil {
		// Log the error but don't fail the whole scan, as other charts might succeed
		logging.GetLogger().Sugar().Warnf("Warning: failed to build ERD chart: %v", err)
	}

	logging.GetLogger().Sugar().Info("Building Enhanced ERD chart...")
	if err := db.BuildEnhancedERDChart(context.Background(), s.DB.DB, datasource.ID.String(), datasource.TenantGoldCopy); err != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to build enhanced ERD chart: %v", err)
	}

	logging.GetLogger().Sugar().Info("Building Technical Lineage chart...")
	if err := db.BuildTechnicalLineageChart(context.Background(), s.DB.DB, datasource.ID.String(), datasource.TenantGoldCopy); err != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to build technical lineage chart: %v", err)
	}

	logging.GetLogger().Sugar().Info("Building Semantic Lineage chart...")
	if err := db.BuildSemanticLineageChart(context.Background(), s.DB.DB, datasource.ID.String(), datasource.TenantGoldCopy); err != nil {
		// If this fails, it might be because BuildEnhancedERDChart doesn't exist.
		// Let's try the alternative.
		logging.GetLogger().Sugar().Warnf("Warning: failed to build semantic lineage chart, trying alternative: %v", err)
		if altErr := db.BuildSemanticLineageChartAlt(context.Background(), s.DB.DB, datasource.ID.String(), datasource.TenantGoldCopy); altErr != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to build semantic lineage chart with alternative method: %v", altErr)
		}
	}

	logging.GetLogger().Sugar().Info("Scan completed successfully")
	return nil
}

// ScanAllDatasources retrieves all datasources and scans each one, prioritizing gold copies.
// It identifies gold copy datasources as those belonging to a tenant with the `gold_copy`
// flag set to true. These are scanned first to ensure their schemas are in the catalog
// before any non-gold copy datasources are processed, which is essential for mapping.
func (s *ScannerService) ScanAllDatasources() error {
	var scanErrors []string
	logging.GetLogger().Sugar().Info("Scanning all datasources, prioritizing gold copies...")

	allDatasources, err := db.GetAllTenantProductDatasources(s.DB)
	if err != nil {
		return fmt.Errorf("failed to retrieve all datasources: %w", err)
	}

	if len(allDatasources) == 0 {
		logging.GetLogger().Sugar().Info("No datasources found to scan.")
		return nil
	}

	var goldCopyDatasources []models.TenantProductDatasource
	var otherDatasources []models.TenantProductDatasource

	// This logic assumes that `models.TenantProductDatasource` has a `TenantGoldCopy` field
	// which is populated by `GetAllTenantProductDatasources` based on the tenant's `gold_copy` flag.
	for _, ds := range allDatasources {
		if ds.TenantGoldCopy {
			goldCopyDatasources = append(goldCopyDatasources, ds)
		} else {
			otherDatasources = append(otherDatasources, ds)
		}
	}

	logging.GetLogger().Sugar().Infof("Found %d gold copy datasources and %d other datasources.", len(goldCopyDatasources), len(otherDatasources))

	// 1. Scan all gold copy datasources first.
	logging.GetLogger().Sugar().Info("--- Starting scan of GOLD COPY datasources ---")
	for _, ds := range goldCopyDatasources {
		logging.GetLogger().Sugar().Infof("Initiating scan for GOLD COPY datasource ID: %s, Name: %s", ds.ID.String(), ds.Name)
		if err := s.ScanDatasource(ds.ID.String()); err != nil {
			errorMsg := fmt.Sprintf("Error scanning datasource %s (ID: %s): %v", ds.Name, ds.ID.String(), err)
			logging.GetLogger().Sugar().Warn(errorMsg)
			scanErrors = append(scanErrors, errorMsg)
			// Continue to the next datasource even if one fails
		}
	}
	logging.GetLogger().Sugar().Info("--- Finished scanning GOLD COPY datasources ---")

	// 2. Scan all other datasources.
	logging.GetLogger().Sugar().Info("--- Starting scan of NON-GOLD datasources ---")
	for _, ds := range otherDatasources {
		logging.GetLogger().Sugar().Infof("Initiating scan for datasource ID: %s, Name: %s", ds.ID.String(), ds.Name)
		if err := s.ScanDatasource(ds.ID.String()); err != nil {
			errorMsg := fmt.Sprintf("Error scanning datasource %s (ID: %s): %v", ds.Name, ds.ID.String(), err)
			logging.GetLogger().Sugar().Warn(errorMsg)
			scanErrors = append(scanErrors, errorMsg)
			// Continue to the next datasource even if one fails
		}
	}
	logging.GetLogger().Sugar().Info("--- Finished scanning NON-GOLD datasources ---")

	if len(scanErrors) > 0 {
		return fmt.Errorf("completed scan with %d errors:\n%s", len(scanErrors), strings.Join(scanErrors, "\n"))
	}

	logging.GetLogger().Sugar().Info("Completed scanning all datasources.")
	return nil
}
