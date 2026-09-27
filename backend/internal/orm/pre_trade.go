package orm

import (
	"context"
	"database/sql"
	"encoding/json"
)

// PreTradeCheck is one row of orm.pre_trade_check.
type PreTradeCheck struct {
	OrderID      string
	CheckType    string
	CheckStatus  string
	RuleID       sql.NullString
	RuleCd       sql.NullString
	BreachAmount sql.NullFloat64
	LimitAmount  sql.NullFloat64
	BreachPct    sql.NullFloat64
	Details      json.RawMessage
	IsBlocking   bool
}

// InsertPreTradeCheck runs after the compliance engine evaluates a rule.
func InsertPreTradeCheck(ctx context.Context, tx *sql.Tx, tenantID string, c *PreTradeCheck) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO orm.pre_trade_check (
			order_id, check_time, check_type, check_status,
			rule_id, rule_cd, breach_amount, limit_amount, breach_pct,
			details, is_blocking, tenant_id
		) VALUES ($1, now(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, c.OrderID, c.CheckType, c.CheckStatus,
		c.RuleID, c.RuleCd, c.BreachAmount, c.LimitAmount, c.BreachPct,
		c.Details, c.IsBlocking, tenantID)
	return err
}

// OverridePreTradeCheck records a manual override on a failed check.
func OverridePreTradeCheck(ctx context.Context, tx *sql.Tx, tenantID,
	checkID, by, reason string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE orm.pre_trade_check
		SET override_reason = $1, overridden_by = $2, overridden_at = now()
		WHERE id = $3 AND tenant_id = $4
	`, reason, by, checkID, tenantID)
	return err
}

// IsSecurityRestricted answers the pre-trade restriction check in one query.
func IsSecurityRestricted(ctx context.Context, tx *sql.Tx, tenantID,
	securityID, accountID string) (sql.NullString, error) {
	var action sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT restriction_action
		FROM orm.restricted_list
		WHERE tenant_id = $1
		  AND security_id = $2
		  AND is_active = true
		  AND effective_from <= now()
		  AND (effective_to IS NULL OR effective_to > now())
		  AND (account_id IS NULL OR account_id = $3)
		ORDER BY CASE restriction_action
			WHEN 'PROHIBIT' THEN 1
			WHEN 'PROHIBIT_BUY' THEN 2
			WHEN 'PROHIBIT_SELL' THEN 3
			WHEN 'PROHIBIT_SHORT' THEN 4
			WHEN 'REQUIRE_APPROVAL' THEN 5
			WHEN 'LIMIT' THEN 6
			ELSE 9 END
		LIMIT 1
	`, tenantID, securityID, accountID).Scan(&action)
	if err == sql.ErrNoRows {
		return action, nil
	}
	return action, err
}
