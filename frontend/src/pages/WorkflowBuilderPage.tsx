import React, { useState } from 'react'
import ExecutionMonitor from '../components/ExecutionMonitor'
import SideNavBar from '../components/SideNavBar'
import { devLog } from '../utils/devLogger';
import '../styles/WorkflowBuilder.css'
import SVGIcon from '../components/relationship/SVGIcon'

const Header: React.FC<{ onRunTest: () => Promise<void>; onDeploy: () => void }> = ({ onRunTest, onDeploy }) => {
  const nodeButtons = [
    { title: 'Start Node', icon: 'play_arrow', color: 'text-green-500', label: 'Start' },
    { title: 'Activity Node', icon: 'terminal', color: 'text-blue-500', label: 'Activity' },
    { title: 'Signal Node', icon: 'sensors', color: 'text-purple-500', label: 'Signal' },
    { title: 'Conditional Branch', icon: 'call_split', color: 'text-amber-500', label: 'Branch' },
    { title: 'End Node', icon: 'stop_circle', color: 'text-red-500', label: 'End' },
  ]

  return (
    <header className="flex justify-between items-center gap-4 px-6 py-3 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900/50">
      <div className="flex items-center gap-4">
        <div className="flex items-center border-r border-gray-200 dark:border-gray-700 pr-4">
          {nodeButtons.map((button) => (
            <button key={button.title} className="flex items-center justify-center gap-2 p-2 rounded-md hover:bg-gray-100 dark:hover:bg-gray-800 cursor-grab" title={button.title}>
              <SVGIcon name={button.icon} className={`${button.color}`} ariaLabel={button.title} />
              <span className="text-sm text-gray-800 dark:text-gray-200">{button.label}</span>
            </button>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button onClick={onRunTest} className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-4 hover:bg-gray-300 dark:hover:bg-gray-600">
          <span className="truncate">Run Test</span>
        </button>
        <button onClick={onDeploy} className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-primary/20 dark:bg-primary/30 text-primary dark:text-primary gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-4 hover:bg-primary/30 dark:hover:bg-primary/40">
          <span className="truncate">Deploy</span>
        </button>
        <button className="flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 bg-primary text-white gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-4 hover:bg-primary/90">
          <SVGIcon name="verified" className="fill text-white text-[20px]" ariaLabel="verified" />
          <span className="truncate">Save (ABAC Check)</span>
        </button>
      </div>
    </header>
  )
}

const WorkflowCanvas: React.FC<{ onAddStart?: () => void }> = ({ onAddStart }) => {
  return (
    <div className="flex-1 p-6 bg-background-light dark:bg-background-dark">
      <div className="flex flex-col items-center justify-center gap-6 rounded-xl border-2 border-dashed border-gray-300 dark:border-gray-700 h-full">
        <div className="flex max-w-[480px] flex-col items-center gap-2">
          <p className="text-gray-900 dark:text-white text-lg font-bold leading-tight tracking-[-0.015em] text-center">Workflow Canvas</p>
          <p className="text-gray-600 dark:text-gray-400 text-sm font-normal leading-normal text-center">Drag a 'Start' node from the toolbar to begin building your workflow.</p>
        </div>
        <button onClick={onAddStart} className="flex min-w-[84px] max-w-[480px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-gray-200 dark:bg-gray-700 text-gray-800 dark:text-gray-200 text-sm font-bold leading-normal tracking-[0.015em] hover:bg-gray-300 dark:hover:bg-gray-600">
          <span className="truncate">Add Start Node</span>
        </button>
      </div>
    </div>
  )
}

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

const Dashboard: React.FC = () => {
  const historyLogs = [
    { timestamp: '2025-10-27 14:30:15', eventId: 'evt-a1b2c3d4', workflowName: 'ClientOnboarding', status: 'Completed', user: 'john.doe@example.com' },
    { timestamp: '2025-10-27 14:25:10', eventId: 'evt-e5f6g7h8', workflowName: 'PortfolioRebalance', status: 'Failed', user: 'system-user' },
  ]

  const getStatusChip = (status: string) => {
    switch (status) {
      case 'Completed': return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-success/10 text-success">Completed</span>
      case 'Failed': return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-error/10 text-error">Failed</span>
      case 'Running': return <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-info/10 text-info">Running</span>
      default: return null
    }
  }

  return (
    <main className="flex-1 p-6 lg:p-8">
      <div className="max-w-7xl mx-auto">
        <header className="flex flex-col gap-6">
          <div className="flex flex-wrap justify-between items-start gap-4">
            <div className="flex flex-col gap-2">
              <h1 className="text-3xl lg:text-4xl font-black leading-tight tracking-[-0.033em] text-gray-900 dark:text-white">Workflow & Audit Dashboard</h1>
              <p className="text-gray-500 dark:text-[#9d9fb8] text-base font-normal leading-normal">Monitor workflow history and audit ABAC security logs.</p>
            </div>
          </div>
        </header>
        <section className="mt-8 bg-white dark:bg-[#1a1b2f] rounded-xl border border-gray-200 dark:border-[#3c3d53]">
          <div className="p-4 flex flex-col md:flex-row gap-4 items-center justify-between">
            <div className="relative w-full md:w-auto md:flex-1 max-w-sm">
              <input className="w-full h-10 pl-10 pr-4 rounded-lg bg-gray-100 dark:bg-[#111221] border-gray-200 dark:border-[#3c3d53] focus:ring-primary focus:border-primary text-sm" placeholder="Search logs..." type="text"/>
            </div>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead className="text-xs text-gray-500 dark:text-[#9d9fb8] uppercase bg-gray-50 dark:bg-[#292938]">
                <tr>
                  <th scope="col" className="px-6 py-3">Timestamp</th>
                  <th scope="col" className="px-6 py-3">Event ID</th>
                  <th scope="col" className="px-6 py-3">Workflow</th>
                  <th scope="col" className="px-6 py-3">Status</th>
                  <th scope="col" className="px-6 py-3">User</th>
                </tr>
              </thead>
              <tbody>
                {historyLogs.map((log, index) => (
                  <tr key={index} className="bg-white dark:bg-[#1a1b2f] border-b border-gray-200 dark:border-[#3c3d53]">
                    <td className="px-6 py-4 text-gray-900 dark:text-white">{log.timestamp}</td>
                    <td className="px-6 py-4 text-gray-900 dark:text-white">{log.eventId}</td>
                    <td className="px-6 py-4 text-gray-900 dark:text-white">{log.workflowName}</td>
                    <td className="px-6 py-4">{getStatusChip(log.status)}</td>
                    <td className="px-6 py-4 text-gray-900 dark:text-white">{log.user}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      </div>
    </main>
  )
}

const WorkflowBuilderPage: React.FC = () => {
  const [activeView, setActiveView] = useState<'builder' | 'dashboard'>('builder')

  const handleRunTest = async () => {
    // TODO: Implement test execution
    devLog('Running workflow test...')
  }

  const handleDeploy = () => {
    // TODO: Implement deployment
    devLog('Deploying workflow...')
  }

  const handleAddStart = () => {
    // TODO: Add start node to canvas
    devLog('Adding start node...')
  }

  return (
    <div className="flex h-screen bg-background-light dark:bg-background-dark">
      <SideNavBar />
      <div className="flex-1 flex flex-col">
        {activeView === 'builder' ? (
          <>
            <Header onRunTest={handleRunTest} onDeploy={handleDeploy} />
            <div className="flex-1 flex">
              <WorkflowCanvas onAddStart={handleAddStart} />
              <PropertiesPanel />
            </div>
          </>
        ) : (
          <Dashboard />
        )}
        <div className="border-t border-gray-200 dark:border-gray-800 p-4">
          <div className="flex justify-center gap-2">
            <button
              onClick={() => setActiveView('builder')}
              className={`px-4 py-2 rounded-lg text-sm font-medium ${
                activeView === 'builder'
                  ? 'bg-primary text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
              }`}
            >
              Builder
            </button>
            <button
              onClick={() => setActiveView('dashboard')}
              className={`px-4 py-2 rounded-lg text-sm font-medium ${
                activeView === 'dashboard'
                  ? 'bg-primary text-white'
                  : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300'
              }`}
            >
              Dashboard
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default WorkflowBuilderPage