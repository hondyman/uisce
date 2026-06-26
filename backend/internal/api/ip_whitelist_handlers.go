package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/utils/ip"
	"github.com/lib/pq"
)

type IpWhitelistAPIHandlers struct {
	DB *sql.DB
}

func NewIpWhitelistAPIHandlers(db *sql.DB) *IpWhitelistAPIHandlers {
	return &IpWhitelistAPIHandlers{DB: db}
}

func (h *IpWhitelistAPIHandlers) RegisterRoutes(r chi.Router) {
	// Expose a lightweight tenant list for the IP whitelist UI
	r.Get("/tenants", h.listTenants)
	r.Get("/tenants/{tenantId}/ip-whitelist", h.getIpWhitelist)
	r.Post("/tenants/{tenantId}/ip-whitelist", h.addIpWhitelist)
	r.Delete("/tenants/{tenantId}/ip-whitelist", h.deleteIpWhitelist)
	// List all whitelist entries across the system (with owning tenant IDs)
	r.Get("/ip-whitelist", h.listAllIpWhitelist)
}

// listAllIpWhitelist returns every whitelist entry with its label and the list of owning tenant IDs
func (h *IpWhitelistAPIHandlers) listAllIpWhitelist(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT e.ip_address,
			   e.label,
			   COALESCE(array_agg(a.tenant_id::text) FILTER (WHERE a.tenant_id IS NOT NULL), ARRAY[]::text[]) AS tenant_ids,
		   COUNT(a.tenant_id) AS assignment_count
		FROM tenant_ip_whitelist_entries e
		LEFT JOIN tenant_ip_whitelist_assignments a ON a.whitelist_id = e.id
		GROUP BY e.ip_address, e.label
		ORDER BY e.ip_address
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type item struct {
		IpAddress  string   `json:"ipAddress"`
		Label      *string  `json:"label"`
		TenantIds  []string `json:"tenantIds"`
		AllTenants bool     `json:"allTenants"`
	}
	var out []item

	for rows.Next() {
		var ip string
		var label sql.NullString
		var tids pq.StringArray
		var assignmentCount int
		if err := rows.Scan(&ip, &label, &tids, &assignmentCount); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var lblPtr *string
		if label.Valid {
			v := label.String
			lblPtr = &v
		}
		// AllTenants means "enforced globally"; in this schema it's represented as no assignment rows
		out = append(out, item{IpAddress: ip, Label: lblPtr, TenantIds: []string(tids), AllTenants: assignmentCount == 0})
	}

	respond(w, r, map[string]interface{}{"whitelist": out}, nil)
}

func (h *IpWhitelistAPIHandlers) listTenants(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.QueryContext(r.Context(), `SELECT id, COALESCE(display_name, name, tenant_code, id::text) as display_name FROM tenants ORDER BY display_name`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []map[string]string
	for rows.Next() {
		var id string
		var display string
		if err := rows.Scan(&id, &display); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, map[string]string{"id": id, "displayName": display})
	}

	respond(w, r, map[string]interface{}{"tenants": out}, nil)
}

func (h *IpWhitelistAPIHandlers) getIpWhitelist(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")

	var whitelist []map[string]interface{}
	// Return entries assigned to this tenant, plus any entries with no assignments (global/unassigned)
	rows, err := h.DB.QueryContext(r.Context(), `
		SELECT e.ip_address, e.label
		FROM tenant_ip_whitelist_entries e
		LEFT JOIN tenant_ip_whitelist_assignments a_self ON a_self.whitelist_id = e.id AND a_self.tenant_id = $1
		WHERE a_self.tenant_id IS NOT NULL
		   OR NOT EXISTS (SELECT 1 FROM tenant_ip_whitelist_assignments a2 WHERE a2.whitelist_id = e.id)
		ORDER BY e.ip_address
	`, tenantId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var ipAddress string
		var label sql.NullString
		if err := rows.Scan(&ipAddress, &label); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		item := map[string]interface{}{"ipAddress": ipAddress}
		if label.Valid {
			item["label"] = label.String
		} else {
			item["label"] = nil
		}
		whitelist = append(whitelist, item)
	}

	respond(w, r, map[string]interface{}{"whitelist": whitelist}, nil)
}

func (h *IpWhitelistAPIHandlers) addIpWhitelist(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")

	var req struct {
		IpAddress  string   `json:"ipAddress"`
		Label      *string  `json:"label,omitempty"`
		TenantIds  []string `json:"tenantIds,omitempty"`  // optional extra tenants to assign
		AllTenants bool     `json:"allTenants,omitempty"` // when true, treat as global (no assignments)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate IP: support IPv4, wildcard '*', or CIDR a.b.c.d/nn
	if !ip.IsValidIPv4WildcardOrCIDR(req.IpAddress) {
		http.Error(w, "Invalid IP address, wildcard pattern, or CIDR. Use IPv4 like 192.168.1.1, wildcard like 192.168.*.*, or CIDR like 10.0.0.0/8", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Upsert entry
	// Prevent adding overlapping entries. If an existing pattern overlaps the requested
	// pattern (e.g. existing 192.168.*.* or 10.0.0.0/8 and new 192.168.1.1), reject to avoid duplicates.
	rows, err := tx.QueryContext(r.Context(), `SELECT ip_address FROM tenant_ip_whitelist_entries`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var existing string
		if err := rows.Scan(&existing); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// If exact same entry, allow upsert to update label
		if existing == req.IpAddress {
			continue
		}
		if ip.PatternsOverlapOrCIDR(existing, req.IpAddress) {
			// Fetch tenants assigned to the conflicting entry so the UI can show who owns it
			rows2, err := tx.QueryContext(r.Context(), `
				SELECT tenant_id FROM tenant_ip_whitelist_assignments a
				JOIN tenant_ip_whitelist_entries e ON e.id = a.whitelist_id
				WHERE e.ip_address = $1
			`, existing)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows2.Close()
			var tenantIds []string
			for rows2.Next() {
				var t string
				if err := rows2.Scan(&t); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				tenantIds = append(tenantIds, t)
			}

			// Return structured JSON with the conflicting ip and owning tenants
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			resp := map[string]interface{}{
				"message": "IP or CIDR overlaps an existing whitelist entry",
				"conflicting": map[string]interface{}{
					"ipAddress": existing,
					"tenantIds": tenantIds,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}
	}

	var entryID string
	err = tx.QueryRowContext(r.Context(), `
		INSERT INTO tenant_ip_whitelist_entries (ip_address, label)
		VALUES ($1, $2)
		ON CONFLICT (ip_address) DO UPDATE SET label = EXCLUDED.label, updated_at = now()
		RETURNING id
	`, req.IpAddress, req.Label).Scan(&entryID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If this is marked as All Tenants, do not create any assignment rows.
	if !req.AllTenants {
		// Assign to the current tenant
		_, err = tx.ExecContext(r.Context(), `
			INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, entryID, tenantId)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Assign additional tenant IDs if provided (ignored for AllTenants)
	if !req.AllTenants {
		for _, t := range req.TenantIds {
			if t == "" {
				continue
			}
			if _, err := tx.ExecContext(r.Context(), `INSERT INTO tenant_ip_whitelist_assignments (whitelist_id, tenant_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, entryID, t); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, r, map[string]string{"status": "success"}, nil)
}

func (h *IpWhitelistAPIHandlers) deleteIpWhitelist(w http.ResponseWriter, r *http.Request) {
	tenantId := chi.URLParam(r, "tenantId")

	var req struct {
		IpAddress string `json:"ipAddress"`
		// If TenantIds provided, remove assignments for those tenants; otherwise remove for current tenant
		TenantIds []string `json:"tenantIds,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tx, err := h.DB.BeginTx(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	// Find entry id
	var entryID string
	err = tx.QueryRowContext(r.Context(), `SELECT id FROM tenant_ip_whitelist_entries WHERE ip_address = $1`, req.IpAddress).Scan(&entryID)
	if err == sql.ErrNoRows {
		respond(w, r, map[string]string{"status": "not_found"}, nil)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Remove assignments
	if len(req.TenantIds) > 0 {
		for _, t := range req.TenantIds {
			if _, err := tx.ExecContext(r.Context(), `DELETE FROM tenant_ip_whitelist_assignments WHERE whitelist_id = $1 AND tenant_id = $2`, entryID, t); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	} else {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM tenant_ip_whitelist_assignments WHERE whitelist_id = $1 AND tenant_id = $2`, entryID, tenantId); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// If no more assignments exist for this entry, delete the entry
	var count int
	err = tx.QueryRowContext(r.Context(), `SELECT COUNT(1) FROM tenant_ip_whitelist_assignments WHERE whitelist_id = $1`, entryID).Scan(&count)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if count == 0 {
		if _, err := tx.ExecContext(r.Context(), `DELETE FROM tenant_ip_whitelist_entries WHERE id = $1`, entryID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respond(w, r, map[string]string{"status": "success"}, nil)
}
