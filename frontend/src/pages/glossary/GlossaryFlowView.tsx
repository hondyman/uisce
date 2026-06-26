import React, { useMemo } from 'react';
import {
  Box,
  Paper,
  Typography,
  CircularProgress,
  Grid,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
} from '@mui/material';
import ReactFlow, {
  Node,
  Edge,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  Handle,
  Position,
} from 'reactflow';
import 'reactflow/dist/style.css';
import {
  useBusinessTerms,
  useSemanticTerms,
  useGlossaryEdges,
  useCreateTermEdge,
  BusinessTerm,
  SemanticTerm,
  CatalogEdge,
} from '../../api/glossary';
import './BusinessTermsTab.css';
import { useTranslation } from 'react-i18next';

type Focus = 'business' | 'semantic';

interface Props {
  focus?: Focus;
}

const BusinessTermNode: React.FC<any> = ({ data }) => (
  <div className="bt-node business-term">
    <div className="bt-node-label">{data.label}</div>
    {data.description && <div className="bt-node-desc">{data.description}</div>}
    <Handle type="target" position={Position.Top} />
    <Handle type="source" position={Position.Bottom} />
  </div>
);

const SemanticTermNode: React.FC<any> = ({ data }) => (
  <div className="bt-node semantic-term">
    <div className="bt-node-label">{data.label}</div>
    {data.description && <div className="bt-node-desc">{data.description}</div>}
    <Handle type="target" position={Position.Top} />
    <Handle type="source" position={Position.Bottom} />
  </div>
);

const nodeTypes = {
  businessTerm: BusinessTermNode,
  semanticTerm: SemanticTermNode,
};

export default function GlossaryFlowView({ focus = 'business' }: Props) {
  const { data: businessTerms, isLoading: businessLoading } = useBusinessTerms();
  const { data: semanticTerms, isLoading: semanticLoading } = useSemanticTerms();
  const { data: catalogEdges, isLoading: edgesLoading } = useGlossaryEdges();
  const _createEdgeMutation = useCreateTermEdge();

  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [flowEdges, setFlowEdges, onEdgesChange] = useEdgesState([]);
  const [selectedTerm, setSelectedTerm] = React.useState<BusinessTerm | SemanticTerm | null>(null);
  const [dialogOpen, setDialogOpen] = React.useState(false);

  useMemo(() => {
    if (!businessTerms || !semanticTerms) return;

    const newNodes: Node[] = [];
    const newFlowEdges: Edge[] = [];

    // Business nodes
    businessTerms.forEach((term, idx) => {
      newNodes.push({
        id: `bt-${term.id}`,
        data: { label: term.description || 'Business Term', description: term.catalog_type_name },
        position: { x: idx * 250, y: 0 },
        type: 'businessTerm',
      });
    });

    // Semantic nodes
    semanticTerms.forEach((term, idx) => {
      newNodes.push({
        id: `st-${term.id}`,
        data: { label: term.description || 'Semantic Term', description: term.catalog_type_name },
        position: { x: idx * 250, y: 250 },
        type: 'semanticTerm',
      });
    });

    if (catalogEdges) {
      catalogEdges.forEach((edge: CatalogEdge) => {
        const sourceIsBusinessTerm = businessTerms.some((bt) => bt.id === edge.subject_node_type_id);
        const targetIsSemanticTerm = semanticTerms.some((st) => st.id === edge.object_node_type_id);

        if (sourceIsBusinessTerm && targetIsSemanticTerm) {
          newFlowEdges.push({
            id: `e-${edge.id}`,
            source: `bt-${edge.subject_node_type_id}`,
            target: `st-${edge.object_node_type_id}`,
            label: edge.edge_type_name,
            animated: true,
            style: { stroke: '#2196f3', strokeWidth: 2 },
          } as Edge);
        }
      });
    }

    setNodes(newNodes);
    setFlowEdges(newFlowEdges);
  }, [businessTerms, semanticTerms, catalogEdges, setNodes, setFlowEdges]);

  const isLoading = businessLoading || semanticLoading || edgesLoading;

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  // Determine left/right lists based on focus
  const leftList = focus === 'business' ? businessTerms : semanticTerms;
  const rightList = focus === 'business' ? semanticTerms : businessTerms;
  const { t } = useTranslation();
  const leftLabel = focus === 'business' ? `${t('tab.business_terms', 'Business Terms')} (${businessTerms?.length || 0})` : `${t('tab.semantic_terms', 'Semantic Terms')} (${semanticTerms?.length || 0})`;
  const rightLabel = focus === 'business' ? `${t('tab.semantic_terms', 'Semantic Terms')} (${semanticTerms?.length || 0})` : `${t('tab.business_terms', 'Business Terms')} (${businessTerms?.length || 0})`;

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        {t('glossary.title', 'Business Glossary - Relationships')}
      </Typography>

      <Grid container spacing={2} sx={{ flex: 1, minHeight: 0 }}>
        <Grid item xs={12} md={3} sx={{ overflow: 'auto' }}>
          <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 'bold' }}>{leftLabel}</Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {leftList?.map((term: any) => (
              <Card key={term.id} sx={{ cursor: 'pointer', '&:hover': { boxShadow: 3 } }} onClick={() => { setSelectedTerm(term); setDialogOpen(true); }}>
                <CardContent sx={{ p: 1 }}>
                  <Typography variant="body2" sx={{ fontWeight: 'bold' }}>{term.description || t('node.untitled', 'Untitled')}</Typography>
                  <Typography variant="caption" color="textSecondary">{term.catalog_type_name}</Typography>
                  <Box sx={{ mt: 1 }}>
                    <Chip size="small" label={term.is_active ? 'Active' : 'Inactive'} color={term.is_active ? 'success' : 'default'} />
                  </Box>
                </CardContent>
              </Card>
            ))}
          </Box>
        </Grid>

        <Grid item xs={12} md={6} sx={{ height: '600px', minHeight: 0 }}>
          <Paper sx={{ height: '100%' }}>
            <ReactFlow nodes={nodes} edges={flowEdges} onNodesChange={onNodesChange} onEdgesChange={onEdgesChange} nodeTypes={nodeTypes} fitView>
              <Background />
              <Controls />
            </ReactFlow>
          </Paper>
        </Grid>

        <Grid item xs={12} md={3} sx={{ overflow: 'auto' }}>
          <Typography variant="subtitle2" sx={{ mb: 1, fontWeight: 'bold' }}>{rightLabel}</Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {rightList?.map((term: any) => (
              <Card key={term.id} sx={{ cursor: 'pointer', '&:hover': { boxShadow: 3 } }}>
                <CardContent sx={{ p: 1 }}>
                  <Typography variant="body2" sx={{ fontWeight: 'bold' }}>{term.description || 'Untitled'}</Typography>
                  <Typography variant="caption" color="textSecondary">{term.catalog_type_name}</Typography>
                  <Box sx={{ mt: 1 }}>
                    <Chip size="small" label={term.is_active ? 'Active' : 'Inactive'} color={term.is_active ? 'success' : 'default'} />
                  </Box>
                </CardContent>
              </Card>
            ))}
          </Box>
        </Grid>
      </Grid>

      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
          <DialogTitle>{t('term.details', 'Term Details')}</DialogTitle>
        <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
          {selectedTerm && (
            <>
              <TextField label={t('field.description', 'Description')} value={selectedTerm.description || ''} fullWidth size="small" disabled />
              <TextField label={t('field.type', 'Type')} value={selectedTerm.catalog_type_name} fullWidth size="small" disabled />
              <Box>
                <Typography variant="caption" sx={{ fontWeight: 'bold' }}>{t('field.status', 'Status')}</Typography>
                <Chip label={selectedTerm.is_active ? t('status.active', 'Active') : t('status.inactive', 'Inactive')} color={selectedTerm.is_active ? 'success' : 'default'} size="small" />
              </Box>
            </>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>{t('button.close', 'Close')}</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
