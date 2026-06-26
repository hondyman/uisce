import React, { useState } from 'react'
import { useNotification } from '../hooks/useNotification';
import ExecutionMonitor from '../components/ExecutionMonitor'
import SideNavBar from '../components/SideNavBar'
import Header from '../components/Header'
import WorkflowCanvas from '../components/WorkflowCanvas'
import PropertiesPanel from '../components/PropertiesPanel'
import Dashboard from '../components/Dashboard'
import '../styles/WorkflowBuilder.css'
import { getCardClasses, getTextClasses } from '../utils/darkModeHelpers';

const WorkflowBuilder: React.FC = () => {
  const [running, setRunning] = useState(false)
  const [lastResult, setLastResult] = useState<string | null>(null)
  const notification = useNotification();

  const tenant = (() => {
    try {
      const s = localStorage.getItem('selected_tenant')
      if (!s) return undefined
      const obj = JSON.parse(s)
      return obj?.id
    } catch (e) {
      return undefined
    }
  })()

  const startWorkflow = async () => {
    setRunning(true)
    setLastResult(null)
    try {
      const body = { workflow: 'Onboarding', input: { tenant_id: tenant || 'local' } }
      const headers: Record<string,string> = { 'Content-Type': 'application/json' }
      if (tenant) headers['X-Tenant-ID'] = tenant

      const res = await fetch('/api/v1/workflows/start', { method: 'POST', headers, body: JSON.stringify(body) })
      if (!res.ok) throw new Error(`status ${res.status}`)
      const j = await res.json()
      setLastResult(JSON.stringify(j))
    } catch (err: any) {
      setLastResult(`error: ${err?.message || err}`)
    } finally {
      setRunning(false)
    }
  }

  const onRunTest = async () => {
    await startWorkflow()
  }

  const onDeploy = () => {
    notification.info('Deploy triggered (implement pipeline integration)')
  }

  const onAddStart = () => {
    notification.info('Add Start Node clicked - drag tool implementation goes here')
  }

  return (
    <div className={`flex h-full min-h-screen ${getCardClasses()}`}>
      <SideNavBar />
      <div className="flex flex-1 flex-col">
        <Header onRunTest={onRunTest} onDeploy={onDeploy} />
        <div className="flex flex-1">
          <WorkflowCanvas onAddStart={onAddStart} />
          <PropertiesPanel />
        </div>
        <div className="border-t border-gray-200 dark:border-border-dark">
          <div className="p-4">
            <h4 className={`text-sm font-medium ${getTextClasses('primary')}`}>Live Executions</h4>
            <div className="mt-2">
              <ExecutionMonitor />
            </div>
            <div className="mt-2">
              <strong>Last result:</strong> {running ? 'Starting...' : lastResult ?? '—'}
            </div>
          </div>
        </div>
      </div>
      <div className="hidden lg:block">
        <Dashboard />
      </div>
    </div>
  )
}

export default WorkflowBuilder
