// React import removed — JSX runtime handles createElement
import { Node as FlowNode, Edge } from 'reactflow';
import { ExportOptions } from '../../types/ExportTypes';

interface ExportPreviewProps {
  filteredNodes: FlowNode[];
  filteredEdges: Edge[];
  getNodeSchema: (node: FlowNode) => string;
  exportOptions: ExportOptions;
  estimatedSize: string;
}

export const ExportPreview: React.FC<ExportPreviewProps> = ({ 
  filteredNodes, 
  filteredEdges, 
  getNodeSchema, 
  exportOptions, 
  estimatedSize 
}) => {
  return (
    <div className="space-y-8">
      <div>
        <div className="flex items-center space-x-3 mb-6">
          <div className="w-8 h-8 bg-green-500 text-white rounded-full flex items-center justify-center text-sm font-bold">✓</div>
          <h2 className="text-xl font-semibold text-gray-900">Export Preview</h2>
        </div>
        
        <div className="grid grid-cols-2 gap-4">
          <div className="bg-white rounded-xl p-6 text-center">
            <div className="w-12 h-12 bg-blue-500 text-white rounded-xl flex items-center justify-center mx-auto mb-3">
              <span className="text-lg font-bold">{new Set(filteredNodes.map(n => getNodeSchema(n))).size}</span>
            </div>
            <div className="text-sm font-semibold text-blue-700">SCHEMAS</div>
          </div>
          
          <div className="bg-white rounded-xl p-6 text-center">
            <div className="w-12 h-12 bg-purple-500 text-white rounded-xl flex items-center justify-center mx-auto mb-3">
              <span className="text-lg font-bold">{filteredNodes.length}</span>
            </div>
            <div className="text-sm font-semibold text-purple-700">TABLES</div>
          </div>
          
          <div className="bg-white rounded-xl p-6 text-center">
            <div className="w-12 h-12 bg-green-500 text-white rounded-xl flex items-center justify-center mx-auto mb-3">
              <span className="text-lg font-bold">
                {filteredNodes.reduce((sum, n) => sum + (n.data?.columns?.length || 0), 0)}
              </span>
            </div>
            <div className="text-sm font-semibold text-green-700">COLUMNS</div>
          </div>
          
          <div className="bg-white rounded-xl p-6 text-center">
            <div className="w-12 h-12 bg-orange-500 text-white rounded-xl flex items-center justify-center mx-auto mb-3">
              <span className="text-lg font-bold">{filteredEdges.length}</span>
            </div>
            <div className="text-sm font-semibold text-orange-700">RELATIONS</div>
          </div>
        </div>
      </div>

      <div>
        <h3 className="text-lg font-semibold text-gray-900 mb-4">File Output</h3>
        <div className="bg-white rounded-xl p-6 border border-gray-200">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm text-gray-600">Ready to download</span>
            <span className="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded-full font-medium">
              {exportOptions.format.toUpperCase()} Format
            </span>
          </div>
          
          <div className="flex items-center space-x-3">
            <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center">
              <svg className="w-5 h-5 text-blue-600" fill="currentColor" viewBox="0 0 24 24">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8l-6-6z"/>
                <path d="M14 2v6h6"/>
              </svg>
            </div>
            <div className="flex-1">
              <div className="font-medium text-gray-900">
                data-catalog-{new Date().toISOString().split('T')[0]}.{exportOptions.format}
              </div>
              <div className="text-sm text-gray-500">
                Estimated size: ~{estimatedSize} MB
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};