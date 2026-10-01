package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

const (
	tenantA = "00000000-0000-0000-0000-00000000000a"
	tenantB = "00000000-0000-0000-0000-00000000000b"
	userA   = "00000000-0000-0000-0000-000000000002"
)

func newMock(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return sqlx.NewDb(db, "postgres"), mock
}

// asTenant puts the claims the JWT middleware would set on the request.
func asTenant(req *http.Request, tenantID string) *http.Request {
	claims := &jwtmiddleware.JWTClaims{UserID: userA, TenantID: tenantID}
	return req.WithContext(context.WithValue(req.Context(), jwtmiddleware.ClaimsContextKey, claims))
}

func expectMet(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func sendBody(tenantID string) *bytes.Reader {
	body, _ := json.Marshal(map[string]string{
		"tenant_id": tenantID,
		"user_id":   userA,
		"type":      "test_notification",
		"subject":   "Test Subject",
		"message":   "Test message content",
	})
	return bytes.NewReader(body)
}

func TestSendNotificationHandler(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`INSERT INTO notification_outbox`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), tenantA, userA).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := asTenant(httptest.NewRequest(http.MethodPost, "/api/notifications/send", sendBody(tenantA)), tenantA)
	w := httptest.NewRecorder()
	sendNotificationHandler(db, zap.NewNop()).ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
	expectMet(t, mock)
}

func TestSendNotificationHandler_DefaultsToTokenTenant(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`INSERT INTO notification_outbox`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), tenantA, userA).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := asTenant(httptest.NewRequest(http.MethodPost, "/api/notifications/send", sendBody("")), tenantA)
	w := httptest.NewRecorder()
	sendNotificationHandler(db, zap.NewNop()).ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("expected status %d, got %d", http.StatusAccepted, w.Code)
	}
	expectMet(t, mock)
}

func TestSendNotificationHandler_RejectsOtherTenant(t *testing.T) {
	db, mock := newMock(t) // no expectations: nothing may be written

	req := asTenant(httptest.NewRequest(http.MethodPost, "/api/notifications/send", sendBody(tenantB)), tenantA)
	w := httptest.NewRecorder()
	sendNotificationHandler(db, zap.NewNop()).ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status %d, got %d", http.StatusForbidden, w.Code)
	}
	expectMet(t, mock)
}

func TestSendNotificationHandler_Unauthenticated(t *testing.T) {
	db, _ := newMock(t)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/send", sendBody(tenantA))
	w := httptest.NewRecorder()
	sendNotificationHandler(db, zap.NewNop()).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func notificationRouter(db *sqlx.DB) chi.Router {
	r := chi.NewRouter()
	r.Get("/{notificationID}", getNotificationStatusHandler(db, zap.NewNop()))
	r.Put("/{notificationID}/read", markAsReadHandler(db, zap.NewNop()))
	return r
}

func TestGetNotificationStatusHandler(t *testing.T) {
	db, mock := newMock(t)
	now := time.Now()
	mock.ExpectQuery(`FROM notifications WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs("test-id", tenantA).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id", "type", "subject", "message", "delivery_status", "read_at", "created_at", "updated_at"}).
			AddRow("test-id", tenantA, "t", "s", "m", "sent", nil, now, now))

	w := httptest.NewRecorder()
	notificationRouter(db).ServeHTTP(w, asTenant(httptest.NewRequest(http.MethodGet, "/test-id", nil), tenantA))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expectMet(t, mock)
}

// Another tenant's notification is not found - the lookup is scoped to the
// caller's tenant, so it cannot be read by ID.
func TestGetNotificationStatusHandler_OtherTenantNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM notifications WHERE id = \$1 AND tenant_id = \$2`).
		WithArgs("their-id", tenantA).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	w := httptest.NewRecorder()
	notificationRouter(db).ServeHTTP(w, asTenant(httptest.NewRequest(http.MethodGet, "/their-id", nil), tenantA))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	expectMet(t, mock)
}

func TestListNotificationsHandler(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM notifications\s+WHERE tenant_id = \$1\s+ORDER BY`).
		WithArgs(tenantA).
		WillReturnRows(sqlmock.NewRows([]string{"id", "type", "subject", "delivery_status", "created_at"}).
			AddRow("n1", "t", "s", "sent", time.Now()))

	// A spoofed header must not change the tenant.
	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-Tenant-ID", tenantB)
	w := httptest.NewRecorder()
	listNotificationsHandler(db, zap.NewNop()).ServeHTTP(w, asTenant(req, tenantA))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expectMet(t, mock)
}

func TestListNotificationsHandler_Unauthenticated(t *testing.T) {
	db, _ := newMock(t)
	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("X-Tenant-ID", tenantA)
	w := httptest.NewRecorder()
	listNotificationsHandler(db, zap.NewNop()).ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestMarkAsReadHandler(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`INSERT INTO notification_outbox .* FROM notifications WHERE id = \$3 AND tenant_id = \$4`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "test-id", tenantA).
		WillReturnResult(sqlmock.NewResult(0, 1))

	w := httptest.NewRecorder()
	notificationRouter(db).ServeHTTP(w, asTenant(httptest.NewRequest(http.MethodPut, "/test-id/read", nil), tenantA))

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
	expectMet(t, mock)
}

func TestMarkAsReadHandler_OtherTenantNotFound(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectExec(`INSERT INTO notification_outbox`).
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "their-id", tenantA).
		WillReturnResult(sqlmock.NewResult(0, 0))

	w := httptest.NewRecorder()
	notificationRouter(db).ServeHTTP(w, asTenant(httptest.NewRequest(http.MethodPut, "/their-id/read", nil), tenantA))

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
	expectMet(t, mock)
}

func TestGetDeliveryStatsHandler(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM notifications WHERE tenant_id = \$1`).
		WithArgs(tenantA).
		WillReturnRows(sqlmock.NewRows([]string{"total", "sent", "failed", "pending"}).AddRow(4, 3, 1, 0))

	w := httptest.NewRecorder()
	getDeliveryStatsHandler(db, zap.NewNop()).ServeHTTP(w, asTenant(httptest.NewRequest(http.MethodGet, "/api/notifications/stats/delivery", nil), tenantA))

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	expectMet(t, mock)
}
