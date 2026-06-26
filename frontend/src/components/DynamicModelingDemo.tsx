import React, { useState } from 'react';
import { ParameterSelector } from './ParameterSelector';
import { DynamicMeasureGenerator } from './DynamicMeasureGenerator';
import { StewardWorkflow } from './StewardWorkflow';
import { EnhancedDashboard } from './EnhancedDashboard';
import { PoPDashboard } from '../types/dynamic';

interface DynamicParameter {
  name: string;
  type: 'dimension' | 'measure' | 'filter' | 'time_range';
  value?: any;
  defaultValue?: any;
  required: boolean;
  options?: string[];
  description: string;
  source?: string;
}

interface DynamicMeasure {
  name: string;
  type: string;
  sql: string;
  parameters?: any[];
  meta?: Record<string, any>;
}

interface StewardAsset {
  id: string;
  name: string;
  type: 'dynamic_measure' | 'dynamic_parameter' | 'dashboard' | 'query';
  source?: string;
  sql?: string;
  parameters?: any[];
  meta?: Record<string, any>;
  status: 'draft' | 'pending_review' | 'approved' | 'rejected' | 'deprecated';
  created_by: string;
  created_at: string;
  reviewed_by?: string;
  reviewed_at?: string;
  review_notes?: string;
  golden_path: boolean;
}

export const DynamicModelingDemo: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'parameters' | 'measures' | 'steward' | 'dashboard'>('parameters');
  const [parameters, _setParameters] = useState<DynamicParameter[]>([
    {
      name: 'city',
      type: 'dimension',
      description: 'Filter by geographic city',
      required: false,
      source: 'clickstream.city'
    },
    {
      name: 'status',
      type: 'dimension',
      description: 'Filter by order status',
      required: false,
      source: 'orders.status'
    },
    {
      name: 'period',
      type: 'time_range',
      description: 'Time period for analysis',
      required: true,
      options: ['1d', '7d', '30d', '90d', '1y'],
      defaultValue: '30d'
    },
    {
      name: 'active_only',
      type: 'filter',
      description: 'Show only active records',
      required: false,
      defaultValue: true
    }
  ]);
  const [parameterValues, setParameterValues] = useState<Record<string, any>>({});
  const [generatedMeasures, setGeneratedMeasures] = useState<DynamicMeasure[]>([]);
  const [_selectedMeasure, _setSelectedMeasure] = useState<DynamicMeasure | null>(null);
  const [stewardAsset, setStewardAsset] = useState<StewardAsset | null>(null);
  const [currentUser] = useState('demo-user');

  // Sample dashboard for demonstration
  const sampleDashboard: PoPDashboard = {
    id: 'dynamic-demo-dashboard',
    name: 'Dynamic Modeling Demo Dashboard',
    description: 'Demonstration of dynamic parameters and measures',
    ownerUserId: currentUser,
    config: {
      layout: 'grid',
      theme: 'light',
      autoRefresh: false,
      refreshInterval: 60,
      alertThresholds: {}
    },
    defaultFilters: parameterValues,
    isPublic: false,
    allowedGroups: ['analysts'],
    widgets: [
      {
        id: 'demo-kpi-1',
        dashboardId: 'dynamic-demo-dashboard',
        widgetType: 'kpi_cards',
        title: 'Dynamic KPIs',
        position: { x: 0, y: 0, width: 2, height: 1 },
        config: {},
        metricIds: generatedMeasures.slice(0, 3).map(m => m.name)
      },
      {
        id: 'demo-chart-1',
        dashboardId: 'dynamic-demo-dashboard',
        widgetType: 'trend_chart',
        title: 'Dynamic Trends',
        position: { x: 2, y: 0, width: 2, height: 2 },
        config: {},
        metricIds: generatedMeasures.slice(0, 1).map(m => m.name)
      }
    ],
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString()
  };

  const handleParameterChange = (name: string, value: any) => {
    setParameterValues(prev => ({ ...prev, [name]: value }));
  };

  const handleMeasuresGenerated = (measures: DynamicMeasure[]) => {
    setGeneratedMeasures(measures);
  };

  const handleMeasureSelected = (measure: DynamicMeasure) => {
    _setSelectedMeasure(measure);

    // Create steward asset for the selected measure
    const asset: StewardAsset = {
      id: `measure-${Date.now()}`,
      name: measure.name,
      type: 'dynamic_measure',
      source: measure.meta?.source_table ? `${measure.meta.source_table}.${measure.meta.source_column}` : undefined,
      sql: measure.sql,
      meta: measure.meta,
      status: 'draft',
      created_by: currentUser,
      created_at: new Date().toISOString(),
      golden_path: false
    };

    setStewardAsset(asset);
    setActiveTab('steward');
  };

  const handleAssetUpdate = (asset: StewardAsset) => {
    setStewardAsset(asset);
  };

  const tabs = [
    { id: 'parameters', label: 'Dynamic Parameters', icon: '🔧' },
    { id: 'measures', label: 'Dynamic Measures', icon: '🧪' },
    { id: 'steward', label: 'Steward Workflow', icon: '🔐' },
    { id: 'dashboard', label: 'Live Dashboard', icon: '📊' }
  ];

  return (
    <div className="dynamic-modeling-demo min-h-screen bg-gray-50">
      {/* Header */}
      <div className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <div>
              <h1 className="text-3xl font-bold text-gray-900">Dynamic Semantic Modeling</h1>
              <p className="text-gray-600 mt-1">
                Build dynamic parameters, auto-generate measures, and govern your semantic layer
              </p>
            </div>
            <div className="text-sm text-gray-500">
              React + Go + Postgres + Cube
            </div>
          </div>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="bg-white border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <nav className="flex space-x-8">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`py-4 px-1 border-b-2 font-medium text-sm ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                <span className="mr-2">{tab.icon}</span>
                {tab.label}
              </button>
            ))}
          </nav>
        </div>
      </div>

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Parameters Tab */}
        {activeTab === 'parameters' && (
          <div className="space-y-6">
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">Dynamic Parameters</h2>
              <ParameterSelector
                parameters={parameters}
                onParameterChange={handleParameterChange}
              />
            </div>

            <div className="bg-blue-50 border border-blue-200 rounded-lg p-6">
              <h3 className="text-lg font-medium text-blue-900 mb-3">How Dynamic Parameters Work</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <h4 className="font-medium text-blue-800 mb-2">Cube Integration</h4>
                  <pre className="text-xs bg-blue-100 p-3 rounded text-blue-900">
{`measures:
  - name: city_clicks
    type: sum
    sql: clicks
    filters:
      - sql: city = '{FILTER_PARAMS.city}'`}
                  </pre>
                </div>
                <div>
                  <h4 className="font-medium text-blue-800 mb-2">Go API</h4>
                  <pre className="text-xs bg-blue-100 p-3 rounded text-blue-900">
{`func GetAvailableCities(c *gin.Context) {
  rows, _ := db.Query("SELECT DISTINCT city FROM clickstream")
  // Return available values for parameter selection
}`}
                  </pre>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Measures Tab */}
        {activeTab === 'measures' && (
          <div className="space-y-6">
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">Dynamic Measure Generator</h2>
              <DynamicMeasureGenerator
                onMeasuresGenerated={handleMeasuresGenerated}
                onMeasureSelected={handleMeasureSelected}
              />
            </div>

            {generatedMeasures.length > 0 && (
              <div className="bg-green-50 border border-green-200 rounded-lg p-6">
                <h3 className="text-lg font-medium text-green-900 mb-3">
                  Generated Measures ({generatedMeasures.length})
                </h3>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {generatedMeasures.map((measure, index) => (
                    <div key={index} className="bg-white p-4 rounded border">
                      <h4 className="font-medium text-gray-900">{measure.name}</h4>
                      <p className="text-sm text-gray-600 mt-1">Type: {measure.type}</p>
                      <p className="text-xs text-gray-500 mt-2 font-mono">{measure.sql}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="bg-purple-50 border border-purple-200 rounded-lg p-6">
              <h3 className="text-lg font-medium text-purple-900 mb-3">How Dynamic Measures Work</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <h4 className="font-medium text-purple-800 mb-2">Go Generator</h4>
                  <pre className="text-xs bg-purple-100 p-3 rounded text-purple-900">
{`func GenerateStatusMeasures() {
  rows, _ := db.Query("SELECT DISTINCT status FROM orders")
  for rows.Next() {
    var status string
    rows.Scan(&status)
    measures = append(measures, DynamicMeasure{
      Name: fmt.Sprintf("total_%s_orders", status),
      SQL:  fmt.Sprintf("CASE WHEN status = '%s' THEN 1 ELSE 0 END", status),
    })
  }
}`}
                  </pre>
                </div>
                <div>
                  <h4 className="font-medium text-purple-800 mb-2">Cube YAML Output</h4>
                  <pre className="text-xs bg-purple-100 p-3 rounded text-purple-900">
{`measures:
  - name: total_processing_orders
    type: count
    sql: CASE WHEN status = 'processing' THEN 1 ELSE 0 END
  - name: total_shipped_orders
    type: count
    sql: CASE WHEN status = 'shipped' THEN 1 ELSE 0 END`}
                  </pre>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Steward Tab */}
        {activeTab === 'steward' && stewardAsset && (
          <div className="space-y-6">
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">Steward Workflow</h2>
              <StewardWorkflow
                asset={stewardAsset}
                currentUser={currentUser}
                onAssetUpdate={handleAssetUpdate}
              />
            </div>

            <div className="bg-orange-50 border border-orange-200 rounded-lg p-6">
              <h3 className="text-lg font-medium text-orange-900 mb-3">Governance Features</h3>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <div className="text-center">
                  <div className="text-2xl mb-2">🔍</div>
                  <h4 className="font-medium text-orange-800">Review Process</h4>
                  <p className="text-sm text-orange-700 mt-1">
                    Structured review workflow with ratings and action items
                  </p>
                </div>
                <div className="text-center">
                  <div className="text-2xl mb-2">✅</div>
                  <h4 className="font-medium text-orange-800">Golden Path</h4>
                  <p className="text-sm text-orange-700 mt-1">
                    Mark approved measures as golden path for team usage
                  </p>
                </div>
                <div className="text-center">
                  <div className="text-2xl mb-2">📊</div>
                  <h4 className="font-medium text-orange-800">Audit Trail</h4>
                  <p className="text-sm text-orange-700 mt-1">
                    Complete history of changes and steward decisions
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Dashboard Tab */}
        {activeTab === 'dashboard' && (
          <div className="space-y-6">
            <div className="bg-white rounded-lg shadow p-6">
              <h2 className="text-xl font-semibold text-gray-900 mb-4">Live Dashboard</h2>
              <p className="text-gray-600 mb-6">
                Dashboard using your dynamic parameters and generated measures
              </p>
              <EnhancedDashboard
                dashboard={sampleDashboard}
                onParameterChange={handleParameterChange}
                enableRealTime={false}
                enablePredictive={false}
                enableCaching={true}
              />
            </div>

            <div className="bg-gray-50 border border-gray-200 rounded-lg p-6">
              <h3 className="text-lg font-medium text-gray-900 mb-3">Integration Summary</h3>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                <div>
                  <h4 className="font-medium text-gray-800 mb-2">Current Parameters</h4>
                  <pre className="text-xs bg-white p-3 rounded border text-gray-700">
                    {JSON.stringify(parameterValues, null, 2)}
                  </pre>
                </div>
                <div>
                  <h4 className="font-medium text-gray-800 mb-2">Generated Measures</h4>
                  <pre className="text-xs bg-white p-3 rounded border text-gray-700">
                    {JSON.stringify(generatedMeasures.slice(0, 2), null, 2)}
                  </pre>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default DynamicModelingDemo;
