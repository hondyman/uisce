import React from 'react';

const PlanDetailsPage = () => {
  return (
    <div className="relative flex h-auto min-h-screen w-full flex-col">
      <header className="sticky top-0 z-20 flex items-center justify-between whitespace-nowrap border-b border-slate-200 dark:border-slate-800 bg-background-light dark:bg-background-dark/80 backdrop-blur-sm px-8 py-3">
        <div className="flex items-center gap-4">
          <span className="material-symbols-outlined text-primary text-2xl">all_inbox</span>
          <h2 className="text-slate-900 dark:text-slate-100 text-lg font-bold tracking-tight">Workday Benefits Custom Validations</h2>
        </div>
        <div className="flex flex-1 justify-end items-center gap-4">
          <div className="flex gap-2">
            <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-slate-200/60 dark:bg-slate-800/60 text-slate-600 dark:text-slate-300">
              <span className="material-symbols-outlined text-xl">settings</span>
            </button>
            <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-slate-200/60 dark:bg-slate-800/60 text-slate-600 dark:text-slate-300">
              <span className="material-symbols-outlined text-xl">notifications</span>
            </button>
            <button className="flex h-10 w-10 cursor-pointer items-center justify-center overflow-hidden rounded-lg bg-slate-200/60 dark:bg-slate-800/60 text-slate-600 dark:text-slate-300">
              <span className="material-symbols-outlined text-xl">help</span>
            </button>
          </div>
          <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" data-alt="User avatar image" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuB-KAZLB6OYpfInF-II1ZVgylkJOvqLXihvYHVFNi1a53HoeyrzeQiEdhI6YDPybTiNtOlq0ms_wM-29gy6cyZnKAQW869tuUaPX2XHbfnOd6EkMYShf6M7EpskvsjG9DDQBOJq2yIyMRcff9tRVQa3mCyMxpEaoowBDmQx4N8lroOjKw1FVV9XC-wvghciEVdMS1ghdG6RyYWCuSw32I0eQeHduE10ukQAVhbhYv2of2GCQiOxxloj7M-E6DHMnCQRQsrxJF3bzWpc")'}}></div>
        </div>
      </header>
      <div className="flex h-full grow">
        <aside className="w-64 flex-shrink-0 border-r border-slate-200 dark:border-slate-800 bg-background-light dark:bg-background-dark p-4">
          <div className="flex h-full flex-col justify-between">
            <div className="flex flex-col gap-4">
              <div className="flex gap-3 items-center">
                <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-lg size-10" data-alt="Abstract logo for benefit plans" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuAHZ6Cm1qEhgT71OnF29wmOrRdcSvL6lJL_JXhsajiJyyHW8CuKKuirkx_lqGHecbMmTZNCrPk2e9seHj9ceJAYtm18isSF423UwmDdb2TK0z4eh1WE8z4XeobLx-_1bVSoGVPqiAUpBX6Z6d7XhrtNE21SXIwTGB9RfcLmV92dd3JGZ2_U5Z55IwFtMwjAkeOd7HKT4woYxiDlwTdwN0mVE9IXgujzz9jlRChv2uOIv6jqtgAlES2-XsuLnYCn7Mx5COeQyQUsWvD_")'}}></div>
                <div className="flex flex-col">
                  <h1 className="text-slate-900 dark:text-slate-100 text-base font-medium leading-normal">Benefit Plans</h1>
                  <p className="text-gray-accent text-sm font-normal leading-normal">Administrator View</p>
                </div>
              </div>
              <div className="flex flex-col gap-1 mt-4">
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/10 text-primary dark:bg-primary/20" href="#plan-overview">
                  <span className="material-symbols-outlined text-lg">info</span>
                  <p className="text-sm font-medium leading-normal">Plan Overview</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-200/60 dark:hover:bg-slate-800/60 text-slate-700 dark:text-slate-300" href="#coverage-details">
                  <span className="material-symbols-outlined text-lg">shield</span>
                  <p className="text-sm font-medium leading-normal">Coverage Details</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-200/60 dark:hover:bg-slate-800/60 text-slate-700 dark:text-slate-300" href="#costs-contributions">
                  <span className="material-symbols-outlined text-lg">payments</span>
                  <p className="text-sm font-medium leading-normal">Costs & Contributions</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-200/60 dark:hover:bg-slate-800/60 text-slate-700 dark:text-slate-300" href="#eligibility">
                  <span className="material-symbols-outlined text-lg">person_check</span>
                  <p className="text-sm font-medium leading-normal">Eligibility</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-slate-200/60 dark:hover:bg-slate-800/60 text-slate-700 dark:text-slate-300" href="#validation-rules">
                  <span className="material-symbols-outlined text-lg">rule</span>
                  <p className="text-sm font-medium leading-normal">Validation Rules</p>
                </a>
              </div>
            </div>
          </div>
        </aside>
        <main className="flex-1 p-8 overflow-y-auto">
          <div className="mx-auto max-w-5xl">
            <div className="flex flex-wrap gap-2">
              <a className="text-gray-accent hover:text-primary text-base font-medium leading-normal" href="#">All Plans</a>
              <span className="text-gray-accent text-base font-medium leading-normal">/</span>
              <a className="text-gray-accent hover:text-primary text-base font-medium leading-normal" href="#">Medical</a>
              <span className="text-gray-accent text-base font-medium leading-normal">/</span>
              <span className="text-slate-900 dark:text-slate-100 text-base font-medium leading-normal">PPO Gold</span>
            </div>
            <div className="sticky top-[69px] z-10 bg-background-light dark:bg-background-dark py-4 -my-4 mb-4">
              <div className="flex flex-wrap items-center justify-between gap-4 py-4 border-b border-slate-200 dark:border-slate-800">
                <h1 className="text-slate-900 dark:text-slate-100 text-4xl font-black tracking-tighter">PPO Gold Medical Plan</h1>
                <div className="flex gap-3">
                  <button className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-slate-200/80 dark:bg-slate-800/80 text-slate-800 dark:text-slate-200 text-sm font-bold">
                    <span className="truncate">Clone</span>
                  </button>
                  <button className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-transparent text-gray-accent dark:text-gray-accent text-sm font-bold hover:bg-slate-200/60 dark:hover:bg-slate-800/60">
                    <span className="truncate">Deactivate</span>
                  </button>
                  <button className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold">
                    <span className="truncate">Edit Plan</span>
                  </button>
                </div>
              </div>
            </div>
            <div className="space-y-12 mt-8">
              <section id="plan-overview">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-4">Plan Overview</h2>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
                  <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800">
                    <h3 className="font-medium text-gray-accent mb-1">Plan Type</h3>
                    <p className="text-lg font-semibold text-slate-900 dark:text-slate-100">Medical</p>
                  </div>
                  <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800">
                    <h3 className="font-medium text-gray-accent mb-1">Status</h3>
                    <div className="inline-flex items-center gap-2 rounded-full bg-success/10 px-3 py-1 text-sm font-medium text-success">
                      <span className="h-2 w-2 rounded-full bg-success"></span>
                      Active
                    </div>
                  </div>
                  <div className="bg-white dark:bg-slate-900 p-6 rounded-xl border border-slate-200 dark:border-slate-800">
                    <h3 className="font-medium text-gray-accent mb-1">Effective Dates</h3>
                    <p className="text-lg font-semibold text-slate-900 dark:text-slate-100">Jan 1, 2024 - Dec 31, 2024</p>
                  </div>
                </div>
              </section>
              <section id="coverage-details">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-4">Coverage Details</h2>
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                  <p className="text-gray-accent">Detailed information about what this plan covers, including deductibles, co-pays, and out-of-pocket maximums for different network tiers.</p>
                </div>
              </section>
              <section id="costs-contributions">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-4">Costs & Contributions</h2>
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 overflow-hidden">
                  <div className="overflow-x-auto">
                    <table className="w-full text-left text-sm">
                      <thead className="bg-slate-50 dark:bg-slate-800/50 text-xs uppercase text-gray-accent">
                        <tr>
                          <th className="px-6 py-3" scope="col">Coverage Tier</th>
                          <th className="px-6 py-3" scope="col">Employee Cost (Per Pay Period)</th>
                          <th className="px-6 py-3" scope="col">Employer Contribution</th>
                          <th className="px-6 py-3" scope="col">Total Premium</th>
                        </tr>
                      </thead>
                      <tbody>
                        <tr className="border-b border-slate-200 dark:border-slate-800">
                          <th className="px-6 py-4 font-medium text-slate-900 dark:text-slate-100 whitespace-nowrap" scope="row">Employee Only</th>
                          <td className="px-6 py-4">$120.50</td>
                          <td className="px-6 py-4">$450.00</td>
                          <td className="px-6 py-4">$570.50</td>
                        </tr>
                        <tr className="border-b border-slate-200 dark:border-slate-800">
                          <th className="px-6 py-4 font-medium text-slate-900 dark:text-slate-100 whitespace-nowrap" scope="row">Employee + Spouse</th>
                          <td className="px-6 py-4">$250.75</td>
                          <td className="px-6 py-4">$800.00</td>
                          <td className="px-6 py-4">$1050.75</td>
                        </tr>
                        <tr className="border-b border-slate-200 dark:border-slate-800">
                          <th className="px-6 py-4 font-medium text-slate-900 dark:text-slate-100 whitespace-nowrap" scope="row">Employee + Child(ren)</th>
                          <td className="px-6 py-4">$235.00</td>
                          <td className="px-6 py-4">$750.00</td>
                          <td className="px-6 py-4">$985.00</td>
                        </tr>
                        <tr>
                          <th className="px-6 py-4 font-medium text-slate-900 dark:text-slate-100 whitespace-nowrap" scope="row">Family</th>
                          <td className="px-6 py-4">$380.25</td>
                          <td className="px-6 py-4">$1100.00</td>
                          <td className="px-6 py-4">$1480.25</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </section>
              <section id="eligibility">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-4">Eligibility</h2>
                <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-6">
                  <p className="text-gray-accent">Defines which employees are eligible for this plan based on employment status, location, hours worked, and other criteria. <a className="text-primary font-medium" href="#">View detailed criteria</a>.</p>
                </div>
              </section>
              <section id="validation-rules">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-slate-100 mb-4">Validation Rules</h2>
                <div className="space-y-4">
                  <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-5">
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="font-bold text-lg text-slate-900 dark:text-slate-100">HSA Ineligibility Check</h3>
                        <p className="text-gray-accent mt-1">Prevents enrollment in an HSA-compatible plan if also enrolled in a non-HDHP medical plan.</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="inline-flex items-center gap-2 rounded-full bg-error/10 px-3 py-1 text-sm font-medium text-error">
                          Error
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-full bg-success/10 px-3 py-1 text-sm font-medium text-success">
                          Enabled
                        </div>
                      </div>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800">
                      <p className="text-sm text-gray-accent"><strong>Trigger:</strong> Employee submits election for this plan. <strong>Message:</strong> "You cannot enroll in an HSA while covered by another medical plan that is not a High-Deductible Health Plan."</p>
                    </div>
                  </div>
                  <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-5">
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="font-bold text-lg text-slate-900 dark:text-slate-100">Spouse Surcharge Confirmation</h3>
                        <p className="text-gray-accent mt-1">Requires confirmation if a spouse has other employer-sponsored coverage available.</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="inline-flex items-center gap-2 rounded-full bg-warning/10 px-3 py-1 text-sm font-medium text-warning">
                          Warning
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-full bg-success/10 px-3 py-1 text-sm font-medium text-success">
                          Enabled
                        </div>
                      </div>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800">
                      <p className="text-sm text-gray-accent"><strong>Trigger:</strong> Employee adds a spouse to coverage. <strong>Message:</strong> "Please confirm if your spouse has access to other medical coverage through their employer. A surcharge may apply."</p>
                    </div>
                  </div>
                  <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800 p-5">
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="font-bold text-lg text-slate-900 dark:text-slate-100">Evidence of Insurability (EOI) for Late Entrants</h3>
                        <p className="text-gray-accent mt-1">Flags employees enrolling outside their initial eligibility period for EOI requirement.</p>
                      </div>
                      <div className="flex items-center gap-3">
                        <div className="inline-flex items-center gap-2 rounded-full bg-gray-accent/10 px-3 py-1 text-sm font-medium text-gray-accent">
                          Info
                        </div>
                        <div className="inline-flex items-center gap-2 rounded-full bg-gray-accent/10 px-3 py-1 text-sm font-medium text-gray-accent">
                          Disabled
                        </div>
                      </div>
                    </div>
                    <div className="mt-4 pt-4 border-t border-slate-200 dark:border-slate-800">
                      <p className="text-sm text-gray-accent"><strong>Trigger:</strong> Enrollment event is a 'Late Enrollment' life event. <strong>Message:</strong> "This enrollment requires Evidence of Insurability. Please complete the required forms."</p>
                    </div>
                  </div>
                </div>
              </section>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

export default PlanDetailsPage;