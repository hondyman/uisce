-- Rollback migration 008
DO $$
DECLARE
    v_gold_copy_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
BEGIN
    DELETE FROM route_aliases WHERE tenant_id = v_gold_copy_id AND page_key IN (
        'intelligence-dashboard', 'intelligence-index-advisor', 'intelligence-storage',
        'intelligence-data-quality', 'optimization-aso', 'observability-dashboard',
        'observability-slos', 'nlq-page', 'global-intelligence',
        'analytics-scenario-analysis', 'analytics-portfolio-master', 'analytics-advisor-dashboard',
        'fabric-calculations', 'fabric-preaggregations', 'fabric-dashboard', 'fabric-settings',
        'reports-library', 'reports-builder', 'reports-queries', 'reports-models',
        'bundle-explorer', 'views', 'business-objects', 'admin-llm', 'admin-temporal-ops',
        'admin-entitlements', 'admin-audit', 'tenants-management', 'security-access-rules',
        'access-explanation', 'jit-requests', 'secrets-config'
    );

    DELETE FROM page_definitions WHERE tenant_id = v_gold_copy_id AND slug IN (
        'intelligence-dashboard', 'intelligence-index-advisor', 'intelligence-storage',
        'intelligence-data-quality', 'optimization-aso', 'observability-dashboard',
        'observability-slos', 'nlq-page', 'global-intelligence',
        'analytics-scenario-analysis', 'analytics-portfolio-master', 'analytics-advisor-dashboard',
        'fabric-calculations', 'fabric-preaggregations', 'fabric-dashboard', 'fabric-settings',
        'reports-library', 'reports-builder', 'reports-queries', 'reports-models',
        'bundle-explorer', 'views', 'business-objects', 'admin-llm', 'admin-temporal-ops',
        'admin-entitlements', 'admin-audit', 'tenants-management', 'security-access-rules',
        'access-explanation', 'jit-requests', 'secrets-config'
    );
END $$;
