## Problem

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name:

Given `Mappings: {"out_name": "node_name"}`:
1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — deletes `node_name`

Any downstream node expecting both `out_name` AND `node_name` receives only `out_name`. The data loss is silent.

## Production Impact

Every pipeline author configuring `column_mapper` with rename-style mappings will silently lose the source field. Downstream loaders and transforms get empty values. Severity is MEDIUM: annoying, discoverable in testing, recoverable by re-running with fixed mappings — not the same class as tenant isolation voids or transactional event gaps.

## Fix Direction

Remove `delete(transformed, srcKey)`. Default-copy behavior (copy all fields, then apply renames) gives rename-without-loss semantics that pipeline authors would expect. If move semantics are genuinely needed, make it an explicit opt-in option.

## Acceptance Criteria

- [ ] Rename mapping copies source field without deleting it
- [ ] Downstream nodes receive both source and target fields
- [ ] Explicit move semantics available as opt-in (if implemented)
- [ ] diamond_persistence_test.go continues to pass (already uses self-referential mappings)
- [ ] go test -count=1 ./internal/datapipeline/... passes

Ref: STATUS_AUDIT_BACKLOG.md → `ColumnMapper-Delete-On-Rename`
