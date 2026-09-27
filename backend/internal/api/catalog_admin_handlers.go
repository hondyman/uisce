package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/catalog"
	"github.com/hondyman/uisce/backend/internal/metadata"
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

	r.Post("/api/catalog/admin/sync-crims", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		goldCopyTenantID, err := getGoldCopyTenantID(ctx, db)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to resolve gold copy tenant: %v", err), http.StatusInternalServerError)
			return
		}

		datasource, err := getCrimsDatasourceForTenant(ctx, db, goldCopyTenantID)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to find CRIMS datasource: %v", err), http.StatusInternalServerError)
			return
		}

		crimsDB, err := metadata.ConnectToDatabaseFromDetails(ctx, datasource.ConnectionDetails)
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to connect to CRIMS: %v", err), http.StatusInternalServerError)
			return
		}
		defer crimsDB.Close()

		colScanner := catalog.NewColumnReferenceScanner(crimsDB, []string{"mdm", "ref", "orm"}, "crims")
		result, err := colScanner.ScanAndEmit(ctx, db, goldCopyTenantID, datasource.ID)
		if err != nil {
			http.Error(w, fmt.Sprintf("scan failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":               "success",
			"message":              "CRIMS column references synced successfully",
			"tables_scanned":       result.TablesScanned,
			"columns_scanned":     result.ColumnsScanned,
			"edges_created":        result.EdgesCreated,
			"fks_found":            result.FKsFound,
			"skipped_non_whitelisted": result.SkippedNonWhitelisted,
		})
	})
}

type crimsDatasource struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	Name              string
	SourceSystem      string
	ConnectionDetails string
	IsGoldCopy        bool
}

func getGoldCopyTenantID(ctx context.Context, db *sql.DB) (uuid.UUID, error) {
	var id uuid.UUID
	err := db.QueryRowContext(ctx, `SELECT id FROM (SELECT public.uisce_gold_copy_tenant_id() AS id) g WHERE id IS NOT NULL`).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("uisce_gold_copy_tenant_id: %w", err)
	}
	return id, nil
}

func getCrimsDatasourceForTenant(ctx context.Context, db *sql.DB, tenantID uuid.UUID) (*crimsDatasource, error) {
	query := `
		SELECT
			tpd.id,
			ti.tenant_id,
			tpd.source_name AS name,
			ad.datasource_code AS source_system,
			CASE
				WHEN c.id IS NOT NULL THEN
					(COALESCE(c.metadata, '{}'::jsonb) || json_build_object(
						'host', c.host,
						'port', c.port,
						'database', c.database,
						'username', c.username,
						'password', c.password,
						'schema', c.schema
					)::jsonb)
				ELSE tpd.config
			END AS connection_details,
			t.is_gold_copy AS is_gold_copy
		FROM
			public.tenant_product_datasource tpd
		LEFT JOIN
			public.connections c ON tpd.connection_id = c.id
		JOIN
			public.alpha_datasource ad ON tpd.alpha_datasource_id = ad.id
		JOIN
			public.tenant_product tp ON tpd.tenant_product_id = tp.id
		JOIN
			public.tenant_instance ti ON ti.id = tp.datasource_id
		JOIN
			public.tenants t ON t.id = ti.tenant_id
		WHERE
			ti.tenant_id = $1
			AND (ad.datasource_code LIKE 'crims%' OR ad.datasource_code LIKE 'FO%ORM%' OR c.database = 'crims')
		LIMIT 1
	`
	var ds crimsDatasource
	var connDetailsJSON []byte
	err := db.QueryRowContext(ctx, query, tenantID).Scan(
		&ds.ID, &ds.TenantID, &ds.Name, &ds.SourceSystem, &connDetailsJSON, &ds.IsGoldCopy,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no CRIMS datasource found for tenant %s", tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("query CRIMS datasource: %w", err)
	}
	ds.ConnectionDetails = string(connDetailsJSON)
	return &ds, nil
}
