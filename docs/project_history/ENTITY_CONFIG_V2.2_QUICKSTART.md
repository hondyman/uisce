# Entity Config v2.2: 5-Minute Quickstart

**Target Audience:** Fabric Builder users, automation agents  
**Time:** 5 minutes  
**Prerequisites:** Tenant selected in UI

---

## 🚀 The New Model (What Changed?)

### Before (v2.1)
- Fields could be created manually
- Semantic terms were optional add-ons
- No field reordering
- Manual naming = errors

### After (v2.2)
- **Fields MUST come from semantic catalog** ✅
- **Names auto-populated from catalog** ✅
- **Full reordering support** ✅
- **Inherited fields protected** ✅

---

## 📖 Tutorial: Add a Custom Field (3 Steps)

### Step 1: Select Entity/Subtype
```
1. Open Entity Config Builder
2. Click sidebar tree → Find your entity
3. Click subtype (e.g., "Individual Investor")
4. Main panel shows:
   - 🔒 Inherited Fields (blue, read-only)
   - ✏️ Assigned Fields (green, editable) + [+Add] button
```

### Step 2: Add Field from Catalog
```
1. Click [+Add] button in "Assigned Fields"
2. Modal opens: "Add Field - Select Semantic Term"
3. Type in search box: "tax" (or any term name)
4. See results: Tax ID, Tax Status, Tax Year, etc.
5. Click [Add] next to "Tax ID"
6. ✅ Field appears in table instantly
```

### Step 3: Save & Deploy
```
1. Make more changes if needed
2. Click [SAVE & APPLY] button (bottom right)
3. See: "✅ Saved! 1 changed, 0 deleted"
4. Changes now persisted to backend
```

---

## 🎯 Common Tasks

### Reorder Fields
```
Assigned Fields table shows:
  Tax ID         [↑][↓][🗑]
  Birth Date     [↑][↓][🗑]
  Status         [↑][↓][🗑]

→ Click ↓ on "Tax ID"
→ Order becomes: Birth Date, Tax ID, Status
→ Sequences auto-update (0→1, 1→0)
→ Save & Apply to persist
```

### Delete Field
```
Assigned Fields table shows:
  Tax ID         [↑][↓][🗑]
  
→ Click 🗑 icon
→ Confirm dialog: "Delete field?"
→ Click OK
→ Field removed from table
→ Save & Apply to persist
```

### View Inherited Fields
```
🔒 Inherited Fields (2)
┌─────────────────────┐
│ Investor ID (core)  │  ← Blue, locked
│ Legal Name (core)   │  ← Cannot edit
└─────────────────────┘

→ These come from parent entity
→ Read-only, protected from changes
```

### Search Semantic Terms
```
[Add Field] → Modal opens

[Search semantic terms...     ] ← Type here
              ↓
Filter results:
  ✓ Tax ID
  ✓ Tax Rate
  ✓ Tax Category
  
→ See only terms matching "tax"
```

---

## 🔍 Understanding the UI

### Color Coding

| Color | Meaning | Interaction |
|-------|---------|-------------|
| 🔵 Blue Badge | Core/inherited | Read-only, locked |
| 🟢 Green Badge | Custom/assigned | Fully editable |
| 🔒 Locked | Inherited field | Cannot delete/reorder |
| ✏️ Edit | Assigned field | Click up/down to reorder |

### Button Reference

| Button | Location | Action | Example |
|--------|----------|--------|---------|
| [+Add] | Assigned Fields header | Open semantic term modal | Click → Select "Tax ID" |
| [↑] | Per field row | Move field up | Click → Reorder |
| [↓] | Per field row | Move field down | Click → Reorder |
| [🗑] | Per field row | Delete field | Click → Confirm → Delete |
| [SAVE & APPLY] | Bottom right (affix) | Save all changes to backend | Click → Upload → Success |

---

## 🧠 Key Concepts

### Semantic Terms = Catalog of Available Fields

Think of semantic terms as a **shared library**:
- IT admin defines: "Tax ID" field (data type: text, technical name: tax_id)
- You select: "Tax ID" when adding a field to your entity
- Result: Field auto-populated with all the values

### businessName vs technicalName

```
User sees (UI):        Backend/Database stores:
─────────────────     ──────────────────────
"Tax ID"        →     "tax_id"  (technicalName)
"Birth Date"    →     "birth_date"
"Legal Entity"  →     "legal_entity_name"

→ Readable names for humans
→ Snake_case names for systems (APIs, databases)
```

### Sequence Field = Display Order

```
sequence: 0  → Shows first
sequence: 1  → Shows second
sequence: 2  → Shows third

When you reorder:
  Old order: sequence 0,1,2
  New order: sequence 1,0,2  (after reordering)
  
→ Display follows sequence order
```

### Inherited vs Assigned Fields

```
INHERITED (from parent entity):
└─ Investor ID (parent's field)
└─ Legal Name (parent's field)
→ Read-only, protected from changes
→ Shows as blue, locked

ASSIGNED (added to this subtype):
└─ Tax ID (you added this)
└─ Birth Date (you added this)
→ Fully editable: reorder, delete, edit
→ Shows as green, unlocked
```

---

## ⚡ Pro Tips

### Tip 1: Search Before Adding
```
Don't add random fields! Search the catalog first:
→ 70% chance the field already exists
→ Reuse existing fields for consistency
→ Reduces duplication
```

### Tip 2: Clone Then Customize
```
Workflow:
1. Clone a core BO ("Client Investor" → "My Investor")
2. Go to subtypes (e.g., "Individual")
3. Add custom fields specific to your needs
4. Keep inherited fields intact (protected)
```

### Tip 3: Save Frequently
```
Changes tracked in real-time:
→ [SAVE & APPLY] button shows count
→ Click save often, don't lose work
→ Auto-save coming in v2.3
```

### Tip 4: Use Semantic Term Names
```
When searching, use semantic term names (not business names):
✅ Search: "Tax" → Finds "Tax ID", "Tax Rate", "Tax Status"
❌ Search: "custom" → Finds nothing (no semantic term named that)
```

---

## 🐛 Troubleshooting

### Q: "Select a tenant" error?
**A:** Use tenant picker at top → Select tenant → Select product → Select datasource → Page refreshes

### Q: Add Field button disabled?
**A:** Check: (1) Tenant selected?, (2) Semantic terms loading?, (3) Try refresh

### Q: Changes not saving?
**A:** (1) Check success toast after clicking SAVE & APPLY, (2) Verify tenant scope selected, (3) Check browser console for errors

### Q: Semantic term not in search?
**A:** (1) Check spelling, (2) Try partial search ("tax" matches "Tax ID"), (3) Check if catalog is loaded (refresh page)

### Q: Can't delete inherited field?
**A:** That's intentional! Inherited fields are locked (blue). Only delete assigned fields (green).

### Q: Where are my changes?
**A:** Changes stored in:
- Browser state (until Save)
- Backend database (after Save)
- Check SAVE & APPLY button count before closing

---

## 📋 Workflow Examples

### Example 1: Add Tax ID to Individual Investor

```
CURRENT STATE:
Entity: Client Investor
  └─ Subtype: Individual Investor
     ├─ Inherited: Investor ID, Legal Name
     └─ Assigned: (empty)

STEPS:
1. Click "Individual Investor" in sidebar
2. See table: Inherited Fields (2), Assigned Fields (0)
3. Click [+Add]
4. Search: "tax"
5. Select: "Tax ID" [Add]
6. See: Tax ID row added to Assigned Fields
7. Click [SAVE & APPLY]
8. Toast: "✅ Saved! 1 changed, 0 deleted"

RESULT:
Entity: Client Investor
  └─ Subtype: Individual Investor
     ├─ Inherited: Investor ID, Legal Name
     └─ Assigned: Tax ID
```

### Example 2: Reorder Fields

```
CURRENT ORDER (sequence):
0. Tax ID
1. Birth Date
2. Status

STEPS:
1. Click ↓ on "Tax ID" (move down)
2. See reordering: Birth Date (0), Tax ID (1), Status (2)
3. Click ↓ on "Tax ID" again
4. See reordering: Birth Date (0), Status (1), Tax ID (2)
5. Click [SAVE & APPLY]

RESULT:
0. Birth Date
1. Status
2. Tax ID
```

### Example 3: Delete Assigned Field

```
CURRENT STATE:
Assigned Fields:
  Tax ID    [↑][↓][🗑]
  Status    [↑][↓][🗑]

STEPS:
1. Click 🗑 on "Tax ID"
2. Dialog: "Delete field?"
3. Click OK
4. Tax ID removed from table
5. Status now shows [↑ disabled][↓ disabled]
6. Click [SAVE & APPLY]

RESULT:
Assigned Fields:
  Status    [↑][↓][🗑]
```

---

## 🎓 Learning Path

**Level 1: Beginner** (5 min)
- [ ] Select entity in sidebar
- [ ] View inherited + assigned fields
- [ ] Add one field from semantic terms
- [ ] Save & see success toast

**Level 2: Intermediate** (15 min)
- [ ] Clone a core BO
- [ ] Add 3+ custom fields
- [ ] Reorder fields using arrows
- [ ] Delete a field and save

**Level 3: Advanced** (30 min)
- [ ] Create custom subtype
- [ ] Add fields to subtype (overriding parent)
- [ ] Build complex schema with multiple subtypes
- [ ] Verify tenant scope isolation

**Level 4: Expert** (60+ min)
- [ ] Understand GraphQL queries behind the scenes
- [ ] Review semantic term properties/metadata
- [ ] Write custom validation rules
- [ ] Contribute to v2.3 features

---

## 📚 Next Steps

### Want More Details?
- Full Architecture: [ENTITY_CONFIG_V2.2_FEATURES.md](./ENTITY_CONFIG_V2.2_FEATURES.md)
- API Specs: [API_LAYER_README.md](../API_LAYER_README.md)
- Tenant Scope: [agents.md](../agents.md)

### Want to Contribute?
- Review [DEVELOPER_NOTES_API.md](../DEVELOPER_NOTES_API.md)
- Check [ENTITY_CONFIG_V2.1_QUICKREF.md](./ENTITY_CONFIG_V2.1_QUICKREF.md) for existing workflows
- Open an issue for bugs or feature requests

### Want to Learn GraphQL?
- See useEnhancedSemanticTerms hook: `frontend/src/hooks/useEnhancedSemanticTerms.ts`
- Query: `GET_SEMANTIC_TERMS_WITH_METADATA`
- Returns: All catalog terms with properties

---

**Version:** v2.2  
**Last Updated:** January 15, 2025  
**Maintained By:** GitHub Copilot
