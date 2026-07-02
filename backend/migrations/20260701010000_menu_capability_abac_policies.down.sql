-- Revert capability-driven UI menu ABAC policies.

DELETE FROM abac_policies
WHERE tenant_id IS NULL
  AND name IN ('Base UI Menus', 'Operator UI Menus');

DELETE FROM security.identity_profile_mappings
WHERE idp_client_id = 'investco-sso-client'
  AND idp_group_claim = 'GG-Uisce-Users';
