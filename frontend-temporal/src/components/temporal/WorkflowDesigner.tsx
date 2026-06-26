import React, { useCallback, useState } from 'react'
import ReactFlow, { addEdge, useNodesState, useEdgesState, Background, Node } from 'reactflow'
import 'reactflow/dist/style.css'
import { useABAC } from '../../abac'
import LiveEventsWidget from './LiveEventsWidget'
import temporalService from '../../services/temporalService'

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

  const saveWorkflow = async () => {
    const ok = await evaluate('create', 'workflow')
    if (!ok) return alert('Access denied')
    try {
      const payload = { workflow_id: undefined, status: 'draft', nodes, edges, meta: { saved_at: new Date().toISOString() } }
      const res = await temporalService.saveWorkflow(payload)
      // show created id or success
      const id = (res && res.id) ? res.id : (res && res.body && res.body.id) ? res.body.id : null
      alert(`Workflow saved successfully${id ? ' (id: ' + String(id) + ')' : ''}`)
    } catch (err: any) {
      console.error('saveWorkflow failed', err)
      alert('Failed to save workflow: ' + (err?.message || String(err)))
    }
  }

  return (
    <div className="flex h-[80vh] p-4 gap-3">
      <div className="flex-1">
        <button onClick={saveWorkflow} className="px-3 py-1 rounded bg-indigo-600 text-white">Save Workflow (ABAC Validated)</button>
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
