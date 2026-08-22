package account

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type testAccountService struct {
	records map[uuid.UUID]AccountRecord
}

func (s *testAccountService) List(ctx context.Context, tenantID uuid.UUID, subtypeCode string) ([]AccountRecord, error) {
	var result []AccountRecord
	for _, rec := range s.records {
		if rec.TenantID == tenantID && (subtypeCode == "" || rec.SubtypeCode == subtypeCode) {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (s *testAccountService) Get(ctx context.Context, tenantID, id uuid.UUID) (*AccountRecord, error) {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return nil, nil
	}
	return &rec, nil
}

func (s *testAccountService) Create(ctx context.Context, tenantID uuid.UUID, rec *AccountRecord) error {
	rec.TenantID = tenantID
	s.records[rec.ID] = *rec
	return nil
}

func (s *testAccountService) SoftDelete(ctx context.Context, tenantID, id uuid.UUID) error {
	rec, ok := s.records[id]
	if !ok || rec.TenantID != tenantID {
		return ErrNotFound
	}
	delete(s.records, id)
	return nil
}

func setupTestAccountRouter(svc *testAccountService) http.Handler {
	r := chi.NewRouter()
	h := NewHandlerWithService(svc)
	h.RegisterRoutes(r)
	return r
}

func withAuthContext(r *http.Request, tenantID uuid.UUID) *http.Request {
	claims := &jwtmiddleware.JWTClaims{
		TenantID: tenantID.String(),
		UserID:   "test-user",
		Email:    "test@example.com",
		IsActive: true,
	}
	ctx := context.WithValue(r.Context(), jwtmiddleware.ClaimsContextKey, claims)
	return r.WithContext(ctx)
}

func TestAccountHandler_List_Unauthorized(t *testing.T) {
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/accounts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestAccountHandler_List_WithAuth(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	rec := AccountRecord{
		ID:           uuid.New(),
		TenantID:     tenantID,
		SubtypeCode:  "institutional",
		AccountName: "Test Account",
		SponsorID:    uuidPtr(uuid.New()),
	}
	svc.records[rec.ID] = rec

	req := httptest.NewRequest("GET", "/api/oms/accounts", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var result []AccountRecord
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 record, got %d", len(result))
	}
}

func TestAccountHandler_Get_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/accounts/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestAccountHandler_Get_InvalidID(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	req := httptest.NewRequest("GET", "/api/oms/accounts/invalid-uuid", nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAccountHandler_Create_InvalidJSON(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	req := httptest.NewRequest("POST", "/api/oms/accounts", bytes.NewBufferString("invalid json"))
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAccountHandler_Create_Valid(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	rec := AccountRecord{
		ID:           uuid.New(),
		SubtypeCode:  "institutional",
		AccountName: "Test Account",
		SponsorID:    uuidPtr(uuid.New()),
	}
	body, _ := json.Marshal(rec)

	req := httptest.NewRequest("POST", "/api/oms/accounts", bytes.NewBuffer(body))
	req = withAuthContext(req, tenantID)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestAccountHandler_Delete_NotFound(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	req := httptest.NewRequest("DELETE", "/api/oms/accounts/"+uuid.New().String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestAccountHandler_Delete_Valid(t *testing.T) {
	tenantID := uuid.New()
	svc := &testAccountService{records: make(map[uuid.UUID]AccountRecord)}
	r := setupTestAccountRouter(svc)

	recID := uuid.New()
	rec := AccountRecord{
		ID:           recID,
		TenantID:     tenantID,
		SubtypeCode:  "institutional",
		AccountName: "Test Account",
		SponsorID:    uuidPtr(uuid.New()),
	}
	svc.records[recID] = rec

	req := httptest.NewRequest("DELETE", "/api/oms/accounts/"+recID.String(), nil)
	req = withAuthContext(req, tenantID)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
