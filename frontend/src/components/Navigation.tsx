import React from 'react';
import BlockableLink from './RouteBlocker/BlockableLink';
import { useTenant } from '../contexts/TenantContext';

// --- Navigation Data Structure ---

type MenuItem = {
  id: string;
  label: string;
  description?: string;
  to: string;
  icon?: React.ReactNode;
};

type MenuGroup = {
  id: string;
  label: string;
  items: MenuItem[];
};

const ANALYTICS_MENU: MenuGroup = {
  id: 'analytics',
  label: 'Analytics',
  items: [
    { id: 'metrics', label: 'Metrics Console', description: 'Centralized metrics definition and monitoring.', to: '/metrics', icon: '📊' },
    { id: 'nlq', label: 'Ask AI', description: 'Natural language query interface.', to: '/nlq', icon: '🤖' },
    { id: 'fixed-income', label: 'Fixed Income Analytics', description: 'Bond analytics and yield curve analysis.', to: '/fixed-income' },
    { id: 'scenario', label: 'Scenario Analysis', description: 'Portfolio scenario projection and analysis.', to: '/analytics/scenario-analysis' },
    { id: 'rebalancer', label: 'Portfolio Rebalancer', description: 'AI-powered portfolio rebalancing.', to: '/analytics/rebalancer' },
  ]
};

const DATA_FABRIC_MENU: MenuGroup = {
  id: 'catalog',
  label: 'Catalog',
  items: [
    { id: 'bundle-explorer', label: 'Bundle Explorer', description: 'Visual explorer for data bundles.', to: '/bundle-explorer' },
    { id: 'bundles', label: 'Data Bundles', description: 'Create and publish semantic data bundles.', to: '/fabric/bundles' },
    { id: 'glossary', label: 'Business Glossary', description: 'Manage semantic and business terms.', to: '/core/glossary' },
    { id: 'semantic-mapper', label: 'Semantic Mapper', description: 'Map database columns to semantic terms.', to: '/core/semantic-mapper' },

    { id: 'domains', label: 'Domains', description: 'Manage data domain hierarchy.', to: '/core/domains' },
    { id: 'preaggs', label: 'Pre-Aggregations', description: 'Manage pre-aggregation definitions.', to: '/fabric/preaggregations' },
    { id: 'calc', label: 'Calculations Library', description: 'Reusable calculation templates.', to: '/fabric/calculations' },
  ]
};

const GOVERNANCE_MENU: MenuGroup = {
  id: 'governance',
  label: 'Governance',
  items: [

    { id: 'validation-run', label: 'Run Validations', description: 'Execute validations and view results.', to: '/core/validation' },
    { id: 'regulator', label: 'Regulator Portal', description: 'Compliance and audit dashboard.', to: '/core/regulator-portal' },
    { id: 'audit', label: 'Audit Explorer', description: 'Unified audit record chain.', to: '/core/audit-explorer' },
    { id: 'access-explanation', label: 'Access Explanation', description: 'Explain why access was granted or denied.', to: '/access-explanation' },
    { id: 'jit', label: 'JIT Requests', description: 'Just-in-time access requests.', to: '/jit-request' },
    { id: 'ip-whitelist', label: 'IP Whitelist', description: 'Manage allowed IP addresses.', to: '/fabric/ip-whitelist' },
  ]
};

const WORKFLOWS_MENU: MenuGroup = {
  id: 'workflows',
  label: 'Workflows',
  items: [
    { id: 'inbox', label: 'Approval Inbox', description: 'Manage pending approvals.', to: '/core/approval-inbox' },
    { id: 'dashboard', label: 'Workflows Dashboard', description: 'Monitor approval workflows.', to: '/core/approval-workflows' },
    { id: 'designer', label: 'Workflow Designer', description: 'Visual process designer.', to: '/core/workflow-designer' },
    { id: 'process-catalog', label: 'Process Catalog', description: 'Versioned process definitions.', to: '/core/process-catalog' },
    { id: 'sla', label: 'SLA Dashboard', description: 'Monitor service level agreements.', to: '/core/sla-dashboard' },
    { id: 'notifications', label: 'Notification Center', description: 'System alerts and notifications.', to: '/core/notifications' },
    { id: 'business-objects', label: 'Business Objects', description: 'Object lifecycle explorer.', to: '/core/business-objects' },
  ]
};

const SCHEDULER_MENU: MenuGroup = {
  id: 'scheduler',
  label: 'Scheduler',
  items: [
    { id: 'scheduler-dashboard', label: 'Dashboard', description: 'Scheduler overview and metrics.', to: '/scheduler', icon: '📊' },
    { id: 'scheduler-jobs', label: 'Jobs', description: 'Manage scheduled jobs.', to: '/scheduler/jobs', icon: '📋' },
    { id: 'scheduler-executions', label: 'Executions', description: 'View job execution history.', to: '/scheduler/executions', icon: '▶️' },
    { id: 'scheduler-dependencies', label: 'Dependencies', description: 'Visualize job dependencies.', to: '/scheduler/dependencies', icon: '🔗' },
    { id: 'scheduler-calendars', label: 'Business Calendars', description: 'Manage business calendars.', to: '/scheduler/calendars', icon: '📅' },
    { id: 'scheduler-notifications', label: 'Notifications', description: 'Configure notification templates.', to: '/scheduler/notifications', icon: '🔔' },
    { id: 'scheduler-compliance', label: 'Compliance', description: 'Audit logs and compliance reports.', to: '/scheduler/compliance', icon: '📊' },
  ]
};

const SECRETS_MENU: MenuGroup = {
  id: 'secrets',
  label: 'Secrets',
  items: [
    { id: 'secrets-config', label: 'Configuration', description: 'Manage secret paths and rotation policies.', to: '/secrets/config', icon: '🔐' },
    { id: 'secrets-audit', label: 'Audit Log', description: 'Track secret access and modifications.', to: '/secrets/audit', icon: '📋' },
    { id: 'secrets-monitoring', label: 'Monitoring', description: 'Health dashboard and rotation status.', to: '/secrets/monitoring', icon: '📊' },
  ]
};

const SETTINGS_MENU: MenuGroup = {
  id: 'settings',
  label: 'Settings',
  items: [
    { id: 'tenants', label: 'Tenant Management', description: 'Manage tenants and data sources.', to: '/tenants' },
    { id: 'roles', label: 'Role Management', description: 'Assign permissions to roles.', to: '/fabric/roles' },
    { id: 'llm', label: 'LLM Configuration', description: 'Configure AI model providers.', to: '/admin/llm' },
    { id: 'temporal', label: 'Temporal Ops', description: 'Monitor workflow executions.', to: '/admin/temporal-ops' },
    { id: 'seeding', label: 'Data Seeding', description: 'Seed database with initial data.', to: '/admin/seeding' },
    { id: 'node-types', label: 'Node Types', description: 'Configure glossary node types.', to: '/core/node-types' },
    { id: 'dynamic-ui', label: 'Dynamic UI Generator', description: 'Generate forms from definitions.', to: '/dynamic-ui' },
    { id: 'custom-components', label: 'Custom Components', description: 'Manage custom UI components.', to: '/fabric/custom-components' },
  ]
};

// --- Components ---

function NavDropdown({ group, rightAligned = false }: { group: MenuGroup; rightAligned?: boolean }) {
  const [open, setOpen] = React.useState(false);
  const containerRef = React.useRef<HTMLDivElement>(null);
  const buttonRef = React.useRef<HTMLButtonElement>(null);

  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') setOpen(false);
  };

  return (
    <div className="relative" ref={containerRef}>
      <button
        ref={buttonRef}
        type="button"
        onClick={() => setOpen(!open)}
        onKeyDown={handleKeyDown}
        className={`px-3 py-2 rounded-md text-sm font-medium hover:bg-gray-200 flex items-center gap-1 ${open ? 'bg-gray-200' : ''}`}
        aria-expanded={open}
        aria-haspopup="true"
      >
        {group.label}
        <span className="text-xs opacity-50">▼</span>
      </button>

      {open && (
        <div
          className={`absolute ${rightAligned ? 'right-0' : 'left-0'} mt-2 w-80 bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50 max-h-[80vh] overflow-y-auto`}
        >
          <div className="py-1" role="menu">
            {group.items.map((item) => (
              <BlockableLink
                key={item.id}
                to={item.to}
                className="block px-4 py-3 hover:bg-gray-50 transition-colors"
                onClick={() => setOpen(false)}
                role="menuitem"
              >
                <div className="flex items-start gap-3">
                  {item.icon && <span className="text-lg mt-0.5">{item.icon}</span>}
                  <div>
                    <div className="text-sm font-medium text-gray-900">{item.label}</div>
                    {item.description && (
                      <div className="text-xs text-gray-500 mt-0.5">{item.description}</div>
                    )}
                  </div>
                </div>
              </BlockableLink>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// Special handling for Recent Bundles in Data Fabric if needed, 
// but for now we'll stick to the clean structure. 
// We can add a "Recent" section to Data Fabric later if requested.

export function Navigation() {
  return (
    <nav className="bg-white border-b border-gray-200 px-4 py-2.5 flex items-center justify-between shadow-sm mb-6">
      <div className="flex items-center gap-2">
        <div className="font-bold text-xl text-indigo-600 mr-4">SemLayer</div>
        
        <div className="flex items-center gap-1">
          <NavDropdown group={ANALYTICS_MENU} />
          <NavDropdown group={DATA_FABRIC_MENU} />
          <NavDropdown group={GOVERNANCE_MENU} />
          <NavDropdown group={WORKFLOWS_MENU} />
          <NavDropdown group={SCHEDULER_MENU} />
          <NavDropdown group={SECRETS_MENU} />
          <NavDropdown group={SETTINGS_MENU} />
        </div>
      </div>

      <div className="flex items-center gap-4">
        {/* Context Indicator */}
        <ContextIndicator />
      </div>
    </nav>
  );
}

function ContextIndicator() {
  const { tenant, datasource, isSelected } = useTenant();

  if (!isSelected || !tenant || !datasource) {
    return (
      <BlockableLink 
        to="/tenants" 
        className="text-sm text-red-600 font-medium hover:bg-red-50 px-3 py-1.5 rounded border border-red-200"
      >
        Select Datasource
      </BlockableLink>
    );
  }

  return (
    <BlockableLink 
      to="/tenants" 
      className="flex flex-col items-end text-xs hover:bg-gray-50 px-3 py-1 rounded transition-colors"
      title="Click to change context"
    >
      <span className="font-semibold text-gray-900">{tenant.name}</span>
      <span className="text-gray-500">{datasource.source_name}</span>
    </BlockableLink>
  );
}
