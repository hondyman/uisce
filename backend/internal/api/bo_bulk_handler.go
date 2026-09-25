package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/jmoiron/sqlx"
)

// maxBulkRecords caps one request; callers (e.g. the data pipeline) chunk.
const maxBulkRecords = 5000

type boBulkRequest struct {
	// Mode is "create" (default) or "upsert" (update the row matching
	// KeyFields for this tenant, else insert it).
	Mode      string                   `json:"mode"`
	KeyFields []string                 `json:"key_fields"`
	DryRun    bool                     `json:"dry_run"`
	Records   []map[string]interface{} `json:"records"`
}

type boBulkFailure struct {
	Index int      `json:"index"`
	Error string   `json:"error"`
	Rules []string `json:"rules,omitempty"` // set when the rule engine rejected the row
	// Missing lists required fields the row left empty.
	Missing []string `json:"missing,omitempty"`
}

type boBulkResponse struct {
	Written int             `json:"written"`
	Failed  []boBulkFailure `json:"failed"`
	DryRun  bool            `json:"dry_run"`
}

// bulkRequestError is a request-level refusal (bad payload, unknown BO);
// status is the HTTP status the endpoint answers with.
type bulkRequestError struct {
	status int
	msg    string
}

func (e *bulkRequestError) Error() string { return e.msg }

func bulkErr(status int, f string, a ...interface{}) error {
	return &bulkRequestError{status: status, msg: fmt.Sprintf(f, a...)}
}

// HandleBulkBORecords writes many records for one BO in one transaction.
// Each row is judged by the master rule engine exactly as a single write is
// (metadata.EnforceWriteBatch); a rejected or invalid row is reported in
// `failed` and the rest still commit. Row events fire only after commit.
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
	var req boBulkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	resp, err := h.bulkWrite(r.Context(), tenantID, chi.URLParam(r, "boKey"), r.URL.Query().Get("subtype"), req)
	if err != nil {
		var be *bulkRequestError
		if errors.As(err, &be) {
			http.Error(w, be.msg, be.status)
			return
		}
		http.Error(w, "bulk write failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// bulkWrite is the enforced bulk write shared by the HTTP endpoint and the
// data pipeline's bo_sink. The caller has already resolved the tenant.
func (h *BOCRUDHandler) bulkWrite(ctx context.Context, tenantID uuid.UUID, boKey, subtype string, req boBulkRequest) (*boBulkResponse, error) {
	if h.enforcer == nil {
		return nil, bulkErr(http.StatusServiceUnavailable, "%s", errNoRuleEnforcer.Error())
	}
	if len(req.Records) == 0 {
		return nil, bulkErr(http.StatusBadRequest, "records is required")
	}
	if len(req.Records) > maxBulkRecords {
		return nil, bulkErr(http.StatusRequestEntityTooLarge, "at most %d records per request", maxBulkRecords)
	}
	switch req.Mode {
	case "", "create":
		req.Mode = "create"
	case "upsert":
		if len(req.KeyFields) == 0 {
			return nil, bulkErr(http.StatusBadRequest, "upsert requires key_fields")
		}
	default:
		return nil, bulkErr(http.StatusBadRequest, "mode must be create or upsert")
	}

	boMeta, err := h.resolveBOMetadata(ctx, boKey, tenantID)
	if err != nil {
		return nil, bulkErr(http.StatusNotFound, "failed resolving BO contract: %v", err)
	}
	writable, err := h.resolveWritableColumns(ctx, boMeta.RecordsDB, boMeta.DrivingTable)
	if err != nil {
		return nil, fmt.Errorf("failed resolving table schema: %w", err)
	}
	for _, k := range req.KeyFields {
		if !writable[k] || strings.EqualFold(k, "tenant_id") {
			return nil, bulkErr(http.StatusBadRequest, "invalid key field '%s'", k)
		}
	}
	tenantScoped := h.tableHasColumn(ctx, boMeta.RecordsDB, boMeta.DrivingTable, "tenant_id")
	if subtype != "" {
		if col, ok := h.resolveDiscriminatorColumn(ctx, boMeta.RecordsDB, boMeta.DrivingTable); ok {
			for _, rec := range req.Records {
				rec[col] = subtype // forced server-side, as on single create
			}
		}
	}

	// ops[i] records whether row i was an update or an insert, so row events
	// use the same trigger keys as the single-record endpoints.
	ops := make([]string, len(req.Records))
	results, err := h.enforcer.EnforceWriteBatch(ctx, tenantID.String(), boKey, len(req.Records), req.DryRun,
		func(tx *sqlx.Tx, i int) (map[string]interface{}, error) {
			rec := req.Records[i]
			if req.Mode == "upsert" {
				q, args, err := buildBOUpsertUpdate(boMeta.DrivingTable, tenantID.String(), tenantScoped, writable, req.KeyFields, rec)
				if err != nil {
					return nil, err
				}
				if q != "" {
					row, err := queryOneRow(ctx, tx, q, args)
					if err == nil {
						ops[i] = "row_update"
						return row, nil
					}
					if !errors.Is(err, metadata.ErrNoRowWritten) {
						return nil, err
					}
				}
			}
			q, args, err := buildBOInsert(boMeta.DrivingTable, tenantID, tenantScoped, writable, rec)
			if err != nil {
				return nil, err
			}
			ops[i] = "row_insert"
			return queryOneRow(ctx, tx, q, args)
		})
	if err != nil {
		return nil, err
	}

	resp := &boBulkResponse{Failed: []boBulkFailure{}, DryRun: req.DryRun}
	for _, res := range results {
		if res.Err != nil {
			f := boBulkFailure{Index: res.Index, Error: res.Err.Error()}
			var rej *metadata.RuleRejectionError
			if errors.As(res.Err, &rej) {
				f.Rules = rej.Rules
			}
			var req *metadata.RequiredFieldsError
			if errors.As(res.Err, &req) {
				f.Missing = req.Fields
			}
			resp.Failed = append(resp.Failed, f)
			continue
		}
		resp.Written++
		if !req.DryRun {
			h.emitBORowEvent(ops[res.Index], tenantID, boKey, fmt.Sprintf("%v", res.Record[boMeta.KeyColumn]), res.Record)
		}
	}
	return resp, nil
}

// buildBOUpsertUpdate builds the UPDATE half of an upsert: set every
// non-key writable field on the tenant's row matching keyFields. Returns
// ("", nil, nil) when the record has no non-key fields to set (the caller
// then just attempts the insert).
func buildBOUpsertUpdate(table, tenantID string, tenantScoped bool, writable map[string]bool, keyFields []string, rec map[string]interface{}) (string, []interface{}, error) {
	isKey := map[string]bool{}
	var where []string
	var args []interface{}
	if tenantScoped {
		args = append(args, tenantID)
		where = append(where, "tenant_id = $1")
	}
	for _, k := range keyFields {
		v, ok := rec[k]
		if !ok || v == nil {
			return "", nil, fmt.Errorf("missing key field '%s'", k)
		}
		isKey[k] = true
		args = append(args, v)
		where = append(where, fmt.Sprintf("%s = $%d", k, len(args)))
	}
	var sets []string
	for _, k := range sortedKeys(rec) {
		lower := strings.ToLower(k)
		if isKey[k] || lower == "id" || lower == "tenant_id" || lower == "created_at" || lower == "updated_at" || lower == "created_by" {
			continue
		}
		if !writable[k] {
			return "", nil, fmt.Errorf("unknown attribute '%s'", k)
		}
		args = append(args, rec[k])
		sets = append(sets, fmt.Sprintf("%s = $%d", k, len(args)))
	}
	if len(sets) == 0 {
		return "", nil, nil
	}
	return fmt.Sprintf(`
		UPDATE %s
		SET %s, updated_at = NOW()
		WHERE %s
		RETURNING *;
	`, table, strings.Join(sets, ", "), strings.Join(where, " AND ")), args, nil
}
