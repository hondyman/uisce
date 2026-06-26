import React from 'react'

const PropertiesPanel: React.FC = () => {
  return (
    <aside className="flex h-full w-96 flex-col border-l border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900/50">
      <div className="flex h-full flex-col">
        <div className="border-b border-gray-200 dark:border-gray-800 p-4">
          <h3 className="text-gray-900 dark:text-white text-lg font-bold leading-tight tracking-[-0.015em]">Properties: Activity - 'ProcessTrade'</h3>
        </div>
        <div className="flex-1 p-4 overflow-y-auto space-y-6">
          <div className="flex flex-col gap-4">
            <label className="flex flex-col">
              <p className="text-gray-900 dark:text-white text-sm font-medium leading-normal pb-2">Activity Name</p>
              <input className="form-input flex w-full min-w-0 flex-1 resize-none overflow-hidden rounded-lg text-gray-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 focus:border-primary h-10 placeholder:text-gray-400 dark:placeholder:text-gray-500 px-3 text-sm font-normal leading-normal" placeholder="e.g., ProcessTrade" defaultValue="ProcessTrade"/>
            </label>
            <label className="flex flex-col">
              <p className="text-gray-900 dark:text-white text-sm font-medium leading-normal pb-2">Input Payload</p>
              <textarea className="form-textarea flex w-full min-w-0 flex-1 resize-y overflow-hidden rounded-lg text-gray-900 dark:text-white focus:outline-0 focus:ring-2 focus:ring-primary/50 border border-gray-300 dark:border-gray-700 bg-white dark:bg-gray-800 focus:border-primary p-3 text-sm font-mono leading-normal" rows={6} defaultValue={`{
  "portfolioId": "12345-abc",
  "symbol": "ACME",
  "quantity": 100,
  "action": "BUY"
}`}></textarea>
            </label>
          </div>
        </div>
      </div>
    </aside>
  )
}

export default PropertiesPanel
