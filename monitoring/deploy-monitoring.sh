#!/bin/bash
# deploy-monitoring.sh - Deploy Semlayer Monitoring Configuration to Dynatrace

set -e

# Configuration
DYNATRACE_ENV="${DYNATRACE_ENV:-your-environment}"
API_TOKEN="${DYNATRACE_API_TOKEN:-your-token}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Validate configuration
if [ "$DYNATRACE_ENV" = "your-environment" ] || [ "$API_TOKEN" = "your-token" ]; then
    echo -e "${RED}❌ Please set DYNATRACE_ENV and DYNATRACE_API_TOKEN environment variables${NC}"
    echo -e "${YELLOW}Example:${NC}"
    echo "export DYNATRACE_ENV='your-environment'"
    echo "export DYNATRACE_API_TOKEN='your-api-token'"
    exit 1
fi

echo -e "${BLUE}🚀 Deploying Semlayer Monitoring Configuration to Dynatrace${NC}"
echo -e "${BLUE}Environment: $DYNATRACE_ENV${NC}"
echo

# Function to make API calls
call_api() {
    local method=$1
    local endpoint=$2
    local data_file=$3
    local description=$4

    echo -e "${YELLOW}📡 $description...${NC}"

    if [ -f "$data_file" ]; then
        response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
            -X "$method" \
            "https://$DYNATRACE_ENV.live.dynatrace.com/api/v2/$endpoint" \
            -H "Authorization: Api-Token $API_TOKEN" \
            -H "Content-Type: application/json" \
            -d @"$data_file")

        http_status=$(echo "$response" | grep "HTTP_STATUS:" | cut -d: -f2)
        response_body=$(echo "$response" | sed '/HTTP_STATUS:/d')

        if [ "$http_status" -ge 200 ] && [ "$http_status" -lt 300 ]; then
            echo -e "${GREEN}✅ $description completed successfully${NC}"
        else
            echo -e "${RED}❌ $description failed (HTTP $http_status)${NC}"
            echo -e "${RED}Response: $response_body${NC}"
            return 1
        fi
    else
        echo -e "${RED}❌ $data_file not found${NC}"
        return 1
    fi
}

# Deploy custom metrics
call_api "POST" "metrics/ingest" "monitoring/custom-metrics.json" "Creating custom metrics"

# Deploy SLOs
call_api "POST" "slo" "monitoring/slos.json" "Creating SLOs"

# Deploy alert rules
call_api "POST" "settings/objects" "monitoring/alerts.json" "Creating alert rules"

# Deploy business events
call_api "POST" "events/ingest" "monitoring/business-events.json" "Creating business events"

# Import dashboard
echo -e "${YELLOW}📊 Importing dashboard...${NC}"
dashboard_response=$(curl -s -w "\nHTTP_STATUS:%{http_code}" \
    -X POST \
    "https://$DYNATRACE_ENV.live.dynatrace.com/api/config/v1/dashboards" \
    -H "Authorization: Api-Token $API_TOKEN" \
    -H "Content-Type: application/json" \
    -d @monitoring/dynatrace-dashboard.json)

dashboard_http_status=$(echo "$dashboard_response" | grep "HTTP_STATUS:" | cut -d: -f2)
dashboard_body=$(echo "$dashboard_response" | sed '/HTTP_STATUS:/d')

if [ "$dashboard_http_status" -ge 200 ] && [ "$dashboard_http_status" -lt 300 ]; then
    dashboard_id=$(echo "$dashboard_body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✅ Dashboard imported successfully${NC}"
    echo -e "${GREEN}📋 Dashboard ID: $dashboard_id${NC}"
    echo -e "${GREEN}🔗 View at: https://$DYNATRACE_ENV.live.dynatrace.com/#dashboard;$dashboard_id${NC}"
else
    echo -e "${RED}❌ Dashboard import failed (HTTP $dashboard_http_status)${NC}"
    echo -e "${RED}Response: $dashboard_body${NC}"
    exit 1
fi

echo
echo -e "${GREEN}🎉 Semlayer Monitoring Configuration Deployed Successfully!${NC}"
echo
echo -e "${BLUE}📋 Summary:${NC}"
echo "✅ Custom metrics configured"
echo "✅ SLOs created with Davis AI anomaly detection"
echo "✅ Alert rules active"
echo "✅ Business events defined"
echo "✅ Dashboard imported and ready"
echo
echo -e "${BLUE}🔗 Quick Links:${NC}"
echo "- Dashboard: https://$DYNATRACE_ENV.live.dynatrace.com/#dashboard;$dashboard_id"
echo "- Metrics: https://$DYNATRACE_ENV.live.dynatrace.com/#metrics"
echo "- SLOs: https://$DYNATRACE_ENV.live.dynatrace.com/#slo"
echo
echo -e "${YELLOW}💡 Next Steps:${NC}"
echo "1. Verify metrics are flowing from your application"
echo "2. Adjust SLO targets based on your baseline performance"
echo "3. Configure alert notification channels"
echo "4. Share dashboard with SRE and governance teams"
echo
echo -e "${GREEN}🚀 Your governance platform is now fully observable!${NC}"
