## What this PR does

Describe the change and why it is needed.

---

### Trino backfill verification notes
If this PR contains the pre-agg / snapshot migration `backend/migrations/20260207_add_region_to_snapshots_and_preaggs.up.sql`, the repository runs an automated verification workflow that validates the Postgres backfill and (optionally) verifies Trino Iceberg snapshot regions.

Before opening the PR, ensure one of the following is configured in repository secrets (Settings → Secrets → Actions):

- Preferred: set `TRINO_HOST` (Tailscale IP or hostname reachable by the GitHub Actions runner) and `TRINO_USER`/`TRINO_PASSWORD`.
- Fallback: if `TRINO_HOST` is not reachable by the runner, provide an SSH tunnel configuration: `SSH_HOST`, `SSH_USER`, `SSH_PRIVATE_KEY`, `REMOTE_TRINO_HOST`, `REMOTE_TRINO_PORT`, plus `TRINO_USER` and `TRINO_PASSWORD`.

If TRINO credentials are set but no connectivity info is provided, the verification workflow will fail and post a comment on the PR explaining how to fix it.

See `docs/TRINO_BACKFILL_README.md` for detailed instructions.