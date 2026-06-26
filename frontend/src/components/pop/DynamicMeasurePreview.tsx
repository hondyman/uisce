import React, { useEffect, useState } from 'react';
import axios from 'axios';
import { devError } from '../../utils/devLogger';

interface StatusOption {
  value: string;
  label: string;
}

interface CubeQueryProps {
  query: any;
}

const CubeQuery: React.FC<CubeQueryProps> = ({ query }) => {
  const [data, setData] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (query) {
      setLoading(true);
      axios.post('/api/cube/query', query)
        .then(response => {
          setData(response.data);
          setLoading(false);
        })
        .catch(error => {
          devError('Cube query error:', error);
          setLoading(false);
        });
    }
  }, [query]);

  if (loading) return <div>Loading...</div>;
  if (!data) return null;

  return (
    <div className="cube-query-result">
      <pre>{JSON.stringify(data, null, 2)}</pre>
    </div>
  );
};

export const DynamicMeasurePreview: React.FC = () => {
  const [statuses, setStatuses] = useState<StatusOption[]>([]);
  const [categories, setCategories] = useState<StatusOption[]>([]);
  const [deviceTypes, setDeviceTypes] = useState<StatusOption[]>([]);
  const [selectedStatus, setSelectedStatus] = useState<string>("");
  const [selectedCategory, setSelectedCategory] = useState<string>("");
  const [selectedDeviceType, setSelectedDeviceType] = useState<string>("");
  const [activeTab, setActiveTab] = useState<'status' | 'category' | 'device'>('status');

  useEffect(() => {
    // Fetch available statuses
    axios.get("/api/parameters/dimension/status/values")
      .then(res => {
        const statusOptions = res.data.values.map((status: string) => ({
          value: status,
          label: status.charAt(0).toUpperCase() + status.slice(1)
        }));
        setStatuses(statusOptions);
      })
  .catch(err => devError('Failed to fetch statuses:', err));

    // Fetch available categories
    axios.get("/api/parameters/dimension/category/values")
      .then(res => {
        const categoryOptions = res.data.values.map((category: string) => ({
          value: category,
          label: category.charAt(0).toUpperCase() + category.slice(1)
        }));
        setCategories(categoryOptions);
      })
  .catch(err => devError('Failed to fetch categories:', err));

    // Fetch available device types
    axios.get("/api/parameters/dimension/device_type/values")
      .then(res => {
        const deviceOptions = res.data.values.map((device: string) => ({
          value: device,
          label: device.charAt(0).toUpperCase() + device.slice(1)
        }));
        setDeviceTypes(deviceOptions);
      })
  .catch(err => devError('Failed to fetch device types:', err));
  }, []);

  const renderMeasureSelector = () => {
    switch (activeTab) {
      case 'status':
        return (
          <div className="measure-selector">
            <label htmlFor="status-select" className="block text-sm font-medium text-gray-700 mb-2">
              Select Order Status
            </label>
            <select
              id="status-select"
              value={selectedStatus}
              onChange={e => setSelectedStatus(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
              aria-label="Select order status to preview dynamic measure"
            >
              <option value="">Choose a status...</option>
              {statuses.map(status => (
                <option key={status.value} value={status.value}>
                  {status.label}
                </option>
              ))}
            </select>
          </div>
        );

      case 'category':
        return (
          <div className="measure-selector">
            <label htmlFor="category-select" className="block text-sm font-medium text-gray-700 mb-2">
              Select Product Category
            </label>
            <select
              id="category-select"
              value={selectedCategory}
              onChange={e => setSelectedCategory(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
              aria-label="Select product category to preview dynamic measure"
            >
              <option value="">Choose a category...</option>
              {categories.map(category => (
                <option key={category.value} value={category.value}>
                  {category.label}
                </option>
              ))}
            </select>
          </div>
        );

      case 'device':
        return (
          <div className="measure-selector">
            <label htmlFor="device-select" className="block text-sm font-medium text-gray-700 mb-2">
              Select Device Type
            </label>
            <select
              id="device-select"
              value={selectedDeviceType}
              onChange={e => setSelectedDeviceType(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-indigo-500 focus:border-indigo-500"
              aria-label="Select device type to preview dynamic measure"
            >
              <option value="">Choose a device type...</option>
              {deviceTypes.map(device => (
                <option key={device.value} value={device.value}>
                  {device.label}
                </option>
              ))}
            </select>
          </div>
        );

      default:
        return null;
    }
  };

  const getCurrentQuery = () => {
    const timeDimensions = [{ dimension: "orders.created_at", granularity: "month" }];

    switch (activeTab) {
      case 'status':
        if (!selectedStatus) return null;
        return {
          measures: [`total_${selectedStatus.toLowerCase().replace(' ', '_')}_order`],
          timeDimensions
        };

      case 'category':
        if (!selectedCategory) return null;
        return {
          measures: [`total_${selectedCategory.toLowerCase().replace(' ', '_')}_product`],
          timeDimensions
        };

      case 'device':
        if (!selectedDeviceType) return null;
        return {
          measures: [`total_${selectedDeviceType.toLowerCase().replace(' ', '_')}_click`],
          timeDimensions
        };

      default:
        return null;
    }
  };

  return (
    <div className="dynamic-measure-preview bg-white rounded-lg shadow-lg p-6">
      <div className="mb-6">
        <h3 className="text-lg font-semibold text-gray-900 mb-4">
          📊 Dynamic Measures Preview
        </h3>

        {/* Tab Navigation */}
        <div className="flex space-x-1 bg-gray-100 p-1 rounded-lg mb-4">
          <button
            onClick={() => setActiveTab('status')}
            className={`flex-1 py-2 px-4 text-sm font-medium rounded-md transition-colors ${
              activeTab === 'status'
                ? 'bg-white text-indigo-600 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            Order Status
          </button>
          <button
            onClick={() => setActiveTab('category')}
            className={`flex-1 py-2 px-4 text-sm font-medium rounded-md transition-colors ${
              activeTab === 'category'
                ? 'bg-white text-indigo-600 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            Product Category
          </button>
          <button
            onClick={() => setActiveTab('device')}
            className={`flex-1 py-2 px-4 text-sm font-medium rounded-md transition-colors ${
              activeTab === 'device'
                ? 'bg-white text-indigo-600 shadow-sm'
                : 'text-gray-500 hover:text-gray-700'
            }`}
          >
            Device Type
          </button>
        </div>

        {/* Measure Selector */}
        {renderMeasureSelector()}
      </div>

      {/* Query Results */}
      <div className="query-results">
        <h4 className="text-md font-medium text-gray-900 mb-3">Query Results</h4>
        <div className="bg-gray-50 rounded-md p-4 min-h-[200px]">
          {getCurrentQuery() ? (
            <CubeQuery query={getCurrentQuery()} />
          ) : (
            <div className="text-gray-500 text-center py-8">
              Select a value above to preview the dynamic measure
            </div>
          )}
        </div>
      </div>

      {/* Info Panel */}
      <div className="mt-6 p-4 bg-blue-50 rounded-md">
        <h5 className="text-sm font-medium text-blue-900 mb-2">ℹ️ How it works</h5>
        <ul className="text-sm text-blue-800 space-y-1">
          <li>• Dynamic measures are auto-generated from database enums</li>
          <li>• Each measure counts records matching the selected value</li>
          <li>• Measures are synced to both Cube schema and catalog</li>
          <li>• Steward review required before measures become active</li>
        </ul>
      </div>
    </div>
  );
};
