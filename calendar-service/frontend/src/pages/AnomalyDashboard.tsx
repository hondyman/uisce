import React, { useState, useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { gql } from '@apollo/client';

const GET_ANOMALIES = gql`
  query GetActiveAnomalies {
    active_anomalies(order_by: {severity: desc, detected_at: desc}) {
      id
      anomaly_type
      severity
      description
      detected_at
      status
      confidence_score
      alerts_sent
      auto_remediation_attempted
    }
    anomaly_stats(limit: 7, order_by: {date: desc}) {
      date
      anomaly_type
      total_anomalies
      resolved_anomalies
    }
  }
`;

const AnomalyDashboard: React.FC = () => {
  const { loading, error, data } = useQuery(GET_ANOMALIES, {
    pollInterval: 30000, // Refresh every 30 seconds
  });

  if (loading) return <div className="p-8 text-center">Loading anomaly data...</div>;
  if (error) return <div className="p-8 text-red-500">Error loading anomalies: {error.message}</div>;

  const anomalies = data?.active_anomalies || [];
  const stats = data?.anomaly_stats || [];

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <h1 className="text-2xl font-bold mb-6 text-gray-800 dark:text-gray-100">Anomaly Detection Dashboard</h1>
      
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-600 dark:text-gray-300">Active Critical</h3>
          <p className="text-3xl font-bold text-red-600 mt-2">
            {anomalies.filter((a: any) => a.severity === 'critical').length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-600 dark:text-gray-300">Active Warnings</h3>
          <p className="text-3xl font-bold text-yellow-500 mt-2">
            {anomalies.filter((a: any) => a.severity === 'warning').length}
          </p>
        </div>
        <div className="bg-white dark:bg-gray-800 p-6 rounded-lg shadow-sm border border-gray-200 dark:border-gray-700">
          <h3 className="text-lg font-medium text-gray-600 dark:text-gray-300">Total Monitored</h3>
          <p className="text-3xl font-bold text-blue-600 mt-2">
            {stats.reduce((acc: number, curr: any) => acc + curr.total_anomalies, 0) || 0}
          </p>
        </div>
      </div>

      <div className="bg-white dark:bg-gray-800 rounded-lg shadow border border-gray-200 dark:border-gray-700 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
          <h2 className="text-xl font-semibold">Active Anomalies</h2>
        </div>
        
        {anomalies.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            No active anomalies detected. System is running smoothly!
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
            <thead className="bg-gray-50 dark:bg-gray-800">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Severity</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Description</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Detected</th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
              {anomalies.map((anomaly: any) => (
                <tr key={anomaly.id}>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full 
                      ${anomaly.severity === 'critical' ? 'bg-red-100 text-red-800' : 'bg-yellow-100 text-yellow-800'}`}>
                      {anomaly.severity}
                    </span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-gray-100">
                    {anomaly.anomaly_type}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 dark:text-gray-400">
                    {anomaly.description}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                    {new Date(anomaly.detected_at).toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium">
                    <button className="text-indigo-600 hover:text-indigo-900 dark:hover:text-indigo-400 mr-3">
                      Investigate
                    </button>
                    <button className="text-green-600 hover:text-green-900 dark:hover:text-green-400">
                      Resolve
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
};

export default AnomalyDashboard;
