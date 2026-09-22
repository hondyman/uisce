package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

const (
	testTenantID      = "99e99e99-99e9-49e9-89e9-99e99e99e999"
	altTenantID       = "11111111-1111-1111-1111-111111111111"
	nonMemberTenantID = "00000000-0000-4000-8000-000000000000"
)

func TestMCP_ToolsList(t *testing.T) {
	handler := NewMCPToolHandler(nil)

	reqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": "1",
		"method": "tools/list"
	}`)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	rec := httptest.NewRecorder()

	handler.HandleRPC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding JSON-RPC response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	resMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map result, got %T", resp.Result)
	}

	tools, ok := resMap["tools"].([]interface{})
	if !ok || len(tools) < 2 {
		t.Fatalf("expected at least 2 tools, got %v", tools)
	}
}

func TestMCP_GetContractCall(t *testing.T) {
	handler := NewMCPToolHandler(nil)

	reqBody := []byte(`{
		"jsonrpc": "2.0",
		"id": "2",
		"method": "tools/call",
		"params": {
			"name": "get_business_object_contract",
			"arguments": {
				"tenant_id": "99e99e99-99e9-49e9-89e9-99e99e99e999",
				"bo_key": "customer_profile"
			}
		}
	}`)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(reqBody))
	req = req.WithContext(contextWithAuth(testTenantID))
	rec := httptest.NewRecorder()

	handler.HandleRPC(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}
}

func contextWithAuth(tenantIDs ...string) context.Context {
	if len(tenantIDs) == 0 {
		return context.Background()
	}
	return security.WithAuthInfo(context.Background(), security.AuthInfo{
		UserID:    "test-user",
		TenantIDs: tenantIDs,
	})
}

func reqWithAuth(method string, toolName string, args map[string]interface{}, tenantIDs ...string) (*http.Request, *httptest.ResponseRecorder) {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      "test-id",
		"method":  method,
		"params":  params,
	}
	body, _ := json.Marshal(reqBody)
	r := httptest.NewRequest("POST", "/mcp", bytes.NewBuffer(body))
	if len(tenantIDs) > 0 {
		r = r.WithContext(contextWithAuth(tenantIDs...))
	}
	return r, httptest.NewRecorder()
}

func TestMCP_TenantMismatch_BodyTenantNotInAuthSet(t *testing.T) {
	handler := NewMCPToolHandler(nil)
	args := map[string]interface{}{
		"tenant_id": nonMemberTenantID,
	}
	r, rec := reqWithAuth("tools/call", "list_pages", args, testTenantID)
	handler.HandleRPC(rec, r)

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error == nil {
		t.Fatal("expected mismatch error when body tenant_id not in auth set")
	}
	if resp.Error.Code != InvalidParams {
		t.Fatalf("expected InvalidParams (%d), got %d", InvalidParams, resp.Error.Code)
	}
	expected := "tenant_id in body is not a member of your authenticated tenant set"
	if resp.Error.Message != expected {
		t.Fatalf("expected message %q, got %q", expected, resp.Error.Message)
	}
}

func TestMCP_TenantMismatch_MatchNonDefaultTenant(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	defer mockDB.Close()

	sqlxDB := sqlx.NewDb(mockDB, "postgres")
	defer sqlxDB.Close()

	gold := uuid.MustParse("99999999-9999-4999-8999-999999999999")
	mock.ExpectQuery("uisce_gold_copy_tenant_id").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(gold))
	rows := sqlmock.NewRows([]string{"id", "name", "slug", "status"})
	mock.ExpectQuery("FROM public.page_definitions").
		WithArgs(uuid.MustParse(altTenantID), gold).
		WillReturnRows(rows)

	handler := NewMCPToolHandler(sqlxDB)
	args := map[string]interface{}{
		"tenant_id": altTenantID,
	}
	r, rec := reqWithAuth("tools/call", "list_pages", args, testTenantID, altTenantID)
	handler.HandleRPC(rec, r)

	var resp JSONRPCResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected RPC error: %v", resp.Error)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met: %v (handler dispatched to wrong tenant or skipped DB)", err)
	}
}
