import React, { useState } from 'react';
import { Box, Typography, Paper, Tabs, Tab, Button, Divider } from '@mui/material';
import { 
    Save as SaveIcon, 
    PlayArrow as PreviewIcon, 
    Layers as FieldsIcon,
    Widgets as WidgetsIcon,
    Storage as DataIcon,
    Speed as PerformanceIcon,
    Description as DocIcon,
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';
import { PageStudioApi } from '../../api/pageStudio';
import ComponentPalette from './ComponentPalette';
import LayoutCanvas from './LayoutCanvas';
import PropertiesPanel from './PropertiesPanel';
import { PagePerformanceDashboard } from './PagePerformanceDashboard';
import { AIDocumentationViewer } from './AIDocumentationViewer';
import { SmartDataBindingStudio } from './SmartDataBindingStudio';
import BOFieldsPalette, { BOField, extractAllBOFields } from '../../components/reporting/BOFieldsPalette';

interface PageEditorProps {
    page: CorePageDefinition;
    selectedBO?: any;
    selectedSubtypeKey?: string | null;
    onSave: (page: CorePageDefinition) => void;
}

const PageEditor: React.FC<PageEditorProps> = ({ page, selectedBO, selectedSubtypeKey, onSave }) => {
    const [draft, setDraft] = useState<CorePageDefinition>(page);
    const [tab, setTab] = useState<'fields' | 'widgets' | 'data' | 'performance' | 'docs'>('fields');
    const [selectedId, setSelectedId] = useState<string | null>(null);

    const handleSave = async () => {
        try {
            const saved = await PageStudioApi.savePage(draft);
            onSave(saved);
        } catch (err) {
            console.error('Save failed', err);
        }
    };

    const handleAddFieldToCanvas = (field: BOField) => {
        const newDraft = { 
            ...draft, 
            layout: { ...draft.layout, nodes: { ...draft.layout.nodes } }, 
            components: { ...draft.components } 
        };

        // Find existing BO_FORM component or create one
        let formCompId = Object.keys(newDraft.components).find(k => newDraft.components[k].type === 'BO_FORM');
        
        if (!formCompId) {
            formCompId = `form_${Math.random().toString(36).substr(2, 5)}`;
            newDraft.components[formCompId] = {
                id: formCompId,
                type: 'BO_FORM',
                props: {
                    title: selectedBO?.displayName ? `${selectedBO.displayName} Form` : 'Entity Form',
                    fields: [field],
                }
            };
            const rootNode = newDraft.layout.nodes[newDraft.layout.root || 'root'];
            if (rootNode) {
                rootNode.children = [...(rootNode.children || []), formCompId];
            } else {
                newDraft.layout.root = 'root';
                newDraft.layout.nodes['root'] = { id: 'root', type: 'Row', children: [formCompId] };
            }
        } else {
            const existingFields: BOField[] = newDraft.components[formCompId].props?.fields || [];
            if (!existingFields.some(f => (f.technicalName || f.name) === (field.technicalName || field.name))) {
                newDraft.components[formCompId] = {
                    ...newDraft.components[formCompId],
                    props: {
                        ...newDraft.components[formCompId].props,
                        fields: [...existingFields, field],
                    }
                };
            }
        }

        setDraft(newDraft);
        setSelectedId(formCompId);
    };

    const handleAddAllAsTable = (fields: BOField[]) => {
        const newDraft = { 
            ...draft, 
            layout: { ...draft.layout, nodes: { ...draft.layout.nodes } }, 
            components: { ...draft.components } 
        };

        let formCompId = Object.keys(newDraft.components).find(k => newDraft.components[k].type === 'BO_FORM');
        if (!formCompId) {
            formCompId = `form_${Math.random().toString(36).substr(2, 5)}`;
            newDraft.components[formCompId] = {
                id: formCompId,
                type: 'BO_FORM',
                props: {
                    title: selectedBO?.displayName ? `${selectedBO.displayName} Form` : 'Entity Form',
                    fields: fields,
                }
            };
            const rootNode = newDraft.layout.nodes[newDraft.layout.root || 'root'];
            if (rootNode) {
                rootNode.children = [...(rootNode.children || []), formCompId];
            } else {
                newDraft.layout.root = 'root';
                newDraft.layout.nodes['root'] = { id: 'root', type: 'Row', children: [formCompId] };
            }
        } else {
            newDraft.components[formCompId] = {
                ...newDraft.components[formCompId],
                props: {
                    ...newDraft.components[formCompId].props,
                    fields: fields,
                }
            };
        }

        setDraft(newDraft);
        setSelectedId(formCompId);
    };

    return (
        <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
            {/* Header / Actions */}
            <Paper elevation={0} sx={{ p: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center', borderBottom: '1px solid #1E293B', bgcolor: '#071526', color: '#F8FAFC' }}>
                <Box>
                    <Typography variant="subtitle1" fontWeight="bold">{draft.name}</Typography>
                    <Typography variant="caption" sx={{ color: '#94A3B8' }}>{draft.slug} • v{draft.version}</Typography>
                </Box>
                <Box sx={{ display: 'flex', gap: 1 }}>
                    <Button variant="outlined" startIcon={<PreviewIcon />} size="small" sx={{ color: '#38BDF8', borderColor: '#0284C7', textTransform: 'none' }}>Preview</Button>
                    <Divider orientation="vertical" flexItem sx={{ mx: 1, borderColor: '#1E293B' }} />
                    <Button variant="contained" startIcon={<SaveIcon />} size="small" onClick={handleSave} sx={{ bgcolor: '#0284C7', color: '#FFF', textTransform: 'none', fontWeight: 700 }}>Save Changes</Button>
                </Box>
            </Paper>

            <Box sx={{ flex: 1, display: 'flex', overflow: 'hidden' }}>
                {/* Left Panel: BO Fields Palette & Widgets */}
                <Paper elevation={0} sx={{ width: 340, borderRight: '1px solid #1E293B', display: 'flex', flexDirection: 'column', bgcolor: '#071526' }}>
                    <Box sx={{ borderBottom: 1, borderColor: '#1E293B' }}>
                        <Tabs 
                            value={tab} 
                            onChange={(_, v) => setTab(v)} 
                            variant="fullWidth" 
                            sx={{ minHeight: 40, '& .MuiTab-root': { color: '#94A3B8', fontSize: '0.75rem', minHeight: 40, py: 0.5, textTransform: 'none', fontWeight: 700 }, '& .Mui-selected': { color: '#38BDF8' } }}
                        >
                            <Tab label="BO Fields" value="fields" icon={<FieldsIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
                            <Tab label="Widgets" value="widgets" icon={<WidgetsIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
                            <Tab label="Data" value="data" icon={<DataIcon sx={{ fontSize: 16 }} />} iconPosition="start" />
                        </Tabs>
                    </Box>
                    <Box sx={{ flex: 1, overflowY: 'auto', p: tab === 'fields' ? 0 : 1 }}>
                        {tab === 'fields' && (
                            <BOFieldsPalette 
                                selectedBO={selectedBO}
                                selectedSubtypeKey={selectedSubtypeKey}
                                onAddFieldToCanvas={handleAddFieldToCanvas}
                                onAddAllAsTable={handleAddAllAsTable}
                                mode="form"
                            />
                        )}
                        {tab === 'widgets' && (
                            <ComponentPalette />
                        )}
                        {tab === 'data' && (
                            <SmartDataBindingStudio 
                                pageId={draft.id} 
                                selectedComponentId={selectedId} 
                                onApplyBinding={(b) => console.log('Applied binding:', b)}
                            />
                        )}
                    </Box>
                </Paper>

                {/* Main: Layout Canvas */}
                <Box sx={{ flex: 1, p: 2, bgcolor: '#0B1E36', overflowY: 'auto' }}>
                    <LayoutCanvas
                        draft={draft}
                        setDraft={setDraft}
                        selectedId={selectedId}
                        onSelect={(id) => setSelectedId(id)}
                    />
                </Box>

                {/* Right: Properties */}
                <Paper elevation={0} sx={{ width: 300, borderLeft: '1px solid #1E293B', display: 'flex', flexDirection: 'column', bgcolor: '#071526' }}>
                    <PropertiesPanel
                        selectedId={selectedId}
                        draft={draft}
                        setDraft={setDraft}
                        tenantId={draft.tenantId || 'default'}
                    />
                </Paper>
            </Box>
        </Box>
    );
};

export default PageEditor;
