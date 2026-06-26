import React, { useEffect, useState } from 'react';
import { Box, Typography, CircularProgress, Chip, Stack, List, ListItem, ListItemText, Divider, Paper } from '@mui/material';
import { impactApi } from '../api/impactApi';
import { NodeType, ImpactSummary, ImpactNode } from '../types';

interface ImpactExplanationProps {
  nodeType: NodeType;
  nodeId: string;
  directionMode?: 'upstream' | 'downstream' | 'both';
}

export const ImpactExplanation: React.FC<ImpactExplanationProps> = ({ nodeType, nodeId, directionMode = 'both' }) => {
  const [summary, setSummary] = useState<ImpactSummary | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      try {
        const data = await impactApi.getExplanation(nodeType, nodeId);
        setSummary(data);
      } catch (error) {
        console.error("Failed to fetch impact explanation:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [nodeType, nodeId]);

  if (loading) return <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}><CircularProgress /></Box>;
  if (!summary) return <Typography>No impact data available.</Typography>;

  return (
    <Box sx={{ pb: 4 }}>
      <Typography variant="subtitle1" fontWeight="bold" gutterBottom>Overview</Typography>
      <Typography variant="body2" paragraph sx={{ color: 'text.secondary', lineHeight: 1.6 }}>
        {summary.explanation}
      </Typography>

      <Box sx={{ mb: 3 }}>
        <Typography variant="subtitle2" fontWeight="bold" gutterBottom>Impacted Entities:</Typography>
        <Stack direction="row" spacing={1} flexWrap="wrap" useFlexGap sx={{ mt: 1 }}>
          {Object.entries(summary.nodesByType).map(([type, count]) => (
            <Chip 
              key={type} 
              label={`${type}: ${count}`} 
              variant="filled" 
              size="small" 
              sx={{ 
                bgcolor: '#f1f5f9', 
                color: '#475569',
                fontSize: '10px',
                fontWeight: 'bold',
                height: '20px'
              }} 
            />
          ))}
        </Stack>
      </Box>
      
      {summary.recommendations && summary.recommendations.length > 0 && (
        <Box sx={{ mb: 3 }}>
             <Typography variant="subtitle2" fontWeight="bold" color="warning.dark" gutterBottom>Recommendations</Typography>
             <Paper 
               elevation={0} 
               sx={{ 
                 bgcolor: '#fffbeb', 
                 borderRadius: '8px', 
                 p: 1.5,
                 border: '1px solid #fef3c7'
               }}
             >
                 {summary.recommendations.map((rec: string, i: number) => (
                    <Typography key={i} variant="body2" sx={{ color: '#92400e', mb: 0.5, display: 'flex' }}>
                      <span style={{ marginRight: '8px' }}>•</span> {rec}
                    </Typography>
                 ))}
             </Paper>
        </Box>
      )}

      {Object.keys(summary.affectedArtifacts).length > 0 && (
         <Box>
            <Typography variant="subtitle1" fontWeight="bold" gutterBottom sx={{ mt: 2 }}>Breakdown</Typography>
            {Object.entries(summary.affectedArtifacts).map(([category, nodes]) => (
                <Box key={category} sx={{ mb: 2 }}>
                    <Typography variant="subtitle2" sx={{ color: 'text.secondary', fontWeight: 'bold', mb: 1, textTransform: 'uppercase', fontSize: '10px', letterSpacing: '0.05em' }}>
                      {category}
                    </Typography>
                    <List dense sx={{ p: 0 }}>
                        {(nodes as ImpactNode[]).map((node: ImpactNode) => (
                            <ListItem key={node.id} sx={{ px: 0, py: 0.5 }}>
                                <ListItemText 
                                  primary={node.label} 
                                  primaryTypographyProps={{ variant: 'body2', fontWeight: 'medium' }}
                                  secondary={node.type} 
                                  secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '9px' } }}
                                />
                            </ListItem>
                        ))}
                    </List>
                    <Divider sx={{ my: 1, opacity: 0.5 }} />
                </Box>
            ))}
         </Box>
      )}
    </Box>
  );
};
