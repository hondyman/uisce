Deploy to remote Tailscale host (Trino example)

This document explains how to deploy the repository to your remote Tailscale host (e.g., 100.84.126.19) and start Trino using `docker-compose.remote.yml`.

Options
- Manual via `scripts/deploy_remote.sh` (run locally; accepts host/user/path as args)
- Automated via GitHub Actions `Deploy Remote` workflow (`.github/workflows/deploy-remote.yml`) — recommended because it uses a private key in repository Secrets and is auditable.

Secrets required for GitHub Action
- REMOTE_SSH_HOST — remote host (Tailscale IP or DNS)
- REMOTE_SSH_USER — remote SSH user (default: `ubuntu`)
- REMOTE_SSH_PORT — SSH port (default: `22`)
- REMOTE_SSH_PRIVATE_KEY — PEM-format private key (add as secret)
- REMOTE_SSH_PATH — path on remote where repository is checked out (default: `~/semlayer`)

How to use the manual scripts

Key-based (recommended):
1. Make the script executable: `chmod +x scripts/deploy_remote.sh`
2. Run it locally (it will SSH and run commands on the remote):

   ./scripts/deploy_remote.sh 100.84.126.19 ubuntu ~/semlayer 22

Password-based (less secure):
1. Ensure `sshpass` is installed (Linux: `sudo apt-get install -y sshpass`; macOS: use `brew`).
2. Make the password script executable: `chmod +x scripts/deploy_remote_password.sh`
3. Run it locally (supply host and password):

   ./scripts/deploy_remote_password.sh 100.84.126.19 'MySshPassword' ubuntu ~/semlayer 22

> Security note: Storing passwords in plaintext is risky. Prefer setting `REMOTE_SSH_PASSWORD` as a repository secret and using the GitHub Action, or use SSH key authentication.

Auto-deploy & post-deploy verification
- This repository now supports automatic deployment: when deploy-related files change on `main` (paths: `trino/**`, `docker-compose.remote.yml`, `scripts/deploy*`, `backend/migrations/**`), the `Auto Deploy on Main` workflow will automatically dispatch the `Deploy Remote` workflow.
- After a successful deploy the `Deploy Remote` workflow will automatically dispatch the `Verify Snapshot Backfill` workflow when Trino credentials are configured.
- For auto-deploy to run, ensure the necessary deploy secrets are set (see earlier section). If you prefer manual control, disable the `Auto Deploy on Main` workflow in Actions settings.

How to use the workflow
1. Add required secrets in GitHub: Settings → Secrets → Actions.
2. Go to Actions → Deploy Remote → Run workflow → select `main` (or desired branch) and click `Run workflow`.

Security notes
- Prefer SSH key auth and use the GitHub Action (do not paste passwords into issues or PR comments).
- If you must use a password, run the `scripts/deploy_remote.sh` locally (I can add a small `sshpass` example if you insist, but it is not recommended).

If you want, I can also:
- Add extra steps to the deploy script to restart other services, run migrations, or trigger the `verify-snapshot-backfill` workflow after deploy.
- Add a health-check step that validates snapshots and Trino reachability and then notifies a Slack/Teams webhook.
