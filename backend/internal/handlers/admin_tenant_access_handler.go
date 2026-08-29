package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/lib/pq"
)

type AdminTenantAccessHandler struct {
	db *sql.DB
}

func NewAdminTenantAccessHandler(db *sql.DB) *AdminTenantAccessHandler {
	return &AdminTenantAccessHandler{db: db}
}

func (h *AdminTenantAccessHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/tenant-access", func(r chi.Router) {
		r.Get("/", h.ListMappings)
		r.Post("/", h.CreateMapping)
		r.Put("/{userId}/{tenantId}", h.UpdateMapping)
		r.Delete("/{userId}/{tenantId}", h.DeleteMapping)
	})
}

type TenantAccessMapping struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	TenantID        string `json:"tenant_id"`
	AccessRole      string `json:"access_role"`
	Email           string `json:"email"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	TenantName      string `json:"tenant_name,omitempty"`
	TenantIsActive  bool   `json:"tenant_is_active,omitempty"`
	TenantIsDeleted bool   `json:"tenant_is_deleted,omitempty"`
}

func (h *AdminTenantAccessHandler) ListMappings(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	limit, offset := PaginateWithMax(r, 1000)

	query := `
		SELECT
			ut.user_id,
			ut.tenant_id,
			COALESCE(ut.access_role, ''),
			COALESCE(us.email, u.email, ''),
			ut.created_at,
			ut.updated_at,
			COALESCE(t.name, ''),
			COALESCE(t.is_active, false),
			COALESCE(t.is_deleted, false)
		FROM public.user_tenant ut
		JOIN public.app_user u ON u.id = ut.user_id
		LEFT JOIN public.users us ON us.id::text = ut.user_id
		JOIN public.tenants t ON t.id = ut.tenant_id
		ORDER BY COALESCE(us.email, u.email, ''), t.name
		LIMIT $1 OFFSET $2
	`

	rows, err := h.db.QueryContext(r.Context(), query, limit, offset)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	mappings := []TenantAccessMapping{}
	for rows.Next() {
		var m TenantAccessMapping
		if err := rows.Scan(
			&m.UserID, &m.TenantID, &m.AccessRole, &m.Email,
			&m.CreatedAt, &m.UpdatedAt, &m.TenantName,
			&m.TenantIsActive, &m.TenantIsDeleted,
		); err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
		m.ID = m.UserID
		mappings = append(mappings, m)
	}

	countQuery := `SELECT COUNT(*) FROM public.user_tenant`
	var total int
	h.db.QueryRowContext(r.Context(), countQuery).Scan(&total)

	SendJSON(w, http.StatusOK, map[string]interface{}{
		"data":   mappings,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ensureUserExists makes sure app_user has a row for userID before a
// user_tenant mapping is created against it. Note: the "users" table is
// actually a read-only view over app_user (see migrations), so there is
// only one underlying table to upsert into — not two.
func (h *AdminTenantAccessHandler) ensureUserExists(userID string) error {
	if _, err := h.db.ExecContext(context.Background(), `
		INSERT INTO public.app_user (id, email, display_name, created_at, is_active)
		VALUES ($1, $1, '', NOW(), true)
		ON CONFLICT (id) DO NOTHING
	`, userID); err != nil {
		return fmt.Errorf("failed to ensure app_user exists: %w", err)
	}
	return nil
}

func (h *AdminTenantAccessHandler) CreateMapping(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		UserID   string `json:"user_id"`
		TenantID string `json:"tenant_id"`
		AccessRole string `json:"access_role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.UserID) == "" {
		SendError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" {
		SendError(w, http.StatusBadRequest, "tenant_id is required")
		return
	}
	if strings.TrimSpace(req.AccessRole) == "" {
		SendError(w, http.StatusBadRequest, "access_role is required")
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	actor, _ := security.AuthInfoFromContext(r.Context())
	createdBy := actor.UserID
	if createdBy == "" {
		createdBy = "system"
	}

	if err := h.ensureUserExists(req.UserID); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	query := `
		INSERT INTO public.user_tenant (user_id, tenant_id, access_role, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (user_id, tenant_id) DO UPDATE SET
			access_role = EXCLUDED.access_role,
			updated_at = NOW()
		RETURNING user_id, tenant_id, access_role, created_at, updated_at
	`

	var mapping TenantAccessMapping
	err = h.db.QueryRowContext(r.Context(), query,
		req.UserID, tenantUUID, strings.TrimSpace(req.AccessRole),
	).Scan(
		&mapping.UserID, &mapping.TenantID, &mapping.AccessRole,
		&mapping.CreatedAt, &mapping.UpdatedAt,
	)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	mapping.ID = mapping.UserID
	SendJSON(w, http.StatusCreated, map[string]interface{}{"data": mapping})
}

func (h *AdminTenantAccessHandler) UpdateMapping(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	userID := chi.URLParam(r, "userId")
	tenantID := chi.URLParam(r, "tenantId")

	if userID == "" || tenantID == "" {
		SendError(w, http.StatusBadRequest, "user_id and tenant_id are required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	var req struct {
		AccessRole string `json:"access_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if strings.TrimSpace(req.AccessRole) == "" {
		SendError(w, http.StatusBadRequest, "access_role is required")
		return
	}

	query := `
		UPDATE public.user_tenant
		SET access_role = $1, updated_at = NOW()
		WHERE user_id = $2 AND tenant_id = $3
		RETURNING user_id, tenant_id, access_role, created_at, updated_at
	`

	var mapping TenantAccessMapping
	err = h.db.QueryRowContext(r.Context(), query,
		strings.TrimSpace(req.AccessRole), userID, tenantUUID,
	).Scan(
		&mapping.UserID, &mapping.TenantID, &mapping.AccessRole,
		&mapping.CreatedAt, &mapping.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			SendError(w, http.StatusNotFound, "mapping not found")
			return
		}
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	mapping.ID = mapping.UserID
	SendJSON(w, http.StatusOK, map[string]interface{}{"data": mapping})
}

func (h *AdminTenantAccessHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	if err := h.requireAdminAuth(r); err != nil {
		SendError(w, http.StatusUnauthorized, err.Error())
		return
	}

	userID := chi.URLParam(r, "userId")
	tenantID := chi.URLParam(r, "tenantId")

	if userID == "" || tenantID == "" {
		SendError(w, http.StatusBadRequest, "user_id and tenant_id are required")
		return
	}

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		SendError(w, http.StatusBadRequest, "invalid tenant_id format")
		return
	}

	query := `DELETE FROM public.user_tenant WHERE user_id = $1 AND tenant_id = $2`

	result, err := h.db.ExecContext(r.Context(), query, userID, tenantUUID)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23503" {
				SendError(w, http.StatusBadRequest, "cannot delete: foreign key constraint violation")
				return
			}
		}
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		SendError(w, http.StatusNotFound, "mapping not found")
		return
	}

	SendJSON(w, http.StatusOK, map[string]interface{}{"deleted": true})
}

func (h *AdminTenantAccessHandler) requireAdminAuth(r *http.Request) error {
	actor, ok := security.AuthInfoFromContext(r.Context())
	if !ok || strings.TrimSpace(actor.UserID) == "" {
		return errors.New("unauthorized")
	}

	if actor.IsGlobalAdmin || hasAdminRole(actor.Roles) {
		return nil
	}

	return errors.New("forbidden")
}
