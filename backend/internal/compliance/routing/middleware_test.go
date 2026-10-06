package routing

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestAccountAffinityMiddleware_HTTPDispatch(t *testing.T) {
	router := NewConsistentHashRouter(100)
	router.AddNode("gateway-pod-alpha")
	router.AddNode("gateway-pod-bravo")
	router.AddNode("gateway-pod-charlie")

	accountA := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000001")
	accountB := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000002")

	var capturedPodA, capturedPodB string

	handler := AccountAffinityMiddleware(router)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pod, _ := r.Context().Value(AssignedPodContextKey).(string)
		accID, _ := r.Context().Value(AccountIDContextKey).(uuid.UUID)

		if accID == accountA {
			capturedPodA = pod
		} else if accID == accountB {
			capturedPodB = pod
		}
		w.WriteHeader(http.StatusOK)
	}))

	// 1. Test dispatch via JSON Request Body (Internal OMS Order)
	reqBodyA := []byte(`{"accountId":"018f2d5e-7a42-7000-8000-000000000001","securityId":"sec-1","quantity":100}`)
	reqA := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBuffer(reqBodyA))
	reqA.Header.Set("Content-Type", "application/json")
	recA := httptest.NewRecorder()
	handler.ServeHTTP(recA, reqA)

	if recA.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", recA.Code)
	}
	assignedHeaderA := recA.Header().Get(HeaderAssignedPod)
	if assignedHeaderA == "" || assignedHeaderA != capturedPodA {
		t.Fatalf("Expected assigned pod header matching context pod %s, got %s", capturedPodA, assignedHeaderA)
	}

	// 2. Test dispatch via FIX Header (External FIX/DMA Gateway)
	reqB := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", nil)
	reqB.Header.Set(HeaderAccountID, accountB.String())
	recB := httptest.NewRecorder()
	handler.ServeHTTP(recB, reqB)

	if recB.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", recB.Code)
	}
	assignedHeaderB := recB.Header().Get(HeaderAssignedPod)
	if assignedHeaderB == "" || assignedHeaderB != capturedPodB {
		t.Fatalf("Expected assigned pod header matching context pod %s, got %s", capturedPodB, assignedHeaderB)
	}

	// 3. Send 1,000 requests for Account A and ensure 100% land on capturedPodA
	for i := 0; i < 1000; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/oms/trade-orders", bytes.NewBuffer(reqBodyA))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Header().Get(HeaderAssignedPod) != capturedPodA {
			t.Fatalf("Order %d for Account A routed to %s instead of deterministic pod %s", i, w.Header().Get(HeaderAssignedPod), capturedPodA)
		}
	}

	t.Logf("Validated: Account A deterministically routed to %s; Account B routed to %s", capturedPodA, capturedPodB)
}
