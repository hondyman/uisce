package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	dbpkg "github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/goldcopy"
	"github.com/jmoiron/sqlx"
)

// maxBulkRecords caps one bulk request. Callers (the data pipeline) chunk.
const maxBulkRecords = 5000

type boBulkRequest struct {
	// Mode is "create" (default) or "upsert".
	Mode string `json:"mode"`
	// KeyFields identify an existing record for upsert (business key columns).
	KeyFields []string `json:"key_fields"`
	// DryRun validates every record without writing.
	DryRun  bool                     `json:"dry_run"`
	Records []map[string]interface{} `json:"records"`
}

type boBulkFailure struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

type boBulkResponse struct {
	Inserted int             `json:"inserted"`
	Updated  int             `json:"updated"`
	Failed   []boBulkFailure `json:"failed"`
}

// HandleBulkBORecords writes many records in one request/transaction.
//
// Each record runs under its own savepoint so one bad row is reported in
// `failed` without discarding the rest. Writes go through the same contract
// resolution and writable-column allowlist as the single-record endpoints, and
// tenant GUCs are applied to the transaction (RLS stays in force).
func (h *BOCRUDHandler) HandleBulkBORecords(w http.ResponseWriter, r *http.Request) {
	tenantID, err := extractTenantUUIDFromRequest(r)
	if err != nil {
		status := http.StatusUnauthorized
		if te, ok := err.(*tenantResolutionError); ok {
			status = te.status
		}
		http.Error(w, err.Error(), status)
		return
	}
	boKey := chi.URLParam(r, "boKey")

	var req boBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Records) == 0 {
		http.Error(w, "records is required", http.StatusBadRequest)
		return
	}
	if len(req.Records) > maxBulkRecords {
		http.Error(w, fmt.Sprintf("at most %d records per request", maxBulkRecords), http.StatusRequestEntityTooLarge)
		return
	}
	switch req.Mode {
	case "", "create":
		req.Mode = "create"
	case "upsert":
		if len(req.KeyFields) == 0 {
			http.Error(w, "upsert requires key_fields", http.StatusBadRequest)
			return
		}
	default:
		http.Error(w, "mode must be create or upsert", http.StatusBadRequest)
		return
	}

	boMeta, err := h.resolveBOMetadata(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving BO contract: %v", err), http.StatusNotFound)
		return
	}
	writable, err := h.resolveWritableColumns(r.Context(), boMeta.DrivingTable)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving table schema: %v", err), http.StatusInternalServerError)
		return
	}
	for _, k := range req.KeyFields {
		if !writable[k] {
			http.Error(w, fmt.Sprintf("unknown key field '%s'", k), http.StatusBadRequest)
			return
		}
	}
	tenantScoped := h.tableHasColumn(r.Context(), boMeta.DrivingTable, "tenant_id")

	tx, err := h.db.BeginTxx(r.Context(), nil)
	if err != nil {
		http.Error(w, "failed starting transaction", http.StatusInternalServerError)
		return
	}
	defer func() { _ = tx.Rollback() }()
	gold := goldcopy.ResolveTenantID(r.Context(), h.db)
	if err := dbpkg.ApplyTenantGUCs(r.Context(), tx.Tx, tenantID.String(), gold.String()); err != nil {
		http.Error(w, "tenant GUC: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := boBulkResponse{Failed: []boBulkFailure{}}
	type event struct {
		key  string
		id   string
		data map[string]interface{}
	}
	var events []event

	for i, rec := range req.Records {
		if _, err := tx.ExecContext(r.Context(), "SAVEPOINT bulk_row"); err != nil {
			http.Error(w, "savepoint: "+err.Error(), http.StatusInternalServerError)
			return
		}
		res, inserted, err := h.bulkWriteOne(r, tx, boMeta.DrivingTable, boMeta.KeyColumn,
			tenantID.String(), tenantScoped, writable, req, rec)
		if err != nil {
			_, _ = tx.ExecContext(r.Context(), "ROLLBACK TO SAVEPOINT bulk_row")
			resp.Failed = append(resp.Failed, boBulkFailure{Index: i, Error: err.Error()})
			continue
		}
		_, _ = tx.ExecContext(r.Context(), "RELEASE SAVEPOINT bulk_row")
		if inserted {
			resp.Inserted++
		} else {
			resp.Updated++
		}
		if res != nil {
			trig := "row_update"
			if inserted {
				trig = "row_insert"
			}
			events = append(events, event{trig, fmt.Sprintf("%v", res[boMeta.KeyColumn]), res})
		}
	}

	if req.DryRun {
		_ = tx.Rollback()
	} else {
		if err := tx.Commit(); err != nil {
			http.Error(w, "commit failed: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// Row events only after a durable commit.
		for _, e := range events {
			h.emitBORowEvent(e.key, tenantID, boKey, e.id, e.data)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// bulkWriteOne inserts or (for upsert) updates one record inside the caller's
// savepoint. It returns the written row and whether it was an insert.
func (h *BOCRUDHandler) bulkWriteOne(
	r *http.Request, tx *sqlx.Tx,
	table, keyCol, tenantID string, tenantScoped bool, writable map[string]bool,
	req boBulkRequest, rec map[string]interface{},
) (map[string]interface{}, bool, error) {
	ctx := r.Context()

	var cols []string
	var vals []interface{}
	for k, v := range rec {
		lower := strings.ToLower(k)
		if lower == "tenant_id" || lower == "created_at" || lower == "updated_at" {
			continue
		}
		if !writable[k] {
			return nil, false, fmt.Errorf("unknown attribute '%s'", k)
		}
		cols = append(cols, k)
		vals = append(vals, v)
	}
	if len(cols) == 0 {
		return nil, false, fmt.Errorf("record has no writable attributes")
	}

	if req.Mode == "upsert" {
		var where []string
		var wargs []interface{}
		n := 0
		if tenantScoped {
			n++
			where = append(where, fmt.Sprintf("tenant_id = $%d", n))
			wargs = append(wargs, tenantID)
		}
		for _, k := range req.KeyFields {
			v, ok := rec[k]
			if !ok || v == nil {
				return nil, false, fmt.Errorf("missing key field '%s'", k)
			}
			n++
			where = append(where, fmt.Sprintf("%s = $%d", quoteIdent(k), n))
			wargs = append(wargs, v)
		}
		var sets []string
		var uargs []interface{}
		for i, c := range cols {
			isKey := false
			for _, k := range req.KeyFields {
				if k == c {
					isKey = true
				}
			}
			if isKey {
				continue
			}
			sets = append(sets, fmt.Sprintf("%s = $%d", quoteIdent(c), n+len(uargs)+1))
			uargs = append(uargs, vals[i])
		}
		if len(sets) > 0 {
			q := fmt.Sprintf("UPDATE %s SET %s WHERE %s RETURNING *",
				table, strings.Join(sets, ", "), strings.Join(where, " AND "))
			args := append(append([]interface{}{}, wargs...), uargs...)
			rows, err := tx.QueryxContext(ctx, q, args...)
			if err != nil {
				return nil, false, err
			}
			res := map[string]interface{}{}
			found := false
			if rows.Next() {
				found = true
				if err := rows.MapScan(res); err != nil {
					rows.Close()
					return nil, false, err
				}
			}
			rows.Close()
			if found {
				cleanScanResult(res)
				return res, false, nil
			}
		} else {
			var exists bool
			q := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM %s WHERE %s)", table, strings.Join(where, " AND "))
			if err := tx.GetContext(ctx, &exists, q, wargs...); err != nil {
				return nil, false, err
			}
			if exists {
				return nil, false, nil
			}
		}
		// not found: fall through to insert
	}

	icols := []string{}
	ph := []string{}
	iargs := []interface{}{}
	if tenantScoped {
		icols = append(icols, "tenant_id")
		ph = append(ph, "$1")
		iargs = append(iargs, tenantID)
	}
	for i, c := range cols {
		icols = append(icols, quoteIdent(c))
		ph = append(ph, fmt.Sprintf("$%d", len(iargs)+1))
		iargs = append(iargs, vals[i])
	}
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING *",
		table, strings.Join(icols, ", "), strings.Join(ph, ", "))
	rows, err := tx.QueryxContext(ctx, q, iargs...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	res := map[string]interface{}{}
	if rows.Next() {
		if err := rows.MapScan(res); err != nil {
			return nil, false, err
		}
	}
	cleanScanResult(res)
	return res, true, nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
