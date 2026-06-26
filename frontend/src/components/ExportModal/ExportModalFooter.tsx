// components/ExportModal/ExportModalFooter.tsx
import type { FC } from 'react';

interface ExportModalFooterProps {
  estimatedSize: string;
  canExport: boolean;
  isExporting: boolean;
  onClose: () => void;
  onExport: () => void;
}

export const ExportModalFooter: FC<ExportModalFooterProps> = ({
  estimatedSize,
  canExport,
  isExporting,
  onClose,
  onExport
}) => (
  <div className="px-8 py-6 bg-gray-50 border-t border-gray-200">
    <div className="flex items-center justify-between">
      <div className="text-sm text-gray-600">
        Estimated export time: ~30 seconds • File size: ~{estimatedSize} MB
      </div>
      <div className="flex space-x-4">
        <button
          onClick={onClose}
          className="px-6 py-3 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onExport}
          disabled={!canExport || isExporting}
          className={`px-6 py-3 text-sm font-medium text-white rounded-lg transition-all flex items-center space-x-2 ${
            !canExport || isExporting
              ? 'bg-gray-400 cursor-not-allowed'
              : 'bg-blue-600 hover:bg-blue-700'
          }`}
        >
          {isExporting ? (
            <>
              <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin"></div>
              <span>Exporting...</span>
            </>
          ) : (
            <>
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10" />
              </svg>
              <span>Export Data Catalog</span>
            </>
          )}
        </button>
      </div>
    </div>
  </div>
);