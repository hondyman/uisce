package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type contextKey string

const (
	AssignedPodContextKey contextKey = "assigned_gateway_pod"
	AccountIDContextKey   contextKey = "account_id"
	HeaderAssignedPod                = "X-Assigned-Gateway-Pod"
	HeaderAccountID                  = "X-Account-ID"
)

// OrderIngressPayload represents minimal fields needed from trade orders for routing
type OrderIngressPayload struct {
	AccountID  *uuid.UUID `json:"account_id,omitempty"`
	AccountID2 *uuid.UUID `json:"accountId,omitempty"`
}

// ExtractAccountID extracts the AccountID from FIX/DMA headers or JSON body
func ExtractAccountID(r *http.Request) (uuid.UUID, error) {
	// 1. Check header first (e.g. FIX/DMA Gateway injection)
	if h := r.Header.Get(HeaderAccountID); h != "" {
		return uuid.Parse(h)
	}

	// 2. Inspect request body if POST/PUT
	if r.Body == nil || (r.Method != http.MethodPost && r.Method != http.MethodPut) {
		return uuid.Nil, errors.New("empty body or unsupported method")
	}

	bodyBytes, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		return uuid.Nil, readErr
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var payload OrderIngressPayload
	if jsonErr := json.Unmarshal(bodyBytes, &payload); jsonErr != nil {
		return uuid.Nil, jsonErr
	}

	if payload.AccountID != nil {
		return *payload.AccountID, nil
	}
	if payload.AccountID2 != nil {
		return *payload.AccountID2, nil
	}

	return uuid.Nil, errors.New("accountId not found in payload")
}

// AccountAffinityMiddleware inspects incoming trade orders and assigns them to the deterministic gateway pod
func AccountAffinityMiddleware(router *ConsistentHashRouter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			accountID, err := ExtractAccountID(r)
			if err == nil && accountID != uuid.Nil {
				targetPod, routeErr := router.RouteAccount(accountID)
				if routeErr == nil {
					w.Header().Set(HeaderAssignedPod, targetPod)
					ctx := context.WithValue(r.Context(), AssignedPodContextKey, targetPod)
					ctx = context.WithValue(ctx, AccountIDContextKey, accountID)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
