import React, { useState } from "react";
import { Card } from "./Card";
import { Table } from "./Table";
import { Spinner, ErrorBanner } from "./Feedback";
import { useErrorFingerprints, useErrorFingerprintHistory } from "../hooks/useOps";
import type { ErrorFingerprint } from "../types";
import "./ErrorFingerprints.css";

export function ErrorFingerprints() {
  const fingerprintsQuery = useErrorFingerprints(50);
  const [selectedFingerprintId, setSelectedFingerprintId] = useState<string | null>(null);
  const historyQuery = useErrorFingerprintHistory(selectedFingerprintId, 50);

  const fingerprints = fingerprintsQuery.data?.data || [];
  const history = historyQuery.data?.data || [];

  const columns = ["Path", "Status", "Sample", "Count", "Last Seen"];
  const rows = fingerprints.map((fp) => [
    <code className="fingerprint-path">{fp.path}</code>,
    (
      <span className={`status-code status-${Math.floor(fp.status_code / 100)}`}>
        {fp.status_code}
      </span>
    ),
    <span className="fingerprint-message">{fp.sample_message}</span>,
    <strong>{fp.count}</strong>,
    new Date(fp.last_seen).toLocaleString(),
  ]);

  const eventColumns = ["Tenant", "Endpoint", "Message", "Time"];
  const eventRows = history.map((event) => [
    event.tenant_id || "N/A",
    event.endpoint,
    <span className="error-message">{event.message}</span>,
    new Date(event.occurred_at).toLocaleString(),
  ]);

  return (
    <div className="error-fingerprints">
      <Card title="Error Fingerprints" subtitle="Grouped error patterns" className="grid-1">
        {fingerprintsQuery.isLoading ? (
          <Spinner size="sm" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            loading={fingerprintsQuery.isLoading}
            empty="No errors recorded"
          />
        )}
      </Card>

      {selectedFingerprintId && (
        <Card title="Recent Occurrences" className="grid-1">
          {historyQuery.isError && (
            <ErrorBanner message="Failed to load error history" />
          )}
          {historyQuery.isLoading ? (
            <Spinner size="sm" />
          ) : (
            <>
              <Table
                columns={eventColumns}
                rows={eventRows}
                loading={historyQuery.isLoading}
                empty="No recent occurrences"
              />
              <button
                onClick={() => setSelectedFingerprintId(null)}
                className="btn btn-secondary"
                style={{ marginTop: "var(--spacing-md)" }}
              >
                Clear Selection
              </button>
            </>
          )}
        </Card>
      )}
    </div>
  );
}
