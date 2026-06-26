-- 20251028_create_uma_tables/down.sql
-- Drop UMA tables and triggers (reverse of up.sql)

DROP TRIGGER IF EXISTS trigger_audit_uma_rebalance_requests ON uma_rebalance_requests;
DROP FUNCTION IF EXISTS audit_uma_rebalance_requests();

DROP TRIGGER IF EXISTS trigger_audit_uma_accounts ON uma_accounts;
DROP FUNCTION IF EXISTS audit_uma_accounts();

DROP TABLE IF EXISTS uma_rebalance_history;
DROP TABLE IF EXISTS uma_rebalance_plans;
DROP TABLE IF EXISTS uma_rebalance_requests;
DROP TABLE IF EXISTS uma_holdings;
DROP TABLE IF EXISTS uma_sleeves;
DROP TABLE IF EXISTS uma_accounts;
