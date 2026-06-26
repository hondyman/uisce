import { useState, useEffect } from 'react';
// Import the Tenant type
import { Tenant } from '../types';
// TenantGrid may not be used in this simplified page; prefix to avoid unused import lint
// TenantGrid intentionally not imported here to keep HomePage minimal
import { JSX } from 'react';

import apiClient from '../utils/apiClient';

export default function HomePage(): JSX.Element {
  // Explicitly define the type for the state array
  const [tenants, setTenants] = useState<Tenant[]>([]);
  // tenants fetched and stored for future use
  const [newTenantName, setNewTenantName] = useState('');
  const [newTenantInstance, setNewTenantInstance] = useState('');

  useEffect(() => {
    // Fetch tenants from your backend API
    apiClient('tenants')
      .then((res) => res.json())
      // Ensure the fetched data is treated as an array of Tenants
      .then((data: Tenant[]) => setTenants(data));
  }, []);

  const handleCreateTenant = () => {
    apiClient('tenants', {
      method: 'POST',
      body: JSON.stringify({
        name: newTenantName,
        instance: newTenantInstance,
      }),
    })
      .then((res) => res.json())
      .then((newTenant: Tenant) => {
        // Now TypeScript knows that newTenant is a Tenant and can be added to the array
        setTenants((prevTenants) => [...prevTenants, newTenant]);
        setNewTenantName('');
        setNewTenantInstance('');
      });
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50/30 to-slate-100 dark:from-slate-950 dark:via-slate-900 dark:to-slate-950 p-8">
      <div className="max-w-4xl mx-auto">
        <div className="bg-white dark:bg-surface-dark rounded-xl border border-slate-200 dark:border-border-dark p-8 shadow-lg">
          <h1 className="text-3xl font-bold text-slate-900 dark:text-text-light mb-2">Tenants and Instances</h1>
          <p className="text-slate-600 dark:text-text-dim mb-8">Manage your tenant configurations and instances</p>

          <div className="mb-6">
            <div className="text-sm text-slate-500 dark:text-text-dim mb-2">
              Loaded tenants: <span className="font-semibold text-slate-900 dark:text-text-light">{tenants.length}</span>
            </div>
          </div>

          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-text-light mb-2">
                  New Tenant Name
                </label>
                <input
                  type="text"
                  value={newTenantName}
                  onChange={(e) => setNewTenantName(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 dark:border-border-dark px-4 py-2 bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light placeholder-slate-400 dark:placeholder-text-dim focus:border-blue-500 dark:focus:border-blue-400 focus:ring-2 focus:ring-blue-500/20 dark:focus:ring-blue-400/20 transition-colors"
                  placeholder="Enter tenant name"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-text-light mb-2">
                  New Instance Name
                </label>
                <input
                  type="text"
                  value={newTenantInstance}
                  onChange={(e) => setNewTenantInstance(e.target.value)}
                  className="w-full rounded-lg border border-slate-300 dark:border-border-dark px-4 py-2 bg-white dark:bg-surface-dark text-slate-900 dark:text-text-light placeholder-slate-400 dark:placeholder-text-dim focus:border-blue-500 dark:focus:border-blue-400 focus:ring-2 focus:ring-blue-500/20 dark:focus:ring-blue-400/20 transition-colors"
                  placeholder="Enter instance name"
                />
              </div>
            </div>

            <button
              onClick={handleCreateTenant}
              className="inline-flex items-center px-6 py-3 bg-blue-600 hover:bg-blue-700 dark:bg-blue-500 dark:hover:bg-blue-600 text-white font-medium rounded-lg transition-colors shadow-sm hover:shadow-md"
            >
              Create Tenant
            </button>
          </div>

          {/* Assuming you have a TenantGrid component that can display the tenants */}
          {/* <TenantGrid tenants={tenants} /> */}
        </div>
      </div>
    </div>
  );
}