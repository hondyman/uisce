package validationsdk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateRecordOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/validation-rule-nodes/evaluate-record" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(EvaluateRecordResponse{
			Valid:               false,
			Blocked:             true,
			EvaluatedRulesCount: 2,
			Violations:          []Violation{{RuleKey: "qty-limit", Severity: "BLOCK", RuleVersion: "1"}},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	res, err := c.ValidateRecord(context.Background(), EvaluateRecordRequest{
		BOName: "order",
		Record: map[string]any{"TargetQuantity": 500},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Blocked || len(res.Violations) != 1 || res.Violations[0].RuleKey != "qty-limit" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestValidateRecordContextRequired(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // domain outcome: 200 + error body
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":                  "ERR_SERVER_CONTEXT_REQUIRED",
			"missing_context_fields": []string{"account_status", "OrderAllocations"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.ValidateRecord(context.Background(), EvaluateRecordRequest{BOName: "order", Record: map[string]any{}})
	scre, ok := err.(*ServerContextRequiredError)
	if !ok {
		t.Fatalf("want *ServerContextRequiredError, got %v", err)
	}
	if len(scre.MissingFields) != 2 {
		t.Fatalf("missing fields = %v", scre.MissingFields)
	}
}

func TestRetryOn500ThenSuccess(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(EvaluateRecordResponse{Valid: true})
	}))
	defer srv.Close()

	c := New(srv.URL, WithRetry(RetryPolicy{MaxAttempts: 3, BaseDelay: time.Millisecond, MaxDelay: 2 * time.Millisecond}))
	res, err := c.ValidateRecord(context.Background(), EvaluateRecordRequest{BOName: "order", Record: map[string]any{}})
	if err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if !res.Valid || calls != 2 {
		t.Fatalf("calls=%d res=%+v", calls, res)
	}
}

func TestCircuitBreakerOpens(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := New(srv.URL,
		WithRetry(RetryPolicy{MaxAttempts: 1}),
		WithBreaker(shortBreaker()),
	)
	for i := 0; i < 6; i++ {
		_, _ = c.ValidateRecord(context.Background(), EvaluateRecordRequest{BOName: "x", Record: map[string]any{}})
	}
	err := func() error {
		_, err := c.ValidateRecord(context.Background(), EvaluateRecordRequest{BOName: "x", Record: map[string]any{}})
		return err
	}()
	if err != ErrCircuitOpen {
		t.Fatalf("want ErrCircuitOpen, got %v", err)
	}
}

// helper to build a fast breaker in tests
func shortBreaker() *CircuitBreaker { return &CircuitBreaker{failureThreshold: 5, cooldown: time.Hour} }
