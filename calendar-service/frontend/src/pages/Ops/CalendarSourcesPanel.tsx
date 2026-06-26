import React, { useState, useCallback } from 'react';
import { useQuery, useMutation } from '@apollo/client';
import { gql } from '@apollo/client/core';
import { AlertCircle, CheckCircle, Clock, Zap } from 'lucide-react';

// ============================================================================
// GraphQL Queries & Mutations
// ============================================================================

const GET_SOURCES = gql`
  query GetCalendarSources {
    mdm_source_registry {
      id
      source_name
      source_type
      is_active
      priority_score
      confidence_base
      endpoint_url
      created_at
    }
  }
`;

const GET_SOURCE_HEALTH = gql`
  query GetSourceHealth {
    mdm_source_health(order_by: { checked_at: desc }, limit: 100) {
      source_name
      status
      last_success
      last_error
      consecutive_failures
      checked_at
    }
  }
`;

const GET_INGESTION_JOBS = gql`
  query GetIngestionJobs($limit: Int!) {
    mdm_ingestion_jobs(order_by: { started_at: desc }, limit: $limit) {
      id
      tenant_id
      job_type
      status
      records_ingested
      records_processed
      conflicts_detected
      error_message
      started_at
      completed_at
    }
  }
`;

const TOGGLE_SOURCE = gql`
  mutation ToggleSource($id: uuid!, $isActive: Boolean!) {
    update_mdm_source_registry_by_pk(
      pk_columns: { id: $id }
      _set: { is_active: $isActive }
    ) {
      id
      is_active
    }
  }
`;

const TRIGGER_INGESTION = gql`
  mutation TriggerIngestion($tenantId: uuid!, $regions: [String!]!, $year: Int!) {
    triggerIngestion(input: { tenant_id: $tenantId, regions: $regions, year: $year }) {
      status
      message
    }
  }
`;

// ============================================================================
// CalendarSourcesPanel Component
// ============================================================================

export const CalendarSourcesPanel = () => {
  const [selectedYear, setSelectedYear] = useState(2026);
  const [selectedRegions, setSelectedRegions] = useState(['US', 'GB']);

  // Queries
  const { data: sourcesData, loading: sourcesLoading, error: sourcesError } = useQuery(GET_SOURCES);
  const { data: healthData, loading: healthLoading } = useQuery(GET_SOURCE_HEALTH);
  const { data: jobsData, loading: jobsLoading } = useQuery(GET_INGESTION_JOBS, {
    variables: { limit: 10 }
  });

  // Mutations
  const [toggleSource] = useMutation(TOGGLE_SOURCE);
  const [triggerIngestion] = useMutation(TRIGGER_INGESTION);

  const handleToggleSource = useCallback(async (id, currentActive) => {
    try {
      await toggleSource({
        variables: { id, isActive: !currentActive }
      });
      // Apollo cache will auto-update
    } catch (err) {
      console.error('Failed to toggle source:', err);
    }
  }, [toggleSource]);

  const handleTriggerIngestion = useCallback(async () => {
    try {
      await triggerIngestion({
        variables: {
          tenantId: '00000000-0000-0000-0000-000000000001', // Replace with actual tenant
          regions: selectedRegions,
          year: selectedYear
        }
      });
      alert('Ingestion cycle triggered successfully!');
    } catch (err) {
      console.error('Failed to trigger ingestion:', err);
      alert('Failed to trigger ingestion');
    }
  }, [triggerIngestion, selectedYear, selectedRegions]);

  const getSourceStatus = (sourceName) => {
    const health = healthData?.mdm_source_health?.find(h => h.source_name === sourceName);
    if (!health) return { status: 'UNKNOWN', icon: '?' };
    return health;
  };

  const regions = ['US', 'GB', 'FR', 'DE', 'JP', 'CN', 'AU'];

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="border-b pb-6">
        <h1 className="text-3xl font-bold text-gray-900">MDM Calendar Management</h1>
        <p className="text-gray-600 mt-2">Manage data sources, monitor ingestion, and resolve conflicts</p>
      </div>

      {/* Ingestion Control Panel */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <h2 className="text-lg font-semibold text-blue-900 mb-4">Trigger Ingestion</h2>
        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Year</label>
            <input
              type="number"
              value={selectedYear}
              onChange={(e) => setSelectedYear(parseInt(e.target.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Regions</label>
            <select
              multiple
              value={selectedRegions}
              onChange={(e) => setSelectedRegions(Array.from(e.target.selectedOptions, opt => opt.value))}
              className="w-full px-3 py-2 border border-gray-300 rounded-md"
            >
              {regions.map(r => (
                <option key={r} value={r}>{r}</option>
              ))}
            </select>
          </div>
          <div className="flex items-end">
            <button
              onClick={handleTriggerIngestion}
              className="w-full bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
            >
              <Zap className="inline mr-2" size={18} />
              Trigger
            </button>
          </div>
        </div>
      </div>

      {/* Data Sources Table */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b bg-gray-50">
          <h2 className="text-lg font-semibold text-gray-900">Data Sources</h2>
        </div>
        
        {sourcesLoading ? (
          <div className="p-6 text-center text-gray-500">Loading sources...</div>
        ) : sourcesError ? (
          <div className="p-6 text-center text-red-500">Error loading sources</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-100 border-b">
                <tr>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Source</th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Type</th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Priority</th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Confidence</th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Status</th>
                  <th className="px-6 py-3 text-left text-sm font-semibold text-gray-700">Action</th>
                </tr>
              </thead>
              <tbody>
                {sourcesData?.mdm_source_registry?.map((src) => {
                  const health = getSourceStatus(src.source_name);
                  return (
                    <tr key={src.id} className="border-b hover:bg-gray-50">
                      <td className="px-6 py-3">
                        <span className="font-medium text-gray-900">{src.source_name}</span>
                      </td>
                      <td className="px-6 py-3 text-sm text-gray-600">{src.source_type}</td>
                      <td className="px-6 py-3 text-sm text-gray-600">{src.priority_score}</td>
                      <td className="px-6 py-3 text-sm text-gray-600">{src.confidence_base}%</td>
                      <td className="px-6 py-3">
                        {health.status === 'UP' && (
                          <span className="flex items-center text-green-600 text-sm">
                            <CheckCircle size={16} className="mr-2" />
                            Healthy
                          </span>
                        )}
                        {health.status === 'DOWN' && (
                          <span className="flex items-center text-red-600 text-sm">
                            <AlertCircle size={16} className="mr-2" />
                            Down ({health.consecutive_failures}x)
                          </span>
                        )}
                      </td>
                      <td className="px-6 py-3">
                        <button
                          onClick={() => handleToggleSource(src.id, src.is_active)}
                          className={`px-3 py-1 rounded text-sm font-medium transition ${
                            src.is_active
                              ? 'bg-green-100 text-green-800 hover:bg-green-200'
                              : 'bg-gray-100 text-gray-800 hover:bg-gray-200'
                          }`}
                        >
                          {src.is_active ? 'Active' : 'Inactive'}
                        </button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Recent Ingestion Jobs */}
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <div className="px-6 py-4 border-b bg-gray-50">
          <h2 className="text-lg font-semibold text-gray-900">Recent Ingestion Jobs</h2>
        </div>
        
        {jobsLoading ? (
          <div className="p-6 text-center text-gray-500">Loading jobs...</div>
        ) : (
          <div className="space-y-2 p-6">
            {jobsData?.mdm_ingestion_jobs?.map((job) => (
              <div key={job.id} className="flex items-center justify-between p-3 border rounded bg-gray-50">
                <div>
                  <span className="font-medium text-gray-900">{job.job_type}</span>
                  <span className="text-sm text-gray-600 ml-2">
                    Records: {job.records_ingested} | Conflicts: {job.conflicts_detected}
                  </span>
                </div>
                <div>
                  {job.status === 'SUCCESS' && (
                    <span className="flex items-center text-green-600 text-sm">
                      <CheckCircle size={16} className="mr-2" />
                      Success
                    </span>
                  )}
                  {job.status === 'IN_PROGRESS' && (
                    <span className="flex items-center text-blue-600 text-sm">
                      <Clock size={16} className="mr-2" />
                      Processing
                    </span>
                  )}
                  {job.status === 'FAILED' && (
                    <span className="flex items-center text-red-600 text-sm">
                      <AlertCircle size={16} className="mr-2" />
                      Failed
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
};

export default CalendarSourcesPanel;
