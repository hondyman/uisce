# 📋 COMPLETE SYSTEM STARTUP SCRIPTS & COMMANDS

## 🎯 Quick Summary

You now have **3 ready-to-use startup scripts** plus comprehensive command references for running the full Semlayer system.

---

## 📦 Available Startup Scripts

### 1. **START_FULL_SYSTEM.sh** - Recommended ⭐

**Start everything with one command:**

```bash
bash START_FULL_SYSTEM.sh
```

**What it does:**
- ✅ Verifies all prerequisites (Go, Node.js, npm, PostgreSQL)
- ✅ Checks port availability
- ✅ Builds backend binary
- ✅ Installs frontend dependencies
- ✅ Starts backend server on port 8080
- ✅ Starts frontend server on port 5173
- ✅ Shows logs and connection URLs
- ✅ Provides logs for troubleshooting

**Features:**
- Color-coded output
- Automatic port conflict resolution
- Exit trap for clean shutdown
- Timestamped logs
- Process monitoring

---

### 2. **START_BACKEND.sh** - Backend Only

**Start just the backend server:**

```bash
bash START_BACKEND.sh
```

**What it does:**
- Checks if port 8080 is available
- Builds backend if needed
- Starts API server
- Shows logs and status

**Use when:**
- Testing backend only
- Frontend already running elsewhere
- Debugging backend issues

---

### 3. **START_FRONTEND.sh** - Frontend Only

**Start just the frontend server:**

```bash
bash START_FRONTEND.sh
```

**What it does:**
- Checks if port 5173 is available
- Installs dependencies if needed
- Starts Vite dev server with HMR
- Shows logs and status

**Use when:**
- Testing frontend only
- Backend already running elsewhere
- Debugging UI issues

---

## 🚀 Quick Start Options

### Option A: Single Command (Easiest)

```bash
# Start everything from project root
bash START_FULL_SYSTEM.sh
```

**Result:**
```
✅ Backend: http://localhost:8080
✅ Frontend: http://localhost:5173
✅ Database: Connected to localhost:5432
```

---

### Option B: Separate Terminals

**Terminal 1 - Backend:**
```bash
cd backend
go build -o server cmd/server/main.go
./server
```

**Terminal 2 - Frontend:**
```bash
cd frontend
npm install  # First time only
npm run dev
```

**Terminal 3 - Database:**
```bash
# Verify PostgreSQL running
psql -U postgres -d alpha -c "SELECT 1"
```

---

### Option C: Docker Compose

```bash
# If docker-compose.yml configured:
docker-compose up -d
```

---

## 📚 Documentation Files Created

| File | Purpose |
|------|---------|
| `COMPLETE_STARTUP_GUIDE.md` | Comprehensive startup guide with all options |
| `COMMANDS_CHEAT_SHEET.md` | Quick reference for common commands |
| `START_FULL_SYSTEM.sh` | Automated full system startup script |
| `START_BACKEND.sh` | Backend-only startup script |
| `START_FRONTEND.sh` | Frontend-only startup script |

---

## ⚡ Essential Commands

### Build Backend
```bash
cd backend
go build -o server cmd/server/main.go
```

### Build Frontend
```bash
cd frontend
npm run build
```

### Start Backend
```bash
cd backend
./server
```

### Start Frontend
```bash
cd frontend
npm run dev
```

### Connect to Database
```bash
psql -U postgres -d alpha
```

### Kill Services
```bash
lsof -ti:8080 | xargs kill -9    # Backend
lsof -ti:5173 | xargs kill -9    # Frontend
lsof -ti:5432 | xargs kill -9    # Database
```

---

## 🔗 System URLs & Ports

| Service | Port | URL |
|---------|------|-----|
| Backend API | 8080 | http://localhost:8080 |
| Frontend UI | 5173 | http://localhost:5173 |
| Database | 5432 | localhost:5432 |
| Dynamic UI | N/A | http://localhost:5173/dynamic-ui |

---

## 🧪 Testing After Startup

### 1. Test Backend Health
```bash
curl http://localhost:8080/health
```

### 2. Test Frontend Load
```bash
open http://localhost:5173
# or
curl http://localhost:5173 | head -20
```

### 3. Create Test Employee
```bash
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "firstName": "Test",
    "lastName": "User",
    "email": "test@example.com",
    "department": "Engineering",
    "status": "Active"
  }'
```

### 4. Open in Browser
```bash
# Open Dynamic UI Generator
http://localhost:5173

# Then navigate to:
Config → Dynamic UI Generator
```

---

## 📊 Log Files

When using startup scripts, logs are saved to:

```
logs/
├── backend_20251021_143022.log    # Backend logs (timestamped)
└── frontend_20251021_143025.log   # Frontend logs (timestamped)
```

View logs:
```bash
tail -f logs/backend_*.log
tail -f logs/frontend_*.log
```

---

## 🔧 Prerequisites Check

Before starting, ensure you have:

```bash
# Go 1.20+
go version

# Node.js 16+
node --version

# npm
npm --version

# PostgreSQL running
psql -U postgres -c "SELECT 1"
```

---

## 💡 Usage Scenarios

### Development
```bash
# Run full system with auto-reload
bash START_FULL_SYSTEM.sh

# Frontend auto-reloads on save (HMR)
# Backend may need rebuild on Go changes
```

### Testing Single Component
```bash
# Test only backend
bash START_BACKEND.sh

# Test only frontend
bash START_FRONTEND.sh
```

### Production Build
```bash
# Build backend for production
cd backend && go build -ldflags="-s -w" -o server cmd/server/main.go

# Build frontend for production
cd frontend && npm run build
```

---

## 🚨 Troubleshooting

### "Address already in use"
```bash
# Kill process on port
lsof -ti:8080 | xargs kill -9
```

### "Cannot find go"
```bash
# Install Go from https://golang.org/dl/
# Then add to PATH
```

### "npm: command not found"
```bash
# Install Node.js and npm from https://nodejs.org/
```

### "Cannot connect to database"
```bash
# Start PostgreSQL
brew services start postgresql  # macOS
sudo systemctl start postgresql # Linux

# Or use Docker
docker run -d -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:12
```

### "Port already in use"
```bash
# The scripts auto-kill processes on startup
# Or manually kill them:
lsof -i :8080 -sTCP:LISTEN -t | xargs kill -9
```

---

## 📖 Reading the Scripts

### START_FULL_SYSTEM.sh includes:
- ✅ Comprehensive error checking
- ✅ Color-coded output for readability
- ✅ Automatic port conflict resolution
- ✅ Prerequisites verification
- ✅ Build automation
- ✅ Service startup with monitoring
- ✅ Clean shutdown handling

### View script contents:
```bash
cat START_FULL_SYSTEM.sh
cat START_BACKEND.sh
cat START_FRONTEND.sh
```

---

## 🎯 Next Steps

### 1. Immediate (Now)
```bash
# Start full system
bash START_FULL_SYSTEM.sh

# System will be ready in ~30 seconds
```

### 2. Test (Minutes 1-5)
```bash
# Open browser
open http://localhost:5173

# Navigate to
Config → Dynamic UI Generator

# Fill form and save
# Check Network tab for 201 response
```

### 3. Development (Ongoing)
```bash
# Make changes to code
# Frontend auto-reloads
# Backend requires rebuild (go build)

# Keep terminals running for logs
```

### 4. Deployment (When Ready)
```bash
# Build production versions
cd backend && go build -o server cmd/server/main.go
cd frontend && npm run build

# Deploy build artifacts
```

---

## 🔄 Development Workflow

```bash
# 1. Start full system
bash START_FULL_SYSTEM.sh

# 2. Edit frontend code (auto-reloads)
# 3. Edit backend code
#    - Rebuild: cd backend && go build -o server cmd/server/main.go
#    - Restart: ./server

# 4. Test in browser
# 5. Check logs for errors
tail -f logs/backend_*.log

# 6. Push changes when ready
git add .
git commit -m "Your changes"
git push
```

---

## 📝 Summary

**You have:**
- ✅ 3 automated startup scripts
- ✅ Comprehensive startup guide (COMPLETE_STARTUP_GUIDE.md)
- ✅ Quick reference cheat sheet (COMMANDS_CHEAT_SHEET.md)
- ✅ Build and deployment instructions
- ✅ Troubleshooting guide
- ✅ Test procedures

**To start:**
```bash
bash START_FULL_SYSTEM.sh
```

**That's it!** 🎉

---

**Created**: October 21, 2025
**Status**: ✅ Ready to Use
**Confidence**: HIGH

See `COMPLETE_STARTUP_GUIDE.md` for detailed information on all startup options.
See `COMMANDS_CHEAT_SHEET.md` for quick command reference.
