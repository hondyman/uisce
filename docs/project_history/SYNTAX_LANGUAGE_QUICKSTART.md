# Quick Start: Adding Code Properties with Syntax Highlighting

## 5-Minute Setup

### Step 1: Navigate to Catalog Setup
Go to `http://localhost:5173/core/catalog-setup`

### Step 2: Create a Node Type
1. Click **Create Node Type**
2. Fill in:
   - **Type Name**: `condition_rule`
   - **Description**: "Rule with SQL conditions"
   - Click **Next** (or scroll to properties)

### Step 3: Add a Code Property
1. In **Properties** section, click **+ Add property**
2. Fill in:
   - **Name**: `sql_condition`
   - **Label**: "SQL Condition"
   - **Data type**: `string`
   - **Input**: `code editor` ⭐
   - **Syntax Highlighting Language**: `SQL` ⭐

### Step 4: Save
Click **Create Node Type**

### Step 5: Use It
Now when creating/editing nodes of type `condition_rule`, the `sql_condition` property will display with SQL syntax highlighting!

---

## Example Use Cases

### SQL Conditions
```
Property Name: size_filter
Language: SQL
Content:
  {CUBE}.size IN ('xl', 'xxl', 'lg')
```

### YAML Configuration
```
Property Name: rule_config
Language: YAML
Content:
  when:
    - condition: active
      value: true
  else:
    value: false
```

### JSON Schema
```
Property Name: mapping_schema
Language: JSON
Content:
  {
    "source": "dimension_id",
    "target": "fact_id",
    "type": "one-to-many"
  }
```

---

## Features You Get

✅ **Syntax Highlighting** - Color-coded syntax based on language
✅ **Line Numbers** - Easy reference  
✅ **Code Folding** - Collapse/expand sections
✅ **Bracket Matching** - See paired brackets highlighted
✅ **Error Detection** - Visual indicators for syntax errors
✅ **Word Wrap** - Long lines wrap automatically

---

## Supported Languages

| Language | Use For | Example |
|----------|---------|---------|
| **SQL** | Queries, filters, expressions | `SELECT * WHERE id = 'x'` |
| **YAML** | Config, conditions, rules | `when:\n  - condition: true` |
| **JSON** | Structured data, mappings | `{"key": "value"}` |
| **None** | Plain text | Any unformatted text |

---

## Tips

1. **Plain Text Mode**: Select "None (plain text)" for unformatted content
2. **Multiline**: All languages support multiline input
3. **Validation**: Monaco will show syntax errors as you type
4. **Saving**: Content is saved as-is (no auto-formatting)
5. **Editing**: Existing values open in the editor when you edit a node

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| No editor appears | Make sure input type is "code editor" |
| No highlighting | Check that language is set (not "None") |
| Errors showing | This is normal! Monaco highlights syntax errors |
| Can't save | Ensure form validation passes (name, label required) |

---

## See Also

- Full documentation: `SYNTAX_LANGUAGE_SUPPORT.md`
- Implementation details: `SYNTAX_LANGUAGE_IMPLEMENTATION.md`
- Catalog Setup: `/core/catalog-setup`
