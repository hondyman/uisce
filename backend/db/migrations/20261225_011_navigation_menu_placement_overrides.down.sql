-- Rollback migration 20261225_011: drops the per-tenant menu visibility overrides.
-- Core entries reappear for every tenant that had hidden them.
DROP TABLE IF EXISTS navigation_menu_placement_overrides;
