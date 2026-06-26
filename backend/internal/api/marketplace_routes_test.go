package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// Ensure that when a DB row has a NULL icon_emoji the handler still returns
// a valid JSON response and does not inject plain-text error lines.
func TestHandleListMarketplaceItems_NullIconEmojiReturnsValidJSON(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	// Build rows where icon_emoji is NULL (represented by nil)
	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "item_type", "version", "category",
		"subcategories", "severity", "icon_emoji", "color_hex", "summary",
		"long_description", "implementation_json", "scope", "rule_type", "frequency",
		"evaluation_order", "is_public", "is_official", "is_core", "status",
		"external_api_providers", "requires_credentials", "usage_count", "rating", "downloads_count",
		"created_at", "updated_at", "published_at",
	}).AddRow(
		"ee9e7d85-a334-482b-92ef-dfc95a97d14e",
		"ESG Compliance",
		"Validate environmental, social and governance compliance requirements",
		"rule",
		"1.0.0",
		"ESG & Sustainability",
		[]byte("{ESG,Compliance}"),
		"BLOCK",
		nil, // icon_emoji NULL in DB
		"#10B981",
		"Ensure ESG compliance before executing trades",
		"Checks environmental, social, and governance metrics against policy thresholds.",
		[]byte("{}"),
		"PORTFOLIO",
		"CONDITION",
		"ON_TRADE",
		1,
		true,
		true,
		true,
		"active",
		[]byte("{}"),
		false,
		0,
		nil,
		0,
		time.Now(),
		time.Now(),
		nil,
	)

	// Expect the main query (allow various spacing/newlines via regex)
	mock.ExpectQuery("SELECT[\\s\\S]*FROM marketplace_items[\\s\\S]*LIMIT").WillReturnRows(rows)

	// Expect the count query
	countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM marketplace_items").WillReturnRows(countRows)

	handler := handleListMarketplaceItems(db)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/marketplace/items?status=active", nil)
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d; body=%s", rr.Code, rr.Body.String())
	}

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected application/json Content-Type, got %s", ct)
	}

	var resp MarketplaceSearchResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response JSON: %v; body=%s", err, rr.Body.String())
	}

	if len(resp.Items) != 1 {
		t.Fatalf("expected 1 item in response, got %d", len(resp.Items))
	}

	if resp.Items[0].IconEmoji != "" {
		t.Fatalf("expected empty IconEmoji for NULL DB value, got '%s'", resp.Items[0].IconEmoji)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

// When the DB returns a value that will cause rows.Scan to error (for example
// a string where json.RawMessage/[]byte is expected), the handler should
// fail-fast and return a structured JSON error (500) rather than corrupting
// the JSON response stream.
func TestHandleListMarketplaceItems_ScanErrorReturnsStructuredJSONError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock database: %v", err)
	}
	defer db.Close()

	// Construct rows where implementation_json is returned as a string value
	// (driver.Value type string) which will not scan into a json.RawMessage
	// (expects []byte) and should trigger a scan error.
	rows := sqlmock.NewRows([]string{
		"id", "name", "description", "item_type", "version", "category",
		"subcategories", "severity", "icon_emoji", "color_hex", "summary",
		"long_description", "implementation_json", "scope", "rule_type", "frequency",
		"evaluation_order", "is_public", "is_official", "is_core", "status",
		"external_api_providers", "requires_credentials", "usage_count", "rating", "downloads_count",
		"created_at", "updated_at", "published_at",
	}).AddRow(
		"bad-row-id",
		"Bad Item",
		"This row contains a bad implementation_json value",
		"rule",
		"1.0.0",
		"Misc",
		[]byte("{misc}"),
		"INFO",
		"🌶️",
		"#000000",
		"Bad item",
		"Long description",
		"not-bytes-json", // <-- string that will cause scan error into json.RawMessage
		"PORTFOLIO",
		"CONDITION",
		"ON_TRADE",
		10,
		true,
		false,
		false,
		"active",
		[]byte("{}"),
		false,
		0,
		nil,
		0,
		time.Now(),
		time.Now(),
		nil,
	)

	mock.ExpectQuery("SELECT[\\s\\S]*FROM marketplace_items[\\s\\S]*LIMIT").WillReturnRows(rows)

	// The count query may not be reached when scan fails; we do not set an
	// expectation for it because handler should fail before counting.

	handler := handleListMarketplaceItems(db)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/marketplace/items?status=active", nil)
	handler(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 Internal Server Error, got %d; body=%s", rr.Code, rr.Body.String())
	}

	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected application/json Content-Type, got %s", ct)
	}

	var errResp ErrorResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to unmarshal error response: %v; body=%s", err, rr.Body.String())
	}

	if errResp.Code != http.StatusInternalServerError {
		t.Fatalf("expected error response code %d, got %d", http.StatusInternalServerError, errResp.Code)
	}

	if errResp.ErrorCode != "scan_error" {
		t.Fatalf("expected error_code 'scan_error', got '%s'", errResp.ErrorCode)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
