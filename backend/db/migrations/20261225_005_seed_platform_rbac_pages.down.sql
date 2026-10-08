-- Down migration: 20261225_005_seed_platform_rbac_pages.down.sql

DELETE FROM route_aliases WHERE page_key IN (
    'platform-teams', 'platform-users', 'platform-user-roles', 'platform-user-tenants',
    'platform-delegations', 'platform-field-permissions', 'platform-roles',
    'platform-ip-whitelist', 'platform-message-catalog', 'platform-seeding'
);

DELETE FROM page_definitions WHERE slug IN (
    'platform-teams', 'platform-users', 'platform-user-roles', 'platform-user-tenants',
    'platform-delegations', 'platform-field-permissions', 'platform-roles',
    'platform-ip-whitelist', 'platform-message-catalog', 'platform-seeding'
);

UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/teams' WHERE node_key = 'platform-organization-teams';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/users' WHERE node_key = 'platform-organization-users';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/user-roles' WHERE node_key = 'platform-organization-user-roles';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/user-tenants' WHERE node_key = 'platform-organization-user-tenants';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/delegations' WHERE node_key = 'platform-security-delegations';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/field-permissions' WHERE node_key = 'platform-security-field-permissions';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/rbac/roles' WHERE node_key = 'platform-security-roles-permissions';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/fabric/ip-whitelist' WHERE node_key = 'platform-security-ip-whitelist';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/message-catalog' WHERE node_key = 'system-message-catalog';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/admin/seeding' WHERE node_key = 'system-seeding';
