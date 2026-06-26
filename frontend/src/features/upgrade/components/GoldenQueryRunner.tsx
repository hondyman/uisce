import { useState, useEffect } from 'react';
import { listGoldenQueries, runGoldenQueries } from '../api';
import { devError } from '../../../utils/devLogger';

interface GoldenQuery {
  name: string;
  query?: string; // SQL text
  sql?: string;   // backward compatibility
  description?: string;
  tags?: string[];
}

interface GoldenQueryResult {
  query_name: string;
  old_result: { rows?: unknown[]; execution_time_ms: number; error?: string };
  new_result: { rows?: unknown[]; execution_time_ms: number; error?: string };
  diff_analysis: {
    row_count_diff: number;
    execution_time_diff_ms: number;
    breaking_changes?: boolean;
    data_differences: unknown[];
  };
}

interface GoldenQueryRunnerProps {
  fromVersion: string;
  toVersion: string;
  onClose?: () => void;
}

export default function GoldenQueryRunner({ fromVersion, toVersion, onClose }: GoldenQueryRunnerProps) {
  const [queries, setQueries] = useState<GoldenQuery[]>([]);
  const [results, setResults] = useState<GoldenQueryResult[]>([]);
  const [running, setRunning] = useState(false);
  const [activeTab, setActiveTab] = useState<'queries' | 'results'>('queries');

  useEffect(() => {
    loadGoldenQueries();
  }, []);

  const loadGoldenQueries = async () => {
    try {
      const goldenQueries = await listGoldenQueries();
      setQueries(goldenQueries);
    } catch (error) {
      devError('Failed to load golden queries:', error);
    }
  };

  const handleRunQueries = async (queryNames?: string[]) => {
    try {
      setRunning(true);
      const runResults = await runGoldenQueries(fromVersion, toVersion, queryNames);
      setResults(runResults);
      setActiveTab('results');
    } catch (error) {
      devError('Failed to run golden queries:', error);
    } finally {
      setRunning(false);
    }
  };

  const handleRunSingleQuery = async (query: GoldenQuery) => {
    await handleRunQueries([query.name]);
  };

  return (
    <div className="golden-query-runner">
      <header className="flex justify-between items-center mb-4">
        <h2 className="text-xl font-semibold">Golden Query Runner</h2>
        <div className="flex gap-2">
          <span className="text-sm text-gray-600">{queries.length} queries</span>
          <span className="text-sm text-gray-600">{results.length} results</span>
        </div>
      </header>

      <p className="text-sm text-gray-600 mb-4">
        Comparing {fromVersion} → {toVersion}
      </p>

      <div className="flex gap-4 mb-4">
        <button
          className={`px-3 py-1 rounded ${activeTab === 'queries' ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}
          onClick={() => setActiveTab('queries')}
        >
          Queries
        </button>
        <button
          className={`px-3 py-1 rounded ${activeTab === 'results' ? 'bg-blue-500 text-white' : 'bg-gray-200'}`}
          onClick={() => setActiveTab('results')}
        >
          Results
        </button>
      </div>

      <div className="mb-4">
        <button
          className="px-4 py-2 bg-green-500 text-white rounded"
          onClick={() => handleRunQueries()}
          disabled={running || queries.length === 0}
        >
          {running ? 'Running...' : `Run All (${queries.length})`}
        </button>
      </div>

      {activeTab === 'queries' && (
        <section className="space-y-4">
          {queries.map((query) => (
            <div key={query.name} className="border rounded p-4">
              <div className="flex justify-between items-center mb-2">
                <h3 className="font-semibold">{query.name}</h3>
                <button
                  className="px-3 py-1 bg-blue-500 text-white rounded text-sm"
                  onClick={() => handleRunSingleQuery(query)}
                >
                  Run
                </button>
              </div>
              <p className="text-sm text-gray-600 mb-2">{query.description}</p>
              <pre className="text-xs bg-gray-100 p-2 rounded overflow-x-auto">
                {query.query || query.sql}
              </pre>
              {query.tags && (
                <div className="mt-2 flex gap-1">
                  {query.tags.map((tag) => (
                    <span key={tag} className="px-2 py-1 bg-gray-200 rounded text-xs">
                      {tag}
                    </span>
                  ))}
                </div>
              )}
            </div>
          ))}

          {queries.length === 0 && (
            <p className="text-gray-500">No golden queries defined.</p>
          )}
        </section>
      )}

      {activeTab === 'results' && (
        <section className="space-y-4">
          {results.map((result) => (
            <div key={result.query_name} className="border rounded p-4">
              <div className="flex justify-between items-center mb-2">
                <h3 className="font-semibold">{result.query_name}</h3>
                <span className={`px-2 py-1 rounded text-xs ${
                  result.diff_analysis.breaking_changes ? 'bg-red-200' :
                  result.diff_analysis.row_count_diff !== 0 ? 'bg-yellow-200' : 'bg-green-200'
                }`}>
                  {result.diff_analysis.breaking_changes ? 'Breaking' :
                   result.diff_analysis.row_count_diff !== 0 ? 'Warning' : 'OK'}
                </span>
              </div>

              <div className="grid grid-cols-2 gap-4 mb-4">
                <div className="border rounded p-2">
                  <h4 className="font-medium">Old Version ({fromVersion})</h4>
                  <p className="text-sm">Rows: {result.old_result.rows?.length || 0}</p>
                  <p className="text-sm">Time: {result.old_result.execution_time_ms}ms</p>
                  {result.old_result.error && (
                    <p className="text-sm text-red-500">Error: {result.old_result.error}</p>
                  )}
                </div>
                <div className="border rounded p-2">
                  <h4 className="font-medium">New Version ({toVersion})</h4>
                  <p className="text-sm">Rows: {result.new_result.rows?.length || 0}</p>
                  <p className="text-sm">Time: {result.new_result.execution_time_ms}ms</p>
                  {result.new_result.error && (
                    <p className="text-sm text-red-500">Error: {result.new_result.error}</p>
                  )}
                </div>
              </div>

              <div className="text-sm">
                <p>Row count diff: {result.diff_analysis.row_count_diff}</p>
                <p>Time diff: {result.diff_analysis.execution_time_diff_ms}ms</p>
                {result.diff_analysis.data_differences.length > 0 && (
                  <p>Data differences: {result.diff_analysis.data_differences.length}</p>
                )}
              </div>
            </div>
          ))}

          {results.length === 0 && !running && (
            <p className="text-gray-500">No results yet. Run some queries to validate your upgrade.</p>
          )}
        </section>
      )}

      {onClose && (
        <div className="mt-4 flex justify-end">
          <button
            className="px-4 py-2 bg-gray-500 text-white rounded"
            onClick={onClose}
          >
            Close
          </button>
        </div>
      )}
    </div>
  );
}
