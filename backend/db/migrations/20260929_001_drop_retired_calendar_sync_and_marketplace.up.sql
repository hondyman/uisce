-- 20260929_001_drop_retired_calendar_sync_and_marketplace.up.sql
-- Remove database objects orphaned by the retirement of two features.
--
-- CALENDAR SYNC (personal Google/Microsoft/Apple calendar integration)
--   The Go code that owned the `calendar` schema was deleted:
--     internal/google, internal/microsoft, internal/apple (providers)
--     internal/sync/{processor,provider,conflict_detector,event_mapper,
--                    event_listener,metrics,recurring_events,timezone_service}
--     internal/api/{sync_handler,conflict_handler}.go
--   The surviving business calendar (internal/calendar) does NOT use this
--   schema: it reads public.business_calendars, public.calendar_holidays and
--   mdm.calendar_hierarchy. The scheduler reads public.tenant_calendar_holidays,
--   public.tenant_exchange_calendars and mdm.calendar_*.
--
--   Every table in the schema was verified empty (0 rows) and no Go code,
--   foreign key from another schema, or cross-schema view referenced it.
--
-- MARKETPLACE (component/customization marketplace)
--   The tables backing the removed component marketplace are unused. The
--   tables backing the RETAINED integration marketplace
--   (marketplace_integrations, installed_integrations, integration_executions)
--   are deliberately NOT touched.
--
--   marketplace_integration_settings is deliberately RETAINED: the live table
--   installed_integrations has a foreign key pointing at it, so dropping it
--   would silently destroy that live table's referential integrity.
--
-- Deliberately NO CASCADE on the marketplace tables: children are dropped
-- explicitly first so that an unexpected dependency aborts the transaction
-- instead of silently cascading into a retained table.

-- ---------------------------------------------------------------------------
-- 1. Component marketplace tables (children before parents)
-- ---------------------------------------------------------------------------
DROP TABLE IF EXISTS marketplace_item_feedback;
DROP TABLE IF EXISTS marketplace_item_parameters;
DROP TABLE IF EXISTS marketplace_item_usage;
DROP TABLE IF EXISTS marketplace_item_versions;
DROP TABLE IF EXISTS tenant_marketplace_items;
DROP TABLE IF EXISTS marketplace_artifacts;
DROP TABLE IF EXISTS marketplace_installs;
DROP TABLE IF EXISTS marketplace_items;
DROP TABLE IF EXISTS marketplace_listings;

-- ---------------------------------------------------------------------------
-- 2. Re-home the one trigger that lives OUTSIDE the schema
--
--    public.user_settings has a BEFORE UPDATE trigger that calls
--    calendar.update_updated_at_column(). Dropping the schema with CASCADE
--    would silently delete that trigger from a live public table.
--    public.update_updated_at_column() already exists with a byte-identical
--    body, so the trigger is re-pointed at it first, preserving behaviour.
-- ---------------------------------------------------------------------------
DROP TRIGGER IF EXISTS update_user_settings_updated_at ON public.user_settings;

CREATE TRIGGER update_user_settings_updated_at
    BEFORE UPDATE ON public.user_settings
    FOR EACH ROW
    EXECUTE FUNCTION public.update_updated_at_column();

-- ---------------------------------------------------------------------------
-- 3. Personal calendar sync schema
--    CASCADE is required to remove its 3 views, 2 functions, 68 triggers and
--    the quarterly partitions. Verified beforehand that nothing outside the
--    schema depends on any of its objects.
-- ---------------------------------------------------------------------------
DROP SCHEMA IF EXISTS calendar CASCADE;
