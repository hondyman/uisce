#!/bin/bash
# Event-Router: Complete End-to-End Test Script
# Copy and paste commands to validate the entire event routing pipeline

set -e

# ============================================================================
# CONFIGURATION
# ============================================================================

TENANT_ID="910638ba-a459-4a3f-bb2d-78391b0595f6"
DATASOURCE_ID="982aef38-418f-46dc-acd0-35fe8f3b97b0"
DB_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
HASURA_URL="http://localhost:8081"
HASURA_ADMIN_SECRET="${HASURA_ADMIN_SECRET:-your-admin-secret}"
CORE_APP_URL="http://localhost:29080"
EVENT_ROUTER_URL="http://localhost:8081"
RABBITMQ_MGMT_URL="http://localhost:15672"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

log_info() {
  echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
  echo -e "${GREEN}✅ $1${NC}"
}

log_warn() {
  echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
  echo -e "${RED}❌ $1${NC}"
}

check_service() {
  local name=$1
  local url=$2
  log_info "Checking $name..."
  if curl -s "$url" > /dev/null 2>&1; then
    log_success "$name is running"
    return 0
  else
    log_error "$name is not responding at $url"
    return 1
  fi
}

# ============================================================================
# SECTION 1: PRE-FLIGHT CHECKS
# ============================================================================

section_1() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 1: PRE-FLIGHT CHECKS${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_info "Checking all services are running..."
  
  check_service "PostgreSQL" "localhost:5432" || return 1
  check_service "Hasura GraphQL" "$HASURA_URL" || return 1
  check_service "Event-Router" "$EVENT_ROUTER_URL/health" || return 1
  check_service "RabbitMQ Management" "$RABBITMQ_MGMT_URL" || return 1
  check_service "Core App" "$CORE_APP_URL" || return 1

  log_success "All services are running!"
}

# ============================================================================
# SECTION 2: VERIFY DATABASE MIGRATIONS
# ============================================================================

section_2() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 2: VERIFY DATABASE MIGRATIONS${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_info "Checking bo_events table exists..."
  psql "$DB_URL" -c "\dt bo_events" > /dev/null 2>&1 || {
    log_error "bo_events table not found. Run migrations first:"
    log_warn "psql $DB_URL -f backend/migrations/000050_create_bo_events_table.sql"
    return 1
  }
  log_success "bo_events table exists"

  log_info "Checking event_configs table exists..."
  psql "$DB_URL" -c "\dt event_configs" > /dev/null 2>&1 || {
    log_error "event_configs table not found. Run migrations first:"
    log_warn "psql $DB_URL -f backend/migrations/000051_create_event_configs_table.sql"
    return 1
  }
  log_success "event_configs table exists"

  # Show table info
  log_info "bo_events columns:"
  psql "$DB_URL" -c "\d bo_events" | head -20

  log_info "event_configs columns:"
  psql "$DB_URL" -c "\d event_configs" | head -20
}

# ============================================================================
# SECTION 3: CREATE TEST ROUTING CONFIG
# ============================================================================

section_3() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 3: CREATE TEST ROUTING CONFIG${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  local config_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
  local queue_name="test_queue_$(date +%s)"

  log_info "Creating routing config..."
  log_info "  Config ID: $config_id"
  log_info "  Queue Name: $queue_name"
  log_info "  Tenant ID: $TENANT_ID"

  psql "$DB_URL" << EOF
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  '$config_id'::uuid,
  '$TENANT_ID'::uuid,
  'fieldchange',
  'test_entity',
  NULL,
  '{}',
  '$queue_name',
  NOW()
);
EOF

  log_success "Routing config created!"
  
  log_info "Verifying config in database..."
  psql "$DB_URL" << EOF
SELECT id, event_type, bo_type, route_queue FROM event_configs 
WHERE id = '$config_id'::uuid;
EOF

  echo "QUEUE_NAME=$queue_name" > /tmp/test_config.env
  echo "CONFIG_ID=$config_id" >> /tmp/test_config.env
}

# ============================================================================
# SECTION 4: TRIGGER TEST EVENT
# ============================================================================

section_4() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 4: TRIGGER TEST EVENT${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  # Source queue name from previous section
  source /tmp/test_config.env 2>/dev/null || QUEUE_NAME="test_queue_unknown"

  log_info "Posting test event to core app..."
  log_info "  BO Type: test_entity"
  log_info "  BO ID: test-001"
  log_info "  Field: status"
  log_info "  Change: pending → approved"

  RESPONSE=$(curl -s -X POST "$CORE_APP_URL/events" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
    -d '{
      "bo_type": "test_entity",
      "bo_id": "test-001",
      "event_type": "fieldchange",
      "field_name": "status",
      "old_value": "pending",
      "new_value": "approved",
      "changed_by": "test-user"
    }')

  log_info "Response: $RESPONSE"
  
  if echo "$RESPONSE" | grep -q "success"; then
    log_success "Event posted successfully!"
  else
    log_error "Event post failed!"
    return 1
  fi

  # Wait for async processing
  log_info "Waiting 2 seconds for async processing..."
  sleep 2
}

# ============================================================================
# SECTION 5: CHECK EVENT IN DATABASE
# ============================================================================

section_5() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 5: CHECK EVENT IN DATABASE${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_info "Querying bo_events table..."

  EVENT_COUNT=$(psql "$DB_URL" -t -c \
    "SELECT COUNT(*) FROM bo_events WHERE bo_type = 'test_entity' AND bo_id = 'test-001';")

  log_info "Events found: $EVENT_COUNT"

  if [ "$EVENT_COUNT" -gt 0 ]; then
    log_success "Event stored in bo_events!"
    
    log_info "Event details:"
    psql "$DB_URL" << EOF
SELECT id, bo_type, bo_id, field_name, old_value, new_value, changed_by, changed_at 
FROM bo_events 
WHERE bo_type = 'test_entity' AND bo_id = 'test-001' 
ORDER BY changed_at DESC 
LIMIT 3;
EOF
  else
    log_error "No events found in database!"
    return 1
  fi
}

# ============================================================================
# SECTION 6: CHECK RABBITMQ QUEUE
# ============================================================================

section_6() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 6: CHECK RABBITMQ QUEUE${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  source /tmp/test_config.env 2>/dev/null || QUEUE_NAME="test_queue_unknown"

  log_info "Fetching queue info from RabbitMQ..."
  log_info "Queue name: $QUEUE_NAME"

  QUEUE_INFO=$(curl -s -u guest:guest \
    "$RABBITMQ_MGMT_URL/api/queues/%2F/$QUEUE_NAME" 2>/dev/null || echo '{}')

  MESSAGE_COUNT=$(echo "$QUEUE_INFO" | jq '.messages' 2>/dev/null || echo 0)

  log_info "Messages in queue: $MESSAGE_COUNT"

  if [ "$MESSAGE_COUNT" -gt 0 ]; then
    log_success "Message(s) routed to RabbitMQ!"
    
    log_info "Queue details:"
    echo "$QUEUE_INFO" | jq '.
  else
    log_warn "No messages in queue yet. Checking if queue exists..."
    
    QUEUE_EXISTS=$(echo "$QUEUE_INFO" | jq '.messages' 2>/dev/null)
    
    if [ -z "$QUEUE_EXISTS" ]; then
      log_error "Queue does not exist! Event routing may have failed."
      log_info "Check event-router logs:"
      log_warn "docker-compose logs event-router"
      return 1
    else
      log_info "Queue exists but is empty. Event may not have been routed."
      return 1
    fi
  fi
}

# ============================================================================
# SECTION 7: CHECK EVENT-ROUTER LOGS
# ============================================================================

section_7() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 7: CHECK EVENT-ROUTER LOGS${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_info "Fetching last 20 lines from event-router logs..."

  docker-compose logs --tail=20 event-router || {
    log_warn "Could not fetch docker logs. Event-router may be running outside Docker."
  }
}

# ============================================================================
# SECTION 8: VERIFY HASURA CONFIG SYNC
# ============================================================================

section_8() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 8: VERIFY HASURA CONFIG SYNC${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_info "Querying event_configs via Hasura GraphQL..."

  GRAPHQL_RESPONSE=$(curl -s -X POST "$HASURA_URL/v1/graphql" \
    -H "Content-Type: application/json" \
    -H "X-Hasura-Admin-Secret: $HASURA_ADMIN_SECRET" \
    -d '{
      "query": "query { event_configs(where: {tenant_id: {_eq: \"'$TENANT_ID'\"}}) { id, event_type, bo_type, route_queue } }"
    }')

  CONFIG_COUNT=$(echo "$GRAPHQL_RESPONSE" | jq '.data.event_configs | length' 2>/dev/null || echo 0)

  log_info "Configs found via Hasura: $CONFIG_COUNT"

  if [ "$CONFIG_COUNT" -gt 0 ]; then
    log_success "Event-router can see configs via Hasura!"
    log_info "Configs:"
    echo "$GRAPHQL_RESPONSE" | jq '.data.event_configs'
  else
    log_warn "No configs found via Hasura. Check Hasura permissions and table tracking."
  fi
}

# ============================================================================
# SECTION 9: TEST FILTER LOGIC (Numeric)
# ============================================================================

section_9() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 9: TEST FILTER LOGIC (NUMERIC)${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  local config_id=$(uuidgen | tr '[:upper:]' '[:lower:]')
  local queue_name="test_numeric_queue_$(date +%s)"

  log_info "Creating numeric filter config (balance > 100)..."
  log_info "  Queue: $queue_name"

  psql "$DB_URL" << EOF
INSERT INTO event_configs (
  id, tenant_id, event_type, bo_type, field_name, filter_json, route_queue, created_at
) VALUES (
  '$config_id'::uuid,
  '$TENANT_ID'::uuid,
  'fieldchange',
  'accounts',
  'balance',
  '{"new_value": {"min_value": 100}}'::jsonb,
  '$queue_name',
  NOW()
);
EOF

  log_success "Config created!"

  # Trigger event that PASSES filter
  log_info "Triggering event that PASSES filter (balance: 50 → 500)..."
  curl -s -X POST "$CORE_APP_URL/events" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
    -d '{
      "bo_type": "accounts",
      "bo_id": "acct-123",
      "event_type": "fieldchange",
      "field_name": "balance",
      "old_value": "50",
      "new_value": "500",
      "changed_by": "system"
    }' > /dev/null

  sleep 1

  # Trigger event that FAILS filter
  log_info "Triggering event that FAILS filter (balance: 500 → 60)..."
  curl -s -X POST "$CORE_APP_URL/events" \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT_ID" \
    -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
    -d '{
      "bo_type": "accounts",
      "bo_id": "acct-123",
      "event_type": "fieldchange",
      "field_name": "balance",
      "old_value": "500",
      "new_value": "60",
      "changed_by": "system"
    }' > /dev/null

  sleep 1

  log_info "Checking queue messages..."
  QUEUE_INFO=$(curl -s -u guest:guest "$RABBITMQ_MGMT_URL/api/queues/%2F/$queue_name" 2>/dev/null || echo '{}')
  MESSAGE_COUNT=$(echo "$QUEUE_INFO" | jq '.messages' 2>/dev/null || echo 0)

  if [ "$MESSAGE_COUNT" -eq 1 ]; then
    log_success "Numeric filter working correctly! (1 message, 1 filtered out)"
  else
    log_warn "Expected 1 message, got $MESSAGE_COUNT"
  fi
}

# ============================================================================
# SECTION 10: FINAL REPORT
# ============================================================================

section_10() {
  echo -e "\n${BLUE}========================================${NC}"
  echo -e "${BLUE}SECTION 10: FINAL REPORT${NC}"
  echo -e "${BLUE}========================================${NC}\n"

  log_success "End-to-end test completed!"

  log_info "Summary of created resources:"
  log_info "  - Event configs: $(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM event_configs WHERE tenant_id = '$TENANT_ID'::uuid;")"
  log_info "  - Events stored: $(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM bo_events WHERE tenant_id = '$TENANT_ID'::uuid;")"
  
  log_info "RabbitMQ queues:"
  curl -s -u guest:guest "$RABBITMQ_MGMT_URL/api/queues" | jq -r '.[] | "  - \(.name): \(.messages) messages"'

  log_info "Next steps:"
  log_info "  1. Deploy this to production with strong secrets"
  log_info "  2. Build downstream consumers for each queue"
  log_info "  3. Add monitoring and alerting"
  log_info "  4. Test with high event volumes"
}

# ============================================================================
# MAIN EXECUTION
# ============================================================================

main() {
  echo -e "\n${BLUE}╔════════════════════════════════════════╗${NC}"
  echo -e "${BLUE}║  Event-Router End-to-End Test Suite   ║${NC}"
  echo -e "${BLUE}╚════════════════════════════════════════╝${NC}\n"

  section_1 || return 1
  section_2 || return 1
  section_3 || return 1
  section_4 || return 1
  section_5 || return 1
  section_6 || return 1
  section_7 || return 1
  section_8 || return 1
  section_9 || return 1
  section_10

  echo -e "\n${GREEN}════════════════════════════════════════${NC}"
  echo -e "${GREEN}         ✅ TEST COMPLETED             ${NC}"
  echo -e "${GREEN}════════════════════════════════════════${NC}\n"
}

# Run main function
main "$@"
