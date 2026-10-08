-- Down migration: 20261225_004_seed_compliance_and_platform_pages.down.sql

DELETE FROM route_aliases WHERE page_key = 'compliance-hub';

UPDATE navigation_menu_nodes
SET target_page_key = NULL,
    target_route = '/governance/compliance'
WHERE node_key = 'operations-governance-compliance';
