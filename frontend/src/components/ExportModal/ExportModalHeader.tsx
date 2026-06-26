// React import removed — JSX runtime handles createElement

interface ExportModalHeaderProps {
  totalTables: number;
  totalSchemas: number;
  onClose: () => void;
}

export const ExportModalHeader: React.FC<ExportModalHeaderProps> = ({
  totalTables,
  totalSchemas,
  onClose
}) => (
  <div className="px-8 py-6 bg-gradient-to-r from-blue-600 to-purple-600 text-white relative">
    <div className="flex items-center justify-between">
      <div className="flex items-center space-x-4">
        <div className="w-12 h-12 bg-white bg-opacity-20 rounded-xl flex items-center justify-center">
          <svg className="w-6 h-6" fill="currentColor" viewBox="0 0 24 24">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z"/>
            <path d="M14 2v6h6"/>
            <path d="M16 13H8"/>
            <path d="M16 17H8"/>
            <path d="M10 9H8"/>
          </svg>
        </div>
        <div>
          <h1 className="text-2xl font-bold">Export Data Catalog</h1>
          <p className="text-blue-100 text-sm">Configure and download your database schema</p>
        </div>
      </div>
      <div className="text-right">
        <div className="text-sm text-blue-100">Ready to Export</div>
        <div className="text-lg font-semibold">
          {totalTables} Tables • {totalSchemas} Schemas
        </div>
      </div>
      <button
        onClick={onClose}
        className="absolute top-4 right-4 p-2 text-white hover:bg-white hover:bg-opacity-20 rounded-lg transition-colors"
        title="Close"
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  </div>
);