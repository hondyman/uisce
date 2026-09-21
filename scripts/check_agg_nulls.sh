#!/usr/bin/env bash
# check_agg_nulls.sh
# Validates that all expected columns in the agg.* StarRocks tables have
# non-zero values. All-NULL columns typically indicate a schema mismatch
# where the loader silently tolerates a type it can't decode (e.g. postgres
# interval landing as NULL because Debezium emits the Kafka Connect Duration
# schema and decodeDecimals() only handles Decimal).
#
# Run AFTER the alpha_trg / crims_trg connector snapshot completes and at
# least some CDC events have been seen.
#
# Usage:
#   ./scripts/check_agg_nulls.sh
# Or against a different host:
#   STARROCKS_HOST=otherhost ./scripts/check_agg_nulls.sh
#
# Exit code: 0 if all columns have non-NULL data, 1 if any all-NULL column
# found (which means a schema mismatch to investigate).

set -euo pipefail

STARROCKS_HOST="${STARROCKS_HOST:-localhost}"
STARROCKS_PORT="${STARROCKS_PORT:-9030}"

# Use `mysql -h` against the StarRocks FE via docker exec since StarRocks
# does not ship a standalone mysql client in the host environment.
DOCKER_EXEC_BASE=(docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root \
                  --skip-column-names -B -e)

tables=(
    alpha_investment_opportunities
    alpha_process_execution_metrics
    alpha_template_ratings
    alpha_crypto_transactions
    alpha_crypto_market_data
    alpha_semantic_query_templates
    alpha_cube_custom_models
    alpha_cube_security_policies
    alpha_security_user_fund_access
    alpha_metrics_registry
    alpha_uma_accounts
    alpha_uma_rebalance_requests
    alpha_validation_patterns
    alpha_crypto_prices
    crims_security_identifier
)

total_failures=0

printf "%-40s %-30s %15s %15s\n" "TABLE" "COLUMN" "NULL_COUNT" "TOTAL_COUNT"
printf "%-40s %-30s %15s %15s\n" "------------------------------------" "------------------------------" "---------------" "---------------"

for table in "${tables[@]}"; do
    # Get list of columns and the row count using information_schema
    row_count=$("${DOCKER_EXEC_BASE[@]}" "SELECT COUNT(*) FROM agg.${table};" 2>/dev/null | tr -d '[:space:]')
    if [[ -z "${row_count}" || "${row_count}" == "0" ]]; then
        printf "%-40s (table empty or does not exist)\n" "${table}"
        continue
    fi

    # Find columns that are 100% NULL (NULL_COUNT == row_count, not just >0)
    mapfile -t null_columns < <(
        "${DOCKER_EXEC_BASE[@]}" "
            SELECT column_name
            FROM   information_schema.columns
            WHERE  table_schema = 'agg' AND table_name = '${table}'
            ORDER BY ordinal_position;
        " 2>/dev/null | tr -d '[:space:]' | grep -v '^$' || true
    )

    for col in "${null_columns[@]}"; do
        [[ -z "${col}" ]] && continue
        null_cnt=$("${DOCKER_EXEC_BASE[@]}" "
            SELECT COUNT(*) FROM agg.\`${table}\` WHERE \`${col}\` IS NULL;
        " 2>/dev/null | tr -d '[:space:]')
        if [[ -z "${null_cnt}" ]]; then
            continue
        fi
        if [[ "${null_cnt}" == "${row_count}" ]]; then
            printf "%-40s %-30s %15s %15s  <-- ALL NULL (schema mismatch?)\n" \
                "${table}" "${col}" "${null_cnt}" "${row_count}"
            total_failures=$((total_failures + 1))
        fi
    done
done

echo ""
if [[ ${total_failures} -gt 0 ]]; then
    echo "FAIL: ${total_failures} all-NULL column(s) detected"
    echo "      A column landing 100% NULL usually means the loader"
    echo "      silently tolerated a Debezium-encoded type it doesn't"
    echo "      decode (interval, time, decimal with non-default precision)."
    echo "      Check Postgres source column type vs the loader's"
    echo "      decodeDecimals() / decodeRecord() coverage."
    exit 1
fi

echo "PASS: no all-NULL columns"
exit 0
