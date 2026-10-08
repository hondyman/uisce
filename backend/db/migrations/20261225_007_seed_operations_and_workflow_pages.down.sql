-- Rollback migration 007
DO $$
DECLARE
    gold_tenant UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
BEGIN
    DELETE FROM route_aliases WHERE tenant_id = gold_tenant AND page_key IN (
        'core-process-catalog', 'core-approval-inbox', 'core-approval-workflows',
        'client-workflow-studio', 'core-workflow-designer', 'client-rules-editor',
        'core-notifications', 'core-notification-templates', 'core-notification-preferences',
        'core-sla-dashboard', 'core-flow-builder', 'core-validation-rules',
        'core-calculated-fields', 'core-validation', 'bp-console',
        'bp-console-instances', 'bp-console-queues', 'governance-changesets'
    );

    DELETE FROM page_definitions WHERE tenant_id = gold_tenant AND page_key IN (
        'core-process-catalog', 'core-approval-inbox', 'core-approval-workflows',
        'client-workflow-studio', 'core-workflow-designer', 'client-rules-editor',
        'core-notifications', 'core-notification-templates', 'core-notification-preferences',
        'core-sla-dashboard', 'core-flow-builder', 'core-validation-rules',
        'core-calculated-fields', 'core-validation', 'bp-console',
        'bp-console-instances', 'bp-console-queues', 'governance-changesets'
    );
END $$;
