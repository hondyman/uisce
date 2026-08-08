# UTC Timezone Compliance Policy

## Overview

This document defines the UTC timezone compliance requirements for the platform to ensure consistent datetime handling across all services and the database.

## Requirements

### 1. Database (PostgreSQL)

- [x] `postgresql.conf` must set `timezone = 'UTC'`
- [x] All timestamp columns must use `TIMESTAMPTZ` (not bare `TIMESTAMP`)
- [x] DSN connections must include `TimeZone=UTC`
- [x] Migration runner sets `SET TIME ZONE 'UTC'` on init

### 2. Schema Files

- All schema definitions must use `TIMESTAMPTZ` for timestamp columns
- Bare `TIMESTAMP` type is **prohibited** in CREATE TABLE statements
- Example:
  ```sql
  -- ❌ WRONG
  created_at timestamp DEFAULT now()

  -- ✅ CORRECT
  created_at timestamptz DEFAULT now()
  ```

### 3. Application Code (Go)

- All `time.Time` values must be in UTC when inserting to the database
- Use `time.Now().UTC()` instead of `time.Now()`
- Example:
  ```go
  // ❌ WRONG
  model.CreatedAt = time.Now()

  // ✅ CORRECT
  model.CreatedAt = time.Now().UTC()
  ```

### 4. Docker Services

All containerized services must be configured with `TZ=UTC`:

| Service | Environment Variable |
|---------|---------------------|
| Temporal | `TIMEZONE=UTC` |
| Keycloak | `JAVA_OPTS=-Duser.timezone=UTC` |
| Lakekeeper | `TZ=UTC` |
| DataFusion | `TZ=UTC` |
| MinIO | `TZ=UTC` |
| StarRocks | `TZ=UTC` |
| Nessie | `JAVA_OPTS=-Duser.timezone=UTC`, `TZ=UTC` |

## Enforcement Mechanisms

### 1. Migration Runner Validation (Primary)

The migration runner (`migrations/migration_runner.go`) validates all migrations before applying:

- Checks for bare `TIMESTAMP` column definitions
- Checks for `ALTER TABLE ... TYPE TIMESTAMP` (not timestamptz)
- **Rejects** non-compliant migrations

### 2. Pre-commit Hook (Optional)

File: `scripts/check_migration_utc.go`

Run manually or in CI:
```bash
go run scripts/check_migration_utc.go migrations/*.sql
```

### 3. Schema Lint Script

File: `scripts/check_utc_compliance.sh`

Validates all schema files:
```bash
./scripts/check_utc_compliance.sh
```

## Migration Checklist

When creating a new migration:

1. Use `TIMESTAMPTZ` for all timestamp columns
2. Use `ALTER TABLE ... ALTER COLUMN ... TYPE TIMESTAMPTZ` (not bare TIMESTAMP)
3. Test migration locally before committing
4. Run UTC compliance check: `go run scripts/check_migration_utc.go <your_migration>.sql`

## Troubleshooting

### "bare TIMESTAMP detected" error in migration

Your migration contains a bare `TIMESTAMP` column definition. Fix by changing to `TIMESTAMPTZ`:

```sql
-- Before
created_at timestamp DEFAULT now()

-- After
created_at timestamptz DEFAULT now()
```

### Services showing wrong timezone

Ensure `TZ=UTC` is set in the service's environment or docker-compose configuration.
