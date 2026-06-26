import { useState, useEffect, useId } from 'react';
import { 
  Plus, 
  Save, 
  X, 
  Type, 
  Code,
  Hash,
  Clock,
  Globe,
  CheckCircle,
  Info,
  EyeOff,
  Calendar,
  ChevronDown,
  ChevronRight,
  Copy,
  Edit3,
  Trash2
} from 'lucide-react';
// SqlMonacoEditor not used in this file variant

// Types based on Cube.js dimension parameters
interface CaseWhen {
  sql: string;
  label: string | { sql: string };
}

interface CaseElse {
  label: string | { sql: string };
}

interface Granularity {
  id: string;
  name: string;
  interval: string;
  offset?: string;
  origin?: string;
  title?: string;
}

interface Dimension {
  id: string;
  name: string;
  sql: string;
  type: 'string' | 'number' | 'time' | 'boolean' | 'geo';
  title?: string;
  description?: string;
  format?: 'link' | 'id' | 'currency' | 'percent' | 'number' | 'external_url' | ''; // Based on common formats
  meta?: Record<string, any>;
  primary_key?: boolean;
  public?: boolean;
  sub_query?: boolean;
  propagate_filters_to_sub_query?: boolean;
  case?: {
    when: CaseWhen[];
    else: CaseElse;
  };
  granularities?: Granularity[];
  isEditing?: boolean;
}

const dimensionTypes = [
  { value: 'string' as const, label: 'String', icon: Type },
  { value: 'number' as const, label: 'Number', icon: Hash },
  { value: 'time' as const, label: 'Time', icon: Clock },
  { value: 'boolean' as const, label: 'Boolean', icon: CheckCircle },
  { value: 'geo' as const, label: 'Geo', icon: Globe }
];

const formatOptions = [
  { value: '' as const, label: 'No Format' },
  { value: 'link' as const, label: 'Link' },
  { value: 'id' as const, label: 'ID' },
  { value: 'currency' as const, label: 'Currency' },
  { value: 'percent' as const, label: 'Percent' },
  { value: 'number' as const, label: 'Number' },
  { value: 'external_url' as const, label: 'External URL' }
];

const timeUnits = ['second', 'minute', 'hour', 'day', 'week', 'month', 'quarter', 'year'];

interface DimensionBuilderProps {
  dimensions: Dimension[];
  onDimensionsChange: (dimensions: Dimension[]) => void;
  onGenerateCode?: () => void;
}

export default function DimensionBuilder({ 
  dimensions, 
  onDimensionsChange,
  onGenerateCode 
}: DimensionBuilderProps) {
  const [newDimension, setNewDimension] = useState<Partial<Dimension>>({
    type: 'string' as const,
    public: true,
    format: '' as const
  });
  const [showNewDimensionForm, setShowNewDimensionForm] = useState(false);
  const [expandedSections, setExpandedSections] = useState<{[key: string]: boolean}>({});

  // Helper functions
  const generateDimensionId = () => `dimension_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;

  const createDimension = () => {
    if (!newDimension.name || !newDimension.sql || !newDimension.type) return;

    const dimension: Dimension = {
      id: generateDimensionId(),
      name: newDimension.name,
      sql: newDimension.sql,
      type: newDimension.type,
      title: newDimension.title,
      description: newDimension.description,
      format: newDimension.format || undefined,
      meta: newDimension.meta || {},
      primary_key: newDimension.primary_key,
      public: newDimension.public !== false,
      sub_query: newDimension.sub_query,
      propagate_filters_to_sub_query: newDimension.propagate_filters_to_sub_query,
      case: undefined,
      granularities: []
    };

    onDimensionsChange([...dimensions, dimension]);
    setNewDimension({ type: 'string' as const, public: true, format: '' as const });
    setShowNewDimensionForm(false);
  };

  const updateDimension = (id: string, updates: Partial<Dimension>) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === id ? { ...d, ...updates, isEditing: false } : d
    ));
  };

  const deleteDimension = (id: string) => {
    onDimensionsChange(dimensions.filter(d => d.id !== id));
  };

  const toggleEdit = (id: string) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === id ? { ...d, isEditing: !d.isEditing } : d
    ));
  };

  const addGranularity = (dimensionId: string, granularity: Granularity) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === dimensionId 
        ? { ...d, granularities: [...(d.granularities || []), granularity] }
        : d
    ));
  };

  const updateGranularity = (dimensionId: string, granularityId: string, updates: Partial<Granularity>) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === dimensionId 
        ? { 
            ...d, 
            granularities: (d.granularities || []).map(g => 
              g.id === granularityId ? { ...g, ...updates } : g
            )
          }
        : d
    ));
  };

  const removeGranularity = (dimensionId: string, granularityId: string) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === dimensionId 
        ? { ...d, granularities: (d.granularities || []).filter(g => g.id !== granularityId) }
        : d
    ));
  };

  const updateCase = (dimensionId: string, caseObj: Dimension['case']) => {
    onDimensionsChange(dimensions.map(d => 
      d.id === dimensionId ? { ...d, case: caseObj } : d
    ));
  };

  const toggleSection = (sectionId: string) => {
    setExpandedSections(prev => ({
      ...prev,
      [sectionId]: !prev[sectionId]
    }));
  };

  const generateDimensionCode = (dimension: Dimension) => {
    let code = `${dimension.name}: {
  sql: \`${dimension.sql}\`,
  type: \`${dimension.type}\`${dimension.title ? `,\n  title: \`${dimension.title}\`` : ''}${dimension.description ? `,\n  description: \`${dimension.description}\`` : ''}${dimension.format ? `,\n  format: \`${dimension.format}\`` : ''}${dimension.primary_key ? `,\n  primary_key: ${dimension.primary_key}` : ''}${dimension.public === false ? `,\n  public: false` : ''}${dimension.sub_query ? `,\n  sub_query: true` : ''}${dimension.propagate_filters_to_sub_query ? `,\n  propagate_filters_to_sub_query: true` : ''}`;

    if (dimension.case) {
      code += `,\n  case: {\n    when: [\n`;
      dimension.case.when.forEach(w => {
        const label = typeof w.label === 'string' ? `\`${w.label}\`` : `{ sql: \`${w.label.sql}\` }`;
        code += `      { sql: \`${w.sql}\`, label: ${label} },\n`;
      });
      code += `    ],\n`;
      const elseLabel = typeof dimension.case.else.label === 'string' ? `\`${dimension.case.else.label}\`` : `{ sql: \`${dimension.case.else.label.sql}\` }`;
      code += `    else: { label: ${elseLabel} }\n  }`;
    }

    if (dimension.granularities && dimension.granularities.length > 0) {
      code += `,\n  granularities: {\n`;
      dimension.granularities.forEach(g => {
        code += `    ${g.name}: {\n      interval: \`${g.interval}\`${g.offset ? `,\n      offset: \`${g.offset}\`` : ''}${g.origin ? `,\n      origin: \`${g.origin}\`` : ''}${g.title ? `,\n      title: \`${g.title}\`` : ''}\n    },\n`;
      });
      code += `  }`;
    }

    if (Object.keys(dimension.meta || {}).length > 0) {
      code += `,\n  meta: ${JSON.stringify(dimension.meta, null, 2)}`;
    }

    code += `\n}`;
    return code;
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  return (
    <div className="dimension-builder">
      {/* Header */}
      <div className="mb-6 bg-gradient-to-r from-indigo-500 to-purple-600 text-white p-6 rounded-xl shadow-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Type className="w-8 h-8" />
            <div>
              <h2 className="text-2xl font-bold">Dimension Builder</h2>
              <p className="text-indigo-100">Create and manage Cube.js dimensions with advanced features</p>
            </div>
          </div>
          
          <div className="flex items-center gap-3">
            <span className="bg-white/20 px-3 py-1 rounded-full text-sm font-medium">
              {dimensions.length} dimension{dimensions.length !== 1 ? 's' : ''}
            </span>
            <button
              onClick={() => setShowNewDimensionForm(prev => !prev)}
              className="bg-white/20 hover:bg-white/30 px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
            >
              <Plus className="w-4 h-4" />
              Add Dimension
            </button>
          </div>
        </div>
      </div>

      {/* New Dimension Form */}
      {showNewDimensionForm && (
        <div className="mb-6 bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <Plus className="w-5 h-5 text-indigo-500" />
            Create New Dimension
          </h3>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Name *</label>
              <input
                type="text"
                value={newDimension.name || ''}
                onChange={(e) => setNewDimension(prev => ({ ...prev, name: e.target.value }))}
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                placeholder="dimension_name"
              />
            </div>
            
            <div>
              <label htmlFor="new-dimension-type" className="block text-sm font-medium text-gray-700 mb-1">Type *</label>
              <select
                id="new-dimension-type"
                value={newDimension.type || 'string'}
                onChange={(e) => setNewDimension(prev => ({ ...prev, type: e.target.value as Dimension['type'] }))}
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                {dimensionTypes.map(type => (
                  <option key={type.value} value={type.value}>{type.label}</option>
                ))}
              </select>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">SQL Expression *</label>
              <input
                type="text"
                value={newDimension.sql || ''}
                onChange={(e) => setNewDimension(prev => ({ ...prev, sql: e.target.value }))}
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
                placeholder="{CUBE}.column_name"
              />
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
              <input
                type="text"
                value={newDimension.title || ''}
                onChange={(e) => setNewDimension(prev => ({ ...prev, title: e.target.value }))}
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                placeholder="Human readable title"
              />
            </div>
          </div>
          
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea
              value={newDimension.description || ''}
              onChange={(e) => setNewDimension(prev => ({ ...prev, description: e.target.value }))}
              className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              rows={2}
              placeholder="Dimension description"
            />
          </div>
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mb-4">
            <div>
              <label htmlFor="new-dimension-format" className="block text-sm font-medium text-gray-700 mb-1">Format</label>
              <select
                id="new-dimension-format"
                value={newDimension.format || ''}
                onChange={(e) => setNewDimension(prev => ({ ...prev, format: e.target.value as Dimension['format'] }))}
                className="w-full p-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                {formatOptions.map(format => (
                  <option key={format.value} value={format.value}>{format.label}</option>
                ))}
              </select>
            </div>
            
            <div className="flex flex-wrap gap-4 mt-6">
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={newDimension.public !== false}
                  onChange={(e) => setNewDimension(prev => ({ ...prev, public: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-gray-700">Public</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!newDimension.primary_key}
                  onChange={(e) => setNewDimension(prev => ({ ...prev, primary_key: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-gray-700">Primary Key</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!newDimension.sub_query}
                  onChange={(e) => setNewDimension(prev => ({ ...prev, sub_query: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-gray-700">Sub Query</span>
              </label>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={!!newDimension.propagate_filters_to_sub_query}
                  onChange={(e) => setNewDimension(prev => ({ ...prev, propagate_filters_to_sub_query: e.target.checked }))}
                  className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
                />
                <span className="text-sm font-medium text-gray-700">Propagate Filters</span>
              </label>
            </div>
          </div>
          
          <div className="flex gap-3">
            <button
              onClick={createDimension}
              disabled={!newDimension.name || !newDimension.sql || !newDimension.type}
              className="bg-indigo-500 hover:bg-indigo-600 disabled:opacity-50 disabled:cursor-not-allowed text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
            >
              <Save className="w-4 h-4" />
              Create Dimension
            </button>
            <button
              onClick={() => setShowNewDimensionForm(false)}
              className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
            >
              <X className="w-4 h-4" />
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Dimensions List */}
      <div className="space-y-4">
        {dimensions.length === 0 ? (
          <div className="text-center py-12 bg-gray-50 rounded-xl border-2 border-dashed border-gray-300">
            <Type className="w-12 h-12 text-gray-400 mx-auto mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No dimensions yet</h3>
            <p className="text-gray-500 mb-4">Create your first dimension to get started</p>
            <button
              onClick={() => setShowNewDimensionForm(true)}
              className="bg-indigo-500 hover:bg-indigo-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2 mx-auto"
            >
              <Plus className="w-4 h-4" />
              Add First Dimension
            </button>
          </div>
        ) : (
      dimensions.map((_dimension) => (
            <DimensionCard
        key={_dimension.id}
        dimension={_dimension}
              onUpdate={updateDimension}
              onDelete={deleteDimension}
              onToggleEdit={toggleEdit}
              onAddGranularity={addGranularity}
              onUpdateGranularity={updateGranularity}
              onRemoveGranularity={removeGranularity}
              onUpdateCase={updateCase}
              expandedSections={expandedSections}
              onToggleSection={toggleSection}
              onGenerateCode={generateDimensionCode}
              onCopyCode={copyToClipboard}
            />
          ))
        )}
      </div>

      {/* Generate All Code */}
      {dimensions.length > 0 && onGenerateCode && (
        <div className="mt-8 p-4 bg-gray-50 rounded-xl">
          <button
            onClick={onGenerateCode}
            className="w-full bg-green-500 hover:bg-green-600 text-white px-4 py-3 rounded-lg transition-colors flex items-center gap-2 justify-center font-medium"
          >
            <Code className="w-5 h-5" />
            Generate Complete Cube Schema
          </button>
        </div>
      )}
    </div>
  );
}

// Individual Dimension Card Component
interface DimensionCardProps {
  dimension: Dimension;
  onUpdate: (id: string, updates: Partial<Dimension>) => void;
  onDelete: (id: string) => void;
  onToggleEdit: (id: string) => void;
  onAddGranularity: (dimensionId: string, granularity: Granularity) => void;
  onUpdateGranularity: (dimensionId: string, granularityId: string, updates: Partial<Granularity>) => void;
  onRemoveGranularity: (dimensionId: string, granularityId: string) => void;
  onUpdateCase: (dimensionId: string, caseObj: Dimension['case']) => void;
  expandedSections: {[key: string]: boolean};
  onToggleSection: (sectionId: string) => void;
  onGenerateCode: (dimension: Dimension) => string;
  onCopyCode: (code: string) => void;
}

function DimensionCard({
  dimension,
  onUpdate,
  onDelete,
  onToggleEdit,
  onAddGranularity,
  onUpdateGranularity,
  onRemoveGranularity,
  onUpdateCase,
  expandedSections,
  onToggleSection,
  onGenerateCode,
  onCopyCode
}: DimensionCardProps) {
  const [editData, setEditData] = useState<Partial<Dimension>>(dimension);
  // use explicit 'json' | null to avoid boolean toggles and align with code panel API
  const [showCode, setShowCode] = useState<string | null>(null);

  useEffect(() => {
    setEditData(dimension);
  }, [dimension]);

  const getDimensionTypeIcon = (type: string) => {
    const typeConfig = dimensionTypes.find(t => t.value === type);
    return typeConfig?.icon || Type;
  };

  const saveEdit = () => {
    onUpdate(dimension.id, editData);
  };

  const cancelEdit = () => {
    setEditData(dimension);
    onToggleEdit(dimension.id);
  };

  const TypeIcon = getDimensionTypeIcon(dimension.type);

  return (
    <div className="bg-white border border-gray-200 rounded-xl shadow-sm overflow-hidden">
      {/* Header */}
      <div className="p-4 bg-gradient-to-r from-gray-50 to-gray-100 border-b border-gray-200">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-indigo-100 rounded-lg">
              <TypeIcon className="w-5 h-5 text-indigo-600" />
            </div>
            <div>
              <h3 className="font-semibold text-gray-900 flex items-center gap-2">
                {dimension.title || dimension.name}
                {!dimension.public && <EyeOff className="w-4 h-4 text-gray-400" />}
              </h3>
              <div className="flex items-center gap-3 text-sm text-gray-500">
                <span className="font-mono bg-gray-200 px-2 py-1 rounded text-xs">
                  {dimension.type}
                </span>
                {dimension.format && (
                  <span className="flex items-center gap-1">
                    {dimension.format}
                  </span>
                )}
                <span className="font-mono">{dimension.name}</span>
              </div>
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowCode(prev => (prev ? null : 'json'))}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
              title="Show/Hide Code"
            >
              <Code className="w-4 h-4" />
            </button>
            <button
              onClick={() => onCopyCode(onGenerateCode(dimension))}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-200 rounded-lg transition-colors"
              title="Copy Code"
            >
              <Copy className="w-4 h-4" />
            </button>
            <button
              onClick={() => onToggleEdit(dimension.id)}
              className="p-2 text-indigo-500 hover:text-indigo-700 hover:bg-indigo-100 rounded-lg transition-colors"
              aria-label="Edit dimension"
              title="Edit"
            >
              <Edit3 className="w-4 h-4" />
            </button>
            <button
              onClick={() => onDelete(dimension.id)}
              className="p-2 text-red-500 hover:text-red-700 hover:bg-red-100 rounded-lg transition-colors"
              aria-label="Delete dimension"
              title="Delete"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Code Preview */}
      {showCode && (
        <div className="p-4 bg-gray-900 text-gray-100">
          <pre className="text-sm font-mono overflow-x-auto whitespace-pre">
            {onGenerateCode(dimension)}
          </pre>
        </div>
      )}

      {/* Content */}
      <div className="p-4">
        {dimension.isEditing ? (
          <EditDimensionForm
            dimension={dimension}
            editData={editData}
            setEditData={setEditData}
            onSave={saveEdit}
            onCancel={cancelEdit}
          />
        ) : (
          <ViewDimensionDetails
            dimension={dimension}
            onAddGranularity={onAddGranularity}
            onUpdateGranularity={onUpdateGranularity}
            onRemoveGranularity={onRemoveGranularity}
            onUpdateCase={onUpdateCase}
            expandedSections={expandedSections}
            onToggleSection={onToggleSection}
          />
        )}
      </div>
    </div>
  );
}

// Edit Form Component
interface EditDimensionFormProps {
  dimension: Dimension;
  editData: Partial<Dimension>;
  setEditData: (data: Partial<Dimension>) => void;
  onSave: () => void;
  onCancel: () => void;
}

function EditDimensionForm({ dimension: _dimension, editData, setEditData, onSave, onCancel }: EditDimensionFormProps) {
  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor={`edit-name-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input
            id={`edit-name-${_dimension.id}`}
            type="text"
            value={editData.name || ''}
            onChange={(e) => setEditData({ ...editData, name: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        
        <div>
          <label htmlFor={`edit-type-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">Type</label>
          <select
            id={`edit-type-${_dimension.id}`}
            value={editData.type}
            onChange={(e) => setEditData({ ...editData, type: e.target.value as Dimension['type'] })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            {dimensionTypes.map(type => (
              <option key={type.value} value={type.value}>{type.label}</option>
            ))}
          </select>
        </div>
        
        <div>
          <label htmlFor={`edit-sql-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">SQL</label>
          <input
            id={`edit-sql-${_dimension.id}`}
            type="text"
            value={editData.sql || ''}
            onChange={(e) => setEditData({ ...editData, sql: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 font-mono"
          />
        </div>
        
        <div>
          <label htmlFor={`edit-title-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">Title</label>
          <input
            id={`edit-title-${_dimension.id}`}
            type="text"
            value={editData.title || ''}
            onChange={(e) => setEditData({ ...editData, title: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
      </div>
      
      <div>
        <label htmlFor={`edit-desc-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">Description</label>
        <textarea
          id={`edit-desc-${_dimension.id}`}
          value={editData.description || ''}
          onChange={(e) => setEditData({ ...editData, description: e.target.value })}
          className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          rows={2}
        />
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label htmlFor={`edit-format-${_dimension.id}`} className="block text-sm font-medium text-gray-700 mb-1">Format</label>
          <select
            id={`edit-format-${_dimension.id}`}
            value={editData.format ?? ''}
            onChange={(e) => setEditData({ ...editData, format: e.target.value as Dimension['format'] })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          >
            {formatOptions.map(format => (
              <option key={format.value} value={format.value}>{format.label}</option>
            ))}
          </select>
        </div>
        
        <div className="flex flex-wrap gap-4 mt-6">
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={editData.public !== false}
              onChange={(e) => setEditData({ ...editData, public: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Public</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.primary_key}
              onChange={(e) => setEditData({ ...editData, primary_key: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Primary Key</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.sub_query}
              onChange={(e) => setEditData({ ...editData, sub_query: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Sub Query</span>
          </label>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={!!editData.propagate_filters_to_sub_query}
              onChange={(e) => setEditData({ ...editData, propagate_filters_to_sub_query: e.target.checked })}
              className="w-4 h-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500"
            />
            <span className="text-sm font-medium text-gray-700">Propagate Filters</span>
          </label>
        </div>
      </div>
      
      <div className="flex gap-3 pt-4 border-t border-gray-200">
        <button
          onClick={onSave}
          className="bg-indigo-500 hover:bg-indigo-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Save
        </button>
        <button
          onClick={onCancel}
          className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Cancel
        </button>
      </div>
    </div>
  );
}

// View Details Component
interface ViewDimensionDetailsProps {
  dimension: Dimension;
  onAddGranularity: (dimensionId: string, granularity: Granularity) => void;
  onUpdateGranularity: (dimensionId: string, granularityId: string, updates: Partial<Granularity>) => void;
  onRemoveGranularity: (dimensionId: string, granularityId: string) => void;
  onUpdateCase: (dimensionId: string, caseObj: Dimension['case']) => void;
  expandedSections: {[key: string]: boolean};
  onToggleSection: (sectionId: string) => void;
}

function ViewDimensionDetails({
  dimension,
  onAddGranularity,
  onUpdateGranularity,
  onRemoveGranularity,
  onUpdateCase,
  expandedSections,
  onToggleSection
}: ViewDimensionDetailsProps) {
  const getSectionKey = (suffix: string) => `${dimension.id}_${suffix}`;

  return (
    <div className="space-y-6">
      {/* Basic Info */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="bg-gray-50 p-3 rounded-lg">
          <label className="block text-xs font-medium text-gray-500 mb-1">SQL Expression</label>
          <code className="text-sm font-mono text-gray-900 break-all">{dimension.sql}</code>
        </div>
        
        {dimension.description && (
          <div className="bg-gray-50 p-3 rounded-lg">
            <label className="block text-xs font-medium text-gray-500 mb-1">Description</label>
            <p className="text-sm text-gray-700">{dimension.description}</p>
          </div>
        )}
      </div>

      {/* Case Section */}
      <div className="border border-gray-200 rounded-lg">
        <button
          onClick={() => onToggleSection(getSectionKey('case'))}
          className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
        >
          <div className="flex items-center gap-2">
            <Info className="w-5 h-5 text-blue-500" />
            <span className="font-medium">Case Statement</span>
            {dimension.case && (
              <span className="text-sm text-gray-500">
                ({dimension.case.when.length} conditions)
              </span>
            )}
          </div>
          {expandedSections[getSectionKey('case')] ? (
            <ChevronDown className="w-5 h-5 text-gray-400" />
          ) : (
            <ChevronRight className="w-5 h-5 text-gray-400" />
          )}
        </button>
        
        {expandedSections[getSectionKey('case')] && (
          <div className="border-t border-gray-200 p-4">
            <CaseEditor
              caseObj={dimension.case}
              onUpdate={(caseObj) => onUpdateCase(dimension.id, caseObj)}
            />
          </div>
        )}
      </div>

      {/* Granularities Section (only for time dimensions) */}
      {dimension.type === 'time' && (
        <div className="border border-gray-200 rounded-lg">
          <button
            onClick={() => onToggleSection(getSectionKey('granularities'))}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Calendar className="w-5 h-5 text-green-500" />
              <span className="font-medium">Granularities</span>
              <span className="text-sm text-gray-500">
                ({(dimension.granularities || []).length})
              </span>
            </div>
            {expandedSections[getSectionKey('granularities')] ? (
              <ChevronDown className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronRight className="w-5 h-5 text-gray-400" />
            )}
          </button>
          
          {expandedSections[getSectionKey('granularities')] && (
            <div className="border-t border-gray-200 p-4 space-y-4">
              {/* Add Granularity */}
              <GranularityForm
                onAdd={(granularity) => onAddGranularity(dimension.id, { ...granularity, id: `gran_${Date.now()}` })}
              />
              
              {/* Granularities List */}
              {dimension.granularities && dimension.granularities.length > 0 ? (
                <div className="space-y-3">
                  {dimension.granularities.map(gran => (
                    <GranularityEditor
                      key={gran.id}
                      granularity={gran}
                      onUpdate={(updates) => onUpdateGranularity(dimension.id, gran.id, updates)}
                      onRemove={() => onRemoveGranularity(dimension.id, gran.id)}
                    />
                  ))}
                </div>
              ) : (
                <p className="text-gray-500 text-sm italic text-center py-4">
                  No custom granularities configured
                </p>
              )}
            </div>
          )}
        </div>
      )}

      {/* Meta Information */}
      {dimension.meta && Object.keys(dimension.meta).length > 0 && (
        <div className="border border-gray-200 rounded-lg">
          <button
            onClick={() => onToggleSection(getSectionKey('meta'))}
            className="w-full flex items-center justify-between p-4 hover:bg-gray-50 transition-colors"
          >
            <div className="flex items-center gap-2">
              <Code className="w-5 h-5 text-purple-500" />
              <span className="font-medium">Meta Information</span>
            </div>
            {expandedSections[getSectionKey('meta')] ? (
              <ChevronDown className="w-5 h-5 text-gray-400" />
            ) : (
              <ChevronRight className="w-5 h-5 text-gray-400" />
            )}
          </button>
          
          {expandedSections[getSectionKey('meta')] && (
            <div className="border-t border-gray-200 p-4">
              <pre className="text-sm bg-gray-50 p-3 rounded-lg overflow-x-auto">
                {JSON.stringify(dimension.meta, null, 2)}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Case Editor Component
interface CaseEditorProps {
  caseObj?: Dimension['case'];
  onUpdate: (caseObj?: Dimension['case']) => void;
}

function CaseEditor({ caseObj, onUpdate }: CaseEditorProps) {
  const uid = useId();
  const [whenList, setWhenList] = useState<CaseWhen[]>(caseObj?.when || []);
  const [elseLabel, setElseLabel] = useState<CaseElse>(caseObj?.else || { label: 'Unknown' });
  const [useDynamicLabel, setUseDynamicLabel] = useState(false);

  const addWhen = () => {
    setWhenList(prev => [...prev, { sql: '', label: '' }]);
  };

  const updateWhen = (index: number, updates: Partial<CaseWhen>) => {
    setWhenList(prev => prev.map((w, i) => i === index ? { ...w, ...updates } : w));
  };

  const removeWhen = (index: number) => {
    setWhenList(prev => prev.filter((_, i) => i !== index));
  };

  const saveCase = () => {
    onUpdate({ when: whenList, else: elseLabel });
  };

  const clearCase = () => {
    setWhenList([]);
    setElseLabel({ label: 'Unknown' });
    onUpdate(undefined);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={useDynamicLabel}
            onChange={(e) => setUseDynamicLabel(e.target.checked)}
            className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <span className="text-sm font-medium text-gray-700">Use Dynamic Labels (SQL)</span>
        </label>
      </div>

      {/* When Conditions */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="font-medium">When Conditions</span>
          <button
            onClick={addWhen}
            className="bg-blue-500 hover:bg-blue-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
            aria-label="Add when condition"
            title="Add Condition"
          >
            <Plus className="w-3 h-3" />
            Add Condition
          </button>
        </div>
        
        {whenList.map((w, index) => (
          <div key={index} className="bg-blue-50 border border-blue-200 p-3 rounded-lg flex gap-3">
            <div className="flex-1">
              <label htmlFor={`${uid}-when-sql-${index}`} className="block text-xs font-medium text-gray-500 mb-1">SQL Condition</label>
              <input
                id={`${uid}-when-sql-${index}`}
                type="text"
                value={w.sql}
                onChange={(e) => updateWhen(index, { sql: e.target.value })}
                className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono"
              />
            </div>
            <div className="flex-1">
              <label htmlFor={`${uid}-when-label-${index}`} className="block text-xs font-medium text-gray-500 mb-1">Label</label>
              {useDynamicLabel ? (
                <input
                  id={`${uid}-when-label-${index}`}
                  type="text"
                  value={typeof w.label === 'string' ? w.label : w.label.sql}
                  onChange={(e) => updateWhen(index, { label: { sql: e.target.value } })}
                  className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono"
                />
              ) : (
                <input
                  id={`${uid}-when-label-${index}`}
                  type="text"
                  value={typeof w.label === 'string' ? w.label : ''}
                  onChange={(e) => updateWhen(index, { label: e.target.value })}
                  className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
                />
              )}
            </div>
            <button
              onClick={() => removeWhen(index)}
              className="text-red-500 hover:text-red-700 p-2"
              aria-label={`Remove when condition ${index + 1}`}
              title={`Remove condition ${index + 1}`}
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        ))}
      </div>

      {/* Else */}
      <div className="space-y-2">
        <span className="font-medium">Else Label</span>
        {useDynamicLabel ? (
          <input
            id={`${uid}-else-label`}
            type="text"
            value={typeof elseLabel.label === 'string' ? elseLabel.label : elseLabel.label.sql}
            onChange={(e) => setElseLabel({ label: { sql: e.target.value } })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono"
          />
        ) : (
          <input
            id={`${uid}-else-label`}
            type="text"
            value={typeof elseLabel.label === 'string' ? elseLabel.label : ''}
            onChange={(e) => setElseLabel({ label: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500"
          />
        )}
      </div>

      <div className="flex gap-3 pt-3 border-t border-gray-200">
        <button
          onClick={saveCase}
          className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Apply Case
        </button>
        <button
          onClick={clearCase}
          className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Clear Case
        </button>
      </div>
    </div>
  );
}

// Granularity Form Component (for adding new)
interface GranularityFormProps {
  onAdd: (granularity: Omit<Granularity, 'id'>) => void;
}

function GranularityForm({ onAdd }: GranularityFormProps) {
  const uid = useId();
  const [name, setName] = useState('');
  const [intervalValue, setIntervalValue] = useState('');
  const [intervalUnit, setIntervalUnit] = useState('day');
  const [offsetValue, setOffsetValue] = useState('');
  const [offsetUnit, setOffsetUnit] = useState('day');
  const [origin, setOrigin] = useState('');
  const [title, setTitle] = useState('');

  const handleAdd = () => {
    if (!name || !intervalValue) return;
    const interval = `${intervalValue} ${intervalUnit}`;
    const offset = offsetValue ? `${offsetValue} ${offsetUnit}` : undefined;
    onAdd({ name, interval, offset, origin, title });
    setName('');
    setIntervalValue('');
    setIntervalUnit('day');
    setOffsetValue('');
    setOffsetUnit('day');
    setOrigin('');
    setTitle('');
  };

  return (
    <div className="space-y-3 bg-gray-50 p-4 rounded-lg">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
            <label htmlFor={`${uid}-name`} className="block text-xs font-medium text-gray-500 mb-1">Name *</label>
          <input
            id={`${uid}-name`}
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
            placeholder="quarter_hour"
          />
        </div>
        <div>
          <label htmlFor={`${uid}-title`} className="block text-xs font-medium text-gray-500 mb-1">Title</label>
          <input
            id={`${uid}-title`}
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
            placeholder="Human-readable title"
          />
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
            <label htmlFor={`${uid}-interval-value`} className="block text-xs font-medium text-gray-500 mb-1">Interval *</label>
          <div className="flex gap-2">
            <input
              id={`${uid}-interval-value`}
              type="number"
              value={intervalValue}
              onChange={(e) => setIntervalValue(e.target.value)}
              className="flex-1 p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
              placeholder="15"
            />
            <select
              id={`${uid}-interval-unit`}
              value={intervalUnit}
              onChange={(e) => setIntervalUnit(e.target.value)}
              className="p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
            >
              {timeUnits.map(unit => (
                <option key={unit} value={unit}>{unit}</option>
              ))}
            </select>
          </div>
        </div>
        <div>
          <label htmlFor={`${uid}-offset-value`} className="block text-xs font-medium text-gray-500 mb-1">Offset</label>
          <div className="flex gap-2">
            <input
              id={`${uid}-offset-value`}
              type="number"
              value={offsetValue}
              onChange={(e) => setOffsetValue(e.target.value)}
              className="flex-1 p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
              placeholder="-1"
            />
            <select
              id={`${uid}-offset-unit`}
              value={offsetUnit}
              onChange={(e) => setOffsetUnit(e.target.value)}
              className="p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
            >
              {timeUnits.map(unit => (
                <option key={unit} value={unit}>{unit}</option>
              ))}
            </select>
          </div>
        </div>
      </div>
      
      <div>
        <label htmlFor={`${uid}-origin`} className="block text-xs font-medium text-gray-500 mb-1">Origin (ISO 8601)</label>
        <input
          id={`${uid}-origin`}
          type="text"
          value={origin}
          onChange={(e) => setOrigin(e.target.value)}
          className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500"
          placeholder="2024-01-01T00:00:00.000Z"
        />
      </div>
      
      <button
        onClick={handleAdd}
        disabled={!name || !intervalValue}
        className="bg-green-500 hover:bg-green-600 disabled:opacity-50 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        aria-label="Add granularity"
        title="Add Granularity"
      >
        <Plus className="w-4 h-4" />
        Add Granularity
      </button>
    </div>
  );
}

// Granularity Editor Component
interface GranularityEditorProps {
  granularity: Granularity;
  onUpdate: (updates: Partial<Granularity>) => void;
  onRemove: () => void;
}

function GranularityEditor({ granularity, onUpdate, onRemove }: GranularityEditorProps) {
  const uid = useId();
  const [editData, setEditData] = useState(granularity);
  const [isEditing, setIsEditing] = useState(false);

  const saveEdit = () => {
    onUpdate(editData);
    setIsEditing(false);
  };

  const cancelEdit = () => {
    setEditData(granularity);
    setIsEditing(false);
  };

  return (
    <div className="bg-green-50 border border-green-200 p-4 rounded-lg">
      {isEditing ? (
        <div className="space-y-3">
          <div>
            <label htmlFor={`${uid}-gran-name`} className="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              id={`${uid}-gran-name`}
              type="text"
              value={editData.name}
              onChange={(e) => setEditData({ ...editData, name: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500"
            />
          </div>
          <div>
            <label htmlFor={`${uid}-gran-interval`} className="block text-sm font-medium text-gray-700 mb-1">Interval</label>
            <input
              id={`${uid}-gran-interval`}
              type="text"
              value={editData.interval}
              onChange={(e) => setEditData({ ...editData, interval: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500"
            />
          </div>
          <div>
            <label htmlFor={`${uid}-gran-offset`} className="block text-sm font-medium text-gray-700 mb-1">Offset</label>
            <input
              id={`${uid}-gran-offset`}
              type="text"
              value={editData.offset || ''}
              onChange={(e) => setEditData({ ...editData, offset: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500"
            />
          </div>
          <div>
            <label htmlFor={`${uid}-gran-origin`} className="block text-sm font-medium text-gray-700 mb-1">Origin</label>
            <input
              id={`${uid}-gran-origin`}
              type="text"
              value={editData.origin || ''}
              onChange={(e) => setEditData({ ...editData, origin: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500"
            />
          </div>
          <div>
            <label htmlFor={`${uid}-gran-title`} className="block text-sm font-medium text-gray-700 mb-1">Title</label>
            <input
              id={`${uid}-gran-title`}
              type="text"
              value={editData.title || ''}
              onChange={(e) => setEditData({ ...editData, title: e.target.value })}
              className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 focus:border-green-500"
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={saveEdit}
              className="bg-green-500 hover:bg-green-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
            >
              <Save className="w-3 h-3" />
              Save
            </button>
            <button
              onClick={cancelEdit}
              className="bg-gray-500 hover:bg-gray-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
            >
              <X className="w-3 h-3" />
              Cancel
            </button>
          </div>
        </div>
      ) : (
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <span className="font-medium block mb-1">{granularity.name}</span>
            <p className="text-sm text-gray-600">Interval: {granularity.interval}</p>
            {granularity.offset && <p className="text-sm text-gray-600">Offset: {granularity.offset}</p>}
            {granularity.origin && <p className="text-sm text-gray-600">Origin: {granularity.origin}</p>}
            {granularity.title && <p className="text-sm text-gray-600">Title: {granularity.title}</p>}
          </div>
          <div className="flex gap-1">
            <button
              onClick={() => setIsEditing(true)}
              className="text-green-600 hover:text-green-800 p-1 rounded transition-colors"
              aria-label="Edit granularity"
              title="Edit"
            >
              <Edit3 className="w-3 h-3" />
            </button>
            <button
              onClick={onRemove}
              className="text-red-500 hover:text-red-700 p-1 rounded transition-colors"
              aria-label="Remove granularity"
              title="Remove"
            >
              <Trash2 className="w-3 h-3" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}