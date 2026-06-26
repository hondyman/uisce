import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  IconButton,
  Tooltip,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Visibility as VisibilityIcon,
  Business as BusinessIcon,
  Storage as StorageIcon,
  Link as LinkIcon,
  Category as CategoryIcon,
} from '@mui/icons-material';
import { auditApi } from '../../../api/auditApi';
import { format } from 'date-fns';

interface RecentChange {
  entity_type: string;
  entity_id: string;
  entity_name?: string;
  change_type: string;
  changed_by: string;
  system_from: string;
  version_count: number;
}

interface Props {
  dateFrom: string;
  dateTo: string;
  entityType: string;
  onEntitySelect: (entityType: string, entityId: string, entityName?: string) => void;
}

const RecentChangesTable: React.FC<Props> = ({ dateFrom, dateTo, entityType, onEntitySelect }) => {
  const [changes, setChanges] = useState<RecentChange[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadChanges();
  }, [dateFrom, dateTo, entityType]);

  const loadChanges = async () => {
    setLoading(true);
    setError(null);

    try {
      const filters: any = {
        limit: 100,
      };

      if (dateFrom) filters.from = new Date(dateFrom).toISOString();
      if (dateTo) filters.to = new Date(dateTo).toISOString();
      if (entityType && entityType !== 'all') filters.entityType = entityType;

      const response = await auditApi.getAllChanges(filters);
      setChanges(response.changes || []);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to load recent changes');
    } finally {
      setLoading(false);
    }
  };

  const getEntityIcon = (type: string) => {
    switch (type) {
      case 'tenant':
        return <BusinessIcon />;
      case 'instance':
        return <StorageIcon />;
      case 'connection':
        return <LinkIcon />;
      case 'product':
        return <CategoryIcon />;
      default:
        return <CategoryIcon />;
    }
  };

  const getChangeColor = (changeType: string) => {
    switch (changeType) {
      case 'INSERT':
        return 'success';
      case 'UPDATE':
        return 'info';
      case 'DELETE':
        return 'error';
      case 'RESTORE':
        return 'warning';
      default:
        return 'default';
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mb: 2 }}>
        {error}
      </Alert>
    );
  }

  if (changes.length === 0) {
    return (
      <Paper sx={{ p: 4, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          No changes found in the selected date range
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Try adjusting your date filters or entity type
        </Typography>
      </Paper>
    );
  }

  return (
    <TableContainer component={Paper}>
      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Entity Type</TableCell>
            <TableCell>Entity Name/ID</TableCell>
            <TableCell>Last Change</TableCell>
            <TableCell>Changed By</TableCell>
            <TableCell>When</TableCell>
            <TableCell>Versions</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {changes.map((change, index) => (
            <TableRow key={`${change.entity_type}-${change.entity_id}-${index}`} hover>
              <TableCell>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  {getEntityIcon(change.entity_type)}
                  <Typography variant="body2" sx={{ textTransform: 'capitalize' }}>
                    {change.entity_type}
                  </Typography>
                </Box>
              </TableCell>

              <TableCell>
                <Typography variant="body2" fontWeight={500}>
                  {change.entity_name || change.entity_id}
                </Typography>
                {change.entity_name && (
                  <Typography variant="caption" color="text.secondary">
                    {change.entity_id}
                  </Typography>
                )}
              </TableCell>

              <TableCell>
                <Chip
                  label={change.change_type}
                  color={getChangeColor(change.change_type) as any}
                  size="small"
                />
              </TableCell>

              <TableCell>
                <Typography variant="body2">{change.changed_by}</Typography>
              </TableCell>

              <TableCell>
                <Typography variant="body2">
                  {format(new Date(change.system_from), 'PPpp')}
                </Typography>
              </TableCell>

              <TableCell>
                <Chip label={`${change.version_count} versions`} size="small" variant="outlined" />
              </TableCell>

              <TableCell align="right">
                <Tooltip title="View version history">
                  <IconButton
                    size="small"
                    color="primary"
                    onClick={() =>
                      onEntitySelect(change.entity_type, change.entity_id, change.entity_name)
                    }
                  >
                    <VisibilityIcon />
                  </IconButton>
                </Tooltip>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
};

export default RecentChangesTable;
