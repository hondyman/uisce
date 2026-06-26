# Schema Consolidation - Visual Guide

## 🔄 Before & After Visualization

### BEFORE: Distributed (Messy)
```
┌─────────────────────────────────────────────────────────────────┐
│ ALPHA DATABASE (localhost:5432)                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
│  │    BANKING       │  │ CAPITAL_MARKETS  │  │  CURRENCY_FX  │  │
│  ├──────────────────┤  ├──────────────────┤  ├───────────────┤  │
│  │metrics_registry  │  │metrics_registry  │  │metrics_registry│  │
│  │ (10 records)  [✓]│  │ (10 records)  [✓]│  │ (11 records) [✓]  │
│  │dax_functions [✓] │  │dax_functions [✓] │  │  [no dax]        │
│  └──────────────────┘  └──────────────────┘  └───────────────┘  │
│                                                                   │
│  ┌──────────────────────┐  ┌──────────────────┐  ┌────────────┐ │
│  │  FINANCIAL_SERVICES  │  │  FIXED_INCOME    │  │ HEALTHCARE │ │
│  ├──────────────────────┤  ├──────────────────┤  ├────────────┤ │
│  │metrics_registry      │  │metrics_registry  │  │metrics_regist│ │
│  │ (60 records)   [✓]   │  │ (9 records)  [✓] │  │ (10 records) │ │
│  │dax_functions [✓]     │  │  [no dax]        │  │dax_functions │ │
│  └──────────────────────┘  └──────────────────┘  └────────────┘ │
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
│  │    INSURANCE     │  │   REGULATORY     │  │    RETAIL     │  │
│  ├──────────────────┤  ├──────────────────┤  ├───────────────┤  │
│  │metrics_registry  │  │metrics_registry  │  │metrics_registry│  │
│  │ (10 records)  [✓]│  │ (10 records)  [✓]│  │ (10 records) [✓]  │
│  │dax_functions [✓] │  │dax_functions [✓] │  │dax_functions [✓]  │
│  └──────────────────┘  └──────────────────┘  └───────────────┘  │
│                                                                   │
│  ┌─────────────────────────────┐  ┌──────────────────────────┐   │
│  │  INVESTMENT_ACCOUNTING      │  │ UNIFIED_FINANCIAL_SVCS  │   │
│  ├─────────────────────────────┤  ├──────────────────────────┤   │
│  │metrics_registry (16 records)│  │metrics_registry (82 rec.)│   │
│  │  [no dax]                   │  │dax_functions        [✓]  │   │
│  └─────────────────────────────┘  └──────────────────────────┘   │
│                                                                   │
│  ┌──────────────────────────┐                                    │
│  │  WEALTH_MANAGEMENT       │                                    │
│  ├──────────────────────────┤                                    │
│  │metrics_registry (26 rec.)│                                    │
│  │  [no dax]                │                                    │
│  └──────────────────────────┘                                    │
│                                                                   │
│  [Plus 5 other schemas: foffice, hdb_catalog, hld,              │
│   report_sys, semantic_layer, sml - not shown for brevity]      │
│                                                                   │
│  ┌─────────────────────────────────────────┐                    │
│  │              PUBLIC SCHEMA              │                    │
│  │  (other application tables here)        │                    │
│  └─────────────────────────────────────────┘                    │
└─────────────────────────────────────────────────────────────────┘

SUMMARY:
  • 12 copies of metrics_registry (264 total records)
  • 8 copies of dax_functions
  • Massive duplication & maintenance burden
```

### AFTER: Consolidated (Clean)
```
┌─────────────────────────────────────────────────────────────────┐
│ ALPHA DATABASE (localhost:5432)                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌───────────────┐  │
│  │    BANKING       │  │ CAPITAL_MARKETS  │  │  CURRENCY_FX  │  │
│  ├──────────────────┤  ├──────────────────┤  ├───────────────┤  │
│  │ [no local tables]│  │ [no local tables]│  │[no local tables│  │
│  │ (metrics moved ↓)│  │ (metrics moved ↓)│  │(metrics moved ↓)  │
│  └──────────────────┘  └──────────────────┘  └───────────────┘  │
│                                                                   │
│  [... other domain schemas - cleaned up ...]                    │
│                                                                   │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                  PUBLIC SCHEMA                            │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                            │  │
│  │  ┌──────────────────────────────────────────────────────┐ │  │
│  │  │ metrics_registry                                     │ │  │
│  │  ├──────────────────────────────────────────────────────┤ │  │
│  │  │ id (PK)                                              │ │  │
│  │  │ node_id                                              │ │  │
│  │  │ schema_domain ← NEW! Tracks which domain            │ │  │
│  │  │ category, description, formula_type, formula        │ │  │
│  │  │ ... (other columns)                                  │ │  │
│  │  │                                                       │ │  │
│  │  │ 264 TOTAL RECORDS (all domains consolidated)        │ │  │
│  │  │                                                       │ │  │
│  │  │ Indexes:                                             │ │  │
│  │  │  • schema_domain (fast filtering by domain)         │ │  │
│  │  │  • node_id       (unique per domain)                │ │  │
│  │  │  • category      (filtering by type)                │ │  │
│  │  └──────────────────────────────────────────────────────┘ │  │
│  │                                                            │  │
│  │  ┌──────────────────────────────────────────────────────┐ │  │
│  │  │ dax_functions                                        │ │  │
│  │  ├──────────────────────────────────────────────────────┤ │  │
│  │  │ id (PK)                                              │ │  │
│  │  │ name                                                 │ │  │
│  │  │ schema_domain ← NEW! Tracks which domain            │ │  │
│  │  │ class, badge, description                           │ │  │
│  │  │ created_at                                           │ │  │
│  │  │                                                       │ │  │
│  │  │ ALL DAX FUNCTIONS CONSOLIDATED                      │ │  │
│  │  │                                                       │ │  │
│  │  │ Indexes:                                             │ │  │
│  │  │  • schema_domain (fast filtering by domain)         │ │  │
│  │  │  • name          (unique per domain)                │ │  │
│  │  └──────────────────────────────────────────────────────┘ │  │
│  │                                                            │  │
│  │  (other application tables here)                          │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘

BENEFITS:
  ✓ Single table for all metrics (easy queries)
  ✓ Single table for all dax functions
  ✓ Domain tracking via schema_domain column
  ✓ ~94% storage reduction
  ✓ Simpler maintenance & governance
```

---

## 📊 Data Flow Diagram

### Query Flow: Before vs After

#### BEFORE: Complicated Multi-Schema Queries
```
Developer
    ↓
Application
    ↓
├─→ SELECT * FROM banking.metrics_registry
│   └─→ 10 records
│
├─→ SELECT * FROM capital_markets.metrics_registry
│   └─→ 10 records
│
├─→ SELECT * FROM retail.metrics_registry
│   └─→ 10 records
│
└─→ Union results in application code
    └─→ Hard to query across domains!
```

#### AFTER: Simple Consolidated Queries
```
Developer
    ↓
Application
    ↓
├─→ SELECT * FROM public.metrics_registry
│   WHERE schema_domain = 'banking'
│   └─→ 10 records from banking
│
└─→ Or easily across multiple domains:
    SELECT * FROM public.metrics_registry
    WHERE schema_domain IN ('banking', 'retail')
    └─→ 20 records from both
```

---

## 🔄 Migration Sequence

```
Step 1: Create New Tables
┌─────────────────────────────────────────┐
│ public.metrics_registry (empty)         │
│ public.dax_functions (empty)            │
└─────────────────────────────────────────┘

Step 2: Create Indexes
┌─────────────────────────────────────────┐
│ idx_metrics_registry_schema_domain      │
│ idx_metrics_registry_node_id            │
│ idx_metrics_registry_category           │
│ idx_dax_functions_schema_domain         │
│ idx_dax_functions_name                  │
└─────────────────────────────────────────┘

Step 3: Migrate Data
┌─────────────────────────────────────────┐
│ banking.metrics_registry (10)           │ ──┐
│ capital_markets.metrics_registry (10)   │   │
│ financial_services.metrics_registry (60)│   ├─→ public.metrics_registry
│ ... (12 total)                          │   │   (264 records)
│                                         │   │
│ banking.dax_functions (N)               │   │
│ capital_markets.dax_functions (N)       │   ├─→ public.dax_functions
│ ... (8 total)                           │ ──┤   (all records)
└─────────────────────────────────────────┘

Step 4: Verify
┌─────────────────────────────────────────┐
│ SELECT COUNT(*) FROM public.metrics_    │
│   registry = 264 ✓                      │
│                                         │
│ SELECT COUNT(DISTINCT schema_domain)    │
│   = 12 (banking, capital_markets, ...) ✓
└─────────────────────────────────────────┘

Step 5: Optional - Create Backwards Compatibility Views
┌─────────────────────────────────────────┐
│ CREATE VIEW banking.metrics_registry    │
│   AS SELECT * FROM public.metrics_...   │
│   WHERE schema_domain = 'banking' ✓     │
│                                         │
│ (Allows old queries to work temporarily)
└─────────────────────────────────────────┘

Step 6: Update Application Code
┌─────────────────────────────────────────┐
│ Replace queries with new format         │
│ FROM banking.metrics_registry           │ ──→ Remove schema prefix
│ FROM public.metrics_registry            │ ──→ Add WHERE schema_domain
│   WHERE schema_domain = 'banking'       │
└─────────────────────────────────────────┘

Step 7: Drop Old Tables (Optional)
┌─────────────────────────────────────────┐
│ DROP TABLE banking.metrics_registry ✓   │
│ DROP TABLE capital_markets.metrics_... ✓│
│ (After code migration complete)         │
└─────────────────────────────────────────┘
```

---

## 📈 Query Pattern Evolution

### Pattern 1: Single Domain (Simple Change)
```
OLD: SELECT * FROM banking.metrics_registry WHERE category = 'perf'
NEW: SELECT * FROM public.metrics_registry 
     WHERE schema_domain = 'banking' AND category = 'perf'
```

### Pattern 2: Multiple Domains (Much Better!)
```
OLD: 
SELECT * FROM banking.metrics_registry WHERE category = 'perf'
UNION ALL
SELECT * FROM retail.metrics_registry WHERE category = 'perf'
UNION ALL
SELECT * FROM insurance.metrics_registry WHERE category = 'perf'
-- Repeat for each domain manually :(

NEW:
SELECT * FROM public.metrics_registry 
WHERE schema_domain IN ('banking', 'retail', 'insurance')
AND category = 'perf'
-- Much cleaner! :)
```

### Pattern 3: All Domains (Simplest!)
```
OLD: Have to query each schema separately :(

NEW:
SELECT * FROM public.metrics_registry
ORDER BY schema_domain, category
-- Get everything in one query!
```

---

## 🎯 Timeline Visualization

### Quick Path (Option A) - 2-3 hours
```
     Hour 0              Hour 1              Hour 2           Hour 3
     |────────────────────|────────────────────|────────────────|────|
Analyze  Backup  Migrate   Find Code  Update   Test   Deploy   Done
  ↓       ↓        ↓       References  Code     ↓       ↓       ✓
5min    2min    1min      10min       1-2h    30min   varies
```

### Staged Path (Option B) - 1-2 days
```
Day 1:                          Day 2:
|─────────────────────|        |─────────────────────|
Analyze Backup Migrate Views   Update Test Deploy Cleanup
  ↓     ↓     ↓     ↓          Code ↓   ↓    ↓
30min total                    1-2hrs  30min varies  5min
```

---

## 💾 Storage Impact

### BEFORE
```
storage_consumed
    ↑
    │  ┌─ banking.metrics_registry
100 │  │
    │  ├─ capital_markets.metrics_registry
    │  │
    │  ├─ financial_services.metrics_registry (largest)
    │  │
    │  ├─ investment_accounting.metrics_registry
    │  │
    │  └─ [10 more copies of same structure]
    │
    └──────────────────────────────────────────→
```

### AFTER
```
storage_consumed
    ↑
    │  ┌─ public.metrics_registry
100 │  │  (all data consolidated)
    │  │
    │  │
    │  │
    │  │
    │  ├─ public.dax_functions
    │  │
    │  └──────────────────────────────────────────→

Result: ~94% storage reduction for these tables!
```

---

## 🔐 Data Integrity Safeguards

```
┌────────────────────────────────────────────────────┐
│ Consolidation Safety Features                      │
├────────────────────────────────────────────────────┤
│                                                    │
│ 1. UNIQUE(node_id, schema_domain)                 │
│    └─→ Prevents duplicate records                │
│                                                    │
│ 2. ON CONFLICT DO NOTHING                         │
│    └─→ Safe if migration runs twice              │
│                                                    │
│ 3. Backup Required                                │
│    └─→ Easy rollback if needed                   │
│                                                    │
│ 4. Backwards Compatibility Views (Optional)       │
│    └─→ Old queries keep working temporarily      │
│                                                    │
│ 5. Verification Queries Included                  │
│    └─→ Check record counts before/after         │
│                                                    │
│ 6. Can Re-Run Migration                           │
│    └─→ Fully idempotent - safe!                  │
│                                                    │
└────────────────────────────────────────────────────┘
```

---

## 📊 Schema_Domain Column Benefits

```
OLD APPROACH:
┌──────────────────┬─────────────────┐
│ Banking Schema   │ Records: 10     │
├──────────────────┼─────────────────┤
│ Capital Markets  │ Records: 10     │
├──────────────────┼─────────────────┤
│ Retail Schema    │ Records: 10     │
└──────────────────┴─────────────────┘
(Spread across 12 separate tables)

NEW APPROACH:
┌─────────────────┬──────────────────────┬────────────┐
│ schema_domain   │ node_id              │ category   │
├─────────────────┼──────────────────────┼────────────┤
│ banking         │ metric_return_on...  │ performance│
│ banking         │ metric_risk_parity...│ risk       │
│ capital_markets │ metric_volatility... │ risk       │
│ capital_markets │ metric_correlation...│ performance│
│ retail          │ metric_transaction...│ volume     │
│ ...             │ ...                  │ ...        │
└─────────────────┴──────────────────────┴────────────┘
(All in one table with domain tracking!)
```

---

## ✅ Success Checklist as Visualization

```
Pre-Migration
  [ ] Read documentation
  [ ] Run analysis
  [ ] Backup database
  
Migration
  [ ] Execute SQL
  [ ] Verify record counts
  [ ] Check indexes
  
Post-Migration
  [ ] Update code
  [ ] Run tests
  [ ] Deploy
  
Cleanup
  [ ] Drop old tables
  [ ] Update docs
  [ ] Team notification
  
                ✅ COMPLETE!
```

---

## 🎓 Key Takeaways

```
┌─────────────────────────────────────────────────────┐
│  What's Changing                                    │
├─────────────────────────────────────────────────────┤
│                                                     │
│  1. Location                                        │
│     12 schemas → 1 schema (public)                 │
│                                                     │
│  2. Organization                                    │
│     12 tables → 1 table (with domain tracking)     │
│                                                     │
│  3. Querying                                        │
│     Add WHERE schema_domain = '...' to queries    │
│                                                     │
│  4. Maintenance                                     │
│     1 place to update instead of 12                │
│                                                     │
│  5. Performance                                     │
│     Better indexes, simpler queries                │
│                                                     │
└─────────────────────────────────────────────────────┘
```

---

This visual guide complements the detailed documentation. Refer back to these diagrams when:
- Explaining the change to team members
- Understanding the before/after state
- Following the migration sequence
- Reviewing query pattern changes
