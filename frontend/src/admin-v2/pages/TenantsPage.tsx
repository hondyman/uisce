import React, { useState } from "react";
import { Card } from "../components/Card";
import { Table } from "../components/Table";
import { Spinner } from "../components/Feedback";
import { CreateTenantModal } from "../components/CreateTenantModal";
import { useTenants } from "../hooks/useTenants";
import "./TenantsPage.css";

export function TenantsPage() {
  const [showCreateModal, setShowCreateModal] = useState(false);
  const tenantsQuery = useTenants();

  const tenants = tenantsQuery.data?.data || [];

  const columns = ["Name", "Code", "Plan", "Region", "Status", "Created"];
  const rows = tenants.map((tenant) => [
    tenant.name,
    tenant.code,
    (
      <span className={`badge badge-${tenant.plan}`}>
        {tenant.plan.charAt(0).toUpperCase() + tenant.plan.slice(1)}
      </span>
    ),
    tenant.region,
    (
      <span
        className={`badge ${
          tenant.suspended ? "badge-suspended" : "badge-active"
        }`}
      >
        {tenant.suspended ? "Suspended" : "Active"}
      </span>
    ),
    new Date(tenant.createdAt).toLocaleDateString(),
  ]);

  return (
    <div className="page">
      <div className="page-header">
        <div>
          <h1>Tenants</h1>
          <p className="page-subtitle">Manage your SaaS customers</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary"
        >
          + Create Tenant
        </button>
      </div>

      <Card className="grid-1">
        {tenantsQuery.isLoading ? (
          <Spinner size="md" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            loading={tenantsQuery.isLoading}
            empty="No tenants yet"
          />
        )}
      </Card>

      <CreateTenantModal
        open={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={() => {
          // List will auto-refresh via React Query invalidation
        }}
      />
    </div>
  );
}
