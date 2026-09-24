package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/identity"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/security"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

const (
	// MaxWorkspaceLayoutPayloadBytes enforces a strict 1MB payload cap
	MaxWorkspaceLayoutPayloadBytes = 1024 * 1024
	DefaultLayoutProfileName       = "default"
	DefaultLayoutSchemaVersion     = "v1"
)

type WorkspaceLayoutHandler struct {
	db *sql.DB
}

func NewWorkspaceLayoutHandler(db *sql.DB) *WorkspaceLayoutHandler {
	return &WorkspaceLayoutHandler{db: db}
}

func (h *WorkspaceLayoutHandler) RegisterRoutes(r chi.Router) {
	r.Route("/user/preferences/workspace-layout", func(r chi.Router) {
		r.Get("/", h.GetLayout)
		r.Post("/", h.SaveLayout)
		r.Get("/profiles", h.ListProfiles)
	})
}

type WorkspaceLayoutResponse struct {
	ID            string          `json:"id"`
	TenantID      string          `json:"tenant_id"`
	UserID        string          `json:"user_id"`
	ProfileName   string          `json:"profile_name"`
	SchemaVersion string          `json:"schema_version"`
	LayoutData    json.RawMessage `json:"layout_data"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type WorkspaceLayoutProfileSummary struct {
	ID            string    `json:"id"`
	ProfileName   string    `json:"profile_name"`
	SchemaVersion string    `json:"schema_version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type SaveWorkspaceLayoutRequest struct {
	ProfileName   string          `json:"profile_name,omitempty"`
	SchemaVersion string          `json:"schema_version,omitempty"`
	LayoutData    json.RawMessage `json:"layout_data"`
}

func (h *WorkspaceLayoutHandler) resolveUserAndTenant(r *http.Request) (string, uuid.UUID, error) {
	// 1. Try JWT Claims in context
	if claims := jwtmiddleware.GetClaimsFromContext(r); claims != nil {
		uid := claims.UserID
		if uid == "" {
			uid = claims.Subject
		}
		if uid != "" && claims.TenantID != "" {
			if tid, err := uuid.Parse(claims.TenantID); err == nil && tid != uuid.Nil {
				return uid, tid, nil
			}
		}
	}

	// 2. Try validating token from Authorization header if not already in context
	if tokenStr, err := jwtmiddleware.ExtractToken(r); err == nil && tokenStr != "" {
		if tokenClaims, err := jwtmiddleware.ValidateToken(tokenStr); err == nil && tokenClaims != nil {
			uid := tokenClaims.UserID
			if uid == "" {
				uid = tokenClaims.Subject
			}
			if uid != "" && tokenClaims.TenantID != "" {
				if tid, err := uuid.Parse(tokenClaims.TenantID); err == nil && tid != uuid.Nil {
					return uid, tid, nil
				}
			}
		}
	}

	// 3. Try Security AuthInfo
	if auth, ok := security.AuthInfoFromContext(r.Context()); ok && len(auth.TenantIDs) > 0 {
		uid := auth.UserID
		if uid != "" {
			if tid, err := uuid.Parse(auth.TenantIDs[0]); err == nil && tid != uuid.Nil {
				return uid, tid, nil
			}
		}
	}

	// 4. Try Identity context
	if uid, ok := identity.ActorIDFromContext(r.Context()); ok && uid != "" {
		if tidStr, ok := identity.TenantIDFromContext(r.Context()); ok {
			if tid, err := uuid.Parse(tidStr); err == nil && tid != uuid.Nil {
				return uid, tid, nil
			}
		}
	}

	// 5. Dev/test fallback headers if permitted
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	allowHeaderFallback := env == "development" || env == "local" || env == "test" || os.Getenv("ALLOW_CLIENT_TENANT_HEADER_FALLBACK") == "true"
	if allowHeaderFallback {
		uidHeader := r.Header.Get("X-User-ID")
		tidHeader := r.Header.Get("X-Tenant-ID")
		if uidHeader != "" && tidHeader != "" {
			if tid, err := uuid.Parse(tidHeader); err == nil && tid != uuid.Nil {
				return uidHeader, tid, nil
			}
		}
	}

	return "", uuid.Nil, errors.New("unauthorized: missing or invalid user and tenant identification")
}

// GetLayout retrieves a user's workspace layout profile by name (default: "default")
func (h *WorkspaceLayoutHandler) GetLayout(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := h.resolveUserAndTenant(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	profileName := r.URL.Query().Get("profile")
	if profileName == "" {
		profileName = DefaultLayoutProfileName
	}

	query := `
		SELECT id, tenant_id, user_id, profile_name, schema_version, layout_data, created_at, updated_at
		FROM public.user_workspace_layouts
		WHERE tenant_id = $1 AND user_id = $2 AND profile_name = $3
	`

	var resp WorkspaceLayoutResponse
	var rawData []byte
	err = h.db.QueryRowContext(r.Context(), query, tenantID, userID, profileName).Scan(
		&resp.ID,
		&resp.TenantID,
		&resp.UserID,
		&resp.ProfileName,
		&resp.SchemaVersion,
		&rawData,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(fmt.Sprintf(`{"error":"layout profile not found for user: %s"}`, profileName)))
			return
		}
		logging.GetLogger().Sugar().Errorf("Failed to query workspace layout: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp.LayoutData = json.RawMessage(rawData)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// SaveLayout persists or updates a user's workspace layout profile
func (h *WorkspaceLayoutHandler) SaveLayout(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := h.resolveUserAndTenant(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Enforce 1MB payload cap
	r.Body = http.MaxBytesReader(w, r.Body, MaxWorkspaceLayoutPayloadBytes)

	var req SaveWorkspaceLayoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, `{"error":"payload too large (exceeds 1MB cap)"}`, http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, `{"error":"invalid JSON request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.LayoutData) == 0 || !json.Valid(req.LayoutData) {
		http.Error(w, `{"error":"layout_data must be a valid non-empty JSON object"}`, http.StatusBadRequest)
		return
	}

	if req.ProfileName == "" {
		req.ProfileName = DefaultLayoutProfileName
	}
	if req.SchemaVersion == "" {
		req.SchemaVersion = DefaultLayoutSchemaVersion
	}

	// Schema version validation: must be "v1" or start with "v"
	if !strings.HasPrefix(req.SchemaVersion, "v") {
		http.Error(w, `{"error":"unsupported schema_version"}`, http.StatusBadRequest)
		return
	}

	upsertQuery := `
		INSERT INTO public.user_workspace_layouts (
			tenant_id, user_id, profile_name, schema_version, layout_data, updated_at
		) VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (tenant_id, user_id, profile_name) DO UPDATE SET
			schema_version = EXCLUDED.schema_version,
			layout_data = EXCLUDED.layout_data,
			updated_at = NOW()
		RETURNING id, tenant_id, user_id, profile_name, schema_version, layout_data, created_at, updated_at
	`

	var resp WorkspaceLayoutResponse
	var rawData []byte
	err = h.db.QueryRowContext(
		r.Context(),
		upsertQuery,
		tenantID,
		userID,
		req.ProfileName,
		req.SchemaVersion,
		req.LayoutData,
	).Scan(
		&resp.ID,
		&resp.TenantID,
		&resp.UserID,
		&resp.ProfileName,
		&resp.SchemaVersion,
		&rawData,
		&resp.CreatedAt,
		&resp.UpdatedAt,
	)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to upsert workspace layout: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}

	resp.LayoutData = json.RawMessage(rawData)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// ListProfiles lists all saved layout profile summaries for the authenticated user
func (h *WorkspaceLayoutHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	userID, tenantID, err := h.resolveUserAndTenant(r)
	if err != nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	query := `
		SELECT id, profile_name, schema_version, updated_at
		FROM public.user_workspace_layouts
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY updated_at DESC
	`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, userID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to list workspace layout profiles: %v", err)
		http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	summaries := make([]WorkspaceLayoutProfileSummary, 0)
	for rows.Next() {
		var s WorkspaceLayoutProfileSummary
		if err := rows.Scan(&s.ID, &s.ProfileName, &s.SchemaVersion, &s.UpdatedAt); err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to scan layout profile summary: %v", err)
			continue
		}
		summaries = append(summaries, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summaries)
}
