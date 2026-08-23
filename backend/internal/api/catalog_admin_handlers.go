package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/catalog"
	"github.com/hondyman/uisce/backend/internal/tenant"
)

func RegisterCatalogAdminRoutes(r chi.Router, db *sql.DB, tenantMgr *tenant.TenantManager) {
	loader := catalog.NewSubtypeRegistryLoader(5 * time.Minute)
	boBuilder := catalog.NewSubtypeBOBuilder(loader)
	scanner := catalog.NewSTIColumnScanner()
	linker := catalog.NewSubtypeSemanticLinker()

	r.Post("/api/catalog/admin/sync-subtypes", func(w http.ResponseWriter, r *http.Request) {
		tenantIDStr := r.Header.Get("X-Tenant-ID")
		if tenantIDStr == "" {
			http.Error(w, "X-Tenant-ID header is required", http.StatusBadRequest)
			return
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			http.Error(w, "Invalid X-Tenant-ID UUID format", http.StatusBadRequest)
			return
		}

		tenantConn, err := tenantMgr.GetTenantConnection(r.Context(), tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tenantConn.Close()

		if err := boBuilder.BuildForTenant(r.Context(), db, tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := scanner.ScanAndEmit(r.Context(), tenantConn, db, tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := linker.LinkTerms(r.Context(), db, tenantID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","message":"STI subtypes synced and linked successfully"}`))
	})
}
