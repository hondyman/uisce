package tiles

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SettlementWriterConfig struct {
	SettlementSubtype string `json:"settlement_subtype"` // "dvp_securities" | "free_of_payment"
	UpsertOnConflict  bool   `json:"upsert_on_conflict"`
}

func NewSettlementWriterLoader(cfg SettlementWriterConfig, db *sql.DB) TileFunc {
	return func(ctx context.Context, records []Record) ([]Record, []string, error) {
		out := make([]Record, 0, len(records))
		var errs []string
		tctx := TenantFromContext(ctx)

		for _, rec := range records {
			semantic, ok := rec["semantic"].(map[string]any)
			if !ok {
				errs = append(errs, "settlement_writer: missing semantic map")
				continue
			}

			txRef, _ := rec["transaction_ref"].(string)
			if txRef == "" {
				var rawBytes []byte
				switch v := rec["raw_bytes"].(type) {
				case []byte:
					rawBytes = v
				case string:
					rawBytes = []byte(v)
				}
				hash := sha256.Sum256(rawBytes)
				txRef = hex.EncodeToString(hash[:])[:32]
			}

			id := uuid.New().String()
			accountId, _ := semantic["account_id"].(string)
			if accountId == "" {
				accountId = uuid.Nil.String()
			}
			amount, _ := semantic["settlement_amount"].(string)
			if amount == "" {
				amount = "0"
			}
			currency, _ := semantic["currency"].(string)
			settlementDate, _ := semantic["settlement_date"].(string)
			if settlementDate == "" {
				settlementDate = time.Now().Format(time.RFC3339)
			}

			// transaction_ref column added by migration 20261001_008.
			custodianID, _ := rec["custodian_id"].(string)
			query := `
INSERT INTO cash_flow.settlement (
    id, tenant_id, account_id, amount, currency, settlement_date, settlement_status,
    subtype_code, transaction_ref, created_at, updated_at, valid_from
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW(), NOW())
`
			args := []any{
				id, tctx.TenantID, accountId, amount, currency, settlementDate, "PENDING",
				cfg.SettlementSubtype, txRef,
			}

			if cfg.UpsertOnConflict {
				query += ` ON CONFLICT (tenant_id, transaction_ref) WHERE transaction_ref IS NOT NULL DO UPDATE SET settlement_status = EXCLUDED.settlement_status, updated_at = NOW()`
			} else {
				query += ` ON CONFLICT (tenant_id, transaction_ref) WHERE transaction_ref IS NOT NULL DO NOTHING`
			}
			_ = custodianID // available for future custodian_id column addition

			_, err := db.ExecContext(ctx, query, args...)
			if err != nil {
				errs = append(errs, fmt.Sprintf("settlement_writer: insert failed: %v", err))
				continue
			}

			out = append(out, rec)
		}
		return out, errs, nil
	}
}
