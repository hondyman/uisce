import { useState } from 'react';
import { ScriptDetail, ScriptVersion } from '../../types/scripts';
import { VersionDiffViewer } from './VersionDiffViewer';

export function ScriptDetailDrawer({
  script, onClose, onImpact, onAssignSteward, onRefresh
}: {
  script: ScriptDetail;
  onClose: () => void;
  onImpact: () => void;
  onAssignSteward: () => void;
  onRefresh: (id: string) => void;
}) {
  const [left, setLeft] = useState<ScriptVersion | null>(script.versions[script.versions.length - 2] || null);
  const [right, setRight] = useState<ScriptVersion>(script.versions[script.versions.length - 1]);

  const canCertify = script.state === 'draft';
  const canPublish = script.state === 'certified';

  const certify = async () => {
    await fetch(`/api/scripts/${script.id}/certify`, { method: 'POST' });
    onRefresh(script.id);
  };
  const publish = async () => {
    await fetch(`/api/scripts/${script.id}/publish`, { method: 'POST' });
    onRefresh(script.id);
  };

  return (
    <div className="drawer">
      <header>
        <h2>{script.name}</h2>
        <button onClick={onClose}>Close</button>
      </header>

      <section>
        <h3>Metadata</h3>
        <ul>
          <li><strong>State:</strong> {script.state}</li>
          <li><strong>Scope:</strong> {script.scope}</li>
          <li><strong>Latest version:</strong> {script.latestVersion}</li>
          <li><strong>Steward:</strong> {script.steward || '—'} <button onClick={onAssignSteward}>Assign</button></li>
          <li><strong>Tags:</strong> {(script.domainTags ?? []).join(', ')}</li>
        </ul>
      </section>

      <section>
        <h3>Version history</h3>
        <div className="history">
          {script.versions.map(v => (
            <div key={v.version} className={`version ${v.version === script.latestVersion ? 'latest' : ''}`}>
              <span>{v.version}</span>
              <span>by {v.createdBy} on {new Date(v.createdAt).toLocaleString()}</span>
              <span>hash {v.hash.slice(0, 8)}…</span>
              {v.tests && <span className={`test ${v.tests.pass ? 'pass' : 'fail'}`}>{v.tests.pass ? 'Tests pass' : 'Tests fail'}</span>}
              {v.approvals && v.approvals.length > 0 && <span>approved by {v.approvals.map(a => a.by).join(', ')}</span>}
            </div>
          ))}
        </div>
      </section>

      <section>
        <h3>Diff</h3>
        <div className="diffControls">
          <label htmlFor="left-version-select">Left:</label>
          <select id="left-version-select" aria-label="Left version" title="Select left version" value={left?.version || ''} onChange={e => setLeft(script.versions.find(v => v.version === e.target.value) || null)}>
            <option value="">None</option>
            {script.versions.map(v => <option key={v.version} value={v.version}>{v.version}</option>)}
          </select>
          <label htmlFor="right-version-select">Right:</label>
          <select id="right-version-select" aria-label="Right version" title="Select right version" value={right.version} onChange={e => setRight(script.versions.find(v => v.version === e.target.value)!)}>
            {script.versions.map(v => <option key={v.version} value={v.version}>{v.version}</option>)}
          </select>
        </div>
        <VersionDiffViewer left={left?.content || ''} right={right.content} />
      </section>

      <section>
        <h3>Lineage</h3>
        <ul>
          {script.lineage.attachedTo.map(a => <li key={a}>{a}</li>)}
        </ul>
        <button onClick={onImpact}>View impact</button>
      </section>

      <footer className="actions">
        <button disabled={!canCertify} onClick={certify}>Submit for certification</button>
        <button disabled={!canPublish} onClick={publish}>Publish</button>
      </footer>
    </div>
  );
}
