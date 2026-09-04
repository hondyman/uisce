# Incident Step 1 Evidence — 2026-09-04
# Target: 100.84.50.65:5432 (alpha), mTLS cert auth as postgres
# Method: psql with SSL env vars (PGSSLMODE=verify-full, PGSSLCERT, PGSSLKEY, PGSSLROOTCERT)

## pg_hba_file_rules — replication section (SELECT ... FROM pg_hba_file_rules)

 line_number |  type   |   database    |         user_name          |  address   |  auth_method
-------------+---------+---------------+----------------------------+------------+---------------
         118 | local   | {all}         | {postgres}                 |            | peer
         119 | local   | {all}         | {all}                      |            | peer
         120 | host    | {all}         | {all}                      | 127.0.0.1  | scram-sha-256
         121 | host    | {all}         | {all}                      | ::1        | scram-sha-256
         124 | hostssl | {all}         | {postgres}                 | 0.0.0.0    | cert
         125 | hostssl | {all}         | {keycloak}                 | 0.0.0.0    | cert
         126 | hostssl | {all}         | {temporal}                 | 0.0.0.0    | cert
         127 | hostssl | {all}         | {nessie}                   | 0.0.0.0    | cert
         128 | hostssl | {all}         | {semlayer_lookups_replica} | 0.0.0.0    | cert
         129 | hostssl | {all}         | {usice_ops}                | 0.0.0.0    | cert
         130 | hostssl | {all}         | {usice_app}                | 0.0.0.0    | cert
         131 | hostssl | {all}         | {app_user}                 | 0.0.0.0    | cert
         132 | hostssl | {all}         | {infisical}                | 0.0.0.0    | scram-sha-256
         136 | local   | {replication} | {all}                      |            | peer
         137 | host    | {replication} | {all}                      | 127.0.0.1  | scram-sha-256
         138 | host    | {replication} | {all}                      | 100.64.0.0 | scram-sha-256
         139 | host    | {replication} | {all}                      | 172.16.0.0 | trust     <-- LIVE

## Server file paths (SHOW hba_file / config_file / data_directory)

              hba_file
-------------------------------------
 /etc/postgresql/18/main/pg_hba.conf

              config_file
-----------------------------------------
 /etc/postgresql/18/main/postgresql.conf

              data_directory
----------------------------------
 /mnt/docker-ssd/postgres/18/main

## ca.key custody search (sudo find / -name 'ca.key')

Result: NOT FOUND on this Mac (workspace). The ca.key is not present in the
filesystem accessible from this workspace. Operator should verify on the dev Mac
and move to durable custody (encrypted vault or similar, not scratch tree).

## Infisical init logs

docker not available on this workspace (this Mac). The uisce-infisical container
runs on the dev Mac. Cannot verify init logs from this session. Operator must
check: docker logs uisce-infisical 2>&1 | grep -iE 'migration|error|denied'

## ALTER ROLE infisical PASSWORD

Executed: ALTER ROLE infisical PASSWORD :'pw'  (stdin heredoc, password not in
command line / process args). Exit: 0 (success).

The infisical Postgres role is used by the Infisical application (secrets manager)
to connect to the database. The new password was NOT stored in the commit
transcript. backend/.env has been updated with:

    INFISICAL_DB_PASSWORD=<ROTATED_2026-09-04_REPLACE_WITH_NEW>

The infisical container must be restarted to pick up the new credential.
Container restart is an operator action.

## Migration log summary (oms.migration_log)

119 migrations applied. Last applied:
  20261019_workflow_definitions.up.sql  applied 2026-09-03 12:59:43

No migration log available for:
  - backend/migrations/ (legacy directory, not tracked by oms.migration_log)
  - backend/internal/database/migrations/ (not tracked by oms.migration_log)

## Operator action required — replication trust line (CRITICAL)

File is owned by postgres:postgres (mode 640). SSH sudo required.

Exact commands to fix the live trust line:

```bash
# Connect and edit as the postgres user
ssh eganpj@100.84.50.65
sudo -u postgres sed -i 's/host.*replication.*all.*172\.16\.0\.0.*trust/host    replication  all     172.16.0.0\/12    scram-sha-256/' /etc/postgresql/18/main/pg_hba.conf

# Verify the edit
sudo -u postgres grep '172.16.0.0' /etc/postgresql/18/main/pg_hba.conf

# Reload config (no restart needed)
psql -c "SELECT pg_reload_conf();"
```

Then verify from this workspace:
```bash
PGSSLMODE=verify-full PGSSLCERT=~/.uisce/certs/postgres-client.crt \
  PGSSLKEY=~/.uisce/certs/postgres-client.key \
  PGSSLROOTCERT=~/.uisce/certs/ca.crt \
  psql -h 100.84.50.65 -U postgres -d alpha \
  -c "SELECT line_number, auth_method FROM pg_hba_file_rules WHERE address = '172.16.0.0'"
# Expected: line 139, auth_method = scram-sha-256 (not trust)
```

## Other operator actions

2. INFISICAL_TOKEN rotation (Infisical UI, operator-side):
   Rotate the app-level INFISICAL_TOKEN, update backend/.env INFISICAL_TOKEN value
   Restart uisce-infisical container

3. Infisical container restart (after ALTER ROLE completes):
   docker restart uisce-infisical
   Verify: docker logs uisce-infisical 2>&1 | grep -i error
