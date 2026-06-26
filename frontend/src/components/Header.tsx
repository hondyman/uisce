import React from 'react'
import { getTextClasses, getButtonClasses } from '../utils/darkModeHelpers'
import SVGIcon from './relationship/SVGIcon'

const Header: React.FC<{ onRunTest: () => Promise<void>; onDeploy: () => void }> = ({ onRunTest, onDeploy }) => {
  const nodeButtons = [
    { title: 'Start Node', icon: 'play_arrow', color: 'text-green-500', label: 'Start' },
    { title: 'Activity Node', icon: 'terminal', color: 'text-blue-500', label: 'Activity' },
    { title: 'Signal Node', icon: 'sensors', color: 'text-purple-500', label: 'Signal' },
    { title: 'Conditional Branch', icon: 'call_split', color: 'text-amber-500', label: 'Branch' },
    { title: 'End Node', icon: 'stop_circle', color: 'text-red-500', label: 'End' },
  ]

  return (
    <header className={`flex justify-between items-center gap-4 px-6 py-3 border-b border-slate-200 dark:border-border-dark bg-white dark:bg-surface-dark`}>
      <div className="flex items-center gap-4">
        <div className={`flex items-center border-r border-slate-200 dark:border-border-dark pr-4`}>
          {nodeButtons.map((button) => (
            <button key={button.title} className={`flex items-center justify-center gap-2 p-2 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800/50 cursor-grab`} title={button.title}>
              <SVGIcon name={button.icon} className={`${button.color}`} ariaLabel={button.title} />
              <span className={`text-sm ${getTextClasses('primary')}`}>{button.label}</span>
            </button>
          ))}
        </div>
      </div>
      <div className="flex items-center gap-2">
        <button onClick={onRunTest} className={`${getButtonClasses('secondary')} flex min-w-[84px] cursor-pointer items-center justify-center overflow-hidden rounded-lg h-10 gap-2 text-sm font-bold leading-normal tracking-[0.015em] min-w-0 px-4`}>
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

export default Header
