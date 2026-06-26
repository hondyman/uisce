import React, { useState, useEffect } from 'react';
import { Play, Code, Download, TrendingUp, Calendar, Filter as FilterIcon, Plus, X } from 'lucide-react';

interface Cube {
  id: string;
  name: string;
  display_name: string;
  dimensions: Dimension[];
  measures: Measure[];
}

interface Dimension {
  name: string;
  display_name: string;
  type: string;
}

interface Measure {
  name: string;
  display_name: string;
  type: string;
}

interface QueryFilter {
  member: string;
  operator: string;
  values: string[];
}

const SemanticQueryBuilder: React.FC = () => {
  const [cubes, setCubes] = useState<Cube[]>([]);
  const [selectedCube, setSelectedCube] = useState<Cube | null>(null);
  const [selectedMeasures, setSelectedMeasures] = useState<string[]>([]);
  const [selectedDimensions, setSelectedDimensions] = useState<string[]>([]);
  const [filters, setFilters] = useState<QueryFilter[]>([]);
  const [timeDimension, setTimeDimension] = useState<string>('');
  const [timeGranularity, setTimeGranularity] = useState<string>('day');
  const [limit, setLimit] = useState<number>(100);
  const [queryResult, setQueryResult] = useState<any>(null);
  const [generatedSQL, setGeneratedSQL] = useState<string>('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    fetchCubes();
  }, []);

  const fetchCubes = async () => {
    try {
      const response = await fetch('/api/semantic/cubes', {
        headers: { 'X-Tenant-ID': 'default-tenant' },
      });
      const data = await response.json();
      setCubes(data);
      if (data.length > 0) {
        setSelectedCube(data[0]);
      }
    } catch (error) {
      console.error('Failed to fetch cubes:', error);
    }
  };

  const executeQuery = async () => {
    if (!selectedCube || selectedMeasures.length === 0) return;

    setLoading(true);
    try {
      const query = {
        measures: selectedMeasures,
        dimensions: selectedDimensions,
        timeDimensions: timeDimension ? [{
          dimension: timeDimension,
          granularity: timeGranularity,
        }] : [],
        filters: filters,
        limit: limit,
      };

      const response = await fetch('/api/semantic/query', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': 'default-tenant',
        },
        body: JSON.stringify(query),
      });

      const result = await response.json();
      setQueryResult(result);
      setGeneratedSQL(result.annotation?.generatedSQL || '');
    } catch (error) {
      console.error('Failed to execute query:', error);
    } finally {
      setLoading(false);
    }
  };

  const toggleMeasure = (measure: string) => {
    setSelectedMeasures(prev =>
      prev.includes(measure) ? prev.filter(m => m !== measure) : [...prev, measure]
    );
  };

  const toggleDimension = (dimension: string) => {
    setSelectedDimensions(prev =>
      prev.includes(dimension) ? prev.filter(d => d !== dimension) : [...prev, dimension]
    );
  };

  const addFilter = () => {
    setFilters([...filters, { member: '', operator: 'equals', values: [''] }]);
  };

  const removeFilter = (index: number) => {
    setFilters(filters.filter((_, i) => i !== index));
  };

  const updateFilter = (index: number, field: keyof QueryFilter, value: any) => {
    const newFilters = [...filters];
    newFilters[index] = { ...newFilters[index], [field]: value };
    setFilters(newFilters);
  };

  return (
    <div className="h-screen flex flex-col bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
      {/* Header */}
      <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 px-6 py-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <div className="p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl">
              <TrendingUp className="w-6 h-6 text-white" />
            </div>
            <div>
              <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                Query Builder
              </h1>
              <p className="text-sm text-slate-600 dark:text-slate-400">
                Build and execute semantic queries
              </p>
            </div>
          </div>

          <button
            onClick={executeQuery}
            disabled={loading || selectedMeasures.length === 0}
            className="flex items-center space-x-2 px-6 py-3 bg-gradient-to-r from-blue-500 to-indigo-600 text-white rounded-xl hover:shadow-lg transition-all disabled:opacity-50"
          >
            <Play className="w-5 h-5" />
            <span className="font-semibold">Run Query</span>
          </button>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Left Panel - Query Builder */}
        <div className="w-1/3 border-r border-slate-200 dark:border-slate-700 overflow-y-auto p-6 space-y-6">
          {/* Cube Selector */}
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Data Source
            </label>
            <select
              value={selectedCube?.name || ''}
              onChange={(e) => {
                const cube = cubes.find(c => c.name === e.target.value);
                setSelectedCube(cube || null);
                setSelectedMeasures([]);
                setSelectedDimensions([]);
              }}
              className="w-full px-4 py-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {cubes.map(cube => (
                <option key={cube.id} value={cube.name}>
                  {cube.display_name}
                </option>
              ))}
            </select>
          </div>

          {/* Measures */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 flex items-center">
              <div className="w-2 h-2 bg-green-500 rounded-full mr-2"></div>
              Measures
            </h3>
            <div className="space-y-2">
              {selectedCube?.measures.map(measure => (
                <button
                  key={measure.name}
                  onClick={() => toggleMeasure(`${selectedCube.name}.${measure.name}`)}
                  className={`w-full text-left px-4 py-3 rounded-lg border-2 transition-all ${
                    selectedMeasures.includes(`${selectedCube.name}.${measure.name}`)
                      ? 'border-green-500 bg-green-50 dark:bg-green-950/20'
                      : 'border-slate-200 dark:border-slate-700 hover:border-green-300'
                  }`}
                >
                  <div className="font-medium text-slate-900 dark:text-white">{measure.display_name}</div>
                  <div className="text-xs text-slate-500 dark:text-slate-500">{measure.type}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Dimensions */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 flex items-center">
              <div className="w-2 h-2 bg-blue-500 rounded-full mr-2"></div>
              Dimensions
            </h3>
            <div className="space-y-2">
              {selectedCube?.dimensions.filter(d => d.type !== 'time').map(dimension => (
                <button
                  key={dimension.name}
                  onClick={() => toggleDimension(`${selectedCube.name}.${dimension.name}`)}
                  className={`w-full text-left px-4 py-3 rounded-lg border-2 transition-all ${
                    selectedDimensions.includes(`${selectedCube.name}.${dimension.name}`)
                      ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/20'
                      : 'border-slate-200 dark:border-slate-700 hover:border-blue-300'
                  }`}
                >
                  <div className="font-medium text-slate-900 dark:text-white">{dimension.display_name}</div>
                  <div className="text-xs text-slate-500 dark:text-slate-500">{dimension.type}</div>
                </button>
              ))}
            </div>
          </div>

          {/* Time Dimension */}
          <div>
            <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 mb-3 flex items-center">
              <Calendar className="w-4 h-4 mr-2" />
              Time Dimension
            </h3>
            <select
              value={timeDimension}
              onChange={(e) => setTimeDimension(e.target.value)}
              className="w-full px-4 py-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 mb-2"
            >
              <option value="">None</option>
              {selectedCube?.dimensions.filter(d => d.type === 'time').map(dimension => (
                <option key={dimension.name} value={`${selectedCube.name}.${dimension.name}`}>
                  {dimension.display_name}
                </option>
              ))}
            </select>
            {timeDimension && (
              <select
                value={timeGranularity}
                onChange={(e) => setTimeGranularity(e.target.value)}
                className="w-full px-4 py-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="hour">Hour</option>
                <option value="day">Day</option>
                <option value="week">Week</option>
                <option value="month">Month</option>
                <option value="quarter">Quarter</option>
                <option value="year">Year</option>
              </select>
            )}
          </div>

          {/* Filters */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-semibold text-slate-700 dark:text-slate-300 flex items-center">
                <FilterIcon className="w-4 h-4 mr-2" />
                Filters
              </h3>
              <button
                onClick={addFilter}
                className="p-1 hover:bg-slate-100 dark:hover:bg-slate-800 rounded"
              >
                <Plus className="w-4 h-4" />
              </button>
            </div>
            <div className="space-y-2">
              {filters.map((filter, index) => (
                <div key={index} className="p-3 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-semibold text-slate-600 dark:text-slate-400">Filter {index + 1}</span>
                    <button onClick={() => removeFilter(index)} className="text-red-500 hover:text-red-700">
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                  <select
                    value={filter.member}
                    onChange={(e) => updateFilter(index, 'member', e.target.value)}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm mb-2"
                  >
                    <option value="">Select field...</option>
                    {selectedCube?.dimensions.map(d => (
                      <option key={d.name} value={`${selectedCube.name}.${d.name}`}>{d.display_name}</option>
                    ))}
                  </select>
                  <select
                    value={filter.operator}
                    onChange={(e) => updateFilter(index, 'operator', e.target.value)}
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm mb-2"
                  >
                    <option value="equals">Equals</option>
                    <option value="notEquals">Not Equals</option>
                    <option value="contains">Contains</option>
                    <option value="gt">Greater Than</option>
                    <option value="lt">Less Than</option>
                  </select>
                  <input
                    type="text"
                    value={filter.values[0] || ''}
                    onChange={(e) => updateFilter(index, 'values', [e.target.value])}
                    placeholder="Value..."
                    className="w-full px-3 py-2 bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg text-sm"
                  />
                </div>
              ))}
            </div>
          </div>

          {/* Limit */}
          <div>
            <label className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
              Limit
            </label>
            <input
              type="number"
              value={limit}
              onChange={(e) => setLimit(parseInt(e.target.value) || 100)}
              className="w-full px-4 py-3 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Right Panel - Results */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Tabs */}
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 px-6">
            <div className="flex space-x-4">
              <button className="px-4 py-3 border-b-2 border-blue-500 text-blue-600 dark:text-blue-400 font-medium">
                Results
              </button>
              <button className="px-4 py-3 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white">
                SQL
              </button>
            </div>
          </div>

          {/* Results Table */}
          <div className="flex-1 overflow-auto p-6">
            {loading ? (
              <div className="flex items-center justify-center h-full">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
              </div>
            ) : queryResult ? (
              <div>
                <div className="mb-4 flex items-center justify-between">
                  <div className="text-sm text-slate-600 dark:text-slate-400">
                    {queryResult.data.length} rows • {queryResult.executionTime}ms
                    {queryResult.cacheHit && <span className="ml-2 px-2 py-1 bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 rounded-full text-xs font-semibold">CACHED</span>}
                  </div>
                  <button className="flex items-center space-x-2 px-4 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700">
                    <Download className="w-4 h-4" />
                    <span className="text-sm">Export</span>
                  </button>
                </div>
                <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
                  <table className="w-full">
                    <thead className="bg-slate-50 dark:bg-slate-900">
                      <tr>
                        {queryResult.data[0] && Object.keys(queryResult.data[0]).map((key: string) => (
                          <th key={key} className="px-4 py-3 text-left text-xs font-semibold text-slate-600 dark:text-slate-400 uppercase">
                            {key}
                          </th>
                        ))}
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                      {queryResult.data.map((row: any, idx: number) => (
                        <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-slate-900/50">
                          {Object.values(row).map((value: any, cellIdx: number) => (
                            <td key={cellIdx} className="px-4 py-3 text-sm text-slate-900 dark:text-white">
                              {value !== null ? String(value) : '-'}
                            </td>
                          ))}
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : (
              <div className="flex items-center justify-center h-full text-slate-400">
                <div className="text-center">
                  <Code className="w-16 h-16 mx-auto mb-4 opacity-50" />
                  <p>Select measures and dimensions, then run your query</p>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default SemanticQueryBuilder;
