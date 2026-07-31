import React, { useEffect, useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Chip
} from '@mui/material';
import { PageComponent } from '../../types/pageDesigner';
import { usePageContextStore } from '../../store/usePageContextStore';

interface DynamicBOGridProps {
  component: PageComponent;
}

export const DynamicBOGrid: React.FC<DynamicBOGridProps> = ({ component }) => {
  const contextMap = usePageContextStore((state) => state.contextMap);
  const setContextValue = usePageContextStore((state) => state.setContextValue);
  const activeSelectedId = contextMap['selected_account_id'];

  const [rows, setRows] = useState<any[]>([
    { id: 'ACC-99812', name: 'Acme Wealth Management', region: 'North America', status: 'ACTIVE', balance: '$4,200,000' },
    { id: 'ACC-99813', name: 'Global Asset Holdings', region: 'EMEA', status: 'PENDING', balance: '$2,800,000' },
    { id: 'ACC-99814', name: 'Apex Capital Management', region: 'APAC', status: 'ACTIVE', balance: '$1,950,000' },
  ]);

  const handleRowClick = (row: any) => {
    // Emit selection to context store
    if (component.interactions?.emits_context) {
      component.interactions.emits_context.forEach((emit) => {
        setContextValue(emit.target_context_key, row[emit.source_field] || row.id);
      });
    } else {
      setContextValue('selected_account_id', row.id);
      setContextValue('selected_account_status', row.status);
    }
  };

  return (
    <Paper sx={{ p: 2, bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155' }}>
      <Typography variant="h6" fontWeight="600" mb={2}>
        {component.title}
      </Typography>
      <Table size="small">
        <TableHead sx={{ bgcolor: '#0f172a' }}>
          <TableRow>
            <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Account ID</TableCell>
            <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Entity Name</TableCell>
            <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Region</TableCell>
            <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Status</TableCell>
            <TableCell sx={{ color: '#94a3b8', fontWeight: 600 }}>Balance</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {rows.map((row) => {
            const isSelected = activeSelectedId === row.id;
            return (
              <TableRow
                key={row.id}
                onClick={() => handleRowClick(row)}
                sx={{
                  cursor: 'pointer',
                  bgcolor: isSelected ? 'rgba(56, 189, 248, 0.15)' : 'transparent',
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.03)' },
                }}
              >
                <TableCell sx={{ color: '#38bdf8', fontWeight: 600 }}>{row.id}</TableCell>
                <TableCell sx={{ color: '#f8fafc' }}>{row.name}</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>{row.region}</TableCell>
                <TableCell>
                  <Chip
                    label={row.status}
                    size="small"
                    color={row.status === 'ACTIVE' ? 'success' : 'warning'}
                  />
                </TableCell>
                <TableCell sx={{ color: '#4ade80', fontWeight: 600 }}>{row.balance}</TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
    </Paper>
  );
};
