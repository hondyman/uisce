#!/bin/bash

# =============================================================================
# Phase 3.15: Start Temporal Workflows with Cron Schedules
# =============================================================================
# This script initializes all Temporal workflows for Phase 3.15:
#   - HourlyRollupWorkflow (cron: 5 * * * * - every hour at 05 min)
#   - DailySLAWorkflow (cron: 0 6 * * * - daily at 06:00 UTC)
#   - MLTrainingWorkflow (on-demand or cron: 0 0 * * 0 - weekly Sunday)
# =============================================================================

set -euo pipefail

# Configuration
TEMPORAL_SERVER="${TEMPORAL_SERVER:-localhost:7233}"
NAMESPACE="${NAMESPACE:-default}"
TASK_QUEUE="${TASK_QUEUE:-analytics-worker}"
TEMPORAL_CLI="${TEMPORAL_CLI:-temporal}"

# Validate services
check_temporal_server() {
    echo "Checking Temporal server connectivity at $TEMPORAL_SERVER..."
    if ! timeout 5 bash -c "exec 3<>/dev/tcp/$TEMPORAL_SERVER; exec 3>&-; exec 3<&-" 2>/dev/null; then
        echo "ERROR: Cannot reach Temporal server at $TEMPORAL_SERVER"
        echo "Start Temporal with: docker-compose up -d temporal"
        exit 1
    fi
    echo "✓ Temporal server reachable"
}

# Start hourly rollup workflow
start_hourly_rollup() {
    echo "Starting HourlyRollupWorkflow with cron schedule '5 * * * *'..."
    
    WORKFLOW_ID="hourly-rollup-$(date +%s)"
    CRON_SCHEDULE="5 * * * *"
    
    $TEMPORAL_CLI workflow start \
        --server "$TEMPORAL_SERVER" \
        --namespace "$NAMESPACE" \
        --workflow-type HourlyRollupWorkflow \
        --task-queue "$TASK_QUEUE" \
        --workflow-id "$WORKFLOW_ID" \
        --cron "$CRON_SCHEDULE" \
        --input '{
            "run_id": "'$WORKFLOW_ID'",
            "regions": ["us-east-1", "eu-west-1", "apac-1"]
        }'
    
    echo "✓ HourlyRollupWorkflow registered (ID: $WORKFLOW_ID)"
    echo "  Execution: Every hour at minute 05"
    echo "  Cron: $CRON_SCHEDULE"
}

# Start daily SLA workflow
start_daily_sla() {
    echo "Starting DailySLAWorkflow with cron schedule '0 6 * * *'..."
    
    WORKFLOW_ID="daily-sla-$(date +%s)"
    CRON_SCHEDULE="0 6 * * *"
    TOMORROW=$(date -d "+1 day" +%Y-%m-%d)
    
    $TEMPORAL_CLI workflow start \
        --server "$TEMPORAL_SERVER" \
        --namespace "$NAMESPACE" \
        --workflow-type DailySLAWorkflow \
        --task-queue "$TASK_QUEUE" \
        --workflow-id "$WORKFLOW_ID" \
        --cron "$CRON_SCHEDULE" \
        --input '{
            "run_id": "'$WORKFLOW_ID'",
            "date": "'$TOMORROW'"
        }'
    
    echo "✓ DailySLAWorkflow registered (ID: $WORKFLOW_ID)"
    echo "  Execution: Daily at 06:00 UTC"
    echo "  Cron: $CRON_SCHEDULE"
}

# Start weekly ML training workflow
start_weekly_ml_training() {
    echo "Starting MLTrainingWorkflow with cron schedule '0 0 * * 0' (weekly Sunday)..."
    
    WORKFLOW_ID="ml-training-weekly-$(date +%s)"
    CRON_SCHEDULE="0 0 * * 0"
    TODAY=$(date +%Y-%m-%d)
    
    $TEMPORAL_CLI workflow start \
        --server "$TEMPORAL_SERVER" \
        --namespace "$NAMESPACE" \
        --workflow-type MLTrainingWorkflow \
        --task-queue "$TASK_QUEUE" \
        --workflow-id "$WORKFLOW_ID" \
        --cron "$CRON_SCHEDULE" \
        --input '{
            "run_id": "'$WORKFLOW_ID'",
            "model_name": "chain_failure_predictor",
            "training_date": "'$TODAY'"
        }'
    
    echo "✓ MLTrainingWorkflow registered (ID: $WORKFLOW_ID)"
    echo "  Execution: Weekly on Sunday at 00:00 UTC"
    echo "  Cron: $CRON_SCHEDULE"
}

# List active workflows
list_workflows() {
    echo ""
    echo "Active Temporal workflows:"
    $TEMPORAL_CLI workflow list \
        --server "$TEMPORAL_SERVER" \
        --namespace "$NAMESPACE" \
        --query "ExecutionStatus='RUNNING'"
}

# Main execution
main() {
    echo "=============================================================================
    Phase 3.15: Temporal Workflow Initialization
    ============================================================================="
    echo "Server: $TEMPORAL_SERVER"
    echo "Namespace: $NAMESPACE"
    echo "Task Queue: $TASK_QUEUE"
    echo ""
    
    check_temporal_server
    
    echo ""
    start_hourly_rollup
    echo ""
    start_daily_sla
    echo ""
    start_weekly_ml_training
    
    list_workflows
    
    echo ""
    echo "=============================================================================
    ✓ All workflows registered successfully!
    
    Monitor workflow execution:
      temporal workflow list --server $TEMPORAL_SERVER --namespace $NAMESPACE
    
    View workflow details:
      temporal workflow describe --server $TEMPORAL_SERVER --namespace $NAMESPACE --workflow-id <WORKFLOW_ID>
    
    Web UI (if available):
      http://localhost:8080
    ============================================================================="
}

main "$@"
