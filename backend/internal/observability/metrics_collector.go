package observability

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MetricsCollector collects and stores metrics with production-ready features
type MetricsCollector struct {
	db            *sqlx.DB
	buffer        chan Metric
	bufferSize    int
	flushInterval time.Duration
	wg            sync.WaitGroup
	stopCh        chan struct{}

	// Stats for monitoring
	metricsReceived uint64
	metricsDropped  uint64
	metricsFlushed  uint64
	flushErrors     uint64
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(db *sqlx.DB, bufferSize int, flushInterval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		db:            db,
		buffer:        make(chan Metric, bufferSize),
		bufferSize:    bufferSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the background flushing of metrics
func (c *MetricsCollector) Start() {
	c.wg.Add(1)
	go c.flushLoop()
	log.Printf("[observability] MetricsCollector started with buffer size %d, flush interval %v", c.bufferSize, c.flushInterval)
}

// Stop gracefully stops the metrics collector with timeout
func (c *MetricsCollector) Stop() {
	log.Printf("[observability] MetricsCollector stopping...")
	close(c.stopCh)

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[observability] MetricsCollector stopped gracefully. Stats: received=%d, dropped=%d, flushed=%d, errors=%d",
			atomic.LoadUint64(&c.metricsReceived),
			atomic.LoadUint64(&c.metricsDropped),
			atomic.LoadUint64(&c.metricsFlushed),
			atomic.LoadUint64(&c.flushErrors))
	case <-time.After(10 * time.Second):
		log.Printf("[observability] MetricsCollector stop timeout, some metrics may be lost")
	}
}

// Record adds a metric to the buffer for later persistence
func (c *MetricsCollector) Record(metric Metric) {
	atomic.AddUint64(&c.metricsReceived, 1)

	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	select {
	case c.buffer <- metric:
		// Successfully buffered
	default:
		// Buffer full, drop metric and increment counter
		atomic.AddUint64(&c.metricsDropped, 1)
	}
}

// RecordValue is a convenience method to record a simple metric
func (c *MetricsCollector) RecordValue(tenantID uuid.UUID, name string, value float64, labels map[string]string) {
	c.Record(Metric{
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
		TenantID:  tenantID,
	})
}

// GetStats returns collector statistics
func (c *MetricsCollector) GetStats() map[string]uint64 {
	return map[string]uint64{
		"received": atomic.LoadUint64(&c.metricsReceived),
		"dropped":  atomic.LoadUint64(&c.metricsDropped),
		"flushed":  atomic.LoadUint64(&c.metricsFlushed),
		"errors":   atomic.LoadUint64(&c.flushErrors),
	}
}

// flushLoop periodically flushes buffered metrics to the database
func (c *MetricsCollector) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()

	var batch []Metric

	for {
		select {
		case <-c.stopCh:
			// Drain remaining buffered metrics
			draining := true
			for draining {
				select {
				case metric := <-c.buffer:
					batch = append(batch, metric)
				default:
					draining = false
				}
			}
			// Flush remaining metrics before stopping
			if len(batch) > 0 {
				c.flushBatch(batch)
			}
			return

		case metric := <-c.buffer:
			batch = append(batch, metric)
			if len(batch) >= 100 { // Flush when batch is large enough
				c.flushBatch(batch)
				batch = nil
			}

		case <-ticker.C:
			if len(batch) > 0 {
				c.flushBatch(batch)
				batch = nil
			}
		}
	}
}

// flushBatch writes a batch of metrics to the database
func (c *MetricsCollector) flushBatch(metrics []Metric) {
	if len(metrics) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use batch insert for efficiency
	query := `
		INSERT INTO obs_metrics (tenant_id, name, value, labels, timestamp)
		VALUES ($1, $2, $3, $4, $5)
	`

	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		atomic.AddUint64(&c.flushErrors, 1)
		log.Printf("[observability] Failed to begin transaction: %v", err)
		return
	}

	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		tx.Rollback()
		atomic.AddUint64(&c.flushErrors, 1)
		log.Printf("[observability] Failed to prepare statement: %v", err)
		return
	}
	defer stmt.Close()

	successCount := 0
	for _, m := range metrics {
		labelsJSON, err := json.Marshal(m.Labels)
		if err != nil {
			labelsJSON = []byte("{}")
		}

		_, err = stmt.ExecContext(ctx, m.TenantID, m.Name, m.Value, string(labelsJSON), m.Timestamp)
		if err != nil {
			// Log but continue with other metrics
			log.Printf("[observability] Failed to insert metric %s: %v", m.Name, err)
			continue
		}
		successCount++
	}

	if err := tx.Commit(); err != nil {
		atomic.AddUint64(&c.flushErrors, 1)
		log.Printf("[observability] Failed to commit transaction: %v", err)
		return
	}

	atomic.AddUint64(&c.metricsFlushed, uint64(successCount))
}

// Query retrieves metrics matching the given query
func (c *MetricsCollector) Query(ctx context.Context, q MetricQuery, tenantID uuid.UUID) ([]TimeSeries, error) {
	query := `
		SELECT name, labels, timestamp, value
		FROM obs_metrics
		WHERE tenant_id = $1
		  AND name = $2
		  AND timestamp >= $3
		  AND timestamp <= $4
		ORDER BY timestamp ASC
		LIMIT 10000
	`

	rows, err := c.db.QueryxContext(ctx, query, tenantID, q.Name, q.StartTime, q.EndTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group by labels
	seriesMap := make(map[string]*TimeSeries)

	for rows.Next() {
		var name, labelsJSON string
		var timestamp time.Time
		var value float64

		if err := rows.Scan(&name, &labelsJSON, &timestamp, &value); err != nil {
			continue
		}

		key := name + labelsJSON
		if _, ok := seriesMap[key]; !ok {
			labels := make(map[string]string)
			json.Unmarshal([]byte(labelsJSON), &labels)
			seriesMap[key] = &TimeSeries{
				Name:   name,
				Labels: labels,
			}
		}

		seriesMap[key].DataPoints = append(seriesMap[key].DataPoints, DataPoint{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	// Convert map to slice
	result := make([]TimeSeries, 0, len(seriesMap))
	for _, ts := range seriesMap {
		result = append(result, *ts)
	}

	return result, nil
}

// GetLatest returns the most recent value for a metric
func (c *MetricsCollector) GetLatest(ctx context.Context, tenantID uuid.UUID, name string) (*Metric, error) {
	query := `
		SELECT tenant_id, name, value, labels, timestamp
		FROM obs_metrics
		WHERE tenant_id = $1 AND name = $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var m Metric
	var labelsJSON string

	err := c.db.QueryRowxContext(ctx, query, tenantID, name).Scan(
		&m.TenantID, &m.Name, &m.Value, &labelsJSON, &m.Timestamp,
	)
	if err != nil {
		return nil, err
	}

	m.Labels = make(map[string]string)
	json.Unmarshal([]byte(labelsJSON), &m.Labels)
	return &m, nil
}

// GetAggregated returns aggregated metric values
func (c *MetricsCollector) GetAggregated(ctx context.Context, tenantID uuid.UUID, name string, aggregate string, duration time.Duration) (float64, error) {
	var aggFunc string
	switch aggregate {
	case "avg":
		aggFunc = "AVG(value)"
	case "sum":
		aggFunc = "SUM(value)"
	case "min":
		aggFunc = "MIN(value)"
	case "max":
		aggFunc = "MAX(value)"
	case "count":
		aggFunc = "COUNT(*)"
	default:
		aggFunc = "AVG(value)"
	}

	// Use parameterized interval to avoid SQL injection
	query := `
		SELECT COALESCE(` + aggFunc + `, 0)
		FROM obs_metrics
		WHERE tenant_id = $1
		  AND name = $2
		  AND timestamp > NOW() - $3::interval
	`

	var result float64
	err := c.db.QueryRowxContext(ctx, query, tenantID, name, duration.String()).Scan(&result)
	if err != nil {
		return 0, err
	}

	return result, nil
}
