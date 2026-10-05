package routing

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
)

func TestGatewayDispatcher_CrossPodForwardingAndLocalExecution(t *testing.T) {
	router := NewConsistentHashRouter(100)
	router.AddNode("pod-alpha")
	router.AddNode("pod-bravo")
	router.AddNode("pod-charlie")

	var podAlphaExecutions int64
	var podBravoExecutions int64
	var podCharlieExecutions int64

	// Create local mock handlers for Pod Alpha, Pod Bravo, Pod Charlie
	handlerAlpha := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&podAlphaExecutions, 1)
		w.Header().Set("X-Executed-By", "pod-alpha")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED","pod":"pod-alpha"}`))
	})
	handlerBravo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&podBravoExecutions, 1)
		w.Header().Set("X-Executed-By", "pod-bravo")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED","pod":"pod-bravo"}`))
	})
	handlerCharlie := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&podCharlieExecutions, 1)
		w.Header().Set("X-Executed-By", "pod-charlie")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED","pod":"pod-charlie"}`))
	})

	serverBravo := httptest.NewServer(handlerBravo)
	defer serverBravo.Close()
	serverCharlie := httptest.NewServer(handlerCharlie)
	defer serverCharlie.Close()

	dispatcherAlpha := NewGatewayDispatcher("pod-alpha", router, handlerAlpha)
	_ = dispatcherAlpha.RegisterPodEndpoint("pod-bravo", serverBravo.URL)
	_ = dispatcherAlpha.RegisterPodEndpoint("pod-charlie", serverCharlie.URL)

	for i := 0; i < 30; i++ {
		accID := uuid.New()
		expectedPod, err := router.RouteAccount(accID)
		if err != nil {
			t.Fatalf("RouteAccount error: %v", err)
		}

		reqBody := fmt.Sprintf(`{"accountId":"%s","securityId":"sec-1","quantity":100}`, accID.String())
		req := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBufferString(reqBody))
		rec := httptest.NewRecorder()

		dispatcherAlpha.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("Request for account %s failed with code %d: %s", accID, rec.Code, rec.Body.String())
		}

		resBody, _ := io.ReadAll(rec.Body)
		executedPod := rec.Header().Get("X-Executed-By")
		if executedPod != expectedPod {
			t.Errorf("Order for account %s assigned to %s but executed on %s (Body: %s)", accID, expectedPod, executedPod, string(resBody))
		}
	}

	t.Logf("Validated Multi-Pod Ingress Dispatch spread: Pod Alpha=%d, Pod Bravo=%d, Pod Charlie=%d (Proxy misroutes: %d)",
		atomic.LoadInt64(&podAlphaExecutions), atomic.LoadInt64(&podBravoExecutions), atomic.LoadInt64(&podCharlieExecutions), dispatcherAlpha.ProxyMisrouteCount())
}

func TestGatewayDispatcher_LoopGuard(t *testing.T) {
	router := NewConsistentHashRouter(100)
	router.AddNode("pod-bravo")

	dispatcherAlpha := NewGatewayDispatcher("pod-alpha", router, nil)
	_ = dispatcherAlpha.RegisterPodEndpoint("pod-bravo", "http://127.0.0.1:9999")

	accID := uuid.New()
	reqBody := fmt.Sprintf(`{"accountId":"%s"}`, accID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBufferString(reqBody))
	// Simulate request that has already been forwarded MaxProxyHops times
	req.Header.Set(HeaderForwardedHopCount, "2")
	rec := httptest.NewRecorder()

	dispatcherAlpha.ServeHTTP(rec, req)

	if rec.Code != http.StatusLoopDetected {
		t.Fatalf("Expected HTTP 508 Loop Detected, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayDispatcher_FailClosedSemantics(t *testing.T) {
	router := NewConsistentHashRouter(100)
	router.AddNode("pod-bravo") // Route to pod-bravo

	// Pod-alpha does NOT have pod-bravo registered in endpoint map
	dispatcherAlpha := NewGatewayDispatcher("pod-alpha", router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("CRITICAL RACE: Local handler was invoked on unassigned pod during outage! Fail-open occurred instead of fail-closed.")
	}))

	accID := uuid.New()
	reqBody := fmt.Sprintf(`{"accountId":"%s"}`, accID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBufferString(reqBody))
	rec := httptest.NewRecorder()

	dispatcherAlpha.ServeHTTP(rec, req)

	// Must fail-closed with 502 Bad Gateway
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("Expected fail-closed 502 Bad Gateway, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGatewayDispatcher_IngressHeaderAuthority(t *testing.T) {
	// Router says pod-bravo for all accounts
	router := NewConsistentHashRouter(100)
	router.AddNode("pod-bravo")

	localExecuted := false
	dispatcherAlpha := NewGatewayDispatcher("pod-alpha", router, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		localExecuted = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"APPROVED"}`))
	}))

	accID := uuid.New()
	reqBody := fmt.Sprintf(`{"accountId":"%s"}`, accID.String())
	req := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBufferString(reqBody))
	// Ingress (Envoy) explicitly stamped pod-alpha as assigned pod
	req.Header.Set(HeaderAssignedPod, "pod-alpha")
	rec := httptest.NewRecorder()

	dispatcherAlpha.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
	}
	if !localExecuted {
		t.Fatalf("Ingress header authority failed: request was not executed locally despite HeaderAssignedPod=pod-alpha")
	}
	if dispatcherAlpha.ProxyMisrouteCount() != 0 {
		t.Errorf("Expected 0 proxy misroutes when ingress header matches, got %d", dispatcherAlpha.ProxyMisrouteCount())
	}
}
