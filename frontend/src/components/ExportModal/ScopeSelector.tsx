// components/ExportModal/ScopeSelector.tsx
import type { FC } from 'react';
import { ExportOptions } from '../../types/ExportTypes';

interface ScopeSelectorProps {
  selectedScope: ExportOptions['exportScope'];
  onScopeChange: (scope: ExportOptions['exportScope']) => void;
}

export const ScopeSelector: FC<ScopeSelectorProps> = ({
  selectedScope,
  onScopeChange
}) => {
  const scopes = [
    { 
      value: 'all' as const, 
      icon: '📦',
      label: 'Export Everything', 
      desc: 'All schemas, tables, and relationships'
    },
    { 
      value: 'schemas' as const, 
      icon: '📁',
      label: 'Select Schemas', 
      desc: 'Choose specific database schemas'
    },
    { 
      value: 'tables' as const, 
      icon: '📋',
      label: 'Select Tables', 
      desc: 'Pick individual tables'
    }
  ];

  return (
    <div>
      <div className="flex items-center space-x-3 mb-6">
        <div className="w-8 h-8 bg-blue-500 text-white rounded-full flex items-center justify-center text-sm font-bold">2</div>
        <h2 className="text-xl font-semibold text-gray-900">Data Scope</h2>
      </div>
      
      <div className="space-y-3">
        {scopes.map(({ value, icon, label, desc }) => (
          <label key={value} className={`flex items-center space-x-4 cursor-pointer p-4 border-2 rounded-xl transition-all duration-200 ${
            selectedScope === value 
              ? 'border-blue-500 bg-blue-50' 
              : 'border-gray-200 hover:bg-gray-50'
          }`}>
            <input
              type="radio"
              name="scope"
              value={value}
              checked={selectedScope === value}
              onChange={(e) => onScopeChange(e.target.value as ExportOptions['exportScope'])}
              className="w-4 h-4 text-blue-600 border-gray-300 focus:ring-blue-500"
            />
            <span className="text-2xl">{icon}</span>
            <div className="flex-1">
              <div className="font-semibold text-gray-900">{label}</div>
              <div className="text-sm text-gray-600">{desc}</div>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
};
