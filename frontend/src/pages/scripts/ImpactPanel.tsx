import { ImpactReport } from '../../types/scripts';
import { useNotification } from '../../hooks/useNotification';

export function ImpactPanel({ report, onClose }: { report: ImpactReport; onClose: () => void }) {
  return (
    <div className="panel">
      <header>
        <h3>Downstream impact</h3>
        <button onClick={onClose}>Close</button>
      </header>

      <section>
        <h4>Impacted bundles</h4>
        <ul>
          {(report.impactedBundles ?? []).map(b => (
            <li key={b.id}>
              <strong>{b.name}</strong> ({b.version}) — {b.state}
              <button onClick={() => window.location.assign(`/bundles/${b.id}`)}>Open bundle</button>
            </li>
          ))}
        </ul>
      </section>

      <section>
        <h4>Impacted views</h4>
        <ul>
          {(report.impactedViews ?? []).map(v => (
            <li key={`${v.bundleId}:${v.name}`}>
              <strong>{v.name}</strong> in {v.bundleName} — {v.state}
              <button onClick={() => window.location.assign(`/bundles/${v.bundleId}/views/${v.name}`)}>Open view</button>
            </li>
          ))}
        </ul>
      </section>

      <section>
        <h4>Impacted objects</h4>
        <ul>
          {(report.impactedObjects ?? []).map(o => (
            <li key={`${o.type}:${o.id}`}>
              {o.type}:{o.id} {o.bundleId ? `in bundle ${o.bundleId}` : ''}
            </li>
          ))}
        </ul>
      </section>

      <footer className="actions">
        <button onClick={() => { const notification = useNotification(); notification.info('Exporting report...'); }}>Export report</button>
      </footer>
    </div>
  );
}
