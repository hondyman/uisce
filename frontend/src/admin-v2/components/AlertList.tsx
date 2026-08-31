import React, { useState } from "react";
import { Card } from "./Card";
import { Table } from "./Table";
import { Spinner, ErrorBanner } from "./Feedback";
import { useAlerts, useEvaluateAlerts } from "../hooks/useOps";
import {
  useExceptions,
  useRerunException,
  useCloseException,
} from "../hooks/useExceptions";
import { ExceptionAutofixPolicyPanel } from "./ExceptionAutofixPolicyPanel";
import type { Alert } from "../types";
import "./AlertList.css";

type AlertListTab = "alerts" | "exceptions";

export function AlertList() {
  const [tab, setTab] = useState<AlertListTab>("alerts");

  return (
    <Card title="Active Alerts" className="grid-1">
      <div className="alert-list-tabs">
        <button
          className={`btn btn-small ${tab === "alerts" ? "btn-active" : ""}`}
          onClick={() => setTab("alerts")}
        >
          Alerts
        </button>
        <button
          className={`btn btn-small ${tab === "exceptions" ? "btn-active" : ""}`}
          onClick={() => setTab("exceptions")}
        >
          Exceptions
        </button>
      </div>

      {tab === "alerts" ? <AlertsView /> : <ExceptionsView />}
    </Card>
  );
}

function AlertsView() {
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
    <>
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
    </>
  );
}

function ExceptionsView() {
  const exceptionsQuery = useExceptions();
  const rerunMutation = useRerunException();
  const closeMutation = useCloseException();

  const exceptions = exceptionsQuery.data || [];

  const columns = [
    "Type",
    "Severity",
    "Status",
    "Occurrences",
    "Source",
    "Actions",
  ];
  const rows = exceptions.map((exc) => [
    exc.type,
    (
      <span className={`alert-status alert-severity-${exc.severity}`}>
        {exc.severity}
      </span>
    ),
    exc.status,
    (
      <span className="occurrence-badge" title="Deduped occurrence count">
        ×{exc.occurrence_count}
      </span>
    ),
    exc.source,
    (
      <div className="exception-row-actions">
        <button
          className="btn btn-small"
          disabled={rerunMutation.isPending}
          onClick={() => rerunMutation.mutate(exc.id)}
        >
          Rerun
        </button>
        <button
          className="btn btn-small"
          disabled={closeMutation.isPending || exc.status === "closed"}
          onClick={() => closeMutation.mutate(exc.id)}
        >
          Close
        </button>
      </div>
    ),
  ]);

  return (
    <>
      {(rerunMutation.isError || closeMutation.isError) && (
        <ErrorBanner message="Failed to update exception" />
      )}

      {exceptionsQuery.isLoading ? (
        <Spinner size="sm" />
      ) : (
        <Table
          columns={columns}
          rows={rows}
          loading={exceptionsQuery.isLoading}
          empty="No exceptions detected"
        />
      )}

      <ExceptionAutofixPolicyPanel />
    </>
  );
}
