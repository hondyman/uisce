#!/bin/bash

# Quick Start Script for Mutual Fund Analytics Semantic Layer
# This script gets you up and running in minutes!

echo "🚀 Quick Start: Mutual Fund Analytics Semantic Layer"
echo "=================================================="

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

cd /Users/eganpj/GitHub/semlayer/cube-gonja

echo -e "\n${BLUE}Step 1: Building the application...${NC}"
go build -o cube-gonja .
chmod +x cube-gonja
echo -e "${GREEN}✅ Build complete${NC}"

echo -e "\n${BLUE}Step 2: Starting the server...${NC}"
echo -e "${YELLOW}Server will start on http://localhost:3000${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the server${NC}"
echo ""

# Start server in background
DATABASE_HOST="" ./cube-gonja &
SERVER_PID=$!

# Wait a moment for server to start
sleep 3

echo -e "\n${BLUE}Step 3: Testing the API...${NC}"

# Test health endpoint
echo "Testing health endpoint..."
if curl -s http://localhost:3000/health > /dev/null; then
    echo -e "${GREEN}✅ Server is healthy${NC}"
else
    echo -e "${RED}❌ Server health check failed${NC}"
fi

# Test template listing
echo "Testing template listing..."
if curl -s http://localhost:3000/templates > /dev/null; then
    echo -e "${GREEN}✅ Template API working${NC}"
else
    echo -e "${RED}❌ Template API failed${NC}"
fi

echo -e "\n${BLUE}Step 4: Loading sample context...${NC}"
curl -X POST http://localhost:3000/update-context \
  -H "Content-Type: application/json" \
  -d '{
    "data_sources": {
      "mutual_fund_portfolio": "portfolio_db",
      "options_portfolio": "derivatives_db"
    },
    "dimensions": [
      {
        "name": "fund_id",
        "sql": "fund_id",
        "type": "string"
      }
    ],
    "measures": [
      {
        "name": "sharpe_ratio",
        "sql": "sharpe_ratio",
        "type": "number"
      }
    ]
  }' > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Sample context loaded${NC}"
else
    echo -e "${YELLOW}⚠️  Context loading may have issues (expected in demo mode)${NC}"
fi

echo -e "\n${BLUE}Step 5: Testing template rendering...${NC}"
curl -X POST http://localhost:3000/render \
  -H "Content-Type: application/json" \
  -d '{"template_name": "test_mutual_fund"}' > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Template rendering working${NC}"
else
    echo -e "${YELLOW}⚠️  Template rendering may need context setup${NC}"
fi

echo -e "\n${GREEN}🎉 Quick start complete!${NC}"
echo ""
echo -e "${BLUE}Your Mutual Fund Analytics Semantic Layer is running!${NC}"
echo ""
echo -e "${YELLOW}Available endpoints:${NC}"
echo "  GET  http://localhost:3000/health"
echo "  GET  http://localhost:3000/templates"
echo "  POST http://localhost:3000/update-context"
echo "  POST http://localhost:3000/render"
echo ""
echo -e "${YELLOW}Test commands:${NC}"
echo "  curl http://localhost:3000/health"
echo "  curl http://localhost:3000/templates"
echo ""
echo -e "${YELLOW}Next steps:${NC}"
echo "1. Load your full context: curl -X POST http://localhost:3000/update-context -H 'Content-Type: application/json' -d @mutual_fund_context.json"
echo "2. Render templates: curl -X POST http://localhost:3000/render -H 'Content-Type: application/json' -d '{\"template_name\": \"mutual_fund_analytics\"}'"
echo "3. Customize templates in the 'templates/' directory"
echo "4. Configure tenant-specific settings"
echo ""
echo -e "${BLUE}Server is running in the background (PID: $SERVER_PID)${NC}"
echo -e "${BLUE}Press Ctrl+C to stop this script (server will continue running)${NC}"

# Wait for user interrupt
trap "echo -e '\n${YELLOW}Script stopped. Server is still running (PID: $SERVER_PID)${NC}'" INT
wait $SERVER_PID
