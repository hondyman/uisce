import React, { useState } from 'react';
import { devLog } from '../../utils/devLogger';
import EnhancedModelCatalog from '../model/EnhancedModelCatalog';
import ModelCreationForm from '../model/ModelCreationForm';
import DatabaseJoinExplorer from '../joins/DatabaseJoinExplorer';
import type { ModelCatalogNode } from '../../types/model';
import type { JoinSuggestion, GeneratedCube } from '../../services/joinExtractionService';

interface ModelWorkspaceProps {
  models: ModelCatalogNode[];
  datasourceId?: string;
  onModelUpdate?: (models: ModelCatalogNode[]) => void;
}

const ModelWorkspace: React.FC<ModelWorkspaceProps> = ({
  models,
  datasourceId
}) => {
  const [selectedModel, setSelectedModel] = useState<ModelCatalogNode | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [searchValue, setSearchValue] = useState('');
  const [activeTab, setActiveTab] = useState<'details' | 'joins'>('details');

  const handleCreateView = (baseModel: ModelCatalogNode) => {
    setSelectedModel(baseModel);
    setShowCreateForm(true);
  };

  const handleFormSubmit = (config: any) => {
  // Here you would typically send the config to your backend
  devLog('View configuration:', config);
    
    // For demo purposes, we'll just close the form
    setShowCreateForm(false);
    
    // You could also update the models list with the new view
    // onModelUpdate?.([...models, newViewModel]);
  };

  const handleFormCancel = () => {
    setShowCreateForm(false);
    setSelectedModel(null);
  };

  const handleJoinSelect = (join: JoinSuggestion) => {
  devLog('Join selected:', join);
    // You could integrate this with the model creation form
  };

  const handleCubeGenerate = (cube: GeneratedCube) => {
  devLog('Cube generated:', cube);
    // You could use this to create a new model in the catalog
  };

  // Extract table name from model key for join exploration
  const getTableName = (modelKey: string) => {
    // Assuming model keys follow patterns like "/cubes/table_name" or "/views/table_name"
    const parts = modelKey.split('/');
    return parts[parts.length - 1];
  };

  return (
    <div className="h-screen flex bg-gray-100">
      {/* Model Catalog Sidebar */}
      <div className="w-80 border-r border-gray-300 bg-white shadow-sm">
        <EnhancedModelCatalog
          models={models}
          selectedModel={selectedModel}
          onModelSelect={setSelectedModel}
          onCreateView={handleCreateView}
          searchValue={searchValue}
          onSearchChange={setSearchValue}
        />
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto">
        {showCreateForm ? (
          <div className="p-6">
            <ModelCreationForm
              availableModels={models}
              onSubmit={handleFormSubmit}
              onCancel={handleFormCancel}
              initialConfig={selectedModel ? { baseCube: selectedModel.model_key } : undefined}
            />
          </div>
        ) : selectedModel ? (
          <div className="p-6">
            <div className="max-w-4xl mx-auto">
              <div className="bg-white rounded-lg shadow-lg p-6">
                <div className="border-b border-gray-200 pb-4 mb-6">
                  <h2 className="text-2xl font-bold text-gray-900">
                    {String(selectedModel.display_name) || selectedModel.model_key}
                  </h2>
                  <p className="mt-2 text-gray-600">
                    Model details and configuration
                  </p>
                  
                  {/* Tab Navigation */}
                  <div className="mt-4">
                    <nav className="flex space-x-8">
                      <button
                        onClick={() => setActiveTab('details')}
                        className={`py-2 px-1 border-b-2 font-medium text-sm ${
                          activeTab === 'details'
                            ? 'border-blue-500 text-blue-600'
                            : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                        }`}
                      >
                        Model Details
                      </button>
                      {datasourceId && (
                        <button
                          onClick={() => setActiveTab('joins')}
                          className={`py-2 px-1 border-b-2 font-medium text-sm ${
                            activeTab === 'joins'
                              ? 'border-blue-500 text-blue-600'
                              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                          }`}
                        >
                          Join Explorer
                        </button>
                      )}
                    </nav>
                  </div>
                </div>

                {/* Tab Content */}
                {activeTab === 'details' ? (
                  <>
                    {/* Model Information */}
                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                  <div>
                    <h3 className="text-lg font-medium text-gray-900 mb-4">Basic Information</h3>
                    <dl className="space-y-2">
                      <div>
                        <dt className="text-sm font-medium text-gray-500">Model Key</dt>
                        <dd className="text-sm text-gray-900 font-mono">{selectedModel.model_key}</dd>
                      </div>
                      {selectedModel.display_name && (
                        <div>
                          <dt className="text-sm font-medium text-gray-500">Display Name</dt>
                          <dd className="text-sm text-gray-900">{String(selectedModel.display_name)}</dd>
                        </div>
                      )}
                      <div>
                        <dt className="text-sm font-medium text-gray-500">Type</dt>
                        <dd className="text-sm text-gray-900">
                          {selectedModel.model_key.includes('view') ? 'View' : 'Cube'}
                        </dd>
                      </div>
                    </dl>
                  </div>

                  <div>
                    <h3 className="text-lg font-medium text-gray-900 mb-4">Actions</h3>
                    <div className="space-y-3">
                      <button
                        onClick={() => handleCreateView(selectedModel)}
                        className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 transition-colors"
                      >
                        Create View from this Model
                      </button>
                      <button
                        className="w-full px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 transition-colors"
                        disabled
                      >
                        Edit Model Configuration
                      </button>
                    </div>
                  </div>
                </div>

                    {/* Configuration Preview */}
                    {selectedModel.resolved_config && (
                      <div className="mt-6">
                        <h3 className="text-lg font-medium text-gray-900 mb-4">Configuration</h3>
                        <div className="bg-gray-50 rounded-lg p-4 overflow-auto max-h-96">
                          <pre className="text-sm text-gray-800">
                            {typeof selectedModel.resolved_config === 'string' 
                              ? selectedModel.resolved_config 
                              : JSON.stringify(selectedModel.resolved_config, null, 2)
                            }
                          </pre>
                        </div>
                      </div>
                    )}
                  </>
                ) : (
                  /* Join Explorer Tab */
                  datasourceId && (
                    <DatabaseJoinExplorer
                      datasourceId={datasourceId}
                      selectedTable={getTableName(selectedModel.model_key)}
                      onJoinSelect={handleJoinSelect}
                      onCubeGenerate={handleCubeGenerate}
                    />
                  )
                )}
              </div>
            </div>
          </div>
        ) : (
          <div className="flex items-center justify-center h-full">
            <div className="text-center text-gray-500">
              <div className="text-4xl mb-4">📊</div>
              <h3 className="text-xl font-medium mb-2">No Model Selected</h3>
              <p className="text-gray-400">
                Select a model from the catalog to view details or create a new view
              </p>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default ModelWorkspace;
