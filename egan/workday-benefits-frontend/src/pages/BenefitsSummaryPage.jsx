import React from 'react';

const BenefitsSummaryPage = () => {
  return (
    <div className="relative flex h-auto min-h-screen w-full flex-col group/design-root overflow-x-hidden">
      <header className="flex items-center justify-between whitespace-nowrap border-b border-solid border-b-gray-200 dark:border-b-gray-700 px-6 py-3 bg-white dark:bg-gray-800 sticky top-0 z-50">
        <div className="flex items-center gap-4 text-gray-900 dark:text-white">
          <span className="material-symbols-outlined text-primary text-3xl">hub</span>
          <h2 className="text-lg font-bold leading-tight tracking-[-0.015em]">Workday Benefits</h2>
        </div>
        <div className="flex flex-1 justify-end gap-6 items-center">
          <label className="flex flex-col min-w-40 !h-10 max-w-64">
            <div className="flex w-full flex-1 items-stretch rounded-lg h-full">
              <div className="text-gray-500 dark:text-gray-400 flex border-none bg-background-light dark:bg-background-dark items-center justify-center pl-4 rounded-l-lg border-r-0">
                <span className="material-symbols-outlined">search</span>
              </div>
              <input className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-gray-900 dark:text-white focus:outline-0 focus:ring-0 border-none bg-background-light dark:bg-background-dark focus:border-none h-full placeholder:text-gray-500 dark:placeholder:text-gray-400 px-4 rounded-l-none border-l-0 pl-2 text-base font-normal leading-normal" placeholder="Search" value=""/>
            </div>
          </label>
          <button className="flex max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-background-light dark:bg-background-dark text-gray-900 dark:text-white gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-2.5">
            <div className="relative">
              <span className="material-symbols-outlined text-2xl">notifications</span>
              <div className="absolute top-0 right-0 size-2 bg-red-500 rounded-full border-2 border-background-light dark:border-background-dark"></div>
            </div>
          </button>
          <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" data-alt="User avatar of a person smiling" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuA0PeX8mtXlLE2ZO68lLseERgmkyffgY2t4_cICEj_-ukW2Nu5cu1QGIb84jlYnKjX7sC71nHN83fREQc0XtbzPjjuHKqVU8B5bz-K4IUJTIFhWnujhrrnv8qAm5gvSY8X_Pc-pcIpO3v13LURuhTRo983020PLnbNxiPcJblpMP9JE5uJU5GaY2kBSylJCCyAWPzeNMBCRPPpfg2wMZlGNU2HpJAjR6gTYXVSw4Sbb8ti36IxwE94WDP1NtPmfJG4pUWkON3TjIYAq")'}}></div>
        </div>
      </header>
      <div className="flex flex-1">
        {/* Side Navigation */}
        <aside className="w-64 flex-shrink-0 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 p-4">
          <div className="flex h-full flex-col justify-between">
            <div className="flex flex-col gap-4">
              <div className="flex items-center gap-3">
                <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" data-alt="User avatar of a person smiling" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuACCGnjDI9xbz6i-tTseIYA48Cq6alfwtqGgHtewdOvfeJhxAqtFCLCmqYKRyvH10bI3suXPIt3VSYFFsvMw1POlN9TU7-R6M13DX6i3Xa20RqjUPz7Y8jIM5VyLuwjgHkdsAv0aPS8RYnqwgseJlSX9hb4hsm2TxZRGy4QASgK_I5YfbjRxPAs-gFRw_e_i3St046SUAZhOVTBYuAjaXZNfxOGPQphhszHnpqrTSqbjzUooLyEe1_wqihu6WcaBr7hiweo6opUHXHF")'}}></div>
                <div className="flex flex-col">
                  <h1 className="text-gray-900 dark:text-white text-base font-medium leading-normal">Welcome, John!</h1>
                  <p className="text-gray-500 dark:text-gray-400 text-sm font-normal leading-normal">john.doe@workday.com</p>
                </div>
              </div>
              <div className="flex flex-col gap-2 mt-4">
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/10 dark:bg-primary/20 text-primary dark:text-white" href="#">
                  <span className="material-symbols-outlined">dashboard</span>
                  <p className="text-sm font-medium leading-normal">Dashboard</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                  <span className="material-symbols-outlined">edit_document</span>
                  <p className="text-sm font-medium leading-normal">Enrollment</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                  <span className="material-symbols-outlined">group</span>
                  <p className="text-sm font-medium leading-normal">Dependents</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                  <span className="material-symbols-outlined">event</span>
                  <p className="text-sm font-medium leading-normal">Life Events</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                  <span className="material-symbols-outlined">description</span>
                  <p className="text-sm font-medium leading-normal">Documents</p>
                </a>
              </div>
            </div>
            <div className="flex flex-col gap-2">
              <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                <span className="material-symbols-outlined">settings</span>
                <p className="text-sm font-medium leading-normal">Settings</p>
              </a>
              <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-300" href="#">
                <span className="material-symbols-outlined">logout</span>
                <p className="text-sm font-medium leading-normal">Logout</p>
              </a>
            </div>
          </div>
        </aside>
        {/* Main Content */}
        <main className="flex-1 p-8 overflow-y-auto">
          <div className="max-w-7xl mx-auto">
            <div className="flex flex-wrap justify-between gap-3 items-center mb-6">
              <p className="text-gray-900 dark:text-white text-4xl font-black leading-tight tracking-[-0.033em] min-w-72">My Benefits Summary</p>
            </div>
            <div className="bg-white dark:bg-gray-800 p-4 rounded-xl @container mb-8">
              <div className="flex flex-col items-stretch justify-start rounded-lg @xl:flex-row @xl:items-center">
                <div className="w-full @xl:w-1/3 bg-center bg-no-repeat aspect-video bg-cover rounded-lg" data-alt="Abstract image of a family enjoying outdoors, representing well-being" style={{backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuCF3R1tziMBZvLKgCR46yM3PE4LiOnvqM08TN2UEbaDAmS3sDtl77Lflbjx6BcknCOvO3Gx5Mw8AF8U9cz11U5qt0fSmI_f2yYbLGO60hxFDkoUtJII1I8t5dz4EhQQ7eNefjx8-7_mrOpDJX8vX4fneWSmoNNi74lxUrUKbWOJG9v9E1RJdGUBg69Wm6o6Qfz4h-VwSumYLbUcmlLPS-RM45m7Jk-K9kHfkAjsD9hgy3aZz7poqocCUV3DYU6TA6K8W-_Z4B4_aao0")'}}></div>
                <div className="flex w-full min-w-72 grow flex-col items-stretch justify-center gap-2 py-4 @xl:px-6">
                  <p className="text-primary dark:text-blue-400 text-sm font-bold leading-normal">Open Enrollment is Now Active!</p>
                  <p className="text-gray-900 dark:text-white text-lg font-bold leading-tight tracking-[-0.015em]">Review your benefits and make your selections before the deadline: Dec 15, 2023.</p>
                  <div className="flex items-center gap-3 justify-start mt-2">
                    <button className="flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-6 bg-primary text-white text-sm font-medium leading-normal">
                      <span className="truncate">Start Enrollment</span>
                    </button>
                    <button className="flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-6 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white text-sm font-medium leading-normal">
                      <span className="truncate">View Plan Comparison</span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <h2 className="text-gray-900 dark:text-white text-[22px] font-bold leading-tight tracking-[-0.015em] px-4 pb-3 pt-5">My Current Plans</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
              {/* Benefit Cards */}
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-blue-100 dark:bg-blue-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-blue-600 dark:text-blue-400 text-2xl">local_hospital</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">Medical</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">PPO Gold Plan</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$150.00 <span className="text-base font-normal text-gray-500 dark:text-gray-400">/ per paycheck</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-green-500 text-sm font-medium bg-green-100 dark:bg-green-900/50 px-2 py-1 rounded-full">Enrolled</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal">
                  <span className="truncate">Manage Plan</span>
                </button>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-teal-100 dark:bg-teal-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-teal-600 dark:text-teal-400 text-2xl">dentistry</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">Dental</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">Preventive Care Plan</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$25.50 <span className="text-base font-normal text-gray-500 dark:text-gray-400">/ per paycheck</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-yellow-600 dark:text-yellow-400 text-sm font-medium bg-yellow-100 dark:bg-yellow-900/50 px-2 py-1 rounded-full">Action Required</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal">
                  <span className="truncate">Complete Enrollment</span>
                </button>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-purple-100 dark:bg-purple-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-purple-600 dark:text-purple-400 text-2xl">visibility</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">Vision</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">VSP Basic Plan</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$12.00 <span className="text-base font-normal text-gray-500 dark:text-gray-400">/ per paycheck</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-green-500 text-sm font-medium bg-green-100 dark:bg-green-900/50 px-2 py-1 rounded-full">Enrolled</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal">
                  <span className="truncate">Manage Plan</span>
                </button>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-green-100 dark:bg-green-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-green-600 dark:text-green-400 text-2xl">trending_up</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">401(k) Retirement</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">Contribution Rate: 6%</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$42,831.90 <span className="text-base font-normal text-gray-500 dark:text-gray-400">total balance</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-green-500 text-sm font-medium bg-green-100 dark:bg-green-900/50 px-2 py-1 rounded-full">Enrolled</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal">
                  <span className="truncate">Manage Investments</span>
                </button>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-pink-100 dark:bg-pink-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-pink-600 dark:text-pink-400 text-2xl">health_and_safety</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">Life Insurance</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">Coverage Amount: $250,000</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$15.00 <span className="text-base font-normal text-gray-500 dark:text-gray-400">/ per paycheck</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-green-500 text-sm font-medium bg-green-100 dark:bg-green-900/50 px-2 py-1 rounded-full">Enrolled</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-medium leading-normal">
                  <span className="truncate">View Beneficiaries</span>
                </button>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-xl p-6 flex flex-col gap-4">
                <div className="flex items-center gap-4">
                  <div className="bg-orange-100 dark:bg-orange-900/50 p-3 rounded-full">
                    <span className="material-symbols-outlined text-orange-600 dark:text-orange-400 text-2xl">savings</span>
                  </div>
                  <h3 className="text-lg font-bold text-gray-900 dark:text-white">Health Savings Account (HSA)</h3>
                </div>
                <div className="flex flex-col gap-2">
                  <p className="text-gray-600 dark:text-gray-300">Contribution Goal: $5,000</p>
                  <p className="text-gray-900 dark:text-white font-semibold text-2xl">$1,245.30 <span className="text-base font-normal text-gray-500 dark:text-gray-400">current balance</span></p>
                  <div className="flex items-center gap-2">
                    <span className="text-gray-500 text-sm font-medium bg-gray-200 dark:bg-gray-700 px-2 py-1 rounded-full">Waived</span>
                  </div>
                </div>
                <button className="mt-2 w-full flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white text-sm font-medium leading-normal">
                  <span className="truncate">Enroll Now</span>
                </button>
              </div>
            </div>
          </div>
        </main>
      </div>
    </div>
  );
};

export default BenefitsSummaryPage;