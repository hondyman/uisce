import React, { 
  useState, 
  useCallback, 
  useMemo, 
  useRef, 
  useEffect
} from 'react';
import {
  Plus,
  Trash2,
  ChevronDown,
  ChevronRight,
  GripVertical,
  Copy,
  AlertCircle,
  Check,
  Search,
  Link2,
  Layers,
  Undo,
  Redo
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

export interface FieldDefinition {
  name: string;
  type: 'string' | 'number' | 'date' | 'boolean' | 'enum' | 'array' | 'object';
  label: string;
  description?: string;
  enumValues?: string[];
  nullable?: boolean;
  entity?: string; // For cross-entity fields
  path?: string;   // Full path for nested fields
}

export interface EntityDefinition {
  name: string;
  label: string;
  fields: FieldDefinition[];
  relationships: Array<{
    name: string;
    targetEntity: string;
    type: 'one-to-one' | 'one-to-many' | 'many-to-one' | 'many-to-many';
    label: string;
  }>;
}

export interface Condition {
  id: string;
  type: 'condition';
  field: string;
  fieldPath?: string; // For cross-entity: "order.customer.name"
  operator: string;
  value: string | number | boolean | string[];
  valueType?: string;
  secondValue?: string | number; // For 'between' operator
  isValid?: boolean;
  validationError?: string;
}

export interface ConditionGroup {
  id: string;
  type: 'group';
  operator: 'AND' | 'OR' | 'NOT';
  conditions: ConditionNode[];
  isCollapsed?: boolean;
  label?: string; // Optional label for complex groups
}

export type ConditionNode = Condition | ConditionGroup;

export interface OperatorDefinition {
  value: string;
  label: string;
  description?: string;
  requiresValue: boolean;
  requiresSecondValue?: boolean;
  valueType?: 'single' | 'multiple' | 'range' | 'none';
  icon?: React.ReactNode;
}

// ============================================================================
// Operator Definitions by Type
// ============================================================================

const OPERATORS_BY_TYPE: Record<string, OperatorDefinition[]> = {
  string: [
    { value: 'equals', label: 'Equals', description: 'Exact match', requiresValue: true, valueType: 'single' },
    { value: 'not_equals', label: 'Not Equals', description: 'Does not match', requiresValue: true, valueType: 'single' },
    { value: 'contains', label: 'Contains', description: 'Contains substring', requiresValue: true, valueType: 'single' },
    { value: 'not_contains', label: 'Does Not Contain', description: 'Does not contain substring', requiresValue: true, valueType: 'single' },
    { value: 'starts_with', label: 'Starts With', description: 'Begins with', requiresValue: true, valueType: 'single' },
    { value: 'ends_with', label: 'Ends With', description: 'Ends with', requiresValue: true, valueType: 'single' },
    { value: 'matches_regex', label: 'Matches Pattern', description: 'Regex match', requiresValue: true, valueType: 'single' },
    { value: 'in', label: 'In List', description: 'One of values', requiresValue: true, valueType: 'multiple' },
    { value: 'not_in', label: 'Not In List', description: 'Not one of values', requiresValue: true, valueType: 'multiple' },
    { value: 'is_empty', label: 'Is Empty', description: 'Null or empty string', requiresValue: false, valueType: 'none' },
    { value: 'is_not_empty', label: 'Is Not Empty', description: 'Has value', requiresValue: false, valueType: 'none' },
    { value: 'length_equals', label: 'Length Equals', description: 'String length equals', requiresValue: true, valueType: 'single' },
    { value: 'length_greater', label: 'Length Greater Than', description: 'String length >', requiresValue: true, valueType: 'single' },
    { value: 'length_less', label: 'Length Less Than', description: 'String length <', requiresValue: true, valueType: 'single' }
  ],
  number: [
    { value: 'equals', label: '= Equals', description: 'Equal to', requiresValue: true, valueType: 'single' },
    { value: 'not_equals', label: '≠ Not Equals', description: 'Not equal to', requiresValue: true, valueType: 'single' },
    { value: 'greater_than', label: '> Greater Than', description: 'Greater than', requiresValue: true, valueType: 'single' },
    { value: 'greater_equal', label: '≥ Greater or Equal', description: 'Greater than or equal', requiresValue: true, valueType: 'single' },
    { value: 'less_than', label: '< Less Than', description: 'Less than', requiresValue: true, valueType: 'single' },
    { value: 'less_equal', label: '≤ Less or Equal', description: 'Less than or equal', requiresValue: true, valueType: 'single' },
    { value: 'between', label: 'Between', description: 'Within range (inclusive)', requiresValue: true, requiresSecondValue: true, valueType: 'range' },
    { value: 'not_between', label: 'Not Between', description: 'Outside range', requiresValue: true, requiresSecondValue: true, valueType: 'range' },
    { value: 'in', label: 'In List', description: 'One of values', requiresValue: true, valueType: 'multiple' },
    { value: 'is_null', label: 'Is Null', description: 'No value', requiresValue: false, valueType: 'none' },
    { value: 'is_not_null', label: 'Is Not Null', description: 'Has value', requiresValue: false, valueType: 'none' },
    { value: 'is_positive', label: 'Is Positive', description: '> 0', requiresValue: false, valueType: 'none' },
    { value: 'is_negative', label: 'Is Negative', description: '< 0', requiresValue: false, valueType: 'none' },
    { value: 'is_zero', label: 'Is Zero', description: '= 0', requiresValue: false, valueType: 'none' }
  ],
  date: [
    { value: 'equals', label: 'On Date', description: 'Exact date', requiresValue: true, valueType: 'single' },
    { value: 'not_equals', label: 'Not On Date', description: 'Not on date', requiresValue: true, valueType: 'single' },
    { value: 'before', label: 'Before', description: 'Before date', requiresValue: true, valueType: 'single' },
    { value: 'after', label: 'After', description: 'After date', requiresValue: true, valueType: 'single' },
    { value: 'on_or_before', label: 'On or Before', description: 'On or before date', requiresValue: true, valueType: 'single' },
    { value: 'on_or_after', label: 'On or After', description: 'On or after date', requiresValue: true, valueType: 'single' },
    { value: 'between', label: 'Between', description: 'Within date range', requiresValue: true, requiresSecondValue: true, valueType: 'range' },
    { value: 'in_last_n_days', label: 'In Last N Days', description: 'Within last N days', requiresValue: true, valueType: 'single' },
    { value: 'in_next_n_days', label: 'In Next N Days', description: 'Within next N days', requiresValue: true, valueType: 'single' },
    { value: 'is_today', label: 'Is Today', description: 'Today', requiresValue: false, valueType: 'none' },
    { value: 'is_this_week', label: 'Is This Week', description: 'Current week', requiresValue: false, valueType: 'none' },
    { value: 'is_this_month', label: 'Is This Month', description: 'Current month', requiresValue: false, valueType: 'none' },
    { value: 'is_this_year', label: 'Is This Year', description: 'Current year', requiresValue: false, valueType: 'none' },
    { value: 'is_null', label: 'Is Null', description: 'No date', requiresValue: false, valueType: 'none' },
    { value: 'is_not_null', label: 'Is Not Null', description: 'Has date', requiresValue: false, valueType: 'none' }
  ],
  boolean: [
    { value: 'is_true', label: 'Is True', description: 'True value', requiresValue: false, valueType: 'none' },
    { value: 'is_false', label: 'Is False', description: 'False value', requiresValue: false, valueType: 'none' },
    { value: 'is_null', label: 'Is Null', description: 'No value', requiresValue: false, valueType: 'none' }
  ],
  enum: [
    { value: 'equals', label: 'Equals', description: 'Selected value', requiresValue: true, valueType: 'single' },
    { value: 'not_equals', label: 'Not Equals', description: 'Not selected value', requiresValue: true, valueType: 'single' },
    { value: 'in', label: 'In List', description: 'One of selected', requiresValue: true, valueType: 'multiple' },
    { value: 'not_in', label: 'Not In List', description: 'None of selected', requiresValue: true, valueType: 'multiple' },
    { value: 'is_null', label: 'Is Null', description: 'No value', requiresValue: false, valueType: 'none' }
  ],
  array: [
    { value: 'contains', label: 'Contains', description: 'Array contains value', requiresValue: true, valueType: 'single' },
    { value: 'not_contains', label: 'Does Not Contain', description: 'Array does not contain', requiresValue: true, valueType: 'single' },
    { value: 'contains_all', label: 'Contains All', description: 'Contains all values', requiresValue: true, valueType: 'multiple' },
    { value: 'contains_any', label: 'Contains Any', description: 'Contains any value', requiresValue: true, valueType: 'multiple' },
    { value: 'is_empty', label: 'Is Empty', description: 'Empty array', requiresValue: false, valueType: 'none' },
    { value: 'is_not_empty', label: 'Is Not Empty', description: 'Has elements', requiresValue: false, valueType: 'none' },
    { value: 'length_equals', label: 'Length Equals', description: 'Array size equals', requiresValue: true, valueType: 'single' },
    { value: 'length_greater', label: 'Length Greater Than', description: 'Array size >', requiresValue: true, valueType: 'single' },
    { value: 'length_less', label: 'Length Less Than', description: 'Array size <', requiresValue: true, valueType: 'single' }
  ]
};

// ============================================================================
// Utility Functions
// ============================================================================

const generateId = () => `cond_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

const isCondition = (node: ConditionNode): node is Condition => node.type === 'condition';

const getDepthColor = (depth: number): string => {
  const colors = [
    'border-l-blue-500',
    'border-l-purple-500',
    'border-l-green-500',
    'border-l-orange-500',
    'border-l-pink-500',
    'border-l-teal-500',
    'border-l-indigo-500',
    'border-l-red-500'
  ];
  return colors[depth % colors.length];
};

const getDepthBgColor = (depth: number): string => {
  const colors = [
    'bg-blue-50/50',
    'bg-purple-50/50',
    'bg-green-50/50',
    'bg-orange-50/50',
    'bg-pink-50/50',
    'bg-teal-50/50',
    'bg-indigo-50/50',
    'bg-red-50/50'
  ];
  return colors[depth % colors.length];
};

// ============================================================================
// Props
// ============================================================================

export interface AdvancedConditionBuilderProps {
  value: ConditionGroup;
  onChange: (value: ConditionGroup) => void;
  // Legacy support
  availableFields?: Array<{ name: string; type: string; label: string }>;
  entityName?: string;
  // Full support
  entities?: EntityDefinition[];
  primaryEntity?: string;
  maxDepth?: number;
  enableDragDrop?: boolean;
  enableCrossEntity?: boolean;
  showValidation?: boolean;
  autosaveDelay?: number;
  onAutosave?: (value: ConditionGroup) => void;
  readOnly?: boolean;
  compact?: boolean;
}

// ============================================================================
// Field Autocomplete Component
// ============================================================================

interface FieldAutocompleteProps {
  entities: EntityDefinition[];
  primaryEntity: string;
  value: string;
  fieldPath?: string;
  onChange: (field: string, path?: string, fieldDef?: FieldDefinition) => void;
  enableCrossEntity?: boolean;
  disabled?: boolean;
}

const FieldAutocomplete: React.FC<FieldAutocompleteProps> = ({
  entities,
  primaryEntity,
  value,
  fieldPath,
  onChange,
  enableCrossEntity,
  disabled
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [currentEntity, setCurrentEntity] = useState(primaryEntity);
  const [pathSegments, setPathSegments] = useState<string[]>([]);
  const inputRef = useRef<HTMLInputElement>(null);

  const entity = entities.find(e => e.name === currentEntity);
  const fields = entity?.fields || [];
  const relationships = entity?.relationships || [];

  const filteredFields = useMemo(() => {
    const q = search.toLowerCase();
    return fields.filter(f => 
      f.name.toLowerCase().includes(q) || 
      f.label.toLowerCase().includes(q)
    );
  }, [fields, search]);

  const filteredRelationships = useMemo(() => {
    if (!enableCrossEntity) return [];
    const q = search.toLowerCase();
    return relationships.filter(r =>
      r.name.toLowerCase().includes(q) ||
      r.label.toLowerCase().includes(q)
    );
  }, [relationships, search, enableCrossEntity]);

  const handleSelectField = (field: FieldDefinition) => {
    const fullPath = [...pathSegments, field.name].join('.');
    onChange(field.name, fullPath, field);
    setIsOpen(false);
    setSearch('');
    setPathSegments([]);
    setCurrentEntity(primaryEntity);
  };

  const handleSelectRelationship = (rel: { name: string; targetEntity: string }) => {
    setPathSegments([...pathSegments, rel.name]);
    setCurrentEntity(rel.targetEntity);
    setSearch('');
  };

  const handleBack = () => {
    const newSegments = pathSegments.slice(0, -1);
    setPathSegments(newSegments);
    // Find parent entity
    let entity = primaryEntity;
    for (const seg of newSegments) {
      const ent = entities.find(e => e.name === entity);
      const rel = ent?.relationships.find(r => r.name === seg);
      if (rel) entity = rel.targetEntity;
    }
    setCurrentEntity(entity);
  };

  const displayValue = fieldPath || value;
  const currentPath = pathSegments.length > 0 ? pathSegments.join(' → ') + ' → ' : '';

  return (
    <div className="relative flex-1">
      <div 
        className={`flex items-center gap-2 px-3 py-2 border rounded-lg cursor-pointer transition-colors ${
          disabled ? 'bg-gray-100 cursor-not-allowed' : 'bg-white hover:border-blue-400'
        } ${isOpen ? 'border-blue-500 ring-2 ring-blue-100' : 'border-gray-300'}`}
        onClick={() => !disabled && setIsOpen(true)}
        role="combobox"
        aria-expanded={isOpen ? 'true' : 'false'}
        aria-label="Field selector"
        aria-controls="field-list"
        aria-haspopup="listbox"
        tabIndex={disabled ? -1 : 0}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            setIsOpen(true);
          }
        }}
      >
        {enableCrossEntity && fieldPath && fieldPath.includes('.') && (
          <Link2 size={14} className="text-purple-500 flex-shrink-0" />
        )}
        <span className={`flex-1 truncate ${!displayValue ? 'text-gray-400' : 'text-gray-900'}`}>
          {displayValue || 'Select field...'}
        </span>
        <ChevronDown size={16} className="text-gray-400 flex-shrink-0" />
      </div>

      {isOpen && (
        <>
          <div className="fixed inset-0 z-40" onClick={() => setIsOpen(false)} />
          <div 
            className="absolute top-full left-0 right-0 mt-1 bg-white rounded-lg shadow-xl border z-50 max-h-80 overflow-hidden"
            id="field-list"
          >
            {/* Search Input */}
            <div className="p-2 border-b sticky top-0 bg-white">
              <div className="relative">
                <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  ref={inputRef}
                  type="text"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  placeholder="Search fields..."
                  aria-label="Search fields"
                  className="w-full pl-9 pr-4 py-2 border rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  autoFocus
                />
              </div>
              {pathSegments.length > 0 && (
                <div className="flex items-center gap-2 mt-2 text-xs">
                  <button
                    onClick={handleBack}
                    className="flex items-center gap-1 text-blue-600 hover:underline"
                  >
                    ← Back
                  </button>
                  <span className="text-gray-500">{currentPath}</span>
                </div>
              )}
            </div>

            {/* Field List */}
            <div className="overflow-y-auto max-h-60" role="listbox" aria-label="Available fields and related entities">
              {/* Relationships */}
              {filteredRelationships.length > 0 && (
                <div className="p-2 border-b">
                  <div className="text-xs font-semibold text-gray-500 uppercase mb-1 px-2">
                    Related Entities
                  </div>
                  {filteredRelationships.map(rel => (
                    <button
                      key={rel.name}
                      onClick={() => handleSelectRelationship(rel)}
                      role="option"
                      className="w-full flex items-center gap-2 px-3 py-2 hover:bg-purple-50 rounded text-left"
                    >
                      <Link2 size={14} className="text-purple-500" />
                      <span className="font-medium text-purple-700">{rel.label}</span>
                      <span className="text-xs text-gray-500">→ {rel.targetEntity}</span>
                    </button>
                  ))}
                </div>
              )}

              {/* Fields */}
              {filteredFields.length > 0 ? (
                <div className="p-2">
                  <div className="text-xs font-semibold text-gray-500 uppercase mb-1 px-2">
                    Fields
                  </div>
                  {filteredFields.map(field => (
                    <button
                      key={field.name}
                      onClick={() => handleSelectField(field)}
                      className="w-full flex items-center gap-2 px-3 py-2 hover:bg-blue-50 rounded text-left group"
                      role="option"
                    >
                      <span className={`px-1.5 py-0.5 rounded text-xs font-mono ${
                        field.type === 'string' ? 'bg-green-100 text-green-700' :
                        field.type === 'number' ? 'bg-blue-100 text-blue-700' :
                        field.type === 'date' ? 'bg-orange-100 text-orange-700' :
                        field.type === 'boolean' ? 'bg-purple-100 text-purple-700' :
                        'bg-gray-100 text-gray-700'
                      }`}>
                        {field.type.slice(0, 3)}
                      </span>
                      <span className="font-medium text-gray-900">{field.label}</span>
                      <span className="text-xs text-gray-400 font-mono">{field.name}</span>
                      {field.nullable && (
                        <span className="text-xs text-gray-400 ml-auto">nullable</span>
                      )}
                    </button>
                  ))}
                </div>
              ) : (
                <div className="p-4 text-center text-gray-500 text-sm">
                  No fields found
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

// ============================================================================
// Operator Selector Component
// ============================================================================

interface OperatorSelectorProps {
  fieldType: string;
  value: string;
  onChange: (operator: string) => void;
  disabled?: boolean;
}

const OperatorSelector: React.FC<OperatorSelectorProps> = ({
  fieldType,
  value,
  onChange,
  disabled
}) => {
  const operators = OPERATORS_BY_TYPE[fieldType] || OPERATORS_BY_TYPE.string;
  const selectedOp = operators.find(op => op.value === value);

  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      disabled={disabled}
      className={`px-3 py-2 border rounded-lg text-sm font-medium min-w-[160px] ${
        disabled ? 'bg-gray-100 cursor-not-allowed' : 'bg-white cursor-pointer hover:border-blue-400'
      }`}
      title={selectedOp?.description || 'Select operator'}
      aria-label="Condition operator"
    >
      {operators.map(op => (
        <option key={op.value} value={op.value} title={op.description}>
          {op.label}
        </option>
      ))}
    </select>
  );
};

// ============================================================================
// Value Input Component
// ============================================================================

interface ValueInputProps {
  fieldDef?: FieldDefinition;
  operator: OperatorDefinition;
  value: string | number | boolean | string[];
  secondValue?: string | number;
  onChange: (value: string | number | boolean | string[], secondValue?: string | number) => void;
  disabled?: boolean;
}

const ValueInput: React.FC<ValueInputProps> = ({
  fieldDef,
  operator,
  value,
  secondValue,
  onChange,
  disabled
}) => {
  if (!operator.requiresValue) {
    return <div className="text-sm text-gray-500 italic px-3 py-2">No value needed</div>;
  }

  const fieldType = fieldDef?.type || 'string';

  // Multi-value input for 'in', 'contains_all', etc.
  if (operator.valueType === 'multiple') {
    const values = Array.isArray(value) ? value : (typeof value === 'string' ? value.split(',').map(v => v.trim()) : []);
    
    return (
      <div className="flex-1">
        <input
          type="text"
          value={values.join(', ')}
          onChange={(e) => onChange(e.target.value.split(',').map(v => v.trim()))}
          placeholder="value1, value2, value3"
          disabled={disabled}
          className="w-full px-3 py-2 border rounded-lg text-sm"
          aria-label="Condition values (comma separated)"
        />
        <div className="text-xs text-gray-500 mt-1">Separate multiple values with commas</div>
      </div>
    );
  }

  // Range input for 'between'
  if (operator.valueType === 'range') {
    return (
      <div className="flex items-center gap-2 flex-1">
        <input
          type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
          value={String(value || '')}
          onChange={(e) => onChange(fieldType === 'number' ? Number(e.target.value) : e.target.value, secondValue)}
          disabled={disabled}
          className="flex-1 px-3 py-2 border rounded-lg text-sm"
          placeholder="From"
          aria-label="Range start value"
        />
        <span className="text-gray-500 text-sm">to</span>
        <input
          type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
          value={String(secondValue || '')}
          onChange={(e) => onChange(value, fieldType === 'number' ? Number(e.target.value) : e.target.value)}
          disabled={disabled}
          className="flex-1 px-3 py-2 border rounded-lg text-sm"
          placeholder="To"
          aria-label="Range end value"
        />
      </div>
    );
  }

  // Enum select
  if (fieldType === 'enum' && fieldDef?.enumValues) {
    return (
      <select
        value={String(value)}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        className="flex-1 px-3 py-2 border rounded-lg text-sm"
        aria-label="Condition value"
      >
        <option value="">Select value...</option>
        {fieldDef.enumValues.map(v => (
          <option key={v} value={v}>{v}</option>
        ))}
      </select>
    );
  }

  // Boolean input
  if (fieldType === 'boolean') {
    return (
      <select
        value={String(value)}
        onChange={(e) => onChange(e.target.value === 'true')}
        disabled={disabled}
        className="flex-1 px-3 py-2 border rounded-lg text-sm"
        aria-label="Condition value"
      >
        <option value="true">True</option>
        <option value="false">False</option>
      </select>
    );
  }

  // Standard input
  return (
    <input
      type={fieldType === 'number' ? 'number' : fieldType === 'date' ? 'date' : 'text'}
      value={String(value || '')}
      onChange={(e) => onChange(fieldType === 'number' ? Number(e.target.value) : e.target.value)}
      placeholder={`Enter ${fieldType} value...`}
      disabled={disabled}
      className="flex-1 px-3 py-2 border rounded-lg text-sm"
      aria-label="Condition value"
    />
  );
};

// ============================================================================
// Single Condition Row Component
// ============================================================================

interface ConditionRowProps {
  condition: Condition;
  entities: EntityDefinition[];
  primaryEntity: string;
  depth: number;
  onUpdate: (condition: Condition) => void;
  onDelete: () => void;
  onDuplicate?: () => void;
  enableCrossEntity?: boolean;
  enableDragDrop?: boolean;
  showValidation?: boolean;
  readOnly?: boolean;
}

const ConditionRow: React.FC<ConditionRowProps> = ({
  condition,
  entities,
  primaryEntity,
  depth,
  onUpdate,
  onDelete,
  onDuplicate,
  enableCrossEntity,
  enableDragDrop,
  showValidation,
  readOnly
}) => {
  // Find field definition
  const fieldDef = useMemo(() => {
    if (!condition.field) return undefined;
    
    if (condition.fieldPath && condition.fieldPath.includes('.')) {
      // Cross-entity field - traverse path
      const segments = condition.fieldPath.split('.');
      let currentEntity = primaryEntity;
      for (let i = 0; i < segments.length - 1; i++) {
        const ent = entities.find(e => e.name === currentEntity);
        const rel = ent?.relationships.find(r => r.name === segments[i]);
        if (rel) currentEntity = rel.targetEntity;
      }
      const finalEntity = entities.find(e => e.name === currentEntity);
      return finalEntity?.fields.find(f => f.name === segments[segments.length - 1]);
    }

    const entity = entities.find(e => e.name === primaryEntity);
    return entity?.fields.find(f => f.name === condition.field);
  }, [condition.field, condition.fieldPath, entities, primaryEntity]);

  const fieldType = fieldDef?.type || 'string';
  const operators = OPERATORS_BY_TYPE[fieldType] || OPERATORS_BY_TYPE.string;
  const currentOperator = operators.find(op => op.value === condition.operator) || operators[0];

  const handleFieldChange = (field: string, path?: string, def?: FieldDefinition) => {
    const newOperators = OPERATORS_BY_TYPE[def?.type || 'string'] || OPERATORS_BY_TYPE.string;
    onUpdate({
      ...condition,
      field,
      fieldPath: path,
      operator: newOperators[0].value,
      value: '',
      valueType: def?.type
    });
  };

  return (
    <div 
      className={`flex items-center gap-2 p-2 rounded-lg border transition-all ${
        showValidation && !condition.isValid 
          ? 'border-red-300 bg-red-50' 
          : 'border-gray-200 bg-white hover:border-gray-300'
      }`}
      role="group"
      aria-label="Condition"
      data-depth={depth}
    >
      {/* Drag Handle */}
      {enableDragDrop && !readOnly && (
        <div 
          className="cursor-grab hover:bg-gray-100 p-1 rounded"
          draggable
          aria-label="Drag to reorder"
        >
          <GripVertical size={16} className="text-gray-400" />
        </div>
      )}

      {/* Field Selector */}
      <FieldAutocomplete
        entities={entities}
        primaryEntity={primaryEntity}
        value={condition.field}
        fieldPath={condition.fieldPath}
        onChange={handleFieldChange}
        enableCrossEntity={enableCrossEntity}
        disabled={readOnly}
      />

      {/* Operator */}
      <OperatorSelector
        fieldType={fieldType}
        value={condition.operator}
        onChange={(op) => onUpdate({ ...condition, operator: op })}
        disabled={readOnly}
      />

      {/* Value */}
      <ValueInput
        fieldDef={fieldDef}
        operator={currentOperator}
        value={condition.value}
        secondValue={condition.secondValue}
        onChange={(val, second) => onUpdate({ ...condition, value: val, secondValue: second })}
        disabled={readOnly}
      />

      {/* Validation indicator */}
      {showValidation && (
        <div className="flex-shrink-0">
          {condition.isValid === true && <Check size={16} className="text-green-500" />}
          {condition.isValid === false && (
            <div className="group relative">
              <AlertCircle size={16} className="text-red-500" />
              {condition.validationError && (
                <div className="absolute right-0 top-full mt-1 bg-red-600 text-white text-xs px-2 py-1 rounded shadow-lg opacity-0 group-hover:opacity-100 transition-opacity whitespace-nowrap z-10">
                  {condition.validationError}
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Actions */}
      {!readOnly && (
        <div className="flex items-center gap-1 flex-shrink-0">
          {onDuplicate && (
            <button
              onClick={onDuplicate}
              className="p-1.5 hover:bg-gray-100 rounded text-gray-500 hover:text-gray-700"
              title="Duplicate condition"
              aria-label="Duplicate condition"
            >
              <Copy size={14} />
            </button>
          )}
          <button
            onClick={onDelete}
            className="p-1.5 hover:bg-red-50 rounded text-gray-500 hover:text-red-600"
            title="Delete condition"
            aria-label="Delete condition"
          >
            <Trash2 size={14} />
          </button>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// Condition Group Component
// ============================================================================

interface ConditionGroupComponentProps {
  group: ConditionGroup;
  entities: EntityDefinition[];
  primaryEntity: string;
  depth: number;
  maxDepth: number;
  onChange: (group: ConditionGroup) => void;
  onDelete?: () => void;
  enableCrossEntity?: boolean;
  enableDragDrop?: boolean;
  showValidation?: boolean;
  readOnly?: boolean;
  compact?: boolean;
}

const ConditionGroupComponent: React.FC<ConditionGroupComponentProps> = ({
  group,
  entities,
  primaryEntity,
  depth,
  maxDepth,
  onChange,
  onDelete,
  enableCrossEntity,
  enableDragDrop,
  showValidation,
  readOnly,
  compact
}) => {
  const handleOperatorChange = (op: 'AND' | 'OR' | 'NOT') => {
    onChange({ ...group, operator: op });
  };

  const addCondition = () => {
    onChange({
      ...group,
      conditions: [
        ...group.conditions,
        {
          id: generateId(),
          type: 'condition',
          field: '',
          operator: 'equals',
          value: '',
          isValid: false
        }
      ]
    });
  };

  const addGroup = () => {
    if (depth >= maxDepth) return;
    onChange({
      ...group,
      conditions: [
        ...group.conditions,
        {
          id: generateId(),
          type: 'group',
          operator: 'AND',
          conditions: [],
          isCollapsed: false
        }
      ]
    });
  };

  const updateNode = (index: number, newNode: ConditionNode) => {
    const newConditions = [...group.conditions];
    newConditions[index] = newNode;
    onChange({ ...group, conditions: newConditions });
  };

  const removeNode = (index: number) => {
    const newConditions = group.conditions.filter((_, i) => i !== index);
    onChange({ ...group, conditions: newConditions });
  };

  const duplicateNode = (index: number) => {
    const nodeToDuplicate = group.conditions[index];
    const newNode = JSON.parse(JSON.stringify(nodeToDuplicate));
    newNode.id = generateId(); // simple implementation
    if (newNode.type === 'condition') {
      const newConditions = [...group.conditions];
      newConditions.splice(index + 1, 0, newNode);
      onChange({ ...group, conditions: newConditions });
    }
  };

  const depthColor = getDepthColor(depth);
  const depthBg = getDepthBgColor(depth);

  return (
    <div 
      className={`border-l-4 ${depthColor} ${depthBg} rounded-r-lg p-3 my-2 transition-all`}
      data-depth={depth}
    >
      {/* Group Header */}
      <div className="flex items-center gap-2 mb-2">
        <div className="flex items-center bg-white rounded-lg border border-gray-200 overflow-hidden shadow-sm">
          <select
            value={group.operator}
            onChange={(e) => handleOperatorChange(e.target.value as any)}
            disabled={readOnly}
            className={`px-3 py-1.5 text-sm font-bold border-none outline-none cursor-pointer ${
              group.operator === 'AND' ? 'text-blue-600 bg-blue-50' :
              group.operator === 'OR' ? 'text-purple-600 bg-purple-50' :
              'text-red-600 bg-red-50'
            }`}
          >
            <option value="AND">AND</option>
            <option value="OR">OR</option>
            <option value="NOT">NOT</option>
          </select>
        </div>

        {!readOnly && (
          <div className="flex items-center gap-1 ml-auto">
            <button
              onClick={addCondition}
              className="flex items-center gap-1 px-2 py-1.5 text-xs font-medium text-gray-600 bg-white border border-gray-200 rounded hover:bg-gray-50 hover:border-gray-300 transition-colors"
            >
              <Plus size={14} />
              Condition
            </button>
            {depth < maxDepth && (
              <button
                onClick={addGroup}
                className="flex items-center gap-1 px-2 py-1.5 text-xs font-medium text-gray-600 bg-white border border-gray-200 rounded hover:bg-gray-50 hover:border-gray-300 transition-colors"
                title="Add nested group"
              >
                <Layers size={14} />
                Group
              </button>
            )}
            {onDelete && (
              <button
                onClick={onDelete}
                className="p-1.5 text-gray-400 hover:text-red-500 rounded hover:bg-red-50 transition-colors"
                title="Delete group"
              >
                <Trash2 size={14} />
              </button>
            )}
          </div>
        )}
      </div>

      {/* Conditions */}
      <div className="space-y-2 pl-2">
        {group.conditions.length === 0 && (
          <div className="py-4 text-center text-sm text-gray-400 border-2 border-dashed border-gray-200 rounded-lg">
            No conditions. Add one to start.
          </div>
        )}
        
        {group.conditions.map((node, index) => (
          <div key={node.id}>
            {node.type === 'condition' ? (
              <ConditionRow
                condition={node}
                entities={entities}
                primaryEntity={primaryEntity}
                depth={depth}
                onUpdate={(updated) => updateNode(index, updated)}
                onDelete={() => removeNode(index)}
                onDuplicate={() => duplicateNode(index)}
                enableCrossEntity={enableCrossEntity}
                enableDragDrop={enableDragDrop}
                showValidation={showValidation}
                readOnly={readOnly}
              />
            ) : (
              <ConditionGroupComponent
                group={node}
                entities={entities}
                primaryEntity={primaryEntity}
                depth={depth + 1}
                maxDepth={maxDepth}
                onChange={(updated) => updateNode(index, updated)}
                onDelete={() => removeNode(index)}
                enableCrossEntity={enableCrossEntity}
                enableDragDrop={enableDragDrop}
                showValidation={showValidation}
                readOnly={readOnly}
                compact={compact}
              />
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

// ============================================================================
// Main Component
// ============================================================================

const AdvancedConditionBuilder: React.FC<AdvancedConditionBuilderProps> = ({
  value,
  onChange,
  // New props
  entities,
  primaryEntity,
  maxDepth = 3,
  enableDragDrop = true,
  enableCrossEntity = false,
  showValidation = true,
  autosaveDelay = 1000,
  onAutosave,
  readOnly = false,
  compact = false,
  // Legacy props
  availableFields,
  entityName = 'Entity'
}) => {
  // Adapter Logic: If legacy availableFields are provided, construct the entities prop
  const effectiveEntities = useMemo(() => {
    if (entities) return entities;
    
    // Construct from availableFields
    if (availableFields) {
      const fields: FieldDefinition[] = availableFields.map(f => ({
        name: f.name,
        label: f.label || f.name,
        type: f.type as any || 'string', // Default to string if type mismatch
        description: f.label,
        nullable: true
      }));

      return [{
        name: entityName,
        label: entityName,
        fields: fields,
        relationships: []
      }] as EntityDefinition[];
    }
    
    return [] as EntityDefinition[];
  }, [entities, availableFields, entityName]);

  const effectivePrimaryEntity = primaryEntity || (effectiveEntities.length > 0 ? effectiveEntities[0].name : '');

  // Autosave Logic
  const saveTimeout = useRef<NodeJS.Timeout>();
  const lastSavedRef = useRef<string>(JSON.stringify(value));

  useEffect(() => {
    if (!onAutosave) return;
    
    const currentJson = JSON.stringify(value);
    if (currentJson === lastSavedRef.current) return;

    if (saveTimeout.current) clearTimeout(saveTimeout.current);
    
    saveTimeout.current = setTimeout(() => {
      onAutosave(value);
      lastSavedRef.current = currentJson;
    }, autosaveDelay);

    return () => {
      if (saveTimeout.current) clearTimeout(saveTimeout.current);
    };
  }, [value, onAutosave, autosaveDelay]);

  // Initial State Check
  useEffect(() => {
    if (!value) {
      onChange({
        id: 'root',
        type: 'group',
        operator: 'AND',
        conditions: []
      });
    }
  }, []);

  if (!value) return null;

  return (
    <div className={`advanced-condition-builder bg-white ${compact ? 'p-2' : ''}`}>
      <ConditionGroupComponent
        group={value}
        entities={effectiveEntities}
        primaryEntity={effectivePrimaryEntity}
        depth={0}
        maxDepth={maxDepth}
        onChange={onChange}
        enableCrossEntity={enableCrossEntity}
        enableDragDrop={enableDragDrop}
        showValidation={showValidation}
        readOnly={readOnly}
        compact={compact}
      />
    </div>
  );
};

// ============================================================================
// Evaluation Logic
// ============================================================================

/**
 * Evaluates a condition or group against a data object.
 * This runs client-side evaluation of the rules.
 */
export const evaluateCondition = (node: ConditionNode, data: any): boolean => {
  if (node.type === 'group') {
    const group = node as ConditionGroup;
    if (group.conditions.length === 0) return true; // Empty group matches all (or should it be false? usually true for filters)

    if (group.operator === 'AND') {
      return group.conditions.every(child => evaluateCondition(child, data));
    } else if (group.operator === 'OR') {
      return group.conditions.some(child => evaluateCondition(child, data));
    } else if (group.operator === 'NOT') {
      return !group.conditions.every(child => evaluateCondition(child, data));
    }
    return false;
  }

  // It's a single condition
  const condition = node as Condition;
  const { field, operator, value, secondValue, fieldPath } = condition;
  
  // Resolve value from data
  let actualValue: any;
  if (fieldPath && fieldPath.includes('.')) {
    // Traverse path for cross-entity or nested fields
    const parts = fieldPath.split('.');
    actualValue = data;
    for (const part of parts) {
      if (actualValue === null || actualValue === undefined) break;
      actualValue = actualValue[part];
    }
  } else {
    actualValue = data[field];
  }

  // Handle null/undefined checks first for operators that don't need values
  if (actualValue === null || actualValue === undefined) {
    if (operator === 'is_null' || operator === 'is_empty') return true;
    if (operator === 'is_not_null' || operator === 'is_not_empty') return false;
    // Most other operators fail on null
    return false;
  }

  // Type specific comparisons
  switch (operator) {
    // --- Common / String ---
    case 'equals': return String(actualValue) === String(value);
    case 'not_equals': return String(actualValue) !== String(value);
    case 'contains': return String(actualValue).toLowerCase().includes(String(value).toLowerCase());
    case 'not_contains': return !String(actualValue).toLowerCase().includes(String(value).toLowerCase());
    case 'starts_with': return String(actualValue).toLowerCase().startsWith(String(value).toLowerCase());
    case 'ends_with': return String(actualValue).toLowerCase().endsWith(String(value).toLowerCase());
    case 'matches_regex': 
      try {
        return new RegExp(String(value)).test(String(actualValue));
      } catch (e) {
        return false;
      }
    case 'is_empty': return actualValue === '' || (Array.isArray(actualValue) && actualValue.length === 0);
    case 'is_not_empty': return actualValue !== '' && (!Array.isArray(actualValue) || actualValue.length > 0);
    
    // --- String Length ---
    case 'length_equals': return String(actualValue).length === Number(value);
    case 'length_greater': return String(actualValue).length > Number(value);
    case 'length_less': return String(actualValue).length < Number(value);

    // --- Number ---
    case 'greater_than': return Number(actualValue) > Number(value);
    case 'greater_equal': return Number(actualValue) >= Number(value);
    case 'less_than': return Number(actualValue) < Number(value);
    case 'less_equal': return Number(actualValue) <= Number(value);
    case 'between': {
      const val = Number(actualValue);
      const min = Number(value);
      const max = Number(secondValue);
      return val >= min && val <= max;
    }
    case 'not_between': {
      const val = Number(actualValue);
      const min = Number(value);
      const max = Number(secondValue);
      return val < min || val > max;
    }
    case 'is_positive': return Number(actualValue) > 0;
    case 'is_negative': return Number(actualValue) < 0;
    case 'is_zero': return Number(actualValue) === 0;

    // --- List / In ---
    case 'in': {
      const list = Array.isArray(value) ? value : String(value).split(',').map(v => v.trim());
      return list.some(item => String(item) === String(actualValue));
    }
    case 'not_in': {
      const list = Array.isArray(value) ? value : String(value).split(',').map(v => v.trim());
      return !list.some(item => String(item) === String(actualValue));
    }

    // --- Boolean ---
    case 'is_true': return actualValue === true || actualValue === 'true';
    case 'is_false': return actualValue === false || actualValue === 'false';

    // --- Date (simplified timestamp comparison) ---
    case 'before': return new Date(actualValue) < new Date(String(value));
    case 'after': return new Date(actualValue) > new Date(String(value));
    case 'on_or_before': return new Date(actualValue) <= new Date(String(value));
    case 'on_or_after': return new Date(actualValue) >= new Date(String(value));

    // --- Array ---
    case 'contains_all': {
        if (!Array.isArray(actualValue)) return false;
        const required = Array.isArray(value) ? value : String(value).split(',').map(v => v.trim());
        return required.every(req => actualValue.some(av => String(av) === String(req)));
    }
    case 'contains_any': {
        if (!Array.isArray(actualValue)) return false;
        const candidates = Array.isArray(value) ? value : String(value).split(',').map(v => v.trim());
        return candidates.some(cand => actualValue.some(av => String(av) === String(cand)));
    }
    
    default:
      console.warn(`Unknown operator: ${operator}`);
      return false;
  }
};

export default AdvancedConditionBuilder;

