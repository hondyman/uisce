import React from "react";
import { useNavigate } from "react-router-dom";
import { Card } from "../components/Card";
import { Table } from "../components/Table";
import { LineChart, BarChart } from "../components/Charts";
import { Spinner } from "../components/Feedback";
import { HealthBadge, HeatmapChart, AlertList, ErrorFingerprints, OpsTimeline } from "../components";
import { OpsEvent } from "../hooks/useOpsTimeline";
import {
  useGlobalUsage,
  useGlobalErrors,
  useGlobalLatency,
  useTopTenants,
  useTopEndpoints,
  useRecentErrors,
} from "../hooks/useUsage";
import {
  useAlerts,
  useTenantHealth,
  useLatencyHeatmap,
  useErrorFingerprints,
} from "../hooks/useOps";
import "./GlobalOpsDashboard.css";

export function GlobalOpsDashboard() {
  const navigate = useNavigate();
  const globalUsageQuery = useGlobalUsage();
  const globalErrorsQuery = useGlobalErrors();
  const globalLatencyQuery = useGlobalLatency();
  const topTenantsQuery = useTopTenants(10);
  const topEndpointsQuery = useTopEndpoints(10);
  const recentErrorsQuery = useRecentErrors(50);

  // Ops Cockpit hooks
  const alertsQuery = useAlerts({ enabled: true });
  const heatmapQuery = useLatencyHeatmap({ window: "3600" });
  const fingerprintsQuery = useErrorFingerprints({ limit: 20 });

  // Handle navigation to incident detail
  const handleEventClick = (event: OpsEvent) => {
    if (event.incident_id) {
      navigate(`/admin/ops/incidents/${event.incident_id}`);
    }
  };

  const usageData = globalUsageQuery.data?.data || [];
  const errorData = globalErrorsQuery.data?.data || [];
  const latencyData = globalLatencyQuery.data?.data || [];
  const topTenants = topTenantsQuery.data?.data || [];
  const topEndpoints = topEndpointsQuery.data?.data || [];
  const recentErrors = recentErrorsQuery.data?.data || [];

  // Calculate summary metrics
  const totalRequests = usageData.reduce(
    (sum, point) => sum + (point.requests || 0),
    0
  );
  const totalErrors = errorData.reduce(
    (sum, point) => sum + (point.errors || 0),
    0
  );
  const avgLatency =
    latencyData.length > 0
      ? latencyData.reduce((sum, point) => sum + (point.p50 || 0), 0) /
        latencyData.length
      : 0;

  const tenantColumns = ["Tenant", "Requests", "% of Total"];
  const tenantRows = topTenants.map((tenant) => [
    tenant.name,
    tenant.requests.toLocaleString(),
    ((tenant.requests / totalRequests) * 100).toFixed(1) + "%",
  ]);

  const endpointColumns = ["Path", "Method", "Requests", "Avg Latency"];
  const endpointRows = topEndpoints.map((endpoint) => [
    endpoint.path,
    endpoint.method,
    endpoint.requests.toLocaleString(),
    endpoint.avgLatency.toFixed(0) + "ms",
  ]);

  const errorColumns = ["Tenant", "Path", "Status", "Count", "Last Seen"];
  const errorRows = recentErrors.map((error) => [
    error.tenantId,
    error.path,
    (
      <span className={`status-code status-${Math.floor(error.status / 100)}`}>
        {error.status}
      </span>
    ),
    error.count,
    new Date(error.lastSeen).toLocaleString(),
  ]);

  return (
    <div className="page">
      <h1>Global Operations Dashboard</h1>

      {/* Ops Cockpit Summary Section */}
      <div className="ops-cockpit-section">
        <h2>⚡ Real-Time Operations Intelligence</h2>
        
        {/* Health and Alerts */}
        <div className="grid-2">
          <Card title="Active Alerts" className="grid-1">
            {alertsQuery.isLoading ? (
              <Spinner size="sm" />
            ) : (
              <AlertList
                alerts={alertsQuery.data?.data || []}
                onEvaluate={() => alertsQuery.refetch()}
              />
            )}
          </Card>
          <Card title="System Health" className="grid-1">
            {alertsQuery.isLoading ? (
              <Spinner size="sm" />
            ) : (
              <div className="health-score-container">
                <HealthBadge score={92} />
              </div>
            )}
          </Card>
        </div>

        {/* Latency Heatmap */}
        <Card title="Latency Heatmap (Last Hour)" className="grid-1">
          {heatmapQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <HeatmapChart heatmap={heatmapQuery.data?.data} />
          )}
        </Card>

        {/* Error Fingerprints */}
        <Card title="Error Fingerprints" className="grid-1">
          {fingerprintsQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <ErrorFingerprints fingerprints={fingerprintsQuery.data?.data || []} />
          )}
        </Card>

        {/* Real-Time Incident Timeline */}
        <Card title="📅 Real-Time Incident Timeline" className="grid-1">
          <OpsTimeline since="1h" limit={100} onEventClick={handleEventClick} />
        </Card>
      </div>

      {/* Divider */}
      <hr className="section-divider" />

      <h2>📊 Historical Usage Analytics</h2>

      {/* Summary Cards */}
      <div className="grid-3">
        <Card title="Total Requests" subtitle="Last 30 days">
          <div className="metric-value">{totalRequests.toLocaleString()}</div>
        </Card>
        <Card title="Total Errors" subtitle="Last 30 days">
          <div className={`metric-value ${totalErrors > 0 ? "error" : ""}`}>
            {totalErrors.toLocaleString()}
          </div>
        </Card>
        <Card title="Average Latency" subtitle="p50">
          <div className="metric-value">{avgLatency.toFixed(0)}ms</div>
        </Card>
      </div>

      {/* Charts */}
      <div className="grid-2">
        <Card className="grid-1">
          {globalUsageQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <LineChart
              data={usageData}
              dataKey="requests"
              title="Requests Over Time"
              height={300}
            />
          )}
        </Card>
        <Card className="grid-1">
          {globalErrorsQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <LineChart
              data={errorData}
              dataKey="errors"
              title="Errors Over Time"
              height={300}
            />
          )}
        </Card>
      </div>

      {/* Latency Chart */}
      <Card className="grid-1">
        {globalLatencyQuery.isLoading ? (
          <Spinner size="sm" />
        ) : (
          <LineChart
            data={latencyData}
            dataKey="p50"
            title="Latency Percentiles (p50)"
            height={300}
          />
        )}
      </Card>

      {/* Tables */}
      <div className="grid-2">
        <Card title="Top Tenants" className="grid-1">
          {topTenantsQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <Table
              columns={tenantColumns}
              rows={tenantRows}
              empty="No tenant data"
            />
          )}
        </Card>
        <Card title="Top Endpoints" className="grid-1">
          {topEndpointsQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <Table
              columns={endpointColumns}
              rows={endpointRows}
              empty="No endpoint data"
            />
          )}
        </Card>
      </div>

      {/* Recent Errors */}
      <Card title="Recent Errors" className="grid-1">
        {recentErrorsQuery.isLoading ? (
          <Spinner size="sm" />
        ) : (
          <Table
            columns={errorColumns}
            rows={errorRows}
            empty="No recent errors"
          />
        )}
      </Card>
    </div>
  );
}
