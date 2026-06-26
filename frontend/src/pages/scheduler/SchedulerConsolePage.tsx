/**
 * Scheduler Console - Main Layout Page with Actor-Aware Routing
 * Provides tabbed navigation with different views for Tenant Ops vs Global Ops
 */

import React, { useState, useEffect } from 'react';
import { 
  Calendar, 
  GitBranch, 
  PlayCircle, 
  BarChart3, 
  Sparkles,
  AlertTriangle,
  Clock,
  Zap,
  Shield,
  Users,
  Settings
} from 'lucide-react';
import { useTenantContext } from '../../hooks/useTenantContext';
import { useSchedulerStats, useAISuggestions } from '../../api/schedulerApi';
import { ActorProvider, useActor, ActorSwitcher, ForActor } from '../../contexts/ActorContext';

// Views
import TenantOpsOverview from './TenantOpsOverview';
import GlobalOpsOverview from './GlobalOpsOverview';
import JobsView from './JobsView';
import DAGsView from './DAGsView';
import RunsExceptionsView from './RunsExceptionsView';
import ExceptionClusterView from './ExceptionClusterView';
import AISuggestionsPanel from './AISuggestionsPanel';
import ChangeSetModal from './ChangeSetModal';

import './SchedulerConsole.css';

type TabId = 'overview' | 'jobs' | 'dags' | 'runs' | 'calendars' | 'ai' | 'governance';

interface Tab {
  id: TabId;
  label: string;
  icon: React.ElementType;
  badge?: number;
  globalOpsOnly?: boolean;
}

const SchedulerConsoleContent: React.FC = () => {
  const [activeTab, setActiveTab] = useState<TabId>('overview');
  const [changeSetModalOpen, setChangeSetModalOpen] = useState(false);
  const [selectedChangeSet, setSelectedChangeSet] = useState<any>(null);
  
  const { selectedTenant } = useTenantContext();
  const tenantId = selectedTenant?.id;
  const { role, permissions, tenantName } = useActor();
  
  const { stats, loading: statsLoading } = useSchedulerStats(tenantId || '');
  const { suggestions } = useAISuggestions(tenantId || '');

  // Build tabs based on actor role
  const tabs: Tab[] = [
    { id: 'overview', label: 'Overview', icon: BarChart3 },
    { id: 'jobs', label: 'Jobs', icon: Calendar, badge: stats?.active_jobs },
    { id: 'dags', label: 'DAGs', icon: GitBranch },
    { id: 'runs', label: 'Runs & Exceptions', icon: PlayCircle, badge: stats?.running_jobs },
    { id: 'calendars', label: 'Calendars', icon: Clock },
    { id: 'ai', label: 'AI Suggestions', icon: Sparkles, badge: suggestions?.length },
    { id: 'governance', label: 'Governance', icon: Shield }
  ];

  const renderContent = () => {
    switch (activeTab) {
      case 'overview':
        // Render different overview based on actor role
        return role === 'GLOBAL_OPS' ? <GlobalOpsOverview /> : <TenantOpsOverview />;
      case 'jobs':
        return <JobsView />;
      case 'dags':
        return <DAGsView />;
      case 'runs':
        return (
          <div className="runs-with-clusters">
            <RunsExceptionsView />
            <ExceptionClusterView 
              onApplyFix={(clusterId) => {
                // Would open ChangeSet modal
                setSelectedChangeSet({ id: clusterId, type: 'exception_fix' });
                setChangeSetModalOpen(true);
              }}
            />
          </div>
        );
      case 'calendars':
        return <CalendarsView />;
      case 'ai':
        return (
          <AISuggestionsPanel 
            maxItems={20}
            showAllLink={false}
            onApplyFix={(suggestionId) => {
              setSelectedChangeSet({ id: suggestionId, type: 'ai_suggestion' });
              setChangeSetModalOpen(true);
            }}
          />
        );
      case 'governance':
        return <GovernanceView onOpenChangeSet={(cs) => {
          setSelectedChangeSet(cs);
          setChangeSetModalOpen(true);
        }} />;
      default:
        return role === 'GLOBAL_OPS' ? <GlobalOpsOverview /> : <TenantOpsOverview />;
    }
  };

  return (
    <div className="scheduler-console">
      <header className="scheduler-header">
        <div className="header-content">
          <div className="header-title">
            <Clock className="header-icon" />
            <div>
              <h1>Scheduler Intelligence Console</h1>
              <p className="subtitle">
                {role === 'GLOBAL_OPS' 
                  ? 'Global Operations — Cross-Tenant Management'
                  : `Tenant Operations — ${tenantName || 'Single Tenant'}`}
              </p>
            </div>
          </div>
          
          {/* Actor Switcher (for dev/testing) */}
          <div className="header-controls">
            <ActorSwitcher />
          </div>
          
          {stats && !statsLoading && (
            <div className="header-stats">
              <div className="stat-item">
                <span className="stat-value">
                  {role === 'GLOBAL_OPS' ? stats.total_jobs?.toLocaleString() : stats.active_jobs}
                </span>
                <span className="stat-label">
                  {role === 'GLOBAL_OPS' ? 'Total Jobs' : 'Active Jobs'}
                </span>
              </div>
              <div className="stat-item">
                <Zap className="stat-icon running" />
                <span className="stat-value">{stats.running_jobs}</span>
                <span className="stat-label">Running</span>
              </div>
              <div className="stat-item">
                <AlertTriangle className="stat-icon warning" />
                <span className="stat-value">{stats.failed_last_24h}</span>
                <span className="stat-label">Failed (24h)</span>
              </div>
              <ForActor role="GLOBAL_OPS">
                <div className="stat-item">
                  <Users className="stat-icon tenants" />
                  <span className="stat-value">{stats.active_tenants || 24}</span>
                  <span className="stat-label">Tenants</span>
                </div>
              </ForActor>
            </div>
          )}
        </div>
      </header>

      <nav className="scheduler-tabs">
        {tabs.map((tab) => {
          // Skip global-ops-only tabs for tenant ops
          if (tab.globalOpsOnly && role !== 'GLOBAL_OPS') {
            return null;
          }
          
          const Icon = tab.icon;
          return (
            <button
              key={tab.id}
              className={`tab-button ${activeTab === tab.id ? 'active' : ''}`}
              onClick={() => setActiveTab(tab.id)}
            >
              <Icon className="tab-icon" />
              <span className="tab-label">{tab.label}</span>
              {tab.badge !== undefined && tab.badge > 0 && (
                <span className="tab-badge">{tab.badge}</span>
              )}
            </button>
          );
        })}
      </nav>

      <main className="scheduler-content">
        {renderContent()}
      </main>

      {/* ChangeSet Modal */}
      <ChangeSetModal
        isOpen={changeSetModalOpen}
        onClose={() => setChangeSetModalOpen(false)}
        changeSet={selectedChangeSet}
        mode={permissions.canApproveChangeSets ? 'approve' : 'view'}
        onApprove={(id, comment) => {
          console.log('Approved:', id, comment);
          setChangeSetModalOpen(false);
        }}
        onReject={(id, reason) => {
          console.log('Rejected:', id, reason);
          setChangeSetModalOpen(false);
        }}
      />
    </div>
  );
};

// Calendars placeholder view
const CalendarsView: React.FC = () => {
  const { role, permissions } = useActor();
  
  return (
    <div className="calendars-view">
      <h2>
        {permissions.canViewGlobalCalendars ? 'Calendar Hierarchy' : 'Tenant Calendar'}
      </h2>
      
      <ForActor role="GLOBAL_OPS">
        <div className="calendar-hierarchy">
          <div className="calendar-level">
            <h3>Global Calendars</h3>
            <p>Base calendars applied across all tenants</p>
          </div>
          <div className="calendar-level">
            <h3>Regional Calendars</h3>
            <p>Region-specific holidays and blackouts</p>
          </div>
          <div className="calendar-level">
            <h3>Market Calendars</h3>
            <p>Market-specific trading days</p>
          </div>
        </div>
      </ForActor>
      
      <div className="calendar-cards">
        <div className="calendar-card">
          <h4>EU Market Calendar</h4>
          <p>Next holiday: Jan 20 — MLK Day (US)</p>
          <span className="linked-jobs">12 linked jobs</span>
        </div>
      </div>
    </div>
  );
};

// Governance placeholder view
const GovernanceView: React.FC<{ onOpenChangeSet: (cs: any) => void }> = ({ onOpenChangeSet }) => {
  const { permissions } = useActor();
  
  const changeSets = [
    { id: 'cs-001', title: 'Update Pre-Agg timeout', status: 'pending_review', riskScore: 0.3, createdAt: '2026-01-17T08:00:00Z' },
    { id: 'cs-002', title: 'Add new Data Quality job', status: 'pending_review', riskScore: 0.1, createdAt: '2026-01-17T07:30:00Z' },
    { id: 'cs-003', title: 'Restructure Risk DAG', status: 'approved', riskScore: 0.5, createdAt: '2026-01-17T06:00:00Z' },
  ];
  
  return (
    <div className="governance-view">
      <h2>Governance</h2>
      <p>Review and approve scheduler changes</p>
      
      <div className="changeset-list">
        {changeSets.map(cs => (
          <div 
            key={cs.id} 
            className={`changeset-row status-${cs.status}`}
            onClick={() => onOpenChangeSet(cs)}
          >
            <div className="cs-info">
              <span className="cs-title">{cs.title}</span>
              <span className="cs-meta">{new Date(cs.createdAt).toLocaleDateString()}</span>
            </div>
            <div className="cs-status">
              <span className={`status-badge ${cs.status}`}>{cs.status.replace('_', ' ')}</span>
              <span className="risk-score" style={{ 
                color: cs.riskScore > 0.4 ? '#ea580c' : '#16a34a' 
              }}>
                {Math.round(cs.riskScore * 100)}% risk
              </span>
            </div>
            {permissions.canApproveChangeSets && cs.status === 'pending_review' && (
              <button className="approve-quick">Review</button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
};

// Wrap with ActorProvider
const SchedulerConsolePage: React.FC = () => {
  return (
    <ActorProvider>
      <SchedulerConsoleContent />
    </ActorProvider>
  );
};

export default SchedulerConsolePage;
