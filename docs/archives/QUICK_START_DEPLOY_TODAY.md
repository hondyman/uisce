# ⚡ DEPLOYMENT QUICK-START - Path B TODAY

## 🎯 Decision: DEPLOY NOW (Option A)

**Timeline:** 50 minutes  
**Risk:** Minimal  
**Status:** ✅ Ready  

---

## 📋 QUICK CHECKLIST

### Pre-Flight (2 min)
- [ ] Read: `DEPLOYMENT_RUNBOOK_PATH_B.md`
- [ ] Backup database
- [ ] Stop current backend/frontend

### Phase 1: Database (5 min)
```bash
psql -U postgres -d alpha -f backend/db/migrations/2025_10_20_add_hierarchy_support.sql
```
- [ ] Verify columns added: `\d validation_rules`
- [ ] Verify indexes: `SELECT * FROM pg_indexes WHERE tablename='validation_rules'`
- [ ] Verify sample data: `SELECT COUNT(*) FROM validation_rules WHERE hierarchy_depth > 0`

### Phase 2: Backend (10 min)
```bash
cd backend && go build -o semlayer-backend ./cmd/server
PORT=8080 ./semlayer-backend &
sleep 3
curl http://localhost:8080/health
```
- [ ] Build succeeds (no errors)
- [ ] Health endpoint returns 200
- [ ] No startup errors in logs

### Phase 3: Frontend (10 min)
```bash
cd frontend && npm run build
npm run preview
# Open http://localhost:5000
```
- [ ] Build succeeds (no errors)
- [ ] Application loads
- [ ] Tenant selector visible
- [ ] No console errors

### Phase 4: Smoke Tests (15 min)

**Test 1: Create Rule**
- [ ] Navigate to Validation Rules
- [ ] Create new rule: "Test Rule"
- [ ] Verify saved (badge + spinner worked)

**Test 2: Cross-Entity**
- [ ] Add sub-entity condition
- [ ] Select: line_items
- [ ] Verify dependency chain displays

**Test 3: API Calls**
```bash
curl -X GET "http://localhost:8080/api/validation-rules" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111"
```
- [ ] Returns 200 + JSON data
- [ ] Headers required (403 without them)

**Test 4: Performance (Debouncing)**
- [ ] DevTools Network tab
- [ ] Edit rule 10 times fast
- [ ] Wait 1 sec
- [ ] Only 1 API call (not 10!)
- [ ] Expected: -90% reduction

**Test 5: Performance (Optimistic)**
- [ ] Delete a rule
- [ ] Disappears immediately
- [ ] Reappears if test network error
- [ ] Expected: 200-500ms faster

### Phase 5: Sign-Off (5 min)
- [ ] All tests passed
- [ ] No console errors
- [ ] Database has rules
- [ ] Zero 500 errors
- [ ] Ready for production

---

## 🚀 ONE-LINER DEPLOY (for experienced ops)

```bash
# Database
psql -U postgres -d alpha -f backend/db/migrations/2025_10_20_add_hierarchy_support.sql

# Backend
cd backend && go build -o semlayer-backend ./cmd/server && PORT=8080 ./semlayer-backend &

# Frontend
cd frontend && npm run build && npm run preview

# Test
curl http://localhost:8080/health
curl http://localhost:5000
```

---

## 📚 Documentation

| Doc | Purpose |
|-----|---------|
| `DEPLOYMENT_RUNBOOK_PATH_B.md` | Detailed step-by-step (read first) |
| `PATH_B_DEPLOYMENT_COMPLETE.md` | Architecture overview |
| `PHASE_1_OPTIMIZATION_INTEGRATION.md` | Code examples |
| `VERIFICATION_ALL_CODE_IN_PLACE.md` | Verification report |

---

## ⚠️ Rollback (if needed)

```bash
# Stop current
pkill -f semlayer-backend
pkill -f "frontend.*preview"

# Revert binary/dist to previous version
# Restart

# Total time: < 5 minutes
```

---

## ✅ SUCCESS LOOKS LIKE

- ✅ Application loads
- ✅ Can create validation rules
- ✅ Cross-entity conditions work
- ✅ Debouncing reduces API calls by 90%
- ✅ Optimistic updates instant
- ✅ No console errors
- ✅ Database migration applied
- ✅ Headers enforced (X-Tenant-ID required)

---

## 🎯 FINAL STATUS

| Component | Status | Time |
|-----------|--------|------|
| Code Quality | ✅ Ready | - |
| Frontend Build | ✅ Ready | 10 min |
| Backend Build | ✅ Ready | 10 min |
| Database | ✅ Ready | 5 min |
| Tests | ✅ Prepared | 15 min |
| Documentation | ✅ Complete | - |

**Total Deployment Time: 50 minutes**

**Confidence Level: 🟢 100%**

**Risk Level: 🟢 MINIMAL**

---

Ready to deploy? Start with Phase 1 (Database) ↑
