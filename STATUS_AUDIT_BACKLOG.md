# STATUS_AUDIT_BACKLOG.md — Open Findings

## Finding: ColumnMapper-Delete-On-Rename

**Name:** `ColumnMapper-Delete-On-Rename`
**Severity:** Medium (silent data-loss footgun)
**Found:** 2026-09-04
**Status:** Open

### Description

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name. Given `Mappings: {"out_name": "node_name"}`:

1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — **deletes** `node_name`

Any downstream node that expects both `out_name` AND `node_name` receives only `out_name`. The data loss is silent — no error, no warning — because the value was successfully mapped to its target field.

### Production Impact

All pipeline authors configuring `column_mapper` nodes with rename-style mappings (e.g. `{"external_id": "id"}`, `{"out_name": "node_name"}`) will silently lose the source field. Any loader or subsequent transform that reads the original field name gets an empty value.

### Workaround in Tests

The diamond persistence test (`diamond_persistence_test.go`) uses self-referential mappings to avoid this behavior: `Mappings: {"node_name": "node_name", "value": "value"}`. This copies fields without deleting originals.

### Fix Direction

In `ColumnMapper.Transform`, remove the `delete(transformed, srcKey)` call. The default-copy behavior (copy all existing fields, then apply renames on top) means renamed fields are effectively renamed without losing the original — which is the behavior pipeline authors would expect. If the intent is to actually move (not copy) a field, that should be an explicit opt-in option.

### References

- `backend/internal/datapipeline/transforms.go:41-46` — the delete-on-rename logic
- `backend/internal/datapipeline/diamond_persistence_test.go` — current test workaround
