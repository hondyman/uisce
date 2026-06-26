#!/bin/bash
# Calendar Service - Build & Test Script

set -e

cd "$(dirname "$0")"

echo "📦 Calendar Service - Build & Test"
echo "===================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Check Go version
echo -e "${YELLOW}[1/5]${NC} Checking Go version..."
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "✓ Go version: $GO_VERSION"
echo ""

# Step 2: Run go mod tidy
echo -e "${YELLOW}[2/5]${NC} Running go mod tidy..."
go mod tidy
echo "✓ Dependencies updated"
echo ""

# Step 3: Format code
echo -e "${YELLOW}[3/5]${NC} Formatting code..."
go fmt ./...
echo "✓ Code formatted"
echo ""

# Step 4: Lint check
echo -e "${YELLOW}[4/5]${NC} Running lint checks..."
if command -v golangci-lint &> /dev/null; then
    golangci-lint run ./internal/api ./internal/availability ./internal/server || true
else
    echo "⚠️  golangci-lint not installed, skipping lint checks"
fi
echo ""

# Step 5: Build
echo -e "${YELLOW}[5/5]${NC} Building calendar-service..."
if go build -o ./bin/calendar-service ./cmd/server 2>&1; then
    echo -e "${GREEN}✓ Build successful!${NC}"
    echo ""
    echo "📊 Build Results:"
    echo "  Binary: ./bin/calendar-service"
    size=$(du -h ./bin/calendar-service | cut -f1)
    echo "  Size: $size"
    echo ""
    echo "🚀 Run the service:"
    echo "  ./bin/calendar-service -port 8080 -loglevel info"
    echo ""
    echo "✅ Build complete!"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi
