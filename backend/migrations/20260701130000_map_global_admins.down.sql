-- +goose Down
DELETE FROM security.identity_profile_mappings 
WHERE idp_client_id IN ('0bfc0c4d-0d18-4908-b5be-f590196d2632', 'semlayer-frontend')
  AND idp_group_id IN ('e57de815-50e5-4b04-a795-ce1da6550105', 'Uisce-Global-Admins');
