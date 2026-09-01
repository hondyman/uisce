package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
)

type ReportShareHandler struct {
	DB *sql.DB
}

func NewReportShareHandler(db *sql.DB) *ReportShareHandler {
	return &ReportShareHandler{DB: db}
}

func (h *ReportShareHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/reports/{reportId}/shares", func(r chi.Router) {
		r.Get("/", h.ListShares)
		r.Post("/", h.AddShare)
		r.Delete("/{userId}", h.RemoveShare)
		r.Post("/stop-all", h.StopAllShares)
		r.Patch("/{userId}", h.UpdateShare)
	})
	r.Get("/api/v1/users/shareable", h.ListShareableUsers)
	r.Post("/api/v1/reports/{reportId}/clone", h.CloneReport)
}

// ============================================================================
// Shared types
// ============================================================================

type ShareResponse struct {
	ID              string     `json:"id"`
	ReportID       string     `json:"report_id"`
	SharedBy       string     `json:"shared_by"`
	RecipientID    string     `json:"recipient_id"`
	RecipientName   string     `json:"recipient_name"`
	RecipientEmail  string     `json:"recipient_email"`
	RecipientRole  string     `json:"recipient_role"`
	RecipientOrg   string     `json:"recipient_organization"`
	AccessPath     string     `json:"access_path"` // direct | entitlement
	Permission     string     `json:"permission"`
	IsActive       bool       `json:"is_active"`
	IsSuspended    bool       `json:"is_suspended"`
	AllowExport    bool       `json:"allow_export"`
	AllowPrint     bool       `json:"allow_print"`
	Watermark      bool       `json:"watermark"`
	CreatedAt      time.Time  `json:"created_at"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	SuspendedAt    *time.Time `json:"suspended_at,omitempty"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
}

type ShareableUser struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	Org         string `json:"organization"`
	AccessPath  string `json:"access_path"` // direct | entitlement
	IsActive    bool   `json:"is_active"`
	TenantID    string `json:"tenant_id,omitempty"`
}

// ============================================================================
// ListShares  GET /api/v1/reports/{reportId}/shares
// ============================================================================

func (h *ReportShareHandler) ListShares(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Determine caller: must be owner or have a share
	callerID := claims.UserID
	callerIsOwner, err := h.isReportOwner(r.Context(), reportID, callerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var rows *sql.Rows
	if callerIsOwner {
		rows, err = h.DB.QueryContext(r.Context(), listSharesOwnerQ, reportID)
	} else {
		rows, err = h.DB.QueryContext(r.Context(), listSharesCollaboratorQ, reportID, callerID)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var shares []ShareResponse
	for rows.Next() {
		var s ShareResponse
		var suspendedAt sql.NullTime
		var lastLogin sql.NullTime
		err := rows.Scan(
			&s.ID, &s.ReportID, &s.SharedBy, &s.RecipientID,
			&s.RecipientName, &s.RecipientEmail, &s.RecipientRole, &s.RecipientOrg,
			&s.AccessPath, &s.Permission, &s.IsActive, &s.IsSuspended,
			&s.AllowExport, &s.AllowPrint, &s.Watermark,
			&s.CreatedAt, &s.ExpiresAt, &s.SuspendedAt, &s.LastLogin,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		s.IsSuspended = suspendedAt.Valid
		if lastLogin.Valid {
			s.LastLogin = &lastLogin.Time
		}
		shares = append(shares, s)
	}
	if shares == nil {
		shares = []ShareResponse{}
	}
	json.NewEncoder(w).Encode(shares)
}

const listSharesOwnerQ = `
SELECT
    rs.id, rs.report_id, rs.shared_by, rs.recipient_id,
    COALESCE(u.name, u.email)             AS recipient_name,
    u.email                                AS recipient_email,
    COALESCE(u.role, '')                  AS recipient_role,
    COALESCE(u.organization, '')           AS recipient_org,
    CASE WHEN ut.user_id IS NOT NULL THEN 'direct'
         ELSE 'entitlement' END           AS access_path,
    rs.permission, u.is_active,
    rs.suspended_at IS NOT NULL           AS is_suspended,
    rs.allow_export, rs.allow_print, rs.watermark,
    rs.created_at, rs.expires_at, rs.suspended_at,
    u.last_login_time
FROM public.report_shares rs
JOIN public.app_user u ON u.id = rs.recipient_id
LEFT JOIN public.user_tenant ut ON ut.user_id = rs.recipient_id AND ut.tenant_id = rs.tenant_id
WHERE rs.report_id = $1 AND rs.share_type = 'direct'
ORDER BY rs.created_at DESC
`

const listSharesCollaboratorQ = `
SELECT
    rs.id, rs.report_id, rs.shared_by, rs.recipient_id,
    COALESCE(u.name, u.email)             AS recipient_name,
    u.email                                AS recipient_email,
    COALESCE(u.role, '')                  AS recipient_role,
    COALESCE(u.organization, '')           AS recipient_org,
    CASE WHEN ut.user_id IS NOT NULL THEN 'direct'
         ELSE 'entitlement' END           AS access_path,
    rs.permission, u.is_active,
    rs.suspended_at IS NOT NULL           AS is_suspended,
    rs.allow_export, rs.allow_print, rs.watermark,
    rs.created_at, rs.expires_at, rs.suspended_at,
    u.last_login_time
FROM public.report_shares rs
JOIN public.app_user u ON u.id = rs.recipient_id
LEFT JOIN public.user_tenant ut ON ut.user_id = rs.recipient_id AND ut.tenant_id = rs.tenant_id
WHERE rs.report_id = $1 AND rs.recipient_id = $2 AND rs.share_type = 'direct'
`

// ============================================================================
// AddShare  POST /api/v1/reports/{reportId}/shares
// ============================================================================

type AddShareRequest struct {
	RecipientID string  `json:"recipient_id"`
	Permission string  `json:"permission"`
	Note       string  `json:"note,omitempty"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
}

func (h *ReportShareHandler) AddShare(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	callerID := claims.UserID

	var req AddShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.RecipientID == "" {
		http.Error(w, `{"error":"recipient_id is required"}`, http.StatusBadRequest)
		return
	}

	// Permission must be 'view' (edit is not allowed per spec)
	perm := strings.ToLower(strings.TrimSpace(req.Permission))
	if perm == "" {
		perm = "view"
	}
	if perm == "edit" || perm == "admin" {
		http.Error(w, `{"error":"permission \'edit\' is not allowed for shared reports. Recipients may only view. Use Clone to create an editable copy."}`, http.StatusBadRequest)
		return
	}
	if perm != "view" && perm != "comment" {
		http.Error(w, `{"error":"permission must be \'view\' or \'comment\'"}`, http.StatusBadRequest)
		return
	}

	// Verify caller is owner
	isOwner, err := h.isReportOwner(r.Context(), reportID, callerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isOwner {
		http.Error(w, `{"error":"only the report owner can share it"}`, http.StatusForbidden)
		return
	}

	// Verify recipient exists and belongs to the same tenant
	tenantID, err := h.getReportTenant(r.Context(), reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	valid, err := h.isValidShareableUser(r.Context(), req.RecipientID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !valid {
		http.Error(w, `{"error":"recipient is not a valid shareable user for this tenant"}`, http.StatusBadRequest)
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}

	var shareID string
	err = h.DB.QueryRowContext(r.Context(), `
		INSERT INTO public.report_shares
			(tenant_id, report_id, shared_by, recipient_id, recipient_type, permission, expires_at, allow_export, allow_print, watermark)
		VALUES ($1, $2, $3, $4, 'user', $5, $6, true, true, false)
		ON CONFLICT (report_id, recipient_id) DO UPDATE
			SET permission = EXCLUDED.permission,
				expires_at = EXCLUDED.expires_at,
				suspended_at = NULL
		RETURNING id`,
		tenantID, reportID, callerID, req.RecipientID, perm, expiresAt,
	).Scan(&shareID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Notification: insert email notification into outbox
	notifPayload, _ := json.Marshal(map[string]interface{}{
		"report_id":    reportID,
		"shared_by":    callerID,
		"recipient_id": req.RecipientID,
		"report_name":  "",
		"action":       "report_shared",
	})
	h.DB.ExecContext(r.Context(), `
		INSERT INTO notification_outbox (aggregate_type, event_type, payload, tenant_id, user_id)
		VALUES ('email', 'report_shared', $1, $2, $3)
	`, notifPayload, tenantID, req.RecipientID)

	// Audit log
	h.audit(r.Context(), shareID, uuid.MustParse(tenantID), reportID, callerID, "share_created", map[string]interface{}{
		"recipient_id": req.RecipientID,
		"permission":   perm,
		"note":         req.Note,
	}, r.RemoteAddr)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": shareID})
}

// ============================================================================
// RemoveShare  DELETE /api/v1/reports/{reportId}/shares/{userId}
// ============================================================================

func (h *ReportShareHandler) RemoveShare(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	recipientID := chi.URLParam(r, "userId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	callerID := claims.UserID

	isOwner, err := h.isReportOwner(r.Context(), reportID, callerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isOwner {
		http.Error(w, `{"error":"only the report owner can revoke shares"}`, http.StatusForbidden)
		return
	}

	tenantID, err := h.getReportTenant(r.Context(), reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var shareID string
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id FROM public.report_shares
		WHERE report_id = $1 AND recipient_id = $2 AND share_type = 'direct'
	`, reportID, recipientID).Scan(&shareID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"share not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = h.DB.ExecContext(r.Context(), `
		DELETE FROM public.report_shares WHERE id = $1
	`, shareID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.audit(r.Context(), shareID, uuid.MustParse(tenantID), reportID, callerID, "share_revoked", map[string]interface{}{
		"recipient_id": recipientID,
	}, r.RemoteAddr)

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// StopAllShares  POST /api/v1/reports/{reportId}/shares/stop-all
// ============================================================================

func (h *ReportShareHandler) StopAllShares(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	callerID := claims.UserID

	isOwner, err := h.isReportOwner(r.Context(), reportID, callerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isOwner {
		http.Error(w, `{"error":"only the report owner can stop sharing"}`, http.StatusForbidden)
		return
	}

	tenantID, _ := h.getReportTenant(r.Context(), reportID)
	result, err := h.DB.ExecContext(r.Context(), `
		DELETE FROM public.report_shares
		WHERE report_id = $1 AND share_type = 'direct'
	`, reportID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = result

	if tenantID != "" {
		h.audit(r.Context(), "", uuid.MustParse(tenantID), reportID, callerID, "share_stopped_all", nil, r.RemoteAddr)
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// UpdateShare  PATCH /api/v1/reports/{reportId}/shares/{userId}  (suspend/resume)
// ============================================================================

type UpdateShareRequest struct {
	Suspend   *bool `json:"suspend,omitempty"`
	Watermark *bool `json:"watermark,omitempty"`
}

func (h *ReportShareHandler) UpdateShare(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	recipientID := chi.URLParam(r, "userId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	callerID := claims.UserID

	isOwner, err := h.isReportOwner(r.Context(), reportID, callerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !isOwner {
		http.Error(w, `{"error":"only the report owner can update shares"}`, http.StatusForbidden)
		return
	}

	var req UpdateShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tenantID, _ := h.getReportTenant(r.Context(), reportID)
	var shareID string
	err = h.DB.QueryRowContext(r.Context(), `
		SELECT id FROM public.report_shares
		WHERE report_id = $1 AND recipient_id = $2 AND share_type = 'direct'
	`, reportID, recipientID).Scan(&shareID)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"share not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if req.Suspend != nil {
		var suspendedAt interface{} = nil
		if *req.Suspend {
			suspendedAt = time.Now()
		}
		_, err = h.DB.ExecContext(r.Context(), `
			UPDATE public.report_shares SET suspended_at = $1 WHERE id = $2
		`, suspendedAt, shareID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		action := "share_suspended"
		if !*req.Suspend {
			action = "share_resumed"
		}
		h.audit(r.Context(), shareID, uuid.MustParse(tenantID), reportID, callerID, action, nil, r.RemoteAddr)
	}

	if req.Watermark != nil {
		_, err = h.DB.ExecContext(r.Context(), `
			UPDATE public.report_shares SET watermark = $1 WHERE id = $2
		`, *req.Watermark, shareID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// ListShareableUsers  GET /api/v1/users/shareable?tenant_id=<x>
// ============================================================================

func (h *ReportShareHandler) ListShareableUsers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, `{"error":"tenant_id query parameter is required"}`, http.StatusBadRequest)
		return
	}

	rows, err := h.DB.QueryContext(r.Context(), listShareableUsersQ, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var users []ShareableUser
	for rows.Next() {
		var u ShareableUser
		var tenantID2 sql.NullString
		var lastLogin sql.NullTime
		err := rows.Scan(
			&u.ID, &u.Name, &u.Email, &u.Role, &u.Org, &u.AccessPath, &u.IsActive, &tenantID2,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tenantID2.Valid {
			u.TenantID = tenantID2.String
		}
		_ = lastLogin
		users = append(users, u)
	}
	if users == nil {
		users = []ShareableUser{}
	}
	json.NewEncoder(w).Encode(users)
}

const listShareableUsersQ = `
SELECT
    u.id, COALESCE(u.name, u.email) AS name, u.email, COALESCE(u.role, '') AS role,
    COALESCE(u.organization, '')    AS org,
    CASE WHEN ut.user_id IS NOT NULL THEN 'direct'
         ELSE 'entitlement' END     AS access_path,
    u.is_active,
    ut.tenant_id
FROM public.app_user u
LEFT JOIN public.user_tenant     ut ON ut.user_id = u.id AND ut.tenant_id = $1
LEFT JOIN public.user_tenant_access uta ON uta.user_id = u.id AND uta.tenant_id = $1
WHERE (ut.user_id IS NOT NULL OR uta.user_id IS NOT NULL)
  AND u.is_active = true
ORDER BY u.email
`

// ============================================================================
// CloneReport  POST /api/v1/reports/{reportId}/clone
// Creates a private copy of the report owned by the calling user.
// ============================================================================

func (h *ReportShareHandler) CloneReport(w http.ResponseWriter, r *http.Request) {
	reportID := chi.URLParam(r, "reportId")
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	callerID := claims.UserID
	tenantID := claims.TenantID
	if tenantID == "" {
		// Fall back to tenant from X-Tenant-Id header
		tenantID = r.Header.Get("X-Tenant-Id")
	}
	if tenantID == "" {
		http.Error(w, `{"error":"tenant context required"}`, http.StatusBadRequest)
		return
	}

	// Fetch source report
	var (
		srcName, srcDisplayName string
		srcDefinition           []byte
		srcCreatedBy            string
	)
	var idDiscard uuid.UUID
	var isCoreDiscard bool
	err := h.DB.QueryRowContext(r.Context(), `
		SELECT id, COALESCE(report_name, display_name), display_name,
		       definition, is_core, COALESCE(created_by, '')
		FROM vend.report_definitions
		WHERE id = $1 AND tenant_id = $2
	`, reportID, tenantID).Scan(&idDiscard, &srcName, &srcDisplayName, &srcDefinition, &isCoreDiscard, &srcCreatedBy)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"report not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check: caller must have a share on this report OR be the owner
	hasShare := false
	if callerID != srcCreatedBy {
		err = h.DB.QueryRowContext(r.Context(), `
			SELECT true FROM public.report_shares
			WHERE report_id = $1 AND recipient_id = $2 AND share_type = 'direct'
			LIMIT 1
		`, reportID, callerID).Scan(&hasShare)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if !hasShare && callerID != srcCreatedBy {
		http.Error(w, `{"error":"you do not have permission to clone this report"}`, http.StatusForbidden)
		return
	}

	// Insert cloned report
	var newID uuid.UUID
	var newName string
	if srcName == "" {
		newName = srcDisplayName + " (Copy)"
	} else {
		newName = srcName + " (Copy)"
	}
	err = h.DB.QueryRowContext(r.Context(), `
		INSERT INTO vend.report_definitions
			(tenant_id, report_key, display_name, description, category, definition,
			 report_type, is_core, created_by, status)
		VALUES ($1, 'clone-' || gen_random_uuid()::text, $2,
			'Cloned from: ' || COALESCE($3, $4), 'My Reports', $5,
			'report', false, $6, 'draft')
		RETURNING id, display_name
	`, tenantID, newName, srcName, srcDisplayName, srcDefinition, callerID).Scan(&newID, &newName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Audit
	h.audit(r.Context(), "", uuid.MustParse(tenantID), newID.String(), callerID, "report_cloned", map[string]interface{}{
		"source_report_id": reportID,
		"source_name":      srcName,
	}, r.RemoteAddr)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   newID,
		"name": newName,
	})
}

// ============================================================================
// Helpers
// ============================================================================

func (h *ReportShareHandler) isReportOwner(ctx context.Context, reportID, userID string) (bool, error) {
	var createdBy string
	err := h.DB.QueryRowContext(ctx, `
		SELECT COALESCE(created_by, '') FROM vend.report_definitions WHERE id = $1
	`, reportID).Scan(&createdBy)
	if err == sql.ErrNoRows {
		return false, fmt.Errorf("report not found")
	}
	if err != nil {
		return false, err
	}
	return createdBy == userID, nil
}

func (h *ReportShareHandler) getReportTenant(ctx context.Context, reportID string) (string, error) {
	var tenantID string
	err := h.DB.QueryRowContext(ctx, `
		SELECT tenant_id::text FROM vend.report_definitions WHERE id = $1
	`, reportID).Scan(&tenantID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("report not found")
	}
	return tenantID, err
}

func (h *ReportShareHandler) isValidShareableUser(ctx context.Context, userID, tenantID string) (bool, error) {
	var exists bool
	err := h.DB.QueryRowContext(ctx, `
		SELECT true
		FROM public.app_user u
		LEFT JOIN public.user_tenant     ut ON ut.user_id = u.id AND ut.tenant_id = $2
		LEFT JOIN public.user_tenant_access uta ON uta.user_id = u.id AND uta.tenant_id = $2
		WHERE u.id = $1 AND (ut.user_id IS NOT NULL OR uta.user_id IS NOT NULL)
		LIMIT 1
	`, userID, tenantID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return exists != false, err
}

func (h *ReportShareHandler) audit(ctx context.Context, shareID string, tenantID uuid.UUID, reportID string, actorID, action string, details map[string]interface{}, ip string) {
	detailsJSON, _ := json.Marshal(details)
	remoteIP := ""
	if ip != "" {
		remoteIP = ip
	}
	h.DB.ExecContext(ctx, `
		INSERT INTO public.report_share_audit_log
			(share_id, tenant_id, report_id, actor_id, action, details, ip_address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, shareID, tenantID, reportID, actorID, action, detailsJSON, remoteIP)
}
