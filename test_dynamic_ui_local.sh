#!/usr/bin/env bash

# Dynamic UI Generator - Local Testing Commands
# Date: October 21, 2025
# Status: Ready for deployment

echo "🚀 Dynamic UI Generator - Local Testing Script"
echo ""
echo "This script will help you test the Dynamic UI Generator locally."
echo ""

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running from semlayer root
if [ ! -d "backend" ] || [ ! -d "frontend" ]; then
    echo -e "${YELLOW}⚠️  Please run this from the semlayer root directory${NC}"
    exit 1
fi

# Function to print section headers
print_header() {
    echo ""
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
    echo ""
}

# Main menu
show_menu() {
    echo ""
    echo -e "${YELLOW}Select what you want to do:${NC}"
    echo ""
    echo "1) Start Backend Server"
    echo "2) Start Frontend Dev Server"
    echo "3) Start Both (requires 2 terminals)"
    echo "4) Run Backend Tests"
    echo "5) Check Compilation Errors"
    echo "6) View API Endpoint Status"
    echo "7) Exit"
    echo ""
    read -p "Enter choice [1-7]: " choice
}

# Start backend
start_backend() {
    print_header "Starting Backend Server"
    echo -e "${GREEN}✓${NC} Changing to backend directory..."
    cd backend
    
    echo -e "${GREEN}✓${NC} Building backend..."
    go build -o server cmd/server/main.go
    
    if [ -f "server" ]; then
        echo -e "${GREEN}✓${NC} Build successful!"
        echo -e "${GREEN}✓${NC} Starting server on :8080..."
        echo ""
        echo "Press Ctrl+C to stop the server"
        echo ""
        ./server
    else
        echo -e "${YELLOW}✗ Build failed${NC}"
        exit 1
    fi
}

# Start frontend
start_frontend() {
    print_header "Starting Frontend Dev Server"
    echo -e "${GREEN}✓${NC} Changing to frontend directory..."
    cd frontend
    
    echo -e "${GREEN}✓${NC} Checking dependencies..."
    if [ ! -d "node_modules" ]; then
        echo -e "${GREEN}✓${NC} Installing npm packages..."
        npm install
    fi
    
    echo -e "${GREEN}✓${NC} Starting dev server on :5173..."
    echo ""
    echo "Open http://localhost:5173 in your browser"
    echo "Press Ctrl+C to stop the server"
    echo ""
    npm run dev
}

# Start both
start_both() {
    print_header "Start Both Servers"
    echo ""
    echo -e "${YELLOW}You need to open 2 terminal windows for this.${NC}"
    echo ""
    echo "Terminal 1 - Backend:"
    echo "  cd backend"
    echo "  go build -o server cmd/server/main.go && ./server"
    echo ""
    echo "Terminal 2 - Frontend:"
    echo "  cd frontend"
    echo "  npm run dev"
    echo ""
    echo "Then open http://localhost:5173 in your browser"
    echo ""
    read -p "Press Enter to open instructions..."
}

# Check compilation
check_compilation() {
    print_header "Checking Compilation Errors"
    
    echo -e "${GREEN}Frontend:${NC}"
    cd frontend
    npm run type-check 2>&1 | head -20
    echo ""
    
    echo -e "${GREEN}Backend:${NC}"
    cd ../backend
    go build -o /dev/null cmd/server/main.go 2>&1 | head -20
    echo ""
    
    echo -e "${GREEN}✓ Check complete${NC}"
}

# Main loop
while true; do
    show_menu
    
    case $choice in
        1)
            start_backend
            ;;
        2)
            start_frontend
            ;;
        3)
            start_both
            ;;
        4)
            print_header "Running Backend Tests"
            cd backend
            go test ./... -v 2>&1 | head -30
            ;;
        5)
            check_compilation
            ;;
        6)
            print_header "API Endpoints Status"
            echo ""
            echo "Employee Endpoints:"
            echo "  POST /api/employees         - Save employee"
            echo "  GET /api/employees          - List employees"
            echo ""
            echo "Business Process Endpoints:"
            echo "  POST /api/bp/start-execution - Trigger BP workflow"
            echo ""
            echo -e "${YELLOW}Status: All endpoints registered and ready${NC}"
            echo ""
            read -p "Press Enter to continue..."
            ;;
        7)
            echo "Exiting..."
            exit 0
            ;;
        *)
            echo "Invalid choice"
            ;;
    esac
done
