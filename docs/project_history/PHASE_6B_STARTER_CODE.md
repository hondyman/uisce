# Phase 6B Tasks 4-6: Code Templates & Starter Code

## 🎯 Copy-Paste Ready Templates

Use these as starting points for each task. Customize with your specific logic.

---

## 📌 Task 4: React BP Builder - Starter Code

### **BPBuilder.tsx** (Main Component Template)

```tsx
import React, { useState, useCallback } from 'react'
import { BPStep } from '../types'
import StepPalette from './StepPalette'
import BPCanvas from './BPCanvas'
import StepEditor from './StepEditor'
import BPPreview from './BPPreview'
import BPActions from './BPActions'
import './BPBuilder.css'

export default function BPBuilder() {
  // State
  const [steps, setSteps] = useState<BPStep[]>([])
  const [selectedStepIndex, setSelectedStepIndex] = useState<number | null>(null)
  const [bpName, setBPName] = useState('')
  const [bpDescription, setBPDescription] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  // Callbacks
  const handleAddStep = useCallback((stepType: string) => {
    const newStep: BPStep = {
      step_order: steps.length + 1,
      step_type: stepType,
      step_name: `Step ${steps.length + 1}`,
      duration_hours: 0,
      assignee_role: '',
      assignee_user: '',
      trigger_ids: [],
      condition_json: null,
      action_config: null
    }
    setSteps([...steps, newStep])
    setSelectedStepIndex(steps.length)
  }, [steps])

  const handleUpdateStep = useCallback((index: number, updatedStep: BPStep) => {
    const newSteps = [...steps]
    newSteps[index] = updatedStep
    setSteps(newSteps)
  }, [])

  const handleDeleteStep = useCallback((index: number) => {
    const newSteps = steps.filter((_, i) => i !== index)
    // Reorder step_order
    newSteps.forEach((step, i) => {
      step.step_order = i + 1
    })
    setSteps(newSteps)
    setSelectedStepIndex(null)
  }, [steps])

  const handleReorderSteps = useCallback((reorderedSteps: BPStep[]) => {
    reorderedSteps.forEach((step, i) => {
      step.step_order = i + 1
    })
    setSteps(reorderedSteps)
  }, [])

  const selectedStep = selectedStepIndex !== null ? steps[selectedStepIndex] : null

  return (
    <div className="bp-builder">
      {/* Header */}
      <div className="bp-header">
        <div className="bp-title-section">
          <input
            type="text"
            placeholder="Business Process Name"
            value={bpName}
            onChange={(e) => setBPName(e.target.value)}
            className="bp-name-input"
          />
          <input
            type="text"
            placeholder="Description"
            value={bpDescription}
            onChange={(e) => setBPDescription(e.target.value)}
            className="bp-description-input"
          />
        </div>
        <BPActions
          bpName={bpName}
          bpDescription={bpDescription}
          steps={steps}
          isSaving={isSaving}
          onSaving={setIsSaving}
          onError={setError}
          onSuccess={setSuccess}
        />
      </div>

      {/* Alerts */}
      {error && <div className="alert alert-error">{error}</div>}
      {success && <div className="alert alert-success">{success}</div>}

      {/* Main Content */}
      <div className="bp-content">
        {/* Left: Palette */}
        <div className="bp-palette">
          <StepPalette onAddStep={handleAddStep} />
        </div>

        {/* Center: Canvas */}
        <div className="bp-canvas">
          <BPCanvas
            steps={steps}
            selectedIndex={selectedStepIndex}
            onSelectStep={setSelectedStepIndex}
            onReorderSteps={handleReorderSteps}
          />
        </div>

        {/* Right: Editor */}
        <div className="bp-editor">
          {selectedStep ? (
            <StepEditor
              step={selectedStep}
              index={selectedStepIndex!}
              onChange={handleUpdateStep}
              onDelete={handleDeleteStep}
            />
          ) : (
            <div className="no-step-selected">
              Select a step to edit
            </div>
          )}
        </div>
      </div>

      {/* Bottom: Preview */}
      <div className="bp-preview">
        <BPPreview
          bpName={bpName}
          bpDescription={bpDescription}
          steps={steps}
        />
      </div>
    </div>
  )
}
```

### **StepPalette.tsx** (Drag Source)

```tsx
import React from 'react'

interface StepPaletteProps {
  onAddStep: (stepType: string) => void
}

const STEP_TYPES = [
  { type: 'data_entry', label: 'Data Entry', icon: '📝', color: '#e3f2fd' },
  { type: 'validate', label: 'Validation', icon: '✓', color: '#f3e5f5' },
  { type: 'approve', label: 'Approval', icon: '👤', color: '#e8f5e9' },
  { type: 'notify', label: 'Notification', icon: '📢', color: '#fff3e0' },
  { type: 'integrate', label: 'Integration', icon: '🔗', color: '#fce4ec' },
  { type: 'compute', label: 'Compute', icon: '⚙️', color: '#f1f8e9' }
]

export default function StepPalette({ onAddStep }: StepPaletteProps) {
  const handleDragStart = (e: React.DragEvent, stepType: string) => {
    e.dataTransfer.effectAllowed = 'copy'
    e.dataTransfer.setData('stepType', stepType)
  }

  return (
    <div className="step-palette">
      <h3>Step Types</h3>
      <div className="palette-items">
        {STEP_TYPES.map((stepType) => (
          <div
            key={stepType.type}
            draggable
            onDragStart={(e) => handleDragStart(e, stepType.type)}
            onClick={() => onAddStep(stepType.type)}
            className="palette-item"
            style={{ backgroundColor: stepType.color }}
            title="Drag or click to add"
          >
            <div className="palette-icon">{stepType.icon}</div>
            <div className="palette-label">{stepType.label}</div>
          </div>
        ))}
      </div>
    </div>
  )
}
```

### **BPCanvas.tsx** (Visualization)

```tsx
import React from 'react'
import { BPStep } from '../types'

interface BPCanvasProps {
  steps: BPStep[]
  selectedIndex: number | null
  onSelectStep: (index: number) => void
  onReorderSteps: (steps: BPStep[]) => void
}

export default function BPCanvas({
  steps,
  selectedIndex,
  onSelectStep,
  onReorderSteps
}: BPCanvasProps) {
  const [draggedIndex, setDraggedIndex] = React.useState<number | null>(null)

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'move'
  }

  const handleDrop = (e: React.DragEvent, index: number) => {
    e.preventDefault()
    if (draggedIndex !== null && draggedIndex !== index) {
      const newSteps = [...steps]
      const [movedStep] = newSteps.splice(draggedIndex, 1)
      newSteps.splice(index, 0, movedStep)
      onReorderSteps(newSteps)
    }
    setDraggedIndex(null)
  }

  return (
    <div className="bp-canvas">
      <h3>Workflow</h3>
      <div className="canvas-content">
        {steps.length === 0 ? (
          <div className="canvas-empty">
            Drag steps here or click in palette
          </div>
        ) : (
          <div className="steps-flow">
            {steps.map((step, index) => (
              <React.Fragment key={index}>
                <div
                  draggable
                  onDragStart={() => setDraggedIndex(index)}
                  onDragOver={handleDragOver}
                  onDrop={(e) => handleDrop(e, index)}
                  onClick={() => onSelectStep(index)}
                  className={`canvas-step ${
                    selectedIndex === index ? 'selected' : ''
                  }`}
                >
                  <div className="step-number">{step.step_order}</div>
                  <div className="step-content">
                    <div className="step-name">{step.step_name}</div>
                    <div className="step-type">{step.step_type}</div>
                    {step.duration_hours > 0 && (
                      <div className="step-duration">{step.duration_hours}h</div>
                    )}
                  </div>
                </div>
                {index < steps.length - 1 && (
                  <div className="step-connector">↓</div>
                )}
              </React.Fragment>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
```

### **StepEditor.tsx** (Right Panel Form)

```tsx
import React from 'react'
import { BPStep } from '../types'

interface StepEditorProps {
  step: BPStep
  index: number
  onChange: (index: number, step: BPStep) => void
  onDelete: (index: number) => void
}

export default function StepEditor({
  step,
  index,
  onChange,
  onDelete
}: StepEditorProps) {
  const handleChange = (field: keyof BPStep, value: any) => {
    onChange(index, { ...step, [field]: value })
  }

  const handleDelete = () => {
    if (window.confirm(`Delete "${step.step_name}"?`)) {
      onDelete(index)
    }
  }

  return (
    <div className="step-editor">
      <h3>Step Editor</h3>
      
      <div className="form-group">
        <label>Step Name</label>
        <input
          type="text"
          value={step.step_name}
          onChange={(e) => handleChange('step_name', e.target.value)}
          placeholder="e.g., Manager Approval"
        />
      </div>

      <div className="form-group">
        <label>Step Type</label>
        <select
          value={step.step_type}
          onChange={(e) => handleChange('step_type', e.target.value)}
        >
          <option value="data_entry">Data Entry</option>
          <option value="validate">Validation</option>
          <option value="approve">Approval</option>
          <option value="notify">Notification</option>
          <option value="integrate">Integration</option>
          <option value="compute">Compute</option>
        </select>
      </div>

      <div className="form-group">
        <label>Assignee Role</label>
        <input
          type="text"
          value={step.assignee_role}
          onChange={(e) => handleChange('assignee_role', e.target.value)}
          placeholder="e.g., Manager, HR, CEO"
        />
      </div>

      <div className="form-group">
        <label>Assignee User (Optional)</label>
        <input
          type="text"
          value={step.assignee_user}
          onChange={(e) => handleChange('assignee_user', e.target.value)}
          placeholder="e.g., john@company.com"
        />
      </div>

      <div className="form-group">
        <label>Duration (hours)</label>
        <input
          type="number"
          value={step.duration_hours}
          onChange={(e) => handleChange('duration_hours', parseInt(e.target.value))}
          min="0"
          max="999"
        />
      </div>

      <div className="form-group">
        <label>Triggers (comma-separated)</label>
        <input
          type="text"
          value={step.trigger_ids.join(', ')}
          onChange={(e) =>
            handleChange('trigger_ids', e.target.value.split(',').map((s) => s.trim()))
          }
          placeholder="e.g., trigger-save, trigger-validate"
        />
      </div>

      <div className="form-actions">
        <button className="btn-delete" onClick={handleDelete}>
          Delete Step
        </button>
      </div>
    </div>
  )
}
```

### **BPActions.tsx** (Save/Deploy)

```tsx
import React, { useState } from 'react'
import { BPStep } from '../types'

interface BPActionsProps {
  bpName: string
  bpDescription: string
  steps: BPStep[]
  isSaving: boolean
  onSaving: (isSaving: boolean) => void
  onError: (error: string) => void
  onSuccess: (message: string) => void
}

export default function BPActions({
  bpName,
  bpDescription,
  steps,
  isSaving,
  onSaving,
  onError,
  onSuccess
}: BPActionsProps) {
  const getTenantId = () => {
    const tenant = localStorage.getItem('selected_tenant')
    return tenant ? JSON.parse(tenant).id : null
  }

  const getDatasourceId = () => {
    const ds = localStorage.getItem('selected_datasource')
    return ds ? JSON.parse(ds).id : null
  }

  const handleSave = async () => {
    // Validation
    if (!bpName.trim()) {
      onError('BP name is required')
      return
    }
    if (steps.length === 0) {
      onError('BP must have at least one step')
      return
    }

    const tenantId = getTenantId()
    const datasourceId = getDatasourceId()
    if (!tenantId || !datasourceId) {
      onError('Tenant and datasource are required')
      return
    }

    // Prepare request
    const payload = {
      process_name: bpName,
      description: bpDescription,
      steps: steps
    }

    try {
      onSaving(true)
      const response = await fetch(
        `/api/bp?tenant_id=${tenantId}&datasource_id=${datasourceId}`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId
          },
          body: JSON.stringify(payload)
        }
      )

      if (!response.ok) {
        const error = await response.text()
        throw new Error(error)
      }

      const data = await response.json()
      onSuccess(`BP "${bpName}" created successfully! ID: ${data.id}`)
    } catch (err) {
      onError(`Failed to save BP: ${err}`)
    } finally {
      onSaving(false)
    }
  }

  const handleExport = () => {
    const payload = {
      process_name: bpName,
      description: bpDescription,
      steps: steps
    }
    const json = JSON.stringify(payload, null, 2)
    const blob = new Blob([json], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${bpName}.json`
    a.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="bp-actions">
      <button
        onClick={handleSave}
        disabled={isSaving}
        className="btn-primary"
      >
        {isSaving ? 'Saving...' : 'Save BP'}
      </button>
      <button
        onClick={handleExport}
        className="btn-secondary"
      >
        Export JSON
      </button>
    </div>
  )
}
```

### **BPPreview.tsx** (JSON Display)

```tsx
import React, { useState } from 'react'
import { BPStep } from '../types'

interface BPPreviewProps {
  bpName: string
  bpDescription: string
  steps: BPStep[]
}

export default function BPPreview({
  bpName,
  bpDescription,
  steps
}: BPPreviewProps) {
  const [isExpanded, setIsExpanded] = useState(false)

  const payload = {
    process_name: bpName,
    description: bpDescription,
    steps: steps
  }

  const json = JSON.stringify(payload, null, 2)

  const handleCopy = () => {
    navigator.clipboard.writeText(json)
    alert('Copied to clipboard!')
  }

  return (
    <div className="bp-preview">
      <div className="preview-header">
        <h3>JSON Preview</h3>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className="btn-toggle"
        >
          {isExpanded ? 'Collapse' : 'Expand'}
        </button>
        <button onClick={handleCopy} className="btn-copy">
          Copy
        </button>
      </div>
      {isExpanded && (
        <pre className="preview-content">{json}</pre>
      )}
    </div>
  )
}
```

### **CSS** (BPBuilder.css)

```css
.bp-builder {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #f5f5f5;
}

.bp-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  background: white;
  border-bottom: 1px solid #ddd;
  gap: 16px;
}

.bp-title-section {
  display: flex;
  gap: 8px;
  flex: 1;
}

.bp-name-input,
.bp-description-input {
  padding: 8px 12px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.bp-content {
  display: flex;
  flex: 1;
  gap: 16px;
  padding: 16px;
  overflow: hidden;
}

.bp-palette,
.bp-canvas,
.bp-editor {
  background: white;
  border-radius: 4px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.bp-palette {
  width: 150px;
  overflow-y: auto;
}

.bp-canvas {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.bp-editor {
  width: 300px;
  overflow-y: auto;
}

.palette-items {
  display: grid;
  gap: 8px;
}

.palette-item {
  padding: 8px;
  border-radius: 4px;
  cursor: grab;
  text-align: center;
  border: 1px solid #ddd;
  transition: all 0.2s;
}

.palette-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.palette-icon {
  font-size: 24px;
}

.palette-label {
  font-size: 12px;
  font-weight: bold;
  margin-top: 4px;
}

.canvas-step {
  padding: 12px;
  margin: 8px 0;
  border: 2px solid #ddd;
  border-radius: 4px;
  cursor: move;
  background: white;
  transition: all 0.2s;
}

.canvas-step:hover,
.canvas-step.selected {
  border-color: #2196f3;
  background: #e3f2fd;
}

.form-group {
  margin-bottom: 12px;
}

.form-group label {
  display: block;
  font-weight: bold;
  margin-bottom: 4px;
  font-size: 14px;
}

.form-group input,
.form-group select {
  width: 100%;
  padding: 8px;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 14px;
}

.bp-preview {
  padding: 16px;
  background: white;
  border-top: 1px solid #ddd;
  max-height: 200px;
  overflow: hidden;
}

.preview-content {
  max-height: 150px;
  overflow-y: auto;
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
}

.btn-primary, .btn-secondary {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: bold;
  transition: all 0.2s;
}

.btn-primary {
  background: #2196f3;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #1976d2;
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.alert {
  padding: 12px;
  margin: 8px;
  border-radius: 4px;
  font-weight: bold;
}

.alert-error {
  background: #ffebee;
  color: #c62828;
  border: 1px solid #ef5350;
}

.alert-success {
  background: #e8f5e9;
  color: #2e7d32;
  border: 1px solid #66bb6a;
}
```

---

## 📌 Task 5: HireEmployee Demo - Starter Code

### **HireEmployeeDemo.tsx**

```tsx
import React, { useState } from 'react'
import StepTimeline from './StepTimeline'
import EventLog from './EventLog'

export default function HireEmployeeDemo() {
  const [step, setStep] = useState<'create' | 'start' | 'monitor' | 'approve' | 'done'>('create')
  const [bpId, setBpId] = useState<string | null>(null)
  const [instanceId, setInstanceId] = useState<string | null>(null)
  const [status, setStatus] = useState<any>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const getTenant = () => {
    const tenant = localStorage.getItem('selected_tenant')
    return tenant ? JSON.parse(tenant).id : 'tenant-1'
  }

  const getDs = () => {
    const ds = localStorage.getItem('selected_datasource')
    return ds ? JSON.parse(ds).id : 'ds-1'
  }

  const handleCreateBP = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/bp?tenant_id=${getTenant()}&datasource_id=${getDs()}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          process_name: 'HireEmployee',
          description: 'End-to-end employee hiring workflow',
          steps: [
            {
              step_order: 1,
              step_type: 'data_entry',
              step_name: 'Collect Employee Info',
              duration_hours: 0,
              assignee_role: 'HR'
            },
            {
              step_order: 2,
              step_type: 'validate',
              step_name: 'Background Check (24h)',
              duration_hours: 24,
              assignee_role: 'HR'
            },
            {
              step_order: 3,
              step_type: 'approve',
              step_name: 'Manager Approval (48h)',
              duration_hours: 48,
              assignee_role: 'Manager'
            },
            {
              step_order: 4,
              step_type: 'notify',
              step_name: 'Send Offer Letter',
              duration_hours: 0,
              assignee_role: 'HR'
            }
          ]
        })
      })

      if (!response.ok) throw new Error('Failed to create BP')
      const data = await response.json()
      setBpId(data.id)
      setStep('start')
      setError(null)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleStartExecution = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/bp/${bpId}/start?tenant_id=${getTenant()}&datasource_id=${getDs()}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          entity_id: 'emp-12345',
          entity_type: 'employee',
          data: {
            first_name: 'John',
            last_name: 'Doe',
            email: 'john.doe@company.com',
            department: 'Engineering',
            position: 'Senior Engineer',
            hire_date: '2024-02-01'
          }
        })
      })

      if (!response.ok) throw new Error('Failed to start execution')
      const data = await response.json()
      setInstanceId(data.instance_id)
      setStep('monitor')
      setError(null)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handlePollStatus = async () => {
    setLoading(true)
    try {
      const response = await fetch(`/api/bp/instance/${instanceId}?tenant_id=${getTenant()}&datasource_id=${getDs()}`)
      if (!response.ok) throw new Error('Failed to get status')
      const data = await response.json()
      setStatus(data)
      setError(null)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleApprove = async () => {
    setLoading(true)
    try {
      await fetch(`/api/bp/instance/${instanceId}/approve?tenant_id=${getTenant()}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          decision: 'approved',
          comment: 'Candidate approved by CEO'
        })
      })
      setStep('done')
      setError(null)
      handlePollStatus()
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ padding: '20px', maxWidth: '1200px', margin: '0 auto' }}>
      <h1>🎯 HireEmployee BP End-to-End Demo</h1>

      {error && (
        <div style={{ padding: '12px', background: '#ffebee', color: '#c62828', borderRadius: '4px', marginBottom: '16px' }}>
          ❌ {error}
        </div>
      )}

      {/* Step 1: Create BP */}
      <section style={{ marginBottom: '24px', padding: '16px', background: '#f5f5f5', borderRadius: '4px' }}>
        <h2>Step 1: Create Business Process</h2>
        <p>Create the HireEmployee BP definition with 4 steps.</p>
        <button
          onClick={handleCreateBP}
          disabled={loading || step !== 'create'}
          style={{ padding: '10px 20px', fontSize: '16px', cursor: 'pointer' }}
        >
          {loading ? '⏳ Creating...' : '✅ Create HireEmployee BP'}
        </button>
        {bpId && <p style={{ marginTop: '8px', color: 'green' }}>✅ BP Created: <code>{bpId}</code></p>}
      </section>

      {/* Step 2: Start Execution */}
      <section style={{ marginBottom: '24px', padding: '16px', background: '#f5f5f5', borderRadius: '4px' }}>
        <h2>Step 2: Start Execution</h2>
        <p>Start a BP instance for a specific employee.</p>
        <button
          onClick={handleStartExecution}
          disabled={loading || step !== 'start' || !bpId}
          style={{ padding: '10px 20px', fontSize: '16px', cursor: 'pointer' }}
        >
          {loading ? '⏳ Starting...' : '▶️ Start Execution'}
        </button>
        {instanceId && (
          <p style={{ marginTop: '8px', color: 'green' }}>
            ✅ Instance Started: <code>{instanceId}</code>
          </p>
        )}
      </section>

      {/* Step 3: Monitor Progress */}
      <section style={{ marginBottom: '24px', padding: '16px', background: '#f5f5f5', borderRadius: '4px' }}>
        <h2>Step 3: Monitor Progress</h2>
        <p>Check the current step and status.</p>
        <button
          onClick={handlePollStatus}
          disabled={loading || step !== 'monitor' || !instanceId}
          style={{ padding: '10px 20px', fontSize: '16px', cursor: 'pointer' }}
        >
          {loading ? '⏳ Polling...' : '📊 Poll Status'}
        </button>
        {status && (
          <div style={{ marginTop: '16px', padding: '12px', background: 'white', borderRadius: '4px' }}>
            <StepTimeline status={status} />
            <EventLog instanceId={instanceId} />
          </div>
        )}
      </section>

      {/* Step 4: Approve */}
      <section style={{ marginBottom: '24px', padding: '16px', background: '#f5f5f5', borderRadius: '4px' }}>
        <h2>Step 4: Approve Step 3 (Manager)</h2>
        <p>Simulate manager approval to proceed to final step.</p>
        <button
          onClick={handleApprove}
          disabled={loading || step !== 'monitor' || status?.current_step !== 3}
          style={{ padding: '10px 20px', fontSize: '16px', cursor: 'pointer' }}
        >
          {loading ? '⏳ Approving...' : '👍 Approve'}
        </button>
      </section>

      {/* Done */}
      {step === 'done' && (
        <section style={{ marginBottom: '24px', padding: '16px', background: '#e8f5e9', borderRadius: '4px', border: '2px solid #4caf50' }}>
          <h2>🎉 Demo Complete!</h2>
          <p>All 4 steps executed successfully. BP workflow works end-to-end!</p>
          {status && (
            <div>
              <p><strong>Total Duration:</strong> {calculateDuration(status.started_at, new Date().toISOString())}</p>
              <p><strong>Steps Completed:</strong> {status.current_step}/4</p>
            </div>
          )}
        </section>
      )}
    </div>
  )
}

function calculateDuration(start: string, end: string): string {
  const ms = new Date(end).getTime() - new Date(start).getTime()
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  if (hours > 0) return `${hours}h ${minutes % 60}m`
  if (minutes > 0) return `${minutes}m ${seconds % 60}s`
  return `${seconds}s`
}
```

---

## 📌 Task 6: Unit Test Template

### **bp_executor_test.go** (Starter)

```go
package temporal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

func TestLoadBPInstanceActivity(t *testing.T) {
	// Arrange
	mockDB := setupMockDB(t)
	ctx := context.WithValue(context.Background(), "db", mockDB)

	// Act
	instance, err := LoadBPInstanceActivity(ctx, "test-instance-123")

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, instance)
	assert.Equal(t, "test-instance-123", instance.InstanceID)
	assert.Equal(t, "pending", instance.Status)
}

func TestExecuteBPStepActivity(t *testing.T) {
	tests := []struct {
		name           string
		stepType       string
		expectedStatus string
	}{
		{"data_entry", "data_entry", "completed"},
		{"validate", "validate", "completed"},
		{"approve", "approve", "completed"},
		{"notify", "notify", "completed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := BPStepConfig{
				StepOrder: 1,
				StepType:  tt.stepType,
				StepName:  "Test Step",
			}

			result, err := ExecuteBPStepActivity(context.Background(), "instance-1", step, map[string]interface{}{})

			require.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Equal(t, 2, result.NextStep) // Default: next step
		})
	}
}

func TestUpdateBPInstanceStepActivity(t *testing.T) {
	mockDB := setupMockDB(t)
	ctx := context.WithValue(context.Background(), "db", mockDB)

	err := UpdateBPInstanceStepActivity(ctx, "instance-1", 2, "in_progress")

	require.NoError(t, err)
}

func setupMockDB(t *testing.T) interface{} {
	// TODO: Implement mock database
	return nil
}
```

---

## Summary

These templates provide 80% of the boilerplate needed for Tasks 4-6. Fill in your specific logic and you're done!

**Next Steps:**
1. Copy templates to files
2. Customize for your use case
3. Test locally
4. Deploy!

Ready to ship! 🚀
