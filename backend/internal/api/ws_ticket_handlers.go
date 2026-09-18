package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/hondyman/uisce/backend/internal/auth"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

// IssueTicketResponse is the JSON payload returned on successful ticket issuance.
type IssueTicketResponse struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
}

// issueWsTicket issues a single-use, 30s opaque ticket for WebSocket authentication.
// Requires a valid Bearer JWT via Authorization header or authenticated user session.
func (s *Server) issueWsTicket(w http.ResponseWriter, r *http.Request) {
	// Extract caller identity: try jwtmiddleware first, fallback to DBSessionService/SecMgr
	var tenantID, userID string

	// Check Authorization header via jwtmiddleware
	claims, err := jwtmiddleware.ValidateTokenFromRequest(r)
	if err == nil && claims != nil {
		tenantID = claims.TenantID
		userID = claims.UserID
	} else {
		// Fallback: check session token cookie or Bearer session token
		authHeader := r.Header.Get("Authorization")
		var token string
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		} else if cookie, err := r.Cookie("session_token"); err == nil && cookie.Value != "" {
			token = cookie.Value
		}

		if token != "" {
			if s.SecMgr != nil {
				if parsed, err := s.SecMgr.ParseToken(token); err == nil {
					if uid, ok := parsed["user_id"].(string); ok {
						userID = uid
					}
					if tid, ok := parsed["tenant_id"].(string); ok {
						tenantID = tid
					}
				}
			}
			if userID == "" && s.DB != nil {
				_ = s.DB.QueryRowContext(r.Context(), `
					SELECT u.id, COALESCE(u.tenant_id, '')
					FROM private_markets_sessions s
					JOIN public.users u ON s.user_id = u.id
					WHERE s.session_token = $1 AND s.expires_at > now() AND s.is_active = true
				`, token).Scan(&userID, &tenantID)
			}
		}
	}

	if userID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	vault := s.WsVault
	if vault == nil {
		vault = auth.NewWsTicketVault(auth.DefaultWsTicketVaultConfig())
		s.WsVault = vault
	}

	ticket, expiresIn, err := vault.IssueTicket(tenantID, userID)
	if err != nil {
		if err == auth.ErrRateLimitExceeded {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
			return
		}
		if err == auth.ErrVaultCapacityExceeded {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "vault capacity reached"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to issue ticket"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(IssueTicketResponse{
		Ticket:    ticket,
		ExpiresIn: expiresIn,
	})
}

// handleWebSocketTicketAndUpgrade handles the WebSocket upgrade using ticket-based auth or legacy fallback.
// Invariant: CheckOrigin must run BEFORE consuming the ticket to prevent hostile origin ticket burning.
// If origin fails, ticket is untouched. Once origin passes, ticket is atomically consumed.
// All authentication failures return an identical HTTP 401 {"error":"unauthorized"} body.
func (s *Server) handleWebSocketTicketAndUpgrade(w http.ResponseWriter, r *http.Request) {
	// 1. Check Origin FIRST before burning or consuming tickets
	if upgrader.CheckOrigin != nil && !upgrader.CheckOrigin(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "forbidden origin"})
		return
	}

	path := r.URL.Path
	ticket := r.URL.Query().Get("ticket")

	var userID, tenantID, audience string
	var jobAudience string

	if ticket != "" {
		// Ticket authentication flow
		vault := s.WsVault
		if vault == nil {
			vault = auth.NewWsTicketVault(auth.DefaultWsTicketVaultConfig())
			s.WsVault = vault
		}

		tid, uid, err := vault.ConsumeTicket(ticket)
		if err != nil {
			// Strict no-oracle rule: identical 401 response regardless of cause
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		userID = uid
		tenantID = tid
		if tenantID != "" {
			r.Header.Set("X-Tenant-ID", tenantID)
		}
	} else if strings.HasPrefix(path, "/ws/profiler/") {
		// Job-scoped profiler connection
		jobId := strings.TrimPrefix(path, "/ws/profiler/")
		if jobId == "" {
			http.Error(w, "missing job id", http.StatusBadRequest)
			return
		}

		tokenString := r.URL.Query().Get("token")
		allowLegacy := os.Getenv("WS_ALLOW_LEGACY_JWT")
		// Default to allowed during migration window unless explicitly disabled
		if allowLegacy == "false" || allowLegacy == "0" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		if tokenString == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		log.Printf("[DEPRECATION WARNING] WebSocket connected via query string ?token= parameter on path %s. Migrate to POST /api/ws/ticket and ?ticket=", path)

		claims, err := s.validateWsToken(tokenString, jobId)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		if t, ok := claims["tenant_id"].(string); ok && t != "" {
			r.Header.Set("X-Tenant-ID", t)
			tenantID = t
		}
		if ds, ok := claims["datasource_id"].(string); ok && ds != "" {
			r.Header.Set("X-Tenant-Datasource-ID", ds)
		}

		jobAudience = jobId
	} else {
		// Check for legacy token param or session auth
		tokenString := r.URL.Query().Get("token")
		allowLegacy := os.Getenv("WS_ALLOW_LEGACY_JWT")

		if tokenString != "" {
			if allowLegacy == "false" || allowLegacy == "0" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			log.Printf("[DEPRECATION WARNING] WebSocket connected via query string ?token= parameter on path %s. Migrate to POST /api/ws/ticket and ?ticket=", path)

			claims, err := jwtmiddleware.ValidateToken(tokenString)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
			userID = claims.UserID
			tenantID = claims.TenantID
			if tenantID != "" {
				r.Header.Set("X-Tenant-ID", tenantID)
			}
		} else {
			// Authorization header or session token flow
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				bearerToken := strings.TrimPrefix(authHeader, "Bearer ")
				if s.DB != nil {
					err := s.DB.QueryRowContext(r.Context(), `
						SELECT u.id, u.role
						FROM private_markets_sessions s
						JOIN public.users u ON s.user_id = u.id
						WHERE s.session_token = $1 AND s.expires_at > now() AND s.is_active = true
					`, bearerToken).Scan(&userID, &audience)
					if err != nil {
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusUnauthorized)
						_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
						return
					}
				}
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
				return
			}
		}
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	// Create and register WebSocket client if hub exists
	if s.WsHub != nil {
		client := &WebSocketClient{
			conn:     conn,
			send:     make(chan []byte, 256),
			userID:   userID,
			tenantID: tenantID,
			audience: audience,
			hub:      s.WsHub,
		}
		if jobAudience != "" {
			client.audience = jobAudience
		}

		s.WsHub.register <- client

		go client.writePump()
		go client.readPump()
	}
}
