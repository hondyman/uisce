import React, { useState } from 'react';
import { Routes, Route } from 'react-router-dom';
import { CubeAdminDashboard } from './pages/CubeAdminDashboard';
import { CubeCatalogPage } from './pages/CubeCatalogPage';
import { CubeQueryAnalyticsPage } from './pages/CubeQueryAnalyticsPage';
import { CubePreAggPage } from './pages/CubePreAggPage';
import { CubeReportsPage } from './pages/CubeReportsPage';
import { CubeOrganizationsPage } from './pages/CubeOrganizationsPage';
import { CubeTenantsPage } from './pages/CubeTenantsPage';
import { CubeSettingsPage } from './pages/CubeSettingsPage';
import { CubeModelWizard } from './CubeModelWizard';
import { CubeYamlEditor } from './CubeYamlEditor';
import CubeWorkersPage from './CubeWorkersPage';
import PreAggregationBuilder from './PreAggregationBuilder';

export const cubeAdminRoutes = [
  {
    path: '/cube-admin',
    element: <CubeAdminLayout />,
    children: [
      { index: true, element: <CubeAdminDashboard /> },
      { path: 'catalog', element: <CubeCatalogPage /> },
      { path: 'analytics', element: <CubeQueryAnalyticsPage /> },
      { path: 'preaggs', element: <CubePreAggPage /> },
      { path: 'preaggs/builder', element: <PreAggregationBuilder /> },
      { path: 'reports', element: <CubeReportsPage /> },
      { path: 'organizations', element: <CubeOrganizationsPage /> },
      { path: 'tenants', element: <CubeTenantsPage /> },
      { path: 'settings', element: <CubeSettingsPage /> },
      { path: 'models', element: <CubeModelsListPage /> },
      { path: 'models/wizard', element: <CubeModelWizard /> },
      { path: 'models/wizard/:sessionId', element: <CubeModelWizard /> },
      { path: 'models/:modelId/edit', element: <CubeYamlEditorPage /> },
      { path: 'workers', element: <CubeWorkersPage /> },
    ],
  },
];

// Wrapper page for model listing
function CubeModelsListPage() {
  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Cube Models</h1>
          <p className="text-gray-500 mt-1">Manage core and custom semantic layer models</p>
        </div>
        <div className="flex gap-2">
          <a
            href="/cube-admin/models/wizard"
            className="inline-flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors"
          >
            <WizardIcon className="w-5 h-5 mr-2" />
            Create with Wizard
          </a>
        </div>
      </div>
      <div className="bg-white rounded-lg shadow p-6">
        <p className="text-gray-500 text-center py-12">
          Model listing will be populated from the catalog integration.
          Use the wizard to create new models.
        </p>
      </div>
    </div>
  );
}

// Wrapper for YAML editor with state management
function CubeYamlEditorPage() {
  const [value, setValue] = useState('');
  
  return (
    <div className="h-full">
      <CubeYamlEditor
        value={value}
        onChange={setValue}
        onSave={(_yaml) => { /* Save handler - would call API */ }}
      />
    </div>
  );
}

function CubeAdminLayout() {
  return (
    <div className="flex h-screen bg-gray-50">
      <CubeAdminSidebar />
      <main className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<CubeAdminDashboard />} />
          <Route path="catalog" element={<CubeCatalogPage />} />
          <Route path="analytics" element={<CubeQueryAnalyticsPage />} />
          <Route path="preaggs" element={<CubePreAggPage />} />
          <Route path="preaggs/builder" element={<PreAggregationBuilder />} />
          <Route path="reports" element={<CubeReportsPage />} />
          <Route path="models" element={<CubeModelsListPage />} />
          <Route path="models/wizard" element={<CubeModelWizard />} />
          <Route path="models/wizard/:sessionId" element={<CubeModelWizard />} />
          <Route path="models/:modelId/edit" element={<CubeYamlEditorPage />} />
          <Route path="workers" element={<CubeWorkersPage />} />
          <Route path="organizations" element={<CubeOrganizationsPage />} />
          <Route path="tenants" element={<CubeTenantsPage />} />
          <Route path="settings" element={<CubeSettingsPage />} />
        </Routes>
      </main>
    </div>
  );
}

function CubeAdminSidebar() {
  const navItems = [
    { path: '/cube-admin', label: 'Dashboard', icon: DashboardIcon },
    { path: '/cube-admin/models', label: 'Model Builder', icon: WizardIcon },
    { path: '/cube-admin/catalog', label: 'Cube Catalog', icon: CubeIcon },
    { path: '/cube-admin/analytics', label: 'Query Analytics', icon: AnalyticsIcon },
    { path: '/cube-admin/preaggs', label: 'Pre-Aggregations', icon: LayersIcon },
    { path: '/cube-admin/workers', label: 'Workers', icon: WorkerIcon },
    { path: '/cube-admin/reports', label: 'Scheduled Reports', icon: ReportIcon },
    { divider: true },
    { path: '/cube-admin/organizations', label: 'Organizations', icon: OrgIcon, superAdmin: true },
    { path: '/cube-admin/tenants', label: 'Tenants', icon: TenantsIcon },
    { path: '/cube-admin/settings', label: 'Settings', icon: SettingsIcon },
  ];

  return (
    <aside className="w-64 bg-white border-r border-gray-200">
      <div className="p-4 border-b border-gray-200">
        <h1 className="text-xl font-bold text-indigo-600 flex items-center gap-2">
          <CubeIcon className="w-6 h-6" />
          Cube Admin
        </h1>
        <p className="text-xs text-gray-500 mt-1">Semantic Layer Console</p>
      </div>
      <nav className="p-4 space-y-1">
        {navItems.map((item, idx) =>
          item.divider ? (
            <hr key={idx} className="my-3 border-gray-200" />
          ) : (
            <NavLink key={item.path || idx} to={item.path || ''} item={item} />
          )
        )}
      </nav>
    </aside>
  );
}

function NavLink({ to, item }: { to: string; item: any }) {
  const Icon = item.icon;
  return (
    <a
      href={to}
      className="flex items-center gap-3 px-3 py-2 text-sm text-gray-700 rounded-lg hover:bg-gray-100 transition-colors"
    >
      <Icon className="w-5 h-5 text-gray-400" />
      {item.label}
      {item.superAdmin && (
        <span className="ml-auto text-xs bg-purple-100 text-purple-700 px-2 py-0.5 rounded">
          Super
        </span>
      )}
    </a>
  );
}

// Icons
function DashboardIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
    </svg>
  );
}

function CubeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  );
}

function AnalyticsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
    </svg>
  );
}

function LayersIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
    </svg>
  );
}

function ReportIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
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

function TenantsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
    </svg>
  );
}

function SettingsIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
    </svg>
  );
}

function WizardIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19.428 15.428a2 2 0 00-1.022-.547l-2.387-.477a6 6 0 00-3.86.517l-.318.158a6 6 0 01-3.86.517L6.05 15.21a2 2 0 00-1.806.547M8 4h8l-1 1v5.172a2 2 0 00.586 1.414l5 5c1.26 1.26.367 3.414-1.415 3.414H4.828c-1.782 0-2.674-2.154-1.414-3.414l5-5A2 2 0 009 10.172V5L8 4z" />
    </svg>
  );
}

function WorkerIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
    </svg>
  );
}
