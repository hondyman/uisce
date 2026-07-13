# Secrets Migration Specification

## Overview

This document tracks the migration from hardcoded/environment-variable secrets to the centralized secrets management system using Infisical.

## Status: PHASE 2 COMPLETE ✅

### Completed Items

- [x] **InfisicalProvider Implementation** (`internal/secrets/infisical.go`)
  - **Centralized via REST API** - calls Infisical API directly (not env-injection)
  - Supports service token (`INFISICAL_TOKEN`) for CI/CD
  - Supports client credentials (`INFISICAL_CLIENT_ID` + `INFISICAL_CLIENT_SECRET`) for service-to-service auth
  - Built-in caching (5 min TTL) to reduce API calls
  - Auto re-authentication on token expiry
  - Implements `Get`, `GetMap`, `Put`, `Delete`, `List`, `Health`

- [x] **Provider Factory** (`internal/secrets/provider.go`)
  - Added `InfisicalClientID`, `InfisicalClientSecret`, `InfisicalProjectID`, `InfisicalEnvironment` config fields
  - Added `GetSecretOrFail()` for fail-fast mandatory secret resolution
  - Factory detects provider type from config

- [x] **Fail-Fast Bootstrap** (`internal/api/server.go`)
  - `initSecretsProvider()` called during server initialization
  - When `config.Secrets.InfisicalToken` is set, secrets are resolved via API
  - Missing required secrets cause immediate fatal error (no silent degradation)

- [x] **JWT Secret Migration** (`internal/api/ws_tokens.go`, `internal/api/api.go`)
  - JWT secret now retrieved via `secProvider.Get("JWT_SECRET")`
  - Config loaded from `config.Secrets.InfisicalToken` etc.

- [x] **Impersonation Flow** (`internal/security/impersonation.go`)
  - Added `secProvider` field to `ContextExchangeService`
  - Exported `ImpersonationTokenPayload` for middleware access
  - Added `NewContextExchangeServiceWithProvider(provider)` for stateless token validation
  - Methodized `ValidateImpersonationToken()` for testability

- [x] **Auth Middleware Integration** (`internal/middleware/auth_context.go`)
  - `AuthContextMiddleware(secMgr, profileSvc, secProvider)` signature
  - Provider-aware validation when `secProvider != nil`

- [x] **Handler Updates** (`internal/handlers/admin_impersonate.go`)
  - Constructor accepts `secProvider secrets.Provider`
  - Wired through server initialization chain

### Verification Results

| Test Suite | Status |
|------------|--------|
| Contract Handler Tests | ✅ 10/10 PASS |
| OpenAPI Generator Tests | ✅ 11/11 PASS |
| Impersonation Tests | ✅ 8/8 PASS |
| Middleware Tests | ✅ ALL PASS |
| Build | ✅ PASS |

## Configuration

### Environment Variables for Infisical API

| Variable | Required | Description |
|----------|----------|-------------|
| `INFISICAL_TOKEN` | Alt to client creds | Service token for Infisical access |
| `INFISICAL_CLIENT_ID` | Alt to token | OAuth client ID (Universal Auth) |
| `INFISICAL_CLIENT_SECRET` | Alt to token | OAuth client secret |
| `INFISICAL_PROJECT_ID` | Yes | Project identifier |
| `INFISICAL_ENVIRONMENT` | No | Environment (default: development) |
| `JWT_SECRET` | No | Not used anymore (fetched from Infisical) |

### Provider Config (config.yaml)

```yaml
secrets:
  type: infisical
  infisical_token: "${INFISICAL_TOKEN}"
  # OR for client credentials:
  # infisical_client_id: "${INFISICAL_CLIENT_ID}"
  # infisical_client_secret: "${INFISICAL_CLIENT_SECRET}"
  infisical_project_id: "${INFISICAL_PROJECT_ID}"
  infisical_environment: "${INFISICAL_ENVIRONMENT:-development}"
```

### Infisical Import Format

Add to Infisical dashboard for each environment:

```env
JWT_SECRET=TND5KO7xY/Fz1ifgTR5QMm9T+R5/aPxxavmMzp+hURJxRWTm2Pns+RC+q9NKMxMB3F/R2KAWXnwo7r8N5JIACQ==
ADMIN_API_KEY=
LAUNCHDARKLY_SDK_KEY=
UNLEASH_API_KEY=
MINIO_ACCESS_KEY=
MINIO_SECRET_KEY=
GEMINI_API_KEY=
GOOGLE_GEMINI_API_KEY=
ANTHROPIC_API_KEY=
OPENAI_API_KEY=
CUBEJS_JWT_SECRET=
HASURA_ADMIN_SECRET=
TRACE_API_KEY_DEFAULT=
TRACE_API_KEY_SRE=
TRACE_API_KEYS=
OAUTH_TOKEN_ENCRYPTION_KEY=
```

## Remaining Items

### P0 - Production Hardening

- [ ] **Infisical Project Setup**
  - Create Infisical project for backend service
  - Configure environments: `development`, `staging`, `production`
  - Add secrets: `JWT_SECRET`, `ADMIN_API_KEY`, etc.

- [ ] **CI/CD Integration**
  - Configure `INFISICAL_PROJECT_ID` and `INFISICAL_ENVIRONMENT` in deployment
  - Use service token (`INFISICAL_TOKEN`) or client credentials for production

### P1 - Secret Cleanup

- [ ] **Remove Legacy Env Vars** (after migration complete)
  - Remove `JWT_SECRET` env var from deployment configs
  - Update `.env.example` with Infisical configuration instructions
  - Remove hardcoded fallbacks from code

### P2 - Monitoring & Alerting

- [ ] **Secret Rotation Alerts**
  - Monitor for secret fetch failures
  - Alert on provider fallback activation (dev mode in production)

- [ ] **Audit Logging**
  - Log secret access (not values) for compliance
  - Track which services fetch which secrets

## Migration Checklist

- [ ] Create Infisical project
- [ ] Add `JWT_SECRET` to all environments (use generated value below)
- [ ] Add `INFISICAL_PROJECT_ID` to deployment config
- [ ] Configure CI/CD with Infisical credentials
- [ ] Deploy to staging, verify secret resolution via API
- [ ] Deploy to production, verify secret resolution via API
- [ ] Remove `JWT_SECRET` from deployment configs
- [ ] Verify impersonation flow works in production

## Generated JWT_SECRET (Production Ready)

```
TND5KO7xY/Fz1ifgTR5QMm9T+R5/aPxxavmMzp+hURJxRWTm2Pns+RC+q9NKMxMB3F/R2KAWXnwo7r8N5JIACQ==
```

Add this to Infisical as `JWT_SECRET` in all environments.
