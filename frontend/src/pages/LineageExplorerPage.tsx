import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import { LineageGraph } from '../components/lineage/LineageGraph';
import ImpactPanel from '../ImpactPanel';
import { Box, Typography, Paper, Divider, Breadcrumbs, Link } from '@mui/material';
 
export const LineageExplorerPage: React.FC = () => {
    // Optionally allow exploring by ID url param
    const { nodeId } = useParams<{ nodeId: string }>();
    const [selectedNodeId, setSelectedNodeId] = useState<string>(nodeId || 'demo');
 
    return (
        <Box sx={{ p: 3, height: '100vh', display: 'flex', flexDirection: 'column' }}>
            <Breadcrumbs aria-label="breadcrumb">
                <Link underline="hover" color="inherit" href="/">
                    Semlayer
                </Link>
                <Link underline="hover" color="inherit" href="/lineage">
                    Lineage
                </Link>
                <Typography color="text.primary">Explorer</Typography>
            </Breadcrumbs>
 
            <Box sx={{ mt: 2, mb: 2 }}>
                <Typography variant="h4" component="h1" gutterBottom>
                    Semantic Lineage
                </Typography>
                <Typography variant="body1" color="text.secondary">
                    Visualizing dependencies and impact across Business Objects, Tables, and Optimizations.
                </Typography>
            </Box>
 
            <Box sx={{ display: 'flex', flexGrow: 1, gap: 2 }}>
                {/* Main Graph Area */}
                <Paper sx={{ flex: 1, p: 2, display: 'flex', flexDirection: 'column' }}>
                    <Typography variant="h6" gutterBottom>
                        Graph View
                    </Typography>
                    <Divider sx={{ mb: 2 }} />
                    <Box sx={{ flexGrow: 1, backgroundColor: '#fafafa', border: '1px solid #eee' }}>
                         {/* Graph Component */}
                         <LineageGraph 
                            nodeId={selectedNodeId} 
                            depth={3}
                            onNodeClick={(id) => setSelectedNodeId(id)}
                         />
                    </Box>
                </Paper>
 
                {/* Side Panel for Details */}
                <Paper sx={{ width: '350px', p: 2 }}>
                    <Typography variant="h6" gutterBottom>
                        Node Details
                    </Typography>
                    <Divider sx={{ mb: 2 }} />
                    {selectedNodeId ? (
                         <Box>
                            <Typography variant="subtitle1">Selected: {selectedNodeId}</Typography>
                            <Typography variant="body2" sx={{ mt: 1 }}>
                                Select a node to see metadata, entitlements, and ASO optimization status.
                            </Typography>
                            {/* Impact Analysis Panel */}
                            <Box sx={{ mt: 3 }}>
                                <ImpactPanel assetId={selectedNodeId} />
                            </Box>
                         </Box>
                    ) : (
                        <Typography variant="body2" color="text.secondary">
                            Select a node in the graph to view details.
                        </Typography>
                    )}
                </Paper>
            </Box>
        </Box>
    );
};

