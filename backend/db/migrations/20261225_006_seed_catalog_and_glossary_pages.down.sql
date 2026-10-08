-- Down migration: 20261225_006_seed_catalog_and_glossary_pages.down.sql

DELETE FROM route_aliases WHERE page_key IN (
    'catalog-api-inventory', 'catalog-glossary', 'catalog-business-terms',
    'catalog-custom-fields', 'core-domains', 'catalog-schema-explorer',
    'core-semantic-mapper', 'catalog-ai-suggestions'
);

DELETE FROM page_definitions WHERE slug IN (
    'catalog-api-inventory', 'catalog-glossary', 'catalog-business-terms',
    'catalog-custom-fields', 'core-domains', 'catalog-schema-explorer',
    'core-semantic-mapper', 'catalog-ai-suggestions'
);

UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/catalog/api-inventory' WHERE node_key = 'catalog-glossary-api-inventory';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/core/glossary' WHERE node_key = 'catalog-glossary-semantic-terms';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/core/business-terms' WHERE node_key = 'catalog-glossary-business-terms';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/catalog/custom-fields' WHERE node_key = 'build-data-manage-custom-fields';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/core/domains' WHERE node_key = 'catalog-glossary-data-domains';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/schema-explorer' WHERE node_key = 'catalog-glossary-datasource-explorer';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/core/semantic-mapper' WHERE node_key = 'catalog-config-semantic-mapper';
UPDATE navigation_menu_nodes SET target_page_key = NULL, target_route = '/catalog/ai-suggestions' WHERE node_key = 'catalog-config-ai-term-suggestions';
