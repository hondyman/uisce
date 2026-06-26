import React, { useState, useEffect } from 'react';
import { Box, Paper, Typography, CircularProgress, Grid, Divider } from '@mui/material';
import type { EffectivePageDefinition, ComponentDefinition, DataSourceDefinition } from '../../types/pageStudio';
import { ApiStudioApi } from '../../api/apiStudio';
import { PageStudioApi } from '../../api/pageStudio';
import { usePageMetrics } from '../../hooks/usePageMetrics';
import { usePageState } from '../../hooks/usePageState';
import { useModalManager } from '../../hooks/useModalManager';
import { runActions, ActionContext } from '../../runtime/actions';
import { evalExpression } from '../../runtime/evalExpression';
import { useNavigate } from 'react-router-dom';
import { Dialog, DialogTitle, DialogContent, IconButton } from '@mui/material';
import { Close as CloseIcon } from '@mui/icons-material';

// Runtime Component Library
const RuntimeComponents: Record<string, React.FC<any>> = {
    Table: ({ rows, columns, pageSize, onEvent }) => (
        <Paper variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
            <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                <thead>
                    <tr style={{ borderBottom: '1px solid #eee' }}>
                        {columns?.map((c: any) => <th key={c.field} style={{ textAlign: 'left', padding: '8px' }}>{c.label}</th>)}
                    </tr>
                </thead>
                <tbody>
                    {rows?.slice(0, pageSize || 10).map((row: any, i: number) => (
                        <tr 
                            key={i} 
                            style={{ borderBottom: '1px solid #f9f9f9', cursor: 'pointer' }}
                            onClick={() => onEvent?.('onRowClick', { row })}
                        >
                            {columns?.map((c: any) => <td key={c.field} style={{ padding: '8px' }}>{row[c.field]}</td>)}
                        </tr>
                    ))}
                </tbody>
            </table>
        </Paper>
    ),
    LineChart: ({ data, xField, yField }) => (
        <Paper variant="outlined" sx={{ p: 2, borderRadius: 2, height: 200, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <Typography variant="caption" color="textSecondary" sx={{ opacity: 0.7 }}>Chart Preview: {yField} vs {xField}</Typography>
        </Paper>
    ),
    KPIGroup: ({ data, label }) => (
        <Paper sx={{ p: 2, borderRadius: 2, textAlign: 'center', bgcolor: 'primary.light', color: 'primary.contrastText' }}>
            <Typography variant="h4" fontWeight="bold">42</Typography>
            <Typography variant="caption">{label || 'Total Measure'}</Typography>
        </Paper>
    )
};

interface RuntimeRendererProps {
    page: EffectivePageDefinition;
    tenantId: string;
}

const RuntimeRenderer: React.FC<RuntimeRendererProps> = ({ page, tenantId }) => {
    const navigate = useNavigate();
    const [pageData, setPageData] = useState<Record<string, any>>({});
    const [loading, setLoading] = useState(true);
    const { state: pageState, setState: setPageState } = usePageState();
    const { modals, openModal, closeModal } = useModalManager();
    const { registerApiCall, reportRenderComplete, reportError } = usePageMetrics(page.id, tenantId);

    const actionContext: ActionContext = {
        pageId: page.id,
        tenantId,
        state: pageState,
        setState: setPageState,
        openModal,
        closeModal,
        refreshComponent: (id) => loadData(), // Simplified
        executeMutation: async (sourceId, vars) => {
            console.log(`Executing mutation ${sourceId}`, vars);
            return { success: true };
        },
        navigate: (pageId, params) => {
            const query = new URLSearchParams(params).toString();
            navigate(`/app/${pageId}${query ? '?' + query : ''}`);
        }
    };

    useEffect(() => {
        loadData();
    }, [page]);

    const loadData = async () => {
        setLoading(true);
        try {
            // 1. Attempt Server-Side Bundle Fetch
            let bundleSuccess = false;
            try {
                // In a real runtime, we would pass route params here
                const params = {}; 
                const bundleData = await PageStudioApi.getPageBundle(page.slug, tenantId, page.env, params);
                
                if (bundleData && Object.keys(bundleData).length > 0) {
                    setPageData(bundleData);
                    reportRenderComplete();
                    bundleSuccess = true;
                }
            } catch (bundleErr) {
                console.warn("Page Data Bundle fetch failed, falling back to client-side resolution", bundleErr);
            }

            // 2. Fallback to Client-Side Resolution (Legacy / Partial)
            if (!bundleSuccess) {
                const newData: Record<string, any> = {};
                const promises = Object.entries(page.dataBindings.sources).map(async ([id, source]: [string, DataSourceDefinition]) => {
                    registerApiCall();
                    // Simulation: in reality, would call runtime endpoint with source config
                    try {
                        const data = await ApiStudioApi.previewEndpoint('/mock', 'GET', page.env, tenantId, source.args || {});
                        newData[id] = data;
                    } catch (e) {
                         console.error(`Failed to load source ${id}`, e);
                         newData[id] = []; // Default to empty on error
                    }
                });
                await Promise.all(promises);
                setPageData(newData);
                reportRenderComplete();
            }
        } catch (err) {
            reportError(err);
        } finally {
            setLoading(false);
        }
    };

    const renderNode = (nodeId: string) => {
        const node = page.layout.nodes[nodeId];
        if (!node) {
            const comp = page.components[nodeId];
            if (comp) return renderComponent(comp);
            return null;
        }

        return (
            <Box key={nodeId} sx={{ display: 'flex', flexDirection: node.type === 'Row' ? 'row' : 'column', gap: 2, flex: 1 }}>
                {node.children?.map((cid: string) => renderNode(cid))}
            </Box>
        );
    };

    const renderComponent = (comp: ComponentDefinition) => {
        // 1. Evaluate Visibility
        const isVisible = !comp.visibility || evalExpression(comp.visibility.expression, { state: pageState, data: pageData });
        if (!isVisible) return null;

        const Impl = RuntimeComponents[comp.type] || (() => <div>Unknown Component {comp.type}</div>);
        
        // 2. Resolve Dynamic Props
        let finalProps = { ...comp.props };
        (comp.dynamicProps || []).forEach(rule => {
            finalProps[rule.prop] = evalExpression(rule.expression, { state: pageState, data: pageData });
        });

        // 3. Resolve Data Bindings
        const boundData = page.dataBindings.bindings
            .filter((b: any) => b.componentId === comp.id)
            .reduce((acc: any, b: any) => ({ ...acc, [b.prop]: pageData[b.sourceId] }), {});

        // 4. Handle Events
        const onEvent = async (eventName: string, payload: any) => {
            const cfg = comp.events?.find(e => e.event === eventName);
            if (cfg) {
                await runActions(cfg.actions, actionContext, payload);
            }
        };

        return (
            <Box key={comp.id} sx={{ flex: 1 }}>
                <Impl {...finalProps} {...boundData} onEvent={onEvent} />
            </Box>
        );
    };

    const renderModals = () => {
        return Object.entries(page.components)
            .filter(([_, comp]) => comp.type === 'Modal')
            .map(([id, comp]) => {
                const modalState = modals[id];
                if (!modalState?.open) return null;

                const contentCompId = comp.props.contentComponentId;
                const contentComp = page.components[contentCompId];

                return (
                    <Dialog 
                        key={id} 
                        open={true} 
                        onClose={() => closeModal(id)}
                        fullWidth
                        maxWidth="md"
                    >
                        <DialogTitle sx={{ m: 0, p: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            {comp.props.title || 'Modal'}
                            <IconButton onClick={() => closeModal(id)} size="small">
                                <CloseIcon />
                            </IconButton>
                        </DialogTitle>
                        <DialogContent dividers>
                            {contentComp ? renderComponent(contentComp) : <Typography>No content specified</Typography>}
                        </DialogContent>
                    </Dialog>
                );
            });
    };

    if (loading) return (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 10 }}>
            <CircularProgress />
        </Box>
    );

    return (
        <Box sx={{ p: 2, height: '100%', overflowY: 'auto' }}>
            {renderNode(page.layout.root)}
            {renderModals()}
        </Box>
    );
};

export default RuntimeRenderer;
