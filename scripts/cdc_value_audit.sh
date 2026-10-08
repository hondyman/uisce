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
  printf "select id::text, coalesce(%s::text,'~') from %s;\n" "$c" "$pgtbl" > /tmp/_pg.sql
  printf "select id, ifnull(cast(%s as string),'~') from %s;\n" "$c" "$srtbl" > /tmp/_sr.sql
  run "$label"
}

O_PG='orm."order"';   O_SR='oms.orm_order'
E_PG='orm.execution'; E_SR='oms.orm_execution'
P_PG='orm.placement'; P_SR='oms.orm_placement'
A_PG='orm.order_allocation';     A_SR='oms.orm_order_allocation'
X_PG='orm.execution_allocation'; X_SR='oms.orm_execution_allocation'

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
if [ "$fail" -eq 0 ]; then
  echo "RESULT: all 15 numeric columns match byte-for-byte"
else
  echo "RESULT: differences found (see DIFF lines above)"
fi
exit $fail