# Northwind ABAC Gold Copy Profiles

> **Status:** Design doc. Schema and seed policies already shipped via
> [`backend/migrations/000062_abac_security_profiles.sql`](../../backend/migrations/000062_abac_security_profiles.sql).
> Profile rows (the catalogue of Gold Copy profiles) are seeded by
> [`backend/migrations/000063_northwind_security_profile_rows.sql`](../../backend/migrations/000063_northwind_security_profile_rows.sql).
>
> **Audience:** Platform engineers wiring the Go enrichment pipeline and
> tenant admins configuring custom overrides.

---

## 1. Why this exists

Uisce is a multi-tenant SaaS. Tenants onboard with their own IdP (Keycloak +
corporate AD/LDAP). Two failure modes are obvious:

1. **Mapping AD groups directly to physical resources** (menus, pages,
   business objects). Every IdP restructure becomes a platform outage.
2. **Hard-coding role checks in Go** (`if user.Roles.Contains("admin")`).
   Tenants can't customize, and security policy lives in code instead of
   data.

The fix is the **3-tier abstraction** the rest of this doc implements:

| Tier | Lives in | Owned by |
|---|---|---|
| Identity | Keycloak, AD groups | Tenant IdP / corporate IT |
| Entitlement | `security.identity_profile_mappings` | Tenant admin |
| Policy | `public.abac_policies` + `security.security_profiles` | Platform (Gold Copy) + tenant (overrides) |

---

## 2. Schema (as shipped in `000062`)

### 2.1 `security.identity_profile_mappings`

Translates the raw IdP group claim into a **functional_role** string the
ABAC engine understands.

```sql
security.identity_profile_mappings (
    mapping_id        UUID PK,
    tenant_id         UUID NOT NULL,       -- tenant fence
    idp_group_claim   VARCHAR(255),        -- e.g. 'GG-Uisce-Compliance'
    functional_role   VARCHAR(100),        -- e.g. 'compliance_officer'
    clearance_level   VARCHAR(50)          -- e.g. 'L3'
);
```

### 2.2 `security.security_profiles`

The catalogue of abstract profiles. **`tenant_id = NULL` means Gold Copy.**

```sql
security.security_profiles (
    profile_id        UUID PK,
    tenant_id         UUID,                -- NULL = System-Level Gold Copy
    profile_key       VARCHAR(50),         -- e.g. 'trader', 'northwind_sales_rep'
    profile_name      VARCHAR(100),
    parent_profile_id UUID REFERENCES security.security_profiles(profile_id)
);
```

### 2.3 `public.abac_policies` (the engine)

Created in `000062` with defensive normalization (handles both fresh
installs and the existing `public.abac_policies` from earlier migrations).

```sql
public.abac_policies (
    id                UUID PK,
    tenant_id         UUID,                -- NULL = global; otherwise tenant fence
    datasource_id     UUID,
    name              TEXT,
    description       TEXT,
    subject_rules     JSONB,              -- e.g. {"roles": ["northwind_sales_rep"]}
    action_rules      JSONB,              -- e.g. {"actions": ["read","create"]}
    resource_rules    JSONB,              -- e.g. {"resources": ["order","customer"]}
    environment_rules JSONB,
    effect            TEXT CHECK (effect IN ('allow','deny')),
    priority          INT DEFAULT 100,    -- lower number = evaluated first
    enabled           BOOLEAN
);
```

> **Schema delta vs. the design sketch:** the actual engine stores the
> matching predicates as JSONB columns per dimension (`subject_rules`,
> `action_rules`, `resource_rules`, `environment_rules`) and expresses the
> combination algorithm as a single `effect` per row. The sketch suggested
> `combination_algorithm` + a generic `conditions` JSONB; the engine does
> not currently implement deny-overrides/permit-overrides composition in
> one row — composition happens in the Go policy evaluator, which sorts
> by `priority ASC` and short-circuits on the first matching `effect`.

---

## 3. Northwind Gold Copy Profiles

Seeded by `000063`. All rows have `tenant_id = NULL`.

| `profile_key` | `profile_name` | Role string used by `subject_rules` | Default policy shipped in `000062` |
|---|---|---|---|
| `northwind_sales_rep` | Gold Copy - Sales Representative | `northwind_sales_rep` | Allow `read/create/update` on `order`, `customer` (priority 100); Deny `create/update/delete` on `product`, `supplier` (priority 90) |
| `northwind_inventory_manager` | Gold Copy - Inventory Specialist | `northwind_inventory_manager` | Allow `read/create/update` on `product`, `supplier` (priority 100); Deny all actions on `order` (priority 90) |
| `northwind_billing_specialist` | Gold Copy - Billing Specialist | `northwind_billing_specialist` | **Policies to be added in a follow-up migration** (the `security_profiles` row is seeded now to keep the catalogue complete) |
| `northwind_executive` | Gold Copy - Commerce Executive | `northwind_executive` | **Policies to be added in a follow-up migration** |

The Northwind Business Objects these policies act on are defined in
[`backend/migrations/20241216_northwind_semantic_models.sql`](../../backend/migrations/20241216_northwind_semantic_models.sql)
and documented in [`docs/guides/README_NORTHWIND_BO.md`](../guides/README_NORTHWIND_BO.md).

---

## 4. Resolution flow at request time

```
   Keycloak token
       │
       │ (raw groups: ["GG-Uisce-Sales-NA"])
       ▼
   Go middleware (security.TokenEnricher)
       │
       │ SELECT functional_role, clearance_level
       │   FROM security.identity_profile_mappings
       │  WHERE tenant_id = $1 AND idp_group_claim = ANY($2)
       ▼
   Enriched subject context
       { functional_role: "northwind_sales_rep", clearance_level: "L2" }
       │
       ▼
   ABAC evaluator (per request)
       │
       │ 1. SELECT * FROM public.abac_policies
       │      WHERE enabled
       │        AND (tenant_id IS NULL OR tenant_id = $tenant)
       │        AND subject_rules @> {"roles": ["northwind_sales_rep"]}
       │    ORDER BY priority ASC
       │
       │ 2. Iterate rows. For each, check action_rules and resource_rules
       │    against the inbound request. First matching row's effect
       │    (allow/deny) is the decision; tie-breaker = lowest priority.
       │
       ▼
   Allow / Deny
```

Because tenant-scoped policies share the same `subject_rules` shape as
Gold Copy policies, the engine cannot distinguish them by content — it
distinguishes them by **scope**. A tenant override with `priority = 10`
will always beat a Gold Copy with `priority = 90`, regardless of effect.

---

## 5. Tenant customization pattern

### 5.1 Restrict (the Acme Logistics example)

Tenant `de305d54-75b4-431b-adb2-eb6b9e546013` requires that its Sales
Representatives (contractors) cannot create or update orders for
non-USA destinations. They do **not** edit the Gold Copy policy — they
add a higher-priority deny row.

```sql
INSERT INTO public.abac_policies
    (id, tenant_id, name, effect, priority, enabled,
     subject_rules, action_rules, resource_rules, environment_rules)
VALUES
    (gen_random_uuid(),
     'de305d54-75b4-431b-adb2-eb6b9e546013',
     'Acme - Sales Rep domestic-only',
     'deny',
     10,                                  -- beats the global baseline (100)
     true,
     '{"roles": ["northwind_sales_rep"]}',
     '{"actions": ["create","update"]}',
     '{"resources": ["order"], "destination_country": {"neq": "USA"}}',
     '{}');
```

Runtime: when an Acme Sales Rep submits an order with
`destination_country = "France"`, the engine returns the
`priority = 10` deny first and stops. When the same user submits a US
order, the engine falls through to the global baseline (`priority = 100`)
and allows it.

### 5.2 Extend

A tenant can grant a profile new capabilities without touching Gold Copy:

```sql
INSERT INTO public.abac_policies
    (id, tenant_id, name, effect, priority, enabled,
     subject_rules, action_rules, resource_rules, environment_rules)
VALUES
    (gen_random_uuid(),
     'de305d54-75b4-431b-adb2-eb6b9e546013',
     'Acme - Sales Rep can read product catalog',
     'allow',
     80,
     true,
     '{"roles": ["northwind_sales_rep"]}',
     '{"actions": ["read"]}',
     '{"resources": ["product"]}',
     '{}');
```

This sits **above** the `priority = 90` deny and changes the verdict from
deny to allow for *reads only*. The catalog write-down rule (priority 90
deny) still fires for `create/update/delete`.

### 5.3 Create a custom profile (no Gold Copy ancestor)

```sql
INSERT INTO security.security_profiles
    (profile_id, tenant_id, profile_key, profile_name)
VALUES
    (gen_random_uuid(),
     'de305d54-75b4-431b-adb2-eb6b9e546013',
     'northwind_sales_rep_us_west',        -- tenant-scoped key
     'Acme - West Coast Sales Rep');

INSERT INTO public.abac_policies
    (id, tenant_id, name, effect, priority, enabled,
     subject_rules, action_rules, resource_rules, environment_rules)
VALUES
    (gen_random_uuid(),
     'de305d54-75b4-431b-adb2-eb6b9e546013',
     'Acme - West Coast Sales Rep full order access',
     'allow',
     100,
     true,
     '{"roles": ["northwind_sales_rep_us_west"]}',
     '{"actions": ["read","create","update"]}',
     '{"resources": ["order","customer"]}',
     '{"region": "us-west"}');
```

### 5.4 Customize via inheritance (`parent_profile_id`)

If a tenant wants a profile to **start from** a Gold Copy and only adjust
specific bits, they can set `parent_profile_id` to the Gold Copy row.
The resolution semantics for `parent_profile_id` are **not yet
implemented** in the Go evaluator — the field is captured for forward
compatibility but evaluation currently treats each `security_profiles`
row as independent. Plan: layer a parent-chain merge in the enrichment
pipeline before `000064`.

---

## 6. Operational guarantees

| Guarantee | Mechanism |
|---|---|
| New tenants get the baseline for free | `tenant_id IS NULL` rows are returned by the `WHERE tenant_id IS NULL OR tenant_id = $tenant` filter |
| Global policy updates reach all tenants instantly | Edit the `tenant_id IS NULL` row in place; no migration, no per-tenant data refresh |
| Tenant cannot mutate a Gold Copy row | All `security_profiles` and `abac_policies` UIs must filter writes to `tenant_id = $current_tenant` (enforced at the handler layer) |
| Deny always wins when it has lower `priority` | The evaluator short-circuits on the first match after sorting by `priority ASC`; a deny row at priority 10 always fires before an allow at priority 100 |
| Audit trail | Every policy decision is logged with `policy_id`, `tenant_id`, `subject_role`, `resource`, `action`, and verdict (see `public.audit_log` migrations `20260208_create_audit_log.up.sql`) |

---

## 7. Open items (not in this branch)

These are tracked separately and intentionally **not** shipped here:

1. Go enrichment pipeline (`security.TokenEnricher.EnrichSubjectAttributes`
   and `security.ProfileRepository.FetchEffectiveProfile`) — design
   sketched, not implemented.
2. `parent_profile_id` resolution semantics in the evaluator.
3. `northwind_billing_specialist` and `northwind_executive` policies.
4. Admin UI CRUD for `identity_profile_mappings`, `security_profiles`,
   and tenant-scoped `abac_policies`.
5. The chat-suggested `security.abac_policies` (schema in the `security`
   schema) is **not** created; the engine lives in `public.abac_policies`.

---

## 8. Related docs

- [`docs/MULTI_TENANT_AUTH.md`](../MULTI_TENANT_AUTH.md) — JWT claim
  contract and the three tenant access patterns.
- [`docs/guides/README_NORTHWIND_BO.md`](../guides/README_NORTHWIND_BO.md) —
  Northwind Business Object framework.
- [`docs/SEMANTIC_LAYER_ARCHITECTURE.md`](../SEMANTIC_LAYER_ARCHITECTURE.md) —
  How Business Objects plug into the semantic query planner.
- [`backend/migrations/000062_abac_security_profiles.sql`](../../backend/migrations/000062_abac_security_profiles.sql) —
  Schema + initial policy seed.
- [`backend/migrations/000063_northwind_security_profile_rows.sql`](../../backend/migrations/000063_northwind_security_profile_rows.sql) —
  Profile catalogue seed.
- [`backend/migrations/20241216_northwind_semantic_models.sql`](../../backend/migrations/20241216_northwind_semantic_models.sql) —
  Northwind Business Object tables.