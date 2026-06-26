import React, { useState, useEffect } from 'react';
import { IconChevronDown, IconChevronRight, IconDatabase } from '@tabler/icons-react';
import type { ModelCatalogNode } from '../../types/model';
import { 
  extractJoinPaths, 
  getAllAvailableMembers,
  type JoinPath,
  type CubeMember,
  type JoinPathReference
} from '../../utils/cubeJoinUtils';

interface JoinPathSelectorProps {
  selectedModel: ModelCatalogNode | null;
  allModels: ModelCatalogNode[];
  selectedJoinPaths: JoinPathReference[];
  onJoinPathsChange: (joinPaths: JoinPathReference[]) => void;
  onMembersChange?: (members: { dimensions: CubeMember[]; measures: CubeMember[] }) => void;
}

const JoinPathSelector: React.FC<JoinPathSelectorProps> = ({
  selectedModel,
  allModels,
  selectedJoinPaths,
  onJoinPathsChange,
  onMembersChange
}) => {
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set());
  const [availableMembers, setAvailableMembers] = useState<{
    mainCube: CubeMember[];
    joinedCubes: { [joinPath: string]: CubeMember[] };
  }>({ mainCube: [], joinedCubes: {} });

  useEffect(() => {
    if (selectedModel) {
      const members = getAllAvailableMembers(selectedModel, allModels);
      setAvailableMembers(members);
      
      // Notify parent of available members
      const allDimensions = [
        ...members.mainCube.filter(m => m.type === 'dimension'),
        ...Object.values(members.joinedCubes).flat().filter(m => m.type === 'dimension')
      ];
      const allMeasures = [
        ...members.mainCube.filter(m => m.type === 'measure'),
        ...Object.values(members.joinedCubes).flat().filter(m => m.type === 'measure')
      ];
      
      onMembersChange?.({ dimensions: allDimensions, measures: allMeasures });
    }
  }, [selectedModel, allModels, onMembersChange]);

  const availableJoinPaths = selectedModel ? extractJoinPaths(selectedModel) : [];

  const togglePath = (path: string) => {
    const newExpanded = new Set(expandedPaths);
    if (newExpanded.has(path)) {
      newExpanded.delete(path);
    } else {
      newExpanded.add(path);
    }
    setExpandedPaths(newExpanded);
  };

  const addJoinPath = (joinPath: JoinPath) => {
    const existing = selectedJoinPaths.find(ref => ref.joinPath === joinPath.path);
    if (!existing) {
      const newReference: JoinPathReference = {
        joinPath: joinPath.path,
        includes: ['*'], // Default to include all
        prefix: true // Default to prefix for clarity
      };
      onJoinPathsChange([...selectedJoinPaths, newReference]);
    }
  };

  const removeJoinPath = (joinPath: string) => {
    onJoinPathsChange(selectedJoinPaths.filter(ref => ref.joinPath !== joinPath));
  };

  const updateJoinPathIncludes = (joinPath: string, includes: string[] | '*') => {
    const updated = selectedJoinPaths.map(ref => 
      ref.joinPath === joinPath ? { ...ref, includes } : ref
    );
    onJoinPathsChange(updated);
  };

  const updateJoinPathOptions = (joinPath: string, options: Partial<JoinPathReference>) => {
    const updated = selectedJoinPaths.map(ref => 
      ref.joinPath === joinPath ? { ...ref, ...options } : ref
    );
    onJoinPathsChange(updated);
  };

  if (!selectedModel) {
    return (
      <div className="text-center text-gray-500 py-8">
        <IconDatabase className="w-8 h-8 mx-auto mb-2 opacity-50" />
        <p>Select a cube to view available join paths</p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Main Cube Info */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
        <div className="flex items-center space-x-2 mb-2">
          <IconDatabase className="w-5 h-5 text-blue-600" />
          <h3 className="font-medium text-blue-900">Main Cube</h3>
        </div>
        <div className="text-sm text-blue-800">
          <div className="font-mono">{selectedModel.model_key}</div>
          <div className="text-xs mt-1">
            {availableMembers.mainCube.filter(m => m.type === 'dimension').length} dimensions, 
            {availableMembers.mainCube.filter(m => m.type === 'measure').length} measures
          </div>
        </div>
      </div>

      {/* Available Join Paths */}
      <div>
        <h4 className="font-medium text-gray-900 mb-3">Available Join Paths</h4>
        
        {availableJoinPaths.length === 0 ? (
          <div className="text-center text-gray-500 py-4 bg-gray-50 rounded-lg">
            <p>No join paths available in this cube</p>
            <p className="text-xs mt-1">This cube doesn't define any joins to other tables</p>
          </div>
        ) : (
          <div className="space-y-2">
            {availableJoinPaths.map((joinPath) => {
              const isSelected = selectedJoinPaths.some(ref => ref.joinPath === joinPath.path);
              const isExpanded = expandedPaths.has(joinPath.path);
              const selectedRef = selectedJoinPaths.find(ref => ref.joinPath === joinPath.path);
              const joinedMembers = availableMembers.joinedCubes[joinPath.path] || [];

              return (
                <div key={joinPath.path} className="border border-gray-200 rounded-lg">
                  {/* Join Path Header */}
                  <div className="p-3 bg-gray-50 flex items-center justify-between">
                    <div className="flex items-center space-x-2">
                      <button
                        onClick={() => togglePath(joinPath.path)}
                        className="text-gray-400 hover:text-gray-600"
                      >
                        {isExpanded ? (
                          <IconChevronDown className="w-4 h-4" />
                        ) : (
                          <IconChevronRight className="w-4 h-4" />
                        )}
                      </button>
                      <div>
                        <div className="font-mono text-sm font-medium">{joinPath.path}</div>
                        <div className="text-xs text-gray-500">
                          {joinPath.relationship} • {joinedMembers.length} members
                        </div>
                      </div>
                    </div>
                    
                    <div className="flex items-center space-x-2">
                      {isSelected ? (
                        <button
                          onClick={() => removeJoinPath(joinPath.path)}
                          className="px-3 py-1 text-xs bg-red-100 text-red-700 rounded hover:bg-red-200"
                        >
                          Remove
                        </button>
                      ) : (
                        <button
                          onClick={() => addJoinPath(joinPath)}
                          className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded hover:bg-blue-200"
                        >
                          Add to View
                        </button>
                      )}
                    </div>
                  </div>

                  {/* Expanded Content */}
                  {isExpanded && (
                    <div className="p-3 border-t border-gray-200">
                      {joinPath.description && (
                        <p className="text-sm text-gray-600 mb-3">{joinPath.description}</p>
                      )}

                      {/* Available Members */}
                      <div className="mb-4">
                        <h5 className="text-sm font-medium mb-2">Available Members</h5>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <h6 className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-1">
                              Dimensions ({joinedMembers.filter(m => m.type === 'dimension').length})
                            </h6>
                            <div className="space-y-1">
                              {joinedMembers
                                .filter(m => m.type === 'dimension')
                                .map(member => (
                                  <div key={member.name} className="text-xs font-mono text-gray-700">
                                    {member.name}
                                    {member.title && member.title !== member.name && (
                                      <span className="text-gray-500"> - {member.title}</span>
                                    )}
                                  </div>
                                ))}
                            </div>
                          </div>
                          
                          <div>
                            <h6 className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-1">
                              Measures ({joinedMembers.filter(m => m.type === 'measure').length})
                            </h6>
                            <div className="space-y-1">
                              {joinedMembers
                                .filter(m => m.type === 'measure')
                                .map(member => (
                                  <div key={member.name} className="text-xs font-mono text-gray-700">
                                    {member.name}
                                    {member.title && member.title !== member.name && (
                                      <span className="text-gray-500"> - {member.title}</span>
                                    )}
                                  </div>
                                ))}
                            </div>
                          </div>
                        </div>
                      </div>

                      {/* Configuration Options (when selected) */}
                      {isSelected && selectedRef && (
                        <div className="bg-blue-50 p-3 rounded">
                          <h5 className="text-sm font-medium mb-3">Configuration Options</h5>
                          
                          <div className="space-y-3">
                            {/* Includes */}
                            <div>
                              <label className="text-xs font-medium text-gray-700">Include Members</label>
                              <div className="mt-1 space-y-1">
                                <label className="flex items-center space-x-2">
                                  <input
                                    type="radio"
                                    checked={selectedRef.includes === '*' || (Array.isArray(selectedRef.includes) && selectedRef.includes.includes('*'))}
                                    onChange={() => updateJoinPathIncludes(joinPath.path, '*')}
                                    className="text-blue-600"
                                  />
                                  <span className="text-xs">All members (*)</span>
                                </label>
                                <label className="flex items-center space-x-2">
                                  <input
                                    type="radio"
                                    checked={Array.isArray(selectedRef.includes) && !selectedRef.includes.includes('*')}
                                    onChange={() => updateJoinPathIncludes(joinPath.path, [])}
                                    className="text-blue-600"
                                  />
                                  <span className="text-xs">Specific members</span>
                                </label>
                              </div>
                            </div>

                            {/* Options */}
                            <div className="grid grid-cols-2 gap-3">
                              <label className="flex items-center space-x-2">
                                <input
                                  type="checkbox"
                                  checked={selectedRef.prefix || false}
                                  onChange={(e) => updateJoinPathOptions(joinPath.path, { prefix: e.target.checked })}
                                  className="text-blue-600"
                                />
                                <span className="text-xs">Prefix member names</span>
                              </label>
                              
                              <div>
                                <label className="block text-xs font-medium text-gray-700">Alias</label>
                                <input
                                  type="text"
                                  value={selectedRef.alias || ''}
                                  onChange={(e) => updateJoinPathOptions(joinPath.path, { alias: e.target.value })}
                                  placeholder="Optional alias"
                                  className="mt-1 block w-full px-2 py-1 text-xs border border-gray-300 rounded"
                                />
                              </div>
                            </div>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Selected Join Paths Summary */}
      {selectedJoinPaths.length > 0 && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4">
          <h4 className="font-medium text-green-900 mb-2">Selected Join Paths</h4>
          <div className="space-y-2">
            {selectedJoinPaths.map((ref) => (
              <div key={ref.joinPath} className="flex items-center justify-between text-sm">
                <div className="font-mono">{ref.joinPath}</div>
                <div className="text-xs text-green-700">
                  {ref.includes === '*' ? 'All members' : `${Array.isArray(ref.includes) ? ref.includes.length : 0} members`}
                  {ref.prefix && ', Prefixed'}
                  {ref.alias && `, Alias: ${ref.alias}`}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};

export default JoinPathSelector;
