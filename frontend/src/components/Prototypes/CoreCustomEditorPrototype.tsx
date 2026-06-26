import React, { useMemo, useState } from 'react';
import yaml from 'js-yaml';
import SqlMonacoEditor from '../SqlMonacoEditor';
import './CoreCustomEditor.css';

type Measure = {
  id: string;
  name: string;
  sql: string;
  unit?: string;
};

type Cube = {
  id: string;
  name: string;
  measures: Measure[];
};

type CustomChange = {
  id: string; // measure id or new id
  action: 'override' | 'remove' | 'add';
  payload?: Partial<Measure>;
};

const sampleCore: Cube = {
  id: 'cube_sales',
  name: 'Sales',
  measures: [
    { id: 'm_revenue', name: 'Revenue', sql: 'SUM(${table}.amount)', unit: 'USD' },
    { id: 'm_orders', name: 'Orders', sql: 'COUNT(${table}.id)' },
    { id: 'm_discount', name: 'Discount', sql: 'SUM(${table}.discount)', unit: 'USD' },
  ],
};

export const CoreCustomEditorPrototype: React.FC = () => {
  const [core] = useState<Cube>(sampleCore);
  const [customChanges, setCustomChanges] = useState<CustomChange[]>([
    // example: override revenue name
    //{ id: 'm_revenue', action: 'override', payload: { name: 'Revenue (net)' } }
  ]);

  const [selectedMeasureId, setSelectedMeasureId] = useState<string | null>(core.measures[0].id);
  const selectedCore = useMemo(() => core.measures.find(m => m.id === selectedMeasureId) || null, [core, selectedMeasureId]);
  const selectedCustom = useMemo(() => customChanges.find(c => c.id === selectedMeasureId) || null, [customChanges, selectedMeasureId]);

  const mergedMeasures = useMemo(() => {
    const map = new Map<string, Measure>();
    core.measures.forEach(m => map.set(m.id, { ...m }));
    customChanges.forEach(c => {
      if (c.action === 'remove') {
        map.delete(c.id);
      } else if (c.action === 'override') {
        const base = map.get(c.id);
        map.set(c.id, { ...(base || { id: c.id, name: c.id, sql: '' }), ...(c.payload || {}) });
      } else if (c.action === 'add' && c.payload) {
        map.set(c.id, { id: c.id, name: (c.payload.name as string) || c.id, sql: (c.payload.sql as string) || '' });
      }
    });
    return Array.from(map.values());
  }, [core, customChanges]);

  const applyOverride = (id: string, patch: Partial<Measure>) => {
    setCustomChanges(prev => {
      const found = prev.find(p => p.id === id && p.action === 'override');
      if (found) {
        return prev.map(p => p.id === id && p.action === 'override' ? { ...p, payload: { ...(p.payload || {}), ...patch } } : p);
      }
      return [...prev, { id, action: 'override', payload: patch }];
    });
  };

  const markRemoved = (id: string) => {
    setCustomChanges(prev => [...prev.filter(p => p.id !== id), { id, action: 'remove' }]);
  };

  const addNewMeasure = () => {
    const id = `m_custom_${Math.random().toString(36).slice(2, 7)}`;
    setCustomChanges(prev => [...prev, { id, action: 'add', payload: { id, name: 'New measure', sql: '', unit: '' } }]);
    setSelectedMeasureId(id);
  };

  const revertChange = (id: string) => {
    setCustomChanges(prev => prev.filter(p => p.id !== id));
  };

  // For the diff view, compute YAML for core and custom
  const coreYaml = selectedCore ? yaml.dump(selectedCore) : '';
  const customYaml = (() => {
    if (!selectedMeasureId) return '';
    const change = customChanges.find(c => c.id === selectedMeasureId);
    if (!change) return coreYaml;
    if (change.action === 'remove') return '# REMOVED IN CUSTOM';
    if (change.action === 'add') return yaml.dump(change.payload || {});
    return yaml.dump({ ...(selectedCore || {}), ...(change.payload || {}) });
  })();

  return (
    <div className="proto-root">
      <div className="proto-left">
        <div className="proto-header">Model Tree</div>
        <div className="tree">
          <div className="tree-cube">{core.name}</div>
          <div className="tree-measures">
            {core.measures.map(m => {
              const ch = customChanges.find(c => c.id === m.id);
              const removed = ch?.action === 'remove';
              const overridden = ch?.action === 'override';
              return (
                <div
                  key={m.id}
                  className={`tree-item ${selectedMeasureId === m.id ? 'selected' : ''} ${removed ? 'removed' : ''} ${overridden ? 'overridden' : ''}`}
                  onClick={() => setSelectedMeasureId(m.id)}
                >
                  <span className="item-name">{m.name}</span>
                  <div className="item-badges">
                    {removed ? <span className="badge removed">removed</span> : overridden ? <span className="badge overridden">overridden</span> : <span className="badge core">core</span>}
                  </div>
                </div>
              );
            })}
            {/* custom added measures */}
            {customChanges.filter(c => c.action === 'add').map(c => (
              <div key={c.id} className={`tree-item ${selectedMeasureId === c.id ? 'selected' : ''} custom`} onClick={() => setSelectedMeasureId(c.id)}>
                <span className="item-name">{(c.payload?.name as string) || c.id}</span>
                <div className="item-badges"><span className="badge custom">custom</span></div>
              </div>
            ))}
          </div>
          <div className="tree-actions">
            <button className="btn" onClick={addNewMeasure}>+ New measure</button>
          </div>
        </div>
      </div>

      <div className="proto-center">
        <div className="proto-header">Definition Editor</div>
        {selectedMeasureId ? (
          <MeasureEditor
            core={selectedCore}
            change={selectedCustom || undefined}
            onApply={(patch) => applyOverride(selectedMeasureId, patch)}
            onRemove={() => markRemoved(selectedMeasureId)}
            onRevert={() => revertChange(selectedMeasureId)}
            merged={mergedMeasures.find(m => m.id === selectedMeasureId) || null}
            coreYaml={coreYaml}
            customYaml={customYaml}
          />
        ) : (
          <div className="placeholder">Select a measure to edit</div>
        )}
      </div>

      <div className="proto-right">
        <div className="proto-header">Preview & Lineage</div>
        <div className="preview">
          <h4>Merged Measures</h4>
          <ul>
            {mergedMeasures.map(m => (
              <li key={m.id}>{m.name} <small className="muted">({m.unit || 'no unit'})</small></li>
            ))}
          </ul>

          <h4>Lineage (selected)</h4>
          <div className="lineage">
            {/* Simple illustrative SVG lineage */}
            <svg width="100%" height="120" viewBox="0 0 600 120">
              <circle cx="70" cy="60" r="20" className="node src" />
              <text x="70" y="60" textAnchor="middle" dominantBaseline="middle" className="node-label">table</text>
              <path d="M90 60 L200 60" stroke="#888" strokeWidth={2} fill="none" markerEnd="url(#arrow)" />
              <circle cx="240" cy="60" r="20" className="node core" />
              <text x="240" y="60" textAnchor="middle" dominantBaseline="middle" className="node-label">core</text>
              <path d="M260 60 L380 60" stroke="#888" strokeWidth={2} fill="none" />
              <circle cx="420" cy="60" r="20" className="node custom" />
              <text x="420" y="60" textAnchor="middle" dominantBaseline="middle" className="node-label">custom</text>
              <defs>
                <marker id="arrow" markerWidth="10" markerHeight="6" refX="10" refY="3" orient="auto">
                      <path d="M0,0 L10,3 L0,6" className="arrow-path" />
                </marker>
              </defs>
            </svg>
          </div>
        </div>
      </div>
    </div>
  );
};

const MeasureEditor: React.FC<{
  core: Measure | null;
  change?: CustomChange;
  onApply: (patch: Partial<Measure>) => void;
  onRemove: () => void;
  onRevert: () => void;
  merged: Measure | null;
  coreYaml: string;
  customYaml: string;
}> = ({ core, change, onApply, onRemove, onRevert, merged: _merged, coreYaml, customYaml }) => {
  const [tab, setTab] = useState<'form' | 'code' | 'diff'>('form');
  const [formState, setFormState] = useState<Partial<Measure>>({ ...(change?.payload || core || {}) });
  const [errors, setErrors] = useState<string | null>(null);

  const handleApply = () => {
    if (!formState.name) return setErrors('Name is required');
    setErrors(null);
    onApply(formState as Partial<Measure>);
  };

  return (
    <div className="editor-root">
      <div className="editor-tabs">
        <button className={tab === 'form' ? 'active' : ''} onClick={() => setTab('form')}>Form</button>
        <button className={tab === 'code' ? 'active' : ''} onClick={() => setTab('code')}>Code</button>
        <button className={tab === 'diff' ? 'active' : ''} onClick={() => setTab('diff')}>Diff</button>
      </div>

      <div className="editor-body">
        {tab === 'form' && (
          <div className="form">
            <label>Name</label>
            <input placeholder="Enter measure name" value={formState.name || ''} onChange={(e) => setFormState(s => ({ ...s, name: e.target.value }))} />
            <label>SQL</label>
            <SqlMonacoEditor
              value={formState.sql || ''}
              onChange={(value) => setFormState(s => ({ ...s, sql: value }))}
              placeholder="SQL expression"
              height="100px"
            />
            <label>Unit</label>
            <input placeholder="Unit (e.g., USD)" value={formState.unit || ''} onChange={(e) => setFormState(s => ({ ...s, unit: e.target.value }))} />
            {errors && <div className="error">{errors}</div>}
            <div className="editor-actions">
              <button className="btn btn-primary" onClick={handleApply}>Apply override</button>
              <button className="btn btn-ghost" onClick={onRemove}>Mark removed</button>
              <button className="btn" onClick={onRevert}>Revert</button>
            </div>
          </div>
        )}

        {tab === 'code' && (
          <div className="code-view">
            <h5>Custom YAML</h5>
            <pre>{customYaml}</pre>
          </div>
        )}

        {tab === 'diff' && (
          <div className="diff-view">
            <div className="diff-col">
              <h5>Core</h5>
              <pre>{coreYaml}</pre>
            </div>
            <div className="diff-col">
              <h5>Custom</h5>
              <pre>{customYaml}</pre>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default CoreCustomEditorPrototype;
