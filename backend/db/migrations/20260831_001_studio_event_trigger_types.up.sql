-- Seeds the Page Studio / API Studio event vocabulary into the existing trigger_types
-- table, mirroring the pattern used for BO row events
-- (20260829_001_bo_row_event_trigger_types.up.sql). This lets Page Studio and API
-- Studio save/submit events be evaluated by trigger_engine.go the same way
-- BOCRUDHandler's row_insert/row_update/row_delete events already are.
INSERT INTO trigger_types (id, key, label, description, category)
VALUES
    (gen_random_uuid(), 'page_load',    'Page Loaded',   'Fires when a Page Studio page finishes loading.', 'page'),
    (gen_random_uuid(), 'page_save',    'Page Saved',    'Fires after a Page Studio page record is saved.', 'page'),
    (gen_random_uuid(), 'field_change', 'Field Changed', 'Fires when a field value changes on a Page Studio page.', 'page'),
    (gen_random_uuid(), 'api_request',  'API Request',   'Fires when an API Studio endpoint receives a request.', 'api'),
    (gen_random_uuid(), 'api_response', 'API Response',  'Fires before an API Studio endpoint returns a response.', 'api')
ON CONFLICT (key) DO NOTHING;
