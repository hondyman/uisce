# Marketplace System - Visual Guide & Checklists

## 🎨 UI Overview

### Main Tabs

```
┌─────────────────────────────────────────────────────────┐
│  MARKETPLACE                                            │
│                                                         │
│  📦 Marketplace | 📋 My Items | 📊 Analytics           │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  [ BROWSE TAB CONTENT ]                                 │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

---

### Browse Tab Layout

```
┌──────────────┬────────────────────────────────────┐
│   FILTERS    │   MAIN CONTENT AREA                │
│              │                                    │
│ Search:      │  Sorting: [Popular ▼]              │
│ [_________]  │  View: [Grid] [List]                │
│              │                                    │
│ Type:        │  ┌──────────┐ ┌──────────┐         │
│ [All       ▼]│  │ 🌱 ESG   │ │ 🛡️ AML   │         │
│              │  │ Compliance│ │Compliance│         │
│ Category:    │  │ BLOCK ★4.8│ │ BLOCK ★4.5│       │
│ ☑ ESG       │  │ [Add]    │ │ [Add]    │         │
│ ☑ Compliance │  └──────────┘ └──────────┘         │
│ ☑ Risk Mgmt  │                                    │
│              │  ┌──────────┐ ┌──────────┐         │
│ Severity:    │  │ 💰 Margin│ │ 📊 Conc.  │         │
│ ☑ BLOCK      │  │Compliance│ │ Limit    │         │
│ ☑ WARNING    │  │ BLOCK ★4.2│ │ WARNING ★4.0│     │
│ ☑ INFO       │  │ [Add]    │ │ [Add]    │         │
│              │  └──────────┘ └──────────┘         │
│ Official     │                                    │
│ ☑ Only       │                                    │
│              │                                    │
└──────────────┴────────────────────────────────────┘
```

---

### My Items Tab Layout

```
┌──────────────────────────────────────────────────────┐
│  MY ITEMS                                            │
├──────────────────────────────────────────────────────┤
│                                                      │
│  You have 3 items added to your platform             │
│                                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │ 🌱 My ESG Validator                            │  │
│  │ Status: ✓ Enabled | Added: 10/27/2024         │  │
│  │ Usage: 42 executions | Version: 1.0.2         │  │
│  │ [Details] [Configure] [Remove]                │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │ 🛡️ AML Compliance                              │  │
│  │ Status: ✓ Enabled | Added: 10/26/2024         │  │
│  │ Usage: 156 executions | Version: 1.0.0        │  │
│  │ [Details] [Configure] [Remove]                │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │ 💰 Margin Compliance                           │  │
│  │ Status: ✓ Enabled | Added: 10/25/2024         │  │
│  │ Usage: 89 executions | Version: 1.0.1         │  │
│  │ [Details] [Configure] [Remove]                │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  [← Back to Browse Marketplace]                      │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

### Item Detail Modal

```
┌──────────────────────────────────────────────────────┐
│  ESG COMPLIANCE                                   [X] │
├──────────────────────────────────────────────────────┤
│                                                      │
│         🌱  ESG COMPLIANCE v1.0.2                    │
│                                                      │
│  [Official] [Recommended]                            │
│                                                      │
│  Validates portfolio compliance with ESG standards   │
│                                                      │
│  ┌────────────────────────────────────────────────┐  │
│  │ Category    │ ESG & Sustainability             │  │
│  │ Type        │ Rule                             │  │
│  │ Severity    │ BLOCK ⚠️                         │  │
│  │ Frequency   │ ON_TRADE                         │  │
│  │ Uses        │ 342 organizations                │  │
│  │ Rating      │ ★★★★★ 4.8 (42 ratings)         │  │
│  └────────────────────────────────────────────────┘  │
│                                                      │
│  External Providers: MSCI, Refinitiv                │
│                                                      │
│  Full Description:                                   │
│  This rule validates that all securities in the     │
│  portfolio meet ESG score thresholds. The rule can  │
│  be configured to check different ESG providers...  │
│                                                      │
│                              [ALREADY ADDED ✓]       │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

## 🔄 User Workflows

### Workflow 1: Browse & Add Item

```
START
  ↓
[Browse Marketplace]
  ↓
Search or Filter Items
  ↓
Review Item Details (Modal)
  ↓
[Add to Platform] Button
  ↓
Item Added ✓
  ↓
See in "My Items" Tab
  ↓
Configure & Manage
  ↓
END
```

---

### Workflow 2: Remove Item

```
START
  ↓
Go to "My Items" Tab
  ↓
Find Item to Remove
  ↓
Click [Remove] Button
  ↓
Confirm Deletion
  ↓
Item Removed from Database ✓
  ↓
No Longer in "My Items"
  ↓
END
```

---

### Workflow 3: Rate Item

```
START
  ↓
View Item Details
  ↓
See Current Rating (e.g., ★4.8)
  ↓
[Rate This Item] Button
  ↓
Select Rating (1-5 stars)
  ↓
Add Comment (optional)
  ↓
Submit Rating ✓
  ↓
See Confirmation
  ↓
END
```

---

## 📋 Pre-Deployment Checklist

### ✅ Phase 1: Preparation (Day 0)

```
[ ] Database backup created
[ ] Deployment plan documented
[ ] Team notified
[ ] Rollback plan ready
[ ] Read MARKETPLACE_QUICK_START.md
```

### ✅ Phase 2: Database Setup (Day 1)

```
[ ] PostgreSQL connection verified
[ ] Migration file reviewed
[ ] Run migration: psql -f 004_marketplace_tables.sql
[ ] Verify tables created: \dt marketplace*
[ ] Verify sample data: SELECT COUNT(*) FROM marketplace_items;
    Expected: 4 rows
[ ] Test connections
[ ] Document connection string
```

### ✅ Phase 3: Backend Setup (Day 1)

```
[ ] Copy marketplace_routes.go to backend/internal/api/
[ ] Review code
[ ] Add RegisterMarketplaceRoutes(router, db) to api.go
[ ] Build: go build ./cmd/server
[ ] Check for errors
[ ] Test API locally:
    curl http://localhost:8080/api/marketplace/items \
      -H "X-Tenant-ID: <uuid>"
[ ] Verify returns 4 items
[ ] Test all 10 endpoints
```

### ✅ Phase 4: Frontend Setup (Day 1)

```
[ ] Copy Marketplace.tsx to frontend/src/pages/marketplace/
[ ] Copy Marketplace.module.css to same directory
[ ] Build: npm run build
[ ] Check for TypeScript errors
[ ] Check for ESLint warnings:
    ✗ 2 select elements missing aria-label (lines 326, 360)
[ ] Fix warnings: Add aria-label attributes
[ ] Rebuild: npm run build
[ ] Verify no errors
[ ] Add route to router
[ ] Add navigation link
```

### ✅ Phase 5: Integration Testing (Day 1)

```
[ ] Start backend: go run ./cmd/server
[ ] Start frontend: npm run dev
[ ] Navigate to http://localhost:3000/marketplace
[ ] Verify page loads
[ ] Verify 4 items display in grid
[ ] Test search: type "ESG", should find 1 item
[ ] Test filter: select severity "BLOCK", should find 3 items
[ ] Test add item: click Add, should succeed
[ ] Verify in database:
    SELECT * FROM tenant_marketplace_items;
[ ] Test My Items tab: should show added item
[ ] Test remove: click Remove, verify deletion
[ ] Test responsive: open DevTools, test mobile view
```

### ✅ Phase 6: Security Testing (Day 1)

```
[ ] Verify tenant isolation:
    - Log in as Tenant A
    - Add item X
    - Switch to Tenant B
    - Verify Tenant B cannot see item X
[ ] Test X-Tenant-ID validation:
    curl http://localhost:8080/api/marketplace/items
    (without header - should fail)
[ ] Verify cross-tenant data leak impossible
[ ] Check that all queries filtered by tenant_id
```

### ✅ Phase 7: Performance Testing (Day 1)

```
[ ] Load test browse endpoint:
    100 concurrent users
    Expected: < 500ms response time
[ ] Load test add endpoint:
    10 concurrent adds
    Expected: All succeed
[ ] Monitor database connections
[ ] Check query times in logs
[ ] Verify no N+1 queries
```

### ✅ Phase 8: Staging Deployment (Day 2)

```
[ ] Deploy to staging environment
[ ] Verify all systems work
[ ] Repeat integration tests on staging
[ ] Get sign-off from QA
[ ] Get sign-off from security
```

### ✅ Phase 9: Production Deployment (Day 2-3)

```
[ ] Notify users of planned deployment
[ ] Run migration in production
[ ] Deploy backend code
[ ] Deploy frontend code
[ ] Monitor logs for errors
[ ] Monitor database performance
[ ] Verify features work in production
[ ] Get sign-off from operations
```

### ✅ Phase 10: Post-Deployment (Day 3+)

```
[ ] Monitor error rates
[ ] Monitor response times
[ ] Collect user feedback
[ ] Document any issues
[ ] Plan next iteration
[ ] Start work on Phase 2 features
```

---

## 🧪 Test Scenarios

### Scenario 1: Happy Path (Browse & Add)

```
1. Navigate to /marketplace
   ✓ Page loads
   ✓ Header shows "Marketplace"
   ✓ 3 tabs visible

2. Browse tab is selected
   ✓ 4 items visible in grid
   ✓ Search box visible
   ✓ Filters visible

3. Type "ESG" in search
   ✓ Grid updates
   ✓ Shows only 1 item

4. Clear search
   ✓ All 4 items visible again

5. Select filter: Severity = "BLOCK"
   ✓ Grid shows 3 items (BLOCK severity)

6. Click on first item
   ✓ Modal opens
   ✓ Shows item name, description, details

7. Click [Add to Platform]
   ✓ Modal closes
   ✓ Item added successfully

8. Go to "My Items" tab
   ✓ Tab shows 1 item
   ✓ Item displays: name, date added, usage

9. Go back to "Browse"
   ✓ Item now shows [Already Added] badge

10. Click [Remove] on item in My Items
    ✓ Item removed
    ✓ Database updated
    ✓ "My Items" is now empty
```

### Scenario 2: Filtering & Sorting

```
1. Browse tab
   ✓ Default sort: relevance

2. Change sort to "Rating"
   ✓ Items reorder by rating

3. Change sort to "Newest"
   ✓ Items reorder by created date

4. Filter by Category
   ✓ Can select multiple categories
   ✓ Results update correctly

5. Filter by Severity
   ✓ Can select multiple severities
   ✓ Results update correctly

6. Combine filters
   ✓ All filters work together
   ✓ Results are intersection (AND logic)

7. Clear all filters
   ✓ Returns to unfiltered state
```

### Scenario 3: Error Handling

```
1. Remove tenant ID header
   ✓ API returns 400 error

2. Add invalid item ID
   ✓ API returns 404 error

3. Rate with invalid rating (e.g., 6)
   ✓ API returns 400 error

4. Add item that's already added
   ✓ API returns 409 Conflict

5. Try to access other tenant's items
   ✓ API returns 403 Forbidden

6. Disconnect from database mid-operation
   ✓ Graceful error message
   ✓ No data corruption
```

---

## 📊 Deployment Checklist

### Pre-Deployment
```
☐ Code review completed
☐ All tests passing
☐ Documentation reviewed
☐ Security scan completed
☐ Performance baseline established
☐ Rollback plan documented
☐ Team trained
☐ Backup created
```

### Deployment
```
☐ Migration executed
☐ Backend deployed
☐ Frontend deployed
☐ Health checks passing
☐ Logs monitored
☐ No error spikes
```

### Post-Deployment
```
☐ Smoke tests passing
☐ End-to-end tests passing
☐ Performance acceptable
☐ No user complaints
☐ Sign-off obtained
☐ Monitoring alerts configured
☐ Documentation updated
```

---

## 🎯 Success Metrics

### Functionality
```
✓ Browse items - works
✓ Search items - works
✓ Filter items - works
✓ Sort items - works
✓ Add items - works
✓ Remove items - works
✓ Rate items - works
✓ View My Items - works
✓ Analytics tab - displays (data coming next)
```

### Performance
```
✓ Browse load: < 500ms
✓ Add item: < 200ms
✓ Remove item: < 100ms
✓ Search: < 100ms
✓ API latency: p95 < 500ms
```

### Security
```
✓ No cross-tenant data leaks
✓ All queries filtered by tenant
✓ X-Tenant-ID header required
✓ All input validated
✓ No SQL injection possible
```

### User Experience
```
✓ UI loads instantly
✓ Interactions responsive
✓ Mobile view works
✓ Filters intuitive
✓ Errors clear
✓ Actions reversible
```

---

## 🚀 Rollback Plan

### If Issues Occur

```
1. Identify issue
   - Check error logs
   - Identify affected users
   - Assess severity

2. Immediate actions
   - Disable marketplace route (API returns 503)
   - Notify users
   - Start investigation

3. Rollback decision
   - Minor issue: hotfix
   - Major issue: rollback

4. Rollback procedure
   - Revert backend code
   - Revert frontend code
   - Restore from backup if needed
   - Notify users

5. Investigation
   - Document what went wrong
   - Identify root cause
   - Prevent recurrence
```

---

## 📈 KPIs to Track

### Usage
- [ ] Daily active users browsing marketplace
- [ ] Avg items added per tenant
- [ ] Adoption rate (% of tenants using)
- [ ] Items per category (adoption by type)

### Performance
- [ ] Page load time (milliseconds)
- [ ] API response time (milliseconds)
- [ ] Error rate (%)
- [ ] Availability (%)

### Quality
- [ ] Bug reports per week
- [ ] User satisfaction score
- [ ] Feature requests
- [ ] Support tickets

### Business
- [ ] Time saved vs manual process
- [ ] Revenue impact
- [ ] User retention
- [ ] Net Promoter Score (NPS)

---

## 📞 Support Contacts

### During Deployment
- Deployment Lead: [Name]
- Backend Engineer: [Name]
- Frontend Engineer: [Name]
- Database Admin: [Name]
- DevOps: [Name]

### Escalation Path
- Level 1: Team Lead
- Level 2: Engineering Manager
- Level 3: Director

---

**Checklist Version:** 1.0  
**Last Updated:** 2024-10-27  
**Status:** ✅ Ready for Deployment
