#!/bin/bash

# SemLayer API Gateway Test Script
# This script tests the REST API endpoints

API_BASE="http://localhost:8080"

echo "Testing SemLayer API Gateway..."
echo "================================="

# Test health check
echo "1. Testing health check..."
curl -s -X GET "$API_BASE/health" | jq . || echo "Health check failed"

echo -e "\n2. Testing business term search..."
curl -s -X POST "$API_BASE/api/search/business-terms" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "customer",
    "limit": 5,
    "tenant_id": "default"
  }' | jq . || echo "Search failed"

echo -e "\n3. Testing business term validation..."
curl -s -X POST "$API_BASE/api/validate/business-term" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Business Term",
    "description": "A test business term",
    "category": "Test Category"
  }' | jq . || echo "Validation failed"

echo -e "\n4. Testing semantic lineage..."
curl -s "$API_BASE/api/lineage/semantic?node_id=test_node&tenant_id=default" | jq . || echo "Lineage failed"

echo -e "\n5. Testing GraphQL proxy..."
curl -s -X POST "$API_BASE/api/graphql" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "query { business_terms(limit: 1) { id name } }"
  }' | jq . || echo "GraphQL proxy failed"

echo -e "\n6. Testing OpenAPI spec..."
curl -s -I "$API_BASE/api/openapi.yaml" | head -1 || echo "OpenAPI spec not accessible"

echo -e "\n7. Testing Swagger UI..."
curl -s -I "$API_BASE/docs/" | head -1 || echo "Swagger UI not accessible"

echo -e "\nAPI Gateway tests completed!"
