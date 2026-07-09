# Security & Access Mesh — Spec-to-Implementation Mapping

This document records every deviation between the canonical Security & Access
Mesh specification and the Uisce Semantic OS implementation as of the
2026-07-09 security mesh migration (Phases 0–3). It exists so that
compliance reviewers and operators can answer "where in the code is
spec requirement X implemented?" without reverse-engineering the codebase.

The Cardinal Rules 1.3, 6.2, 7.4, 8.1, 8.3 referenced below are documented
in `AGENTS.md`.

---

## 1. Architectural Pillars

| Spec concept | Implementation | Where |
|---|---|---|
| Identity Engine (Keycloak) | Two realms: `uisce-core-ops` (operators) + `uisce` (federated customers, renamed in concept to `uisce-tenants`) | `keycloak/realm-uisce-core-ops.json`, `keycloak/realm-uisce-tenants.json` |
| Relational Datastore (Postgres) | `security` schema + `public.abac_policies` (write path) + `platform_admin_audit` / `impersonation_action_audit` (PG fast-path) | `backend/migrations/20260709_*.sql`, `backend/sql/platform_admin_audit.sql` |
| Streaming Layer (Redpanda) | 12-partition `uisce.audit.operator.v1` topic + Avro schema enforcement | `backend/audit-infrastructure/start.sh`, `backend/internal/audit/schemas/operator_impersonation_event.avsc` |
| Data Lakehouse (Apache Iceberg) | `iceberg.audit.operator_impersonation_audit` (Parquet + partitioning on `target_tenant_id`, `year_month_day`) | `backend/internal/audit/iceberg_schema.sql`, `backend/internal/audit/operator_impersonation_sink.go` |

---

## 2. Part 1 — Platform Operators (Impersonation & Audit)

### 2.1 Keycloak Configuration

| Spec requirement | Implementation | Notes |
|---|---|---|
| Realm `uisce-core-ops` (isolated from customer auth paths) | `keycloak/realm-uisce-core-ops.json` | Realm enabled, `sslRequired=external`, brute-force-protected. Customers authenticate in the separate `uisce` tenant realm. |
| Confidential client `uisce-studio-ops` | `clients[clientId=uisce-studio-ops]` in `realm-uisce-core-ops.json` | `publicClient=false`, `standardFlowEnabled=false`, `bearerOnly=false`, `serviceAccountsEnabled=true`, `oauth2.token.exchange.grant.enabled=true` (Keycloak token-exchange feature flag). |
| **Token Exchange Flow enabled** | ✅ + internal HMAC-SHA256 fallback | **Deviation:** the spec calls for RFC 8693 (Keycloak token-exchange); the Uisce implementation uses an internal HMAC-SHA256 protocol keyed by `IMPERSONATION_TOKEN_SECRET` (falls back to `JWT_SECRET`). Rationale: the internal protocol is simpler, removes a Keycloak feature-flag dependency at runtime, and gives us per-session revocation via `security.operator_leases`. The Keycloak `oauth2.token.exchange.grant.enabled=true` flag remains set so a future migration to RFC 8693 is a config-only change. |
| Hardcoded protocol mapper for `uisce_metadata.operator_role` | `clientScopes[name=uisce_metadata].protocolMappers[uisce_metadata_operator_role]` | Uses `oidc-usermodel-attribute-mapper` reading the `operator_role` user attribute and emitting it as the nested claim `uisce_metadata.operator_role`. The Go backend reads it from `claims.uisce_metadata.operator_role` first, then falls back to the top-level `operator_role` claim (`security_manager.go:693-705`). |
| Operator groups: `GG-Uisce-GlobalAdmin`, `GG-Uisce-Helpdesk`, `GG-Uisce-ProfessionalServices` | `groups[]` block in `realm-uisce-core-ops.json` | Three groups, each with a single `realmRoleMappings` entry binding to the corresponding tier role. |
| Tier roles: `global_admin`, `helpdesk`, `professional_services` | `roles.realm[]` block in `realm-uisce-core-ops.json` + `global_ops` (extra tier for read-only support staff) | All four roles live in the core-ops realm only; tenant realm has its own `tenant_user` role. |

### 2.2 PostgreSQL Lease Ledger

| Spec column | Implementation column | Notes |
|---|---|---|
| Table `security.operator_leases` | ✅ | Renamed from `staff_tenant_assignments` via `backend/migrations/20260709_rename_operator_leases.up.sql`. Backward-compat VIEW `staff_tenant_assignments` retained for one release. |
| `lease_id UUID PK` | ✅ | Renamed from `assignment_id`. |
| `operator_principal VARCHAR(255)` | ✅ | Renamed from `operator_user_id`. Cardinal Rule 7.4: this is the immutable binding key. |
| `operator_role VARCHAR(100)` | ✅ | Added by the rename migration; defaults to `'helpdesk'` for legacy rows. |
| `target_tenant_id UUID NOT NULL` | ✅ | Unchanged. |
| `assigned_profile_key VARCHAR(100)` | ✅ | Added by the rename migration; defaults to `'standard_guest'` for legacy rows. |
| `ticket_reference_id VARCHAR(100)` | ✅ | Added by the rename migration. Populated by the `LogStart` audit hook in `impersonation_audit.go`. |
| `issued_at TIMESTAMPTZ` | ✅ | Renamed from `created_at`. |
| `expires_at TIMESTAMPTZ NOT NULL` | ✅ | Unchanged. Check constraint `chk_lease_expiration` enforces `expires_at > issued_at` per spec. |
| `revoked_at TIMESTAMPTZ` | ✅ | Added by the rename migration. Soft-revocation timestamp. |
| `revocation_reason VARCHAR(255)` | ✅ | Added by the rename migration. |
| `idx_active_operator_leases` partial index | ✅ | `(lease_id, target_tenant_id) WHERE revoked_at IS NULL AND expires_at > CURRENT_TIMESTAMP`. Cardinal Rule 7.4: resolver only ever reads unexpired, unrevoked rows. |

### 2.3 Redpanda Stream

| Spec requirement | Implementation | Notes |
|---|---|---|
| Topic `uisce.audit.operator.v1` | ✅ | Created by `backend/audit-infrastructure/start.sh` with 12 partitions, `retention.ms=604800000` (7 days), `cleanup.policy=delete`. |
| Strict structural contract via Schema Registry | ✅ | Avro schema at `backend/internal/audit/schemas/operator_impersonation_event.avsc` registered as subject `uisce.audit.operator.v1-value`. Cardinal Rule 1 (config-before-code): schema is data, code consumes the contract. |
| Cleanup policy `delete` | ✅ | Set explicitly in `start.sh`. |
| 7-day rolling retention | ✅ | `retention.ms=604800000` ms = 168 hours. |
| Long-term cold storage offloaded | ✅ | OLTP fast-path stays in `platform_admin_audit` + `impersonation_action_audit` (Cardinal Rule 4 — Hot/Cold Watermark). |

### 2.4 Apache Iceberg Cold Storage

| Spec column | Implementation column | Notes |
|---|---|---|
| Table `iceberg.audit.operator_impersonation_audit` | ✅ | Defined in `backend/internal/audit/iceberg_schema.sql`. |
| `event_id VARCHAR` | ✅ | |
| `event_timestamp TIMESTAMP(3) WITH TIME ZONE` | ✅ | Microsecond precision per spec. |
| `operator_principal VARCHAR` | ✅ | |
| `operator_role VARCHAR` | ✅ | |
| `lease_id VARCHAR` | ✅ | |
| `ticket_reference_id VARCHAR` | ✅ | |
| `target_tenant_id VARCHAR` | ✅ | Partition key — allows O(partition prune) regulatory queries. |
| `assumed_profile_key VARCHAR` | ✅ | |
| `clearance_level VARCHAR` | ✅ | |
| `http_method VARCHAR` | ✅ | |
| `request_path VARCHAR` | ✅ | |
| `business_object_key VARCHAR` | ✅ | |
| `action_payload_json VARCHAR` | ✅ | JSON-serialized array of `{backendId, rawSql, parameterBindings[]}`. |
| `ip_address VARCHAR` | ✅ | |
| `year_month_day VARCHAR` | ✅ | Partition key — time-bucketed compaction granularity. |
| `format = 'PARQUET'` | ✅ | |
| `partitioning = ARRAY['target_tenant_id', 'year_month_day']` | ✅ | |
| Location `s3a://audit/compliance/operator_impersonation/` | ✅ | Note: spec text used `s3a://uisce-audit-lakehouse/...`; the production MinIO bucket is named `audit` (see `audit-infrastructure/docker-compose.yml` line 96). Path under bucket matches the spec. |

### 2.5 End-to-End Telemetry Path

```
BOSQLGenerator.GenerateSQL(ctx, req)
   │
   ├─→ InjectTenantScopingToGraph  (Cardinal Rule 7 — Compiler check)
   │
   ├─→ Recorder.RecordAIQueryGenerated   (existing AI Gate audit)
   │
   └─→ if impersonation_active:
         AuditPublisher.PublishOperatorImpersonationAction  (Cardinal Rule 8.1 — sync, fail-closed)
            │
            └─→ segmentio/kafka-go → Redpanda topic uisce.audit.operator.v1
                                          │
                                          └─→ OperatorImpersonationIcebergSink (cmd/audit-sink)
                                                 │
                                                 ├─→ segmentio/parquet-go (Parquet rows)
                                                 ├─→ minio-go → s3a://audit/compliance/operator_impersonation/...
                                                 └─→ Iceberg REST catalog snapshot registration
                                                       │
                                                       └─→ Trino SELECT (regulatory queries)
```

---

## 3. Part 2 — Tenants (Self-Service Entitlements)

### 3.1 Keycloak Configuration

| Spec requirement | Implementation | Notes |
|---|---|---|
| Standard realm for external users | `keycloak/realm-uisce-tenants.json` (was `realm-uisce.json`) | Realm enabled, SSO-enabled, customer federation. |
| Identity Provider Federation (Azure AD / Okta / Ping) | ⚠️ Partial | The tenant realm accepts IdP brokers but does NOT ship with Azure AD / Okta / Ping pre-configured. Operators provision per-tenant brokers at onboarding time via `scripts/setup-keycloak-realm.sh --idp-config <path>`. Example broker JSON files are at `keycloak/idp-brokers/{azure-ad,okta,ping}.example.json`. |
| Claim pass-through mappers for `groups` | ✅ | `clientScopes[name=uisce-groups].protocolMappers[oidc-group-membership-mapper]` emits the raw external group strings (e.g., Azure Object UUIDs) verbatim into the `groups` claim. Cardinal Rule 1: NO mapping to internal Keycloak roles is performed at the realm level — translation happens at the database plane. |
| Pass-Through Identity Engine | ✅ | `ProfileService.ResolveTenantAndRoleWithFallback` resolves `tenant_id` + `profile_key` from raw `groups` claims by joining to `security.identity_profile_mappings`. |

### 3.2 PostgreSQL Profile Mapping Framework

| Spec table / column | Implementation | Notes |
|---|---|---|
| Table `security.identity_profile_mappings` | ✅ | Renamed columns via `backend/migrations/20260709_rename_identity_profile_columns.up.sql`. |
| `id UUID PK` | ✅ | Renamed from `mapping_id`. |
| `idp_client_id VARCHAR(255)` | ✅ | Unchanged. |
| `idp_group_id VARCHAR(255)` | ✅ | Renamed from `idp_group_claim` in a prior migration. Cardinal Rule 7.4: this is the immutable external group identifier. |
| `tenant_id UUID` | ✅ | Unchanged. |
| `profile_key VARCHAR(100)` | ✅ | Renamed from `functional_role`. |
| `clearance_level VARCHAR(10) DEFAULT 'L1'` | ✅ | Stored as `VARCHAR(50)` (more headroom than spec's 10 chars). |
| `idx_idp_perimeter_lookup` | ✅ | New index on `(idp_client_id, idp_group_id)` for the resolver hot path. |
| Table `security.app_user_tenant_overrides` (fallback catalog) | ✅ | New table via `backend/migrations/20260709_app_user_tenant_overrides.up.sql`. Used by `ProfileService.ResolveAppUserOverride`. |
| Resolver fallback chain | ✅ | `ProfileService.ResolveTenantAndRoleWithFallback` tries `identity_profile_mappings` first, falls through to `app_user_tenant_overrides`, returns `sql.ErrNoRows` if neither matches. Wired into `auth_context.go` B1 branch. |

### 3.3 ABAC Policy Ledger

| Spec table / column | Implementation | Notes |
|---|---|---|
| Table `security.abac_policies` | ✅ VIEW | Defined as a VIEW in `backend/migrations/20260709_security_abac_policies_view.up.sql` that flattens `public.abac_policies` JSONB rule blobs into spec columns. The write path stays in `public.abac_policies` (Cardinal Rule 1 — config-before-code). |
| `policy_id UUID` | ✅ | Maps to `public.abac_policies.id`. |
| `tenant_id UUID` | ✅ | Nullable; NULL = Core Blueprint, UUID = Tenant Override. |
| `target_profile_key VARCHAR(100)` | ✅ | COALESCE over `subject_rules->>'role' \| profile_key \| profile`. |
| `action_attribute VARCHAR(100)` | ✅ | COALESCE over `action_rules->>'action' \| action_attribute`. |
| `condition_dsl TEXT` | ✅ | JSONB blob (subject_rules, action_rules, resource_rules, environment_rules, mandatory_filters) serialized to TEXT. |
| `effect VARCHAR(10)` | ✅ | Maps to `effect`. |
| `priority_rank INT` | ✅ | Maps to `priority`. |
| `idx_abac_policy_resolver` | ✅ | Index on `public.abac_policies (subject_rules->>'role', action_rules->>'action', tenant_id) WHERE enabled = TRUE`. |
| Deny-overrides precedence matrix | ✅ | `entitlement_handlers.go:evaluateABACPolicy` orders `ORDER BY priority DESC` and short-circuits on first deny. Cardinal Rule 7.4: rows with `tenant_id IS NULL` (Core Blueprint) overlay tenant-specific rows. |

---

## 4. Core Execution Principles (The Three Guardrails)

### 4.1 Gateway Check (`backend/internal/middleware/auth_context.go`)

| Spec requirement | Implementation | Notes |
|---|---|---|
| Strip `X-Tenant-ID` header | ✅ | `r.Header.Del("X-Tenant-ID")` runs at the top of `ServeHTTP` (Cardinal Rule 7 safeguard). Captured into `requestedTenantID` for legitimate flows (lease verification, dev fallback, global-admin scope). |
| Lease verification for `professional_services` / `helpdesk` | ✅ | `VerifyStaffAssignment` calls `SELECT EXISTS ... FROM security.operator_leases WHERE operator_principal = $1 AND target_tenant_id = $2 AND revoked_at IS NULL AND expires_at > NOW()`. |

### 4.2 Compiler Check (`backend/internal/boresolver/bo_sql_generator.go`)

| Spec requirement | Implementation | Notes |
|---|---|---|
| Embed tenant_id into AST | ✅ | `InjectTenantScopingToGraph` plants `t0.tenant_id = $N` on the root alias and every join (parenthesized to neutralize OR short-circuit). Cardinal Rule 5.4 (compilation-time physical abstraction): the `BOSQLGenerator` resolves the target through `bo_binding_id` rather than raw table name assumptions. |
| Operator impersonation telemetry emit | ✅ | New hook at the end of `GenerateSQL` emits `OperatorImpersonationActionEvent` synchronously to `uisce.audit.operator.v1`. Cardinal Rule 8.1: write-before-invalidate; fail-closed on publish error. |

### 4.3 Storage Check (`backend/migrations/20260709_rls_audit_tables_uisce_guc.up.sql`)

| Spec requirement | Implementation | Notes |
|---|---|---|
| Row-Level Security policies referencing `current_setting('uisce.current_tenant')` | ✅ | RLS enabled on `platform_admin_audit`, `impersonation_action_audit`, `security.operator_leases`. Policy predicate: `COALESCE(current_setting('uisce.current_tenant', TRUE), '') = '' OR target_tenant_id::text = current_setting('uisce.current_tenant', TRUE)`. Empty GUC sentinel grants cross-tenant visibility to platform/admin context. |
| Middleware sets `uisce.current_tenant` per request | ✅ | `backend/internal/api/api.go::tenantRLSMiddleware` calls `set_config('uisce.current_tenant', $3, false)` on every authenticated request. |

---

## 5. Known Deviations from the Spec

| # | Deviation | Why | Migration path |
|---|---|---|---|
| 1 | Token Exchange uses internal HMAC instead of RFC 8693 | Simpler operational model; per-session revocation via `security.operator_leases` | Set `oauth2.token.exchange.grant.enabled=true` in Keycloak (already done) and rewire `signImpersonationToken` to issue RFC 8693-compliant tokens |
| 2 | `security.abac_policies` is a VIEW not a table | Cardinal Rule 1 (config-before-code) requires the canonical data to stay in `public.abac_policies` | Migrate the VIEW to a materialised table with trigger-driven sync if write latency becomes a concern |
| 3 | Iceberg location `s3a://audit/compliance/...` (not `uisce-audit-lakehouse`) | Existing MinIO bucket is `audit`; renaming would invalidate all in-flight Trino queries | Create a new `uisce-audit-lakehouse` bucket and re-point the `s3a://` location when governance review approves |
| 4 | IdP Federation brokers are not pre-configured for any tenant | Tenant onboarding is a per-tenant operation; shipping defaults would leak metadata | Run `setup-keycloak-realm.sh --idp-config <path>` per tenant at provisioning time |
| 5 | Group names `GG-Uisce-GlobalAdmin/Helpdesk/ProfessionalServices` map to `Uisce-Global-Admins/Helpdesk/ProfessionalServices` in the existing tenant realm | Tenant realm already had the un-prefixed names from earlier migrations | Drop the old names in the next major schema bump; the spec's `GG-` prefix is the canonical form |

---

## 6. Compliance Query Examples

The spec's Part 1.5 example regulatory query (search all operator actions
inside a specific tenant over a date range) translates to Trino:

```sql
SELECT
    event_timestamp,
    operator_principal,
    ticket_reference_id,
    request_path,
    json_extract_scalar(action_payload_json, '$.actionContext.generatedQueries[0].rawSql') AS executed_statement
FROM iceberg.audit.operator_impersonation_audit
WHERE target_tenant_id = 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11'
  AND year_month_day >= '2026-01-01'
ORDER BY event_timestamp DESC;
```

The Hot/Cold separation guarantees the OLTP fast-path remains in
`platform_admin_audit` + `impersonation_action_audit` for low-latency
in-app queries; the Iceberg table is the immutable regulatory ledger.

---

## 7. Verification Checklist

A reviewer can validate the implementation against this checklist:

- [x] `security.operator_leases` table exists with all spec columns
- [x] `security.identity_profile_mappings` table exists with all spec columns
- [x] `security.app_user_tenant_overrides` table exists
- [x] `security.abac_policies` view exposes spec columns
- [x] `idx_active_operator_leases` is a partial index
- [x] `idx_idp_perimeter_lookup` exists
- [x] `idx_abac_policy_resolver` exists
- [x] `uisce-core-ops` Keycloak realm is provisioned separately
- [x] `uisce-studio-ops` confidential client exists in the ops realm
- [x] `GG-Uisce-*` operator groups exist in the ops realm
- [x] `uisce_metadata.operator_role` nested claim mapper is configured
- [x] `uisce.audit.operator.v1` Redpanda topic is provisioned with 7-day retention
- [x] Avro schema for the topic is registered in the Schema Registry
- [x] `iceberg.audit.operator_impersonation_audit` table is partitioned by `target_tenant_id` and `year_month_day`
- [x] `BOSQLGenerator` emits impersonation events synchronously (Cardinal Rule 8.1)
- [x] `cmd/audit-sink` writes Parquet files to MinIO at the spec location
- [x] `X-Tenant-ID` header is stripped at the gateway entry point
- [x] RLS policies reference `uisce.current_tenant`