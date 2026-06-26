import React, { useState } from "react";
import { Card } from "./Card";
import { Table } from "./Table";
import { Spinner, ErrorBanner } from "./Feedback";
import { useAlerts, useEvaluateAlerts } from "../hooks/useOps";
import type { Alert } from "../types";
import "./AlertList.css";

export function AlertList() {
  const alertsQuery = useAlerts();
  const evaluateMutation = useEvaluateAlerts();
  const [showDisabled, setShowDisabled] = useState(false);

  const alerts = alertsQuery.data?.data || [];
  const filteredAlerts = showDisabled
    ? alerts
    : alerts.filter((a) => a.enabled);

  const columns = ["Name", "Metric", "Scope", "Threshold", "Window", "Status"];
  const rows = filteredAlerts.map((alert) => [
    alert.name,
    alert.metric,
    alert.scope.charAt(0).toUpperCase() + alert.scope.slice(1),
    `${alert.comparison} ${alert.threshold.toFixed(2)}`,
    `${alert.window_secs}s`,
    (
      <span
        className={`alert-status ${alert.enabled ? "alert-enabled" : "alert-disabled"}`}
      >
        {alert.enabled ? "Enabled" : "Disabled"}
      </span>
    ),
  ]);

  return (
    <Card title="Active Alerts" className="grid-1">
      <div className="alert-list-controls">
        <label className="checkbox-label">
          <input
            type="checkbox"
            checked={showDisabled}
            onChange={(e) => setShowDisabled(e.target.checked)}
          />
          <span>Show disabled</span>
        </label>
        <button
          onClick={() => evaluateMutation.mutate()}
          disabled={evaluateMutation.isPending}
          className="btn btn-small"
        >
          {evaluateMutation.isPending ? "Evaluating..." : "Evaluate Now"}
        </button>
      </div>

      {evaluateMutation.isError && (
        <ErrorBanner message="Failed to evaluate alerts" />
      )}

      {alertsQuery.isLoading ? (
        <Spinner size="sm" />
      ) : (
        <Table
          columns={columns}
          rows={rows}
          loading={alertsQuery.isLoading}
          empty="No alerts configured"
        />
      )}
    </Card>
  );
}
