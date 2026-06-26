import React, { useState } from 'react';
import { Database, Layers, Link2, BarChart3 } from 'lucide-react';
import MetadataExplorer from '../components/MetadataExplorer';
import FieldMappingVisualizer from '../components/FieldMappingVisualizer';
import CacheMetricsDashboard from '../components/CacheMetricsDashboard';

type ViewType = 'explorer' | 'mappings' | 'metrics';

const MetadataManagementPage: React.FC = () => {
  const [activeView, setActiveView] = useState<ViewType>('explorer');

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
      {/* Navigation */}
      <nav className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-3">
              <div className="p-2 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-lg">
                <Database className="w-5 h-5 text-white" />
              </div>
              <span className="text-xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                Metadata Management
              </span>
            </div>
            
            <div className="flex items-center space-x-2">
              <NavButton
                icon={<Database className="w-4 h-4" />}
                label="Explorer"
                active={activeView === 'explorer'}
                onClick={() => setActiveView('explorer')}
              />
              <NavButton
                icon={<Link2 className="w-4 h-4" />}
                label="Mappings"
                active={activeView === 'mappings'}
                onClick={() => setActiveView('mappings')}
              />
              <NavButton
                icon={<BarChart3 className="w-4 h-4" />}
                label="Metrics"
                active={activeView === 'metrics'}
                onClick={() => setActiveView('metrics')}
              />
            </div>
          </div>
        </div>
      </nav>

      {/* Content */}
      <div className="max-w-7xl mx-auto px-6 py-8">
        {activeView === 'explorer' && <MetadataExplorer />}
        {activeView === 'mappings' && (
          <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl shadow-xl border border-slate-200 dark:border-slate-700 p-8">
            <FieldMappingVisualizer />
          </div>
        )}
        {activeView === 'metrics' && <CacheMetricsDashboard />}
      </div>
    </div>
  );
};

const NavButton: React.FC<{
  icon: React.ReactNode;
  label: string;
  active: boolean;
  onClick: () => void;
}> = ({ icon, label, active, onClick }) => (
  <button
    onClick={onClick}
    className={`flex items-center space-x-2 px-4 py-2 rounded-lg transition-all ${
      active
        ? 'bg-gradient-to-r from-blue-500 to-indigo-600 text-white shadow-lg'
        : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white border border-slate-200 dark:border-slate-700'
    }`}
  >
    {icon}
    <span className="font-medium">{label}</span>
  </button>
);

export default MetadataManagementPage;
