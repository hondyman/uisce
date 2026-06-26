import React, { useState } from 'react';

export const BusinessObjectExplorerPage: React.FC = () => {
  const [selectedObject, setSelectedObject] = useState<string | null>(null);

  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-background-dark text-text-light-primary dark:text-text-dark-primary">
      {/* Side Navigation Bar (Reused from Regulator Dashboard for consistency) */}
      <aside className="flex w-64 flex-col bg-white/5 dark:bg-background-dark border-r border-white/10 dark:border-white/10">
        <div className="flex h-full flex-col justify-between p-4">
          <div className="flex flex-col gap-6">
            <div className="flex items-center gap-3 p-2">
              <span className="material-symbols-outlined text-primary text-3xl">verified_user</span>
              <h1 className="text-white text-xl font-bold">Regulator</h1>
            </div>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/regulator-portal">
                  <span className="material-symbols-outlined">dashboard</span>
                  <p className="text-sm font-medium leading-normal">Dashboard</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/process-catalog">
                  <span className="material-symbols-outlined">menu_book</span>
                  <p className="text-sm font-medium leading-normal">Process Catalog</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/20 text-primary" href="#">
                  <span className="material-symbols-outlined">data_object</span>
                  <p className="text-sm font-medium leading-normal">Object Explorer</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/audit-explorer">
                  <span className="material-symbols-outlined">policy</span>
                  <p className="text-sm font-medium leading-normal">Audit Explorer</p>
                </a>
              </div>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 p-8 overflow-y-auto">
        <div className="flex flex-col gap-8">
          {/* Page Header */}
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            <div className="flex flex-col gap-1">
              <p className="text-white text-3xl font-bold leading-tight">Business Object Explorer</p>
              <p className="text-gray-400 text-sm font-normal leading-normal">
                Visualize lifecycle states and transitions for core business objects.
              </p>
            </div>
             <div className="flex items-center gap-4">
              <div className="relative">
                  <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">search</span>
                  <input
                    className="w-64 pl-10 pr-4 py-2 rounded-lg border border-white/20 bg-[#1C2127] text-white focus:ring-primary focus:border-primary placeholder:text-gray-500"
                    placeholder="Search objects (ID, Type)..."
                    type="text"
                  />
              </div>
            </div>
          </div>

          {/* Object List & Detail Split View */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 h-[calc(100vh-200px)]">
            
            {/* Left Column: Object List */}
            <div className="lg:col-span-1 flex flex-col gap-4 rounded-xl border border-white/10 bg-white/5 p-4 overflow-y-auto">
              <h3 className="text-lg font-bold text-white mb-2">Recent Objects</h3>
              
              {/* Mock Object Item 1 */}
              <div 
                className={`p-4 rounded-lg border cursor-pointer transition-colors ${selectedObject === 'ORD-2023-001' ? 'border-primary bg-primary/10' : 'border-white/10 bg-[#1C2127] hover:border-white/30'}`}
                onClick={() => setSelectedObject('ORD-2023-001')}
              >
                <div className="flex justify-between items-start mb-2">
                  <span className="font-mono text-sm text-primary font-bold">ORD-2023-001</span>
                  <span className="inline-flex items-center rounded-full bg-blue-900/30 px-2 py-0.5 text-xs font-medium text-blue-400">Order</span>
                </div>
                <p className="text-white text-sm font-medium">Purchase of IT Equipment</p>
                <div className="mt-3 flex items-center justify-between text-xs text-gray-400">
                  <span>State: <span className="text-white">Approval Pending</span></span>
                  <span>2m ago</span>
                </div>
              </div>

              {/* Mock Object Item 2 */}
              <div 
                className={`p-4 rounded-lg border cursor-pointer transition-colors ${selectedObject === 'JRN-2023-882' ? 'border-primary bg-primary/10' : 'border-white/10 bg-[#1C2127] hover:border-white/30'}`}
                onClick={() => setSelectedObject('JRN-2023-882')}
              >
                <div className="flex justify-between items-start mb-2">
                  <span className="font-mono text-sm text-primary font-bold">JRN-2023-882</span>
                  <span className="inline-flex items-center rounded-full bg-purple-900/30 px-2 py-0.5 text-xs font-medium text-purple-400">Journal</span>
                </div>
                <p className="text-white text-sm font-medium">Q3 Revenue Recognition</p>
                <div className="mt-3 flex items-center justify-between text-xs text-gray-400">
                  <span>State: <span className="text-green-400">Posted</span></span>
                  <span>1h ago</span>
                </div>
              </div>

               {/* Mock Object Item 3 */}
               <div 
                className={`p-4 rounded-lg border cursor-pointer transition-colors ${selectedObject === 'CTR-2023-105' ? 'border-primary bg-primary/10' : 'border-white/10 bg-[#1C2127] hover:border-white/30'}`}
                onClick={() => setSelectedObject('CTR-2023-105')}
              >
                <div className="flex justify-between items-start mb-2">
                  <span className="font-mono text-sm text-primary font-bold">CTR-2023-105</span>
                  <span className="inline-flex items-center rounded-full bg-orange-900/30 px-2 py-0.5 text-xs font-medium text-orange-400">Contract</span>
                </div>
                <p className="text-white text-sm font-medium">Vendor Agreement - Acme Corp</p>
                <div className="mt-3 flex items-center justify-between text-xs text-gray-400">
                  <span>State: <span className="text-yellow-400">Draft</span></span>
                  <span>4h ago</span>
                </div>
              </div>

            </div>

            {/* Right Column: Detail View */}
            <div className="lg:col-span-2 flex flex-col gap-6 rounded-xl border border-white/10 bg-white/5 p-6 overflow-y-auto">
              {selectedObject ? (
                <>
                  <div className="flex justify-between items-start border-b border-white/10 pb-4">
                    <div>
                      <h2 className="text-2xl font-bold text-white mb-1">{selectedObject}</h2>
                      <p className="text-gray-400 text-sm">Purchase of IT Equipment • Created by John Doe</p>
                    </div>
                    <div className="flex gap-2">
                         <button className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white bg-white/10 hover:bg-white/20">
                            <span className="material-symbols-outlined text-base">history</span> History
                        </button>
                        <button className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white bg-primary hover:bg-primary/90">
                            <span className="material-symbols-outlined text-base">edit</span> Edit
                        </button>
                    </div>
                  </div>

                  {/* Lifecycle Visualization */}
                  <div className="bg-[#1C2127] rounded-lg p-6 border border-white/10">
                    <h4 className="text-sm font-bold text-gray-400 uppercase mb-6">Lifecycle Progress</h4>
                    <div className="flex items-center justify-between relative">
                        {/* Connecting Line */}
                        <div className="absolute top-1/2 left-0 w-full h-0.5 bg-gray-700 -z-0 transform -translate-y-1/2"></div>
                        
                        {/* Step 1 */}
                        <div className="relative z-10 flex flex-col items-center gap-2">
                            <div className="w-8 h-8 rounded-full bg-green-500 flex items-center justify-center text-black font-bold">
                                <span className="material-symbols-outlined text-sm">check</span>
                            </div>
                            <span className="text-xs font-medium text-green-500">Draft</span>
                        </div>

                         {/* Step 2 */}
                         <div className="relative z-10 flex flex-col items-center gap-2">
                            <div className="w-8 h-8 rounded-full bg-green-500 flex items-center justify-center text-black font-bold">
                                <span className="material-symbols-outlined text-sm">check</span>
                            </div>
                            <span className="text-xs font-medium text-green-500">Review</span>
                        </div>

                         {/* Step 3 (Current) */}
                         <div className="relative z-10 flex flex-col items-center gap-2">
                            <div className="w-10 h-10 rounded-full bg-primary border-4 border-[#1C2127] flex items-center justify-center text-white shadow-[0_0_15px_rgba(43,140,238,0.5)]">
                                <span className="material-symbols-outlined text-base">pending</span>
                            </div>
                            <span className="text-sm font-bold text-white">Approval</span>
                        </div>

                         {/* Step 4 */}
                         <div className="relative z-10 flex flex-col items-center gap-2">
                            <div className="w-8 h-8 rounded-full bg-gray-700 border border-gray-600 flex items-center justify-center text-gray-400">
                                4
                            </div>
                            <span className="text-xs font-medium text-gray-500">Execution</span>
                        </div>

                         {/* Step 5 */}
                         <div className="relative z-10 flex flex-col items-center gap-2">
                            <div className="w-8 h-8 rounded-full bg-gray-700 border border-gray-600 flex items-center justify-center text-gray-400">
                                5
                            </div>
                            <span className="text-xs font-medium text-gray-500">Archived</span>
                        </div>
                    </div>
                  </div>

                  {/* Tabs / Details */}
                  <div className="flex flex-col gap-4">
                      <div className="flex border-b border-white/10">
                          <button className="px-4 py-2 text-sm font-medium text-primary border-b-2 border-primary">Narrative</button>
                          <button className="px-4 py-2 text-sm font-medium text-gray-400 hover:text-white">Compliance</button>
                          <button className="px-4 py-2 text-sm font-medium text-gray-400 hover:text-white">Related Objects</button>
                      </div>
                      
                      <div className="p-4 bg-[#1C2127] rounded-lg border border-white/10 text-sm text-gray-300 leading-relaxed">
                          <p className="mb-2"><strong className="text-white">AI Generated Narrative:</strong></p>
                          <p>
                              This order exceeds the standard departmental budget threshold of $5,000. 
                              It was flagged for <strong>Level 2 Approval</strong> due to the inclusion of specialized hardware (GPU Cluster). 
                              The vendor, <em>Acme Corp</em>, is a preferred supplier, which expedited the initial vetting stage.
                          </p>
                          <div className="mt-4 p-3 bg-blue-900/20 border border-blue-800/50 rounded flex gap-3 items-start">
                              <span className="material-symbols-outlined text-blue-400">info</span>
                              <div>
                                  <p className="text-blue-200 font-medium">Policy Note</p>
                                  <p className="text-blue-300/80 text-xs">Compliance Check <strong>POL-IT-004</strong> passed automatically based on vendor certification.</p>
                              </div>
                          </div>
                      </div>
                  </div>

                </>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-gray-500">
                    <span className="material-symbols-outlined text-6xl mb-4 opacity-20">data_object</span>
                    <p className="text-lg">Select an object to view details</p>
                </div>
              )}
            </div>

          </div>
        </div>
      </main>
    </div>
  );
};
