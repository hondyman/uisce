package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// AuthRequestsTotal tracks authentication requests
var AuthRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "auth_requests_total",
		Help: "Total authentication requests",
	},
	[]string{"status", "reason"},
)

// AuthorizationFailures tracks authorization failures
var AuthorizationFailures = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "authorization_failures_total",
		Help: "Total authorization failures",
	},
	[]string{"tenant_id", "resource_type", "reason"},
)

// HTTPRequestsTotal tracks HTTP requests
var HTTPRequestsTotal = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests",
	},
	[]string{"method", "endpoint", "status_code"},
)
