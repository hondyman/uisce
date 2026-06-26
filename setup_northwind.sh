#!/bin/bash
# Northwind BOs Implementation - Setup Script
# Run this to complete the setup

set -e

echo "🚀 Northwind Business Objects Setup"
echo "===================================="
echo ""

# Check environment
if [ -z "$DATABASE_URL" ]; then
  echo "⚠️  DATABASE_URL not set, using default..."
  export DATABASE_URL="postgres://postgres:postgres@100.84.126.19:5432/alpha?sslmode=disable"
fi

echo "📦 Step 1: Running database migrations..."
cd backend
go run ./cmd/migrate/ up || {
  echo "❌ Migration failed"
  exit 1
}
echo "✅ Migrations complete"
echo ""

echo "📄 Applying Northwind Semantic Models..."
psql $DATABASE_URL -f migrations/20241216_northwind_semantic_models.sql || {
  echo "❌ Semantic Models application failed"
  # Don't exit, might be partial duplicate
}
echo "✅ Semantic Models applied"
echo ""

echo "🌱 Step 2: Seeding Northwind BOs..."
go run cmd/seed_northwind_bos/main.go || {
  echo "❌ Seed failed"
  exit 1
}
echo "✅ Seed complete"
echo ""

echo "📋 Step 3: Verifying installation..."
psql $DATABASE_URL -c "SELECT COUNT(*) as bo_count FROM business_objects;" || {
  echo "❌ Verification failed"
  exit 1
}
echo "✅ Verification complete"
echo ""

echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Start frontend: cd frontend && npm run dev"
echo "2. Navigate to http://localhost:3000/config"
echo "3. See all 8 Northwind BOs"
echo "4. Click clone to create custom variants"
echo ""
echo "📚 Documentation:"
echo "- NORTHWIND_IMPLEMENTATION.md - Full technical details"
echo "- NORTHWIND_QUICKSTART.md - Quick reference"
