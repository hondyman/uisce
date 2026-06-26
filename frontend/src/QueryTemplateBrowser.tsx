import { useState, useEffect } from 'react';
import { listQueryTemplates } from './api';
import type { QueryTemplateMeta } from './types';

const Badge = ({ children }: { children: React.ReactNode }) => <span className="badge">{children}</span>;

/* eslint-disable no-unused-vars */
/* eslint-disable @typescript-eslint/no-unused-vars */
interface QueryTemplateBrowserProps {
  datasourceId: string;
  onSelect: (template: QueryTemplateMeta) => void;
}
/* eslint-enable @typescript-eslint/no-unused-vars */
/* eslint-enable no-unused-vars */

export default function QueryTemplateBrowser({ datasourceId, onSelect }: QueryTemplateBrowserProps) {
  const [templates, setTemplates] = useState<QueryTemplateMeta[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    listQueryTemplates(datasourceId)
      .then(setTemplates)
  .catch((e) => { import('./utils/devLogger').then(({ devError }) => devError(e)).catch(() => {}); })
      .finally(() => setLoading(false));
  }, [datasourceId]);

  if (loading) {
    return <div>Loading templates...</div>;
  }

  return (
    <div className="template-browser">
      <h4>Query Templates</h4>
      <div className="template-grid">
        {templates.map((t) => (
          <div key={t.id} className="template-card" onClick={() => onSelect(t)} title={t.description}>
            <strong>{t.name}</strong>
            <small>{t.description}</small>
            <div className="card-footer">
              {t.certified && <Badge>Certified</Badge>}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}