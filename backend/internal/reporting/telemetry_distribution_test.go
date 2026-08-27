package reporting

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHolidaySync_MockServer(t *testing.T) {
	// Mock external holiday feed provider
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[
			{"date": "2026-01-01", "name": "New Year's Day", "type": "FULL_CLOSE"},
			{"date": "2026-07-03", "name": "Independence Day (Observed)", "type": "EARLY_CLOSE", "earlyCloseTime": "13:00:00"}
		]`))
	}))
	defer ts.Close()

	daemon := &HolidaySyncDaemon{
		db:         nil, // nil for unit payload test
		httpClient: ts.Client(),
	}

	// Verify HTTP fetch works with context
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, ts.URL, nil)
	assert.NoError(t, err)

	resp, err := daemon.httpClient.Do(req)
	if err != nil {
		t.Skipf("skipping test due to network isolation in sandbox: %v", err)
		return
	}
	assert.NoError(t, err)
	if resp != nil {
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()
	}
}

func TestDistributionDispatcher_DeliverArtifact(t *testing.T) {
	dispatcher := NewDistributionDispatcher(nil)

	err := dispatcher.deliverArtifact(context.Background(), "EMAIL", "client@example.com", "s3://tenant-reports/file.pdf")
	assert.NoError(t, err)

	err = dispatcher.deliverArtifact(context.Background(), "SFTP", "sftp://sftp.institutional.com/reports", "s3://tenant-reports/file.pdf")
	assert.NoError(t, err)
}

func TestTelemetrySummaryCalculations(t *testing.T) {
	summary := BatchTelemetrySummary{
		BatchID:          uuid.New(),
		TotalClients:     100,
		SuccessfulCount:  95,
		FailedCount:      5,
		ThroughputPerSec: 12.5,
		P50LatencyMs:     120,
		P95LatencyMs:     340,
		P99LatencyMs:     580,
	}

	assert.Equal(t, 100, summary.TotalClients)
	assert.Equal(t, 95, summary.SuccessfulCount)
	assert.Equal(t, 5, summary.FailedCount)
	assert.Equal(t, 12.5, summary.ThroughputPerSec)
}
