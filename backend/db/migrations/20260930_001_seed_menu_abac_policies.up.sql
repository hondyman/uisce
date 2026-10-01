-- Seed default menu:* ABAC policies for the Ivy platform navigation.
-- tenant_id = NULL means gold-copy baseline — inherited by all tenants.
--
-- Key contract — these strings MUST stay in sync with the requiredCapability
-- values declared in frontend/src/components/MainNavigation.tsx:
--   menu:platform      top-level Platform category gate
--   menu:organization  Platform -> Organization submenu gate
--   menu:security      Platform -> Security submenu gate
--   menu:system        Platform -> System submenu gate
--   menu:entitlements  Platform -> Organization -> Entitlement Management gate
--
-- The other five categories (Catalog, Build, Operations, Intelligence,
-- Consume) declare no requiredCapability in the frontend, so they need no
-- policy rows: filterNavigationByCapabilities treats an absent key as "no
-- gate" only when the prop itself is undefined, and those categories leave
-- it undefined. Add rows here if they are ever gated.
--
-- Deny is written at a higher priority than any tenant allow so that a
-- tenant-level override cannot silently re-grant the Platform category to a
-- base user. CapabilitiesHandler merges rows with "deny wins at equal
-- priority", so an equal-priority tenant allow would otherwise tie.

INSERT INTO studio.tenant_abac_policies
  (policy_id, tenant_id, target_profile_key, name, description, effect, priority, enabled, action_attribute)
VALUES
  -- BASE_USER: Platform category is operator-only.
  (gen_random_uuid(), NULL, 'BASE_USER', 'Deny Platform menu', 'Platform category is operator-only', 'deny', 200, true, 'menu:platform'),

  -- PLATFORM_OPERATOR: full Platform access.
  (gen_random_uuid(), NULL, 'PLATFORM_OPERATOR', 'Allow Platform menu', 'Operator access to Platform category', 'allow', 100, true, 'menu:platform'),
  (gen_random_uuid(), NULL, 'PLATFORM_OPERATOR', 'Allow Organization menu', 'Operator access to Org management', 'allow', 100, true, 'menu:organization'),
  (gen_random_uuid(), NULL, 'PLATFORM_OPERATOR', 'Allow Security menu', 'Operator access to Security management', 'allow', 100, true, 'menu:security'),
  (gen_random_uuid(), NULL, 'PLATFORM_OPERATOR', 'Allow System menu', 'Operator access to System admin', 'allow', 100, true, 'menu:system'),
  (gen_random_uuid(), NULL, 'PLATFORM_OPERATOR', 'Allow Entitlement Management', 'Operator access to tenant entitlements', 'allow', 100, true, 'menu:entitlements')

-- uq_tenant_abac_policies_global is a partial unique index
-- (target_profile_key, action_attribute) WHERE tenant_id IS NULL, so the
-- conflict target has to carry the same predicate for inference to work.
ON CONFLICT (target_profile_key, action_attribute) WHERE tenant_id IS NULL DO NOTHING;
