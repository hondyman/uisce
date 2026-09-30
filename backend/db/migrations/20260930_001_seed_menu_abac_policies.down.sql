-- +goose Down
DELETE FROM studio.tenant_abac_policies
WHERE tenant_id IS NULL
  AND target_profile_key IN ('BASE_USER', 'PLATFORM_OPERATOR')
  AND action_attribute LIKE 'menu:%';
