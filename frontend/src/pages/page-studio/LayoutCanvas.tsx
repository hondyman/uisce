import React from 'react';
import { Box, Paper, Typography, IconButton, Grid } from '@mui/material';
import { Delete as DeleteIcon, DragIndicator as DragIcon } from '@mui/icons-material';
import { CorePageDefinition, LayoutNode, ComponentDefinition } from '../../types/pageStudio';

interface LayoutCanvasProps {
    draft: CorePageDefinition;
    setDraft: (d: CorePageDefinition) => void;
    selectedId: string | null;
    onSelect: (id: string | null) => void;
}

const LayoutCanvas: React.FC<LayoutCanvasProps> = ({ draft, setDraft, selectedId, onSelect }) => {
    const renderNode = (nodeId: string) => {
        const node = draft.layout.nodes[nodeId];
        if (!node) {
            // Check if it's a component
            const component = draft.components[nodeId];
            if (component) return renderComponent(component);
            return null;
        }

        return (
            <Box 
                key={nodeId}
                onClick={(e) => { e.stopPropagation(); onSelect(nodeId); }}
                sx={{ 
                    border: '1px dashed',
                    borderColor: selectedId === nodeId ? 'primary.main' : 'rgba(0,0,0,0.1)',
                    p: 1.5,
                    mb: 2,
                    borderRadius: 2,
                    bgcolor: selectedId === nodeId ? 'rgba(25, 118, 210, 0.04)' : 'transparent',
                    position: 'relative',
                    '&:hover': { borderColor: 'primary.light' }
                }}
            >
                <Typography variant="caption" sx={{ position: 'absolute', top: -10, left: 10, bgcolor: '#f1f5f9', px: 0.5, color: 'text.secondary' }}>
                    {node.type} ({node.id})
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: node.type === 'Row' ? 'row' : 'column', gap: 2 }}>
                    {(node.children || []).map(childId => renderNode(childId))}
                    {(!node.children || node.children.length === 0) && (
                        <Box 
                            onDragOver={(e) => e.preventDefault()}
                            onDrop={(e) => handleDrop(e, nodeId)}
                            sx={{ p: 4, textAlign: 'center', border: '1px dashed rgba(0,0,0,0.1)', flex: 1, borderRadius: 2 }}
                        >
                            <Typography variant="caption" color="textSecondary">Drop component here</Typography>
                        </Box>
                    )}
                </Box>
            </Box>
        );
    };

    const renderComponent = (comp: ComponentDefinition) => (
        <Paper 
            key={comp.id}
            onClick={(e) => { e.stopPropagation(); onSelect(comp.id); }}
            elevation={0}
            sx={{ 
                p: 2, 
                border: '1px solid',
                borderColor: selectedId === comp.id ? 'primary.main' : 'rgba(0,0,0,0.05)',
                borderRadius: 2,
                flex: 1,
                bgcolor: 'white',
                '&:hover': { boxShadow: '0 4px 12px rgba(0,0,0,0.05)' }
            }}
        >
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <Typography variant="subtitle2" fontWeight="bold" color="primary">{comp.id} ({comp.type})</Typography>
                <Box>
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleDelete(comp.id); }}>
                        <DeleteIcon fontSize="small" />
                    </IconButton>
                </Box>
            </Box>
            <Box sx={{ height: 60, bgcolor: 'rgba(0,0,0,0.02)', borderRadius: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <Typography variant="caption" color="textSecondary">Component Preview: {comp.type}</Typography>
            </Box>
        </Paper>
    );

    const handleDrop = (e: React.DragEvent, parentId: string) => {
        const type = e.dataTransfer.getData('componentType');
        if (!type) return;

        const newId = `${type.toLowerCase()}_${Math.random().toString(36).substr(2, 5)}`;
        const newDraft = { ...draft };

        if (['Row', 'Column'].includes(type)) {
            // New Layout Node
            newDraft.layout.nodes[newId] = { id: newId, type: type as any, children: [] };
        } else {
            // New Data Component
            newDraft.components[newId] = { id: newId, type, props: {} };
        }

        const parent = newDraft.layout.nodes[parentId];
        parent.children = [...(parent.children || []), newId];
        
        setDraft(newDraft);
        onSelect(newId);
    };

    const handleDelete = (id: string) => {
        // Logic to remove from layout and components
        const newDraft = { ...draft };
        delete newDraft.components[id];
        delete newDraft.layout.nodes[id];
        
        // Remove from any parent children lists
        Object.values(newDraft.layout.nodes).forEach(node => {
            if (node.children) {
                node.children = node.children.filter(cid => cid !== id);
            }
        });

        setDraft(newDraft);
        onSelect(null);
    };

    return (
        <Box sx={{ minHeight: '100%', pb: 20 }}>
            {renderNode(draft.layout.root)}
        </Box>
    );
};

export default LayoutCanvas;
