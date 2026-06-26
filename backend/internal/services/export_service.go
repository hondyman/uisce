package services

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/tenant"
)

// ExportFormat specifies the output format for exports
type ExportFormat string

const (
	ExportFormatCSV     ExportFormat = "csv"
	ExportFormatJSON    ExportFormat = "json"
	ExportFormatParquet ExportFormat = "parquet"
)

// ExportService handles exporting job results in various formats
type ExportService interface {
	// CreateExport queues a new export job
	CreateExport(ctx context.Context, jobID uuid.UUID, format ExportFormat, filterCriteria map[string]interface{}) (uuid.UUID, error)

	// GetExportStatus retrieves export job status
	GetExportStatus(ctx context.Context, exportID uuid.UUID) (*models.JobExport, error)

	// GetDownloadURL generates a presigned download URL
	GetDownloadURL(ctx context.Context, exportID uuid.UUID, expiryHours int) (string, error)

	// ListExports lists exports for a job
	ListExports(ctx context.Context, jobID uuid.UUID) ([]*models.JobExport, error)

	// DownloadExport returns the export file content
	DownloadExport(ctx context.Context, exportID uuid.UUID) (io.ReadCloser, string, error)

	// ProcessExport executes the export (called by background processor)
	ProcessExport(ctx context.Context, exportID uuid.UUID) error
}

// PostgresExportService implements ExportService using PostgreSQL
type PostgresExportService struct {
	db                *sql.DB
	exportStoragePath string
	urlBasePath       string
}

// NewPostgresExportService creates a new export service
func NewPostgresExportService(db *sql.DB, storagePath, urlBasePath string) *PostgresExportService {
	// Create storage directory if it doesn't exist
	os.MkdirAll(storagePath, 0755)

	return &PostgresExportService{
		db:                db,
		exportStoragePath: storagePath,
		urlBasePath:       urlBasePath,
	}
}

// CreateExport creates a new export job
func (s *PostgresExportService) CreateExport(ctx context.Context, jobID uuid.UUID, format ExportFormat, filterCriteria map[string]interface{}) (uuid.UUID, error) {
	exportID := uuid.New()

	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to extract tenant: %w", err)
	}

	// Get job details to find user ID for created_by
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Set RLS context
	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return uuid.Nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Get job to retrieve created_by
	var createdBy uuid.UUID
	err = tx.QueryRowContext(ctx, `
		SELECT created_by FROM edm.async_jobs WHERE id = $1
	`, jobID).Scan(&createdBy)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get job details: %w", err)
	}

	// Convert filter criteria to JSON
	filterJSON, err := json.Marshal(filterCriteria)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to marshal filter criteria: %w", err)
	}

	// Insert export record
	_, err = tx.ExecContext(ctx, `
		INSERT INTO edm.job_exports (
			id, job_id, tenant_id, export_format, status,
			filter_criteria, created_by, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, exportID, jobID, tenantID, format, "queued", filterJSON, createdBy, time.Now(), time.Now().AddDate(0, 0, 7))

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to insert export record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return uuid.Nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return exportID, nil
}

// GetExportStatus retrieves export status
func (s *PostgresExportService) GetExportStatus(ctx context.Context, exportID uuid.UUID) (*models.JobExport, error) {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	export := &models.JobExport{}
	var filterJSON []byte

	err = tx.QueryRowContext(ctx, `
		SELECT id, job_id, tenant_id, export_format, status, file_location, file_size, record_count,
		       presigned_url, presigned_url_expires, download_count, filter_criteria, created_at,
		       started_at, completed_at, expires_at, error_message, include_errors
		FROM edm.job_exports
		WHERE id = $1 AND tenant_id = $2
	`, exportID, tenantID).Scan(
		&export.ID, &export.JobID, &export.TenantID, &export.ExportFormat, &export.Status,
		&export.FileLocation, &export.FileSize, &export.RecordCount, &export.PresignedURL,
		&export.PresignedURLExpires, &export.DownloadCount, &filterJSON, &export.CreatedAt,
		&export.StartedAt, &export.CompletedAt, &export.ExpiresAt, &export.ErrorMessage, &export.IncludeErrors,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("export not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query export: %w", err)
	}

	// Parse filter criteria
	if filterJSON != nil {
		json.Unmarshal(filterJSON, &export.FilterCriteria)
	}

	tx.Commit()
	return export, nil
}

// GetDownloadURL generates a presigned URL
func (s *PostgresExportService) GetDownloadURL(ctx context.Context, exportID uuid.UUID, expiryHours int) (string, error) {
	export, err := s.GetExportStatus(ctx, exportID)
	if err != nil {
		return "", err
	}

	if export.Status != "completed" {
		return "", fmt.Errorf("export not ready for download: status=%s", export.Status)
	}

	// Generate presigned URL (simple hash-based)
	token := uuid.New().String()
	expiresAt := time.Now().AddDate(0, 0, 0).Add(time.Duration(expiryHours) * time.Hour)
	baseURL := fmt.Sprintf("%s/api/v1/exports/%s/download?token=%s", s.urlBasePath, exportID, token)

	// Store presigned URL in database
	tenantID, _ := tenant.ExtractTenantFromContext(ctx)
	tx, _ := s.db.BeginTx(ctx, nil)
	defer tx.Rollback()

	tenant.SetRLSContext(ctx, tx, tenantID.String())

	tx.ExecContext(ctx, `
		UPDATE edm.job_exports
		SET presigned_url = $1, presigned_url_expires = $2
		WHERE id = $3
	`, baseURL, expiresAt, exportID)

	tx.Commit()

	return baseURL, nil
}

// ListExports lists all exports for a job
func (s *PostgresExportService) ListExports(ctx context.Context, jobID uuid.UUID) ([]*models.JobExport, error) {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to extract tenant: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return nil, fmt.Errorf("failed to set RLS context: %w", err)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT id, job_id, tenant_id, export_format, status, file_location, file_size, record_count,
		       created_at, completed_at, expires_at, error_message
		FROM edm.job_exports
		WHERE job_id = $1 AND tenant_id = $2
		ORDER BY created_at DESC
	`, jobID, tenantID)

	if err != nil {
		return nil, fmt.Errorf("failed to query exports: %w", err)
	}
	defer rows.Close()

	exports := []*models.JobExport{}
	for rows.Next() {
		export := &models.JobExport{}
		err := rows.Scan(
			&export.ID, &export.JobID, &export.TenantID, &export.ExportFormat,
			&export.Status, &export.FileLocation, &export.FileSize, &export.RecordCount,
			&export.CreatedAt, &export.CompletedAt, &export.ExpiresAt, &export.ErrorMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan export: %w", err)
		}
		exports = append(exports, export)
	}

	tx.Commit()
	return exports, nil
}

// DownloadExport retrieves the export file
func (s *PostgresExportService) DownloadExport(ctx context.Context, exportID uuid.UUID) (io.ReadCloser, string, error) {
	export, err := s.GetExportStatus(ctx, exportID)
	if err != nil {
		return nil, "", err
	}

	if export.Status != "completed" {
		return nil, "", fmt.Errorf("export not completed: status=%s", export.Status)
	}

	if export.FileLocation == "" {
		return nil, "", fmt.Errorf("no file location available")
	}

	file, err := os.Open(export.FileLocation)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open export file: %w", err)
	}

	contentType := getContentType(ExportFormat(export.ExportFormat))
	return file, contentType, nil
}

// ProcessExport executes the actual export
func (s *PostgresExportService) ProcessExport(ctx context.Context, exportID uuid.UUID) error {
	tenantID, err := tenant.ExtractTenantFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to extract tenant: %w", err)
	}

	// Mark as processing
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := tenant.SetRLSContext(ctx, tx, tenantID.String()); err != nil {
		return fmt.Errorf("failed to set RLS context: %w", err)
	}

	// Get export details
	export := &models.JobExport{}
	var filterJSON []byte

	err = tx.QueryRowContext(ctx, `
		SELECT id, job_id, export_format, filter_criteria
		FROM edm.job_exports
		WHERE id = $1
	`, exportID).Scan(&export.ID, &export.JobID, &export.ExportFormat, &filterJSON)

	if err != nil {
		return fmt.Errorf("failed to get export: %w", err)
	}

	// Update status to processing
	_, _ = tx.ExecContext(ctx, `
		UPDATE edm.job_exports
		SET status = 'processing', started_at = NOW()
		WHERE id = $1
	`, exportID)
	tx.Commit()

	// Get job results
	results, err := s.getJobResults(ctx, export.JobID, filterJSON)
	if err != nil {
		s.recordExportError(ctx, exportID, err.Error())
		return fmt.Errorf("failed to get job results: %w", err)
	}

	// Create export file
	filePath := filepath.Join(s.exportStoragePath, fmt.Sprintf("%s.%s", exportID, export.ExportFormat))

	switch ExportFormat(export.ExportFormat) {
	case ExportFormatCSV:
		err = s.exportAsCSV(filePath, results)
	case ExportFormatJSON:
		err = s.exportAsJSON(filePath, results)
	default:
		err = fmt.Errorf("unsupported export format: %s", export.ExportFormat)
	}

	if err != nil {
		s.recordExportError(ctx, exportID, err.Error())
		return fmt.Errorf("failed to export data: %w", err)
	}

	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		s.recordExportError(ctx, exportID, err.Error())
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Mark as completed
	tx, _ = s.db.BeginTx(ctx, nil)
	defer tx.Rollback()
	tenant.SetRLSContext(ctx, tx, tenantID.String())

	_, _ = tx.ExecContext(ctx, `
		UPDATE edm.job_exports
		SET status = 'completed', completed_at = NOW(), file_location = $1,
		    file_size = $2, record_count = $3, expires_at = NOW() + INTERVAL '7 days'
		WHERE id = $4
	`, filePath, info.Size(), len(results), exportID)

	tx.Commit()
	return nil
}

// Helper functions

func (s *PostgresExportService) getJobResults(ctx context.Context, jobID uuid.UUID, filterJSON []byte) ([]map[string]interface{}, error) {
	// This would query the job_items table and format results
	// Implementation would depend on the specific job type and data structure
	results := []map[string]interface{}{}
	return results, nil
}

func (s *PostgresExportService) exportAsCSV(filePath string, results []map[string]interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if len(results) == 0 {
		return nil
	}

	// Get headers from first record
	headers := []string{}
	for key := range results[0] {
		headers = append(headers, key)
	}
	writer.Write(headers)

	// Write records
	for _, record := range results {
		row := []string{}
		for _, header := range headers {
			val := record[header]
			row = append(row, fmt.Sprintf("%v", val))
		}
		writer.Write(row)
	}

	return nil
}

func (s *PostgresExportService) exportAsJSON(filePath string, results []map[string]interface{}) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

func (s *PostgresExportService) recordExportError(ctx context.Context, exportID uuid.UUID, errMsg string) {
	tenantID, _ := tenant.ExtractTenantFromContext(ctx)
	tx, _ := s.db.BeginTx(ctx, nil)
	defer tx.Rollback()

	tenant.SetRLSContext(ctx, tx, tenantID.String())
	tx.ExecContext(ctx, `
		UPDATE edm.job_exports
		SET status = 'failed', error_message = $1, completed_at = NOW()
		WHERE id = $2
	`, errMsg, exportID)
	tx.Commit()
}

func getContentType(format ExportFormat) string {
	switch format {
	case ExportFormatCSV:
		return "text/csv"
	case ExportFormatJSON:
		return "application/json"
	case ExportFormatParquet:
		return "application/x-parquet"
	default:
		return "application/octet-stream"
	}
}
