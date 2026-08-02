package handlers

import (
	"net/http"

	"github.com/hondyman/uisce/libs/jwt-middleware"
)

func getSecureTenantID(r *http.Request) string {
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil && claims.TenantID != "" {
		return claims.TenantID
	}
	return r.Header.Get("X-Tenant-ID")
}
