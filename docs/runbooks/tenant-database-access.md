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

2. **Tell the worker** to use it: `TENANT_DB_ROLE_GROUP=ivy_tenant_apps` in the worker's environment. A name that is not a plain
   lower-case identifier is ignored. If the variable is set and the group does not exist, `ProvisionTenantDatabaseAccess`
   fails closed and non-retryably with the statement to run, rather than leaving a tenant whose role cannot connect.

3. **Add the rule** (before any catch-all, after the `local`/loopback lines; first match wins):

   ```
   hostssl  all  +ivy_tenant_apps  <the networks the application connects from>  scram-sha-256
   ```

   `hostssl`, not `host`: tenant traffic must be encrypted. Name the networks the API and workers really connect from; do not
   use `0.0.0.0/0` on a cluster that is reachable from anywhere you do not control. Reload (`SELECT pg_reload_conf()`), no restart.

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
-- the rule is loaded and has no error
SELECT line_number, type, database, user_name, address, auth_method, error
  FROM pg_hba_file_rules WHERE 'ivy_tenant_apps' = ANY (user_name) OR '+ivy_tenant_apps' = ANY (user_name);
```

From the host that will connect as a tenant: connect with the tenant's role and its stored password and `sslmode=require`; it
must reach its own database and be refused `FATAL: permission denied` on any other.

## The dev host (100.84.50.65), as read on 2026-10-04

`pg_hba.conf` there lets non-cert roles in only from the Docker network (`host all all 172.20.0.0 scram-sha-256`). The backend runs
on a developer's Mac over Tailscale (`100.64.0.0/10`), where no rule admits a tenant role. The line above, scoped to the
Tailscale range, is what the first tenant on that host needs. It was not applied: it is a change to a shared host.
