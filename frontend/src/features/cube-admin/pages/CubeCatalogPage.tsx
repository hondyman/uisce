import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../context/TenantContext';

interface SemanticModel {
  id: string;
  name: string;
  description: string;
  cube_type: 'cube' | 'view';
  measures_count: number;
  dimensions_count: number;
  joins_count: number;
  version: string;
  status: 'active' | 'draft' | 'deprecated';
  created_at: string;
  updated_at: string;
}

export function CubeCatalogPage() {
  const { tenant, datasource } = useTenant();
  const [models, setModels] = useState<SemanticModel[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<'all' | 'cube' | 'view'>('all');
  const [selectedModel, setSelectedModel] = useState<SemanticModel | null>(null);

  useEffect(() => {
    if (!tenant?.id || !datasource?.id) return;
    loadModels();
  }, [tenant?.id, datasource?.id]);

  const loadModels = async () => {
    setLoading(true);
    try {
      const res = await fetch(
        `/api/cube-admin/semantic-models?tenant_id=${tenant!.id}&tenant_instance_id=${datasource!.id}`
      );
      if (res.ok) {
        const data = await res.json();
        setModels(data || []);
      }
    } catch (err) {
      console.error('Failed to load models:', err);
    } finally {
      setLoading(false);
    }
  };

  const filteredModels = models.filter((m) => {
    const matchesSearch =
      m.name.toLowerCase().includes(search.toLowerCase()) ||
      m.description?.toLowerCase().includes(search.toLowerCase());
    const matchesType = typeFilter === 'all' || m.cube_type === typeFilter;
    return matchesSearch && matchesType;
  });

  if (!tenant || !datasource) {
    return (
      <div className="p-8">
        <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold text-yellow-800">Select a Tenant</h2>
          <p className="text-yellow-700 mt-2">
            Please select a tenant and datasource to browse the semantic catalog.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Semantic Catalog</h1>
          <p className="text-gray-500 mt-1">
            Browse and manage cube definitions for {tenant.display_name}
          </p>
        </div>
        <button className="px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors flex items-center gap-2">
          <PlusIcon className="w-5 h-5" />
          New Model
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4 mb-6">
        <div className="flex-1 relative">
          <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
          <input
            type="text"
            placeholder="Search models..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <div className="flex items-center gap-2 bg-gray-100 rounded-lg p-1">
          {(['all', 'cube', 'view'] as const).map((type) => (
            <button
              key={type}
              onClick={() => setTypeFilter(type)}
              className={`px-3 py-1.5 text-sm rounded-md transition-colors ${
                typeFilter === type
                  ? 'bg-white text-gray-900 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {type === 'all' ? 'All Types' : type === 'cube' ? 'Cubes' : 'Views'}
            </button>
          ))}
        </div>
      </div>

      {/* Models Grid */}
      {loading ? (
        <LoadingSkeleton />
      ) : filteredModels.length === 0 ? (
        <EmptyState search={search} />
      ) : (
        <div className="grid grid-cols-3 gap-6">
          {filteredModels.map((model) => (
            <ModelCard
              key={model.id}
              model={model}
              onSelect={() => setSelectedModel(model)}
            />
          ))}
        </div>
      )}

      {/* Model Detail Drawer */}
      {selectedModel && (
        <ModelDetailDrawer
          model={selectedModel}
          onClose={() => setSelectedModel(null)}
        />
      )}
    </div>
  );
}

interface ModelCardProps {
  model: SemanticModel;
  onSelect: () => void;
}

function ModelCard({ model, onSelect }: ModelCardProps) {
  const statusColors = {
    active: 'bg-green-100 text-green-700',
    draft: 'bg-yellow-100 text-yellow-700',
    deprecated: 'bg-gray-100 text-gray-600',
  };

  return (
    <div
      onClick={onSelect}
      className="bg-white rounded-xl border border-gray-200 p-6 hover:border-indigo-300 hover:shadow-md transition-all cursor-pointer"
    >
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className={`p-2 rounded-lg ${
              model.cube_type === 'cube' ? 'bg-indigo-100 text-indigo-600' : 'bg-purple-100 text-purple-600'
            }`}
          >
            {model.cube_type === 'cube' ? (
              <CubeIcon className="w-5 h-5" />
            ) : (
              <ViewIcon className="w-5 h-5" />
            )}
          </div>
          <div>
            <h3 className="font-semibold text-gray-900">{model.name}</h3>
            <span className={`text-xs px-2 py-0.5 rounded-full ${statusColors[model.status]}`}>
              {model.status}
            </span>
          </div>
        </div>
        <span className="text-xs text-gray-400">v{model.version}</span>
      </div>

      <p className="text-sm text-gray-600 mb-4 line-clamp-2">
        {model.description || 'No description'}
      </p>

      <div className="flex items-center gap-4 text-xs text-gray-500">
        <span className="flex items-center gap-1">
          <MeasureIcon className="w-4 h-4" />
          {model.measures_count} measures
        </span>
        <span className="flex items-center gap-1">
          <DimensionIcon className="w-4 h-4" />
          {model.dimensions_count} dimensions
        </span>
        <span className="flex items-center gap-1">
          <JoinIcon className="w-4 h-4" />
          {model.joins_count} joins
        </span>
      </div>
    </div>
  );
}

interface ModelDetailDrawerProps {
  model: SemanticModel;
  onClose: () => void;
}

function ModelDetailDrawer({ model, onClose }: ModelDetailDrawerProps) {
  return (
    <div className="fixed inset-0 z-50 flex justify-end">
      <div className="absolute inset-0 bg-black/30" onClick={onClose} />
      <div className="relative w-[600px] bg-white h-full shadow-xl overflow-auto">
        <div className="sticky top-0 bg-white border-b border-gray-200 px-6 py-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-gray-900">{model.name}</h2>
          <button
            onClick={onClose}
            className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          >
            <CloseIcon className="w-5 h-5 text-gray-500" />
          </button>
        </div>

        <div className="p-6 space-y-6">
          {/* Overview */}
          <section>
            <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-3">
              Overview
            </h3>
            <div className="bg-gray-50 rounded-lg p-4 space-y-3">
              <div className="flex justify-between">
                <span className="text-gray-600">Type</span>
                <span className="font-medium capitalize">{model.cube_type}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Version</span>
                <span className="font-medium">{model.version}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Status</span>
                <span className="font-medium capitalize">{model.status}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-gray-600">Last Updated</span>
                <span className="font-medium">
                  {new Date(model.updated_at).toLocaleDateString()}
                </span>
              </div>
            </div>
          </section>

          {/* Description */}
          <section>
            <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-3">
              Description
            </h3>
            <p className="text-gray-700">{model.description || 'No description provided'}</p>
          </section>

          {/* Statistics */}
          <section>
            <h3 className="text-sm font-medium text-gray-500 uppercase tracking-wider mb-3">
              Statistics
            </h3>
            <div className="grid grid-cols-3 gap-4">
              <div className="bg-indigo-50 rounded-lg p-4 text-center">
                <p className="text-2xl font-bold text-indigo-600">{model.measures_count}</p>
                <p className="text-sm text-indigo-700">Measures</p>
              </div>
              <div className="bg-purple-50 rounded-lg p-4 text-center">
                <p className="text-2xl font-bold text-purple-600">{model.dimensions_count}</p>
                <p className="text-sm text-purple-700">Dimensions</p>
              </div>
              <div className="bg-green-50 rounded-lg p-4 text-center">
                <p className="text-2xl font-bold text-green-600">{model.joins_count}</p>
                <p className="text-sm text-green-700">Joins</p>
              </div>
            </div>
          </section>

          {/* Actions */}
          <section className="pt-4 border-t border-gray-200">
            <div className="flex gap-3">
              <button className="flex-1 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
                Edit Model
              </button>
              <button className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors">
                View YAML
              </button>
              <button className="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 transition-colors">
                History
              </button>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}

function EmptyState({ search }: { search: string }) {
  return (
    <div className="text-center py-12">
      <CubeIcon className="w-12 h-12 text-gray-300 mx-auto mb-4" />
      <h3 className="text-lg font-medium text-gray-900">
        {search ? 'No models found' : 'No semantic models yet'}
      </h3>
      <p className="text-gray-500 mt-1">
        {search
          ? `No models match "${search}"`
          : 'Create your first cube or view to get started'}
      </p>
      {!search && (
        <button className="mt-4 px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 transition-colors">
          Create Model
        </button>
      )}
    </div>
  );
}

function LoadingSkeleton() {
  return (
    <div className="grid grid-cols-3 gap-6">
      {[...Array(6)].map((_, i) => (
        <div key={i} className="bg-white rounded-xl border p-6 animate-pulse">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-10 h-10 bg-gray-200 rounded-lg" />
            <div className="h-5 w-32 bg-gray-200 rounded" />
          </div>
          <div className="h-4 w-full bg-gray-100 rounded mb-2" />
          <div className="h-4 w-2/3 bg-gray-100 rounded" />
        </div>
      ))}
    </div>
  );
}

// Icons
function PlusIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
    </svg>
  );
}

function SearchIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
    </svg>
  );
}

function CubeIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4" />
    </svg>
  );
}

function ViewIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
    </svg>
  );
}

function MeasureIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 12l3-3 3 3 4-4M8 21l4-4 4 4M3 4h18M4 4h16v12a1 1 0 01-1 1H5a1 1 0 01-1-1V4z" />
    </svg>
  );
}

function DimensionIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 10h16M4 14h16M4 18h16" />
    </svg>
  );
}

function JoinIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
    </svg>
  );
}

function CloseIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
    </svg>
  );
}
