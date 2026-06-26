import React from 'react'

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

export default WorkflowCanvas
