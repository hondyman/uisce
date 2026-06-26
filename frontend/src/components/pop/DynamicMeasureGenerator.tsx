import React, { useState, useEffect, useCallback } from 'react';
import axios from 'axios';
import { devError } from '../../utils/devLogger';

interface DynamicMeasure {
  name: string;
  type: string;
  sql: string;
  parameters?: any[];
  meta?: Record<string, any>;
}

interface MeasureCatalog {
  id: string;
  name: string;
  source_table: string;
  source_column: string;
  measures: string[];
  last_updated: string;
  golden_path: boolean;
}

interface DynamicMeasureGeneratorProps {
  onMeasuresGenerated?: (measures: DynamicMeasure[]) => void;
  onMeasureSelected?: (measure: DynamicMeasure) => void;
  className?: string;
}

export const DynamicMeasureGenerator: React.FC<DynamicMeasureGeneratorProps> = ({
  onMeasuresGenerated,
  onMeasureSelected,
  className = ''
}) => {
  const [catalog, setCatalog] = useState<MeasureCatalog[]>([]);
  const [selectedCatalog, setSelectedCatalog] = useState<MeasureCatalog | null>(null);
  const [generatedMeasures, setGeneratedMeasures] = useState<DynamicMeasure[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [customTable, setCustomTable] = useState('');
  const [customColumn, setCustomColumn] = useState('');

  // Load measure catalog on mount
  useEffect(() => {
    loadMeasureCatalog();
  }, []);

  const loadMeasureCatalog = async () => {
    try {
      const response = await axios.get('/api/measures/catalog');
      setCatalog(response.data.catalog || []);
    } catch (err) {
      devError('Failed to load measure catalog:', err);
      setError('Failed to load measure catalog');
    }
  };

  const generateMeasuresFromCatalog = useCallback(async (catalogItem: MeasureCatalog) => {
    setLoading(true);
    setError(null);

    try {
      const response = await axios.get('/api/measures/generate', {
        params: {
          table: catalogItem.source_table,
          column: catalogItem.source_column
        }
      });

      const measures = response.data.measures || [];
      setGeneratedMeasures(measures);
      setSelectedCatalog(catalogItem);
      onMeasuresGenerated?.(measures);
    } catch (err) {
      devError('Failed to generate measures:', err);
      setError('Failed to generate measures from catalog');
    } finally {
      setLoading(false);
    }
  }, [onMeasuresGenerated]);

  const generateCustomMeasures = useCallback(async () => {
    if (!customTable || !customColumn) {
      setError('Please provide both table and column names');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await axios.get('/api/measures/generate', {
        params: {
          table: customTable,
          column: customColumn
        }
      });

      const measures = response.data.measures || [];
      setGeneratedMeasures(measures);
      setSelectedCatalog(null);
      onMeasuresGenerated?.(measures);
    } catch (err) {
      devError('Failed to generate custom measures:', err);
      setError('Failed to generate measures from custom source');
    } finally {
      setLoading(false);
    }
  }, [customTable, customColumn, onMeasuresGenerated]);

  const validateMeasure = async (measure: DynamicMeasure) => {
    try {
      const response = await axios.post('/api/measures/validate', measure);
      return response.data.valid;
    } catch (err) {
      devError('Failed to validate measure:', err);
      return false;
    }
  };

  const handleMeasureSelect = async (measure: DynamicMeasure) => {
    const isValid = await validateMeasure(measure);
    if (isValid) {
      onMeasureSelected?.(measure);
    } else {
      setError(`Measure "${measure.name}" failed validation`);
    }
  };

  return (
    <div className={`dynamic-measure-generator space-y-6 ${className}`}>
      <div className="flex justify-between items-center">
        <h3 className="text-lg font-semibold text-gray-900">Dynamic Measure Generator</h3>
        <div className="text-sm text-gray-500">
          Generate measures from database enums
        </div>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-md p-4">
          <div className="text-red-800 text-sm">{error}</div>
        </div>
      )}

      {/* Catalog Selection */}
      <div className="bg-white border rounded-lg p-4">
        <h4 className="text-md font-medium text-gray-900 mb-3">Available Catalogs</h4>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {catalog.map((item) => (
            <div
              key={item.id}
              className={`border rounded-lg p-3 cursor-pointer transition-colors ${
                selectedCatalog?.id === item.id
                  ? 'border-blue-500 bg-blue-50'
                  : 'border-gray-200 hover:border-gray-300'
              }`}
              onClick={() => generateMeasuresFromCatalog(item)}
            >
              <div className="flex justify-between items-start">
                <div>
                  <h5 className="font-medium text-gray-900">{item.name}</h5>
                  <p className="text-sm text-gray-600">
                    {item.source_table}.{item.source_column}
                  </p>
                  <p className="text-xs text-gray-500 mt-1">
                    {item.measures.length} measures • Updated {new Date(item.last_updated).toLocaleDateString()}
                  </p>
                </div>
                {item.golden_path && (
                  <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                    Golden Path
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Custom Generation */}
      <div className="bg-white border rounded-lg p-4">
        <h4 className="text-md font-medium text-gray-900 mb-3">Custom Generation</h4>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Table Name
            </label>
            <input
              type="text"
              value={customTable}
              onChange={(e) => setCustomTable(e.target.value)}
              placeholder="e.g., orders"
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Column Name
            </label>
            <input
              type="text"
              value={customColumn}
              onChange={(e) => setCustomColumn(e.target.value)}
              placeholder="e.g., status"
              className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
          <div className="flex items-end">
            <button
              onClick={generateCustomMeasures}
              disabled={loading || !customTable || !customColumn}
              className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Generating...' : 'Generate Measures'}
            </button>
          </div>
        </div>
      </div>

      {/* Generated Measures */}
      {generatedMeasures.length > 0 && (
        <div className="bg-white border rounded-lg p-4">
          <div className="flex justify-between items-center mb-3">
            <h4 className="text-md font-medium text-gray-900">
              Generated Measures ({generatedMeasures.length})
            </h4>
            {selectedCatalog && (
              <span className="text-sm text-gray-600">
                From: {selectedCatalog.source_table}.{selectedCatalog.source_column}
              </span>
            )}
          </div>

          <div className="space-y-3">
            {generatedMeasures.map((measure, index) => (
              <div
                key={index}
                className="border border-gray-200 rounded-lg p-3 hover:border-gray-300 transition-colors"
              >
                <div className="flex justify-between items-start">
                  <div className="flex-1">
                    <h5 className="font-medium text-gray-900">{measure.name}</h5>
                    <p className="text-sm text-gray-600 mt-1">
                      Type: {measure.type} • SQL: {measure.sql}
                    </p>
                    {measure.meta && (
                      <div className="text-xs text-gray-500 mt-2">
                        Source: {measure.meta.source_table}.{measure.meta.source_column} •
                        Value: {measure.meta.filter_value}
                      </div>
                    )}
                  </div>
                  <button
                    onClick={() => handleMeasureSelect(measure)}
                    className="ml-4 px-3 py-1 bg-green-600 text-white text-sm rounded hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-green-500"
                  >
                    Select
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Usage Instructions */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h4 className="text-md font-medium text-blue-900 mb-2">How to Use</h4>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• Select a catalog item to generate measures from predefined sources</li>
          <li>• Or use custom generation to create measures from any table/column</li>
          <li>• Generated measures are automatically validated for security</li>
          <li>• Click "Select" to use a measure in your queries or dashboards</li>
          <li>• Golden Path catalogs are pre-approved and optimized</li>
        </ul>
      </div>
    </div>
  );
};

export default DynamicMeasureGenerator;
