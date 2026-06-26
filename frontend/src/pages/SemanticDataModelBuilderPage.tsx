
import React, { useState, useEffect, useMemo } from 'react';
import { Box, Typography, Button, Paper, CircularProgress } from '@mui/material';
import { useSearchParams } from 'react-router-dom';
import ModelCatalogSidebar from '../components/UnifiedSemanticBuilder/ModelCatalogSidebar';
import SemanticModelEditor from '../components/UnifiedSemanticBuilder/SemanticModelEditor';
// import SemanticModelOverview from '../components/UnifiedSemanticBuilder/SemanticModelOverview';
import { ModelCatalogNode } from '../types/model';
import { SemanticModel } from '../components/UnifiedSemanticBuilder/types';
import axios from 'axios';
import { toast } from 'sonner';
import { useTenant } from '../contexts/TenantContext';
import { devDebug } from '../utils/devLogger';

// Type definitions matching backend
interface Cube {
  id: string;
  name: string;
  is_system: boolean;
  source_cube_id?: string;
  dimensions: any[];
  measures: any[];
  description?: string;
  source_table: string;
}

const SemanticDataModelBuilderPage: React.FC = () => {
  const [cubes, setCubes] = useState<Cube[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedModel, setSelectedModel] = useState<ModelCatalogNode | null>(null);
  const [activeTab, setActiveTab] = useState<'core' | 'custom'>('core');
  
  // Fetch cubes
  const fetchCubes = async () => {
    try {
      setLoading(true);
      const response = await axios.get('/api/semantic/cubes', {
        headers: { 'X-Tenant-ID': 'default' } // TODO: Get specific tenant ID
      });
      setCubes(response.data);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch cubes:', err);
      setError('Failed to load semantic models');
      toast.error('Failed to load semantic models');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchCubes();
  }, []);

  // Transform cubes to ModelCatalogNodes
  const models: ModelCatalogNode[] = useMemo(() => {
    return cubes.map(cube => {
      const isCustom = !cube.is_system;
      const isCore = cube.is_system;
      
      // Check if this core model has a custom extension
      const customExtension = cubes.find(c => c.source_cube_id === cube.id);
      // Check if this custom model extends a core model
      const coreParent = cube.source_cube_id ? cubes.find(c => c.id === cube.source_cube_id) : null;

      return {
        id: cube.id,
        model_key: cube.name,
        display_name: cube.name, // Can be enhanced with a display name field if added
        description: cube.description,
        status: 'published', // Simplified for now
        version: 1,
        is_current: true,
        is_core: isCore,
        is_custom: isCustom,
        can_edit: isCustom,
        parent_model_key: coreParent?.name,
        core_model_exists: isCore, // If it's core, it exists. If custom, invalid property?
        custom_model_exists: !!customExtension, // Does a custom version exist?
        created_at: new Date().toISOString(),
        metadata: {
          generator: 'semantic-layer',
          table_count: 1,
          measure_count: cube.measures?.length || 0,
          dimension_count: cube.dimensions?.length || 0,
          can_create: isCore && !customExtension // Can create custom ONLY if one doesn't exist
        },
        resolved_config: {
          id: cube.id,
          name: cube.name,
          dimensions: cube.dimensions || [],
          measures: cube.measures || [],
          filters: [],
          joins: [],
          is_custom: isCustom,
          is_core: isCore,
          description: cube.description,
          source_table: cube.source_table,
          // Add other required SemanticModel properties
        } as SemanticModel // Type casting as quick fix, better to map fully
      };
    });
  }, [cubes]);

  const handleModelSelect = (model: ModelCatalogNode, targetTab: 'core' | 'custom') => {
    setSelectedModel(model);
    setActiveTab(targetTab);
  };

  const [semanticModel, setSemanticModel] = useState<SemanticModel | null>(null);

  useEffect(() => {
    if (selectedModel?.resolved_config) {
      setSemanticModel(selectedModel.resolved_config as SemanticModel);
    } else {
      setSemanticModel(null);
    }
  }, [selectedModel]);

  const updateSemanticElement = (type: string, id: string, updates: any) => {
    if (!semanticModel) return;
    const list = (semanticModel as any)[type] as any[] || [];
    const updatedList = list.map((item: any) => item.id === id ? { ...item, ...updates } : item);
    setSemanticModel({ ...semanticModel, [type]: updatedList });
  };

  const removeSemanticElement = (id: string) => {
    if (!semanticModel) return;
    // SemanticModelEditor passes only ID, so we must find the type
    const types = ['dimensions', 'measures', 'filters', 'joins', 'pre_aggregations'];
    let foundType = '';
    for (const t of types) {
       if ((semanticModel as any)[t]?.some((item: any) => item.id === id)) {
           foundType = t;
           break;
       }
    }
    if (foundType) {
        const list = (semanticModel as any)[foundType] as any[] || [];
        const updatedList = list.filter((item: any) => item.id !== id);
        setSemanticModel({ ...semanticModel, [foundType]: updatedList });
    }
  };

  const toggleElementEdit = (type: string, id: string) => {
      // Implement toggle logic if needed, or just no-op if inline editing isn't crucial
      // For now, no-op or simple log
      devDebug('Toggle edit', type, id);
  };

  const { tenant } = useTenant();
  const tenantId = tenant?.id || 'default';

  const onAdd = (type: 'dimension' | 'measure' | 'filter' | 'join' | 'extends', targetTable?: string | { id: string; qualified_path: string } | null) => {
      if (!semanticModel) return;
      
      const newId = `${type}_${Date.now()}`;
      const newItem = {
          id: newId,
          name: `new_${type}`,
          title: `New ${type.charAt(0).toUpperCase() + type.slice(1)}`,
          type: 'string', // Default
          is_custom: true, // It's a custom addition
          isEditing: true
      };

      // Handle specific fields based on type if needed
      if (type === 'extends') {
          // Extends is special, handled via onChangeExtends usually
          return;
      }

      const listKey = type + 's'; // dimensions, measures...
      // Handle pluralization exceptions if any (none for these)
      
      const list = (semanticModel as any)[listKey] as any[] || [];
      const updatedList = [...list, newItem];
      setSemanticModel({ ...semanticModel, [listKey]: updatedList });
      
      // Ideally scroll to new item or set selection
  };

  const handleCreateCustomModel = async (baseModelKey: string) => {
    // Find base model
    const baseModel = cubes.find(c => c.name === baseModelKey);
    if (!baseModel) return;

    try {
        const newName = `${baseModelKey}_custom`;
        const newCube = {
            name: newName,
            source_table: baseModel.source_table,
            description: `Custom extension of ${baseModelKey}`,
            is_system: false,
            source_cube_id: baseModel.id
        };

        await axios.post('/api/semantic/cubes', newCube, {
             headers: { 'X-Tenant-ID': tenantId }
        });
        
        toast.success(`Created custom model ${newName}`);
        fetchCubes();
    } catch (err) {
        console.error('Failed to create custom model:', err);
        toast.error('Failed to create custom model');
    }
  };
  
  const handleSaveModel = async () => {
      if (!semanticModel) return;
      try {
          const cubeUpdate = {
              name: semanticModel.name,
              dimensions: semanticModel.dimensions,
              measures: semanticModel.measures,
              is_system: false,
              source_table: (semanticModel as any).source_table || "", 
              description: semanticModel.description
          };
          
          await axios.put(`/api/semantic/cubes/${semanticModel.name}`, cubeUpdate, {
               headers: { 'X-Tenant-ID': tenantId }
          });
          toast.success('Model saved successfully');
          fetchCubes();
      } catch (err) {
          console.error("Failed to save model:", err);
          toast.error("Failed to save model");
      }
  };

  return (
    <Box sx={{ display: 'flex', height: '100vh', overflow: 'hidden' }}>
      <Box sx={{ width: '320px', borderRight: '1px solid #e0e0e0', display: 'flex', flexDirection: 'column' }}>
        <ModelCatalogSidebar
          models={models}
          selectedModel={selectedModel}
          setSelectedModel={setSelectedModel as any}
          activeTab={activeTab}
          onModelSelect={handleModelSelect}
          onCreateCustomModel={handleCreateCustomModel}
          loading={loading}
          error={error || undefined}
        />
      </Box>
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #e0e0e0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6">{selectedModel?.display_name || 'Select a Model'}</Typography>
            {semanticModel && activeTab === 'custom' && (
                <Button variant="contained" color="primary" onClick={handleSaveModel}>
                    Save Changes
                </Button>
            )}
        </Box>
        {semanticModel ? (
          <Box sx={{ flex: 1, overflow: 'auto', p: 2 }}>
             {/* Using SemanticModelOverview from existing components */}
             <div className="semantic-model-overview-container">
                {/* Note: SemanticDataModelBuilderPage needs to import SemanticModelOverview */}
             </div>
             {/* Fallback to simple JSON editor or basic list if Overview fails */}
             <SemanticModelEditor
                semanticModel={semanticModel} // Correct prop name
                modelName={semanticModel.name}
                showCode={null}
                onToggleCode={() => {}}
                generateJSON={() => JSON.stringify(semanticModel, null, 2)}
                generateYAML={() => ""}
                updateSemanticElement={updateSemanticElement}
                removeSemanticElement={removeSemanticElement}
                toggleElementEdit={toggleElementEdit}
                onChange={(m) => setSemanticModel(m)} // If Editor supports onChange
                editMode={activeTab === 'custom'}
                // Fill required props
                selectedColumn={null}
                coreOptions={[]}
                onChangeExtends={() => {}}
                onAdd={onAdd}
             />
          </Box>
        ) : (
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#666' }}>
            <Typography variant="h6">Select a model to view or edit</Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default SemanticDataModelBuilderPage;