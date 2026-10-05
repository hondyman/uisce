package routing

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	MaxProxyHops              = 2
	HeaderForwardedHopCount   = "X-Forwarded-Hop-Count"
	HeaderForwardedByPod      = "X-Forwarded-By-Pod"
)

// MetricProxyMisroutes counts requests backstopped via inter-pod proxying
var MetricProxyMisroutes = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "uisce",
		Subsystem: "compliance",
		Name:      "proxy_misroutes_total",
		Help:      "Total number of order evaluations received by non-authoritative pods requiring inter-pod proxying",
	},
	[]string{"current_pod", "assigned_pod"},
)

// GatewayDispatcher enforces single-replica ownership per AccountID.
//
// Topology Architecture:
// 1. Primary Path: Ingress Load Balancer (Envoy hash policy on AccountID) routes ~100% of orders directly to the owner pod.
// 2. Backstop Path: This in-pod dispatcher catches residual misroutes (ring transition / LB drift) and proxies to the owner pod.
// 3. Fail-Closed Policy: If the target authoritative pod is unreachable, the dispatcher strictly FAILS CLOSED (rejects order)
//    to prevent multi-pod concurrent race conditions.
type GatewayDispatcher struct {
	currentPodID   string
	router         *ConsistentHashRouter
	podEndpoints   map[string]*url.URL
	localHandler   http.Handler
	mu             sync.RWMutex
	proxyMisroutes int64 // Atomic counter for proxy-misroute backstop telemetry
}

// NewGatewayDispatcher creates an authoritative gateway dispatcher
func NewGatewayDispatcher(currentPodID string, router *ConsistentHashRouter, localHandler http.Handler) *GatewayDispatcher {
	return &GatewayDispatcher{
		currentPodID: currentPodID,
		router:       router,
		podEndpoints: make(map[string]*url.URL),
		localHandler: localHandler,
	}
}

// RegisterPodEndpoint registers reachable network address of a pod replica
func (d *GatewayDispatcher) RegisterPodEndpoint(podID string, rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid pod endpoint URL: %w", err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.podEndpoints[podID] = u
	return nil
}

// ProxyMisrouteCount returns the total number of requests backstopped via reverse-proxying
func (d *GatewayDispatcher) ProxyMisrouteCount() int64 {
	return atomic.LoadInt64(&d.proxyMisroutes)
}

// ServeHTTP inspects AccountID, routes it, and either handles locally or reverse-proxies to the assigned pod
func (d *GatewayDispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Loop Guard: Check hop count
	hopCount := 0
	if hopHeader := r.Header.Get(HeaderForwardedHopCount); hopHeader != "" {
		if parsedHops, err := strconv.Atoi(hopHeader); err == nil {
			hopCount = parsedHops
		}
	}
	if hopCount >= MaxProxyHops {
		http.Error(w, fmt.Sprintf("Routing Error: proxy forwarding loop detected (exceeded %d hops)", MaxProxyHops), http.StatusLoopDetected)
		return
	}

	accountID, err := ExtractAccountID(r)
	if err != nil || accountID == uuid.Nil {
		http.Error(w, fmt.Sprintf("Routing Error: missing or invalid accountId in request: %v", err), http.StatusBadRequest)
		return
	}

	// Ingress Authority Principle: If Envoy / Ingress LB already assigned the target pod, trust it directly.
	// The in-pod Go router ring is used ONLY as a fallback for unassigned internal OMS direct traffic.
	assignedPod := r.Header.Get(HeaderAssignedPod)
	if assignedPod == "" {
		var routeErr error
		assignedPod, routeErr = d.router.RouteAccount(accountID)
		if routeErr != nil {
			// Fail-Closed
			http.Error(w, fmt.Sprintf("Routing Error: %v", routeErr), http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Set(HeaderAssignedPod, assignedPod)

	// Case 1: Assigned to THIS pod replica -> Execute locally against in-memory state
	if assignedPod == d.currentPodID {
		ctx := context.WithValue(r.Context(), AssignedPodContextKey, assignedPod)
		ctx = context.WithValue(ctx, AccountIDContextKey, accountID)
		d.localHandler.ServeHTTP(w, r.WithContext(ctx))
		return
	}

	// Case 2: Assigned to ANOTHER pod replica -> Backstop proxy to authoritative pod
	atomic.AddInt64(&d.proxyMisroutes, 1)
	MetricProxyMisroutes.WithLabelValues(d.currentPodID, assignedPod).Inc()

	d.mu.RLock()
	targetURL, exists := d.podEndpoints[assignedPod]
	d.mu.RUnlock()

	if !exists {
		// Strict Fail-Closed: Do NOT fail-open to local execution
		http.Error(w, fmt.Sprintf("Routing Error (Fail-Closed): authoritative gateway pod %s is unavailable", assignedPod), http.StatusBadGateway)
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	// Custom ErrorHandler enforces Fail-Closed when upstream pod connection fails
	proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, proxyErr error) {
		http.Error(rw, fmt.Sprintf("Routing Error (Fail-Closed): upstream pod %s connection refused: %v", assignedPod, proxyErr), http.StatusServiceUnavailable)
	}

	// Increment hop count for loop guard
	r.Header.Set(HeaderForwardedHopCount, strconv.Itoa(hopCount+1))
	r.Header.Set(HeaderForwardedByPod, d.currentPodID)

	proxy.ServeHTTP(w, r)
}
