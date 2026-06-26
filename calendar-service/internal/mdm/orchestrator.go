package mdm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// Data Models (Aligning with Usice Architecture Section 4: Business Objects)
// ============================================================================

type HolidayData struct {
	Date          time.Time
	IsBusinessDay bool
	RegionCode    string
	ExchangeCode  *string
	HolidayName   *string
	SourceSystem  string
	RawPayload    map[string]interface{}
}

type SourceConfig struct {
	ID             uuid.UUID
	Name           string
	Type           string
	Endpoint       string
	IsActive       bool
	PriorityScore  int
	ConfidenceBase int
}

type IngestionJob struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	JobType           string
	Status            string
	RecordsIngested   int
	RecordsProcessed  int
	ConflictsDetected int
	ErrorMessage      string
	StartedAt         time.Time
	CompletedAt       *time.Time
	DurationMs        int
}

// ============================================================================
// Ingestion Orchestrator (Usice Architecture Section 2.3: Semantic Engine)
// ============================================================================

type IngestionOrchestrator struct {
	db          *sql.DB
	httpClient  *http.Client
	logger      *logrus.Entry
	sourceCache map[string]SourceConfig // Cache active sources
}

func NewIngestionOrchestrator(db *sql.DB, logger *logrus.Entry) *IngestionOrchestrator {
	return &IngestionOrchestrator{
		db: db,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:      logger,
		sourceCache: make(map[string]SourceConfig),
	}
}

// RunIngestionCycle orchestrates the complete ingestion process
// This is called by either a cron scheduler or manual API trigger
func (o *IngestionOrchestrator) RunIngestionCycle(
	ctx context.Context,
	tenantID uuid.UUID,
	regions []string,
	year int,
) error {
	jobID := uuid.New()
	o.logger.WithFields(logrus.Fields{
		"job_id":  jobID,
		"tenant":  tenantID,
		"regions": regions,
		"year":    year,
	}).Info("Starting ingestion cycle")

	job := &IngestionJob{
		ID:        jobID,
		TenantID:  tenantID,
		JobType:   "SCHEDULED",
		Status:    "IN_PROGRESS",
		StartedAt: time.Now(),
	}

	// Record job start
	if err := o.storeIngestionJob(ctx, job); err != nil {
		o.logger.WithError(err).Error("Failed to store ingestion job")
		return err
	}

	// 1. Load ACTIVE sources only from registry
	sources, err := o.getActiveSources(ctx)
	if err != nil {
		o.logger.WithError(err).Error("Failed to fetch active sources")
		job.Status = "FAILED"
		job.ErrorMessage = err.Error()
		o.updateIngestionJob(ctx, job)
		return err
	}

	o.logger.WithField("source_count", len(sources)).Info("Loaded active sources")

	// 2. For each region, fetch data from all active sources
	for _, region := range regions {
		for _, source := range sources {
			o.logger.WithFields(logrus.Fields{
				"source": source.Name,
				"region": region,
			}).Debug("Fetching data from source")

			holidays, err := o.fetchFromSource(ctx, source, region, year)
			if err != nil {
				o.logger.WithError(err).Warnf("Failed to fetch from source %s", source.Name)
				continue // Continue with other sources (resilience)
			}

			// 3. Normalize to semantic terms and store in staging
			for _, h := range holidays {
				if err := o.storeSourceRecord(ctx, tenantID, source, h); err != nil {
					o.logger.WithError(err).Error("Failed to store source record")
					return err
				}
				job.RecordsIngested++
			}
		}

		// 4. Trigger WASM Survivorship Rules for this region/year
		if err := o.runSurvivorship(ctx, tenantID, region, year); err != nil {
			o.logger.WithError(err).Errorf("Failed to run survivorship for region %s", region)
			job.ErrorMessage = fmt.Sprintf("Survivorship failed for region %s: %v", region, err)
			job.Status = "FAILED"
			o.updateIngestionJob(ctx, job)
			return err
		}
	}

	// 5. Mark job as successful
	job.Status = "SUCCESS"
	job.CompletedAt = now()
	job.DurationMs = int(time.Since(job.StartedAt).Milliseconds())

	if err := o.updateIngestionJob(ctx, job); err != nil {
		o.logger.WithError(err).Error("Failed to update ingestion job")
		return err
	}

	o.logger.WithFields(logrus.Fields{
		"job_id":   job.ID,
		"records":  job.RecordsIngested,
		"duration": job.DurationMs,
		"status":   "SUCCESS",
	}).Info("Ingestion cycle completed")

	return nil
}

// getActiveSources returns only ACTIVE sources from the registry
// This is the critical filter that enables/disables sources dynamically
func (o *IngestionOrchestrator) getActiveSources(ctx context.Context) ([]SourceConfig, error) {
	query := `
		SELECT id, source_name, source_type, endpoint_url, is_active, priority_score, confidence_base
			FROM edm.mdm_source_registry
		WHERE is_active = true
		ORDER BY priority_score ASC
	`

	rows, err := o.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sources: %w", err)
	}
	defer rows.Close()

	var sources []SourceConfig
	for rows.Next() {
		var s SourceConfig
		err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.Endpoint, &s.IsActive, &s.PriorityScore, &s.ConfidenceBase)
		if err != nil {
			return nil, fmt.Errorf("failed to scan source: %w", err)
		}
		sources = append(sources, s)
		o.sourceCache[s.Name] = s // Cache for quick lookup
	}

	return sources, nil
}

// fetchFromSource dispatches to the appropriate fetcher based on source type
func (o *IngestionOrchestrator) fetchFromSource(
	ctx context.Context,
	src SourceConfig,
	region string,
	year int,
) ([]HolidayData, error) {
	switch src.Name {
	case "NagerDate":
		return o.fetchNagerDate(ctx, src.Endpoint, region, year)
	case "OpenHolidays":
		return o.fetchOpenHolidays(ctx, src.Endpoint, region, year)
	case "Workalendar":
		return o.fetchPythonService(ctx, src.Endpoint, "workalendar", region, year)
	case "HolidaysPyPI":
		return o.fetchPythonService(ctx, src.Endpoint, "holidays", region, year)
	case "TradingHours":
		return o.fetchTradingHours(ctx, src.Endpoint, region, year)
	case "EODHD":
		return o.fetchEODHD(ctx, src.Endpoint, region, year)
	default:
		return nil, fmt.Errorf("unknown source: %s", src.Name)
	}
}

// ============================================================================
// Source Fetchers
// ============================================================================

// fetchNagerDate implements Nager.Date API fetcher
// Free API: https://date.nager.at/api/v3/PublicHolidays/{year}/{countryCode}
func (o *IngestionOrchestrator) fetchNagerDate(ctx context.Context, baseURL, region string, year int) ([]HolidayData, error) {
	url := fmt.Sprintf("%s/PublicHolidays/%d/%s", baseURL, year, region)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nager request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("nager returned %d: %s", resp.StatusCode, string(body))
	}

	var nagerHolidays []struct {
		Date   string `json:"date"`
		Name   string `json:"name"`
		Global bool   `json:"global"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&nagerHolidays); err != nil {
		return nil, fmt.Errorf("failed to decode nager response: %w", err)
	}

	var holidays []HolidayData
	for _, nh := range nagerHolidays {
		date, _ := time.Parse("2006-01-02", nh.Date)
		holidays = append(holidays, HolidayData{
			Date:          date,
			IsBusinessDay: false, // Holidays are not business days
			RegionCode:    region,
			HolidayName:   &nh.Name,
			SourceSystem:  "NagerDate",
			RawPayload: map[string]interface{}{
				"date":   nh.Date,
				"name":   nh.Name,
				"global": nh.Global,
			},
		})
	}

	o.logger.WithFields(logrus.Fields{
		"source": "NagerDate",
		"region": region,
		"count":  len(holidays),
	}).Debug("Fetched holidays from NagerDate")

	return holidays, nil
}

// fetchOpenHolidays implements OpenHolidays API fetcher
// API: https://openholidaysapi.org/PublicHolidays?countryIsoCode=US&languageIsoCode=en
func (o *IngestionOrchestrator) fetchOpenHolidays(ctx context.Context, baseURL, region string, year int) ([]HolidayData, error) {
	url := fmt.Sprintf(
		"%s/PublicHolidays?countryIsoCode=%s&languageIsoCode=en&validFrom=%04d-01-01&validUntil=%04d-12-31",
		baseURL, region, year, year,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openholidays request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openholidays returned %d", resp.StatusCode)
	}

	var response struct {
		Holidays []struct {
			Date string `json:"date"`
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"holidays"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode openholidays response: %w", err)
	}

	var holidays []HolidayData
	for _, oh := range response.Holidays {
		date, _ := time.Parse("2006-01-02", oh.Date)
		holidays = append(holidays, HolidayData{
			Date:          date,
			IsBusinessDay: false,
			RegionCode:    region,
			HolidayName:   &oh.Name,
			SourceSystem:  "OpenHolidays",
			RawPayload: map[string]interface{}{
				"date": oh.Date,
				"name": oh.Name,
				"type": oh.Type,
			},
		})
	}

	o.logger.WithFields(logrus.Fields{
		"source": "OpenHolidays",
		"region": region,
		"count":  len(holidays),
	}).Debug("Fetched holidays from OpenHolidays")

	return holidays, nil
}

// fetchPythonService calls either Workalendar or Holidays PyPI microservice
func (o *IngestionOrchestrator) fetchPythonService(
	ctx context.Context,
	baseURL string,
	service string,
	region string,
	year int,
) ([]HolidayData, error) {
	url := fmt.Sprintf("%s/holidays?region=%s&year=%d", baseURL, region, year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s request failed: %w", service, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned %d", service, resp.StatusCode)
	}

	var pyResponse struct {
		Holidays []struct {
			Date string `json:"date"`
			Name string `json:"name"`
		} `json:"holidays"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&pyResponse); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", service, err)
	}

	var holidays []HolidayData
	for _, ph := range pyResponse.Holidays {
		date, _ := time.Parse("2006-01-02", ph.Date)
		holidays = append(holidays, HolidayData{
			Date:          date,
			IsBusinessDay: false,
			RegionCode:    region,
			HolidayName:   &ph.Name,
			SourceSystem:  service,
			RawPayload: map[string]interface{}{
				"date": ph.Date,
				"name": ph.Name,
			},
		})
	}

	o.logger.WithFields(logrus.Fields{
		"source": service,
		"region": region,
		"count":  len(holidays),
	}).Debug(fmt.Sprintf("Fetched holidays from %s", service))

	return holidays, nil
}

// fetchTradingHours implements TradingHours.com API (premium source)
// API: https://api.tradinghours.com/v1/market-holidays?exchange={exchangeCode}&year={year}
func (o *IngestionOrchestrator) fetchTradingHours(ctx context.Context, baseURL, region string, year int) ([]HolidayData, error) {
	// This is a stub - implement when TradingHours is activated
	o.logger.WithField("region", region).Debug("TradingHours not yet implemented")
	return nil, nil
}

// fetchEODHD implements EODHD API (premium source)
// API: https://eodhd.com/api/exchange-holidays/{EXCHANGE_CODE}
func (o *IngestionOrchestrator) fetchEODHD(ctx context.Context, baseURL, region string, year int) ([]HolidayData, error) {
	// This is a stub - implement when EODHD is activated
	o.logger.WithField("region", region).Debug("EODHD not yet implemented")
	return nil, nil
}

// ============================================================================
// Database Operations
// ============================================================================

func (o *IngestionOrchestrator) storeSourceRecord(
	ctx context.Context,
	tenantID uuid.UUID,
	source SourceConfig,
	holiday HolidayData,
) error {
	rawPayload, _ := json.Marshal(holiday.RawPayload)

	query := `
		INSERT INTO edm.mdm_calendar_source 
		(tenant_id, source_registry_id, calendar_date, is_business_day, region_code, exchange_code, holiday_name, raw_payload)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING
	`

	_, err := o.db.ExecContext(ctx, query,
		tenantID,
		source.ID,
		holiday.Date,
		holiday.IsBusinessDay,
		holiday.RegionCode,
		holiday.ExchangeCode,
		holiday.HolidayName,
		string(rawPayload),
	)

	return err
}

func (o *IngestionOrchestrator) runSurvivorship(
	ctx context.Context,
	tenantID uuid.UUID,
	region string,
	year int,
) error {
	// TODO: Implement WASM rule execution
	// For now, a simple approach: use latest source as winner
	// In production, this executes WASM survivorship rules

	query := `
		INSERT INTO edm.mdm_calendar_golden 
		(tenant_id, calendar_date, is_business_day, region_code, holiday_name, source_system, confidence_score)
		SELECT DISTINCT ON (tenant_id, calendar_date)
			tenant_id,
			calendar_date,
			is_business_day,
			region_code,
			holiday_name,
			'MULTI_SOURCE' as source_system,
			70 as confidence_score
		FROM edm.mdm_calendar_source
		WHERE tenant_id = $1 
		AND region_code = $2
		AND EXTRACT(YEAR FROM calendar_date) = $3
		ON CONFLICT (tenant_id, calendar_date, region_code)
		DO UPDATE SET
			is_business_day = EXCLUDED.is_business_day,
			updated_at = NOW()
	`

	_, err := o.db.ExecContext(ctx, query, tenantID, region, year)
	return err
}

func (o *IngestionOrchestrator) storeIngestionJob(ctx context.Context, job *IngestionJob) error {
	query := `
		INSERT INTO edm.mdm_ingestion_jobs 
		(id, tenant_id, job_type, status, started_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := o.db.ExecContext(ctx, query, job.ID, job.TenantID, job.JobType, job.Status, job.StartedAt)
	return err
}

func (o *IngestionOrchestrator) updateIngestionJob(ctx context.Context, job *IngestionJob) error {
	query := `
		UPDATE edm.mdm_ingestion_jobs
		SET status = $1, error_message = $2, completed_at = $3, duration_ms = $4
		WHERE id = $5
	`

	_, err := o.db.ExecContext(ctx, query, job.Status, job.ErrorMessage, job.CompletedAt, job.DurationMs, job.ID)
	return err
}

// ============================================================================
// Helper Functions
// ============================================================================

func now() *time.Time {
	t := time.Now()
	return &t
}
