// Tenants Page - Main tenant management interface

import React, { useState } from "react";
import { useTenants } from "../hooks/useAdmin";
import { Tenant, PLANS } from "../types";
import { useAuth } from "../../contexts/AuthContext";
import { ImpersonationModal } from "../../components/admin/ImpersonationModal";
import "./TenantsPage.css";

interface TenantFormData {
  name: string;
  code: string;
  region: string;
  plan: "free" | "pro" | "enterprise";
}

export const TenantsPage: React.FC = () => {
  const [limit, setLimit] = useState(50);
  const [offset, setOffset] = useState(0);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [formData, setFormData] = useState<TenantFormData>({
    name: "",
    code: "",
    region: "us-east-1",
    plan: "free",
  });
  const [impersonateTenant, setImpersonateTenant] = useState<Tenant | null>(null);

  const { isGlobalAdmin } = useAuth();
  const { tenants, total, loading, error, refetch } = useTenants(limit, offset);

  const handleCreateClick = () => {
    setShowCreateForm(true);
  };

  const handleFormChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({
      ...prev,
      [name]: value,
    }));
  };

  const handleCreateTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch("/api/admin/tenants", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });
      if (!response.ok) {
        const error = await response.json();
        alert(`Error: ${error.error || "Failed to create tenant"}`);
        return;
      }
      refetch();
      setShowCreateForm(false);
      setFormData({
        name: "",
        code: "",
        region: "us-east-1",
        plan: "free",
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
    <div className="tenants-page">
      <div className="page-header">
        <h1>Tenants</h1>
        <button className="btn btn-primary" onClick={handleCreateClick}>
          + New Tenant
        </button>
      </div>

      {error && <div className="alert alert-error">{error}</div>}

      {/* Create Form Modal */}
      {showCreateForm && (
        <div className="modal-overlay">
          <div className="modal">
            <div className="modal-header">
              <h2>Create New Tenant</h2>
              <button
                className="modal-close"
                onClick={() => setShowCreateForm(false)}
              >
                ✕
              </button>
            </div>
            <form onSubmit={handleCreateTenant} className="tenant-form">
              <div className="form-group">
                <label htmlFor="name">Tenant Name *</label>
                <input
                  id="name"
                  type="text"
                  name="name"
                  value={formData.name}
                  onChange={handleFormChange}
                  placeholder="e.g., Acme Corp"
                  required
                />
              </div>

              <div className="form-group">
                <label htmlFor="code">Tenant Code *</label>
                <input
                  id="code"
                  type="text"
                  name="code"
                  value={formData.code}
                  onChange={handleFormChange}
                  placeholder="e.g., acme-corp"
                  required
                />
              </div>

              <div className="form-row">
                <div className="form-group">
                  <label htmlFor="region">Region</label>
                  <select
                    id="region"
                    name="region"
                    value={formData.region}
                    onChange={handleFormChange}
                  >
                    <option value="us-east-1">US East 1</option>
                    <option value="us-west-2">US West 2</option>
                    <option value="eu-west-1">EU West 1</option>
                    <option value="eu-central-1">EU Central 1</option>
                    <option value="ap-southeast-1">AP Southeast 1</option>
                    <option value="ap-northeast-1">AP Northeast 1</option>
                  </select>
                </div>

                <div className="form-group">
                  <label htmlFor="plan">Plan</label>
                  <select
                    id="plan"
                    name="plan"
                    value={formData.plan}
                    onChange={handleFormChange as any}
                  >
                    <option value="free">Free (100 req/day)</option>
                    <option value="pro">Pro (10k req/day)</option>
                    <option value="enterprise">Enterprise (Unlimited)</option>
                  </select>
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
                  Create Tenant
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Tenants Table */}
      <div className="tenants-table-container">
        {loading && <div className="loading">Loading tenants...</div>}

        {!loading && tenants.length === 0 && (
          <div className="empty-state">
            <p>No tenants found. Create one to get started.</p>
          </div>
        )}

        {!loading && tenants.length > 0 && (
          <table className="tenants-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Code</th>
                <th>Region</th>
                <th>Plan</th>
                <th>Status</th>
                <th>Rate Limit</th>
                <th>Created</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {tenants.map((tenant) => (
                <tr
                  key={tenant.id}
                  className={tenant.is_suspended ? "suspended" : ""}
                >
                  <td className="name-col">
                    <strong>{tenant.name}</strong>
                  </td>
                  <td className="code-col">{tenant.code}</td>
                  <td className="region-col">{tenant.region}</td>
                  <td className="plan-col">
                    <span className={`badge badge-${tenant.plan}`}>
                      {tenant.plan.charAt(0).toUpperCase() +
                        tenant.plan.slice(1)}
                    </span>
                  </td>
                  <td className="status-col">
                    {tenant.is_suspended ? (
                      <span className="badge badge-suspended">Suspended</span>
                    ) : (
                      <span className="badge badge-active">Active</span>
                    )}
                  </td>
                  <td className="limit-col">
                    {tenant.max_requests} / {tenant.window_seconds}s
                  </td>
                  <td className="created-col">
                    {new Date(tenant.created_at).toLocaleDateString()}
                  </td>
                  <td className="actions-col">
                    <button className="link-btn">Details</button>
                    <button className="link-btn">Edit</button>
                    {isGlobalAdmin() && (
                      <button 
                        className="link-btn"
                        onClick={() => setImpersonateTenant(tenant)}
                        style={{ color: '#d97706', fontWeight: 600 }}
                      >
                        Assume Context
                      </button>
                    )}
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

      {impersonateTenant && (
        <ImpersonationModal
          open={!!impersonateTenant}
          onClose={() => setImpersonateTenant(null)}
          targetTenantId={impersonateTenant.id}
          targetTenantName={impersonateTenant.name}
        />
      )}
    </div>
  );
};

export default TenantsPage;
