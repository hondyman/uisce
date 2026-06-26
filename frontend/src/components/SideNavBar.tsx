import React from 'react'
import '../styles/WorkflowBuilder.css'
import { getTextClasses } from '../utils/darkModeHelpers'
import SVGIcon from './relationship/SVGIcon'

const SideNavBar: React.FC = () => {
  const navItems = [
    { name: 'Dashboard', icon: 'dashboard' },
    { name: 'Workflows', icon: 'hub', active: true },
    { name: 'Executions', icon: 'play_circle' },
    { name: 'Schedules', icon: 'event_repeat' },
    { name: 'Archives', icon: 'archive' },
  ]

  const bottomNavItems = [
    { name: 'Settings', icon: 'settings' },
    { name: 'Help', icon: 'help' },
  ]

  return (
    <aside className={`flex h-full w-64 flex-col border-r border-slate-200 dark:border-border-dark bg-white dark:bg-surface-dark`}>
      <div className="flex h-full flex-col justify-between p-4">
        <div className="flex flex-col gap-4">
          <div className="flex items-center gap-3">
            <div className="avatar-bg rounded-full w-10 h-10 bg-center bg-cover" />
            <div className="flex flex-col">
              <h1 className={`text-base font-medium leading-normal ${getTextClasses('primary')}`}>WealthFlow Orchestrator</h1>
              <p className={`text-sm font-normal leading-normal ${getTextClasses('secondary')}`}>Enterprise Edition</p>
            </div>
          </div>
          <nav className="flex flex-col gap-2">
            {navItems.map((item) => (
              <a
                key={item.name}
                href="#"
                className={`flex items-center gap-3 px-3 py-2 rounded-lg ${
                  item.active
                    ? 'bg-primary/10 dark:bg-primary/20 text-primary dark:text-primary'
                    : `text-slate-700 dark:text-text-dim hover:bg-slate-100 dark:hover:bg-slate-800/50`
                }`}
              >
                <SVGIcon name={item.icon} className={`${item.active ? 'fill' : ''}`} ariaLabel={item.name} />
                <p className="text-sm font-medium leading-normal">{item.name}</p>
              </a>
            ))}
          </nav>
        </div>
        <div className="flex flex-col gap-4">
          <button id="new-workflow-button" className="flex w-full cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-[0.015em] hover:bg-primary/90">
            <span className="truncate">New Workflow</span>
          </button>
          <div className="flex flex-col gap-1">
            {bottomNavItems.map((item) => (
              <a key={item.name} href="#" className={`flex items-center gap-3 px-3 py-2 text-slate-700 dark:text-text-dim hover:bg-slate-100 dark:hover:bg-slate-800/50 rounded-lg`}>
                <SVGIcon name={item.icon} ariaLabel={item.name} />
                <p className="text-sm font-medium leading-normal">{item.name}</p>
              </a>
            ))}
          </div>
        </div>
      </div>
    </aside>
  )
}

export default SideNavBar
