package redpanda

import (
	"context"
	"encoding/json"
	"testing"
)

type mockEventListener struct {
	createdCallCount int
	updatedCallCount int
	deletedCallCount int
	lastUserID       string
	lastEventID      string
}

func (m *mockEventListener) OnEventCreated(ctx context.Context, userID, eventID string) {
	m.createdCallCount++
	m.lastUserID = userID
	m.lastEventID = eventID
}

func (m *mockEventListener) OnEventUpdated(ctx context.Context, userID, eventID string) {
	m.updatedCallCount++
	m.lastUserID = userID
	m.lastEventID = eventID
}

func (m *mockEventListener) OnEventDeleted(ctx context.Context, userID, eventID string) {
	m.deletedCallCount++
	m.lastUserID = userID
	m.lastEventID = eventID
}

func TestHandleInternalEventChange(t *testing.T) {
	mockListener := &mockEventListener{}

	processor := &CDCProcessor{
		eventListener: mockListener,
	}

	testCases := []struct {
		name       string
		op         string
		after      string
		before     string
		expectFunc func(m *mockEventListener) bool
	}{
		{
			name:   "Create Event",
			op:     "c",
			after:  `{"user_id": "user-1", "id": "event-1"}`,
			before: `null`,
			expectFunc: func(m *mockEventListener) bool {
				return m.createdCallCount == 1 && m.lastUserID == "user-1" && m.lastEventID == "event-1"
			},
		},
		{
			name:   "Update Event",
			op:     "u",
			after:  `{"user_id": "user-2", "id": "event-2"}`,
			before: `{"user_id": "user-2", "id": "event-2"}`,
			expectFunc: func(m *mockEventListener) bool {
				return m.updatedCallCount == 1 && m.lastUserID == "user-2" && m.lastEventID == "event-2"
			},
		},
		{
			name:   "Delete Event",
			op:     "d",
			after:  `null`,
			before: `{"user_id": "user-3", "id": "event-3"}`,
			expectFunc: func(m *mockEventListener) bool {
				return m.deletedCallCount == 1 && m.lastUserID == "user-3" && m.lastEventID == "event-3"
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			event := CDCEvent{
				Op:     tc.op,
				After:  json.RawMessage(tc.after),
				Before: json.RawMessage(tc.before),
			}

			err := processor.HandleInternalEventChange(context.Background(), event)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if !tc.expectFunc(mockListener) {
				t.Errorf("mock listener state did not match expectations")
			}
		})
	}
}
