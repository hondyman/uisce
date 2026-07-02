-- +goose Up
-- Map Keycloak group Uisce-Global-Admins to global_admin functional_role.
-- We seed all variations of client ID and group ID/names to support both UUIDs and strings.
INSERT INTO security.identity_profile_mappings (
    mapping_id, tenant_id, idp_client_id, idp_group_id, functional_role, clearance_level
) VALUES 
(
    gen_random_uuid(),
    '00000000-0000-0000-0000-000000000000',
    '0bfc0c4d-0d18-4908-b5be-f590196d2632',
    'e57de815-50e5-4b04-a795-ce1da6550105',
    'global_admin',
    'L3'
),
(
    gen_random_uuid(),
    '00000000-0000-0000-0000-000000000000',
    'semlayer-frontend',
    'e57de815-50e5-4b04-a795-ce1da6550105',
    'global_admin',
    'L3'
),
(
    gen_random_uuid(),
    '00000000-0000-0000-0000-000000000000',
    '0bfc0c4d-0d18-4908-b5be-f590196d2632',
    'Uisce-Global-Admins',
    'global_admin',
    'L3'
),
(
    gen_random_uuid(),
    '00000000-0000-0000-0000-000000000000',
    'semlayer-frontend',
    'Uisce-Global-Admins',
    'global_admin',
    'L3'
)
ON CONFLICT (idp_client_id, idp_group_id) DO UPDATE
SET functional_role = EXCLUDED.functional_role,
    tenant_id = EXCLUDED.tenant_id;
