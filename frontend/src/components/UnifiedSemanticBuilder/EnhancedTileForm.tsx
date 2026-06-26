import { useState, useEffect, useMemo } from 'react';
import type { FC } from 'react';
import * as TablerIcons from '@tabler/icons-react';
import SqlMonacoEditor from '../SqlMonacoEditor';
import TableTypeahead from '../../components/common/TableTypeahead';
import './EnhancedTileForm.css';
import { useTenant } from '../../contexts/TenantContext';
import ActionButton from '../ui/ActionButton';
import { useNotification } from '../../hooks/useNotification';
import { getTableIdFromVal } from '../../utils/tableHelpers';
import { useExtensionsService } from '../../services/extensions';

// Local typed fallbacks to avoid blanket `as any` casts in test/no-provider environments
type TenantCtx = { tenant: unknown | null; product: unknown | null; datasource?: { id?: string } | null; base_model_key?: string | undefined };
type ExtService = { validateExtension: (datasourceId: string, payload: unknown) => Promise<{ issues: Array<Record<string, unknown>> }> };

interface CoreOption {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  sourceTable?: string;
  sourceColumn?: string;
  format?: string;
  aggregationType?: string;
  defaultValue?: string;
}

interface EnhancedTileFormProps {
  // Minimal typed shape for the tile element used in this form.
  element: TileElement | null;
  type: 'dimension' | 'measure' | 'filter' | 'join';
  isCore: boolean;
  isNew: boolean;
  isOverride?: boolean;
  coreOptions: CoreOption[];
  modelName?: string;
  onUpdate: (updates: TileElement) => void;
  onSave: () => void;
  onCancel: () => void;
  onClose?: () => void;
  // When true, renders the form in a read-only "details" mode regardless of core/custom
  readOnly?: boolean;
}

// Local types used to avoid blanket `any` in the form
type TableRef = string | { id?: string; node_name?: string; fetchKey?: string; [k: string]: any };

interface Mapping {
  table?: TableRef;
  column?: string;
  [k: string]: any;
}

interface JoinDef {
  leftTable?: TableRef;
  rightTable?: TableRef;
  [k: string]: any;
}

export interface TileElement {
  id?: string;
  name?: string;
  title?: string;
  sql?: string;
  mappings?: Record<string, Mapping>;
  joins?: JoinDef[];
  sourceTable?: TableRef;
  sourceColumn?: string;
  leftTable?: TableRef;
  rightTable?: TableRef;
  type?: string;
  joinType?: string;
  relationship?: string;
  [k: string]: any;
}

const EnhancedTileForm: FC<EnhancedTileFormProps> = ({
  element,
  type,
  isCore,
  isNew,
  isOverride: _isOverride = false,
  coreOptions,
  modelName,
  onUpdate,
  onSave,
  onCancel,
  onClose: _onClose,
  readOnly = false
}) => {
  const notification = useNotification();
  // Guard context hooks for test environments without providers
  const tenant: TenantCtx = (() => {
    try { return useTenant(); } catch { return { tenant: null, product: null, datasource: null } as TenantCtx; }
  })();
  const { validateExtension } = (() => {
    try { return useExtensionsService(); } catch { return { validateExtension: async () => ({ issues: [] }) } as ExtService; }
  })();
  const [selectedCore, setSelectedCore] = useState<string>('');
  // formData strongly-typed as TileElement to avoid blanket `any`
  const [formData, setFormData] = useState<TileElement>(element ?? ({} as TileElement));
  const [expandedSections, setExpandedSections] = useState({
    basic: true,
    advanced: true,
    sql: true
  });
  const [applying, setApplying] = useState(false);

  // Determine if the current SQL is a calculation (e.g., sum, cast, substr, arithmetic, case)
  const isSqlCalculation = useMemo(() => {
    const sql = (formData?.sql || '').toString().trim().toLowerCase();
    if (!sql) return false; // empty means likely direct column mapping
    // Heuristics: presence of function calls/parens, arithmetic, casts, or CASE expression
    if (/\w+\s*\(/.test(sql)) return true; // any func(
    if (/(\+|\-|\*|\/)/.test(sql)) return true; // arithmetic
    if (/::\w+/.test(sql)) return true; // postgres cast
    if (/\bcase\b|\bwhen\b|\bthen\b|\belse\b|\bend\b/.test(sql)) return true; // CASE
    // specific common functions
    if (/(sum|avg|count|min|max|substr|substring|cast|coalesce|round|concat)\s*\(/.test(sql)) return true;
    return false;
  }, [formData?.sql]);

  // Show/require source fields when not a calculation (direct mapping)
  const showSourceFields = useMemo(() => !isSqlCalculation, [isSqlCalculation]);

  // Keep local form state in sync when the selected element changes (by id).
  // Avoid resetting form state if the same element is re-rendered with a new object reference.
  useEffect(() => {
    try {
      const next = element || {};
  setFormData((prev) => (prev && prev.id === next.id ? prev : next));
      // reset selected core only when switching to a different element
      setSelectedCore('');
    } catch {
      // ignore
    }
    // Depend on element id to minimize unnecessary resets
  }, [element?.id]);

  // When core item is selected, populate form with its data
  useEffect(() => {
    if (isCore && selectedCore && isNew) {
      const coreOption = coreOptions.find(opt => opt.name === selectedCore);
      if (coreOption) {
        setFormData({ ...element, ...coreOption, baseName: selectedCore });
      }
    }
  }, [selectedCore, isCore, isNew, coreOptions, element]);

  // Parameter mapping helper functions
  const parseParameters = (sql: string): string[] => {
    if (!sql) return [];
    // Extract parameters from function calls like XIRR(param1, param2)
    const funcMatch = sql.match(/\w+\s*\(([^)]+)\)/);
    if (funcMatch) {
      return funcMatch[1]
        .split(',')
        .map(p => p.trim())
        .filter(p => p && !p.match(/^\d+$/) && !p.match(/^['"].*['"]$/)); // Exclude numbers and strings
    }
    return [];
  };

  const _inferParameterType = (param: string): string => {
    const paramLower = param.toLowerCase();
    if (paramLower.includes('date') || paramLower.includes('time')) return 'date';
    if (paramLower.includes('value') || paramLower.includes('amount') || paramLower.includes('rate')) return 'number';
    return 'string';
  };

  const _getParameterStatus = (param: string): string => {
    const mapping: Mapping | undefined = formData.mappings?.[param];
    return mapping?.column ? 'mapped' : 'unmapped';
  };

  const _handleParameterMapping = (param: string, field: 'table' | 'column', value: string) => {
    const mappings = formData.mappings || {};
    if (!mappings[param]) mappings[param] = {};
    
    if (field === 'table') {
      mappings[param].table = value;
      mappings[param].column = ''; // Reset column when table changes
    } else {
      mappings[param][field] = value;
    }
    
    handleFieldChange('mappings', mappings);
  };

  const _clearParameterMapping = (param: string) => {
    const mappings = { ...formData.mappings };
    delete mappings[param];
    handleFieldChange('mappings', mappings);
  };

  const _getAvailableTables = (): Array<{ value: string; label: string }> => {
    const tables = [{ value: 'model_table', label: 'Model Table' }];
    
    // Add joined tables from the current model
    if (formData.joins && Array.isArray(formData.joins)) {
      formData.joins.forEach((join: JoinDef) => {
        const right = join.rightTable;
        let tableId = '';
        if (typeof right === 'string') tableId = right;
        else if (right && typeof right === 'object') tableId = right.id || right.node_name || right.fetchKey || '';
        if (tableId && tableId !== 'model_table') {
          const tableLabel = tableId.replace(/_/g, ' ').replace(/\b\w/g, (l: string) => l.toUpperCase());
          if (!tables.find(t => t.value === tableId)) {
            tables.push({ value: tableId, label: tableLabel });
          }
        }
      });
    }
    
    return tables;
  };

  const _getAvailableColumns = (table: string, dataType: string): string[] => {
    // Mock data - in real implementation, this would come from the data catalog
    const mockColumns: Record<string, Record<string, string[]>> = {
      model_table: {
        number: ['revenue', 'profit', 'amount', 'quantity', 'rate', 'cashflow_values', 'order_amount'],
        date: ['transaction_date', 'created_at', 'updated_at', 'due_date', 'cashflow_dates', 'order_date'],
        string: ['customer_name', 'product_name', 'category', 'status', 'transaction_id', 'customer_id']
      },
      customers_table: {
        number: ['lifetime_value', 'annual_spend', 'credit_limit'],
        date: ['registration_date', 'last_purchase_date', 'birth_date'],
        string: ['customer_id', 'customer_name', 'customer_segment', 'email', 'phone']
      },
      products_table: {
        number: ['price', 'cost', 'weight', 'rating'],
        date: ['launch_date', 'discontinued_date', 'last_updated'],
        string: ['product_id', 'product_name', 'category', 'brand', 'sku']
      },
      orders_table: {
        number: ['order_total', 'tax_amount', 'discount_amount', 'shipping_cost'],
        date: ['order_date', 'shipped_date', 'delivered_date', 'expected_delivery'],
        string: ['order_id', 'customer_id', 'shipping_address', 'payment_method']
      }
    };
    
    if (!table || !mockColumns[table]) return [];
    return mockColumns[table][dataType] || [];
  };

  const _getMappedParametersCount = (): number => {
    const sql = formData.sql || '';
    const params = parseParameters(sql);
    return params.filter(param => formData.mappings?.[param]?.column).length;
  };

  const handleFieldChange = (field: string, value: unknown) => {
    // Merge field update into formData. We keep a final typed TileElement cast to satisfy callers.
    const newData = { ...formData, [field]: value } as TileElement;
    setFormData(newData);
    onUpdate(newData);
  };

  const toggleSection = (section: 'basic' | 'advanced' | 'sql') => {
    setExpandedSections(prev => ({ ...prev, [section]: !prev[section as keyof typeof prev] }));
  };

  // Mark helpers as intentionally present to avoid unused warnings in some build paths
  void _inferParameterType;
  void _getParameterStatus;
  void _handleParameterMapping;
  void _clearParameterMapping;
  void _getAvailableTables;
  void _getAvailableColumns;
  void _getMappedParametersCount;

  const handleSave = async () => {
    if (isCore && isNew && !selectedCore) {
      notification.error('Please select a core item first');
      return;
    }
    // Require source table/column for direct mappings (non-SQL calculations)
    if (showSourceFields && !readOnly && !isCore) {
      const stVal = formData?.sourceTable;
      const st = typeof stVal === 'string' ? stVal : (stVal?.id || stVal?.node_name || '').toString();
      const sc = (formData?.sourceColumn || '').toString().trim();
      if (!st || !sc) {
        const eid = element?.id || `${type}__${formData?.name || formData?.title || 'unnamed'}`;
        const issues = [{ level: 'error', code: 'source_required', message: 'Source table and column are required for non-SQL fields.', element_id: eid }];
        try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
        return;
      }
    }
    // Build a minimal model_object containing just this item so backend can validate constraints
    // We include parent_model_key (extends) context through the overall model on server side
    try {
      setApplying(true);
  const datasourceId = (tenant && tenant.datasource && tenant.datasource.id) ? tenant.datasource.id : '';
      if (!datasourceId) {
        notification.error('No datasource selected. Choose a datasource to validate.');
        return;
      }
      // Serialize formData: convert any selected table objects into id strings for validation
      const serializeFormData = (fd: TileElement) => {
        if (!fd) return fd;
        const s: TileElement = { ...fd };
        if (s.sourceTable) s.sourceTable = getTableIdFromVal(s.sourceTable as TableRef);
        if (s.leftTable) s.leftTable = getTableIdFromVal(s.leftTable as TableRef);
        if (s.rightTable) s.rightTable = getTableIdFromVal(s.rightTable as TableRef);
        if (s.joins && Array.isArray(s.joins)) {
          s.joins = s.joins.map((j: JoinDef) => ({
            ...j,
            leftTable: getTableIdFromVal(j.leftTable as TableRef),
            rightTable: getTableIdFromVal(j.rightTable as TableRef),
          }));
        }
        return s;
      };

      const serialized = serializeFormData(formData);

      const model_object: any = { cubes: [] };
      if (type === 'dimension') {
        model_object.dimensions = { [serialized.name || serialized.title || 'unnamed_dimension']: { ...serialized } };
      } else if (type === 'measure') {
        model_object.measures = { [serialized.name || serialized.title || 'unnamed_measure']: { ...serialized } };
      } else if (type === 'filter') {
        // represent filters similarly
        model_object.dimensions = { [serialized.name || serialized.title || 'unnamed_filter']: { ...serialized, type: serialized.type || 'string' } };
      } else if (type === 'join') {
        model_object.joins = { [serialized.name || 'unnamed_join']: { ...serialized } };
      }

      // Attempt validation; backend expects base_model_key + model_object
  const baseKey = (tenant && typeof tenant === 'object' && (tenant as TenantCtx).base_model_key) ? String((tenant as TenantCtx).base_model_key) : undefined; // optional; many paths infer from selected model server-side
      const payload: any = { base_model_key: baseKey, model_object };
      const resp = await validateExtension(datasourceId, payload);
      const issues = resp?.issues || [];
      if (issues.length > 0) {
        // Broadcast and stop so the tile turns red
        try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
        return;
      }
  onSave();
    } catch (e: unknown) {
      // ensure UI highlights error
      const msg = e instanceof Error ? e.message : String(e ?? 'Validation failed');
      const issues = [{ level: 'error', code: 'validation_error', message: msg, element_id: element?.id }];
      try { window.dispatchEvent(new CustomEvent('semlayer.validationIssues', { detail: { issues } })); } catch {}
    } finally {
      setApplying(false);
    }
  };

  return (
    <div className="enhanced-tile-form">
      {/* Header */}
      <div className="tile-form-header">
        <div className="tile-form-title">
          {/* Leading item icon */}
          <span className="element-leading-icon" aria-hidden>
            {type === 'dimension' && <TablerIcons.IconDatabase size={18} />}
            {type === 'measure' && <TablerIcons.IconChartBar size={18} />}
            {type === 'filter' && <TablerIcons.IconFilter size={18} />}
            {type === 'join' && <TablerIcons.IconPlugConnected size={18} />}
          </span>
          {/* Chips row: Core/Custom + Data type */}
          <div className="chips-row">
            <span className={`chip ${isCore ? 'core' : 'custom'}`}>{isCore ? 'Core' : 'Custom'}</span>
            {(() => {
              const dt = type === 'join'
                ? (formData.joinType || formData.relationship || '')
                : (formData.type || '');
              return dt ? <span className="chip datatype">{String(dt)}</span> : null;
            })()}
          </div>
        </div>
        <div className="tile-form-actions">
          {readOnly ? null : (
            <>
              <ActionButton variant="primary" size="sm" onClick={handleSave} pending={!!isCore || applying}>
                Apply
              </ActionButton>
              <ActionButton variant="ghost" size="sm" onClick={onCancel}>
                Cancel
              </ActionButton>
            </>
          )}
        </div>
      </div>
      {/* Cube/Model name on separate prominent line */}
      {modelName && (
        <div className="model-name-large" aria-label="Model name">
          {modelName}
        </div>
      )}

    {/* Basic Section */}
    {(() => { /* stabilize aria-expanded value for lint */ return null; })()}
    {/** Provide literal string values for aria-expanded */}
    {/** basic */}
    {(() => { /* no-op block for clarity */ return null; })()}
    { /* eslint-disable-next-line */ }
    { /* variables for aria compliance */ }
    { /* Note: keeping local to file scope would re-compute; simple inline constants suffice */ }
    { /* Basic */ }
    { /* Advanced and SQL will have their own constants */ }
      <div className="form-section">
  <div 
          className="section-header" 
          role="button"
          tabIndex={0}
          onClick={() => toggleSection('basic')}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('basic'); } }}
          title="Toggle Basic Properties"
        >
          <TablerIcons.IconChevronDown 
            size={16} 
            aria-hidden
            style={{ 
              transform: expandedSections.basic ? 'rotate(0deg)' : 'rotate(-90deg)',
              transition: 'transform 0.2s'
            }} 
          />
          <span>Basic Properties</span>
        </div>
        
        {expandedSections.basic && (
          <div className="section-content">
            {/* Core Selection for new core items */}
            {isCore && isNew && (
              <div className="form-group">
                <label>Select Core {type}:</label>
                <select
                  value={selectedCore}
                  title={`Select Core ${type}`}
                  onChange={(e) => setSelectedCore(e.target.value)}
                  className="form-select"
                >
                  <option value="">Choose a core {type}...</option>
                  {coreOptions.map(option => (
                    <option key={option.name} value={option.name}>
                      {option.title || option.name}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Name field (disabled for core overrides) */}
            <div className="form-group">
              <label>Name:</label>
              <input
                type="text"
                value={formData.name || ''}
                onChange={(e) => handleFieldChange('name', e.target.value)}
                disabled={isCore || readOnly}
                className="form-input"
                placeholder="Enter name"
              />
            </div>

            <div className="form-group">
              <label>Display Title:</label>
              <input
                type="text"
                value={formData.title || ''}
                onChange={(e) => handleFieldChange('title', e.target.value)}
                className="form-input"
                placeholder="Human-readable title"
                disabled={isCore || readOnly}
              />
            </div>

            <div className="form-group">
              <label>Description:</label>
              <textarea
                value={formData.description || ''}
                onChange={(e) => handleFieldChange('description', e.target.value)}
                className="form-textarea"
                placeholder="Optional description"
                rows={2}
                readOnly={isCore || readOnly}
              />
            </div>

            <div className="form-group">
              <label>Data Type:</label>
              <select
                value={formData.type || 'string'}
                title="Data Type"
                onChange={(e) => handleFieldChange('type', e.target.value)}
                className="form-select"
                disabled={isCore || readOnly}
              >
                <option value="string">String</option>
                <option value="number">Number</option>
                <option value="boolean">Boolean</option>
                <option value="time">Time</option>
                <option value="date">Date</option>
              </select>
            </div>
          </div>
        )}
      </div>

    {/* Advanced Section */}
      <div className="form-section">
  <div 
          className="section-header" 
          role="button"
          tabIndex={0}
          onClick={() => toggleSection('advanced')}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('advanced'); } }}
          title="Toggle Advanced Properties"
        >
          <TablerIcons.IconChevronDown 
            size={16} 
            aria-hidden
            style={{ 
              transform: expandedSections.advanced ? 'rotate(0deg)' : 'rotate(-90deg)',
              transition: 'transform 0.2s'
            }} 
          />
          <span>Advanced Properties</span>
        </div>
        
        {expandedSections.advanced && (
          <div className="section-content">
            {showSourceFields && (
              <>
                <div className="form-group">
                  <label>Source Table:</label>
                  <input
                    type="text"
                    value={getTableIdFromVal(formData.sourceTable) || ''}
                    onChange={(e) => handleFieldChange('sourceTable', e.target.value)}
                    className="form-input"
                    placeholder="Source table name"
                    disabled={isCore || readOnly}
                  />
                </div>

                <div className="form-group">
                  <label>Source Column:</label>
                  <input
                    type="text"
                    value={formData.sourceColumn || ''}
                    onChange={(e) => handleFieldChange('sourceColumn', e.target.value)}
                    className="form-input"
                    placeholder="Source column name"
                    disabled={isCore || readOnly}
                  />
                </div>
              </>
            )}

            {type === 'measure' && (
              <>
                <div className="form-group">
                  <label>Number Format:</label>
                  <input
                    type="text"
                    value={formData.format || ''}
                    onChange={(e) => handleFieldChange('format', e.target.value)}
                    className="form-input"
                    placeholder="e.g., #,##0.00"
                    disabled={isCore}
                  />
                </div>

                <div className="form-group">
                  <label>Aggregation:</label>
                  <select
                    value={formData.aggregationType || 'sum'}
                    title="Aggregation"
                    onChange={(e) => handleFieldChange('aggregationType', e.target.value)}
                    className="form-select"
                    disabled={isCore || readOnly}
                  >
                    <option value="sum">Sum</option>
                    <option value="count">Count</option>
                    <option value="avg">Average</option>
                    <option value="min">Minimum</option>
                    <option value="max">Maximum</option>
                  </select>
                </div>
              </>
            )}

            {type === 'filter' && (
              <div className="form-group">
                <label>Default Value:</label>
                <input
                  type="text"
                  value={formData.defaultValue || ''}
                  onChange={(e) => handleFieldChange('defaultValue', e.target.value)}
                  className="form-input"
                  placeholder="Default filter value"
                  disabled={isCore || readOnly}
                />
              </div>
            )}

            {type === 'join' && (
              <>
                <div className="form-group">
                  <label>Left Table:</label>
                  <TableTypeahead
                    fetchOptions={true}
                    datasourceId={tenant?.datasource?.id}
                    value={getTableIdFromVal(formData.leftTable) || ''}
                    onChange={(v) => {
                      // normalize selection to an ID string for internal model
                      return handleFieldChange('leftTable', getTableIdFromVal(v) || '');
                    }}
                    placeholder="Left table name"
                    disabled={isCore || readOnly}
                  />
                </div>
                <div className="form-group">
                  <label>Right Table:</label>
                  <TableTypeahead
                    fetchOptions={true}
                    datasourceId={tenant?.datasource?.id}
                    value={getTableIdFromVal(formData.rightTable) || ''}
                    onChange={(v) => {
                      return handleFieldChange('rightTable', getTableIdFromVal(v) || '');
                    }}
                    placeholder="Right table name"
                    disabled={isCore || readOnly}
                  />
                </div>
                <div className="form-group">
                  <label>Join Type:</label>
                  <select
                    value={formData.joinType || 'inner'}
                    onChange={(e) => handleFieldChange('joinType', e.target.value)}
                    className="form-input"
                    disabled={isCore || readOnly}
                    title="Select the type of join"
                  >
                    <option value="inner">Inner Join</option>
                    <option value="left">Left Join</option>
                    <option value="right">Right Join</option>
                    <option value="full">Full Outer Join</option>
                  </select>
                </div>
              </>
            )}
          </div>
        )}
      </div>

    {/* SQL Section */}
      <div className="form-section">
  <div 
          className="section-header" 
          role="button"
          tabIndex={0}
          onClick={() => toggleSection('sql')}
          onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); toggleSection('sql'); } }}
          title="Toggle SQL Expression"
        >
          <TablerIcons.IconChevronDown 
            size={16} 
            aria-hidden
            style={{ 
              transform: expandedSections.sql ? 'rotate(0deg)' : 'rotate(-90deg)',
              transition: 'transform 0.2s'
            }} 
          />
          <span>SQL Expression</span>
        </div>
        
        {expandedSections.sql && (
          <div className="section-content">
            <div className="form-group">
              <label>SQL Expression:</label>
              <SqlMonacoEditor
                value={formData.sql || ''}
                onChange={(value) => handleFieldChange('sql', value)}
                placeholder={`SQL expression for ${type}`}
                height={120}
                readOnly={isCore || readOnly}
              />
              <small className="field-hint">
                {showSourceFields
                  ? 'Provide Source Table and Source Column for direct column mappings.'
                  : 'Using a SQL calculation (e.g., SUM, CAST, SUBSTR); source fields are optional.'}
              </small>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default EnhancedTileForm;
