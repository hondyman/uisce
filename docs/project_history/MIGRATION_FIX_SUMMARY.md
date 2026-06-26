# Migration Fix Summary

## Issue
The database migration script `db/migrations/20251104_add_metric_registry_notify_trigger.sql` was referencing the wrong table name, causing deployment to fail.

**Errors Encountered**:
1. `ERROR: relation "metric_registry" does not exist` 
2. `ERROR: column "description" of relation "schema_migrations" does not exist`

## Root Cause
- The actual database table is `metrics_registry` (plural), not `metric_registry` (singular)
- The migration script was written with incorrect table names
- The migration also attempted to insert logging into `schema_migrations` which doesn't have a "description" column

## Solution Applied

### 1. Fixed Migration Script
**File**: `db/migrations/20251104_add_metric_registry_notify_trigger.sql`

**Changes**:
- Updated all references from `metric_registry` to `metrics_registry`
- Updated function name from `notify_metric_registry_changed()` to `notify_metrics_registry_changed()`
- Updated trigger name from `metric_registry_notify_trigger` to `metrics_registry_notify_trigger`
- Updated notification channel from `metric_registry_changed` to `metrics_registry_changed`
- Removed the problematic `INSERT INTO schema_migrations` statement
- Simplified the notification payload to use actual columns from `metrics_registry` table (node_id, schema_domain)

### 2. Fixed Semantic Sync Service
**File**: `services/semantic-sync/main.go`

**Changes**:
- Updated listener channel from `metric_registry_changed` to `metrics_registry_changed`
- Updated SQL query from `FROM metric_registry` to `FROM metrics_registry`
- Updated log message to reference correct channel name

## Verification Results

✅ **Migration execution**: SUCCESS
```
CREATE FUNCTION
NOTICE: trigger "metrics_registry_notify_trigger" for relation "metrics_registry" does not exist, skipping  DROP TRIGGER
CREATE TRIGGER
```

✅ **Trigger creation verified**:
```sql
SELECT tgname FROM pg_trigger WHERE tgname = 'metrics_registry_notify_trigger';
-- Result: metrics_registry_notify_trigger (1 row)
```

✅ **Trigger definition verified**:
```
CREATE TRIGGER metrics_registry_notify_trigger AFTER INSERT OR DELETE OR UPDATE ON public.metrics_registry 
FOR EACH ROW EXECUTE FUNCTION notify_metrics_registry_changed()
```

## Database Schema Reference

The `metrics_registry` table has the following structure:
- `id` (int) - Primary key
- `node_id` (varchar) - Node identifier
- `schema_domain` (varchar) - Schema domain
- `category` (varchar)
- `description` (text)
- `formula_type` (varchar)
- `formula` (text)
- `arguments` (jsonb)
- `badge` (varchar)
- `function_class` (varchar)
- `functions_used` (text[])
- `governance_status` (varchar)
- `audience` (text[])
- `tags` (text[])

## Next Steps

1. ✅ Migration has been successfully executed
2. ✅ Trigger is active and listening for changes
3. 🔄 Next: Start the Semantic Sync service via docker-compose
4. 🔄 Verify service connects and begins listening to the `metrics_registry_changed` channel
5. 🔄 Test end-to-end: Create/update metric in UI → Trigger fires → Schema regenerates

## Testing the Event Flow

To manually verify the trigger works:

```bash
# In one terminal, connect and listen for notifications:
psql postgres://postgres:postgres@localhost:5432/alpha
> LISTEN metrics_registry_changed;

# In another terminal, trigger a change:
UPDATE metrics_registry SET category = 'test' WHERE id = 1;

# You should see a notification appear in the LISTEN terminal
```

## Service Dependencies

- **Semantic Sync** listens to the `metrics_registry_changed` channel
- When metrics change, the trigger fires and sends notifications
- Semantic Sync receives these notifications and regenerates Cube.js schemas
- Schemas are written to `./cube-schemas/` directory
- Service also runs periodic refresh every 1 hour as fallback

