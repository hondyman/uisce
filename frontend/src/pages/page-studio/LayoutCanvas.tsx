import React from 'react';
import { 
    Box, 
    Paper, 
    Typography, 
    IconButton, 
    Chip, 
    TextField, 
    Grid, 
    Switch, 
    FormControlLabel, 
    MenuItem, 
    Select, 
    FormControl, 
    InputLabel 
} from '@mui/material';
import { 
    Delete as DeleteIcon, 
    TableChart as TableIcon, 
    Description as FormIcon, 
    DynamicForm as DynamicFormIcon, 
    Add as AddIcon,
    ArrowUpward as MoveUpIcon,
    ArrowDownward as MoveDownIcon,
    DragIndicator as DragIcon
} from '@mui/icons-material';
import { CorePageDefinition, LayoutNode, ComponentDefinition } from '../../types/pageStudio';
import { getTypeIconConfig, BOField } from '../../components/reporting/BOFieldsPalette';
import { CoreIcon, CustomIcon } from '../../components/common/CoreCustomIcons';

interface LayoutCanvasProps {
    draft: CorePageDefinition;
    setDraft: (d: CorePageDefinition) => void;
    selectedId: string | null;
    onSelect: (id: string | null) => void;
}

const LayoutCanvas: React.FC<LayoutCanvasProps> = ({ draft, setDraft, selectedId, onSelect }) => {
    const handleDrop = (e: React.DragEvent, parentId: string) => {
        e.preventDefault();
        e.stopPropagation();

        const componentType = e.dataTransfer.getData('componentType');

        let boFieldData: any = null;
        let boFieldsBatch: any = null;
        try {
            const raw = e.dataTransfer.getData('application/json');
            if (raw) {
                const parsed = JSON.parse(raw);
                if (parsed.type === 'bofield' || parsed.type === 'BO_FIELD') {
                    boFieldData = parsed.field || parsed;
                } else if (parsed.type === 'bofield_batch') {
                    boFieldsBatch = parsed.fields;
                }
            }
            if (!boFieldData && !boFieldsBatch) {
                const rawBundle = e.dataTransfer.getData('bo-field-bundle');
                if (rawBundle) {
                    boFieldsBatch = JSON.parse(rawBundle);
                }
            }
        } catch {}

        const newDraft = { 
            ...draft, 
            layout: { ...draft.layout, nodes: { ...draft.layout.nodes } }, 
            components: { ...draft.components } 
        };

        if (boFieldData || boFieldsBatch) {
            const fieldsToAdd: BOField[] = boFieldsBatch || (boFieldData ? [boFieldData] : []);
            // Check if dropping directly onto an existing component
            if (newDraft.components[parentId]) {
                const targetComp = newDraft.components[parentId];
                const existingFields: BOField[] = targetComp.props?.fields || [];
                const updatedFields = [...existingFields];
                fieldsToAdd.forEach((f) => {
                    if (!updatedFields.some(ef => (ef.technicalName || ef.name) === (f.technicalName || f.name))) {
                        updatedFields.push(f);
                    }
                });
                newDraft.components[parentId] = {
                    ...targetComp,
                    props: {
                        ...targetComp.props,
                        fields: updatedFields,
                    }
                };
                setDraft(newDraft);
                onSelect(parentId);
                return;
            }

            // Otherwise create new Form component in target layout node
            const compId = `form_${Math.random().toString(36).substr(2, 5)}`;
            newDraft.components[compId] = {
                id: compId,
                type: 'BO_FORM',
                props: {
                    fields: fieldsToAdd,
                    title: fieldsToAdd.length === 1 ? fieldsToAdd[0].label || fieldsToAdd[0].name : 'Form Section',
                }
            };
            const targetNode = newDraft.layout.nodes[parentId];
            if (targetNode) {
                targetNode.children = [...(targetNode.children || []), compId];
            } else {
                newDraft.layout.nodes[parentId] = { id: parentId, type: 'Row', children: [compId] };
            }
            setDraft(newDraft);
            onSelect(compId);
            return;
        }

        if (componentType) {
            const newId = `${componentType.toLowerCase()}_${Math.random().toString(36).substr(2, 5)}`;
            if (['Row', 'Column'].includes(componentType)) {
                newDraft.layout.nodes[newId] = { id: newId, type: componentType as any, children: [] };
            } else {
                newDraft.components[newId] = { id: newId, type: componentType, props: { title: `${componentType} Widget` } };
            }
            const parent = newDraft.layout.nodes[parentId];
            if (parent) {
                parent.children = [...(parent.children || []), newId];
            }
            setDraft(newDraft);
            onSelect(newId);
        }
    };

    const handleMoveField = (compId: string, fieldIndex: number, direction: 'up' | 'down') => {
        const comp = draft.components[compId];
        if (!comp || !comp.props?.fields) return;
        const fields: BOField[] = [...comp.props.fields];
        const targetIndex = direction === 'up' ? fieldIndex - 1 : fieldIndex + 1;
        if (targetIndex < 0 || targetIndex >= fields.length) return;

        const temp = fields[fieldIndex];
        fields[fieldIndex] = fields[targetIndex];
        fields[targetIndex] = temp;

        const newDraft = {
            ...draft,
            components: {
                ...draft.components,
                [compId]: {
                    ...comp,
                    props: {
                        ...comp.props,
                        fields,
                    }
                }
            }
        };
        setDraft(newDraft);
    };

    const handleDeleteField = (compId: string, fieldIndex: number) => {
        const comp = draft.components[compId];
        if (!comp || !comp.props?.fields) return;
        const fields: BOField[] = comp.props.fields.filter((_: any, idx: number) => idx !== fieldIndex);

        const newDraft = {
            ...draft,
            components: {
                ...draft.components,
                [compId]: {
                    ...comp,
                    props: {
                        ...comp.props,
                        fields,
                    }
                }
            }
        };
        setDraft(newDraft);
    };

    const handleDelete = (id: string) => {
        const newDraft = { 
            ...draft,
            layout: { ...draft.layout, nodes: { ...draft.layout.nodes } },
            components: { ...draft.components }
        };
        delete newDraft.components[id];
        delete newDraft.layout.nodes[id];
        
        Object.values(newDraft.layout.nodes).forEach(node => {
            if (node.children) {
                node.children = node.children.filter(cid => cid !== id);
            }
        });

        setDraft(newDraft);
        onSelect(null);
    };

    const renderFieldControl = (f: BOField) => {
        const rawType = (f.dataType || f.type || 'string').toLowerCase();
        
        // Boolean / Switch control
        if (['bool', 'boolean', 'flag'].some(k => rawType.includes(k))) {
            return (
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', height: 40, px: 1, bgcolor: '#0B132B', borderRadius: 1, border: '1px solid #1E293B' }}>
                    <Typography variant="caption" sx={{ color: '#94A3B8' }}>{f.label || f.name}</Typography>
                    <Switch size="small" defaultChecked color="primary" disabled />
                </Box>
            );
        }

        // Date / Time control
        if (['date', 'time', 'timestamp', 'datetime'].some(k => rawType.includes(k))) {
            return (
                <TextField
                    type="date"
                    size="small"
                    fullWidth
                    disabled
                    defaultValue="2026-08-26"
                    sx={{
                        bgcolor: '#0B132B',
                        borderRadius: 1,
                        '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.75, color: '#94A3B8' }
                    }}
                />
            );
        }

        // Numeric / Currency control
        if (['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some(k => rawType.includes(k))) {
            return (
                <TextField
                    type="number"
                    size="small"
                    fullWidth
                    disabled
                    placeholder="0.00"
                    sx={{
                        bgcolor: '#0B132B',
                        borderRadius: 1,
                        '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.75, color: '#94A3B8' }
                    }}
                />
            );
        }

        // Select / Lookup / Code list control
        if (['code', 'status', 'type', 'category', 'lookup'].some(k => (f.technicalName || f.name).toLowerCase().includes(k))) {
            return (
                <FormControl fullWidth size="small" disabled sx={{ bgcolor: '#0B132B', borderRadius: 1 }}>
                    <Select value="option1" sx={{ height: 36, fontSize: '0.75rem', color: '#94A3B8' }}>
                        <MenuItem value="option1">Selected Value</MenuItem>
                    </Select>
                </FormControl>
            );
        }

        // Default Text input
        return (
            <TextField 
                size="small" 
                fullWidth 
                disabled 
                placeholder={`[${f.technicalName || f.name}]`}
                sx={{ 
                    bgcolor: '#0B132B',
                    borderRadius: 1,
                    '& .MuiInputBase-input': { fontSize: '0.75rem', py: 0.75, color: '#94A3B8' } 
                }} 
            />
        );
    };

    const renderComponent = (comp: ComponentDefinition) => {
        const fields: BOField[] = comp.props?.fields || [];
        const boKey = comp.props?.boKey;
        const subtypeKey = comp.props?.subtypeKey;

        return (
            <Paper 
                key={comp.id}
                onClick={(e) => { e.stopPropagation(); onSelect(comp.id); }}
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => handleDrop(e, comp.id)}
                elevation={0}
                sx={{ 
                    p: 2, 
                    border: '1px solid',
                    borderColor: selectedId === comp.id ? '#0284C7' : '#334155',
                    borderRadius: 2,
                    flex: 1,
                    bgcolor: '#0F172A',
                    color: '#F8FAFC',
                    boxShadow: selectedId === comp.id ? '0 0 0 2px rgba(2, 132, 199, 0.4)' : 'none',
                    '&:hover': { borderColor: '#38BDF8' }
                }}
            >
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1.5, borderBottom: '1px solid #1E293B', pb: 1 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <DynamicFormIcon sx={{ fontSize: 18, color: '#38BDF8' }} />
                        <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#F8FAFC' }}>
                            {comp.props?.title || (boKey ? `${boKey.toUpperCase()} Form` : `${comp.id} (${comp.type})`)}
                        </Typography>
                        {subtypeKey && (
                            <Chip 
                                label={subtypeKey} 
                                size="small" 
                                sx={{ height: 20, fontSize: '0.65rem', bgcolor: 'rgba(2, 132, 199, 0.2)', color: '#38BDF8', fontWeight: 700 }} 
                            />
                        )}
                        <Chip 
                            label={`${fields.length} Fields`} 
                            size="small" 
                            sx={{ height: 18, fontSize: '0.6rem', bgcolor: '#1E293B', color: '#94A3B8' }} 
                        />
                    </Box>
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); handleDelete(comp.id); }} sx={{ color: '#94A3B8', '&:hover': { color: '#EF4444' } }}>
                        <DeleteIcon fontSize="small" />
                    </IconButton>
                </Box>

                {fields.length > 0 ? (
                    <Grid container spacing={2}>
                        {fields.map((f, idx) => {
                            const { Icon, color, label: typeLabel } = getTypeIconConfig(f.dataType || f.type);
                            return (
                                <Grid item xs={12} sm={6} md={4} key={`${f.name}-${idx}`}>
                                    <Box sx={{ p: 1.25, bgcolor: '#1E293B', borderRadius: 1.5, border: '1px solid #334155', position: 'relative' }}>
                                        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
                                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
                                                <Icon sx={{ fontSize: 14, color }} />
                                                <Typography variant="caption" fontWeight="700" sx={{ color: '#F8FAFC' }}>
                                                    {f.label || f.name}
                                                </Typography>
                                            </Box>
                                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.25 }}>
                                                {f.isCore ? (
                                                    <CoreIcon fontSize="small" sx={{ fontSize: 14 }} />
                                                ) : (
                                                    <CustomIcon fontSize="small" sx={{ fontSize: 14 }} />
                                                )}
                                                <IconButton 
                                                    size="small" 
                                                    disabled={idx === 0} 
                                                    onClick={(e) => { e.stopPropagation(); handleMoveField(comp.id, idx, 'up'); }}
                                                    sx={{ p: 0.25, color: '#94A3B8', '&:hover': { color: '#38BDF8' } }}
                                                    title="Move Up"
                                                >
                                                    <MoveUpIcon sx={{ fontSize: 12 }} />
                                                </IconButton>
                                                <IconButton 
                                                    size="small" 
                                                    disabled={idx === fields.length - 1} 
                                                    onClick={(e) => { e.stopPropagation(); handleMoveField(comp.id, idx, 'down'); }}
                                                    sx={{ p: 0.25, color: '#94A3B8', '&:hover': { color: '#38BDF8' } }}
                                                    title="Move Down"
                                                >
                                                    <MoveDownIcon sx={{ fontSize: 12 }} />
                                                </IconButton>
                                                <IconButton 
                                                    size="small" 
                                                    onClick={(e) => { e.stopPropagation(); handleDeleteField(comp.id, idx); }}
                                                    sx={{ p: 0.25, color: '#94A3B8', '&:hover': { color: '#EF4444' } }}
                                                    title="Remove Field"
                                                >
                                                    <DeleteIcon sx={{ fontSize: 12 }} />
                                                </IconButton>
                                            </Box>
                                        </Box>
                                        {renderFieldControl(f)}
                                    </Box>
                                </Grid>
                            );
                        })}
                    </Grid>
                ) : (
                    <Box 
                        sx={{ 
                            p: 3, 
                            textAlign: 'center', 
                            border: '1px dashed #334155', 
                            borderRadius: 1.5, 
                            bgcolor: 'rgba(30, 41, 59, 0.5)',
                            display: 'flex',
                            flexDirection: 'column',
                            alignItems: 'center',
                            gap: 1
                        }}
                    >
                        <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                            Drag & drop Business Object fields from the left palette into this form
                        </Typography>
                    </Box>
                )}
            </Paper>
        );
    };

    const renderNode = (nodeId: string) => {
        if (!draft?.layout?.nodes || !nodeId) return null;
        const node = draft.layout.nodes[nodeId];
        if (!node) {
            const component = draft.components?.[nodeId];
            if (component) return renderComponent(component);
            return null;
        }

        return (
            <Box 
                key={nodeId}
                onClick={(e) => { e.stopPropagation(); onSelect(nodeId); }}
                sx={{ 
                    border: '1px dashed',
                    borderColor: selectedId === nodeId ? '#0284C7' : '#334155',
                    p: 1.5,
                    mb: 2,
                    borderRadius: 2,
                    bgcolor: selectedId === nodeId ? 'rgba(2, 132, 199, 0.08)' : 'transparent',
                    position: 'relative',
                    '&:hover': { borderColor: '#38BDF8' }
                }}
            >
                <Typography variant="caption" sx={{ position: 'absolute', top: -10, left: 10, bgcolor: '#0B1E36', px: 0.5, color: '#94A3B8', fontWeight: 600 }}>
                    {node.type} ({node.id})
                </Typography>
                <Box sx={{ display: 'flex', flexDirection: node.type === 'Row' ? 'row' : 'column', gap: 2 }}>
                    {(node.children || []).map(childId => renderNode(childId))}
                    {(!node.children || node.children.length === 0) && (
                        <Box 
                            onDragOver={(e) => e.preventDefault()}
                            onDrop={(e) => handleDrop(e, nodeId)}
                            sx={{ p: 4, textAlign: 'center', border: '1px dashed #334155', flex: 1, borderRadius: 2 }}
                        >
                            <Typography variant="caption" sx={{ color: '#94A3B8' }}>Drop components or BO fields here</Typography>
                        </Box>
                    )}
                </Box>
            </Box>
        );
    };

    return (
        <Box sx={{ minHeight: '100%', pb: 20 }}>
            {draft?.layout?.root ? renderNode(draft.layout.root) : (
                <Typography variant="caption" sx={{ color: '#94A3B8', p: 2 }}>
                    No layout defined.
                </Typography>
            )}
        </Box>
    );
};

export default LayoutCanvas;
