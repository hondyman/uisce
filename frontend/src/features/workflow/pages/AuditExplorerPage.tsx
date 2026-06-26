import React, { useState } from 'react';

export const AuditExplorerPage: React.FC = () => {
  const [selectedEventId, setSelectedEventId] = useState<string>('3e4f-5g6h-7i8j');

  const events = [
    {
      id: '3e4f-5g6h-7i8j',
      type: 'Policy Update',
      narrative: "Policy 'auth-v2' updated",
      actor: 'Ciara@system.com',
      timestamp: '2023-10-27 10:05:00 UTC',
      prevHash: 'a1b2-c3d4-e5f6',
      hash: 'b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9',
      signature: '3045022100e4...',
      roles: ['Admin', 'PolicyEditor'],
      policyRefs: 'pol_1a2b3c',
      selected: true,
    },
    {
      id: 'a1b2-c3d4-e5f6',
      type: 'User Login',
      narrative: 'User login successful from IP 192.168.1.1',
      actor: 'admin@system.com',
      timestamp: '2023-10-27 10:00:00 UTC',
      prevHash: 'x7y8-z9a0-b1c2',
      hash: 'c8e9...f1a2',
      signature: '...',
      roles: ['SuperAdmin'],
      policyRefs: 'pol_session_mngmt',
      selected: false,
    },
    {
      id: 'x7y8-z9a0-b1c2',
      type: 'Workflow Create',
      narrative: "Created new workflow 'onboarding-v3'",
      actor: 'dev-ops@service.acc',
      timestamp: '2023-10-27 09:55:15 UTC',
      prevHash: 'NULL',
      hash: 'd3f4...a5b6',
      signature: '...',
      roles: ['DevOps'],
      policyRefs: 'pol_workflow_lifecycle',
      selected: false,
    },
  ];

  const selectedEvent = events.find(e => e.id === selectedEventId) || events[0];

  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-background-dark text-text-light dark:text-text-dark">
      {/* SideNavBar */}
      <aside className="flex h-full flex-col justify-between bg-white dark:bg-[#1a2530] p-4 border-r border-gray-200 dark:border-[#283039] w-64 shrink-0">
        <div className="flex flex-col gap-4">
          <div className="flex gap-3 items-center">
            <div className="bg-center bg-no-repeat aspect-square bg-cover rounded-full size-10" style={{ backgroundImage: 'url("https://lh3.googleusercontent.com/aida-public/AB6AXuCOlT0a5iaWHr6jpnNxv2MVf4Vee5lE86TBstmSPWgfV17YoPOGG3X7DXCFXS8qREjL8xyYiAee4WiqAsbgJbGte85W1sHApKJSceivyXj1aO9wxLmH3dtbNXG9YjRZZt0bFqUBl9RQrOLfI-uZkWPf5Y9dD632JM9Yy_QGsVdcpPAyR8HqbxjZTZAqbWXAJMoA7DgxpLiVHjk9gcSQnns7f7HHNDr8bzCrQfTunMw4vvhmSY49hPU0GkHawfhu-6FBKiQew8O3UUOA")' }}></div>
            <div className="flex flex-col">
              <h1 className="text-gray-900 dark:text-white text-base font-medium leading-normal">Temporal</h1>
              <p className="text-gray-500 dark:text-[#9dabb9] text-sm font-normal leading-normal">Business Suite</p>
            </div>
          </div>
          <div className="flex flex-col gap-2 mt-4">
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-primary/10 text-gray-500 dark:text-[#9dabb9]" href="/core/regulator-portal">
              <span className="material-symbols-outlined">dashboard</span>
              <p className="text-sm font-medium leading-normal">Dashboard</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-primary/10 text-primary" href="#">
              <span className="material-symbols-outlined !fill-1">shield</span>
              <p className="text-sm font-bold leading-normal">Audit Explorer</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-primary/10 text-gray-500 dark:text-[#9dabb9]" href="#">
              <span className="material-symbols-outlined">settings</span>
              <p className="text-sm font-medium leading-normal">Settings</p>
            </a>
          </div>
        </div>
        <div className="flex flex-col gap-4">
          <button className="flex w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em]">
            <span className="truncate">New Workflow</span>
          </button>
          <div className="flex flex-col gap-1">
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-primary/10 text-gray-500 dark:text-[#9dabb9]" href="#">
              <span className="material-symbols-outlined">help_center</span>
              <p className="text-sm font-medium leading-normal">Help</p>
            </a>
            <a className="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-primary/10 text-gray-500 dark:text-[#9dabb9]" href="#">
              <span className="material-symbols-outlined">logout</span>
              <p className="text-sm font-medium leading-normal">Logout</p>
            </a>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <main className="flex flex-1 flex-col h-screen overflow-hidden">
        {/* Page Heading */}
        <header className="flex flex-wrap justify-between gap-4 p-6 border-b border-gray-200 dark:border-[#283039] bg-white dark:bg-[#1a2530] items-center shrink-0">
          <div className="flex min-w-72 flex-col gap-2">
            <p className="text-gray-900 dark:text-white text-3xl font-black leading-tight tracking-tight">Audit Explorer</p>
            <p className="text-gray-500 dark:text-[#9dabb9] text-base font-normal leading-normal">Visualize, validate, and export the immutable audit trail.</p>
          </div>
          <div className="flex items-center gap-3">
            <button className="flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white gap-2 pl-3 text-sm font-bold leading-normal tracking-[0.015em]">
              <span className="material-symbols-outlined text-base">shield</span>
              <span className="truncate">Chain Validation</span>
            </button>
            <button className="flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-background-light dark:bg-[#1a2530] border border-gray-200 dark:border-[#283039] text-gray-900 dark:text-white gap-2 text-sm font-bold leading-normal tracking-[0.015em]">
              <span className="material-symbols-outlined text-base">download</span>
              <span className="truncate">Export</span>
            </button>
          </div>
        </header>

        {/* Two-Panel Layout */}
        <div className="flex flex-1 overflow-hidden">
          {/* Left Panel: Event List */}
          <div className="w-1/3 flex flex-col border-r border-gray-200 dark:border-[#283039] bg-white dark:bg-[#1a2530]">
            <div className="p-4 border-b border-gray-200 dark:border-[#283039] shrink-0">
              <label className="flex flex-col min-w-40 h-11 w-full">
                <div className="flex w-full flex-1 items-stretch rounded-lg h-full border border-gray-200 dark:border-[#283039] bg-background-light dark:bg-background-dark">
                  <div className="text-gray-500 dark:text-[#9dabb9] flex items-center justify-center pl-4 rounded-l-lg border-r-0">
                    <span className="material-symbols-outlined">search</span>
                  </div>
                  <input 
                    className="flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-gray-900 dark:text-white focus:outline-0 focus:ring-0 border-none bg-transparent h-full placeholder:text-gray-500 placeholder:dark:text-[#9dabb9] px-4 rounded-l-none border-l-0 pl-2 text-base font-normal leading-normal" 
                    placeholder="Search events..." 
                  />
                </div>
              </label>
            </div>
            <div className="flex-1 overflow-y-auto">
              {events.map((event) => (
                <div 
                  key={event.id}
                  onClick={() => setSelectedEventId(event.id)}
                  className={`flex gap-4 px-4 py-3 justify-between border-l-4 cursor-pointer ${selectedEventId === event.id ? 'bg-primary/10 border-primary' : 'hover:bg-primary/5 border-transparent'}`}
                >
                  <div className="flex items-start gap-4">
                    <div className={`flex items-center justify-center rounded-lg shrink-0 size-12 ${selectedEventId === event.id ? 'text-primary bg-primary/20' : 'text-gray-500 dark:text-[#9dabb9] bg-background-light dark:bg-background-dark'}`}>
                      <span className="material-symbols-outlined">square</span>
                    </div>
                    <div className="flex flex-1 flex-col justify-center">
                      <p className={`text-base font-medium leading-normal ${selectedEventId === event.id ? 'text-primary font-bold' : 'text-gray-900 dark:text-white'}`}>Event ID: {event.id}</p>
                      <p className="text-gray-500 dark:text-[#9dabb9] text-sm font-normal leading-normal truncate w-48">{event.narrative}</p>
                      <p className="text-gray-500 dark:text-[#9dabb9] text-sm font-normal leading-normal">Actor: {event.actor}</p>
                    </div>
                  </div>
                  <div className="shrink-0"><p className="text-gray-500 dark:text-[#9dabb9] text-xs font-normal leading-normal">{event.timestamp.split(' ')[1]}</p></div>
                </div>
              ))}
            </div>
          </div>

          {/* Right Panel: Detail Inspector */}
          <div className="w-2/3 flex flex-col p-6 overflow-y-auto bg-background-light dark:bg-background-dark">
            <h2 className="text-2xl font-bold mb-4 text-gray-900 dark:text-white">Event Details</h2>
            <div className="grid grid-cols-2 gap-x-8 gap-y-4">
              
              {/* Key/Value Pairs */}
              <div className="col-span-2 space-y-3 p-4 bg-white dark:bg-[#1a2530] rounded-xl border border-gray-200 dark:border-[#283039]">
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Event ID</p>
                  <div className="flex items-center gap-2">
                    <p className="font-mono text-sm text-gray-900 dark:text-white">{selectedEvent.id}</p>
                    <button className="text-gray-500 dark:text-[#9dabb9] hover:text-primary"><span className="material-symbols-outlined text-base">content_copy</span></button>
                  </div>
                </div>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Previous Event ID</p>
                  <div className="flex items-center gap-2">
                    <p className="font-mono text-sm text-gray-900 dark:text-white">{selectedEvent.prevHash}</p>
                    <button className="text-gray-500 dark:text-[#9dabb9] hover:text-primary"><span className="material-symbols-outlined text-base">content_copy</span></button>
                  </div>
                </div>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Timestamp</p>
                  <p className="text-sm text-gray-900 dark:text-white">{selectedEvent.timestamp}</p>
                </div>
              </div>

              <div className="col-span-1 space-y-3 p-4 bg-white dark:bg-[#1a2530] rounded-xl border border-gray-200 dark:border-[#283039]">
                <h3 className="font-bold text-lg mb-2 text-gray-900 dark:text-white">Actor</h3>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">User/Service ID</p>
                  <p className="text-sm text-gray-900 dark:text-white">{selectedEvent.actor}</p>
                </div>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Roles</p>
                  <div className="flex gap-1">
                    {selectedEvent.roles.map(role => (
                      <span key={role} className="bg-primary/20 text-primary text-xs font-semibold px-2 py-1 rounded-full">{role}</span>
                    ))}
                  </div>
                </div>
              </div>

              <div className="col-span-1 space-y-3 p-4 bg-white dark:bg-[#1a2530] rounded-xl border border-gray-200 dark:border-[#283039]">
                <h3 className="font-bold text-lg mb-2 text-gray-900 dark:text-white">Action</h3>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Narrative</p>
                  <p className="text-sm text-gray-900 dark:text-white truncate w-40 text-right" title={selectedEvent.narrative}>{selectedEvent.narrative}</p>
                </div>
                <div className="flex justify-between items-center">
                  <p className="text-sm font-medium text-gray-500 dark:text-[#9dabb9]">Policy Refs</p>
                  <p className="text-sm font-mono text-gray-900 dark:text-white">{selectedEvent.policyRefs}</p>
                </div>
              </div>

              {/* JSON Viewer for Payload Digest */}
              <div className="col-span-2">
                <h3 className="font-bold text-lg mb-2 text-gray-900 dark:text-white">Payload Digest</h3>
                <div className="bg-white dark:bg-[#1a2530] rounded-xl border border-gray-200 dark:border-[#283039] p-4 font-mono text-sm text-gray-500 dark:text-[#9dabb9] relative overflow-hidden group">
                  <button className="absolute top-3 right-3 text-gray-500 dark:text-[#9dabb9] hover:text-primary opacity-0 group-hover:opacity-100 transition-opacity"><span className="material-symbols-outlined text-base">content_copy</span></button>
                  <pre className="overflow-x-auto"><code>{`{
  "algorithm": "sha256",
  "value": "${selectedEvent.hash}"
}`}</code></pre>
                </div>
              </div>

              {/* JSON Viewer for Signature */}
              <div className="col-span-2">
                <h3 className="font-bold text-lg mb-2 text-gray-900 dark:text-white">Digital Signature</h3>
                <div className="bg-white dark:bg-[#1a2530] rounded-xl border border-gray-200 dark:border-[#283039] p-4 font-mono text-sm text-gray-500 dark:text-[#9dabb9] relative overflow-hidden group">
                  <button className="absolute top-3 right-3 text-gray-500 dark:text-[#9dabb9] hover:text-primary opacity-0 group-hover:opacity-100 transition-opacity"><span className="material-symbols-outlined text-base">content_copy</span></button>
                  <pre className="overflow-x-auto"><code>{`{
  "publicKey": "key-pub-12345...",
  "signature": "${selectedEvent.signature}",
  "algorithm": "ecdsa-p256"
}`}</code></pre>
                </div>
              </div>

            </div>
          </div>
        </div>
      </main>
    </div>
  );
};
