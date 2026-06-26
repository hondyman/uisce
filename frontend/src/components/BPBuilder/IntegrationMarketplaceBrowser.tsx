import React, { useState, useEffect } from 'react';
import {
  Search,
  Package,
  Zap,
  MessageSquare,
  Mail,
  Webhook,
  Database,
  BarChart3,
  Settings,
  Check,
  X,
  ExternalLink,
  Play,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Clock,
  AlertCircle,
  CheckCircle2,
  XCircle,
  TrendingUp,
  Download,
  Plus,
  Activity,
  Eye,
  RefreshCw,
} from 'lucide-react';

interface IntegrationMarketplaceBrowserProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

interface MarketplaceIntegration {
  id: string;
  integration_key: string;
  name: string;
  description: string;
  category: string;
  provider: string;
  icon_url: string;
  version: string;
  is_official: boolean;
  is_active: boolean;
  config_schema: Record<string, any>;
  auth_type: string;
  supports_webhooks: boolean;
  supports_polling: boolean;
  supports_actions: boolean;
  documentation_url: string;
  setup_guide: string;
  example_payload: Record<string, any>;
  install_count: number;
  rating: number;
  created_at: string;
  updated_at: string;
}

interface InstalledIntegration {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  integration_id: string;
  installed_by: string;
  installed_at: string;
  is_enabled: boolean;
  config: Record<string, any>;
  last_used_at?: string;
  execution_count: number;
  success_count: number;
  failure_count: number;
  integration_name: string;
  integration_key: string;
  integration_icon_url: string;
}

interface IntegrationExecution {
  id: string;
  action: string;
  status: string;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  error_message?: string;
  workflow_type?: string;
  step_name?: string;
}

export const IntegrationMarketplaceBrowser: React.FC<IntegrationMarketplaceBrowserProps> = ({
  tenant,
  datasource,
}) => {
  const [viewMode, setViewMode] = useState<'marketplace' | 'installed' | 'logs'>('marketplace');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [marketplaceIntegrations, setMarketplaceIntegrations] = useState<MarketplaceIntegration[]>([]);
  const [installedIntegrations, setInstalledIntegrations] = useState<InstalledIntegration[]>([]);
  const [executionLogs, setExecutionLogs] = useState<IntegrationExecution[]>([]);
  const [selectedIntegration, setSelectedIntegration] = useState<MarketplaceIntegration | null>(null);
  const [showInstallModal, setShowInstallModal] = useState(false);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [configFormData, setConfigFormData] = useState<Record<string, any>>({});
  const [isLoading, setIsLoading] = useState(false);

  const categories = [
    { key: 'all', label: 'All Integrations', icon: Package },
    { key: 'communication', label: 'Communication', icon: MessageSquare },
    { key: 'automation', label: 'Automation', icon: Zap },
    { key: 'storage', label: 'Storage', icon: Database },
    { key: 'analytics', label: 'Analytics', icon: BarChart3 },
    { key: 'ai', label: 'AI', icon: Activity },
  ];

  useEffect(() => {
    if (viewMode === 'marketplace') {
      fetchMarketplaceIntegrations();
    } else if (viewMode === 'installed') {
      fetchInstalledIntegrations();
    } else if (viewMode === 'logs') {
      fetchExecutionLogs();
    }
  }, [viewMode, selectedCategory, tenant.id, datasource.id]);

  async function fetchMarketplaceIntegrations() {
    try {
      const params = new URLSearchParams();
      if (selectedCategory !== 'all') {
        params.set('category', selectedCategory);
      }
      if (searchQuery) {
        params.set('search', searchQuery);
      }

      const response = await fetch(`/api/integrations/marketplace?${params}`);
      if (!response.ok) throw new Error('Failed to fetch');

      const data = await response.json();
      setMarketplaceIntegrations(data || []);
    } catch (error) {
      console.error('Error fetching marketplace integrations:', error);
    }
  }

  async function fetchInstalledIntegrations() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/installed?${params}`);
      if (!response.ok) throw new Error('Failed to fetch');

      const data = await response.json();
      setInstalledIntegrations(data || []);
    } catch (error) {
      console.error('Error fetching installed integrations:', error);
    }
  }

  async function fetchExecutionLogs() {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/executions?${params}`);
      if (!response.ok) throw new Error('Failed to fetch');

      const data = await response.json();
      setExecutionLogs(data || []);
    } catch (error) {
      console.error('Error fetching execution logs:', error);
    }
  }

  async function installIntegration() {
    if (!selectedIntegration) return;

    setIsLoading(true);
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/install?${params}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          integration_key: selectedIntegration.integration_key,
          config: configFormData,
          credentials: {}, // Would be populated from form
        }),
      });

      if (!response.ok) throw new Error('Installation failed');

      alert('Integration installed successfully!');
      setShowInstallModal(false);
      setSelectedIntegration(null);
      setConfigFormData({});
      fetchInstalledIntegrations();
    } catch (error) {
      console.error('Error installing integration:', error);
      alert('Failed to install integration: ' + error);
    } finally {
      setIsLoading(false);
    }
  }

  async function toggleIntegration(installationId: string, currentState: boolean) {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/installed/${installationId}/toggle?${params}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled: !currentState }),
      });

      if (!response.ok) throw new Error('Toggle failed');

      fetchInstalledIntegrations();
    } catch (error) {
      console.error('Error toggling integration:', error);
    }
  }

  async function uninstallIntegration(installationId: string) {
    if (!confirm('Are you sure you want to uninstall this integration?')) return;

    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/installed/${installationId}?${params}`, {
        method: 'DELETE',
      });

      if (!response.ok) throw new Error('Uninstall failed');

      alert('Integration uninstalled successfully!');
      fetchInstalledIntegrations();
    } catch (error) {
      console.error('Error uninstalling integration:', error);
      alert('Failed to uninstall integration');
    }
  }

  async function testIntegration(installationId: string) {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(`/api/integrations/test/${installationId}?${params}`, {
        method: 'POST',
      });

      if (!response.ok) throw new Error('Test failed');

      const result = await response.json();
      alert(result.success ? 'Connection test successful!' : 'Connection test failed: ' + result.message);
    } catch (error) {
      console.error('Error testing integration:', error);
      alert('Test failed: ' + error);
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'success':
        return <CheckCircle2 size={16} className="text-green-600" />;
      case 'failed':
        return <XCircle size={16} className="text-red-600" />;
      case 'pending':
        return <Clock size={16} className="text-yellow-600" />;
      default:
        return <AlertCircle size={16} className="text-gray-600" />;
    }
  };

  const getCategoryIcon = (category: string) => {
    const cat = categories.find((c) => c.key === category);
    return cat ? cat.icon : Package;
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-white border-b px-6 py-4">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-2xl font-bold text-gray-900 flex items-center gap-2">
              <Package size={28} className="text-blue-600" />
              Integration Marketplace
            </h1>
            <p className="text-sm text-gray-500 mt-1">
              Browse, install, and manage workflow integrations
            </p>
          </div>
          <div className="flex gap-3">
            <button
              onClick={() => setViewMode('marketplace')}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                viewMode === 'marketplace'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              <Download size={18} className="inline mr-2" />
              Marketplace
            </button>
            <button
              onClick={() => setViewMode('installed')}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                viewMode === 'installed'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              <Check size={18} className="inline mr-2" />
              Installed ({installedIntegrations.length})
            </button>
            <button
              onClick={() => setViewMode('logs')}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                viewMode === 'logs'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              <Activity size={18} className="inline mr-2" />
              Execution Logs
            </button>
          </div>
        </div>

        {/* Search and Filters */}
        {viewMode === 'marketplace' && (
          <div className="flex gap-4">
            <div className="flex-1 relative">
              <Search size={20} className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" />
              <input
                type="text"
                placeholder="Search integrations..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && fetchMarketplaceIntegrations()}
                className="w-full pl-10 pr-4 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
            <button
              onClick={fetchMarketplaceIntegrations}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all"
            >
              Search
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-hidden flex">
        {/* Sidebar - Categories */}
        {viewMode === 'marketplace' && (
          <div className="w-64 bg-white border-r p-4 overflow-y-auto">
            <h3 className="text-sm font-semibold text-gray-700 mb-3">Categories</h3>
            <div className="space-y-1">
              {categories.map((cat) => {
                const Icon = cat.icon;
                return (
                  <button
                    key={cat.key}
                    onClick={() => setSelectedCategory(cat.key)}
                    className={`w-full flex items-center gap-3 px-3 py-2 rounded-lg text-left transition-all ${
                      selectedCategory === cat.key
                        ? 'bg-blue-100 text-blue-700 font-medium'
                        : 'text-gray-700 hover:bg-gray-100'
                    }`}
                  >
                    <Icon size={18} />
                    <span className="text-sm">{cat.label}</span>
                  </button>
                );
              })}
            </div>
          </div>
        )}

        {/* Main Content */}
        <div className="flex-1 overflow-y-auto p-6">
          {viewMode === 'marketplace' && (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              {marketplaceIntegrations.map((integration) => {
                const Icon = getCategoryIcon(integration.category);
                const isInstalled = installedIntegrations.some(
                  (inst) => inst.integration_id === integration.id
                );

                return (
                  <div
                    key={integration.id}
                    className="bg-white rounded-lg border p-4 hover:shadow-lg transition-all cursor-pointer"
                    onClick={() => {
                      setSelectedIntegration(integration);
                      if (!isInstalled) {
                        setShowInstallModal(true);
                      }
                    }}
                  >
                    <div className="flex items-start gap-3 mb-3">
                      <div className="p-3 rounded-lg bg-blue-100">
                        <Icon size={24} className="text-blue-600" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="font-semibold text-gray-900">{integration.name}</h3>
                          {integration.is_official && (
                            <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full font-medium">
                              Official
                            </span>
                          )}
                        </div>
                        <p className="text-xs text-gray-500 mt-1">{integration.provider}</p>
                      </div>
                    </div>

                    <p className="text-sm text-gray-600 mb-3 line-clamp-2">{integration.description}</p>

                    <div className="flex items-center justify-between text-xs text-gray-500 mb-3">
                      <span className="flex items-center gap-1">
                        <TrendingUp size={14} />
                        {integration.install_count} installs
                      </span>
                      <span className="flex items-center gap-1">
                        ⭐ {integration.rating.toFixed(1)}
                      </span>
                    </div>

                    <div className="flex gap-2">
                      {isInstalled ? (
                        <span className="flex-1 px-3 py-2 bg-green-100 text-green-700 rounded-lg text-sm font-medium text-center flex items-center justify-center gap-2">
                          <Check size={16} />
                          Installed
                        </span>
                      ) : (
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            setSelectedIntegration(integration);
                            setShowInstallModal(true);
                          }}
                          className="flex-1 px-3 py-2 bg-blue-600 text-white rounded-lg text-sm font-medium hover:bg-blue-700 transition-all"
                        >
                          <Plus size={16} className="inline mr-1" />
                          Install
                        </button>
                      )}
                      {integration.documentation_url && (
                        <a
                          href={integration.documentation_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          onClick={(e) => e.stopPropagation()}
                          className="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-all"
                        >
                          <ExternalLink size={16} />
                        </a>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {viewMode === 'installed' && (
            <div className="space-y-4">
              {installedIntegrations.length === 0 ? (
                <div className="text-center py-12 text-gray-500">
                  <Package size={48} className="mx-auto mb-4 text-gray-400" />
                  <p className="text-lg font-medium">No integrations installed</p>
                  <p className="text-sm mt-2">Browse the marketplace to install integrations</p>
                </div>
              ) : (
                installedIntegrations.map((installation) => {
                  const successRate =
                    installation.execution_count > 0
                      ? (installation.success_count / installation.execution_count) * 100
                      : 0;

                  return (
                    <div key={installation.id} className="bg-white rounded-lg border p-4">
                      <div className="flex items-start justify-between">
                        <div className="flex items-start gap-3 flex-1">
                          <div className="p-3 rounded-lg bg-blue-100">
                            <Package size={24} className="text-blue-600" />
                          </div>
                          <div className="flex-1">
                            <h3 className="font-semibold text-gray-900 flex items-center gap-2">
                              {installation.integration_name}
                              {installation.is_enabled ? (
                                <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs rounded-full">
                                  Enabled
                                </span>
                              ) : (
                                <span className="px-2 py-0.5 bg-gray-100 text-gray-700 text-xs rounded-full">
                                  Disabled
                                </span>
                              )}
                            </h3>
                            <p className="text-xs text-gray-500 mt-1">
                              Installed {new Date(installation.installed_at).toLocaleDateString()}
                            </p>

                            <div className="flex items-center gap-4 mt-3 text-sm">
                              <div className="flex items-center gap-1">
                                <Activity size={16} className="text-blue-600" />
                                <span className="font-medium">{installation.execution_count}</span>
                                <span className="text-gray-500">executions</span>
                              </div>
                              <div className="flex items-center gap-1">
                                <CheckCircle2 size={16} className="text-green-600" />
                                <span className="font-medium">{successRate.toFixed(1)}%</span>
                                <span className="text-gray-500">success</span>
                              </div>
                              {installation.last_used_at && (
                                <div className="flex items-center gap-1 text-gray-500">
                                  <Clock size={16} />
                                  Last used {new Date(installation.last_used_at).toLocaleString()}
                                </div>
                              )}
                            </div>
                          </div>
                        </div>

                        <div className="flex gap-2">
                          <button
                            onClick={() => toggleIntegration(installation.id, installation.is_enabled)}
                            className="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-all"
                            title={installation.is_enabled ? 'Disable' : 'Enable'}
                          >
                            {installation.is_enabled ? (
                              <ToggleRight size={18} className="text-green-600" />
                            ) : (
                              <ToggleLeft size={18} />
                            )}
                          </button>
                          <button
                            onClick={() => testIntegration(installation.id)}
                            className="px-3 py-2 bg-blue-100 text-blue-700 rounded-lg hover:bg-blue-200 transition-all"
                            title="Test Connection"
                          >
                            <Play size={18} />
                          </button>
                          <button
                            onClick={() => {
                              /* Open config modal */
                            }}
                            className="px-3 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-all"
                            title="Configure"
                          >
                            <Settings size={18} />
                          </button>
                          <button
                            onClick={() => uninstallIntegration(installation.id)}
                            className="px-3 py-2 bg-red-100 text-red-700 rounded-lg hover:bg-red-200 transition-all"
                            title="Uninstall"
                          >
                            <Trash2 size={18} />
                          </button>
                        </div>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          )}

          {viewMode === 'logs' && (
            <div className="bg-white rounded-lg border">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Status
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Action
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Workflow
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Duration
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Time
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {executionLogs.map((log) => (
                    <tr key={log.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          {getStatusIcon(log.status)}
                          <span className="text-sm font-medium">{log.status}</span>
                        </div>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-900">{log.action}</td>
                      <td className="px-4 py-3 text-sm text-gray-600">
                        {log.workflow_type || '-'}
                        {log.step_name && (
                          <div className="text-xs text-gray-500">{log.step_name}</div>
                        )}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-600">{log.duration_ms}ms</td>
                      <td className="px-4 py-3 text-sm text-gray-600">
                        {new Date(log.started_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3">
                        <button className="text-blue-600 hover:text-blue-800 text-sm font-medium">
                          <Eye size={16} className="inline mr-1" />
                          View
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>

      {/* Install Modal */}
      {showInstallModal && selectedIntegration && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full mx-4 max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b">
              <div className="flex items-center justify-between">
                <h2 className="text-xl font-bold text-gray-900">Install {selectedIntegration.name}</h2>
                <button
                  onClick={() => {
                    setShowInstallModal(false);
                    setSelectedIntegration(null);
                  }}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X size={24} />
                </button>
              </div>
            </div>

            <div className="p-6">
              <p className="text-gray-600 mb-4">{selectedIntegration.description}</p>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-2">
                    Configuration
                  </label>
                  <p className="text-sm text-gray-500 mb-2">
                    {selectedIntegration.setup_guide || 'Configure the integration settings below.'}
                  </p>
                  {/* Configuration form would go here based on config_schema */}
                  <div className="bg-gray-50 border rounded-lg p-4">
                    <p className="text-sm text-gray-600">
                      Configuration form based on integration schema will be displayed here.
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <div className="p-6 border-t flex justify-end gap-3">
              <button
                onClick={() => {
                  setShowInstallModal(false);
                  setSelectedIntegration(null);
                }}
                className="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition-all"
              >
                Cancel
              </button>
              <button
                onClick={installIntegration}
                disabled={isLoading}
                className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-all disabled:opacity-50 flex items-center gap-2"
              >
                {isLoading ? (
                  <>
                    <RefreshCw size={18} className="animate-spin" />
                    Installing...
                  </>
                ) : (
                  <>
                    <Plus size={18} />
                    Install Integration
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
