import React from 'react';

export const RegulatorDashboardPage: React.FC = () => {
  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-background-dark text-gray-800 dark:text-gray-200">
      {/* SideNavBar */}
      <aside className="flex h-full w-64 flex-col justify-between border-r border-gray-200/10 dark:border-white/10 bg-white/5 dark:bg-black/10 p-4 sticky top-0">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-3 p-2">
            <div className="text-primary size-8">
              <svg fill="none" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
                <path d="M4 4H17.3334V17.3334H30.6666V30.6666H44V44H4V4Z" fill="currentColor"></path>
              </svg>
            </div>
            <div className="flex flex-col">
              <h1 className="text-lg font-bold text-gray-900 dark:text-white">Regulator Portal</h1>
              <p className="text-sm text-gray-500 dark:text-gray-400">Temporal UI</p>
            </div>
          </div>
          <nav className="flex flex-col gap-2 mt-4">
            <a className="flex items-center gap-3 rounded-lg bg-primary/10 px-3 py-2 text-primary dark:text-primary" href="#">
              <span className="material-symbols-outlined text-lg">dashboard</span>
              <p className="text-sm font-semibold">Dashboard</p>
            </a>
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/5" href="/core/process-catalog">
              <span className="material-symbols-outlined text-lg">toc</span>
              <p className="text-sm font-medium">Process Catalog</p>
            </a>
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/5" href="/core/approval-inbox">
              <span className="material-symbols-outlined text-lg">rule</span>
              <p className="text-sm font-medium">Approval Explorer</p>
            </a>
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/5" href="/core/audit-explorer">
              <span className="material-symbols-outlined text-lg">shield</span>
              <p className="text-sm font-medium">Audit Explorer</p>
            </a>
            <a className="flex items-center gap-3 rounded-lg px-3 py-2 text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-white/5" href="/core/notifications">
              <span className="material-symbols-outlined text-lg">notifications</span>
              <p className="text-sm font-medium">Notification Center</p>
            </a>
          </nav>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 overflow-y-auto">
        {/* TopNavBar */}
        <header className="flex items-center justify-between whitespace-nowrap border-b border-solid border-gray-200/10 dark:border-white/10 px-10 py-3 sticky top-0 bg-background-light/80 dark:bg-background-dark/80 backdrop-blur-sm z-10">
          <div className="flex items-center gap-8">
            <div className="flex items-center gap-2">
              <p className="text-sm font-medium text-gray-500 dark:text-gray-400">Tenant:</p>
              <select className="form-select w-full rounded-lg border-gray-200/50 bg-white/5 dark:border-white/10 dark:bg-black/10 text-sm font-medium text-gray-800 dark:text-gray-200 focus:border-primary focus:ring-primary">
                <option>Acmecorp</option>
                <option>Initech</option>
                <option>Globex</option>
              </select>
            </div>
          </div>
          <div className="flex flex-1 items-center justify-end gap-4">
            <label className="relative flex h-10 w-full max-w-sm">
              <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-gray-400 dark:text-gray-500">
                <span className="material-symbols-outlined text-xl">search</span>
              </div>
              <input className="form-input h-full w-full rounded-lg border-gray-200/50 bg-white/5 pl-10 text-sm placeholder:text-gray-500 focus:border-primary focus:ring-primary dark:border-white/10 dark:bg-black/10 dark:text-gray-200" placeholder="Search processes, approvals, audits..." />
            </label>
            <div className="flex gap-2">
              <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-white/5 text-gray-600 hover:bg-gray-100 dark:bg-black/10 dark:text-gray-300 dark:hover:bg-white/5">
                <span className="material-symbols-outlined text-xl">notifications</span>
              </button>
              <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-white/5 text-gray-600 hover:bg-gray-100 dark:bg-black/10 dark:text-gray-300 dark:hover:bg-white/5">
                <span className="material-symbols-outlined text-xl">help_outline</span>
              </button>
            </div>
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuDD-R0D0A2swlCcw4P5uz1IoeX1vsVfxX92p78TZGWzW38SSmddvWxU4Jga-yxY_iXVnoRgmaMha7ToaHBvU-f803K18dvQ3Nu3Y_W4Gv_1_ioGA0_y_3YFe_yyuY9p6ysRJ5OYk46jXDbIpG5SfYtdRGorZd5t2cPXfLbBcbfE8x9raBTltC19WkBZuE91-o5JYrcCtxhtss1tK_-YLr4pzRq3J8RiSUOupSoDUwE7ua7nSxPY18pwQENI4RkhTnOdX1si0Mvl4tQ7")' }}></div>
          </div>
        </header>

        <div className="p-10">
          {/* PageHeading */}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="min-w-72 text-4xl font-black tracking-[-0.033em] text-gray-900 dark:text-white">Regulator Dashboard</p>
            <div className="flex gap-3">
              <button className="flex h-10 cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-white/5 px-4 text-sm font-bold text-gray-600 ring-1 ring-inset ring-gray-200/50 hover:bg-gray-100 dark:bg-black/10 dark:text-gray-300 dark:ring-white/10 dark:hover:bg-white/5">
                <span className="material-symbols-outlined text-lg">download</span>
                Download Report
              </button>
              <button className="flex h-10 cursor-pointer items-center justify-center gap-2 overflow-hidden rounded-lg bg-primary px-4 text-sm font-bold text-white hover:bg-primary/90">
                <span className="material-symbols-outlined text-lg">verified_user</span>
                Initiate Chain Validation
              </button>
            </div>
          </div>

          {/* Stats */}
          <div className="grid grid-cols-1 gap-6 py-6 md:grid-cols-2 xl:grid-cols-4">
            <div className="flex flex-1 flex-col gap-2 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
              <p className="text-base font-medium text-gray-600 dark:text-gray-300">Active Processes</p>
              <p className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">1,423</p>
              <p className="text-sm font-medium text-green-500">+2.5% vs last month</p>
            </div>
            <div className="flex flex-1 flex-col gap-2 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
              <p className="text-base font-medium text-gray-600 dark:text-gray-300">Pending Approvals</p>
              <p className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">87</p>
              <p className="text-sm font-medium text-green-500">+5.1% vs last month</p>
            </div>
            <div className="flex flex-1 flex-col gap-2 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
              <p className="text-base font-medium text-gray-600 dark:text-gray-300">Compliance Violations</p>
              <p className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">12</p>
              <p className="text-sm font-medium text-red-500">-1.2% vs last month</p>
            </div>
            <div className="flex flex-1 flex-col gap-2 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
              <p className="text-base font-medium text-gray-600 dark:text-gray-300">Audit Chain Integrity</p>
              <p className="text-3xl font-bold tracking-tight text-gray-900 dark:text-white">99.98%</p>
              <p className="text-sm font-medium text-green-500">+0.01% vs last month</p>
            </div>
          </div>

          {/* Chart */}
          <div className="flex min-w-72 flex-1 flex-col gap-4 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <p className="text-lg font-bold text-gray-900 dark:text-white">Compliance Status</p>
                <p className="text-sm text-gray-500 dark:text-gray-400">Activity over the last 30 days</p>
              </div>
              <div className="flex items-center gap-1">
                <p className="text-sm font-medium text-gray-800 dark:text-gray-200">Overall: <span className="font-bold text-green-500">98.7%</span></p>
              </div>
            </div>
            <div className="grid min-h-[220px] grid-flow-col items-end gap-6 pt-4">
              <div className="relative flex h-full flex-col-reverse gap-2">
                <p className="text-center text-xs font-medium text-gray-500 dark:text-gray-400">1-7</p>
                <div className="h-[75%] w-full rounded-t-lg bg-primary/20 hover:bg-primary/40 dark:bg-primary/30 dark:hover:bg-primary/50"></div>
              </div>
              <div className="relative flex h-full flex-col-reverse gap-2">
                <p className="text-center text-xs font-medium text-gray-500 dark:text-gray-400">8-14</p>
                <div className="h-[80%] w-full rounded-t-lg bg-primary/20 hover:bg-primary/40 dark:bg-primary/30 dark:hover:bg-primary/50"></div>
              </div>
              <div className="relative flex h-full flex-col-reverse gap-2">
                <p className="text-center text-xs font-medium text-gray-500 dark:text-gray-400">15-21</p>
                <div className="h-[60%] w-full rounded-t-lg bg-primary/20 hover:bg-primary/40 dark:bg-primary/30 dark:hover:bg-primary/50"></div>
              </div>
              <div className="relative flex h-full flex-col-reverse gap-2">
                <p className="text-center text-xs font-medium text-gray-500 dark:text-gray-400">22-30</p>
                <div className="h-[95%] w-full rounded-t-lg bg-primary/20 hover:bg-primary/40 dark:bg-primary/30 dark:hover:bg-primary/50"></div>
              </div>
            </div>
          </div>

          {/* Data Table */}
          <div className="mt-6 flex flex-col gap-4 rounded-xl border border-gray-200/50 bg-white p-6 dark:border-white/10 dark:bg-black/10">
            <h3 className="text-lg font-bold text-gray-900 dark:text-white">Recent High-Priority Events</h3>
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200/50 dark:divide-white/10">
                <thead>
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400" scope="col">Event Type</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400" scope="col">Process Name/ID</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400" scope="col">Status</th>
                    <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400" scope="col">Date</th>
                    <th className="relative px-4 py-3" scope="col"><span className="sr-only">Details</span></th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200/50 dark:divide-white/10">
                  <tr>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-medium text-red-500">Violation</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-mono text-gray-500 dark:text-gray-400">onboard-customer-8f3a1b</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm">
                      <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900/40 dark:text-red-300">SLA Breach</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm text-gray-500 dark:text-gray-400">2024-07-21 14:30 UTC</td>
                    <td className="whitespace-nowrap px-4 py-4 text-right text-sm font-medium">
                      <a className="text-primary hover:underline" href="#">View Details</a>
                    </td>
                  </tr>
                  <tr>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-medium text-yellow-500">Approval</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-mono text-gray-500 dark:text-gray-400">loan-application-c4e9d2</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm">
                      <span className="inline-flex items-center rounded-full bg-yellow-100 px-2.5 py-0.5 text-xs font-medium text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300">Pending</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm text-gray-500 dark:text-gray-400">2024-07-21 11:15 UTC</td>
                    <td className="whitespace-nowrap px-4 py-4 text-right text-sm font-medium">
                      <a className="text-primary hover:underline" href="#">View Details</a>
                    </td>
                  </tr>
                  <tr>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-medium text-green-500">Audit</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-mono text-gray-500 dark:text-gray-400">quarterly-report-g6h7j8</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm">
                      <span className="inline-flex items-center rounded-full bg-green-100 px-2.5 py-0.5 text-xs font-medium text-green-800 dark:bg-green-900/40 dark:text-green-300">Completed</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm text-gray-500 dark:text-gray-400">2024-07-20 09:00 UTC</td>
                    <td className="whitespace-nowrap px-4 py-4 text-right text-sm font-medium">
                      <a className="text-primary hover:underline" href="#">View Details</a>
                    </td>
                  </tr>
                  <tr>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-medium text-red-500">Violation</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm font-mono text-gray-500 dark:text-gray-400">trade-settlement-k2m3n4</td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm">
                      <span className="inline-flex items-center rounded-full bg-red-100 px-2.5 py-0.5 text-xs font-medium text-red-800 dark:bg-red-900/40 dark:text-red-300">Data Mismatch</span>
                    </td>
                    <td className="whitespace-nowrap px-4 py-4 text-sm text-gray-500 dark:text-gray-400">2024-07-19 17:45 UTC</td>
                    <td className="whitespace-nowrap px-4 py-4 text-right text-sm font-medium">
                      <a className="text-primary hover:underline" href="#">View Details</a>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};
