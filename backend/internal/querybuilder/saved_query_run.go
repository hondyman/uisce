package querybuilder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/security"
)

// ErrSavedQueryNotFound: no such saved query visible to the user in the tenant.
var ErrSavedQueryNotFound = errors.New("saved query not found")

// SavedQueryRef is a saved query a user can run.
type SavedQueryRef struct {
	ID          string `db:"id"`
	Name        string `db:"name"`
	Description string `db:"description"`
}

// FindSavedQuery returns the saved query if userID may run it in tenantID
// (their own, or shared in the tenant).
func (h *SavedQueryHandler) FindSavedQuery(ctx context.Context, tenantID, userID, queryID string) (*SavedQuery, error) {
	var row savedQueryRow
	err := h.db.GetContext(ctx, &row, `SELECT `+savedQuerySelectCols+`
		FROM data_explorer.saved_query
		WHERE id::text = $1 AND tenant_id::text = $2 AND (visibility = 'shared' OR user_id::text = $3)`,
		queryID, tenantID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSavedQueryNotFound
	}
	if err != nil {
		return nil, err
	}
	sq := row.toSavedQuery()
	return &sq, nil
}

// ListRunnableSavedQueries lists the saved queries userID may run in tenantID.
func (h *SavedQueryHandler) ListRunnableSavedQueries(ctx context.Context, tenantID, userID string) ([]SavedQueryRef, error) {
	var out []SavedQueryRef
	err := h.db.SelectContext(ctx, &out, `SELECT id::text, name, COALESCE(description, '') AS description
		FROM data_explorer.saved_query
		WHERE tenant_id::text = $1 AND (visibility = 'shared' OR user_id::text = $2)
		ORDER BY name`, tenantID, userID)
	return out, err
}

// RunSaved executes a saved query without an HTTP request - for the
// scheduler - exactly as its preview does, as userID in tenantID's
// datasource. The datasource is re-checked against the tenant on every run
// (security.BuildContext), never trusted from storage alone.
func (h *SavedQueryHandler) RunSaved(ctx context.Context, tenantID, userID, datasourceID, region, queryID string,
	params map[string][]string, limit int) (*boresolver.QueryExecuteResponse, *SavedQuery, error) {
	sq, err := h.FindSavedQuery(ctx, tenantID, userID, queryID)
	if err != nil {
		return nil, nil, err
	}
	if region == "" {
		region = "us-east-1"
	}
	secCtx, err := security.BuildContext(ctx, security.AuthInfo{UserID: userID, TenantIDs: []string{tenantID}},
		security.BuildContextRequest{DatasourceID: datasourceID, Region: region}, h.deps.Resolver)
	if err != nil {
		return nil, sq, fmt.Errorf("security context for saved query %s: %w", queryID, err)
	}
	filters, err := resolveParams(sq.State, params)
	if err != nil {
		return nil, sq, err
	}
	if limit <= 0 {
		limit = sq.State.Limit
	}
	db := h.executor.QueryDB(secCtx.DatasourceID)
	if db == nil {
		return nil, sq, fmt.Errorf("no database connection for datasource %s", secCtx.DatasourceID)
	}
	resp, err := h.service.Execute(security.WithContext(ctx, secCtx), secCtx, savedQueryDef(*sq, tenantID, filters, limit), db)
	return resp, sq, err
}
