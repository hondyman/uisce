package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// mockHasuraClient implements a simple mock for testing
type mockHasuraClient struct {
	queryResponse  map[string]interface{}
	mutateResponse map[string]interface{}
	queryErr       error
	mutateErr      error
}

func (m *mockHasuraClient) Query(query string, variables map[string]interface{}) (map[string]interface{}, error) {
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResponse, nil
}

func (m *mockHasuraClient) Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error) {
	if m.mutateErr != nil {
		return nil, m.mutateErr
	}
	return m.mutateResponse, nil
}

func TestSendNotificationHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockClient := &mockHasuraClient{
		mutateResponse: map[string]interface{}{
			"insert_notifications_one": map[string]interface{}{
				"id": "test-notification-id-123",
			},
		},
	}

	handler := sendNotificationHandler(mockClient, logger)

	reqBody := map[string]string{
		"tenant_id": "00000000-0000-0000-0000-000000000001",
		"user_id":   "00000000-0000-0000-0000-000000000002",
		"type":      "test_notification",
		"subject":   "Test Subject",
		"message":   "Test message content",
	}
	body, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/send", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["notification_id"] != "test-notification-id-123" {
		t.Errorf("expected notification_id test-notification-id-123, got %s", resp["notification_id"])
	}
}

func TestGetNotificationStatusHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockClient := &mockHasuraClient{
		queryResponse: map[string]interface{}{
			"notifications_by_pk": map[string]interface{}{
				"id":              "test-id",
				"tenant_id":       "tenant-123",
				"type":            "test_type",
				"subject":         "Test",
				"message":         "Message",
				"delivery_status": "sent",
			},
		},
	}

	handler := getNotificationStatusHandler(mockClient, logger)

	r := chi.NewRouter()
	r.Get("/{notificationID}", handler)

	req := httptest.NewRequest(http.MethodGet, "/test-id", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["id"] != "test-id" {
		t.Errorf("expected id test-id, got %v", resp["id"])
	}
}

func TestListNotificationsHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockClient := &mockHasuraClient{
		queryResponse: map[string]interface{}{
			"notifications": []interface{}{
				map[string]interface{}{
					"id":              "notif-1",
					"type":            "alert",
					"subject":         "Alert 1",
					"delivery_status": "sent",
				},
				map[string]interface{}{
					"id":              "notif-2",
					"type":            "info",
					"subject":         "Info 2",
					"delivery_status": "pending",
				},
			},
		},
	}

	handler := listNotificationsHandler(mockClient, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	count := int(resp["count"].(float64))
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestMarkAsReadHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockClient := &mockHasuraClient{
		mutateResponse: map[string]interface{}{
			"update_notifications_by_pk": map[string]interface{}{
				"id": "test-id",
			},
		},
	}

	handler := markAsReadHandler(mockClient, logger)

	r := chi.NewRouter()
	r.Put("/{notificationID}/read", handler)

	req := httptest.NewRequest(http.MethodPut, "/test-id/read", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func TestGetDeliveryStatsHandler(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	mockClient := &mockHasuraClient{
		queryResponse: map[string]interface{}{
			"notifications_aggregate": map[string]interface{}{
				"aggregate": map[string]interface{}{
					"count": float64(100),
				},
			},
		},
	}

	handler := getDeliveryStatsHandler(mockClient, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/stats/delivery", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	total := int64(resp["total"].(float64))
	if total != 100 {
		t.Errorf("expected total 100, got %d", total)
	}
}
