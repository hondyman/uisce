DROP TABLE IF EXISTS data_explorer.saved_query_schedule;

ALTER TABLE data_explorer.saved_query
    DROP COLUMN IF EXISTS folder_id,
    DROP COLUMN IF EXISTS is_favorite,
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS is_core,
    DROP COLUMN IF EXISTS related_bo_ids,
    DROP COLUMN IF EXISTS created_by;

DROP TABLE IF EXISTS data_explorer.saved_query_folder;
