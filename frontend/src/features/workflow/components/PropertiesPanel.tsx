import React from 'react';
import { Node } from 'reactflow';

interface PropertiesPanelProps {
  selectedNode: Node | null;
}

export const PropertiesPanel: React.FC<PropertiesPanelProps> = ({ selectedNode }) => {
  if (!selectedNode) {
    return (
      <div className="flex h-full items-center justify-center p-4 text-center text-gray-500 dark:text-gray-400">
        <p>Select a node to view properties</p>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-white dark:bg-[#18232f]">
      <div className="p-4 border-b border-gray-200 dark:border-gray-800">
        <p className="text-lg font-bold text-gray-900 dark:text-white">Properties</p>
        <p className="text-sm text-gray-500 dark:text-gray-400">Selected: {selectedNode.data.label}</p>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300" htmlFor="node-name">Name</label>
            <input className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" id="node-name" type="text" defaultValue={selectedNode.data.label} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300" htmlFor="node-type">Type</label>
            <input className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" id="node-type" readOnly type="text" value={selectedNode.type} />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300" htmlFor="activity-ref">Activity Reference</label>
            <input className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" id="activity-ref" placeholder="e.g., temporal.approveRequest" type="text" defaultValue={selectedNode.data.activityRef || 'temporal.managerApproval'} />
          </div>
        </div>
        <div>
          <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3">ROLES & ENTITLEMENTS</h3>
          <div className="space-y-2">
            <input className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" placeholder="Add role..." type="text" defaultValue={selectedNode.data.role || 'Manager'} />
          </div>
        </div>
        <div>
          <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3">SLA & ESCALATIONS</h3>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300" htmlFor="sla-duration">SLA Duration</label>
              <input className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" id="sla-duration" type="text" defaultValue={selectedNode.data.sla || '24h'} />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-gray-700 dark:text-gray-300" htmlFor="escalation-rules">Escalation Rules</label>
              <textarea className="w-full rounded-md border-gray-300 dark:border-gray-700 bg-background-light dark:bg-background-dark text-sm focus:ring-primary focus:border-primary text-gray-900 dark:text-white" id="escalation-rules" rows={2} defaultValue={selectedNode.data.escalation || "Notify Director if SLA is breached by 4 hours."}></textarea>
            </div>
          </div>
        </div>
        <div>
          <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3">POLICY REFERENCES</h3>
          <div className="space-y-2">
            <div className="flex items-center justify-between rounded-md p-3 bg-background-light dark:bg-background-dark border border-gray-200 dark:border-gray-700">
              <div className="flex items-center gap-3">
                <span className="material-symbols-outlined text-gray-500 dark:text-gray-400">gavel</span>
                <span className="text-sm font-medium text-gray-900 dark:text-white">Corp Approval Policy v2.1</span>
              </div>
              <a className="text-primary text-sm font-semibold hover:underline" href="#">View</a>
            </div>
          </div>
        </div>
        <div>
          <h3 className="text-sm font-semibold text-gray-500 dark:text-gray-400 mb-3">AUDIT OPTIONS</h3>
          <div className="space-y-3">
            <div className="flex items-center justify-between rounded-md p-3 bg-background-light dark:bg-background-dark border border-gray-200 dark:border-gray-700">
              <label className="text-sm font-medium text-gray-900 dark:text-white" htmlFor="hash-chaining">Hash Chaining</label>
              <button aria-checked="true" className="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent bg-primary transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-white dark:focus:ring-offset-[#18232f]" id="hash-chaining" role="switch" type="button">
                <span className="pointer-events-none inline-block h-5 w-5 translate-x-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"></span>
              </button>
            </div>
            <div className="flex items-center justify-between rounded-md p-3 bg-background-light dark:bg-background-dark border border-gray-200 dark:border-gray-700">
              <label className="text-sm font-medium text-gray-900 dark:text-white" htmlFor="digital-signature">Digital Signature</label>
              <button aria-checked="false" className="relative inline-flex h-6 w-11 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent bg-gray-200 dark:bg-gray-600 transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 focus:ring-offset-white dark:focus:ring-offset-[#18232f]" id="digital-signature" role="switch" type="button">
                <span className="pointer-events-none inline-block h-5 w-5 translate-x-0 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"></span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
