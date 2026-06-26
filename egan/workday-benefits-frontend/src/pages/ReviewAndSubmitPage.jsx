import React from 'react';

const ReviewAndSubmitPage = () => {
  return (
    <div className="relative flex min-h-screen w-full flex-col group/design-root overflow-x-hidden">
      {/* Top Nav Bar */}
      <header className="flex items-center justify-between whitespace-nowrap border-b border-slate-200 dark:border-slate-800 px-6 md:px-10 py-3 bg-white dark:bg-slate-900/50 sticky top-0 z-10">
        <div className="flex items-center gap-4 text-slate-900 dark:text-white">
          <div className="size-6">
            <svg className="text-primary" fill="none" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
              <path d="M4 4H17.3334V17.3334H30.6666V30.6666H44V44H4V4Z" fill="currentColor"></path>
            </svg>
          </div>
          <h2 className="text-lg font-bold leading-tight tracking-[-0.015em]">Workday Benefits</h2>
        </div>
        <div className="flex flex-1 justify-end items-center gap-4">
          <div className="hidden md:flex items-center gap-8">
            <a className="text-slate-700 dark:text-slate-300 hover:text-primary dark:hover:text-primary text-sm font-medium leading-normal" href="#">Dashboard</a>
            <a className="text-primary dark:text-primary text-sm font-bold leading-normal" href="#">Benefits</a>
            <a className="text-slate-700 dark:text-slate-300 hover:text-primary dark:hover:text-primary text-sm font-medium leading-normal" href="#">Profile</a>
            <a className="text-slate-700 dark:text-slate-300 hover:text-primary dark:hover:text-primary text-sm font-medium leading-normal" href="#">Help</a>
          </div>
          <div className="flex gap-2">
            <button className="flex max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-slate-200/50 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-2.5">
              <span className="material-symbols-outlined text-xl">notifications</span>
            </button>
            <button className="flex max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-slate-200/50 dark:bg-slate-800 text-slate-700 dark:text-slate-300 hover:bg-slate-200 dark:hover:bg-slate-700 gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-2.5">
              <span className="material-symbols-outlined text-xl">settings</span>
            </button>
          </div>
          <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" data-alt="User profile picture, a person smiling" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuA7CID6mb7Wi-AEfoAtcw4i6a99kxiCWXLnR3RJFBZ8u8VRi8k8nEWjJ8nOiSqidSFX0jt4lz99mIlq472pGdkeSLGxy28JUxM-8104uyZIPzXC4anx3YYukwRh5QtjZAiRtD4OHnC2sFhDbacmaxgKJpFnTlHBkZ4MLacOz06OttLn-Jvug7dsvlTOwnjf-JpsLJnPwdVEJVkqdnhq4EMifxK1yYzjWnfQkVnkTOYMGtuytkWxUkUjKK_f2nreEXF0qSFzszKAPh3j")'}}></div>
        </div>
      </header>
      <main className="flex-1 w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 md:py-12">
        <div className="flex flex-col lg:flex-row gap-8">
          {/* Left Column: Main Content */}
          <div className="w-full lg:w-2/3 flex flex-col gap-6">
            {/* Page Heading & Progress Bar */}
            <div className="flex flex-col gap-4">
              <div className="flex flex-wrap justify-between gap-3">
                <div className="flex min-w-72 flex-col gap-2">
                  <p className="text-slate-900 dark:text-white text-4xl font-black leading-tight tracking-[-0.033em]">Review and Submit Your Elections</p>
                  <p className="text-slate-500 dark:text-slate-400 text-base font-normal leading-normal">Please review your selections carefully before final submission.</p>
                </div>
              </div>
              <div className="flex flex-col gap-3">
                <div className="flex gap-6 justify-between"><p className="text-slate-900 dark:text-white text-base font-medium leading-normal">Step 4 of 4: Complete</p></div>
                <div className="rounded-full bg-slate-200 dark:bg-slate-800"><div className="h-2 rounded-full bg-primary" style={{width: '100%'}}></div></div>
              </div>
            </div>
            {/* Alert Banner */}
            <div className="flex items-start gap-3 p-4 rounded-lg bg-yellow-100 dark:bg-yellow-900/30 border border-yellow-300 dark:border-yellow-800">
              <span className="material-symbols-outlined text-yellow-600 dark:text-yellow-400 mt-0.5">warning</span>
              <div className="flex flex-col">
                <p className="font-bold text-yellow-800 dark:text-yellow-300">Attention Required</p>
                <p className="text-sm text-yellow-700 dark:text-yellow-400">You have not elected dental coverage for your dependents. Please ensure this is correct before submitting.</p>
              </div>
            </div>
            {/* Accordions for Benefits */}
            <div className="flex flex-col gap-3">
              <details className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-5 group" open="">
                <summary className="flex cursor-pointer items-center justify-between gap-6 py-4">
                  <p className="text-slate-900 dark:text-white text-lg font-bold">Medical Plan</p>
                  <span className="material-symbols-outlined text-slate-500 dark:text-slate-400 group-open:rotate-180 transition-transform">expand_more</span>
                </summary>
                <div className="pb-4 border-t border-slate-200 dark:border-slate-800">
                  <div className="text-slate-600 dark:text-slate-400 text-sm space-y-3 pt-4">
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Plan Name:</span><span>PPO Gold Plan</span></div>
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Coverage Level:</span><span>Employee + Family</span></div>
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Covered Dependents:</span><span>Jane Doe, John Doe Jr.</span></div>
                    <div className="flex justify-between border-t border-slate-200 dark:border-slate-800 mt-3 pt-3"><span className="font-bold text-slate-800 dark:text-slate-200">Per Pay Period Cost:</span><span className="font-bold text-slate-900 dark:text-white">$250.00</span></div>
                  </div>
                </div>
              </details>
              <details className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-5 group">
                <summary className="flex cursor-pointer items-center justify-between gap-6 py-4">
                  <p className="text-slate-900 dark:text-white text-lg font-bold">Dental Plan</p>
                  <span className="material-symbols-outlined text-slate-500 dark:text-slate-400 group-open:rotate-180 transition-transform">expand_more</span>
                </summary>
                <div className="pb-4 border-t border-slate-200 dark:border-slate-800">
                  <div className="text-slate-600 dark:text-slate-400 text-sm space-y-3 pt-4">
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Status:</span><span className="font-bold text-yellow-600 dark:text-yellow-400">Waived</span></div>
                    <p className="text-xs text-center pt-2">You have chosen not to enroll in this benefit.</p>
                  </div>
                </div>
              </details>
              <details className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-5 group" open="">
                <summary className="flex cursor-pointer items-center justify-between gap-6 py-4">
                  <p className="text-slate-900 dark:text-white text-lg font-bold">Vision Plan</p>
                  <span className="material-symbols-outlined text-slate-500 dark:text-slate-400 group-open:rotate-180 transition-transform">expand_more</span>
                </summary>
                <div className="pb-4 border-t border-slate-200 dark:border-slate-800">
                  <div className="text-slate-600 dark:text-slate-400 text-sm space-y-3 pt-4">
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Plan Name:</span><span>Vision Standard</span></div>
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Coverage Level:</span><span>Employee Only</span></div>
                    <div className="flex justify-between border-t border-slate-200 dark:border-slate-800 mt-3 pt-3"><span className="font-bold text-slate-800 dark:text-slate-200">Per Pay Period Cost:</span><span className="font-bold text-slate-900 dark:text-white">$15.50</span></div>
                  </div>
                </div>
              </details>
              <details className="flex flex-col rounded-xl border border-slate-200 dark:border-slate-800 bg-white dark:bg-slate-900/50 px-5 group">
                <summary className="flex cursor-pointer items-center justify-between gap-6 py-4">
                  <p className="text-slate-900 dark:text-white text-lg font-bold">Additional Benefits</p>
                  <span className="material-symbols-outlined text-slate-500 dark:text-slate-400 group-open:rotate-180 transition-transform">expand_more</span>
                </summary>
                <div className="pb-4 border-t border-slate-200 dark:border-slate-800">
                  <div className="text-slate-600 dark:text-slate-400 text-sm space-y-3 pt-4">
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Life Insurance:</span><span>$50,000 Coverage</span></div>
                    <div className="flex justify-between"><span className="font-medium text-slate-700 dark:text-slate-300">Short-Term Disability:</span><span>Enrolled</span></div>
                    <div className="flex justify-between border-t border-slate-200 dark:border-slate-800 mt-3 pt-3"><span className="font-bold text-slate-800 dark:text-slate-200">Per Pay Period Cost:</span><span className="font-bold text-slate-900 dark:text-white">$22.75</span></div>
                  </div>
                </div>
              </details>
            </div>
          </div>
          {/* Right Column: Sticky Summary */}
          <div className="w-full lg:w-1/3">
            <div className="sticky top-24 flex flex-col gap-6">
              <div className="bg-white dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-xl p-6">
                <h3 className="text-xl font-bold text-slate-900 dark:text-white mb-4">Your Cost Summary</h3>
                <div className="space-y-4">
                  <div className="flex flex-col gap-2 bg-slate-100 dark:bg-slate-800/50 p-4 rounded-lg">
                    <p className="text-sm text-slate-500 dark:text-slate-400">Total Per Pay Period</p>
                    <p className="text-4xl font-black text-slate-900 dark:text-white">$288.25</p>
                  </div>
                  <div className="text-sm space-y-2 pt-2">
                    <div className="flex justify-between">
                      <span className="text-slate-600 dark:text-slate-400">Your Contribution:</span>
                      <span className="font-medium text-slate-800 dark:text-slate-200">$288.25</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-slate-600 dark:text-slate-400">Employer Contribution:</span>
                      <span className="font-medium text-success dark:text-green-400">$650.00</span>
                    </div>
                  </div>
                  <a className="text-sm font-medium text-primary hover:underline" href="#">View Detailed Cost Breakdown</a>
                </div>
              </div>
              <div className="bg-white dark:bg-slate-900/50 border border-slate-200 dark:border-slate-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex flex-col gap-3">
                  <label className="flex items-start gap-3">
                    <input className="mt-1 h-5 w-5 shrink-0 rounded border-slate-300 dark:border-slate-600 bg-transparent text-primary checked:bg-primary checked:border-primary focus:ring-2 focus:ring-primary/50 focus:ring-offset-0 focus:ring-offset-background-light dark:focus:ring-offset-background-dark" type="checkbox"/>
                    <p className="text-slate-700 dark:text-slate-300 text-sm font-normal leading-normal">I have reviewed my elections and confirm they are correct. I understand these elections will remain in effect for the entire plan year unless I experience a qualifying life event.</p>
                  </label>
                </div>
                <div className="flex flex-col gap-3">
                  <button className="w-full flex items-center justify-center rounded-lg h-12 px-6 bg-primary text-white text-base font-bold hover:bg-primary/90 disabled:bg-slate-300 dark:disabled:bg-slate-700 disabled:cursor-not-allowed">Submit My Elections</button>
                  <button className="w-full flex items-center justify-center rounded-lg h-12 px-6 bg-transparent text-primary font-bold border-2 border-primary hover:bg-primary/10">Back to Make Changes</button>
                </div>
                <a className="text-sm text-center font-medium text-slate-500 dark:text-slate-400 hover:text-primary dark:hover:text-primary hover:underline mt-2" href="#">Print Summary</a>
              </div>
            </div>
          </div>
        </div>
      </main>
    </div>
  );
};

export default ReviewAndSubmitPage;