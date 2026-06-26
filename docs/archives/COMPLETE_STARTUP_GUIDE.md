# 🚀 Complete System Startup Guide

## Quick Start - All in One

### Option 1: Run Everything with Bash Script (Recommended)

```bash
# Make script executable
chmod +x START_FULL_SYSTEM.sh

# Run the complete system
bash START_FULL_SYSTEM.sh
```

**What this does:**
- ✅ Checks all prerequisites (Go, Node.js, npm, PostgreSQL)
- ✅ Verifies port availability (8080, 5173)
- ✅ Builds backend binary
- ✅ Installs frontend dependencies
- ✅ Starts backend server on :8080
- ✅ Starts frontend server on :5173
- ✅ Shows connection URLs and logs

---

## Individual Component Startup

### Start Just Backend

```bash
# Option A: Using script
chmod +x START_BACKEND.sh
bash START_BACKEND.sh

# Option B: Manual commands
cd backend
go build -o server cmd/server/main.go
./server
```

**Expected output:**
```
Backend server listening on :8080
Connected to database: alpha
All services initialized
```

### Start Just Frontend

```bash
# Option A: Using script
chmod +x START_FRONTEND.sh
bash START_FRONTEND.sh

# Option B: Manual commands
cd frontend
npm install          # Only needed first time
npm run dev          # Starts dev server with HMR
```

**Expected output:**
```
VITE v5.4.20 ready in 299 ms
➜  Local:   http://localhost:5173/
➜  Network: http://192.168.x.x:5173/
```

---

## Manual Terminal-by-Terminal Startup

If you prefer to start things in separate terminals:

### Terminal 1: Backend Server

```bash
# Navigate to backend
cd backend

# Build the server
go build -o server cmd/server/main.go

# Run the server
./server

# Expected output:
# [INFO] Server starting on :8080
# [INFO] Database connection established
# [INFO] Services initialized
```

### Terminal 2: Frontend Server

```bash
# Navigate to frontend
cd frontend

# Install dependencies (first time only)
npm install

# Start development server
npm run dev

# Expected output:
# VITE v5.4.20  ready in 299 ms
# ➜  Local:   http://localhost:5173/
# ➜  Network: http://192.168.x.x:5173/
```

### Terminal 3: Database (if needed)

```bash
# Verify PostgreSQL is running
psql -U postgres -d alpha -c "SELECT 1"

# If not running, start it:
# macOS with Homebrew:
brew services start postgresql

# Linux with systemd:
sudo systemctl start postgresql

# Docker:
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:12
```

---

## Docker-Based Startup (Alternative)

If you prefer containerized setup:

### Start All Services with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

### Backend in Docker

```bash
# Build image
cd backend
docker build -t semlayer-backend .

# Run container
docker run -d \
  -p 8080:8080 \
  -e DATABASE_URL=postgres://postgres:postgres@db:5432/alpha \
  --name backend \
  semlayer-backend
```

### Frontend in Docker

```bash
# Build image
cd frontend
docker build -t semlayer-frontend .

# Run container
docker run -d \
  -p 5173:5173 \
  -e VITE_API_URL=http://localhost:8080 \
  --name frontend \
  semlayer-frontend
```

---

## Development Workflow

### Watch Mode (Auto-rebuild)

#### Backend with Watch

```bash
cd backend

# Using air (install first: go install github.com/cosmtrek/air@latest)
air

# Or manual watch
while inotifywait -e modify *.go; do go build -o server cmd/server/main.go; done
```

#### Frontend with Watch

```bash
cd frontend

# Vite automatically watches and hot-reloads
npm run dev

# Browser will auto-refresh when you save files
```

---

## Common Commands Reference

### Build Commands

```bash
# Backend build (production)
cd backend && go build -o server cmd/server/main.go

# Backend build (with optimizations)
cd backend && go build -ldflags="-s -w" -o server cmd/server/main.go

# Frontend build (production)
cd frontend && npm run build

# Frontend build with source maps
cd frontend && npm run build -- --sourcemap
```

### Runtime Commands

```bash
# Run backend
cd backend && ./server

# Run frontend (dev)
cd frontend && npm run dev

# Run frontend (preview production build)
cd frontend && npm run preview

# Run tests
cd backend && go test ./...
cd frontend && npm test
```

### Database Commands

```bash
# Connect to database
psql -U postgres -d alpha

# List tables
psql -U postgres -d alpha -c "\dt"

# Query employees
psql -U postgres -d alpha -c "SELECT * FROM employees LIMIT 10;"

# Reset database
psql -U postgres -d alpha -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"

# Backup database
pg_dump -U postgres alpha > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore database
psql -U postgres alpha < backup.sql
```

### System Commands

```bash
# Kill process on specific port
lsof -ti:8080 | xargs kill -9      # Kill backend
lsof -ti:5173 | xargs kill -9      # Kill frontend
lsof -ti:5432 | xargs kill -9      # Kill PostgreSQL

# Check port status
lsof -i :8080                       # Backend
lsof -i :5173                       # Frontend
lsof -i :5432                       # Database

# Process status
ps aux | grep "server\|npm\|postgres"
```

---

## Testing the System

### API Health Check

```bash
# Check backend is running
curl http://localhost:8080/health

# Create test employee
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "firstName": "John",
    "lastName": "Doe",
    "email": "john@example.com"
  }'

# Get employees
curl http://localhost:8080/api/employees \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111"
```

### Frontend Health Check

```bash
# Check frontend is running
curl http://localhost:5173

# Open in browser
open http://localhost:5173    # macOS
xdg-open http://localhost:5173  # Linux
start http://localhost:5173   # Windows
```

---

## Troubleshooting

### Backend Won't Start

```bash
# Check if port is in use
lsof -i :8080

# Kill existing process
lsof -ti:8080 | xargs kill -9

# Check Go installation
go version

# Check dependencies
go mod tidy

# Rebuild
go build -o server cmd/server/main.go
```

### Frontend Won't Start

```bash
# Check if port is in use
lsof -i :5173

# Kill existing process
lsof -ti:5173 | xargs kill -9

# Check Node.js
node --version
npm --version

# Reinstall dependencies
rm -rf node_modules package-lock.json
npm install

# Start again
npm run dev
```

### Database Connection Issues

```bash
# Check PostgreSQL is running
psql -U postgres -d postgres -c "SELECT 1"

# Check database exists
psql -U postgres -d postgres -c "\l"

# Create database if needed
psql -U postgres -c "CREATE DATABASE alpha;"

# Connection string
postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable
```

---

## Production Deployment

### Build for Production

```bash
# Backend
cd backend
go build -o server cmd/server/main.go

# Frontend
cd frontend
npm run build

# Output: dist/ directory with optimized files
```

### Deploy Backend

```bash
# Copy binary to server
scp backend/server user@server:/app/

# Set permissions
ssh user@server "chmod +x /app/server"

# Start with systemd or supervisor
# Create /etc/systemd/system/semlayer.service
```

### Deploy Frontend

```bash
# Copy build output to web server
scp -r frontend/dist/* user@server:/var/www/html/

# Or use with nginx
```

---

## Monitoring

### View Logs

```bash
# Backend logs (if using script)
tail -f logs/backend_*.log

# Frontend logs (if using script)
tail -f logs/frontend_*.log

# Real-time logs from processes
ps aux | grep server
ps aux | grep npm
```

### Monitor Performance

```bash
# CPU and Memory usage
top -p $(pgrep -f "server|npm" | tr '\n' ',')

# Network connections
netstat -an | grep -E "(8080|5173|5432)"

# Database queries (in psql)
psql -U postgres -d alpha
SELECT * FROM pg_stat_statements LIMIT 10;
```

---

## Environment Variables

### Backend

```bash
# Create .env file in backend/
export DATABASE_URL="postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
export SERVER_PORT="8080"
export LOG_LEVEL="info"
export ENVIRONMENT="development"
```

### Frontend

```bash
# Create .env file in frontend/
export VITE_API_URL="http://localhost:8080"
export VITE_ENVIRONMENT="development"
```

---

## Quick Reference

| Component | Port | URL | Command |
|-----------|------|-----|---------|
| Backend | 8080 | http://localhost:8080 | `./server` |
| Frontend | 5173 | http://localhost:5173 | `npm run dev` |
| Database | 5432 | localhost:5432 | `psql -U postgres -d alpha` |

---

## Complete System URLs

- **Frontend**: http://localhost:5173
- **Backend API**: http://localhost:8080
- **Database**: localhost:5432 (alpha)
- **Dynamic UI Generator**: http://localhost:5173/dynamic-ui (after login)

---

**Status**: 🟢 Ready to Run
**Last Updated**: October 21, 2025
