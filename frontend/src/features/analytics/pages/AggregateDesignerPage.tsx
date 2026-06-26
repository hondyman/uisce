import React, { useState } from 'react';
import { devDebug } from '../../../utils/devLogger';

interface AggregateDefinition {
  name: string;
  sourceTable: string;
  dimensions: string[];
  measures: string[];
  filter: string;
  target: 'StarRocks' | 'Cube' | 'Both';
}

export const AggregateDesignerPage: React.FC = () => {
  const [definition, setDefinition] = useState<AggregateDefinition>({
    name: '',
    sourceTable: 'trades',
    dimensions: [],
    measures: [],
    filter: '',
    target: 'Both',
  });

  const [previewData, setPreviewData] = useState<any[]>([]);
  const [generatedSQL, setGeneratedSQL] = useState('');
  const [generatedCube, setGeneratedCube] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSave = async () => {
    setLoading(true);
    setError(null);
    try {
      devDebug('Saving Aggregate:', definition);
      
      const response = await fetch('/api/analytics/aggregates', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
      });

      if (!response.ok) {
        throw new Error(`Failed to save aggregate: ${response.statusText}`);
      }

      const result = await response.json();
      
      // Update UI with generated artifacts returned from backend
      if (result.starrocks_sql) setGeneratedSQL(result.starrocks_sql);
      if (result.cube_schema) setGeneratedCube(result.cube_schema);

      alert('Aggregate Saved Successfully! Audit Record Created.');
    } catch (err: any) {
      setError(err.message);
      console.error('Save failed:', err);
    } finally {
      setLoading(false);
    }
  };

  const handlePreview = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/analytics/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(definition),
      });

      if (!response.ok) {
        throw new Error(`Failed to fetch preview: ${response.statusText}`);
      }

      const data = await response.json();
      setPreviewData(data);
    } catch (err: any) {
      setError(err.message);
      console.error('Preview failed:', err);
      // Fallback for demo if API is not yet live
      setPreviewData([
          { desk_id: 'DESK-A', total_pnl: 1234.56, note: 'Mock Data (API Failed)' },
          { desk_id: 'DESK-B', total_pnl: 987.65, note: 'Mock Data (API Failed)' },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-6 bg-gray-50 min-h-screen">
      <h1 className="text-2xl font-bold mb-6 text-gray-800">Aggregate Designer (Dual-Mode)</h1>
      
      {error && (
        <div className="mb-4 p-4 bg-red-100 text-red-700 rounded-md">
          Error: {error}
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Designer Panel */}
        <div className="bg-white p-6 rounded-lg shadow-md">
          <h2 className="text-xl font-semibold mb-4">Definition</h2>
          
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700">Aggregate Name</label>
              <input 
                type="text" 
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                value={definition.name}
                onChange={(e) => setDefinition({...definition, name: e.target.value})}
                placeholder="e.g., daily_pnl_by_desk"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Source Table</label>
              <select 
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                value={definition.sourceTable}
                onChange={(e) => setDefinition({...definition, sourceTable: e.target.value})}
              >
                <option value="trades">trades</option>
                <option value="compliance_events">compliance_events</option>
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Dimensions (Comma separated)</label>
              <input 
                type="text" 
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                placeholder="desk_id, trade_date"
                onChange={(e) => setDefinition({...definition, dimensions: e.target.value.split(',').map(s => s.trim())})}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Measures (Comma separated)</label>
              <input 
                type="text" 
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                placeholder="SUM(pnl), AVG(price)"
                onChange={(e) => setDefinition({...definition, measures: e.target.value.split(',').map(s => s.trim())})}
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">Deployment Target</label>
              <div className="mt-2 space-x-4">
                <label className="inline-flex items-center">
                  <input type="radio" className="form-radio" name="target" value="StarRocks" checked={definition.target === 'StarRocks'} onChange={() => setDefinition({...definition, target: 'StarRocks'})} />
                  <span className="ml-2">StarRocks (Lakehouse)</span>
                </label>
                <label className="inline-flex items-center">
                  <input type="radio" className="form-radio" name="target" value="Cube" checked={definition.target === 'Cube'} onChange={() => setDefinition({...definition, target: 'Cube'})} />
                  <span className="ml-2">Cube (Semantic)</span>
                </label>
                <label className="inline-flex items-center">
                  <input type="radio" className="form-radio" name="target" value="Both" checked={definition.target === 'Both'} onChange={() => setDefinition({...definition, target: 'Both'})} />
                  <span className="ml-2">Both</span>
                </label>
              </div>
            </div>

            <div className="flex space-x-3 pt-4">
                <button 
                  onClick={handlePreview} 
                  disabled={loading}
                  className={`px-4 py-2 rounded ${loading ? 'bg-gray-300' : 'bg-blue-100 text-blue-700 hover:bg-blue-200'}`}
                >
                  {loading ? 'Loading...' : 'Preview Results'}
                </button>
                <button 
                  onClick={handleSave} 
                  disabled={loading}
                  className={`px-4 py-2 rounded ${loading ? 'bg-gray-300' : 'bg-blue-600 text-white hover:bg-blue-700'}`}
                >
                  {loading ? 'Saving...' : 'Save Aggregate'}
                </button>
            </div>
          </div>
        </div>

        {/* Preview & Output Panel */}
        <div className="space-y-6">
            {/* Preview */}
            <div className="bg-white p-6 rounded-lg shadow-md">
                <h2 className="text-xl font-semibold mb-4">Preview</h2>
                {previewData.length > 0 ? (
                    <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                            <tr>
                                {Object.keys(previewData[0]).map(k => (
                                    <th key={k} className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">{k}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-gray-200">
                            {previewData.map((row, i) => (
                                <tr key={i}>
                                    {Object.values(row).map((v: any, j) => (
                                        <td key={j} className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{v}</td>
                                    ))}
                                </tr>
                            ))}
                        </tbody>
                    </table>
                ) : (
                    <p className="text-gray-500 italic">Run preview to see sample data.</p>
                )}
            </div>

            {/* Generated Artifacts */}
            {(generatedSQL || generatedCube) && (
                <div className="bg-gray-800 text-gray-100 p-6 rounded-lg shadow-md font-mono text-sm overflow-auto">
                    <h2 className="text-xl font-semibold mb-4 text-white">Generated Artifacts</h2>
                    {generatedSQL && (
                        <div className="mb-4">
                            <h3 className="text-green-400 mb-2">StarRocks SQL (Iceberg)</h3>
                            <pre>{generatedSQL}</pre>
                        </div>
                    )}
                    {generatedCube && (
                        <div>
                            <h3 className="text-purple-400 mb-2">Cube Schema</h3>
                            <pre>{generatedCube}</pre>
                        </div>
                    )}
                </div>
            )}
        </div>
      </div>
    </div>
  );
};