Integration tests

Run the lightweight integration checks locally. They spin up a Postgres test DB, build the backend, start it against the test DB, and perform a few HTTP requests to validate API behavior (mainly presence of structured error payloads).

Usage:

  ./scripts/run_integration_tests.sh

Requirements:
- Docker and docker-compose available on PATH
- Go toolchain for building the backend

This is intentionally simple and designed to be run locally or in CI with Docker support.
