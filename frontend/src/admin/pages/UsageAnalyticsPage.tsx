// Usage Analytics Page - Usage statistics and charts

import React, { useState } from "react";
import {
  useTenantDailyUsage,
  useTenantEndpointUsage,
  useTenants,
} from "../hooks/useAdmin";
import { Tenant } from "../types";
import "./UsageAnalyticsPage.css";

export const UsageAnalyticsPage: React.FC = () => {
  const [selectedTenantId, setSelectedTenantId] = useState<string>("");
  const [dayRange, setDayRange] = useState(30);

  const { tenants, loading: tenantsLoading } = useTenants(100, 0);

  const { stats: dailyStats, loading: dailyLoading } = useTenantDailyUsage(
    selectedTenantId,
    dayRange
  );

  const { stats: endpointStats, loading: endpointLoading } =
    useTenantEndpointUsage(selectedTenantId, 10);

  const handleTenantChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setSelectedTenantId(e.target.value);
  };

  const handleDayRangeChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    setDayRange(parseInt(e.target.value));
  };

  // Calculate totals
  const totalUsage = dailyStats.reduce((sum, stat) => sum + stat.count, 0);
  const avgDaily =
    dailyStats.length > 0 ? Math.round(totalUsage / dailyStats.length) : 0;
  const maxDaily =
    dailyStats.length > 0
      ? Math.max(...dailyStats.map((stat) => stat.count))
      : 0;

  return (
    <div className="usage-analytics-page">
      <div className="page-header">
        <h1>Usage Analytics</h1>
      </div>

      {/* Filter Section */}
      <div className="filter-section">
        <div className="filter-group">
          <label htmlFor="tenant-select">Select Tenant:</label>
          <select
            id="tenant-select"
            value={selectedTenantId}
            onChange={handleTenantChange}
            disabled={tenantsLoading}
          >
            <option value="">-- Choose a tenant --</option>
            {tenants.map((tenant: Tenant) => (
              <option key={tenant.id} value={tenant.id}>
                {tenant.name} ({tenant.code})
              </option>
            ))}
          </select>
        </div>

        {selectedTenantId && (
          <div className="filter-group">
            <label htmlFor="day-range">Day Range:</label>
            <select id="day-range" value={dayRange} onChange={handleDayRangeChange}>
              <option value="7">Last 7 days</option>
              <option value="30">Last 30 days</option>
              <option value="90">Last 90 days</option>
            </select>
          </div>
        )}
      </div>

      {!selectedTenantId && (
        <div className="empty-state">
          <p>Select a tenant to view usage analytics.</p>
        </div>
      )}

      {selectedTenantId && (
        <>
          {/* Summary Cards */}
          <div className="summary-cards">
            <div className="card">
              <div className="card-label">Total Requests</div>
              <div className="card-value">{totalUsage.toLocaleString()}</div>
              <div className="card-period">Last {dayRange} days</div>
            </div>

            <div className="card">
              <div className="card-label">Average Daily</div>
              <div className="card-value">{avgDaily.toLocaleString()}</div>
              <div className="card-period">Avg per day</div>
            </div>

            <div className="card">
              <div className="card-label">Peak Daily</div>
              <div className="card-value">{maxDaily.toLocaleString()}</div>
              <div className="card-period">Highest day</div>
            </div>
          </div>

          {/* Daily Trend Chart */}
          <div className="chart-section">
            <h2>Daily Request Trend</h2>
            {dailyLoading ? (
              <div className="loading">Loading daily usage data...</div>
            ) : dailyStats.length === 0 ? (
              <div className="empty-state">No usage data available.</div>
            ) : (
              <div className="chart-container">
                {/* Simple bar chart using CSS */}
                <div className="bar-chart">
                  {dailyStats.slice(-14).map((stat, idx) => {
                    const maxValue = Math.max(
                      ...dailyStats.map((s) => s.count)
                    );
                    const height = (stat.count / maxValue) * 100;
                    return (
                      <div key={idx} className="bar-item">
                        <div
                          className="bar"
                          style={{ height: `${height}%` }}
                          title={`${stat.day}: ${stat.count} requests`}
                        />
                        <div className="bar-label">
                          {new Date(stat.day).toLocaleDateString("en-US", {
                            month: "short",
                            day: "numeric",
                          })}
                        </div>
                      </div>
                    );
                  })}
                </div>
                <div className="chart-legend">
                  Last 14 days (showing {dailyStats.slice(-14).length} days)
                </div>
              </div>
            )}
          </div>

          {/* Top Endpoints */}
          <div className="endpoints-section">
            <h2>Top Endpoints</h2>
            {endpointLoading ? (
              <div className="loading">Loading endpoint data...</div>
            ) : endpointStats.length === 0 ? (
              <div className="empty-state">No endpoint data available.</div>
            ) : (
              <div className="endpoints-table">
                <div className="endpoints-header">
                  <div className="col-path">Endpoint</div>
                  <div className="col-count">Requests</div>
                  <div className="col-bar">Distribution</div>
                </div>
                {endpointStats.map((stat, idx) => {
                  const maxCount = Math.max(...endpointStats.map((s) => s.count));
                  const width = (stat.count / maxCount) * 100;
                  return (
                    <div key={idx} className="endpoint-row">
                      <div className="col-path">{stat.path}</div>
                      <div className="col-count">
                        {stat.count.toLocaleString()}
                      </div>
                      <div className="col-bar">
                        <div
                          className="bar-fill"
                          style={{ width: `${width}%` }}
                        />
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>

          {/* Data Export */}
          <div className="export-section">
            <button className="btn btn-secondary">
              📥 Export as CSV
            </button>
            <button className="btn btn-secondary">
              📊 Generate Report
            </button>
          </div>
        </>
      )}
    </div>
  );
};

export default UsageAnalyticsPage;
