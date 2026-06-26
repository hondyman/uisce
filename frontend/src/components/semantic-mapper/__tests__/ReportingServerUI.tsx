/* eslint-disable no-console */

import React, { useState, useRef, useEffect } from 'react';
import { 
  FileText, 
  Plus, 
  Save, 
  Download, 
  Settings, 
  Database,
  Table,
  BarChart3,
  Filter,
  Grid3x3,
  Type,
  Image,
  Code,
  ChevronDown,
  Trash2,
  Copy,
  FolderOpen,
  Layers,
  X,
  CheckCircle
} from 'lucide-react';

// Drag and drop imports
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import { Rnd } from 'react-rnd';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import { devDebug } from '../../../../utils/devLogger';

// ============================================================================
// TypeScript Interfaces for Backend Communication
// ============================================================================

interface ReportConfig {
  id: string;
  name: string;
  description: string;
  dataSourceId: string;
  pageSettings: PageSettings;
  parameters: ReportParameter[];
  elements: ReportElement[];
  exportFormats: ExportFormat[];
  createdBy: string;
  createdAt: string;
  lastModified: string;
}

interface PageSettings {
  size: 'letter' | 'a4' | 'legal' | 'tabloid';
  orientation: 'portrait' | 'landscape';
  margins: {
    top: number;
    right: number;
    bottom: number;
    left: number;
  };
  units: 'in' | 'cm' | 'mm';
}

interface ReportParameter {
  id: string;
  name: string;
  label: string;
  type: 'string' | 'number' | 'date' | 'boolean' | 'list';
  defaultValue?: any;
  required: boolean;
  visible: boolean;
  availableValues?: { label: string; value: any }[];
  dataSourceQuery?: string;
}

interface ReportElement {
  id: string;
  type: 'table' | 'chart' | 'text' | 'image' | 'parameter';
  position: { x: number; y: number };
  size: { width: number; height: number };
  properties: ElementProperties;
}

interface ElementProperties {
  dataSource?: string;
  query?: QueryDefinition;
  style?: StyleProperties;
  chartConfig?: ChartConfig;
  textContent?: string;
  imageUrl?: string;
}

interface QueryDefinition {
  type: 'cube' | 'sql';
  cubeQuery?: CubeQuery;
  sqlQuery?: string;
  parameters?: string[];
}

interface CubeQuery {
  measures: string[];
  dimensions: string[];
  timeDimensions?: Array<{
    dimension: string;
    granularity?: string;
    dateRange?: string[];
  }>;
  filters?: Array<{
    member: string;
    operator: string;
    values: any[];
  }>;
  order?: { [key: string]: 'asc' | 'desc' };
  limit?: number;
}

interface ChartConfig {
  chartType: 'bar' | 'line' | 'pie' | 'area' | 'scatter';
  xAxis: string;
  yAxis: string[];
  legend: boolean;
  colors?: string[];
}

interface StyleProperties {
  backgroundColor?: string;
  textColor?: string;
  fontSize?: number;
  fontFamily?: string;
  fontWeight?: string;
  textAlign?: 'left' | 'center' | 'right';
  border?: string;
  padding?: number;
}

type ExportFormat = 'pdf' | 'excel' | 'word' | 'html' | 'csv';

interface DataSource {
  id: string;
  name: string;
  type: 'cube' | 'postgres';
  connectionString: string;
  cubeConfig?: {
    apiUrl: string;
    apiToken?: string;
  };
  postgresConfig?: {
    host: string;
    port: number;
    database: string;
    schema?: string;
  };
}

interface PreviewRequest {
  reportConfig: ReportConfig;
  parameters: { [key: string]: any };
  format?: 'html' | 'json';
}

interface RenderRequest {
  reportId: string;
  parameters: { [key: string]: any };
  format: ExportFormat;
}

// ============================================================================
// API Client for Golang Backend
// ============================================================================

class ReportingAPIClient {
  private baseUrl: string;

  constructor(baseUrl: string = 'http://localhost:9088/api') {
    this.baseUrl = baseUrl;
  }

  async saveReport(config: ReportConfig): Promise<{ id: string; success: boolean }> {
    const response = await fetch(`${this.baseUrl}/reports`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    return response.json();
  }

  async updateReport(id: string, config: ReportConfig): Promise<{ success: boolean }> {
    const response = await fetch(`${this.baseUrl}/reports/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config),
    });
    return response.json();
  }

  async getReport(id: string): Promise<ReportConfig> {
    const response = await fetch(`${this.baseUrl}/reports/${id}`);
    return response.json();
  }

  async listReports(): Promise<ReportConfig[]> {
    const response = await fetch(`${this.baseUrl}/reports`);
    return response.json();
  }

  async deleteReport(id: string): Promise<{ success: boolean }> {
    const response = await fetch(`${this.baseUrl}/reports/${id}`, {
      method: 'DELETE',
    });
    return response.json();
  }

  async previewReport(request: PreviewRequest): Promise<{ html: string; data?: any }> {
    const response = await fetch(`${this.baseUrl}/reports/preview`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    return response.json();
  }

  async renderReport(request: RenderRequest): Promise<Blob> {
    const response = await fetch(`${this.baseUrl}/reports/render`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(request),
    });
    return response.blob();
  }

  async testDataSource(dataSource: DataSource): Promise<{ success: boolean; message: string }> {
    const response = await fetch(`${this.baseUrl}/datasources/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dataSource),
    });
    return response.json();
  }

  async getDataSourceSchema(dataSourceId: string): Promise<any> {
    const response = await fetch(`${this.baseUrl}/datasources/${dataSourceId}/schema`);
    return response.json();
  }

  async executeQuery(dataSourceId: string, query: QueryDefinition): Promise<any> {
    const response = await fetch(`${this.baseUrl}/datasources/${dataSourceId}/query`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(query),
    });
    return response.json();
  }
}

// ============================================================================
// Main Component
// ============================================================================

const ReportingServerUI = (): JSX.Element | null => {
  const [activeTab, setActiveTab] = useState<'reports' | 'designer' | 'data'>('reports');
  const [selectedReport, setSelectedReport] = useState<ReportConfig | null>(null);
  const [designerMode, setDesignerMode] = useState<'design' | 'preview'>('design');
  const [showQueryBuilder, setShowQueryBuilder] = useState(false);
  const [showParameterConfig, setShowParameterConfig] = useState(false);
  const [selectedElement, setSelectedElement] = useState<ReportElement | null>(null);
  
  // Initialize hooks
  const confirm = useConfirm();
  const notification = useNotification();

  // Initialize API client
  const [reportApiClient] = useState(() => new ReportingAPIClient());
  
  const [reports, setReports] = useState<ReportConfig[]>([
    {
      id: '1',
      name: 'Sales Analysis Report',
      description: 'Monthly sales breakdown by region',
      dataSourceId: 'cube-1',
      pageSettings: {
        size: 'letter',
        orientation: 'portrait',
        margins: { top: 1, right: 1, bottom: 1, left: 1 },
        units: 'in'
      },
      parameters: [],
      elements: [],
      exportFormats: ['pdf', 'excel'],
      createdBy: 'admin',
      createdAt: '2025-10-01',
      lastModified: '2025-10-08'
    }
  ]);

  const [dataSources] = useState<DataSource[]>([
    {
      id: 'cube-1',
      name: 'cube-analytics',
      type: 'cube',
      connectionString: 'http://localhost:4000/cubejs-api/v1',
      cubeConfig: {
        apiUrl: 'http://localhost:4000/cubejs-api/v1',
        apiToken: 'your-api-token'
      }
    },
    {
      id: 'pg-1',
      name: 'postgres-main',
      type: 'postgres',
      connectionString: 'postgresql://localhost:5432/reporting',
      postgresConfig: {
        host: 'localhost',
        port: 5432,
        database: 'reporting',
        schema: 'public'
      }
    }
  ]);

  // Query Builder State
  const [queryConfig, setQueryConfig] = useState<CubeQuery>({
    measures: [],
    dimensions: [],
    timeDimensions: [],
    filters: []
  });

  // Parameter Configuration State
  // const [currentParameter, setCurrentParameter] = useState<ReportParameter>({
  //   id: '',
  //   name: '',
  //   label: '',
  //   type: 'string',
  //   required: false,
  //   visible: true
  // });

  // ============================================================================
  // Query Builder Component
  // ============================================================================

  const QueryBuilder: React.FC<{
    onClose: () => void;
    onApply: (query: CubeQuery) => void;
  }> = ({ onClose, onApply }) => {
    const [localQuery, setLocalQuery] = useState<CubeQuery>(queryConfig);
    const [sqlPreview, setSqlPreview] = useState('');

    const availableMeasures = [
      'Orders.count',
      'Orders.totalAmount',
      'Orders.averageAmount',
      'Users.count'
    ];

    const availableDimensions = [
      'Orders.status',
      'Orders.region',
      'Users.city',
      'Users.country',
      'Products.category'
    ];

    const addMeasure = (measure: string) => {
      if (!localQuery.measures.includes(measure)) {
        setLocalQuery({
          ...localQuery,
          measures: [...localQuery.measures, measure]
        });
      }
    };

    const addDimension = (dimension: string) => {
      if (!localQuery.dimensions.includes(dimension)) {
        setLocalQuery({
          ...localQuery,
          dimensions: [...localQuery.dimensions, dimension]
        });
      }
    };

    const addFilter = () => {
      setLocalQuery({
        ...localQuery,
        filters: [
          ...(localQuery.filters || []),
          { member: '', operator: 'equals', values: [] }
        ]
      });
    };

    const generateSQLPreview = () => {
      const sql = `-- Cube.dev Query (will be converted to SQL by backend)
SELECT 
  ${localQuery.measures.join(',\n  ')}${localQuery.dimensions.length > 0 ? ',\n  ' + localQuery.dimensions.join(',\n  ') : ''}
FROM analytics
${localQuery.filters && localQuery.filters.length > 0 ? `WHERE ${localQuery.filters.map(f => `${f.member} ${f.operator} ${JSON.stringify(f.values)}`).join('\n  AND ')}` : ''}
${localQuery.timeDimensions && localQuery.timeDimensions.length > 0 ? `GROUP BY ${localQuery.timeDimensions[0].dimension}` : ''}`;
      setSqlPreview(sql);
    };

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg w-full max-w-5xl max-h-[90vh] flex flex-col">
          <div className="flex items-center justify-between p-4 border-b">
            <h2 className="text-xl font-semibold">Query Builder</h2>
            <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded" aria-label="Close"><X className="w-5 h-5" /></button>
          </div>
          <div className="flex-1 p-6 overflow-y-auto">
            <div className="grid grid-cols-2 gap-6">
              {/* Left Column - Configuration */}
              <div className="space-y-6">
                <div>
                  <h3 className="font-semibold mb-3 flex items-center gap-2">
                    <BarChart3 className="w-4 h-4" />
                    Measures
                  </h3>
                  <div className="space-y-2 mb-3">
                    {localQuery.measures.map((measure, idx) => (
                      <div key={idx} className="flex items-center gap-2 bg-blue-50 px-3 py-2 rounded">
                        <span className="flex-1 text-sm">{measure}</span>
                        <button
                          onClick={() => setLocalQuery({
                            ...localQuery,
                            measures: localQuery.measures.filter((_, i) => i !== idx)
                          })}
                          className="text-red-600 hover:text-red-800"
                          aria-label="Remove measure"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <select
                    id="add-measure-select"
                    aria-label="Add measure"
                    title="Add measure"
                    className="w-full px-3 py-2 border rounded-lg"
                    onChange={(e) => {
                      if (e.target.value) {
                        addMeasure(e.target.value);
                        e.target.value = '';
                      }
                    }}
                  >
                    <option value="">Add measure...</option>
                    {availableMeasures.map(m => (
                      <option key={m} value={m}>{m}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <h3 className="font-semibold mb-3 flex items-center gap-2">
                    <Layers className="w-4 h-4" />
                    Dimensions
                  </h3>
                  <div className="space-y-2 mb-3">
                    {localQuery.dimensions.map((dimension, idx) => (
                      <div key={idx} className="flex items-center gap-2 bg-green-50 px-3 py-2 rounded">
                        <span className="flex-1 text-sm">{dimension}</span>
                        <button
                          onClick={() => setLocalQuery({
                            ...localQuery,
                            dimensions: localQuery.dimensions.filter((_, i) => i !== idx)
                          })}
                          className="text-red-600 hover:text-red-800"
                          aria-label="Remove dimension"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <select
                    id="add-dimension-select"
                    aria-label="Add dimension"
                    title="Add dimension"
                    className="w-full px-3 py-2 border rounded-lg"
                    onChange={(e) => {
                      if (e.target.value) {
                        addDimension(e.target.value);
                        e.target.value = '';
                      }
                    }}
                  >
                    <option value="">Add dimension...</option>
                    {availableDimensions.map(d => (
                      <option key={d} value={d}>{d}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <h3 className="font-semibold mb-3 flex items-center gap-2">
                    <Filter className="w-4 h-4" />
                    Filters
                  </h3>
                  <div className="space-y-2 mb-3">
                    {(localQuery.filters || []).map((filter, idx) => (
                      <div key={idx} className="flex items-center gap-2">
                        <select
                          className="flex-1 px-2 py-1 border rounded text-sm"
                          value={filter.member}
                          onChange={(e) => {
                            const newFilters = [...(localQuery.filters || [])];
                            newFilters[idx].member = e.target.value;
                            setLocalQuery({ ...localQuery, filters: newFilters });
                          }}
                          aria-label="Select field"
                          title="Select field"
                        >
                          <option value="">Select field...</option>
                          {availableDimensions.map(d => (
                            <option key={d} value={d}>{d}</option>
                          ))}
                        </select>

                        <select
                          className="px-2 py-1 border rounded text-sm"
                          value={filter.operator}
                          onChange={(e) => {
                            const newFilters = [...(localQuery.filters || [])];
                            newFilters[idx].operator = e.target.value;
                            setLocalQuery({ ...localQuery, filters: newFilters });
                          }}
                          aria-label="Select operator"
                          title="Select operator"
                        >
                          <option value="equals">Equals</option>
                          <option value="notEquals">Not Equals</option>
                          <option value="contains">Contains</option>
                          <option value="gt">Greater Than</option>
                          <option value="lt">Less Than</option>
                        </select>

                        <button
                          onClick={() => setLocalQuery({
                            ...localQuery,
                            filters: localQuery.filters?.filter((_, i) => i !== idx)
                          })}
                          className="text-red-600 hover:text-red-800"
                          aria-label="Remove filter"
                        >
                          <X className="w-4 h-4" />
                        </button>
                      </div>
                    ))}
                  </div>
                  <button
                    onClick={addFilter}
                    className="w-full px-3 py-2 border border-dashed border-gray-300 rounded-lg text-gray-600 hover:bg-gray-50"
                  >
                    + Add Filter
                  </button>
                </div>
              </div>

              {/* Right Column - Preview */}
              <div>
                <h3 className="font-semibold mb-3 flex items-center gap-2">
                  <Code className="w-4 h-4" />
                  Query Preview
                </h3>
                <div className="bg-gray-900 text-gray-100 p-4 rounded-lg font-mono text-sm overflow-auto max-h-96">
                  <pre>{JSON.stringify(localQuery, null, 2)}</pre>
                </div>
                <button
                  onClick={generateSQLPreview}
                  className="mt-3 w-full px-4 py-2 bg-gray-100 hover:bg-gray-200 rounded-lg text-sm"
                >
                  Generate SQL Preview
                </button>
                {sqlPreview && (
                  <div className="mt-3 bg-blue-900 text-blue-100 p-4 rounded-lg font-mono text-sm overflow-auto max-h-64">
                    <pre>{sqlPreview}</pre>
                  </div>
                )}
              </div>
            </div>
          </div>

          <div className="flex justify-end gap-2 p-4 border-t">
            <button
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                onApply(localQuery);
                onClose();
              }}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Apply Query
            </button>
          </div>
        </div>
      </div>
    );
  };

  // ============================================================================
  // Parameter Configuration Component
  // ============================================================================

  const ParameterConfig: React.FC<{
    onClose: () => void;
    onSave: (param: ReportParameter) => void;
    parameter?: ReportParameter;
  }> = ({ onClose, onSave, parameter }) => {
    const [param, setParam] = useState<ReportParameter>(
      parameter || {
        id: `param-${Date.now()}`,
        name: '',
        label: '',
        type: 'string',
        required: false,
        visible: true
      }
    );

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
        <div className="bg-white rounded-lg w-full max-w-2xl">
          <div className="flex items-center justify-between p-4 border-b">
            <h2 className="text-xl font-semibold">Configure Parameter</h2>
            <button onClick={onClose} className="p-2 hover:bg-gray-100 rounded" aria-label="Close">
              <X className="w-5 h-5" />
            </button>
          </div>

          <div className="p-6 space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2">Parameter Name</label>
                <input
                  type="text"
                  className="w-full px-3 py-2 border rounded-lg"
                  value={param.name}
                  onChange={(e) => setParam({ ...param, name: e.target.value })}
                  placeholder="startDate"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">Display Label</label>
                <input
                  type="text"
                  className="w-full px-3 py-2 border rounded-lg"
                  value={param.label}
                  onChange={(e) => setParam({ ...param, label: e.target.value })}
                  placeholder="Start Date"
                />
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium mb-2" htmlFor="parameter-type-select">Parameter Type</label>
                <select
                  id="parameter-type-select"
                  title="Parameter type"
                  className="w-full px-3 py-2 border rounded-lg"
                  value={param.type}
                  onChange={(e) => setParam({ ...param, type: e.target.value as any })}
                >
                  <option value="string">String</option>
                  <option value="number">Number</option>
                  <option value="date">Date</option>
                  <option value="boolean">Boolean</option>
                  <option value="list">List (Multiple Values)</option>
                </select>
              </div>

              <div>
                <label className="block text-sm font-medium mb-2">Default Value</label>
                <input
                  type={param.type === 'number' ? 'number' : param.type === 'date' ? 'date' : 'text'}
                  className="w-full px-3 py-2 border rounded-lg"
                  value={param.defaultValue || ''}
                  onChange={(e) => setParam({ ...param, defaultValue: e.target.value })}
                  placeholder="Enter default value"
                  aria-label="Default value"
                  title="Default value"
                />
              </div>
            </div>

            <div className="flex gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={param.required}
                  onChange={(e) => setParam({ ...param, required: e.target.checked })}
                  className="rounded"
                  aria-label="Required parameter"
                  title="Required parameter"
                />
                <span className="text-sm">Required</span>
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={param.visible}
                  onChange={(e) => setParam({ ...param, visible: e.target.checked })}
                  className="rounded"
                  aria-label="Visible to users"
                  title="Visible to users"
                />
                <span className="text-sm">Visible to Users</span>
              </label>
            </div>

            {param.type === 'list' && (
              <div>
                <label className="block text-sm font-medium mb-2">Available Values (JSON)</label>
                <textarea
                  className="w-full px-3 py-2 border rounded-lg font-mono text-sm"
                  rows={4}
                  placeholder='[{"label": "Option 1", "value": "opt1"}]'
                  onChange={(e) => {
                    try {
                      const values = JSON.parse(e.target.value);
                      setParam({ ...param, availableValues: values });
                    } catch (err) {
                      // Invalid JSON, ignore
                    }
                  }}
                  aria-label="Available values JSON"
                />
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 p-4 border-t">
            <button
              onClick={onClose}
              className="px-4 py-2 border border-gray-300 rounded-lg hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              onClick={() => {
                onSave(param);
                onClose();
              }}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
            >
              Save Parameter
            </button>
          </div>
        </div>
      </div>
    );
  };

  // ============================================================================
  // Drag and Drop / Resizable Element Components
  // ============================================================================

  // Define item types for react-dnd
  const ItemTypes = {
    TOOLBOX_ITEM: 'toolboxItem',
  };

  // Draggable Toolbox Item Component
  interface DraggableToolboxItemProps {
    type: ReportElement['type'];
    label: string;
    icon: React.ElementType;
  }

  const DraggableToolboxItem: React.FC<DraggableToolboxItemProps> = ({ type, label, icon: Icon }) => {
  const [{ isDragging }, drag] = useDrag(() => ({
    type: ItemTypes.TOOLBOX_ITEM,
    item: { type, label }, // Data carried by the drag operation
    collect: (monitor: any) => ({
      isDragging: monitor.isDragging(),
    }),
  }));
  return (
      <div
        ref={drag}
        className={`flex flex-col items-center p-3 border rounded-lg cursor-grab ${isDragging ? 'opacity-50 border-blue-500' : 'hover:bg-gray-50'}`}
      >
        <Icon className="w-6 h-6 mb-2 text-gray-600" />
        <span className="text-xs text-center">{label}</span>
      </div>
    );
  };

  // Resizable & Draggable Report Element Component
  interface ResizableDraggableElementProps {
    element: ReportElement;
    onUpdate: (id: string, newPosition: { x: number; y: number }, newSize: { width: number; height: number }) => void;
    onSelect: (element: ReportElement) => void;
    isSelected: boolean;
  }

  const ResizableDraggableElement: React.FC<ResizableDraggableElementProps> = ({ element, onUpdate, onSelect, isSelected }) => {
    const { id, position, size, type, properties } = element;

    const renderContent = () => {
      switch (type) {
        case 'table':
          return <div className="bg-blue-100 p-2 text-center text-sm">Table ({properties.dataSource || 'No Data'})</div>;
        case 'chart':
          return <div className="bg-green-100 p-2 text-center text-sm">Chart ({properties.chartConfig?.chartType || 'Bar'})</div>;
        case 'text':
          return <div className="bg-yellow-100 p-2 text-sm overflow-hidden">{properties.textContent || 'Text Element'}</div>;
        case 'image':
          return <div className="bg-purple-100 p-2 text-center text-sm">Image</div>;
        case 'parameter':
          return <div className="bg-red-100 p-2 text-center text-sm">Parameter</div>;
        default:
          return <div className="bg-gray-100 p-2 text-center text-sm">Unknown Element</div>;
      }
    };

    return (
      <Rnd
        size={{ width: size.width, height: size.height }}
        position={{ x: position.x, y: position.y }}
        onDragStop={(_e, d) => {
          onUpdate(id, { x: d.x, y: d.y }, size);
        }}
        onResizeStop={(_e, _direction, ref, _delta, newPosition) => {
          onUpdate(
            id,
            { x: newPosition.x, y: newPosition.y },
            { width: parseInt(ref.style.width), height: parseInt(ref.style.height) }
          );
        }}
        bounds="parent" // Keep within the canvas
        minWidth={50}
        minHeight={25}
        className={`border-2 ${isSelected ? 'border-blue-500' : 'border-gray-300'} bg-white shadow-md`}
        onClick={(e: any) => {
          e.stopPropagation(); // Prevent canvas click from deselecting
          onSelect(element);
        }}
      >
        {renderContent()}
      </Rnd>
    );
  };

  // Droppable Canvas Component
  interface DroppableCanvasProps {
    children: React.ReactNode;
    onDrop: (item: { type: ReportElement['type']; label: string }, dropPosition: { x: number; y: number }) => void;
    canvasRef: React.RefObject<HTMLDivElement>;
  }

  const DroppableCanvas: React.FC<DroppableCanvasProps> = ({ children, onDrop, canvasRef }) => {
    const [{ isOver }, drop] = useDrop(() => ({
      accept: ItemTypes.TOOLBOX_ITEM,
      drop: (item: { type: ReportElement['type']; label: string }, monitor: any) => {
        const clientOffset = monitor.getClientOffset();
        if (clientOffset && canvasRef.current) {
          const canvasRect = canvasRef.current.getBoundingClientRect();
          const dropX = clientOffset.x - canvasRect.left;
          const dropY = clientOffset.y - canvasRect.top;
          onDrop(item, { x: dropX, y: dropY });
        }
      },
      collect: (monitor: any) => ({
        isOver: monitor.isOver(),
      }),
    }));

    return (
      <div
        ref={(el) => {
          drop(el); // Connect to react-dnd
          (canvasRef as React.MutableRefObject<HTMLDivElement | null>).current = el; // Connect to local ref
        }}
        className={`p-4 h-full border-2 border-dashed ${isOver ? 'border-blue-500 bg-blue-50' : 'border-gray-300'} relative`}
      >
        {children}
      </div>
    );
  };

  // ============================================================================
  // Report Actions
  // ============================================================================

  const handleSaveReport = async () => {
    if (!selectedReport) return;
    
    try {
      if (selectedReport.id.startsWith('new-')) {
        const result = await reportApiClient.saveReport(selectedReport);
        devDebug('Report saved:', result);
        notification.success('Report saved successfully!');
      } else {
        await reportApiClient.updateReport(selectedReport.id, selectedReport);
        devDebug('Report updated');
        notification.success('Report updated successfully!');
      }
    } catch (error) {
      console.error('Error saving report:', error);
      notification.error('Failed to save report. Check console for details.');
    }
  };

  const handlePreviewReport = async () => {
    if (!selectedReport) return;
    
    try {
      const request: PreviewRequest = {
        reportConfig: selectedReport,
        parameters: {},
        format: 'html'
      };
      const result = await reportApiClient.previewReport(request);
      devDebug('Preview generated:', result);
      setDesignerMode('preview');
    } catch (error) {
      console.error('Error generating preview:', error);
      notification.error('Failed to generate preview. Check console for details.');
    }
  };

  const handleExportReport = async (format: ExportFormat) => {
    if (!selectedReport) return;
    
    try {
      const request: RenderRequest = {
        reportId: selectedReport.id,
        parameters: {},
        format
      };
      const blob = await reportApiClient.renderReport(request);
      
      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${selectedReport.name}.${format}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } catch (error) {
      console.error('Error exporting report:', error);
      notification.error('Failed to export report. Check console for details.');
    }
  };

  const handleDeleteReport = async (e: React.MouseEvent, reportId: string) => {
    e.stopPropagation();
    if (!(await confirm({ title: 'Delete report', description: 'Are you sure you want to delete this report?' }))) {
      return;
    }
    try {
      // await reportApiClient.deleteReport(reportId);
      setReports(reports.filter(r => r.id !== reportId));
      notification.success('Report deleted successfully!');
    } catch (error) {
      console.error('Error deleting report:', error);
      notification.error('Failed to delete report.');
    }
  };

  const handleCloneReport = (e: React.MouseEvent, reportToClone: ReportConfig) => {
    e.stopPropagation(); // Prevent card click event
    const newReport: ReportConfig = {
      ...reportToClone,
      id: `new-${Date.now()}`,
      name: `Copy of ${reportToClone.name}`,
      createdAt: new Date().toISOString(),
      lastModified: new Date().toISOString(),
      createdBy: 'current-user',
    };

    // Add to state and navigate to designer with the cloned report
    setReports(prevReports => [newReport, ...prevReports]);
    setSelectedReport(newReport);
    setActiveTab('designer');
  };

  // ============================================================================
  // Element Management Functions
  // ============================================================================

  const canvasRef = useRef<HTMLDivElement>(null);

  const handleAddElement = (item: { type: ReportElement['type']; label: string }, dropPosition: { x: number; y: number }) => {
    if (!selectedReport) return;

    const itemType = item.type;
    const dropX = dropPosition.x;
    const dropY = dropPosition.y;

    const baseProperties: ElementProperties = {};
    if (itemType === 'text') {
      baseProperties.textContent = 'New Text Element';
    } else if (itemType === 'chart') {
      baseProperties.chartConfig = { chartType: 'bar', xAxis: '', yAxis: [], legend: true };
    } else if (itemType === 'table') {
      baseProperties.query = { type: 'cube', cubeQuery: { measures: [], dimensions: [] } };
    }
    // Add more default properties for other types as needed

    const newElement: ReportElement = {
      id: `element-${Date.now()}`,
      type: itemType,
      position: { x: Math.max(0, dropX - 50), y: Math.max(0, dropY - 25) }, // Adjust for initial size
      size: { width: 100, height: 50 },
      properties: baseProperties,
    };

    setSelectedReport(prev => {
      if (!prev) return null;
      return {
        ...prev,
        elements: [...prev.elements, newElement],
      };
    });
    setSelectedElement(newElement); // Select the newly added element
  };

  const handleUpdateElement = (id: string, newPosition: { x: number; y: number }, newSize: { width: number; height: number }) => {
    if (!selectedReport) return;

    setSelectedReport(prev => {
      if (!prev) return null;
      return {
        ...prev,
        elements: prev.elements.map(el =>
          el.id === id ? { ...el, position: newPosition, size: newSize } : el
        ),
      };
    });
  };

  const handleSelectElement = (element: ReportElement) => {
    setSelectedElement(element);
  };

  // Clear selected element if the report changes or the element is no longer in the report
  useEffect(() => {
    if (selectedReport && selectedElement && !selectedReport.elements.some(el => el.id === selectedElement.id)) {
      setSelectedElement(null);
    } else if (!selectedReport) {
      setSelectedElement(null);
    }
  }, [selectedReport, selectedElement]);


  // ============================================================================
  // Render Methods
  // ============================================================================

  const renderReportsList = () => (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between p-4 border-b bg-white">
        <h2 className="text-lg font-semibold text-gray-800">Reports</h2>
        <div className="flex gap-2">
          <button
            onClick={() => {
              const newReport: ReportConfig = {
                id: `new-${Date.now()}`,
                name: 'New Report',
                description: '',
                dataSourceId: dataSources[0].id,
                pageSettings: {
                  size: 'letter',
                  orientation: 'portrait',
                  margins: { top: 1, right: 1, bottom: 1, left: 1 },
                  units: 'in'
                },
                parameters: [],
                elements: [],
                exportFormats: ['pdf', 'excel'],
                createdBy: 'current-user',
                createdAt: new Date().toISOString(),
                lastModified: new Date().toISOString()
              };
              setSelectedReport(newReport);
              setActiveTab('designer');
            }}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700"
          >
            <Plus className="w-4 h-4" />
            New Report
          </button>
          <button className="p-2 text-gray-600 hover:bg-gray-100 rounded-lg" aria-label="Open folder">
            <FolderOpen className="w-5 h-5" />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-auto p-6 bg-gray-50">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {reports.map((report) => (
            <div
              key={report.id}
              className="bg-white rounded-lg border border-gray-200 p-4 hover:shadow-lg transition-shadow cursor-pointer"
              onClick={() => {
                setSelectedReport(report);
                setActiveTab('designer');
              }}
            >
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <FileText className="w-8 h-8 text-blue-600 flex-shrink-0" />
                  <div>
                    <h3 className="font-semibold text-gray-900 mb-1 leading-tight">{report.name}</h3>
                    <p className="text-sm text-gray-600">{report.description}</p>
                  </div>
                </div>
                <div className="flex items-center gap-1">
                  <button onClick={(e) => handleCloneReport(e, report)} className="p-2 text-gray-400 hover:bg-gray-100 hover:text-gray-600 rounded-md" aria-label="Clone report">
                    <Copy className="w-4 h-4" />
                  </button>
                  <button onClick={(e) => handleDeleteReport(e, report.id)} className="p-2 text-gray-400 hover:bg-gray-100 hover:text-red-600 rounded-md" aria-label="Delete report">
                    <Trash2 className="w-4 h-4" />
                  </button>
                </div>
              </div>
              <div className="flex items-center justify-between text-xs text-gray-500 mt-4">
                <span className="flex items-center gap-1">
                  <Database className="w-3 h-3" />
                  {dataSources.find(ds => ds.id === report.dataSourceId)?.name}
                </span>
                <span>{new Date(report.lastModified).toLocaleDateString()}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );

  const renderDesigner = () => {
    if (!selectedReport) {
      return (
        <div className="flex items-center justify-center h-full text-gray-500">
          Select a report to start designing or create a new one.
        </div>
      );
    }

    return (
      <div className="flex flex-col h-full bg-gray-100">
        {/* Designer Header */}
        <div className="flex items-center justify-between p-2 border-b bg-white">
          <div className="flex items-center gap-4">
            <h2 className="text-lg font-semibold text-gray-800">{selectedReport.name}</h2>
            <div className="flex items-center bg-gray-100 rounded-lg p-1">
              <button 
                onClick={() => { setDesignerMode('design'); setSelectedElement(null); }} // Deselect element when switching to design
                className={`px-3 py-1 text-sm rounded-md ${designerMode === 'design' ? 'bg-white shadow' : ''}`}
              >
                Design
              </button>
              <button 
                onClick={handlePreviewReport}
                className={`px-3 py-1 text-sm rounded-md ${designerMode === 'preview' ? 'bg-white shadow' : ''}`}
              >
                Preview
              </button>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button onClick={handleSaveReport} className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700">
              <Save className="w-4 h-4" />
              Save
            </button>
            <div className="relative">
              <button onClick={() => handleExportReport('pdf')} className="flex items-center gap-2 px-4 py-2 border border-gray-300 bg-white rounded-lg hover:bg-gray-50">
                <Download className="w-4 h-4" />
                Export
                <ChevronDown className="w-4 h-4" />
              </button>
              {/* Dropdown for more export options would go here */}
            </div>
          </div>
        </div>

        {/* Designer Body */}
        <DndProvider backend={HTML5Backend}>
          <div className="flex-1 flex overflow-hidden">
            {/* Left Panel - Toolbox */}
            <div className="w-64 bg-white border-r p-4 overflow-y-auto">
              <h3 className="font-semibold mb-4">Toolbox</h3>
              <div className="grid grid-cols-2 gap-4">
                {[
                  { icon: Table, label: 'Table', type: 'table' },
                  { icon: BarChart3, label: 'Chart', type: 'chart' },
                  { icon: Type, label: 'Text', type: 'text' },
                  { icon: Image, label: 'Image', type: 'image' },
                  { icon: Layers, label: 'Parameter', type: 'parameter' } // Changed icon for parameter
                ].map(item => (
                  <DraggableToolboxItem key={item.label} type={item.type as ReportElement['type']} label={item.label} icon={item.icon} />
                ))}
              </div>
            </div>

            {/* Center Panel - Canvas */}
            <div className="flex-1 p-8 overflow-auto">
              <div className="bg-white shadow-lg mx-auto w-[8.5in] h-[11in]" onClick={() => setSelectedElement(null)}>
                <DroppableCanvas onDrop={handleAddElement} canvasRef={canvasRef}>
                  {designerMode === 'design' ? (
                    <>
                      {selectedReport.elements.length === 0 && (
                        <p className="text-center text-gray-400 pt-4">Drag elements from the toolbox here</p>
                      )}
                      {selectedReport.elements.map(element => (
                        <ResizableDraggableElement
                          key={element.id}
                          element={element}
                          onUpdate={handleUpdateElement}
                          onSelect={handleSelectElement}
                          isSelected={selectedElement?.id === element.id}
                        />
                      ))}
                    </>
                  ) : (
                    <div className="p-4 h-full">
                      <h2 className="text-2xl font-bold mb-4">Sales Analysis Report</h2>
                      <p className="text-gray-600 mb-6">This is a preview of the report.</p>
                      <div className="border p-4 rounded-lg">
                        <p className="text-center">Report content preview would be rendered here.</p>
                      </div>
                    </div>
                  )}
                </DroppableCanvas>
              </div>
            </div>

            {/* Right Panel - Properties */}
            <div className="w-80 bg-white border-l p-4 overflow-y-auto">
              <h3 className="font-semibold mb-4">Properties</h3>
              {selectedElement ? (
                <div>
                  <p className="text-sm font-medium mb-2">Selected Element: {selectedElement.type}</p>
                  <label htmlFor="selected-element-id" className="block text-sm font-medium mb-1">Element ID</label>
                  <input
                    id="selected-element-id"
                    title="Element ID"
                    placeholder={selectedElement.id}
                    aria-label="Element ID"
                    type="text"
                    value={selectedElement.id}
                    readOnly
                    className="w-full px-2 py-1 border rounded bg-gray-50"
                  />
                  {/* Add more property editors based on selectedElement.type */}
                  {selectedElement.type === 'text' && (
                    <>
                      <label className="block text-sm font-medium mt-3 mb-1">Text Content</label>
                      <textarea
                        className="w-full px-2 py-1 border rounded"
                        value={selectedElement.properties.textContent || ''}
                        onChange={(e) => {
                          setSelectedReport(prev => {
                            if (!prev) return null;
                            return {
                              ...prev,
                              elements: prev.elements.map(el =>
                                el.id === selectedElement.id ? { ...el, properties: { ...el.properties, textContent: e.target.value } } : el
                              ),
                            };
                          });
                          setSelectedElement(prev => prev ? { ...prev, properties: { ...prev.properties, textContent: e.target.value } } : null);
                        }}
                        placeholder="Enter text content"
                        aria-label="Text content"
                      />
                    </>
                  )}
                  {/* Add property editors for other element types (chart, table, image, parameter) */}
                </div>
              ) : (
                <div>
                  <label className="block text-sm font-medium mb-1">Report Name</label>
                  <input
                    type="text"
                    value={selectedReport.name}
                    className="w-full px-2 py-1 border rounded"
                    onChange={(e) => setSelectedReport(prev => prev ? { ...prev, name: e.target.value } : null)}
                    placeholder="Report name"
                    aria-label="Report name"
                  />
                  {/* Other report-level properties */}
                  <label className="block text-sm font-medium mt-3 mb-1">Description</label>
                  <textarea
                    className="w-full px-2 py-1 border rounded"
                    value={selectedReport.description}
                    onChange={(e) => setSelectedReport(prev => prev ? { ...prev, description: e.target.value } : null)}
                    placeholder="Report description"
                    aria-label="Report description"
                  />
                  {/* Add more report-level properties like dataSourceId, pageSettings, etc. */}
                </div>
              )}
            </div>
          </div>
        </DndProvider>
      </div>
    );
  };

  const renderDataSources = () => (
    <div className="p-6 bg-gray-50 h-full">
      <h2 className="text-xl font-semibold mb-4">Data Sources</h2>
      <div className="space-y-4">
        {dataSources.map(ds => (
          <div key={ds.id} className="bg-white p-4 rounded-lg border flex items-center justify-between">
            <div>
              <h3 className="font-semibold flex items-center gap-2">
                <Database className="w-5 h-5 text-blue-600" />
                {ds.name}
              </h3>
              <p className="text-sm text-gray-500 mt-1 font-mono">{ds.connectionString}</p>
            </div>
            <div className="flex gap-2">
              <button className="flex items-center gap-2 px-3 py-1.5 text-sm border rounded-lg hover:bg-gray-50">
                <CheckCircle className="w-4 h-4 text-green-600" />
                Test
              </button>
              <button className="p-2 hover:bg-gray-100 rounded-lg" aria-label="Settings">
                <Settings className="w-5 h-5 text-gray-600" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  const renderActiveTab = () => {
    switch (activeTab) {
      case 'reports':
        return renderReportsList();
      case 'designer':
        return renderDesigner();
      case 'data':
        return renderDataSources();
      default:
        return null;
    }
  };

  return (
    <div className="h-screen w-screen flex bg-gray-100 font-sans">
      {showQueryBuilder && <QueryBuilder onClose={() => setShowQueryBuilder(false)} onApply={(q) => setQueryConfig(q)} />}
      {showParameterConfig && <ParameterConfig onClose={() => setShowParameterConfig(false)} onSave={(_p) => { /* save logic */ }} />}

      {/* Left Navigation */}
      <nav className="w-20 bg-gray-800 text-white flex flex-col items-center py-4">
        <div className="text-2xl font-bold mb-8">R</div>
        <div className="flex flex-col gap-4">
          <button onClick={() => setActiveTab('reports')} className={`p-3 rounded-lg ${activeTab === 'reports' ? 'bg-blue-600' : 'hover:bg-gray-700'}`} aria-label="Reports">
            <FileText className="w-6 h-6" />
          </button>
          <button onClick={() => setActiveTab('designer')} className={`p-3 rounded-lg ${activeTab === 'designer' ? 'bg-blue-600' : 'hover:bg-gray-700'}`} aria-label="Designer">
            <Grid3x3 className="w-6 h-6" />
          </button>
          <button onClick={() => setActiveTab('data')} className={`p-3 rounded-lg ${activeTab === 'data' ? 'bg-blue-600' : 'hover:bg-gray-700'}`} aria-label="Data Sources">
            <Database className="w-6 h-6" />
          </button>
        </div>
      </nav>

      {/* Main Content */}
      <main className="flex-1 flex flex-col">
        {renderActiveTab()}
      </main>
    </div>
  );
};

export default ReportingServerUI;