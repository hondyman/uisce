import React, { useState } from "react";
import { Card } from "../components/Card";
import { Table } from "../components/Table";
import { Spinner } from "../components/Feedback";
import { CreateAPIKeyModal } from "../components/CreateAPIKeyModal";
import { useAPIKeys } from "../hooks/useAPIKeys";
import "./APIKeysPage.css";

export function APIKeysPage() {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const apiKeysQuery = useAPIKeys();

  const apiKeys = apiKeysQuery.data?.data || [];

  const columns = ["Name", "Key Preview", "Created", "Last Used", "Status"];
  const rows = apiKeys.map((key) => [
    key.name,
    (
      <code className="key-preview">
        {key.key.substring(0, 20)}...
      </code>
    ),
    new Date(key.createdAt).toLocaleDateString(),
    key.lastUsedAt ? new Date(key.lastUsedAt).toLocaleDateString() : "Never",
    (
      <span className={`badge ${key.revoked ? "badge-revoked" : "badge-active"}`}>
        {key.revoked ? "Revoked" : "Active"}
      </span>
    ),
  ]);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>API Keys</h1>
          <p className="page-subtitle">Manage authentication tokens</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary"
        >
          + Create Key
        </button>
      </div>

      <Card className="grid-1">
        {apiKeysQuery.isLoading ? (
          <Spinner size="md" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            loading={apiKeysQuery.isLoading}
            empty="No API keys yet"
          />
        )}
      </Card>

      <CreateAPIKeyModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={() => {
          // List will auto-refresh via React Query invalidation
        }}
      />
    </div>
  );
}
