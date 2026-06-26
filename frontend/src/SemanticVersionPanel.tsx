import { useState, useEffect } from 'react';
import { listSemanticViewVersions } from './api';
import type { SemanticViewVersion } from './types';

interface SemanticVersionPanelProps {
  viewName: string;
  onCompare: (from: number, to: number) => void;
}

export default function SemanticVersionPanel({ viewName, onCompare }: SemanticVersionPanelProps) {
  const [versions, setVersions] = useState<SemanticViewVersion[]>([]);
  const [fromVersion, setFromVersion] = useState<number | ''>('');
  const [toVersion, setToVersion] = useState<number | ''>('');

  useEffect(() => {
    listSemanticViewVersions(viewName).then(data => {
      setVersions(data);
      if (data.length > 1) {
        setFromVersion(data[1].version);
        setToVersion(data[0].version);
      }
    });
  }, [viewName]);

  const handleCompare = () => {
    if (fromVersion && toVersion) {
      onCompare(fromVersion, toVersion);
    }
  };

  return (
    <div className="semantic-version-panel">
      <h4>View Versions</h4>
      <div className="version-selectors">
        <select aria-label="From Version" value={fromVersion} onChange={e => setFromVersion(Number(e.target.value))}>
          {versions.map(v => <option key={v.version} value={v.version}>v{v.version}: {v.description}</option>)}
        </select>
        <span>vs</span>
        <select aria-label="To Version" value={toVersion} onChange={e => setToVersion(Number(e.target.value))}>
          {versions.map(v => <option key={v.version} value={v.version}>v{v.version}: {v.description}</option>)}
        </select>
      </div>
      <button onClick={handleCompare} disabled={!fromVersion || !toVersion || fromVersion === toVersion}>Compare</button>
    </div>
  );
}