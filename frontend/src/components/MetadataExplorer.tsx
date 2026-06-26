import React, { useState, useEffect } from 'react';
import { Search, Database, Layers, TrendingUp, RefreshCw, Zap, Link2, Filter } from 'lucide-react';

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description: string;
  status: string;
  version: number;
  fields: Array<{
    name: string;
    type: string;
    label: string;
    is_required: boolean;
  }>;
  is_core: boolean;
  category: string;
}

interface SemanticView {
  view_id: string;
  view_name: string;
  tenant_id: string;
  version: number;
  fields: Array<{
    field_name: string;
    field_type: string;
    description?: string;
  }>;
}

interface UnifiedMetadata {
  tenant_id: string;
  business_objects: BusinessObject[];
  semantic_views: SemanticView[];
  cached_at: string;
}

interface CacheMetrics {
  business_objects: {
    hits: number;
    misses: number;
    hit_rate: number;
    item_count: number;
    memory_bytes: number;
  };
  semantic_views: {
    semantic_view_count: number;
    cache_ttl_hours: number;
  };
}

const MetadataExplorer: React.FC = () => {
  const [metadata, setMetadata] = useState<UnifiedMetadata | null>(null);
  const [metrics, setMetrics] = useState<CacheMetrics | null>(null);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'business-objects' | 'semantic-views' | 'mappings'>('business-objects');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBO, setSelectedBO] = useState<BusinessObject | null>(null);
  const [selectedView, setSelectedView] = useState<SemanticView | null>(null);

  useEffect(() => {
    fetchMetadata();
    fetchMetrics();
  }, []);

  const fetchMetadata = async () => {
    try {
      setLoading(true);
      const response = await fetch('/api/metadata/unified', {
        headers: {
          'X-Tenant-ID': 'default-tenant', // Replace with actual tenant
        },
      });
      const data = await response.json();
      setMetadata(data);
    } catch (error) {
      console.error('Failed to fetch metadata:', error);
    } finally {
      setLoading(false);
    }
  };

  const fetchMetrics = async () => {
    try {
      const response = await fetch('/api/metadata/unified/metrics');
      const data = await response.json();
      setMetrics(data);
    } catch (error) {
      console.error('Failed to fetch metrics:', error);
    }
  };

  const invalidateCache = async () => {
    try {
      await fetch('/api/metadata/unified/invalidate', {
        method: 'POST',
        headers: {
          'X-Tenant-ID': 'default-tenant',
          'Content-Type': 'application/json',
        },
      });
      fetchMetadata();
      fetchMetrics();
    } catch (error) {
      console.error('Failed to invalidate cache:', error);
    }
  };

  const filteredBusinessObjects = metadata?.business_objects.filter(bo =>
    bo.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    bo.display_name.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];

  const filteredSemanticViews = metadata?.semantic_views.filter(view =>
    view.view_name.toLowerCase().includes(searchQuery.toLowerCase())
  ) || [];

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50 to-indigo-50 dark:from-slate-900 dark:via-slate-800 dark:to-indigo-950">
      {/* Header */}
      <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl border-b border-slate-200 dark:border-slate-700 sticky top-0 z-50">
        <div className="max-w-7xl mx-auto px-6 py-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-4">
              <div className="p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl shadow-lg">
                <Database className="w-6 h-6 text-white" />
              </div>
              <div>
                <h1 className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-indigo-600 bg-clip-text text-transparent">
                  Metadata Explorer
                </h1>
                <p className="text-sm text-slate-600 dark:text-slate-400">
                  Unified Business Objects & Semantic Views
                </p>
              </div>
            </div>
            
            <div className="flex items-center space-x-3">
              <button
                onClick={invalidateCache}
                className="flex items-center space-x-2 px-4 py-2 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-all shadow-sm"
              >
                <RefreshCw className="w-4 h-4" />
                <span className="text-sm font-medium">Refresh</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Metrics Dashboard */}
      {metrics && (
        <div className="max-w-7xl mx-auto px-6 py-6">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
            <MetricCard
              icon={<Database className="w-5 h-5" />}
              label="Business Objects"
              value={metrics.business_objects.item_count.toString()}
              subtitle={`${(metrics.business_objects.hit_rate * 100).toFixed(1)}% hit rate`}
              color="blue"
            />
            <MetricCard
              icon={<Layers className="w-5 h-5" />}
              label="Semantic Views"
              value={metrics.semantic_views.semantic_view_count.toString()}
              subtitle={`${metrics.semantic_views.cache_ttl_hours}h TTL`}
              color="indigo"
            />
            <MetricCard
              icon={<Zap className="w-5 h-5" />}
              label="Cache Hits"
              value={metrics.business_objects.hits.toLocaleString()}
              subtitle={`${metrics.business_objects.misses} misses`}
              color="green"
            />
            <MetricCard
              icon={<TrendingUp className="w-5 h-5" />}
              label="Memory Usage"
              value={`${(metrics.business_objects.memory_bytes / 1024).toFixed(0)} KB`}
              subtitle="In-memory cache"
              color="purple"
            />
          </div>
        </div>
      )}

      {/* Main Content */}
      <div className="max-w-7xl mx-auto px-6 py-6">
        <div className="bg-white/80 dark:bg-slate-900/80 backdrop-blur-xl rounded-2xl shadow-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
          {/* Tabs */}
          <div className="flex border-b border-slate-200 dark:border-slate-700">
            <TabButton
              active={activeTab === 'business-objects'}
              onClick={() => setActiveTab('business-objects')}
              icon={<Database className="w-4 h-4" />}
              label="Business Objects"
              count={metadata?.business_objects.length || 0}
            />
            <TabButton
              active={activeTab === 'semantic-views'}
              onClick={() => setActiveTab('semantic-views')}
              icon={<Layers className="w-4 h-4" />}
              label="Semantic Views"
              count={metadata?.semantic_views.length || 0}
            />
            <TabButton
              active={activeTab === 'mappings'}
              onClick={() => setActiveTab('mappings')}
              icon={<Link2 className="w-4 h-4" />}
              label="Mappings"
              count={0}
            />
          </div>

          {/* Search */}
          <div className="p-6 border-b border-slate-200 dark:border-slate-700">
            <div className="relative">
              <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 w-5 h-5 text-slate-400" />
              <input
                type="text"
                placeholder="Search metadata..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="w-full pl-12 pr-4 py-3 bg-slate-50 dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-xl focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
              />
            </div>
          </div>

          {/* Content */}
          <div className="p-6">
            {loading ? (
              <div className="flex items-center justify-center py-12">
                <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
              </div>
            ) : (
              <>
                {activeTab === 'business-objects' && (
                  <BusinessObjectsGrid
                    businessObjects={filteredBusinessObjects}
                    selectedBO={selectedBO}
                    onSelect={setSelectedBO}
                  />
                )}
                {activeTab === 'semantic-views' && (
                  <SemanticViewsGrid
                    semanticViews={filteredSemanticViews}
                    selectedView={selectedView}
                    onSelect={setSelectedView}
                  />
                )}
                {activeTab === 'mappings' && (
                  <MappingsView />
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Detail Panel */}
      {(selectedBO || selectedView) && (
        <DetailPanel
          businessObject={selectedBO}
          semanticView={selectedView}
          onClose={() => {
            setSelectedBO(null);
            setSelectedView(null);
          }}
        />
      )}
    </div>
  );
};

// Metric Card Component
const MetricCard: React.FC<{
  icon: React.ReactNode;
  label: string;
  value: string;
  subtitle: string;
  color: 'blue' | 'indigo' | 'green' | 'purple';
}> = ({ icon, label, value, subtitle, color }) => {
  const colorClasses = {
    blue: 'from-blue-500 to-blue-600',
    indigo: 'from-indigo-500 to-indigo-600',
    green: 'from-green-500 to-green-600',
    purple: 'from-purple-500 to-purple-600',
  };

  return (
    <div className="bg-white dark:bg-slate-800 rounded-xl p-6 border border-slate-200 dark:border-slate-700 hover:shadow-lg transition-all">
      <div className={`inline-flex p-3 rounded-lg bg-gradient-to-br ${colorClasses[color]} mb-4`}>
        <div className="text-white">{icon}</div>
      </div>
      <div className="text-3xl font-bold text-slate-900 dark:text-white mb-1">{value}</div>
      <div className="text-sm font-medium text-slate-600 dark:text-slate-400 mb-1">{label}</div>
      <div className="text-xs text-slate-500 dark:text-slate-500">{subtitle}</div>
    </div>
  );
};

// Tab Button Component
const TabButton: React.FC<{
  active: boolean;
  onClick: () => void;
  icon: React.ReactNode;
  label: string;
  count: number;
}> = ({ active, onClick, icon, label, count }) => (
  <button
    onClick={onClick}
    className={`flex items-center space-x-2 px-6 py-4 border-b-2 transition-all ${
      active
        ? 'border-blue-500 text-blue-600 dark:text-blue-400 bg-blue-50/50 dark:bg-blue-950/20'
        : 'border-transparent text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-slate-200'
    }`}
  >
    {icon}
    <span className="font-medium">{label}</span>
    <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${
      active ? 'bg-blue-100 dark:bg-blue-900 text-blue-700 dark:text-blue-300' : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
    }`}>
      {count}
    </span>
  </button>
);

// Business Objects Grid
const BusinessObjectsGrid: React.FC<{
  businessObjects: BusinessObject[];
  selectedBO: BusinessObject | null;
  onSelect: (bo: BusinessObject) => void;
}> = ({ businessObjects, selectedBO, onSelect }) => (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {businessObjects.map((bo) => (
      <div
        key={bo.id}
        onClick={() => onSelect(bo)}
        className={`p-6 rounded-xl border-2 cursor-pointer transition-all hover:shadow-lg ${
          selectedBO?.id === bo.id
            ? 'border-blue-500 bg-blue-50 dark:bg-blue-950/20'
            : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 hover:border-blue-300'
        }`}
      >
        <div className="flex items-start justify-between mb-3">
          <div className="flex-1">
            <h3 className="font-semibold text-lg text-slate-900 dark:text-white mb-1">
              {bo.display_name}
            </h3>
            <p className="text-sm text-slate-600 dark:text-slate-400">{bo.name}</p>
          </div>
          {bo.is_core && (
            <span className="px-2 py-1 bg-gradient-to-r from-amber-400 to-orange-500 text-white text-xs font-semibold rounded-lg">
              CORE
            </span>
          )}
        </div>
        <p className="text-sm text-slate-600 dark:text-slate-400 mb-4 line-clamp-2">
          {bo.description || 'No description available'}
        </p>
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500 dark:text-slate-500">
            {bo.fields.length} fields
          </span>
          <span className={`px-2 py-1 rounded-full font-medium ${
            bo.status === 'active' ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400' : 'bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400'
          }`}>
            {bo.status}
          </span>
        </div>
      </div>
    ))}
  </div>
);

// Semantic Views Grid
const SemanticViewsGrid: React.FC<{
  semanticViews: SemanticView[];
  selectedView: SemanticView | null;
  onSelect: (view: SemanticView) => void;
}> = ({ semanticViews, selectedView, onSelect }) => (
  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
    {semanticViews.map((view) => (
      <div
        key={view.view_id}
        onClick={() => onSelect(view)}
        className={`p-6 rounded-xl border-2 cursor-pointer transition-all hover:shadow-lg ${
          selectedView?.view_id === view.view_id
            ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/20'
            : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 hover:border-indigo-300'
        }`}
      >
        <div className="flex items-center space-x-3 mb-4">
          <div className="p-2 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg">
            <Layers className="w-5 h-5 text-white" />
          </div>
          <div className="flex-1">
            <h3 className="font-semibold text-lg text-slate-900 dark:text-white">
              {view.view_name}
            </h3>
          </div>
        </div>
        <div className="flex items-center justify-between text-xs">
          <span className="text-slate-500 dark:text-slate-500">
            {view.fields.length} fields
          </span>
          <span className="px-2 py-1 bg-indigo-100 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-400 rounded-full font-medium">
            v{view.version}
          </span>
        </div>
      </div>
    ))}
  </div>
);

// Mappings View
const MappingsView: React.FC = () => (
  <div className="text-center py-12">
    <div className="inline-flex p-4 bg-slate-100 dark:bg-slate-800 rounded-full mb-4">
      <Link2 className="w-8 h-8 text-slate-400" />
    </div>
    <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
      Field Mappings
    </h3>
    <p className="text-slate-600 dark:text-slate-400 max-w-md mx-auto">
      Create and manage mappings between business objects and semantic views to enable unified data access.
    </p>
  </div>
);

// Detail Panel
const DetailPanel: React.FC<{
  businessObject: BusinessObject | null;
  semanticView: SemanticView | null;
  onClose: () => void;
}> = ({ businessObject, semanticView, onClose }) => (
  <div className="fixed inset-y-0 right-0 w-full md:w-1/3 bg-white dark:bg-slate-900 shadow-2xl border-l border-slate-200 dark:border-slate-700 overflow-y-auto z-50 transform transition-transform">
    <div className="p-6">
      <div className="flex items-center justify-between mb-6">
        <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
          {businessObject?.display_name || semanticView?.view_name}
        </h2>
        <button
          onClick={onClose}
          className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg transition-colors"
        >
          ✕
        </button>
      </div>

      {businessObject && (
        <div className="space-y-6">
          <div>
            <h3 className="text-sm font-semibold text-slate-600 dark:text-slate-400 mb-2">FIELDS</h3>
            <div className="space-y-2">
              {businessObject.fields.map((field, idx) => (
                <div key={idx} className="p-3 bg-slate-50 dark:bg-slate-800 rounded-lg">
                  <div className="flex items-center justify-between mb-1">
                    <span className="font-medium text-slate-900 dark:text-white">{field.label}</span>
                    {field.is_required && (
                      <span className="text-xs text-red-600 dark:text-red-400 font-semibold">REQUIRED</span>
                    )}
                  </div>
                  <div className="text-sm text-slate-600 dark:text-slate-400">
                    {field.name} • {field.type}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {semanticView && (
        <div className="space-y-6">
          <div>
            <h3 className="text-sm font-semibold text-slate-600 dark:text-slate-400 mb-2">FIELDS</h3>
            <div className="space-y-2">
              {semanticView.fields.map((field, idx) => (
                <div key={idx} className="p-3 bg-slate-50 dark:bg-slate-800 rounded-lg">
                  <div className="font-medium text-slate-900 dark:text-white mb-1">{field.field_name}</div>
                  <div className="text-sm text-slate-600 dark:text-slate-400">
                    {field.field_type}
                    {field.description && ` • ${field.description}`}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  </div>
);

export default MetadataExplorer;
