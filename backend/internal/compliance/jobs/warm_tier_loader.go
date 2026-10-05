package jobs

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
)

// WarmTierLoaderConfig holds configuration for syncing Hot PG evaluations to Warm StarRocks
type WarmTierLoaderConfig struct {
	PostgresDB         *sql.DB
	StarRocksHTTPHost  string
	StarRocksHTTPPort  int
	StarRocksUser      string
	StarRocksPassword  string
	StarRocksDatabase  string
	StarRocksTable     string
	BatchSize          int
	HTTPClientTimeout  time.Duration
}

// SyncResult captures the telemetry of a completed Warm Tier batch sync
type SyncResult struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	RowsLoaded  int64     `json:"rows_loaded"`
	StartLSN    int64     `json:"start_lsn"`
	HighestLSN  int64     `json:"highest_lsn"`
	TxnID       int64     `json:"txn_id"`
	DurationMs  int64     `json:"duration_ms"`
}

// EvaluationRecord represents the row schema for Hot PG to StarRocks Warm Tier sync
type EvaluationRecord struct {
	LineageID       string          `json:"lineage_id"`
	EvaluatedAt     string          `json:"evaluated_at"`
	TenantID        string          `json:"tenant_id"`
	OrderID         *string         `json:"order_id,omitempty"`
	AccountID       *string         `json:"account_id,omitempty"`
	SecurityID      *string         `json:"security_id,omitempty"`
	RuleID          string          `json:"rule_id"`
	RuleVersion     int             `json:"rule_version"`
	ActionTaken     string          `json:"action_taken"`
	Passed          bool            `json:"passed"`
	LatencyMicros   int64           `json:"latency_micros"`
	EvaluationHash  string          `json:"evaluation_hash"`
	IngestLSN       int64           `json:"ingest_lsn"`
	InputParams     json.RawMessage `json:"input_params"`
	MetricSnapshots json.RawMessage `json:"metric_snapshots"`
	CreatedAt       string          `json:"created_at"`
}

// StarRocksStreamLoadResponse represents the JSON response returned by StarRocks Stream Load API
type StarRocksStreamLoadResponse struct {
	TxnID            int64  `json:"TxnId"`
	Label            string `json:"Label"`
	Status           string `json:"Status"`
	Message          string `json:"Message"`
	NumberTotalRows  int64  `json:"NumberTotalRows"`
	NumberLoadedRows int64  `json:"NumberLoadedRows"`
	LoadBytes        int64  `json:"LoadBytes"`
	LoadTimeMs       int64  `json:"LoadTimeMs"`
}

// WarmTierLoader manages continuous, LSN-certified batch streaming from Hot PostgreSQL to StarRocks Warm Tier
type WarmTierLoader struct {
	cfg        WarmTierLoaderConfig
	httpClient *http.Client
}

// NewWarmTierLoader instantiates a WarmTierLoader with custom HTTP redirect handling for StarRocks FE/BE routing
func NewWarmTierLoader(cfg WarmTierLoaderConfig) *WarmTierLoader {
	if cfg.StarRocksHTTPHost == "" {
		cfg.StarRocksHTTPHost = os.Getenv("STARROCKS_HTTP_HOST")
		if cfg.StarRocksHTTPHost == "" {
			cfg.StarRocksHTTPHost = "100.84.50.65"
		}
	}
	if cfg.StarRocksHTTPPort <= 0 {
		cfg.StarRocksHTTPPort = 8030
	}
	if cfg.StarRocksUser == "" {
		cfg.StarRocksUser = os.Getenv("STARROCKS_USER")
		if cfg.StarRocksUser == "" {
			cfg.StarRocksUser = "root"
		}
	}
	if cfg.StarRocksDatabase == "" {
		cfg.StarRocksDatabase = "oms"
	}
	if cfg.StarRocksTable == "" {
		cfg.StarRocksTable = "compliance_evaluations"
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 1000
	}
	if cfg.HTTPClientTimeout <= 0 {
		cfg.HTTPClientTimeout = 30 * time.Second
	}

	targetHost := cfg.StarRocksHTTPHost

	client := &http.Client{
		Timeout: cfg.HTTPClientTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			// When FE redirects to BE container IP/hostname, remap host back to reachable IP
			u, err := url.Parse(req.URL.String())
			if err == nil {
				host, port, splitErr := net.SplitHostPort(u.Host)
				if splitErr == nil && host != targetHost && host != "localhost" && host != "127.0.0.1" {
					u.Host = net.JoinHostPort(targetHost, port)
					req.URL = u
				}
			}
			req.SetBasicAuth(cfg.StarRocksUser, cfg.StarRocksPassword)
			return nil
		},
	}

	return &WarmTierLoader{
		cfg:        cfg,
		httpClient: client,
	}
}

// GetWatermark retrieves the last certified LSN watermark for a tenant's WARM tier
func (l *WarmTierLoader) GetWatermark(ctx context.Context, tenantID uuid.UUID) (int64, error) {
	var lsnInt int64
	query := `
		SELECT (current_lsn - '0/0'::pg_lsn)::bigint
		FROM compliance.compliance_watermark_checkpoint
		WHERE tenant_id = $1 AND tier = 'WARM'
	`
	err := l.cfg.PostgresDB.QueryRowContext(ctx, query, tenantID).Scan(&lsnInt)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("query warm watermark: %w", err)
	}
	return lsnInt, nil
}

// UpdateWatermark updates or creates the certified LSN watermark for a tenant's WARM tier
func (l *WarmTierLoader) UpdateWatermark(ctx context.Context, tenantID uuid.UUID, lsn int64) error {
	query := `
		INSERT INTO compliance.compliance_watermark_checkpoint (
			id, tenant_id, tier, current_lsn, certified_at, updated_at
		) VALUES (
			gen_random_uuid(), $1, 'WARM', '0/0'::pg_lsn + $2, NOW(), NOW()
		)
		ON CONFLICT (tenant_id, tier) DO UPDATE SET
			current_lsn = EXCLUDED.current_lsn,
			certified_at = EXCLUDED.certified_at,
			updated_at = NOW()
	`
	_, err := l.cfg.PostgresDB.ExecContext(ctx, query, tenantID, lsn)
	if err != nil {
		return fmt.Errorf("update warm watermark: %w", err)
	}
	return nil
}

// SyncTenantBatch executes a single batch sync for a given tenant from Hot PG to StarRocks
func (l *WarmTierLoader) SyncTenantBatch(ctx context.Context, tenantID uuid.UUID) (*SyncResult, error) {
	start := time.Now()

	startLSN, err := l.GetWatermark(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	var maxDBLSN int64
	err = l.cfg.PostgresDB.QueryRowContext(ctx, "SELECT (pg_current_wal_lsn() - '0/0'::pg_lsn)::bigint").Scan(&maxDBLSN)
	if err != nil {
		return nil, fmt.Errorf("get current pg_current_wal_lsn: %w", err)
	}

	query := `
		SELECT 
			lineage_id, evaluated_at, tenant_id, order_id, rule_id, rule_version,
			action_taken, passed, latency_micros, evaluation_hash, ingest_lsn,
			COALESCE(input_params, '{}'::jsonb), COALESCE(metric_snapshots, '{}'::jsonb), created_at
		FROM compliance.compliance_evaluation_event
		WHERE tenant_id = $1
		  AND ingest_lsn > $2
		  AND ingest_lsn <= $3
		ORDER BY ingest_lsn ASC
		LIMIT $4
	`

	rows, err := l.cfg.PostgresDB.QueryContext(ctx, query, tenantID, startLSN, maxDBLSN, l.cfg.BatchSize)
	if err != nil {
		return nil, fmt.Errorf("query evaluation events: %w", err)
	}
	defer rows.Close()

	var records []EvaluationRecord
	var highestLSN int64 = startLSN

	for rows.Next() {
		var r EvaluationRecord
		var evaluatedAt, createdAt time.Time
		var lineageID, tenantIDStr, ruleID string
		var orderID sql.NullString
		var inputParams, metricSnapshots []byte

		if err := rows.Scan(
			&lineageID, &evaluatedAt, &tenantIDStr, &orderID, &ruleID, &r.RuleVersion,
			&r.ActionTaken, &r.Passed, &r.LatencyMicros, &r.EvaluationHash, &r.IngestLSN,
			&inputParams, &metricSnapshots, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan evaluation row: %w", err)
		}

		r.LineageID = lineageID
		r.TenantID = tenantIDStr
		r.RuleID = ruleID
		if orderID.Valid {
			r.OrderID = &orderID.String
		}
		r.EvaluatedAt = evaluatedAt.UTC().Format("2006-01-02 15:04:05")
		r.CreatedAt = createdAt.UTC().Format("2006-01-02 15:04:05")
		r.InputParams = json.RawMessage(inputParams)
		r.MetricSnapshots = json.RawMessage(metricSnapshots)

		if r.IngestLSN > highestLSN {
			highestLSN = r.IngestLSN
		}

		records = append(records, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate evaluation rows: %w", err)
	}

	if len(records) == 0 {
		return &SyncResult{
			TenantID:   tenantID,
			RowsLoaded: 0,
			StartLSN:   startLSN,
			HighestLSN: startLSN,
			DurationMs: time.Since(start).Milliseconds(),
		}, nil
	}

	// Stream Load to StarRocks
	loadResp, err := l.streamLoadToStarRocks(ctx, records, tenantID, startLSN, highestLSN)
	if err != nil {
		return nil, fmt.Errorf("stream load to starrocks: %w", err)
	}

	if loadResp.Status != "Success" && loadResp.Status != "OK" {
		return nil, fmt.Errorf("starrocks stream load failed: status=%s, message=%s", loadResp.Status, loadResp.Message)
	}

	// Watermark checkpoint upon StarRocks success
	if err := l.UpdateWatermark(ctx, tenantID, highestLSN); err != nil {
		return nil, fmt.Errorf("checkpoint watermark after stream load: %w", err)
	}

	return &SyncResult{
		TenantID:   tenantID,
		RowsLoaded: loadResp.NumberLoadedRows,
		StartLSN:   startLSN,
		HighestLSN: highestLSN,
		TxnID:      loadResp.TxnID,
		DurationMs: time.Since(start).Milliseconds(),
	}, nil
}

func (l *WarmTierLoader) streamLoadToStarRocks(ctx context.Context, records []EvaluationRecord, tenantID uuid.UUID, startLSN, endLSN int64) (*StarRocksStreamLoadResponse, error) {
	jsonData, err := json.Marshal(records)
	if err != nil {
		return nil, fmt.Errorf("marshal stream load records: %w", err)
	}

	label := fmt.Sprintf("warm_load_%s_%d_%d_%d", tenantID.String()[:8], startLSN, endLSN, time.Now().UnixNano())
	urlStr := fmt.Sprintf("http://%s:%d/api/%s/%s/_stream_load",
		l.cfg.StarRocksHTTPHost, l.cfg.StarRocksHTTPPort, l.cfg.StarRocksDatabase, l.cfg.StarRocksTable)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, urlStr, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create stream load request: %w", err)
	}

	req.Header.Set("Expect", "100-continue")
	req.Header.Set("format", "json")
	req.Header.Set("strip_outer_array", "true")
	req.Header.Set("label", label)
	req.SetBasicAuth(l.cfg.StarRocksUser, l.cfg.StarRocksPassword)

	resp, err := l.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute stream load http request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read stream load response body: %w", err)
	}

	var slResp StarRocksStreamLoadResponse
	if err := json.Unmarshal(bodyBytes, &slResp); err != nil {
		return nil, fmt.Errorf("unmarshal starrocks response (status=%d, body=%s): %w", resp.StatusCode, string(bodyBytes), err)
	}

	return &slResp, nil
}
