import React from 'react';

// Types mirroring the Go structs
type ViewField = {
  label: string;
  attr: string;
  component: string;
  readOnly?: boolean;
  required?: boolean;
  helpText?: string;
};

type UIView = {
  id: string;
  type: 'Form' | 'Table' | 'Dashboard';
  sections: ViewField[][];
  dataSource: string;
  actions?: string[];
  theme?: Record<string, string>;
};

interface MetaRendererProps {
  view: UIView;
  data: Record<string, any>;
  onChange: (attr: string, val: any) => void;
  onAction?: (action: string) => void;
}

export const MetaRenderer: React.FC<MetaRendererProps> = ({ view, data, onChange, onAction }) => {
  const token = (k: string, fallback: string) => view.theme?.[k] || fallback;

  const renderField = (f: ViewField) => {
    const val = data[f.attr];
    
    switch (f.component) {
      case 'Text':
        return (
          <input
            type="text"
            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
            style={{ color: token('textColor', '#111') }}
            value={val ?? ''}
            readOnly={f.readOnly}
            onChange={(e) => onChange(f.attr, e.target.value)}
          />
        );
      case 'Number':
        return (
          <input
            type="number"
            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
            value={val ?? ''}
            readOnly={f.readOnly}
            onChange={(e) => onChange(f.attr, Number(e.target.value))}
          />
        );
      case 'Date':
        return (
          <input
            type="date"
            className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
            value={val ?? ''}
            readOnly={f.readOnly}
            onChange={(e) => onChange(f.attr, e.target.value)}
          />
        );
      case 'Select':
         // Simplified Select for demo
         return (
            <select
                className="mt-1 block w-full border border-gray-300 rounded-md shadow-sm p-2"
                value={val ?? ''}
                disabled={f.readOnly}
                onChange={(e) => onChange(f.attr, e.target.value)}
            >
                <option value="">Select...</option>
                <option value="Option A">Option A</option>
                <option value="Option B">Option B</option>
            </select>
         );
      default:
        return <span className="text-red-500">Unsupported component: {f.component}</span>;
    }
  };

  return (
    <div className="p-6 bg-white rounded-lg shadow-md" style={{ fontFamily: token('font', 'system-ui') }}>
      <h2 className="text-xl font-bold mb-4 text-gray-800">{view.type} View</h2>
      
      {view.sections.map((row, i) => (
        <div 
            key={i} 
            className="grid gap-4 mb-4"
            style={{ 
                gridTemplateColumns: `repeat(${row.length}, 1fr)`, 
                gap: token('gap', '16px') 
            }}
        >
          {row.map((f, j) => (
            <div key={j} className="flex flex-col">
              <label className="block text-sm font-medium text-gray-700 mb-1">
                {f.label}{f.required ? <span className="text-red-500 ml-1">*</span> : ''}
              </label>
              {renderField(f)}
              {f.helpText && <small className="text-gray-500 mt-1">{f.helpText}</small>}
            </div>
          ))}
        </div>
      ))}

      {view.actions && view.actions.length > 0 && (
          <div className="mt-6 flex space-x-3 border-t pt-4">
              {view.actions.map(action => (
                  <button
                    key={action}
                    onClick={() => onAction && onAction(action)}
                    className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 transition-colors"
                  >
                      {action}
                  </button>
              ))}
          </div>
      )}
    </div>
  );
};
