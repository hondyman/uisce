# Trino (Iceberg) Backfill Verification — Setup & Run

This document explains how to configure and run the Trino (Iceberg) backfill verification workflows/tests for snapshot `region` backfill.

Prereqs
- A Trino instance reachable from the GitHub Actions runner (directly via Tailscale is preferred). If your Trino host is on the same Tailscale network and its Tailscale IP is reachable from the runner, set `TRINO_HOST` to that IP and you do not need an SSH tunnel.
- If you run Trino on your remote Tailscale server, you can add it to `docker-compose.remote.yml` (example service is included in this repository) and expose the coordinator HTTP port so the runner can connect.
- Repository secrets (See below) stored in GitHub Actions Secrets (Settings → Secrets → Actions).

Required repository secrets (minimum, direct access)
- TRINO_HOST — Trino host (IP or DNS reachable by runner)
- TRINO_PORT — (optional; default 8080)
- TRINO_USER
- TRINO_PASSWORD
- TRINO_CATALOG — (optional; default `iceberg`)
- TRINO_SCHEMA — (optional; default `audit`)

Optional (specific snapshot checks)
- TRINO_TEST_SNAP_IDS — comma-separated snapshot ids to validate (e.g. `snap-001,test-snap-01`)
- TRINO_EXPECTED_REGIONS — comma-separated expected regions matching `TRINO_TEST_SNAP_IDS` (e.g. `eu-west,us-east`)

SSH tunnel — fallback (only if direct Tailscale access is not possible)
- We *prefer* direct Tailscale access (set `TRINO_HOST` to the Tailscale IP); use an SSH tunnel only as a fallback if the runner cannot reach the Trino host directly. If you must use an SSH tunnel, set the following secrets:
  - SSH_HOST — public host for SSH (bastion)
  - SSH_USER — user for SSH
  - SSH_PRIVATE_KEY — private key (PEM format)
  - SSH_PORT — (optional, default 22)
  - REMOTE_TRINO_HOST — Trino host as seen from the SSH bastion
  - REMOTE_TRINO_PORT — Trino port as seen from the SSH bastion

How the workflow runs
- Workflow: `.github/workflows/verify-snapshot-backfill.yml`
- It runs the Postgres backfill verification then (optionally) runs the Trino backfill integration test if Trino secrets or SSH secrets are present.
- If `TRINO_TEST_SNAP_IDS` and `TRINO_EXPECTED_REGIONS` are provided, the test checks specific snapshot rows and verifies expected region values and idempotency.

Manual steps to trigger verification (GH CLI)
1. Ensure secrets are added.
2. Use GitHub CLI from your machine:

```bash
# Trigger workflow on the default branch (or set ref to the branch you want)
gh workflow run verify-snapshot-backfill.yml -f ref=main
```

Local / in-network manual test (run from a machine inside Tailscale that can reach Trino)

This repository includes a small example Trino catalog at `trino/catalog/iceberg.properties` and a sample `trino` service you can add to `docker-compose.remote.yml`.

If you run Trino in `docker-compose.remote.yml` the example maps container port `8080` to host port `8084` (`8084:8080`). When using that compose, set `TRINO_HOST` to the remote host's Tailscale IP and `TRINO_PORT` to `8084`.

```bash
export TRINO_HOST=100.84.126.19
export TRINO_PORT=8084
export TRINO_USER=trino_user
export TRINO_PASSWORD=secret
export TRINO_CATALOG=iceberg
export TRINO_SCHEMA=audit
# Optional snapshot tests
export TRINO_TEST_SNAP_IDS=test-snap-01
export TRINO_EXPECTED_REGIONS=eu-west

# Run the Go test (requires repo Go env that matches go.work requirements):
INTEGRATION_TEST=1 go test ./backend/internal/audit -run TestTrinoSnapshotBackfillIntegration -v
```

Connectivity check helper (optional)
- See `scripts/check_trino_connectivity.sh` — a lightweight check using curl to confirm the Trino HTTP endpoint responds.

If you want, I can add a self-hosted runner example (for an internal machine) or an SSH tunneling example that uses Autossh and a GitHub Actions secret to spin up the tunnel automatically during the workflow. Let me know which you prefer.
