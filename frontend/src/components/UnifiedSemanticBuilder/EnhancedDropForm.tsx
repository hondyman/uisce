import { useState, useEffect } from 'react';
import ActionButton from '../ui/ActionButton';
import * as TablerIcons from '@tabler/icons-react';
import SqlMonacoEditor from '../SqlMonacoEditor';
import { getTableIdFromVal } from '../../utils/tableHelpers';

interface DroppedItem {
  type: 'dimension' | 'measure' | 'filter' | 'join';
  isCore: boolean;
}

interface CoreDimension {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  sourceTable?: string;
  sourceColumn?: string;
}

interface CoreMeasure {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  format?: string;
  aggregationType?: string;
}

interface CoreFilter {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  defaultValue?: string;
}

interface EnhancedDropFormProps {
  droppedItem: DroppedItem;
  coreDimensions?: CoreDimension[];
  coreMeasures?: CoreMeasure[];
  coreFilters?: CoreFilter[];
  onSubmit: (data: any) => void;
  onCancel: () => void;
}

const EnhancedDropForm: React.FC<EnhancedDropFormProps> = ({
  droppedItem,
  coreDimensions = [],
  coreMeasures = [],
  coreFilters = [],
  onSubmit,
  onCancel
}) => {
  const [selectedCoreItem, setSelectedCoreItem] = useState<string>('');
  const [formData, setFormData] = useState<any>({});
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({
    basic: true,
    advanced: true,
    sql: true
  });

  // Get core items based on type
  const getCoreItems = () => {
    switch (droppedItem.type) {
      case 'dimension': return coreDimensions;
      case 'measure': return coreMeasures;
      case 'filter': return coreFilters;
      default: return [];
    }
  };

  const coreItems = getCoreItems();

  // Load core item data when selected
  useEffect(() => {
    if (selectedCoreItem && droppedItem.isCore) {
      const coreItem = coreItems.find(item => item.name === selectedCoreItem);
      if (coreItem) {
        setFormData({ ...coreItem });
      }
    }
  }, [selectedCoreItem, droppedItem.isCore, coreItems]);

  const handleFieldChange = (field: string, value: any) => {
    setFormData((prev: any) => ({ ...prev, [field]: value }));
  };

  const toggleSection = (section: 'basic' | 'advanced' | 'sql') => {
    setExpandedSections(prev => ({ ...prev, [section]: !prev[section] }));
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    
    if (droppedItem.isCore && selectedCoreItem) {
      // For core items, only include changed values
      const coreItem = coreItems.find(item => item.name === selectedCoreItem);
      if (coreItem) {
        const changes: any = {};
        Object.keys(formData).forEach(key => {
          if (formData[key] !== coreItem[key as keyof typeof coreItem]) {
            changes[key] = formData[key];
          }
        });
        onSubmit({ ...changes, baseName: selectedCoreItem, isOverride: true });
      }
    } else {
      // For custom items, include all data but serialize any table objects
      const serialize = (fd: any) => {
        if (!fd) return fd;
        const s = { ...fd };
        if (s.sourceTable) s.sourceTable = getTableIdFromVal(s.sourceTable);
        if (s.leftTable) s.leftTable = getTableIdFromVal(s.leftTable);
        if (s.rightTable) s.rightTable = getTableIdFromVal(s.rightTable);
        if (s.joins && Array.isArray(s.joins)) s.joins = s.joins.map((j: any) => ({ ...j, leftTable: getTableIdFromVal(j.leftTable), rightTable: getTableIdFromVal(j.rightTable) }));
        return s;
      };
      onSubmit({ ...serialize(formData), isOverride: false });
    }
  };

  const renderBasicFields = () => (
    <div className="form-section">
      <div 
        className="section-header" 
        onClick={() => toggleSection('basic')}
      >
        <TablerIcons.IconChevronDown 
          size={16} 
          style={{ 
            transform: expandedSections.basic ? 'rotate(0deg)' : 'rotate(-90deg)',
            transition: 'transform 0.2s'
          }} 
        />
        <span>Basic Properties</span>
      </div>
      
      {expandedSections.basic && (
        <div className="section-content">
          {droppedItem.isCore ? (
            <div className="form-group">
              <label htmlFor="coreSelect">Select {droppedItem.type}:</label>
              <select
                id="coreSelect"
                value={selectedCoreItem}
                onChange={(e) => setSelectedCoreItem(e.target.value)}
                required
              >
                <option value="">Choose a core {droppedItem.type}...</option>
                {coreItems.map(item => (
                  <option key={item.name} value={item.name}>
                    {item.title || item.name}
                  </option>
                ))}
              </select>
            </div>
          ) : (
            <div className="form-group">
              <label htmlFor="name">Name:</label>
              <input
                type="text"
                id="name"
                value={formData.name || ''}
                onChange={(e) => handleFieldChange('name', e.target.value)}
                required
                placeholder={`Enter ${droppedItem.type} name`}
              />
            </div>
          )}

          <div className="form-group">
            <label htmlFor="title">Display Title:</label>
            <input
              type="text"
              id="title"
              value={formData.title || ''}
              onChange={(e) => handleFieldChange('title', e.target.value)}
              placeholder="Human-readable title"
            />
          </div>

          <div className="form-group">
            <label htmlFor="description">Description:</label>
            <textarea
              id="description"
              value={formData.description || ''}
              onChange={(e) => handleFieldChange('description', e.target.value)}
              placeholder="Optional description"
              rows={2}
            />
          </div>

          {droppedItem.type !== 'join' && (
            <div className="form-group">
              <label htmlFor="type">Data Type:</label>
              <select
                id="type"
                value={formData.type || 'string'}
                onChange={(e) => handleFieldChange('type', e.target.value)}
              >
                <option value="string">String</option>
                <option value="number">Number</option>
                <option value="boolean">Boolean</option>
                <option value="time">Time</option>
                <option value="date">Date</option>
              </select>
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderAdvancedFields = () => (
    <div className="form-section">
      <div 
        className="section-header" 
        onClick={() => toggleSection('advanced')}
      >
        <TablerIcons.IconChevronDown 
          size={16} 
          style={{ 
            transform: expandedSections.advanced ? 'rotate(0deg)' : 'rotate(-90deg)',
            transition: 'transform 0.2s'
          }} 
        />
        <span>Advanced Properties</span>
      </div>
      
      {expandedSections.advanced && (
        <div className="section-content">
          <div className="form-group">
            <label htmlFor="sourceTable">Source Table:</label>
            <input
              type="text"
              id="sourceTable"
              value={formData.sourceTable || ''}
              onChange={(e) => handleFieldChange('sourceTable', e.target.value)}
              placeholder="Source table name"
            />
          </div>

          <div className="form-group">
            <label htmlFor="sourceColumn">Source Column:</label>
            <input
              type="text"
              id="sourceColumn"
              value={formData.sourceColumn || ''}
              onChange={(e) => handleFieldChange('sourceColumn', e.target.value)}
              placeholder="Source column name"
            />
          </div>

          {droppedItem.type === 'measure' && (
            <>
              <div className="form-group">
                <label htmlFor="format">Number Format:</label>
                <input
                  type="text"
                  id="format"
                  value={formData.format || ''}
                  onChange={(e) => handleFieldChange('format', e.target.value)}
                  placeholder="e.g., #,##0.00"
                />
              </div>

              <div className="form-group">
                <label htmlFor="aggregationType">Aggregation:</label>
                <select
                  id="aggregationType"
                  value={formData.aggregationType || 'sum'}
                  onChange={(e) => handleFieldChange('aggregationType', e.target.value)}
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

          {droppedItem.type === 'filter' && (
            <div className="form-group">
              <label htmlFor="defaultValue">Default Value:</label>
              <input
                type="text"
                id="defaultValue"
                value={formData.defaultValue || ''}
                onChange={(e) => handleFieldChange('defaultValue', e.target.value)}
                placeholder="Default filter value"
              />
            </div>
          )}
        </div>
      )}
    </div>
  );

  const renderSqlFields = () => (
    <div className="form-section">
      <div 
        className="section-header" 
        onClick={() => toggleSection('sql')}
      >
        <TablerIcons.IconChevronDown 
          size={16} 
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
            <label htmlFor="sql">SQL Expression:</label>
            <SqlMonacoEditor
              value={formData.sql || ''}
              onChange={(value) => handleFieldChange('sql', value)}
              placeholder={`SQL expression for ${droppedItem.type}`}
              height={120}
            />
            <small className="field-hint">
              Use column references like ${'{'}table.column{'}'} or SQL functions
            </small>
          </div>
        </div>
      )}
    </div>
  );

  return (
    <div className="drop-form-overlay">
      <div className="enhanced-drop-form-modal">
        <div className="modal-header">
          <h3>
            {droppedItem.isCore ? 'Override' : 'Add'} {droppedItem.type.charAt(0).toUpperCase() + droppedItem.type.slice(1)}
          </h3>
          <button type="button" onClick={onCancel} className="close-button" title="Close form">
            <TablerIcons.IconX size={20} />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="enhanced-form">
          <div className="form-body">
            {renderBasicFields()}
            {renderAdvancedFields()}
            {renderSqlFields()}
          </div>

          <div className="form-actions">
              <ActionButton variant="secondary" onClick={onCancel}>
                Cancel
              </ActionButton>
              <ActionButton variant="primary" type="submit" pending={droppedItem.isCore && !selectedCoreItem}>
                {droppedItem.isCore ? 'Override' : 'Add'} {droppedItem.type}
              </ActionButton>
          </div>
        </form>
      </div>
    </div>
  );
};

export default EnhancedDropForm;
