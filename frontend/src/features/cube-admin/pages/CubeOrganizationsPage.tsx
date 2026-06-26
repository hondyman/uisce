import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface Organization {
  id: string;
  name: string;
  display_name: string;
  description: string;
  contact_email: string;
  tenant_count: number;
  created_at: string;
  updated_at: string;
}

interface OrganizationTenant {
  tenant_id: string;
  tenant_name: string;
  role: string;
  added_at: string;
}

export function CubeOrganizationsPage() {
  const { tenant, datasource } = useTenant();
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedOrg, setSelectedOrg] = useState<Organization | null>(null);
  const [orgTenants, setOrgTenants] = useState<OrganizationTenant[]>([]);

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadOrganizations();
  }, [tenant?.id, datasource?.id]);

  const loadOrganizations = async () => {
    setLoading(true);
    try {
      const res = await fetch(
        `/api/cube-admin/organizations?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}`
      );
      if (res.ok) {
        const data = await res.json();
        setOrganizations(data || []);
      }
    } catch (err) {
      console.error('Failed to load organizations:', err);
    } finally {
      setLoading(false);
    }
  };

  const loadOrgTenants = async (orgId: string) => {
    try {
      const res = await fetch(
        `/api/cube-admin/organizations/${orgId}/tenants?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}`
      );
      if (res.ok) {
        const data = await res.json();
        setOrgTenants(data || []);
      }
    } catch (err) {
      console.error('Failed to load org tenants:', err);
    }
  };

  const selectOrganization = (org: Organization) => {
    setSelectedOrg(org);
    loadOrgTenants(org.id);
  };

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to manage organizations.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Organizations</h1>
          <p className="text-gray-500 mt-1">
            Manage multi-tenant organization hierarchy for MSP management
          </p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2"
        >
          <PlusIcon className="w-5 h-5" />
          New Organization
        </button>
      </div>

      {/* Info Banner */}
      <div className="bg-purple-50 border border-purple-200 rounded-lg p-4 mb-8">
        <div className="flex items-start gap-3">
          <InfoIcon className="w-5 h-5 text-purple-600 mt-0.5" />
          <div>
            <h3 className="font-medium text-purple-900">MSP Organization Model</h3>
            <p className="text-sm text-purple-700 mt-1">
              Organizations allow you to group multiple tenants under a single management umbrella.
              This is ideal for managed service providers (MSPs) who manage Cube analytics for their clients.
              Each organization can have multiple tenants assigned with different roles.
            </p>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-8">
        {/* Organizations List */}
        <div className="col-span-2">
          {loading ? (
            <LoadingSkeleton />
          ) : organizations.length === 0 ? (
            <EmptyState onCreateClick={() => setShowCreateModal(true)} />
          ) : (
            <div className="space-y-4">
              {organizations.map((org) => (
                <OrganizationCard
                  key={org.id}
                  organization={org}
                  isSelected={selectedOrg?.id === org.id}
                  onSelect={() => selectOrganization(org)}
                />
              ))}
            </div>
          )}
        </div>

        {/* Selected Organization Detail */}
        <div className="col-span-1">
          {selectedOrg ? (
            <OrganizationDetail
              organization={selectedOrg}
              tenants={orgTenants}
              onAddTenant={() => {}}
              onRemoveTenant={() => {}}
            />
          ) : (
            <div className="bg-gray-50 rounded-xl border border-dashed border-gray-300 p-8 text-center">
              <OrgIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
              <p className="text-gray-500">Select an organization to view details</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Modal */}
      {showCreateModal && (
        <CreateOrganizationModal
          onClose={() => setShowCreateModal(false)}
          onCreated={() => {
            setShowCreateModal(false);
            loadOrganizations();
          }}
          tenantId={tenant.id}
          datasourceId={datasource.id}
        />
      )}
    </div>
  );
}

interface OrganizationCardProps {
  organization: Organization;
  isSelected: boolean;
  onSelect: () => void;
}

function OrganizationCard({ organization, isSelected, onSelect }: OrganizationCardProps) {
  return (
    <div
      onClick={onSelect}
      className={`bg-white rounded-xl border p-6 cursor-pointer transition-all ${
        isSelected
          ? 'border-indigo-500 ring-2 ring-indigo-200'
          : 'border-gray-200 hover:border-indigo-300'
      }`}
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-purple-100 text-purple-600">
            <OrgIcon className="w-6 h-6" />
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">{organization.display_name}</h3>
            <p className="text-sm text-gray-500">{organization.name}</p>
          </div>
        </div>
        <span className="text-sm bg-gray-100 text-gray-600 px-2 py-1 rounded">
          {organization.tenant_count || 0} tenants
        </span>
      </div>
      <p className="text-sm text-gray-600 mb-4">
        {organization.description || 'No description'}
      </p>
      <div className="flex items-center justify-between text-xs text-gray-400">
        <span>{organization.contact_email || 'No contact email'}</span>
        <span>Created {new Date(organization.created_at).toLocaleDateString()}</span>
      </div>
    </div>
  );
}

interface OrganizationDetailProps {
  organization: Organization;
  tenants: OrganizationTenant[];
  onAddTenant: () => void;
  onRemoveTenant: (tenantId: string) => void;
}

function OrganizationDetail({ organization, tenants, onAddTenant, onRemoveTenant }: OrganizationDetailProps) {
  return (
    <div className="bg-white rounded-xl border border-gray-200 overflow-hidden">
      <div className="p-6 border-b border-gray-200">
        <h2 className="text-lg font-semibold text-gray-900">{organization.display_name}</h2>
        <p className="text-sm text-gray-500 mt-1">{organization.description}</p>
      </div>

      <div className="p-6 space-y-4">
        <div>
          <label className="text-xs font-medium text-gray-500 uppercase">Contact Email</label>
          <p className="text-gray-900">{organization.contact_email || '-'}</p>
        </div>
        <div>
          <label className="text-xs font-medium text-gray-500 uppercase">Created</label>
          <p className="text-gray-900">{new Date(organization.created_at).toLocaleDateString()}</p>
        </div>
      </div>

      <div className="border-t border-gray-200">
        <div className="px-6 py-4 flex items-center justify-between bg-gray-50">
          <h3 className="font-medium text-gray-900">Assigned Tenants</h3>
          <button
            onClick={onAddTenant}
            className="text-sm text-indigo-600 hover:text-indigo-700"
          >
            + Add Tenant
          </button>
        </div>
        <div className="divide-y divide-gray-100">
          {tenants.length === 0 ? (
            <div className="p-6 text-center text-gray-500 text-sm">
              No tenants assigned to this organization
            </div>
          ) : (
            tenants.map((t) => (
              <div key={t.tenant_id} className="px-6 py-3 flex items-center justify-between">
                <div>
                  <p className="font-medium text-gray-900">{t.tenant_name}</p>
                  <p className="text-xs text-gray-500">{t.role}</p>
                </div>
                <button
                  onClick={() => onRemoveTenant(t.tenant_id)}
                  className="text-red-500 hover:text-red-600"
                  aria-label={`Remove tenant ${t.tenant_name}`}
                >
                  <TrashIcon className="w-4 h-4" />
                </button>
              </div>
            ))
          )}
        </div>
      </div>

      <div className="p-6 border-t border-gray-200 flex gap-3">
        <button className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
          Edit Organization
        </button>
        <button className="px-4 py-2 border border-red-300 text-red-600 rounded-lg hover:bg-red-50 transition-colors">
          Delete
        </button>
      </div>
    </div>
  );
}

interface CreateOrganizationModalProps {
  onClose: () => void;
  onCreated: () => void;
  tenantId: string;
  datasourceId: string;
}

function CreateOrganizationModal({ onClose, onCreated, tenantId, datasourceId }: CreateOrganizationModalProps) {
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [contactEmail, setContactEmail] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    try {
      const res = await fetch(
        `/api/cube-admin/organizations?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name,
            display_name: displayName,
            description,
            contact_email: contactEmail,
          }),
        }
      );
      if (res.ok) {
        onCreated();
      }
    } catch (err) {
      console.error('Failed to create organization:', err);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/40" onClick={onClose} />
      <div className="relative bg-white rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-xl font-semibold text-gray-900 mb-6">Create Organization</h2>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label htmlFor="org-name" className="block text-sm font-medium text-gray-700 mb-1">
              Name (slug)
            </label>
            <input
              id="org-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="my-organization"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
          <div>
            <label htmlFor="org-display-name" className="block text-sm font-medium text-gray-700 mb-1">
              Display Name
            </label>
            <input
              id="org-display-name"
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="My Organization"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
              required
            />
          </div>
          <div>
            <label htmlFor="org-description" className="block text-sm font-medium text-gray-700 mb-1">
              Description
            </label>
            <textarea
              id="org-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="Description of the organization..."
              rows={3}
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div>
            <label htmlFor="org-email" className="block text-sm font-medium text-gray-700 mb-1">
              Contact Email
            </label>
            <input
              id="org-email"
              type="email"
              value={contactEmail}
              onChange={(e) => setContactEmail(e.target.value)}
              placeholder="admin@organization.com"
              className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
            />
          </div>
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={submitting || !name || !displayName}
              className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors disabled:opacity-50"
            >
              {submitting ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

function EmptyState({ onCreateClick }: { onCreateClick: () => void }) {
  return (
    <div className="text-center py-12 bg-white rounded-xl border border-gray-200">
      <OrgIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">No organizations yet</h3>
      <p className="text-gray-500 mt-1">Create your first organization to start grouping tenants</p>
      <button
        onClick={onCreateClick}
        className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
      >
        Create Organization
      </button>
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="space-y-4">
      {[...Array(3)].map((_, i) => (
        <div key={i} className="bg-white rounded-xl border p-6 animate-pulse">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-gray-200 rounded-lg" />
            <div className="h-5 w-32 bg-gray-200 rounded" />
          </div>
          <div className="h-4 w-full bg-gray-100 rounded mb-2" />
          <div className="h-4 w-2/3 bg-gray-100 rounded" />
        </div>
      ))}
    </div>
  );
}

// Icons
function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

function InfoIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  );
}

function OrgIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4" />
    </svg>
  );
}

function TrashIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
    </svg>
  );
}
