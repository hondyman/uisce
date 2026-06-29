-- +goose Up
-- Migration: Add baseline policies for billing specialist and executive roles
-- Created: 2026-06-29

-- Seeds for Gold Copy global ABAC policies for billing specialist
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Billing Specialist Operations',
    'Allows read, create, update on invoice, payment, and billing resources',
    'allow',
    100,
    true,
    '{"roles": ["northwind_billing_specialist"]}'::jsonb,
    '{"actions": ["read", "create", "update"]}'::jsonb,
    '{"resources": ["invoice", "payment", "billing"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Block Billing Specialist Inventory Modifications',
    'Explicitly deny create, update, delete actions on product and supplier resources for Billing Specialists',
    'deny',
    90,
    true,
    '{"roles": ["northwind_billing_specialist"]}'::jsonb,
    '{"actions": ["create", "update", "delete"]}'::jsonb,
    '{"resources": ["product", "supplier"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- Seeds for Gold Copy global ABAC policies for executive (read-only)
INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Executive Read Only Operations',
    'Allows read on all resources (orders, customers, products, suppliers, invoices, payments, billing)',
    'allow',
    100,
    true,
    '{"roles": ["northwind_executive"]}'::jsonb,
    '{"actions": ["read"]}'::jsonb,
    '{"resources": ["order", "customer", "product", "supplier", "invoice", "payment", "billing"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

INSERT INTO abac_policies (
    id, tenant_id, name, description, effect, priority, enabled,
    subject_rules, action_rules, resource_rules, environment_rules
) VALUES (
    gen_random_uuid(),
    NULL, -- Global Gold Copy
    'Global Baseline - Block Executive Modifications',
    'Explicitly deny create, update, delete actions on all resources for Executives',
    'deny',
    90,
    true,
    '{"roles": ["northwind_executive"]}'::jsonb,
    '{"actions": ["create", "update", "delete"]}'::jsonb,
    '{"resources": ["order", "customer", "product", "supplier", "invoice", "payment", "billing"]}'::jsonb,
    '{}'::jsonb
) ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM abac_policies WHERE tenant_id IS NULL AND name IN (
    'Global Baseline - Billing Specialist Operations',
    'Global Baseline - Block Billing Specialist Inventory Modifications',
    'Global Baseline - Executive Read Only Operations',
    'Global Baseline - Block Executive Modifications'
);
