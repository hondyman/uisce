# ✅ SemLayer Platform - Verification Checklist

## Pre-Flight Checks

Run these commands to verify your system is ready:

### 1. PostgreSQL Connection
```bash
psql -U postgres -h localhost -d alpha -c "SELECT 1;"
```
Expected: `?column?` with value `1`

### 2. Backend Health
```bash
curl http://localhost:8080/health
```
Expected: `{"status":"healthy","timestamp":"...}`

### 3. Frontend Accessibility
```bash
curl -s http://localhost:5173 | head -10
```
Expected: HTML content with `<!DOCTYPE html>`

## Full Platform Startup Verification

### Step 1: Start Backend
```bash
# Terminal 1
bash START_BACKEND.sh
```

Wait for message:
```
✅ Backend server started
   URL: http://localhost:8080
```

### Step 2: Start Frontend
```bash
# Terminal 2
bash START_FRONTEND.sh
```

Wait for message:
```
✅ Frontend server started
   URL: http://localhost:5173
```

### Step 3: Verify Both Are Running
```bash
# Terminal 3
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:5173 | head -5
```

Both should return data.

### Step 4: Test Browser
Open browser and navigate to:
- http://localhost:5173

Expected: SemLayer web interface loads

## API Endpoint Tests

### Test REST APIs with Backend Headers

```bash
# Get tenants (with proper scope headers)
curl -s \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/tenants

# Get validation rules
curl -s \
  -H "X-Tenant-ID: 910638ba-a459-4a3f-bb2d-78391b0595f6" \
  -H "X-Tenant-Datasource-ID: 982aef38-418f-46dc-acd0-35fe8f3b97b0" \
  http://localhost:8080/api/validation-rules

# Check available routes
curl -s http://localhost:8080/_routes | head -20
```

### Browse Swagger UI
- Open: http://localhost:8080/swagger/index.html
- Should see all available endpoints
- Can test endpoints directly from UI

## System Requirements Verification

### Operating System
```bash
uname -s
# Expected: Darwin (for macOS)
```

### PostgreSQL Version
```bash
psql --version
# Expected: PostgreSQL 12+ (any recent version)
```

### Node.js Version
```bash
node --version
# Expected: v16+ (any recent LTS version)
```

### Go Version
```bash
go version
# Expected: go 1.18+ (any recent version)
```

## Configuration Verification

### Backend Config
```bash
cat backend/config.yaml
```

Should show:
- `dsn: "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"`
- `port: 8080`

### Frontend Config
```bash
cat frontend/.env.local
```

Should show:
- `VITE_USE_PROXY=true`
- `VITE_BACKEND_TARGET=http://localhost:8080`
- `VITE_API_BASE_URL=http://localhost:8080`

### Tenant Scope
Check browser DevTools after opening frontend:
```javascript
// In browser console:
console.log('Tenant:', JSON.parse(localStorage.getItem('selected_tenant')));
console.log('Datasource:', JSON.parse(localStorage.getItem('selected_datasource')));
```

Expected output:
```
Tenant: {id: '910638ba-a459-4a3f-bb2d-78391b0595f6', display_name: 'Test Tenant', ...}
Datasource: {id: '982aef38-418f-46dc-acd0-35fe8f3b97b0', source_name: 'Test Datasource', ...}
```

## Troubleshooting Guide

### Backend Won't Start

**Error**: `Database connection failed`
```bash
# Verify PostgreSQL
psql -U postgres -h localhost -c "SELECT 1;"
# Should return: 1

# Create alpha database if missing
psql -U postgres -h localhost -c "CREATE DATABASE alpha;"
```

**Error**: `Port 8080 already in use`
```bash
lsof -ti:8080 | xargs kill -9
bash START_BACKEND.sh
```

### Frontend Won't Start

**Error**: `Port 5173 already in use`
```bash
lsof -ti:5173 | xargs kill -9
bash START_FRONTEND.sh
```

**Error**: `npm: command not found`
```bash
# Install Node.js from nodejs.org or use:
brew install node
```

### API Calls Failing

**Error**: `404 Not Found` on API endpoints
- Verify backend is running: `curl http://localhost:8080/health`
- Check tenant headers are included
- Verify proxy is configured in frontend

**Error**: `GraphQL endpoint returns 404`
- This is expected - Hasura is not running
- REST APIs work fine without GraphQL
- Most features don't require GraphQL

### Performance Issues

**Slow API responses**
- Check PostgreSQL is not overloaded: `psql -U postgres -h localhost -c "SELECT * FROM pg_stat_activity;"`
- Check network connectivity: `ping localhost`

**Frontend slow to load**
- Clear browser cache: DevTools > Settings > Storage > Clear site data
- Check network tab for large downloads

## Success Indicators

✅ You're good to go when:

- [ ] PostgreSQL connects: `psql -U postgres -h localhost -d alpha -c "SELECT 1;"`
- [ ] Backend health OK: `curl http://localhost:8080/health`
- [ ] Frontend loads at http://localhost:5173
- [ ] Swagger UI accessible at http://localhost:8080/swagger/index.html
- [ ] Browser console shows tenant scope in localStorage
- [ ] No critical errors in backend logs
- [ ] No critical errors in frontend dev console

## Documentation Files

- `PLATFORM_QUICK_START.md` - Detailed quick start guide
- `FIXES_APPLIED_SUMMARY.md` - Summary of all fixes applied
- `RUN_PLATFORM.sh` - Automated platform startup script
- `agents.md` - Tenant scoping reference

## Support

For specific issues:

1. **Check logs**: `tail -f logs/backend_*.log` or `logs/frontend_*.log`
2. **Review error messages** in browser console and server logs
3. **Consult documentation** in the files listed above
4. **Verify configuration** matches expected values (see Configuration Verification section)

---

**Last Updated**: November 11, 2025
**Status**: ✅ All Systems Ready
