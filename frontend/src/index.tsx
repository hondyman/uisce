import { useState, useEffect } from 'react';
import ClaimsTable from './ClaimsTable';
import RoleClaimMatrix from './RoleClaimMatrix';
import RequestInbox from './RequestInbox';
import SimulationViewer from './SimulationViewer';
import AuditTrail from './AuditTrail';
import GovernanceOverview from './GovernanceOverview';
import IndexMonitorDashboard from './IndexMonitorDashboard';
import ClaimLifecycleDashboard from './ClaimLifecycleDashboard';
import { getStewardDomains } from './api';
import './AccessControlDashboard.css';

type Tab = 'claims' | 'roles' | 'requests' | 'simulations' | 'audit' | 'overview' | 'indexing' | 'lifecycle';

export default function AccessControlDashboard() {
  const [activeTab, setActiveTab] = useState<Tab>('overview');
  // NEW: Steward state
  const [stewardDomains, setStewardDomains] = useState<string[]>([]);
  const [selectedDomain, setSelectedDomain] = useState<string>('all');
  const currentUser = "data_steward"; // Mock current user who is a steward

  // NEW: Fetch steward domains on load
  useEffect(() => {
    // In a real app, you'd get the current user from an auth context.
    getStewardDomains(currentUser).then(setStewardDomains).catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); });
  }, [currentUser]);

  const renderTabContent = () => {
    const domainFilter = selectedDomain === 'all' ? undefined : selectedDomain;

    switch (activeTab) {
      case 'claims':
        return <ClaimsTable domain={domainFilter} />;
      case 'roles':
        return <RoleClaimMatrix domain={domainFilter} />;
      case 'requests':
        return <RequestInbox domain={domainFilter} />;
      case 'simulations':
        return <SimulationViewer />;
      case 'audit':
        return <AuditTrail />;
      case 'overview':
        return <GovernanceOverview />;
      case 'indexing':
        return <IndexMonitorDashboard />;
      case 'lifecycle':
        return <ClaimLifecycleDashboard />;
      default:
        return null;
    }
  };

  return (
    <div className="access-control-dashboard">
      <header className="dashboard-header">
        <h1>Access Control Dashboard</h1>
        <p>Manage claims, roles, requests, and audit logs for your semantic models.</p>
        {/* NEW: Domain selector for stewards */}
        {stewardDomains.length > 0 && (
          <div className="steward-console-filter">
            <label htmlFor="domain-selector">Managing Domain:</label>
            <select
              id="domain-selector"
              value={selectedDomain}
              onChange={(e) => setSelectedDomain(e.target.value)}
            >
              <option value="all">All My Domains</option>
              {stewardDomains.map(domain => (
                <option key={domain} value={domain}>{domain.charAt(0).toUpperCase() + domain.slice(1)}</option>
              ))}
            </select>
            <span className="steward-badge">🧑‍⚖️ Steward</span>
          </div>
        )}
      </header>
      <div className="dashboard-tabs">
        <button onClick={() => setActiveTab('overview')} className={activeTab === 'overview' ? 'active' : ''}>Overview</button>
        <button onClick={() => setActiveTab('requests')} className={activeTab === 'requests' ? 'active' : ''}>Access Requests</button>
        <button onClick={() => setActiveTab('claims')} className={activeTab === 'claims' ? 'active' : ''}>Direct Claims</button>
        <button onClick={() => setActiveTab('roles')} className={activeTab === 'roles' ? 'active' : ''}>Role Mappings</button>
        <button onClick={() => setActiveTab('simulations')} className={activeTab === 'simulations' ? 'active' : ''}>Simulations</button>
        <button onClick={() => setActiveTab('audit')} className={activeTab === 'audit' ? 'active' : ''}>Audit Trail</button>
        <button onClick={() => setActiveTab('lifecycle')} className={activeTab === 'lifecycle' ? 'active' : ''}>Claim Lifecycle</button>
        <button onClick={() => setActiveTab('indexing')} className={activeTab === 'indexing' ? 'active' : ''}>Index Monitor</button>
      </div>
      <main className="dashboard-content">
        {renderTabContent()}
      </main>
    </div>
  );
}