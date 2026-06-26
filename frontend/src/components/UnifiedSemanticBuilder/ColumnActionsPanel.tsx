import React, { useState } from 'react';
import { SelectedColumn, BusinessTerm } from './types';
import { getDataTypeIcon } from './utils';
import { IconStack3, IconChartBar, IconFilter, IconPlus, IconX, IconTrash } from './icons';
import CoreCustomIndicator from '../CoreCustomIndicator'; // FIX: Corrected to default import

interface ColumnActionsPanelProps {
  selectedColumn: SelectedColumn;
  addDimension: (tableName: string, column: any) => void;
  addMeasure: (tableName: string, column: any, additionalColumns?: any[]) => void;
  addFilter: (tableName: string, column: any) => void;
  getBusinessTermForColumn: (nodeId: string, columnName: string) => BusinessTerm | undefined;
}

const ColumnActionsPanel: React.FC<ColumnActionsPanelProps> = ({
  selectedColumn,
  addDimension,
  addMeasure,
  addFilter,
  getBusinessTermForColumn,
}) => {
  const { nodeId, tableName, column } = selectedColumn;
  const businessTerm = getBusinessTermForColumn(nodeId, column.name);
  
  const [additionalMeasureColumns, setAdditionalMeasureColumns] = useState<any[]>([]);
  const [showMultiColumnMode, setShowMultiColumnMode] = useState(false);
  const [businessTermExpanded, setBusinessTermExpanded] = useState(false);

  const addAdditionalColumn = () => {
    setAdditionalMeasureColumns([...additionalMeasureColumns, { 
      id: Date.now(), 
      tableName: '', 
      columnName: '', 
      operation: 'SUM' 
    }]);
  };

  const removeAdditionalColumn = (index: number) => {
    setAdditionalMeasureColumns(additionalMeasureColumns.filter((_, i) => i !== index));
  };

  const updateAdditionalColumn = (index: number, field: string, value: string) => {
    const updated = [...additionalMeasureColumns];
    updated[index] = { ...updated[index], [field]: value };
    setAdditionalMeasureColumns(updated);
  };

  const clearAllAdditionalColumns = () => {
    setAdditionalMeasureColumns([]);
    setShowMultiColumnMode(false);
  };

  const handleCreateMeasure = () => {
    if (showMultiColumnMode && additionalMeasureColumns.length > 0) {
      const validAdditionalColumns = additionalMeasureColumns.filter(
        col => col.tableName && col.columnName
      );
      addMeasure(tableName, column, validAdditionalColumns);
    } else {
      addMeasure(tableName, column);
    }
    clearAllAdditionalColumns();
  };

  return (
    <div className="column-actions-panel enhanced">
      <div className="panel-header enhanced">
        <div className="column-main-info">
          <div className="column-title-row">
            <div className="column-title-with-indicator">
              <h3 className="column-full-name">
                <span className="table-name-part">{tableName}</span>
                <span className="separator">.</span>
                <span className="column-name-part">{column.name}</span>
              </h3>
              {/* FIX: Removed invalid props */}
              <CoreCustomIndicator isCore={column.isCore} />
            </div>
          </div>
          
          <div className="column-metadata-row">
            <div className="data-type-container">
              <span className="data-type-icon">{getDataTypeIcon(column.type)}</span>
              <span className="data-type-text">{column.type}</span>
            </div>
            
            {businessTerm && (
              <div 
                className="business-term-indicator clickable"
                onClick={() => setBusinessTermExpanded(!businessTermExpanded)}
                title="Click to expand business term details"
              >
                <span className="business-term-icon">💼</span>
                <span className="business-term-text">Business Term</span>
                <span className={`expand-arrow ${businessTermExpanded ? 'expanded' : ''}`}>▼</span>
              </div>
            )}
          </div>
        </div>
      </div>

      {businessTerm && businessTermExpanded && (
        <div className="business-term-details-section expanded">
          <div className="business-term-header-with-indicator">
            <div className="business-term-name">{businessTerm.node_name}</div>
            {/* FIX: Removed invalid props */}
            <CoreCustomIndicator isCore={businessTerm.isCore} />
          </div>
          {businessTerm.description && (
            <div className="business-term-description">{businessTerm.description}</div>
          )}
        </div>
      )}

      {showMultiColumnMode && (
        <div className="multi-column-section">
          <div className="multi-column-header">
            <h4>Additional Columns for Measure</h4>
            <div className="multi-column-controls">
              <button onClick={addAdditionalColumn} className="add-column-btn">
                <IconPlus size={14} /> Add Column
              </button>
              <button onClick={clearAllAdditionalColumns} className="clear-columns-btn">
                <IconTrash size={14} /> Clear All
              </button>
            </div>
          </div>
          
          <div className="additional-columns-list">
            {additionalMeasureColumns.map((col, index) => (
              <div key={col.id} className="additional-column-row">
                <input type="text" placeholder="Table name" value={col.tableName} onChange={(e) => updateAdditionalColumn(index, 'tableName', e.target.value)} className="column-input" />
                <input type="text" placeholder="Column name" value={col.columnName} onChange={(e) => updateAdditionalColumn(index, 'columnName', e.target.value)} className="column-input" />
                <select value={col.operation} onChange={(e) => updateAdditionalColumn(index, 'operation', e.target.value)} className="operation-select">
                  <option value="SUM">SUM</option>
                  <option value="AVG">AVG</option>
                  <option value="COUNT">COUNT</option>
                  <option value="MIN">MIN</option>
                  <option value="MAX">MAX</option>
                </select>
                <button onClick={() => removeAdditionalColumn(index)} className="remove-column-btn">
                  <IconX size={14} />
                </button>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="panel-actions enhanced">
        <button onClick={() => addDimension(tableName, column)} className="action-btn dimension">
          <IconStack3 size={16} />
          <span>Add as Dimension</span>
        </button>
        
        <div className="measure-button-group">
          <button onClick={handleCreateMeasure} className="action-btn measure">
            <IconChartBar size={16} />
            <span>{showMultiColumnMode ? 'Create Multi-Column Measure' : 'Add as Measure'}</span>
          </button>
          <button onClick={() => setShowMultiColumnMode(!showMultiColumnMode)} className={`measure-mode-toggle ${showMultiColumnMode ? 'active' : ''}`} title={showMultiColumnMode ? 'Switch to single column mode' : 'Enable multi-column measure mode'}>
            <IconPlus size={14} />
          </button>
        </div>
        
        <button onClick={() => addFilter(tableName, column)} className="action-btn filter">
          <IconFilter size={16} />
          <span>Add as Filter</span>
        </button>
      </div>

      {showMultiColumnMode && (
        <div className="selection-summary">
          <div className="primary-column">
            <strong>Primary:</strong> {tableName}.{column.name}
            {/* FIX: Removed invalid props */}
            <CoreCustomIndicator isCore={column.isCore} />
          </div>
          {additionalMeasureColumns.length > 0 && (
            <div className="additional-columns-summary">
              <strong>Additional:</strong>
              <ul>
              {additionalMeasureColumns.map((col) => (
                  <li key={col.id}>
                    {col.tableName}.{col.columnName} ({col.operation})
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ColumnActionsPanel;
