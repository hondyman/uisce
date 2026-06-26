import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { devError } from '../../utils/devLogger';

interface DynamicDimension {
  name: string;
  type: string;
  description: string;
  source: string;
  cardinality: string;
  usage: string;
}

interface DimensionValue {
  value: string;
  count: number;
  label: string;
}

interface ScopedFilter {
  name: string;
  description: string;
  type: string;
  sql: string;
  source: string;
  category: string;
}

interface DynamicDimensionsProps {
  onDimensionSelect?: (dimension: string, value: string) => void;
  onFilterApply?: (filterName: string, parameters?: any) => void;
}

export const DynamicDimensions: React.FC<DynamicDimensionsProps> = ({
  onDimensionSelect,
  onFilterApply
}) => {
  const [dimensions, setDimensions] = useState<DynamicDimension[]>([]);
  const [selectedDimension, setSelectedDimension] = useState<string>('');
  const [dimensionValues, setDimensionValues] = useState<DimensionValue[]>([]);
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [filters, setFilters] = useState<ScopedFilter[]>([]);
  const [activeTab, setActiveTab] = useState<'dimensions' | 'filters'>('dimensions');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    loadDimensions();
    loadFilters();
  }, []);

  useEffect(() => {
    if (selectedDimension) {
      loadDimensionValues(selectedDimension);
    }
  }, [selectedDimension]);

  const loadDimensions = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/dimensions');
      setDimensions(response.data.dimensions || []);
    } catch (error) {
      devError('Failed to load dimensions:', error);
      setDimensions([]);
    } finally {
      setLoading(false);
    }
  };

  const loadDimensionValues = async (dimensionName: string) => {
    try {
      const response = await axios.get(`/api/dimensions/${dimensionName}/values`);
      setDimensionValues(response.data.values || []);
    } catch (error) {
      devError('Failed to load dimension values:', error);
      setDimensionValues([]);
    }
  };

  const loadFilters = async () => {
    try {
      const response = await axios.get('/api/filters/scoped');
      setFilters(response.data.filters || []);
    } catch (error) {
      devError('Failed to load filters:', error);
      setFilters([]);
    }
  };

  const handleDimensionSelect = (dimension: string, value: string) => {
    setSelectedValue(value);
    if (onDimensionSelect) {
      onDimensionSelect(dimension, value);
    }
  };

  const handleFilterApply = async (filterName: string) => {
    try {
      const response = await axios.post('/api/filters/scoped/apply', {
        filter_name: filterName,
        base_query: "SELECT * FROM target_table",
        parameters: {}
      });

      if (onFilterApply) {
        onFilterApply(filterName, response.data);
      }
    } catch (error) {
      devError('Failed to apply filter:', error);
    }
  };

  const getCardinalityColor = (cardinality: string) => {
    switch (cardinality) {
      case 'low': return 'bg-green-100 text-green-800';
      case 'medium': return 'bg-yellow-100 text-yellow-800';
      case 'high': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  const getUsageIcon = (usage: string) => {
    switch (usage) {
      case 'segmentation': return '👥';
      case 'filtering': return '🔍';
      case 'grouping': return '📊';
      default: return '📋';
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-6">
        <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600"></div>
      </div>
    );
  }

  return (
    <div className="dynamic-dimensions bg-white rounded-lg shadow-lg">
      <div className="px-6 py-4 border-b border-gray-200">
        <h2 className="text-lg font-semibold text-gray-900">
          🎯 Dynamic Dimensions & Scoped Filters
        </h2>
        <p className="text-sm text-gray-600 mt-1">
          Select dimensions and apply scoped filters for targeted analysis
        </p>
      </div>

      {/* Tab Navigation */}
      <div className="flex space-x-1 bg-gray-100 p-1 rounded-lg mx-6 mt-4">
        <button
          onClick={() => setActiveTab('dimensions')}
          className={`flex-1 py-2 px-4 text-sm font-medium rounded-md transition-colors ${
            activeTab === 'dimensions'
              ? 'bg-white text-indigo-600 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          Dimensions
        </button>
        <button
          onClick={() => setActiveTab('filters')}
          className={`flex-1 py-2 px-4 text-sm font-medium rounded-md transition-colors ${
            activeTab === 'filters'
              ? 'bg-white text-indigo-600 shadow-sm'
              : 'text-gray-500 hover:text-gray-700'
          }`}
        >
          Scoped Filters
        </button>
      </div>

      <div className="p-6">
        {activeTab === 'dimensions' ? (
          <div className="dimensions-section">
            <div className="mb-6">
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Select Dimension
              </label>
              <select
                value={selectedDimension}
                onChange={(e) => setSelectedDimension(e.target.value)}
                className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                aria-label="Select dynamic dimension for analysis"
              >
                <option value="">Choose a dimension...</option>
                {dimensions.map(dimension => (
                  <option key={dimension.name} value={dimension.name}>
                    {dimension.name} - {dimension.description}
                  </option>
                ))}
              </select>
            </div>

            {selectedDimension && (
              <div className="dimension-details mb-6">
                <div className="bg-gray-50 rounded-md p-4 mb-4">
                  <h4 className="text-sm font-medium text-gray-900 mb-2">Dimension Details</h4>
                  {(() => {
                    const dim = dimensions.find(d => d.name === selectedDimension);
                    return dim ? (
                      <div className="grid grid-cols-2 gap-4 text-sm">
                        <div>
                          <span className="font-medium text-gray-600">Type:</span>
                          <span className="ml-2 text-gray-900">{dim.type}</span>
                        </div>
                        <div>
                          <span className="font-medium text-gray-600">Source:</span>
                          <span className="ml-2 text-gray-900">{dim.source}</span>
                        </div>
                        <div>
                          <span className="font-medium text-gray-600">Cardinality:</span>
                          <span className={`ml-2 px-2 py-1 text-xs rounded-full ${getCardinalityColor(dim.cardinality)}`}>
                            {dim.cardinality}
                          </span>
                        </div>
                        <div>
                          <span className="font-medium text-gray-600">Usage:</span>
                          <span className="ml-2 text-gray-900">{getUsageIcon(dim.usage)} {dim.usage}</span>
                        </div>
                      </div>
                    ) : null;
                  })()}
                </div>

                <div className="dimension-values">
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Select Value
                  </label>
                  <select
                    value={selectedValue}
                    onChange={(e) => handleDimensionSelect(selectedDimension, e.target.value)}
                    className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
                    aria-label="Select dimension value for filtering"
                  >
                    <option value="">Choose a value...</option>
                    {dimensionValues.map(value => (
                      <option key={value.value} value={value.value}>
                        {value.label}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="filters-section">
            <div className="space-y-4">
              {filters.map(filter => (
                <div key={filter.name} className="filter-item bg-gray-50 rounded-md p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex-1">
                      <h4 className="text-sm font-medium text-gray-900 mb-1">
                        {filter.name}
                      </h4>
                      <p className="text-sm text-gray-600 mb-2">
                        {filter.description}
                      </p>
                      <div className="flex items-center space-x-4 text-xs text-gray-500">
                        <span>Category: {filter.category}</span>
                        <span>Source: {filter.source}</span>
                      </div>
                    </div>
                    <button
                      onClick={() => handleFilterApply(filter.name)}
                      className="px-3 py-1 text-sm bg-indigo-600 text-white rounded hover:bg-indigo-700 transition-colors"
                    >
                      Apply Filter
                    </button>
                  </div>

                  <div className="bg-white rounded border p-2">
                    <code className="text-xs text-gray-800 font-mono">
                      {filter.sql}
                    </code>
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Info Panel */}
      <div className="px-6 py-4 bg-blue-50 border-t border-gray-200">
        <h5 className="text-sm font-medium text-blue-900 mb-2">ℹ️ How it works</h5>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• <strong>Dimensions:</strong> Dynamic attributes for grouping and filtering data</li>
          <li>• <strong>Scoped Filters:</strong> Pre-defined filters for common analysis patterns</li>
          <li>• <strong>Cardinality:</strong> Indicates data distribution (low/medium/high)</li>
          <li>• <strong>Real-time Values:</strong> Dimension values loaded dynamically from database</li>
        </ul>
      </div>
    </div>
  );
};
