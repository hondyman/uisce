import { useState, useMemo } from 'react';
import { devError } from '../../utils/devLogger';
import { Node as FlowNode, Edge } from 'reactflow';

export interface ExportOptions {
  format: 'csv' | 'json' | 'xml';
  delimiter: string;
  includeRelationships: boolean;
  selectedSchemas: string[];
  selectedTables: string[];
  exportScope: 'all' | 'schemas' | 'tables';
  includeIndexes: boolean;
  includeComments: boolean;
}

export interface EnhancedExportOverlayProps {
  nodes: FlowNode[];
  edges: Edge[];
  onExport: (options: ExportOptions) => Promise<void>;
  onCancel: () => void;
}

function getNodeSchema(node: FlowNode): string {
  return node.data?.schema || 'public';
}

export const EnhancedExportOverlay: React.FC<EnhancedExportOverlayProps> = ({ nodes, edges, onExport, onCancel }) => {
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'csv',
    delimiter: ',',
    includeRelationships: false,
    selectedSchemas: [],
    selectedTables: [],
    exportScope: 'all',
    includeIndexes: false,
    includeComments: false,
  });

  const [isExporting, setIsExporting] = useState(false);
  const [tableSearch, setTableSearch] = useState('');
  const [showSelectedTables, setShowSelectedTables] = useState(false);

  const schemaGroups = useMemo(() => {
    const groups: { [schema: string]: FlowNode[] } = {};
    nodes.forEach(node => {
      const schema = getNodeSchema(node);
      if (!groups[schema]) groups[schema] = [];
      groups[schema].push(node);
    });
    return groups;
  }, [nodes]);

  const availableSchemas = Object.keys(schemaGroups);
  const totalTables = nodes.length;
  const totalSchemas = availableSchemas.length;
  const totalColumns = nodes.reduce((sum, node) => sum + (node.data?.columns?.length || 0), 0);
  const totalRelations = edges.length;
  const estimatedSize = (nodes.length * 0.05).toFixed(1);

  const handleOptionChange = (key: keyof ExportOptions, value: any) => {
    setExportOptions(prev => ({ ...prev, [key]: value }));
  };

  const handleExport = async () => {
    setIsExporting(true);
    try {
      await onExport(exportOptions);
      onCancel();
    } catch (error) {
      try { devError('Export failed:', error); } catch {}
    } finally {
      setIsExporting(false);
    }
  };

  const filteredTablesBySchema = useMemo(() => {
    if (!tableSearch.trim()) return schemaGroups;

    const filtered: { [schema: string]: FlowNode[] } = {};
    Object.entries(schemaGroups).forEach(([schema, tables]) => {
      const matchingTables = tables.filter(table =>
        ((table.data?.label as string) || table.id).toLowerCase().includes(tableSearch.toLowerCase()) ||
        schema.toLowerCase().includes(tableSearch.toLowerCase())
      );
      if (matchingTables.length > 0) filtered[schema] = matchingTables;
    });
    return filtered;
  }, [schemaGroups, tableSearch]);

  const selectedTableDetails = useMemo(() => {
    const selected: Array<{ id: string; name: string; schema: string; columns: number }> = [];
    Object.entries(schemaGroups).forEach(([schema, tables]) => {
      tables.forEach(table => {
        if (exportOptions.selectedTables.includes(table.id)) {
          selected.push({
            id: table.id,
            name: (table.data?.label as string) || table.id,
            schema,
            columns: table.data?.columns?.length || 0,
          });
        }
      });
    });
    return selected;
  }, [schemaGroups, exportOptions.selectedTables]);

  const canExport =
    exportOptions.exportScope === 'all' ||
    (exportOptions.exportScope === 'schemas' && exportOptions.selectedSchemas.length > 0) ||
    (exportOptions.exportScope === 'tables' && exportOptions.selectedTables.length > 0);

  return (
    <div className="export-modal-root visible">
      <div className="export-modal-dialog visible">
        <div className="export-modal-header">
          <div className="export-modal-header-inner">
            <div className="export-modal-header-left">
              <div className="export-modal-icon">📥</div>
              <div>
                <h1 className="export-modal-title">Export Data Catalog</h1>
                <p className="export-modal-sub">Configure and download your database schema</p>
              </div>
            </div>
            <div className="export-modal-header-right">
              <div className="export-ready">Ready to Export</div>
              <div className="export-count">{totalTables} Tables • {totalSchemas} Schemas</div>
            </div>
          </div>
        </div>

        <div className="export-modal-body">
          <div className="format-grid">
            {(['csv', 'json', 'xml'] as const).map((format) => (
              <label key={format} className="format-option">
                <input
                  type="radio"
                  name="format"
                  value={format}
                  checked={exportOptions.format === format}
                  onChange={() => handleOptionChange('format', format)}
                  className="visually-hidden"
                />
                <div className={exportOptions.format === format ? 'format-card selected' : 'format-card'}>
                  <div className="format-icon">{format === 'csv' ? '📄' : format === 'json' ? '🧩' : '📦'}</div>
                  <div className="format-name">{format.toUpperCase()}</div>
                  <div className="format-sub">{format === 'csv' ? 'Excel Ready' : format === 'json' ? 'API Friendly' : 'Enterprise'}</div>
                </div>
              </label>
            ))}
          </div>

          {exportOptions.format === 'csv' && (
            <div className="csv-box">
              <h3 className="csv-title">Delimiter</h3>
              <div className="delimiter-grid">
                {[
                  { value: ',', label: 'Comma (,)' },
                  { value: ';', label: 'Semicolon (;)' },
                  { value: '\t', label: 'Tab' },
                  { value: '|', label: 'Pipe (|)' },
                ].map((delimiter) => (
                  <label key={delimiter.value} className="delimiter-option">
                    <input
                      type="radio"
                      name="delimiter"
                      value={delimiter.value}
                      checked={exportOptions.delimiter === delimiter.value}
                      onChange={() => handleOptionChange('delimiter', delimiter.value)}
                      className="visually-hidden"
                    />
                    <div className={exportOptions.delimiter === delimiter.value ? 'delimiter-card selected' : 'delimiter-card'}>
                      {delimiter.label}
                    </div>
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="export-section">
            <h2 className="section-title">Data Scope</h2>
            <div className="scope-list">
              {[
                { value: 'all', icon: '🌐', title: 'Export Everything', subtitle: 'All schemas, tables, and relationships' },
                { value: 'schemas', icon: '📁', title: 'Select Schemas', subtitle: 'Choose specific database schemas' },
                { value: 'tables', icon: '📊', title: 'Select Tables', subtitle: 'Pick individual tables' },
              ].map(scope => (
                <label key={scope.value} className={exportOptions.exportScope === scope.value ? 'scope-option selected' : 'scope-option'}>
                  <input
                    type="radio"
                    name="scope"
                    value={scope.value}
                    checked={exportOptions.exportScope === scope.value}
                    onChange={() => handleOptionChange('exportScope', scope.value as any)}
                    className="visually-hidden"
                  />
                  <span className="scope-icon">{scope.icon}</span>
                  <div>
                    <div className="scope-title">{scope.title}</div>
                    <div className="scope-sub">{scope.subtitle}</div>
                  </div>
                </label>
              ))}
            </div>
          </div>

          <div className="export-section">
            <h2 className="section-title">Advanced Options</h2>
            <div className="advanced-list">
              {[
                { key: 'includeRelationships', icon: '🔗', title: 'Include Relationships', subtitle: 'Export foreign keys and constraints' },
                { key: 'includeIndexes', icon: '🔑', title: 'Include Indexes', subtitle: 'Export database indexes information' },
              ].map(option => (
                <label key={option.key} className="advanced-option">
                  <div className="advanced-left">
                    <span className="advanced-icon">{option.icon}</span>
                    <div>
                      <div className="advanced-title">{option.title}</div>
                      <div className="advanced-sub">{option.subtitle}</div>
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    checked={exportOptions[option.key as keyof ExportOptions] as boolean}
                    onChange={(e) => handleOptionChange(option.key as keyof ExportOptions, e.target.checked)}
                    className="visually-hidden-checkbox"
                  />
                </label>
              ))}
            </div>
          </div>

          <div>
            <div className="preview-box">
              <h2 className="section-title preview-title">Export Preview</h2>
              <div className="preview-stats">
                {[
                  { icon: '🗄️', value: totalSchemas, label: 'Schemas' },
                  { icon: '📊', value: totalTables, label: 'Tables' },
                  { icon: '📋', value: totalColumns, label: 'Columns' },
                  { icon: '🔗', value: totalRelations, label: 'Relations' },
                ].map((stat, index) => (
                  <div key={index} className="preview-stat-card">
                    <div className="preview-stat-icon">{stat.icon}</div>
                    <div className="preview-stat-value">{stat.value}</div>
                    <div className="preview-stat-label">{stat.label}</div>
                  </div>
                ))}
              </div>

              <div className="file-output-box">
                <div className="file-output-header">
                  <h4 className="file-output-title">File Output</h4>
                  <span className="file-output-ready">Ready to download</span>
                </div>
                <div className="file-output-body">
                  <span className="file-output-icon">📄</span>
                  <div className="file-output-main">
                    <div className="file-output-name">data-catalog-{new Date().toISOString().split('T')[0]}.{exportOptions.format}</div>
                    <div className="file-output-size">Estimated size: ~{estimatedSize} MB</div>
                  </div>
                  <div className="file-output-format">{exportOptions.format.toUpperCase()} Format</div>
                </div>
              </div>
            </div>

            {exportOptions.exportScope === 'schemas' && (
              <div className="schema-box">
                <div className="schema-header">
                  <h3 className="schema-title">Select Schemas</h3>
                  <div className="schema-actions">
                    <button className="chip action" onClick={() => handleOptionChange('selectedSchemas', availableSchemas)}>✅ Select All</button>
                    <button className="chip" onClick={() => handleOptionChange('selectedSchemas', [])}>❌ Clear</button>
                  </div>
                </div>
                <div className="schema-grid">
                  {availableSchemas.map(schema => (
                    <label key={schema} className="schema-row">
                      <div className="schema-row-left">
                        <input
                          type="checkbox"
                          checked={exportOptions.selectedSchemas.includes(schema)}
                          onChange={(e) => {
                            const newSelected = e.target.checked
                              ? [...exportOptions.selectedSchemas, schema]
                              : exportOptions.selectedSchemas.filter(s => s !== schema);
                            handleOptionChange('selectedSchemas', newSelected);
                          }}
                        />
                        <span className="schema-name">{schema}</span>
                      </div>
                      <span className="schema-count">{schemaGroups[schema]?.length || 0} tables</span>
                    </label>
                  ))}
                </div>
              </div>
            )}

            {exportOptions.exportScope === 'tables' && (
              <div className="table-box">
                <div className="table-header">
                  <h3 className="table-title">Select Tables</h3>
                  <div className="table-actions">
                    <button className={showSelectedTables ? 'chip action' : 'chip'} onClick={() => setShowSelectedTables(!showSelectedTables)}>{showSelectedTables ? '📋 Show All' : `✅ Show Selected (${exportOptions.selectedTables.length})`}</button>
                    <button className="chip" onClick={() => handleOptionChange('selectedTables', nodes.map(n => n.id))}>✅ Select All</button>
                    <button className="chip" onClick={() => handleOptionChange('selectedTables', [])}>❌ Clear</button>
                  </div>
                </div>

                {!showSelectedTables && (
                  <div className="search-row">
                    <input
                      type="text"
                      placeholder="🔍 Search tables or schemas..."
                      value={tableSearch}
                      onChange={(e) => setTableSearch(e.target.value)}
                      className="search-input"
                    />
                  </div>
                )}

                <div className="table-list">
                  {showSelectedTables ? (
                    selectedTableDetails.length > 0 ? (
                      <div className="selected-tables-box">
                        <div className="selected-tables-header">Selected Tables ({selectedTableDetails.length})</div>
                        {selectedTableDetails.map((table) => (
                          <div key={table.id} className="selected-table-row">
                            <div className="selected-table-left">
                              <button className="remove-btn" onClick={() => { const newSelected = exportOptions.selectedTables.filter(t => t !== table.id); handleOptionChange('selectedTables', newSelected); }}>×</button>
                              <div>
                                <div className="selected-table-name">{table.name}</div>
                                <div className="selected-table-schema">{table.schema} schema</div>
                              </div>
                            </div>
                            <span className="selected-table-count">{table.columns} cols</span>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <div className="empty-box">No tables selected</div>
                    )
                  ) : (
                    Object.keys(filteredTablesBySchema).length > 0 ? (
                      Object.entries(filteredTablesBySchema).map(([schema, tables]) => (
                        <div key={schema} className="schema-block">
                          <div className="schema-block-header">
                            <span>{schema} Schema</span>
                            <span className="schema-selected-count">{tables.filter(t => exportOptions.selectedTables.includes(t.id)).length}/{tables.length} selected</span>
                          </div>
                          <div className="schema-block-body">
                            {tables.map((table) => (
                              <label key={table.id} className={exportOptions.selectedTables.includes(table.id) ? 'table-row selected' : 'table-row'}>
                                <div className="table-row-left">
                                  <input
                                    type="checkbox"
                                    checked={exportOptions.selectedTables.includes(table.id)}
                                    onChange={(e) => {
                                      const newSelected = e.target.checked
                                        ? [...exportOptions.selectedTables, table.id]
                                        : exportOptions.selectedTables.filter(t => t !== table.id);
                                      handleOptionChange('selectedTables', newSelected);
                                    }}
                                  />
                                  <span className="table-name">{(table.data?.label as string) || table.id}</span>
                                </div>
                                <span className="table-count">{table.data?.columns?.length || 0} cols</span>
                              </label>
                            ))}
                          </div>
                        </div>
                      ))
                    ) : (
                      <div className="empty-box">{tableSearch ? `No tables found matching "${tableSearch}"` : 'No tables available'}</div>
                    )
                  )}
                </div>
              </div>
            )}
          </div>

        </div>

        <div className="export-modal-footer">
          <div className="export-footer-left">
            <div className="export-time">⏰ Estimated export time: ~30 seconds</div>
            <div className="export-size">📥 File size: ~{estimatedSize} MB</div>
          </div>
          <div className="export-footer-actions">
            <button onClick={onCancel} className="btn secondary">❌ Cancel</button>
            <button onClick={handleExport} disabled={!canExport || isExporting} className={canExport && !isExporting ? 'btn primary' : 'btn disabled'}>
              {isExporting ? '⏳ Exporting...' : '📥 Export Data Catalog'}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
 