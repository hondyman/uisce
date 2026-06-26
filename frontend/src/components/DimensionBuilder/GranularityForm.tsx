// src/components/DimensionBuilder/GranularityForm.tsx
import { useState } from 'react';
import { Plus } from 'lucide-react';
import { Granularity, timeUnits } from './types';

interface GranularityFormProps {
  onAdd: (granularity: Omit<Granularity, 'id'>) => void;
}

export function GranularityForm({ onAdd }: GranularityFormProps) {
  const [name, setName] = useState('');
  const [intervalValue, setIntervalValue] = useState('');
  const [intervalUnit, setIntervalUnit] = useState('day');
  const [offsetValue, setOffsetValue] = useState('');
  const [offsetUnit, setOffsetUnit] = useState('day');
  const [origin, setOrigin] = useState('');
  const [title, setTitle] = useState('');

  const handleAdd = () => {
    if (!name || !intervalValue) return;
    const interval = `${intervalValue} ${intervalUnit}`;
    const offset = offsetValue ? `${offsetValue} ${offsetUnit}` : undefined;
    onAdd({ name, interval, offset, origin, title });
    
    // Reset form
    setName('');
    setIntervalValue('');
    setIntervalUnit('day');
    setOffsetValue('');
    setOffsetUnit('day');
    setOrigin('');
    setTitle('');
  };

  return (
    <div className="space-y-3 bg-gray-50 p-4 rounded-lg">
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">Name *</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
            placeholder="quarter_hour"
          />
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
            placeholder="Human-readable title"
          />
        </div>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">Interval *</label>
          <div className="flex gap-2">
            <input
              type="number"
              value={intervalValue}
              onChange={(e) => setIntervalValue(e.target.value)}
              className="flex-1 p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
              placeholder="15"
            />
            <select
              value={intervalUnit}
              onChange={(e) => setIntervalUnit(e.target.value)}
              className="p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
            >
              {timeUnits.map(unit => (
                <option key={unit} value={unit}>{unit}</option>
              ))}
            </select>
          </div>
        </div>
        <div>
          <label className="block text-xs font-medium text-gray-500 mb-1">Offset</label>
          <div className="flex gap-2">
            <input
              type="number"
              value={offsetValue}
              onChange={(e) => setOffsetValue(e.target.value)}
              className="flex-1 p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
              placeholder="-1"
            />
            <select
              value={offsetUnit}
              onChange={(e) => setOffsetUnit(e.target.value)}
              className="p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
            >
              {timeUnits.map(unit => (
                <option key={unit} value={unit}>{unit}</option>
              ))}
            </select>
          </div>
        </div>
      </div>
      
      <div>
        <label className="block text-xs font-medium text-gray-500 mb-1">Origin (ISO 8601)</label>
        <input
          type="text"
          value={origin}
          onChange={(e) => setOrigin(e.target.value)}
          className="w-full p-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-green-500 text-gray-900"
          placeholder="2024-01-01T00:00:00.000Z"
        />
      </div>
      
      <button
        onClick={handleAdd}
        disabled={!name || !intervalValue}
        className="bg-green-500 hover:bg-green-600 disabled:opacity-50 text-white px-4 py-2 rounded-lg transition-colors flex items-center gap-2"
      >
        <Plus className="w-4 h-4" />
        Add Granularity
      </button>
    </div>
  );
}
