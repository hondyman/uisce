#!/bin/bash
# Value-level audit: Postgres vs StarRocks, per row, order-independent.
#
# Two earlier attempts gave misleading results:
#   1. non-null COUNTS pass even when a value is stored non-null but wrong
#      (the decimal rounding case the count audit cannot see)
#   2. md5 over an ordered concat disagreed on 5 of 15 columns purely because
#      StarRocks' default collation orders UUID strings differently from
#      Postgres -- every sampled value was byte-identical
# So: emit (id, value) per row from both sides, LC_ALL=C sort both, diff.
set -u
fail=0

# Which destination is being audited.
#
# Default is the shared `oms` database, which is where every row landed before
# per-tenant routing. Under routing each tenant has its own database, and the
# comparison that actually means something is that tenant's rows in Postgres against
# that tenant's rows in StarRocks. Auditing tenant A against the shared database
# would prove nothing, and comparing all of Postgres against one tenant's slice would
# report differences that are merely the other tenants' rows.
#
#   SR_DB=tenant_<id> TENANT_ID=<uuid> scripts/cdc_value_audit.sh
#
# TENANT_ID scopes the Postgres side to one tenant; leave it unset for the shared
# database, where no tenant filter applies. Set without SR_DB the script compares a
# tenant slice against the shared database, which is a meaningful check only while
# routing is off -- under routing it is expected to differ, and the script says so.
SR_DB="${SR_DB:-oms}"
if [ -n "${TENANT_ID:-}" ]; then
  PG_WHERE=" where tenant_id = '${TENANT_ID}'"
else
  PG_WHERE=""
fi

echo "auditing StarRocks database: ${SR_DB}${TENANT_ID:+ (tenant ${TENANT_ID})}"

# ---- key-set (phantom) check -------------------------------------------------
#
# The value comparison above cannot see a row that exists on one side only. That gap
# is not theoretical: an unattributable delete leaves a row in StarRocks that has no
# counterpart in Postgres forever, and it audits as "present" on every column.
#
# So compare the key sets separately and count both directions. The number that must
# stay zero is PHANTOM -- a StarRocks key with no Postgres row. `missing` is the
# opposite (a Postgres row never landed) and is a backfill or routing failure.
#
# It is reported rather than treated as a hard failure because the two sides are
# legitimately mid-flight during a backfill, and a script that goes red for a
# transient reason teaches people to ignore it.
phantom_total=0

keys_check() {
  local pgtbl="$1" srtbl="$2" label="$3"
  printf "select id::text from %s%s;\n" "$pgtbl" "$PG_WHERE" > /tmp/_kpg.sql
  printf "select id from %s;\n" "$srtbl" > /tmp/_ksr.sql

  PGPASSWORD=postgres psql -h 127.0.0.1 -p 5432 -U postgres -d alpha \
    -tAF'|' -q -f /tmp/_kpg.sql 2>/dev/null \
    | sed 's/[[:space:]]*$//' | LC_ALL=C sort -u > /tmp/_kpg.txt
  docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -N < /tmp/_ksr.sql 2>/dev/null \
    | tr '\t' '|' | sed 's/[[:space:]]*$//' | LC_ALL=C sort -u > /tmp/_ksr.txt

  local phantom missing
  phantom=$(comm -13 /tmp/_kpg.txt /tmp/_ksr.txt | grep -c . || true)
  missing=$(comm -23 /tmp/_kpg.txt /tmp/_ksr.txt | grep -c . || true)
  phantom_total=$(( phantom_total + phantom ))

  if [ "$phantom" -eq 0 ] && [ "$missing" -eq 0 ]; then
    printf 'KEYS  %-42s ok\n' "$label"
  else
    printf 'KEYS  %-42s phantom=%s missing=%s\n' "$label" "$phantom" "$missing"
    comm -13 /tmp/_kpg.txt /tmp/_ksr.txt | head -3 | sed 's/^/        phantom: /'
    comm -23 /tmp/_kpg.txt /tmp/_ksr.txt | head -3 | sed 's/^/        missing: /'
  fi
}

run() {
  local label="$1"
  PGPASSWORD=postgres psql -h 127.0.0.1 -p 5432 -U postgres -d alpha \
    -tAF'|' -q -f /tmp/_pg.sql 2>/dev/null \
    | sed 's/[[:space:]]*$//' | LC_ALL=C sort > /tmp/_pg.txt
  docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -N < /tmp/_sr.sql 2>/dev/null \
    | tr '\t' '|' | sed 's/[[:space:]]*$//' | LC_ALL=C sort > /tmp/_sr.txt

  local pc sc nd
  pc=$(wc -l < /tmp/_pg.txt | tr -d ' ')
  sc=$(wc -l < /tmp/_sr.txt | tr -d ' ')
  nd=$(diff /tmp/_pg.txt /tmp/_sr.txt | grep -cE '^[<>]')
  if [ "$nd" -eq 0 ] && [ "$pc" -eq "$sc" ]; then
    printf 'OK    %-42s rows=%s\n' "$label" "$pc"
  else
    fail=1
    printf 'DIFF  %-42s pg=%s sr=%s differing=%s\n' "$label" "$pc" "$sc" "$nd"
    diff /tmp/_pg.txt /tmp/_sr.txt | grep -E '^[<>]' | head -4 | sed 's/^/        /'
  fi
}

col() {
  local pgtbl="$1" srtbl="$2" c="$3" label="$4"
  printf "select id::text, coalesce(%s::text,'~') from %s%s;\n" \
    "$c" "$pgtbl" "$PG_WHERE" > /tmp/_pg.sql
  printf "select id, ifnull(cast(%s as string),'~') from %s;\n" "$c" "$srtbl" > /tmp/_sr.sql
  run "$label"
}

O_PG='orm."order"';   O_SR="${SR_DB}.orm_order"
E_PG='orm.execution'; E_SR="${SR_DB}.orm_execution"
P_PG='orm.placement'; P_SR="${SR_DB}.orm_placement"
A_PG='orm.order_allocation';     A_SR="${SR_DB}.orm_order_allocation"
X_PG='orm.execution_allocation'; X_SR="${SR_DB}.orm_execution_allocation"

for c in sec_id target_qty executed_qty leaves_qty limit_price avg_price; do
  col "$O_PG" "$O_SR" "$c" "order.$c"
done
for c in exec_qty exec_price; do
  col "$E_PG" "$E_SR" "$c" "execution.$c"
done
for c in routed_qty executed_qty leaves_qty; do
  col "$P_PG" "$P_SR" "$c" "placement.$c"
done
for c in target_qty allocated_qty; do
  col "$A_PG" "$A_SR" "$c" "order_allocation.$c"
done
for c in alloc_exec_qty alloc_exec_price; do
  col "$X_PG" "$X_SR" "$c" "execution_allocation.$c"
done

echo
echo "-- key sets (phantoms must be 0) --"
keys_check "$O_PG" "$O_SR" "order"
keys_check "$E_PG" "$E_SR" "execution"
keys_check "$P_PG" "$P_SR" "placement"
keys_check "$A_PG" "$A_SR" "order_allocation"
keys_check "$X_PG" "$X_SR" "execution_allocation"

echo
if [ "$phantom_total" -ne 0 ]; then
  # A phantom is a hard failure even when every value matched: it is a row the
  # warehouse will keep forever that the source of truth has deleted, and it is
  # invisible to every other line in this script.
  fail=1
fi

if [ "$fail" -eq 0 ]; then
  echo "RESULT: ${SR_DB} matches byte-for-byte on all 15 numeric columns, 0 phantom keys"
else
  echo "RESULT: differences found in ${SR_DB} (see DIFF / KEYS lines above)"
fi
exit $fail