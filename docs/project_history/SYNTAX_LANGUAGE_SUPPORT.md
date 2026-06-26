# Syntax Language Support for Node and Edge Properties

This document explains how to configure and use the syntax highlighting feature for string properties in catalog node and edge types.

## Overview

You can now add string properties to your catalog nodes and edges that display with syntax highlighting for specific languages (SQL, YAML, or JSON). This is useful for configuration values, SQL expressions, or structured data that benefits from syntax highlighting.

## Configuration

### 1. Creating a Property with Code Editor

When creating or editing a node type or edge type in the Catalog Setup page (`/core/catalog-setup`):

1. Navigate to **Node Types** or **Edge Types** tab
2. Click **Create Node Type** or **Create Edge Type** (or edit an existing one)
3. In the **Properties** section, click **+ Add property**
4. Fill in the property details:
   - **Name**: Machine-friendly name (e.g., `size_condition`, `sql_filter`)
   - **Label**: Human-readable label (e.g., "Size Condition", "SQL Filter")
   - **Data type**: Select `string`
   - **Input**: Select `code editor`
   - **Syntax Highlighting Language**: Choose from:
     - **None (plain text)** - for unformatted content
     - **SQL** - for SQL expressions and queries
     - **YAML** - for YAML configuration
     - **JSON** - for JSON structures

### 2. Example Configuration

Here's an example of configuring a property for SQL conditions (matching your use case):

```yaml
case:
  when:
    - sql: "{CUBE}.size_value = 'xl-en'"
      label: xl
    - sql: "{CUBE}.size_value = 'xl'"
      label: xl
    - sql: "{CUBE}.size_value = 'xxl-en'"
      label: xxl
    - sql: "{CUBE}.size_value = 'xxl'"
      label: xxl
  else:
    label: Unknown
```

**Property Setup:**
- **Name**: `size_filter_sql`
- **Label**: "Size Filter SQL"
- **Data type**: `string`
- **Input**: `code editor`
- **Syntax Highlighting Language**: `SQL`

## Using Properties

### In Node/Edge Type Forms

When editing properties on a node or edge instance:

1. If the property has a **code editor** input type with a syntax language:
   - A Monaco Editor will appear with syntax highlighting for the selected language
   - The editor supports:
     - Syntax highlighting and error squiggles
     - Code folding and line numbers
     - Automatic layout and word wrap
     - Bracket pair colorization

2. If no syntax language is selected:
   - A plain text editor appears without syntax highlighting

### Example: SQL Property

For a SQL property, users can input complex SQL expressions with full syntax highlighting:

```sql
SELECT * FROM {CUBE} WHERE {CUBE}.size_value IN ('xl-en', 'xl', 'xxl-en', 'xxl')
```

### Example: YAML Property

For a YAML property:

```yaml
when:
  - condition: active
    value: true
  - condition: pending
    value: false
else:
  value: unknown
```

### Example: JSON Property

For a JSON property:

```json
{
  "conditions": [
    {
      "sql": "{CUBE}.size_value = 'xl'",
      "label": "extra large"
    }
  ],
  "default": "unknown"
}
```

## Technical Details

### Property Configuration Structure

Properties with code editor support are stored in the database with the following structure:

```typescript
interface NodeProperty {
  name: string;
  label: string;
  data_type: 'string' | 'integer' | 'boolean' | 'date' | 'float' | 'json' | 'text' | 'array';
  input_type: 'code-editor' | /* other types */;
  syntax_language?: 'sql' | 'yaml' | 'json' | null;
  nullable: boolean;
  options?: string[];
  order: number;
  // ... other fields
}
```

### Frontend Components

**SyntaxPropertyEditor** (`frontend/src/components/properties/SyntaxPropertyEditor.tsx`)
- Displays code content with optional syntax highlighting
- Uses Monaco Editor for language-specific syntax support
- Falls back to plain text if no language is specified
- Lazy-loads Monaco to reduce bundle size

**PropertyEditor** (`frontend/src/components/properties/PropertyEditor.tsx`)
- Handles all property input types including code-editor
- Integrates SyntaxPropertyEditor for code-editor input types
- Maintains compatibility with existing property types

**PropertySchemaEditor** (`frontend/src/components/properties/PropertySchemaEditor.tsx`)
- Configuration UI for defining properties
- Now includes language selection for code-editor input type
- Supports SQL, YAML, and JSON language options

### Supported Languages

1. **SQL** - Full SQL syntax highlighting
   - Use for: SQL queries, conditions, expressions
   - Best for: Data filtering, calculated columns, conditions

2. **YAML** - YAML structure highlighting
   - Use for: Configuration files, nested structures
   - Best for: Conditions, when/then rules

3. **JSON** - JSON structure highlighting
   - Use for: Structured data, configurations
   - Best for: Complex nested data, API responses

## API Endpoints

When saving node/edge types with code-editor properties, the backend receives:

```http
POST /api/node-types
Content-Type: application/json

{
  "tenant_id": "uuid",
  "catalog_type_name": "my_type",
  "description": "...",
  "properties": [
    {
      "name": "size_filter_sql",
      "label": "Size Filter SQL",
      "data_type": "string",
      "input_type": "code-editor",
      "syntax_language": "sql",
      "nullable": false,
      "order": 0
    }
  ]
}
```

## Browser Support

The syntax highlighting feature uses Monaco Editor, which requires:
- Modern browser with ES6+ support
- All modern versions of Chrome, Firefox, Safari, Edge

## Performance Considerations

1. **Lazy Loading**: Monaco Editor is lazy-loaded to minimize initial bundle size
2. **Per-Instance Loading**: Editors are only loaded when a code-editor property is displayed
3. **Memory**: Each Monaco Editor instance uses memory; avoid having many editors on the same page

## Troubleshooting

### Editor Not Showing

1. Verify the input type is set to `code-editor`
2. Check browser console for errors
3. Ensure Monaco Editor dependencies are installed

### Syntax Highlighting Not Working

1. Verify `syntax_language` is set to a supported value (`sql`, `yaml`, or `json`)
2. Check that the content is valid for the selected language
3. Monaco may show error squiggles for invalid syntax (this is expected)

### Content Not Saving

1. Verify the form validation passes
2. Check browser network tab for API errors
3. Ensure `syntax_language` is properly serialized in the request

## Example Workflow

1. **Create Property Definition**
   ```
   Navigate → Catalog Setup → Node Types → Create Node Type
   Add Property:
     - Name: query_filter
     - Label: Query Filter
     - Data Type: string
     - Input: code editor
     - Language: SQL
   Save
   ```

2. **Use the Property**
   ```
   Users can now enter SQL queries with full syntax highlighting
   when creating/editing nodes of this type
   ```

3. **View Configuration**
   ```
   Click Properties button on any node type to see configured properties
   with preview of the input types
   ```

## Future Enhancements

Potential improvements for this feature:

1. **Additional Languages**: Python, JavaScript, HTML, CSS, etc.
2. **Schema Validation**: JSON Schema validation for JSON properties
3. **Formatting**: Auto-format buttons for each language
4. **Themes**: Dark/light theme support for editors
5. **Snippets**: Language-specific code snippets and templates
6. **Import/Export**: Export properties to external files
