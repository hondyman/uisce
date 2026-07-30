import React from 'react';
import { Calculate, VpnKey, Storage } from '@mui/icons-material';

export interface FieldMeta {
  field_name: string;
  semantic_term_key: string;
  display_label: string;
  data_type: 'string' | 'number' | 'date' | 'boolean';
  field_role: 'DIMENSION' | 'MEASURE' | 'KEY';
  binding_status: 'RESOLVED' | 'UNRESOLVED' | 'CALCULATED';
  is_editable: boolean;
  column_width: number;
  component_hint: string;
}

interface Props {
  fields: FieldMeta[];
  data: any[];
}

export const DynamicBODataGrid: React.FC<Props> = ({ fields, data }) => {
  const renderCellContent = (field: FieldMeta, value: any) => {
    if (value === null || value === undefined) {
      return <span className="text-slate-600 font-mono text-xs">—</span>;
    }

    switch (field.data_type) {
      case 'number':
        return (
          <span className="font-mono text-emerald-400 font-medium">
            {typeof value === 'number' ? value.toLocaleString(undefined, { minimumFractionDigits: 2 }) : value}
          </span>
        );
      case 'date':
        return <span className="font-mono text-slate-300 text-xs">{new Date(value).toLocaleDateString()}</span>;
      default:
        return <span className="text-slate-200 text-xs font-sans">{String(value)}</span>;
    }
  };

  return (
    <div className="overflow-x-auto bg-slate-800/80 border border-slate-700/60 rounded-xl shadow-xl">
      <table className="w-full text-left border-collapse">
        <thead>
          <tr className="bg-slate-950/80 border-b border-slate-700 text-[11px] font-bold text-slate-400 uppercase tracking-wider">
            {fields.map((field) => (
              <th key={field.field_name} style={{ width: field.column_width }} className="py-3.5 px-4">
                <div className="flex items-center gap-1.5">
                  {field.field_role === 'KEY' && <VpnKey sx={{ fontSize: 12 }} className="text-amber-400" />}
                  {field.binding_status === 'CALCULATED' && <Calculate sx={{ fontSize: 12 }} className="text-sky-400" />}
                  {field.binding_status === 'RESOLVED' && <Storage sx={{ fontSize: 12 }} className="text-slate-500" />}
                  <span>{field.display_label}</span>
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-800 text-xs">
          {data.length === 0 ? (
            <tr>
              <td colSpan={fields.length} className="text-center py-8 text-slate-500 font-mono">
                No records available for active Business Object context.
              </td>
            </tr>
          ) : (
            data.map((row, idx) => (
              <tr key={idx} className="hover:bg-slate-700/30 transition-colors">
                {fields.map((field) => (
                  <td key={field.field_name} className="py-3 px-4">
                    {renderCellContent(field, row[field.field_name])}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
};
