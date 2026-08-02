# Docker Init Scripts

This directory contains initialization scripts used by `docker-compose.production.yml`.

## How It Works

`docker-compose.production.yml` mounts `./backend/db/migrations` as a PostgreSQL
`docker-entrypoint-initdb.d` volume. On first boot, PostgreSQL executes all
`*.sql` and `*.up.sql` files in alphabetical order before the database is
ready to accept connections.

## Canonical Migration Path

```
backend/db/migrations/          ← canonical (active development)
backend/migrations/             ← legacy / archival (not mounted)
```

All new enterprise features (survivorship rules, maker-checker governance,
shadow replay, pg_trgm) target `backend/db/migrations/`.

## Init Script Order

| File | Purpose |
|------|---------|
| `001_create_metadata_tables.sql` | Core tenant, ABAC, schema tables |
| `20260731_pg_trgm.up.sql` | Trigram extension for fuzzy field-name matching |
| `20260802_shadow_replay.up.sql` | Shadow replay jobs + diff tables |
| `20260731_survivorship_rules.up.sql` | Survivorship resolution rules |
| `20260731_semantic_term_tags.up.sql` | Semantic term taxonomy tags |

## Adding a New Migration

1. Create `backend/db/migrations/YYYYMMDD_descriptive_name.up.sql`
2. Include a corresponding `YYYYMMDD_descriptive_name.down.sql` for rollback
3. Test with: `docker compose -f docker-compose.production.yml down -v && docker compose -f docker-compose.production.yml up --build -d`

## Note on Local Development

`docker-compose.backend.yml` uses a different initialization pattern
(`hasura/graphql-engine` migrations via `hasura-cli migrate apply`).
Do not mix migration paths — use `backend/db/migrations/` for backend-native
schema changes.
