import React, { useState, useMemo } from 'react';
import { 
  IconCube, 
  IconEye, 
  IconPlus, 
  IconSearch, 
  IconFilter,
  IconX,
  IconChevronDown,
  IconChevronRight,
  IconDatabase
} from '@tabler/icons-react';
import type { ModelCatalogNode } from '../../types/model';
import { extractJoinPaths, getAllAvailableMembers, type JoinPath } from '../../utils/cubeJoinUtils';
import { validateEntityName } from '../../utils/nameValidation';

interface EnhancedModelCatalogProps {
  models: ModelCatalogNode[];
  selectedModel: ModelCatalogNode | null;
  onModelSelect: (model: ModelCatalogNode) => void;
  onCreateView: (baseModel: ModelCatalogNode) => void;
  searchValue?: string;
  onSearchChange?: (value: string) => void;
}

interface ModelDetails {
  joinPaths: JoinPath[];
  dimensions: number;
  measures: number;
  totalMembers: number;
}

const EnhancedModelCatalog: React.FC<EnhancedModelCatalogProps> = ({
  models,
  selectedModel,
  onModelSelect,
  onCreateView,
  searchValue = '',
  onSearchChange
}) => {
  const [expandedModels, setExpandedModels] = useState<Set<string>>(new Set());
  const [filterType, setFilterType] = useState<'all' | 'cubes' | 'views'>('all');
  const [showDetails, setShowDetails] = useState<boolean>(true);

  // Enhanced model processing with join path and member analysis
  const enhancedModels = useMemo(() => {
    return models.map(model => {
      const joinPaths = extractJoinPaths(model);
      const members = getAllAvailableMembers(model, models);
      const mainDimensions = members.mainCube.filter(m => m.type === 'dimension').length;
      const mainMeasures = members.mainCube.filter(m => m.type === 'measure').length;
      const joinedMembers = Object.values(members.joinedCubes).flat().length;
      
      // Validate model name
      const nameValidation = validateEntityName(model.model_key, 'cube');
      
      const details: ModelDetails = {
        joinPaths,
        dimensions: mainDimensions,
        measures: mainMeasures,
        totalMembers: mainDimensions + mainMeasures + joinedMembers
      };

      return {
        ...model,
        details,
        nameValidation,
        hasJoins: joinPaths.length > 0,
        isComplex: joinPaths.length > 2 || details.totalMembers > 20
      };
    });
  }, [models]);

  // Filtered models based on search and type
  const filteredModels = useMemo(() => {
    let filtered = enhancedModels;

    // Filter by type
    if (filterType !== 'all') {
      filtered = filtered.filter(model => {
        const isView = model.model_key.includes('view') || model.display_name?.includes('view');
        return filterType === 'views' ? isView : !isView;
      });
    }

    // Filter by search
    if (searchValue.trim()) {
      const searchLower = searchValue.toLowerCase();
      filtered = filtered.filter(model =>
        model.model_key.toLowerCase().includes(searchLower) ||
        model.display_name?.toLowerCase().includes(searchLower) ||
        model.details.joinPaths.some(join => join.path.toLowerCase().includes(searchLower))
      );
    }

    return filtered.sort((a, b) => {
      // Sort by complexity (simpler first), then alphabetically
      const complexityDiff = (a.isComplex ? 1 : 0) - (b.isComplex ? 1 : 0);
      if (complexityDiff !== 0) return complexityDiff;
      return a.model_key.localeCompare(b.model_key);
    });
  }, [enhancedModels, searchValue, filterType]);

  const toggleModelExpansion = (modelKey: string) => {
    const newExpanded = new Set(expandedModels);
    if (newExpanded.has(modelKey)) {
      newExpanded.delete(modelKey);
    } else {
      newExpanded.add(modelKey);
    }
    setExpandedModels(newExpanded);
  };

  const getModelIcon = (model: any) => {
    if (model.hasJoins) {
      return <IconDatabase className="w-4 h-4 text-blue-600" />;
    }
    return <IconCube className="w-4 h-4 text-gray-600" />;
  };

  const getComplexityColor = (model: any) => {
    if (model.details.totalMembers > 50) return 'text-red-600';
    if (model.details.totalMembers > 20) return 'text-orange-600';
    if (model.details.totalMembers > 10) return 'text-yellow-600';
    return 'text-green-600';
  };

  return (
    <div className="h-full flex flex-col bg-white">
      {/* Header */}
      <div className="p-4 border-b border-gray-200">
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-lg font-semibold text-gray-900">Model Catalog</h2>
          <div className="flex items-center space-x-2">
            <button
              onClick={() => setShowDetails(!showDetails)}
              className="p-2 text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded"
              title={showDetails ? 'Hide details' : 'Show details'}
            >
              <IconEye className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Search */}
        <div className="relative mb-3">
          <IconSearch className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-gray-400" />
          <input
            type="text"
            value={searchValue}
            onChange={(e) => onSearchChange?.(e.target.value)}
            placeholder="Search models, joins, or members..."
            className="w-full pl-9 pr-9 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          />
          {searchValue && (
            <button
              onClick={() => onSearchChange?.('')}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
              title="Clear search"
            >
              <IconX className="w-4 h-4" />
            </button>
          )}
        </div>

        {/* Filter Tabs */}
        <div className="flex space-x-1 bg-gray-100 rounded-lg p-1">
          {(['all', 'cubes', 'views'] as const).map((type) => (
            <button
              key={type}
              onClick={() => setFilterType(type)}
              className={`flex-1 px-3 py-1 text-sm font-medium rounded-md transition-colors ${
                filterType === type
                  ? 'bg-white text-blue-600 shadow-sm'
                  : 'text-gray-600 hover:text-gray-900'
              }`}
            >
              {type.charAt(0).toUpperCase() + type.slice(1)}
            </button>
          ))}
        </div>
      </div>

      {/* Model List */}
      <div className="flex-1 overflow-y-auto">
        {filteredModels.length === 0 ? (
          <div className="p-4 text-center text-gray-500">
            <IconFilter className="w-8 h-8 mx-auto mb-2 opacity-50" />
            <p>No models found</p>
            {searchValue && (
              <p className="text-sm mt-1">Try adjusting your search terms</p>
            )}
          </div>
        ) : (
          <div className="space-y-1 p-2">
            {filteredModels.map((model) => {
              const isSelected = selectedModel?.model_key === model.model_key;
              const isExpanded = expandedModels.has(model.model_key);
              
              return (
                <div key={model.model_key} className="rounded-lg border border-gray-200">
                  {/* Model Header */}
                  <div
                    className={`p-3 cursor-pointer hover:bg-gray-50 ${
                      isSelected ? 'bg-blue-50 border-blue-200' : ''
                    }`}
                    onClick={() => onModelSelect(model)}
                  >
                    <div className="flex items-start space-x-3">
                      <div className="flex items-center space-x-2">
                        {model.details.joinPaths.length > 0 && (
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              toggleModelExpansion(model.model_key);
                            }}
                            className="text-gray-400 hover:text-gray-600"
                            title={isExpanded ? 'Collapse details' : 'Expand details'}
                          >
                            {isExpanded ? (
                              <IconChevronDown className="w-4 h-4" />
                            ) : (
                              <IconChevronRight className="w-4 h-4" />
                            )}
                          </button>
                        )}
                        {getModelIcon(model)}
                      </div>
                      
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center space-x-2">
                          <h3 className="font-medium text-gray-900 truncate">
                            {model.display_name || model.model_key}
                          </h3>
                          {!model.nameValidation.isValid && (
                            <span className="w-2 h-2 bg-yellow-400 rounded-full" title="Name validation issues" />
                          )}
                        </div>
                        
                        {showDetails && (
                          <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-gray-500">
                            <span className={`font-mono ${getComplexityColor(model)}`}>
                              {model.details.dimensions}D / {model.details.measures}M
                            </span>
                            {model.details.joinPaths.length > 0 && (
                              <span className="bg-blue-100 text-blue-700 px-2 py-0.5 rounded">
                                {model.details.joinPaths.length} joins
                              </span>
                            )}
                            {model.isComplex && (
                              <span className="bg-orange-100 text-orange-700 px-2 py-0.5 rounded">
                                Complex
                              </span>
                            )}
                          </div>
                        )}
                      </div>

                      <div className="flex items-center space-x-1">
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            onCreateView(model);
                          }}
                          className="p-1 text-gray-400 hover:text-blue-600 hover:bg-blue-50 rounded"
                          title="Create view from this model"
                        >
                          <IconPlus className="w-4 h-4" />
                        </button>
                      </div>
                    </div>
                  </div>

                  {/* Expanded Details */}
                  {isExpanded && (
                    <div className="border-t border-gray-100 bg-gray-50 p-3">
                      {/* Join Paths */}
                      {model.details.joinPaths.length > 0 && (
                        <div className="mb-3">
                          <h4 className="text-xs font-medium text-gray-700 uppercase tracking-wide mb-2">
                            Available Joins
                          </h4>
                          <div className="space-y-1">
                            {model.details.joinPaths.map((join) => (
                              <div
                                key={join.path}
                                className="flex items-center space-x-2 text-xs"
                              >
                                <IconDatabase className="w-3 h-3 text-gray-400" />
                                <span className="font-mono text-gray-700">{join.path}</span>
                                <span className="text-gray-500">({join.relationship})</span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* Name Validation Issues */}
                      {!model.nameValidation.isValid && (
                        <div className="mb-3">
                          <h4 className="text-xs font-medium text-yellow-700 uppercase tracking-wide mb-2">
                            Validation Issues
                          </h4>
                          <div className="space-y-1">
                            {model.nameValidation.errors.map((error, index) => (
                              <div key={index} className="text-xs text-yellow-700">
                                • {error}
                              </div>
                            ))}
                            {model.nameValidation.warnings.map((warning, index) => (
                              <div key={index} className="text-xs text-yellow-600">
                                ⚠ {warning}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* Quick Stats */}
                      <div className="grid grid-cols-2 gap-3 text-xs">
                        <div>
                          <span className="text-gray-500">Dimensions:</span>
                          <span className="ml-1 font-mono">{model.details.dimensions}</span>
                        </div>
                        <div>
                          <span className="text-gray-500">Measures:</span>
                          <span className="ml-1 font-mono">{model.details.measures}</span>
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Footer Stats */}
      <div className="border-t border-gray-200 p-3 bg-gray-50">
        <div className="text-xs text-gray-500 text-center">
          {filteredModels.length} of {models.length} models
          {searchValue && ` • Filtered by "${searchValue}"`}
        </div>
      </div>
    </div>
  );
};

export default EnhancedModelCatalog;
