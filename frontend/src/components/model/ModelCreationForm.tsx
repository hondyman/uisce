import React, { useState, useCallback } from 'react';
import { 
  IconPlus, 
  IconTrash, 
  IconEye, 
  IconDatabase, 
  IconSettings,
  IconCheck,
  IconAlertTriangle
} from '@tabler/icons-react';
import NameValidationInput from '../common/NameValidationInput';
import JoinPathSelector from '../common/JoinPathSelector';
import HierarchyDrillMembersEditor from './HierarchyDrillMembersEditor';
import type { ModelCatalogNode } from '../../types/model';
import { 
  generateViewConfig, 
  type JoinPathReference,
  type CubeMember 
} from '../../utils/cubeJoinUtils';
import { validateEntityName } from '../../utils/nameValidation';

interface PreAggregation {
  name: string;
  dimensions: string[];
  measures: string[];
  timeDimensions: Array<{
    dimension: string;
    granularity: string;
  }>;
}

interface Hierarchy {
  name: string;
  title: string;
  levels: Array<{
    name: string;
    title: string;
    dimension: string;
    time_granularity?: string;
  }>;
}

interface ViewConfig {
  name: string;
  description: string;
  baseCube: string;
  joinPaths: JoinPathReference[];
  dimensions: string[];
  measures: string[];
  hierarchies: Hierarchy[];
  drillMembers: string[];
  preAggregations: PreAggregation[];
}

interface ModelCreationFormProps {
  availableModels: ModelCatalogNode[];
  onSubmit: (config: ViewConfig) => void;
  onCancel: () => void;
  initialConfig?: Partial<ViewConfig>;
}

const ModelCreationForm: React.FC<ModelCreationFormProps> = ({
  availableModels,
  onSubmit,
  onCancel,
  initialConfig
}) => {
  const [config, setConfig] = useState<ViewConfig>({
    name: initialConfig?.name || '',
    description: initialConfig?.description || '',
    baseCube: initialConfig?.baseCube || '',
    joinPaths: initialConfig?.joinPaths || [],
    dimensions: initialConfig?.dimensions || [],
    measures: initialConfig?.measures || [],
    hierarchies: initialConfig?.hierarchies || [],
    drillMembers: initialConfig?.drillMembers || [],
    preAggregations: initialConfig?.preAggregations || []
  });

  const [availableMembers, setAvailableMembers] = useState<{
    dimensions: CubeMember[];
    measures: CubeMember[];
  }>({ dimensions: [], measures: [] });

  const [activeTab, setActiveTab] = useState<'basic' | 'joins' | 'members' | 'hierarchies' | 'preaggs' | 'preview'>('basic');
  const [validationErrors, setValidationErrors] = useState<string[]>([]);

  const selectedModel = availableModels.find(m => m.model_key === config.baseCube);

  const validateForm = useCallback(() => {
    const errors: string[] = [];

    // Validate name
    const nameValidation = validateEntityName(config.name, 'view');
    if (!nameValidation.isValid) {
      errors.push(`Name: ${nameValidation.errors.join(', ')}`);
    }

    // Validate base cube
    if (!config.baseCube) {
      errors.push('Base cube is required');
    }

    // Validate description
    if (!config.description.trim()) {
      errors.push('Description is required');
    }

    // Validate pre-aggregations
    config.preAggregations.forEach((preAgg, index) => {
      const preAggValidation = validateEntityName(preAgg.name, 'pre-aggregation');
      if (!preAggValidation.isValid) {
        errors.push(`Pre-aggregation ${index + 1} name: ${preAggValidation.errors.join(', ')}`);
      }
      if (preAgg.dimensions.length === 0 && preAgg.measures.length === 0) {
        errors.push(`Pre-aggregation ${index + 1} must have at least one dimension or measure`);
      }
    });

    setValidationErrors(errors);
    return errors.length === 0;
  }, [config]);

  const handleSubmit = () => {
    if (validateForm()) {
      onSubmit(config);
    }
  };

  const handleMembersChange = useCallback((members: { dimensions: CubeMember[]; measures: CubeMember[] }) => {
    setAvailableMembers(members);
  }, []);

  const addPreAggregation = () => {
    const newPreAgg: PreAggregation = {
      name: `${config.name}_agg_${config.preAggregations.length + 1}`,
      dimensions: [],
      measures: [],
      timeDimensions: []
    };
    setConfig(prev => ({
      ...prev,
      preAggregations: [...prev.preAggregations, newPreAgg]
    }));
  };

  const updatePreAggregation = (index: number, updates: Partial<PreAggregation>) => {
    setConfig(prev => ({
      ...prev,
      preAggregations: prev.preAggregations.map((preAgg, i) => 
        i === index ? { ...preAgg, ...updates } : preAgg
      )
    }));
  };

  const removePreAggregation = (index: number) => {
    setConfig(prev => ({
      ...prev,
      preAggregations: prev.preAggregations.filter((_, i) => i !== index)
    }));
  };

  const generatePreview = () => {
    if (!selectedModel) return '';
    
    try {
      return JSON.stringify(generateViewConfig({
        name: config.name,
        description: config.description,
        baseCube: config.baseCube,
        joinPathReferences: config.joinPaths
      }), null, 2);
    } catch (error) {
      return `Error generating preview: ${error instanceof Error ? error.message : 'Unknown error'}`;
    }
  };

  const tabs = [
    { id: 'basic', label: 'Basic Info', icon: IconSettings },
    { id: 'joins', label: 'Join Paths', icon: IconDatabase },
    { id: 'members', label: 'Members', icon: IconPlus },
    { id: 'hierarchies', label: 'Hierarchies & Drill', icon: IconDatabase },
    { id: 'preaggs', label: 'Pre-Aggregations', icon: IconSettings },
    { id: 'preview', label: 'Preview', icon: IconEye }
  ] as const;

  return (
    <div className="max-w-4xl mx-auto bg-white rounded-lg shadow-lg">
      {/* Header */}
      <div className="border-b border-gray-200 p-6">
        <h2 className="text-2xl font-bold text-gray-900">
          {initialConfig ? 'Edit View' : 'Create New View'}
        </h2>
        <p className="mt-1 text-gray-600">
          Configure a semantic view based on your cube models
        </p>
      </div>

      {/* Tabs */}
      <div className="border-b border-gray-200">
        <div className="flex space-x-8 px-6">
          {tabs.map((tab) => {
            const Icon = tab.icon;
            return (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`py-4 px-1 border-b-2 font-medium text-sm flex items-center space-x-2 ${
                  activeTab === tab.id
                    ? 'border-blue-500 text-blue-600'
                    : 'border-transparent text-gray-500 hover:text-gray-700'
                }`}
              >
                <Icon className="w-4 h-4" />
                <span>{tab.label}</span>
              </button>
            );
          })}
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {/* Basic Info Tab */}
        {activeTab === 'basic' && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              <NameValidationInput
                label="View Name"
                value={config.name}
                onChange={(name) => setConfig(prev => ({ ...prev, name }))}
                type="view"
                required
                placeholder="Enter view name"
              />
              
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Base Cube *
                </label>
                <select
                  value={config.baseCube}
                  onChange={(e) => setConfig(prev => ({ ...prev, baseCube: e.target.value }))}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  required
                  title="Select a base cube for the view"
                >
                  <option value="">Select a base cube</option>
                  {availableModels.map((model) => (
                    <option key={model.model_key} value={model.model_key}>
                      {model.model_key}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-2">
                Description *
              </label>
              <textarea
                value={config.description}
                onChange={(e) => setConfig(prev => ({ ...prev, description: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                rows={3}
                placeholder="Describe what this view represents and how it should be used"
                required
              />
            </div>
          </div>
        )}

        {/* Join Paths Tab */}
        {activeTab === 'joins' && (
          <JoinPathSelector
            selectedModel={selectedModel || null}
            allModels={availableModels}
            selectedJoinPaths={config.joinPaths}
            onJoinPathsChange={(joinPaths) => setConfig(prev => ({ ...prev, joinPaths }))}
            onMembersChange={handleMembersChange}
          />
        )}

        {/* Members Tab */}
        {activeTab === 'members' && (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-6">
              {/* Dimensions */}
              <div>
                <h3 className="text-lg font-medium mb-4">Dimensions</h3>
                <div className="max-h-96 overflow-y-auto border border-gray-200 rounded-lg">
                  {availableMembers.dimensions.map((dimension) => {
                    const memberKey = dimension.joinPath ? `${dimension.joinPath}.${dimension.name}` : dimension.name;
                    return (
                      <label
                        key={memberKey}
                        className="flex items-center space-x-3 p-3 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                      >
                        <input
                          type="checkbox"
                          checked={config.dimensions.includes(memberKey)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setConfig(prev => ({
                                ...prev,
                                dimensions: [...prev.dimensions, memberKey]
                              }));
                            } else {
                              setConfig(prev => ({
                                ...prev,
                                dimensions: prev.dimensions.filter(d => d !== memberKey)
                              }));
                            }
                          }}
                          className="text-blue-600"
                        />
                        <div className="flex-1">
                          <div className="font-mono text-sm">{memberKey}</div>
                          {dimension.title && dimension.title !== dimension.name && (
                            <div className="text-xs text-gray-500">{dimension.title}</div>
                          )}
                        </div>
                      </label>
                    );
                  })}
                </div>
              </div>

              {/* Measures */}
              <div>
                <h3 className="text-lg font-medium mb-4">Measures</h3>
                <div className="max-h-96 overflow-y-auto border border-gray-200 rounded-lg">
                  {availableMembers.measures.map((measure) => {
                    const memberKey = measure.joinPath ? `${measure.joinPath}.${measure.name}` : measure.name;
                    return (
                      <label
                        key={memberKey}
                        className="flex items-center space-x-3 p-3 hover:bg-gray-50 border-b border-gray-100 last:border-b-0"
                      >
                        <input
                          type="checkbox"
                          checked={config.measures.includes(memberKey)}
                          onChange={(e) => {
                            if (e.target.checked) {
                              setConfig(prev => ({
                                ...prev,
                                measures: [...prev.measures, memberKey]
                              }));
                            } else {
                              setConfig(prev => ({
                                ...prev,
                                measures: prev.measures.filter(m => m !== memberKey)
                              }));
                            }
                          }}
                          className="text-blue-600"
                        />
                        <div className="flex-1">
                          <div className="font-mono text-sm">{memberKey}</div>
                          {measure.title && measure.title !== measure.name && (
                            <div className="text-xs text-gray-500">{measure.title}</div>
                          )}
                        </div>
                      </label>
                    );
                  })}
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Hierarchies & Drill Members Tab */}
        {activeTab === 'hierarchies' && (
          <div className="p-6">
            <HierarchyDrillMembersEditor
              hierarchies={config.hierarchies}
              drillMembers={config.drillMembers}
              availableDimensions={availableMembers.dimensions.map(d => d.name)}
              onHierarchiesChange={(hierarchies) => setConfig(prev => ({ ...prev, hierarchies }))}
              onDrillMembersChange={(drillMembers) => setConfig(prev => ({ ...prev, drillMembers }))}
            />
          </div>
        )}

        {/* Pre-Aggregations Tab */}
        {activeTab === 'preaggs' && (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-medium">Pre-Aggregations</h3>
              <button
                onClick={addPreAggregation}
                className="flex items-center space-x-2 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
              >
                <IconPlus className="w-4 h-4" />
                <span>Add Pre-Aggregation</span>
              </button>
            </div>

            {config.preAggregations.length === 0 ? (
              <div className="text-center py-8 text-gray-500">
                <p>No pre-aggregations defined</p>
                <p className="text-sm mt-1">Add pre-aggregations to improve query performance</p>
              </div>
            ) : (
              <div className="space-y-4">
                {config.preAggregations.map((preAgg, index) => (
                  <div key={index} className="border border-gray-200 rounded-lg p-4">
                    <div className="flex items-center justify-between mb-4">
                      <NameValidationInput
                        label=""
                        value={preAgg.name}
                        onChange={(name) => updatePreAggregation(index, { name })}
                        type="pre-aggregation"
                        placeholder="Pre-aggregation name"
                        className="flex-1 mr-4"
                      />
                      <button
                        onClick={() => removePreAggregation(index)}
                        className="text-red-600 hover:text-red-700"
                        title="Remove pre-aggregation"
                      >
                        <IconTrash className="w-4 h-4" />
                      </button>
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Dimensions
                        </label>
                        <div className="max-h-32 overflow-y-auto border border-gray-200 rounded">
                          {availableMembers.dimensions.map((dimension) => {
                            const memberKey = dimension.joinPath ? `${dimension.joinPath}.${dimension.name}` : dimension.name;
                            return (
                              <label
                                key={memberKey}
                                className="flex items-center space-x-2 p-2 hover:bg-gray-50"
                              >
                                <input
                                  type="checkbox"
                                  checked={preAgg.dimensions.includes(memberKey)}
                                  onChange={(e) => {
                                    const dimensions = e.target.checked
                                      ? [...preAgg.dimensions, memberKey]
                                      : preAgg.dimensions.filter(d => d !== memberKey);
                                    updatePreAggregation(index, { dimensions });
                                  }}
                                  className="text-blue-600"
                                />
                                <span className="text-xs font-mono">{memberKey}</span>
                              </label>
                            );
                          })}
                        </div>
                      </div>

                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Measures
                        </label>
                        <div className="max-h-32 overflow-y-auto border border-gray-200 rounded">
                          {availableMembers.measures.map((measure) => {
                            const memberKey = measure.joinPath ? `${measure.joinPath}.${measure.name}` : measure.name;
                            return (
                              <label
                                key={memberKey}
                                className="flex items-center space-x-2 p-2 hover:bg-gray-50"
                              >
                                <input
                                  type="checkbox"
                                  checked={preAgg.measures.includes(memberKey)}
                                  onChange={(e) => {
                                    const measures = e.target.checked
                                      ? [...preAgg.measures, memberKey]
                                      : preAgg.measures.filter(m => m !== memberKey);
                                    updatePreAggregation(index, { measures });
                                  }}
                                  className="text-blue-600"
                                />
                                <span className="text-xs font-mono">{memberKey}</span>
                              </label>
                            );
                          })}
                        </div>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Preview Tab */}
        {activeTab === 'preview' && (
          <div className="space-y-6">
            <h3 className="text-lg font-medium">Generated Configuration</h3>
            <div className="bg-gray-50 rounded-lg p-4">
              <pre className="text-sm text-gray-800 whitespace-pre-wrap overflow-auto max-h-96">
                {generatePreview()}
              </pre>
            </div>
          </div>
        )}
      </div>

      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <div className="mx-6 mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
          <div className="flex items-center space-x-2 mb-2">
            <IconAlertTriangle className="w-5 h-5 text-red-600" />
            <h4 className="font-medium text-red-800">Please fix the following errors:</h4>
          </div>
          <ul className="list-disc list-inside space-y-1 text-sm text-red-700">
            {validationErrors.map((error, index) => (
              <li key={index}>{error}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Footer */}
      <div className="border-t border-gray-200 p-6 flex items-center justify-between">
        <button
          onClick={onCancel}
          className="px-6 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50"
        >
          Cancel
        </button>
        
        <div className="flex items-center space-x-3">
          <button
            onClick={() => validateForm()}
            className="px-4 py-2 text-blue-600 border border-blue-600 rounded-md hover:bg-blue-50"
          >
            Validate
          </button>
          <button
            onClick={handleSubmit}
            disabled={validationErrors.length > 0}
            className="flex items-center space-x-2 px-6 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <IconCheck className="w-4 h-4" />
            <span>{initialConfig ? 'Update View' : 'Create View'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};

export default ModelCreationForm;
