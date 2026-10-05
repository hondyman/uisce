# Letting tenants' application roles connect to their own databases

**Owner: platform. Applies to:** any Postgres cluster that hosts tenant databases (ADR-030, ADR-048).

A tenant's database is reached by its own role, `<database>_app`, with a password the saga stores in the secrets store. That
role has to be allowed in by `pg_hba.conf`. On a cluster whose `pg_hba.conf` has been tightened to specific roles (the dev
host's has, for mTLS), a new tenant role matches **no rule** and is refused with `no pg_hba.conf entry`, whatever its password.
The saga's probe then passes only from where an earlier rule happens to admit it (the Docker network), and the application
fails from anywhere else.

## The one line, and the group that makes it one line

`pg_hba.conf` cannot match a role by prefix, but it can match a role's **group**. Tenant roles join one group at provisioning,
so the cluster needs one rule instead of one per tenant.

1. **Create the group once** (an administrator decision; the saga never creates cluster roles):

   ```sql
   CREATE ROLE ivy_tenant_apps NOLOGIN;
   ```

2. **Tell the worker** to use it: `TENANT_DB_ROLE_GROUP=ivy_tenant_apps` in the worker's environment.
   - **Unset** is a legitimate no-op, for a cluster whose `pg_hba.conf` already admits every tenant role (cert auth for everyone).
   - **Set and malformed** (not a plain lower-case identifier: letters, digits, underscores, starting with a letter, 63 characters at
     most) **refuses the run**, non-retryably, with the offending value in the message, from the planning step (before anything is
     created) and from every tenant-database step. It never degrades to "no group": a gate that a typo can switch off is not a gate.
     The worker also logs it at start.
   - **Set, well formed, and the group does not exist**: `ProvisionTenantDatabaseAccess` fails closed and non-retryably with the
     statement to run, rather than leaving a tenant whose role cannot connect.

3. **Add the rule, in the right place and as narrowly as the network allows.**

   *Placement.* `pg_hba.conf` is first-match-wins, so read the file before choosing a line. Any earlier rule whose database and role
   fields match a tenant role decides the connection, and the new line never sees it. A rule for `all` roles from an address is the
   one to look for. On the dev host (read from `pg_hba_file_rules` on 2026-10-04) there are exactly two kinds: the loopback lines and
   `host all all 172.20.0.0/16 scram-sha-256` (the Docker network, line 121), which admits tenant roles from Docker over plain `host`
   (no TLS required) before anything below it. The per-role `cert` lines (125-132) name roles and cannot match a tenant role. So the
   new line goes anywhere after line 122 and before the first rule that would catch the tenant role from the application's address;
   directly after the per-role lines (after 133) is the conventional place.

   *Scope.* The password is doing real work here, so keep the rule to the hosts that need it:

   ```
   hostssl  all  +ivy_tenant_apps  <backend host address>/32  scram-sha-256
   ```

   One `/32` line per stable address of an application host (the existing `app_admin_read` rules on the dev host already do this:
   `100.84.50.65/32`, `100.90.97.15/32`). Do **not** use the whole Tailscale range `100.64.0.0/10`: it admits every device on the
   tailnet, and then the stored password is the only thing between any of them and a tenant's data. Use a range only for a pool of
   hosts whose addresses are not stable, and say why in a comment on the line. `hostssl`, not `host`: tenant traffic is encrypted.
   Never `0.0.0.0/0` on a cluster reachable from anywhere you do not control.

4. **Reload, then read the file as the server parsed it.** A reconnect test proves one connection; it does not prove the rule is
   loaded, has no error, or sits where you think.

   ```sql
   SELECT pg_reload_conf();
   SELECT line_number, type, database, user_name, address, netmask, auth_method, error
     FROM pg_hba_file_rules ORDER BY line_number;
   ```

   Check that the new line's `error` is null, that its `line_number` is after the lines you meant it to follow and before any rule
   that matches `all` roles from the same address, and that nothing earlier matches a tenant role from the application's address.

## What the rule does and does not grant

It grants the ability to *authenticate*. What a tenant's role can do once in is unchanged: `CONNECT` on its own database only
(`REVOKE CONNECT ... FROM PUBLIC` is applied to every database; `scripts/harden-tenant-cluster.sh` verifies it), DML and
sequence use on the tenant's schemas, no DDL, not a superuser. The saga's isolation probe still tries every other database as
the tenant's role and refuses to continue if any accepts it. Group membership adds no privilege.

## Verify

```sql
-- the group exists and has members
SELECT g.rolname, array_agg(m.rolname) FROM pg_auth_members am
  JOIN pg_roles g ON g.oid = am.roleid JOIN pg_roles m ON m.oid = am.member
 WHERE g.rolname = 'ivy_tenant_apps' GROUP BY 1;
```

and the `pg_hba_file_rules` read above, **after** `pg_reload_conf()`. Then, from the host that will connect as a tenant: connect with the
tenant's role and its stored password and `sslmode=require`; it must reach its own database and be refused `FATAL: permission denied`
on any other (the saga's isolation probe checks the same from the worker).

## The tenant password is the credential of record

Over `hostssl` with scram, each tenant database's access is **one password per tenant role**: where it lives, how it changes and what
a leak costs are recorded in ADR-048 ("The tenant password is the credential of record"). In short: stored in the secrets store, not in
the datasource row (which holds only `secret_path`); **rotation is manual**; a leaked one exposes one tenant's data to anyone the
`pg_hba` rule admits, and nothing else.

## The first request

The template is a property of the gold copy: **one datasource is marked as the template for an app**
(`tenant_product_datasource.structure_template_app`), with a unique index behind it, and only the gold-copy tenant's datasources can
carry the mark. The request names no id; the saga resolves the template by the mark and **refuses when none, or more than one, is
marked**, listing the ids it found.

```sql
-- what is marked today (expect exactly one row per app)
SELECT d.id, d.structure_template_app AS app, d.source_name, d.config->>'database' AS database, d.config->>'schema' AS schemas
  FROM public.tenant_product_datasource d
 WHERE d.structure_template_app IS NOT NULL;
```

Marking is a deliberate act by the owner, on the datasource the deployed backend's gold-copy provisioning actually uses (not a
decision this document makes):

```sql
UPDATE public.tenant_product_datasource SET structure_template_app = '<app>' WHERE id = '<the gold-copy datasource>';
```

```json
POST /system/tenants/provision        (global admin; bp_queue)
{
  "tenant_name": "<name>",
  "instance_name": "<instance>",
  "tenant_code": "<lower-case code>",
  "app": "<application label, a plain lower-case identifier>",
  "structure_from_gold_copy": true
}
```

`template_datasource_id` (an explicit gold-copy datasource id) is still accepted for a run that must name one; it is exclusive with
`structure_from_gold_copy`. The plan the saga records (in the workflow's history) carries the template id and the plan hash, so a
run states what it deployed.

`app` no longer chooses the datasource in this mode (the template does); it must still be a plain identifier.
