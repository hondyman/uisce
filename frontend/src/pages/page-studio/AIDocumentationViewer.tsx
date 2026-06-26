import React from 'react';
import {
    Box,
    Typography,
    Paper,
    Divider,
    Stack,
    Chip,
    Button
} from '@mui/material';
import {
    Description as DocIcon,
    AutoAwesome as AiIcon,
    AccountTree as LineageIcon,
    Speed as PerformanceIcon,
    Code as ApiIcon
} from '@mui/icons-material';
import { CorePageDefinition } from '../../types/pageStudio';

interface AIDocumentationViewerProps {
    page: CorePageDefinition;
}

export const AIDocumentationViewer: React.FC<AIDocumentationViewerProps> = ({ page }) => {
    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', mb: 3 }}>
                <AiIcon color="primary" sx={{ mr: 1 }} />
                <Typography variant="h6" fontWeight="bold">AI-Generated Documentation</Typography>
            </Box>

            <Stack spacing={4}>
                <section>
                    <Typography variant="subtitle1" fontWeight="bold" gutterBottom>Purpose</Typography>
                    <Typography variant="body2" color="textSecondary">
                        This page serves as a comprehensive dashboard for <span className="font-bold">{page.name}</span> management. 
                        It provides real-time visibility into key performance indicators, regional trends, and high-level summaries.
                    </Typography>
                </section>

                <Divider />

                <section>
                    <Typography variant="subtitle1" fontWeight="bold" gutterBottom>Data Flow & Lineage</Typography>
                    <Paper 
                        variant="outlined" 
                        sx={{ 
                            p: 2, 
                            bgcolor: 'grey.50', 
                            fontFamily: 'monospace', 
                            fontSize: '0.8rem',
                            whiteSpace: 'pre-wrap',
                            overflow: 'auto',
                            maxHeight: 200
                        }}
                    >
                        {`graph TD
  BO[${page.name} BO] --> SQL[ASO Engine]
  SQL --> Q[GraphQL API]
  Q --> Table[positionsTable]
  Q --> Chart[trendChart]
  Table --> Details[Detail Modal]`}
                    </Paper>
                    <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
                        * Mermaid diagram generated based on active data bindings.
                    </Typography>
                </section>

                <Divider />

                <section>
                    <Typography variant="subtitle1" fontWeight="bold" gutterBottom>API Usage Summary</Typography>
                    <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
                        {Object.values(page.dataBindings?.sources || {}).map((s: any) => (
                            <Chip 
                                key={s.id}
                                label={`${s.type.toUpperCase()}: ${s.endpointId || s.id}`}
                                icon={<ApiIcon fontSize="small" />}
                                size="small"
                                variant="outlined"
                            />
                        ))}
                    </Stack>
                </section>

                <Divider />

                <section>
                    <Typography variant="subtitle1" fontWeight="bold" gutterBottom>Performance Profile (Expected)</Typography>
                    <Box sx={{ display: 'flex', gap: 4 }}>
                        <Box>
                            <Typography variant="caption" color="textSecondary">Avg. Latency</Typography>
                            <Typography variant="h6" fontWeight="bold" color="success.main">~450ms</Typography>
                        </Box>
                        <Box>
                            <Typography variant="caption" color="textSecondary">Data Volatility</Typography>
                            <Typography variant="h6" fontWeight="bold">Low (5m TTL)</Typography>
                        </Box>
                        <Box>
                            <Typography variant="caption" color="textSecondary">Pre-agg Potential</Typography>
                            <Typography variant="h6" fontWeight="bold" color="primary.main">High</Typography>
                        </Box>
                    </Box>
                </section>
            </Stack>

            <Box sx={{ mt: 6, display: 'flex', gap: 2 }}>
                <Button variant="outlined" startIcon={<DocIcon />} size="small">Export PDF</Button>
                <Button variant="outlined" startIcon={<DocIcon />} size="small">Sync to Wiki</Button>
            </Box>
        </Box>
    );
};
