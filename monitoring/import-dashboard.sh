#!/bin/bash

# Dynatrace Dashboard Import Script for Semlayer Platform
# This script imports the governance dashboard and sets up basic SLOs

set -e

# Configuration
DYNATRACE_ENV="${DYNATRACE_ENV:-your-environment}"
DYNATRACE_API_TOKEN="${DYNATRACE_API_TOKEN:-your-api-token}"
DASHBOARD_FILE="monitoring/dynatrace-dashboard.json"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Importing Semlayer Governance Dashboard to Dynatrace${NC}"

# Check if dashboard file exists
if [ ! -f "$DASHBOARD_FILE" ]; then
    echo -e "${RED}❌ Dashboard file not found: $DASHBOARD_FILE${NC}"
    exit 1
fi

# Import dashboard
echo -e "${YELLOW}📊 Importing dashboard...${NC}"
IMPORT_RESPONSE=$(curl -s -X POST \
    "https://$DYNATRACE_ENV.live.dynatrace.com/api/config/v1/dashboards" \
    -H "Authorization: Api-Token $DYNATRACE_API_TOKEN" \
    -H "Content-Type: application/json" \
    -d @"$DASHBOARD_FILE")

if echo "$IMPORT_RESPONSE" | grep -q "id"; then
    DASHBOARD_ID=$(echo "$IMPORT_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo -e "${GREEN}✅ Dashboard imported successfully!${NC}"
    echo -e "${GREEN}📋 Dashboard ID: $DASHBOARD_ID${NC}"
    echo -e "${GREEN}🔗 View at: https://$DYNATRACE_ENV.live.dynatrace.com/#dashboard;$DASHBOARD_ID${NC}"
else
    echo -e "${RED}❌ Failed to import dashboard${NC}"
    echo -e "${RED}Response: $IMPORT_RESPONSE${NC}"
    exit 1
fi

echo -e "${GREEN}🎯 Next Steps:${NC}"
echo "1. Verify dashboard loads correctly"
echo "2. Update entity selectors if needed"
echo "3. Configure SLOs and alerts"
echo "4. Set up Davis AI anomaly detection"
echo "5. Share with SRE and governance teams"

echo -e "${GREEN}✨ Dashboard import complete!${NC}"
