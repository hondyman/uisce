import React, { useState } from 'react';

export const SLADashboardPage: React.FC = () => {
  const [expandedRow, setExpandedRow] = useState<string | null>(null);

  const toggleRow = (id: string) => {
    if (expandedRow === id) {
      setExpandedRow(null);
    } else {
      setExpandedRow(id);
    }
  };

  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-background-dark text-gray-900 dark:text-white">
      {/* Sidebar */}
      <aside className="flex w-64 flex-col border-r border-gray-200 dark:border-gray-800 bg-background-light dark:bg-background-dark p-4">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuDNAJKjX0zFjfzjlZdW4Cp9L2YlpZVRuyEK2FESO4_oNof3-uVxyFnw-8ftr80qQgDcpw6B7PWuxhuwSb63IPDpzTndvs6ZuvsGNfkJKXPD9cZvsrM9nzV_KkrK-rfxtW6_rKaoBsiiX_Q30fg2XaPAAy66GWkPdQ9GZOyQf4DsVCzqT6IB0W7wh1aPpVcChc_AAK2aWT85VBmMlXQiv3XpyU6fUrp3UfSzeyyPgayGPtgZ6XHKY1gm4lkEPXk6AS_vDjM_g92W-hSc")' }}></div>
            <div className="flex flex-col">
              <h1 className="text-gray-900 dark:text-white text-base font-medium leading-normal">Eleanor Vance</h1>
              <p className="text-gray-500 dark:text-[#9dabb9] text-sm font-normal leading-normal">Product Manager</p>
            </div>
          </div>
          <nav className="flex flex-col gap-2 mt-4">
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-700 dark:text-white hover:bg-gray-200 dark:hover:bg-[#283039]" href="/core/regulator-portal">
              <span className="material-symbols-outlined text-2xl">dashboard</span>
              <p className="text-sm font-medium leading-normal">Dashboard</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/20 text-primary" href="#">
              <span className="material-symbols-outlined text-2xl" style={{ fontVariationSettings: "'FILL' 1" }}>lock</span>
              <p className="text-sm font-medium leading-normal">SLA Dashboard</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-700 dark:text-white hover:bg-gray-200 dark:hover:bg-[#283039]" href="/core/process-catalog">
              <span className="material-symbols-outlined text-2xl">view_kanban</span>
              <p className="text-sm font-medium leading-normal">Processes</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-700 dark:text-white hover:bg-gray-200 dark:hover:bg-[#283039]" href="#">
              <span className="material-symbols-outlined text-2xl">pie_chart</span>
              <p className="text-sm font-medium leading-normal">Analytics</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-700 dark:text-white hover:bg-gray-200 dark:hover:bg-[#283039]" href="#">
              <span className="material-symbols-outlined text-2xl">settings</span>
              <p className="text-sm font-medium leading-normal">Settings</p>
            </a>
          </nav>
        </div>
        <div className="mt-auto">
          <button className="flex w-full min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em]">
            <span className="truncate">Create Process</span>
          </button>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex-1 flex flex-col h-screen overflow-y-auto">
        <div className="p-8 space-y-6">
          <header className="flex flex-wrap justify-between items-center gap-4">
            <div className="flex-1 min-w-72">
              <p className="text-gray-900 dark:text-white text-4xl font-black leading-tight tracking-[-0.033em]">SLA Breaches</p>
              <p className="text-gray-500 dark:text-gray-400 mt-1">Monitor, manage, and audit all SLA violations in real-time.</p>
            </div>
            <div className="flex items-center gap-2">
              <span className="material-symbols-outlined text-green-500 animate-pulse">rss_feed</span>
              <span className="text-sm font-medium text-gray-600 dark:text-gray-300">Real-time updates enabled</span>
            </div>
          </header>

          <section className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div>
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300" htmlFor="tenant-filter">Tenant</label>
              <select className="mt-1 block w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-background-light dark:bg-[#1C2127] text-gray-800 dark:text-gray-200 focus:ring-primary focus:border-primary" id="tenant-filter">
                <option>All Tenants</option>
                <option>Tenant A</option>
                <option>Tenant B</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300" htmlFor="role-filter">Role</label>
              <select className="mt-1 block w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-background-light dark:bg-[#1C2127] text-gray-800 dark:text-gray-200 focus:ring-primary focus:border-primary" id="role-filter">
                <option>All Roles</option>
                <option>Legal Team</option>
                <option>Finance</option>
                <option>Compliance</option>
              </select>
            </div>
            <div>
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300" htmlFor="process-filter">Process</label>
              <select className="mt-1 block w-full rounded-lg border border-gray-300 dark:border-gray-600 bg-background-light dark:bg-[#1C2127] text-gray-800 dark:text-gray-200 focus:ring-primary focus:border-primary" id="process-filter">
                <option>All Processes</option>
                <option>New Client Onboarding</option>
                <option>Supplier Vetting</option>
              </select>
            </div>
            <div className="relative">
              <label className="text-sm font-medium text-gray-700 dark:text-gray-300" htmlFor="search-input">Search</label>
              <span className="material-symbols-outlined absolute left-3 bottom-2.5 text-gray-500">search</span>
              <input className="mt-1 w-full pl-10 pr-4 py-2 rounded-lg border border-gray-300 dark:border-gray-600 bg-background-light dark:bg-[#1C2127] text-gray-800 dark:text-gray-200 focus:ring-primary focus:border-primary" id="search-input" placeholder="Search by ID, Owner..." type="text" />
            </div>
          </section>

          <section className="flex flex-col gap-4">
            <div className="overflow-x-auto">
              <div className="inline-block min-w-full align-middle">
                <div className="overflow-hidden rounded-lg border border-gray-200 dark:border-[#3b4754]">
                  <table className="min-w-full divide-y divide-gray-200 dark:divide-[#3b4754]">
                    <thead className="bg-gray-50 dark:bg-[#1c2127]">
                      <tr>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Process</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Stage</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Due Date</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Owner</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Severity</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Status</th>
                        <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-white uppercase tracking-wider" scope="col">Audit</th>
                        <th className="w-12 px-4 py-3" scope="col"><span className="sr-only">Expand</span></th>
                      </tr>
                    </thead>
                    <tbody className="bg-white dark:bg-background-dark divide-y divide-gray-200 dark:divide-[#3b4754]">
                      
                      {/* Row 1 */}
                      <tr className="hover:bg-gray-50 dark:hover:bg-[#1c2127]">
                        <td className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">New Client Onboarding</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">Contract Review</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">2023-10-26 (2 days ago)</td>
                        <td className="px-4 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full w-8 h-8" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuBZu1i9gdSHVv3G3aQ0o72y0y5j3PKEkM3AOpIcDkAE2fs61eCfaQFuFqnfXf4kAttXjPC6ZCiP99lt1Ejqk7VPhKrxwcCyT07JBk37A9QSsYQQKOLO22_B_UcV0PNJOHCl5HffStNpO4Tf3QHaNUf2z7N3-wRFzs5HMcQeqLSsWXR14D-HonKXHApsuNl5INayaPE-crsqdE2CxPazUE1b1ue6f_VOGv3p8zeNUMmXhH6uyDMz9tsQtyLxPwqmD_zABhgxF0DtYl7m")' }}></div>
                            <span>L. Evans</span>
                          </div>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/40">
                            <span className="size-1.5 rounded-full bg-red-600 dark:bg-red-400"></span>Critical
                          </span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/40">Breached</span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <a className="flex items-center gap-1 text-primary hover:underline" href="#">
                            <span className="material-symbols-outlined text-base">link</span>
                            <span>UAR-3341</span>
                          </a>
                        </td>
                        <td className="px-4 py-4">
                          <button onClick={() => toggleRow('row1')} className="p-1 text-gray-500 hover:text-gray-900 dark:hover:text-white">
                            <span className="material-symbols-outlined">{expandedRow === 'row1' ? 'expand_less' : 'expand_more'}</span>
                          </button>
                        </td>
                      </tr>
                      {expandedRow === 'row1' && (
                        <tr className="bg-gray-50 dark:bg-[#1C2127]">
                          <td className="p-4" colSpan={8}>
                            <div className="space-y-3">
                              <h4 className="text-sm font-semibold text-gray-800 dark:text-gray-200">Escalation Path</h4>
                              <div className="flex items-start gap-4">
                                <div className="flex flex-col items-center">
                                  <span className="flex items-center justify-center size-8 rounded-full bg-red-100 dark:bg-red-900/40 text-red-600 dark:text-red-300">
                                    <span className="material-symbols-outlined text-lg">notifications</span>
                                  </span>
                                  <div className="w-px h-8 bg-gray-300 dark:bg-gray-600"></div>
                                </div>
                                <div className="flex-1 pb-8">
                                  <p className="text-sm font-medium text-gray-900 dark:text-white">Breach Notification Sent</p>
                                  <p className="text-xs text-gray-500 dark:text-gray-400">Notified: Legal Team DL, Sarah Jennings (Manager)</p>
                                  <p className="text-xs text-gray-500 dark:text-gray-400">Timestamp: 2023-10-27 09:05 UTC</p>
                                </div>
                              </div>
                              <div className="flex items-start gap-4">
                                <div className="flex flex-col items-center">
                                  <span className="flex items-center justify-center size-8 rounded-full bg-green-100 dark:bg-green-900/40 text-green-600 dark:text-green-300">
                                    <span className="material-symbols-outlined text-lg">check_circle</span>
                                  </span>
                                </div>
                                <div className="flex-1">
                                  <p className="text-sm font-medium text-gray-900 dark:text-white">Outcome: Action Plan Initiated</p>
                                  <p className="text-xs text-gray-500 dark:text-gray-400">Assigned to: L. Evans for immediate resolution.</p>
                                  <p className="text-xs text-gray-500 dark:text-gray-400">Timestamp: 2023-10-27 10:00 UTC</p>
                                </div>
                              </div>
                            </div>
                          </td>
                        </tr>
                      )}

                      {/* Row 2 */}
                      <tr className="hover:bg-gray-50 dark:hover:bg-[#1c2127]">
                        <td className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">Supplier Vetting</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">Financial Background Check</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">2023-10-28 (1 day ago)</td>
                        <td className="px-4 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full w-8 h-8" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuDchkuFWepYeaDfyIbpt2a-zYpK4oQtH2M2r93mnHLHiw4QbejzfZFZqF9Mo3YqeEV8iDD-AE1HVZg1hB35HT2wpzpbCHsAzuBJRtFH1xQnC0F6xoToKPrN-L9SoyYHxvDS1vcKlYJaHVjKLE7dhOKLuHJM_knPrJnyDZyrpISbNomOIDK0uxskPoVeflNaMU8YE_Ea4ETdEyhr7S9Jp25HNkwOjOYjegiRwCbGnXvaWR0HIwG2mR5hZNADd3CI8NFNIoIucAsLhOn6")' }}></div>
                            <span>M. Rodriguez</span>
                          </div>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium text-orange-700 bg-orange-100 dark:text-orange-300 dark:bg-orange-900/40">
                            <span className="size-1.5 rounded-full bg-orange-600 dark:bg-orange-400"></span>High
                          </span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium text-red-700 bg-red-100 dark:text-red-300 dark:bg-red-900/40">Breached</span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <a className="flex items-center gap-1 text-primary hover:underline" href="#">
                            <span className="material-symbols-outlined text-base">link</span>
                            <span>UAR-3329</span>
                          </a>
                        </td>
                        <td className="px-4 py-4">
                          <button onClick={() => toggleRow('row2')} className="p-1 text-gray-500 hover:text-gray-900 dark:hover:text-white">
                            <span className="material-symbols-outlined">{expandedRow === 'row2' ? 'expand_less' : 'expand_more'}</span>
                          </button>
                        </td>
                      </tr>

                      {/* Row 3 */}
                      <tr className="hover:bg-gray-50 dark:hover:bg-[#1c2127]">
                        <td className="px-4 py-4 whitespace-nowrap text-sm font-medium text-gray-900 dark:text-white">Employee Offboarding</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">Exit Interview</td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-[#9dabb9]">2023-10-29 (Today)</td>
                        <td className="px-4 py-4 whitespace-nowrap">
                          <div className="flex items-center gap-2">
                            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full w-8 h-8" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuBi8tNI6czqMo7Yw00J3xlGfHZzs4pUXzB0uq59gt1tilDwBEaXJpTjKMuhZY_9XaNiZan3WhllYh3dyZ8TUIvoMe2zq7pv5dHstLGUvbROgtECBnv1WEP6e-knTObSRh9RKnRillUhcpjKeigIeEuZ8pZp0oylfZasIIbLD-OeJ0PJ7U-QUmqW9hxO8Rf-Ryj7z0sPpWHXigeQiyxoBfZiy61v8mgjx6Tvux0R6PZeHy_40_SP4GltbdjyAStIe7dj3uq0CaiTzdkO")' }}></div>
                            <span>J. Chen</span>
                          </div>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center gap-1.5 rounded-full px-2 py-1 text-xs font-medium text-yellow-700 bg-yellow-100 dark:text-yellow-300 dark:bg-yellow-900/40">
                            <span className="size-1.5 rounded-full bg-yellow-600 dark:bg-yellow-400"></span>Medium
                          </span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="inline-flex items-center rounded-full px-2 py-1 text-xs font-medium text-orange-700 bg-orange-100 dark:text-orange-300 dark:bg-orange-900/40">At Risk</span>
                        </td>
                        <td className="px-4 py-4 whitespace-nowrap text-sm">
                          <span className="text-gray-400 dark:text-gray-500">N/A</span>
                        </td>
                        <td className="px-4 py-4">
                          <button onClick={() => toggleRow('row3')} className="p-1 text-gray-500 hover:text-gray-900 dark:hover:text-white">
                            <span className="material-symbols-outlined">{expandedRow === 'row3' ? 'expand_less' : 'expand_more'}</span>
                          </button>
                        </td>
                      </tr>

                    </tbody>
                  </table>
                </div>
              </div>
            </div>

            {/* Footer */}
            <footer className="flex items-center justify-between pt-2">
              <span className="text-sm text-gray-600 dark:text-gray-400">
                Showing <span className="font-semibold text-gray-900 dark:text-white">1</span> to <span className="font-semibold text-gray-900 dark:text-white">3</span> of <span className="font-semibold text-gray-900 dark:text-white">86</span> Breaches
              </span>
              <div className="inline-flex -space-x-px rounded-md text-sm">
                <a className="flex items-center justify-center px-3 h-8 ms-0 leading-tight text-gray-500 bg-white border border-gray-300 rounded-s-lg hover:bg-gray-100 hover:text-gray-700 dark:bg-[#1c2127] dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-white" href="#">Previous</a>
                <a className="flex items-center justify-center px-3 h-8 leading-tight text-primary bg-primary/20 border border-primary hover:bg-primary/30 hover:text-primary dark:border-gray-700 dark:bg-gray-700 dark:text-white" href="#">1</a>
                <a className="flex items-center justify-center px-3 h-8 leading-tight text-gray-500 bg-white border border-gray-300 hover:bg-gray-100 hover:text-gray-700 dark:bg-[#1c2127] dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-white" href="#">2</a>
                <a className="flex items-center justify-center px-3 h-8 leading-tight text-gray-500 bg-white border border-gray-300 rounded-e-lg hover:bg-gray-100 hover:text-gray-700 dark:bg-[#1c2127] dark:border-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-white" href="#">Next</a>
              </div>
            </footer>
          </section>
        </div>
      </main>
    </div>
  );
};
