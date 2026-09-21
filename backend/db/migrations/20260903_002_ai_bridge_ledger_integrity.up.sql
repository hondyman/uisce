-- Tamper-evident hash chain for the AI bridge sync ledger.
-- payload_hash (added in 20260903_create_semantic_ai_bridge) is a plain
-- SHA-256 of the payload, useful for dedup/display but not tamper-evident:
-- anyone with DB access can edit a row and recompute it. This adds:
--   * prev_hmac: the hmac_signature of the previous row for this tenant,
--     forming a hash chain (each row attests to everything before it).
--   * hmac_signature: HMAC-SHA256(server secret, prev_hmac || payload_hash),
--     which cannot be recomputed without the server's encryption key.
ALTER TABLE catalog_ai.ai_bridge_sync_logs
    ADD COLUMN IF NOT EXISTS prev_hmac VARCHAR(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS hmac_signature VARCHAR(64) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_ai_bridge_sync_logs_tenant_created
    ON catalog_ai.ai_bridge_sync_logs(tenant_id, created_at ASC);
