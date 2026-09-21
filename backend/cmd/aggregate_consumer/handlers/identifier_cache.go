package handlers

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
)

// IdentifierCache transposes the crims trigger `orm.sync_identifier_cache`.
//
// When an orm.security_identifier row is inserted/updated and is_primary=true
// and effective_to IS NULL, copy the id_value into the matching column on
// orm.security (isin/cusip/ticker based on id_type).
type IdentifierCache struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const identifierHandler = "identifier_cache"

func NewIdentifierCache(pool *pgxpool.Pool, d *dedupe.Store) *IdentifierCache {
	return &IdentifierCache{pool: pool, dedupe: d}
}

func (h *IdentifierCache) Topic() string { return "crims_trg.orm.security_identifier" }

func (h *IdentifierCache) Handle(ctx context.Context, evt Event) error {
	if evt.Op != "c" && evt.Op != "u" {
		return nil
	}

	// Apply the trigger gate from the current row (after for inserts/updates).
	isPrimary := boolOf(evt.After["is_primary"])
	effectiveTo := evt.After["effective_to"]
	if !isPrimary || effectiveTo != nil {
		return nil
	}

	if err := h.dedupe.MarkIfNew(ctx, identifierHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	idType := stringOf(evt.After["id_type"])
	idValue := stringOf(evt.After["id_value"])
	securityID := stringOf(evt.After["security_id"])
	if securityID == "" || idValue == "" {
		return nil
	}

	var col string
	switch idType {
	case "ISIN":
		col = "isin"
	case "CUSIP":
		col = "cusip"
	case "TICKER":
		col = "ticker"
	default:
		// Unknown id_type; skip silently.
		return nil
	}

	query := fmt.Sprintf(`UPDATE orm.security SET %s = $1 WHERE id = $2::uuid`, col)
	if _, err := h.pool.Exec(ctx, query, idValue, securityID); err != nil {
		return fmt.Errorf("orm.security %s sync: %w", col, err)
	}
	return nil
}
