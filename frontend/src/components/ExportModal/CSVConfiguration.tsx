// components/ExportModal/CSVConfiguration.tsx
import type { FC } from 'react';

interface CSVConfigurationProps {
  delimiter: string;
  onDelimiterChange: (delimiter: string) => void;
}

export const CSVConfiguration: FC<CSVConfigurationProps> = ({
  delimiter,
  onDelimiterChange
}) => {
  const delimiters = [
    { label: 'Comma (,)', value: ',' },
    { label: 'Semicolon (;)', value: ';' },
    { label: 'Tab', value: '\t' },
    { label: 'Pipe (|)', value: '|' }
  ];

  return (
    <div className="bg-gray-50 rounded-xl p-6">
      <h3 className="text-lg font-semibold text-gray-900 mb-4 flex items-center space-x-2">
        <span>⚙️</span>
        <span>CSV Configuration</span>
      </h3>
      
      <div className="grid grid-cols-4 gap-3 mb-4">
        {delimiters.map(option => (
          <label key={option.value} className="relative cursor-pointer">
            <input
              type="radio"
              name="delimiter"
              value={option.value}
              checked={delimiter === option.value}
              onChange={(e) => onDelimiterChange(e.target.value)}
              className="sr-only"
            />
            <div className={`p-3 border-2 rounded-lg text-center text-sm font-medium transition-all ${
              delimiter === option.value 
                ? 'border-blue-500 bg-blue-50 text-blue-700' 
                : 'border-gray-200 text-gray-600 hover:border-gray-300'
            }`}>
              {option.label}
            </div>
          </label>
        ))}
      </div>
      
      <input
        type="text"
        value={delimiter}
        onChange={(e) => onDelimiterChange(e.target.value)}
        className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 text-sm"
        placeholder="Custom delimiter..."
        maxLength={5}
      />
    </div>
  );
};