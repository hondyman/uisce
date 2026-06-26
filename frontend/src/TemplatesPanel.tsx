import type { ViewMeta, QueryState, ViewTemplate } from './types';
import { getViewIdentifier } from './types/views';

interface TemplatesPanelProps {
  view: ViewMeta;
  onSelectTemplate: (query: Partial<QueryState>) => void;
}

export default function TemplatesPanel({ view, onSelectTemplate }: TemplatesPanelProps) {
  const templates = view.templates ?? [];
  if (templates.length === 0) return null;

  const id = getViewIdentifier(view);
  const displayName = (view as any).title || view.name || (id ? String(id).slice(0,8) : 'Unnamed View');
  return (
    <div className="templates-panel">
      <h4>Templates for {displayName}</h4>
      <ul>
        {templates.map((template: ViewTemplate) => (
          <li key={template.name}>
            <button onClick={() => onSelectTemplate(template.query || {})} title={template.description}>
              {template.name}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}