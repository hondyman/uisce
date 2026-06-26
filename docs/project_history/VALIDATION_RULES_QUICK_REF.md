# Quick Reference: Validation Rules Tab

## 🎯 Quick Start

### Location
**Entity Details Page** → **⚡ Validations Tab**

### How to Access
1. Go to `/admin/entity-manager`
2. Double-click an entity card
3. Click the **⚡ Validations** tab

### What You Can Do
- Create validation rules specific to an entity
- Define field format validations
- Set cardinality constraints
- Configure uniqueness checks
- Establish referential integrity rules
- Create cross-entity conditions
- Set rule severity levels
- Manage rule dependencies

---

## 📁 Modified Files

| File | Changes | Status |
|------|---------|--------|
| `frontend/src/pages/EntityDetailsPage.tsx` | Added ValidationRulesContainer, import AdvancedRuleConfiguration | ✅ Complete |
| `frontend/src/pages/EntityDetailsPage.module.css` | Added styling for validation rules section | ✅ Complete |

---

## 🎨 Styling Classes

```css
.validationRulesContainer      /* Main container */
.validationRulesHeader         /* Title + description area */
.validationRulesTitle          /* "Validation Rules for [Entity]" */
.validationRulesDescription    /* Subtitle text */
.validationRulesCard           /* Card wrapper */
```

---

## 🏗️ Component Structure

```
EntityDetailsPage
  ├── Tabs
  │   ├── 📋 Entity Tab
  │   ├── 🔗 Related Objects Tab
  │   └── ⚡ Validations Tab
  │       └── ValidationRulesContainer
  │           ├── Header (Title + Description)
  │           └── Card
  │               └── AdvancedRuleConfiguration
```

---

## 🔧 State Management

```typescript
// In EntityDetailsPage
const [validationRules, setValidationRules] = useState<ValidationRule[]>([]);

// Passed to ValidationRulesContainer
<ValidationRulesContainer
  rules={validationRules}
  onRulesUpdate={setValidationRules}
  onCrossEntitySave={(condition) => {
    console.log("Cross-entity condition saved:", condition);
    // TODO: Persist to backend
  }}
  entity={entity}
/>
```

---

## 📊 Rule Types Supported

| Type | Icon | Description | Use Case |
|------|------|-------------|----------|
| Field Format | 📝 | Regex patterns | Email validation, phone format |
| Cardinality | 📊 | Count/threshold | Min/max values |
| Uniqueness | 🔑 | Unique constraints | Unique IDs, usernames |
| Referential Integrity | 🔗 | Foreign keys | Cross-entity references |
| Business Logic | ⚙️ | Custom rules | Domain-specific rules |

---

## 🚀 Severity Levels

| Level | Behavior | Color |
|-------|----------|-------|
| ❌ Error | Blocks operations | Red (#ff4d4f) |
| ⚠️ Warning | Alerts but allows | Orange (#faad14) |
| ℹ️ Info | Informational only | Blue (#1890ff) |

---

## 💡 Example Use Cases

### 1. Email Validation
```
Rule: Email Format
Entity: User
Type: Field Format
Condition: field matches /^[^\s@]+@[^\s@]+\.[^\s@]+$/
Severity: Error
```

### 2. Salary Range
```
Rule: Salary Must Be Positive
Entity: Employee
Type: Business Logic
Condition: salary > 0
Severity: Error
```

### 3. Department Exists
```
Rule: Valid Department Reference
Entity: Employee
Type: Referential Integrity
Condition: Employee.department_id exists in Department.id
Severity: Error
```

### 4. Cross-Entity Condition
```
Rule: Manager is Senior
Entity: Employee
Type: Business Logic
Condition: 
  IF Employee.role = "Manager"
  THEN Employee.experience_years >= 5
Severity: Warning
```

---

## 🔄 Data Flow

```
User Edits Entity
    ↓
EntityDetailsPage loads
    ↓
User clicks "⚡ Validations" tab
    ↓
ValidationRulesContainer renders
    ↓
AdvancedRuleConfiguration displays rules
    ↓
User creates/edits/deletes rules
    ↓
onRulesUpdate callback triggered
    ↓
validationRules state updated
    ↓
Rules persist to backend (TODO)
```

---

## 🎨 Styling Details

### Container Styling
```css
padding: 24px 0;                    /* Top/bottom padding */
```

### Header Area
```css
margin-bottom: 24px;                /* Space from content */
```

### Title
```css
margin-bottom: 8px;                 /* Space from description */
font-size: h5 (Ant Design Level 5)
font-weight: 600;
```

### Description
```css
color: rgba(0, 0, 0, 0.45);        /* Secondary text color */
font-size: 14px;                    /* Body text size */
```

### Card
```css
border: 1px solid #f0f0f0;         /* Light gray border */
background: #ffffff;               /* White background */
padding: 24px;                      /* Ant Card padding */
```

---

## ✅ Checklist for Users

- [ ] Navigate to Entity Manager
- [ ] Find and edit the entity
- [ ] Click the "⚡ Validations" tab
- [ ] See the styled validation rules interface
- [ ] Create a new rule
- [ ] Configure rule conditions
- [ ] Set severity level
- [ ] Review cross-entity conditions
- [ ] Test the rule (future feature)

---

## 🐛 Troubleshooting

### Issue: Tab doesn't show validation rules
**Solution:** Make sure you're in the entity detail page (after editing an entity)

### Issue: Styling looks off
**Solution:** Clear browser cache (Ctrl+Shift+Delete or Cmd+Shift+Delete)

### Issue: Rules not persisting
**Solution:** Backend integration is TODO - currently rules exist only in the session

### Issue: Cross-entity references not working
**Solution:** Ensure related entities are defined in Entity Manager

---

## 📝 Notes

- ✅ Validation rules are entity-scoped
- ✅ Tenant and datasource isolation maintained
- ✅ Supports complex cross-entity conditions
- ❌ Backend persistence not yet implemented
- ❌ Real-time rule execution not yet available
- 🔜 Rule testing feature coming soon
- 🔜 Rule templates library planned

---

## 📚 Related Documentation

- [VALIDATION_RULES_INTEGRATION.md](./VALIDATION_RULES_INTEGRATION.md) - Full implementation details
- [VALIDATION_RULES_UI_GUIDE.md](./VALIDATION_RULES_UI_GUIDE.md) - UI/UX visual guide
- [AdvancedRuleConfiguration Component](./frontend/src/components/validation/AdvancedRuleConfiguration.tsx)
- [EntityDetailsPage](./frontend/src/pages/EntityDetailsPage.tsx)

---

## 🎓 Learning Resources

### Workday Pattern
Validation rules are managed **contextually** within the business object editor, not on a separate page. This mirrors Workday's "Configure Custom Object Validations" approach.

### Key Benefits
1. **Contextual** - Rules visible alongside fields and relationships
2. **Efficient** - No context switching needed
3. **Safe** - Tenant-scoped by default
4. **Scalable** - Supports complex conditions

---

## 🤝 Support

For issues or questions:
1. Check this quick reference
2. Review the UI guide
3. Check implementation docs
4. Examine the component code
5. Create an issue in the repository

---

**Last Updated:** October 25, 2025
**Status:** ✅ Ready for Use
**Version:** 1.0
