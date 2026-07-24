package bp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

// httpHasuraClient is a tiny local HasuraClient implementation for tests that
// posts raw GraphQL JSON to a provided endpoint.
type httpHasuraClient struct {
	endpoint string
}

func (h *httpHasuraClient) do(body string, variables map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{"query": body, "variables": variables}
	b, _ := json.Marshal(payload)
	resp, err := http.Post(h.endpoint, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if _, ok := out["errors"]; ok {
		return out, nil
	}
	return out["data"].(map[string]interface{}), nil
}

func (h *httpHasuraClient) Query(q string, vars map[string]interface{}) (map[string]interface{}, error) {
	return h.do(q, vars)
}

func (h *httpHasuraClient) Mutate(m string, vars map[string]interface{}) (map[string]interface{}, error) {
	return h.do(m, vars)
}

// bytes and strings are used above

func TestIntegration_HasuraLoadTrigger(t *testing.T) {
	// Setup test Hasura server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		q, _ := payload["query"].(string)
		if q == "" {
			http.Error(w, "missing query", http.StatusBadRequest)
			return
		}

		if contains := strings.Contains(q, "bp_adaptive_triggers"); contains {
			resp := map[string]interface{}{"data": map[string]interface{}{"bp_adaptive_triggers": []interface{}{map[string]interface{}{
				"id":                "t-1",
				"tenant_id":         "tenant-x",
				"step_id":           "s-1",
				"trigger_name":      "Hasura Trigger",
				"trigger_condition": "cond",
				"trigger_type":      "event",
				"action_type":       "notify",
				"action_config":     map[string]interface{}{"v": 1},
				"context_variables": []interface{}{"a"},
				"is_active":         true,
			}}}}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		http.Error(w, "unknown query", http.StatusNotFound)
	}))
	defer srv.Close()

	// Create client that will post to /v1/graphql
	client := &httpHasuraClient{endpoint: srv.URL + "/v1/graphql"}

	te := NewTriggerEngineWithHasura(nil, client, nil, "tenant-x", log.New(io.Discard, "", 0))

	trg, err := te.loadTrigger(context.Background(), "t-1")
	if err != nil {
		t.Fatalf("loadTrigger integration failed: %v", err)
	}
	if trg.TriggerName != "Hasura Trigger" {
		t.Fatalf("unexpected trigger name: %s", trg.TriggerName)
	}
}

func TestIntegration_HasuraMutateRecordSuccess(t *testing.T) {
	// Start httptest server that responds to the mutation
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		q, _ := payload["query"].(string)
		if strings.Contains(q, "update_bp_trigger_events") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": map[string]interface{}{"update_bp_trigger_events": map[string]interface{}{"affected_rows": 1}}})
			return
		}
		http.Error(w, "unknown", http.StatusNotFound)
	}))
	defer srv.Close()

	client := &httpHasuraClient{endpoint: srv.URL + "/v1/graphql"}

	// Setup sqlmock DB to prove no fallback happens
	dbSQL, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbSQL, "sqlmock")
	defer sqlxDB.Close()

	te := NewTriggerEngineWithHasura(sqlxDB, client, nil, "tenant-x", log.New(io.Discard, "", 0))
	te.recordTriggerSuccess(context.Background(), "evt-10", "wf-99")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock expectations not met (no fallback expected): %v", err)
	}
}
