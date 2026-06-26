// components/ExportModal/FormatSelector.tsx
// React import removed (automatic JSX runtime in use)
import { ExportOptions } from '../../types/ExportTypes';

interface FormatSelectorProps {
  selectedFormat: ExportOptions['format'];
  onFormatChange: (format: ExportOptions['format']) => void;
}

export const FormatSelector: React.FC<FormatSelectorProps> = ({
  selectedFormat,
  onFormatChange
}) => {
  const formats = [
    {
      format: 'csv' as const,
      label: 'CSV',
      sublabel: 'Excel Ready',
      icon: '📊',
      color: 'blue',
    },
    {
      format: 'json' as const,
      label: 'JSON',
      sublabel: 'API Friendly',
      icon: '⚡',
      color: 'green',
    },
    {
      format: 'xml' as const,
      label: 'XML',
      sublabel: 'Enterprise',
      icon: '🔧',
      color: 'purple',
    },
  ];

  return (
    <div>
      <div className="flex items-center space-x-3 mb-6">
        <div className="w-8 h-8 bg-blue-500 text-white rounded-full flex items-center justify-center text-sm font-bold">1</div>
        <h2 className="text-xl font-semibold text-gray-900">Export Format</h2>
      </div>
      
      <div className="grid grid-cols-3 gap-4">
  {formats.map(({ format, label, sublabel, icon }) => (
          <label key={format} className="relative cursor-pointer group">
            <input
              type="radio"
              name="format"
              value={format}
              checked={selectedFormat === format}
              onChange={(e) => onFormatChange(e.target.value as ExportOptions['format'])}
              className="sr-only"
            />
            <div className={`p-4 border-2 rounded-xl text-center transition-all duration-200 ${
              selectedFormat === format 
                ? format === 'csv' ? 'border-blue-500 bg-blue-50 shadow-lg' :
                  format === 'json' ? 'border-green-500 bg-green-50 shadow-lg' :
                  'border-purple-500 bg-purple-50 shadow-lg'
                : 'border-gray-200 hover:border-gray-300 hover:shadow-md'
            }`}>
              <div className="text-3xl mb-2">{icon}</div>
              <div className={`font-semibold ${
                selectedFormat === format 
                  ? format === 'csv' ? 'text-blue-700' :
                    format === 'json' ? 'text-green-700' :
                    'text-purple-700'
                  : 'text-gray-700'
              }`}>
                {label}
              </div>
              <div className={`text-xs mt-1 ${
                selectedFormat === format 
                  ? format === 'csv' ? 'text-blue-600' :
                    format === 'json' ? 'text-green-600' :
                    'text-purple-600'
                  : 'text-gray-500'
              }`}>
                {sublabel}
              </div>
            </div>
          </label>
        ))}
      </div>
    </div>
  );
};
