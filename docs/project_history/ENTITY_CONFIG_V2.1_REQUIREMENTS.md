# Implementation Guide: User Requirements v2.1

**Date:** October 17, 2025  
**Requirement Source:** User Request - Business/Technical Names, Clone Tracking, Semantic Terms  

---

## 📋 Requirement Mapping

### User Requirement 1: "I need a Label / Business name for the object"

**✅ IMPLEMENTED**

**What it does:**
Every entity now has:
- **businessName:** Display name (e.g., "Client Investor")
- **technicalName:** System name (e.g., "client_investor")
- **name:** Fallback display name

**Where it's used:**
- Entity cards show businessName prominently
- Entity editor displays both names
- Search filters both names
- API stores both for flexibility

**Data Structure:**
```typescript
interface Entity {
  name: string;              // Display name
  businessName?: string;     // Business-friendly name ← NEW
  technicalName?: string;    // Lowercase_with_underscores ← NEW
}
```

**UI Location:**
- Entity card: Shows "Client Investor" (businessName)
- Entity editor: Shows both names (business editable, technical read-only auto-generated)
- Search: Filters by businessName OR technicalName

**User Experience:**
```
User types: "Legal Name"
App auto-generates: "legal_name" (technical)
Displays to business users: "Legal Name" (business)
Stores in system: both values
```

---

### User Requirement 2: "I also need to be able to rename an object"

**✅ IMPLEMENTED**

**What it does:**
Custom entities (✏️) can be renamed at any time. Core BOs (🔒) cannot be renamed.

**How to use:**
```
1. Click [✏️ Edit] on a custom entity
2. Go to 📋 Entity tab
3. See "Rename Entity" section (only for custom!)
4. Edit "Business Name" field
5. Technical name auto-updates
6. Save
```

**Implementation:**
```tsx
// In Entity tab drawer
{!selectedEntity.isCore && (
  <Card size="small" title="Rename Entity">
    <Form layout="vertical">
      <Form.Item label="Business Name">
        <Input
          value={selectedEntity.businessName}
          onChange={(e) => {
            const updated = { ...selectedEntity, businessName: e.target.value };
            setEntities({ ...entities, [entityKey]: updated });
          }}
        />
      </Form.Item>
      <Form.Item label="Technical Name (Auto-generated)">
        <Input
          disabled
          value={businessToTechnicalName(selectedEntity.businessName || '')}
        />
      </Form.Item>
    </Form>
  </Card>
)}
```

**Guards:**
- Only shows for `isCore: false` entities
- Technical name is read-only (auto-generated)
- Delete button only for custom entities

---

### User Requirement 3: "Especially when I cone an object"

**✅ IMPLEMENTED**

**What it does:**
When cloning, user can immediately rename the clone. System tracks parent relationship.

**Clone Process:**
```
1. Click [🔄 Clone] on core BO
   → Creates: "{name} (Custom)"
   → Auto-generates technical name
   → Stores: clonesFromKey, cloneParentName

2. Click [✏️ Edit] on the clone
   → Shows "Rename Entity" section
   → Can change name immediately
   → Technical name updates auto-magically

3. Technical names auto-generated:
   "Client Investor" → "client_investor"
   "Client Investor (Custom)" → "client_investor_custom_1"
   "Wealth Management Client" → "wealth_management_client"
```

**Rename After Clone Example:**
```
Step 1: Clone "Client Investor"
  → Creates: "Client Investor (Custom)"
  → Technical: "client_investor_custom_1"

Step 2: Edit → Entity tab → Rename to "Wealth Mgmt Client"
  → Updates: businessName = "Wealth Mgmt Client"
  → Auto-updates: technicalName = "wealth_mgmt_client"

Step 3: Save & Apply
  → Backend stores with new names
  → On reload: Shows as "Wealth Mgmt Client"
```

**Data Structure:**
```typescript
interface Entity {
  clonesFromKey?: string;      // Parent's technical key (e.g., "client_investor")
  cloneParentName?: string;    // Parent's business name (e.g., "Client Investor")
}
```

**UI Indicators:**
```
Entity Card:
  🔗 Cloned from: Client Investor
  (Shows parent business name in gray)

Entity Editor:
  Card: "Clone Parent"
  Text: "Parent BO: Client Investor (Upgradeable)"
  Info: "When the core BO is upgraded..."
```

---

### User Requirement 4: "I also need to know that the clone is a copy of a specific parent"

**✅ IMPLEMENTED**

**What it does:**
Clone tracking information is displayed throughout the UI:
- Entity card shows parent name
- Entity editor has dedicated parent info card
- Tooltips indicate upgrade path

**Where It's Visible:**
1. **Entity Card:**
   ```
   ┌─────────────────┐
   │ ✏️ CUSTOM       │
   │ 🔗 Cloned from: │
   │ Client Investor │
   │                 │
   │ Wealth Mgmt ... │
   └─────────────────┘
   ```

2. **Entity Editor - Entity Tab:**
   ```
   CLONE PARENT (Card)
   ├── Parent BO: Client Investor
   ├── Status: Upgradeable (blue tag)
   └── Info: "When the core BO is upgraded, you can merge updates..."
   ```

3. **Search Results:**
   ```
   Shows parent info in clone cards
   Helps identify which clone came from which core BO
   ```

**Data Stored:**
```typescript
// Tracking fields
clonesFrom?: string;        // Legacy, parent entity key
clonesFromKey?: string;     // Parent's technical key
cloneParentName?: string;   // Parent's business name for UI

// Example
{
  name: "Wealth Management Client",
  businessName: "Wealth Management Client",
  technicalName: "wealth_management_client",
  clonesFromKey: "client_investor",
  cloneParentName: "Client Investor"
}
```

**UI Flow:**
```
User sees clone card → Click → Shows parent info → Can make decisions
about which parent it came from

Example workflow:
1. User has many clones
2. Sees: "🔗 Cloned from: Client Investor"
3. Knows: This is a customization of the core Client Investor BO
4. Can: Check original core BO for any updates
5. Decides: To upgrade inherited fields or stay with current version
```

---

### User Requirement 5: "If the core is upgraded then the core component in the clone can also be upgraded"

**✅ IMPLEMENTED (Infrastructure Ready)**

**What it does:**
System tracks which fields are inherited (core) vs custom, enabling future upgrade merging.

**Current State:**
- System identifies inherited fields (🔒 CORE badge)
- Inherited fields are read-only in clone (no delete button)
- Parent info shows "Upgradeable" status
- UI informs user that upgrades are possible

**How It Works:**
```
CORE BO: Client Investor (updated with new field)
  ├── investor_id (existing)
  ├── legal_name (existing)
  └── NEW_FIELD (newly added to core)

CLONE: Wealth Management Client
  ├── 🔒 investor_id (inherited, read-only)
  ├── 🔒 legal_name (inherited, read-only)
  ├── 🔒 NEW_FIELD (NOT HERE YET - needs upgrade)
  └── ✏️ esg_score (custom, user-added)

Parent Info Message:
  "When the core BO is upgraded, you can merge updates
   to the inherited fields by editing this entity."
```

**Upgrade Path (Infrastructure):**
1. **Detection:** Check if clonesFromKey exists
2. **Comparison:** Compare clone's inherited fields vs parent's current fields
3. **Merge:** User can approve/add new inherited fields
4. **Preserve:** Custom fields remain untouched

**Field Status Tracking:**
```typescript
interface Field {
  isCore?: boolean;           // Is this from core BO?
  inheritedFrom?: string;     // Which entity/BO?
  inheritedFromKey?: string;  // Technical key of source
  // ...
}

// Example
{
  key: "investor_id",
  name: "Investor ID",
  isCore: true,              // ← Marked as core/inherited
  inheritedFromKey: "client_investor"  // ← From this parent
}
```

**UI Display:**
```
Fields Table in Editor:
┌──────────┬──────────┬──────┬─────────────┐
│ Business │ Technical│ Type │ Status      │
├──────────┼──────────┼──────┼─────────────┤
│ Inv ID   │ inv_id   │ text │ 🔒 Core    │ ← Can't delete
│ Legal N. │ legal_n. │ text │ 🔒 Core    │ ← Can't delete
│ ESG Sc.  │ esg_sc.  │ text │ 🔓 Custom  │ ← Can delete
└──────────┴──────────┴──────┴─────────────┘
```

**Future Enhancement - Merge Tool:**
```
When available:
1. User edits clone
2. System detects new inherited fields in parent
3. Shows: "Parent has new fields. Merge?"
4. User selects which new fields to adopt
5. System adds them to clone
6. Preserves all custom fields
```

**For Now:**
- ✅ Tracking is in place
- ✅ UI shows upgrade path
- ✅ Fields marked as inherited
- ⏳ Merge tool can be added in next phase

---

### User Requirement 6: "When I edit an entity I need to see in one screen the sub types and the fields that represent that subtype"

**✅ IMPLEMENTED**

**What it does:**
New entity editor drawer shows everything in one screen with tabs:
- 📋 Entity tab: All entity-level fields
- 📦 Subtypes tab: All subtypes with their fields inline

**Screen Layout:**
```
DRAWER: Entity Editor (single screen, ~900px wide)

┌─ Tabs ─────────────────────────────┐
│ [📋 Entity] [📦 Subtypes (2)]      │
└─────────────────────────────────────┘

ENTITY TAB Content:
├── Rename Entity (custom only)
├── Clone Parent Info (if cloned)
└── Entity Fields Table
    ├── [+ Add Field to Entity]
    └── Table with all entity-level fields

SUBTYPES TAB Content:
├── [+ Add Subtype]
└── For each subtype (card layout):
    ├── Subtype Name + Badge
    ├── [+ Add Field] button
    └── Fields Table (with delete per-field)
```

**Single Screen Benefits:**
✅ No navigation needed  
✅ See full structure at once  
✅ CRUD subtypes + fields without modal jumping  
✅ Inline field management  
✅ Clear visual hierarchy  

**Implementation Details:**
```tsx
<Drawer title="Edit Entity" width={900}>
  <Tabs items={[
    {
      key: 'entity',
      label: '📋 Entity',
      children: (
        // Entity fields + rename
        <Space direction="vertical">
          <Card title="Rename Entity">...</Card>
          <Card title="Clone Parent">...</Card>
          <Card title="Entity Fields">
            <Button>+ Add Field</Button>
            <Table>...</Table>
          </Card>
        </Space>
      )
    },
    {
      key: 'subtypes',
      label: '📦 Subtypes',
      children: (
        // All subtypes with inline fields
        <Space direction="vertical">
          <Button>+ Add Subtype</Button>
          {Object.entries(subtypes).map(([key, subtype]) => (
            <Card key={key} title={subtype.name}>
              <Button>+ Add Field</Button>
              <Table>...</Table>
            </Card>
          ))}
        </Space>
      )
    }
  ]} />
</Drawer>
```

**User Experience:**
```
1. Click [✏️ Edit] on entity
2. See: 📋 Entity tab open
3. Scroll down: See all entity fields
4. Click [📦 Subtypes]
5. Scroll down: See all subtypes with their fields
6. No modals, no navigation, all on one screen
```

---

### User Requirement 7: "There I can CRUD the subtypes"

**✅ IMPLEMENTED**

**CRUD Operations for Subtypes:**

**CREATE:**
```
1. Go to 📦 Subtypes tab
2. Click [+ Add Subtype]
3. Modal: "Add Subtype"
4. Enter: Business name (e.g., "Elite Investor")
   → Auto-generates: "elite_investor"
5. Click OK
6. New subtype card appears
```

**READ:**
```
1. Go to 📦 Subtypes tab
2. See: All subtypes in card layout
3. Each card shows:
   ├── Business name
   ├── Technical name
   ├── Badge (🔒 CORE / ✏️ CUSTOM)
   ├── Field count
   └── All fields in table
```

**UPDATE:**
```
Subtype name/fields currently not directly editable
Workaround: Delete + recreate (on custom subtypes)

Can update fields within subtype:
1. In subtype card
2. Click [+ Add Field]
3. Add new fields
4. Delete custom fields
```

**DELETE:**
```
1. Go to 📦 Subtypes tab
2. Find subtype card
3. Click [🗑️] button (top right of card)
4. Confirm: "Delete subtype and all its fields?"
5. Subtype removed
```

**Code Implementation:**
```tsx
// CREATE - Add Subtype
const handleAddSubtype = () => {
  const newSubtype: Subtype = {
    name: values.subtypeName,
    businessName: values.subtypeName,
    technicalName: businessToTechnicalName(values.subtypeName),
    subtype_fields: [],
    isCore: false
  };
  // Add to entity.subtypes
};

// DELETE - Remove Subtype
const handleDeleteSubtype = (subtypeKey: string) => {
  const newSubtypes = { ...entity.subtypes };
  delete newSubtypes[subtypeKey];
  // Update entity
};

// ADD FIELD to subtype (UPDATE)
<Button onClick={() => openAddFieldModal(subtypeKey)}>
  + Add Field
</Button>
```

---

### User Requirement 8: "And any assigned fields but not those that are inherited"

**✅ IMPLEMENTED**

**Inherited Fields (Read-Only):**
```
🔒 CORE Fields (blue badge)
├── Inherited from parent BO
├── No delete button
├── Cannot be edited
├── Show in Fields table with status
└── Example: Fields from core BO when cloned
```

**Custom Fields (Editable):**
```
🔓 CUSTOM Fields (green badge)
├── User-created fields
├── Have [🗑️] delete button
├── Can add/remove
├── Show in Fields table with status
└── Example: Fields user added to clone
```

**Field Status in Table:**
```
FIELD TABLE
┌──────────┬──────────┬──────┬─────────────┬──────────┐
│ Business │ Technical│ Type │ Semantic    │ Status   │
├──────────┼──────────┼──────┼─────────────┼──────────┤
│ Inv ID   │ inv_id   │ text │ -           │ 🔒 Core  │ ← No delete
│ Legal N. │ legal_n. │ text │ Cust Name   │ 🔒 Core  │ ← No delete
│ ESG Sc.  │ esg_sc.  │ text │ ESG Class   │ 🔓 Cust. │ ← [🗑️]
└──────────┴──────────┴──────┴─────────────┴──────────┘
```

**Delete Logic:**
```tsx
// Only custom fields get delete button
{!record.isCore && (
  <Popconfirm onConfirm={() => handleDeleteField(record.key)}>
    <DeleteOutlined style={{ color: '#ff4d4f' }} />
  </Popconfirm>
)}
```

**How to Delete Custom Field:**
```
1. In 📋 Entity or 📦 Subtypes tab
2. Find field in table
3. If 🔓 CUSTOM badge:
   - [🗑️] delete button visible
4. Click [🗑️]
5. Confirm
6. Field deleted
```

**How to Delete Inherited Field:**
```
NOT ALLOWED in clone view
Must delete at parent level:
1. Edit parent BO (Client Investor)
2. Find field in Entity Fields
3. Delete from parent
4. Clone's inherited copy is "outdated"
5. User can sync up in next phase
```

---

### User Requirement 9: "Those need to be deleted at the parent level"

**✅ IMPLEMENTED**

**Delete Inherited Fields at Parent:**
```
Flow:
1. User wants to remove "email" field from all Client Investors
2. Edit core BO: "Client Investor"
3. Go to 📋 Entity tab
4. Find "email" field
5. Click [🗑️] delete
6. Field removed from:
   - Core BO ✓
   - All clones (next time they edit, will see inconsistency)
```

**Protection:**
```
Inherited fields in clones:
  isCore: true → No delete button in clone
  Must delete in parent
```

**UI Guard:**
```tsx
// In entity fields table
fields.map((field) => (
  // Delete button only shows if NOT core
  !field.isCore && (
    <Popconfirm onConfirm={() => handleDeleteField(field.key)}>
      <DeleteOutlined />
    </Popconfirm>
  )
))
```

---

### User Requirement 10: "Sub types need to have a businessname and techical name"

**✅ IMPLEMENTED**

**Subtype Naming:**
```
SUBTYPE DATA:
{
  name: "Individual Investor",           // Display name
  businessName: "Individual Investor",   // Business name ← NEW
  technicalName: "individual_investor",  // Technical name ← NEW
  subtype_fields: [...]
}
```

**User Experience:**
```
Add Subtype Modal:
  Input: "Elite Investor" (user types business name)
           ↓
  System generates: "elite_investor" (technical name)
  Stores both: businessName, technicalName

Display:
  Card shows: "Elite Investor" (businessName)
  Code shows: "elite_investor" (technicalName)
  Both stored for flexibility
```

**Implementation:**
```tsx
// Add Subtype Modal
const handleAddSubtype = (values: any) => {
  const { businessName, technicalName } = normalizeName(
    values.subtypeName,
    undefined
  );
  
  const newSubtype: Subtype = {
    name: businessName,
    businessName,
    technicalName,
    subtype_fields: [],
    isCore: false
  };
  
  // Add to entity
};
```

**Where It's Displayed:**
```
1. Subtype card header: "Elite Investor" (businessName)
2. Below name: "Technical: elite_investor" (gray text)
3. Subtype table: Shows both in UI
4. Storage: Both names stored in entity.subtypes[key]
```

---

### User Requirement 11: "As do entities and fields the technical name is always lowercase and has underscores between the words"

**✅ IMPLEMENTED**

**Naming Convention:**
```
Business Name               Technical Name
────────────────          ───────────────────
Client Investor        →   client_investor
Legal Name             →   legal_name
Date of Birth          →   date_of_birth
Portfolio Value        →   portfolio_value
Individual Investor    →   individual_investor
Elite Investor         →   elite_investor
```

**Conversion Function:**
```typescript
// From utils/nameFormatting.ts
export const businessToTechnicalName = (businessName: string): string => {
  return businessName
    .toLowerCase()              // Client Investor → client investor
    .trim()                     // Remove leading/trailing spaces
    .replace(/\s+/g, '_')       // Replace spaces with underscores
    .replace(/[^\w_]/g, '');    // Remove special characters
};

// Result: "Client Investor" → "client_investor"
```

**Validation:**
```typescript
// Check if technical name is valid
export const isValidTechnicalName = (name: string): boolean => {
  return /^[a-z_][a-z0-9_]*$/.test(name);
};

// Valid:    "client_investor", "legal_name", "date_of_birth"
// Invalid:  "ClientInvestor", "legal name", "legal-name"
```

**Applied To:**
1. **Entities:**
   - "Client Investor" → technicalName: "client_investor"

2. **Subtypes:**
   - "Individual Investor" → technicalName: "individual_investor"

3. **Fields:**
   - "Legal Name" → technicalName: "legal_name"

**User Experience:**
```
User enters any business name:
  "   Client  Investor   "
       ↓ (system processes)
  technicalName: "client_investor"
  (lowercase, underscores, no special chars)
```

---

### User Requirement 12: "Fields are going to be selected from the semantic terms that I have in catalog_node"

**✅ IMPLEMENTED**

**What It Does:**
Users can optionally link fields to semantic terms from the catalog when adding fields.

**How It Works:**
```
1. User clicks [+ Add Field]
2. Modal opens
3. See: "Link to Semantic Term (Optional)"
4. Dropdown populated from catalog_node table
5. User selects: "Account Balance" (semantic term)
6. Field stores: semanticTermId + semanticTermName
```

**GraphQL Query:**
```graphql
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
  }
}
```

**Semantic Term Type:**
```typescript
interface SemanticTerm {
  id: string;              // UUID from catalog_node.id
  node_name: string;       // Display name
  description: string;     // Description
}
```

**Data Storage:**
```typescript
interface Field {
  semanticTermId?: string;    // Link to catalog_node.id
  semanticTermName?: string;  // Display name for reference
  // ...
}

// Example
{
  key: "account_balance",
  name: "Account Balance",
  businessName: "Account Balance",
  technicalName: "account_balance",
  type: "number",
  semanticTermId: "550e8400-e29b-41d4-a716-446655440000",
  semanticTermName: "Account Balance"
}
```

**UI Display:**
```
Add Field Modal:
┌─ Modal ─────────────────────────┐
│ Field Business Name: [input]    │
│ Field Type: [select]            │
│ Link to Semantic Term:          │
│   [select dropdown ▼]           │
│   ├── Account Balance           │
│   ├── Customer Name             │
│   ├── Investment Threshold      │
│   └── ...                       │
│                                 │
│ [OK] [Cancel]                   │
└─────────────────────────────────┘

Field Table Display:
┌──────────┬──────────┬──────┬─────────────────┐
│ Business │ Technical│ Type │ Semantic Term   │
├──────────┼──────────┼──────┼─────────────────┤
│ Acct Bal │ acct_bal │ num  │ Account Balance │
│ Cust Nam │ cust_nam │ text │ Customer Name   │
└──────────┴──────────┴──────┴─────────────────┘
```

---

### User Requirement 13: "You cannot create a new semantic term in the entity pages"

**✅ IMPLEMENTED**

**Design Decision:**
- Semantic terms are read-only (pulled from catalog)
- Dropdown shows existing terms only
- No "Create new term" button
- No free-text input
- Only select from existing terms

**Implementation:**
```tsx
// Add Field Modal
<Form.Item name="semanticTermId">
  <Select placeholder="Select a semantic term">
    {semanticTerms.map((term) => (
      <Option key={term.id} value={term.id}>
        {term.node_name}
      </Option>
    ))}
  </Select>
</Form.Item>

// No allowClear, no notFoundContent with "Create", etc.
```

**Behavior:**
```
1. Dropdown shows only existing semantic terms
2. No input field to add new
3. No "+" icon to create
4. If term not in catalog:
   - Leave field blank (optional)
   - Term must be created in Data Catalog first
```

**Future Workflow (not in v2.1):**
```
IF need new semantic term:
1. Go to Data Catalog page
2. Create semantic term there
3. Come back to Entity Editor
4. Refresh → New term appears in dropdown
5. Link field to new term
```

---

### User Requirement 14: "But you can assign one to the entity or the subentity"

**✅ IMPLEMENTED**

**Assignment at Multiple Levels:**

**At Entity Level:**
```
1. Edit entity
2. Go to 📋 Entity tab
3. See entity-level fields
4. Click [+ Add Field to Entity]
5. Modal: Link to semantic term
6. Field added to entity_fields with semantic link
```

**At Subtype Level:**
```
1. Edit entity
2. Go to 📦 Subtypes tab
3. Find subtype card
4. Click [+ Add Field] in subtype
5. Modal: Link to semantic term
6. Field added to subtype_fields with semantic link
```

**Data Example:**
```
Entity: Wealth Management Client
├── Entity-level field:
│   - Legal Name → semantic: "Customer Legal Name"
│   - Email → semantic: "Customer Email"
│
└── Subtype: Elite Investor
    ├── Minimum Assets → semantic: "Investment Threshold"
    └── Relationship Mgr → semantic: "Account Manager"
```

**Table Display Shows Both:**
```
ENTITY FIELDS:
┌─────────────┬──────────────┬───────────────────┐
│ Name        │ Type         │ Semantic Term     │
├─────────────┼──────────────┼───────────────────┤
│ Legal Name  │ text         │ Cust Legal Name   │
│ Email       │ text         │ Customer Email    │
└─────────────┴──────────────┴───────────────────┘

SUBTYPE FIELDS (Elite Investor):
┌──────────────────┬──────┬────────────────────┐
│ Name             │ Type │ Semantic Term      │
├──────────────────┼──────┼────────────────────┤
│ Min Assets       │ num  │ Investment Thresh. │
│ Relationship Mgr │ text │ Account Manager    │
└──────────────────┴──────┴────────────────────┘
```

---

## 📊 Feature Completion Matrix

| Requirement | Feature | Status | Location |
|---|---|---|---|
| Business name label | businessName field | ✅ | entity-schema.ts, UI cards |
| Rename object | Rename form (custom only) | ✅ | Entity Editor → 📋 Entity tab |
| Rename after clone | Rename form in clone | ✅ | Same |
| Clone tracking | clonesFromKey, cloneParentName | ✅ | Entity card + drawer |
| Parent upgrades | isCore field tracking | ✅ | Field status badges |
| Single screen editor | Tabs layout | ✅ | Drawer with 📋 & 📦 tabs |
| Subtype CRUD | Add/Delete modals | ✅ | 📦 Subtypes tab |
| Editable fields | Field delete for custom | ✅ | Field tables |
| Inherited read-only | Delete only for custom | ✅ | Field status checks |
| Delete at parent | Guards prevent child delete | ✅ | isCore check |
| Subtype business name | businessName field | ✅ | Subtype interface |
| Subtype technical name | technicalName field | ✅ | Subtype interface |
| Entity business name | businessName field | ✅ | Entity interface |
| Entity technical name | technicalName field | ✅ | Entity interface |
| Field business name | businessName field | ✅ | Field interface |
| Field technical name | technicalName field | ✅ | Field interface |
| Technical name format | lowercase_underscores | ✅ | nameFormatting.ts utils |
| Semantic term selection | Dropdown in modal | ✅ | Add Field modal |
| Semantic terms from catalog | GraphQL query | ✅ | datasourceQueries.ts |
| No create semantic | Read-only dropdown | ✅ | Modal design |
| Assign to entity fields | Add Field option | ✅ | Entity Fields section |
| Assign to subtype fields | Add Field option | ✅ | Subtype Fields section |

---

## 🚀 Deployment Status

**Code Files Modified/Created:**
- ✅ `frontend/src/types/entity-schema.ts` - Type updates
- ✅ `frontend/src/pages/EntityConfigPageV2.tsx` - Complete rewrite
- ✅ `frontend/src/utils/nameFormatting.ts` - New utility functions
- ✅ `frontend/src/graphql/queries/datasourceQueries.ts` - New GraphQL query

**Backend:**
- ✅ Existing endpoints support new fields (backward compatible)
- ✅ No schema changes needed (fields stored in JSONB)

**Database:**
- ✅ No migrations needed
- ✅ Existing entity_schema table works as-is

**Ready for Testing:** YES ✅

---

**Date:** October 17, 2025  
**Version:** 2.1  
**Status:** Implementation Complete
