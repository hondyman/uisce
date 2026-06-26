# Entity Schema Builder v2.1 - Business/Technical Names & Clone Tracking

**Release Date:** October 17, 2025  
**Version:** 2.1  
**Status:** Ready for Testing  

## 📋 What's New

This release adds enterprise-grade naming, semantic term linking, and smart clone tracking to match Workday's Business Object architecture.

### ✨ Major Features

#### 1. **Business Name + Technical Name System**
Every entity, subtype, and field now has two names:
- **Business Name** (Display): "Legal Name", "Date of Birth", "Client Investor"
- **Technical Name** (System): "legal_name", "date_of_birth", "client_investor"

**How It Works:**
```
User enters:    "Legal Name"
                       ↓
Auto-generated: "legal_name" (lowercase, underscores)
```

**Benefits:**
- Business analysts work with readable names
- Developers get clean technical identifiers
- Consistent naming convention (snake_case)
- Easy migration to data warehouses

**Where It Applies:**
- Entity level: Business Name + Technical Name
- Subtype level: Business Name + Technical Name  
- Field level: Business Name + Technical Name
- Clone tracking: Shows parent name in UI

#### 2. **Smart Clone Tracking & Parent Upgrades**
When you clone a core BO, the system tracks:
- Which core BO it came from
- Parent name for UI display
- Which fields are inherited vs custom
- Upgrade path when core is updated

**Example:**
```
Cloned: "Client Investor (Custom)" from "client_investor"
        ↓
Shows: "Cloned from: Client Investor" with 🔗 icon
        ↓
Users can: See which fields are core (read-only) vs custom (editable)
           Upgrade inherited fields when core BO updates
```

**Clone Anatomy:**
```
PARENT (Core BO - Immutable)
├── 🔒 Entity Fields
│   ├── investor_id (inherited)
│   ├── legal_name (inherited)
│   └── aum (inherited)
└── 🔒 Subtypes
    ├── IndividualInvestor (inherited)
    └── InstitutionalInvestor (inherited)

CLONE (Custom - Editable)
├── ✏️ Parent Reference → "client_investor"
├── ✏️ Parent Display Name → "Client Investor"
├── 🔒 Core Fields (read-only - from parent)
│   ├── investor_id
│   ├── legal_name
│   └── aum
├── ✏️ Custom Fields (editable - user-added)
│   └── esg_focus (new field)
└── ✏️ Subtypes (editable)
    ├── IndividualInvestor (core - inherited)
    └── HighNetWorth (custom - new subtype)
```

#### 3. **Semantic Term Linking**
Fields can now be linked to semantic terms from your data catalog.

**How It Works:**
1. User creates field "Account Balance"
2. Modal shows: "Choose semantic term..." dropdown
3. User selects: "Account Balance" from catalog
4. Field stores: `semanticTermId` and `semanticTermName`
5. UI displays: Link icon showing field → semantic term

**Benefits:**
- Bridges business models to data governance
- Links entity fields to business glossary
- Enables semantic analysis tools
- Prepares for data lineage automation

**Example in UI:**
```
FIELD TABLE
┌─────────────┬──────────────┬────────┬─────────────────────┐
│ Business    │ Technical    │ Type   │ Semantic Term       │
├─────────────┼──────────────┼────────┼─────────────────────┤
│ Legal Name  │ legal_name   │ text   │ Customer Name       │
│ Account     │ account_bal  │ number │ Account Balance     │
│ Balance     │              │        │                     │
└─────────────┴──────────────┴────────┴─────────────────────┘
```

#### 4. **Unified Entity Editor (Single Screen)**
The new drawer shows everything on one screen with tabs:
- **📋 Entity Tab**: Entity fields + rename controls + clone parent info
- **📦 Subtypes Tab**: All subtypes with their fields

**Single Screen Benefits:**
- See full entity structure without navigation
- CRUD subtypes and fields without modal hopping
- View inherited vs custom fields side-by-side
- Lock/unlock icons show field status (🔒 Core / 🔓 Custom)

#### 5. **Field Status Indicators**
Each field shows its status with color + icon:
```
🔒 CORE    (blue)   → Inherited from parent BO, read-only
🔓 CUSTOM  (green)  → User-created, can delete
```

---

## 🎯 Use Cases

### Use Case 1: Clone & Customize Core BO

**Scenario:**
You want a custom version of "Client Investor" with extra fields for wealth management.

**Steps:**
```
1. Find "Client Investor" (🔒 CORE BO)
2. Click 🔄 Clone button
3. New entity created: "Client Investor (Custom)"
4. Click ✏️ Edit
5. Go to "🔧 Entity Fields" tab
6. Click "Add Field to Entity"
7. Enter: Business Name = "ESG Focus"
   → Auto-generates: Technical Name = "esg_focus"
8. Select type: "text"
9. Link to semantic term: "ESG Classification" (from catalog)
10. Click OK
11. Field added to ✏️ CUSTOM FIELDS section
12. Click "SAVE & APPLY"
```

**Result:**
```
CUSTOM: Client Investor (Custom)
├── 🔒 CORE FIELDS (inherited from Client Investor)
│   ├── investor_id
│   ├── legal_name
│   ├── email
│   ├── phone
│   └── aum
└── ✏️ CUSTOM FIELDS (new)
    └── esg_focus → Semantic: ESG Classification
```

### Use Case 2: Rename Clone for Clarity

**Scenario:**
User cloned "Client Investor" but wants to call it "Wealth Management Client" in their org.

**Steps:**
```
1. Click ✏️ Edit on the cloned entity
2. Go to 📋 Entity tab
3. See "Rename Entity" card (only for custom entities!)
4. Change "Business Name" field from "Client Investor (Custom)" 
   to "Wealth Management Client"
5. Technical Name auto-updates to "wealth_management_client"
6. Save
```

**Result:**
```
Entity renamed ✓
Business Name: Wealth Management Client
Technical Name: wealth_management_client (auto)
Parent Track: 🔗 Cloned from: Client Investor
```

### Use Case 3: Add Subtype with Fields

**Scenario:**
Custom entity needs new subtype "Elite Investor" with special fields.

**Steps:**
```
1. Click ✏️ Edit on "Wealth Management Client"
2. Go to 📦 Subtypes tab
3. Click "Add Subtype"
4. Enter: "Elite Investor"
   → Auto-generates technical: "elite_investor"
5. New subtype created
6. Click "Add Field" on the new subtype
7. Enter:
   - Business Name: "Minimum Asset Threshold"
   - Type: number
   - Semantic Term: "Investment Threshold" (from catalog)
8. Click OK
9. Field added to "Elite Investor" subtype
10. "SAVE & APPLY"
```

**Result:**
```
📦 Subtypes (including new):
├── 🔒 Individual Investor (core)
├── 🔒 Institutional Investor (core)
└── ✏️ Elite Investor (custom)
    └── Fields:
        ├── 🔓 Minimum Asset Threshold (number)
           → Semantic: Investment Threshold
```

### Use Case 4: Track Core Upgrades

**Scenario:**
Core "Portfolio" BO adds a new field. Users want to see if their clone can use it.

**Steps:**
```
1. User has clone: "My Portfolio" (cloned from "portfolio")
2. Core "Portfolio" BO is updated with new field
3. User edits "My Portfolio"
4. In 📋 Entity tab, sees:
   "🔗 Clone Parent: Portfolio (Upgradeable)"
   "When the core BO is upgraded, you can merge updates..."
5. User can manually add inherited fields or wait for merge tool
```

**Result:**
- Upgrade path is documented
- Users aware core BO evolved
- Can adopt new inherited fields on their schedule

---

## 🛠️ Technical Details

### Data Structure

**Entity Type (Updated):**
```typescript
interface Entity {
  key?: string;                    // technical_name (lowercase_with_underscores)
  name: string;                    // Display name
  businessName?: string;           // Business-friendly name
  technicalName?: string;          // Lowercase_with_underscores
  description?: string;            // Description
  entity_fields: Field[];          // All entity-level fields
  subtypes: Record<string, Subtype>;
  isCore?: boolean;                // true = core BO, false = custom
  coreFields?: Field[];            // Separated core fields
  customFields?: Field[];          // Separated custom fields
  clonesFrom?: string;             // Old: Parent entity key
  clonesFromKey?: string;          // New: Parent technical key
  cloneParentName?: string;        // New: Parent display name for UI
}
```

**Field Type (Updated):**
```typescript
interface Field {
  key: string;                     // technical_name
  name: string;                    // Display name
  businessName?: string;           // Business-friendly name
  technicalName?: string;          // Lowercase_with_underscores
  type: 'text' | 'number' | 'date' | 'boolean';
  isCore?: boolean;                // From core BO?
  inheritedFrom?: string;          // Inherited from which entity
  inheritedFromKey?: string;       // Inherited from which technical key
  semanticTermId?: string;         // Link to catalog semantic term
  semanticTermName?: string;       // Display name of semantic term
}
```

**Subtype Type (Updated):**
```typescript
interface Subtype {
  key?: string;                    // technical_name
  name: string;                    // Display name
  businessName?: string;           // Business-friendly name
  technicalName?: string;          // Lowercase_with_underscores
  subtype_fields: Field[];         // Fields in this subtype
  isCore?: boolean;                // Core vs custom
  basedOnEntity?: string;          // Which entity it's based on
}
```

### Naming Utility Functions

**File:** `frontend/src/utils/nameFormatting.ts`

```typescript
// Convert business name → technical name
businessToTechnicalName(businessName: string): string
// "Legal Name" → "legal_name"
// "Date of Birth" → "date_of_birth"

// Convert technical name → business name
technicalToBusinessName(technicalName: string): string
// "legal_name" → "Legal Name"
// "date_of_birth" → "Date of Birth"

// Validate technical name format
isValidTechnicalName(technicalName: string): boolean
// "legal_name" → true
// "Legal Name" → false (has spaces)

// Normalize both names together
normalizeName(businessName: string | undefined, technicalName: string | undefined)
// Returns: { businessName: "...", technicalName: "..." }
// Auto-generates missing name based on provided one
```

### GraphQL Query for Semantic Terms

**File:** `frontend/src/graphql/queries/datasourceQueries.ts`

```typescript
export const GET_AVAILABLE_SEMANTIC_TERMS = gql`
  query GetAvailableSemanticTerms($datasourceId: uuid!) {
    catalog_node(
      where: {
        tenant_datasource_id: { _eq: $datasourceId }
        node_type_id: { _eq: "820b942a-9c9e-4abc-acdc-84616db33098" }
      }
      order_by: { node_name: asc }
    ) {
      id
      node_name
      description
      properties
      qualified_path
      created_at
      updated_at
    }
  }
`;
```

Queries semantic terms (node_type_id: `820b942a...`) from catalog_node table.
Returns: List of semantic terms available for field linking.

---

## 📊 UI Changes

### Entity Cards - Enhanced

**Before:**
```
┌─────────────────────┐
│ ClientInvestor BO   │
│ Subtypes: 2         │
│ [Edit] [Clone]      │
└─────────────────────┘
```

**Now:**
```
┌─────────────────────────────────────┐
│ 🔒 CORE BO                          │
│                                     │
│ Client Investor                     │
│ Technical: client_investor          │
│                                     │
│ Core BO: Investor profile with      │
│ relationship tracking               │
│                                     │
│ Subtypes:                           │
│ [Individual Investor] [Inst...]     │
│                                     │
│ [Edit] [Clone] [Delete]             │
└─────────────────────────────────────┘
```

### Entity Editor Drawer - Reorganized

**Tabs:**
1. **📋 Entity** - Entity fields + rename + parent info
2. **📦 Subtypes** - All subtypes with inline field CRUD

**Entity Tab Shows:**
- Rename controls (custom only)
- Clone parent info (if cloned)
- Entity-level field table
- Add field button

**Subtypes Tab Shows:**
- Add subtype button
- For each subtype:
  - Subtype name + technical name + badge
  - Inline fields table
  - Add field button
  - Delete button (custom only)

### Field Status Indicators

**Visual Feedback:**
```
🔒 CORE (blue)   - Inherited, read-only, delete not allowed
🔓 CUSTOM (green) - User-created, can delete
📌 SEMANTIC TERM - Green link icon shows field → term binding
🔗 CLONE PARENT  - Shows which core BO this entity cloned from
```

---

## 🔄 Data Flow

### Clone Creation

```
User clicks Clone on core BO
         ↓
System checks name collision (e.g., client_investor_custom_1)
         ↓
Creates new entity with:
  - All core fields marked as inherited
  - clonesFromKey = "client_investor"
  - cloneParentName = "Client Investor"
  - isCore = false
  - customFields = [] (empty initially)
         ↓
Entity added to entities state
         ↓
User clicks SAVE & APPLY
         ↓
Delta payload sent: { changed: { client_investor_custom_1: {...} } }
         ↓
Backend stores in entity_schema table
         ↓
On reload: fetchEntitySchema() retrieves clone + core BO
         ↓
Merged state: CORE_ENTITIES + saved schema
```

### Field Addition to Clone

```
User clicks "Add Field" on custom subtype
         ↓
Modal opens with:
  - Field name input
  - Type selector
  - Semantic term dropdown (populated from catalog)
         ↓
User enters: "ESG Focus" (business name)
  - Auto-generates: "esg_focus" (technical name)
  - Selects type: "text"
  - Links to: "ESG Classification" (semantic term id + name)
         ↓
Field created:
  {
    key: "esg_focus",
    name: "ESG Focus",
    businessName: "ESG Focus",
    technicalName: "esg_focus",
    type: "text",
    isCore: false,
    semanticTermId: "term-123",
    semanticTermName: "ESG Classification"
  }
         ↓
Added to customFields array
         ↓
User clicks SAVE & APPLY
         ↓
Entire entity (with new field) sent to backend
         ↓
Backend stores updated entity_schema
```

---

## ✅ Testing Checklist

- [ ] Clone core BO → new custom entity created with all fields
- [ ] Clone name appears as "parent_name (Custom)"
- [ ] Clone shows 🔗 "Cloned from: parent_name"
- [ ] Rename clone → technical name auto-updates
- [ ] Add field with business name → technical name auto-generates
- [ ] Add field with semantic term link → displays in field list
- [ ] Add subtype → creates with business + technical names
- [ ] Add field to subtype → field appears with status badge
- [ ] Delete custom field → deleted, confirm shows
- [ ] Delete core field → delete button not shown (read-only)
- [ ] Inherited fields → show 🔒 CORE badge, no delete button
- [ ] Custom fields → show 🔓 CUSTOM badge, can delete
- [ ] Save & Apply → delta sent, counts shown
- [ ] Reload page → persisted schema loads, clone tracking maintained
- [ ] Search → filters by business name, technical name, description
- [ ] Semantic terms dropdown → populated from catalog_node table
- [ ] Multiple clones → each gets unique name (e.g., _custom_1, _custom_2)

---

## 📚 User Documentation

### For Business Analysts
- Use **Business Names** (e.g., "Legal Name") for all entity/subtype/field names
- Technical names auto-generate - no need to worry about them
- Link fields to semantic terms to document business meaning
- Clone core BOs for quick customization

### For Data Engineers
- **Technical Names** (e.g., "legal_name") are available for code/scripts
- Clone tracking (clonesFromKey) enables automated upgrade merge tools
- Semantic term links (semanticTermId) bridge to data catalog
- Field inheritance (isCore flag) separates core vs custom for reports

### For IT/Governance
- Core BOs define standards (immutable, marked 🔒)
- Custom clones allow org-specific extensions
- Clone parent tracking enables compliance audits
- Semantic term linking enables data lineage automation

---

## 🚀 Deployment Checklist

- [x] entity-schema.ts types updated (business/technical names)
- [x] EntityConfigPageV2.tsx rewritten with new features
- [x] nameFormatting.ts utility created
- [x] GraphQL query GET_AVAILABLE_SEMANTIC_TERMS added
- [ ] Frontend dev server tested
- [ ] Backend persistence verified
- [ ] All edge cases tested (empty clones, special chars, etc.)
- [ ] Documentation finalized
- [ ] User training materials created

---

## 🔮 Future Enhancements

1. **Bulk Clone** - Clone multiple entities at once
2. **Clone Merge Tool** - Automatically merge updated core fields into clones
3. **Field Constraints** - Add validation rules (min/max, pattern, enum)
4. **Computed Fields** - Fields that derive from other fields
5. **Field Permissions** - User roles that can edit which fields
6. **Entity Versioning** - Track entity evolution over time
7. **Diff Viewer** - Compare clone vs core BO side-by-side
8. **Export/Import** - Bundle entities for sharing between tenants

---

## 📞 Support

**Questions about naming:**
- See `frontend/src/utils/nameFormatting.ts` for conversion logic
- Business names shown in UI, technical names in storage/APIs

**Questions about clones:**
- Check `clonesFromKey` and `cloneParentName` fields
- Indicates which core BO a custom entity came from

**Questions about semantic terms:**
- Query `catalog_node` table with node_type_id `820b942a...`
- Fields link via `semanticTermId` and `semanticTermName`

**Questions about inheritance:**
- `isCore: true` = inherited field (read-only)
- `isCore: false` = custom field (can delete)
- Use for UI badges and permission logic

---

**Status:** Ready for User Testing  
**Last Updated:** October 17, 2025  
**Version:** 2.1
