Profiler E2E test (ephemeral Postgres)

This test runs an ephemeral Postgres container and exercises the profiler end-to-end.

How to run

- Ensure Docker is running on the machine.
- From the repository root, run:

```bash
# Opt-in to running the slow docker-backed E2E test
RUN_PROFILER_E2E=1 go test ./backend/internal/api -run TestProfilerE2E -v
```

Notes

- The test will pull the `postgres:15-alpine` image if not already available.
- It creates a `sml.column_profiles` table and a wide test table to make batching matter.
- The test can be slow due to container startup; run selectively in CI or locally as needed.
