# Tenant-Specific Cube Schemas

Automation writes tenant/datasource overrides here using the Phase 5 provisioning job (`go run ./cmd/tenant_automation`). See `docs/TENANT_AUTOMATION_RUNBOOK.md` for operational details.

Layout:

```
<tenant-id>/
  <datasource-id>/
    auto/              # generated files (safe to overwrite)
    manual/            # optional developer-authored overrides
```

Only the `auto` subtree is managed by the sync script; anything outside of it is left untouched so that manual overrides can coexist with generated content.
