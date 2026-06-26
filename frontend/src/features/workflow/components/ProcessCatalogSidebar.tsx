import React from 'react';

interface Process {
  id: string;
  name: string;
  version: string;
  status: 'Published' | 'Draft' | 'Archived';
  icon: string;
}

const mockProcesses: Process[] = [
  { id: '1', name: 'Order Lifecycle', version: '1.0', status: 'Draft', icon: 'account_tree' },
  { id: '2', name: 'Invoice Approval', version: '1.7', status: 'Draft', icon: 'request_quote' },
  { id: '3', name: 'Expense Report', version: '3.2', status: 'Published', icon: 'card_travel' },
  { id: '4', name: 'Procurement Request', version: '1.0', status: 'Archived', icon: 'local_shipping' },
];

interface ProcessCatalogSidebarProps {
  selectedProcessId: string | null;
  onSelectProcess: (id: string) => void;
}

export const ProcessCatalogSidebar: React.FC<ProcessCatalogSidebarProps> = ({ selectedProcessId, onSelectProcess }) => {
  return (
    <aside className="flex w-80 shrink-0 flex-col border-r border-border-light dark:border-border-dark bg-panel-light dark:bg-panel-dark">
      {/* PageHeading and SearchBar */}
      <div className="p-4 border-b border-border-light dark:border-border-dark">
        <p className="text-2xl font-bold tracking-tight">Process Catalog</p>
        <div className="mt-4">
          <label className="flex flex-col min-w-40 h-10 w-full">
            <div className="flex w-full flex-1 items-stretch rounded-lg h-full">
              <div className="text-text-light-secondary dark:text-text-dark-secondary flex bg-background-light dark:bg-background-dark items-center justify-center pl-3 rounded-l-lg border-r-0">
                <span className="material-symbols-outlined text-xl">search</span>
              </div>
              <input
                className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-sm focus:outline-0 focus:ring-0 border-none bg-background-light dark:bg-background-dark h-full placeholder:text-text-light-secondary placeholder:dark:text-text-dark-secondary px-4 rounded-l-none border-l-0 pl-2"
                placeholder="Find a process..."
              />
            </div>
          </label>
        </div>
      </div>
      {/* List Items */}
      <nav className="flex-1 overflow-y-auto p-2 space-y-1">
        {mockProcesses.map((process) => (
          <div
            key={process.id}
            onClick={() => onSelectProcess(process.id)}
            className={`flex items-center gap-4 px-4 min-h-[72px] py-2 justify-between rounded-lg cursor-pointer ${
              selectedProcessId === process.id
                ? 'bg-primary/20'
                : 'hover:bg-background-light dark:hover:bg-background-dark'
            }`}
          >
            <div className="flex items-center gap-4">
              <div
                className={`flex items-center justify-center rounded-lg shrink-0 size-12 ${
                  selectedProcessId === process.id
                    ? 'text-primary bg-primary/30'
                    : 'text-text-light-secondary dark:text-text-dark-secondary bg-background-light dark:bg-background-dark'
                }`}
              >
                <span className="material-symbols-outlined text-3xl">{process.icon}</span>
              </div>
              <div className="flex flex-col justify-center">
                <p
                  className={`text-base font-medium ${
                    selectedProcessId === process.id
                      ? 'text-text-light-primary dark:text-text-dark-primary font-semibold'
                      : ''
                  }`}
                >
                  {process.name}
                </p>
                <p className="text-sm text-text-light-secondary dark:text-text-dark-secondary">
                  Version {process.version} - {process.status}
                </p>
              </div>
            </div>
          </div>
        ))}
      </nav>
      {/* Tabs for selected process */}
      {selectedProcessId && (
        <div className="border-t border-border-light dark:border-border-dark p-4">
          <div className="flex border-b border-border-light dark:border-border-dark">
            <button className="px-4 py-2 text-sm font-medium border-b-2 border-primary text-primary">
              Version History
            </button>
            <button className="px-4 py-2 text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-text-light-primary dark:hover:text-text-dark-primary">
              Change Log
            </button>
            <button className="px-4 py-2 text-sm font-medium text-text-light-secondary dark:text-text-dark-secondary hover:text-text-light-primary dark:hover:text-text-dark-primary">
              Impact Analysis
            </button>
          </div>
          <div className="pt-4 space-y-3">
            <div className="flex justify-between text-sm">
              <p className="font-medium">
                Version 1.0{' '}
                <span className="ml-2 rounded-full bg-blue-500/20 px-2 py-0.5 text-xs font-semibold text-blue-700 dark:text-blue-400">
                  Draft
                </span>
              </p>
              <p className="text-text-light-secondary dark:text-text-dark-secondary">3 minutes ago</p>
            </div>
          </div>
        </div>
      )}
    </aside>
  );
};
