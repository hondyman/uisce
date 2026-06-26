Local environment and LLM key handling (development)

Purpose
- Keep secrets out of version control while making them easy to load locally for development.

Files we added/updated
- `.env.local` (your local secrets file) — DO NOT commit this file.
- `.env.local.template` — template without secrets to guide teammates.
- `scripts/load-local-env.sh` — helper to `source` `.env.local` into your shell.
- `.gitignore` updated to ignore `.env.local`.

Quick start (local dev)
1. Copy template to `.env.local` and add your keys:

```bash
cp .env.local.template .env.local
# edit .env.local and set GEMINI_API_KEY and DATABASE_URL
```

2. Load the local env into your current shell session:

```bash
source scripts/load-local-env.sh
# now GEMINI_API_KEY, DATABASE_URL, etc are exported in your shell
```

3. Run dev commands that need the key (examples):

```bash
# run the embedding generator (python script)
/Users/$(whoami)/GitHub/semlayer/.venv/bin/python scripts/generate_embeddings_py.py \
  --tenant=<TENANT_ID> --datasource=<DATASOURCE_ID> --db="$DATABASE_URL" --api-key="$GEMINI_API_KEY"

# or start backend with env loaded
source scripts/load-local-env.sh && make run-backend
```

Security notes (dev)
- Do NOT commit `.env.local`. We removed it from the repository and added it to `.gitignore`.
- If you suspect an exposure, rotate the key in the provider console immediately.
- For production, use a secrets manager (Vault / AWS Secrets Manager / GCP Secret Manager) and inject secrets at runtime.

Optional improvements
- Use OS Keychain (macOS Keychain) or a local secret store for improved local security.
- Add a small wrapper script or systemd unit to load secrets when starting services in dev VMs.

Contact
- If you want, I can implement secret manager integration or a macOS keychain helper next.
