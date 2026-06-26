// API Keys Page - API key management interface

import React, { useState } from "react";
import { useAPIKeys } from "../hooks/useAdmin";
import { APIKey } from "../types";
import "./APIKeysPage.css";

export const APIKeysPage: React.FC = () => {
  const [limit, setLimit] = useState(50);
  const [offset, setOffset] = useState(0);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formData, setFormData] = useState({
    name: "",
    tenant_ids: "",
    roles: ["USER"],
  });

  const { keys, total, loading, error, refetch } = useAPIKeys(limit, offset);

  const handleCreateClick = () => {
    setShowCreateForm(true);
  };

  const handleFormChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleRoleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const role = e.target.value;
    setFormData((prev) => ({
      ...prev,
      roles: e.target.checked
        ? [...prev.roles, role]
        : prev.roles.filter((r) => r !== role),
    }));
  };

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const tenantIds = formData.tenant_ids
        .split(",")
        .map((id) => id.trim())
        .filter((id) => id);

      const response = await fetch("/api/admin/api-keys", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: formData.name,
          tenant_ids: tenantIds,
          roles: formData.roles,
        }),
      });

      if (!response.ok) {
        const error = await response.json();
        alert(`Error: ${error.error || "Failed to create API key"}`);
        return;
      }

      const data = await response.json();
      alert(
        `API Key created successfully!\n\nKey: ${data.api_key.key}\n\nMake sure to copy and save this key securely. You won't be able to see it again.`
      );

      refetch();
      setShowCreateForm(false);
      setFormData({
        name: "",
        tenant_ids: "",
        roles: ["USER"],
      });
    } catch (err) {
      alert(
        `Error: ${err instanceof Error ? err.message : "Unknown error"}`
      );
    }
  };

  const pageCount = Math.ceil(total / limit);
  const currentPage = Math.floor(offset / limit) + 1;

  return (
    <div className="api-keys-page">
      <div className="page-header">
        <h1>API Keys</h1>
        <button className="btn btn-primary" onClick={handleCreateClick}>
          + New API Key
        </button>
      </div>

      {error && <div className="alert alert-error">{error}</div>}

      {/* Create Form Modal */}
      {showCreateForm && (
        <div className="modal-overlay">
          <div className="modal">
            <div className="modal-header">
              <h2>Create New API Key</h2>
              <button
                className="modal-close"
                onClick={() => setShowCreateForm(false)}
              >
                ✕
              </button>
            </div>
            <form onSubmit={handleCreateKey} className="api-key-form">
              <div className="form-group">
                <label htmlFor="name">API Key Name *</label>
                <input
                  id="name"
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleFormChange}
                  placeholder="e.g., Production Key"
                  required
                />
              </div>

              <div className="form-group">
                <label htmlFor="tenant_ids">Tenant IDs *</label>
                <textarea
                  id="tenant_ids"
                  name="tenant_ids"
                  value={formData.tenant_ids}
                  onChange={handleFormChange}
                  placeholder="Enter tenant UUIDs, comma-separated"
                  rows={3}
                  required
                />
                <small>
                  Leave empty for global access. Multiple UUIDs: comma-separated.
                </small>
              </div>

              <div className="form-group">
                <label>Roles</label>
                <div className="checkbox-group">
                  {["USER", "TENANT_ADMIN", "GLOBAL_OPS"].map((role) => (
                    <label key={role} className="checkbox-label">
                      <input
                        type="checkbox"
                        value={role}
                        checked={formData.roles.includes(role)}
                        onChange={handleRoleChange}
                      />
                      {role}
                    </label>
                  ))}
                </div>
              </div>

              <div className="form-actions">
                <button
                  type="button"
                  className="btn btn-secondary"
                  onClick={() => setShowCreateForm(false)}
                >
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary">
                  Create API Key
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* API Keys Table */}
      <div className="api-keys-table-container">
        {loading && <div className="loading">Loading API keys...</div>}

        {!loading && keys.length === 0 && (
          <div className="empty-state">
            <p>No API keys found. Create one to get started.</p>
          </div>
        )}

        {!loading && keys.length > 0 && (
          <table className="api-keys-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Tenant IDs</th>
                <th>Roles</th>
                <th>Status</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {keys.map((key: APIKey) => (
                <tr
                  key={key.id}
                  className={key.is_revoked ? "revoked" : ""}
                >
                  <td className="name-col">
                    <strong>{key.name}</strong>
                  </td>
                  <td className="tenant-ids-col">
                    {key.tenant_ids?.length === 0 ? (
                      <span className="badge badge-global">Global</span>
                    ) : (
                      <span className="badge badge-scoped">
                        {key.tenant_ids?.length || 0} tenant(s)
                      </span>
                    )}
                  </td>
                  <td className="roles-col">
                    {key.roles?.map((role) => (
                      <span key={role} className="role-tag">
                        {role}
                      </span>
                    ))}
                  </td>
                  <td className="status-col">
                    {key.is_revoked ? (
                      <span className="badge badge-revoked">Revoked</span>
                    ) : (
                      <span className="badge badge-active">Active</span>
                    )}
                  </td>
                  <td className="created-col">
                    {new Date(key.created_at).toLocaleDateString()}
                  </td>
                  <td className="actions-col">
                    <button className="link-btn">Usage</button>
                    <button className="link-btn">Revoke</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Pagination */}
      {pageCount > 1 && (
        <div className="pagination">
          <button
            disabled={offset === 0}
            onClick={() => setOffset(Math.max(0, offset - limit))}
          >
            Previous
          </button>
          <span className="page-info">
            Page {currentPage} of {pageCount} ({total} total)
          </span>
          <button
            disabled={offset + limit >= total}
            onClick={() => setOffset(offset + limit)}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};

export default APIKeysPage;
