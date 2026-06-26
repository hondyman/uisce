export type NodePropertyConfig = {
  name: string;
  label?: string;
  order?: number;
  nullable?: boolean;
  data_type?: string;
  input_type?: string;
  enumValues?: string[];
};

// Format a single property value according to its NodeProperty config
export function formatPropertyValue(prop: NodePropertyConfig, value: any): any {
  // handle null/empty
  if (value === undefined || value === null || value === '') {
    return prop.nullable ? null : value;
  }

  const dataType = (prop.data_type || '').toLowerCase();
  const inputType = (prop.input_type || '').toLowerCase();

  if (inputType === 'checkbox' || dataType === 'boolean') {
    // Ensure boolean
    return !!value;
  }

  const options = (prop as any).options || (prop as any).enumValues;
  if (options && Array.isArray(options)) {
    // If the property supports options, store selection as the raw string
    return options.includes(value) ? value : value;
  }

  if (dataType === 'number' || dataType === 'decimal') {
    const parsed = Number(value);
    return isNaN(parsed) ? value : parsed;
  }

  if (dataType === 'date' || dataType === 'datetime') {
    // Try to convert to ISO date string
    const d = new Date(value);
    if (!isNaN(d.getTime())) return d.toISOString();
    return value;
  }

  // Default: return as-is (string)
  return value;
}

// Format a set of properties using the list of property configs
export function formatProperties(propertyConfigs: NodePropertyConfig[] | undefined, values: Record<string, any>): Record<string, any> {
  if (!propertyConfigs || !Array.isArray(propertyConfigs)) {
    // No schema; return values as-is
    return values || {};
  }

  const out: Record<string, any> = {};

  for (const cfg of propertyConfigs) {
    const key = cfg.name;
    if (key in values) {
      out[key] = formatPropertyValue(cfg, values[key]);
    } else if (cfg.nullable) {
      out[key] = null;
    }
  }

  // Keep any additional properties that aren't in the config
  for (const k of Object.keys(values || {})) {
    if (!(k in out)) out[k] = values[k];
  }

  return out;
}

// Validate a single property using NodePropertyConfig rules; returns an
// error message string if invalid or null if valid.
export function validateProperty(prop: NodePropertyConfig, value: any, allProperties?: Record<string, any>): string | null {
  // Skip validation for cascaded fields when parent is not selected
  // (they are disabled and shouldn't block form submission)
  const cascadeFrom = (prop as any).cascade_from;
  if (cascadeFrom && allProperties) {
    const parentValue = allProperties[cascadeFrom];
    if (!parentValue) {
      // Parent not selected, field is disabled - no validation needed
      return null;
    }
  }

  // required/nullable
  if (!prop.nullable && (value === undefined || value === null || value === '')) {
    return `${prop.label || prop.name} is required`;
  }

  const dataType = (prop.data_type || '').toLowerCase();
  const inputType = (prop.input_type || '').toLowerCase();

  if ((dataType === 'integer' || dataType === 'float' || inputType === 'number') && value !== undefined && value !== '') {
    const num = Number(value);
    if (Number.isNaN(num)) return `${prop.label || prop.name} must be a number`;
    if (prop.validation?.min !== undefined && num < prop.validation?.min) return `${prop.label || prop.name} must be >= ${prop.validation.min}`;
    if (prop.validation?.max !== undefined && num > prop.validation?.max) return `${prop.label || prop.name} must be <= ${prop.validation.max}`;
  }

  if ((inputType === 'text' || dataType === 'string') && typeof value === 'string') {
    if (prop.validation?.minLength !== undefined && value.length < prop.validation.minLength) return `${prop.label || prop.name} must be at least ${prop.validation.minLength} characters`;
    if (prop.validation?.maxLength !== undefined && value.length > prop.validation.maxLength) return `${prop.label || prop.name} must be at most ${prop.validation.maxLength} characters`;
    if (prop.validation?.pattern) {
      try { const re = new RegExp(prop.validation.pattern); if (!re.test(value)) return `${prop.label || prop.name} must match pattern`; } catch(e) { }
    }
  }

  if (inputType === 'json-editor' || dataType === 'json') {
    if (typeof value === 'string' && value.trim() !== '') {
      try { JSON.parse(value); } catch(e) { return `${prop.label || prop.name} is not valid JSON`; }
    }
  }

  if (prop.validation?.multiple && Array.isArray(value)) {
    if (prop.validation?.minLength !== undefined && value.length < prop.validation.minLength) return `${prop.label || prop.name} must have at least ${prop.validation.minLength} items`;
    if (prop.validation?.maxLength !== undefined && value.length > prop.validation.maxLength) return `${prop.label || prop.name} must have at most ${prop.validation.maxLength} items`;
  }

  return null;
}

// Build a JSON schema object for a property; used for Monaco diagnostics when
// editing JSON properties. This attempts to map NodePropertyConfig validation
// metadata into a simple JSON Schema that will highlight basic issues.
export function getJsonSchemaForProperty(prop: NodePropertyConfig): object | null {
  // If the property provides an explicit json schema in validation use that.
  if (prop.validation && (prop.validation as any).jsonSchema) {
    return (prop.validation as any).jsonSchema;
  }

  // Only build a helpful schema for JSON editors; otherwise return null.
  const isJsonEditor = (prop.input_type || '').toLowerCase() === 'json-editor' || (prop.data_type || '').toLowerCase() === 'json';
  if (!isJsonEditor) return null;

  const schema: any = { $schema: 'http://json-schema.org/draft-07/schema#' };

  // If the property has format 'object' or 'array', prefer that type, otherwise allow both
  if (prop.format === 'array' || (prop.data_type === 'array')) {
    schema.type = 'array';
    schema.items = {};
  } else if (prop.format === 'object') {
    schema.type = 'object';
    schema.properties = {};
  } else {
    schema.type = ['object', 'array'];
  }

  // Apply basic constraints that map well to JSON Schema
  if (prop.validation) {
    if (typeof prop.validation?.min === 'number') schema.minimum = prop.validation.min;
    if (typeof prop.validation?.max === 'number') schema.maximum = prop.validation.max;
    if (typeof prop.validation?.minLength === 'number') schema.minLength = prop.validation.minLength;
    if (typeof prop.validation?.maxLength === 'number') schema.maxLength = prop.validation.maxLength;
    if (typeof prop.validation?.pattern === 'string') schema.pattern = prop.validation.pattern;
  }

  // If the property supports enum values, provide them in the schema so Monaco can suggest and warn
  if (Array.isArray((prop as any).enumValues) && (prop as any).enumValues.length > 0) {
    schema.enum = (prop as any).enumValues;
  }

  // For array typed properties attempt to provide item-level schema information if available
  if (schema.type === 'array') {
    // If metadata supplies an 'itemsType' or 'items' hint, use it, otherwise allow any
    const itemsType = (prop as any).itemsType || (prop as any).items?.type;
    if (itemsType) {
      if (itemsType === 'string') schema.items = { type: 'string' };
      else if (itemsType === 'integer' || itemsType === 'number') schema.items = { type: 'number' };
      else if (itemsType === 'object') schema.items = { type: 'object' };
      else schema.items = {};
    } else {
      schema.items = {};
    }
  }

  return schema;
}
