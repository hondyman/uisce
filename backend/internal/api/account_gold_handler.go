package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

// AccountGoldHandler exposes Account gold-copy and override APIs on the data plane.
type AccountGoldHandler struct {
	DataPool  *sqlx.DB
	Publisher *goldcopy.Publisher
}

func NewAccountGoldHandler(data *sqlx.DB) *AccountGoldHandler {
	h := &AccountGoldHandler{DataPool: data}
	if brokers := strings.TrimSpace(os.Getenv("KAFKA_BROKERS")); brokers != "" {
		h.Publisher = goldcopy.NewPublisher(brokers)
	}
	return h
}

func (h *AccountGoldHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/mdm/account-gold", func(r chi.Router) {
		r.Post("/build", h.Build)
		r.Post("/overrides", h.CreateOverride)
		r.Post("/overrides/{id}/approve", h.ApproveOverride)
		r.Post("/overrides/{id}/reject", h.RejectOverride)
		r.Get("/overrides", h.ListOverrides)
	})
}

func (h *AccountGoldHandler) tenantID(r *http.Request) (uuid.UUID, error) {
	tenant, err := security.ResolveTenantForRequest(r)
	if err != nil {
		return uuid.Nil, err
	}
	return uuid.Parse(tenant)
}

func (h *AccountGoldHandler) Build(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeAcctGoldJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		AccountCd     string `json:"account_cd"`
		ChangeReason  string `json:"change_reason"`
		Publish       bool   `json:"publish"`
		CorrelationID string `json:"correlation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.AccountCd) == "" {
		writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": "account_cd required"})
		return
	}
	eng := goldcopy.NewAccountEngine(h.DataPool)
	rec, err := eng.BuildAndPersist(r.Context(), tenantID, body.AccountCd, "")
	if err != nil {
		writeAcctGoldJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if body.Publish && h.Publisher != nil {
		reason := body.ChangeReason
		if reason == "" {
			reason = "account gold publish"
		}
		if err := h.Publisher.PublishAccountMasterGoldCopy(r.Context(), rec, "updated", reason, "api", body.CorrelationID); err != nil {
			writeAcctGoldJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("built but publish failed: %v", err)})
			return
		}
	}
	writeAcctGoldJSON(w, http.StatusOK, rec)
}

func (h *AccountGoldHandler) CreateOverride(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeAcctGoldJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	var body struct {
		AccountCd      string `json:"account_cd"`
		FieldCd        string `json:"field_cd"`
		OverrideValue  string `json:"override_value"`
		Reason         string `json:"reason"`
		SemanticTermID string `json:"semantic_term_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if body.AccountCd == "" || body.FieldCd == "" || body.Reason == "" {
		writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": "account_cd, field_cd, reason required"})
		return
	}
	var termID *uuid.UUID
	if body.SemanticTermID != "" {
		id, err := uuid.Parse(body.SemanticTermID)
		if err != nil {
			writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid semantic_term_id"})
			return
		}
		termID = &id
	}
	var id uuid.UUID
	err = h.withTenant(r, tenantID, func(tx *sqlx.Tx) error {
		return tx.GetContext(r.Context(), &id, `
			INSERT INTO mdm.account_field_override
				(tenant_id, account_cd, semantic_term_id, field_cd, override_value, reason, approval_status)
			VALUES ($1,$2,$3,$4,$5,$6,'pending')
			RETURNING id`,
			tenantID, body.AccountCd, termID, body.FieldCd, body.OverrideValue, body.Reason)
	})
	if err != nil {
		writeAcctGoldJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeAcctGoldJSON(w, http.StatusCreated, map[string]any{"id": id, "approval_status": "pending"})
}

func (h *AccountGoldHandler) ApproveOverride(w http.ResponseWriter, r *http.Request) {
	h.setOverrideStatus(w, r, "approved")
}

func (h *AccountGoldHandler) RejectOverride(w http.ResponseWriter, r *http.Request) {
	h.setOverrideStatus(w, r, "rejected")
}

func (h *AccountGoldHandler) setOverrideStatus(w http.ResponseWriter, r *http.Request, status string) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeAcctGoldJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	err = h.withTenant(r, tenantID, func(tx *sqlx.Tx) error {
		res, err := tx.ExecContext(r.Context(), `
			UPDATE mdm.account_field_override
			SET approval_status = $2, approved_at = now(), updated_at = now()
			WHERE id = $1 AND tenant_id = $3 AND is_active`, id, status, tenantID)
		if err != nil {
			return err
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return fmt.Errorf("override not found")
		}
		return nil
	})
	if err != nil {
		writeAcctGoldJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeAcctGoldJSON(w, http.StatusOK, map[string]string{"id": id.String(), "approval_status": status})
}

func (h *AccountGoldHandler) ListOverrides(w http.ResponseWriter, r *http.Request) {
	tenantID, err := h.tenantID(r)
	if err != nil {
		writeAcctGoldJSON(w, http.StatusUnauthorized, map[string]string{"error": err.Error()})
		return
	}
	accountCd := r.URL.Query().Get("account_cd")
	var out []goldcopy.AccountFieldOverride
	err = h.withTenant(r, tenantID, func(tx *sqlx.Tx) error {
		q := `
			SELECT id, tenant_id, account_cd, semantic_term_id, field_cd, override_value,
			       reason, approval_status, expires_at
			FROM mdm.account_field_override
			WHERE tenant_id = $1 AND is_active`
		args := []any{tenantID}
		if accountCd != "" {
			q += ` AND account_cd = $2`
			args = append(args, accountCd)
		}
		q += ` ORDER BY created_at DESC`
		return tx.SelectContext(r.Context(), &out, q, args...)
	})
	if err != nil {
		writeAcctGoldJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if out == nil {
		out = []goldcopy.AccountFieldOverride{}
	}
	writeAcctGoldJSON(w, http.StatusOK, map[string]any{"overrides": out})
}

func (h *AccountGoldHandler) withTenant(r *http.Request, tenantID uuid.UUID, fn func(*sqlx.Tx) error) error {
	tx, err := h.DataPool.BeginTxx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(r.Context(),
		`SELECT set_config('app.current_tenant', $1, true), set_config('uisce.current_tenant', $1, true)`,
		tenantID.String()); err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func writeAcctGoldJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
