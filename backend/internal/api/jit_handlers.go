package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/audit"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/backend/internal/services"
	coremodels "github.com/hondyman/semlayer/backend/models"
)

// GET /micro-bundles
func HandleListMicroBundles(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		bundles, err := services.ListMicroBundles(r.Context(), db)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(bundles)
	}
}

// POST /micro-bundles
func HandleCreateMicroBundle(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var b coremodels.MicroBundle
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := services.CreateMicroBundle(r.Context(), db, &b); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(b)
	}
}

// GET /micro-bundles/{id}
func HandleGetMicroBundle(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", 400)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}
		b, err := services.GetMicroBundle(r.Context(), db, id)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(b)
	}
}

// PUT /micro-bundles/{id}
func HandleUpdateMicroBundle(db *sql.DB, policySvc services.PolicyService, auditSvc *audit.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", 400)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}
		// capture existing bundle (for audit)
		var oldBundle coremodels.MicroBundle
		if ob, err := services.GetMicroBundle(r.Context(), db, id); err == nil {
			oldBundle = ob
		}

		var b coremodels.MicroBundle
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		// Enforce RBAC via policy service
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		resource := "micro_bundle"
		policies, _ := services.GetPoliciesForMicroBundle(r.Context(), db, resource)
		allowed, perr := policySvc.Can(user, "update", resource, policies)
		if perr != nil {
			http.Error(w, perr.Error(), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if err := services.UpdateMicroBundle(r.Context(), db, id, &b); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// audit
		if auditSvc != nil {
			var oldMap map[string]interface{}
			var newMap map[string]interface{}
			if ob, err := json.Marshal(oldBundle); err == nil {
				_ = json.Unmarshal(ob, &oldMap)
			}
			if nb, err := json.Marshal(b); err == nil {
				_ = json.Unmarshal(nb, &newMap)
			}
			_ = auditSvc.LogDataModification(r.Context(), user.ID, "", "", "micro_bundle", id.String(), "update", oldMap, newMap)
		}

		json.NewEncoder(w).Encode(b)
	}
}

// DELETE /micro-bundles/{id}
func HandleDeleteMicroBundle(db *sql.DB, policySvc services.PolicyService, auditSvc *audit.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			http.Error(w, "missing id", 400)
			return
		}
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}
		// Enforce RBAC via policy service
		user, ok := auth.GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "unauthenticated", http.StatusUnauthorized)
			return
		}

		resource := "micro_bundle"
		policies, _ := services.GetPoliciesForMicroBundle(r.Context(), db, resource)
		allowed, perr := policySvc.Can(user, "delete", resource, policies)
		if perr != nil {
			http.Error(w, perr.Error(), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		// fetch existing for audit
		var oldBundle coremodels.MicroBundle
		if ob, err := services.GetMicroBundle(r.Context(), db, id); err == nil {
			oldBundle = ob
		}

		if err := services.DeleteMicroBundle(r.Context(), db, id); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if auditSvc != nil {
			var oldMap map[string]interface{}
			if ob, err := json.Marshal(oldBundle); err == nil {
				_ = json.Unmarshal(ob, &oldMap)
			}
			_ = auditSvc.LogDataModification(r.Context(), user.ID, "", "", "micro_bundle", id.String(), "delete", oldMap, nil)
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// POST /jit-grants
func HandleCreateJITGrant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var g coremodels.JITAddonGrant
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if err := services.CreateJITAddonGrant(r.Context(), db, &g); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(g)
	}
}

// GET /jit-grants?user_id=...
func HandleListJITGrants(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		grants, err := services.ListJITAddonGrants(r.Context(), db, userID)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		json.NewEncoder(w).Encode(grants)
	}
}

// POST /jit-grants/:id/renew
func HandleRenewJITGrant(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid id", 400)
			return
		}
		var req struct{ ExpiresAt string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		expiry, err := time.Parse(time.RFC3339, req.ExpiresAt)
		if err != nil {
			http.Error(w, "invalid expiry", 400)
			return
		}
		if err := services.RenewJITAddonGrant(r.Context(), db, id, expiry); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
