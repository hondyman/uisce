# Generated Cube Artifacts

Artifacts in this directory are produced by the tenant automation CLI (`go run ./cmd/tenant_automation`).

- `tenant-scopes.json` — snapshot of tenant/datasource metadata powering Cube scheduled refresh contexts and QoS routing.
- Additional json/sql files may be emitted to describe resource groups or other infra hints.

Files here are safe to regenerate at any time and should not be edited by hand.
