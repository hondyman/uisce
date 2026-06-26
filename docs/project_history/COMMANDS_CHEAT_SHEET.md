# ⚡ Quick Commands Cheat Sheet

## 🚀 START EVERYTHING (Single Command)

```bash
bash START_FULL_SYSTEM.sh
```

---

## 🏃 Individual Starts

### Start Backend Only
```bash
# Script method
bash START_BACKEND.sh

# Manual method
cd backend && go build -o server cmd/server/main.go && ./server
```

### Start Frontend Only
```bash
# Script method
bash START_FRONTEND.sh

# Manual method
cd frontend && npm run dev
```

---

## 🛑 Stop Services

```bash
# Kill backend (port 8080)
lsof -ti:8080 | xargs kill -9

# Kill frontend (port 5173)
lsof -ti:5173 | xargs kill -9

# Kill all (Ctrl+C if running in foreground)
```

---

## 📊 Check Status

```bash
# Is backend running?
curl http://localhost:8080/health

# Is frontend running?
curl http://localhost:5173

# What's using ports?
lsof -i :8080    # Backend
lsof -i :5173    # Frontend
lsof -i :5432    # Database
```

---

## 🗄️ Database Commands

```bash
# Connect
psql -U postgres -d alpha

# List all tables
psql -U postgres -d alpha -c "\dt"

# View employees
psql -U postgres -d alpha -c "SELECT * FROM employees;"

# Reset database
psql -U postgres -d alpha -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
```

---

## 🔨 Build Commands

```bash
# Build backend
cd backend && go build -o server cmd/server/main.go

# Build frontend
cd frontend && npm run build

# Build for production (optimized)
cd backend && go build -ldflags="-s -w" -o server cmd/server/main.go
```

---

## 📝 Useful URLs

| Service | URL |
|---------|-----|
| Frontend | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| Dynamic UI Generator | http://localhost:5173/dynamic-ui |
| Database | localhost:5432 (psql) |

---

## 🧪 Test the System

### Test Backend
```bash
# Get health
curl http://localhost:8080/health

# Create employee
curl -X POST http://localhost:8080/api/employees \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{"firstName":"John","lastName":"Doe","email":"john@example.com"}'

# List employees
curl http://localhost:8080/api/employees \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111"
```

### Test Frontend
```bash
# Visit in browser
open http://localhost:5173

# Navigate to
Config → Dynamic UI Generator
```

---

## 🐛 Common Issues & Fixes

| Problem | Solution |
|---------|----------|
| Port already in use | `lsof -ti:8080 \| xargs kill -9` |
| Go not found | Install Go 1.20+ |
| npm not found | Install Node.js & npm |
| Database error | Start PostgreSQL or Docker |
| Build fails | `go mod tidy` then retry |
| No node_modules | `npm install` in frontend dir |

---

## 📋 Multi-Terminal Setup

### Terminal 1: Backend
```bash
cd backend
go build -o server cmd/server/main.go
./server
```

### Terminal 2: Frontend
```bash
cd frontend
npm install  # First time only
npm run dev
```

### Terminal 3: Database (optional)
```bash
# Check database
psql -U postgres -d alpha

# Or if using Docker
docker-compose up postgres
```

---

## 🔄 Development Loop

### Make changes → Rebuild/Restart

#### Backend
```bash
# Edit Go files, then:
go build -o server cmd/server/main.go
# OR let auto-reload handle it (air)
```

#### Frontend
```bash
# Edit React/TypeScript files
# Browser auto-reloads (HMR) - no restart needed!
```

---

## 📊 System Info

```bash
# Go version
go version

# Node version
node --version && npm --version

# PostgreSQL version
psql --version

# System architecture
uname -a

# Check if services running
ps aux | grep -E "server|npm|postgres"
```

---

## 🚀 Deployment

### Build for Production
```bash
# Backend
cd backend && go build -o server cmd/server/main.go

# Frontend
cd frontend && npm run build
# Output: dist/ directory
```

### Preview Production
```bash
cd frontend
npm run preview
```

---

## 📚 Documentation

- **Full Guide**: See `COMPLETE_STARTUP_GUIDE.md`
- **Backend Fixes**: See `BACKEND_COMPILATION_FIXES.md`
- **System Status**: See `SYSTEM_RUNNING.md`
- **Deployment**: See `DEPLOYMENT_READY.md`

---

## 💡 Pro Tips

```bash
# Run backend in background
cd backend && ./server &

# Run frontend in background
cd frontend && npm run dev &

# Run everything in tmux (multiple panes)
tmux new-session -d -s dev
tmux split-window -h
# Then run backend in left, frontend in right

# Save port killers as functions in ~/.bashrc
killport8080='lsof -ti:8080 | xargs kill -9'
killport5173='lsof -ti:5173 | xargs kill -9'
```

---

**Last Updated**: October 21, 2025
**Status**: ✅ Ready to Use
