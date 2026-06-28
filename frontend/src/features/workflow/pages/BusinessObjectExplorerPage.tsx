import React, { useState, useEffect, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
} from '@mui/material';
import { EditBusinessObjectModal } from '../../../components/BusinessObjectManager/EditBusinessObjectModal';
import { BusinessObjectWizard } from '../../../components/BusinessObjectManager/BusinessObjectWizard';
import { useTenant } from '../../../contexts/TenantContext';
import { useConfirm } from '../../../components/ConfirmProvider';
import { useNotification } from '../../../hooks/useNotification';
import { devDebug } from '../../../utils/devLogger';
import { getSelectedRegion } from '../../../lib/region';

interface BusinessObject {
  id: string;
  name: string;
  display_name: string;
  description?: string;
  config?: {
    is_active?: boolean;
    fields?: Array<{
      key: string;
      name: string;
      displayName?: string;
      technicalName?: string;
      type: string;
      isCore?: boolean;
    }>;
  };
  is_active?: boolean;
  enable_history?: boolean;
  status?: 'draft' | 'active' | 'deprecated';
  updated_at?: string;
  subtypes?: Record<
    string,
    {
      id: string;
      key: string;
      name: string;
      display_name: string;
      technical_name: string;
      description?: string;
      is_core: boolean;
      config?: {
        inheritedFields?: any[];
        customFields?: any[];
      };
    }
  >;
}

export const BusinessObjectExplorerPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const confirm = useConfirm();
  const notification = useNotification();
  const navigate = useNavigate();
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || datasource?.alpha_tenant_instance_id || '';

  const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

  const [selectedObjectId, setSelectedObjectId] = useState<string | null>(null);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingObject, setEditingObject] = useState<BusinessObject | null>(null);
  const [wizardOpen, setWizardOpen] = useState(false);

  const [activeTab, setActiveTab] = useState<'narrative' | 'fields' | 'subtypes'>('fields');

  // Helper to build headers with authentication
  const getAuthHeaders = (additionalHeaders: Record<string, string> = {}): Record<string, string> => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
    const authHeader = token && !token.includes('demo') ? `Bearer ${token}` : '';
    
    return {
      'Authorization': authHeader,
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
      'X-Tenant-Datasource-ID': datasourceId,
      'X-Tenant-Region': getSelectedRegion(),
      ...additionalHeaders,
    };
  };

  const fetchBusinessObjects = async () => {
    if (!tenantId || !datasourceId) {
      setBusinessObjects([]);
      setError('Please select a tenant and datasource');
      return;
    }

    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/business-objects', {
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to fetch business objects');
      }

      const data = await response.json();
      const dataArray = Array.isArray(data) ? data : Object.entries(data).map(([id, obj]: [string, any]) => ({ ...obj, id }));
      
      let objectsArray = dataArray.map((obj: any) => {
        const id = obj.id;
        const config = obj.config || {};
        
        const normalizedConfig = {
          ...config,
          fields: (config.entity_fields || []).map((field: any) => ({
             key: field.key,
             name: field.name,
             displayName: field.businessName || field.displayName || field.name,
             technicalName: field.technicalName || field.name,
             type: field.type,
             isCore: field.isCore,
          })),
        };

        const processedSubtypes: Record<string, any> = {};
        if (obj.subtypes) {
            Object.entries(obj.subtypes).forEach(([stId, st]: [string, any]) => {
                processedSubtypes[stId] = { ...st, is_core: st.config?.isCore ?? false };
            });
        }

        return {
          id: id,
          name: obj.name || obj.technical_name || id,
          display_name: obj.display_name || obj.name || obj.technical_name || id,
          description: obj.description,
          config: normalizedConfig,
          subtypes: processedSubtypes,
          is_active: normalizedConfig.is_active !== false,
          enable_history: obj.enableHistory || false,
          updated_at: obj.updated_at,
          parent_id: (obj.parentId && typeof obj.parentId === 'object' && 'Valid' in obj.parentId) 
            ? (obj.parentId.Valid ? obj.parentId.String : null)
            : (obj.parentId || null),
        };
      }).filter(obj => !obj.parent_id);

      if (objectsArray.length === 0) {
        const resp2 = await fetch('/api/business-objects/list', {
          headers: getAuthHeaders(),
        });
        if (resp2.ok) {
          const items = await resp2.json();
          objectsArray = (Array.isArray(items) ? items : []).map((item: any) => ({
            id: item.id,
            name: item.name || item.display_name || item.id,
            display_name: item.display_name || item.name || item.id,
            description: item.description,
            config: item.config || {},
            subtypes: {},
            is_active: (item.config?.is_active !== false),
            enable_history: false,
            updated_at: undefined,
            parent_id: null,
          }));
        }
      }

      setBusinessObjects(objectsArray);
      
      // Auto-select first object if none selected
      if (objectsArray.length > 0 && !selectedObjectId) {
        setSelectedObjectId(objectsArray[0].id);
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to fetch business objects';
      setError(errorMsg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (tenantId && datasourceId) {
      fetchBusinessObjects();
    }
  }, [tenantId, datasourceId]);

  const filteredBusinessObjects = useMemo(() => {
    if (!searchQuery.trim()) return businessObjects;
    const term = searchQuery.toLowerCase();
    return businessObjects.filter(obj => 
      obj.name.toLowerCase().includes(term) ||
      obj.display_name.toLowerCase().includes(term) ||
      obj.description?.toLowerCase().includes(term)
    );
  }, [businessObjects, searchQuery]);

  const selectedObject = useMemo(() => {
    return businessObjects.find(obj => obj.id === selectedObjectId) || null;
  }, [businessObjects, selectedObjectId]);

  const handleEditObject = (object: BusinessObject, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    setEditingObject(object);
    setEditModalOpen(true);
  };

  const handleDeleteObject = async (object: BusinessObject, e?: React.MouseEvent) => {
    if (e) e.stopPropagation();
    
    const confirmed = await confirm({
      title: 'Delete Business Object',
      description: `Are you sure you want to delete "${object.display_name}"? This action cannot be undone.`,
      confirmText: 'Delete',
      cancelText: 'Cancel',
    });

    if (!confirmed) return;

    try {
      const response = await fetch(`/api/business-objects/${object.id}`, {
        method: 'DELETE',
        headers: getAuthHeaders(),
      });

      if (!response.ok) {
        throw new Error('Failed to delete business object');
      }

      setBusinessObjects(prev => prev.filter(obj => obj.id !== object.id));
      notification.success(`"${object.display_name}" deleted successfully`);
      if (selectedObjectId === object.id) {
        setSelectedObjectId(null);
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to delete business object';
      notification.error(errorMsg);
    }
  };

  const handleToggleStatus = async (object: BusinessObject, e: React.MouseEvent) => {
    e.stopPropagation();
    const newStatus = !object.is_active;
    
    try {
      const payload = {
        ...object,
        is_active: newStatus,
        config: {
          ...object.config,
          is_active: newStatus
        }
      };

      const response = await fetch(`/api/business-objects/${object.id}`, {
        method: 'PATCH',
        headers: getAuthHeaders(),
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        throw new Error('Failed to update status');
      }

      setBusinessObjects(prev => prev.map(obj => obj.id === object.id ? { ...obj, is_active: newStatus } : obj));
      notification.success(`Business Object "${object.display_name}" is now ${newStatus ? 'Active' : 'Draft'}`);
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Failed to update status');
    }
  };

  const handleSaveBusinessObject = async (objectData: any) => {
    try {
      const isEditMode = !!editingObject?.id;
      const method = isEditMode ? 'PATCH' : 'POST';
      const url = isEditMode 
        ? `/api/business-objects/${editingObject?.id}`
        : '/api/business-objects';

      const response = await fetch(url, {
        method,
        headers: getAuthHeaders(),
        body: JSON.stringify(objectData),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || 'Failed to save business object');
      }

      const savedObject = await response.json();
      notification.success(`Business Object saved successfully!`);
      setEditModalOpen(false);
      setEditingObject(null);
      fetchBusinessObjects();
      
      if (!isEditMode && savedObject?.id) {
        setSelectedObjectId(savedObject.id);
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : 'Failed to save business object';
      notification.error(errorMsg);
    }
  };

  const handleWizardSave = (boId: string) => {
    notification.success('Business Object created successfully!');
    setWizardOpen(false);
    fetchBusinessObjects();
    setSelectedObjectId(boId);
  };

  return (
    <div className="flex h-screen w-full font-display bg-background-light dark:bg-[#0B0F13] text-text-light-primary dark:text-gray-100">
      {/* Side Navigation Bar */}
      <aside className="flex w-64 flex-col bg-white/5 dark:bg-[#11161D] border-r border-white/10 dark:border-white/10">
        <div className="flex h-full flex-col justify-between p-4">
          <div className="flex flex-col gap-6">
            <div className="flex items-center gap-3 p-2">
              <span className="material-symbols-outlined text-blue-500 text-3xl">verified_user</span>
              <h1 className="text-white text-xl font-bold">Regulator</h1>
            </div>
            <div className="flex flex-col gap-4">
              <div className="flex flex-col gap-2">
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/regulator-portal">
                  <span className="material-symbols-outlined">dashboard</span>
                  <p className="text-sm font-medium leading-normal">Dashboard</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/process-catalog">
                  <span className="material-symbols-outlined">menu_book</span>
                  <p className="text-sm font-medium leading-normal">Process Catalog</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg bg-blue-500/20 text-blue-400" href="#">
                  <span className="material-symbols-outlined">data_object</span>
                  <p className="text-sm font-medium leading-normal">Object Explorer</p>
                </a>
                <a className="flex items-center gap-3 px-3 py-2 rounded-lg text-gray-400 hover:bg-white/10 hover:text-white" href="/core/audit-explorer">
                  <span className="material-symbols-outlined">policy</span>
                  <p className="text-sm font-medium leading-normal">Audit Explorer</p>
                </a>
              </div>
            </div>
          </div>
        </div>
      </aside>

      {/* Main Content Area */}
      <main className="flex-1 p-8 overflow-y-auto">
        <div className="flex flex-col gap-8">
          {/* Page Header */}
          <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
            <div className="flex flex-col gap-1">
              <p className="text-white text-3xl font-bold leading-tight">Business Object Explorer</p>
              <p className="text-gray-400 text-sm font-normal leading-normal">
                Visualize lifecycle states and transitions for core business objects.
              </p>
            </div>
            <div className="flex items-center gap-4">
              <div className="relative">
                <span className="material-symbols-outlined absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">search</span>
                <input
                  className="w-64 pl-10 pr-4 py-2 rounded-lg border border-white/20 bg-[#1C2127] text-white focus:ring-blue-500 focus:border-blue-500 placeholder:text-gray-500"
                  placeholder="Search objects..."
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </div>
              <button 
                onClick={() => setWizardOpen(true)}
                className="flex items-center gap-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-bold py-2.5 px-4 text-sm transition-colors"
              >
                <span className="material-symbols-outlined text-base">add</span>
                Create Object
              </button>
            </div>
          </div>

          {error && (
            <div className="bg-red-950/40 border border-red-800 text-red-200 p-4 rounded-xl text-sm">
              {error}
            </div>
          )}

          {/* Object List & Detail Split View */}
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 h-[calc(100vh-200px)]">
            
            {/* Left Column: Object List */}
            <div className="lg:col-span-1 flex flex-col gap-4 rounded-xl border border-white/10 bg-white/5 p-4 overflow-y-auto">
              <h3 className="text-lg font-bold text-white mb-2">Registered Objects</h3>
              
              {loading && businessObjects.length === 0 ? (
                <div className="flex justify-center py-10">
                  <span className="text-gray-400 animate-pulse text-sm">Loading business objects...</span>
                </div>
              ) : filteredBusinessObjects.length === 0 ? (
                <div className="text-center py-10 text-gray-500 text-sm">
                  No business objects found.
                </div>
              ) : (
                filteredBusinessObjects.map((obj) => (
                  <div 
                    key={obj.id}
                    className={`p-4 rounded-lg border cursor-pointer transition-all ${selectedObjectId === obj.id ? 'border-blue-500 bg-blue-500/10' : 'border-white/10 bg-[#1C2127] hover:border-white/30'}`}
                    onClick={() => setSelectedObjectId(obj.id)}
                  >
                    <div className="flex justify-between items-start mb-2">
                      <span className="font-mono text-sm text-blue-400 font-bold">{obj.name}</span>
                      <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${obj.is_active ? 'bg-green-900/30 text-green-400' : 'bg-yellow-900/30 text-yellow-400'}`}>
                        {obj.is_active ? 'Active' : 'Draft'}
                      </span>
                    </div>
                    <p className="text-white text-sm font-medium">{obj.display_name}</p>
                    {obj.description && (
                      <p className="text-gray-400 text-xs mt-1.5 line-clamp-2">{obj.description}</p>
                    )}
                    <div className="mt-3 flex items-center justify-between text-xs text-gray-400">
                      <span>Fields: <span className="text-white">{obj.config?.fields?.length || 0}</span></span>
                      <span>Subtypes: <span className="text-white">{Object.keys(obj.subtypes || {}).length}</span></span>
                    </div>
                  </div>
                ))
              )}
            </div>

            {/* Right Column: Detail View */}
            <div className="lg:col-span-2 flex flex-col gap-6 rounded-xl border border-white/10 bg-white/5 p-6 overflow-y-auto">
              {selectedObject ? (
                <>
                  <div className="flex justify-between items-start border-b border-white/10 pb-4">
                    <div>
                      <h2 className="text-2xl font-bold text-white mb-1">{selectedObject.display_name}</h2>
                      <p className="text-gray-400 text-sm">{selectedObject.description || 'No description provided.'}</p>
                    </div>
                    <div className="flex gap-2">
                      <button 
                        onClick={(e) => handleToggleStatus(selectedObject, e)}
                        className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white bg-white/10 hover:bg-white/20 transition-colors"
                      >
                        <span className="material-symbols-outlined text-base">swap_horiz</span> {selectedObject.is_active ? 'Set to Draft' : 'Activate'}
                      </button>
                      <button 
                        onClick={() => handleEditObject(selectedObject)}
                        className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 transition-colors"
                      >
                        <span className="material-symbols-outlined text-base">edit</span> Edit
                      </button>
                      <button 
                        onClick={() => handleDeleteObject(selectedObject)}
                        className="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium text-white bg-red-600/80 hover:bg-red-700 transition-colors"
                      >
                        <span className="material-symbols-outlined text-base">delete</span> Delete
                      </button>
                    </div>
                  </div>

                  {/* Lifecycle Visualization */}
                  <div className="bg-[#1C2127] rounded-lg p-6 border border-white/10">
                    <h4 className="text-sm font-bold text-gray-400 uppercase mb-6">Lifecycle Progress</h4>
                    <div className="flex items-center justify-between relative">
                      {/* Connecting Line */}
                      <div className="absolute top-1/2 left-0 w-full h-0.5 bg-gray-700 -z-0 transform -translate-y-1/2"></div>
                      
                      {/* Step 1 */}
                      <div className="relative z-10 flex flex-col items-center gap-2">
                        <div className="w-8 h-8 rounded-full bg-green-500 flex items-center justify-center text-black font-bold">
                          <span className="material-symbols-outlined text-sm">check</span>
                        </div>
                        <span className="text-xs font-medium text-green-500">Draft</span>
                      </div>

                      {/* Step 2 */}
                      <div className="relative z-10 flex flex-col items-center gap-2">
                        <div className={`w-8 h-8 rounded-full flex items-center justify-center font-bold ${selectedObject.is_active ? 'bg-green-500 text-black' : 'bg-gray-700 border border-gray-600 text-gray-400'}`}>
                          {selectedObject.is_active ? <span className="material-symbols-outlined text-sm">check</span> : '2'}
                        </div>
                        <span className={`text-xs font-medium ${selectedObject.is_active ? 'text-green-500' : 'text-gray-500'}`}>Review</span>
                      </div>

                      {/* Step 3 */}
                      <div className="relative z-10 flex flex-col items-center gap-2">
                        <div className={`w-10 h-10 rounded-full flex items-center justify-center font-bold border-4 border-[#1C2127] ${selectedObject.is_active ? 'bg-blue-500 text-white shadow-[0_0_15px_rgba(59,130,246,0.5)]' : 'bg-gray-700 border-gray-600 text-gray-400'}`}>
                          {selectedObject.is_active ? <span className="material-symbols-outlined text-base">verified</span> : '3'}
                        </div>
                        <span className={`text-sm font-bold ${selectedObject.is_active ? 'text-white' : 'text-gray-500'}`}>Active</span>
                      </div>

                      {/* Step 4 */}
                      <div className="relative z-10 flex flex-col items-center gap-2">
                        <div className="w-8 h-8 rounded-full bg-gray-700 border border-gray-600 flex items-center justify-center text-gray-400">
                          4
                        </div>
                        <span className="text-xs font-medium text-gray-500">Archived</span>
                      </div>
                    </div>
                  </div>

                  {/* Tabs */}
                  <div className="flex flex-col gap-4">
                    <div className="flex border-b border-white/10">
                      <button 
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'fields' ? 'text-blue-500 border-blue-500' : 'text-gray-400 hover:text-white border-transparent'}`}
                        onClick={() => setActiveTab('fields')}
                      >
                        Fields ({selectedObject.config?.fields?.length || 0})
                      </button>
                      <button 
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'subtypes' ? 'text-blue-500 border-blue-500' : 'text-gray-400 hover:text-white border-transparent'}`}
                        onClick={() => setActiveTab('subtypes')}
                      >
                        Subtypes ({Object.keys(selectedObject.subtypes || {}).length})
                      </button>
                      <button 
                        className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${activeTab === 'narrative' ? 'text-blue-500 border-blue-500' : 'text-gray-400 hover:text-white border-transparent'}`}
                        onClick={() => setActiveTab('narrative')}
                      >
                        Narrative
                      </button>
                    </div>
                    
                    {activeTab === 'fields' && (
                      <div className="bg-[#1C2127] rounded-lg border border-white/10 overflow-hidden">
                        <table className="w-full text-left border-collapse text-sm">
                          <thead>
                            <tr className="border-b border-white/10 bg-white/5">
                              <th className="p-3 text-gray-400 font-medium">Display Name</th>
                              <th className="p-3 text-gray-400 font-medium">Technical Name</th>
                              <th className="p-3 text-gray-400 font-medium">Type</th>
                            </tr>
                          </thead>
                          <tbody>
                            {(!selectedObject.config?.fields || selectedObject.config.fields.length === 0) ? (
                              <tr>
                                <td colSpan={3} className="p-4 text-center text-gray-500">No fields defined.</td>
                              </tr>
                            ) : (
                              selectedObject.config.fields.map((f) => (
                                <tr key={f.key} className="border-b border-white/5 hover:bg-white/5">
                                  <td className="p-3 font-medium text-white">{f.displayName || f.name}</td>
                                  <td className="p-3 font-mono text-xs text-blue-400">{f.technicalName || f.key}</td>
                                  <td className="p-3"><span className="px-2 py-0.5 text-xs bg-white/10 rounded">{f.type || 'text'}</span></td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    )}

                    {activeTab === 'subtypes' && (
                      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                        {Object.keys(selectedObject.subtypes || {}).length === 0 ? (
                          <div className="col-span-2 text-center py-6 text-gray-500 text-sm">
                            No subtypes defined.
                          </div>
                        ) : (
                          Object.entries(selectedObject.subtypes || {}).map(([key, st]) => (
                            <div key={key} className="p-4 rounded-lg border border-white/10 bg-[#1C2127]">
                              <span className="text-blue-400 font-bold font-mono text-sm">{st.technical_name || key}</span>
                              <h4 className="text-white font-medium mt-1">{st.display_name || st.name}</h4>
                              {st.description && <p className="text-gray-400 text-xs mt-1.5">{st.description}</p>}
                            </div>
                          ))
                        )}
                      </div>
                    )}

                    {activeTab === 'narrative' && (
                      <div className="p-4 bg-[#1C2127] rounded-lg border border-white/10 text-sm text-gray-300 leading-relaxed">
                        <p className="mb-2"><strong className="text-white">AI Generated Narrative:</strong></p>
                        <p>
                          Business Object {selectedObject.display_name} is configured for the current active datasource. 
                          It has {selectedObject.config?.fields?.length || 0} fields mapped to catalog attributes.
                        </p>
                        <div className="mt-4 p-3 bg-blue-500/10 border border-blue-500/20 rounded flex gap-3 items-start">
                          <span className="material-symbols-outlined text-blue-400">info</span>
                          <div>
                            <p className="text-blue-200 font-medium">Auto-Ingestion Status</p>
                            <p className="text-blue-300/80 text-xs">This object automatically maps schema lineages when registered to a core driving table.</p>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                </>
              ) : (
                <div className="flex flex-col items-center justify-center h-full text-gray-500">
                  <span className="material-symbols-outlined text-6xl mb-4 opacity-20">data_object</span>
                  <p className="text-lg">Select an object to view details</p>
                </div>
              )}
            </div>

          </div>
        </div>
      </main>

      {/* Edit Object Modal */}
      {editModalOpen && editingObject && (
        <EditBusinessObjectModal
          open={editModalOpen}
          onClose={() => {
            setEditModalOpen(false);
            setEditingObject(null);
          }}
          onSave={handleSaveBusinessObject}
          businessObject={editingObject as any}
        />
      )}

      {/* Create Object Wizard */}
      {wizardOpen && (
        <BusinessObjectWizard
          open={wizardOpen}
          onClose={() => setWizardOpen(false)}
          onSave={handleWizardSave}
        />
      )}
    </div>
  );
};
