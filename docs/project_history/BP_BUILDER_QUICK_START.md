# BP Builder - 5-Minute Setup Guide

## ⚡ Quick Start (Copy-Paste Ready)

### 1. Add Database Schema

```bash
psql postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable << 'EOF'
CREATE TABLE IF NOT EXISTS business_processes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource_id UUID,
    process_name VARCHAR(255) NOT NULL,
    entity VARCHAR(100) NOT NULL,
    description TEXT,
    steps_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    is_active BOOLEAN DEFAULT false,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    version INTEGER DEFAULT 1,
    tags_json JSONB DEFAULT '[]'::jsonb,
    CONSTRAINT fk_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX idx_bp_tenant_id ON business_processes(tenant_id);
CREATE INDEX idx_bp_datasource ON business_processes(datasource_id);
CREATE INDEX idx_bp_active ON business_processes(is_active);
CREATE INDEX idx_bp_entity ON business_processes(entity);
CREATE INDEX idx_bp_created_at ON business_processes(created_at DESC);

\dt business_processes
EOF
```

✅ **Verification**: You should see the `business_processes` table listed

### 2. Register Routes in Backend

Open `backend/cmd/server/main.go` and add to your router setup:

```go
// Find where you setup your chi.Router, add:
bpHandlers := api.NewBPBuilderHandlers(db)
bpHandlers.RegisterRoutes(r)
```

### 3. Rebuild Backend

```bash
go build -tags bp_versioned -o ./bin/server ./backend/cmd/server
```

### 4. Start Services

```bash
# Terminal 1: Backend
./bin/server

# Terminal 2: Frontend (if not already running)
cd frontend && npm run dev
```

### 5. Access BP Builder

1. Open: `http://localhost:3000`
2. Select tenant/datasource in menu
3. Navigate to: `Config → BP Builder`

---

## 🧪 Test It (Copy-Paste Endpoints)

### Create a Process

```bash
TENANT_ID="00000000-0000-0000-0000-000000000000"
DATASOURCE_ID="11111111-1111-1111-1111-111111111111"

curl -X POST "http://localhost:8080/api/business-processes?tenant_id=$TENANT_ID&datasource_id=$DATASOURCE_ID" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "X-Tenant-Datasource-ID: $DATASOURCE_ID" \
  -d '{
    "processName": "Test Process",
    "entity": "Employee",
    "description": "Quick test",
    "steps": [{
      "id": "step1",
      "stepOrder": 1,
      "stepType": "data_entry",
      "stepName": "Collect Data",
      "durationHours": 1
    }],
    "isActive": false,
    "createdBy": "test_user",
    "version": 1,
    "tags": []
  }'
```

### List Processes

```bash
curl "http://localhost:8080/api/business-processes?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"
```

### Publish Process

```bash
PROCESS_ID="<id-from-create-response>"

curl -X POST "http://localhost:8080/api/business-processes/$PROCESS_ID/publish?tenant_id=$TENANT_ID" \
  -H "X-Tenant-ID: $TENANT_ID"
```

---

## ✅ Verification Checklist

- [ ] Database table created: `\dt business_processes`
- [ ] Indexes created: `\di business_processes*`
- [ ] Backend builds: `go build` succeeds
- [ ] Backend starts: No errors on `./bin/server`
- [ ] Frontend loads: http://localhost:3000 works
- [ ] Menu appears: "BP Builder" under Config
- [ ] Create process: Form loads and saves data
- [ ] View process: List shows saved process
- [ ] Publish works: Status changes to "Published"

---

## 🎯 Common Issues & Fixes

| Issue | Fix |
|-------|-----|
| "tenant_id is required" | Select tenant from menu first |
| Database table not found | Run schema creation SQL above |
| API returns 404 | Check backend registered routes |
| No data appears | Verify tenant_id matches in all calls |
| Build error: undefined | Ensure bp_builder_handlers.go is in api/ folder |

---

## 📁 Files Reference

| File | Purpose | Lines |
|------|---------|-------|
| `frontend/src/components/BPBuilder/BusinessProcessBuilderEnhanced.tsx` | Main UI component | 814 |
| `frontend/src/components/BPBuilder/useBPBuilderAPI.ts` | API hooks | 142 |
| `frontend/src/pages/BPBuilderPage.tsx` | Page wrapper | 9 |
| `backend/internal/api/bp_builder_handlers.go` | API handlers | 450 |

---

## 🚀 You're Done!

The BP Builder is now:
- ✅ Fully integrated
- ✅ Database-backed
- ✅ Multi-tenant scoped
- ✅ Production ready

**Start using it:** Create, edit, publish, and simulate business processes!

