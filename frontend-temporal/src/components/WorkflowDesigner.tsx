import React, { useCallback, useState } from 'react'
import ReactFlow, { addEdge, useNodesState, useEdgesState, Background, Node } from '../shims/reactflow'
import '../shims/reactflow.css'
import { useABAC } from '../hooks/useABAC'
import LiveEventsWidget from './LiveEventsWidget'

const initialNodes = [{ id: '1', data: { label: 'Start (ABAC Check)' }, position: { x: 0, y: 0 }, type: 'input' }]
const initialEdges: any[] = []

export default function WorkflowDesigner() {
  const { evaluate } = useABAC()
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes)
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges)
  const [selectedEvent, setSelectedEvent] = useState<any | null>(null)
  const [rfInstance, setRfInstance] = useState<any | null>(null)
  const [autoPan, setAutoPan] = useState<boolean>(true)
  const [autoHighlight, setAutoHighlight] = useState<boolean>(true)
  const [panDuration, setPanDuration] = useState<number>(800)
  const [highlightDurationClass, setHighlightDurationClass] = useState<string>('wf-duration-900')
  const [saving, setSaving] = useState(false)
  const [workflowName, setWorkflowName] = useState('')

  // Save workflow to backend
  const saveWorkflow = useCallback(async () => {
    if (!workflowName.trim()) {
      alert('Please enter a workflow name')
      return
    }

    // Check ABAC permission
    const allowed = await evaluate('create', 'workflow')
    if (!allowed) {
      alert('Access denied: Insufficient permissions to create workflows')
      return
    }

    setSaving(true)
    try {
      const workflowData = {
        name: workflowName,
        nodes: nodes,
        edges: edges,
        metadata: {
          createdBy: 'user', // TODO: Get from auth context
          description: 'Workflow created via Temporal UI'
        }
      }

      const response = await fetch('/api/temporal/workflows', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          // Tenant headers will be added by useABAC hook via fetch interceptor
        },
        body: JSON.stringify(workflowData)
      })

      if (!response.ok) {
        throw new Error(`Failed to save workflow: ${response.statusText}`)
      }

      const result = await response.json()
      alert(`Workflow saved successfully with ID: ${result.workflowId}`)
    } catch (error) {
      console.error('Error saving workflow:', error)
      alert(`Error saving workflow: ${error instanceof Error ? error.message : 'Unknown error'}`)
    } finally {
      setSaving(false)
    }
  }, [workflowName, nodes, edges, evaluate])

  // Load workflow from backend
  const loadWorkflow = useCallback(async (workflowId: string) => {
    try {
      const allowed = await evaluate('read', `workflow-${workflowId}`)
      if (!allowed) {
        alert('Access denied: Insufficient permissions to view this workflow')
        return
      }

      const response = await fetch(`/api/temporal/workflows/${workflowId}`)
      if (!response.ok) {
        throw new Error(`Failed to load workflow: ${response.statusText}`)
      }

      const workflowData = await response.json()
      setWorkflowName(workflowData.name || '')
      setNodes(workflowData.nodes || initialNodes)
      setEdges(workflowData.edges || initialEdges)
    } catch (error) {
      console.error('Error loading workflow:', error)
      alert(`Error loading workflow: ${error instanceof Error ? error.message : 'Unknown error'}`)
    }
  }, [evaluate])

  // Add new node
  const addNode = useCallback((type: string) => {
    const newNode = {
      id: `${nodes.length + 1}`,
      data: { label: `${type} Node` },
      position: { x: Math.random() * 400, y: Math.random() * 400 },
      type: type.toLowerCase()
    }
    setNodes((nds) => [...nds, newNode])
  }, [nodes.length])

  // Highlight a node when an event is selected. Uses a few heuristic fields from the event
  const highlightNodeForEvent = useCallback((ev: any) => {
    if (!ev) return

    // Prefer canonical field payload.nodeId first
    const canonical = ev && ev.payload && (ev.payload.nodeId || ev.payload.targetNodeId) || ev.nodeId || ev.node_id || ev.targetNodeId || ev.target_node_id || ev.target || ev.id || null

    const candidates = new Set<string>()
    if (canonical) candidates.add(String(canonical))
    const tryStr = (v: any) => { if (v !== undefined && v !== null) candidates.add(String(v)) }
    tryStr(ev.nodeId)
    tryStr(ev.node_id)
    tryStr(ev.target)
    tryStr(ev.targetNodeId)
    tryStr(ev.target_node_id)
    tryStr(ev.payload && ev.payload.nodeId)
    tryStr(ev.payload && ev.payload.targetNodeId)
    tryStr(ev.resource && ev.resource.nodeId)
    tryStr(ev.resource && ev.resource.id)
    tryStr(ev.id)

    const typeHint = ev.type || ev.eventType || ev.name

    if (!autoHighlight && !autoPan) return

    setNodes((nds: Node[]) => {
      let matchId: string | null = null
      for (const n of nds) {
        if (candidates.has(n.id)) { matchId = n.id; break }
      }
      if (!matchId && typeHint) {
        for (const n of nds) {
          const label = n.data && n.data.label ? String(n.data.label) : ''
          if (label && label.toLowerCase().includes(String(typeHint).toLowerCase())) { matchId = n.id; break }
        }
      }

      if (!matchId) return nds

      const next = nds.map((n: Node) => {
        if (!autoHighlight) return ({ ...n })
        const base = (n.className || '').replace('shadow-wf-highlight', '').replace('animate-wf-pulse', '').replace('border-indigo-600', '').replace('wf-duration-300','').replace('wf-duration-600','').replace('wf-duration-900','').trim()
        return { ...n, className: n.id === matchId ? `${base} shadow-wf-highlight animate-wf-pulse border-indigo-600 ${highlightDurationClass}`.trim() : base }
      })

      // pan/center to the node (if enabled and reactflow instance available)
      if (autoPan) {
        try {
          const nodeForCenter = (nds.find(n => n.id === matchId) || next.find(n => n.id === matchId)) as Node | undefined
          if (rfInstance && nodeForCenter && nodeForCenter.position) {
            const x = (nodeForCenter.position as any).x || 0
            const y = (nodeForCenter.position as any).y || 0
            if (typeof rfInstance.setCenter === 'function') {
              try { rfInstance.setCenter(x, y, { duration: panDuration }) } catch (_) { /* ignore */ }
            } else if (typeof rfInstance.fitView === 'function') {
              try { rfInstance.fitView({ nodes: [matchId] }) } catch (_) { /* ignore */ }
            }
          }
        } catch (ex) {
          // ignore pan errors
        }
      }

      // clear highlight after a short flash
      setTimeout(() => {
        setNodes((cur: Node[]) => cur.map((nn: Node) => ({ ...nn, className: (nn.className || '').replace('shadow-wf-highlight', '').replace('animate-wf-pulse', '').replace('border-indigo-600','').replace('wf-duration-300','').replace('wf-duration-600','').replace('wf-duration-900','').trim() })))
      }, 3000)

      return next
    })
  }, [setNodes, rfInstance, autoPan, autoHighlight, panDuration])

  const onConnect = useCallback((params: any) => setEdges((eds: any) => addEdge(params, eds)), [setEdges])

  return (
    <div className="flex h-[80vh] p-4 gap-3">
      <div className="flex-1">
        <div className="mb-3 flex gap-2 items-center">
          <input
            type="text"
            placeholder="Workflow name"
            value={workflowName}
            onChange={(e) => setWorkflowName(e.target.value)}
            className="px-3 py-1 border rounded"
          />
          <button
            onClick={saveWorkflow}
            disabled={saving}
            className="px-3 py-1 rounded bg-indigo-600 text-white disabled:bg-gray-400"
          >
            {saving ? 'Saving...' : 'Save Workflow'}
          </button>
          <button
            onClick={() => {
              const id = prompt('Enter workflow ID to load:')
              if (id) loadWorkflow(id)
            }}
            className="px-3 py-1 rounded bg-green-600 text-white"
          >
            Load Workflow
          </button>
        </div>
        <div className="h-[70vh] mt-3 bg-gray-50 rounded border">
          <ReactFlow nodes={nodes} edges={edges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} onConnect={onConnect} onInit={(inst: any) => setRfInstance(inst)}>
            <Background />
          </ReactFlow>
        </div>
      </div>

      <aside className="w-80">
        <div className="p-2 border-b border-gray-200 mb-2">
          <h4 className="m-0 text-sm font-medium">Live Event Settings</h4>
            <div className="flex gap-2 items-center mt-2">
              <label className="flex gap-2 items-center text-sm">
                <input type="checkbox" className="form-checkbox h-4 w-4" checked={autoPan} onChange={(e) => setAutoPan(e.target.checked)} />
                <span>Auto-pan</span>
              </label>
              <label className="flex gap-2 items-center text-sm">
                <input type="checkbox" className="form-checkbox h-4 w-4" checked={autoHighlight} onChange={(e) => setAutoHighlight(e.target.checked)} />
                <span>Highlight</span>
              </label>
            </div>
            <div className="flex gap-2 items-center mt-2">
              <label className="text-sm">Pan duration (ms)</label>
              <input aria-label="Pan duration in ms" placeholder="ms" className="w-24 px-2 py-1 border rounded text-sm" type="number" value={panDuration} onChange={(e) => setPanDuration(Number(e.target.value) || 0)} />
            </div>
            <div className="flex gap-2 items-center mt-2">
              <label className="text-sm">Highlight duration</label>
              <select aria-label="Highlight duration" value={highlightDurationClass} onChange={(e) => setHighlightDurationClass(e.target.value)} className="px-2 py-1 border rounded text-sm">
                <option value="wf-duration-300">300ms</option>
                <option value="wf-duration-600">600ms</option>
                <option value="wf-duration-900">900ms</option>
              </select>
            </div>
        </div>

        <LiveEventsWidget onSelect={(ev) => { setSelectedEvent(ev); highlightNodeForEvent(ev) }} />

        {selectedEvent && (
          <div className="mt-3">
            <h5 className="text-sm font-medium">Selected Event</h5>
            <pre className="whitespace-pre-wrap max-h-44 overflow-auto bg-slate-900 text-sky-100 p-3 rounded mt-2">{JSON.stringify(selectedEvent, null, 2)}</pre>
          </div>
        )}
      </aside>
    </div>
  )
}
