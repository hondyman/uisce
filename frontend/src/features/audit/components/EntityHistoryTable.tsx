import React, { useState } from 'react';
import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Checkbox,
  Chip,
  IconButton,
  Tooltip,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
} from '@mui/material';
import {
  Restore as RestoreIcon,
  CheckCircle as CheckCircleIcon,
  Visibility as VisibilityIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { EntitySnapshot } from '../../../api/auditApi';
import { format, parseISO } from 'date-fns';

interface Props {
  history: EntitySnapshot[];
  onVersionSelect: (version: EntitySnapshot) => void;
  onRestoreClick: (version: EntitySnapshot) => void;
  selectedVersions: EntitySnapshot[];
}

const EntityHistoryTable: React.FC<Props> = ({
  history,
  onVersionSelect,
  onRestoreClick,
  selectedVersions,
}) => {
  const [viewDataDialog, setViewDataDialog] = useState<EntitySnapshot | null>(null);

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

  const isSelected = (version: EntitySnapshot) =>
    selectedVersions.some((v) => v.version_id === version.version_id);

  return (
    <>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell padding="checkbox">Select</TableCell>
              <TableCell>Change Type</TableCell>
              <TableCell>System Time</TableCell>
              <TableCell>Valid Time</TableCell>
              <TableCell>Changed By</TableCell>
              <TableCell>Reason</TableCell>
              <TableCell>Status</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {history.map((version) => (
              <TableRow
                key={version.version_id}
                hover
                selected={isSelected(version)}
                sx={{
                  bgcolor: version.is_current ? 'action.hover' : 'inherit',
                }}
              >
                <TableCell padding="checkbox">
                  <Checkbox
                    checked={isSelected(version)}
                    onChange={() => onVersionSelect(version)}
                    disabled={selectedVersions.length >= 2 && !isSelected(version)}
                  />
                </TableCell>

                <TableCell>
                  <Chip
                    label={version.change_type}
                    color={getChangeColor(version.change_type) as any}
                    size="small"
                  />
                </TableCell>

                <TableCell>
                  <Typography variant="body2">
                    {format(parseISO(version.system_from), 'PPpp')}
                  </Typography>
                  {version.system_to && (
                    <Typography variant="caption" color="text.secondary">
                      → {format(parseISO(version.system_to), 'PPpp')}
                    </Typography>
                  )}
                </TableCell>

                <TableCell>
                  <Typography variant="body2">
                    {format(parseISO(version.valid_from), 'PPpp')}
                  </Typography>
                  {version.valid_to && (
                    <Typography variant="caption" color="text.secondary">
                      → {format(parseISO(version.valid_to), 'PPpp')}
                    </Typography>
                  )}
                </TableCell>

                <TableCell>
                  <Typography variant="body2">{version.changed_by}</Typography>
                </TableCell>

                <TableCell>
                  <Typography variant="body2" sx={{ maxWidth: 200 }} noWrap>
                    {version.change_reason || '-'}
                  </Typography>
                </TableCell>

                <TableCell>
                  {version.is_current && (
                    <Chip
                      label="Current"
                      color="success"
                      size="small"
                      icon={<CheckCircleIcon />}
                    />
                  )}
                  {version.is_deleted && (
                    <Chip label="Deleted" color="error" size="small" variant="outlined" />
                  )}
                </TableCell>

                <TableCell align="right">
                  <Tooltip title="View snapshot data">
                    <IconButton
                      size="small"
                      color="info"
                      onClick={() => setViewDataDialog(version)}
                    >
                      <VisibilityIcon />
                    </IconButton>
                  </Tooltip>
                  {!version.is_current && (
                    <Tooltip title="Restore to this version">
                      <IconButton
                        size="small"
                        color="primary"
                        onClick={() => onRestoreClick(version)}
                      >
                        <RestoreIcon />
                      </IconButton>
                    </Tooltip>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* JSON Snapshot Viewer Dialog */}
      <Dialog
        open={!!viewDataDialog}
        onClose={() => setViewDataDialog(null)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Box>
            <Typography variant="h6">Snapshot Data</Typography>
            {viewDataDialog && (
              <Typography variant="caption" color="text.secondary">
                Version: {viewDataDialog.version_id} • {format(parseISO(viewDataDialog.system_from), 'PPpp')}
              </Typography>
            )}
          </Box>
          <IconButton onClick={() => setViewDataDialog(null)} size="small">
            <CloseIcon />
          </IconButton>
        </DialogTitle>
        <DialogContent dividers>
          <Box
            component="pre"
            sx={{
              bgcolor: 'grey.100',
              p: 2,
              borderRadius: 1,
              overflow: 'auto',
              fontSize: '0.875rem',
              fontFamily: 'monospace',
              maxHeight: '60vh',
            }}
          >
            {viewDataDialog && JSON.stringify(viewDataDialog.entity_data, null, 2)}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewDataDialog(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </>
  );
};

export default EntityHistoryTable;
