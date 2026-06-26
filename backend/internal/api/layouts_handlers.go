package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// registerLayoutRoutes registers simple CRUD endpoints for saved layouts.
func (s *Server) registerLayoutRoutes(r chi.Router) {
	// List layouts for tenant/datasource
	r.Get("/layouts", func(w http.ResponseWriter, r *http.Request) {
		tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
		datasourceID := strings.TrimSpace(r.URL.Query().Get("datasource_id"))

		if tenantID == "" || datasourceID == "" {
			http.Error(w, "tenant_id and datasource_id query params are required", http.StatusBadRequest)
			return
		}

		rows, err := s.DB.QueryContext(r.Context(), `SELECT id, name, view, created_at, updated_at FROM public.views WHERE tenant_id = $1 AND tenant_datasource_id = $2 ORDER BY updated_at DESC`, tenantID, datasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to query layouts: %v", err)
			http.Error(w, fmt.Sprintf("failed to query layouts: %v", err), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type item struct {
			ID        string          `json:"id"`
			Name      string          `json:"name"`
			Layout    json.RawMessage `json:"layout"`
			CreatedAt time.Time       `json:"created_at"`
			UpdatedAt time.Time       `json:"updated_at"`
		}
		var items []item
		for rows.Next() {
			var id, name string
			var viewRaw []byte
			var createdAt, updatedAt time.Time
			if err := rows.Scan(&id, &name, &viewRaw, &createdAt, &updatedAt); err != nil {
				continue
			}
			items = append(items, item{ID: id, Name: name, Layout: json.RawMessage(viewRaw), CreatedAt: createdAt, UpdatedAt: updatedAt})
		}
		respond(w, r, map[string]any{"layouts": items}, nil)
	})

	// Get single layout by id
	r.Get("/layouts/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
			return
		}

		var viewRaw []byte
		var name string
		err := s.DB.QueryRowContext(r.Context(), `SELECT name, view FROM public.views WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3`, id, tenantID, datasourceID).Scan(&name, &viewRaw)
		if err != nil {
			if err == sql.ErrNoRows {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			logging.GetLogger().Sugar().Errorf("failed to fetch layout: %v", err)
			http.Error(w, fmt.Sprintf("failed to fetch layout: %v", err), http.StatusInternalServerError)
			return
		}
		respond(w, r, map[string]any{"id": id, "name": name, "layout": json.RawMessage(viewRaw)}, nil)
	})

	// Save (create or update) layout
	r.Post("/layouts", func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
			return
		}

		var body struct {
			ID     string          `json:"id,omitempty"`
			Name   string          `json:"name"`
			Layout json.RawMessage `json:"layout"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(body.Name) == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		now := time.Now()
		if body.ID == "" {
			// insert
			id := uuid.New().String()
			_, err := s.DB.ExecContext(r.Context(), `INSERT INTO public.views (id, tenant_id, tenant_datasource_id, name, view, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, id, tenantID, datasourceID, body.Name, body.Layout, now, now)
			if err != nil {
				logging.GetLogger().Sugar().Errorf("failed to insert layout: %v", err)
				http.Error(w, fmt.Sprintf("failed to insert layout: %v", err), http.StatusInternalServerError)
				return
			}
			respond(w, r, map[string]any{"id": id, "name": body.Name, "created_at": now, "updated_at": now}, nil)
			return
		}
		// update existing, ensure tenant scoping
		_, err := s.DB.ExecContext(r.Context(), `UPDATE public.views SET name = $1, view = $2, updated_at = $3 WHERE id = $4 AND tenant_id = $5 AND tenant_datasource_id = $6`, body.Name, body.Layout, now, body.ID, tenantID, datasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to update layout: %v", err)
			http.Error(w, fmt.Sprintf("failed to update layout: %v", err), http.StatusInternalServerError)
			return
		}
		respond(w, r, map[string]any{"id": body.ID, "name": body.Name, "updated_at": now}, nil)
	})

	// Delete layout by id
	r.Delete("/layouts/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "X-Tenant-ID and X-Tenant-Datasource-ID headers are required", http.StatusBadRequest)
			return
		}
		_, err := s.DB.ExecContext(r.Context(), `DELETE FROM public.views WHERE id = $1 AND tenant_id = $2 AND tenant_datasource_id = $3`, id, tenantID, datasourceID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("failed to delete layout: %v", err)
			http.Error(w, fmt.Sprintf("failed to delete layout: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
}
