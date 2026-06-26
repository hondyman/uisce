import React from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Chip,
  IconButton,
  Tooltip,
  Card,
  CardContent,
  Checkbox,
  Button,
} from '@mui/material';
import {
  Restore as RestoreIcon,
  CheckCircle as CheckCircleIcon,
  Delete as DeleteIcon,
  Add as AddIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import { EntitySnapshot } from '../../../api/auditApi';
import { format, parseISO } from 'date-fns';

interface Props {
  history: EntitySnapshot[];
  onVersionSelect: (version: EntitySnapshot) => void;
  onRestoreClick: (version: EntitySnapshot) => void;
  selectedVersions: EntitySnapshot[];
}

const EntityTimelineView: React.FC<Props> = ({
  history,
  onVersionSelect,
  onRestoreClick,
  selectedVersions,
}) => {
  const getChangeIcon = (changeType: string) => {
    switch (changeType) {
      case 'INSERT':
        return <AddIcon />;
      case 'UPDATE':
        return <EditIcon />;
      case 'DELETE':
        return <DeleteIcon />;
      case 'RESTORE':
        return <RestoreIcon />;
      default:
        return <EditIcon />;
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

  const isSelected = (version: EntitySnapshot) =>
    selectedVersions.some((v) => v.version_id === version.version_id);

  return (
    <Box>
      {history.map((version, index) => (
        <Box key={version.version_id} sx={{ position: 'relative', mb: 3 }}>
          {/* Timeline Line */}
          {index < history.length - 1 && (
            <Box
              sx={{
                position: 'absolute',
                left: 24,
                top: 60,
                bottom: -24,
                width: 2,
                bgcolor: 'divider',
              }}
            />
          )}

          {/* Timeline Item */}
          <Card
            sx={{
              ml: 6,
              border: isSelected(version) ? 2 : 1,
              borderColor: isSelected(version) ? 'primary.main' : 'divider',
              bgcolor: version.is_current ? 'action.hover' : 'background.paper',
              transition: 'all 0.2s',
              '&:hover': {
                boxShadow: 4,
                transform: 'translateX(4px)',
              },
            }}
          >
            <CardContent>
              <Stack direction="row" spacing={2} alignItems="flex-start">
                {/* Timeline Icon */}
                <Box
                  sx={{
                    position: 'absolute',
                    left: -30,
                    width: 48,
                    height: 48,
                    borderRadius: '50%',
                    bgcolor: `${getChangeColor(version.change_type)}.main`,
                    color: 'white',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    boxShadow: 3,
                  }}
                >
                  {getChangeIcon(version.change_type)}
                </Box>

                {/* Selection Checkbox */}
                <Checkbox
                  checked={isSelected(version)}
                  onChange={() => onVersionSelect(version)}
                  disabled={selectedVersions.length >= 2 && !isSelected(version)}
                />

                {/* Content */}
                <Box sx={{ flex: 1 }}>
                  <Stack direction="row" spacing={1} alignItems="center" sx={{ mb: 1 }}>
                    <Chip
                      label={version.change_type}
                      color={getChangeColor(version.change_type) as any}
                      size="small"
                    />
                    {version.is_current && (
                      <Chip
                        label="CURRENT"
                        color="success"
                        size="small"
                        icon={<CheckCircleIcon />}
                      />
                    )}
                    {version.is_deleted && (
                      <Chip label="DELETED" color="error" size="small" variant="outlined" />
                    )}
                  </Stack>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    <strong>System Time:</strong>{' '}
                    {format(parseISO(version.system_from), 'PPpp')}
                    {version.system_to && ` → ${format(parseISO(version.system_to), 'PPpp')}`}
                  </Typography>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    <strong>Valid Time:</strong> {format(parseISO(version.valid_from), 'PPpp')}
                    {version.valid_to && ` → ${format(parseISO(version.valid_to), 'PPpp')}`}
                  </Typography>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                    <strong>Changed By:</strong> {version.changed_by}
                  </Typography>

                  {version.change_reason && (
                    <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                      <strong>Reason:</strong> {version.change_reason}
                    </Typography>
                  )}

                  <Typography variant="caption" color="text.disabled">
                    Version ID: {version.version_id}
                  </Typography>
                </Box>

                {/* Actions */}
                <Stack direction="row" spacing={1}>
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
                </Stack>
              </Stack>
            </CardContent>
          </Card>
        </Box>
      ))}
    </Box>
  );
};

export default EntityTimelineView;
