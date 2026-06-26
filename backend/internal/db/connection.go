package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
	_ "github.com/jackc/pgx/v5/stdlib" // PostgreSQL driver
	"github.com/jmoiron/sqlx"
)

// BasicAuth holds the username and password from the "basic" JSON object.
type BasicAuth struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// Auth contains the authentication method from the "auth" JSON object.
type Auth struct {
	Basic BasicAuth `json:"basic"`
}

// DBConfig defines the structure for unmarshalling the full database connection details.
type DBConfig struct {
	Auth     Auth   `json:"auth"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Retries  int    `json:"retries"`
	SSLMode  string `json:"sslmode"`
	Database string `json:"database"`
}

// DBStatus holds the results of a connection test attempt.
type DBStatus struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
	Attempts  int    `json:"attempts"`
	Duration  string `json:"duration"`
}

// GetDatasourceConfigByID retrieves the JSON config and type for a specific datasource.
func GetDatasourceConfigByID(db *sqlx.DB, id string) (DBConfig, string, error) {
	var configJSON []byte
	var datasourceType string
	var cfg DBConfig

	query := `
        SELECT
            tpd.config,
            ad.datasource_code
        FROM
            public.tenant_product_datasource tpd
        JOIN
            public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
        WHERE
            tpd.id = $1`

	err := db.QueryRowx(query, id).Scan(&configJSON, &datasourceType)
	if err != nil {
		if err == sql.ErrNoRows {
			return cfg, "", fmt.Errorf("no datasource found with id: %s", id)
		}
		return cfg, "", fmt.Errorf("error querying for datasource config: %w", err)
	}

	logging.GetLogger().Sugar().Debugf("Raw config JSON from database: %s", string(configJSON))
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return cfg, "", fmt.Errorf("error unmarshalling datasource config json: %w", err)
	}

	logging.GetLogger().Sugar().Debugf("datasourceType from DB: %s, config: %+v", datasourceType, cfg)
	return cfg, datasourceType, nil
}

// TestDBConnection attempts to connect to the specified external database.
func TestDBConnection(cfg DBConfig, dbType string) DBStatus {
	var dsn string
	var driver string

	dbType = strings.ToLower(strings.TrimSpace(dbType))
	logging.GetLogger().Sugar().Debugf("Normalized database type: %s", dbType)

	switch dbType {
	case "postgres", "postgresql", "database":
		driver = "pgx"
		ssl := cfg.SSLMode
		if ssl == "" {
			ssl = "disable"
		}
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.Auth.Basic.Username, cfg.Auth.Basic.Password, cfg.Database, ssl)
	default:
		return DBStatus{
			Connected: false,
			Message:   fmt.Sprintf("Unsupported database type: %s", dbType),
		}
	}

	logging.GetLogger().Sugar().Info("Attempting connection with DSN (password redacted)")
	var db *sql.DB
	var err error
	start := time.Now()

	retries := cfg.Retries
	if retries <= 0 {
		retries = 1
	}

	for attempt := 1; attempt <= retries; attempt++ {
		db, err = sql.Open(driver, dsn)
		if err == nil {
			err = db.Ping()
		}

		if err == nil {
			defer db.Close()
			status := DBStatus{
				Connected: true,
				Message:   "Connection successful",
				Attempts:  attempt,
				Duration:  time.Since(start).String(),
			}
			logging.GetLogger().Sugar().Infof("Connection successful. Status: %+v", status)
			return status
		}

		logging.GetLogger().Sugar().Warnf("Connection attempt %d for host %s failed: %v", attempt, cfg.Host, err)
		if attempt < retries {
			time.Sleep(500 * time.Millisecond)
		}
	}

	status := DBStatus{
		Connected: false,
		Message:   fmt.Errorf("connection failed after %d attempts: %v", retries, err).Error(),
		Attempts:  retries,
		Duration:  time.Since(start).String(),
	}
	logging.GetLogger().Sugar().Errorf("Connection failed. Status: %+v", status)
	return status
}

// GetTenantProductDatasource has been updated with the new query to include tenant_id.
func GetTenantProductDatasource(db *sqlx.DB, id string) (models.TenantProductDatasource, error) {
	var datasource models.TenantProductDatasource
	query := `
        SELECT
            tpd.id,
            tp.datasource_id,
            tpd.source_name AS name,
            tpd.config,
            ad.datasource_code,
            ti.tenant_id,
            tp.alpha_product_id,
            tpd.alpha_datasource_id,
            t.gold_copy AS tenant_gold_copy
        FROM
            public.tenant_product_datasource tpd
        JOIN
            public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
        JOIN
            public.tenant_product tp ON tpd.tenant_product_id = tp.id
        JOIN
            public.tenant_instance ti ON ti.id = tp.datasource_id
        JOIN
            public.tenants t ON t.id = ti.tenant_id
        WHERE tpd.id = $1`

	if err := db.Get(&datasource, query, id); err != nil {
		return datasource, fmt.Errorf("error fetching datasource: %w", err)
	}
	return datasource, nil
}

// GetAllTenantProductDatasources is also updated with the new query to include tenant_id.
func GetAllTenantProductDatasources(db *sqlx.DB) ([]models.TenantProductDatasource, error) {
	var datasources []models.TenantProductDatasource
	query := `
        SELECT
            tpd.id,
            tp.datasource_id,
            tpd.source_name AS name,
            tpd.config,
            ad.datasource_code,
            ti.tenant_id,
            tp.alpha_product_id,
            tpd.alpha_datasource_id,
            t.gold_copy AS tenant_gold_copy
        FROM
            public.tenant_product_datasource tpd
        JOIN
            public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
        JOIN
            public.tenant_product tp ON tpd.tenant_product_id = tp.id
        JOIN
            public.tenant_instance ti ON ti.id = tp.datasource_id
        JOIN
            public.tenants t ON t.id = ti.tenant_id`

	if err := db.Select(&datasources, query); err != nil {
		return nil, fmt.Errorf("error querying for all tenant product datasources: %w", err)
	}
	return datasources, nil
}

// FindGoldCopyDatasourceByAlphaIDs finds the corresponding gold copy datasource
// based on the alpha product and alpha datasource. This is more robust than name matching.
func FindGoldCopyDatasourceByAlphaIDs(db *sqlx.DB, alphaProductID, alphaDatasourceID uuid.UUID) (models.TenantProductDatasource, error) {
	var datasource models.TenantProductDatasource
	query := `
        SELECT
            tpd.id,
            tp.datasource_id,
            tpd.source_name AS name,
            tpd.config,
            ad.datasource_code,
            ti.tenant_id,
            tp.alpha_product_id,
            tpd.alpha_datasource_id,
            t.gold_copy AS tenant_gold_copy
        FROM public.tenant_product_datasource tpd
        JOIN public.tenant_product tp ON tpd.tenant_product_id = tp.id
        JOIN public.tenant_instance ti ON tp.datasource_id = ti.id
        JOIN public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
        JOIN public.tenants t ON t.id = ti.tenant_id
        WHERE t.gold_copy = true AND tp.alpha_product_id = $1 AND tpd.alpha_datasource_id = $2
        LIMIT 1`
	err := db.Get(&datasource, query, alphaProductID, alphaDatasourceID)
	return datasource, err
}
