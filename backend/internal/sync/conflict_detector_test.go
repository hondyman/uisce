package sync

import (
	"context"
	"testing"

	"github.com/hondyman/semlayer/backend/internal/repository"
	"github.com/sirupsen/logrus"
)

// MockHasuraClient implements repository.HasuraClient
type MockHasuraClient struct {
	QueryResult  map[string]interface{}
	MutateResult map[string]interface{}
	QueryErr     error
	MutateErr    error
}

func (m *MockHasuraClient) Query(q string, v map[string]interface{}) (map[string]interface{}, error) {
	return m.QueryResult, m.QueryErr
}

func (m *MockHasuraClient) Mutate(q string, v map[string]interface{}) (map[string]interface{}, error) {
	return m.MutateResult, m.MutateErr
}

func TestResolveConflict(t *testing.T) {
	// Setup
	mockClient := &MockHasuraClient{
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

	repo := repository.NewGoogleSyncRepo(mockClient)
	detector := NewConflictDetector(ConflictDetectorConfig{
		SyncRepo: repo,
		Logger:   logrus.NewEntry(logrus.New()),
	})

	// Test KeepGoogle
	err := detector.ResolveConflict(context.Background(), "conflict-1", StrategyKeepGoogle)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}
