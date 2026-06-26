/**
 * SemanticAssetsTab Component
 * 
 * Displays core and custom semantic models/views for a business entity
 * in the entity details page.
 */

import React, { useState } from 'react';
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '../ui/tabs';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '../ui/card';
import { Button } from '../ui/button';
import { CircularProgress } from '@mui/material';
import { Alert, AlertDescription } from '../ui/alert';
import { Badge } from '../ui/badge';
import { AlertCircle, Plus } from 'lucide-react';
import type { CoreSemanticAssets } from '../../services/businessEntitySemanticService';
import type { HierarchyNode, Field } from '../../types/entity-schema';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import { Box, Stack, Typography, Divider } from '@mui/material';
import { CalculationEditorDrawer } from '../BusinessObjectManager/CalculationEditorDrawer';
import './SemanticAssetsTab.css';

interface SemanticAssetsTabProps {
  semanticAssets: CoreSemanticAssets;
  isLoading: boolean;
  error: Error | null;
  onGenerateCoreModel: () => Promise<any>;
  onGenerateCoreView: () => Promise<any>;
  onCreateCustomModel: (name: string) => Promise<any>;
  onCreateCustomView: (name: string) => Promise<any>;
  businessEntityName: string;
  selectedNodeType?: 'root' | 'subtype' | 'field' | 'group';
  selectedNodeName?: string;
  hierarchyNodes?: HierarchyNode[];
  boId?: string;
}

const SemanticAssetsTab: React.FC<SemanticAssetsTabProps> = ({
  semanticAssets,
  isLoading,
  error,
  onGenerateCoreModel,
  onGenerateCoreView,
  onCreateCustomModel,
  onCreateCustomView,
  businessEntityName,
  selectedNodeType,
  selectedNodeName,
  hierarchyNodes = [],
}) => {
  const [generatingCoreModel, setGeneratingCoreModel] = useState(false);
  const [generatingCoreView, setGeneratingCoreView] = useState(false);
  const [customModelName, setCustomModelName] = useState('');
  const [customViewName, setCustomViewName] = useState('');
  const [creatingCustomModel, setCreatingCustomModel] = useState(false);
  const [creatingCustomView, setCreatingCustomView] = useState(false);
  const [calculationEditorOpen, setCalculationEditorOpen] = useState(false);

  const handleGenerateCoreModel = async () => {
    setGeneratingCoreModel(true);
    try {
      await onGenerateCoreModel();
    } finally {
      setGeneratingCoreModel(false);
    }
  };

  const handleGenerateCoreView = async () => {
    setGeneratingCoreView(true);
    try {
      await onGenerateCoreView();
    } finally {
      setGeneratingCoreView(false);
    }
  };

  const handleCreateCustomModel = async () => {
    if (!customModelName.trim()) return;
    setCreatingCustomModel(true);
    try {
      await onCreateCustomModel(customModelName);
      setCustomModelName('');
    } finally {
      setCreatingCustomModel(false);
    }
  };

  const handleCreateCustomView = async () => {
    if (!customViewName.trim()) return;
    setCreatingCustomView(true);
    try {
      await onCreateCustomView(customViewName);
      setCustomViewName('');
    } finally {
      setCreatingCustomView(false);
    }
  };

  if (isLoading) {
    return (
      <div className="semantic-assets-tab-loading">
        <CircularProgress />
        <p>Loading semantic assets...</p>
      </div>
    );
  }

  return (
    <div className="semantic-assets-tab">
      {error && (
        <Alert variant="destructive" className="mb-4">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error.message}</AlertDescription>
        </Alert>
      )}

      <Stack direction="row" spacing={3} sx={{ height: '100%', minHeight: '600px' }}>
        {/* Right Side: Assets Management */}
        <Box sx={{ flexGrow: 1 }}>
          <Box sx={{ mb: 3, pb: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
            <Typography variant="subtitle2" sx={{ color: 'text.secondary', fontWeight: 500 }}>
              Semantic Models & View for: <strong>{businessEntityName}</strong>
              {selectedNodeType === 'subtype' && (
                <Badge className="ml-1">Subtype</Badge>
              )}
            </Typography>
          </Box>
        <Tabs defaultValue="models" className="w-full">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="models">Semantic Models</TabsTrigger>
          <TabsTrigger value="views">Semantic Views</TabsTrigger>
        </TabsList>

        {/* Models Tab */}
        <TabsContent value="models" className="space-y-4">
          {/* Core Model */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Core Model</CardTitle>
                <Badge variant="secondary">Foundation</Badge>
              </div>
              <CardDescription>
                Auto-generated from semantic terms for {businessEntityName}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {semanticAssets.coreModel ? (
                <div className="core-model-card">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm">{semanticAssets.coreModel.node_name}</h4>
                      <p className="text-xs text-gray-500 mt-1">
                        {semanticAssets.coreModel.description || 'No description'}
                      </p>
                      {semanticAssets.coreModel.properties?.source_tables && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {(semanticAssets.coreModel.properties.source_tables as string[]).map(
                            (table) => (
                              <Badge key={table} variant="outline" className="text-xs">
                                {table}
                              </Badge>
                            )
                          )}
                        </div>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled
                    >
                      →
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-sm text-gray-600 mb-3">
                    No core model exists yet. Generate one from semantic terms.
                  </p>
                  <Button
                    onClick={handleGenerateCoreModel}
                    disabled={generatingCoreModel}
                    size="sm"
                  >
                    {generatingCoreModel ? (
                      <>
                        <CircularProgress size={20} sx={{ mr: 1 }} />
                        Generating...
                      </>
                    ) : (
                      <>
                        <Plus className="h-4 w-4 mr-2" />
                        Generate Core Model
                      </>
                    )}
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Custom Model */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Custom Model</CardTitle>
                <Badge variant="outline">Extension</Badge>
              </div>
              <CardDescription>Extends core model with custom dimensions/measures</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {semanticAssets.customModel ? (
                <div className="custom-model-card">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm">{semanticAssets.customModel.node_name}</h4>
                      <p className="text-xs text-gray-500 mt-1">
                        {semanticAssets.customModel.description || 'No description'}
                      </p>
                      <div className="mt-2">
                        <Badge variant="outline" className="text-xs">
                          extends {semanticAssets.coreModel?.node_name || 'core'}
                        </Badge>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled
                    >
                      →
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-sm text-gray-600 mb-3">
                    {semanticAssets.coreModel
                      ? 'Create a custom model extending the core model.'
                      : 'Generate a core model first.'}
                  </p>
                  {semanticAssets.coreModel && (
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="Custom model name"
                        value={customModelName}
                        onChange={(e) => setCustomModelName(e.target.value)}
                        className="flex-1 px-2 py-1 text-sm border rounded"
                        disabled={creatingCustomModel}
                      />
                      <Button
                        onClick={handleCreateCustomModel}
                        disabled={!customModelName.trim() || creatingCustomModel}
                        size="sm"
                      >
                        {creatingCustomModel ? (
                          <>
                            <CircularProgress size={20} sx={{ mr: 1 }} />
                            Creating...
                          </>
                        ) : (
                          <>
                            <Plus className="h-4 w-4 mr-2" />
                            Create
                          </>
                        )}
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Views Tab */}
        <TabsContent value="views" className="space-y-4">
          {/* Core View */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Core View</CardTitle>
                <Badge variant="secondary">Foundation</Badge>
              </div>
              <CardDescription>
                Auto-generated view backed by core model
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {semanticAssets.coreView ? (
                <div className="core-view-card">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm">{semanticAssets.coreView.node_name}</h4>
                      <p className="text-xs text-gray-500 mt-1">
                        {semanticAssets.coreView.description || 'No description'}
                      </p>
                      <div className="mt-2">
                        <Badge variant="outline" className="text-xs">
                          model: {semanticAssets.coreModel?.node_name || 'core'}
                        </Badge>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled
                    >
                      →
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-sm text-gray-600 mb-3">
                    {semanticAssets.coreModel
                      ? 'Generate a view from the core model.'
                      : 'Generate a core model first.'}
                  </p>
                  {semanticAssets.coreModel && (
                    <Button
                      onClick={handleGenerateCoreView}
                      disabled={generatingCoreView}
                      size="sm"
                    >
                      {generatingCoreView ? (
                        <>
                          <CircularProgress size={20} sx={{ mr: 1 }} />
                          Generating...
                        </>
                      ) : (
                        <>
                          <Plus className="h-4 w-4 mr-2" />
                          Generate Core View
                        </>
                      )}
                    </Button>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Custom View */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle className="text-base">Custom View</CardTitle>
                <Badge variant="outline">Extension</Badge>
              </div>
              <CardDescription>Extends core view with custom columns</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {semanticAssets.customView ? (
                <div className="custom-view-card">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <h4 className="font-semibold text-sm">{semanticAssets.customView.node_name}</h4>
                      <p className="text-xs text-gray-500 mt-1">
                        {semanticAssets.customView.description || 'No description'}
                      </p>
                      <div className="mt-2">
                        <Badge variant="outline" className="text-xs">
                          extends {semanticAssets.coreView?.node_name || 'core'}
                        </Badge>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled
                    >
                      →
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="empty-state">
                  <p className="text-sm text-gray-600 mb-3">
                    {semanticAssets.coreView
                      ? 'Create a custom view extending the core view.'
                      : 'Generate a core view first.'}
                  </p>
                  {semanticAssets.coreView && (
                    <div className="flex gap-2">
                      <input
                        type="text"
                        placeholder="Custom view name"
                        value={customViewName}
                        onChange={(e) => setCustomViewName(e.target.value)}
                        className="flex-1 px-2 py-1 text-sm border rounded"
                        disabled={creatingCustomView}
                      />
                      <Button
                        onClick={handleCreateCustomView}
                        disabled={!customViewName.trim() || creatingCustomView}
                        size="sm"
                      >
                        {creatingCustomView ? (
                          <>
                            <CircularProgress size={20} sx={{ mr: 1 }} />
                            Creating...
                          </>
                        ) : (
                          <>
                            <Plus className="h-4 w-4 mr-2" />
                            Create
                          </>
                        )}
                      </Button>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
        </Tabs>
        </Box>
      </Stack>

      <CalculationEditorDrawer
        open={calculationEditorOpen}
        onClose={() => setCalculationEditorOpen(false)}
        boId={semanticAssets.coreModel?.id || ''} // Passing core model ID (or BO ID if preferred) as context
      />
    </div>
  );
};

// Tree components (copied/adapted from Details page for now)
function HierarchyTree({
  nodes,
  expandedNodes,
  onNodeToggle,
}: {
  nodes: HierarchyNode[];
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
}) {
  return (
    <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0 }}>
      {nodes.map((node) => (
        <HierarchyTreeNode
          key={node.id}
          node={node}
          expandedNodes={expandedNodes}
          onNodeToggle={onNodeToggle}
        />
      ))}
    </Box>
  );
}

function HierarchyTreeNode({
  node,
  expandedNodes,
  onNodeToggle,
}: {
  node: HierarchyNode;
  expandedNodes: Set<string>;
  onNodeToggle: (nodeId: string) => void;
}) {
  const isExpanded = expandedNodes.has(node.id);
  const hasChildren = (node.children && node.children.length > 0) || (node.fields && node.fields.length > 0);

  return (
    <Box component="li" sx={{ listStyle: 'none', mb: 0.5 }}>
      <Stack
        direction="row"
        spacing={1}
        alignItems="center"
        onClick={() => hasChildren && onNodeToggle(node.id)}
        sx={{
          p: 0.75,
          borderRadius: 1,
          cursor: hasChildren ? 'pointer' : 'default',
          bgcolor: node.id === 'root' ? 'primary.light' : 'transparent',
          color: node.id === 'root' ? 'primary.main' : 'text.primary',
          fontWeight: node.id === 'root' ? 700 : 400,
          transition: 'all 0.2s ease',
          '&:hover': {
            bgcolor: node.id === 'root' ? 'primary.light' : 'action.hover',
          },
        }}
      >
        {hasChildren && (
          <ExpandMoreIcon
            sx={{
              fontSize: '1.2rem',
              transform: isExpanded ? 'rotate(0deg)' : 'rotate(-90deg)',
              transition: 'transform 0.2s ease',
            }}
          />
        )}
        {!hasChildren && <Box sx={{ width: 20 }} />}
        <Box
          component="span"
          className="material-symbols-outlined"
          sx={{ fontSize: '1.1rem', color: node.id === 'root' ? 'primary.main' : 'text.secondary' }}
        >
          {node.icon}
        </Box>
        <Typography variant="body2" sx={{ fontSize: '0.85rem' }}>{node.displayName || node.name}</Typography>
      </Stack>

      {hasChildren && isExpanded && (
        <Box component="ul" sx={{ listStyle: 'none', p: 0, m: 0, pl: 1, borderLeft: '1px solid', borderLeftColor: 'divider', ml: 1.5 }}>
          {/* Render children nodes (subtypes/groups) */}
          {node.children?.map((child) => (
            <HierarchyTreeNode
              key={child.id}
              node={child}
              expandedNodes={expandedNodes}
              onNodeToggle={onNodeToggle}
            />
          ))}
          {/* Render field nodes if any */}
          {node.fields?.map((field) => (
            <HierarchyTreeNode
              key={`field-${field.key}`}
              node={{
                id: `field-${field.key}`,
                name: field.name,
                displayName: field.businessName || field.name,
                icon: 'text_fields'
              }}
              expandedNodes={expandedNodes}
              onNodeToggle={onNodeToggle}
            />
          ))}
        </Box>
      )}
    </Box>
  );
}

export default SemanticAssetsTab;
