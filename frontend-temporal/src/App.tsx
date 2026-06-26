import React from 'react'
import { Routes, Route, Link } from 'react-router-dom'
import Home from './pages/Home'
import WorkflowDesigner from './components/WorkflowDesigner'
import ExecutionMonitor from './components/ExecutionMonitor'
import NamespaceManager from './components/NamespaceManager'
import DebugPanel from './components/DebugPanel'
import './styles.css'

export default function App() {
  return (
    <div>
      <nav className="topnav">
        <Link to="/">Home</Link>
        <Link to="/designer">Designer</Link>
        <Link to="/executions">Executions</Link>
        <Link to="/namespaces">Namespaces</Link>
        <Link to="/debug">Debug</Link>
      </nav>
      <main className="main">
        <Routes>
          <Route path="/" element={<Home />} />
          <Route path="/designer" element={<WorkflowDesigner />} />
          <Route path="/executions" element={<ExecutionMonitor />} />
          <Route path="/namespaces" element={<NamespaceManager />} />
          <Route path="/debug" element={<DebugPanel />} />
        </Routes>
      </main>
    </div>
  )
}
