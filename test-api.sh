#!/bin/bash

# Test script for Dynamic Parameters & Measures API
echo "🧪 Testing Dynamic Parameters & Measures API"
echo "============================================"

BASE_URL="http://localhost:8080"

# Test 1: Parameter Schema
echo ""
echo "📋 Test 1: Parameter Schema"
echo "---------------------------"
curl -s -X GET "$BASE_URL/api/parameters/schema" | jq '.' || echo "Endpoint not available (expected if server not running)"

# Test 2: Available Values
echo ""
echo "📊 Test 2: Available Values (City)"
echo "----------------------------------"
curl -s -X GET "$BASE_URL/api/parameters/dimension/city/values" | jq '.' || echo "Endpoint not available (expected if server not running)"

# Test 3: Measure Generation
echo ""
echo "🧪 Test 3: Measure Generation"
echo "-----------------------------"
curl -s -X POST "$BASE_URL/api/measures/generate" \
  -H "Content-Type: application/json" \
  -d '{
    "source_table": "orders",
    "source_column": "status",
    "measure_type": "count",
    "filters": ["city = '\''{FILTER_PARAMS.city}'\''"]
  }' | jq '.' || echo "Endpoint not available (expected if server not running)"

# Test 4: Measure Catalog
echo ""
echo "📚 Test 4: Measure Catalog"
echo "--------------------------"
curl -s -X GET "$BASE_URL/api/measures/catalog" | jq '.' || echo "Endpoint not available (expected if server not running)"

# Test 5: Measure Validation
echo ""
echo "✅ Test 5: Measure Validation"
echo "-----------------------------"
curl -s -X POST "$BASE_URL/api/measures/validate" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "total_processing_orders",
    "sql": "CASE WHEN status = '\''processing'\'' THEN 1 ELSE 0 END",
    "type": "count"
  }' | jq '.' || echo "Endpoint not available (expected if server not running)"

# Test 6: Dynamic Query
echo ""
echo "🔍 Test 6: Dynamic Query"
echo "------------------------"
curl -s -X POST "$BASE_URL/api/v1/dynamic/query" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "SELECT * FROM orders WHERE city = $1",
    "parameters": ["New York"],
    "measures": ["total_orders"]
  }' | jq '.' || echo "Endpoint not available (expected if server not running)"

echo ""
echo "🎉 API Tests Completed!"
echo "======================="
echo "Note: If endpoints show 'not available', start the backend server first:"
echo "cd backend && go run main.go"
echo ""
echo "Expected server startup command:"
echo "go run main.go --config=config.yaml"
