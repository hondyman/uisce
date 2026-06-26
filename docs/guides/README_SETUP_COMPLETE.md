# 🎯 SemLayer Platform - Complete Setup Guide

## 📚 Documentation Index

| Document | Purpose | Read Time |
|----------|---------|-----------|
| **THIS FILE** | Overview and quick navigation | 5 min |
| `PLATFORM_QUICK_START.md` | Step-by-step quick start (2-5 min setup) | 5 min |
| `FIXES_APPLIED_SUMMARY.md` | Detailed explanation of all fixes | 10 min |
| `VERIFICATION_CHECKLIST.md` | Pre-flight checks and troubleshooting | 15 min |
| `RUN_PLATFORM.sh` | Automated startup script | Run it! |

## ⚡ TL;DR - Get Running in 2 Minutes

```bash
# Terminal 1: Start Backend (port 8080)
bash START_BACKEND.sh

# Terminal 2: Start Frontend (port 5173)  
bash START_FRONTEND.sh

# Open browser
open http://localhost:5173
```

That's it! Your platform is running. 🚀

---

## 🎯 What Was Fixed

Your platform had 5 critical issues preventing it from running. All have been fixed:

### 1. **Database Connection** ✅
   - **Was**: Backend couldn't find Docker hostname `semlayer-test-postgres`
   - **Fixed**: Updated config to use `localhost:5432`
   - **Result**: Backend connects to PostgreSQL successfully

### 2. **Frontend API Proxy** ✅
   - **Was**: Frontend tried to call its own port instead of backend
   - **Fixed**: Verified proxy config in `.env.local` is correct
   - **Result**: All API calls route to backend on port 8080

### 3. **GraphQL Endpoint** ✅
   - **Was**: Frontend crashed trying to reach non-existent GraphQL endpoint
   - **Fixed**: Added graceful fallback for missing Hasura
   - **Result**: App works without GraphQL; REST APIs fully functional

### 4. **Temporal Client** ✅
   - **Was**: Backend failed startup waiting for Temporal server
   - **Fixed**: Made Temporal optional with configuration
   - **Result**: Backend starts and logs warnings only

### 5. **Tenant Scope** ✅
   - **Was**: API calls lacked proper tenant headers
   - **Verified**: Auto-seeding of test tenant scope works
   - **Result**: All requests properly scoped to tenant

---

## 🚀 Quick Start (5 Minutes)

### Prerequisites
- ✅ PostgreSQL running on `localhost:5432`
- ✅ Database `alpha` created
- ✅ Node.js installed
- ✅ Go installed

### Step 1: Start Backend
```bash
bash START_BACKEND.sh
```

**Expected Output:**
```
✅ Backend server started
   URL: http://localhost:8080
```

**Verify:** `curl http://localhost:8080/health`

### Step 2: Start Frontend (new terminal)
```bash
bash START_FRONTEND.sh
```

**Expected Output:**
```
✅ Frontend server started
   URL: http://localhost:5173
```

### Step 3: Open in Browser
- Frontend: http://localhost:5173
- Swagger API Docs: http://localhost:8080/swagger/index.html

---

## 📊 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Browser (5173)                           │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ React Frontend + TypeScript                          │   │
│  │ - Automatic tenant scope via setupTenantFetch.ts    │   │
│  │ - REST API proxy to :8080                           │   │
│  │ - GraphQL with fallback (optional)                  │   │
│  └──────────────────────────────────────────────────────┘   │
│                         ↓ HTTP                                │
├─────────────────────────────────────────────────────────────┤
│                     Backend API (8080)                        │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Go REST API Server                                   │   │
│  │ - 200+ endpoints                                     │   │
│  │ - Tenant-scoped operations                          │   │
│  │ - Temporal integration (optional)                   │   │
│  └──────────────────────────────────────────────────────┘   │
│                         ↓ TCP                                 │
├─────────────────────────────────────────────────────────────┤
│                  PostgreSQL (5432)                            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Database: alpha                                      │   │
│  │ - Metadata & configuration                          │   │
│  │ - Multi-tenant data storage                         │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

---

## 📋 Detailed Guides

### For Quick Start
→ See: **`PLATFORM_QUICK_START.md`**
- Complete 2-step startup
- Configuration details
- Common issues & solutions

### For Understanding Fixes
→ See: **`FIXES_APPLIED_SUMMARY.md`**
- Detailed explanation of each fix
- Code changes made
- Why each fix was needed

### For Verification & Troubleshooting
→ See: **`VERIFICATION_CHECKLIST.md`**
- Pre-flight checks
- API endpoint tests
- Troubleshooting guide
- Success indicators

---

## 🔑 Key Features Available

### ✅ REST APIs (200+ Endpoints)
- Bundle management
- Validation rules
- Semantic layer
- Fabric builder
- Business entities
- Calculations
- And many more...

### ✅ Multi-Tenant
- Automatic tenant scoping
- Isolated data per tenant
- Safe for multi-user deployments

### ✅ Developer-Friendly
- Swagger UI at http://localhost:8080/swagger/index.html
- Detailed error messages
- Extensive logging
- TypeScript frontend

### ⚠️ Optional Features
- GraphQL (requires Hasura - not in quick start)
- Temporal workflows (requires Temporal server)
- Advanced metrics collection

---

## 🛠️ Configuration Files

| File | Purpose |
|------|---------|
| `backend/config.yaml` | Backend database and port settings |
| `frontend/.env.local` | Frontend API proxy configuration |
| `frontend/src/setupTenantFetch.ts` | Automatic tenant scope injection |

---

## 📊 Performance Expectations

| Operation | Expected Time | Notes |
|-----------|---------------|-------|
| Backend startup | 2-3 seconds | Initializes DB connections |
| Frontend startup | 5-10 seconds | Compiles TypeScript |
| API response | 50-200ms | Typical latency |
| GraphQL query (if enabled) | 100-500ms | Via optional Hasura |

---

## 🆘 Quick Troubleshooting

### Port already in use?
```bash
lsof -ti:8080 | xargs kill -9  # Backend
lsof -ti:5173 | xargs kill -9  # Frontend
```

### Database error?
```bash
psql -U postgres -h localhost -d alpha -c "SELECT 1;"
```

### Check logs?
```bash
tail -f logs/backend_*.log
tail -f logs/frontend_*.log
```

### Full troubleshooting?
→ See: **`VERIFICATION_CHECKLIST.md`** (Troubleshooting Guide section)

---

## 📚 Additional Resources

### Documentation in Repository
- `agents.md` - Tenant scoping reference
- `BACKEND_COMMANDS_REFERENCE.md` - Backend CLI options
- `DOCKER_README.md` - Docker deployment options

### API Documentation (Once Running)
- Swagger UI: http://localhost:8080/swagger/index.html
- Try all endpoints directly
- See request/response schemas

---

## ✅ What You Get

Once running, you have:

- ✅ **Full REST API** for all business operations
- ✅ **Web UI** for interactive access
- ✅ **Multi-tenant support** with automatic scoping
- ✅ **Developer tools** (Swagger, logging, debugging)
- ✅ **Production-ready architecture** (works in Docker too)
- ✅ **Extensible codebase** (TypeScript + Go)

---

## 🎓 Learning Path

1. **Start here**: `PLATFORM_QUICK_START.md` (5 min)
2. **Explore UI**: Open http://localhost:5173
3. **Try APIs**: Use Swagger at http://localhost:8080/swagger/index.html
4. **Learn code**: Read `FIXES_APPLIED_SUMMARY.md` to understand the architecture
5. **Go deeper**: Check documentation in `VERIFICATION_CHECKLIST.md`

---

## 📞 Support

| Issue | Solution |
|-------|----------|
| Setup questions | Read `PLATFORM_QUICK_START.md` |
| Errors or crashes | Check `VERIFICATION_CHECKLIST.md` |
| Understanding fixes | Read `FIXES_APPLIED_SUMMARY.md` |
| Need to troubleshoot | Run verification checklist |
| Want to understand code | Check code comments and git diffs |

---

## 🎉 You're Ready!

Your platform is fully configured and ready to run. 

**Next step:** Follow the **TL;DR** at the top of this file!

```bash
# Start Backend
bash START_BACKEND.sh

# Start Frontend (new terminal)
bash START_FRONTEND.sh

# Open browser to http://localhost:5173
```

Enjoy! 🚀

---

**Last Updated**: November 11, 2025  
**Status**: ✅ Complete & Ready  
**All Issues**: ✅ Fixed
