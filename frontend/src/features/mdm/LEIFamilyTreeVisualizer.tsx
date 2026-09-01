import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert
} from '@mui/material';
import {
  AccountTree as TreeIcon,
  CheckCircle as ValidIcon,
  AutoAwesome as AiIcon
} from '@mui/icons-material';

interface HierarchyMember {
  nodeId: string;
  lei: string;
  entityName: string;
  relationshipType: string;
  depth: number;
}

export const LEIFamilyTreeVisualizer: React.FC<{ tenantId?: string }> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999'
}) => {
  const [hierarchy] = useState<HierarchyMember[]>([
    {
      nodeId: 'node-sub-01',
      lei: '5493001KJTIIGC8Y1R12',
      entityName: 'Apple Operations International Ltd',
      relationshipType: 'SELF',
      depth: 0
    },
    {
      nodeId: 'node-parent-01',
      lei: 'HWUPKR0MPOU8FGXBT394',
      entityName: 'Apple Inc. (Ultimate Parent)',
      relationshipType: 'ULTIMATE_PARENT_OF',
      depth: 1
    }
  ]);

  const [aiNotice] = useState<string | null>(
    'GraphRAG Linkage Suggestion: Newly onboarded subsidiary [Apple Sales UK] matched via pgvector cosine similarity (98.2%) to parent [HWUPKR0MPOU8FGXBT394].'
  );

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      {/* Header */}
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <TreeIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Autonomous LEI Hierarchy & Family Tree Mesh (GLEIF)
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-tier corporate ownership graph traversal & AI parent linkage suggestions
            </Typography>
          </Box>
        </Stack>
        <Chip icon={<AiIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />} label="GraphRAG Active" size="small" sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, border: '1px solid #1E293B' }} />
      </Box>

      {aiNotice && (
        <Alert severity="info" sx={{ mb: 3, bgcolor: '#082F49', color: '#38BDF8', border: '1px solid #0284C7' }}>
          {aiNotice}
        </Alert>
      )}

      {/* Hierarchy Table */}
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Entity Name</TableCell>
              <TableCell>LEI Identifier</TableCell>
              <TableCell align="center">Graph Depth</TableCell>
              <TableCell align="center">Relationship Type</TableCell>
              <TableCell align="center">Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {hierarchy.map(h => (
              <TableRow key={h.nodeId} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell>
                  <Typography variant="body2" sx={{ fontWeight: 700, color: '#38BDF8', fontSize: 12 }}>{h.entityName}</Typography>
                </TableCell>
                <TableCell sx={{ fontSize: 11, fontFamily: 'monospace', color: '#CBD5E1' }}>{h.lei}</TableCell>
                <TableCell align="center" sx={{ fontFamily: 'monospace', fontWeight: 700 }}>Level {h.depth}</TableCell>
                <TableCell align="center">
                  <Chip label={h.relationshipType} size="small" sx={{ bgcolor: '#1E293B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                </TableCell>
                <TableCell align="center">
                  <Chip icon={<ValidIcon sx={{ fontSize: 12, color: '#10B981 !important' }} />} label="Verified" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default LEIFamilyTreeVisualizer;
