#!/bin/bash

# Generate Embeddings Script
# This script builds and runs the embedding generation tool

set -e

echo "🔨 Building embedding generator..."
cd backend/cmd/generate-embeddings
go build -o ../../../bin/generate-embeddings .
cd ../../..

echo "✅ Build complete!"
echo ""
echo "Usage examples:"
echo "  ./bin/generate-embeddings --tenant=YOUR_TENANT_ID --datasource=YOUR_DATASOURCE_ID"
echo "  ./bin/generate-embeddings --tenant=YOUR_TENANT_ID --datasource=YOUR_DATASOURCE_ID --db='postgres://user:pass@localhost:5432/dbname'"
echo "  ./bin/generate-embeddings --tenant=YOUR_TENANT_ID --datasource=YOUR_DATASOURCE_ID --api-key='your-gemini-key'"
echo ""
echo "Environment variables:"
echo "  DATABASE_URL     - PostgreSQL connection string (default: from env)"
echo "  GEMINI_API_KEY   - Your Gemini API key (default: from env)"
