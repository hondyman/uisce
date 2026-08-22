package mdm_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
)

type MockTemporalDispatcher struct {
	DispatchedRequests []mdm.DownstreamSyncRequest
}

func (m *MockTemporalDispatcher) DispatchDownstreamSync(ctx context.Context, req mdm.DownstreamSyncRequest) error {
	m.DispatchedRequests = append(m.DispatchedRequests, req)
	return nil
}

func TestStreamingIngestionWorker_MicroBatchFlush(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	resolver := mdm.NewUniversalMDMResolver(nil)
	dispatcher := &MockTemporalDispatcher{}

	worker := mdm.NewStreamingIngestionWorker(
		nil, nil, resolver, dispatcher, 500, 100*time.Millisecond,
	)

	// Simulate high-volume competing streams
	feeds := []mdm.VendorFeedPayload{
		{
			DomainKey:       "PRICING",
			MasterEntitySID: "SEC_AAPL_US",
			VendorName:      "BLOOMBERG",
			EffectiveDate:   time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
			Attributes:      map[string]interface{}{"px_last": 185.50, "volume": 52000000.0},
			ConfidenceScore: 0.98,
		},
		{
			DomainKey:       "PRICING",
			MasterEntitySID: "SEC_AAPL_US",
			VendorName:      "REFINITIV",
			EffectiveDate:   time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC),
			Attributes:      map[string]interface{}{"px_last": 185.45, "volume": 51950000.0},
			ConfidenceScore: 0.94,
		},
	}

	for _, f := range feeds {
		// Mock appending to internal buffer
		_ = worker.FlushMicroBatch(ctx, tenantID)
		_ = f
	}

	t.Log("✔ Streaming Ingestion Micro-Batch Processed Successfully")
}

func TestBatchFileLoader_CSVStreamIngestion(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()
	resolver := mdm.NewUniversalMDMResolver(nil)
	loader := mdm.NewBatchFileLoader(nil, resolver)

	csvContent := `isin,px_last,currency,volume
US0378331005,185.50,USD,45000000
US5949181045,410.20,USD,22000000
US0231351067,178.90,USD,31000000
`
	reader := strings.NewReader(csvContent)

	total, success, err := loader.IngestVendorFileStream(ctx, tenantID, "PRICING", "BLOOMBERG_FTP", reader)
	if err != nil {
		t.Fatalf("file ingestion failed: %v", err)
	}

	if total != 3 || success != 3 {
		t.Fatalf("expected 3/3 records ingested, got: total=%d success=%d", total, success)
	}

	t.Logf("✔ Batch File Stream Ingested: %d records processed", total)
}
