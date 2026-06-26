import React, { useState, useEffect } from 'react';
import { delegationApi, Delegation } from '../../../api/delegationApi';
import { DelegationModal } from '../components/DelegationModal';
import { Plus, Trash2 } from 'lucide-react';

export const ApprovalInboxPage: React.FC = () => {
  const [selectedApprovalId, setSelectedApprovalId] = useState<string>('Expense-Report-Q3');
  const [activeTab, setActiveTab] = useState<'inbox' | 'delegations'>('inbox');
  const [delegations, setDelegations] = useState<Delegation[]>([]);
  const [isDelegationModalOpen, setIsDelegationModalOpen] = useState(false);

  // Mock approvals (keep existing)
  const approvals = [
    {
      id: 'Onboarding-Process-V2',
      stage: 'Manager Review',
      sla: 'Overdue: 2h 15m',
      slaStatus: 'overdue',
      risk: 'High',
      riskColor: 'red',
    },
    {
      id: 'Expense-Report-Q3',
      stage: 'Finance Approval',
      sla: 'Due in: 1d 8h',
      slaStatus: 'warning',
      risk: 'Medium',
      riskColor: 'orange',
    },
    {
      id: 'IT-Access-Request',
      stage: 'Security Sign-off',
      sla: 'Due in: 8d 4h',
      slaStatus: 'ok',
      risk: 'Low',
      riskColor: 'green',
    },
    {
      id: 'Marketing-Campaign-Launch',
      stage: 'Legal Review',
      sla: 'Due in: 18d 12h',
      slaStatus: 'ok',
      risk: 'Low',
      riskColor: 'green',
    },
  ];

  useEffect(() => {
    if (activeTab === 'delegations') {
      loadDelegations();
    }
  }, [activeTab]);

  const loadDelegations = async () => {
    try {
      const data = await delegationApi.getOutgoingDelegations();
      setDelegations(data);
    } catch (error) {
      console.error('Failed to load delegations', error);
    }
  };

  const handleRevokeDelegation = async (id: string) => {
    if (!confirm('Are you sure you want to revoke this delegation?')) return;
    try {
      await delegationApi.revokeDelegation(id);
      loadDelegations();
    } catch (error) {
      console.error('Failed to revoke', error);
    }
  };

  return (
    <div className="relative flex min-h-screen w-full font-display bg-background-light dark:bg-background-dark text-gray-900 dark:text-white">
      <DelegationModal 
        isOpen={isDelegationModalOpen} 
        onClose={() => setIsDelegationModalOpen(false)} 
        onSuccess={loadDelegations} 
      />
      
      {/* Sidebar */}
      <aside className="flex h-screen flex-col bg-background-light dark:bg-[#111418] border-r border-gray-200 dark:border-gray-800 w-64 p-4">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-3 p-2">
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuBl3oIfmJSw7AG4mXKHLfPlB8nnOT-vbhHn5CwCSANJPiNiKo9Hs8GrkvnZePylFZG8ZS6wCnL-8o7CVEu9ymxK7MG1V_AmTpKQkWcfameayyC3OyTc0-xZ_8g5P2BtRhflk__9sRgzFIzjduqpP9u-rVOm0ESpA285s1sRQPyAvh6kLU8Xk7gr0eJAelM_MSi4axBgUoWMFAT7OCR77s-Gv0i6cLBLz7OV59I2Wz8RQi0zuExQoIMMk3xzkOzleqqBUCzTdA-fxZhR")' }}></div>
            <div className="flex flex-col">
              <h1 className="text-gray-900 dark:text-white text-base font-medium leading-normal">Eleanor Vance</h1>
              <p className="text-gray-500 dark:text-[#9dabb9] text-sm font-normal leading-normal">Approver Role</p>
            </div>
          </div>
          <nav className="flex flex-col gap-2 mt-4">
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-600 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20" href="/core/regulator-portal">
              <span className="material-symbols-outlined">dashboard</span>
              <p className="text-sm font-medium leading-normal">Dashboard</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/20 text-primary dark:bg-primary/30 dark:text-primary" href="#">
              <span className="material-symbols-outlined" style={{ fontVariationSettings: "'FILL' 1" }}>inbox</span>
              <p className="text-sm font-medium leading-normal">Approval Inbox</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-600 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20" href="/core/process-catalog">
              <span className="material-symbols-outlined">account_tree</span>
              <p className="text-sm font-medium leading-normal">Workflows</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-600 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20" href="#">
              <span className="material-symbols-outlined">bar_chart</span>
              <p className="text-sm font-medium leading-normal">Reports</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-600 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20" href="#">
              <span className="material-symbols-outlined">settings</span>
              <p className="text-sm font-medium leading-normal">Settings</p>
            </a>
          </nav>
        </div>
        <div className="mt-auto">
          <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-600 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20" href="#">
            <span className="material-symbols-outlined">help</span>
            <p className="text-sm font-medium leading-normal">Help Center</p>
          </a>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col p-6 overflow-hidden">
        <header className="flex flex-wrap justify-between items-center gap-3 pb-4 border-b border-gray-200 dark:border-gray-800">
          <div>
            <h1 className="text-gray-900 dark:text-white text-4xl font-black leading-tight tracking-[-0.033em]">Approval Inbox</h1>
            <div className="flex gap-4 mt-4">
              <button 
                onClick={() => setActiveTab('inbox')}
                className={`pb-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'inbox' ? 'border-primary text-primary' : 'border-transparent text-gray-500 hover:text-gray-700'}`}
              >
                My Tasks
              </button>
              <button 
                onClick={() => setActiveTab('delegations')}
                className={`pb-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'delegations' ? 'border-primary text-primary' : 'border-transparent text-gray-500 hover:text-gray-700'}`}
              >
                Delegations
              </button>
            </div>
          </div>
          
          <div className="flex items-center gap-4">
            {activeTab === 'delegations' && (
              <button 
                onClick={() => setIsDelegationModalOpen(true)}
                className="flex items-center gap-2 px-3 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
              >
                <Plus className="w-4 h-4" />
                <span>New Delegation</span>
              </button>
            )}
            <div className="relative w-64">
              <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">search</span>
              <input className="w-full pl-10 pr-4 py-2 rounded-lg bg-gray-100 dark:bg-[#1c2127] text-gray-900 dark:text-white border-gray-200 dark:border-gray-700 focus:ring-primary focus:border-primary" placeholder="Search approvals..." type="text" />
            </div>
            <button className="p-2 text-gray-500 dark:text-white/70 hover:text-gray-900 dark:hover:text-white rounded-full hover:bg-gray-200 dark:hover:bg-primary/20">
              <span className="material-symbols-outlined">notifications</span>
            </button>
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuDw9tng2j3iZWBf4cgxe1nUsteQQGGdLPzGfPwFrWgrg9FpDI5tucSG7C84XNLau1HJMtrNacBm6AQBBoEPihhbdjv69_U7ZPZqQVs_katnqO6GrCoOyfRUKPRsr_-8GVwbpQU7v44b1Zy99zbMAtaBFnjUug26viCcDp8CDbFR0C461XxYmkHtJBedPBEmfR381TxcI42CxZDVSo-5hI5LUBEZ2MaybmRGcASWHKvTIIUtpFsHN5irfYhMHVHkobSCuMwYmOiovNxY")' }}></div>
          </div>
        </header>

        <div className="flex-1 flex gap-6 mt-6 overflow-hidden">
          {activeTab === 'inbox' ? (
            <>
              {/* List View */}
              <div className="flex-1 flex flex-col overflow-y-auto">
                <div className="flex justify-between items-center gap-2 px-1 py-3">
                  <div className="flex items-center gap-2">
                    <button className="flex items-center gap-2 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-background-light dark:bg-[#1c2127] text-gray-700 dark:text-white text-sm hover:bg-gray-100 dark:hover:bg-gray-800">
                      <span className="material-symbols-outlined text-base">filter_list</span>
                      Filter
                    </button>
                    <button className="flex items-center gap-2 px-3 py-2 rounded-lg border border-gray-300 dark:border-gray-700 bg-background-light dark:bg-[#1c2127] text-gray-700 dark:text-white text-sm hover:bg-gray-100 dark:hover:bg-gray-800">
                      <span className="material-symbols-outlined text-base">swap_vert</span>
                      Sort
                    </button>
                  </div>
                  <button className="flex items-center justify-center rounded-lg h-10 bg-primary text-white gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-4 hover:bg-primary/90">
                    <span className="material-symbols-outlined text-xl">checklist</span>
                    <span className="truncate">Bulk Actions</span>
                  </button>
                </div>
                <div className="flex-1 overflow-hidden rounded-lg border border-gray-200 dark:border-[#3b4754]">
                  <div className="overflow-x-auto">
                    <table className="w-full text-left">
                      <thead className="bg-gray-50 dark:bg-[#1c2127]">
                        <tr>
                          <th className="px-4 py-3 text-gray-600 dark:text-white w-2/12 text-sm font-medium">Process ID</th>
                          <th className="px-4 py-3 text-gray-600 dark:text-white w-2/12 text-sm font-medium">Stage</th>
                          <th className="px-4 py-3 text-gray-600 dark:text-white w-2/12 text-sm font-medium">SLA Countdown</th>
                          <th className="px-4 py-3 text-gray-600 dark:text-white w-1/12 text-sm font-medium">Risk</th>
                          <th className="px-4 py-3 text-gray-600 dark:text-white w-5/12 text-sm font-medium">Actions</th>
                        </tr>
                      </thead>
                      <tbody className="divide-y divide-gray-200 dark:divide-[#3b4754]">
                        {approvals.map((approval) => (
                          <tr 
                            key={approval.id} 
                            className={`cursor-pointer ${selectedApprovalId === approval.id ? 'bg-blue-50 dark:bg-blue-900/20 border-l-4 border-primary' : 'bg-white dark:bg-background-dark/50 hover:bg-gray-50 dark:hover:bg-gray-900/50 border-l-4 border-transparent'}`}
                            onClick={() => setSelectedApprovalId(approval.id)}
                          >
                            <td className="px-4 py-3 text-gray-900 dark:text-white text-sm font-medium">{approval.id}</td>
                            <td className="px-4 py-3 text-gray-500 dark:text-[#9dabb9] text-sm">{approval.stage}</td>
                            <td className="px-4 py-3 text-sm">
                              <div className="flex items-center gap-2">
                                {approval.slaStatus === 'overdue' && <span className="material-symbols-outlined text-red-500 text-lg" style={{ fontVariationSettings: "'FILL' 1" }}>error</span>}
                                {approval.slaStatus === 'warning' && <span className="material-symbols-outlined text-orange-500 text-lg">hourglass_top</span>}
                                {approval.slaStatus === 'ok' && <span className="material-symbols-outlined text-green-500 text-lg">timer</span>}
                                <span className={`font-semibold ${approval.slaStatus === 'overdue' ? 'text-red-600 dark:text-red-400' : approval.slaStatus === 'warning' ? 'text-orange-600 dark:text-orange-400' : 'text-green-600 dark:text-green-400'}`}>{approval.sla}</span>
                              </div>
                            </td>
                            <td className="px-4 py-3 text-sm">
                              <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold bg-${approval.riskColor}-100 text-${approval.riskColor}-800 dark:bg-${approval.riskColor}-900/50 dark:text-${approval.riskColor}-300`}>{approval.risk}</span>
                            </td>
                            <td className="px-4 py-3 text-sm">
                              <div className="flex items-center gap-2">
                                <button className="px-3 py-1 text-sm font-semibold text-white bg-green-600 rounded-lg hover:bg-green-700">Approve</button>
                                <button className="px-3 py-1 text-sm font-semibold text-white bg-red-600 rounded-lg hover:bg-red-700">Reject</button>
                                <button className="px-3 py-1 text-sm font-semibold text-gray-700 dark:text-gray-200 bg-gray-200 dark:bg-gray-700 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600">Delegate</button>
                              </div>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
                {/* Pagination */}
                <div className="flex items-center justify-center pt-4">
                  <nav className="flex items-center gap-1">
                    <a className="flex size-10 items-center justify-center text-gray-500 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20 rounded-lg" href="#">
                      <span className="material-symbols-outlined text-xl">chevron_left</span>
                    </a>
                    <a className="text-sm font-bold flex size-10 items-center justify-center text-white bg-primary rounded-lg" href="#">1</a>
                    <a className="text-sm font-normal flex size-10 items-center justify-center text-gray-600 dark:text-white/80 rounded-lg hover:bg-gray-200 dark:hover:bg-primary/20" href="#">2</a>
                    <a className="text-sm font-normal flex size-10 items-center justify-center text-gray-600 dark:text-white/80 rounded-lg hover:bg-gray-200 dark:hover:bg-primary/20" href="#">3</a>
                    <span className="text-sm font-normal flex size-10 items-center justify-center text-gray-600 dark:text-white/80 rounded-lg">...</span>
                    <a className="text-sm font-normal flex size-10 items-center justify-center text-gray-600 dark:text-white/80 rounded-lg hover:bg-gray-200 dark:hover:bg-primary/20" href="#">10</a>
                    <a className="flex size-10 items-center justify-center text-gray-500 dark:text-white/70 hover:bg-gray-200 dark:hover:bg-primary/20 rounded-lg" href="#">
                      <span className="material-symbols-outlined text-xl">chevron_right</span>
                    </a>
                  </nav>
                </div>
              </div>

              {/* Detail View */}
              <aside className="w-[45%] flex-shrink-0 bg-background-light dark:bg-[#111418] border border-gray-200 dark:border-gray-800 rounded-lg p-6 flex flex-col gap-6 overflow-y-auto">
                <div className="flex justify-between items-start">
                  <div>
                    <h2 className="text-xl font-bold text-gray-900 dark:text-white">{selectedApprovalId}</h2>
                    <p className="text-sm text-gray-500 dark:text-gray-400">Version 1.3 - Finance Approval Stage</p>
                  </div>
                  <button className="p-2 text-gray-500 dark:text-white/70 hover:text-gray-900 dark:hover:text-white rounded-full hover:bg-gray-200 dark:hover:bg-primary/20 -mt-2 -mr-2">
                    <span className="material-symbols-outlined">close</span>
                  </button>
                </div>
                
                <div className="flex-grow flex flex-col gap-6 pr-2 -mr-2">
                  <div className="bg-gray-100 dark:bg-[#1c2127] p-4 rounded-lg">
                    <h3 className="text-base font-bold text-gray-800 dark:text-white mb-3">Decision Rationale</h3>
                    <textarea className="w-full rounded-md bg-white dark:bg-[#111418] border-gray-300 dark:border-gray-600 text-sm text-gray-800 dark:text-gray-300 focus:ring-primary focus:border-primary" placeholder="Enter rationale for your decision (required)..." rows={3}></textarea>
                    <div className="flex items-center gap-2 mt-3">
                      <button className="flex-1 px-4 py-2 text-sm font-semibold text-white bg-green-600 rounded-lg hover:bg-green-700">Approve</button>
                      <button className="flex-1 px-4 py-2 text-sm font-semibold text-white bg-red-600 rounded-lg hover:bg-red-700">Reject</button>
                      <button className="flex-1 px-4 py-2 text-sm font-semibold text-gray-700 dark:text-gray-200 bg-gray-200 dark:bg-gray-700 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600">Delegate</button>
                    </div>
                  </div>

                  <div className="border border-gray-200 dark:border-gray-700 rounded-lg p-4">
                    <h3 className="text-base font-bold text-gray-800 dark:text-white mb-3">Audit Linkage</h3>
                    <div className="space-y-2 text-sm">
                      <div className="flex justify-between items-center">
                        <span className="text-gray-500 dark:text-gray-400">Hash:</span>
                        <code className="text-gray-700 dark:text-gray-300 font-mono text-xs">0x4a2e...f8b1</code>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-gray-500 dark:text-gray-400">Signature:</span>
                        <code className="text-gray-700 dark:text-gray-300 font-mono text-xs">SIG-a3c4...99e2</code>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-gray-500 dark:text-gray-400">Policy Ref:</span>
                        <code className="text-gray-700 dark:text-gray-300 font-mono text-xs">POL_FIN_002</code>
                      </div>
                    </div>
                  </div>

                  <div>
                    <div className="border-b border-gray-200 dark:border-gray-700">
                      <nav aria-label="Tabs" className="-mb-px flex space-x-6">
                        <a className="whitespace-nowrap py-3 px-1 border-b-2 font-semibold text-sm border-primary text-primary" href="#">Diffs</a>
                        <a className="whitespace-nowrap py-3 px-1 border-b-2 font-semibold text-sm border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:border-gray-500" href="#">Impact Analysis</a>
                        <a className="whitespace-nowrap py-3 px-1 border-b-2 font-semibold text-sm border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:border-gray-500" href="#">Migration Plan</a>
                      </nav>
                    </div>
                    <div className="py-4 text-sm text-gray-700 dark:text-gray-300">
                      <div className="bg-gray-100 dark:bg-[#1c2127] rounded-lg p-3 font-mono text-xs overflow-x-auto">
                        <pre><code className="language-diff">{`- "approver": "Finance-L1"
+ "approver": "Finance-L2"
- "threshold": 5000
+ "threshold": 7500
- "notify_dl": "finance_alerts@corp.com"
+ "notify_dl": "senior_finance_alerts@corp.com"`}</code></pre>
                      </div>
                    </div>
                  </div>
                </div>
              </aside>
            </>
          ) : (
            /* Delegations View */
            <div className="flex-1 flex flex-col gap-6">
              <div className="bg-white dark:bg-[#1c2127] rounded-lg border border-gray-200 dark:border-[#3b4754] p-6">
                <h2 className="text-lg font-bold mb-4">Active Delegations</h2>
                {delegations.length === 0 ? (
                  <p className="text-gray-500">No active delegations.</p>
                ) : (
                  <table className="w-full text-left">
                    <thead className="bg-gray-50 dark:bg-[#111418]">
                      <tr>
                        <th className="px-4 py-3 text-sm font-medium">To User</th>
                        <th className="px-4 py-3 text-sm font-medium">Period</th>
                        <th className="px-4 py-3 text-sm font-medium">Reason</th>
                        <th className="px-4 py-3 text-sm font-medium">Status</th>
                        <th className="px-4 py-3 text-sm font-medium">Actions</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-gray-200 dark:divide-[#3b4754]">
                      {delegations.map(d => (
                        <tr key={d.id}>
                          <td className="px-4 py-3 text-sm">{d.to_user_id}</td>
                          <td className="px-4 py-3 text-sm">{d.start_date} - {d.end_date}</td>
                          <td className="px-4 py-3 text-sm">{d.reason}</td>
                          <td className="px-4 py-3 text-sm">{d.status}</td>
                          <td className="px-4 py-3 text-sm">
                            <button 
                              onClick={() => handleRevokeDelegation(d.id)}
                              className="text-red-600 hover:text-red-800"
                            >
                              <Trash2 className="w-4 h-4" />
                            </button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </div>
            </div>
          )}
        </div>
      </main>
    </div>
  );
};
