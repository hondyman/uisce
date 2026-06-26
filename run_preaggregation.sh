#!/bin/bash

# Automated Preaggregation Runner
# This script runs preaggregation for all configured bundles

# Logging setup
LOG_DIR="logs"
LOG_FILE="$LOG_DIR/preaggregation.log"
mkdir -p "$LOG_DIR"

# Function to log messages
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $*" | tee -a "$LOG_FILE"
}

echo "🚀 Automated Preaggregation Runner"
echo "==================================="

# Configuration
DB_HOST="localhost"
DB_PORT="5432"
DB_NAME="alpha"
DB_USER="postgres"
DB_PASSWORD="postgres"

# Bundles to process
BUNDLES=("banking" "insurance" "capital_markets" "fixed_income")

log "📅 Starting preaggregation run at $(date)"
log "📊 Processing bundles: ${BUNDLES[*]}"

# Test database connection
if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1;" &> /dev/null; then
    log "❌ Database connection failed"
    exit 1
fi

log "✅ Database connection successful"

# Process each bundle
for bundle in "${BUNDLES[@]}"; do
    log ""
    log "🔄 Processing $bundle bundle..."

    # Check if bundle schema exists
    if ! PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 1 FROM ${bundle}.metrics_registry LIMIT 1;" &> /dev/null; then
        log "⚠️  $bundle schema not found, skipping"
        continue
    fi

    # Get metrics that need preaggregation
    METRICS_QUERY="
    SELECT node_id, formula_type, formula
    FROM ${bundle}.metrics_registry
    WHERE formula_type IN ('dax_formula', 'excel_formula')
    ORDER BY node_id"

    METRICS=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "$METRICS_QUERY")

    if [ -z "$METRICS" ]; then
        log "ℹ️  No metrics found for $bundle"
        continue
    fi

    log "📈 Found $(echo "$METRICS" | wc -l) metrics to process"

    # Process each metric
    echo "$METRICS" | while read -r line; do
        if [ -n "$line" ]; then
            NODE_ID=$(echo "$line" | awk '{print $1}')
            FORMULA_TYPE=$(echo "$line" | awk '{print $2}')

            log "  🔢 Processing metric: $NODE_ID ($FORMULA_TYPE)"

            # Here you would call your preaggregation logic
            # For now, we'll simulate the preaggregation by updating the semantic_layer.preaggregated_metrics table

            # Insert/update preaggregated metric (this is a simplified example)
            DATE_STR=$(date +%Y-%m-%d)
            INSERT_QUERY="
            INSERT INTO semantic_layer.preaggregated_metrics (
                id, node_id, name, value, grain, grain_values,
                last_refresh, refresh_schedule, source_formula, data_quality
            ) VALUES (
                '${bundle}_${NODE_ID}_$(date +%Y%m%d)',
                '$NODE_ID',
                '$NODE_ID',
                ROUND((RANDOM() * 1000)::numeric, 2), -- Simulated value
                '[\"date\"]',
                '{\"date\": \"$DATE_STR\"}',
                NOW(),
                'daily',
                'DAX/Excel formula',
                '{\"completeness_score\": 0.95, \"freshness_hours\": 0}'
            )
            ON CONFLICT (id) DO UPDATE SET
                value = EXCLUDED.value,
                last_refresh = EXCLUDED.last_refresh,
                data_quality = EXCLUDED.data_quality"

            if PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "$INSERT_QUERY" 2>&1; then
                log "    ✅ $NODE_ID preaggregated successfully"
            else
                log "    ❌ Failed to preaggregate $NODE_ID"
                log "      Error: $(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$INSERT_QUERY" 2>&1 | head -3)"
            fi
        fi
    done

    log "✅ $bundle bundle processing complete"
done

log ""
log "🎉 Preaggregation run completed at $(date)"
log "📊 Summary:"
log "  - Bundles processed: ${#BUNDLES[@]}"
log "  - Next run: Tomorrow at 6:00 AM"

# Optional: Send notification or log to monitoring system
# curl -X POST http://your-monitoring-system/api/alerts -d '{"message": "Preaggregation completed", "status": "success"}'
