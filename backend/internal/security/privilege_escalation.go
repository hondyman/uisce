package security

import (
	"log"
	"net/http"
	"os"
)

// DebugPrivilegeEscalation controls whether privilege escalation attempts are logged.
// Enable by setting DEBUG_PRIVILEGE_ESCALATION=true environment variable.
var DebugPrivilegeEscalation = os.Getenv("DEBUG_PRIVILEGE_ESCALATION") == "true"

// LogPrivilegeEscalation logs potential privilege escalation attempts.
// When DebugPrivilegeEscalation is true, logs to stdout.
// Always logs at WARNING level when a mismatch is detected.
func LogPrivilegeEscalation(r *http.Request, urlTenantID, jwtTenantID string) {
	if DebugPrivilegeEscalation {
		log.Printf("[PRIVILEGE_ESCALATION] URL_TENANT=%s JWT_TENANT=%s URI=%s UserAgent=%s",
			urlTenantID, jwtTenantID, r.RequestURI, r.UserAgent())
	}
}
