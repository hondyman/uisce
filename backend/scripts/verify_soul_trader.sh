#!/usr/bin/env bash
# verify_soul_trader.sh
# Verifies soul_trader provisioning in both the central alpha DB and the orm DB.

set -euo pipefail

ALPHA_HOST="${ALPHA_HOST:-100.84.50.65}"
ORM_HOST="${ORM_HOST:-100.84.50.65}"
ALPHA_DB="${ALPHA_DB:-alpha}"
ORM_DB="${ORM_DB:-orm}"
PGUSER="${PGUSER:-postgres}"
export PGPASSWORD="${PGPASSWORD:-postgres}"

ALPHA_DSN="host=$ALPHA_HOST port=5432 dbname=$ALPHA_DB user=$PGUSER"
ORM_DSN="host=$ORM_HOST port=5432 dbname=$ORM_DB user=$PGUSER"

ERRORS=0

echo "=========================================="
echo "  soul_trader provisioning verification"
echo "=========================================="
echo ""

# ---------------------------------------------------------------------------
# 1. ORM DB — verify all 14 tables exist
# ---------------------------------------------------------------------------
echo "[1/5] ORM DB — checking tables..."
ORM_TABLES=(
    "ref.asset_class"          "ref.currency"               "ref.exchange"
    "ref.order_type"          "ref.time_in_force"          "ref.side"
    "ref.order_status"        "ref.order_event_type"       "ref.event_source"
    "ref.order_link_type"     "ref.venue_type"             "ref.allocation_status"
    "ref.settlement_status"     "ref.liquidity_flag"
    "mds.counterparty"        "mds.exchange_membership"    "mds.account"
    "mds.portfolio"           "mds.security_master"       "mds.venue"
    "oms.orders"              "oms.order_slice"            "oms.order_link"
    "oms.order_event"         "oms.execution"              "oms.allocation"
    "oms.settlement"          "oms.position_lots"          "oms.current_positions"
)

for tbl in "${ORM_TABLES[@]}"; do
    SCHEMA=$(echo "$tbl" | cut -d. -f1)
    TABLE=$(echo "$tbl" | cut -d. -f2)
    EXISTS=$(psql "$ORM_DSN" -t -c "
        SELECT 1 FROM information_schema.tables
        WHERE table_schema='$SCHEMA' AND table_name='$TABLE'
        UNION ALL
        SELECT 1 FROM pg_matviews
        WHERE schemaname='$SCHEMA' AND matviewname='$TABLE'" 2>/dev/null | tr -d ' ')
    if [ "$EXISTS" = "1" ]; then
        echo "  ✓ $tbl"
    else
        echo "  ✗ MISSING: $tbl"
        ERRORS=$((ERRORS+1))
    fi
done

# ---------------------------------------------------------------------------
# 2. ORM DB — verify ref data was seeded
# ---------------------------------------------------------------------------
echo ""
echo "[2/5] ORM DB — checking reference data..."
REF_CHECKS=(
    "ref.asset_class:SELECT count(*) > 0 FROM ref.asset_class WHERE code='EQUITY'"
    "ref.currency:SELECT count(*) > 0 FROM ref.currency WHERE iso3_code='USD'"
    "ref.order_type:SELECT count(*) > 0 FROM ref.order_type WHERE code='LIMIT'"
    "ref.order_status:SELECT count(*) > 0 FROM ref.order_status WHERE code='NEW'"
    "ref.side:SELECT count(*) > 0 FROM ref.side WHERE code='BUY'"
)
for check in "${REF_CHECKS[@]}"; do
    tbl="${check%%:*}"
    sql="${check#*:}"
    COUNT=$(psql "$ORM_DSN" -t -c "$sql" 2>/dev/null | tr -d ' ')
    if [ "$COUNT" = "t" ] || [ "$COUNT" = "1" ]; then
        echo "  ✓ $tbl seed data present"
    else
        echo "  ✗ $tbl seed data missing or empty"
        ERRORS=$((ERRORS+1))
    fi
done

# ---------------------------------------------------------------------------
# 3. Central alpha DB — verify soul_trader tenant exists
# ---------------------------------------------------------------------------
echo ""
echo "[3/5] Central DB — verifying tenant..."
TENANT_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM tenants WHERE tenant_code='soul_trader'" 2>/dev/null | tr -d ' -' )
if [ -n "$TENANT_ID" ]; then
    echo "  ✓ tenant: soul_trader  id=$TENANT_ID"
else
    echo "  ✗ tenant 'soul_trader' NOT FOUND"
    ERRORS=$((ERRORS+1))
    TENANT_ID=""
fi

# ---------------------------------------------------------------------------
# 4. Central alpha DB — verify instance, product, datasource
# ---------------------------------------------------------------------------
echo ""
echo "[4/5] Central DB — verifying instance, product, datasource..."

if [ -n "$TENANT_ID" ]; then
    INST_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM tenant_instance WHERE tenant_id='$TENANT_ID' AND instance_name='soul_instance'" 2>/dev/null | tr -d ' -')
    if [ -n "$INST_ID" ]; then
        echo "  ✓ tenant_instance: soul_instance  id=$INST_ID"
    else
        echo "  ✗ tenant_instance 'soul_instance' NOT FOUND"
        ERRORS=$((ERRORS+1))
    fi
fi

PROD_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM alpha_product WHERE product_code='order_management'" 2>/dev/null | tr -d ' -')
if [ -n "$PROD_ID" ]; then
    echo "  ✓ alpha_product: Order Management  id=$PROD_ID"
else
    echo "  ✗ alpha_product 'order_management' NOT FOUND"
    ERRORS=$((ERRORS+1))
fi

DS_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM alpha_datasource WHERE datasource_code='orm' LIMIT 1" 2>/dev/null | tr -d ' -')
if [ -n "$DS_ID" ]; then
    echo "  ✓ alpha_datasource: ORM  id=$DS_ID (note: may have duplicate rows due to CDC limitations)"
else
    echo "  ✗ alpha_datasource 'orm' NOT FOUND"
    ERRORS=$((ERRORS+1))
fi

CONN_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM connections WHERE name='orm' LIMIT 1" 2>/dev/null | tr -d ' -')
if [ -n "$CONN_ID" ]; then
    echo "  ✓ connections: orm  id=$CONN_ID"
else
    echo "  ✗ connections 'orm' NOT FOUND"
    ERRORS=$((ERRORS+1))
fi

# ---------------------------------------------------------------------------
# 5. Central alpha DB — verify tenant_product + tenant_product_datasource
# ---------------------------------------------------------------------------
echo ""
echo "[5/5] Central DB — verifying tenant_product and tenant_product_datasource..."

if [ -n "$TENANT_ID" ] && [ -n "$INST_ID" ] && [ -n "$PROD_ID" ]; then
    TP_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM tenant_product WHERE tenant_id='$TENANT_ID' AND datasource_id='$INST_ID' AND alpha_product_id='$PROD_ID'" 2>/dev/null | tr -d ' -')
    if [ -n "$TP_ID" ]; then
        echo "  ✓ tenant_product  id=$TP_ID"
    else
        echo "  ✗ tenant_product NOT FOUND"
        ERRORS=$((ERRORS+1))
        TP_ID=""
    fi
fi

if [ -n "$TP_ID" ]; then
    TPD_ID=$(psql "$ALPHA_DSN" -t -c "SELECT id FROM tenant_product_datasource WHERE tenant_product_id='$TP_ID' AND source_name='orm'" 2>/dev/null | tr -d ' -')
    if [ -n "$TPD_ID" ]; then
        echo "  ✓ tenant_product_datasource  id=$TPD_ID"
    else
        echo "  ✗ tenant_product_datasource NOT FOUND"
        ERRORS=$((ERRORS+1))
    fi
fi

# ---------------------------------------------------------------------------
# Summary
# ---------------------------------------------------------------------------
echo ""
echo "=========================================="
if [ "$ERRORS" -eq 0 ]; then
    echo "  All checks passed ✓"
    echo "=========================================="
    echo ""
    echo "  Next step — start catalog scan:"
    echo "    POST /api/catalog/scan"
    echo "    body: {\"tenant_product_datasource_id\":\"$TPD_ID\"}"
    exit 0
else
    echo "  $ERRORS check(s) FAILED"
    echo "=========================================="
    exit 1
fi
