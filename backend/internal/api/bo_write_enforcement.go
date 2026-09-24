package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/hondyman/uisce/backend/internal/metadata"
	"github.com/jmoiron/sqlx"
)

// boWriteEnforcer is the master rule engine's write gate
// (metadata.BusinessObjectService). Every /bo/{boKey}/records write runs its
// SQL inside it, so a BO's validation/mdm/compliance rules judge the write
// in the same transaction exactly as they do on /business-objects/{id}/data.
type boWriteEnforcer interface {
	EnforceWrite(ctx context.Context, tenantID, boKeyOrID string, doWrite func(tx *sqlx.Tx) (map[string]interface{}, error)) (map[string]interface{}, error)
	EnforceWriteBatch(ctx context.Context, tenantID, boKeyOrID string, n int, dryRun bool, doWrite func(tx *sqlx.Tx, i int) (map[string]interface{}, error)) ([]metadata.BatchRowResult, error)
}

var errNoRuleEnforcer = errors.New("rule enforcement is not configured; BO writes are refused")

// enforcedWrite runs one INSERT/UPDATE ... RETURNING * under the rule engine
// and returns the written row (byte slices cleaned for JSON). A statement
// that matches no row yields metadata.ErrNoRowWritten.
func (h *BOCRUDHandler) enforcedWrite(ctx context.Context, tenantID, boKey, query string, args []interface{}) (map[string]interface{}, error) {
	if h.enforcer == nil {
		return nil, errNoRuleEnforcer // fail closed: never write around the rules
	}
	return h.enforcer.EnforceWrite(ctx, tenantID, boKey, func(tx *sqlx.Tx) (map[string]interface{}, error) {
		return queryOneRow(ctx, tx, query, args)
	})
}

func queryOneRow(ctx context.Context, tx *sqlx.Tx, query string, args []interface{}) (map[string]interface{}, error) {
	rows, err := tx.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, metadata.ErrNoRowWritten
	}
	result := make(map[string]interface{})
	if err := rows.MapScan(result); err != nil {
		return nil, err
	}
	cleanScanResult(result)
	return result, nil
}

// writeBOWriteError maps an enforced-write error to a response:
// rule rejection -> 422 with the blocking rule names, no row -> 404,
// missing enforcer -> 503, anything else -> fallbackStatus (body prefixed
// with fallbackMsg when non-empty).
func writeBOWriteError(w http.ResponseWriter, err error, notFoundMsg, fallbackMsg string, fallbackStatus int) {
	var rej *metadata.RuleRejectionError
	switch {
	case errors.As(err, &rej):
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": rej.Error(), "rules": rej.Rules})
	case errors.Is(err, metadata.ErrNoRowWritten):
		http.Error(w, notFoundMsg, http.StatusNotFound)
	case errors.Is(err, errNoRuleEnforcer):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	default:
		msg := err.Error()
		if fallbackMsg != "" {
			msg = fallbackMsg + ": " + msg
		}
		http.Error(w, msg, fallbackStatus)
	}
}
