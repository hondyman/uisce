import React from 'react'
import { getBadgeClasses, getTableClasses, getTextClasses, getCardClasses } from '../utils/darkModeHelpers'

const Dashboard: React.FC = () => {
  const historyLogs = [
    { timestamp: '2025-10-27 14:30:15', eventId: 'evt-a1b2c3d4', workflowName: 'ClientOnboarding', status: 'Completed', user: 'john.doe@example.com' },
    { timestamp: '2025-10-27 14:25:10', eventId: 'evt-e5f6g7h8', workflowName: 'PortfolioRebalance', status: 'Failed', user: 'system-user' },
  ]

  const getStatusChip = (status: string) => {
    switch (status) {
      case 'Completed': return <span className={getBadgeClasses('success')}>Completed</span>
      case 'Failed': return <span className={getBadgeClasses('error')}>Failed</span>
      case 'Running': return <span className={getBadgeClasses('info')}>Running</span>
      default: return null
    }
  }

  const tableClasses = getTableClasses();

  return (
    <main className="flex-1 p-6 lg:p-8 bg-background-light dark:bg-background-dark">
      <div className="max-w-7xl mx-auto">
        <header className="flex flex-col gap-6">
          <div className="flex flex-wrap justify-between items-start gap-4">
            <div className="flex flex-col gap-2">
              <h1 className="text-3xl lg:text-4xl font-black leading-tight tracking-[-0.033em] text-slate-900 dark:text-text-light">Workflow & Audit Dashboard</h1>
              <p className={`${getTextClasses('secondary')} text-base font-normal leading-normal`}>Monitor workflow history and audit ABAC security logs.</p>
            </div>
          </div>
        </header>
        <section className={`mt-8 ${getCardClasses()}`}>
          <div className="p-4 flex flex-col md:flex-row gap-4 items-center justify-between">
            <div className="relative w-full md:w-auto md:flex-1 max-w-sm">
              <input className="w-full h-10 pl-10 pr-4 rounded-lg bg-slate-100 dark:bg-surface-dark border border-slate-200 dark:border-border-dark focus:ring-primary focus:border-primary text-sm text-slate-900 dark:text-text-light placeholder-slate-400 dark:placeholder-text-dim" placeholder="Search logs..." type="text"/>
            </div>
          </div>
          <div className={tableClasses.container}>
            <table className={tableClasses.table}>
              <thead className={tableClasses.thead}>
                <tr>
                  <th className={tableClasses.th} scope="col">Timestamp</th>
                  <th className={tableClasses.th} scope="col">Event ID</th>
                  <th className={tableClasses.th} scope="col">Workflow Name</th>
                  <th className={tableClasses.th} scope="col">Status</th>
                  <th className={tableClasses.th} scope="col">User</th>
                </tr>
              </thead>
              <tbody className={tableClasses.tbody}>
                {historyLogs.map((log, index) => (
                  <tr key={index} className={tableClasses.tr}>
                    <td className={`${tableClasses.td} font-medium`}>{log.timestamp}</td>
                    <td className={tableClasses.td}>{log.eventId}</td>
                    <td className={tableClasses.td}>{log.workflowName}</td>
                    <td className={tableClasses.td}>{getStatusChip(log.status)}</td>
                    <td className={tableClasses.td}>{log.user}</td>
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

export default Dashboard
