// React import removed — JSX runtime handles createElement
import { ExportOptions } from '../../types/ExportTypes'; // Ensure this path is correct

interface AdvancedOptionsProps {
  options: Pick<ExportOptions, 'includeRelationships' | 'includeIndexes' | 'includeComments'>;
  onOptionChange: (key: keyof ExportOptions, value: boolean) => void;
}

const optionDetails = [
  { key: 'includeRelationships' as const, icon: '🔗', label: 'Include Relationships', desc: 'Export foreign keys and constraints' },
  { key: 'includeIndexes' as const, icon: '⚡', label: 'Include Indexes', desc: 'Export database indexes information' },
  { key: 'includeComments' as const, icon: '💬', label: 'Include Comments', desc: 'Export table and column descriptions' },
];

export const AdvancedOptions: React.FC<AdvancedOptionsProps> = ({ options, onOptionChange }) => (
  <div>
    <div className="flex items-center space-x-3 mb-6">
      <div className="w-8 h-8 bg-green-600 text-white rounded-full flex items-center justify-center text-sm font-bold">3</div>
      <h2 className="text-xl font-semibold text-gray-900">Advanced Options</h2>
    </div>
    <div className="space-y-4">
      {optionDetails.map(({ key, icon, label, desc }) => (
        <label key={key} className="flex items-center space-x-4 cursor-pointer p-4 border border-gray-200 rounded-xl hover:bg-gray-50 transition-colors">
          <input
            type="checkbox"
            checked={options[key]}
            onChange={(e) => onOptionChange(key, e.target.checked)}
            className="w-5 h-5 text-green-600 border-gray-300 rounded focus:ring-green-500"
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