import React from 'react';
import { NotificationCenter } from '../../../components/Notifications/NotificationCenter';
import { useTenant } from '../../../contexts/TenantContext';
import { useAuth } from '../../../contexts/AuthContext';

export const NotificationCenterPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { user } = useAuth();

  if (!tenant || !datasource) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-xl font-bold text-gray-900 mb-2">Tenant and Datasource Required</h2>
          <p className="text-gray-600">Please select a tenant and datasource to view notifications.</p>
        </div>
      </div>
    );
  }

  if (!user?.id) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <h2 className="text-xl font-bold text-gray-900 mb-2">Authentication Required</h2>
          <p className="text-gray-600">Please log in to view your notifications.</p>
        </div>
      </div>
    );
  }

  return (
    <NotificationCenter
      tenant={tenant}
      datasource={datasource}
      userId={user.id}
    />
  );
};

// Legacy placeholder implementation below (not used)
const LegacyNotificationCenterPlaceholder: React.FC = () => {
  return (
    <div className="flex h-screen font-display bg-background-light dark:bg-background-dark text-text-light-primary dark:text-text-dark-primary">
      {/* SideNavBar */}
      <aside className="w-64 shrink-0 flex-col border-r border-border-light dark:border-border-dark bg-panel-light dark:bg-panel-dark p-4">
        <div className="flex flex-col gap-8">
          <h1 className="text-lg font-bold">Workflow Designer</h1>
          <nav className="flex flex-col gap-2">
            <a className="flex items-center gap-3 rounded-lg bg-primary/20 px-3 py-2 text-primary" href="#">
              <span className="material-symbols-outlined text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>
                notifications
              </span>
              <p className="text-sm font-medium">All Notifications</p>
            </a>
            <a
              className="flex items-center gap-3 rounded-lg px-3 py-2 text-text-light-secondary dark:text-text-dark-secondary hover:bg-gray-100 dark:hover:bg-gray-800"
              href="#"
            >
              <span className="material-symbols-outlined text-2xl">mark_email_unread</span>
              <p className="text-sm font-medium">Unread</p>
            </a>
          </nav>
          <div className="flex flex-col gap-4">
            <h2 className="px-3 text-xs font-semibold uppercase text-text-light-secondary dark:text-text-dark-secondary">
              Filter by Severity
            </h2>
            <div className="flex flex-col gap-2">
              <div className="flex h-8 shrink-0 cursor-pointer items-center justify-start gap-x-2 rounded-lg px-3 hover:bg-gray-100 dark:hover:bg-gray-800">
                <span className="material-symbols-outlined text-xl" style={{ color: '#D9534F' }}>
                  error
                </span>
                <p className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary">Critical</p>
              </div>
              <div className="flex h-8 shrink-0 cursor-pointer items-center justify-start gap-x-2 rounded-lg px-3 hover:bg-gray-100 dark:hover:bg-gray-800">
                <span className="material-symbols-outlined text-xl" style={{ color: '#F0AD4E' }}>
                  warning
                </span>
                <p className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary">Warning</p>
              </div>
              <div className="flex h-8 shrink-0 cursor-pointer items-center justify-start gap-x-2 rounded-lg px-3 hover:bg-gray-100 dark:hover:bg-gray-800">
                <span className="material-symbols-outlined text-xl" style={{ color: '#5BC0DE' }}>
                  info
                </span>
                <p className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary">Info</p>
              </div>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto p-6 lg:p-8">
        <div className="mx-auto max-w-4xl">
          {/* PageHeading */}
          <header className="mb-8 flex flex-wrap items-center justify-between gap-4">
            <h1 className="text-4xl font-black tracking-tighter">Notification Center</h1>
            <button className="flex h-10 min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-primary px-4 text-sm font-bold text-white transition-colors hover:bg-primary/90">
              <span>Mark All as Read</span>
            </button>
          </header>

          {/* Notification List */}
          <div className="flex flex-col gap-6">
            {/* SectionHeader: Today */}
            <h3 className="text-lg font-bold tracking-tight">Today</h3>

            {/* ListItem: Critical */}
            <div className="flex gap-4 rounded-xl border border-transparent bg-panel-light dark:bg-panel-dark p-4 transition-all hover:border-border-light dark:hover:border-border-dark hover:bg-white/80 dark:hover:bg-panel-dark/80">
              <div className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: '#D9534F' }}></div>
              <div className="flex flex-1 flex-col gap-1">
                <div className="flex items-center justify-between">
                  <p className="font-medium">Workflow 'Q4 Review' Failed</p>
                  <p className="text-xs text-text-light-secondary dark:text-text-dark-secondary">2 minutes ago</p>
                </div>
                <p className="text-sm text-text-light-secondary dark:text-text-dark-secondary">
                  Step 'Data Validation' returned a critical error. Immediate action required.
                </p>
                <div className="mt-2 flex items-center gap-4">
                  <button className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-primary">
                    Mark as Read
                  </button>
                  <button className="text-sm font-medium text-primary hover:underline">Open Workflow</button>
                </div>
              </div>
            </div>

            {/* ListItem: Warning */}
            <div className="flex gap-4 rounded-xl border border-transparent bg-panel-light dark:bg-panel-dark p-4 transition-all hover:border-border-light dark:hover:border-border-dark hover:bg-white/80 dark:hover:bg-panel-dark/80">
              <div className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: '#F0AD4E' }}></div>
              <div className="flex flex-1 flex-col gap-1">
                <div className="flex items-center justify-between">
                  <p className="font-medium">High Memory Usage Detected</p>
                  <p className="text-xs text-text-light-secondary dark:text-text-dark-secondary">15 minutes ago</p>
                </div>
                <p className="text-sm text-text-light-secondary dark:text-text-dark-secondary">
                  The 'Image Processing' workflow is using 90% of allocated memory.
                </p>
                <div className="mt-2 flex items-center gap-4">
                  <button className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-primary">
                    Mark as Read
                  </button>
                  <button className="text-sm font-medium text-primary hover:underline">Open Workflow</button>
                </div>
              </div>
            </div>

            {/* ListItem: Info */}
            <div className="flex gap-4 rounded-xl border border-transparent bg-panel-light dark:bg-panel-dark p-4 transition-all hover:border-border-light dark:hover:border-border-dark hover:bg-white/80 dark:hover:bg-panel-dark/80">
              <div className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: '#5BC0DE' }}></div>
              <div className="flex flex-1 flex-col gap-1">
                <div className="flex items-center justify-between">
                  <p className="font-medium">New Task Assigned to You</p>
                  <p className="text-xs text-text-light-secondary dark:text-text-dark-secondary">1 hour ago</p>
                </div>
                <p className="text-sm text-text-light-secondary dark:text-text-dark-secondary">
                  You have been assigned to 'Review User Feedback' in the 'App V2 Launch' project.
                </p>
                <div className="mt-2 flex items-center gap-4">
                  <button className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-primary">
                    Mark as Read
                  </button>
                  <button className="text-sm font-medium text-primary hover:underline">Open Task</button>
                </div>
              </div>
            </div>

            {/* SectionHeader: Yesterday */}
            <h3 className="pt-4 text-lg font-bold tracking-tight">Yesterday</h3>

            {/* ListItem: Info */}
            <div className="flex gap-4 rounded-xl border border-transparent bg-panel-light dark:bg-panel-dark p-4 opacity-60 transition-all hover:border-border-light dark:hover:border-border-dark hover:bg-white/80 dark:hover:bg-panel-dark/80 hover:opacity-100">
              <div className="mt-1 h-3 w-3 shrink-0 rounded-full" style={{ backgroundColor: '#5BC0DE' }}></div>
              <div className="flex flex-1 flex-col gap-1">
                <div className="flex items-center justify-between">
                  <p className="font-medium">Workflow 'Daily Backup' Completed</p>
                  <p className="text-xs text-text-light-secondary dark:text-text-dark-secondary">Yesterday at 11:30 PM</p>
                </div>
                <p className="text-sm text-text-light-secondary dark:text-text-dark-secondary">
                  The daily database backup workflow ran successfully.
                </p>
                <div className="mt-2 flex items-center gap-4">
                  <button className="text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-primary">
                    Mark as Unread
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};
