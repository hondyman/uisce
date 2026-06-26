// src/components/DimensionBuilder/CaseEditor.tsx
import { useState } from 'react';
import { Plus, Save, X, Trash2 } from 'lucide-react';
import { Dimension, CaseWhen, CaseElse } from './types';

interface CaseEditorProps {
  caseObj?: Dimension['case'];
  onUpdate: (caseObj?: Dimension['case']) => void;
}

export function CaseEditor({ caseObj, onUpdate }: CaseEditorProps) {
  const [whenList, setWhenList] = useState<CaseWhen[]>(caseObj?.when || []);
  const [elseLabel, setElseLabel] = useState<CaseElse>(caseObj?.else || { label: 'Unknown' });
  const [useDynamicLabel, setUseDynamicLabel] = useState(false);

  const addWhen = () => {
    setWhenList(prev => [...prev, { sql: '', label: '' }]);
  };

  const updateWhen = (index: number, updates: Partial<CaseWhen>) => {
    setWhenList(prev => prev.map((w, i) => i === index ? { ...w, ...updates } : w));
  };

  const removeWhen = (index: number) => {
    setWhenList(prev => prev.filter((_, i) => i !== index));
  };

  const saveCase = () => {
    onUpdate({ when: whenList, else: elseLabel });
  };

  const clearCase = () => {
    setWhenList([]);
    setElseLabel({ label: 'Unknown' });
    onUpdate(undefined);
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-2">
        <label className="flex items-center gap-2 cursor-pointer">
          <input
            type="checkbox"
            checked={useDynamicLabel}
            onChange={(e) => setUseDynamicLabel(e.target.checked)}
            className="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
          />
          <span className="text-sm font-medium text-gray-700">Use Dynamic Labels (SQL)</span>
        </label>
      </div>

      {/* When Conditions */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="font-medium">When Conditions</span>
          <button
            onClick={addWhen}
            className="bg-blue-500 hover:bg-blue-600 text-white px-3 py-1 rounded text-sm transition-colors flex items-center gap-1"
          >
            <Plus className="w-3 h-3" />
            Add Condition
          </button>
        </div>
        
        {whenList.map((w, index) => (
          <div key={index} className="bg-blue-50 border border-blue-200 p-3 rounded-lg flex gap-3">
            <div className="flex-1">
              <label className="block text-xs font-medium text-gray-500 mb-1">SQL Condition</label>
              <input
                type="text"
                value={w.sql}
                onChange={(e) => updateWhen(index, { sql: e.target.value })}
                className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-gray-900"
              />
            </div>
            <div className="flex-1">
              <label className="block text-xs font-medium text-gray-500 mb-1">Label</label>
              {useDynamicLabel ? (
                <input
                  type="text"
                  value={typeof w.label === 'string' ? w.label : w.label.sql}
                  onChange={(e) => updateWhen(index, { label: { sql: e.target.value } })}
                  className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-gray-900"
                />
              ) : (
                <input
                  type="text"
                  value={typeof w.label === 'string' ? w.label : ''}
                  onChange={(e) => updateWhen(index, { label: e.target.value })}
                  className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 text-gray-900"
                />
              )}
            </div>
            <button
              onClick={() => removeWhen(index)}
              className="text-red-500 hover:text-red-700 p-2"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          </div>
        ))}
      </div>

      {/* Else */}
      <div className="space-y-2">
        <span className="font-medium">Else Label</span>
        {useDynamicLabel ? (
          <input
            type="text"
            value={typeof elseLabel.label === 'string' ? elseLabel.label : elseLabel.label.sql}
            onChange={(e) => setElseLabel({ label: { sql: e.target.value } })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 font-mono text-gray-900"
          />
        ) : (
          <input
            type="text"
            value={typeof elseLabel.label === 'string' ? elseLabel.label : ''}
            onChange={(e) => setElseLabel({ label: e.target.value })}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 text-gray-900"
          />
        )}
      </div>

      <div className="flex gap-3 pt-3 border-t border-gray-200">
        <button
          onClick={saveCase}
          className="bg-blue-500 hover:bg-blue-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <Save className="w-4 h-4" />
          Apply Case
        </button>
        <button
          onClick={clearCase}
          className="bg-gray-500 hover:bg-gray-600 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
        >
          <X className="w-4 h-4" />
          Clear Case
        </button>
      </div>
    </div>
  );
}
