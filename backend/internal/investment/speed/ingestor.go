package speed

import (
	"context"
	"fmt"
	"time"
)

// PerformanceRecord represents a single row for the daily_performance table
type PerformanceRecord struct {
	PortfolioID  uint64
	Date         time.Time
	ReturnFactor float64
	MarketValue  float64
	CashFlow     float64
}

// StarRocksClient interface allows mocking the actual DB interaction
type StarRocksClient interface {
	InsertBatch(ctx context.Context, table string, rows []interface{}) error
}

// ClickHouseClient is deprecated - use StarRocksClient
type ClickHouseClient = StarRocksClient

// IngestionWorker handles buffered insertion into StarRocks
type IngestionWorker struct {
	client        StarRocksClient
	inputChan     <-chan PerformanceRecord
	batchSize     int
	flushInterval time.Duration
}

func NewIngestionWorker(client StarRocksClient, input <-chan PerformanceRecord, batchSize int, flushInterval time.Duration) *IngestionWorker {
	return &IngestionWorker{
		client:        client,
		inputChan:     input,
		batchSize:     batchSize,
		flushInterval: flushInterval,
	}
}

// Start runs the worker loop
func (w *IngestionWorker) Start(ctx context.Context) {
	buffer := make([]interface{}, 0, w.batchSize)
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	flush := func() {
		if len(buffer) > 0 {
			if err := w.client.InsertBatch(ctx, "daily_performance", buffer); err != nil {
				// In a real system, we would Nack messages here or retry
				fmt.Printf("Error flushing batch to StarRocks: %v\n", err)
			} else {
				fmt.Printf("Flushed %d records to StarRocks\n", len(buffer))
			}
			buffer = buffer[:0] // Reset buffer
		}
	}

	for {
		select {
		case record, ok := <-w.inputChan:
			if !ok {
				flush() // Flush remaining on close
				return
			}
			buffer = append(buffer, record)
			if len(buffer) >= w.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			flush()
			return
		}
	}
}
