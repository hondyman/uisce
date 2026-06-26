# Missing Dockerfiles and Recent Build Errors

This file summarizes missing Dockerfiles and build-time errors observed when attempting to start the compose stack.

## Missing/Skipped Dockerfiles (detected by launch script)
- `./services/fabric-builder/Dockerfile` - skipped
- `./services/wealth-management/Dockerfile` - skipped
- `./backend/Dockerfile.rule-engine` - missing
- `./backend/Dockerfile.notifications` - missing
- `./backend/Dockerfile.policy` - missing
- `./backend/Dockerfile.search` - missing
- `./backend/Dockerfile.validation` - missing (note: backend has `./backend/Dockerfile` present)
- `./backend/Dockerfile.event-router` - missing
- `./services/event-router/Dockerfile` - missing

> These were detected during `./docker-start.sh backend` runs where the script prints warnings and skips builds when Dockerfiles are not found.

## Observed Build Failures

During earlier runs, the following build errors were captured from the Docker build logs:

1. go.mod requires go >= 1.24.0 (running go 1.21.13)
   - Affected services: `compliance-engine`, `governance`, `ai-builder` (they used `golang:1.21-alpine` images)
   - Action taken: Bumped those services' Dockerfiles to `golang:1.25-alpine`.

2. `semantic-engine` required go >= 1.25.3; base image used `golang:1.21-alpine`.
   - Action taken: Bumped `services/semantic-engine/Dockerfile` to `golang:1.25-alpine`.

3. Private image pull failure: `xai-org/grok-beta` failed with `pull access denied`.
   - Action taken earlier: added a lightweight `ai-service` override in `docker-compose.override.yml` for development to avoid private image pulls.

4. Port conflict: when attempting to start `postgres` container while a host Postgres is running, Docker reported `ports are not available: ... bind: address already in use`.
   - Action taken: Added `USE_LOCAL_POSTGRES` support to `docker-start.sh` to prevent starting `postgres` when using a host DB.

## Recommended Next Steps

1. Restore or add the missing Dockerfiles listed above. If those microservices have moved to new locations, update `docker-compose.yml` to point to the correct build context / dockerfile paths.

2. If you prefer to run everything in containers, ensure your host does not have services (like Postgres) bound to the same ports or change compose ports.

3. Consider standardizing all Go-based service Dockerfiles to use a single pinned Go version (we updated several to `golang:1.25-alpine`).

4. For private images (like `xai-org/grok-beta`), either login to the registry before `docker-compose up` or document how to supply an alternate image for dev.

5. If you want, I can open PR-style patches to add minimal Dockerfile stubs for the missing services so the compose build won't fail and you can incrementally implement them.

---
Generated automatically by the docker-start helper on `$(date)`.
