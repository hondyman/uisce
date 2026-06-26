import React from 'react';

export const ProcessCatalogPage: React.FC = () => {
  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-background-dark text-text-light-primary dark:text-text-dark-primary">
      {/* Side Navigation Bar */}
      <aside className="flex w-64 flex-col bg-white/5 dark:bg-background-dark border-r border-white/10 dark:border-white/10">
        <div className="flex h-full flex-col justify-between p-4">
          <div className="flex flex-col gap-6">
            <div className="flex items-center gap-3 p-2">
              <span className="material-symbols-outlined text-primary text-3xl">verified_user</span>
              <h1 className="text-white text-xl font-bold">Regulator</h1>
            </div>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
                  href="#"
                >
                  <span className="material-symbols-outlined">dashboard</span>
                  <p className="text-sm font-medium leading-normal">Dashboard</p>
                </a>
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/20 text-primary"
                  href="#"
                >
                  <span className="material-symbols-outlined">menu_book</span>
                  <p className="text-sm font-medium leading-normal">Process Catalog</p>
                </a>
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
                  href="#"
                >
                  <span className="material-symbols-outlined">rule</span>
                  <p className="text-sm font-medium leading-normal">Approval Explorer</p>
                </a>
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
                  href="#"
                >
                  <span className="material-symbols-outlined">notifications</span>
                  <p className="text-sm font-medium leading-normal">Notification Center</p>
                </a>
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
                  href="#"
                >
                  <span className="material-symbols-outlined">hourglass_top</span>
                  <p className="text-sm font-medium leading-normal">SLA & Escalation</p>
                </a>
                <a
                  className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
                  href="#"
                >
                  <span className="material-symbols-outlined">policy</span>
                  <p className="text-sm font-medium leading-normal">Audit Explorer</p>
                </a>
              </div>
            </div>
          </div>
          <div className="flex flex-col gap-2 border-t border-white/10 pt-4">
            <div className="flex items-center gap-3 px-3 py-2">
              <div
                className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10"
                style={{
                  backgroundImage:
                    'url("https://lh3.googleusercontent.com/aida-public/AB6AXuCXKsfgrXiXRpbnwRVQS-VJ3DS7lwZ7K5Y5tLncGinshivXACRCX4Oz0z_mPXnW01pYPbJMDaFnGkPdm1xMHF92dNbXF0mjz1WI61QHD6_ewF7GRq8fNwXKJ4Lq1h9YKDipT_badwjJRSCQxeMIGVSEI9Hn-zdL_xZUANESNlDzbrFIfwFhnByhQ5dSx8CVPWcuxJf2zUl8gE281_7kqsNe9hcWGQzpaRm0UrBWSv1U3dPFkobiCaFLy5PbJ_Ju1UqM23CaTSjmtHRA")',
                }}
              ></div>
              <div className="flex flex-col">
                <h1 className="text-white text-sm font-medium leading-normal">Regulatory Officer</h1>
                <p className="text-gray-400 text-xs font-normal leading-normal">regulator@gov.org</p>
              </div>
            </div>
            <a
              className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
              href="#"
            >
              <span className="material-symbols-outlined">settings</span>
              <p className="text-sm font-medium leading-normal">Settings</p>
            </a>
            <a
              className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white"
              href="#"
            >
              <span className="material-symbols-outlined">logout</span>
              <p className="text-sm font-medium leading-normal">Logout</p>
            </a>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 p-8 overflow-y-auto">
        <div className="flex flex-col gap-8">
          {/* Page Header */}
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            <div className="flex flex-col gap-1">
              <p className="text-white text-3xl font-bold leading-tight">Process Catalog</p>
              <p className="text-gray-400 text-sm font-normal leading-normal">
                View and audit all workflow processes and their version history.
              </p>
            </div>
            <div className="flex items-center gap-4">
              <label className="flex flex-col min-w-40">
                <p className="text-white text-sm font-medium leading-normal pb-2">Tenant</p>
                <select className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-white focus:outline-0 focus:ring-0 border border-white/20 bg-[#1C2127] focus:border-primary h-12 placeholder:text-gray-400 px-3 py-2 text-base font-normal leading-normal">
                  <option value="one">Global Corp Inc.</option>
                  <option selected value="two">
                    Innovate Solutions Ltd.
                  </option>
                  <option value="three">Quantum Dynamics</option>
                </select>
              </label>
            </div>
          </div>

          {/* Catalog Table */}
          <div className="overflow-x-auto rounded-xl border border-white/10 bg-white/5">
            <table className="w-full text-left text-sm text-gray-300">
              <thead className="text-xs text-gray-400 uppercase bg-white/5">
                <tr>
                  <th className="px-6 py-3 w-12"></th>
                  <th className="px-6 py-3" scope="col">
                    ID
                  </th>
                  <th className="px-6 py-3" scope="col">
                    Name
                  </th>
                  <th className="px-6 py-3" scope="col">
                    Object Type
                  </th>
                  <th className="px-6 py-3" scope="col">
                    Current Version
                  </th>
                  <th className="px-6 py-3" scope="col">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {/* Row 1 (Expanded) */}
                <tr className="border-b border-white/10">
                  <td className="px-6 py-4">
                    <span className="material-symbols-outlined text-gray-400 cursor-pointer">expand_more</span>
                  </td>
                  <th className="px-6 py-4 font-medium text-white whitespace-nowrap" scope="row">
                    PROC-001
                  </th>
                  <td className="px-6 py-4">New Supplier Onboarding</td>
                  <td className="px-6 py-4">Supplier</td>
                  <td className="px-6 py-4">3.0</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center rounded-full bg-green-900/20 px-2 py-1 text-xs font-medium text-green-500">
                      Active
                    </span>
                  </td>
                </tr>
                <tr className="border-b border-white/10 bg-background-dark">
                  <td className="p-0" colSpan={6}>
                    <div className="p-6">
                      <h3 className="text-lg font-bold text-white mb-4">Version History: New Supplier Onboarding</h3>
                      <div className="space-y-4">
                        {/* Version 3.0 */}
                        <div className="p-4 rounded-lg bg-white/5 border border-white/10">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4">
                              <p className="font-bold text-white">Version 3.0</p>
                              <span className="inline-flex items-center rounded-full bg-green-900/20 px-2 py-1 text-xs font-medium text-green-500">
                                Published
                              </span>
                              <p className="text-gray-400 text-xs">Published on 2023-10-15 by admin@innovate.com</p>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                className="form-checkbox bg-[#1C2127] border-white/20 rounded text-primary focus:ring-primary"
                                type="checkbox"
                              />
                              <button className="flex items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/10">
                                <span className="material-symbols-outlined text-base">difference</span>Compare
                              </button>
                            </div>
                          </div>
                          <div className="mt-3 pt-3 border-t border-white/10">
                            <p className="text-sm font-medium text-gray-300">UAR Linkage:</p>
                            <div className="text-xs text-gray-400 mt-1 font-mono">
                              <a className="text-primary/80 hover:underline" href="#">
                                hash: 0x...a4f2
                              </a>{' '}
                              -&gt;{' '}
                              <a className="text-primary/80 hover:underline" href="#">
                                prevHash: 0x...c3d1
                              </a>
                            </div>
                          </div>
                        </div>
                        {/* Version 2.0 */}
                        <div className="p-4 rounded-lg bg-white/5 border border-white/10">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4">
                              <p className="font-bold text-white">Version 2.0</p>
                              <span className="inline-flex items-center rounded-full bg-yellow-500/20 px-2 py-1 text-xs font-medium text-yellow-400">
                                Deprecated
                              </span>
                              <p className="text-gray-400 text-xs">Deprecated on 2023-10-14 by admin@innovate.com</p>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                className="form-checkbox bg-[#1C2127] border-white/20 rounded text-primary focus:ring-primary"
                                type="checkbox"
                              />
                              <button className="flex items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/10">
                                <span className="material-symbols-outlined text-base">difference</span>Compare
                              </button>
                            </div>
                          </div>
                          <div className="mt-3 pt-3 border-t border-white/10">
                            <p className="text-sm font-medium text-gray-300">UAR Linkage:</p>
                            <div className="text-xs text-gray-400 mt-1 font-mono">
                              <a className="text-primary/80 hover:underline" href="#">
                                hash: 0x...c3d1
                              </a>{' '}
                              -&gt;{' '}
                              <a className="text-primary/80 hover:underline" href="#">
                                prevHash: 0x...b9e8
                              </a>
                            </div>
                          </div>
                        </div>
                        {/* Version 1.0 */}
                        <div className="p-4 rounded-lg bg-white/5 border border-white/10">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center gap-4">
                              <p className="font-bold text-white">Version 1.0</p>
                              <span className="inline-flex items-center rounded-full bg-gray-500/20 px-2 py-1 text-xs font-medium text-gray-400">
                                Archived
                              </span>
                              <p className="text-gray-400 text-xs">Archived on 2023-01-20 by system</p>
                            </div>
                            <div className="flex items-center gap-2">
                              <input
                                className="form-checkbox bg-[#1C2127] border-white/20 rounded text-primary focus:ring-primary"
                                type="checkbox"
                              />
                              <button className="flex items-center gap-2 rounded-md px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/10">
                                <span className="material-symbols-outlined text-base">difference</span>Compare
                              </button>
                            </div>
                          </div>
                          <div className="mt-3 pt-3 border-t border-white/10">
                            <p className="text-sm font-medium text-gray-300">UAR Linkage:</p>
                            <div className="text-xs text-gray-400 mt-1 font-mono">
                              <a className="text-primary/80 hover:underline" href="#">
                                hash: 0x...b9e8
                              </a>{' '}
                              -&gt;{' '}
                              <a className="text-primary/80 hover:underline" href="#">
                                prevHash: 0x...a1f0
                              </a>
                            </div>
                          </div>
                        </div>
                      </div>
                      <div className="mt-6 flex justify-end">
                        <button className="flex items-center justify-center rounded-lg h-10 bg-primary text-white gap-2 text-sm font-bold leading-normal tracking-[0.015em] px-4 hover:bg-primary/90 disabled:bg-gray-500 disabled:cursor-not-allowed">
                          <span className="material-symbols-outlined text-xl">compare_arrows</span>
                          <span className="truncate">Compare Selected (2)</span>
                        </button>
                      </div>
                    </div>
                  </td>
                </tr>
                {/* Row 2 */}
                <tr className="border-b border-white/10">
                  <td className="px-6 py-4">
                    <span className="material-symbols-outlined text-gray-400 cursor-pointer">chevron_right</span>
                  </td>
                  <th className="px-6 py-4 font-medium text-white whitespace-nowrap" scope="row">
                    PROC-002
                  </th>
                  <td className="px-6 py-4">Expense Report Approval</td>
                  <td className="px-6 py-4">Finance</td>
                  <td className="px-6 py-4">1.2</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center rounded-full bg-green-900/20 px-2 py-1 text-xs font-medium text-green-500">
                      Active
                    </span>
                  </td>
                </tr>
                {/* Row 3 */}
                <tr className="border-b border-white/10">
                  <td className="px-6 py-4">
                    <span className="material-symbols-outlined text-gray-400 cursor-pointer">chevron_right</span>
                  </td>
                  <th className="px-6 py-4 font-medium text-white whitespace-nowrap" scope="row">
                    PROC-003
                  </th>
                  <td className="px-6 py-4">Employee Offboarding</td>
                  <td className="px-6 py-4">HR</td>
                  <td className="px-6 py-4">2.1</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center rounded-full bg-yellow-500/20 px-2 py-1 text-xs font-medium text-yellow-400">
                      Deprecated
                    </span>
                  </td>
                </tr>
                {/* Row 4 */}
                <tr className="">
                  <td className="px-6 py-4">
                    <span className="material-symbols-outlined text-gray-400 cursor-pointer">chevron_right</span>
                  </td>
                  <th className="px-6 py-4 font-medium text-white whitespace-nowrap" scope="row">
                    PROC-004
                  </th>
                  <td className="px-6 py-4">IT Asset Request</td>
                  <td className="px-6 py-4">IT</td>
                  <td className="px-6 py-4">4.0</td>
                  <td className="px-6 py-4">
                    <span className="inline-flex items-center rounded-full bg-green-900/20 px-2 py-1 text-xs font-medium text-green-500">
                      Active
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </main>
    </div>
  );
};
