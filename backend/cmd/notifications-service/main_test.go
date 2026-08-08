package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func TestSendNotificationHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	logger, _ := zap.NewDevelopment()
	handler := sendNotificationHandler(sqlxDB, logger)

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
}

func TestGetNotificationStatusHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	logger, _ := zap.NewDevelopment()
	handler := getNotificationStatusHandler(sqlxDB, logger)

	r := chi.NewRouter()
	r.Get("/{notificationID}", handler)

	req := httptest.NewRequest(http.MethodGet, "/test-id", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestListNotificationsHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	logger, _ := zap.NewDevelopment()
	handler := listNotificationsHandler(sqlxDB, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestMarkAsReadHandler(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	logger, _ := zap.NewDevelopment()
	handler := markAsReadHandler(sqlxDB, logger)

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
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	defer db.Close()
	sqlxDB := sqlx.NewDb(db, "postgres")

	logger, _ := zap.NewDevelopment()
	handler := getDeliveryStatsHandler(sqlxDB, logger)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications/stats/delivery", nil)
	req.Header.Set("X-Tenant-ID", "00000000-0000-0000-0000-000000000001")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}
