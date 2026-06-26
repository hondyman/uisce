Developer notes: backend/internal/api Routes pattern & tests

This repository uses a small convention inside `backend/internal/api` to keep HTTP route
registration modular and testable. Key points:

- A `Routes` helper (see `backend/internal/api/routes.go`) centralizes grouped
  registration helpers like `RegisterBundles`, `RegisterPolicies`, `RegisterViews`, etc.
- Individual route groups live in their own files (for example `bundles_routes.go`,
  `policies_routes.go`, `roles_routes.go`) and expose `RegisterRoutes(r chi.Router)`
  methods which the `Routes` helper calls from `SetupRouter`.
- Tests for the registration wrappers live next to the route files (e.g.
  `bundles_routes_test.go`) and assert the registration functions don't panic and
  wire paths correctly.

Quick commands for developers:

```bash
# Run only the api package tests (fast)
go test ./backend/internal/api -v

# Run all repository tests (slower)
go test ./... -v
```

If you see duplicate type redeclaration errors during `go test ./...`, search
for small DTOs (Request/Response types) duplicated across files in
`backend/internal/api` and consolidate them to a single small file (for
example `types.go`, `governance_types.go`, `profiler_types.go`).
