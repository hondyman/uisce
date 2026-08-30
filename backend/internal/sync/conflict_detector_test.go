package sync

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// mockSyncClient is an unused placeholder client (CalendarSyncRepo ignores its constructor argument)
type mockSyncClient struct {
	QueryResult  map[string]interface{}
	MutateResult map[string]interface{}
	QueryErr     error
	MutateErr    error
}

func (m *mockSyncClient) Query(q string, v map[string]interface{}) (map[string]interface{}, error) {
	return m.QueryResult, m.QueryErr
}

func (m *mockSyncClient) Mutate(q string, v map[string]interface{}) (map[string]interface{}, error) {
	return m.MutateResult, m.MutateErr
}

func TestResolveConflict(t *testing.T) {
	// Setup
	mockClient := &mockSyncClient{
		QueryResult: map[string]interface{}{
			"sync_conflicts_by_pk": map[string]interface{}{
				"id":                "conflict-1",
				"resolution_status": "pending",
				"conflict_type":     "title_mismatch",
				"severity":          "warning",
				"description":       "Title mismatch",
			},
		},
		MutateResult: map[string]interface{}{
			"update_sync_conflicts_by_pk": map[string]interface{}{
				"id": "conflict-1",
			},
		},
	}

	repo := repository.NewCalendarSyncRepo(mockClient)
	detector := NewConflictDetector(ConflictDetectorConfig{
		SyncRepo: repo,
		Logger:   logrus.NewEntry(logrus.New()),
	})

	// Test KeepExternal
	err := detector.ResolveConflict(context.Background(), "conflict-1", StrategyKeepExternal)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
