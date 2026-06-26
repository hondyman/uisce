/* eslint-disable react/forbid-dom-props */
/* eslint-disable react/forbid-component-props */
import React, { useState, useRef } from 'react';
import { useNotification } from '../hooks/useNotification';
import { 
  Grid, Plus as _Plus, Trash2, Copy, Settings, Save, Download, Eye as _Eye, Code, 
  Layers, Move, BarChart3, LineChart, PieChart, Table, TrendingUp,
  Database, Filter as _Filter, SortAsc as _SortAsc, Layout as _Layout, Sparkles
} from 'lucide-react';
import _styles from './SemanticLayoutBuilder.module.css';

// Types for semantic layer integration
interface SemanticDimension {
  id: string;
  name: string;
  type: 'string' | 'number' | 'time' | 'boolean';
  sql: string;
  title?: string;
  description?: string;
}

interface SemanticMeasure {
  id: string;
  name: string;
  type: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_distinct';
  sql: string;
  title?: string;
  description?: string;
  format?: string;
}

interface SemanticView {
  id: string;
  name: string;
  title: string;
  description?: string;
  dimensions: SemanticDimension[];
  measures: SemanticMeasure[];
}

interface LayoutComponent {
  id: string;
  type: 'MetricCard' | 'DataTable' | 'LineChart' | 'BarChart' | 'PieChart' | 'AreaChart';
  layout: {
    row: number;
    col: number;
    width: number;
    height: number;
  };
  config: {
    title?: string;
    semanticView?: string;
    dimensions?: string[];
    measures?: string[];
    filters?: any[];
    sort?: { field: string; direction: 'asc' | 'desc' }[];
  };
}

const SemanticLayoutBuilder: React.FC = () => {
  const [components, setComponents] = useState<LayoutComponent[]>([]);
  const [selectedComponent, setSelectedComponent] = useState<LayoutComponent | null>(null);
  const [gridSize, setGridSize] = useState({ cols: 12, rows: 12 });
  // state variable intentionally unused in rendering; underscore prefix silences the linter
  const [_isDragging, setIsDragging] = useState(false);
  const [draggedComponentType, setDraggedComponentType] = useState<string | null>(null);
  const [showCode, setShowCode] = useState(false);
  const gridRef = useRef<HTMLDivElement>(null);
  const notification = useNotification();

  // Mock semantic views - replace with your actual semantic layer data
  const [semanticViews] = useState<SemanticView[]>([
    {
      id: 'portfolio_positions',
      name: 'portfolio_positions',
      title: 'Portfolio Positions',
      description: 'Current portfolio holdings and performance',
      dimensions: [
        { id: 'security_name', name: 'security_name', type: 'string', sql: '{CUBE}.security_name', title: 'Security Name' },
        { id: 'sector', name: 'sector', type: 'string', sql: '{CUBE}.sector', title: 'Sector' },
        { id: 'asset_class', name: 'asset_class', type: 'string', sql: '{CUBE}.asset_class', title: 'Asset Class' },
        { id: 'purchase_date', name: 'purchase_date', type: 'time', sql: '{CUBE}.purchase_date', title: 'Purchase Date' }
      ],
      measures: [
        { id: 'market_value', name: 'market_value', type: 'sum', sql: 'SUM({CUBE}.market_value)', title: 'Market Value', format: 'currency' },
        { id: 'cost_basis', name: 'cost_basis', type: 'sum', sql: 'SUM({CUBE}.cost_basis)', title: 'Cost Basis', format: 'currency' },
        { id: 'gain_loss', name: 'gain_loss', type: 'sum', sql: 'SUM({CUBE}.market_value - {CUBE}.cost_basis)', title: 'Gain/Loss', format: 'currency' },
        { id: 'position_count', name: 'position_count', type: 'count', sql: 'COUNT(*)', title: 'Position Count' }
      ]
    },
    {
      id: 'trade_history',
      name: 'trade_history',
      title: 'Trade History',
      description: 'Historical trade executions and performance',
      dimensions: [
        { id: 'trade_date', name: 'trade_date', type: 'time', sql: '{CUBE}.trade_date', title: 'Trade Date' },
        { id: 'security_symbol', name: 'security_symbol', type: 'string', sql: '{CUBE}.security_symbol', title: 'Symbol' },
        { id: 'trade_type', name: 'trade_type', type: 'string', sql: '{CUBE}.trade_type', title: 'Trade Type' }
      ],
      measures: [
        { id: 'trade_volume', name: 'trade_volume', type: 'sum', sql: 'SUM({CUBE}.quantity)', title: 'Volume' },
        { id: 'trade_value', name: 'trade_value', type: 'sum', sql: 'SUM({CUBE}.amount)', title: 'Trade Value', format: 'currency' },
        { id: 'avg_price', name: 'avg_price', type: 'avg', sql: 'AVG({CUBE}.price)', title: 'Avg Price', format: 'currency' }
      ]
    }
  ]);

  const componentLibrary = [
    { type: 'MetricCard', icon: TrendingUp, label: 'KPI Card', defaultSize: { width: 3, height: 2 } },
    { type: 'DataTable', icon: Table, label: 'Data Table', defaultSize: { width: 12, height: 6 } },
    { type: 'LineChart', icon: LineChart, label: 'Line Chart', defaultSize: { width: 6, height: 4 } },
    { type: 'BarChart', icon: BarChart3, label: 'Bar Chart', defaultSize: { width: 6, height: 4 } },
    { type: 'PieChart', icon: PieChart, label: 'Pie Chart', defaultSize: { width: 4, height: 4 } },
    { type: 'AreaChart', icon: TrendingUp, label: 'Area Chart', defaultSize: { width: 8, height: 4 } }
  ];

  const handleDragStart = (componentType: string) => {
    setIsDragging(true);
    setDraggedComponentType(componentType);
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    if (!draggedComponentType || !gridRef.current) return;

    const rect = gridRef.current.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const y = e.clientY - rect.top;

    const cellWidth = rect.width / gridSize.cols;
    const cellHeight = rect.height / gridSize.rows;

    const col = Math.floor(x / cellWidth) + 1;
    const row = Math.floor(y / cellHeight) + 1;

    const comp = componentLibrary.find(c => c.type === draggedComponentType);
    if (!comp) return;

    const newComponent: LayoutComponent = {
      id: `component-${Date.now()}`,
      // narrow the library-provided type to our LayoutComponent['type'] union
      type: comp.type as LayoutComponent['type'],
      layout: {
        row,
        col,
        width: comp.defaultSize.width,
        height: comp.defaultSize.height
      },
      config: {
        title: `New ${comp.label}`,
        semanticView: semanticViews[0]?.id,
        dimensions: [],
        measures: [],
        filters: [],
        sort: []
      }
    };

    setComponents([...components, newComponent]);
    setSelectedComponent(newComponent);
    setDraggedComponentType(null);
  };

  const handleComponentClick = (comp: LayoutComponent) => {
    setSelectedComponent(comp);
  };

  const handleDeleteComponent = (id: string) => {
    setComponents(components.filter(c => c.id !== id));
    if (selectedComponent?.id === id) {
      setSelectedComponent(null);
    }
  };

  const handleDuplicateComponent = (comp: LayoutComponent) => {
    const newComp: LayoutComponent = {
      ...comp,
      id: `component-${Date.now()}`,
      layout: {
        ...comp.layout,
        row: comp.layout.row + comp.layout.height
      }
    };
    setComponents([...components, newComp]);
  };

  const handleUpdateComponent = (id: string, updates: Partial<LayoutComponent>) => {
    setComponents(components.map(c => 
      c.id === id ? { ...c, ...updates } : c
    ));
    if (selectedComponent?.id === id) {
      setSelectedComponent({ ...selectedComponent, ...updates });
    }
  };

  const exportConfig = () => {
    const config = {
      layout: {
        id: 'semantic-dashboard',
        name: 'Semantic Dashboard',
        version: '1.0.0'
      },
      components: components.map(comp => ({
        id: comp.id,
        type: comp.type,
        layout: comp.layout,
        config: comp.config
      }))
    };
    return JSON.stringify(config, null, 2);
  };

  const handleDownload = () => {
    const config = exportConfig();
    const blob = new Blob([config], { type: 'application/json' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'semantic-dashboard-config.json';
    a.click();
    URL.revokeObjectURL(url);
  };

  const getGridPosition = (layout: LayoutComponent['layout']) => {
    return {
      gridColumn: `${layout.col} / span ${layout.width}`,
      gridRow: `${layout.row} / span ${layout.height}`
    };
  };

  const selectedView = semanticViews.find(v => v.id === selectedComponent?.config.semanticView);

  return (
    <div className="min-h-screen bg-slate-900 text-white">
      {/* Header */}
      <div className="bg-slate-800 border-b border-slate-700 px-6 py-4">
        <div className="flex items-center justify-between max-w-[1920px] mx-auto">
          <div className="flex items-center gap-4">
            <Sparkles className="w-8 h-8 text-purple-400" />
            <div>
              <h1 className="text-2xl font-bold">Semantic Layout Builder</h1>
              <p className="text-sm text-slate-400">Build dashboards from your semantic layer with no code</p>
            </div>
          </div>
          
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowCode(!showCode)}
              className="flex items-center gap-2 px-4 py-2 bg-slate-700 hover:bg-slate-600 rounded-lg transition"
            >
              <Code className="w-4 h-4" />
              {showCode ? 'Hide' : 'Show'} JSON
            </button>
            <button
              onClick={handleDownload}
              className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition"
            >
              <Download className="w-4 h-4" />
              Export
            </button>
            <button className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg transition">
              <Save className="w-4 h-4" />
              Save
            </button>
          </div>
        </div>
      </div>

      <div className="flex h-[calc(100vh-80px)]">
        {/* Component Library Sidebar */}
        <div className="w-64 bg-slate-800 border-r border-slate-700 p-4 overflow-y-auto">
          <h2 className="text-lg font-bold mb-4 flex items-center gap-2">
            <Layers className="w-5 h-5 text-purple-400" />
            Components
          </h2>
          
          <div className="space-y-2">
            {componentLibrary.map((comp, idx) => {
              const Icon = comp.icon;
              return (
                <div
                  key={idx}
                  draggable
                  onDragStart={() => handleDragStart(comp.type)}
                  className="bg-slate-700 hover:bg-slate-600 rounded-lg p-3 cursor-move transition border-2 border-transparent hover:border-purple-500"
                >
                  <div className="flex items-center gap-3">
                    <Icon className="w-6 h-6 text-purple-400" />
                    <div>
                      <div className="font-medium text-sm">{comp.label}</div>
                      <div className="text-xs text-slate-400">
                        {comp.defaultSize.width}×{comp.defaultSize.height}
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          <div className="mt-6 p-3 bg-purple-900/30 border border-purple-500/30 rounded-lg">
            <div className="text-xs text-purple-300 mb-2 flex items-center gap-1">
              <Database className="w-3 h-3" />
              Semantic Views
            </div>
            <div className="text-xs text-slate-300 space-y-1">
              {semanticViews.map(view => (
                <div key={view.id} className="p-2 bg-slate-700/50 rounded">
                  {view.title}
                </div>
              ))}
            </div>
          </div>
        </div>

        {/* Main Canvas */}
        <div className="flex-1 flex flex-col">
          {/* Toolbar */}
          <div className="bg-slate-800 border-b border-slate-700 px-6 py-3 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <label htmlFor="grid-size-select" className="text-sm text-slate-400">Grid:</label>
                <select
                  id="grid-size-select"
                  value={gridSize.cols}
                  onChange={(e) => setGridSize({ ...gridSize, cols: parseInt(e.target.value) })}
                  className="bg-slate-700 text-white px-2 py-1 rounded text-sm"
                  aria-label="Select grid size"
                >
                  <option value={8}>8 cols</option>
                  <option value={12}>12 cols</option>
                  <option value={16}>16 cols</option>
                </select>
              </div>
              <div className="text-sm text-slate-400">
                {components.length} component{components.length !== 1 ? 's' : ''}
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button 
                onClick={() => setComponents([])}
                className="text-sm px-3 py-1 bg-red-600 hover:bg-red-700 rounded transition"
              >
                Clear All
              </button>
            </div>
          </div>

          {/* Canvas Area */}
          <div className="flex-1 p-6 overflow-auto bg-slate-900">
            {/* eslint-disable-next-line react/forbid-dom-props, react/forbid-component-props */}
            <div
              ref={gridRef}
              onDragOver={handleDragOver}
              onDrop={handleDrop}
              className="relative bg-slate-800/50 rounded-xl border-2 border-dashed border-slate-600 min-h-[800px]"
              // Dynamic grid dimensions - must use inline style
              style={{
                display: 'grid',
                gridTemplateColumns: `repeat(${gridSize.cols}, 1fr)`,
                gridTemplateRows: `repeat(${gridSize.rows}, 1fr)`,
                gap: '8px',
                padding: '16px'
              }}
            >
              {/* Components */}
              {components.map((comp) => {
                const view = semanticViews.find(v => v.id === comp.config.semanticView);
                return (
                  // eslint-disable-next-line react/forbid-dom-props, react/forbid-component-props
                  <div
                    key={comp.id}
                    onClick={() => handleComponentClick(comp)}
                    className={`relative bg-slate-700 rounded-lg p-4 cursor-pointer transition-all hover:ring-2 hover:ring-purple-500 ${
                      selectedComponent?.id === comp.id ? 'ring-2 ring-purple-500' : ''
                    }`}
                    // Dynamic grid positioning - must use inline style
                    style={getGridPosition(comp.layout)}
                  >
                    <div className="flex items-center justify-between mb-2">
                      <div className="flex items-center gap-2">
                        <Move className="w-4 h-4 text-slate-400" />
                        <span className="font-medium text-sm">{comp.config.title}</span>
                      </div>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDuplicateComponent(comp);
                          }}
                          className="p-1 hover:bg-slate-600 rounded"
                          aria-label="Duplicate component"
                          title="Duplicate component"
                        >
                          <Copy className="w-3 h-3" />
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteComponent(comp.id);
                          }}
                          className="p-1 hover:bg-red-600 rounded"
                          aria-label="Delete component"
                          title="Delete component"
                        >
                          <Trash2 className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                    
                    <div className="text-xs text-slate-400 mb-2">
                      View: {view?.title || 'Not selected'}
                    </div>

                    {/* Component Preview */}
                    <div className="mt-3 p-3 bg-slate-800 rounded border border-slate-600 text-xs">
                      <div className="text-purple-300 mb-1">Dimensions: {comp.config.dimensions?.length || 0}</div>
                      <div className="text-green-300">Measures: {comp.config.measures?.length || 0}</div>
                    </div>
                  </div>
                );
              })}

              {/* Empty State */}
              {components.length === 0 && (
                <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
                  <div className="text-center">
                    <Grid className="w-16 h-16 mx-auto mb-4 text-slate-600" />
                    <h3 className="text-xl font-bold text-slate-500 mb-2">Empty Canvas</h3>
                    <p className="text-slate-500">Drag components from the left to build your dashboard</p>
                  </div>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Configuration Panel */}
        <div className="w-96 bg-slate-800 border-l border-slate-700 p-4 overflow-y-auto">
          {selectedComponent && selectedView ? (
            <>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-bold flex items-center gap-2">
                  <Settings className="w-5 h-5 text-green-400" />
                  Configure
                </h2>
                <button
                  onClick={() => setSelectedComponent(null)}
                  className="text-slate-400 hover:text-white"
                >
                  ✕
                </button>
              </div>

              <div className="space-y-4">
                {/* Title */}
                <div className="bg-slate-700 rounded-lg p-3">
                  <label htmlFor="component-title" className="text-sm font-bold mb-2 block">Title</label>
                  <input
                    id="component-title"
                    type="text"
                    value={selectedComponent.config.title}
                    onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                      config: { ...selectedComponent.config, title: e.target.value }
                    })}
                    className="w-full bg-slate-600 text-white px-3 py-2 rounded"
                    aria-label="Component title"
                    placeholder="Enter component title"
                  />
                </div>

                {/* Semantic View Selection */}
                <div className="bg-slate-700 rounded-lg p-3">
                  <label htmlFor="semantic-view-select" className="text-sm font-bold mb-2 block">Semantic View</label>
                  <select
                    id="semantic-view-select"
                    value={selectedComponent.config.semanticView}
                    onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                      config: { 
                        ...selectedComponent.config, 
                        semanticView: e.target.value,
                        dimensions: [],
                        measures: []
                      }
                    })}
                    className="w-full bg-slate-600 text-white px-3 py-2 rounded"
                    aria-label="Select semantic view"
                  >
                    {semanticViews.map(view => (
                      <option key={view.id} value={view.id}>{view.title}</option>
                    ))}
                  </select>
                </div>

                {/* Dimensions */}
                <div className="bg-slate-700 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-sm font-bold">Dimensions</label>
                    <Database className="w-4 h-4 text-purple-400" />
                  </div>
                  <div className="space-y-2 max-h-48 overflow-y-auto">
                    {selectedView.dimensions.map(dim => (
                      <label key={dim.id} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={selectedComponent.config.dimensions?.includes(dim.name)}
                          onChange={(e) => {
                            const dims = selectedComponent.config.dimensions || [];
                            const newDims = e.target.checked
                              ? [...dims, dim.name]
                              : dims.filter(d => d !== dim.name);
                            handleUpdateComponent(selectedComponent.id, {
                              config: { ...selectedComponent.config, dimensions: newDims }
                            });
                          }}
                          className="rounded"
                        />
                        <span className="flex-1">{dim.title || dim.name}</span>
                        <span className="text-xs text-slate-400">{dim.type}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Measures */}
                <div className="bg-slate-700 rounded-lg p-3">
                  <div className="flex items-center justify-between mb-2">
                    <label className="text-sm font-bold">Measures</label>
                    <TrendingUp className="w-4 h-4 text-green-400" />
                  </div>
                  <div className="space-y-2 max-h-48 overflow-y-auto">
                    {selectedView.measures.map(measure => (
                      <label key={measure.id} className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          checked={selectedComponent.config.measures?.includes(measure.name)}
                          onChange={(e) => {
                            const measures = selectedComponent.config.measures || [];
                            const newMeasures = e.target.checked
                              ? [...measures, measure.name]
                              : measures.filter(m => m !== measure.name);
                            handleUpdateComponent(selectedComponent.id, {
                              config: { ...selectedComponent.config, measures: newMeasures }
                            });
                          }}
                          className="rounded"
                        />
                        <span className="flex-1">{measure.title || measure.name}</span>
                        <span className="text-xs text-slate-400">{measure.type}</span>
                      </label>
                    ))}
                  </div>
                </div>

                {/* Layout Settings */}
                <div className="bg-slate-700 rounded-lg p-3">
                  <div className="text-sm font-bold mb-3">Layout Position</div>
                  
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label htmlFor="layout-col" className="text-xs text-slate-400">Column</label>
                      <input
                        id="layout-col"
                        type="number"
                        min="1"
                        max={gridSize.cols}
                        value={selectedComponent.layout.col}
                        onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                          layout: { ...selectedComponent.layout, col: parseInt(e.target.value) }
                        })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm mt-1"
                        aria-label="Column position"
                      />
                    </div>
                    <div>
                      <label htmlFor="layout-row" className="text-xs text-slate-400">Row</label>
                      <input
                        id="layout-row"
                        type="number"
                        min="1"
                        value={selectedComponent.layout.row}
                        onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                          layout: { ...selectedComponent.layout, row: parseInt(e.target.value) }
                        })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm mt-1"
                        aria-label="Row position"
                      />
                    </div>
                    <div>
                      <label htmlFor="layout-width" className="text-xs text-slate-400">Width</label>
                      <input
                        id="layout-width"
                        type="number"
                        min="1"
                        max={gridSize.cols}
                        value={selectedComponent.layout.width}
                        onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                          layout: { ...selectedComponent.layout, width: parseInt(e.target.value) }
                        })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm mt-1"
                        aria-label="Component width"
                      />
                    </div>
                    <div>
                      <label htmlFor="layout-height" className="text-xs text-slate-400">Height</label>
                      <input
                        id="layout-height"
                        type="number"
                        min="1"
                        value={selectedComponent.layout.height}
                        onChange={(e) => handleUpdateComponent(selectedComponent.id, {
                          layout: { ...selectedComponent.layout, height: parseInt(e.target.value) }
                        })}
                        className="w-full bg-slate-600 text-white px-2 py-1 rounded text-sm mt-1"
                        aria-label="Component height"
                      />
                    </div>
                  </div>
                </div>

                {/* Quick Actions */}
                <div className="space-y-2">
                  <button
                    onClick={() => handleDuplicateComponent(selectedComponent)}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition"
                  >
                    <Copy className="w-4 h-4" />
                    Duplicate
                  </button>
                  <button
                    onClick={() => handleDeleteComponent(selectedComponent.id)}
                    className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-red-600 hover:bg-red-700 rounded-lg transition"
                  >
                    <Trash2 className="w-4 h-4" />
                    Delete
                  </button>
                </div>
              </div>
            </>
          ) : (
            <div className="flex items-center justify-center h-full text-center text-slate-500">
              <div>
                <Settings className="w-12 h-12 mx-auto mb-3 opacity-50" />
                <p>Select a component to configure</p>
                <p className="text-xs mt-2">Choose dimensions & measures from your semantic views</p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* JSON Preview Modal */}
      {showCode && (
        <div className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8">
          <div className="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-4xl max-h-[80vh] flex flex-col">
            <div className="flex items-center justify-between p-6 border-b border-slate-700">
              <h2 className="text-xl font-bold flex items-center gap-2">
                <Code className="w-6 h-6 text-green-400" />
                Dashboard Configuration
              </h2>
              <button
                onClick={() => setShowCode(false)}
                className="text-slate-400 hover:text-white"
              >
                ✕
              </button>
            </div>
            
            <div className="flex-1 overflow-auto p-6">
              <pre className="bg-slate-900 p-4 rounded-lg text-sm text-green-400 font-mono overflow-x-auto">
                {exportConfig()}
              </pre>
            </div>

            <div className="p-6 border-t border-slate-700 flex gap-3">
              <button
                onClick={() => {
                  navigator.clipboard.writeText(exportConfig());
                  const notification = useNotification();
                  notification.success('Copied to clipboard!');
                }}
                className="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition"
              >
                Copy to Clipboard
              </button>
              <button
                onClick={handleDownload}
                className="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg transition"
              >
                Download JSON
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default SemanticLayoutBuilder;
