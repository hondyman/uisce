/**
 * Example: Entity Details Page Integration with Semantic Layer
 * 
 * This file shows how to integrate the business entity semantic layer
 * into your existing EntityDetailsPage component.
 */

import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '../ui/tabs';
import { Alert, AlertDescription } from '../ui/alert';
import { Badge } from '../ui/badge';
import { useTenant } from '../contexts/TenantContext';
import { useBusinessEntitySemanticLayer } from '../hooks/useBusinessEntitySemanticLayer';
import SemanticAssetsTab from '../components/entity/SemanticAssetsTab';
import RelationshipSuggestionPanel from '../components/entity/RelationshipSuggestionPanel';
import RelatedObjectsNavigator from '../components/entity/RelatedObjectsNavigator';
import { devLog, devError } from '../utils/devLogger';

interface ExampleIntegration {
  // This would be your existing props
  entityKey?: string;
}

/**
 * Example Integration of Semantic Layer into EntityDetailsPage
 */
export const EntityDetailsPageWithSemanticLayer: React.FC<ExampleIntegration> = ({
  entityKey = 'employee',
}) => {
  const { tenant, datasource } = useTenant();
  const [activeTab, setActiveTab] = useState('semantic-assets');

  // Example entity data (you'd fetch this from your app state)
  const entity = {
    id: 'uuid-employee-entity',
    key: entityKey,
    name: 'Employee',
    businessName: 'Employee',
    description: 'Employee business entity',
    semantic_term_ids: ['uuid-term-employee', 'uuid-term-person'],
    source_tables: ['employees', 'department_members'],
  };

  // Initialize semantic layer hook
  const semanticLayer = useBusinessEntitySemanticLayer({
    tenantId: tenant?.id || '',
    datasourceId: datasource?.id || '',
    businessEntityId: entity.id,
    businessEntityName: entity.name,
    semanticTermIds: entity.semantic_term_ids || [],
    sourceTableNames: entity.source_tables || [],
  });

  // Handler: Generate core model with error handling
  const handleGenerateCoreModel = async () => {
    try {
      devLog('📊 Generating core model for', entity.name);
      const result = await semanticLayer.generateCoreModel();
      if (result) {
        devLog('✅ Core model generated successfully', result);
        // Optional: Show toast notification
        // toast.success(`Core model generated: ${result.node_name}`);
      } else {
        devLog('❌ Failed to generate core model');
        // Optional: Show error toast
        // toast.error('Failed to generate core model');
      }
    } catch (error) {
      devError('Error generating core model:', error);
    }
  };

  // Handler: Generate core view
  const handleGenerateCoreView = async () => {
    try {
      devLog('📊 Generating core view for', entity.name);
      const result = await semanticLayer.generateCoreView();
      if (result) {
        devLog('✅ Core view generated successfully', result);
      } else {
        devLog('❌ Failed to generate core view');
      }
    } catch (error) {
      devError('Error generating core view:', error);
    }
  };

  // Handler: Create custom model
  const handleCreateCustomModel = async (name: string) => {
    try {
      devLog('🔧 Creating custom model:', name);
      const result = await semanticLayer.createCustomModel(name);
      if (result) {
        devLog('✅ Custom model created:', result);
      } else {
        devLog('❌ Failed to create custom model');
      }
    } catch (error) {
      devError('Error creating custom model:', error);
    }
  };

  // Handler: Create custom view
  const handleCreateCustomView = async (name: string) => {
    try {
      devLog('🔧 Creating custom view:', name);
      const result = await semanticLayer.createCustomView(name);
      if (result) {
        devLog('✅ Custom view created:', result);
      } else {
        devLog('❌ Failed to create custom view');
      }
    } catch (error) {
      devError('Error creating custom view:', error);
    }
  };

  // Handler: Apply suggestion
  const handleApplyRelationshipSuggestion = async (suggestion: any) => {
    try {
      devLog('✅ Applying relationship suggestion');
      const result = await semanticLayer.applyRelationshipSuggestion(suggestion);
      devLog('✅ Relationship applied:', result);
    } catch (error) {
      devError('Error applying relationship:', error);
    }
  };

  // Handler: Traverse object graph
  const handleTraverseObjectGraph = async (dotPath: string) => {
    try {
      devLog('🌐 Traversing graph:', dotPath);
      const result = await semanticLayer.traverseObjectGraph(
        semanticLayer.semanticAssets.coreModel?.id || '',
        dotPath
      );
      devLog('✅ Graph traversal complete:', result);
    } catch (error) {
      devError('Error traversing graph:', error);
    }
  };

  // Handler: Model/View click (navigate to editor)
  const handleModelClick = (model: any) => {
    devLog('📊 Model selected:', model.node_name);
    // Navigate to model editor or show detail view
    // navigate(`/semantic-models/${model.id}`);
  };

  const handleViewClick = (view: any) => {
    devLog('📊 View selected:', view.node_name);
    // Navigate to view editor
    // navigate(`/semantic-views/${view.id}`);
  };

  // Render loading state
  if (semanticLayer.assetsLoading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="spinner mr-3" />
        <p>Loading semantic assets...</p>
      </div>
    );
  }

  return (
    <div className="entity-details-with-semantic-layer">
      {/* Header */}
      <div className="entity-header mb-6">
        <div className="flex items-center justify-between mb-2">
          <h1 className="text-2xl font-bold">{entity.name}</h1>
          <div className="flex gap-2">
            <Badge variant="outline">Entity</Badge>
            {semanticLayer.semanticAssets.coreModel && (
              <Badge variant="secondary">Has Core Model</Badge>
            )}
            {semanticLayer.semanticAssets.customModel && (
              <Badge variant="secondary">Has Custom Model</Badge>
            )}
          </div>
        </div>
        <p className="text-gray-600">{entity.description}</p>
      </div>

      {/* Semantic Layer Integration */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="semantic-assets">
            Semantic Assets
            {semanticLayer.semanticAssets.coreModel && (
              <span className="ml-2 text-xs bg-green-100 text-green-800 px-2 py-0.5 rounded">
                Ready
              </span>
            )}
          </TabsTrigger>

          <TabsTrigger value="suggestions">
            Suggestions
            {semanticLayer.relationshipSuggestions.length > 0 && (
              <span className="ml-2 text-xs bg-blue-100 text-blue-800 px-2 py-0.5 rounded">
                {semanticLayer.relationshipSuggestions.length}
              </span>
            )}
          </TabsTrigger>

          <TabsTrigger value="related-objects">
            Related Objects
            {(semanticLayer.relatedObjects.linksTo.length > 0 ||
              semanticLayer.relatedObjects.linksFrom.length > 0) && (
              <span className="ml-2 text-xs bg-purple-100 text-purple-800 px-2 py-0.5 rounded">
                {semanticLayer.relatedObjects.linksTo.length +
                  semanticLayer.relatedObjects.linksFrom.length}
              </span>
            )}
          </TabsTrigger>
        </TabsList>

        {/* Tab 1: Semantic Assets */}
        <TabsContent value="semantic-assets" className="space-y-4">
          {semanticLayer.modelError && (
            <Alert variant="destructive">
              <AlertDescription>{semanticLayer.modelError.message}</AlertDescription>
            </Alert>
          )}

          {semanticLayer.viewError && (
            <Alert variant="destructive">
              <AlertDescription>{semanticLayer.viewError.message}</AlertDescription>
            </Alert>
          )}

          <SemanticAssetsTab
            semanticAssets={semanticLayer.semanticAssets}
            isLoading={
              semanticLayer.modelGenerationLoading || semanticLayer.viewGenerationLoading
            }
            error={semanticLayer.modelError || semanticLayer.viewError}
            onGenerateCoreModel={handleGenerateCoreModel}
            onGenerateCoreView={handleGenerateCoreView}
            onCreateCustomModel={handleCreateCustomModel}
            onCreateCustomView={handleCreateCustomView}
            onModelClick={handleModelClick}
            onViewClick={handleViewClick}
            businessEntityName={entity.name}
          />
        </TabsContent>

        {/* Tab 2: Relationship Suggestions */}
        <TabsContent value="suggestions" className="space-y-4">
          <RelationshipSuggestionPanel
            suggestions={semanticLayer.relationshipSuggestions}
            isLoading={semanticLayer.suggestionsLoading}
            error={semanticLayer.suggestionsError}
            onApplySuggestion={handleApplyRelationshipSuggestion}
            entityName={entity.name}
          />
        </TabsContent>

        {/* Tab 3: Related Objects */}
        <TabsContent value="related-objects" className="space-y-4">
          <RelatedObjectsNavigator
            linksTo={semanticLayer.relatedObjects.linksTo}
            linksFrom={semanticLayer.relatedObjects.linksFrom}
            isLoading={semanticLayer.relatedObjectsLoading}
            error={null}
            businessEntityName={entity.name}
            onTraverse={handleTraverseObjectGraph}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
};

/**
 * Usage Example
 * 
 * In your existing EntityDetailsPage.tsx:
 * 
 * const EntityDetailsPage = () => {
 *   // ... existing code ...
 *   
 *   return (
 *     <div className="entity-details-page">
 *       <EntityDetailsPageWithSemanticLayer 
 *         entityKey={entityKey} 
 *       />
 *       
 *       (existing entity tabs and content omitted)
 *     </div>
 *   );
 * };
 */

export default EntityDetailsPageWithSemanticLayer;

/**
 * Configuration & Customization
 * 
 * You can customize behavior via environment variables or config:
 * 
 * // Show/hide suggestions tab
 * VITE_ENABLE_RELATIONSHIP_SUGGESTIONS=true
 * 
 * // Minimum confidence threshold for auto-accept
 * VITE_SUGGESTION_AUTO_ACCEPT_THRESHOLD=0.85
 * 
 * // Cache durations (milliseconds)
 * VITE_SEMANTIC_CACHE_DURATION=3600000
 * VITE_SUGGESTION_CACHE_DURATION=1800000
 */

/**
 * Styling Integration
 * 
 * The components use Ant Design / MUI styling. Ensure you have:
 * 
 * - @mui/material
 * - @mui/icons-material
 * - lucide-react
 * - Custom UI component library (tabs, cards, buttons, etc.)
 * 
 * CSS modules are co-located with components:
 * - SemanticAssetsTab.css
 * - RelationshipSuggestionPanel.css
 * - RelatedObjectsNavigator.css
 */

/**
 * Testing Integration Example
 * 
 * import { render, screen, waitFor } from '@testing-library/react';
 * import userEvent from '@testing-library/user-event';
 * 
 * describe('EntityDetailsPage with Semantic Layer', () => {
 *   it('should generate core model on button click', async () => {
 *     render(<EntityDetailsPageWithSemanticLayer entityKey="employee" />);
 *     
 *     const generateBtn = screen.getByText('Generate Core Model');
 *     await userEvent.click(generateBtn);
 *     
 *     await waitFor(() => {
 *       expect(screen.getByText(/Core model generated/)).toBeInTheDocument();
 *     });
 *   });
 *   
 *   it('should display relationship suggestions', async () => {
 *     render(<EntityDetailsPageWithSemanticLayer entityKey="employee" />);
 *     
 *     await userEvent.click(screen.getByText('Suggestions'));
 *     
 *     await waitFor(() => {
 *       expect(screen.getByText(/suggested relationships/i)).toBeInTheDocument();
 *     });
 *   });
 * });
 */
