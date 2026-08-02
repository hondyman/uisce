package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type DevBypassConfig struct {
	Fabric       bool
	Models       bool
	Catalog      bool
	BusinessTerm bool
	Views        bool
	XUser        bool
}

func DefaultDevBypassConfig() DevBypassConfig {
	return DevBypassConfig{
		Fabric:       true,
		Models:       true,
		Catalog:      true,
		BusinessTerm: true,
		Views:        true,
		XUser:        true,
	}
}

func CheckDevBypass(c *gin.Context, cfg DevBypassConfig) bool {
	path := c.Request.URL.Path
	method := c.Request.Method

	if strings.HasPrefix(path, "/api/fabric/") && cfg.Fabric {
		return true
	}

	if strings.HasPrefix(path, "/api/models") && cfg.Models {
		return true
	}

	if strings.HasPrefix(path, "/api/catalog") && cfg.Catalog {
		return true
	}

	if method == "GET" && path == "/api/business-term" && cfg.BusinessTerm {
		return true
	}

	if strings.HasPrefix(path, "/api/views") && cfg.Views {
		return true
	}

	if path == "/api/graphql" && method == "POST" {
		return true
	}

	commonDevPaths := []string{
		"/api/policies", "/api/bundles", "/api/semantic",
		"/api/business", "/api/data-domains", "/api/profiler",
		"/api/entity-schema", "/api/validation-rules", "/api/relationships",
		"/api/lineage", "/api/node-types", "/api/edge-types",
		"/api/bp-notifications", "/api/impact",
	}
	if cfg.Fabric {
		for _, p := range commonDevPaths {
			if strings.HasPrefix(path, p) {
				return true
			}
		}
	}

	if path == "/api/tenants" || strings.HasPrefix(path, "/api/tenants/") {
		if cfg.Fabric {
			return true
		}
	}

	if path == "/api/ip-whitelist" || strings.HasPrefix(path, "/api/ip-whitelist") {
		if cfg.Fabric {
			return true
		}
	}

	return false
}

func ShouldBypassAuth(c *gin.Context, cfg DevBypassConfig) bool {
	if CheckDevBypass(c, cfg) {
		return true
	}

	if cfg.XUser && c.GetHeader("X-User-ID") != "" {
		return true
	}

	return false
}
