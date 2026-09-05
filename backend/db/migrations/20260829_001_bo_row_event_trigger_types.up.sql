-- Seeds the BO row-event vocabulary into the existing trigger_types table so that
-- BOCRUDHandler's row_insert/row_update/row_delete event emissions (bo_crud_handler.go)
-- have matching trigger_types.key rows for validation_triggers to reference.
INSERT INTO trigger_types (id, key, label, description, category)
VALUES
    (gen_random_uuid(), 'row_insert', 'Row Inserted', 'Fires after a Business Object record is created.', 'bo_lifecycle'),
    (gen_random_uuid(), 'row_update', 'Row Updated', 'Fires after a Business Object record is updated.', 'bo_lifecycle'),
    (gen_random_uuid(), 'row_delete', 'Row Deleted', 'Fires after a Business Object record is deleted.', 'bo_lifecycle')
ON CONFLICT (key) DO NOTHING;
